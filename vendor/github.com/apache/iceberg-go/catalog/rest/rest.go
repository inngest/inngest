// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package rest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"iter"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	iceio "github.com/apache/iceberg-go/io"
	"github.com/apache/iceberg-go/metrics"
	"github.com/apache/iceberg-go/table"
	"github.com/apache/iceberg-go/udf"
	"github.com/apache/iceberg-go/view"
	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/sync/semaphore"
)

var (
	_ catalog.Catalog              = (*Catalog)(nil)
	_ catalog.TransactionalCatalog = (*Catalog)(nil)
)

const (
	pageSizeKey contextKey = "page_size"
	skipOAuth   contextKey = "skip_oauth"

	defaultPageSize = 20

	keyOauthToken        = "token"
	keyWarehouseLocation = "warehouse"
	keyMetadataLocation  = "metadata_location"
	keyOauthCredential   = "credential"
	keyScope             = "scope"
	keyAudience          = "audience"
	keyResource          = "resource"

	authorizationHeader = "Authorization"
	bearerPrefix        = "Bearer"
	keyPrefix           = "prefix"

	// headerIcebergAccessDelegation requests server-side storage-credential
	// vending. It is a session default (defaultAccessDelegation) and a
	// per-request header on the scan-planning endpoints that vend credentials.
	headerIcebergAccessDelegation = "X-Iceberg-Access-Delegation"
	defaultAccessDelegation       = "vended-credentials"

	keyNamespaceSeparator     = "namespace-separator"
	defaultNamespaceSeparator = "%1F"

	icebergRestSpecVersion = "0.14.1"

	keyRestSigV4        = "rest.sigv4-enabled"
	keyRestSigV4Region  = "rest.signing-region"
	keyRestSigV4Service = "rest.signing-name"
	keyAuthUrl          = "rest.authorization-url"
	// keyOAuth2ServerURI is the portable, spec-aligned property for the OAuth2
	// token endpoint used by Java, PyIceberg and iceberg-rust. It is the
	// preferred key; keyAuthUrl is retained as a compatibility alias. When both
	// are set, oauth2-server-uri takes precedence.
	keyOAuth2ServerURI = "oauth2-server-uri"
	keyTlsSkipVerify   = "rest.tls.skip-verify"

	keyViewEndpointsSupported = "view-endpoints-supported"
	keySnapshotLoadingMode    = "snapshot-loading-mode"
)

var (
	ErrRESTError            = errors.New("REST error")
	ErrBadRequest           = fmt.Errorf("%w: bad request", ErrRESTError)
	ErrForbidden            = fmt.Errorf("%w: forbidden", ErrRESTError)
	ErrUnauthorized         = fmt.Errorf("%w: unauthorized", ErrRESTError)
	ErrAuthorizationExpired = fmt.Errorf("%w: authorization expired", ErrRESTError)
	ErrServiceUnavailable   = fmt.Errorf("%w: service unavailable", ErrRESTError)
	ErrServerError          = fmt.Errorf("%w: server error", ErrRESTError)
	// ErrCommitFailed wraps both ErrRESTError and table.ErrCommitFailed
	// so that callers can detect retryable commit conflicts via
	// errors.Is(err, table.ErrCommitFailed).
	ErrCommitFailed       = fmt.Errorf("%w: %w", ErrRESTError, table.ErrCommitFailed)
	ErrCommitStateUnknown = fmt.Errorf("%w: commit failed due to unknown reason", ErrRESTError)
	ErrOAuthError         = fmt.Errorf("%w: oauth error", ErrRESTError)
)

// SnapshotMode controls which snapshots are included in a loadTable response.
type SnapshotMode string

const (
	// SnapshotModeAll requests all currently valid snapshots (server default).
	SnapshotModeAll SnapshotMode = "all"
	// SnapshotModeRefs requests only snapshots that are referenced by at least one named branch or tag.
	// The server omits unreferenced historical snapshots, which reduces response size for tables
	// with long snapshot histories. Note: time-travel to a snapshot not referenced by any branch
	// or tag will fail, as that snapshot will not be present in the returned metadata.
	SnapshotModeRefs SnapshotMode = "refs"
)

func init() {
	reg := catalog.RegistrarFunc(func(ctx context.Context, name string, p iceberg.Properties) (catalog.Catalog, error) {
		return newCatalogFromProps(ctx, name, p.Get("uri", ""), p)
	})

	catalog.Register(string(catalog.REST), reg)
	catalog.Register("http", reg)
	catalog.Register("https", reg)
}

type errorResponse struct {
	Message string   `json:"message"`
	Type    string   `json:"type"`
	Code    int      `json:"code"`
	Stack   []string `json:"stack,omitempty"`

	wrapping   error
	statusCode int
	retryAfter string
}

type contextKey string

func (e errorResponse) Unwrap() error { return e.wrapping }
func (e errorResponse) Error() string {
	switch {
	case e.Type != "" && e.Message != "":
		return e.Type + ": " + e.Message
	case e.Message != "":
		return e.Message
	case e.Type != "":
		return e.Type
	case e.wrapping != nil:
		return e.wrapping.Error()
	default:
		return "unknown REST error"
	}
}

type identifier struct {
	Namespace []string `json:"namespace"`
	Name      string   `json:"name"`
}

type commitTableResponse struct {
	MetadataLoc string          `json:"metadata-location"`
	RawMetadata json.RawMessage `json:"metadata"`
}

// StorageCredential carries REST-vended storage credentials scoped to matching
// object-location prefixes.
type StorageCredential struct {
	Prefix string             `json:"prefix"`
	Config iceberg.Properties `json:"config"`
}

type loadTableResponse struct {
	MetadataLoc        string              `json:"metadata-location"`
	RawMetadata        json.RawMessage     `json:"metadata"`
	Config             iceberg.Properties  `json:"config"`
	StorageCredentials []StorageCredential `json:"storage-credentials"`
	Labels             *iceberg.Labels     `json:"labels,omitempty"`
	Metadata           table.Metadata      `json:"-"`
}

type loadCredentialsResponse struct {
	StorageCredentials []StorageCredential `json:"storage-credentials"`
}

func (t *loadTableResponse) UnmarshalJSON(b []byte) (err error) {
	type Alias loadTableResponse
	if err = json.Unmarshal(b, (*Alias)(t)); err != nil {
		return err
	}

	t.Metadata, err = table.ParseMetadataBytes(t.RawMetadata)

	return err
}

// createTableRequest is the create-table wire format. This client only encodes
// it, from a spec and order already bound to the schema being sent. Source IDs
// a server decodes refer to the schema in the same request, under whatever
// numbering the client chose, so they need not be positive: decode into
// iceberg.UnboundPartitionSpec and table.UnboundSortOrder, then bind through
// table.NewMetadata (see #1664).
type createTableRequest struct {
	Name          string                 `json:"name"`
	Schema        *iceberg.Schema        `json:"schema"`
	Location      string                 `json:"location,omitempty"`
	PartitionSpec *iceberg.PartitionSpec `json:"partition-spec,omitempty"`
	WriteOrder    *table.SortOrder       `json:"write-order,omitempty"`
	StageCreate   bool                   `json:"stage-create"`
	Props         iceberg.Properties     `json:"properties,omitempty"`
}

type configResponse struct {
	Defaults  iceberg.Properties `json:"defaults"`
	Overrides iceberg.Properties `json:"overrides"`
	Endpoints []string           `json:"endpoints,omitempty"`
}

type sessionTransport struct {
	http.RoundTripper

	authManager    AuthManager
	defaultHeaders http.Header
	signer         v4.HTTPSigner
	cfg            aws.Config
	service        string
	newHash        func() hash.Hash
}

// from https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/aws/signer/v4#Signer.SignHTTP
const emptyStringHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

func (s *sessionTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	// A session default is applied unless the request already carries that
	// header (a per-request override of any default, not just Content-Type
	// wins) or explicitly opted out of it via withSuppressedHeaders (carried on
	// the context as an explicit set, never inferred from header values).
	suppressed := suppressedHeadersFrom(r.Context())
	for k, v := range s.defaultHeaders {
		ck := http.CanonicalHeaderKey(k)
		if _, ok := r.Header[ck]; ok {
			continue
		}
		if _, ok := suppressed[ck]; ok {
			continue
		}
		for _, hdr := range v {
			r.Header.Add(k, hdr)
		}
	}

	// Ordering is load-bearing: authManager runs after the default-header loop
	// and Set-overwrites, so the managed auth header always wins even though the
	// loop above lets an arbitrary per-request header (withHeaders) override a
	// session default of the same key. A caller cannot suppress or spoof the
	// Authorization header by supplying its own. Do not reorder this before the
	// default-header loop.
	if s.authManager != nil && r.Context().Value(skipOAuth) == nil {
		var (
			k, v string
			err  error
		)
		// Prefer the context-aware path so the request deadline also bounds auth,
		// falling back to the context-free method for managers that do not
		// implement ContextAuthManager.
		if cm, ok := s.authManager.(ContextAuthManager); ok {
			k, v, err = cm.AuthHeaderWithContext(r.Context())
		} else {
			k, v, err = s.authManager.AuthHeader()
		}
		if err != nil {
			return nil, err
		}
		r.Header.Set(k, v)
	}

	if s.signer != nil {
		var payloadHash string
		if r.Body == nil {
			payloadHash = emptyStringHash
		} else {
			rdr, err := r.GetBody()
			if err != nil {
				return nil, err
			}

			h := s.newHash()
			if _, err = io.Copy(h, rdr); err != nil {
				if closeErr := rdr.Close(); closeErr != nil {
					err = errors.Join(err, closeErr)
				}

				return nil, err
			}

			if err = rdr.Close(); err != nil {
				return nil, err
			}

			payloadHash = hex.EncodeToString(h.Sum(nil))
		}

		creds, err := s.cfg.Credentials.Retrieve(r.Context())
		if err != nil {
			return nil, err
		}

		// Set the x-amz-content-sha256 header before signing.
		// This header is required for AWS SigV4 signature verification.
		r.Header.Set("x-amz-content-sha256", payloadHash)

		// modifies the request in place
		err = s.signer.SignHTTP(r.Context(), creds, r, payloadHash,
			s.service, s.cfg.Region, time.Now())
		if err != nil {
			return nil, err
		}
	}

	return s.RoundTripper.RoundTrip(r)
}

// suppressHeadersKey carries the set of session-default header keys (canonical
// form) that a request opted out of, so sessionTransport.RoundTrip can consult
// an explicit set rather than inferring suppression from header values.
type suppressHeadersKey struct{}

// reqConfig holds the optional per-request knobs shared by do and doPost:
// headers to set, session-default headers to suppress, a body error.type ->
// error override, and whether an HTTP 204 is a valid empty result.
type reqConfig struct {
	headers           map[string]string
	suppressHeaders   []string
	errorTypeOverride map[string]error
	allowNoContent    bool
	requireBody       bool
	queryParams       url.Values
}

type reqOption func(*reqConfig)

// withHeaders sets per-request headers, merging with any already set so options
// compose. A per-request header takes precedence over a session default because
// sessionTransport.RoundTrip skips any default whose key is already present.
func withHeaders(headers map[string]string) reqOption {
	return func(c *reqConfig) {
		if c.headers == nil {
			c.headers = make(map[string]string, len(headers))
		}
		maps.Copy(c.headers, headers)
	}
}

