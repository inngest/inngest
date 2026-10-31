package executor

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// These tests drive step.subscribe through the real run state (miniredis, the real
// Lua) and the real queue, one step at a time, so every interleaving is a sequence
// the test writes down.

type subscribeQueue struct {
	queue.Queue
	accepted []queue.Item
	fail     error
}

func (q *subscribeQueue) Enqueue(ctx context.Context, item queue.Item, at time.Time, opts queue.EnqueueOpts) error {
	if q.fail != nil {
		return q.fail
	}
	err := q.Queue.Enqueue(ctx, item, at, opts)
	if err == nil {
		q.accepted = append(q.accepted, item)
	}

	return err
}

type subscribePauses struct {
	pauses.Manager
	written map[uuid.UUID]*state.Pause
	deleted []uuid.UUID
}

func (m *subscribePauses) Write(_ context.Context, _ pauses.Index, p ...*state.Pause) (int, error) {
	for _, pause := range p {
		m.written[pause.ID] = pause
	}

	return len(p), nil
}

func (m *subscribePauses) PauseIDsForRun(context.Context, ulid.ULID) ([]uuid.UUID, error) {
	return make([]uuid.UUID, len(m.written)), nil
}

func (m *subscribePauses) PauseByID(_ context.Context, _ pauses.Index, id uuid.UUID) (*state.Pause, error) {
	if p, ok := m.written[id]; ok {
		return p, nil
	}

	return nil, state.ErrPauseNotFound
}

func (m *subscribePauses) Delete(_ context.Context, _ pauses.Index, p state.Pause, _ ...state.DeletePauseOpt) error {
	m.deleted = append(m.deleted, p.ID)
	delete(m.written, p.ID)

	return nil
}

type subscribeFixture struct {
	t      *testing.T
	ctx    context.Context
	mr     *miniredis.Miniredis
	e      *executor
	q      *subscribeQueue
	shard  queue.QueueShard
	rs     sv2.RunService
	pm     *subscribePauses
	md     sv2.Metadata
	events int
	group  OpcodeGroup
}

func newSubscribeFixture(t *testing.T) *subscribeFixture {
	t.Helper()

	ctx := logger.WithStdlib(context.Background(), logger.VoidLogger())
	mr := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{mr.Addr()}, DisableCache: true})
	require.NoError(t, err)
	t.Cleanup(rc.Close)

	uc := redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	sc := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
		UnshardedClient: uc, FunctionRunStateClient: rc, BatchClient: rc,
		StateDefaultKey: redis_state.StateDefaultKey, QueueDefaultKey: redis_state.QueueDefaultKey,
		FnRunIsSharded: redis_state.AlwaysShardOnRun,
	})
	mgr, err := redis_state.New(ctx, redis_state.WithShardedClient(sc), redis_state.WithPauseDeleter(redis_state.NewPauseStore(uc)))
	require.NoError(t, err)
	rs := redis_state.MustRunServiceV2(mgr)

	shard := redis_state.NewQueueShard("local", redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey))
	registry, err := queue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	base, err := queue.New(ctx, "subscribe", registry)
	require.NoError(t, err)

	md := sv2.Metadata{
		ID: sv2.ID{
			RunID:      ulid.MustNew(ulid.Now(), rand.Reader),
			FunctionID: uuid.New(),
			Tenant:     sv2.Tenant{AccountID: uuid.New(), EnvID: uuid.New(), AppID: uuid.New()},
		},
		Config: *sv2.InitConfig(&sv2.Config{EventIDs: []ulid.ULID{ulid.MustNew(ulid.Now(), rand.Reader)}, RequestVersion: 2}),
	}
	_, err = rs.Create(ctx, sv2.CreateState{Metadata: md, Events: []json.RawMessage{json.RawMessage(`{"name":"start"}`)}})
	require.NoError(t, err)

	q := &subscribeQueue{Queue: base}
	pm := &subscribePauses{written: map[uuid.UUID]*state.Pause{}}

	return &subscribeFixture{
		t: t, ctx: ctx, mr: mr, q: q, shard: shard, rs: rs, pm: pm, md: md,
		e: &executor{log: logger.VoidLogger(), smv2: rs, queue: q, shards: registry, pm: pm, tracerProvider: tracing.NewNoopTracerProvider()},
	}
}

