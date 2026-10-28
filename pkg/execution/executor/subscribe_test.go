package executor

import (
	"crypto/sha1"
	"encoding/hex"
	"testing"

	"github.com/inngest/inngest/pkg/event"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionStepIDs(t *testing.T) {
	sum := sha1.Sum([]byte("inbox:3"))
	require.Equal(t, hex.EncodeToString(sum[:]), subscriptionStepID("inbox", 3))

	ack := sha1.Sum([]byte("inbox:ack:3"))
	require.Equal(t, hex.EncodeToString(ack[:]), subscriptionAckID("inbox", 3))

	require.NotEqual(t, subscriptionStepID("inbox", 1), subscriptionStepID("inbox", 2))
	require.NotEqual(t, subscriptionStepID("a", 1), subscriptionStepID("b", 1))
}

func TestFinalizeEventSeed(t *testing.T) {
	evt := event.Event{ID: "msg-1", Name: "agent/message", Timestamp: 1700000000000}

	first, err := finalizeEventSeed(evt).ToULID()
	require.NoError(t, err)
	second, err := finalizeEventSeed(evt).ToULID()
	require.NoError(t, err)
	require.Equal(t, first, second, "the same event ID gives the same internal ID")

	evt.ID = "msg-2"
	other, err := finalizeEventSeed(evt).ToULID()
	require.NoError(t, err)
	require.NotEqual(t, first, other)

	require.Nil(t, finalizeEventSeed(event.Event{ID: "x", Name: "inngest/function.finished", Timestamp: 1}))
	require.Nil(t, finalizeEventSeed(event.Event{Name: "agent/message", Timestamp: 1}))
}
