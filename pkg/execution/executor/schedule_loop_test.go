package executor

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/domain"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/loops"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/kv/kvredis"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	telemetrytrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/util"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

const loopTestKeyExpr = "event.data.id"

// loopHarness runs Schedule against real state, queue, and loop store Lua.
// finished collects the events Finalize publishes, so a test can read the
// run output from the function finished event.
type loopHarness struct {
	ctx      context.Context
	e        *executor
	store    loops.Store
	q        queue.Queue
	shard    queue.QueueShard
	req      execution.ScheduleRequest
	finished []event.Event
}

func newLoopHarness(t *testing.T) *loopHarness {
	t.Helper()
	ctx := logger.WithStdlib(context.Background(), logger.VoidLogger())
	mr := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{mr.Addr()}, DisableCache: true})
	require.NoError(t, err)
	t.Cleanup(rc.Close)
	uc := redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	sc := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
		UnshardedClient:        uc,
		FunctionRunStateClient: rc,
		BatchClient:            rc,
		StateDefaultKey:        redis_state.StateDefaultKey,
		QueueDefaultKey:        redis_state.QueueDefaultKey,
		FnRunIsSharded:         redis_state.AlwaysShardOnRun,
	})
	mgr, err := redis_state.New(ctx, redis_state.WithShardedClient(sc), redis_state.WithPauseDeleter(redis_state.NewPauseStore(uc)))
	require.NoError(t, err)
	shard := redis_state.NewQueueShard("local", redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey))
	registry, err := queue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	q, err := queue.New(ctx, "loop-schedule", registry)
	require.NoError(t, err)
	store := loops.NewStore(kvredis.New(rc, ""))

	req := capScheduleRequest(nil)
	req.Function.Name = "loop-fn"
	req.Function.Slug = "loop-fn"
	req.Function.Loop = &inngest.Loop{
		Kind: "loop",
		Key:  loopTestKeyExpr,
	}
	req.Function.Steps = []inngest.Step{
		{
			ID:   "step",
			Name: "step",
			URI:  "http://example.com/api/inngest",
		},
	}
	key := loopTestKeyExpr
	req.Function.Concurrency = &inngest.ConcurrencyLimits{
		Fn: []inngest.FnConcurrency{
			{
				Limit: 1,
				Key:   &key,
			},
		},
	}
	evt := req.Events[0].(event.InternalEvent)
	evt.Event.Data = map[string]any{"id": "user-1"}
	req.Events[0] = evt

	e := &executor{
		log:               logger.VoidLogger(),
		clock:             clockwork.NewRealClock(),
		smv2:              redis_state.MustRunServiceV2(mgr),
		queue:             q,
		shards:            registry,
		loops:             store,
		fl:                singletonRetryFunctionLoader{fn: &req.Function},
		tracerProvider:    newRecordingTracerProvider(),
		conditionalTracer: telemetrytrace.NoopConditionalTracer(),
	}
	h := &loopHarness{
		ctx:   ctx,
		e:     e,
		store: store,
		q:     q,
		shard: shard,
		req:   req,
	}
	e.finishHandler = func(_ context.Context, _ sv2.ID, evts []event.Event) error {
		h.finished = append(h.finished, evts...)
		return nil
	}
	return h
}

// request returns a copy of the base request with a fresh event and run id.
// a nil input uses the base trigger event.  a tick or wake event from the
// loops package replaces it.
func (h *loopHarness) request(input *event.Event) (execution.ScheduleRequest, sv2.ID) {
	req := cloneScheduleRequest(h.req)
	evt := req.Events[0].(event.InternalEvent)
	evt.ID = ulid.Make()
	evt.Event.ID = evt.ID.String()
	if input != nil {
		evt.Event = *input
	}
	req.Events[0] = evt
	runID := ulid.Make()
	req.RunID = &runID
	id := sv2.ID{
		RunID:      runID,
		FunctionID: req.Function.ID,
		Tenant: sv2.Tenant{
			AccountID: req.AccountID,
			EnvID:     req.WorkspaceID,
			AppID:     req.AppID,
		},
	}
	return req, id
}

func (h *loopHarness) loopID(key string) loops.ID {
	return loops.NewID(domain.NewTenant(h.req.AccountID, h.req.WorkspaceID), h.req.Function.ID, key)
}