func (f *subscribeFixture) pause(key string, stream int) *state.Pause {
	name := "agent/message"

	return &state.Pause{
		ID: uuid.New(), WorkspaceID: f.md.ID.Tenant.EnvID, Event: &name,
		Identifier: sv2.NewPauseIdentifier(f.md.ID), Incoming: "step",
		Subscribe: &state.PauseSubscribe{Key: key, Stream: stream},
	}
}

func (f *subscribeFixture) event() event.InternalEvent {
	f.events++

	return event.InternalEvent{ID: ulid.MustNew(uint64(f.events), rand.Reader), Event: event.Event{Name: "agent/message"}}
}

func (f *subscribeFixture) deliverEvent(key string, evt event.InternalEvent) error {
	return f.e.deliverSubscription(f.ctx, evt, f.pause(key, 0))
}

func (f *subscribeFixture) deliver(key string) event.InternalEvent {
	f.t.Helper()
	evt := f.event()
	require.NoError(f.t, f.deliverEvent(key, evt))

	return evt
}

func (f *subscribeFixture) burst(key string, n int) {
	for i := 0; i < n; i++ {
		f.deliver(key)
	}
}

func (f *subscribeFixture) op(key string, mode string, id string, timeout string, streams ...state.SubscribeStream) error {
	gen := state.GeneratorOpcode{Op: enums.OpcodeWaitForEvent, ID: id}
	opts := &state.WaitForEventOpts{Event: "agent/message", Subscribe: &state.SubscribeOpts{Key: key, Mode: mode, Timeout: timeout, Streams: streams}}

	return f.e.handleGeneratorSubscribe(f.ctx, &mockRunContext{md: f.md}, gen, queue.PayloadEdge{Edge: inngest.Edge{Incoming: "step"}}, f.group, opts)
}

func (f *subscribeFixture) next(key string, id string) {
	f.t.Helper()
	require.NoError(f.t, f.op(key, state.SubscribeNext, id, ""))
}

func (f *subscribeFixture) stack() []string {
	f.t.Helper()
	stack, err := f.rs.LoadStack(f.ctx, f.md.ID)
	require.NoError(f.t, err)

	return stack
}

// messages are the message steps of the inbox in stack order.
func (f *subscribeFixture) messages(key string) []string {
	var ids []string
	prefix := subscriptionMessagePrefix + subscriptionField(key) + ":"
	for _, id := range f.stack() {
		if strings.HasPrefix(id, prefix) {
			ids = append(ids, id)
		}
	}

	return ids
}

// field is the state of the inbox in the run's hash: "" when idle.
func (f *subscribeFixture) field(key string) string {
	f.t.Helper()
	for _, k := range f.mr.Keys() {
		if strings.Contains(k, ":subs:") {
			return f.mr.HGet(k, subscriptionField(key))
		}
	}

	return ""
}

func (f *subscribeFixture) hasStep(id string) bool {
	steps, err := f.rs.LoadSteps(f.ctx, f.md.ID)
	require.NoError(f.t, err)
	_, ok := steps[id]

	return ok
}

func (f *subscribeFixture) stepData(id string) string {
	f.t.Helper()
	steps, err := f.rs.LoadSteps(f.ctx, f.md.ID)
	require.NoError(f.t, err)

	return string(steps[id])
}

func (f *subscribeFixture) item(kind string) *queue.Item {
	for i := len(f.q.accepted) - 1; i >= 0; i-- {
		if f.q.accepted[i].Kind == kind {
			return &f.q.accepted[i]
		}
	}

	return nil
}

func (f *subscribeFixture) discoveries(key string) int {
	n := 0
	for _, item := range f.q.accepted {
		if v, _ := item.Metadata[subscriptionWakeMetaKey].(string); strings.HasPrefix(v, subscriptionField(key)+":") {
			n++
		}
	}

	return n
}

// startWake is Execute starting the discovery a wake queued.
func (f *subscribeFixture) startWake(item queue.Item) {
	f.t.Helper()
	require.NoError(f.t, f.e.clearSubscriptionWait(f.ctx, sv2.V1FromMetadata(f.md), item))
}

func (f *subscribeFixture) fireTimer(item queue.Item, opID string) bool {
	f.t.Helper()
	won, err := f.e.claimSubscriptionTimeout(f.ctx, sv2.V1FromMetadata(f.md), item, opID)
	require.NoError(f.t, err)

	return won
}