// withSuppressedHeaders prevents the named session-default headers from being
// sent on this request. Names accumulate so options compose. Suppression is
// carried through the request context (see RoundTrip), not encoded as a header
// value, so req.Header only ever means "headers to send".
func withSuppressedHeaders(names ...string) reqOption {
	return func(c *reqConfig) { c.suppressHeaders = append(c.suppressHeaders, names...) }
}

// withErrorTypeOverride maps a non-200 response to a specific error by the REST
// error.type in the response body. It takes precedence over the status-code
// override so a shared status (e.g. a 404 that can be a missing plan, table, or
// namespace) can split into distinct sentinels. Entries accumulate so options
// compose; an unset or unmatched error.type falls through to the status-code
// override.
func withErrorTypeOverride(byType map[string]error) reqOption {
	return func(c *reqConfig) {
		if c.errorTypeOverride == nil {
			c.errorTypeOverride = make(map[string]error, len(byType))
		}
		maps.Copy(c.errorTypeOverride, byType)
	}
}

// allowNoContent treats an HTTP 204 No Content response as a valid empty result
// rather than trying to decode a body.
func allowNoContent() reqOption {
	return func(c *reqConfig) { c.allowNoContent = true }
}

// requireBody rejects an HTTP 200 with an empty body rather than decoding it to
// the zero result. Use it when a 200 must carry a payload, so a truncated or
// proxy-stripped response cannot masquerade as a successful empty result (e.g.
// fetchScanTasks returning "zero tasks" and silently dropping work).
func requireBody() reqOption {
	return func(c *reqConfig) { c.requireBody = true }
}

// Appends query parameters to the request.
func withQueryParams(params url.Values) reqOption {
	return func(c *reqConfig) {
		if c.queryParams == nil {
			c.queryParams = make(url.Values, len(params))
		}
		maps.Copy(c.queryParams, params)
	}
}

func newReqConfig(opts []reqOption) reqConfig {
	var cfg reqConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}

// withSuppressedHeadersCtx returns ctx carrying the canonicalized suppression
// set, or ctx unchanged when nothing is suppressed.
func withSuppressedHeadersCtx(ctx context.Context, names []string) context.Context {
	if len(names) == 0 {
		return ctx
	}

	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		set[http.CanonicalHeaderKey(n)] = struct{}{}
	}

	return context.WithValue(ctx, suppressHeadersKey{}, set)
}

func suppressedHeadersFrom(ctx context.Context) map[string]struct{} {
	set, _ := ctx.Value(suppressHeadersKey{}).(map[string]struct{})

	return set
}

func do[T any](ctx context.Context, method string, baseURI *url.URL, path []string, cl *http.Client, override map[int]error, opts ...reqOption) (ret T, err error) {
	cfg := newReqConfig(opts)

	var (
		req *http.Request
		rsp *http.Response
	)

	u := baseURI.JoinPath(path...)
	if len(cfg.queryParams) > 0 {
		q := u.Query()
		for k, vs := range cfg.queryParams {
			for _, v := range vs {
				q.Add(k, v)
			}
		}
		u.RawQuery = q.Encode()
	}
	uri := u.String()
	ctx = withSuppressedHeadersCtx(ctx, cfg.suppressHeaders)
	if req, err = http.NewRequestWithContext(ctx, method, uri, nil); err != nil {
		return ret, err
	}
	setRequestHeaders(req, cfg.headers)

	if rsp, err = cl.Do(req); err != nil {
		return ret, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, rsp.Body)
		_ = rsp.Body.Close()
	}()

	if cfg.allowNoContent && rsp.StatusCode == http.StatusNoContent {
		return ret, err
	}

	if rsp.StatusCode != http.StatusOK {
		return ret, handleNon200(rsp, override, cfg.errorTypeOverride)
	}

	if method == http.MethodHead || method == http.MethodDelete {
		return ret, err
	}

	if err = decodeJSONResponse(rsp.Body, &ret); err != nil {
		return ret, err
	}

	return ret, err
}

func doGet[T any](ctx context.Context, baseURI *url.URL, path []string, cl *http.Client, override map[int]error, opts ...reqOption) (ret T, err error) {
	return do[T](ctx, http.MethodGet, baseURI, path, cl, override, opts...)
}

func doDelete[T any](ctx context.Context, baseURI *url.URL, path []string, cl *http.Client, override map[int]error, opts ...reqOption) (ret T, err error) {
	return do[T](ctx, http.MethodDelete, baseURI, path, cl, override, append([]reqOption{allowNoContent()}, opts...)...)
}

func doHead(ctx context.Context, baseURI *url.URL, path []string, cl *http.Client, override map[int]error) error {
	_, err := do[struct{}](ctx, http.MethodHead, baseURI, path, cl, override, allowNoContent())

	return err
}

func doPost[Payload, Result any](ctx context.Context, baseURI *url.URL, path []string, payload Payload, cl *http.Client, override map[int]error, opts ...reqOption) (ret Result, err error) {
	cfg := newReqConfig(opts)

	var (
		req  *http.Request
		rsp  *http.Response
		data []byte
	)

	uri := baseURI.JoinPath(path...).String()
	data, err = json.Marshal(payload)
	if err != nil {
		return ret, err
	}

	ctx = withSuppressedHeadersCtx(ctx, cfg.suppressHeaders)
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, uri, bytes.NewReader(data))
	if err != nil {
		return ret, err
	}
	setRequestHeaders(req, cfg.headers)

	rsp, err = cl.Do(req)
	if err != nil {
		return ret, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, rsp.Body)
		_ = rsp.Body.Close()
	}()

	if cfg.allowNoContent && rsp.StatusCode == http.StatusNoContent {
		return ret, err
	}

	if rsp.StatusCode != http.StatusOK {
		return ret, handleNon200(rsp, override, cfg.errorTypeOverride)
	}

	if rsp.ContentLength == 0 {
		if cfg.requireBody {
			return ret, fmt.Errorf("%w: empty response body on %s %s", ErrRESTError, req.Method, req.URL.Path)
		}

		return ret, err
	}

	if err = decodeJSONResponse(rsp.Body, &ret); err != nil {
		return ret, err
	}

	return ret, err
}

func decodeJSONResponse(body io.Reader, dst any) error {
	decoder := json.NewDecoder(body)
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("%w: error decoding json payload: `%s`", ErrRESTError, err.Error())
	}

	if err := rejectTrailingJSON(decoder); err != nil {
		return fmt.Errorf("%w: %s", ErrRESTError, err)
	}

	return nil
}

func rejectTrailingJSON(decoder *json.Decoder) error {
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errors.New("response contains multiple JSON values")
		}

		return fmt.Errorf("error decoding trailing json payload: `%s`", err.Error())
	}

	return nil
}

func setRequestHeaders(req *http.Request, headers map[string]string) {
	for k, v := range headers {
		req.Header.Set(k, v)
	}
}

func handleNon200(rsp *http.Response, override map[int]error, typeOverride map[string]error) error {
	e := errorResponse{
		statusCode: rsp.StatusCode,
		retryAfter: rsp.Header.Get("Retry-After"),
	}

	// Only try to decode if there's a body (HEAD requests don't have one)
	if rsp.ContentLength != 0 {

		payload := struct {
			Error   *errorResponse `json:"error"`
			Message string         `json:"message"`
			Type    string         `json:"type"`
		}{
			Error: &e,
		}

		decoder := json.NewDecoder(rsp.Body)
		decErr := decoder.Decode(&payload)
		if decErr == nil {
			decErr = rejectTrailingJSON(decoder)
		}
		if decErr != nil && decErr != io.EOF {
			// Preserve the HTTP metadata even when the server returned a non-JSON
			// error page. Callers such as WaitForPlan still need the status to apply
			// transport-level retry policy. A status the caller mapped keeps that
			// classification (e.g. an ambiguous commit 5xx stays ErrCommitStateUnknown)
			// even when a proxy garbled the body; unmapped statuses keep ErrRESTError.
			e.wrapping = ErrRESTError
			if statusErr, ok := override[rsp.StatusCode]; ok {
				e.wrapping = statusErr
			}

			return fmt.Errorf("%w: failed to decode error response: %s", e, decErr.Error())
		}

		if e.Message == "" && e.Type == "" {
			e.Message = payload.Message
			e.Type = payload.Type
		}
	}

	// The status-code override maps a status to a sentinel; a body error.type
	// override refines it further (e.g. splitting a 404 into missing-plan vs.
	// table vs. namespace). The type refinement only applies to a status the
	// caller already mapped, because these error.type values are defined for
	// specific statuses (404) in the REST spec: a 503 body carrying, say,
	// NoSuchPlanIdException must still resolve to ErrServiceUnavailable rather
	// than a 404 sentinel.
	if override != nil {
		if statusErr, ok := override[rsp.StatusCode]; ok {
			if typeOverride != nil && e.Type != "" {
				if typeErr, ok := typeOverride[e.Type]; ok {
					e.wrapping = typeErr

					return e
				}
			}
			e.wrapping = statusErr

			return e
		}
	}

	switch rsp.StatusCode {
	case http.StatusBadRequest:
		e.wrapping = ErrBadRequest
	case http.StatusUnauthorized:
		e.wrapping = ErrUnauthorized
	case http.StatusForbidden:
		e.wrapping = ErrForbidden
	case http.StatusUnprocessableEntity:
		e.wrapping = ErrRESTError
	case 419:
		e.wrapping = ErrAuthorizationExpired
	case http.StatusNotImplemented:
		e.wrapping = iceberg.ErrNotImplemented
	case http.StatusServiceUnavailable:
		e.wrapping = ErrServiceUnavailable
	default:
		if 500 <= rsp.StatusCode && rsp.StatusCode < 600 {
			e.wrapping = ErrServerError
		} else {
			e.wrapping = ErrRESTError
		}
	}

	return e
}

func fromProps(props iceberg.Properties, o *options) error {
	// The OAuth token endpoint may be configured via either the portable
	// oauth2-server-uri key or the legacy rest.authorization-url alias. Capture
	// both and resolve precedence after the loop, since map iteration order is
	// non-deterministic.
	var authURL, oauth2ServerURI *url.URL
	for k, v := range props {
		switch k {
		case keyOauthToken:
			if o.oauthToken == "" {
				o.oauthToken = v
			}
		case keyWarehouseLocation:
			o.warehouseLocation = v
		case keyMetadataLocation:
			o.metadataLocation = v
		case keyRestSigV4:
			o.enableSigv4 = strings.ToLower(v) == "true"
		case keyRestSigV4Region:
			o.sigv4Region = v
		case keyRestSigV4Service:
			o.sigv4Service = v
		case keyAuthUrl:
			u, err := parseAuthURL(keyAuthUrl, v)
			if err != nil {
				return err
			}
			authURL = u
		case keyOAuth2ServerURI:
			u, err := parseAuthURL(keyOAuth2ServerURI, v)
			if err != nil {
				return err
			}
			oauth2ServerURI = u
		case keyOauthCredential:
			o.credential = v
		case keyScope:
			o.scope = v
		case keyAudience:
			o.audience = v
		case keyResource:
			o.resource = v
		case keyPrefix:
			o.prefix = v
		case keyTlsSkipVerify:
			verify := strings.ToLower(v) == "true"
			if o.tlsConfig == nil {
				o.tlsConfig = &tls.Config{
					InsecureSkipVerify: verify,
				}
			} else {
				o.tlsConfig.InsecureSkipVerify = verify
			}
		case "uri", "type":
		default:
			if v != "" {
				if o.additionalProps == nil {
					o.additionalProps = iceberg.Properties{}
				}
				o.additionalProps[k] = v
			}
		}
	}

	// oauth2-server-uri is the portable, preferred key and takes precedence over
	// the rest.authorization-url alias when both are set.
	switch {
	case oauth2ServerURI != nil:
		o.authUri = oauth2ServerURI
	case authURL != nil:
		o.authUri = authURL
	}

	return nil
}

