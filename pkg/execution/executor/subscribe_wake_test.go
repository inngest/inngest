package executor

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	mrand "math/rand"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
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

// These tests drive step.subscribe wakes through the real queue and run state
// (miniredis, the real Lua), one step at a time, so every interleaving is a
// sequence the test writes down. A request is the function's request: it reads
// the run's state (read) and ends with a waiting read (end), which is when the
// executor sets the waiting flag. A wake is the discovery item a delivery or
// that end queues; the queue removes it (dequeue) after its request was handled.

type recordingQueue struct {
	queue.Queue
	accepted []string
	rejected int
}

func (q *recordingQueue) Enqueue(ctx context.Context, item queue.Item, at time.Time, opts queue.EnqueueOpts) error {
	err := q.Queue.Enqueue(ctx, item, at, opts)
	switch {
	case err == nil:
		q.accepted = append(q.accepted, *item.JobID)
	case errors.Is(err, queue.ErrQueueItemExists):
		q.rejected++
	}
	return err
}

// hookedRunService runs a function after the waiting flag is set and after the
// recheck loads the stack, to land a delivery exactly there.
type hookedRunService struct {
	sv2.RunService
	afterSet   func()
	afterStack func()
	// lostClearReply makes the next compare-and-delete apply and then report an error.
	lostClearReply bool
}

func (h *hookedRunService) SetSubscriptionWaiting(ctx context.Context, id sv2.ID, hash string, token string) (string, error) {
	res, err := h.RunService.(sv2.SubscriptionWaiter).SetSubscriptionWaiting(ctx, id, hash, token)
	if h.afterSet != nil {
		h.afterSet()
	}
	return res, err
}

func (h *hookedRunService) GetSubscriptionWaiting(ctx context.Context, id sv2.ID, hash string) (string, bool, error) {
	return h.RunService.(sv2.SubscriptionWaiter).GetSubscriptionWaiting(ctx, id, hash)
}

func (h *hookedRunService) ClearSubscriptionWaiting(ctx context.Context, id sv2.ID) error {
	return h.RunService.(sv2.SubscriptionWaiter).ClearSubscriptionWaiting(ctx, id)
}

func (h *hookedRunService) LoadStack(ctx context.Context, id sv2.ID) ([]string, error) {
	stack, err := h.RunService.LoadStack(ctx, id)
	if h.afterStack != nil {
		h.afterStack()
	}
	return stack, err
}

type wakeFixture struct {
	t      *testing.T
	ctx    context.Context
	mr     *miniredis.Miniredis
	e      *executor
	rq     *recordingQueue
	shard  queue.QueueShard
	rs     sv2.RunService
	hooks  *hookedRunService
	md     sv2.Metadata
	events int

	// delivered counts the messages saved per subscription.
	delivered map[string]int
	// saw is how many messages the function's last request read, per
	// subscription.
	saw map[string]int
	// requests counts the requests the function made, per subscription.
	requests map[string]int
}

func newWakeFixture(t *testing.T, forceStepPlan bool) *wakeFixture {
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
	hooks := &hookedRunService{RunService: rs}

	shard := redis_state.NewQueueShard("local", redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey))
	registry, err := queue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	q, err := queue.New(ctx, "subscribe-wake", registry)
	require.NoError(t, err)

	eventID := ulid.MustNew(ulid.Now(), rand.Reader)
	md := sv2.Metadata{
		ID: sv2.ID{
			RunID:      ulid.MustNew(ulid.Now(), rand.Reader),
			FunctionID: uuid.New(),
			Tenant:     sv2.Tenant{AccountID: uuid.New(), EnvID: uuid.New(), AppID: uuid.New()},
		},
		Config: *sv2.InitConfig(&sv2.Config{EventIDs: []ulid.ULID{eventID}, RequestVersion: 2, ForceStepPlan: forceStepPlan}),
	}
	_, err = rs.Create(ctx, sv2.CreateState{Metadata: md, Events: []json.RawMessage{json.RawMessage(`{"name":"start"}`)}})
	require.NoError(t, err)
	_, err = rs.SaveStep(ctx, md.ID, "open", []byte(`{"data":{"opened":true}}`))
	require.NoError(t, err)

	rq := &recordingQueue{Queue: q}
	return &wakeFixture{
		t: t, ctx: ctx, mr: mr, rq: rq, shard: shard, rs: rs, hooks: hooks, md: md,
		e: &executor{
			log: logger.VoidLogger(), smv2: hooks, queue: rq, shards: registry,
			pm: &limitPauseManager{}, tracerProvider: tracing.NewNoopTracerProvider(),
		},
		delivered: map[string]int{}, saw: map[string]int{}, requests: map[string]int{},
	}
}

