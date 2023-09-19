package httpdriver

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/dateutil"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"golang.org/x/mod/semver"
)

var (
	dialer = &net.Dialer{
		KeepAlive: 15 * time.Second,
	}

	DefaultExecutor = &executor{
		Client: &http.Client{
			Timeout:       consts.MaxFunctionTimeout,
			CheckRedirect: CheckRedirect,
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				DialContext:           dialer.DialContext,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          0,
				IdleConnTimeout:       0,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				// New, ensuring that services can take their time before
				// responding with headers as they process long running
				// kjobs.
				ResponseHeaderTimeout: consts.MaxFunctionTimeout,
			},
		},
	}

	ErrEmptyResponse = fmt.Errorf("no response data")

	ErrNoRetryAfter = fmt.Errorf("no retry after present")
)

func CheckRedirect(req *http.Request, via []*http.Request) (err error) {
	if len(via) > 10 {
		return fmt.Errorf("stopped after 10 redirects")
	}

	// If we're redirected we want to ensure that we retain the HTTP method.
	req.Method = via[0].Method
	req.Body, err = via[0].GetBody()
	if err != nil {
		return err
	}
	req.ContentLength = via[0].ContentLength
	req.Header = via[0].Header
	return nil
}

func Execute(ctx context.Context, s state.State, item queue.Item, edge inngest.Edge, step inngest.Step, idx, attempt int) (*state.DriverResponse, error) {
	return DefaultExecutor.Execute(ctx, s, item, edge, step, idx, attempt)
}

type executor struct {
	Client     *http.Client
	signingKey []byte
}

// RuntimeType fulfiils the inngest.Runtime interface.
func (e executor) RuntimeType() string {
	return "http"
}

// Sign signs the body with a private key, ensuring that HTTP handlers can verify
// that the request comes from us.
func Sign(ctx context.Context, key, body []byte) string {
	if key == nil {
		return ""
	}

	now := time.Now().Unix()
	mac := hmac.New(sha256.New, key)

	_, _ = mac.Write(body)
	// Write the timestamp as a unix timestamp to the hmac to prevent
	// timing attacks.
	_, _ = mac.Write([]byte(fmt.Sprintf("%d", now)))

	sig := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("t=%d&s=%s", now, sig)
}

func ParseGenerator(ctx context.Context, byt []byte) ([]*state.GeneratorOpcode, error) {
	// When we return a 206, we always expect that this is
	// a generator function.  Users SHOULD NOT return a 206
	// in any other circumstance.

	if len(byt) == 0 {
		return nil, ErrEmptyResponse
	}

	// Is this a slice of opcodes or a single opcode?  The SDK can return both:
	// parallelism was added as an incremental improvement.  It would have been nice
	// to always return an array and we can enfore this as an SDK requirement in V1+
	switch byt[0] {
	// 0.x.x SDKs return a single opcode.
	case '{':
		gen := &state.GeneratorOpcode{}
		if err := json.Unmarshal(byt, gen); err != nil {
			return nil, fmt.Errorf("error reading generator opcode response: %w", err)
		}
		return []*state.GeneratorOpcode{gen}, nil
	// 1.x.x+ SDKs return an array of opcodes.
	case '[':
		gen := []*state.GeneratorOpcode{}
		if err := json.Unmarshal(byt, &gen); err != nil {
			return nil, fmt.Errorf("error reading generator opcode response: %w", err)
		}
		// Normalize the response to always return at least an empty op code in the
		// array. With this, a non-generator is represented as an empty array.
		if len(gen) == 0 {
			return []*state.GeneratorOpcode{
				{Op: enums.OpcodeNone},
			}, nil
		}
		return gen, nil
	}

	// Finally, if the length of resp.Generator == 0 then this is implicitly an enums.OpcodeNone
	// step.  This is added to reduce bandwidth across many calls.
	return []*state.GeneratorOpcode{
		{Op: enums.OpcodeNone},
	}, nil
}

