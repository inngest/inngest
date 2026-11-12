package auth

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
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
				// IssuedAt:  jwt.NewNumericDate(time.Now()),
				ID: "fake-token",
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

func TestDeriveSessionJwtSecret(t *testing.T) {
	accountId, envId := uuid.New(), uuid.New()
	signingKey := "aabbccddeeff00112233445566778899"

	derived, err := DeriveSessionJwtSecret(signingKey)
	require.NoError(t, err)
	require.Len(t, derived, sessionJwtSecretSize)
	require.NotEqual(t, []byte(signingKey), derived)

	// Deterministic: the same signing key derives the same secret across
	// restarts, so existing deployments keep verifying after a restart.
	again, err := DeriveSessionJwtSecret(signingKey)
	require.NoError(t, err)
	require.Equal(t, derived, again)

	// Distinct per signing key.
	other, err := DeriveSessionJwtSecret("00112233445566778899aabbccddeeff")
	require.NoError(t, err)
	require.NotEqual(t, derived, other)

	t.Run("tokens signed with derived secret are rejected under the public dev constant", func(t *testing.T) {
		// A token minted by an instance that derived its secret from the
		// signing key must not verify against the public repo-wide constant.
		created, err := signSessionToken(derived, accountId, envId, DefaultExpiry, Entitlements{})
		require.NoError(t, err)

		_, err = VerifySessionToken([]byte("this-does-not-need-to-be-secret"), created)
		require.Error(t, err)
	})

	t.Run("tokens forged with the public dev constant are rejected under the derived secret", func(t *testing.T) {
		// The attack direction: an attacker who only knows the public repo
		// constant cannot mint a token that verifies against the derived secret.
		forged, err := signSessionToken([]byte("this-does-not-need-to-be-secret"), accountId, envId, DefaultExpiry, Entitlements{})
		require.NoError(t, err)

		_, err = VerifySessionToken(derived, forged)
		require.Error(t, err)
	})

	t.Run("tokens minted with the derived secret verify against it", func(t *testing.T) {
		created, err := signSessionToken(derived, accountId, envId, DefaultExpiry, Entitlements{})
		require.NoError(t, err)

		response, err := VerifySessionToken(derived, created)
		require.NoError(t, err)
		require.Equal(t, accountId, response.AccountID)
		require.Equal(t, envId, response.EnvID)
	})
}