func (f *wakeFixture) pause(key string) *state.Pause {
	name := "agent/message"
	return &state.Pause{
		ID: uuid.New(), WorkspaceID: f.md.ID.Tenant.EnvID, Event: &name,
		Identifier: sv2.NewPauseIdentifier(f.md.ID), Incoming: "step",
		Subscribe: &state.PauseSubscribe{Key: key},
	}
}

// deliver saves one more message of the subscription, as an event match does.
func (f *wakeFixture) deliver(key string) {
	f.t.Helper()
	f.events++
	evt := event.InternalEvent{ID: ulid.MustNew(uint64(f.events), rand.Reader), Event: event.Event{Name: "agent/message"}}
	require.NoError(f.t, f.e.deliverSubscription(f.ctx, evt, f.pause(key), logger.VoidLogger()))
	f.delivered[key]++
}

// flush turns what the subscription buffered into steps, as the reader's next
// op does.
func (f *wakeFixture) flush(key string) {
	f.t.Helper()
	require.NoError(f.t, f.e.flushSubscription(f.ctx, f.md.ID, key))
}

func (f *wakeFixture) burst(key string, n int) {
	for i := 0; i < n; i++ {
		f.deliver(key)
	}
}

func (f *wakeFixture) wakePrefix(key string) string {
	return fmt.Sprintf("%s-subscription-wake-%s-", f.md.ID.RunID, subscriptionKeyHash(key))
}

// wakes are the wake items of the subscription in the queue, queued or leased.
func (f *wakeFixture) wakes(key string) []queue.QueueItem {
	var items []queue.QueueItem
	for _, id := range f.rq.accepted {
		if !strings.HasPrefix(id, f.wakePrefix(key)) {
			continue
		}
		qi, err := f.shard.LoadQueueItem(f.ctx, queue.HashID(f.ctx, id))
		if errors.Is(err, queue.ErrQueueItemNotFound) {
			continue
		}
		require.NoError(f.t, err)
		items = append(items, *qi)
	}
	return items
}

func (f *wakeFixture) queued(key string) bool {
	return len(f.wakes(key)) > 0
}

// accepts is how many wake items were ever queued for the subscription.
func (f *wakeFixture) accepts(key string) int {
	n := 0
	for _, id := range f.rq.accepted {
		if strings.HasPrefix(id, f.wakePrefix(key)) {
			n++
		}
	}
	return n
}

// request is one request of the function for a subscription.
type request struct {
	f   *wakeFixture
	key string
	// qi is the wake item the request runs for, nil for any other request.
	qi     *queue.QueueItem
	read   bool
	closed bool
	ended  bool
}

// begin starts a request that is not a wake: the first one, or the one a timer
// runs.
func (f *wakeFixture) begin(key string) *request {
	f.requests[key]++
	return &request{f: f, key: key}
}

// beginAs starts a request the way Execute does: the start hook runs before the
// request reads state. item is the wake item, empty for any other request.
func (f *wakeFixture) beginAs(key string, item queue.Item) *request {
	f.t.Helper()
	require.NoError(f.t, f.e.startSubscriptionWake(f.ctx, f.md.ID, item))
	return f.begin(key)
}

// start leases a queued wake and starts its request.
func (f *wakeFixture) start(key string) *request {
	f.t.Helper()
	for _, qi := range f.wakes(key) {
		if qi.LeaseID != nil {
			continue
		}
		lease, err := f.shard.Lease(f.ctx, qi, 30*time.Second, time.Now())
		require.NoError(f.t, err)
		qi.LeaseID = lease
		r := f.beginAs(key, qi.Data)
		r.qi = &qi
		return r
	}
	return nil
}

// readState is the request reading the run's state: what it holds now is what
// the function sees when it blocks.
func (r *request) readState() {
	r.f.t.Helper()
	if r.read {
		return
	}
	stack, err := r.f.rs.LoadStack(r.f.ctx, r.f.md.ID)
	require.NoError(r.f.t, err)
	r.f.saw[r.key] = len(subscriptionMessageIDs(stack, r.key))
	r.closed = slices.Contains(stack, subscriptionClosedID(r.key))
	r.read = true
}

