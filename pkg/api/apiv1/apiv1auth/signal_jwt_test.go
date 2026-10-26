package apiv1auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestSignalJWT(t *testing.T) {
	secret := []byte("secret")
	env := uuid.New()

	t.Run("round trip", func(t *testing.T) {
		token, err := CreateSignalJWT(secret, env, "build:1", time.Now().Add(time.Hour))
		require.NoError(t, err)
		claims, err := VerifySignalJWT(secret, token)
		require.NoError(t, err)
		require.Equal(t, env, claims.Env)
		require.Equal(t, "build:1", claims.SignalID)
	})

	t.Run("expired", func(t *testing.T) {
		token, err := CreateSignalJWT(secret, env, "build:1", time.Now().Add(-time.Minute))
		require.NoError(t, err)
		_, err = VerifySignalJWT(secret, token)
		require.Error(t, err)
	})

	t.Run("wrong secret", func(t *testing.T) {
		token, err := CreateSignalJWT(secret, env, "build:1", time.Now().Add(time.Hour))
		require.NoError(t, err)
		_, err = VerifySignalJWT([]byte("other"), token)
		require.Error(t, err)
	})

	t.Run("run token is not a signal token", func(t *testing.T) {
		token, err := CreateRunJWT(secret, env, ulid.Make())
		require.NoError(t, err)
		_, err = VerifySignalJWT(secret, token)
		require.Error(t, err)
	})

	t.Run("empty signal id", func(t *testing.T) {
		_, err := CreateSignalJWT(secret, env, "", time.Now().Add(time.Hour))
		require.Error(t, err)
	})
}
