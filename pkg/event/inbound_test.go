package event

import (
	"context"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/stretchr/testify/require"
)

func TestNormalizeInbound(t *testing.T) {
	receivedAt := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	evt := Event{
		Name: "app/user.created",
		Data: map[string]any{
			"message":                     "hello",
			consts.InngestEventDataPrefix: map[string]any{"invoke": true},
		},
	}

	_, err := NormalizeInbound(context.Background(), &evt, receivedAt)

	require.NoError(t, err)
	require.Equal(t, receivedAt.UnixMilli(), evt.Timestamp)
	require.Equal(t, map[string]any{"message": "hello"}, evt.Data)
	require.Equal(t, map[string]any{}, evt.User)
}

func TestNormalizeInboundRejectsReservedNamesCaseInsensitively(t *testing.T) {
	evt := Event{Name: "INNGEST/function.finished"}

	_, err := NormalizeInbound(context.Background(), &evt, time.Now())

	require.ErrorContains(t, err, "reserved for internal use")
	require.ErrorIs(t, err, ErrInternalEventName)
}
