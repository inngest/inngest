package ratelimit

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/stretchr/testify/require"
)

func TestRateLimitKey(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	t.Run("It returns an event key", func(t *testing.T) {
		key, err := RateLimitKey(
			ctx,
			id,
			inngest.RateLimit{
				Key: str("event.data.orderId"),
			},
			map[string]any{
				"data": map[string]any{
					"orderId": "me and yoko ono",
				},
			},
		)
		require.NoError(t, err)
		require.EqualValues(t, hash("me and yoko ono", id), key)
	})

	t.Run("It concats strings and stuff", func(t *testing.T) {
		key, err := RateLimitKey(
			ctx,
			id,
			inngest.RateLimit{
				Key: str("event.data.name + '--' + event.data.id"),
			},
			map[string]any{
				"data": map[string]any{
					"name": "jj",
					"id":   "1",
				},
			},
		)
		require.NoError(t, err)
		require.EqualValues(t, hash("jj--1", id), key)
	})

	t.Run("It works with missing data", func(t *testing.T) {
		key, err := RateLimitKey(
			ctx,
			id,
			inngest.RateLimit{
				Key: str("event.data.name + '--' + event.data.id"),
			},
			map[string]any{
				"data": map[string]any{
					"name": "jj",
				},
			},
		)
		require.NoError(t, err)
		require.EqualValues(t, hash("jj--", id), key)
	})
}

func str(s string) *string {
	return &s
}