// resolveAuthURLAlias collapses the oauth2-server-uri / rest.authorization-url
// alias within a single configuration layer into the canonical
// oauth2-server-uri key. oauth2-server-uri wins over the rest.authorization-url
// alias within the same layer. Collapsing per layer before merging keeps
// defaults -> client -> overrides precedence intact.
func resolveAuthURLAlias(props iceberg.Properties) {
	v, ok := props[keyOAuth2ServerURI]
	if !ok {
		v, ok = props[keyAuthUrl]
	}
	delete(props, keyAuthUrl)
	if ok {
		props[keyOAuth2ServerURI] = v
	}
}

func parseAuthURL(key, raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid %s %q: %w", key, raw, err)
	}
	if u.Scheme == "" {
		return nil, fmt.Errorf("invalid %s %q: missing scheme", key, raw)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("invalid %s %q: missing host", key, raw)
	}

	return u, nil
}

func toProps(o *options) iceberg.Properties {
	props := iceberg.Properties{}
	maps.Copy(props, o.additionalProps)

	setIf := func(key, v string) {
		if v != "" {
			props[key] = v
		}
	}

	setIf(keyOauthToken, o.oauthToken)
	setIf(keyOauthCredential, o.credential)
	setIf(keyWarehouseLocation, o.warehouseLocation)
	setIf(keyMetadataLocation, o.metadataLocation)
	if o.enableSigv4 {
		props[keyRestSigV4] = "true"
		setIf(keyRestSigV4Region, o.sigv4Region)
		setIf(keyRestSigV4Service, o.sigv4Service)

		// Best-effort fallback: propagate the SigV4 signing region as
		// a client region hint so that S3 I/O can determine the correct
		// regional endpoint when s3.region is not explicitly set.
		// Only applied for S3/S3Tables services where the signing region
		// is likely to match the bucket region.
		if o.sigv4Region != "" && (o.sigv4Service == "s3" || o.sigv4Service == "s3tables") {
			if _, ok := props[iceio.S3ClientRegion]; !ok {
				props[iceio.S3ClientRegion] = o.sigv4Region
			}
		}
	}

	setIf(keyPrefix, o.prefix)
	if o.authUri != nil {
		// Advertise the endpoint only under the portable oauth2-server-uri key.
		// We canonicalize to this key everywhere else (fromProps precedence,
		// resolveAuthURLAlias), so emitting the legacy rest.authorization-url
		// alias too would leave the endpoint under two names in r.props and any
		// table config cloned from it - two sources of truth that can drift.
		setIf(keyOAuth2ServerURI, o.authUri.String())
	}

	return props
}

var _ catalog.PurgeableTable = (*Catalog)(nil)

type Catalog struct {
	baseURI *url.URL
	cl      *http.Client
	// closeSession releases transports owned by the catalog. Caller-provided
	// transports are excluded when the session is constructed.
	closeSession     func()
	closeSessionOnce sync.Once
	closeErr         error

	name string
	// Retained catalog properties are reused for table/view IO and may carry
	// authentication material such as credential or token.
	props     iceberg.Properties
	endpoints endpointSet

	namespaceSeparator string
	snapshotMode       SnapshotMode

	// reporter builds and caches the catalog's metrics reporter once, so it is
	// constructed per-catalog rather than per table load. Released by Close.
	reporter metrics.CachedReporter
	// reporterProps holds only the client-supplied properties, captured before
	// the server's /v1/config defaults and overrides are merged into props. The
	// metrics reporter is resolved from these so a server-vended
	// metrics-reporter-impl cannot select (or, via an unregistered name, break)
	// a reporter the client never asked for.
	reporterProps iceberg.Properties
	// metricsDispatcher POSTs scan/commit reports to the server's metrics
	// endpoint on a bounded worker pool shared across this catalog's table
	// reporters. It is nil unless the client opted in (reportMetricsEnabled, read
	// from reporterProps) and the server advertises the endpoint. Owned here and
	// closed by Close.
	metricsDispatcher *metricsDispatcher
}

func newCatalogFromProps(ctx context.Context, name string, uri string, p iceberg.Properties) (*Catalog, error) {
	var ops options
	if err := fromProps(p, &ops); err != nil {
		return nil, err
	}

	r := &Catalog{name: name}
	if err := r.init(ctx, &ops, uri); err != nil {
		return nil, err
	}

	return r, nil
}

func NewCatalog(ctx context.Context, name, uri string, opts ...Option) (*Catalog, error) {
	ops := &options{}
	for _, o := range opts {
		o(ops)
	}

	if ops.tlsConfig != nil && ops.transport != nil {
		return nil, errors.New("invalid catalog config with non-nil tlsConfig and transport: tlsConfig will be ignored, it should be added to the provided transport instead")
	}

	r := &Catalog{name: name}
	if err := r.init(ctx, ops, uri); err != nil {
		return nil, err
	}

	return r, nil
}

// defaultOAuthTimeout bounds a single OAuth token-refresh request made by the
// built-in Oauth2AuthManager. oauth2's TokenSource.Token() takes no context, so
// this Timeout on the refresh client is what keeps a stalled or black-holing
// token endpoint from blocking auth — and therefore any request that triggers a
// refresh, including asynchronous metrics reports — indefinitely.
const defaultOAuthTimeout = 60 * time.Second

// setupOAuthManager creates an Oauth2AuthManager based on the provided options.
// It uses golang.org/x/oauth2 for token management, caching, and thread-safe refresh.
// The returned cleanup closes an internally-created OAuth transport, if any.
func setupOAuthManager(r *Catalog, cl *http.Client, opts *options) (AuthManager, func()) {
	// If a static token is provided, use it directly.
	if opts.oauthToken != "" {
		return &Oauth2AuthManager{
			tokenSource: oauth2.StaticTokenSource(&oauth2.Token{
				AccessToken: opts.oauthToken,
				TokenType:   "Bearer",
			}),
		}, nil
	}

	// If no credential, no auth needed.
	if opts.credential == "" {
		return nil, nil
	}

	authURL := opts.authUri
	if authURL == nil {
		authURL = r.baseURI.JoinPath("oauth/tokens")
	}

	clientID, clientSecret, found := strings.Cut(opts.credential, ":")
	if !found {
		clientID = ""
		clientSecret = opts.credential
	}

	cfg := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     authURL.String(),
		AuthStyle:    oauth2.AuthStyleInParams,
	}

	scope := "catalog"
	if opts.scope != "" {
		scope = opts.scope
	}
	cfg.Scopes = []string{scope}

	if opts.audience != "" || opts.resource != "" {
		cfg.EndpointParams = url.Values{}
		if opts.audience != "" {
			cfg.EndpointParams.Set("audience", opts.audience)
		}
		if opts.resource != "" {
			cfg.EndpointParams.Set("resource", opts.resource)
		}
	}

	// Add skip oauth so we don't get in cycles trying to refresh the token
	ctx := context.WithValue(context.Background(), skipOAuth, true)

	// Token refresh uses a client with a Timeout so a stalled token endpoint
	// cannot block auth forever (oauth2's Token() takes no context). By default
	// this reuses the catalog client's transport — preserving its TLS, proxy and
	// header behavior — but as a distinct *http.Client so the Timeout applies to
	// refresh alone and not to ordinary catalog requests.
	oauthClient := &http.Client{Transport: cl.Transport, Timeout: defaultOAuthTimeout}
	var closeIdleConnections func()
	if opts.oauthTLSConfig != nil {
		transport := &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: opts.oauthTLSConfig,
		}
		oauthClient = &http.Client{
			Transport: transport,
			Timeout:   defaultOAuthTimeout,
		}
		closeIdleConnections = transport.CloseIdleConnections
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, oauthClient)

	return &Oauth2AuthManager{
		tokenSource: cfg.TokenSource(ctx),
	}, closeIdleConnections
}

func (r *Catalog) init(ctx context.Context, ops *options, uri string) error {
	baseuri, err := url.Parse(uri)
	if err != nil {
		return err
	}

	r.baseURI = baseuri.JoinPath("v1")

	// Capture the client-supplied properties before fetchConfig folds in the
	// server's /v1/config defaults and overrides, so the metrics reporter is
	// resolved from the client's choice alone (see the reporterProps field).
	r.reporterProps = toProps(ops)

	if ops, err = r.fetchConfig(ctx, ops); err != nil {
		return err
	}

	r.cl, r.closeSession, err = r.createSession(ctx, ops)
	if err != nil {
		return err
	}
	if ops.prefix != "" {
		r.baseURI = r.baseURI.JoinPath(ops.prefix)
	}
	r.props = toProps(ops)

	// Stand up the metrics dispatcher only when the client opted in and the
	// server advertises the endpoint. Enablement of outbound telemetry is read
	// from the client-supplied properties alone (see reporterProps): a server
	// must never be able to turn on reporting the client never asked for.
	//
	// Disabling, however, is always safe, so an explicit server-side false is
	// honored: r.props is the merged config (server defaults < client < server
	// overrides), so an override setting the key false resolves to false here and
	// suppresses a client that opted in — matching how a Java client resolves it
	// and letting an operator switch reporting off fleet-wide. A mere server
	// default cannot flip off a client opt-in, since client props beat defaults in
	// the merge.
	if reportMetricsEnabled(r.reporterProps) && reportMetricsEnabled(r.props) &&
		r.endpoints.check(endpointReportMetrics) == nil {
		r.metricsDispatcher = newMetricsDispatcher(
			metricsDispatchWorkers, metricsDispatchQueueSize, reportMetricsTimeout(r.reporterProps), nil)
	}

	return nil
}