// Messages that arrive while nothing waits cost no step. A next op flushes them
// all in arrival order and queues one discovery; the op is never saved as a step.
func TestBufferedMessagesFlushOnNext(t *testing.T) {
	f := newSubscribeFixture(t)

	var want []string
	for i := 0; i < 3; i++ {
		want = append(want, subscriptionMessageID("inbox", f.deliver("inbox").GetInternalID().String()))
	}
	require.Empty(t, f.messages("inbox"))
	require.Empty(t, f.q.accepted)

	f.next("inbox", "next-1")

	require.Equal(t, want, f.messages("inbox"))
	require.Equal(t, 1, f.discoveries("inbox"))
	require.False(t, f.hasStep("next-1"))
	require.Empty(t, f.field("inbox"))
}

// A next op with nothing buffered waits. The delivery saves the message as a step
// and queues the discovery, but leaves the flag for the discovery to clear when it
// starts; messages after that buffer again.
func TestWaitingNextIsWokenByTheDelivery(t *testing.T) {
	f := newSubscribeFixture(t)

	f.next("inbox", "next-1")
	require.Equal(t, "next-1", f.waiting("inbox"))

	evt := f.deliver("inbox")
	require.Equal(t, []string{subscriptionMessageID("inbox", evt.GetInternalID().String())}, f.messages("inbox"))
	require.Equal(t, 1, f.discoveries("inbox"))
	require.Equal(t, "next-1", f.waiting("inbox"))

	f.startWake(*f.item(queue.KindEdge))
	require.Empty(t, f.field("inbox"))

	f.deliver("inbox")
	require.Len(t, f.messages("inbox"), 1)
	require.Contains(t, f.field("inbox"), `"i"`)
}

// The same event is the same message: delivered twice while buffered it is one
// message, and delivered twice to a waiting op it is one step and one discovery.
func TestRedeliveredEventIsOneMessage(t *testing.T) {
	f := newSubscribeFixture(t)

	evt := f.event()
	require.NoError(t, f.deliverEvent("inbox", evt))
	require.NoError(t, f.deliverEvent("inbox", evt))
	f.next("inbox", "next-1")
	require.Len(t, f.messages("inbox"), 1)

	f.next("inbox", "next-2")
	evt = f.event()
	require.NoError(t, f.deliverEvent("inbox", evt))
	require.NoError(t, f.deliverEvent("inbox", evt))
	require.Len(t, f.messages("inbox"), 2)
	require.Equal(t, 2, f.discoveries("inbox"), "one discovery for the flush and one for the wait")
}

// A crash after the delivery script and before the enqueue loses nothing: the
// message is a step, the op still waits, and the redelivery queues the wake.
func TestCrashBetweenTheDeliveryAndTheEnqueue(t *testing.T) {
	f := newSubscribeFixture(t)

	f.next("inbox", "next-1")
	evt := f.event()
	f.q.fail = errors.New("queue down")
	require.Error(t, f.deliverEvent("inbox", evt))
	f.q.fail = nil

	require.Len(t, f.messages("inbox"), 1)
	require.Equal(t, "next-1", f.waiting("inbox"))
	require.Zero(t, f.discoveries("inbox"))

	require.NoError(t, f.deliverEvent("inbox", evt))
	require.Len(t, f.messages("inbox"), 1)
	require.Equal(t, 1, f.discoveries("inbox"))
}

// A wake that runs twice (a retry) does not clear a wait that started after it.
func TestRerunOfAWakeLeavesALaterWait(t *testing.T) {
	f := newSubscribeFixture(t)

	f.next("inbox", "next-1")
	f.deliver("inbox")
	wake := *f.item(queue.KindEdge)
	f.startWake(wake)

	f.next("inbox", "next-2")
	f.startWake(wake)

	require.Equal(t, "next-2", f.waiting("inbox"))
}

// A next op sent again (the SDK repeats the pending op after a replay) leaves one
// wait, and one timer.
func TestNextSentAgainWaitsOnce(t *testing.T) {
	f := newSubscribeFixture(t)

	require.NoError(t, f.op("inbox", state.SubscribeNext, "next-1", "1h"))
	require.NoError(t, f.op("inbox", state.SubscribeNext, "next-1", "1h"))

	require.Equal(t, "next-1", f.waiting("inbox"))
	timers := 0
	for _, item := range f.q.accepted {
		if item.Kind == queue.KindSleep {
			timers++
		}
	}
	require.Equal(t, 1, timers)
}

