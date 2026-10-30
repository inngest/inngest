package executor

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// Review A3. A wake's request ends (the response set a newer flag) and the item
// is not dequeued: a crash, or an error after the wait op was handled. The
// queue runs the same item again. The rerun only clears its own token, so the
// newer flag stays set while the rerun runs the function, and a message saved
// meanwhile queues a discovery for a reader that is busy.
func TestBreakRerunOfAWakeLeavesAFlagWhileBusy(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("inbox").end()
	f.deliver("inbox")

	first := f.start("inbox")
	require.NotNil(t, first)
	first.handle()
	_, ok := f.flag("inbox")
	require.True(t, ok, "the first run's response set the flag of its new wait")

	// The lease expired before the item was dequeued: the same item runs again.
	rerun := f.beginAs("inbox", first.qi.Data)
	rerun.readState()

	// The rerun is running the function now, so the reader is busy.
	f.deliver("inbox")
	require.Zero(t, f.accepts("inbox")-1, "a message saved while the rerun is busy queued a wake")
}

// The same interleaving converges when nothing is saved while the rerun runs:
// the rerun's response sets a newer flag and every message is seen.
func TestBreakRerunOfAWakeConverges(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("inbox").end()
	f.deliver("inbox")

	first := f.start("inbox")
	first.handle()
	f.deliver("inbox")
	rerun := f.beginAs("inbox", first.qi.Data)
	rerun.end()
	first.finish()

	f.settle("inbox")
	f.requireNothingLost("inbox")
}

// A wait beside a waitForEvent or an invoke is marked busy, so it sets no flag.
// The reader is idle: nothing runs until the other op resolves, which can be
// days. A message saved meanwhile queues no wake and the read never sees it.
func TestBreakWaitBesideAnAsyncWaitIsStillWakeable(t *testing.T) {
	for _, op := range []enums.Opcode{enums.OpcodeWaitForEvent, enums.OpcodeInvokeFunction} {
		t.Run(op.String(), func(t *testing.T) {
			f := newWakeFixture(t, false)
			r := f.begin("inbox")
			r.readState()
			wait := r.waitOp()
			rc := &mockRunContext{md: f.md, lifecycleItem: queue.Item{Payload: queue.PayloadEdge{}}}

			ops := []*state.GeneratorOpcode{{Op: op, ID: "other"}, &wait}
			markBusySubscriptionWaits(ops)
			require.NoError(t, f.e.handleGeneratorOp(f.ctx, rc, wait, OpcodeGroup{}))

			f.deliver("inbox")
			require.Equal(t, 1, f.accepts("inbox"), "the reader waits on the message and on %s; the message woke nothing", op)
		})
	}
}

// S9. A message wins a timed read and its wake starts. The read's timer is not
// cancelled, so when it fires its request reads the same message while the
// wake's request is still handling it: both run what follows the read.
func TestBreakTimerRunsWhileTheWakeHandlesTheMessage(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("inbox").end()
	f.deliver("inbox")

	wake := f.start("inbox")
	require.NotNil(t, wake)
	wake.readState()
	seenByWake := f.saw["inbox"]

	// The timer of the same read fires while the wake's request is in flight.
	timer := f.beginAs("inbox", queue.Item{Kind: queue.KindSleep})
	timer.readState()
	seenByTimer := f.saw["inbox"]

	require.False(t, seenByWake >= 1 && seenByTimer >= 1, "two overlapping requests both read message 1, so both run the step after it")
}

// A wake item whose metadata is gone (an older item version) clears every flag
// like any other entry. It still runs the function after clearing, so nothing
// is lost.
func TestBreakWakeWithoutMetadataStillWakes(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("inbox").end()
	f.deliver("inbox")

	qi := f.wakes("inbox")[0]
	qi.Data.Metadata = nil
	lease, err := f.shard.Lease(f.ctx, qi, 30*time.Second, time.Now())
	require.NoError(t, err)
	qi.LeaseID = lease
	r := f.beginAs("inbox", qi.Data)
	r.qi = &qi
	_, ok := f.flag("inbox")
	require.False(t, ok)
	f.deliver("inbox")
	r.end()

	f.settle("inbox")
	f.requireNothingLost("inbox")
}