// createSession returns a cleanup that closes only transports created by this
// function, never transports supplied by the caller.
func (r *Catalog) createSession(ctx context.Context, opts *options) (*http.Client, func(), error) {
	var baseTransport http.RoundTripper
	var cleanupFuncs []func()
	if opts.transport != nil {
		baseTransport = opts.transport
	} else if opts.transportFactory != nil {
		var cleanup func()
		baseTransport, cleanup = opts.transportFactory(opts.tlsConfig)
		if cleanup != nil {
			cleanupFuncs = append(cleanupFuncs, cleanup)
		}
	} else {
		transport := &http.Transport{Proxy: http.ProxyFromEnvironment, TLSClientConfig: opts.tlsConfig}
		baseTransport = transport
		cleanupFuncs = append(cleanupFuncs, transport.CloseIdleConnections)
	}
	cleanup := func() {
		for _, cleanupFunc := range slices.Backward(cleanupFuncs) {
			cleanupFunc()
		}
	}

	session := &sessionTransport{
		RoundTripper:   baseTransport,
		defaultHeaders: http.Header{},
	}
	cl := &http.Client{Transport: session}

	// If the user does not set an AuthManager, construct one for this session
	// without storing it in opts. Bootstrap authentication must not leak into
	// the final session after server configuration has been applied.
	authManager := opts.authManager
	if authManager == nil {
		var cleanupOAuth func()
		authManager, cleanupOAuth = setupOAuthManager(r, cl, opts)
		if cleanupOAuth != nil {
			cleanupFuncs = append(cleanupFuncs, cleanupOAuth)
		}
	}

	session.defaultHeaders.Set("X-Client-Version", icebergRestSpecVersion)
	session.defaultHeaders.Set("Content-Type", "application/json")
	session.defaultHeaders.Set("User-Agent", "GoIceberg/"+iceberg.Version())
	session.defaultHeaders.Set(headerIcebergAccessDelegation, defaultAccessDelegation)

	for k, v := range opts.headers {
		session.defaultHeaders.Set(k, v)
	}

	for k, v := range opts.additionalProps {
		if headerKey, found := strings.CutPrefix(k, "header."); found {
			session.defaultHeaders.Set(headerKey, v)
		}
	}

	if authManager != nil {
		session.authManager = authManager
	}

	if opts.enableSigv4 {
		cfg := opts.awsConfig
		if !opts.awsConfigSet {
			// If no config provided, load defaults from environment.
			var err error
			cfg, err = config.LoadDefaultConfig(ctx)
			if err != nil {
				cleanup()

				return nil, nil, err
			}
		}
		if opts.sigv4Region != "" {
			cfg.Region = opts.sigv4Region
		}

		session.cfg, session.service = cfg, opts.sigv4Service
		session.signer, session.newHash = v4.NewSigner(), sha256.New
	}

	return cl, cleanup, nil
}

func (r *Catalog) fetchConfig(ctx context.Context, opts *options) (*options, error) {
	params := url.Values{}
	if opts.warehouseLocation != "" {
		params.Set(keyWarehouseLocation, opts.warehouseLocation)
	}

	route := r.baseURI.JoinPath("config")
	route.RawQuery = params.Encode()

	sess, cleanup, err := r.createSession(ctx, opts)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	rsp, err := doGet[configResponse](ctx, route, []string{}, sess, nil)
	if err != nil {
		return nil, err
	}

	cfg := rsp.Defaults
	if cfg == nil {
		cfg = iceberg.Properties{}
	}

	// Collapse the oauth2-server-uri / rest.authorization-url alias within each
	// layer before merging so the defaults -> client -> overrides precedence is
	// preserved. Merging first would leave both keys in the map, letting a
	// client oauth2-server-uri shadow a server rest.authorization-url override.
	clientProps := toProps(opts)
	resolveAuthURLAlias(cfg)
	resolveAuthURLAlias(clientProps)
	resolveAuthURLAlias(rsp.Overrides)

	maps.Copy(cfg, clientProps)
	maps.Copy(cfg, rsp.Overrides)

	r.namespaceSeparator = cfg.Get(keyNamespaceSeparator, defaultNamespaceSeparator)
	r.snapshotMode = SnapshotMode(cfg.Get(keySnapshotLoadingMode, ""))
	if r.snapshotMode != "" && r.snapshotMode != SnapshotModeAll && r.snapshotMode != SnapshotModeRefs {
		return nil, fmt.Errorf("%w: invalid %s %q (want %q or %q)",
			ErrRESTError, keySnapshotLoadingMode, r.snapshotMode, SnapshotModeAll, SnapshotModeRefs)
	}

	// Negotiate capabilities from the endpoints the server advertises, falling
	// back to a backward-compatible default set when none are provided.
	r.endpoints = resolveEndpoints(rsp.Endpoints, cfg.GetBool(keyViewEndpointsSupported, false))

	o := *opts
	if err := fromProps(cfg, &o); err != nil {
		return nil, err
	}

	if uri, ok := cfg["uri"]; ok {
		r.baseURI, err = url.Parse(uri)
		if err != nil {
			return nil, err
		}
		r.baseURI = r.baseURI.JoinPath("v1")
	}

	return &o, nil
}

func (r *Catalog) Name() string              { return r.name }
func (r *Catalog) CatalogType() catalog.Type { return catalog.REST }

// Close drains idle connections from transports created by the catalog and
// releases its metrics reporter. Closing the metrics dispatcher cancels any
// in-flight reports and drains its workers so outbound telemetry stops.
// Caller-provided transports remain caller-owned. Close is safe to call more
// than once. Callers holding a [catalog.Catalog] can reach this via a
// [catalog.Closer] type assertion.
func (r *Catalog) Close() error {
	r.closeSessionOnce.Do(func() {
		if r.closeSession != nil {
			r.closeSession()
		}
		if r.metricsDispatcher != nil {
			r.metricsDispatcher.close()
		}
		r.closeErr = r.reporter.Close()
	})

	return r.closeErr
}

var _ catalog.Closer = (*Catalog)(nil)

func (r *Catalog) checkValidNamespace(ident table.Identifier) error {
	if err := catalog.ValidateNamespaceIdentifier(ident); err != nil {
		return err
	}

	separator := r.decodedNamespaceSeparator()
	if separator == "" {
		return nil
	}

	for _, part := range ident {
		if strings.Contains(part, separator) {
			return fmt.Errorf("%w: namespace component %q contains the namespace separator %q in %v",
				catalog.ErrNoSuchNamespace, part, separator, strings.Join(ident, "."))
		}
	}

	return nil
}

func (r *Catalog) tableFromResponse(
	_ context.Context,
	identifier []string,
	metadata table.Metadata,
	loc string,
	config iceberg.Properties,
	scanPlanningConfig iceberg.Properties,
	credsVended bool,
) (*table.Table, error) {
	var fsF func(context.Context) (iceio.IO, error)
	if credsVended {
		refresher := &vendedCredentialRefresher{
			mu:         semaphore.NewWeighted(1),
			identifier: identifier,
			location:   loc,
			props:      config,
			fetchCreds: func(ctx context.Context, ident []string) (iceberg.Properties, error) {
				return r.fetchTableCreds(ctx, ident, loc)
			},
		}
		fsF = refresher.loadFS
	} else {
		fsF = iceio.LoadFSFunc(config, loc)
	}

	// Resolve the reporter from the client-supplied properties only (see the
	// reporterProps field). r.props folds in server-vended /v1/config and, per
	// call, table metadata and creds — any of which could otherwise shadow the
	// client's reporter choice or turn an unknown server-side name into a hard
	// load error. Server-vended reporter selection, if ever wanted, should be an
	// explicit decision rather than a side effect of merge order.
	reporter, err := r.reporter.Get(r.reporterProps)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize metrics reporter: %w", err)
	}
	// Opt-in: POST scan/commit reports to the catalog's metrics endpoint via the
	// catalog-owned dispatcher. It is non-nil only when the client enabled
	// reporting and the server advertises the endpoint (see init).
	if r.metricsDispatcher != nil {
		if ns, tbl, idErr := r.splitIdentForPath(identifier); idErr != nil {
			// Unreachable for a valid identifier that already loaded above, but if it
			// ever regresses, log why the REST reporter was dropped rather than let
			// metrics silently vanish with no signal.
			slog.Debug("iceberg: skipping REST metrics reporter, cannot split identifier",
				"error", idErr)
		} else if path, pErr := endpointReportMetrics.reqPath(ns, tbl); pErr != nil {
			slog.Debug("iceberg: skipping REST metrics reporter, cannot build metrics path",
				"error", pErr)
		} else {
			reporter = metrics.Combine(reporter, &restMetricsReporter{
				baseURI:    r.baseURI,
				cl:         r.cl,
				path:       path,
				dispatcher: r.metricsDispatcher,
			})
		}
	}

	return table.New(
		identifier,
		metadata,
		loc,
		fsF,
		r,
		table.WithMetricsReporter(reporter),
		table.WithScanPlanningIOProperties(scanPlanningConfig),
	), nil
}

func (r *Catalog) fetchTableCreds(ctx context.Context, ident []string, location string) (iceberg.Properties, error) {
	if err := r.endpoints.check(endpointTableCredentials); err != nil {
		return nil, err
	}

	ns, tbl, err := r.splitIdentForPath(ident)
	if err != nil {
		return nil, err
	}

	path, err := endpointTableCredentials.reqPath(ns, tbl)
	if err != nil {
		return nil, err
	}

	ret, err := doGet[loadCredentialsResponse](ctx, r.baseURI, path,
		r.cl, map[int]error{http.StatusNotFound: catalog.ErrNoSuchTable})
	if err != nil {
		return nil, err
	}

	return resolveStorageCredentials(ret.StorageCredentials, location), nil
}

type identifierPageFetcher func(pageToken string) ([]table.Identifier, string, error)

func paginateIdentifiers(fetch identifierPageFetcher, yield func(table.Identifier) bool) error {
	seenTokens := make(map[string]struct{})
	var pageToken string

	for {
		identifiers, nextPageToken, err := fetch(pageToken)
		if err != nil {
			return err
		}
		for _, identifier := range identifiers {
			if !yield(identifier) {
				return nil
			}
		}
		if nextPageToken == "" {
			return nil
		}
		if _, ok := seenTokens[nextPageToken]; ok {
			return fmt.Errorf("%w: pagination cycle: repeated page token", ErrRESTError)
		}

		seenTokens[nextPageToken] = struct{}{}
		pageToken = nextPageToken
	}
}

// ListTables lists the tables in a namespace. If the server does not advertise
// the list-tables endpoint, it yields no tables rather than an error.
func (r *Catalog) ListTables(ctx context.Context, namespace table.Identifier) iter.Seq2[table.Identifier, error] {
	return func(yield func(table.Identifier, error) bool) {
		pageSize := r.getPageSize(ctx)
		err := paginateIdentifiers(func(pageToken string) ([]table.Identifier, string, error) {
			return r.listTablesPage(ctx, namespace, pageToken, pageSize)
		}, func(identifier table.Identifier) bool {
			return yield(identifier, nil)
		})
		if err != nil {
			yield(table.Identifier{}, err)
		}
	}
}

func (r *Catalog) listTablesPage(ctx context.Context, namespace table.Identifier, pageToken string, pageSize int) ([]table.Identifier, string, error) {
	if err := r.checkValidNamespace(namespace); err != nil {
		return nil, "", err
	}
	// Unsupported listing yields an empty result rather than an error.
	if !r.endpoints.allowed(endpointListTables) {
		return nil, "", nil
	}
	ns := r.encodeNamespace(namespace)
	path, err := endpointListTables.reqPath(ns)
	if err != nil {
		return nil, "", err
	}
	uri := r.baseURI.JoinPath(path...)

	v := url.Values{}
	if pageSize > 0 {
		v.Set("pageSize", strconv.Itoa(pageSize))
	}
	if pageToken != "" {
		v.Set("pageToken", pageToken)
	}

	uri.RawQuery = v.Encode()
	type resp struct {
		Identifiers   []identifier `json:"identifiers"`
		NextPageToken string       `json:"next-page-token,omitempty"`
	}
	rsp, err := doGet[resp](ctx, uri, []string{}, r.cl, map[int]error{http.StatusNotFound: catalog.ErrNoSuchNamespace})
	if err != nil {
		return nil, "", err
	}
	out := make([]table.Identifier, len(rsp.Identifiers))
	for i, id := range rsp.Identifiers {
		out[i] = append(id.Namespace, id.Name)
	}

	return out, rsp.NextPageToken, nil
}