// waitOp is the waiting op the SDK returns for a read that found nothing.
func (r *request) waitOp() state.GeneratorOpcode {
	return state.GeneratorOpcode{
		Op: enums.OpcodeNone, ID: fmt.Sprintf("%s:%d", r.key, r.f.saw[r.key]+1),
		Opts: map[string]any{"subscribe": map[string]any{"key": r.key, "seq": r.f.saw[r.key] + 1}},
	}
}

// handle is the executor handling the response of the request: the waiting op.
func (r *request) handle() {
	r.f.t.Helper()
	r.readState()
	// A subscription closed when the request read ends the read instead of waiting.
	if r.closed {
		r.ended = true
		return
	}
	rc := &mockRunContext{md: r.f.md}
	err := r.f.e.handleGeneratorSubscribeWait(r.f.ctx, rc, r.waitOp(), queue.PayloadEdge{Edge: inngest.Edge{Incoming: "step"}})
	require.NoError(r.f.t, err)
	r.ended = true
}

// finish is the queue removing the wake item once its request is handled.
func (r *request) finish() {
	r.f.t.Helper()
	if r.qi != nil {
		require.NoError(r.f.t, r.f.rq.Dequeue(r.f.ctx, r.f.shard.Name(), *r.qi))
	}
}

// end handles the response and removes the item.
func (r *request) end() {
	r.handle()
	r.finish()
}

// settle runs wakes until none is queued. The bound catches a wake that never settles.
func (f *wakeFixture) settle(keys ...string) {
	f.t.Helper()
	for i := 0; i < 1000; i++ {
		ran := false
		for _, key := range keys {
			if r := f.start(key); r != nil {
				r.end()
				ran = true
			}
		}
		if !ran {
			return
		}
	}
	f.t.Fatal("wakes did not settle")
}

// requireNothingLost is the end state of every cell: no wake is queued, the
// function's last request saw every message that was saved, and the run is
// waiting (the flag is set, so the next message wakes it).
func (f *wakeFixture) requireNothingLost(keys ...string) {
	f.t.Helper()
	waiter := f.rs.(sv2.SubscriptionWaiter)
	for _, key := range keys {
		require.False(f.t, f.queued(key), "wake of %q left queued", key)
		require.Equal(f.t, f.delivered[key], f.saw[key], "messages of %q the function never saw", key)
		_, waiting, err := waiter.GetSubscriptionWaiting(f.ctx, f.md.ID, subscriptionKeyHash(key))
		require.NoError(f.t, err)
		require.True(f.t, waiting, "the run is not waiting on %q, so the next message would not wake it", key)
	}
}

// A run that is busy has no flag: messages saved while a request runs, or a
// lone step runs inline in it, queue nothing, so no discovery overlaps the
// step. The request that ends waiting finds them with one wake.
func TestWakeBusyRunGetsNoDiscoveries(t *testing.T) {
	f := newWakeFixture(t, false)
	r := f.begin("inbox")
	r.readState()

	// The request runs a step inline while messages arrive.
	f.burst("inbox", 20)
	require.Zero(t, f.accepts("inbox"))
	require.Zero(t, f.rq.rejected)

	r.end()
	require.Equal(t, 1, f.accepts("inbox"), "the end of the request wakes the run once")

	f.settle("inbox")
	f.requireNothingLost("inbox")
}

// A delivery during every moment of an inline step queues nothing, in both of
// the run's modes, however many requests the function makes.
func TestWakeDeliveryDuringInlineStepNeverQueues(t *testing.T) {
	for name, forceStepPlan := range map[string]bool{"sequential": false, "parallel": true} {
		t.Run(name, func(t *testing.T) {
			f := newWakeFixture(t, forceStepPlan)

			// Wait, wake, run a step inline, wait again: three times.
			f.begin("inbox").end()
			for i := 0; i < 3; i++ {
				f.deliver("inbox")
				w := f.start("inbox")
				require.NotNil(t, w)
				before := f.accepts("inbox")
				w.readState()
				// The step runs inline in this request.
				f.burst("inbox", 5)
				require.Equal(t, before, f.accepts("inbox"), "a delivery queued a wake while a step ran inline")
				w.end()
				f.settle("inbox")
			}
			f.requireNothingLost("inbox")
		})
	}
}

