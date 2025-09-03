package apiv2

import (
	"context"
	"testing"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
)

func TestFetchAccountEventKeys_NoProvider(t *testing.T) {
	// Test when no event keys provider is configured (dev mode)
	service := NewService(NewServiceOptions(ServiceConfig{}))

	ctx := context.Background()
	req := &apiv2.FetchAccountEventKeysRequest{}

	resp, err := service.FetchAccountEventKeys(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, resp.Data)
	require.NotNil(t, resp.Metadata)
	require.NotNil(t, resp.Page)
	require.False(t, resp.Page.HasMore)
}

func TestFetchAccountEventKeys_WithProvider(t *testing.T) {
	// Test with event keys provider (start mode)
	eventKeys := []string{"test-event-key-1", "test-event-key-2"}
	provider := NewEventKeysProvider(eventKeys)

	service := NewService(NewServiceOptions(ServiceConfig{
		EventKeysProvider: provider,
	}))

	ctx := context.Background()
	req := &apiv2.FetchAccountEventKeysRequest{}

	resp, err := service.FetchAccountEventKeys(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Data, 2)

	// Check the event keys data
	for i, key := range resp.Data {
		require.Empty(t, key.Id)
		require.Empty(t, key.Name)
		require.Equal(t, "dev", key.Environment)
		require.Equal(t, eventKeys[i], key.Key)
		require.NotNil(t, key.CreatedAt)
	}

	// Check metadata
	require.NotNil(t, resp.Metadata)
	require.NotNil(t, resp.Page)
	require.False(t, resp.Page.HasMore)
}

func TestFetchAccountEventKeys_EmptyKeys(t *testing.T) {
	// Test with empty event keys list
	provider := NewEventKeysProvider([]string{})

	service := NewService(NewServiceOptions(ServiceConfig{
		EventKeysProvider: provider,
	}))

	ctx := context.Background()
	req := &apiv2.FetchAccountEventKeysRequest{}

	resp, err := service.FetchAccountEventKeys(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, resp.Data)
	require.NotNil(t, resp.Metadata)
	require.NotNil(t, resp.Page)
	require.False(t, resp.Page.HasMore)
}

func TestEventKeysProvider(t *testing.T) {
	t.Run("returns empty list when no keys provided", func(t *testing.T) {
		provider := NewEventKeysProvider([]string{})
		keys, err := provider.GetEventKeys(context.Background())
		require.NoError(t, err)
		require.Empty(t, keys)
	})

	t.Run("returns single event key", func(t *testing.T) {
		eventKey := "test-key-123"
		provider := NewEventKeysProvider([]string{eventKey})

		keys, err := provider.GetEventKeys(context.Background())
		require.NoError(t, err)
		require.Len(t, keys, 1)

		key := keys[0]
		require.Empty(t, key.Id)
		require.Empty(t, key.Name)
		require.Equal(t, "dev", key.Environment)
		require.Equal(t, eventKey, key.Key)
		require.NotNil(t, key.CreatedAt)
	})

	t.Run("returns multiple event keys", func(t *testing.T) {
		eventKeys := []string{"key-1", "key-2", "key-3"}
		provider := NewEventKeysProvider(eventKeys)

		keys, err := provider.GetEventKeys(context.Background())
		require.NoError(t, err)
		require.Len(t, keys, 3)

		for i, key := range keys {
			require.Empty(t, key.Id)
			require.Empty(t, key.Name)
			require.Equal(t, "dev", key.Environment)
			require.Equal(t, eventKeys[i], key.Key)
			require.NotNil(t, key.CreatedAt)
		}
	})
}

func TestFetchAccountEventKeys_PaginationValidation(t *testing.T) {
	service := NewService(NewServiceOptions(ServiceConfig{}))
	ctx := context.Background()

	t.Run("validates minimum limit", func(t *testing.T) {
		limit := int32(0)
		req := &apiv2.FetchAccountEventKeysRequest{Limit: &limit}

		_, err := service.FetchAccountEventKeys(ctx, req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Limit must be at least 1")
	})

	t.Run("validates maximum limit", func(t *testing.T) {
		limit := int32(101)
		req := &apiv2.FetchAccountEventKeysRequest{Limit: &limit}

		_, err := service.FetchAccountEventKeys(ctx, req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Limit cannot exceed 100")
	})

	t.Run("accepts valid limit", func(t *testing.T) {
		limit := int32(50)
		req := &apiv2.FetchAccountEventKeysRequest{Limit: &limit}

		resp, err := service.FetchAccountEventKeys(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}