func (r *Catalog) nsSeparator() string {
	if r.namespaceSeparator == "" {
		return defaultNamespaceSeparator
	}

	return r.namespaceSeparator
}

// encodeNamespace URL-encodes each namespace level and joins them with the
// server-advertised, URL-encoded namespace separator for use as a REST path
// segment. Mirrors RESTUtil.encodeNamespace in the Java implementation.
func (r *Catalog) encodeNamespace(namespace table.Identifier) string {
	encoded := make([]string, len(namespace))
	for i, level := range namespace {
		encoded[i] = url.PathEscape(level)
	}

	return strings.Join(encoded, r.nsSeparator())
}

func (r *Catalog) decodedNamespaceSeparator() string {
	sep, err := url.PathUnescape(r.nsSeparator())
	if err != nil {
		return r.nsSeparator()
	}

	return sep
}

// namespaceToQueryParam joins the raw namespace levels with the decoded
// separator for use as a query-parameter value, which the HTTP layer then
// percent-encodes. Mirrors RESTUtil.namespaceToQueryParam in Java.
func (r *Catalog) namespaceToQueryParam(namespace table.Identifier) string {
	return strings.Join(namespace, r.decodedNamespaceSeparator())
}

func (r *Catalog) splitIdentForPath(ident table.Identifier) (string, string, error) {
	if err := catalog.ValidateTableIdentifier(ident); err != nil {
		return "", "", err
	}

	return r.encodeNamespace(catalog.NamespaceFromIdent(ident)), catalog.ObjectNameFromIdent(ident), nil
}

func (r *Catalog) splitViewIdentForPath(ident table.Identifier) (string, string, error) {
	if err := catalog.ValidateViewIdentifier(ident); err != nil {
		return "", "", err
	}

	return r.encodeNamespace(catalog.NamespaceFromIdent(ident)), catalog.ObjectNameFromIdent(ident), nil
}

func (r *Catalog) splitFunctionIdentForPath(ident table.Identifier) (string, string, error) {
	if err := catalog.ValidateFunctionIdentifier(ident); err != nil {
		return "", "", err
	}

	return r.encodeNamespace(catalog.NamespaceFromIdent(ident)), catalog.ObjectNameFromIdent(ident), nil
}

func (r *Catalog) CreateTable(ctx context.Context, identifier table.Identifier, schema *iceberg.Schema, opts ...catalog.CreateTableOpt) (*table.Table, error) {
	if err := r.endpoints.check(endpointCreateTable); err != nil {
		return nil, err
	}

	ns, tbl, err := r.splitIdentForPath(identifier)
	if err != nil {
		return nil, err
	}

	// Resolve the reporter before the create POST mutates the server: it is
	// otherwise first built in the trailing tableFromResponse, so a bad
	// metrics-reporter-impl would turn a table the server already created into a
	// reported failure whose retry hits ErrTableAlreadyExists. CachedReporter
	// caches this for that tableFromResponse call. Resolved from the
	// client-supplied properties only (see the reporterProps field).
	if _, err := r.reporter.Get(r.reporterProps); err != nil {
		return nil, fmt.Errorf("failed to initialize metrics reporter: %w", err)
	}

	cfg := catalog.NewCreateTableCfg()
	for _, o := range opts {
		o(&cfg)
	}

	if cfg.SortOrder.Fields() == nil && cfg.SortOrder.OrderID() == 0 {
		cfg.SortOrder = table.UnsortedSortOrder
	}

	stagedCreate := len(cfg.StagedUpdates) > 0

	payload := createTableRequest{
		Name:          tbl,
		Schema:        schema,
		Location:      cfg.Location,
		PartitionSpec: cfg.PartitionSpec,
		WriteOrder:    &cfg.SortOrder,
		StageCreate:   stagedCreate,
		Props:         cfg.Properties,
	}

	path, err := endpointCreateTable.reqPath(ns)
	if err != nil {
		return nil, err
	}

	ret, err := doPost[createTableRequest, loadTableResponse](ctx, r.baseURI, path, payload,
		r.cl, map[int]error{http.StatusNotFound: catalog.ErrNoSuchNamespace, http.StatusConflict: catalog.ErrTableAlreadyExists})
	if err != nil {
		return nil, err
	}

	if stagedCreate {
		meta, metaLoc, err := r.commitStagedCreate(ctx, identifier, ret.Metadata, cfg.StagedUpdates)
		if err != nil {
			return nil, err
		}
		ret.Metadata = meta
		ret.MetadataLoc = metaLoc
	}

	config := maps.Clone(r.props)
	maps.Copy(config, ret.Metadata.Properties())
	maps.Copy(config, ret.Config)
	scanPlanningConfig := maps.Clone(config)
	credsVended := len(ret.StorageCredentials) > 0
	maps.Copy(config, resolveStorageCredentials(ret.StorageCredentials, ret.MetadataLoc))

	return r.tableFromResponse(ctx, identifier, ret.Metadata, ret.MetadataLoc, config, scanPlanningConfig, credsVended)
}

// commitStagedCreate performs the second phase of a staged table
// creation by committing the table with an assert-create requirement.
// Changes are extracted from the phase 1 response metadata (which may
// have been normalized by the server), matching the Java implementation's
// createChanges approach. User-provided staged updates are appended
// after server-derived changes so that user intent takes precedence.
//
// If phase 2 fails, the server is left with an orphaned staged table
// (no automatic rollback).
func (r *Catalog) commitStagedCreate(ctx context.Context, identifier table.Identifier, meta table.Metadata, stagedUpdates []table.Update) (table.Metadata, string, error) {
	// Build updates from server-returned metadata, mirroring Java's
	// RESTSessionCatalog.createChanges.
	updates := make([]table.Update, 0, len(stagedUpdates)+10)
	updates = append(updates,
		table.NewAssignUUIDUpdate(meta.TableUUID()),
		table.NewUpgradeFormatVersionUpdate(meta.Version()),
		table.NewAddSchemaUpdate(meta.CurrentSchema()),
		table.NewSetCurrentSchemaUpdate(-1),
	)

	spec := meta.PartitionSpec()
	if !spec.IsUnpartitioned() {
		updates = append(updates, table.NewAddPartitionSpecUpdate(&spec, true))
	} else {
		updates = append(updates, table.NewAddPartitionSpecUpdate(iceberg.UnpartitionedSpec, true))
	}
	updates = append(updates, table.NewSetDefaultSpecUpdate(-1))

	order := meta.SortOrder()
	if !order.IsUnsorted() {
		updates = append(updates, table.NewAddSortOrderUpdate(&order))
	} else {
		updates = append(updates, table.NewAddSortOrderUpdate(&table.UnsortedSortOrder))
	}
	updates = append(updates, table.NewSetDefaultSortOrderUpdate(-1))

	if loc := meta.Location(); loc != "" {
		updates = append(updates, table.NewSetLocationUpdate(loc))
	}
	if props := meta.Properties(); len(props) > 0 {
		updates = append(updates, table.NewSetPropertiesUpdate(props))
	}

	// Append user-provided staged updates at the end.
	updates = append(updates, stagedUpdates...)

	requirements := []table.Requirement{table.AssertCreate()}

	return r.CommitTable(ctx, identifier, requirements, updates)
}

func (r *Catalog) CommitTable(ctx context.Context, ident table.Identifier, requirements []table.Requirement, updates []table.Update) (table.Metadata, string, error) {
	if err := r.endpoints.check(endpointUpdateTable); err != nil {
		return nil, "", err
	}

	ns, tblName, err := r.splitIdentForPath(ident)
	if err != nil {
		return nil, "", err
	}

	restIdentifier := identifier{
		Namespace: catalog.NamespaceFromIdent(ident),
		Name:      tblName,
	}

	type payload struct {
		Identifier   identifier          `json:"identifier"`
		Requirements []table.Requirement `json:"requirements"`
		Updates      []table.Update      `json:"updates"`
	}

	path, err := endpointUpdateTable.reqPath(ns, tblName)
	if err != nil {
		return nil, "", err
	}

	ret, err := doPost[payload, commitTableResponse](ctx, r.baseURI, path,
		payload{Identifier: restIdentifier, Requirements: requirements, Updates: updates}, r.cl,
		map[int]error{
			http.StatusNotFound:            catalog.ErrNoSuchTable,
			http.StatusConflict:            ErrCommitFailed,
			http.StatusInternalServerError: ErrCommitStateUnknown,
			http.StatusBadGateway:          ErrCommitStateUnknown,
			http.StatusServiceUnavailable:  ErrCommitStateUnknown,
			http.StatusGatewayTimeout:      ErrCommitStateUnknown,
		})
	if err != nil {
		return nil, "", err
	}
	metadata, err := r.parseCommitMetadata(ret.RawMetadata)
	if err != nil {
		return nil, "", err
	}

	return metadata, ret.MetadataLoc, nil
}

func (r *Catalog) parseCommitMetadata(raw json.RawMessage) (table.Metadata, error) {
	if r.snapshotMode == SnapshotModeRefs {
		return table.ParseMetadataBytesDeferredSnapshots(raw)
	}

	return table.ParseMetadataBytes(raw)
}

// CommitTransaction atomically commits changes to multiple tables in a
// single request. It implements [catalog.TransactionalCatalog].
//
// The server applies all changes or none (all-or-nothing). On success
// (204 No Content) the method returns nil. Callers must LoadTable
// individually to obtain updated metadata.
func (r *Catalog) CommitTransaction(ctx context.Context, commits []table.TableCommit) error {
	if len(commits) == 0 {
		return catalog.ErrEmptyCommitList
	}

	if err := r.endpoints.check(endpointCommitTransaction); err != nil {
		return err
	}

	type tableChange struct {
		Identifier   identifier          `json:"identifier"`
		Requirements []table.Requirement `json:"requirements"`
		Updates      []table.Update      `json:"updates"`
	}

	type payload struct {
		TableChanges []tableChange `json:"table-changes"`
	}

	changes := make([]tableChange, len(commits))
	for i, c := range commits {
		if len(c.Identifier) == 0 {
			return catalog.ErrMissingIdentifier
		}
		if err := catalog.ValidateTableIdentifier(c.Identifier); err != nil {
			return err
		}

		reqs := c.Requirements
		if reqs == nil {
			reqs = []table.Requirement{}
		}
		updates := c.Updates
		if updates == nil {
			updates = []table.Update{}
		}

		changes[i] = tableChange{
			Identifier: identifier{
				Namespace: catalog.NamespaceFromIdent(c.Identifier),
				Name:      catalog.TableNameFromIdent(c.Identifier),
			},
			Requirements: reqs,
			Updates:      updates,
		}
	}

	path, err := endpointCommitTransaction.reqPath()
	if err != nil {
		return err
	}

	_, err = doPost[payload, struct{}](
		ctx, r.baseURI, path,
		payload{TableChanges: changes}, r.cl,
		map[int]error{
			http.StatusNotFound:            catalog.ErrNoSuchTable,
			http.StatusConflict:            ErrCommitFailed,
			http.StatusInternalServerError: ErrCommitStateUnknown,
			http.StatusBadGateway:          ErrCommitStateUnknown,
			http.StatusServiceUnavailable:  ErrCommitStateUnknown,
			http.StatusGatewayTimeout:      ErrCommitStateUnknown,
		},
		allowNoContent(),
	)

	return err
}