// requireStartSemaphore asserts the run's one start item carries the keyed
// loop semaphore for key.
func (h *loopHarness) requireStartSemaphore(t *testing.T, id sv2.ID, key string) {
	t.Helper()
	scope := queue.Scope{
		AccountID:  id.Tenant.AccountID,
		EnvID:      id.Tenant.EnvID,
		FunctionID: id.FunctionID,
	}
	items, err := h.q.ItemsByRunID(h.ctx, h.shard, scope, id.RunID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, []constraintapi.Semaphore{
		{
			ID:               constraintapi.SemaphoreIDFnKey(id.FunctionID, loopTestKeyExpr),
			EvaluatedKeyHash: util.XXHash(key),
			Weight:           1,
			Release:          constraintapi.SemaphoreReleaseManual,
		},
	}, items[0].Data.Semaphores)
}

func TestScheduleLoopTrigger(t *testing.T) {
	h := newLoopHarness(t)
	req, id := h.request(nil)

	runID, md, err := h.e.Schedule(h.ctx, req)
	require.NoError(t, err)
	require.Equal(t, id.RunID, *runID)
	require.NotNil(t, md)

	snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
	require.NoError(t, err)
	require.Equal(t, int64(0), snap.Counter)

	_, err = h.e.smv2.LoadMetadata(h.ctx, id)
	require.NoError(t, err)
	h.requireStartSemaphore(t, id, "user-1")
}

func TestScheduleLoopInputs(t *testing.T) {
	tests := []struct {
		name    string
		open    bool
		input   *event.Event
		data    map[string]any
		keyExpr string
		wantErr error
		// wantMsg expects an error with no sentinel.
		wantMsg string
	}{
		{
			name:    "tick on a missing instance",
			input:   new(loops.NewTickEvent("user-1")),
			wantErr: ErrLoopEnded,
		},
		{
			name:  "tick on an open instance",
			open:  true,
			input: new(loops.NewTickEvent("user-1")),
		},
		{
			name:  "wake on an open instance",
			open:  true,
			input: new(loops.NewWakeEvent("user-1", "wake-1")),
		},
		{
			name: "tick without a key",
			open: true,
			input: &event.Event{
				Name: loops.EventTick,
				Data: map[string]any{},
			},
			wantMsg: "has no key",
		},
		{
			name: "wake without an id",
			open: true,
			input: &event.Event{
				Name: loops.EventWake,
				Data: map[string]any{"key": "user-1"},
			},
			wantMsg: "has no wake id",
		},
		{
			name:    "event without the key field",
			data:    map[string]any{},
			wantErr: loops.ErrNoKey,
		},
		{
			name:    "key expression that fails to evaluate",
			keyExpr: "int(event.data.id)",
			wantMsg: "evaluating key",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newLoopHarness(t)
			if tt.open {
				first, _ := h.request(nil)
				_, _, err := h.e.Schedule(h.ctx, first)
				require.NoError(t, err)
			}

			req, id := h.request(tt.input)
			if tt.data != nil {
				evt := req.Events[0].(event.InternalEvent)
				evt.Event.Data = tt.data
				req.Events[0] = evt
			}
			if tt.keyExpr != "" {
				req.Function.Loop.Key = tt.keyExpr
			}

			runID, md, err := h.e.Schedule(h.ctx, req)
			if tt.wantErr != nil || tt.wantMsg != "" {
				require.Error(t, err)
				if tt.wantErr != nil {
					require.ErrorIs(t, err, tt.wantErr)
				}
				if tt.wantMsg != "" {
					require.ErrorContains(t, err, tt.wantMsg)
				}
				require.Nil(t, runID)
				require.Nil(t, md)
				_, err = h.e.smv2.LoadMetadata(h.ctx, id)
				require.ErrorIs(t, err, state.ErrRunNotFound)
				return
			}
			require.NoError(t, err)
			require.Equal(t, id.RunID, *runID)

			_, err = h.e.smv2.LoadMetadata(h.ctx, id)
			require.NoError(t, err)
			h.requireStartSemaphore(t, id, "user-1")
		})
	}
}

func TestScheduleLoopWithoutSemaphore(t *testing.T) {
	h := newLoopHarness(t)
	h.req.Function.Concurrency = nil
	req, id := h.request(nil)

	_, _, err := h.e.Schedule(h.ctx, req)
	require.ErrorIs(t, err, ErrNoLoopSemaphore)
	_, err = h.e.smv2.LoadMetadata(h.ctx, id)
	require.ErrorIs(t, err, state.ErrRunNotFound)
	_, err = h.store.Load(h.ctx, h.loopID("user-1"))
	require.ErrorIs(t, err, loops.ErrNotFound)
}

