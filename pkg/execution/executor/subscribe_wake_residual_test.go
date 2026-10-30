package executor

import (
	"errors"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/stretchr/testify/require"
)

// A message saved while a wake finishes is refused a wake of its own (the job
// ID is held), so what the finishing wake leaves behind has to survive every
// failure around its removal: repeated errors, a crash, a lost reply. The item
// is only ever removed by one lease-checked script that keeps it when a message
// was refused meanwhile, so these never leave the run without a wake.

var errCrash = errors.New("process crashed")

// recover is the queue after the wake's process died or gave up: the lease
// expires and the scavenger queues the item again.
func (f *wakeFixture) recoverExpiredLeases() {
	f.t.Helper()
	time.Sleep(2 * f.leaseFor)
	_, err := f.shard.Scavenge(f.ctx, 100)
	require.NoError(f.t, err)
}

func newFaultFixture(t *testing.T) *wakeFixture {
	f := newWakeFixture(t, false)
	f.leaseFor = 100 * time.Millisecond
	return f
}

// Five failures in a row of the removal.
func TestWakeSurvivesRepeatedRemovalErrors(t *testing.T) {
	f := newFaultFixture(t)
	f.deliver("inbox")

	r := f.start("inbox")
	r.read()
	f.deliver("inbox")

	for i := 0; i < 5; i++ {
		f.rq.failDequeues = 1
		require.Error(t, f.rq.Dequeue(f.ctx, f.shard.Name(), r.qi, queue.DequeueLeaseHeld()))
	}

	f.recoverExpiredLeases()
	f.drain("inbox")
	f.requireNothingLost("inbox")
}

// The process dies before the removal, or between the removal's script and the
// requeue it makes: in both the item is still there with its lease.
func TestWakeSurvivesACrashAroundTheRemoval(t *testing.T) {
	f := newFaultFixture(t)
	f.deliver("inbox")

	r := f.start("inbox")
	r.read()
	f.deliver("inbox")

	// Nothing more runs in the dead process.
	require.True(t, f.queued("inbox"))

	f.recoverExpiredLeases()
	f.drain("inbox")
	f.requireNothingLost("inbox")
}

// The queue applies the removal and the reply is lost, with a refused message
// and without one.
func TestWakeSurvivesALostRemovalReply(t *testing.T) {
	for _, refused := range []bool{true, false} {
		f := newFaultFixture(t)
		f.deliver("inbox")

		r := f.start("inbox")
		r.read()
		if refused {
			f.deliver("inbox")
		}

		f.rq.afterDequeue = func() error { return errors.New("connection reset") }
		require.Error(t, f.rq.Dequeue(f.ctx, f.shard.Name(), r.qi, queue.DequeueLeaseHeld()))

		// The processor retries the same removal: the item is queued again or gone.
		f.queueRemoves(r.qi)

		if !refused {
			f.deliver("inbox")
		}
		f.recoverExpiredLeases()
		f.drain("inbox")
		f.requireNothingLost("inbox")
	}
}

// A removal retried after it was applied must not remove or requeue twice what
// a newer wake took over.
func TestWakeRemovalRetryDoesNotTouchNewerWake(t *testing.T) {
	f := newFaultFixture(t)
	f.deliver("inbox")

	r := f.start("inbox")
	r.read()
	f.queueRemoves(r.qi)
	require.False(t, f.queued("inbox"))

	f.deliver("inbox")
	f.queueRemoves(r.qi)
	require.True(t, f.queued("inbox"))

	f.drain("inbox")
	f.requireNothingLost("inbox")
}
