package executor

import (
	"crypto/rand"
	"testing"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func (f *wakeFixture) messageSteps(key string) int {
	f.t.Helper()
	stack, err := f.rs.LoadStack(f.ctx, f.md.ID)
	require.NoError(f.t, err)

	return len(subscriptionMessageIDsOfHash(stack, subscriptionKeyHash(key)))
}

func (f *wakeFixture) buffered(key string) int {
	f.t.Helper()
	waiter, err := f.e.subscriptionWaiter()
	require.NoError(f.t, err)
	buffered, err := waiter.BufferedSubscriptionMessages(f.ctx, f.md.ID, subscriptionKeyHash(key))
	require.NoError(f.t, err)

	return len(buffered)
}

// Messages that arrive while the reader is not waiting are no steps: they
// wait in the buffer, and one flush saves them all.
func TestBufferBurstWhileBusyIsNoStepsUntilTheReaderAsks(t *testing.T) {
	f := newWakeFixture(t, false)
	f.burst("inbox", 20)

	require.Equal(t, 0, f.messageSteps("inbox"))
	require.Equal(t, 20, f.buffered("inbox"))
	require.Zero(t, f.accepts("inbox"), "a run that is not waiting is not woken")

	// The reader asks: its wait flushes the buffer, and wakes the run once.
	f.begin("inbox").end()
	f.settle("inbox")

	require.Equal(t, 20, f.messageSteps("inbox"))
	require.Equal(t, 0, f.buffered("inbox"))
	f.requireNothingLost("inbox")
}

// An event delivered twice is one message: buffered once, and a redelivery of
// one that is already a step adds nothing.
func TestBufferRedeliveredEventIsNoDuplicateStep(t *testing.T) {
	f := newWakeFixture(t, false)
	evt := event.InternalEvent{ID: ulid.MustNew(1, rand.Reader), Event: event.Event{Name: "agent/message"}}

	for i := 0; i < 3; i++ {
		require.NoError(t, f.e.deliverSubscription(f.ctx, evt, f.pause("inbox"), logger.VoidLogger()))
	}
	require.Equal(t, 1, f.buffered("inbox"))

	f.flush("inbox")
	require.Equal(t, 1, f.messageSteps("inbox"))

	require.NoError(t, f.e.deliverSubscription(f.ctx, evt, f.pause("inbox"), logger.VoidLogger()))
	require.Equal(t, 0, f.buffered("inbox"), "a message that is a step is not buffered again")
	f.flush("inbox")
	require.Equal(t, 1, f.messageSteps("inbox"))
}

// A flush that stopped halfway (crashed after saving some steps, before
// dropping the buffer) is finished by the next one, with no duplicates.
func TestBufferFlushThatCrashedIsFinishedByTheNext(t *testing.T) {
	f := newWakeFixture(t, false)
	f.burst("inbox", 3)

	waiter, err := f.e.subscriptionWaiter()
	require.NoError(t, err)
	buffered, err := waiter.BufferedSubscriptionMessages(f.ctx, f.md.ID, subscriptionKeyHash("inbox"))
	require.NoError(t, err)
	_, err = f.rs.SaveStep(f.ctx, f.md.ID, buffered[0].ID, buffered[0].Value)
	require.NoError(t, err)

	f.flush("inbox")
	require.Equal(t, 3, f.messageSteps("inbox"))
	require.Equal(t, 0, f.buffered("inbox"))
}

// A message delivered to a waiting reader is a step at once, and wakes it;
// it is not buffered.
func TestBufferWaitingReaderGetsTheMessageAsAStep(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("inbox").end()
	f.deliver("inbox")

	require.Equal(t, 1, f.messageSteps("inbox"))
	require.Equal(t, 0, f.buffered("inbox"))
	require.Equal(t, 1, f.accepts("inbox"))
}
