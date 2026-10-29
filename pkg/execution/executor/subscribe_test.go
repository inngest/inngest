package executor

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionStepIDs(t *testing.T) {
	sum := sha1.Sum([]byte("inbox:3"))
	require.Equal(t, hex.EncodeToString(sum[:]), subscriptionStepID("inbox", "", 3, 0))

	require.NotEqual(t, subscriptionStepID("inbox", "", 1, 0), subscriptionStepID("inbox", "", 2, 0))
	// A park that timed out is followed by a new one for the same message.
	require.NotEqual(t, subscriptionStepID("inbox", "", 1, 0), subscriptionStepID("inbox", "", 1, 1))
	require.NotEqual(t, subscriptionStepID("inbox", "", 1, 1), subscriptionStepID("inbox", "", 1, 2))
	require.NotEqual(t, subscriptionStepID("a", "", 1, 0), subscriptionStepID("b", "", 1, 0))
	// A view has parks of its own, which never share an ID with the inbox's.
	require.NotEqual(t, subscriptionStepID("inbox", "", 1, 0), subscriptionStepID("inbox", "v1", 1, 0))
	require.NotEqual(t, subscriptionStepID("inbox", "v1", 1, 0), subscriptionStepID("inbox", "v2", 1, 0))
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

// A marker is keyed by the park step, which is unique to the park. The run is
// woken for message 1 and its next park awaits message 3 (two more messages were
// saved meanwhile): the two markers must not share an ID, or the late cleanup of
// the first would delete the second and leave the run parked.
func TestSubscriptionParkIDs(t *testing.T) {
	one := subscriptionStepID("inbox", "", 1, 0)
	three := subscriptionStepID("inbox", "", 3, 0)
	first := subscriptionParkID("run", "inbox", one)

	require.Equal(t, first, subscriptionParkID("run", "inbox", one))
	require.NotEqual(t, first, subscriptionParkID("run", "inbox", three))
	require.NotEqual(t, first, subscriptionParkID("run", "inbox", subscriptionStepID("inbox", "", 2, 0)))
	require.NotEqual(t, first, subscriptionParkID("run", "other", one))
	require.NotEqual(t, first, subscriptionParkID("run2", "inbox", one))
	require.NotEqual(t, first, subscriptionPauseID("run", "inbox", 0))
}

// The markers of one run and subscription share a small index, which a delivery
// reads to find the current park. It is a separate index per run and
// subscription, still recognised as a park index.
func TestSubscriptionParkIndexName(t *testing.T) {
	name := subscriptionParkIndexName("run", "inbox")

	require.Equal(t, name, subscriptionParkIndexName("run", "inbox"))
	require.NotEqual(t, name, subscriptionParkIndexName("run", "other"))
	require.NotEqual(t, name, subscriptionParkIndexName("run2", "inbox"))
	require.True(t, isSubscriptionParkPause(&state.Pause{Event: &name}))
}

func TestSubscriptionParkPause(t *testing.T) {
	event := consts.SubscriptionParkName

	require.True(t, isSubscriptionParkPause(&state.Pause{Event: &event}))
	require.False(t, isSubscriptionParkPause(&state.Pause{}))

	other := "agent/message"
	require.False(t, isSubscriptionParkPause(&state.Pause{Event: &other}))
}

// The closed step starts with the message prefix, so the SDK does not wait for
// it, and is never one of the subscription's messages.
func TestSubscriptionClosedID(t *testing.T) {
	closed := subscriptionClosedID("inbox")

	require.True(t, strings.HasPrefix(closed, subscriptionMessagePrefix))
	require.NotEqual(t, closed, subscriptionClosedID("other"))
	require.Empty(t, subscriptionMessageIDs([]string{closed}, "inbox"))
}
