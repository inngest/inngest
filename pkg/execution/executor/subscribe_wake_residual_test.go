package executor

import (
	"errors"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/stretchr/testify/require"
)

// These tests reproduce the three ways a wake used to be lost without a second
// message to wake the thread again: a message saved while the wake finishes is
// refused a wake of its own (the job ID is held), so the finishing wake has to
// leave the run a wake whatever fails, crashes or is lost around its removal.

var errCrash = errors.New("process crashed")

// finish ends a wake the way the processor does and returns whatever the
// processor would have returned. A process that crashed does nothing more.
func (f *wakeFixture) finish(r *wakeRun) (crashed bool) {
	f.t.Helper()
	r.read()

	defer func() {
		if rec := recover(); rec != nil {
			require.Equal(f.t, errCrash, rec)
			crashed = true
		}
	}()

	err := f.e.subscriptionWakeFollowUp(f.ctx, f.md.ID, r.wake, okResp(), nil)
	if err == nil {
		f.queueRemoves(r.qi)
		return false
	}

	// A failed follow-up is retried by the queue as the same item; an item
	// that is gone cannot be retried and the retry is dropped.
	if rerr := f.rq.Requeue(f.ctx, f.shard.Name(), r.qi, time.Now()); rerr != nil {
		require.ErrorIs(f.t, rerr, queue.ErrQueueItemNotFound)
	}
	return false
}

// Five failures in a row of what runs after the release.
func TestWakeLostWhenEveryRetryAfterReleaseFails(t *testing.T) {
	f := newWakeFixture(t, false)
	f.deliver("inbox")

	r := f.start("inbox")
	f.rq.beforeDequeue = func() {
		f.deliver("inbox")
		f.rq.failEnqueues = 5
	}

	f.finish(r)
	f.drain("inbox")
	f.requireNothingLost("inbox")
}

// The process dies between the release and the wake that follows it.
func TestWakeLostWhenProcessCrashesAfterRelease(t *testing.T) {
	f := newWakeFixture(t, false)
	f.deliver("inbox")

	r := f.start("inbox")
	f.rq.beforeDequeue = func() { f.deliver("inbox") }
	f.rq.afterDequeue = func() error { panic(errCrash) }

	require.True(t, f.finish(r))
	f.drain("inbox")
	f.requireNothingLost("inbox")
}

// The queue applies the release and the reply is lost.
func TestWakeLostWhenReleaseReplyIsLost(t *testing.T) {
	f := newWakeFixture(t, false)
	f.deliver("inbox")

	r := f.start("inbox")
	f.rq.beforeDequeue = func() { f.deliver("inbox") }
	f.rq.afterDequeue = func() error { return errors.New("connection reset") }

	f.finish(r)
	f.drain("inbox")
	f.requireNothingLost("inbox")
}
