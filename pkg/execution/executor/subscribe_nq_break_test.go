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

			ops := []*state.GeneratorOpcode{{Op: op, ID: "other", Opts: map[string]any{"event": "e"}}, &wait}
			markBusySubscriptionWaits(ops)
			require.NoError(t, f.e.handleGeneratorOp(f.ctx, rc, wait, OpcodeGroup{}))

			f.deliver("inbox")
			require.Equal(t, 1, f.accepts("inbox"), "the reader waits on the message and on %s; the message woke nothing", op)
		})
	}
}

// S9. A timed read's message and its timer both want to run what follows. Each
// claims the wait by saving the timer's step; the first one wins and the other
// sees the duplicate and runs nothing.
func TestBreakTimerAndWakeAreSingleWinner(t *testing.T) {
	wakeItem := func(f *wakeFixture) queue.Item {
		token := subscriptionWaitToken("timer", 1)
		return queue.Item{Metadata: map[string]any{subscriptionWakeMetaKey: subscriptionWakeMeta(subscriptionKeyHash("inbox"), token)}}
	}

	t.Run("MessageWins", func(t *testing.T) {
		f := newWakeFixture(t, false)
		item := wakeItem(f)
		f.deliver("inbox")

		ok, err := f.e.claimSubscriptionTimer(f.ctx, f.md.ID, item)
		require.NoError(t, err)
		require.True(t, ok)

		// A rerun of the same wake still runs.
		ok, err = f.e.claimSubscriptionTimer(f.ctx, f.md.ID, item)
		require.NoError(t, err)
		require.True(t, ok)

		// The timer saves null, sees the duplicate, and sees the message won.
		_, err = f.rs.SaveStep(f.ctx, f.md.ID, "timer", []byte("null"))
		require.Error(t, err)
		won, err := f.e.messageWonTimer(f.ctx, f.md.ID, "timer")
		require.NoError(t, err)
		require.True(t, won)
	})

	t.Run("TimerWins", func(t *testing.T) {
		f := newWakeFixture(t, false)
		f.deliver("inbox")
		_, err := f.rs.SaveStep(f.ctx, f.md.ID, "timer", []byte("null"))
		require.NoError(t, err)

		ok, err := f.e.claimSubscriptionTimer(f.ctx, f.md.ID, wakeItem(f))
		require.NoError(t, err)
		require.False(t, ok, "the timer's request runs the function")

		// A retry of the timer sees its own null, so it carries on.
		_, err = f.rs.SaveStep(f.ctx, f.md.ID, "timer", []byte("null"))
		require.Error(t, err)
		won, err := f.e.messageWonTimer(f.ctx, f.md.ID, "timer")
		require.NoError(t, err)
		require.False(t, won)
	})

	t.Run("MissingMessageClaimsNothing", func(t *testing.T) {
		f := newWakeFixture(t, false)

		ok, err := f.e.claimSubscriptionTimer(f.ctx, f.md.ID, wakeItem(f))
		require.NoError(t, err)
		require.True(t, ok)

		// The timer step is still free for the timer to save.
		_, err = f.rs.SaveStep(f.ctx, f.md.ID, "timer", []byte("null"))
		require.NoError(t, err)
	})

	t.Run("UntimedWakeAlwaysRuns", func(t *testing.T) {
		f := newWakeFixture(t, false)
		item := queue.Item{Metadata: map[string]any{subscriptionWakeMetaKey: subscriptionWakeMeta(subscriptionKeyHash("inbox"), subscriptionWaitToken("", 0))}}
		ok, err := f.e.claimSubscriptionTimer(f.ctx, f.md.ID, item)
		require.NoError(t, err)
		require.True(t, ok)
	})
}