// The timer wins only while the op still waits: it saves null under the op's ID
// and clears the flag. Its retry after winning wins again. A message that won
// first leaves the timer nothing to do.
func TestTimedNext(t *testing.T) {
	t.Run("the timer wins", func(t *testing.T) {
		f := newSubscribeFixture(t)

		require.NoError(t, f.op("inbox", state.SubscribeNext, "next-1", "1h"))
		timer := *f.item(queue.KindSleep)
		require.Equal(t, "next-1", timer.Payload.(queue.PayloadEdge).Edge.Outgoing)

		require.True(t, f.fireTimer(timer, "next-1"))
		require.True(t, f.hasStep("next-1"))
		require.Empty(t, f.field("inbox"))
		require.True(t, f.fireTimer(timer, "next-1"), "a retry of the winning timer")

		f.deliver("inbox")
		require.Empty(t, f.messages("inbox"), "after the timer a message buffers")
	})

	t.Run("a message wins", func(t *testing.T) {
		f := newSubscribeFixture(t)

		require.NoError(t, f.op("inbox", state.SubscribeNext, "next-1", "1h"))
		timer := *f.item(queue.KindSleep)
		f.deliver("inbox")
		f.startWake(*f.item(queue.KindEdge))

		require.False(t, f.fireTimer(timer, "next-1"))
		require.False(t, f.hasStep("next-1"))
	})

	t.Run("a later op is not timed out by an earlier timer", func(t *testing.T) {
		f := newSubscribeFixture(t)

		require.NoError(t, f.op("inbox", state.SubscribeNext, "next-1", "1h"))
		timer := *f.item(queue.KindSleep)
		f.deliver("inbox")
		f.startWake(*f.item(queue.KindEdge))
		f.next("inbox", "next-2")

		require.False(t, f.fireTimer(timer, "next-1"))
		require.Equal(t, "next-2", f.waiting("inbox"))
	})
}

// An inbox's ops touch only its own field.
func TestTwoInboxesAreSeparate(t *testing.T) {
	f := newSubscribeFixture(t)

	f.burst("a", 2)
	f.burst("b", 3)
	f.next("a", "next-a")

	require.Len(t, f.messages("a"), 2)
	require.Empty(t, f.messages("b"))

	f.next("b", "next-b")
	require.Len(t, f.messages("b"), 3)
}

// all flushes what is buffered and never waits; the op is answered by a step.
func TestAllFlushesWithoutWaiting(t *testing.T) {
	f := newSubscribeFixture(t)

	f.burst("inbox", 2)
	require.NoError(t, f.op("inbox", state.SubscribeAll, "all-1", ""))

	require.Len(t, f.messages("inbox"), 2)
	require.True(t, f.hasStep("all-1"))
	require.Contains(t, f.stepData("all-1"), `"total":2`)
	require.Empty(t, f.field("inbox"))

	require.NoError(t, f.op("inbox", state.SubscribeAll, "all-2", ""))
	require.True(t, f.hasStep("all-2"))
}

// close flushes the buffer, closes the inbox and answers with one step. Closing
// again, and a delivery after it, are harmless.
func TestCloseFlushesAndDropsLaterDeliveries(t *testing.T) {
	f := newSubscribeFixture(t)

	f.burst("inbox", 2)
	require.NoError(t, f.op("inbox", state.SubscribeClose, "close-1", "", state.SubscribeStream{Stream: 0, Event: "agent/message"}))
	require.Len(t, f.messages("inbox"), 2)
	require.True(t, f.hasStep("close-1"))
	require.JSONEq(t, `"closed"`, f.field("inbox"))

	f.deliver("inbox")
	require.Len(t, f.messages("inbox"), 2)

	require.NoError(t, f.op("inbox", state.SubscribeClose, "close-2", ""))
	require.JSONEq(t, `"closed"`, f.field("inbox"))
}

// Two inboxes closing, and the same one closing twice, each leave the others' state alone.
func TestOverlappingCloses(t *testing.T) {
	f := newSubscribeFixture(t)

	f.burst("a", 1)
	f.burst("b", 1)
	require.NoError(t, f.op("a", state.SubscribeClose, "close-a", ""))
	require.NoError(t, f.op("a", state.SubscribeClose, "close-a2", ""))

	require.JSONEq(t, `"closed"`, f.field("a"))
	require.Contains(t, f.field("b"), `"i"`)
	f.next("b", "next-b")
	require.Len(t, f.messages("b"), 1)
}

