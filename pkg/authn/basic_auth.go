// StrategyBasicAuth provides HTTP Basic Authentication for protecting the
// self-hosted dashboard in a browser.
package authn

import (
	"crypto/subtle"
	"net/http"
)

// BasicAuthMiddleware returns middleware that challenges the browser with
// HTTP Basic Auth. If either username or password is empty, the middleware
// is a no-op so self-hosted deployments without credentials configured are
// unaffected.
func BasicAuthMiddleware(username, password, realm string) func(http.Handler) http.Handler {
	if realm == "" {
		realm = "Inngest"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if username == "" || password == "" {
				next.ServeHTTP(w, r)
				return
			}

			u, p, ok := r.BasicAuth()
			userMatch := subtle.ConstantTimeCompare([]byte(u), []byte(username)) == 1
			passMatch := subtle.ConstantTimeCompare([]byte(p), []byte(password)) == 1
			if !ok || !userMatch || !passMatch {
				w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`", charset="UTF-8"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