// Set the flag, then the message: the delivery takes the flag and queues one wake.
func TestWakeFlagThenMessage(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("inbox").end()
	require.Zero(t, f.accepts("inbox"))

	f.deliver("inbox")
	require.Equal(t, 1, f.accepts("inbox"))

	f.settle("inbox")
	f.requireNothingLost("inbox")
}

// The message, then the flag: the request read its state before the message
// landed, so the executor's recheck finds it and wakes the run itself.
func TestWakeMessageThenFlag(t *testing.T) {
	f := newWakeFixture(t, false)
	r := f.begin("inbox")
	r.readState()
	f.deliver("inbox")
	require.Zero(t, f.accepts("inbox"))

	r.end()
	require.Equal(t, 1, f.accepts("inbox"))

	f.settle("inbox")
	f.requireNothingLost("inbox")
}

// The message lands between setting the flag and the recheck: the delivery
// takes the flag, the recheck finds it gone, and exactly one wake is queued.
func TestWakeMessageBetweenFlagAndRecheck(t *testing.T) {
	f := newWakeFixture(t, false)
	r := f.begin("inbox")
	r.readState()

	f.hooks.afterSet = func() {
		f.hooks.afterSet = nil
		f.deliver("inbox")
	}
	r.end()
	require.Equal(t, 1, f.accepts("inbox"))
	require.Equal(t, 1, f.rq.rejected, "the recheck refuses the delivery's wake of the same wait")

	f.settle("inbox")
	f.requireNothingLost("inbox")
}

// The message lands right after the recheck counted: the flag is set, so the
// delivery takes it.
func TestWakeMessageAfterRecheck(t *testing.T) {
	f := newWakeFixture(t, false)
	r := f.begin("inbox")
	r.readState()

	f.hooks.afterStack = func() {
		f.hooks.afterStack = nil
		f.deliver("inbox")
	}
	r.end()
	require.Equal(t, 1, f.accepts("inbox"))

	f.settle("inbox")
	f.requireNothingLost("inbox")
}

// A message that lands while the wake's own request is handled, after the item
// would be removed last: the wake's end sets the flag and the delivery that
// follows queues a wake while the earlier item is still there. Each wait has
// its own job ID, so the two never collide.
func TestWakeDeliveryWhileThePreviousItemIsStillQueued(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("inbox").end()

	f.deliver("inbox")
	w := f.start("inbox")
	w.readState()
	w.handle()

	// The queue has not removed the item yet.
	require.Len(t, f.wakes("inbox"), 1)
	f.deliver("inbox")
	require.Equal(t, 2, f.accepts("inbox"))
	require.Zero(t, f.rq.rejected)
	w.finish()

	f.settle("inbox")
	f.requireNothingLost("inbox")
}

// Twenty messages while the run waits: one wake, however many arrive before it runs.
func TestWakeBurstWhileBlocked(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("inbox").end()
	f.burst("inbox", 20)

	require.Equal(t, 1, f.accepts("inbox"))
	require.Equal(t, 19, f.rq.rejected, "later messages read the same flag and are refused")

	f.settle("inbox")
	f.requireNothingLost("inbox")
	require.Equal(t, 2, f.requests["inbox"])
	require.Equal(t, 20, f.saw["inbox"])
}

// The wake is an ordinary edge item with no idempotency period.
func TestWakeItemShape(t *testing.T) {
	for _, forceStepPlan := range []bool{false, true} {
		f := newWakeFixture(t, forceStepPlan)
		before, err := f.rs.LoadMetadata(f.ctx, f.md.ID)
		require.NoError(t, err)
		f.begin("inbox").end()
		f.deliver("inbox")

		items := f.wakes("inbox")
		require.Len(t, items, 1)
		qi := items[0]
		require.Equal(t, queue.KindEdge, qi.Data.Kind)
		require.NotNil(t, qi.IdempotencyPeriod)
		require.Equal(t, subscriptionWakeIdempotency, *qi.IdempotencyPeriod)
		require.Equal(t, "step", qi.Data.Payload.(queue.PayloadEdge).Edge.Incoming)
		// The run's metadata says whether the SDK plans before it runs, not the item.
		require.Zero(t, qi.Data.ParallelMode)
		require.Empty(t, qi.Data.GetConcurrencyKeys())

		md, err := f.rs.LoadMetadata(f.ctx, f.md.ID)
		require.NoError(t, err)
		require.Equal(t, before.Config.ForceStepPlan, md.Config.ForceStepPlan)
	}
}