// A next that reaches an inbox already closed is answered with null, which the SDK
// reads as the end.
func TestNextOnAClosedInboxIsAnsweredNull(t *testing.T) {
	f := newSubscribeFixture(t)

	require.NoError(t, f.op("inbox", state.SubscribeClose, "close-1", ""))
	f.next("inbox", "next-1")

	require.Zero(t, f.discoveries("inbox"))
	require.JSONEq(t, `{"data":null}`, f.stepData("next-1"))
}

// Open writes one pause per source, with the listed streams, in one op; stop
// deletes exactly the pauses it lists.
func TestOpenAndStopSources(t *testing.T) {
	f := newSubscribeFixture(t)

	sources := []state.SubscribeStream{{Stream: 0, Event: "agent/a"}, {Stream: 1, Event: "agent/b"}, {Stream: 2, Event: "agent/c"}}
	require.NoError(t, f.op("inbox", state.SubscribeOpen, "open-1", "", sources...))
	require.Len(t, f.pm.written, 3)
	require.True(t, f.hasStep("open-1"))

	require.NoError(t, f.op("inbox", state.SubscribeStop, "stop-1", "", sources[1:]...))
	require.Len(t, f.pm.written, 1)
	require.Len(t, f.pm.deleted, 2)
}

func TestOpenRefusesBadSources(t *testing.T) {
	f := newSubscribeFixture(t)

	err := f.op("inbox", state.SubscribeOpen, "open-1", "", state.SubscribeStream{Stream: 0, Event: "a"}, state.SubscribeStream{Stream: 0, Event: "b"})
	require.ErrorIs(t, err, state.ErrSubscriptionLimit)

	err = f.op(strings.Repeat("k", consts.MaxSubscriptionKeyLength+1), state.SubscribeOpen, "open-2", "", state.SubscribeStream{Event: "a"})
	require.ErrorIs(t, err, state.ErrSubscriptionLimit)

	f.pm.written = map[uuid.UUID]*state.Pause{}
	for i := 0; i < consts.MaxSubscriptionPausesPerRun; i++ {
		f.pm.written[uuid.New()] = &state.Pause{}
	}
	err = f.op("inbox", state.SubscribeOpen, "open-3", "", state.SubscribeStream{Event: "a"})
	require.ErrorIs(t, err, state.ErrSubscriptionLimit)
}

// A full buffer closes the inbox after flushing it, and shows a waiting op the end.
func TestFullBufferClosesTheInbox(t *testing.T) {
	f := newSubscribeFixture(t)

	f.burst("inbox", consts.MaxSubscriptionMessages)
	f.deliver("inbox")

	require.JSONEq(t, `"closed"`, f.field("inbox"))
	require.Len(t, f.messages("inbox"), consts.MaxSubscriptionMessages)
	require.True(t, f.hasStep(subscriptionClosedID("inbox")))
}

func TestLimitCloseWakesAWaitingNext(t *testing.T) {
	f := newSubscribeFixture(t)

	f.next("inbox", "next-1")
	// The run's step limit is reached by saving unrelated steps.
	limits := f.e
	limits.steplimit = func(sv2.ID) int { return 3 }
	for i := 0; i < 3; i++ {
		_, err := f.rs.SaveStep(f.ctx, f.md.ID, fmt.Sprintf("step-%d", i), []byte(`1`))
		require.NoError(t, err)
	}

	f.deliver("inbox")

	require.JSONEq(t, `"closed"`, f.field("inbox"))
	require.True(t, f.hasStep(subscriptionClosedID("inbox")))
	require.Empty(t, f.messages("inbox"))
	wake := f.item(queue.KindEdge)
	require.Equal(t, subscriptionClosedID("inbox"), wake.Payload.(queue.PayloadEdge).Edge.Outgoing)
}

// A flush that would pass a limit saves what fits, drops the rest and closes.
func TestFlushEnforcesTheRunLimit(t *testing.T) {
	f := newSubscribeFixture(t)

	f.burst("inbox", 5)
	f.e.steplimit = func(sv2.ID) int { return 6 }
	f.next("inbox", "next-1")

	require.Len(t, f.messages("inbox"), 3, "half of the step limit")
	require.JSONEq(t, `"closed"`, f.field("inbox"))
	require.True(t, f.hasStep(subscriptionClosedID("inbox")))
	require.Equal(t, 1, f.discoveries("inbox"))
}

