package executor

import (
	"crypto/rand"
	"strings"
	"testing"

	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

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

// The closed step starts with the message prefix, so the SDK does not wait for
// it, and is never one of the subscription's messages.
func TestSubscriptionClosedID(t *testing.T) {
	closed := subscriptionClosedID("inbox")

	require.True(t, strings.HasPrefix(closed, subscriptionMessagePrefix))
	require.NotEqual(t, closed, subscriptionClosedID("other"))
	require.Empty(t, subscriptionMessageIDs([]string{closed}, "inbox"))
}

func TestSubscriptionTriggerExclusion(t *testing.T) {
	trigger := ulid.MustNew(ulid.Now(), rand.Reader)
	md := &sv2.Metadata{Config: sv2.Config{EventIDs: []ulid.ULID{trigger}}}

	require.Nil(t, subscriptionTriggerExclusion(&state.SubscribeOpts{Key: "plain"}, md))
	require.Nil(t, subscriptionTriggerExclusion(&state.SubscribeOpts{Key: "inbox", TriggerIsFirst: true}, &sv2.Metadata{}))

	got := subscriptionTriggerExclusion(&state.SubscribeOpts{Key: "inbox", TriggerIsFirst: true}, md)
	require.NotNil(t, got)
	require.Equal(t, trigger.String(), *got)
}