// Messages arrive in bursts while the function waits between them: one wake per
// wait, never one per message.
func TestWakeCountFollowsWaitsNotMessages(t *testing.T) {
	for name, forceStepPlan := range map[string]bool{"sequential": false, "already parallel": true} {
		t.Run(name, func(t *testing.T) {
			f := newWakeFixture(t, forceStepPlan)
			require.NoError(t, f.rs.SavePending(f.ctx, f.md.ID, []string{"bg-1", "bg-2"}))
			f.begin("inbox").end()

			const bursts, size = 8, 20
			for b := 0; b < bursts; b++ {
				f.burst("inbox", size)
				f.settle("inbox")
			}

			f.requireNothingLost("inbox")
			require.Equal(t, bursts*size, f.delivered["inbox"])
			require.Equal(t, bursts, f.accepts("inbox"))
		})
	}
}

// Two subscriptions in one run have a flag each: a message of one never wakes
// the other.
func TestWakeSubscriptionsAreSeparate(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("a").end()
	f.begin("b").end()

	f.burst("b", 5)
	require.Zero(t, f.accepts("a"))
	require.Equal(t, 1, f.accepts("b"))

	f.deliver("a")
	require.Equal(t, 1, f.accepts("a"))

	f.settle("a", "b")
	f.requireNothingLost("a", "b")
}

// A request waiting on one subscription sets only that subscription's flag, and
// a recheck counts only that subscription's messages.
func TestWakeRecheckCountsOnlyItsSubscription(t *testing.T) {
	f := newWakeFixture(t, false)
	r := f.begin("a")
	r.readState()
	f.burst("b", 3)
	r.end()
	require.Zero(t, f.accepts("a"))
	require.Zero(t, f.accepts("b"))
}

// A timed read ends with a sleep as well as the waiting op. If the message wins,
// the wake runs; the timer firing later changes nothing that matters.
func TestWakeTimedReadMessageWins(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("inbox").end()
	f.deliver("inbox")
	f.settle("inbox")

	// The timer of the read that got its message fires: its request waits again.
	require.NoError(t, f.e.startSubscriptionWake(f.ctx, f.md.ID, queue.Item{}))
	f.begin("inbox").end()

	f.deliver("inbox")
	f.settle("inbox")
	f.requireNothingLost("inbox")
}

// A timed read whose timer wins leaves no flag behind: the function moved on and
// is busy, so the next message queues nothing. A flag left from before would
// wake it while a step ran inline.
func TestWakeTimedReadTimerWins(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("inbox").end()

	// The timer fires, and the request it runs is busy.
	require.NoError(t, f.e.startSubscriptionWake(f.ctx, f.md.ID, queue.Item{}))
	r := f.begin("inbox")
	r.readState()
	f.deliver("inbox")
	require.Zero(t, f.accepts("inbox"))

	// It ends waiting and sees the message that arrived.
	r.end()
	f.settle("inbox")
	f.requireNothingLost("inbox")
}

// The timer firing clears a flag that a newer wait set; the request the timer
// runs reports the wait again, so nothing is lost.
func TestWakeTimerClearsANewerWait(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("inbox").end()

	// A message lands as the timer fires, before the timer's request reads.
	f.deliver("inbox")
	f.settle("inbox")
	require.NoError(t, f.e.startSubscriptionWake(f.ctx, f.md.ID, queue.Item{}))
	r := f.begin("inbox")
	f.deliver("inbox")
	r.end()

	f.settle("inbox")
	f.requireNothingLost("inbox")
}

