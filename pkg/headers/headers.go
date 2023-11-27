package headers

import (
	"net/http"
)

const (
	// Tells the consumers (e.g. SDKs) what kind of Inngest server they're
	// communicating with (Cloud or Dev Server).
	HeaderKeyServerKind = "X-Inngest-Server-Kind"
	// Used by an SDK to tell the Inngest server what kind the SDK expects it
	// to be, used to validate that every part of a registration is performed
	// against the same target.
	HeaderKeyExpectedServerKind = "X-Inngest-Expected-Server-Kind"
)

const (
	ServerKindCloud = "cloud"
	ServerKindDev   = "dev"
)

func StaticHeadersMiddleware(serverKind string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(HeaderKeyServerKind, serverKind)
			next.ServeHTTP(w, r)
		})
	}
}