func (e executor) Execute(ctx context.Context, s state.State, item queue.Item, edge inngest.Edge, step inngest.Step, idx, attempt int) (*state.DriverResponse, error) {
	uri, err := url.Parse(step.URI)
	if err != nil || (uri.Scheme != "http" && uri.Scheme != "https") {
		return nil, fmt.Errorf("Unable to use HTTP executor for non-HTTP runtime")
	}

	input, err := driver.MarshalV1(ctx, s, step, idx, "", attempt)
	if err != nil {
		return nil, err
	}

	// If we have a generator step name, ensure we add the step ID parameter
	values, _ := url.ParseQuery(uri.RawQuery)
	if edge.IncomingGeneratorStep != "" {
		values.Set("stepId", edge.IncomingGeneratorStep)
		uri.RawQuery = values.Encode()
	} else {
		values.Set("stepId", edge.Incoming)
		uri.RawQuery = values.Encode()
	}

	resp, err := e.do(ctx, uri.String(), input)
	if err != nil {
		return nil, err
	}

	if resp.statusCode == 206 {
		// This is a generator-based function returning opcodes.
		dr := &state.DriverResponse{
			Step:       step,
			Duration:   resp.duration,
			OutputSize: len(resp.body),
			NoRetry:    resp.noRetry,
			RetryAt:    resp.retryAt,
		}
		dr.Generator, err = ParseGenerator(ctx, resp.body)
		if err != nil {
			return nil, err
		}
		return dr, nil
	}

	var body interface{}
	body = json.RawMessage(resp.body)
	if len(resp.body) > 0 {
		// Is the response valid JSON?  If so, ensure that we don't re-marshal the
		// JSON string.
		respjson := map[string]interface{}{}
		if err := json.Unmarshal(resp.body, &respjson); err == nil {
			body = respjson
		} else {
			// This isn't a map, so check the first character for json encoding.  If this isn't
			// a string or array, then the body must be treated as text.
			//
			// This is a stop-gap safety check to see if SDKs respond with text that's not JSON,
			// in the case of an internal issue or a host processing error we can't control.
			if resp.body[0] != '[' && resp.body[0] != '"' {
				body = string(resp.body)
			}
		}
	} else {
		body = nil
	}

	// Add an error to driver.Response if the status code isn't 2XX.
	err = nil
	if resp.statusCode < 200 || resp.statusCode > 299 {
		err = fmt.Errorf("invalid status code: %d", resp.statusCode)
	}

	var errstr *string
	if err != nil {
		str := err.Error()
		errstr = &str
	}

	return &state.DriverResponse{
		Step: step,
		Output: map[string]interface{}{
			"status": resp.statusCode,
			"body":   body,
		},
		Err:            errstr,
		Duration:       resp.duration,
		OutputSize:     len(resp.body),
		NoRetry:        resp.noRetry,
		RetryAt:        resp.retryAt,
		RequestVersion: resp.requestVersion,
	}, nil
}

type response struct {
	body           []byte
	statusCode     int
	duration       time.Duration
	requestVersion int
	// retryAt is the time to retry this step at, on failure, if specified in the
	// Retry-After headers, or X-Retry-After.
	//
	// This adheres to the HTTP spec; we support both seconds and times in this header.
	retryAt *time.Time
	noRetry bool
}

func (e executor) do(ctx context.Context, url string, input []byte) (*response, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(input))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")

	if len(e.signingKey) > 0 {
		req.Header.Add("X-Inngest-Signature", Sign(ctx, e.signingKey, input))
	}
	pre := time.Now()
	resp, err := e.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()
	dur := time.Since(pre)
	byt, err := io.ReadAll(io.LimitReader(resp.Body, consts.MaxBodySize))
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	hv, _ := strconv.Atoi(resp.Header.Get("x-inngest-req-version"))

	var retryAt *time.Time
	if after := resp.Header.Get("retry-after"); after != "" {
		if at, err := ParseRetry(after); err == nil {
			retryAt = &at
		}
	}

	// If the responding status code is 201 Created, the response has been
	// streamed back to us. In this case, the response body will be namespaced
	// under the "body" key, and the status code will be namespaced under the
	// "status" key.
	//
	// Only SDK versions that include the status in the body are expected to
	// send a 201 status code and namespace in this way, so failing to parse
	// here is an error.
	if resp.StatusCode != 201 {
		return &response{
			body:           byt,
			statusCode:     resp.StatusCode,
			duration:       dur,
			requestVersion: hv,
			retryAt:        retryAt,
			noRetry:        !shouldRetry(resp.StatusCode, resp),
		}, nil
	}

	stream, err := ParseStream(byt)
	if err != nil {
		return nil, err
	}
	if stream.RetryAt != nil {
		if at, err := ParseRetry(*stream.RetryAt); err == nil {
			retryAt = &at
		}
	}

	return &response{
		body:           stream.Body,
		statusCode:     stream.StatusCode,
		duration:       dur,
		retryAt:        retryAt,
		noRetry:        stream.NoRetry,
		requestVersion: hv,
	}, nil

}