// A run that ended has nothing to wake: its flags go with its state, setting one
// does nothing, and a message for it is dropped.
func TestWakeOfARunThatEnded(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("inbox").end()
	key := redis_state.StateDefaultKey
	_ = key
	require.NotEmpty(t, f.mr.Keys())
	flagKeys := func() (n int) {
		for _, k := range f.mr.Keys() {
			if strings.Contains(k, ":subs:") {
				n++
			}
		}
		return n
	}
	require.Equal(t, 1, flagKeys())

	require.NoError(t, f.rs.Delete(f.ctx, f.md.ID))
	require.Zero(t, flagKeys())

	// A request that ends after the run is gone sets nothing.
	r := f.begin("inbox")
	r.readState()
	rc := &mockRunContext{md: f.md}
	require.NoError(t, f.e.handleGeneratorSubscribeWait(f.ctx, rc, r.waitOp(), queue.PayloadEdge{}))
	require.Zero(t, flagKeys())

	err := f.e.deliverSubscription(f.ctx, event.InternalEvent{ID: ulid.Make(), Event: event.Event{Name: "agent/message"}}, f.pause("inbox"), logger.VoidLogger())
	require.ErrorIs(t, err, state.ErrRunNotFound)
	require.Zero(t, f.accepts("inbox"))
}

// A wake that cannot be queued puts the flag back, so the delivery, retried,
// wakes the run.
func TestWakeEnqueueFailureKeepsTheFlag(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("inbox").end()

	f.e.queue = failingQueue{Queue: f.rq}
	f.events++
	evt := event.InternalEvent{ID: ulid.MustNew(uint64(f.events), rand.Reader), Event: event.Event{Name: "agent/message"}}
	require.Error(t, f.e.deliverSubscription(f.ctx, evt, f.pause("inbox"), logger.VoidLogger()))
	require.Zero(t, f.accepts("inbox"))

	f.e.queue = f.rq
	require.NoError(t, f.e.deliverSubscription(f.ctx, evt, f.pause("inbox"), logger.VoidLogger()))
	f.delivered["inbox"]++
	require.Equal(t, 1, f.accepts("inbox"))

	f.settle("inbox")
	f.requireNothingLost("inbox")
}

type failingQueue struct{ queue.Queue }

func (failingQueue) Enqueue(context.Context, queue.Item, time.Time, queue.EnqueueOpts) error {
	return errors.New("queue down")
}

// A waiting op without a subscription key is not a subscription read.
func TestWaitingOpWithoutASubscriptionDoesNothing(t *testing.T) {
	f := newWakeFixture(t, false)
	rc := &mockRunContext{md: f.md, lifecycleItem: queue.Item{Payload: queue.PayloadEdge{}}}
	op := state.GeneratorOpcode{Op: enums.OpcodeNone, ID: "x"}
	require.NoError(t, f.e.handleGeneratorSubscribeWait(f.ctx, rc, op, queue.PayloadEdge{}))
	require.NoError(t, f.e.handleGeneratorOp(f.ctx, rc, op, OpcodeGroup{}))
}

// Closing at a limit gives a waiting read the chance to see the close.
func TestWakeAfterALimitClose(t *testing.T) {
	f := newWakeFixture(t, false)
	f.begin("inbox").end()
	f.burst("inbox", consts.MaxSubscriptionMessages)
	f.settle("inbox")
	f.requireNothingLost("inbox")

	// The next message breaks the limit: it is dropped, the subscription is
	// closed, and the waiting run is woken to see the close.
	f.events++
	evt := event.InternalEvent{ID: ulid.MustNew(uint64(f.events), rand.Reader), Event: event.Event{Name: "agent/message"}}
	require.NoError(t, f.e.deliverSubscription(f.ctx, evt, f.pause("inbox"), logger.VoidLogger()))
	require.True(t, f.queued("inbox"))
	stack, err := f.rs.LoadStack(f.ctx, f.md.ID)
	require.NoError(t, err)
	require.Contains(t, stack, subscriptionClosedID("inbox"))
	f.settle("inbox")
	require.False(t, f.queued("inbox"))
}

// A request that waits when the subscription closed since it read finds the close
// in the recheck.
func TestWakeRecheckSeesAClose(t *testing.T) {
	f := newWakeFixture(t, false)
	r := f.begin("inbox")
	r.readState()
	require.NoError(t, f.e.endSubscription(f.ctx, f.md.ID, "inbox", []byte("true")))
	r.end()
	require.Equal(t, 1, f.accepts("inbox"))
}

