package pauses

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// The pauses a run blocks on while it waits for a subscription message are
// woken by an O(1) lookup by ID and live for a moment: they are never flushed.
func TestSkipFlushingSubscriptionWait(t *testing.T) {
	ws := uuid.New()

	require.True(t, SkipFlushing(Index{WorkspaceID: ws, EventName: SubscriptionParkIndexPrefix}, nil))
	require.True(t, SkipFlushing(Index{WorkspaceID: ws, EventName: SubscriptionParkIndexPrefix + ":run:hash"}, nil))
	require.False(t, SkipFlushing(Index{WorkspaceID: ws, EventName: "app/some.event"}, nil))
}
