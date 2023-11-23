package httpdriver

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	headerSDK            = "x-inngest-sdk"
	headerRequestVersion = "x-inngest-req-version"
	headerNoRetry        = "x-inngest-no-retry"
)

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

func checkRedirect(req *http.Request, via []*http.Request) (err error) {
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

// shouldRetry determines if a request should be retried based on the response
// status code and headers.
//
// This is a best-effort attempt to determine if a request should be retried; we
// fall back to retrying if the request doesn't give us a firm answer.
func shouldRetry(status int, noRetryHeader, sdkVersion string) bool {
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