// A waiting thread is not a step: counting it would make a lone background step
// look parallel, and holding it as pending would stop that step's completion
// from ever running a discovery.
func TestWaitingThreadIsNotAStep(t *testing.T) {
	wait := &state.GeneratorOpcode{Op: enums.OpcodeNone, ID: "inbox:1"}
	step := &state.GeneratorOpcode{Op: enums.OpcodeStepPlanned, ID: "bg"}
	empty := &state.GeneratorOpcode{Op: enums.OpcodeNone}

	require.Equal(t, 1, nonLazyOpCount([]*state.GeneratorOpcode{step, wait}))
	require.Equal(t, []string{"bg"}, opGroups([]*state.GeneratorOpcode{step, wait}).NonLazyIDs())
	require.Equal(t, 1, nonLazyOpCount([]*state.GeneratorOpcode{empty}))
}

// Every interleaving of deliveries, requests (the first, a wake's, a timer's),
// their reads and ends, and the queue's removal ends with the function having
// seen every message, nothing queued, and the run waiting.
func TestWakeRandomInterleavings(t *testing.T) {
	keys := []string{"a", "b"}
	for seed := int64(0); seed < 24; seed++ {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			rng := mrand.New(mrand.NewSource(seed))
			f := newWakeFixture(t, seed%2 == 0)

			running := map[string]*request{}
			handled := map[string]bool{}
			for _, key := range keys {
				running[key] = f.begin(key)
			}
			finish := func(key string) {
				r := running[key]
				if !handled[key] {
					r.handle()
				}
				r.finish()
				running[key], handled[key] = nil, false
			}

			for step := 0; step < 150; step++ {
				key := keys[rng.Intn(len(keys))]
				r := running[key]
				switch rng.Intn(6) {
				case 0, 1:
					f.deliver(key)
				case 2:
					// A wake item is leased, or a timer fires and runs a request.
					if r == nil {
						if rng.Intn(4) == 0 {
							require.NoError(t, f.e.startSubscriptionWake(f.ctx, f.md.ID, queue.Item{}))
							running[key] = f.begin(key)
						} else {
							running[key] = f.start(key)
						}
					}
				case 3:
					if r != nil {
						r.readState()
					}
				case 4:
					if r != nil && !handled[key] {
						r.handle()
						handled[key] = true
					}
				case 5:
					if r != nil && handled[key] {
						finish(key)
					}
				}
			}
			for _, key := range keys {
				if running[key] != nil {
					finish(key)
				}
			}
			f.settle(keys...)
			f.requireNothingLost(keys...)

			for _, key := range keys {
				require.LessOrEqual(t, f.accepts(key), f.delivered[key]+1)
			}
		})
	}
}

func (h *hookedRunService) ClearSubscriptionWaitingFor(ctx context.Context, id sv2.ID, hash string) error {
	err := h.RunService.(sv2.SubscriptionWaiter).ClearSubscriptionWaitingFor(ctx, id, hash)
	if err == nil && h.lostClearReply {
		h.lostClearReply = false
		return errors.New("reply lost")
	}
	return err
}

func (h *hookedRunService) RegisterSubscriptionTimer(ctx context.Context, id sv2.ID, timer string) error {
	return h.RunService.(sv2.SubscriptionWaiter).RegisterSubscriptionTimer(ctx, id, timer)
}

func (h *hookedRunService) ClaimSubscriptionTimer(ctx context.Context, id sv2.ID, timer string, who string) (string, error) {
	return h.RunService.(sv2.SubscriptionWaiter).ClaimSubscriptionTimer(ctx, id, timer, who)
}

func (h *hookedRunService) DeliverSubscriptionMessage(ctx context.Context, id sv2.ID, hash string, stepID string, value []byte, limit int) (state.SubscriptionDelivery, error) {
	return h.RunService.(sv2.SubscriptionWaiter).DeliverSubscriptionMessage(ctx, id, hash, stepID, value, limit)
}

func (h *hookedRunService) BufferedSubscriptionMessages(ctx context.Context, id sv2.ID, hash string) ([]state.BufferedMessage, error) {
	return h.RunService.(sv2.SubscriptionWaiter).BufferedSubscriptionMessages(ctx, id, hash)
}

func (h *hookedRunService) RemoveBufferedSubscriptionMessages(ctx context.Context, id sv2.ID, hash string, messages []state.BufferedMessage) error {
	return h.RunService.(sv2.SubscriptionWaiter).RemoveBufferedSubscriptionMessages(ctx, id, hash, messages)
}

func (h *hookedRunService) CloseSubscription(ctx context.Context, id sv2.ID, hash string) error {
	return h.RunService.(sv2.SubscriptionWaiter).CloseSubscription(ctx, id, hash)
}
