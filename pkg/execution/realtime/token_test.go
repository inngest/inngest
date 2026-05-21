package realtime

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/realtime/streamingtypes"
	"github.com/inngest/inngest/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		Expiry: util.ToPtr(customExpiry),
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
