package headers

import (
	"net/http"
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

	HeaderAuthorization = "Authorization"
	HeaderContentType   = "Content-Type"
	HeaderUserAgent     = "User-Agent"
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
