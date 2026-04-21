package auth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	connect "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/stretchr/testify/require"
)

func TestSessionToken(t *testing.T) {
	accountId, envId := uuid.New(), uuid.New()
	secret := []byte("this-is-a-very-strong-secret")

	t.Run("should accept valid token", func(t *testing.T) {
		created, err := signSessionToken(secret, accountId, envId, DefaultExpiry, Entitlements{})
		require.NoError(t, err)

		response, err := VerifySessionToken(secret, created)
		require.NoError(t, err)

		require.Equal(t, accountId, response.AccountID)
		require.Equal(t, envId, response.EnvID)
	})

	t.Run("should reject expired token", func(t *testing.T) {
		created, err := signSessionToken(secret, accountId, envId, time.Millisecond, Entitlements{})
		require.NoError(t, err)

		<-time.After(time.Millisecond * 5)

		response, err := VerifySessionToken(secret, created)
		require.Error(t, err)
		require.ErrorContains(t, err, "invalid token")
		require.ErrorContains(t, err, "token has invalid claims: token is expired")
		require.Nil(t, response)
	})

	t.Run("should reject token with invalid sig", func(t *testing.T) {
		secretForged := []byte("this-is-the-wrong-secret")

		created, err := signSessionToken(secretForged, accountId, envId, DefaultExpiry, Entitlements{})
		require.NoError(t, err)

		response, err := VerifySessionToken(secret, created)
		require.Error(t, err)
		require.ErrorContains(t, err, "invalid token")
		require.ErrorContains(t, err, "token signature is invalid: signature is invalid")
		require.Nil(t, response)
	})

	t.Run("should reject token with fake sig", func(t *testing.T) {
		fakeToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    wellKnownClaimIssuer,
				Subject:   accountId.String(),
				Audience:  []string{wellKnownClaimAudience},
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(DefaultExpiry)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				ID:        "fake-token",
			},
			Env: envId,
		})

		incompleteToken, err := fakeToken.SigningString()
		require.NoError(t, err)

		fakedToken := incompleteToken + ".fake"

		response, err := VerifySessionToken(secret, fakedToken)
		require.Error(t, err)
		require.ErrorContains(t, err, "invalid token")
		require.ErrorContains(t, err, "token signature is invalid: signature is invalid")
		require.Nil(t, response)
	})

	t.Run("should reject token without sig", func(t *testing.T) {
		fakeToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    wellKnownClaimIssuer,
				Subject:   accountId.String(),
				Audience:  []string{wellKnownClaimAudience},
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(DefaultExpiry)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				ID:        "fake-token",
			},
			Env: envId,
		})

		incompleteToken, err := fakeToken.SigningString()
		require.NoError(t, err)

		response, err := VerifySessionToken(secret, incompleteToken)
		require.Error(t, err)
		require.ErrorContains(t, err, "invalid token")
		require.ErrorContains(t, err, "invalid token: token is malformed: token contains an invalid number of segments")
		require.Nil(t, response)
	})

	t.Run("should reject token with different alg", func(t *testing.T) {
		secretForged := []byte("this-is-the-wrong-secret")

		fakeToken := jwt.NewWithClaims(jwt.SigningMethodHS384, claims{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    wellKnownClaimIssuer,
				Subject:   accountId.String(),
				Audience:  []string{wellKnownClaimAudience},
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(DefaultExpiry)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				ID:        "fake-token",
			},
			Env: envId,
		})

		fakedToken, err := fakeToken.SignedString(secretForged)
		require.NoError(t, err)

		response, err := VerifySessionToken(secret, fakedToken)
		require.Error(t, err)
		require.ErrorContains(t, err, "invalid token")
		require.ErrorContains(t, err, "invalid token: token signature is invalid: signing method HS384 is invalid")
		require.Nil(t, response)
	})

	t.Run("should reject token with alg=none", func(t *testing.T) {
		fakeToken := jwt.NewWithClaims(jwt.SigningMethodNone, claims{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    wellKnownClaimIssuer,
				Subject:   accountId.String(),
				Audience:  []string{wellKnownClaimAudience},
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(DefaultExpiry)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				ID:        "fake-token",
			},
			Env: envId,
		})
		require.Equal(t, fakeToken.Header["alg"], "none")

		incompleteToken, err := fakeToken.SigningString()
		require.NoError(t, err)

		// be evil and append something that looks like a signature so that we have sufficient segments
		incompleteToken += ".fake"

		response, err := VerifySessionToken(secret, incompleteToken)
		require.Error(t, err)
		require.ErrorContains(t, err, "invalid token")
		require.ErrorContains(t, err, "invalid token: token signature is invalid: signing method none is invalid")
		require.Nil(t, response)
	})

	t.Run("should reject token without required claims", func(t *testing.T) {
		fakeToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:   wellKnownClaimIssuer,
				Subject:  accountId.String(),
				Audience: []string{wellKnownClaimAudience},
				// ExpiresAt: jwt.NewNumericDate(time.Now().Add(DefaultExpiry)),
				IssuedAt: jwt.NewNumericDate(time.Now()),
				ID:       "fake-token",
			},
			Env: envId,
		})

		completeButInvalidToken, err := fakeToken.SignedString(secret)
		require.NoError(t, err)

		response, err := VerifySessionToken(secret, completeButInvalidToken)
		require.Error(t, err)
		require.ErrorContains(t, err, "invalid token")
		require.ErrorContains(t, err, "token has invalid claims: token is missing required claim: exp claim is required")

		require.Nil(t, response)
	})

}

