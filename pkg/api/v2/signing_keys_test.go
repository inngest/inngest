package apiv2

import (
	"context"
	"testing"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
)

func TestFetchAccountSigningKeys_NoProvider(t *testing.T) {
	// Test when no signing keys provider is configured (dev mode)
	service := NewService(NewServiceOptions(ServiceConfig{}))

	ctx := context.Background()
	req := &apiv2.FetchAccountSigningKeysRequest{}

	resp, err := service.FetchAccountSigningKeys(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, resp.Data)
	require.NotNil(t, resp.Metadata)
	require.NotNil(t, resp.Page)
	require.False(t, resp.Page.HasMore)
}

func TestFetchAccountSigningKeys_WithProvider(t *testing.T) {
	// Test with signing keys provider (start mode)
	signingKey := "test-signing-key"
	provider := NewSigningKeysProvider(&signingKey)

	service := NewService(NewServiceOptions(ServiceConfig{
		SigningKeysProvider: provider,
	}))

	ctx := context.Background()
	req := &apiv2.FetchAccountSigningKeysRequest{}

	resp, err := service.FetchAccountSigningKeys(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Data, 1)

	// Check the signing key data
	key := resp.Data[0]
	require.Empty(t, key.Id)
	require.Empty(t, key.Name)
	require.Equal(t, "dev", key.Environment)
	require.Equal(t, signingKey, key.Key)
	require.NotNil(t, key.CreatedAt)

	// Check metadata
	require.NotNil(t, resp.Metadata)
	require.NotNil(t, resp.Page)
	require.False(t, resp.Page.HasMore)
}

func TestSigningKeysProvider(t *testing.T) {
	t.Run("returns signing key even when empty", func(t *testing.T) {
		emptyKey := ""
		provider := NewSigningKeysProvider(&emptyKey)
		keys, err := provider.GetSigningKeys(context.Background())
		require.NoError(t, err)
		require.Len(t, keys, 1)

		key := keys[0]
		require.Empty(t, key.Id)
		require.Empty(t, key.Name)
		require.Equal(t, "dev", key.Environment)
		require.Empty(t, key.Key)
		require.NotNil(t, key.CreatedAt)
	})

	t.Run("returns signing key with value", func(t *testing.T) {
		signingKey := "test-key-123"
		provider := NewSigningKeysProvider(&signingKey)

		keys, err := provider.GetSigningKeys(context.Background())
		require.NoError(t, err)
		require.Len(t, keys, 1)

		key := keys[0]
		require.Empty(t, key.Id)
		require.Empty(t, key.Name)
		require.Equal(t, "dev", key.Environment)
		require.Equal(t, signingKey, key.Key)
		require.NotNil(t, key.CreatedAt)
	})
}