func (r *Catalog) RegisterTable(ctx context.Context, identifier table.Identifier, metadataLoc string) (*table.Table, error) {
	if err := r.endpoints.check(endpointRegisterTable); err != nil {
		return nil, err
	}

	ns, tbl, err := r.splitIdentForPath(identifier)
	if err != nil {
		return nil, err
	}

	type payload struct {
		Name        string `json:"name"`
		MetadataLoc string `json:"metadata-location"`
	}

	// Resolve the reporter before the register POST mutates the server, for the
	// same reason as CreateTable: it is otherwise first built in the trailing
	// tableFromResponse, so an invalid reporter would turn a successful
	// registration into a reported failure. CachedReporter caches this. Resolved
	// from the client-supplied properties only (see the reporterProps field).
	if _, err := r.reporter.Get(r.reporterProps); err != nil {
		return nil, fmt.Errorf("failed to initialize metrics reporter: %w", err)
	}

	path, err := endpointRegisterTable.reqPath(ns)
	if err != nil {
		return nil, err
	}

	ret, err := doPost[payload, loadTableResponse](ctx, r.baseURI, path,
		payload{Name: tbl, MetadataLoc: metadataLoc}, r.cl, map[int]error{
			http.StatusNotFound: catalog.ErrNoSuchNamespace, http.StatusConflict: catalog.ErrTableAlreadyExists,
		})
	if err != nil {
		return nil, err
	}

	config := maps.Clone(r.props)
	maps.Copy(config, ret.Metadata.Properties())
	maps.Copy(config, ret.Config)
	scanPlanningConfig := maps.Clone(config)
	credsVended := len(ret.StorageCredentials) > 0
	maps.Copy(config, resolveStorageCredentials(ret.StorageCredentials, ret.MetadataLoc))

	return r.tableFromResponse(ctx, identifier, ret.Metadata, ret.MetadataLoc, config, scanPlanningConfig, credsVended)
}

// LoadTable loads a table from the catalog. It implements [catalog.Catalog].
// When snapshot-loading-mode is set to "refs" in the catalog properties, only
// snapshots referenced by a named branch or tag are included in the response.
// Callers that need to time-travel to an unreferenced snapshot should use
// snapshot-loading-mode "all" or omit the property entirely.
func (r *Catalog) LoadTable(ctx context.Context, identifier table.Identifier) (*table.Table, error) {
	return r.loadTableWithMode(ctx, identifier, r.snapshotMode)
}

func (r *Catalog) loadTableWithMode(ctx context.Context, identifier table.Identifier, mode SnapshotMode) (*table.Table, error) {
	if err := r.endpoints.check(endpointLoadTable); err != nil {
		return nil, err
	}

	ns, tbl, err := r.splitIdentForPath(identifier)
	if err != nil {
		return nil, err
	}

	path, err := endpointLoadTable.reqPath(ns, tbl)
	if err != nil {
		return nil, err
	}

	var opts []reqOption
	if mode != "" {
		opts = append(opts, withQueryParams(url.Values{"snapshots": {string(mode)}}))
	}

	ret, err := doGet[loadTableResponse](ctx, r.baseURI, path,
		r.cl, map[int]error{http.StatusNotFound: catalog.ErrNoSuchTable}, opts...)
	if err != nil {
		return nil, err
	}

	config := maps.Clone(r.props)
	maps.Copy(config, ret.Metadata.Properties())
	maps.Copy(config, ret.Config)
	scanPlanningConfig := maps.Clone(config)
	credsVended := len(ret.StorageCredentials) > 0
	maps.Copy(config, resolveStorageCredentials(ret.StorageCredentials, ret.MetadataLoc))

	return r.tableFromResponse(ctx, identifier, ret.Metadata, ret.MetadataLoc, config, scanPlanningConfig, credsVended)
}

func (r *Catalog) UpdateTable(ctx context.Context, ident table.Identifier, requirements []table.Requirement, updates []table.Update) (*table.Table, error) {
	if err := r.endpoints.check(endpointUpdateTable); err != nil {
		return nil, err
	}

	ns, tbl, err := r.splitIdentForPath(ident)
	if err != nil {
		return nil, err
	}

	restIdentifier := identifier{
		Namespace: catalog.NamespaceFromIdent(ident),
		Name:      tbl,
	}
	type payload struct {
		Identifier   identifier          `json:"identifier"`
		Requirements []table.Requirement `json:"requirements"`
		Updates      []table.Update      `json:"updates"`
	}
	path, err := endpointUpdateTable.reqPath(ns, tbl)
	if err != nil {
		return nil, err
	}

	ret, err := doPost[payload, commitTableResponse](ctx, r.baseURI, path,
		payload{Identifier: restIdentifier, Requirements: requirements, Updates: updates}, r.cl,
		map[int]error{
			http.StatusNotFound:            catalog.ErrNoSuchTable,
			http.StatusConflict:            ErrCommitFailed,
			http.StatusInternalServerError: ErrCommitStateUnknown,
			http.StatusBadGateway:          ErrCommitStateUnknown,
			http.StatusServiceUnavailable:  ErrCommitStateUnknown,
			http.StatusGatewayTimeout:      ErrCommitStateUnknown,
		})
	if err != nil {
		return nil, err
	}
	metadata, err := r.parseCommitMetadata(ret.RawMetadata)
	if err != nil {
		return nil, err
	}

	config := maps.Clone(r.props)
	maps.Copy(config, metadata.Properties())

	return r.tableFromResponse(ctx, ident, metadata, ret.MetadataLoc, config, config, false)
}

func (r *Catalog) DropTable(ctx context.Context, identifier table.Identifier) error {
	if err := r.endpoints.check(endpointDeleteTable); err != nil {
		return err
	}

	ns, tbl, err := r.splitIdentForPath(identifier)
	if err != nil {
		return err
	}

	path, err := endpointDeleteTable.reqPath(ns, tbl)
	if err != nil {
		return err
	}

	uri := r.baseURI.JoinPath(path...)
	v := url.Values{}
	v.Set("purgeRequested", "false")
	uri.RawQuery = v.Encode()

	_, err = doDelete[struct{}](ctx, uri, []string{}, r.cl,
		map[int]error{http.StatusNotFound: catalog.ErrNoSuchTable})

	return err
}

func (r *Catalog) PurgeTable(ctx context.Context, identifier table.Identifier) error {
	if err := r.endpoints.check(endpointDeleteTable); err != nil {
		return err
	}

	ns, tbl, err := r.splitIdentForPath(identifier)
	if err != nil {
		return err
	}

	path, err := endpointDeleteTable.reqPath(ns, tbl)
	if err != nil {
		return err
	}

	uri := r.baseURI.JoinPath(path...)
	v := url.Values{}
	v.Set("purgeRequested", "true")
	uri.RawQuery = v.Encode()

	_, err = doDelete[struct{}](ctx, uri, []string{}, r.cl,
		map[int]error{http.StatusNotFound: catalog.ErrNoSuchTable})

	return err
}

func (r *Catalog) RenameTable(ctx context.Context, from, to table.Identifier) (*table.Table, error) {
	if err := r.endpoints.check(endpointRenameTable); err != nil {
		return nil, err
	}
	if err := catalog.ValidateTableIdentifier(from); err != nil {
		return nil, err
	}
	if err := catalog.ValidateTableIdentifier(to); err != nil {
		return nil, err
	}

	type payload struct {
		Source      identifier `json:"source"`
		Destination identifier `json:"destination"`
	}
	src := identifier{
		Namespace: catalog.NamespaceFromIdent(from),
		Name:      catalog.TableNameFromIdent(from),
	}
	dst := identifier{
		Namespace: catalog.NamespaceFromIdent(to),
		Name:      catalog.TableNameFromIdent(to),
	}

	path, err := endpointRenameTable.reqPath()
	if err != nil {
		return nil, err
	}

	_, err = doPost[payload, any](ctx, r.baseURI, path, payload{Source: src, Destination: dst}, r.cl,
		map[int]error{http.StatusNotFound: catalog.ErrNoSuchTable}, allowNoContent())
	if err != nil {
		return nil, err
	}

	return r.LoadTable(ctx, to)
}

func (r *Catalog) CreateNamespace(ctx context.Context, namespace table.Identifier, props iceberg.Properties) error {
	if err := r.endpoints.check(endpointCreateNamespace); err != nil {
		return err
	}

	if err := r.checkValidNamespace(namespace); err != nil {
		return err
	}

	path, err := endpointCreateNamespace.reqPath()
	if err != nil {
		return err
	}

	_, err = doPost[map[string]any, struct{}](ctx, r.baseURI, path,
		map[string]any{"namespace": namespace, "properties": props}, r.cl, map[int]error{
			http.StatusNotFound: catalog.ErrNoSuchNamespace, http.StatusConflict: catalog.ErrNamespaceAlreadyExists,
		})

	return err
}

func (r *Catalog) DropNamespace(ctx context.Context, namespace table.Identifier) error {
	if err := r.endpoints.check(endpointDeleteNamespace); err != nil {
		return err
	}

	if err := r.checkValidNamespace(namespace); err != nil {
		return err
	}

	path, err := endpointDeleteNamespace.reqPath(r.encodeNamespace(namespace))
	if err != nil {
		return err
	}

	_, err = doDelete[struct{}](ctx, r.baseURI, path,
		r.cl, map[int]error{
			http.StatusNotFound: catalog.ErrNoSuchNamespace,
			http.StatusConflict: catalog.ErrNamespaceNotEmpty,
		})

	return err
}

// ListNamespaces lists namespaces under parent. If the server does not advertise
// the list-namespaces endpoint, it returns an empty slice rather than an error.
func (r *Catalog) ListNamespaces(ctx context.Context, parent table.Identifier) ([]table.Identifier, error) {
	var allNamespaces []table.Identifier
	pageSize := r.getPageSize(ctx)
	err := paginateIdentifiers(func(pageToken string) ([]table.Identifier, string, error) {
		return r.listNamespacesPage(ctx, parent, pageToken, pageSize)
	}, func(identifier table.Identifier) bool {
		allNamespaces = append(allNamespaces, identifier)

		return true
	})
	if err != nil {
		return nil, err
	}

	return allNamespaces, nil
}

func (r *Catalog) listNamespacesPage(ctx context.Context, parent table.Identifier, pageToken string, pageSize int) ([]table.Identifier, string, error) {
	if len(parent) != 0 {
		if err := r.checkValidNamespace(parent); err != nil {
			return nil, "", err
		}
	}

	// Unsupported listing yields an empty result rather than an error.
	if !r.endpoints.allowed(endpointListNamespaces) {
		return nil, "", nil
	}

	path, err := endpointListNamespaces.reqPath()
	if err != nil {
		return nil, "", err
	}

	uri := r.baseURI.JoinPath(path...)

	v := url.Values{}
	if len(parent) != 0 {
		v.Set("parent", r.namespaceToQueryParam(parent))
	}
	if pageSize > 0 {
		v.Set("pageSize", strconv.Itoa(pageSize))
	}
	if pageToken != "" {
		v.Set("pageToken", pageToken)
	}
	if len(v) > 0 {
		uri.RawQuery = v.Encode()
	}

	type rsptype struct {
		Namespaces    []table.Identifier `json:"namespaces"`
		NextPageToken string             `json:"next-page-token,omitempty"`
	}

	rsp, err := doGet[rsptype](ctx, uri, []string{}, r.cl, map[int]error{http.StatusNotFound: catalog.ErrNoSuchNamespace})
	if err != nil {
		return nil, "", err
	}

	return rsp.Namespaces, rsp.NextPageToken, nil
}

