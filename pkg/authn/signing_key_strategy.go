// StrategySigningKey ensures that, if present, the signing key in the request header is valid.
package authn

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"regexp"
	"time"
)

const (
	SigningKeyPrefix       = "signkey-"
	SigningKeyPrefixTest   = "signkey-test-"
	SigningKeyPrefixBranch = "signkey-branch-"
	SigningKeyPrefixProd   = "signkey-prod-"
	workspaceTTL           = time.Hour * 24 * 3
)

var (
	keyRegexp = regexp.MustCompile(`^signkey-\w+-`)
)

type contextKey string

const authContextKey contextKey = "auth"

func SigningKeyMiddleware(signingKey *string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication if no signing key is configured
			if signingKey == nil {
				next.ServeHTTP(w, r)
				return
			}

			token := TokenFromHeader(r)

			authCtx, err := HandleSigningKey(r.Context(), token, *signingKey)
			if err != nil {
				http.Error(w, "Authentication failed", http.StatusUnauthorized)
				return
			}

			// Add auth context to request context
			ctx := context.WithValue(r.Context(), authContextKey, authCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func HandleSigningKey(ctx context.Context, clientProvidedKey string, trustedSigningKey string) (*AuthContext, error) {
	if len(clientProvidedKey) == 0 {
		return nil, errors.New("invalid signing key")
	}
	// normalize both trusted and provided keys (strip human readable headers)
	normalizedClientKey := normalizeKey(clientProvidedKey)
	normalizedTrustedKey := normalizeKey(trustedSigningKey)

	// trusted key is provided in plain text, user can provide either plain text or hashed so we need to check against
	hashedTrustedKey, _ := HashedSigningKey(normalizedTrustedKey)

	// Check if client key matches either the plain text or hashed version
	if subtle.ConstantTimeCompare([]byte(normalizedClientKey), []byte(normalizedTrustedKey)) == 1 ||
		subtle.ConstantTimeCompare([]byte(normalizedClientKey), []byte(hashedTrustedKey)) == 1 {
		return &AuthContext{
			isAuthenticated: true,
		}, nil
	}

	return nil, errors.New("invalid signing key")
}

func normalizeKey(key string) string {
	return keyRegexp.ReplaceAllString(key, "")
}

func HashedSigningKey(key string) (string, error) {
	key = normalizeKey(key)

	dst := make([]byte, hex.DecodedLen(len(key)))
	if _, err := hex.Decode(dst, []byte(key)); err != nil {
		return "", err
	}

	sum := sha256.Sum256(dst)
	enc := hex.EncodeToString(sum[:])

	return enc, nil
}
