package headers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/dateutil"
)

const (
	// SDK version (e.g. "js:v3.2.1")
	HeaderKeySDK = "X-Inngest-SDK"

	// Tells the consumers (e.g. SDKs) what kind of Inngest server they're
	// communicating with (Cloud or Dev Server).
	HeaderKeyServerKind = "X-Inngest-Server-Kind"
	// Used by an SDK to tell the Inngest server what kind the SDK expects it
	// to be, used to validate that every part of a registration is performed
	// against the same target.
	HeaderKeyExpectedServerKind = "X-Inngest-Expected-Server-Kind"

	HeaderKeySignature = "X-Inngest-Signature"

	// HeaderRequestVersion represents the request version header.
	// XXX: This is exctracted from httpdriver and needs documenting.
	HeaderKeyRequestVersion = "x-inngest-req-version"

	// HeaderInngestStepID represents the step we wish to execute when
	// processing parallel steps.
	HeaderInngestStepID = "X-Inngest-Step-ID"

	HeaderAuthorization = "Authorization"
	HeaderContentType   = "Content-Type"
	HeaderUserAgent     = "User-Agent"

	// HeaderEventIDSeed is the header key used to send the event ID seed to the
	// Inngest server. Its of the form "millis,entropy", where millis is the
	// number of milliseconds since the Unix epoch, and entropy is a
	// base64-encoded 10-byte value that's sufficiently random for ULID
	// generation. For example: "1743130137367,eii2YKXRVTJPuA==".
	HeaderEventIDSeed = "x-inngest-event-id-seed"

	// HeaderKeyForceStepPlan tells the SDK to use step planning instead of
	// immediate execution. This is used when parallel steps are detected.
	HeaderKeyForceStepPlan = "X-Inngest-Force-Step-Plan"
)

const (
	ServerKindCloud = "cloud"
	ServerKindDev   = "dev"
)

func StaticHeadersMiddleware(serverKind string) func(next http.Handler) http.Handler {
	if serverKind == "" {
		panic("server kind must be set")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(HeaderKeyServerKind, serverKind)
			next.ServeHTTP(w, r)
		})
	}
}

// ContentTypeJsonResponse sets the HTTP response's Content-Type header to JSON
func ContentTypeJsonResponse() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			next.ServeHTTP(w, r)
		})
	}
}

// IsSDK returns whether the SDK header is set.  This should always be set in every
// SDK response, allowing us to check whether the response is controlled by Inngest.
func IsSDK(headers http.Header) bool {
	return headers.Get(HeaderKeySDK) != ""
}

// RequestVersion returns the value of the HeaderKeyRequestVersion header as an int.
func RequestVersion(headers http.Header) int {
	rv, _ := strconv.Atoi(headers.Get(HeaderKeyRequestVersion))
	return rv
}

// NoRetry indicates whether the custom no-retry haeder is set to true, preventing any
// future retries of this rqeuest/step.
func NoRetry(headers http.Header) bool {
	return headers.Get("x-inngest-no-retry") == "true"
}

// RetryAfter returns the parsed value of the retry-after header, or nil if the value
// is empty or unable to be parsed.
func RetryAfter(headers http.Header) *time.Time {
	return parseRetry(headers.Get("retry-after"))
}

// ParseRetry attempts to parse the retry-after header value.  It first checks to see
// if we have a reasonably sized second value (<= weeks), then parses the value as unix
// seconds.
//
// It falls back to parsing value in multiple formats: RFC3339, RFC1123, etc.
//
// This clips time within the minimums and maximums specified within consts.
func parseRetry(retry string) *time.Time {
	at := parseRetryTime(retry)
	if at == nil {
		return at
	}

	now := time.Now().UTC().Truncate(time.Second)

	dur := time.Until(*at)
	if dur > consts.MaxRetryDuration {
		// apply max duration
		next := now.Add(consts.MaxRetryDuration)
		return &next
	}
	if dur < consts.MinRetryDuration {
		// apply min duration
		next := now.Add(consts.MinRetryDuration)
		return &next
	}
	return at
}

func parseRetryTime(retry string) *time.Time {
	if retry == "" {
		return nil
	}
	if len(retry) <= 7 {
		// Assume this is an int;  no dates can be <= 7 characters.
		secs, _ := strconv.Atoi(retry)
		if secs > 0 {
			parsed := time.Now().UTC().Truncate(time.Second).Add(time.Second * time.Duration(secs))
			return &parsed
		}
	}
	if val, err := dateutil.ParseString(retry); err == nil {
		return &val
	}
	return nil
}
