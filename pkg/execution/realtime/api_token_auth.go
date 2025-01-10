package realtime

import (
	"context"
	"fmt"
	"net/http"
)

type claimsKeyTyp string

const claimsKey = claimsKeyTyp("rt-claims")

// realtimeAuthMW attempts to auth via realtime JWTs, falling back to the original auth
// middleware if no JWT was found.
func realtimeAuthMW(jwtSecret []byte, mw func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			var key string

			// Check to see if a valid realtime JWT is being sent as a bearer token.
			if k := r.Header.Get("Authorization"); k != "" && len(k) > 8 {
				// Remove "Bearer " prefix
				key = k[7:]
			}
			// Check the ?token query param, for websocket libraries that cannot use
			// http headers.
			if key == "" {
				key = r.URL.Query().Get("token")
			}

			claims, err := ValidateJWT(r.Context(), jwtSecret, key)
			if err == nil {
				// Update the request's context with the claims
				r = r.WithContext(context.WithValue(ctx, claimsKey, claims))
				// We have a valid set of claims
				next.ServeHTTP(w, r)
				return
			}

			// Call the original middelware.
			if mw != nil {
				mw(next)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func realtimeAuth(ctx context.Context) (*JWTClaims, error) {
	if claims, ok := ctx.Value(claimsKey).(*JWTClaims); ok {
		return claims, nil
	}
	return nil, fmt.Errorf("no jwt found")
}