// A delivery to a run that no longer exists is dropped without writing anything.
func TestDeliveryToAFinishedRun(t *testing.T) {
	f := newSubscribeFixture(t)

	require.NoError(t, f.rs.Delete(f.ctx, f.md.ID))
	err := f.deliverEvent("inbox", f.event())
	require.ErrorIs(t, err, state.ErrRunNotFound)
	require.Empty(t, f.field("inbox"))
}

// Message order is arrival order, and reading it back is reading the stack.
func TestMessagesAreInArrivalOrder(t *testing.T) {
	f := newSubscribeFixture(t)

	var want []string
	for i := 0; i < 10; i++ {
		want = append(want, subscriptionMessageID("inbox", f.deliver("inbox").GetInternalID().String()))
	}
	f.next("inbox", "next-1")

	require.Equal(t, want, f.messages("inbox"))
}

// A view that skips a message sends its next op again under the same ID. The
// delivery that wakes it the second time is a wake of its own, not a repeat of the
// first.
func TestNextSentAgainAfterAWakeIsWokenAgain(t *testing.T) {
	f := newSubscribeFixture(t)

	f.next("inbox", "next-1")
	f.deliver("inbox")
	f.startWake(*f.item(queue.KindEdge))

	f.next("inbox", "next-1")
	f.deliver("inbox")

	require.Equal(t, 2, f.discoveries("inbox"))
}

// A next op beside an op that is not only a wait leaves the run idle for nothing:
// that op's discovery runs the function again, which sends the next op again.
func TestNextBesideABusyOpDoesNothing(t *testing.T) {
	f := newSubscribeFixture(t)

	ops := []*state.GeneratorOpcode{
		{Op: enums.OpcodeStepRun, ID: "step"},
		{Op: enums.OpcodeWaitForEvent, ID: "next-1", Opts: map[string]any{"event": "e", "subscribe": map[string]any{"key": "inbox", "mode": "next"}}},
	}
	require.True(t, opGroups(ops).OtherGroup.SubscribeBusy)

	f.group = opGroups(ops).PriorityGroup
	f.burst("inbox", 2)
	f.next("inbox", "next-1")

	require.Empty(t, f.messages("inbox"))
	require.Zero(t, f.discoveries("inbox"))

	only := opGroups(ops[1:])
	require.False(t, only.PriorityGroup.SubscribeBusy)
}

// A burst that arrives while an op waits is one discovery: the flag stays until
// the discovery starts, and every message of the burst finds the same wait.
func TestBurstWhileWaitingIsOneDiscovery(t *testing.T) {
	f := newSubscribeFixture(t)

	f.next("inbox", "next-1")
	f.burst("inbox", 10)

	require.Len(t, f.messages("inbox"), 10)
	require.Equal(t, 1, f.discoveries("inbox"))
}

// waiting is the op the inbox waits for: "" when it waits for none.
func (f *subscribeFixture) waiting(key string) string {
	f.t.Helper()

	var state struct {
		Waiting string `json:"waiting"`
	}
	_ = json.Unmarshal([]byte(f.field(key)), &state)

	return state.Waiting
}

// The inbox hash goes with the run.
func TestInboxStateIsDeletedWithTheRun(t *testing.T) {
	f := newSubscribeFixture(t)

	f.next("inbox", "next-1")
	require.NotEmpty(t, f.field("inbox"))

	require.NoError(t, f.rs.Delete(f.ctx, f.md.ID))
	for _, k := range f.mr.Keys() {
		require.NotContains(t, k, ":subs:")
	}
}

// A timer that won and is retried still wins when a message was buffered between
// the attempts, so its discovery is not dropped.
func TestRetriedTimerWinsAfterABufferedMessage(t *testing.T) {
	f := newSubscribeFixture(t)

	require.NoError(t, f.op("inbox", state.SubscribeNext, "next-1", "1h"))
	timer := *f.item("sleep")
	require.True(t, f.fireTimer(timer, "next-1"))

	f.deliver("inbox")

	require.True(t, f.fireTimer(timer, "next-1"))
}