func ParseStream(resp []byte) (*StreamResponse, error) {
	body := &StreamResponse{}
	if err := json.Unmarshal(resp, &body); err != nil {
		return nil, fmt.Errorf("error reading response body to check for status code: %w", err)
	}
	// Check to see if the body is double-encoded.
	if len(body.Body) > 0 && body.Body[0] == '"' && body.Body[len(body.Body)-1] == '"' {
		var str string
		if err := json.Unmarshal(body.Body, &str); err == nil {
			body.Body = []byte(str)
		}
	}
	return body, nil
}

type StreamResponse struct {
	StatusCode int               `json:"status"`
	Body       json.RawMessage   `json:"body"`
	RetryAt    *string           `json:"retryAt"`
	NoRetry    bool              `json:"noRetry"`
	Headers    map[string]string `json:"headers"`
}

// ParseRetry attempts to parse the retry-after header value.  It first checks to see
// if we have a reasonably sized second value (<= weeks), then parses the value as unix
// seconds.
//
// It falls back to parsing value in multiple formats: RFC3339, RFC1123, etc.
//
// This clips time within the minimums and maximums specified within consts.
func ParseRetry(retry string) (time.Time, error) {
	at, err := parseRetry(retry)
	if err != nil {
		return at, err
	}

	now := time.Now().UTC().Truncate(time.Second)

	dur := time.Until(at)
	if dur > consts.MaxRetryDuration {
		return now.Add(consts.MaxRetryDuration), nil
	}
	if dur < consts.MinRetryDuration {
		return now.Add(consts.MinRetryDuration), nil
	}
	return at, nil
}

func parseRetry(retry string) (time.Time, error) {
	if retry == "" {
		return time.Time{}, ErrNoRetryAfter
	}
	if len(retry) <= 7 {
		// Assume this is an int;  no dates can be <= 7 characters.
		secs, _ := strconv.Atoi(retry)
		if secs > 0 {
			return time.Now().UTC().Truncate(time.Second).Add(time.Second * time.Duration(secs)), nil
		}
	}
	return dateutil.ParseString(retry)
}

// shouldRetry determines if a request should be retried based on the response
// status code and headers.
//
// This is a best-effort attempt to determine if a request should be retried; we
// fall back to retrying if the request doesn't give us a firm answer.
func shouldRetry(status int, resp *http.Response) bool {
	noRetryHeader := resp.Header.Get("x-inngest-no-retry")
	// Always obey the no-retry header if it's set.
	if noRetryHeader != "" {
		return noRetryHeader != "true"
	}

	// In the absence of a no-retry header, this is only a no-retry response if
	// the status code is 4XX.
	if status < 400 || status > 499 {
		return true
	}

	// e.g. inngest-js:v1.2.3-beta.5
	versionHeader := strings.Split(resp.Header.Get("x-inngest-sdk"), ":")
	if len(versionHeader) != 2 {
		// Unexpected version string; we can't determine if this is a
		// no-retry, so we'll assume we should retry.
		return true
	}

	sdkLang := versionHeader[0]
	sdkVersion := versionHeader[1]

	if !semver.IsValid(sdkVersion) {
		// Unexpected version string; we can't determine if this is a
		// no-retry, so we'll assume we should retry.
		return true
	}

	// If we're here, we're assessing a 4XX response with no
	// `x-inngest-no-retry` header. We'll determine if this is a no-retry based
	// on the SDK version.
	if sdkLang == "inngest-js" {
		switch {
		// 4XX should not be retried if <v2.4.1
		case semver.Major(sdkVersion) == "v2" && semver.Compare(sdkVersion, "v2.4.1") == -1:
			return false
		// 4XX should not be retried if <v1.10.1
		case semver.Major(sdkVersion) == "v1" && semver.Compare(sdkVersion, "v1.10.1") == -1:
			return false
		}
	}

	return true
}
