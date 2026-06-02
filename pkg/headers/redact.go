package headers

import (
	"net/http"
	"strings"
)

// sensitiveHeaders contains header names (lowercase) that should be redacted
// to prevent exposure of credentials, tokens, and session data in traces.
var sensitiveHeaders = map[string]bool{
	"authorization":       true,
	"proxy-authorization": true,
	"cookie":              true,
	"set-cookie":          true,
	"x-api-key":           true,
	"x-auth-token":        true,
	"www-authenticate":    true,
	"proxy-authenticate":  true,
}

// Redact returns a copy of the given headers with sensitive headers redacted.
func Redact(headers http.Header) http.Header {
	redacted := http.Header{}
	const redactedValue = "[REDACTED]"

	for key, values := range headers {
		if _, isSensitiveHeader := sensitiveHeaders[strings.ToLower(key)]; isSensitiveHeader {
			redacted[key] = []string{redactedValue}
		} else {
			redacted[key] = values
		}
	}

	return redacted
}