func TestScheduleLoopWithoutStore(t *testing.T) {
	h := newLoopHarness(t)
	h.e.loops = nil
	req, id := h.request(nil)

	_, _, err := h.e.Schedule(h.ctx, req)
	require.ErrorIs(t, err, ErrNoLoopStore)
	_, err = h.e.smv2.LoadMetadata(h.ctx, id)
	require.ErrorIs(t, err, state.ErrRunNotFound)
	_, err = h.store.Load(h.ctx, h.loopID("user-1"))
	require.ErrorIs(t, err, loops.ErrNotFound)
}

func TestLoopRequest(t *testing.T) {
	t.Run("open instance returns the block", func(t *testing.T) {
		h := newLoopHarness(t)
		req, id := h.request(nil)
		_, _, err := h.e.Schedule(h.ctx, req)
		require.NoError(t, err)
		md, err := h.e.smv2.LoadMetadata(h.ctx, id)
		require.NoError(t, err)
		evts, err := h.e.smv2.LoadEvents(h.ctx, id)
		require.NoError(t, err)

		// the state is nil until the first commit and marshals as null.
		loop, replay, err := h.e.loopRequest(h.ctx, &runInstance{
			md:     md,
			f:      req.Function,
			events: evts,
		})
		require.NoError(t, err)
		require.Nil(t, replay)
		require.Equal(t, &driver.SDKLoopRequest{
			State:   nil,
			Version: 0,
			Key:     "user-1",
			Input: driver.SDKLoopInput{
				Kind: loops.InputEvent,
			},
		}, loop)

		// a commit by an earlier run is the state the next request sees.
		_, err = h.store.Commit(h.ctx, loops.Commit{
			ID:      h.loopID("user-1"),
			RunID:   ulid.Make(),
			Counter: 0,
			State:   json.RawMessage(`{"n":1}`),
		})
		require.NoError(t, err)

		loop, replay, err = h.e.loopRequest(h.ctx, &runInstance{
			md:     md,
			f:      req.Function,
			events: evts,
		})
		require.NoError(t, err)
		require.Nil(t, replay)
		require.JSONEq(t, `{"n":1}`, string(loop.State))
		require.Equal(t, int64(1), loop.Version)
	})

	t.Run("a commit by the same run replays the return", func(t *testing.T) {
		h := newLoopHarness(t)
		req, id := h.request(nil)
		_, _, err := h.e.Schedule(h.ctx, req)
		require.NoError(t, err)
		md, err := h.e.smv2.LoadMetadata(h.ctx, id)
		require.NoError(t, err)
		evts, err := h.e.smv2.LoadEvents(h.ctx, id)
		require.NoError(t, err)
		_, err = h.store.Commit(h.ctx, loops.Commit{
			ID:      h.loopID("user-1"),
			RunID:   id.RunID,
			Counter: 0,
			State:   json.RawMessage(`{"n":1}`),
		})
		require.NoError(t, err)

		loop, replay, err := h.e.loopRequest(h.ctx, &runInstance{
			md:     md,
			f:      req.Function,
			events: evts,
		})
		require.NoError(t, err)
		require.Nil(t, loop)
		require.NotNil(t, replay)
		require.Len(t, replay.Generator, 1)
		require.Equal(t, enums.OpcodeLoopReturn, replay.Generator[0].Op)
		ret, err := replay.Generator[0].LoopReturn()
		require.NoError(t, err)
		require.JSONEq(t, `{"n":1}`, string(ret.State))
		require.Equal(t, int64(0), ret.Version)
		require.False(t, ret.End)
		require.Nil(t, ret.Wake)
	})

	t.Run("a saved return replays the end", func(t *testing.T) {
		h := newLoopHarness(t)
		req, id := h.request(nil)
		_, _, err := h.e.Schedule(h.ctx, req)
		require.NoError(t, err)
		evts, err := h.e.smv2.LoadEvents(h.ctx, id)
		require.NoError(t, err)
		saved, err := json.Marshal(savedLoopReturn{
			Return: state.LoopReturn{
				State:   json.RawMessage(`{"done":true}`),
				Version: 0,
				End:     true,
			},
		})
		require.NoError(t, err)
		_, err = h.e.smv2.SaveStep(h.ctx, id, loops.ReturnStepID, saved)
		require.NoError(t, err)
		// the saved step is on the stack of the metadata the next attempt loads.
		md, err := h.e.smv2.LoadMetadata(h.ctx, id)
		require.NoError(t, err)
		require.Contains(t, md.Stack, loops.ReturnStepID)
		snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
		require.NoError(t, err)
		require.NoError(t, h.store.End(h.ctx, snap))

		loop, replay, err := h.e.loopRequest(h.ctx, &runInstance{
			md:     md,
			f:      req.Function,
			events: evts,
		})
		require.NoError(t, err)
		require.Nil(t, loop)
		require.NotNil(t, replay)
		ret, err := replay.Generator[0].LoopReturn()
		require.NoError(t, err)
		require.True(t, ret.End)
		require.Equal(t, int64(0), ret.Version)
		require.JSONEq(t, `{"done":true}`, string(ret.State))

		// a fresh instance opened by a later trigger does not change that.
		_, err = h.store.Open(h.ctx, h.loopID("user-1"), ulid.Make())
		require.NoError(t, err)
		loop, replay, err = h.e.loopRequest(h.ctx, &runInstance{
			md:     md,
			f:      req.Function,
			events: evts,
		})
		require.NoError(t, err)
		require.Nil(t, loop)
		require.NotNil(t, replay)
		ret, err = replay.Generator[0].LoopReturn()
		require.NoError(t, err)
		require.True(t, ret.End)
	})

	t.Run("a trigger on a missing instance opens a fresh one", func(t *testing.T) {
		h := newLoopHarness(t)
		req, id := h.request(nil)
		_, _, err := h.e.Schedule(h.ctx, req)
		require.NoError(t, err)
		md, err := h.e.smv2.LoadMetadata(h.ctx, id)
		require.NoError(t, err)
		evts, err := h.e.smv2.LoadEvents(h.ctx, id)
		require.NoError(t, err)
		snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
		require.NoError(t, err)
		require.NoError(t, h.store.End(h.ctx, snap))

		loop, replay, err := h.e.loopRequest(h.ctx, &runInstance{
			md:     md,
			f:      req.Function,
			events: evts,
		})
		require.NoError(t, err)
		require.Nil(t, replay)
		require.Equal(t, &driver.SDKLoopRequest{
			State:   nil,
			Version: 0,
			Key:     "user-1",
			Input: driver.SDKLoopInput{
				Kind: loops.InputEvent,
			},
		}, loop)
		snap, err = h.store.Load(h.ctx, h.loopID("user-1"))
		require.NoError(t, err)
		require.Equal(t, int64(0), snap.Counter)
	})

	t.Run("a tick on a missing instance returns ErrLoopEnded", func(t *testing.T) {
		h := newLoopHarness(t)
		first, _ := h.request(nil)
		_, _, err := h.e.Schedule(h.ctx, first)
		require.NoError(t, err)
		req, id := h.request(new(loops.NewTickEvent("user-1")))
		_, _, err = h.e.Schedule(h.ctx, req)
		require.NoError(t, err)
		md, err := h.e.smv2.LoadMetadata(h.ctx, id)
		require.NoError(t, err)
		evts, err := h.e.smv2.LoadEvents(h.ctx, id)
		require.NoError(t, err)
		snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
		require.NoError(t, err)
		require.NoError(t, h.store.End(h.ctx, snap))

		loop, replay, err := h.e.loopRequest(h.ctx, &runInstance{
			md:     md,
			f:      req.Function,
			events: evts,
		})
		require.ErrorIs(t, err, ErrLoopEnded)
		require.Nil(t, loop)
		require.Nil(t, replay)
		_, err = h.store.Load(h.ctx, h.loopID("user-1"))
		require.ErrorIs(t, err, loops.ErrNotFound)
	})

	t.Run("function without Loop gets nil", func(t *testing.T) {
		h := newLoopHarness(t)
		req, id := h.request(nil)
		_, _, err := h.e.Schedule(h.ctx, req)
		require.NoError(t, err)
		md, err := h.e.smv2.LoadMetadata(h.ctx, id)
		require.NoError(t, err)
		evts, err := h.e.smv2.LoadEvents(h.ctx, id)
		require.NoError(t, err)

		fn := req.Function
		fn.Loop = nil
		loop, replay, err := h.e.loopRequest(h.ctx, &runInstance{
			md:     md,
			f:      fn,
			events: evts,
		})
		require.NoError(t, err)
		require.Nil(t, loop)
		require.Nil(t, replay)
	})
}
