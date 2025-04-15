package httpdriver

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"golang.org/x/mod/semver"
)

const (
	headerSDK            = "x-inngest-sdk"
	headerRequestVersion = "x-inngest-req-version"
	headerNoRetry        = "x-inngest-no-retry"
)

var (
	ErrUnableToReach       = fmt.Errorf("Unable to reach SDK URL")
	ErrDenied              = fmt.Errorf("Your server blocked the connection") // "connection timed out"
	ErrServerClosed        = fmt.Errorf("Your server closed the request before finishing.")
	ErrConnectionReset     = fmt.Errorf("Your server reset the connection while we were sending the request.")
	ErrUnexpectedEnd       = fmt.Errorf("Your server reset the connection while we were reading the reply: Unexpected ending response")
	ErrInvalidResponse     = fmt.Errorf("Error performing request to SDK URL")
	ErrBodyTooLarge        = fmt.Errorf("http response size is greater than the limit")
	ErrTLSHandshakeTimeout = fmt.Errorf("Your server didn't complete the TLS handshake in time")
	ErrDNSLookupTimeout    = fmt.Errorf("Gateway proxy request timeout")
)

// ExecuteRequest executes an HTTP request.  This returns the HTTP response, the body (limited by
// our max step size), the duration for the request, and any connection errors.
//
// NOTE: This does NOT handle HTTP errors, and instead only handles system errors.
func ExecuteRequest(ctx context.Context, c HTTPDoer, req *http.Request) (*http.Response, []byte, time.Duration, error) {
	pre := time.Now()
	resp, err := c.Do(req)
	dur := time.Since(pre)
	if err != nil {
		return resp, nil, dur, err
	}
	defer func() {
		if resp != nil {
			_ = resp.Body.Close()
		}
	}()

	// Read 1 extra byte above the max so that we can check if the response is
	// too large
	byt, err := io.ReadAll(io.LimitReader(resp.Body, consts.MaxSDKResponseBodySize+1))
	if err != nil {
		return resp, nil, dur, fmt.Errorf("error reading response body: %w", err)
	}

	if errors.Is(err, io.EOF) && resp == nil {
		return resp, nil, dur, ErrUnableToReach
	}

	if len(byt) > consts.MaxSDKResponseBodySize {
		return resp, byt, dur, ErrBodyTooLarge
	}

	// parse errors into common responses
	err = CommonHTTPErrors(err)

	return resp, byt, dur, err
}

func CommonHTTPErrors(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, io.EOF) {
		return err
	}

	{
		// timeouts
		if urlErr, ok := err.(*url.Error); ok && urlErr.Err == context.DeadlineExceeded {
			// This timed out.
			return context.DeadlineExceeded
		}
		if errors.Is(err, context.DeadlineExceeded) {
			// timed out
			return context.DeadlineExceeded
		}
	}

	if errors.Is(err, syscall.EPIPE) {
		return ErrServerClosed
	}
	if errors.Is(err, syscall.ECONNRESET) {
		return ErrConnectionReset
	}
	// If we get an unexpected EOF and the response is nil, error immediately.
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return ErrUnexpectedEnd
	}
	// Try to detect internal DNS lookup timeouts to user's do not see raw errors
	if IsDNSLookupTimeout(err) {
		return ErrDNSLookupTimeout
	}

	// use the error as-is, wrapped with a prefix for users.
	return fmt.Errorf("%s: %w", ErrInvalidResponse, err)
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

func CheckRedirect(req *http.Request, via []*http.Request) (err error) {
	if len(via) == 0 {
		return nil
	}

	if len(via) > 10 {
		return fmt.Errorf("stopped after 10 redirects")
	}

	if via[0].Body != nil && via[0].GetBody != nil {
		req.Body, err = via[0].GetBody()
		if err != nil {
			return err
		}
	}

	req.ContentLength = via[0].ContentLength

	// Combine headers from the original request and the redirect request
	for k, v := range via[0].Header {
		if len(v) > 0 {
			req.Header.Set(k, v[0])
		}
	}

	// Retain the original query params
	qp := req.URL.Query()
	for k, v := range via[0].URL.Query() {
		qp.Set(k, v[0])
	}
	req.URL.RawQuery = qp.Encode()

	return nil
}

// ShouldRetry determines if a request should be retried based on the response
// status code and headers.
//
// This is a best-effort attempt to determine if a request should be retried; we
// fall back to retrying if the request doesn't give us a firm answer.
func ShouldRetry(status int, noRetryHeader, sdkVersion string) bool {
	// noRetryHeader := resp.Header.Get("x-inngest-no-retry")
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
	versionHeader := strings.Split(sdkVersion, ":")
	if len(versionHeader) != 2 {
		// Unexpected version string; we can't determine if this is a
		// no-retry, so we'll assume we should retry.
		return true
	}

	lang := versionHeader[0]
	version := versionHeader[1]

	if !semver.IsValid(version) {
		// Unexpected version string; we can't determine if this is a
		// no-retry, so we'll assume we should retry.
		return true
	}

	// If we're here, we're assessing a 4XX response with no
	// `x-inngest-no-retry` header. We'll determine if this is a no-retry based
	// on the SDK version.
	if lang == "inngest-js" {
		switch {
		// 4XX should not be retried if <v2.4.1
		case semver.Major(version) == "v2" && semver.Compare(version, "v2.4.1") == -1:
			return false
		// 4XX should not be retried if <v1.10.1
		case semver.Major(version) == "v1" && semver.Compare(version, "v1.10.1") == -1:
			return false
		}
	}

	return true
}

// IsDNSLookupTimeout checks if the error matches a SDK gateway DNS lookup timeout
func IsDNSLookupTimeout(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	matched, _ := regexp.MatchString("lookup .* on.*: read udp.*: i/o timeout", errStr)
	if matched {
		return true
	}
	return false
}