// Idempotency expiry. A delivery read the flag of a wait and stalls past the
// idempotency period of that wait's finished wake. Its late enqueue is accepted
// as a second discovery of a wait that was already answered. Harmless to an
// idle reader; a duplicate request while the reader is mid-step.
func TestBreakLateDuplicateAfterTheIdempotencyPeriod(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("inbox").end()
	token, _ := f.flag("inbox")
	hash := subscriptionKeyHash("inbox")

	f.deliver("inbox")
	f.settle("inbox")
	require.Equal(t, 1, f.accepts("inbox"))

	f.mr.FastForward(subscriptionWakeIdempotency + time.Minute)
	require.NoError(t, f.e.enqueueSubscriptionWake(f.ctx, &f.md, "step", nil, nil, hash, token, "x"))
	require.Equal(t, 2, f.accepts("inbox"), "documents the duplicate: it is accepted after the period")

	f.settle("inbox")
	f.requireNothingLost("inbox")
}

// A delivery that saved the message and died before it read the flag: the
// redelivery finds the message saved, still reads the flag and wakes.
func TestBreakDeliveryDiesAfterSaveStep(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("inbox").end()

	f.events++
	evt := event.InternalEvent{ID: ulid.MustNew(uint64(f.events), rand.Reader), Event: event.Event{Name: "agent/message"}}
	_, err := f.rs.SaveStep(f.ctx, f.md.ID, subscriptionMessageID("inbox", evt.ID.String()), []byte(`{"data":{}}`))
	require.NoError(t, err)
	require.Zero(t, f.accepts("inbox"))

	require.NoError(t, f.e.deliverSubscription(f.ctx, evt, f.pause("inbox"), logger.VoidLogger()))
	f.delivered["inbox"]++
	require.Equal(t, 1, f.accepts("inbox"))

	f.settle("inbox")
	f.requireNothingLost("inbox")
}

// A wake starts, clears its flag, and dies before its response. A message saved
// in that gap finds no flag, and the rerun, which reads state after the gap,
// sees it.
func TestBreakWakeDiesAfterTheClear(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("inbox").end()
	f.deliver("inbox")

	dead := f.start("inbox")
	require.NotNil(t, dead)
	f.deliver("inbox")
	require.Equal(t, 1, f.accepts("inbox"), "the second message found no flag")

	rerun := f.beginAs("inbox", dead.qi.Data)
	rerun.qi = dead.qi
	rerun.end()

	f.settle("inbox")
	f.requireNothingLost("inbox")
}

// A step beside the wait, then the step finishing: its completion enters
// through Execute, clears flags, and the next discovery sets the flag again.
func TestBreakBusyThenIdleSetsTheFlag(t *testing.T) {
	f := newWakeFixture(t, false)
	r := f.begin("inbox")
	r.readState()
	wait := r.waitOp()
	rc := &mockRunContext{md: f.md, lifecycleItem: queue.Item{Payload: queue.PayloadEdge{}}}
	ops := []*state.GeneratorOpcode{{Op: enums.OpcodeStepPlanned, ID: "p"}, &wait}
	markBusySubscriptionWaits(ops)
	require.NoError(t, f.e.handleGeneratorOp(f.ctx, rc, wait, OpcodeGroup{}))
	f.deliver("inbox")
	require.Zero(t, f.accepts("inbox"))

	require.NoError(t, f.e.startSubscriptionWake(f.ctx, f.md.ID, queue.Item{Kind: queue.KindEdge, Payload: queue.PayloadEdge{Edge: inngest.Edge{Incoming: "step"}}}))
	next := f.begin("inbox")
	next.end()

	f.requireNothingLost("inbox")
}
