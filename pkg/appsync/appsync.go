// Package appsync performs an in-band sync: signed PUT to an SDK URL, signed
// response parsed into sdk.RegisterRequest. Persistence is the caller's job.
package appsync

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/inngest/inngest/pkg/execution/exechttp"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngestgo"
)

var (
	ErrMissingURL           = errors.New("url is required")
	ErrMissingSigningKey    = errors.New("signing key is required")
	ErrMissingExpectedAppID = errors.New("expected app id is required")
)

// 10 MiB clears any realistic register payload while bounding a misbehaving
// endpoint. Matches pkg/deploy.
const maxResponseBytes = 10 * 1024 * 1024

// Caps total request duration, independent of any per-stage transport timeouts.
const requestTimeout = 10 * time.Second

// Opts configures a Sync call.
type Opts struct {
	// AllowInsecureHTTP permits http:// URLs.
	AllowInsecureHTTP bool

	// ExpectedAppID, when non-empty, requires the SDK's reported app_id to
	// match. Mismatch fails the sync.
	ExpectedAppID string

	// HTTPClient overrides the built-in client; scheme check still runs.
	HTTPClient *http.Client

	// ServerKind sets X-Inngest-Server-Kind. Defaults to ServerKindCloud.
	ServerKind string

	SigningKey string

	URL string
}

func (o *Opts) Validate() error {
	if o.ExpectedAppID == "" {
		return ErrMissingExpectedAppID
	}
	if o.SigningKey == "" {
		return ErrMissingSigningKey
	}
	if o.URL == "" {
		return ErrMissingURL
	}
	return nil
}

// Response mirrors the SDK's in-band sync handler body.
type Response struct {
	AppID       string            `json:"app_id"`
	Env         *string           `json:"env"`
	Framework   *string           `json:"framework"`
	Functions   []sdk.SDKFunction `json:"functions"`
	Inspection  map[string]any    `json:"inspection"`
	Platform    *string           `json:"platform"`
	SDKAuthor   string            `json:"sdk_author"`
	SDKLanguage string            `json:"sdk_language"`
	SDKVersion  string            `json:"sdk_version"`
	URL         string            `json:"url"`
}

// ToRegisterRequest returns a normalized RegisterRequest so checksums match
// the /fn/register pipeline.
func (r *Response) ToRegisterRequest() *sdk.RegisterRequest {
	req := &sdk.RegisterRequest{
		AppName:      r.AppID,
		Capabilities: r.capabilities(),
		Framework:    deref(r.Framework),
		Functions:    r.Functions,
		Headers: sdk.Headers{
			Env:      deref(r.Env),
			Platform: deref(r.Platform),
		},
		SDK: fmt.Sprintf("%s:%s", r.SDKLanguage, r.SDKVersion),
		URL: r.URL,
	}
	_ = req.Normalize(false)
	return req
}

// capabilities is best-effort: zero value when absent or malformed.
func (r *Response) capabilities() sdk.Capabilities {
	raw, ok := r.Inspection["capabilities"]
	if !ok {
		return sdk.Capabilities{}
	}
	byt, err := json.Marshal(raw)
	if err != nil {
		return sdk.Capabilities{}
	}
	var caps sdk.Capabilities
	_ = json.Unmarshal(byt, &caps)
	return caps
}

