package executor

import (
	"crypto/rand"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	"testing"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/queue"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func (f *wakeFixture) flag(key string) (string, bool) {
	f.t.Helper()
	token, ok, err := f.rs.(sv2.SubscriptionWaiter).GetSubscriptionWaiting(f.ctx, f.md.ID, subscriptionKeyHash(key))
	require.NoError(f.t, err)
	return token, ok
}

// The delivery reads the flag and dies before it enqueues: nothing was
// destroyed, so the redelivered event wakes the reader.
func TestNQCrashBetweenFlagReadAndEnqueue(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("inbox").end()
	token, ok := f.flag("inbox")
	require.True(t, ok)

	f.e.queue = failingQueue{Queue: f.rq}
	f.events++
	evt := event.InternalEvent{ID: ulid.MustNew(uint64(f.events), rand.Reader), Event: event.Event{Name: "agent/message"}}
	require.Error(t, f.e.deliverSubscription(f.ctx, evt, f.pause("inbox"), logger.VoidLogger()))
	after, ok := f.flag("inbox")
	require.True(t, ok, "the flag survives the failed delivery")
	require.Equal(t, token, after)

	f.e.queue = f.rq
	require.NoError(t, f.e.deliverSubscription(f.ctx, evt, f.pause("inbox"), logger.VoidLogger()))
	f.delivered["inbox"]++
	require.Equal(t, 1, f.accepts("inbox"))

	f.settle("inbox")
	f.requireNothingLost("inbox")
}

// A delivery that read the flag of a wait enqueues after that wake ran and its
// item is gone: the job ID is still taken, so no second discovery starts.
func TestNQStaleLateEnqueueIsRefused(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("inbox").end()
	token, _ := f.flag("inbox")
	hash := subscriptionKeyHash("inbox")

	f.deliver("inbox")
	f.settle("inbox")
	require.Equal(t, 1, f.accepts("inbox"))
	require.False(t, f.queued("inbox"))

	before := f.rq.rejected
	require.NoError(t, f.e.enqueueSubscriptionWake(f.ctx, &f.md, "step", nil, nil, hash, token, "x"))
	require.Equal(t, before+1, f.rq.rejected)
	require.Equal(t, 1, f.accepts("inbox"))
	require.False(t, f.queued("inbox"))
	f.requireNothingLost("inbox")
}

// The compare-and-delete applies and its reply is lost: the wake errors, the
// queue retries it, the retry finds no match and carries on.
func TestNQLostClearReply(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("inbox").end()
	f.deliver("inbox")

	f.hooks.lostClearReply = true
	qi := f.wakes("inbox")[0]
	require.Error(t, f.e.startSubscriptionWake(f.ctx, f.md.ID, qi.Data))
	_, ok := f.flag("inbox")
	require.False(t, ok)

	// The retry, with the flag already gone.
	require.NoError(t, f.e.startSubscriptionWake(f.ctx, f.md.ID, qi.Data))
	r := f.begin("inbox")
	r.qi = &qi
	r.end()

	f.settle("inbox")
	f.requireNothingLost("inbox")
}

// A wake and a timer both run for one wait: a delivery queued the wake and the
// timer of the timed read fired. Both requests report the same wait; nothing is
// lost and nothing stays queued.
func TestNQTimerAndWakeBothRun(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("inbox").end()
	f.deliver("inbox")

	wake := f.start("inbox")
	require.NotNil(t, wake)
	require.NoError(t, f.e.startSubscriptionWake(f.ctx, f.md.ID, queue.Item{}))
	timer := f.begin("inbox")

	timer.readState()
	wake.readState()
	f.deliver("inbox")
	wake.end()
	timer.end()

	f.settle("inbox")
	f.requireNothingLost("inbox")
	require.LessOrEqual(t, f.accepts("inbox"), f.delivered["inbox"]+1)
}

// Any execution that is not the wake of the current wait clears the flags, so a
// parallel step finishing or an invoke returning leaves no flag on a busy run.
// A wake of an older wait leaves a newer wait's flag alone.
func TestNQFlagClearedOnEveryEntry(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("inbox").end()
	token, _ := f.flag("inbox")

	// A wake with another token: no match, the flag stays.
	old := queue.Item{Metadata: map[string]any{subscriptionWakeMetaKey: subscriptionWakeMeta(subscriptionKeyHash("inbox"), "old")}}
	require.NoError(t, f.e.startSubscriptionWake(f.ctx, f.md.ID, old))
	got, ok := f.flag("inbox")
	require.True(t, ok)
	require.Equal(t, token, got)

	// A step completion (an edge item with no wake metadata) clears it.
	require.NoError(t, f.e.startSubscriptionWake(f.ctx, f.md.ID, queue.Item{Kind: queue.KindEdge, Payload: queue.PayloadEdge{Edge: inngest.Edge{Incoming: "step"}}}))
	_, ok = f.flag("inbox")
	require.False(t, ok)

	// A delivery now finds no flag.
	f.deliver("inbox")
	require.Zero(t, f.accepts("inbox"))
}

// A response with a planned step beside the waiting op leaves no flag: the run
// is busy, so a delivery queues no discovery under it. A timer beside the wait
// does not make the run busy.
func TestNQPlannedStepBesideTheWaitSetsNoFlag(t *testing.T) {
	f := newWakeFixture(t, false)
	r := f.begin("inbox")
	r.readState()
	wait := func() *state.GeneratorOpcode {
		op := r.waitOp()
		return &op
	}
	rc := &mockRunContext{md: f.md, lifecycleItem: queue.Item{Payload: queue.PayloadEdge{}}}

	busy := []*state.GeneratorOpcode{{Op: enums.OpcodeStepPlanned, ID: "p"}, wait()}
	markBusySubscriptionWaits(busy)
	require.NoError(t, f.e.handleGeneratorOp(f.ctx, rc, *busy[1], OpcodeGroup{}))
	_, ok := f.flag("inbox")
	require.False(t, ok)
	f.deliver("inbox")
	require.Zero(t, f.accepts("inbox"))

	timed := []*state.GeneratorOpcode{{Op: enums.OpcodeSleep, ID: "t"}, wait()}
	markBusySubscriptionWaits(timed)
	require.NoError(t, f.e.handleGeneratorOp(f.ctx, rc, *timed[1], OpcodeGroup{}))
	_, ok = f.flag("inbox")
	require.True(t, ok)
}