func (r *Catalog) LoadNamespaceProperties(ctx context.Context, namespace table.Identifier) (iceberg.Properties, error) {
	if err := r.endpoints.check(endpointLoadNamespace); err != nil {
		return nil, err
	}

	if err := r.checkValidNamespace(namespace); err != nil {
		return nil, err
	}

	type nsresponse struct {
		Namespace table.Identifier   `json:"namespace"`
		Props     iceberg.Properties `json:"properties"`
	}

	path, err := endpointLoadNamespace.reqPath(r.encodeNamespace(namespace))
	if err != nil {
		return nil, err
	}

	rsp, err := doGet[nsresponse](ctx, r.baseURI, path,
		r.cl, map[int]error{http.StatusNotFound: catalog.ErrNoSuchNamespace})
	if err != nil {
		return nil, err
	}

	return rsp.Props, nil
}

func (r *Catalog) UpdateNamespaceProperties(ctx context.Context, namespace table.Identifier,
	removals []string, updates iceberg.Properties,
) (catalog.PropertiesUpdateSummary, error) {
	if err := r.endpoints.check(endpointUpdateNamespace); err != nil {
		return catalog.PropertiesUpdateSummary{}, err
	}

	if err := r.checkValidNamespace(namespace); err != nil {
		return catalog.PropertiesUpdateSummary{}, err
	}

	type payload struct {
		Remove  []string           `json:"removals"`
		Updates iceberg.Properties `json:"updates"`
	}

	ns := r.encodeNamespace(namespace)

	path, err := endpointUpdateNamespace.reqPath(ns)
	if err != nil {
		return catalog.PropertiesUpdateSummary{}, err
	}

	return doPost[payload, catalog.PropertiesUpdateSummary](ctx, r.baseURI, path,
		payload{Remove: removals, Updates: updates}, r.cl, map[int]error{http.StatusNotFound: catalog.ErrNoSuchNamespace})
}

