package executor

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/inngest/inngest/pkg/enums"
	mrand "math/rand"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// These tests drive step.subscribe wakes through the real queue and run state
// (miniredis, the real Lua), one step at a time, so every interleaving is a
// sequence the test writes down. A wake is processed the way the queue and
// Execute process it: lease the item, record the message count (start), let the
// SDK read the run's state (read), then handle the response (end): a follow-up
// error requeues the same item, anything else dequeues it.

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

type wakeFixture struct {
	t      *testing.T
	ctx    context.Context
	e      *executor
	rq     *recordingQueue
	shard  queue.QueueShard
	rs     sv2.RunService
	md     sv2.Metadata
	events int

	// delivered counts the messages saved per subscription.
	delivered map[string]int
	// saw is how many messages the function's last request read, per
	// subscription: what the SDK saw when it last blocked.
	saw map[string]int
	// starts counts the wake discoveries that ran, per subscription.
	starts map[string]int
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
		t: t, ctx: ctx, rq: rq, shard: shard, rs: rs, md: md,
		e: &executor{
			log: logger.VoidLogger(), smv2: rs, queue: rq, shards: registry,
			pm: &limitPauseManager{}, tracerProvider: tracing.NewNoopTracerProvider(),
		},
		delivered: map[string]int{}, saw: map[string]int{}, starts: map[string]int{},
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

func (f *wakeFixture) burst(key string, n int) {
	for i := 0; i < n; i++ {
		f.deliver(key)
	}
}

func wakeJobID(f *wakeFixture, key string) string {
	return fmt.Sprintf("%s-subscription-wake-%s", f.md.ID.RunID, subscriptionKeyHash(key))
}

// queued reports whether the subscription's wake is in the queue, queued or leased.
func (f *wakeFixture) queued(key string) bool {
	_, err := f.shard.LoadQueueItem(f.ctx, queue.HashID(f.ctx, wakeJobID(f, key)))
	if errors.Is(err, queue.ErrQueueItemNotFound) {
		return false
	}
	require.NoError(f.t, err)
	return true
}

type wakeRun struct {
	f    *wakeFixture
	key  string
	qi   queue.QueueItem
	wake subscriptionWake
	// done is set once the request has read the run's state.
	done bool
}

// start leases the wake of the subscription and records its message count, as
// Execute does before it calls the SDK. It returns nil when nothing is queued.
func (f *wakeFixture) start(key string) *wakeRun {
	f.t.Helper()
	qi, err := f.shard.LoadQueueItem(f.ctx, queue.HashID(f.ctx, wakeJobID(f, key)))
	if errors.Is(err, queue.ErrQueueItemNotFound) {
		return nil
	}
	require.NoError(f.t, err)
	require.Nil(f.t, qi.LeaseID, "a second wake started while one runs")

	lease, err := f.shard.Lease(f.ctx, *qi, 30*time.Second, time.Now())
	require.NoError(f.t, err)
	qi.LeaseID = lease

	md, err := f.rs.LoadMetadata(f.ctx, f.md.ID)
	require.NoError(f.t, err)
	f.starts[key]++

	return &wakeRun{f: f, key: key, qi: *qi, wake: newSubscriptionWake(qi.Data, md.Stack)}
}

// read is the SDK's request reading the run's state: what it holds now is what
// the function sees when it blocks again.
func (r *wakeRun) read() {
	r.f.t.Helper()
	stack, err := r.f.rs.LoadStack(r.f.ctx, r.f.md.ID)
	require.NoError(r.f.t, err)
	r.f.saw[r.key] = len(subscriptionMessageIDs(stack, r.key))
	r.done = true
}

// end handles the response: a follow-up error requeues the same item, as the
// queue does for an always-retry error; any other outcome dequeues it.
func (r *wakeRun) end(resp *state.DriverResponse, driverErr error) (again bool) {
	r.f.t.Helper()
	// A request always reads the state it runs on.
	if !r.done {
		r.read()
	}
	err := r.f.e.subscriptionWakeFollowUp(r.f.ctx, r.f.md.ID, r.wake, resp, driverErr)

	if queue.IsAlwaysRetryable(err) {
		require.ErrorContains(r.f.t, err, errSubscriptionWakeAgain.Error())
		require.NoError(r.f.t, r.f.rq.Requeue(r.f.ctx, r.f.shard.Name(), r.qi, time.Now()))
		return true
	}
	require.NoError(r.f.t, err)

	if driverErr != nil || (resp != nil && resp.Err != nil) {
		// A failed request is retried as the same item, one attempt later.
		r.qi.Data.Attempt++
		require.NoError(r.f.t, r.f.rq.Requeue(r.f.ctx, r.f.shard.Name(), r.qi, time.Now()))
		return false
	}

	require.NoError(r.f.t, r.f.rq.Dequeue(r.f.ctx, r.f.shard.Name(), r.qi))
	return false
}

func okResp() *state.DriverResponse { return &state.DriverResponse{} }

// run is one whole wake with nothing in between.
func (f *wakeFixture) run(key string) {
	f.t.Helper()
	r := f.start(key)
	require.NotNil(f.t, r)
	r.read()
	r.end(okResp(), nil)
}

// drain runs wakes until none is queued. The bound catches a wake that never settles.
func (f *wakeFixture) drain(keys ...string) {
	f.t.Helper()
	for i := 0; i < 1000; i++ {
		ran := false
		for _, key := range keys {
			if f.queued(key) {
				f.run(key)
				ran = true
			}
		}
		if !ran {
			return
		}
	}
	f.t.Fatal("wakes did not settle")
}

// requireNothingLost is the end state of every cell: no wake is queued (no
// hang), and the function's last request saw every message that was saved (no
// lost wake).
func (f *wakeFixture) requireNothingLost(keys ...string) {
	f.t.Helper()
	for _, key := range keys {
		require.False(f.t, f.queued(key), "wake of %q left queued", key)
		require.Equal(f.t, f.delivered[key], f.saw[key], "messages of %q the function never saw", key)
	}
}

// Twenty messages while the wake cannot run (flow control, concurrency): one
// wake in the queue, which sees all twenty.
func TestWakeBurstWhileBlocked(t *testing.T) {
	f := newWakeFixture(t, false)
	f.deliver("inbox")
	f.burst("inbox", 19)

	require.Len(t, f.rq.accepted, 1)
	require.Equal(t, 19, f.rq.rejected)
	require.Equal(t, []string{wakeJobID(f, "inbox")}, f.rq.accepted)

	f.run("inbox")
	f.requireNothingLost("inbox")
	require.Equal(t, 1, f.starts["inbox"])
	require.Equal(t, 20, f.saw["inbox"])
}

// The wake is an ordinary edge item with no idempotency period: the ID is free
// once it is done, and held for as long as it queues or runs.
func TestWakeJobIDHeldWhileQueuedAndFreedWhenDone(t *testing.T) {
	f := newWakeFixture(t, false)
	f.deliver("inbox")

	qi, err := f.shard.LoadQueueItem(f.ctx, queue.HashID(f.ctx, wakeJobID(f, "inbox")))
	require.NoError(t, err)
	require.Equal(t, queue.KindEdge, qi.Data.Kind)
	require.NotNil(t, qi.IdempotencyPeriod)
	require.Zero(t, *qi.IdempotencyPeriod)
	require.Equal(t, "step", qi.Data.Payload.(queue.PayloadEdge).Edge.Incoming)

	// Held while leased, too.
	r := f.start("inbox")
	f.deliver("inbox")
	require.Equal(t, 1, f.rq.rejected)
	r.read()
	require.True(t, r.end(okResp(), nil))
	f.run("inbox")

	// Free once done: the next message makes a new item at once, with no key left.
	require.False(t, f.queued("inbox"))
	before := len(f.rq.accepted)
	f.deliver("inbox")
	require.Len(t, f.rq.accepted, before+1)
	f.run("inbox")
	f.requireNothingLost("inbox")
}

// A message that lands while a wake runs, after its request read the state, is
// the lost wake this design exists to prevent: the same item runs again.
func TestWakeMessageAfterStateRead(t *testing.T) {
	f := newWakeFixture(t, false)
	f.burst("inbox", 3)

	r := f.start("inbox")
	r.read()
	f.deliver("inbox")
	require.True(t, r.end(okResp(), nil), "the wake must run again")
	require.True(t, f.queued("inbox"))

	f.run("inbox")
	f.requireNothingLost("inbox")
	require.Equal(t, 2, f.starts["inbox"])
	require.Len(t, f.rq.accepted, 1, "the second run is the same item, never a second one")
}

// A message between the start and the read is seen by the read; the wake may
// still run once more, harmlessly.
func TestWakeMessageBeforeStateRead(t *testing.T) {
	f := newWakeFixture(t, false)
	f.deliver("inbox")

	r := f.start("inbox")
	f.deliver("inbox")
	r.read()
	r.end(okResp(), nil)

	f.drain("inbox")
	f.requireNothingLost("inbox")
	require.LessOrEqual(t, f.starts["inbox"], 2)
}

// A long step: messages keep arriving while a wake runs. They all join the one
// item, and the wake runs once more for all of them.
func TestWakeMessagesDuringALongDiscovery(t *testing.T) {
	f := newWakeFixture(t, false)
	f.deliver("inbox")

	r := f.start("inbox")
	r.read()
	f.burst("inbox", 30)
	require.Equal(t, 30, f.rq.rejected)
	require.True(t, r.end(okResp(), nil))

	f.drain("inbox")
	f.requireNothingLost("inbox")
	require.Equal(t, 2, f.starts["inbox"])
	require.Len(t, f.rq.accepted, 1)
}

// Messages arrive in a stream while the function runs wake after wake: the
// number of wakes follows the number of bursts, never the number of messages.
func TestWakeCountFollowsBurstsNotMessages(t *testing.T) {
	for name, forceStepPlan := range map[string]bool{"sequential": false, "already parallel": true} {
		t.Run(name, func(t *testing.T) {
			f := newWakeFixture(t, forceStepPlan)
			mode, err := f.rs.LoadMetadata(f.ctx, f.md.ID)
			require.NoError(t, err)

			// Parallel steps are pending the whole time.
			require.NoError(t, f.rs.SavePending(f.ctx, f.md.ID, []string{"bg-1", "bg-2"}))

			const bursts, size = 8, 20
			for b := 0; b < bursts; b++ {
				f.burst("inbox", size)
				r := f.start("inbox")
				r.read()
				// One message lands while the wake runs.
				f.deliver("inbox")
				require.True(t, r.end(okResp(), nil))
				f.run("inbox")
			}

			f.requireNothingLost("inbox")
			require.Equal(t, bursts*(size+1), f.delivered["inbox"])
			require.Equal(t, 2*bursts, f.starts["inbox"], "two wakes per burst, not one per message")
			require.Len(t, f.rq.accepted, bursts, "one item per burst")
			for _, id := range f.rq.accepted {
				require.Equal(t, wakeJobID(f, "inbox"), id)
			}

			// A wake leaves the run's mode alone.
			md, err := f.rs.LoadMetadata(f.ctx, f.md.ID)
			require.NoError(t, err)
			require.Equal(t, mode.Config.ForceStepPlan, md.Config.ForceStepPlan)
		})
	}
}

// The wake carries no parallel mode of its own: it is the run's metadata that
// says whether the SDK plans before it runs.
func TestWakeItemHasNoParallelMode(t *testing.T) {
	for _, forceStepPlan := range []bool{false, true} {
		f := newWakeFixture(t, forceStepPlan)
		f.deliver("inbox")
		qi, err := f.shard.LoadQueueItem(f.ctx, queue.HashID(f.ctx, wakeJobID(f, "inbox")))
		require.NoError(t, err)
		require.Zero(t, qi.Data.ParallelMode)
		require.Empty(t, qi.Data.GetConcurrencyKeys())
	}
}

// Two subscriptions in one run are counted apart: messages of one never make
// the other's wake run again, and each has its own item.
func TestWakeSubscriptionsAreSeparate(t *testing.T) {
	f := newWakeFixture(t, false)
	f.deliver("a")
	f.deliver("b")
	require.Len(t, f.rq.accepted, 2)
	require.NotEqual(t, f.rq.accepted[0], f.rq.accepted[1])

	ra := f.start("a")
	ra.read()
	f.burst("b", 5)
	require.False(t, ra.end(okResp(), nil), "messages of b are not news to the wake of a")

	f.drain("a", "b")
	f.requireNothingLost("a", "b")
	require.Equal(t, 1, f.starts["a"])
	require.Equal(t, 1, f.starts["b"])
}

// A request that failed is retried by the queue as the same item, and only the
// final completion looks for more messages.
func TestWakeRetriedRequestChecksOnItsFinalCompletion(t *testing.T) {
	f := newWakeFixture(t, false)
	f.deliver("inbox")

	r := f.start("inbox")
	f.burst("inbox", 4)
	// The SDK request failed (a response error, then a transport error).
	require.False(t, r.end(&state.DriverResponse{Err: new("boom")}, nil))
	r = f.start("inbox")
	require.Equal(t, 5, r.wake.seen, "the retry records the count at its own start")
	require.False(t, r.end(okResp(), errors.New("connection reset")))

	r = f.start("inbox")
	r.read()
	require.False(t, r.end(okResp(), nil), "the final completion saw every message")
	f.requireNothingLost("inbox")
	require.Len(t, f.rq.accepted, 1)
}

// A run that ended or failed has nothing to see: no follow-up is enqueued, and
// the item is done.
func TestWakeOfARunThatEnded(t *testing.T) {
	f := newWakeFixture(t, false)
	f.deliver("inbox")

	r := f.start("inbox")
	r.read()
	f.deliver("inbox")
	require.NoError(t, f.rs.Delete(f.ctx, f.md.ID))

	require.False(t, r.end(okResp(), nil))
	require.False(t, f.queued("inbox"))

	// And a message for a run that is gone is dropped, not queued.
	err := f.e.deliverSubscription(f.ctx, event.InternalEvent{ID: ulid.Make(), Event: event.Event{Name: "agent/message"}}, f.pause("inbox"), logger.VoidLogger())
	require.ErrorIs(t, err, state.ErrRunNotFound)
	require.False(t, f.queued("inbox"))
}

// An item that is not a wake never looks for messages.
func TestWakeFollowUpIgnoresOtherItems(t *testing.T) {
	f := newWakeFixture(t, false)
	f.deliver("inbox")
	other := newSubscriptionWake(queue.Item{Metadata: map[string]any{}}, nil)
	require.NoError(t, f.e.subscriptionWakeFollowUp(f.ctx, f.md.ID, other, okResp(), nil))
	require.Zero(t, newSubscriptionWake(queue.Item{}, []string{subscriptionMessageID("inbox", "x")}).seen)
}

// Closing at a limit gives a waiting read the chance to see the close.
func TestWakeAfterALimitClose(t *testing.T) {
	f := newWakeFixture(t, false)
	f.burst("inbox", consts.MaxSubscriptionMessages)
	f.run("inbox")
	f.requireNothingLost("inbox")

	// The next message breaks the limit: it is dropped, the subscription is
	// closed, and the run is woken to see the close.
	f.events++
	evt := event.InternalEvent{ID: ulid.MustNew(uint64(f.events), rand.Reader), Event: event.Event{Name: "agent/message"}}
	require.NoError(t, f.e.deliverSubscription(f.ctx, evt, f.pause("inbox"), logger.VoidLogger()))
	require.True(t, f.queued("inbox"))
	stack, err := f.rs.LoadStack(f.ctx, f.md.ID)
	require.NoError(t, err)
	require.Contains(t, stack, subscriptionClosedID("inbox"))
	f.run("inbox")
	require.False(t, f.queued("inbox"))
	f.requireNothingLost("inbox")
}

// Every interleaving of deliveries and wake steps ends with the function having
// seen every message and nothing queued, and no more than one item per
// subscription is ever created at a time.
func TestWakeRandomInterleavings(t *testing.T) {
	keys := []string{"a", "b"}
	for seed := int64(0); seed < 24; seed++ {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			rng := mrand.New(mrand.NewSource(seed))
			f := newWakeFixture(t, seed%2 == 0)

			running := map[string]*wakeRun{}
			readDone := map[string]bool{}
			for step := 0; step < 120; step++ {
				key := keys[rng.Intn(len(keys))]
				switch rng.Intn(4) {
				case 0, 1:
					f.deliver(key)
				case 2:
					if running[key] == nil {
						if r := f.start(key); r != nil {
							running[key] = r
							readDone[key] = false
						}
					} else if !readDone[key] {
						running[key].read()
						readDone[key] = true
					}
				case 3:
					if r := running[key]; r != nil {
						if !readDone[key] && rng.Intn(2) == 0 {
							r.read()
						}
						r.end(okResp(), nil)
						running[key] = nil
					}
				}
			}
			for key, r := range running {
				if r != nil {
					r.read()
					r.end(okResp(), nil)
				}
				_ = key
			}
			f.drain(keys...)
			f.requireNothingLost(keys...)

			for _, id := range f.rq.accepted {
				require.Contains(t, []string{wakeJobID(f, "a"), wakeJobID(f, "b")}, id)
			}
			for _, key := range keys {
				require.LessOrEqual(t, f.starts[key], f.delivered[key]+1)
			}
		})
	}
}

// A read that has nothing to read reports a waiting thread (OpcodeNone with an
// ID). It is not a step: counting it would make a lone background step look
// parallel, and holding it as pending would stop that step's completion from
// ever running a discovery.
func TestWaitingThreadIsNotAStep(t *testing.T) {
	wait := &state.GeneratorOpcode{Op: enums.OpcodeNone, ID: "inbox:1"}
	step := &state.GeneratorOpcode{Op: enums.OpcodeStepPlanned, ID: "bg"}
	empty := &state.GeneratorOpcode{Op: enums.OpcodeNone}

	require.Equal(t, 1, nonLazyOpCount([]*state.GeneratorOpcode{step, wait}))
	require.Equal(t, []string{"bg"}, opGroups([]*state.GeneratorOpcode{step, wait}).NonLazyIDs())
	require.Equal(t, 1, nonLazyOpCount([]*state.GeneratorOpcode{empty}))
}
