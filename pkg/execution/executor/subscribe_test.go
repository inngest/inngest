package executor

import (
	"crypto/sha1"
	"encoding/hex"
	"slices"
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

func TestParkSeq(t *testing.T) {
	require.Equal(t, 3, parkSeq(&state.Pause{Metadata: map[string]any{"seq": 3}}))
	require.Equal(t, 3, parkSeq(&state.Pause{Metadata: map[string]any{"seq": int64(3)}}))
	require.Equal(t, 3, parkSeq(&state.Pause{Metadata: map[string]any{"seq": float64(3)}}))
	require.Equal(t, 0, parkSeq(&state.Pause{}))
	require.Equal(t, 0, parkSeq(&state.Pause{Metadata: map[string]any{"seq": "3"}}))
}

// A marker is keyed by the park step, which is unique to the park. The run is
// woken for message 1 and its next park awaits message 3 (two more messages were
// saved meanwhile): the two markers must not share an ID, or the late cleanup of
// the first would delete the second and leave the run parked.
func TestSubscriptionParkIDs(t *testing.T) {
	one := subscriptionStepID("inbox", 1)
	three := subscriptionStepID("inbox", 3)
	first := subscriptionParkID("run", "inbox", one)

	require.Equal(t, first, subscriptionParkID("run", "inbox", one))
	require.NotEqual(t, first, subscriptionParkID("run", "inbox", three))
	require.NotEqual(t, first, subscriptionParkID("run", "inbox", subscriptionStepID("inbox", 2)))
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

// The park races a delivery: the stack the park read before it wrote the marker
// can lack a message that the delivery saved just after, and that delivery
// found no marker. The read after the write must see it. A copy of the park
// that was woken in between is answered and must not wake the run again.
func TestParkStateOf(t *testing.T) {
	step := "park-step"
	first := subscriptionMessageID("inbox", "evt-1")
	second := subscriptionMessageID("inbox", "evt-2")
	other := subscriptionMessageID("other", "evt-3")

	// Before the marker: message 1 is saved, the park awaits message 2.
	before := []string{"a", first}
	require.Equal(t, parkWaiting, parkStateOf(before, "inbox", step, 2))

	// The delivery of message 2 saves it and finds no marker. After the marker
	// is written the park reads the stack again and wakes at once.
	after := append(slices.Clone(before), second)
	require.Equal(t, parkReady, parkStateOf(after, "inbox", step, 2))

	// Another subscription's messages do not count.
	require.Equal(t, parkWaiting, parkStateOf(append(slices.Clone(before), other), "inbox", step, 2))

	// A copy that was woken and answered in the meantime stops, even though the
	// message exists: waking again would queue a second discovery.
	answered := append(slices.Clone(after), step)
	require.Equal(t, parkAnswered, parkStateOf(answered, "inbox", step, 2))
}
