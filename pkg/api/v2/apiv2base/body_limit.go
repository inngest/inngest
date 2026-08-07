package apiv2base

import (
	"fmt"
	"net/http"
)

// MaxRequestBodyBytesMiddleware bounds request bodies before grpc-gateway decodes them.
func MaxRequestBodyBytesMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if maxBytes <= 0 || r.Body == nil {
				next.ServeHTTP(w, r)
				return
			}
			if r.ContentLength > maxBytes {
				writeHTTPError(
					w,
					http.StatusRequestEntityTooLarge,
					ErrorInvalidRequest,
					fmt.Sprintf("Request body cannot exceed %d bytes", maxBytes),
				)
				return
			}

			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
