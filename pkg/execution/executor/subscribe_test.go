package executor

import (
	"crypto/sha1"
	"encoding/hex"
	"testing"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionStepIDs(t *testing.T) {
	sum := sha1.Sum([]byte("inbox:3"))
	require.Equal(t, hex.EncodeToString(sum[:]), subscriptionStepID("inbox", 3))

	require.NotEqual(t, subscriptionStepID("inbox", 1), subscriptionStepID("inbox", 2))
	require.NotEqual(t, subscriptionStepID("a", 1), subscriptionStepID("b", 1))
}

// A message is the step of its event in a subscription: the same event is the same
// message, different events never share an ID, and the stack gives the order.
func TestSubscriptionMessages(t *testing.T) {
	first := subscriptionMessageID("inbox", "evt-1")
	second := subscriptionMessageID("inbox", "evt-2")

	require.Equal(t, first, subscriptionMessageID("inbox", "evt-1"))
	require.NotEqual(t, first, second)
	require.NotEqual(t, first, subscriptionMessageID("other", "evt-1"))

	stack := []string{"step-a", second, "step-b", subscriptionMessageID("other", "evt-9"), first}
	require.Equal(t, []string{second, first}, subscriptionMessageIDs(stack, "inbox"))
	require.Equal(t, []string{subscriptionMessageID("other", "evt-9")}, subscriptionMessageIDs(stack, "other"))
	require.Empty(t, subscriptionMessageIDs(stack, "missing"))
}

func TestWaitSeq(t *testing.T) {
	require.Equal(t, 3, waitSeq(&state.Pause{Metadata: map[string]any{"seq": 3}}))
	require.Equal(t, 3, waitSeq(&state.Pause{Metadata: map[string]any{"seq": int64(3)}}))
	require.Equal(t, 3, waitSeq(&state.Pause{Metadata: map[string]any{"seq": float64(3)}}))
	require.Equal(t, 0, waitSeq(&state.Pause{}))
	require.Equal(t, 0, waitSeq(&state.Pause{Metadata: map[string]any{"seq": "3"}}))
}

// The parked marker alternates between two IDs by the awaited message, so the
// next park never shares an ID with the marker being cleaned up.
func TestSubscriptionParkIDs(t *testing.T) {
	first := subscriptionParkID("run", "inbox", 1)

	require.Equal(t, first, subscriptionParkID("run", "inbox", 1))
	require.Equal(t, first, subscriptionParkID("run", "inbox", 3))
	require.NotEqual(t, first, subscriptionParkID("run", "inbox", 2))
	require.NotEqual(t, first, subscriptionParkID("run", "other", 1))
	require.NotEqual(t, first, subscriptionParkID("run2", "inbox", 1))
	require.NotEqual(t, first, subscriptionPauseID("run", "inbox", 0))
}

func TestSubscriptionWaitPause(t *testing.T) {
	event := consts.SubscriptionWaitName

	require.True(t, isSubscriptionWaitPause(&state.Pause{Event: &event}))
	require.False(t, isSubscriptionWaitPause(&state.Pause{}))

	other := "agent/message"
	require.False(t, isSubscriptionWaitPause(&state.Pause{Event: &other}))
}