// The executor tells a timed read's waiting op which sleep is its timer.
func TestBreakWaitingOpLearnsItsTimer(t *testing.T) {
	wait := &state.GeneratorOpcode{Op: enums.OpcodeNone, ID: "w", Opts: map[string]any{"subscribe": map[string]any{"key": "inbox", "seq": 1}}}
	wait.Userland = &struct {
		ID    string `json:"id"`
		Index int    `json:"index,omitempty"`
	}{ID: "read:1:waiting"}
	sleep := &state.GeneratorOpcode{Op: enums.OpcodeSleep, ID: "hashed-timer"}
	sleep.Userland = &struct {
		ID    string `json:"id"`
		Index int    `json:"index,omitempty"`
	}{ID: "read:1"}
	other := &state.GeneratorOpcode{Op: enums.OpcodeSleep, ID: "user-sleep"}
	other.Userland = &struct {
		ID    string `json:"id"`
		Index int    `json:"index,omitempty"`
	}{ID: "nap"}

	markSubscriptionTimers([]*state.GeneratorOpcode{other, sleep, wait})
	require.Equal(t, "hashed-timer", wait.Opts.(map[string]any)["timer"])
}

// A rerun of an earlier wake clears only its own subscription's flag, whatever
// token the flag holds.
func TestBreakRerunClearsOnlyItsOwnSubscription(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("a").end()
	f.begin("b").end()

	stale := queue.Item{Metadata: map[string]any{subscriptionWakeMetaKey: subscriptionWakeMeta(subscriptionKeyHash("a"), "old")}}
	require.NoError(t, f.e.startSubscriptionWake(f.ctx, f.md.ID, stale))

	_, ok := f.flag("a")
	require.False(t, ok)
	_, ok = f.flag("b")
	require.True(t, ok)
}

// The parking ops leave a run idle; a subscription op beside the read does not.
func TestBreakParkingOps(t *testing.T) {
	for _, op := range []enums.Opcode{enums.OpcodeSleep, enums.OpcodeWaitForEvent, enums.OpcodeInvokeFunction, enums.OpcodeWaitForSignal} {
		wait := &state.GeneratorOpcode{Op: enums.OpcodeNone, ID: "w", Opts: map[string]any{}}
		markBusySubscriptionWaits([]*state.GeneratorOpcode{{Op: op, ID: "x", Opts: map[string]any{"event": "e"}}, wait})
		require.Nil(t, wait.Opts.(map[string]any)["busy"], op.String())
	}

	wait := &state.GeneratorOpcode{Op: enums.OpcodeNone, ID: "w", Opts: map[string]any{}}
	sub := &state.GeneratorOpcode{Op: enums.OpcodeWaitForEvent, ID: "s", Opts: map[string]any{"event": "e", "subscribe": map[string]any{"key": "k"}}}
	markBusySubscriptionWaits([]*state.GeneratorOpcode{sub, wait})
	require.Equal(t, true, wait.Opts.(map[string]any)["busy"])

	wait = &state.GeneratorOpcode{Op: enums.OpcodeNone, ID: "w", Opts: map[string]any{}}
	markBusySubscriptionWaits([]*state.GeneratorOpcode{{Op: enums.OpcodeStepPlanned, ID: "p"}, wait})
	require.Equal(t, true, wait.Opts.(map[string]any)["busy"])
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

// A late write of a finished timed wait's token must not overwrite the flag of a
// newer wait: the wake it would name would stop at the finished timer's step
// and the newer wait would not be woken.
func TestBreakLateFlagOfAFinishedTimedWait(t *testing.T) {
	f := newWakeFixture(t, false)
	r := f.begin("inbox")
	r.readState()

	// The timer of the old wait fired; a newer wait set its flag.
	_, err := f.rs.SaveStep(f.ctx, f.md.ID, "timer1", []byte("null"))
	require.NoError(t, err)
	f.begin("inbox").end()
	newer, ok := f.flag("inbox")
	require.True(t, ok)

	// The old request now reports its wait, with its timer.
	wait := r.waitOp()
	wait.Opts.(map[string]any)["timer"] = "timer1"
	rc := &mockRunContext{md: f.md}
	require.NoError(t, f.e.handleGeneratorSubscribeWait(f.ctx, rc, wait, queue.PayloadEdge{Edge: inngest.Edge{Incoming: "step"}}))

	got, ok := f.flag("inbox")
	require.True(t, ok)
	require.Equal(t, newer, got)
}