// Sync performs an in-band sync. Exactly one return is non-nil:
//   - *Response: success, signature-validated.
//   - *syscode.Error: protocol-level failure (signature, non-2xx, unreachable,
//     Cloudflare, policy). Switch on Code.
//   - error: bug on our side (bad opts, marshaling).
func Sync(ctx context.Context, opts Opts) (*Response, *syscode.Error, error) {
	if err := opts.Validate(); err != nil {
		return nil, nil, err
	}
	if syscodeErr := checkScheme(opts.URL, opts.AllowInsecureHTTP); syscodeErr != nil {
		return nil, syscodeErr, nil
	}

	serverKind := opts.ServerKind
	if serverKind == "" {
		serverKind = headers.ServerKindCloud
	}

	client := opts.HTTPClient
	if client == nil {
		client = newClient(opts)
	}

	reqByt, err := json.Marshal(map[string]string{"url": opts.URL})
	if err != nil {
		return nil, nil, fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		opts.URL,
		bytes.NewReader(reqByt),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set(headers.HeaderContentType, "application/json")
	req.Header.Set(headers.HeaderKeyServerKind, serverKind)
	req.Header.Set(inngestgo.HeaderKeySyncKind, inngestgo.SyncKindInBand)

	sig, err := inngestgo.Sign(ctx, time.Now(), []byte(opts.SigningKey), reqByt)
	if err != nil {
		return nil, nil, fmt.Errorf("sign request: %w", err)
	}
	req.Header.Set(headers.HeaderKeySignature, sig)

	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, errRedirectDenied) {
			return nil, &syscode.Error{
				Code:    syscode.CodeRedirectDenied,
				Message: "redirects are not permitted",
			}, nil
		}
		// Go's dial errors include the resolved IP and lookup failure mode;
		// don't reflect them to API callers.
		return nil, &syscode.Error{
			Code:    syscode.CodeHTTPUnreachable,
			Message: "unable to reach SDK URL",
		}, nil
	}
	defer resp.Body.Close()

	if resp.Header.Get("Cf-Mitigated") != "" {
		return nil, &syscode.Error{
			Code:    syscode.CodeCloudflareMitigated,
			Message: "request was mitigated by cloudflare",
			Data: syscode.DataHTTPErr{
				Headers:    resp.Header,
				StatusCode: resp.StatusCode,
			}.ToMap(),
		}, nil
	}

	// +1 distinguishes "at limit" from "over limit".
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, &syscode.Error{
			Code:    syscode.CodeHTTPUnreachable,
			Message: "failed to read SDK response body",
		}, nil
	}
	if len(body) > maxResponseBytes {
		return nil, &syscode.Error{
			Code: syscode.CodeOutputTooLarge,
			Message: fmt.Sprintf(
				"response body exceeds %d bytes", maxResponseBytes,
			),
		}, nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Body is pre-signature-validation and may be attacker-controlled. Do
		// not echo it into Message; callers reflect Message to API responses.
		return nil, &syscode.Error{
			Code:    syscode.CodeHTTPNotOK,
			Message: fmt.Sprintf("SDK returned non-2xx response: status=%d", resp.StatusCode),
			Data: syscode.DataHTTPErr{
				Headers:    resp.Header,
				StatusCode: resp.StatusCode,
			}.ToMap(),
		}, nil
	}

	respSig := resp.Header.Get(headers.HeaderKeySignature)
	if respSig == "" {
		return nil, &syscode.Error{
			Code: syscode.CodeHTTPMissingHeader,
			Message: fmt.Sprintf(
				"missing %s response header", headers.HeaderKeySignature,
			),
		}, nil
	}

	valid, sigErr := inngestgo.ValidateResponseSignature(
		ctx,
		respSig,
		[]byte(opts.SigningKey),
		body,
	)
	if sigErr != nil || !valid {
		// Don't reflect sigErr.Error(); it distinguishes expired/invalid/bad-ts
		// to the API caller, which is a small probe oracle on key state.
		return nil, &syscode.Error{
			Code:    syscode.CodeSigVerificationFailed,
			Message: "invalid response signature",
		}, nil
	}

	var out Response
	if err := json.Unmarshal(body, &out); err != nil {
		// json error strings can include byte offsets and raw token bytes from
		// the body; keep them out of the public message.
		return nil, &syscode.Error{
			Code:    syscode.CodeMalformedResponse,
			Message: "failed to parse SDK response",
		}, nil
	}

	if out.AppID != opts.ExpectedAppID {
		return nil, &syscode.Error{
			Code:    syscode.CodeAppIDMismatch,
			Message: fmt.Sprintf("app_id mismatch: expected %q, SDK reported %q", opts.ExpectedAppID, out.AppID),
		}, nil
	}

	return &out, nil, nil
}

// checkScheme rejects unsupported schemes and gates http:// behind the flag.
func checkScheme(rawURL string, allowInsecureHTTP bool) *syscode.Error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return &syscode.Error{
			Code:    syscode.CodeURLSchemeDenied,
			Message: fmt.Sprintf("invalid url: %s", err.Error()),
		}
	}
	switch u.Scheme {
	case "https":
		return nil
	case "http":
		if !allowInsecureHTTP {
			return &syscode.Error{
				Code:    syscode.CodeURLSchemeDenied,
				Message: "insecure http:// scheme not permitted",
			}
		}
		return nil
	default:
		return &syscode.Error{
			Code:    syscode.CodeURLSchemeDenied,
			Message: fmt.Sprintf("unsupported url scheme %q", u.Scheme),
		}
	}
}

// newClient builds an HTTP client that allows private networks (single-host
// self-hosted setups commonly run the SDK on the same box as Inngest) and
// refuses redirects.
func newClient(opts Opts) *http.Client {
	transport := exechttp.Transport(exechttp.SecureDialerOpts{
		AllowPrivate:    true,
		AllowHostDocker: true,
	})
	return &http.Client{
		Timeout:       requestTimeout,
		Transport:     transport,
		CheckRedirect: refuseRedirects,
	}
}

// errRedirectDenied is returned by refuseRedirects so callers can distinguish
// our redirect refusal from generic dial failures via errors.Is.
var errRedirectDenied = errors.New("redirects are not permitted")

// refuseRedirects: the SDK URL is canonical, and the request carries a signed
// body we don't want forwarded cross-origin.
func refuseRedirects(_ *http.Request, _ []*http.Request) error {
	return errRedirectDenied
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
