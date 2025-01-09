package realtime

import (
	"context"
	"net/http"

	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
)

const claimsKey = "rt-claims"

// realtimeAuthMW attempts to auth via realtime JWTs, falling back to the original auth
// middleware if no JWT was found.
func realtimeAuthMW(jwtSecret []byte, mw func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Check to see if a valid realtime JWT is being sent as a bearer token.
			key := r.Header.Get("Authorization")
			if key != "" && len(key) > 8 {
				// Remove "Bearer " prefix
				key = key[7:]
			}

			claims, err := ValidateJWT(r.Context(), jwtSecret, key)
			if err == nil {
				// Update the request's context with the claims
				r.WithContext(context.WithValue(ctx, claimsKey, claims))
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

func realtimeAuth(ctx context.Context, af apiv1auth.AuthFinder) (apiv1auth.V1Auth, error) {
	if claims, ok := ctx.Value(claimsKey).(*JWTClaims); ok {
		return claims, nil
	}
	return af(ctx)
}
