package realtime

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/realtime/streamingtypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishJWTExecutionLifetime(t *testing.T) {
	for _, tc := range []struct {
		name     string
		deadline time.Duration
		lifetime time.Duration
	}{
		{"maximum", 0, consts.MaxFunctionTimeout + time.Minute},
		{"shorter request", 5 * time.Minute, 6 * time.Minute},
		{"capped request", 4 * time.Hour, consts.MaxFunctionTimeout + time.Minute},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			if tc.deadline > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tc.deadline)
				defer cancel()
			}
			secret := []byte("test-secret")
			account, env := uuid.New(), uuid.New()
			token, err := NewPublishJWT(ctx, secret, account, env)
			require.NoError(t, err)
			claims, err := ValidateJWT(ctx, secret, token)
			require.NoError(t, err)
			require.True(t, claims.Publish)
			require.Equal(t, account.String(), claims.Subject)
			require.Equal(t, env, claims.Env)
			require.InDelta(t, tc.lifetime.Seconds(), claims.ExpiresAt.Sub(claims.IssuedAt.Time).Seconds(), 1)
			parseAt := func(at time.Time) error {
				_, err := jwt.ParseWithClaims(token, &JWTClaims{}, func(*jwt.Token) (any, error) { return secret, nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(Issuer), jwt.WithExpirationRequired(), jwt.WithTimeFunc(func() time.Time { return at }))
				return err
			}
			require.NoError(t, parseAt(claims.IssuedAt.Add(65*time.Second)))
			require.ErrorIs(t, parseAt(claims.ExpiresAt.Add(time.Second)), jwt.ErrTokenExpired)
		})
	}
	token, err := NewJWT(t.Context(), []byte("test-secret"), uuid.New(), uuid.New(), nil)
	require.NoError(t, err)
	claims, err := ValidateJWT(t.Context(), []byte("test-secret"), token)
	require.NoError(t, err)
	require.Equal(t, DefaultExpiry, claims.ExpiresAt.Sub(claims.IssuedAt.Time))
}

func TestNewJWT_DefaultExpiry(t *testing.T) {
	r := require.New(t)
	secret := []byte("test-secret")
	accountID := uuid.New()
	envID := uuid.New()

	token, err := NewJWT(context.Background(), secret, accountID, envID, []Topic{{
		Kind:    streamingtypes.TopicKindRun,
		Channel: "run-123",
		Name:    streamingtypes.TopicNameStream,
		EnvID:   envID,
	}})
	r.NoError(err)
	r.NotEmpty(token)

	claims, err := ValidateJWT(context.Background(), secret, token)
	r.NoError(err)
	r.Equal(accountID.String(), claims.Subject)
	r.Equal(envID, claims.Env)
	r.False(claims.Publish, "subscribe JWT should not have publish claim")
	r.Len(claims.Topics, 1)
	r.Equal("run-123", claims.Topics[0].Channel)
	r.Equal(streamingtypes.TopicNameStream, claims.Topics[0].Name)
}

func TestNewJWT_CustomExpiry(t *testing.T) {
	r := require.New(t)
	secret := []byte("test-secret")
	accountID := uuid.New()
	envID := uuid.New()
	customExpiry := MaxDurpStreamingRun + time.Minute

	token, err := NewJWT(context.Background(), secret, accountID, envID, []Topic{{
		Kind:    streamingtypes.TopicKindRun,
		Channel: "run-456",
		Name:    streamingtypes.TopicNameStream,
		EnvID:   envID,
	}}, NewJWTOpts{
		Expiry: new(customExpiry),
	})
	r.NoError(err)
	r.NotEmpty(token)

	claims, err := ValidateJWT(context.Background(), secret, token)
	r.NoError(err)

	// The token should be valid for roughly customExpiry from now.
	expiresAt := claims.ExpiresAt.Time
	issuedAt := claims.IssuedAt.Time
	diff := expiresAt.Sub(issuedAt)
	r.InDelta(customExpiry.Seconds(), diff.Seconds(), 1.0,
		"expiry should match the custom duration")
}

func TestNewJWT_WrongSecretFails(t *testing.T) {
	token, err := NewJWT(context.Background(), []byte("secret-a"), uuid.New(), uuid.New(), nil)
	require.NoError(t, err)

	_, err = ValidateJWT(context.Background(), []byte("secret-b"), token)
	assert.Error(t, err)
}