func TestJWTAuthHandler(t *testing.T) {
	accountId, envId := uuid.New(), uuid.New()
	secret := []byte("this-is-a-very-strong-secret")

	newRequest := func(token string) *connect.WorkerConnectRequestData {
		return &connect.WorkerConnectRequestData{
			AuthData: &connect.AuthData{SessionToken: token},
		}
	}

	t.Run("missing token returns (nil, nil)", func(t *testing.T) {
		handler := NewJWTAuthHandler(secret)
		resp, err := handler(context.Background(), newRequest(""))
		require.NoError(t, err)
		require.Nil(t, resp)
	})

	t.Run("valid token returns response", func(t *testing.T) {
		token, err := signSessionToken(secret, accountId, envId, DefaultExpiry, Entitlements{})
		require.NoError(t, err)

		handler := NewJWTAuthHandler(secret)
		resp, err := handler(context.Background(), newRequest(token))
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, accountId, resp.AccountID)
		require.Equal(t, envId, resp.EnvID)
	})

	t.Run("invalid-signature token surfaces a wrapped error instead of silent skip", func(t *testing.T) {
		forgedSecret := []byte("this-is-the-wrong-secret")
		token, err := signSessionToken(forgedSecret, accountId, envId, DefaultExpiry, Entitlements{})
		require.NoError(t, err)

		handler := NewJWTAuthHandler(secret)
		resp, err := handler(context.Background(), newRequest(token))
		require.Error(t, err, "invalid-signature token must not silently coerce to (nil, nil)")
		require.ErrorContains(t, err, "connect JWT verification failed")
		require.ErrorContains(t, err, "token signature is invalid")
		require.Nil(t, resp)
	})

	t.Run("expired token surfaces a wrapped error", func(t *testing.T) {
		token, err := signSessionToken(secret, accountId, envId, time.Millisecond, Entitlements{})
		require.NoError(t, err)
		<-time.After(time.Millisecond * 5)

		handler := NewJWTAuthHandler(secret)
		resp, err := handler(context.Background(), newRequest(token))
		require.Error(t, err)
		require.ErrorContains(t, err, "connect JWT verification failed")
		require.ErrorContains(t, err, "token is expired")
		require.Nil(t, resp)
	})
}