func (r *Catalog) CheckNamespaceExists(ctx context.Context, namespace table.Identifier) (bool, error) {
	if err := r.checkValidNamespace(namespace); err != nil {
		return false, err
	}

	// No HEAD endpoint: fall back to GET (load). If load is unsupported too, its
	// ErrEndpointNotSupported surfaces rather than a bogus "not found".
	if !r.endpoints.contains(endpointNamespaceExists) {
		_, err := r.LoadNamespaceProperties(ctx, namespace)
		if err != nil {
			if errors.Is(err, catalog.ErrNoSuchNamespace) {
				return false, nil
			}

			return false, err
		}

		return true, nil
	}

	path, err := endpointNamespaceExists.reqPath(r.encodeNamespace(namespace))
	if err != nil {
		return false, err
	}

	err = doHead(ctx, r.baseURI, path,
		r.cl, map[int]error{http.StatusNotFound: catalog.ErrNoSuchNamespace})
	if err != nil {
		if errors.Is(err, catalog.ErrNoSuchNamespace) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (r *Catalog) CheckTableExists(ctx context.Context, identifier table.Identifier) (bool, error) {
	ns, tbl, err := r.splitIdentForPath(identifier)
	if err != nil {
		return false, err
	}

	// No HEAD endpoint: fall back to GET (load). If load is unsupported too, its
	// ErrEndpointNotSupported surfaces rather than a bogus "not found".
	if !r.endpoints.contains(endpointTableExists) {
		_, err := r.LoadTable(ctx, identifier)
		if err != nil {
			if errors.Is(err, catalog.ErrNoSuchTable) {
				return false, nil
			}

			return false, err
		}

		return true, nil
	}

	path, err := endpointTableExists.reqPath(ns, tbl)
	if err != nil {
		return false, err
	}

	err = doHead(ctx, r.baseURI, path,
		r.cl, map[int]error{http.StatusNotFound: catalog.ErrNoSuchTable})
	if err != nil {
		if errors.Is(err, catalog.ErrNoSuchTable) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// ListViews lists the views in a namespace. If the server does not advertise the
// list-views endpoint, it yields no views rather than an error.
func (r *Catalog) ListViews(ctx context.Context, namespace table.Identifier) iter.Seq2[table.Identifier, error] {
	return func(yield func(table.Identifier, error) bool) {
		pageSize := r.getPageSize(ctx)
		err := paginateIdentifiers(func(pageToken string) ([]table.Identifier, string, error) {
			return r.listViewsPage(ctx, namespace, pageToken, pageSize)
		}, func(identifier table.Identifier) bool {
			return yield(identifier, nil)
		})
		if err != nil {
			yield(table.Identifier{}, err)
		}
	}
}

func (r *Catalog) listViewsPage(ctx context.Context, namespace table.Identifier, pageToken string, pageSize int) ([]table.Identifier, string, error) {
	if err := r.checkValidNamespace(namespace); err != nil {
		return nil, "", err
	}
	// Unsupported listing yields an empty result rather than an error.
	if !r.endpoints.allowed(endpointListViews) {
		return nil, "", nil
	}
	ns := r.encodeNamespace(namespace)
	path, err := endpointListViews.reqPath(ns)
	if err != nil {
		return nil, "", err
	}
	uri := r.baseURI.JoinPath(path...)

	v := url.Values{}
	if pageSize > 0 {
		v.Set("pageSize", strconv.Itoa(pageSize))
	}
	if pageToken != "" {
		v.Set("pageToken", pageToken)
	}

	uri.RawQuery = v.Encode()
	type resp struct {
		Identifiers   []identifier `json:"identifiers"`
		NextPageToken string       `json:"next-page-token,omitempty"`
	}

	rsp, err := doGet[resp](ctx, uri, []string{}, r.cl, map[int]error{http.StatusNotFound: catalog.ErrNoSuchNamespace})
	if err != nil {
		return nil, "", err
	}

	out := make([]table.Identifier, len(rsp.Identifiers))
	for i, id := range rsp.Identifiers {
		out[i] = append(id.Namespace, id.Name)
	}

	return out, rsp.NextPageToken, nil
}

func (r *Catalog) getPageSize(ctx context.Context) int {
	if pageSize, ok := ctx.Value(pageSizeKey).(int); ok {
		return pageSize
	}

	return defaultPageSize
}

func (r *Catalog) SetPageSize(ctx context.Context, sz int) context.Context {
	return context.WithValue(ctx, pageSizeKey, sz)
}

func (r *Catalog) DropView(ctx context.Context, identifier table.Identifier) error {
	if err := r.endpoints.check(endpointDeleteView); err != nil {
		return err
	}

	ns, view, err := r.splitViewIdentForPath(identifier)
	if err != nil {
		return err
	}

	path, err := endpointDeleteView.reqPath(ns, view)
	if err != nil {
		return err
	}

	_, err = doDelete[struct{}](ctx, r.baseURI, path, r.cl,
		map[int]error{http.StatusNotFound: catalog.ErrNoSuchView})

	return err
}

func (r *Catalog) CheckViewExists(ctx context.Context, identifier table.Identifier) (bool, error) {
	ns, view, err := r.splitViewIdentForPath(identifier)
	if err != nil {
		return false, err
	}

	// No HEAD endpoint: fall back to GET (load). If load is unsupported too, its
	// ErrEndpointNotSupported surfaces rather than a bogus "not found".
	if !r.endpoints.contains(endpointViewExists) {
		_, err := r.LoadView(ctx, identifier)
		if err != nil {
			if errors.Is(err, catalog.ErrNoSuchView) {
				return false, nil
			}

			return false, err
		}

		return true, nil
	}

	path, err := endpointViewExists.reqPath(ns, view)
	if err != nil {
		return false, err
	}

	err = doHead(ctx, r.baseURI, path,
		r.cl, map[int]error{http.StatusNotFound: catalog.ErrNoSuchView})
	if err != nil {
		if errors.Is(err, catalog.ErrNoSuchView) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

type createViewRequest struct {
	Name        string             `json:"name"`
	Schema      *iceberg.Schema    `json:"schema"`
	Location    string             `json:"location,omitempty"`
	Props       iceberg.Properties `json:"properties"`
	ViewVersion *view.Version      `json:"view-version"`
}

type viewResponse struct {
	MetadataLoc string             `json:"metadata-location"`
	RawMetadata json.RawMessage    `json:"metadata"`
	Config      iceberg.Properties `json:"config"`
	Metadata    view.Metadata      `json:"-"`
}

func (v *viewResponse) UnmarshalJSON(b []byte) (err error) {
	type Alias viewResponse
	if err = json.Unmarshal(b, (*Alias)(v)); err != nil {
		return err
	}

	v.Metadata, err = view.ParseMetadataBytes(v.RawMetadata)

	return
}

// CreateView creates a new view in the catalog.
func (r *Catalog) CreateView(ctx context.Context, identifier table.Identifier, version *view.Version, schema *iceberg.Schema, opts ...catalog.CreateViewOpt) (*view.View, error) {
	if err := r.endpoints.check(endpointCreateView); err != nil {
		return nil, err
	}
	if version == nil {
		return nil, fmt.Errorf("%w: view version cannot be nil", iceberg.ErrInvalidArgument)
	}

	ns, viewName, err := r.splitViewIdentForPath(identifier)
	if err != nil {
		return nil, err
	}

	if opts == nil {
		opts = []catalog.CreateViewOpt{}
	}
	cfg := catalog.NewCreateViewCfg()
	for _, opt := range opts {
		opt(&cfg)
	}

	freshSchema, err := iceberg.AssignFreshSchemaIDs(schema, nil)
	if err != nil {
		return nil, err
	}

	// Shallow copy for overriding certain attrs
	newVersion := *version
	version = &newVersion

	// Enforce starting ID of 1
	version.VersionID = 1

	// Set default catalog unless set by caller
	if len(version.DefaultCatalog) == 0 {
		version.DefaultCatalog = r.name
	}

	payload := createViewRequest{
		Name:        viewName,
		Location:    cfg.Location,
		Schema:      freshSchema,
		Props:       cfg.Properties,
		ViewVersion: version,
	}

	path, err := endpointCreateView.reqPath(ns)
	if err != nil {
		return nil, err
	}

	ret, err := doPost[createViewRequest, viewResponse](ctx, r.baseURI, path, payload,
		r.cl, map[int]error{
			http.StatusNotFound: catalog.ErrNoSuchNamespace,
			http.StatusConflict: catalog.ErrViewAlreadyExists,
		})
	if err != nil {
		return nil, err
	}

	return view.New(identifier, ret.Metadata, ret.MetadataLoc), nil
}

// UpdateView updates a view in the catalog.
func (r *Catalog) UpdateView(ctx context.Context, ident table.Identifier, requirements []view.Requirement, updates []view.Update) (*view.View, error) {
	if err := r.endpoints.check(endpointUpdateView); err != nil {
		return nil, err
	}

	ns, viewName, err := r.splitViewIdentForPath(ident)
	if err != nil {
		return nil, err
	}

	restIdentifier := identifier{
		Namespace: catalog.NamespaceFromIdent(ident),
		Name:      viewName,
	}
	type payload struct {
		Identifier   identifier         `json:"identifier"`
		Requirements []view.Requirement `json:"requirements"`
		Updates      []view.Update      `json:"updates"`
	}
	path, err := endpointUpdateView.reqPath(ns, viewName)
	if err != nil {
		return nil, err
	}

	ret, err := doPost[payload, viewResponse](ctx, r.baseURI, path,
		payload{Identifier: restIdentifier, Requirements: requirements, Updates: updates}, r.cl,
		map[int]error{http.StatusNotFound: catalog.ErrNoSuchView, http.StatusConflict: ErrCommitFailed})
	if err != nil {
		return nil, err
	}

	return view.New(ident, ret.Metadata, ret.MetadataLoc), nil
}

// loadViewResponse contains the response from loading a view
type loadViewResponse struct {
	MetadataLoc string             `json:"metadata-location"`
	RawMetadata json.RawMessage    `json:"metadata"`
	Config      iceberg.Properties `json:"config"`
	Labels      *iceberg.Labels    `json:"labels,omitempty"`
}

// RegisterView registers an existing view in the catalog using its metadata file location.
// The metadata file must already be accessible to the catalog. This is the view equivalent
// of RegisterTable, using the REST endpoint POST /namespaces/{ns}/register-view defined in
// the Iceberg REST catalog specification.
func (r *Catalog) RegisterView(ctx context.Context, identifier table.Identifier, metadataLoc string) (*view.View, error) {
	if err := r.endpoints.check(endpointRegisterView); err != nil {
		return nil, err
	}

	ns, v, err := r.splitViewIdentForPath(identifier)
	if err != nil {
		return nil, err
	}

	type payload struct {
		Name        string `json:"name"`
		MetadataLoc string `json:"metadata-location"`
	}

	path, err := endpointRegisterView.reqPath(ns)
	if err != nil {
		return nil, err
	}

	rsp, err := doPost[payload, loadViewResponse](ctx, r.baseURI, path,
		payload{Name: v, MetadataLoc: metadataLoc}, r.cl, map[int]error{
			http.StatusNotFound: catalog.ErrNoSuchNamespace, http.StatusConflict: catalog.ErrViewAlreadyExists,
		})
	if err != nil {
		return nil, err
	}

	metadata, err := view.ParseMetadataBytes(rsp.RawMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to parse view metadata: %w", err)
	}

	return view.New(identifier, metadata, rsp.MetadataLoc), nil
}

// LoadView loads a view from the catalog.
func (r *Catalog) LoadView(ctx context.Context, identifier table.Identifier) (*view.View, error) {
	if err := r.endpoints.check(endpointLoadView); err != nil {
		return nil, err
	}

	ns, v, err := r.splitViewIdentForPath(identifier)
	if err != nil {
		return nil, err
	}

	path, err := endpointLoadView.reqPath(ns, v)
	if err != nil {
		return nil, err
	}

	rsp, err := doGet[loadViewResponse](ctx, r.baseURI, path,
		r.cl, map[int]error{
			http.StatusNotFound: catalog.ErrNoSuchView,
		})
	if err != nil {
		return nil, err
	}

	metadata, err := view.ParseMetadataBytes(rsp.RawMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to parse view metadata: %w", err)
	}

	return view.New(identifier, metadata, rsp.MetadataLoc), nil
}

func (r *Catalog) RenameView(ctx context.Context, from, to table.Identifier) (*view.View, error) {
	if err := r.endpoints.check(endpointRenameView); err != nil {
		return nil, err
	}
	if err := catalog.ValidateViewIdentifier(from); err != nil {
		return nil, err
	}
	if err := catalog.ValidateViewIdentifier(to); err != nil {
		return nil, err
	}

	type payload struct {
		Source      identifier `json:"source"`
		Destination identifier `json:"destination"`
	}
	src := identifier{
		Namespace: catalog.NamespaceFromIdent(from),
		Name:      catalog.TableNameFromIdent(from),
	}
	dst := identifier{
		Namespace: catalog.NamespaceFromIdent(to),
		Name:      catalog.TableNameFromIdent(to),
	}

	path, err := endpointRenameView.reqPath()
	if err != nil {
		return nil, err
	}

	_, err = doPost[payload, any](ctx, r.baseURI, path, payload{Source: src, Destination: dst}, r.cl,
		map[int]error{
			http.StatusNotFound: catalog.ErrNoSuchView,
			http.StatusConflict: catalog.ErrViewAlreadyExists,
		}, allowNoContent())
	if err != nil {
		return nil, err
	}

	return r.LoadView(ctx, to)
}

// FunctionCatalog is an optional interface for catalogs that support the
// function (SQL UDF) read endpoints defined by the REST spec: list and load.
// The function endpoints are not part of the spec's assumed default endpoint
// set, so they are only used when the server advertises them: unsupported
// loads fail with ErrEndpointNotSupported and unsupported listings yield no
// results. Callers holding a catalog.Catalog can check for this capability
// via a type assertion:
//
//	if fc, ok := cat.(rest.FunctionCatalog); ok {
//	    fn, err := fc.LoadFunction(ctx, ident)
//	}
type FunctionCatalog interface {
	// ListFunctions returns the function identifiers under a namespace,
	// with the returned identifiers containing the information required
	// to load the function via this catalog.
	ListFunctions(ctx context.Context, namespace table.Identifier) iter.Seq2[table.Identifier, error]
	// LoadFunction loads a function from the catalog. All overloaded
	// definitions are included in the single metadata response.
	LoadFunction(ctx context.Context, identifier table.Identifier) (*udf.UDF, error)
	// CheckFunctionExists returns if the function exists. The REST spec
	// defines no HEAD endpoint for functions, so existence is checked by
	// loading the function.
	CheckFunctionExists(ctx context.Context, identifier table.Identifier) (bool, error)
}

// ListFunctions returns the function (SQL UDF) identifiers under a
// namespace, with the returned identifiers containing the information
// required to load the function via this catalog. The function endpoints
// are not part of the spec's assumed default endpoint set; if the server
// does not advertise the list-functions endpoint, it yields no functions
// rather than an error.
func (r *Catalog) ListFunctions(ctx context.Context, namespace table.Identifier) iter.Seq2[table.Identifier, error] {
	return func(yield func(table.Identifier, error) bool) {
		pageSize := r.getPageSize(ctx)
		err := paginateIdentifiers(func(pageToken string) ([]table.Identifier, string, error) {
			return r.listFunctionsPage(ctx, namespace, pageToken, pageSize)
		}, func(identifier table.Identifier) bool {
			return yield(identifier, nil)
		})
		if err != nil {
			yield(table.Identifier{}, err)
		}
	}
}

func (r *Catalog) listFunctionsPage(ctx context.Context, namespace table.Identifier, pageToken string, pageSize int) ([]table.Identifier, string, error) {
	if err := r.checkValidNamespace(namespace); err != nil {
		return nil, "", err
	}
	// Unsupported listing yields an empty result rather than an error.
	if !r.endpoints.allowed(endpointListFunctions) {
		return nil, "", nil
	}
	ns := r.encodeNamespace(namespace)
	path, err := endpointListFunctions.reqPath(ns)
	if err != nil {
		return nil, "", err
	}
	uri := r.baseURI.JoinPath(path...)

	v := url.Values{}
	if pageSize > 0 {
		v.Set("pageSize", strconv.Itoa(pageSize))
	}
	if pageToken != "" {
		v.Set("pageToken", pageToken)
	}

	uri.RawQuery = v.Encode()
	// Function identifiers are CatalogObjectIdentifier values: ordered
	// hierarchy levels (the namespace levels followed by the function
	// name), not the {namespace, name} object tables and views use.
	type resp struct {
		Identifiers   [][]string `json:"identifiers"`
		NextPageToken string     `json:"next-page-token,omitempty"`
	}

	rsp, err := doGet[resp](ctx, uri, []string{}, r.cl, map[int]error{http.StatusNotFound: catalog.ErrNoSuchNamespace})
	if err != nil {
		return nil, "", err
	}

	out := make([]table.Identifier, len(rsp.Identifiers))
	for i, levels := range rsp.Identifiers {
		out[i] = table.Identifier(levels)
	}

	return out, rsp.NextPageToken, nil
}

// loadFunctionResponse contains the response from loading a function.
type loadFunctionResponse struct {
	MetadataLoc string          `json:"metadata-location"`
	RawMetadata json.RawMessage `json:"metadata"`
}

// LoadFunction loads a function (SQL UDF) from the catalog. All overloaded
// definitions are included in the single metadata response. The function
// endpoints are not part of the spec's assumed default endpoint set; if the
// server does not advertise the load-function endpoint, LoadFunction fails
// with ErrEndpointNotSupported.
func (r *Catalog) LoadFunction(ctx context.Context, identifier table.Identifier) (*udf.UDF, error) {
	if err := r.endpoints.check(endpointLoadFunction); err != nil {
		return nil, err
	}

	ns, fn, err := r.splitFunctionIdentForPath(identifier)
	if err != nil {
		return nil, err
	}

	return r.loadFunction(ctx, identifier, ns, fn)
}

// loadFunction fetches and parses a function after endpoint negotiation and
// identifier validation, so its ErrNoSuchFunction is always server-reported.
func (r *Catalog) loadFunction(ctx context.Context, identifier table.Identifier, ns, fn string) (*udf.UDF, error) {
	path, err := endpointLoadFunction.reqPath(ns, fn)
	if err != nil {
		return nil, err
	}

	rsp, err := doGet[loadFunctionResponse](ctx, r.baseURI, path,
		r.cl, map[int]error{
			http.StatusNotFound: catalog.ErrNoSuchFunction,
		})
	if err != nil {
		// The spec's load 404 covers both a missing namespace and a missing
		// function; discriminate on the error type so a missing namespace is
		// not reported as a missing function.
		var errRsp errorResponse
		if errors.As(err, &errRsp) && errRsp.Type == "NoSuchNamespaceException" {
			errRsp.wrapping = catalog.ErrNoSuchNamespace

			return nil, errRsp
		}

		return nil, err
	}

	// The spec marks metadata as required; guard so a buggy server yields a
	// clear error instead of a JSON parsing artifact.
	if len(rsp.RawMetadata) == 0 {
		return nil, fmt.Errorf("%w: load function response is missing metadata", ErrRESTError)
	}

	metadata, err := udf.ParseMetadataBytes(rsp.RawMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to parse function metadata: %w", err)
	}

	return udf.New(identifier, metadata, rsp.MetadataLoc), nil
}

// CheckFunctionExists returns if the function exists. The REST spec defines
// no HEAD endpoint for functions, so existence is checked by loading the
// function; if loading is unsupported, ErrEndpointNotSupported surfaces
// rather than a bogus "not found". The identifier is validated once here, so
// an invalid one surfaces as an error and only a server-reported "not found"
// becomes (false, nil), matching the table and view existence checks.
func (r *Catalog) CheckFunctionExists(ctx context.Context, identifier table.Identifier) (bool, error) {
	if err := r.endpoints.check(endpointLoadFunction); err != nil {
		return false, err
	}

	ns, fn, err := r.splitFunctionIdentForPath(identifier)
	if err != nil {
		return false, err
	}

	if _, err := r.loadFunction(ctx, identifier, ns, fn); err != nil {
		if errors.Is(err, catalog.ErrNoSuchFunction) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}
