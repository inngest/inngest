package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/domain"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/loops"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// loopReturnDriver answers every request the way an SDK answers a loop run.
// ops are sent as a 206.  without ops the response is a plain 200 with
// output.  with echo set, every LoopReturn carries the version from the
// request's loop block, the way an SDK copies it.
type loopReturnDriver struct {
	calls  int
	ops    []*state.GeneratorOpcode
	output any
	echo   bool
	seen   *driver.SDKLoopRequest
}

func (d *loopReturnDriver) Name() string {
	return "http"
}

func (d *loopReturnDriver) Execute(ctx context.Context, _ sv2.StateLoader, _ sv2.Metadata, _ queue.Item, _ inngest.Edge, step inngest.Step, _ int, _ int) (*state.DriverResponse, error) {
	d.calls++
	d.seen = driver.LoopFromContext(ctx)
	resp := &state.DriverResponse{
		Step:       step,
		StatusCode: 200,
		Output:     d.output,
	}
	if len(d.ops) > 0 {
		resp.StatusCode = 206
		resp.Generator = d.ops
	}
	if d.echo && d.seen != nil {
		for i, op := range resp.Generator {
			if op == nil || op.Op != enums.OpcodeLoopReturn {
				continue
			}
			ret, err := op.LoopReturn()
			if err != nil {
				return nil, err
			}
			ret.Version = d.seen.Version
			data, err := json.Marshal(ret)
			if err != nil {
				return nil, err
			}
			echoed := *op
			echoed.Data = data
			resp.Generator[i] = &echoed
		}
	}
	return resp, nil
}

// crashingStore wraps the loop store and fails the first n commits or ends
// after they land, the way a crash between the store write and finalize
// looks to the executor.
type crashingStore struct {
	loops.Store
	commitFails int
	endFails    int
	// endSkips fails the first n ends before the delete, the way a crash
	// between the saved return and the delete looks.
	endSkips int
}

var errStoreCrashed = errors.New("store crashed after the write")

func (s *crashingStore) Commit(ctx context.Context, c loops.Commit) (loops.Snapshot, error) {
	snap, err := s.Store.Commit(ctx, c)
	if err == nil && s.commitFails > 0 {
		s.commitFails--
		return loops.Snapshot{}, errStoreCrashed
	}
	return snap, err
}

func (s *crashingStore) End(ctx context.Context, snap loops.Snapshot) error {
	if s.endSkips > 0 {
		s.endSkips--
		return errStoreCrashed
	}
	err := s.Store.End(ctx, snap)
	if err == nil && s.endFails > 0 {
		s.endFails--
		return errStoreCrashed
	}
	return err
}

func loopReturnOp(t *testing.T, ret state.LoopReturn) *state.GeneratorOpcode {
	t.Helper()
	data, err := json.Marshal(ret)
	require.NoError(t, err)
	return &state.GeneratorOpcode{
		ID:   "loop-return",
		Op:   enums.OpcodeLoopReturn,
		Data: data,
	}
}

// runLoop schedules one input and drives its start item through Execute.
// a nil input schedules the trigger, which opens the instance.
func (h *loopHarness) runLoop(t *testing.T, d *loopReturnDriver, input *event.Event) (sv2.ID, *state.DriverResponse, error) {
	t.Helper()
	h.e.driverv1 = map[string]driver.DriverV1{
		d.Name(): d,
	}
	req, id := h.request(input)
	_, _, err := h.e.Schedule(h.ctx, req)
	require.NoError(t, err)
	resp, err := h.execute(t, id)
	return id, resp, err
}

// retry drives the run's start item through Execute again as attempt n.
func (h *loopHarness) retry(t *testing.T, id sv2.ID, attempt int) (*state.DriverResponse, error) {
	t.Helper()
	scope := queue.Scope{
		AccountID:  id.Tenant.AccountID,
		EnvID:      id.Tenant.EnvID,
		FunctionID: id.FunctionID,
	}
	items, err := h.q.ItemsByRunID(h.ctx, h.shard, scope, id.RunID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	item := items[0].Data
	item.Attempt = attempt
	edge, err := queue.GetEdge(item)
	require.NoError(t, err)
	return h.e.Execute(h.ctx, item.Identifier, item, edge.Edge)
}

func (h *loopHarness) requireRunGone(t *testing.T, id sv2.ID) {
	t.Helper()
	_, err := h.e.smv2.LoadMetadata(h.ctx, id)
	require.ErrorIs(t, err, state.ErrRunNotFound)
}

// requireLoopFailed asserts the run finished as failed without retry and
// that the run output names msg.
func (h *loopHarness) requireLoopFailed(t *testing.T, id sv2.ID, resp *state.DriverResponse, err error, msg string) {
	t.Helper()
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Err)
	require.Equal(t, state.InngestErrLoopInvalid, *resp.Err)
	require.True(t, resp.NoRetry)
	out, ok := resp.Output.(string)
	require.True(t, ok, "output is %T", resp.Output)
	require.Contains(t, out, msg)
	h.requireRunGone(t, id)
	require.Equal(t, event.FnFailedName, h.lastFinished(t, event.FnFailedName).Name)
}

// lastFinished returns the last event Finalize published with the name.
func (h *loopHarness) lastFinished(t *testing.T, name string) event.Event {
	t.Helper()
	for i := len(h.finished) - 1; i >= 0; i-- {
		if h.finished[i].Name == name {
			return h.finished[i]
		}
	}
	require.Failf(t, "no event published", "want %s in %d events", name, len(h.finished))
	return event.Event{}
}

// finishedResult returns the result of the last function finished event as
// JSON.
func (h *loopHarness) finishedResult(t *testing.T) string {
	t.Helper()
	evt := h.lastFinished(t, event.FnFinishedName)
	switch v := evt.Data["result"].(type) {
	case json.RawMessage:
		return string(v)
	case []byte:
		return string(v)
	default:
		b, err := json.Marshal(v)
		require.NoError(t, err)
		return string(b)
	}
}

func (h *loopHarness) wakeJobID(wakeID string) string {
	return fmt.Sprintf("wake:%s:%s", h.loopID("user-1").InstanceID(), wakeID)
}

func (h *loopHarness) loadWake(t *testing.T, wakeID string) (*queue.QueueItem, error) {
	t.Helper()
	return h.shard.LoadQueueItem(h.ctx, queue.HashID(h.ctx, h.wakeJobID(wakeID)))
}

func (h *loopHarness) loadTick(t *testing.T) (*queue.QueueItem, error) {
	t.Helper()
	jobID := "tick:" + h.loopID("user-1").InstanceID().String()
	return h.shard.LoadQueueItem(h.ctx, queue.HashID(h.ctx, jobID))
}

// setTick gives the harness function a tick and reloads it.
func (h *loopHarness) setTick(t *testing.T, tick time.Duration) {
	t.Helper()
	h.req.Function.Loop.Tick = tick
	fn := h.req.Function
	h.e.fl = singletonRetryFunctionLoader{fn: &fn}
}

func requireWithin(t *testing.T, want time.Time, gotMS int64, tolerance time.Duration) {
	t.Helper()
	got := time.UnixMilli(gotMS)
	require.WithinDuration(t, want, got, tolerance)
}

func TestLoopReturnCommit(t *testing.T) {
	h := newLoopHarness(t)

	// the trigger opens the instance at counter 0 and commits the first
	// state.  the driver copies the version from the request.
	d := &loopReturnDriver{
		echo: true,
		ops: []*state.GeneratorOpcode{
			loopReturnOp(t, state.LoopReturn{
				State:   json.RawMessage(`{"n":1}`),
				Version: 99,
			}),
		},
	}
	id, resp, err := h.runLoop(t, d, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Nil(t, resp.Err)
	require.Equal(t, 1, d.calls)
	require.NotNil(t, d.seen)
	require.Equal(t, int64(0), d.seen.Version)

	snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
	require.NoError(t, err)
	require.Equal(t, int64(1), snap.Counter)
	require.JSONEq(t, `{"n":1}`, string(snap.State))
	require.NotNil(t, snap.LastRunID)
	require.Equal(t, id.RunID, *snap.LastRunID)
	// the harness function has no idle_ttl, so the commit uses the default.
	require.WithinDuration(t, time.Now().Add(consts.DefaultLoopIdleTTL), snap.ExpiresAt, 10*time.Second)

	h.requireRunGone(t, id)
	require.JSONEq(t, `{"n":1}`, h.finishedResult(t))

	// the next input sees version 1 and the state carries over.
	d = &loopReturnDriver{
		echo: true,
		ops: []*state.GeneratorOpcode{
			loopReturnOp(t, state.LoopReturn{
				State:   json.RawMessage(`{"n":2}`),
				Version: 99,
			}),
		},
	}
	id, resp, err = h.runLoop(t, d, new(loops.NewTickEvent("user-1")))
	require.NoError(t, err)
	require.Nil(t, resp.Err)
	require.Equal(t, int64(1), d.seen.Version)
	require.JSONEq(t, `{"n":1}`, string(d.seen.State))
	require.Equal(t, driver.SDKLoopInput{
		Kind: loops.InputTick,
	}, d.seen.Input)

	snap, err = h.store.Load(h.ctx, h.loopID("user-1"))
	require.NoError(t, err)
	require.Equal(t, int64(2), snap.Counter)
	require.JSONEq(t, `{"n":2}`, string(snap.State))
	require.Equal(t, id.RunID, *snap.LastRunID)
	h.requireRunGone(t, id)
	require.JSONEq(t, `{"n":2}`, h.finishedResult(t))
}

func TestLoopReturnIdleTTL(t *testing.T) {
	h := newLoopHarness(t)
	h.req.Function.Loop.IdleTTL = time.Hour
	fn := h.req.Function
	h.e.fl = singletonRetryFunctionLoader{fn: &fn}

	d := &loopReturnDriver{
		ops: []*state.GeneratorOpcode{
			loopReturnOp(t, state.LoopReturn{
				State:   json.RawMessage(`{}`),
				Version: 0,
			}),
		},
	}
	_, resp, err := h.runLoop(t, d, nil)
	require.NoError(t, err)
	require.Nil(t, resp.Err)

	snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
	require.NoError(t, err)
	require.WithinDuration(t, time.Now().Add(time.Hour), snap.ExpiresAt, 10*time.Second)
}

func TestLoopReturnWakeIdleTTL(t *testing.T) {
	// commit returns a driver whose LoopReturn commits version 0 and wakes
	// after the duration.
	commit := func(after string) *loopReturnDriver {
		return &loopReturnDriver{
			ops: []*state.GeneratorOpcode{
				loopReturnOp(t, state.LoopReturn{
					State:   json.RawMessage(`{}`),
					Version: 0,
					Wake: &state.LoopWake{
						ID:    "w1",
						After: after,
					},
				}),
			},
		}
	}
	harness := func(t *testing.T, ttl time.Duration) *loopHarness {
		h := newLoopHarness(t)
		h.req.Function.Loop.IdleTTL = ttl
		fn := h.req.Function
		h.e.fl = singletonRetryFunctionLoader{fn: &fn}
		return h
	}

	t.Run("wake past the idle ttl rejected", func(t *testing.T) {
		h := harness(t, time.Hour)
		id, resp, err := h.runLoop(t, commit("61m"), nil)
		h.requireLoopFailed(t, id, resp, err, "the limit is the idle_ttl of 1h0m0s")

		_, err = h.loadWake(t, "w1")
		require.ErrorIs(t, err, queue.ErrQueueItemNotFound)
	})

	t.Run("wake at the idle ttl expires one hour after the wake", func(t *testing.T) {
		h := harness(t, time.Hour)
		_, resp, err := h.runLoop(t, commit("1h"), nil)
		require.NoError(t, err)
		require.Nil(t, resp.Err)

		item, err := h.loadWake(t, "w1")
		require.NoError(t, err)
		requireWithin(t, time.Now().Add(time.Hour), item.AtMS, 5*time.Second)

		snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
		require.NoError(t, err)
		require.WithinDuration(t, time.Now().Add(2*time.Hour), snap.ExpiresAt, 10*time.Second)
	})

	t.Run("short wake expires one idle ttl after the commit", func(t *testing.T) {
		h := harness(t, 3*time.Hour)
		_, resp, err := h.runLoop(t, commit("30m"), nil)
		require.NoError(t, err)
		require.Nil(t, resp.Err)

		snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
		require.NoError(t, err)
		require.WithinDuration(t, time.Now().Add(3*time.Hour), snap.ExpiresAt, 10*time.Second)
	})
}

func TestLoopReturnVersionMismatch(t *testing.T) {
	h := newLoopHarness(t)
	d := &loopReturnDriver{
		ops: []*state.GeneratorOpcode{
			loopReturnOp(t, state.LoopReturn{
				State:   json.RawMessage(`{"n":1}`),
				Version: 7,
			}),
		},
	}
	id, resp, err := h.runLoop(t, d, nil)
	h.requireLoopFailed(t, id, resp, err, "loop state moved under the run")

	snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
	require.NoError(t, err)
	require.Equal(t, int64(0), snap.Counter)
	require.Nil(t, snap.State)
	require.Nil(t, snap.LastRunID)
}

func TestLoopReturnEnd(t *testing.T) {
	h := newLoopHarness(t)
	h.openAndCommit(t)

	d := &loopReturnDriver{
		ops: []*state.GeneratorOpcode{
			loopReturnOp(t, state.LoopReturn{
				State:   json.RawMessage(`{"done":true}`),
				Version: 1,
				End:     true,
				Wake: &state.LoopWake{
					ID:    "w1",
					After: "30s",
				},
			}),
		},
	}
	id, resp, err := h.runLoop(t, d, new(loops.NewWakeEvent("user-1", "w0")))
	require.NoError(t, err)
	require.Nil(t, resp.Err)
	require.Equal(t, "user-1", d.seen.Key)
	require.Equal(t, driver.SDKLoopInput{
		Kind: loops.InputWake,
		ID:   "w0",
	}, d.seen.Input)

	_, err = h.store.Load(h.ctx, h.loopID("user-1"))
	require.ErrorIs(t, err, loops.ErrNotFound)
	h.requireRunGone(t, id)
	require.JSONEq(t, `{"done":true}`, h.finishedResult(t))

	_, err = h.loadWake(t, "w1")
	require.ErrorIs(t, err, queue.ErrQueueItemNotFound)
}

func TestLoopReturnWake(t *testing.T) {
	h := newLoopHarness(t)

	d := &loopReturnDriver{
		ops: []*state.GeneratorOpcode{
			loopReturnOp(t, state.LoopReturn{
				State:   json.RawMessage(`{"n":1}`),
				Version: 0,
				Wake: &state.LoopWake{
					ID:    "w1",
					After: "30s",
				},
			}),
		},
	}
	id, resp, err := h.runLoop(t, d, nil)
	require.NoError(t, err)
	require.Nil(t, resp.Err)
	h.requireRunGone(t, id)

	item, err := h.loadWake(t, "w1")
	require.NoError(t, err)
	requireWithin(t, time.Now().Add(30*time.Second), item.AtMS, 5*time.Second)
	require.Equal(t, queue.KindLoopWake, item.Data.Kind)
	require.Equal(t, queue.PayloadLoop{
		Key:   "user-1",
		Kind:  loops.InputWake,
		ID:    "w1",
		Nonce: id.RunID.String(),
	}, item.Data.Payload)
	require.Equal(t, id.FunctionID, item.Data.Identifier.WorkflowID)
	require.Equal(t, id.Tenant.AccountID, item.Data.Identifier.AccountID)
	require.Equal(t, id.Tenant.EnvID, item.Data.Identifier.WorkspaceID)
	require.Equal(t, id.Tenant.AppID, item.Data.Identifier.AppID)
	require.Equal(t, ulid.ULID{}, item.Data.Identifier.RunID)
	require.Empty(t, item.Data.Semaphores)
	require.Empty(t, item.Data.CustomConcurrencyKeys)

	// the same wake id from the next run moves the item to the new time.
	d = &loopReturnDriver{
		ops: []*state.GeneratorOpcode{
			loopReturnOp(t, state.LoopReturn{
				State:   json.RawMessage(`{"n":2}`),
				Version: 1,
				Wake: &state.LoopWake{
					ID:    "w1",
					After: "5m",
				},
			}),
		},
	}
	_, resp, err = h.runLoop(t, d, new(loops.NewTickEvent("user-1")))
	require.NoError(t, err)
	require.Nil(t, resp.Err)

	moved, err := h.loadWake(t, "w1")
	require.NoError(t, err)
	require.Equal(t, item.ID, moved.ID)
	requireWithin(t, time.Now().Add(5*time.Minute), moved.AtMS, 5*time.Second)

	// a different wake id is a second item.
	d = &loopReturnDriver{
		ops: []*state.GeneratorOpcode{
			loopReturnOp(t, state.LoopReturn{
				State:   json.RawMessage(`{"n":3}`),
				Version: 2,
				Wake: &state.LoopWake{
					ID:    "w2",
					After: "1m",
				},
			}),
		},
	}
	_, resp, err = h.runLoop(t, d, new(loops.NewTickEvent("user-1")))
	require.NoError(t, err)
	require.Nil(t, resp.Err)
	other, err := h.loadWake(t, "w2")
	require.NoError(t, err)
	require.NotEqual(t, item.ID, other.ID)
	requireWithin(t, time.Now().Add(time.Minute), other.AtMS, 5*time.Second)
}

func TestLoopReturnRetryAfterCommit(t *testing.T) {
	h := newLoopHarness(t)
	h.setTick(t, time.Minute)
	cs := &crashingStore{
		Store:       h.store,
		commitFails: 1,
	}
	h.e.loops = cs

	// the commit lands and the attempt fails after it.
	d := &loopReturnDriver{
		echo: true,
		ops: []*state.GeneratorOpcode{
			loopReturnOp(t, state.LoopReturn{
				State: json.RawMessage(`{"n":1}`),
				Wake: &state.LoopWake{
					ID:    "w1",
					After: "30s",
				},
			}),
		},
	}
	id, _, err := h.runLoop(t, d, nil)
	require.ErrorIs(t, err, errStoreCrashed)
	require.Equal(t, 1, d.calls)
	snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
	require.NoError(t, err)
	require.Equal(t, int64(1), snap.Counter)
	require.Equal(t, id.RunID, *snap.LastRunID)
	_, err = h.e.smv2.LoadMetadata(h.ctx, id)
	require.NoError(t, err)
	wake, err := h.loadWake(t, "w1")
	require.NoError(t, err)

	// the tick ran before the retry, so the queue dropped it.
	tick, err := h.loadTick(t)
	require.NoError(t, err)
	require.NoError(t, h.shard.Dequeue(h.ctx, *tick))

	// the retry never reaches the driver.  the counter stays, the wake
	// item stays, no tick is enqueued again, and the run finishes with
	// the committed state.
	resp, err := h.retry(t, id, 1)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Nil(t, resp.Err)
	require.Equal(t, 1, d.calls)
	snap, err = h.store.Load(h.ctx, h.loopID("user-1"))
	require.NoError(t, err)
	require.Equal(t, int64(1), snap.Counter)
	require.JSONEq(t, `{"n":1}`, string(snap.State))
	same, err := h.loadWake(t, "w1")
	require.NoError(t, err)
	require.Equal(t, wake.AtMS, same.AtMS)
	_, err = h.loadTick(t)
	require.ErrorIs(t, err, queue.ErrQueueItemNotFound)
	h.requireRunGone(t, id)
	require.JSONEq(t, `{"n":1}`, h.finishedResult(t))
	require.Len(t, resp.Generator, 1)
	require.Equal(t, enums.OpcodeRunComplete, resp.Generator[0].Op)
	require.JSONEq(t, `{"n":1}`, string(resp.Generator[0].Data))
}

func TestLoopReturnRetryAfterEnd(t *testing.T) {
	h := newLoopHarness(t)
	h.setTick(t, time.Minute)
	h.openAndCommit(t)
	cs := &crashingStore{
		Store:    h.store,
		endFails: 1,
	}
	h.e.loops = cs

	d := &loopReturnDriver{
		echo: true,
		ops: []*state.GeneratorOpcode{
			loopReturnOp(t, state.LoopReturn{
				State: json.RawMessage(`{"done":true}`),
				End:   true,
			}),
		},
	}
	id, _, err := h.runLoop(t, d, new(loops.NewTickEvent("user-1")))
	require.ErrorIs(t, err, errStoreCrashed)
	require.Equal(t, 1, d.calls)
	_, err = h.store.Load(h.ctx, h.loopID("user-1"))
	require.ErrorIs(t, err, loops.ErrNotFound)

	// the return is saved in run state before the end.
	steps, err := h.e.smv2.LoadStepsWithIDs(h.ctx, id, []string{loops.ReturnStepID})
	require.NoError(t, err)
	require.Contains(t, steps, loops.ReturnStepID)

	// a new trigger opens a fresh instance before the retry.
	_, err = h.store.Open(h.ctx, h.loopID("user-1"), ulid.Make())
	require.NoError(t, err)

	// the retry replays the saved return and never reaches the driver.
	// the attempt counter is not read.  the fresh instance is left alone
	// and no tick is enqueued.
	resp, err := h.retry(t, id, 0)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Nil(t, resp.Err)
	require.Equal(t, 1, d.calls)
	h.requireRunGone(t, id)
	require.JSONEq(t, `{"done":true}`, h.finishedResult(t))
	snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
	require.NoError(t, err)
	require.Equal(t, int64(0), snap.Counter)
	_, err = h.loadTick(t)
	require.ErrorIs(t, err, queue.ErrQueueItemNotFound)
}

// TestLoopReturnRetryBeforeEnd crashes between the saved return and the
// delete.  the retry deletes the record the run saved, and leaves a record
// a later trigger created.
func TestLoopReturnRetryBeforeEnd(t *testing.T) {
	end := func(t *testing.T, h *loopHarness) sv2.ID {
		t.Helper()
		h.openAndCommit(t)
		h.e.loops = &crashingStore{
			Store:    h.store,
			endSkips: 1,
		}
		d := &loopReturnDriver{
			echo: true,
			ops: []*state.GeneratorOpcode{
				loopReturnOp(t, state.LoopReturn{
					State: json.RawMessage(`{"done":true}`),
					End:   true,
				}),
			},
		}
		id, _, err := h.runLoop(t, d, new(loops.NewTickEvent("user-1")))
		require.ErrorIs(t, err, errStoreCrashed)
		require.Equal(t, 1, d.calls)
		snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
		require.NoError(t, err)
		require.Equal(t, int64(1), snap.Counter)
		return id
	}

	t.Run("the retry deletes the record", func(t *testing.T) {
		h := newLoopHarness(t)
		id := end(t, h)

		resp, err := h.retry(t, id, 0)
		require.NoError(t, err)
		require.Nil(t, resp.Err)
		h.requireRunGone(t, id)
		require.JSONEq(t, `{"done":true}`, h.finishedResult(t))
		_, err = h.store.Load(h.ctx, h.loopID("user-1"))
		require.ErrorIs(t, err, loops.ErrNotFound)
	})

	t.Run("a reopened record is kept", func(t *testing.T) {
		h := newLoopHarness(t)
		id := end(t, h)

		// a later trigger ends and reopens the instance before the retry.
		snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
		require.NoError(t, err)
		require.NoError(t, h.store.End(h.ctx, snap))
		reopener := ulid.Make()
		_, err = h.store.Open(h.ctx, h.loopID("user-1"), reopener)
		require.NoError(t, err)

		resp, err := h.retry(t, id, 0)
		require.NoError(t, err)
		require.Nil(t, resp.Err)
		h.requireRunGone(t, id)
		require.JSONEq(t, `{"done":true}`, h.finishedResult(t))
		fresh, err := h.store.Load(h.ctx, h.loopID("user-1"))
		require.NoError(t, err)
		require.Equal(t, reopener, fresh.OpenedBy)
		require.Equal(t, int64(0), fresh.Counter)
	})
}

// TestLoopOpenedBy covers the opener an end saves when the handler has
// none from its run instance.
func TestLoopOpenedBy(t *testing.T) {
	h := newLoopHarness(t)
	id := h.loopID("user-1")

	// a missing record gives zero.
	got, err := h.e.loopOpenedBy(h.ctx, id, domain.RunID{})
	require.NoError(t, err)
	require.Equal(t, domain.RunID{}, got)

	// a zero opener is read from the record.
	opener := ulid.Make()
	_, err = h.store.Open(h.ctx, id, opener)
	require.NoError(t, err)
	got, err = h.e.loopOpenedBy(h.ctx, id, domain.RunID{})
	require.NoError(t, err)
	require.Equal(t, opener, got)

	// a known opener is kept.
	known := ulid.Make()
	got, err = h.e.loopOpenedBy(h.ctx, id, known)
	require.NoError(t, err)
	require.Equal(t, known, got)
}

// TestLoopReturnTriggerBehindEnd runs a trigger whose instance ended after
// the trigger was scheduled.  the run opens a fresh instance at counter 0.
func TestLoopReturnTriggerBehindEnd(t *testing.T) {
	h := newLoopHarness(t)
	h.openAndCommit(t)
	d := &loopReturnDriver{
		echo: true,
		ops: []*state.GeneratorOpcode{
			loopReturnOp(t, state.LoopReturn{
				State: json.RawMessage(`{"n":1}`),
			}),
		},
	}
	h.e.driverv1 = map[string]driver.DriverV1{
		d.Name(): d,
	}
	req, id := h.request(nil)
	_, _, err := h.e.Schedule(h.ctx, req)
	require.NoError(t, err)
	snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
	require.NoError(t, err)
	require.Equal(t, int64(1), snap.Counter)
	require.NoError(t, h.store.End(h.ctx, snap))

	resp, err := h.execute(t, id)
	require.NoError(t, err)
	require.Nil(t, resp.Err)
	require.Equal(t, 1, d.calls)
	require.Equal(t, int64(0), d.seen.Version)
	require.Nil(t, d.seen.State)
	require.Equal(t, loops.InputEvent, d.seen.Input.Kind)

	snap, err = h.store.Load(h.ctx, h.loopID("user-1"))
	require.NoError(t, err)
	require.Equal(t, int64(1), snap.Counter)
	require.JSONEq(t, `{"n":1}`, string(snap.State))
	require.Equal(t, id.RunID, *snap.LastRunID)
	h.requireRunGone(t, id)
	require.JSONEq(t, `{"n":1}`, h.finishedResult(t))
}

// TestLoopReturnTickAfterEnd runs a tick whose instance ended after the
// tick was scheduled.  the run is cancelled without a request.
func TestLoopReturnTickAfterEnd(t *testing.T) {
	h := newLoopHarness(t)
	h.openAndCommit(t)
	d := &loopReturnDriver{
		echo: true,
		ops: []*state.GeneratorOpcode{
			loopReturnOp(t, state.LoopReturn{
				State: json.RawMessage(`{"n":1}`),
			}),
		},
	}
	h.e.driverv1 = map[string]driver.DriverV1{
		d.Name(): d,
	}
	req, id := h.request(new(loops.NewTickEvent("user-1")))
	_, _, err := h.e.Schedule(h.ctx, req)
	require.NoError(t, err)
	snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
	require.NoError(t, err)
	require.NoError(t, h.store.End(h.ctx, snap))

	resp, err := h.execute(t, id)
	require.ErrorIs(t, err, state.ErrFunctionCancelled)
	require.Nil(t, resp)
	require.Equal(t, 0, d.calls)
	h.requireRunGone(t, id)
	require.Equal(t, event.FnCancelledName, h.lastFinished(t, event.FnCancelledName).Name)
	_, err = h.store.Load(h.ctx, h.loopID("user-1"))
	require.ErrorIs(t, err, loops.ErrNotFound)
}

func TestLoopReturnWakeIDReused(t *testing.T) {
	h := newLoopHarness(t)

	d := &loopReturnDriver{
		echo: true,
		ops: []*state.GeneratorOpcode{
			loopReturnOp(t, state.LoopReturn{
				State: json.RawMessage(`{"n":1}`),
				Wake: &state.LoopWake{
					ID:    "w1",
					After: "30s",
				},
			}),
		},
	}
	_, resp, err := h.runLoop(t, d, nil)
	require.NoError(t, err)
	require.Nil(t, resp.Err)
	item, err := h.loadWake(t, "w1")
	require.NoError(t, err)

	// the wake item runs, so the queue drops it.
	require.NoError(t, h.shard.Dequeue(h.ctx, *item))
	_, err = h.loadWake(t, "w1")
	require.ErrorIs(t, err, queue.ErrQueueItemNotFound)

	// the same wake id from the next run is a new item.
	d = &loopReturnDriver{
		echo: true,
		ops: []*state.GeneratorOpcode{
			loopReturnOp(t, state.LoopReturn{
				State: json.RawMessage(`{"n":2}`),
				Wake: &state.LoopWake{
					ID:    "w1",
					After: "5m",
				},
			}),
		},
	}
	_, resp, err = h.runLoop(t, d, new(loops.NewWakeEvent("user-1", "w1")))
	require.NoError(t, err)
	require.Nil(t, resp.Err)
	again, err := h.loadWake(t, "w1")
	require.NoError(t, err)
	requireWithin(t, time.Now().Add(5*time.Minute), again.AtMS, 5*time.Second)
}

func TestValidateLoopResponseStepOps(t *testing.T) {
	fn := inngest.Function{
		Loop: &inngest.Loop{
			Kind: "loop",
			Key:  loopTestKeyExpr,
		},
	}
	resp := &state.DriverResponse{
		Generator: []*state.GeneratorOpcode{
			{
				ID:   "a",
				Op:   enums.OpcodeStepPlanned,
				Name: "a",
			},
			{
				ID:   "b",
				Op:   enums.OpcodeStepRun,
				Name: "b",
			},
		},
	}
	require.NoError(t, validateLoopResponse(fn, resp))
	resp.Generator = []*state.GeneratorOpcode{
		{
			ID: "sleep",
			Op: enums.OpcodeSleep,
		},
	}
	require.NoError(t, validateLoopResponse(fn, resp))
}

func TestLoopReturnTick(t *testing.T) {
	ret := func() *loopReturnDriver {
		return &loopReturnDriver{
			echo: true,
			ops: []*state.GeneratorOpcode{
				loopReturnOp(t, state.LoopReturn{
					State: json.RawMessage(`{"n":1}`),
				}),
			},
		}
	}

	t.Run("a return without end enqueues the tick", func(t *testing.T) {
		h := newLoopHarness(t)
		h.setTick(t, time.Minute)
		id, resp, err := h.runLoop(t, ret(), nil)
		require.NoError(t, err)
		require.Nil(t, resp.Err)

		item, err := h.loadTick(t)
		require.NoError(t, err)
		requireWithin(t, time.Now().Add(time.Minute), item.AtMS, 5*time.Second)
		require.Equal(t, queue.KindLoopTick, item.Data.Kind)
		require.Equal(t, queue.PayloadLoop{
			Key:   "user-1",
			Kind:  loops.InputTick,
			Nonce: id.RunID.String(),
		}, item.Data.Payload)
		require.Equal(t, h.req.Function.ID, item.Data.Identifier.WorkflowID)
		require.Equal(t, ulid.ULID{}, item.Data.Identifier.RunID)

		// a pending tick is kept by the next return, with its nonce.
		_, resp, err = h.runLoop(t, ret(), new(loops.NewTickEvent("user-1")))
		require.NoError(t, err)
		require.Nil(t, resp.Err)
		same, err := h.loadTick(t)
		require.NoError(t, err)
		require.Equal(t, item.AtMS, same.AtMS)
		require.Equal(t, item.Data.Payload, same.Data.Payload)
	})

	t.Run("no tick when the function has none", func(t *testing.T) {
		h := newLoopHarness(t)
		_, resp, err := h.runLoop(t, ret(), nil)
		require.NoError(t, err)
		require.Nil(t, resp.Err)
		_, err = h.loadTick(t)
		require.ErrorIs(t, err, queue.ErrQueueItemNotFound)
	})

	t.Run("no tick when the return ends the loop", func(t *testing.T) {
		h := newLoopHarness(t)
		h.setTick(t, time.Minute)
		d := &loopReturnDriver{
			echo: true,
			ops: []*state.GeneratorOpcode{
				loopReturnOp(t, state.LoopReturn{
					State: json.RawMessage(`{"n":1}`),
					End:   true,
				}),
			},
		}
		_, resp, err := h.runLoop(t, d, nil)
		require.NoError(t, err)
		require.Nil(t, resp.Err)
		_, err = h.loadTick(t)
		require.ErrorIs(t, err, queue.ErrQueueItemNotFound)
	})

	t.Run("a run that fails validation still ticks", func(t *testing.T) {
		h := newLoopHarness(t)
		h.setTick(t, time.Minute)
		d := &loopReturnDriver{
			output: map[string]any{"ok": true},
		}
		id, resp, err := h.runLoop(t, d, nil)
		h.requireLoopFailed(t, id, resp, err, "loop run ended without a LoopReturn")
		item, err := h.loadTick(t)
		require.NoError(t, err)
		requireWithin(t, time.Now().Add(time.Minute), item.AtMS, 5*time.Second)
	})

	t.Run("a cancelled run still ticks", func(t *testing.T) {
		h := newLoopHarness(t)
		h.setTick(t, time.Minute)
		req, id := h.request(nil)
		_, _, err := h.e.Schedule(h.ctx, req)
		require.NoError(t, err)

		require.NoError(t, h.e.Cancel(h.ctx, id, execution.CancelRequest{}))
		h.requireRunGone(t, id)
		require.Equal(t, event.FnCancelledName, h.lastFinished(t, event.FnCancelledName).Name)
		item, err := h.loadTick(t)
		require.NoError(t, err)
		requireWithin(t, time.Now().Add(time.Minute), item.AtMS, 5*time.Second)
	})

	t.Run("a run that fails permanently still ticks", func(t *testing.T) {
		h := newLoopHarness(t)
		h.setTick(t, time.Minute)
		req, id := h.request(nil)
		_, _, err := h.e.Schedule(h.ctx, req)
		require.NoError(t, err)
		scope := queue.Scope{
			AccountID:  id.Tenant.AccountID,
			EnvID:      id.Tenant.EnvID,
			FunctionID: id.FunctionID,
		}
		items, err := h.q.ItemsByRunID(h.ctx, h.shard, scope, id.RunID)
		require.NoError(t, err)
		require.Len(t, items, 1)

		require.NoError(t, h.e.FinalizePermanentlyFailedRun(h.ctx, items[0].Data, nil, errors.New("gave up")))
		h.requireRunGone(t, id)
		item, err := h.loadTick(t)
		require.NoError(t, err)
		requireWithin(t, time.Now().Add(time.Minute), item.AtMS, 5*time.Second)
	})
}

func TestLoopReturnWithStepRun(t *testing.T) {
	h := newLoopHarness(t)
	cs := &crashingStore{
		Store:       h.store,
		commitFails: 1,
	}
	h.e.loops = cs

	d := &loopReturnDriver{
		echo: true,
		ops: []*state.GeneratorOpcode{
			{
				ID:   "count",
				Op:   enums.OpcodeStepRun,
				Name: "count",
				Data: json.RawMessage(`{"data":10}`),
			},
			loopReturnOp(t, state.LoopReturn{
				State: json.RawMessage(`{"n":1}`),
			}),
		},
	}

	// the commit lands and the attempt fails after it.  the step output is
	// saved and no discovery item is enqueued next to the start item.
	id, _, err := h.runLoop(t, d, nil)
	require.ErrorIs(t, err, errStoreCrashed)
	require.Equal(t, 1, d.calls)
	steps, err := h.e.smv2.LoadStepsWithIDs(h.ctx, id, []string{"count"})
	require.NoError(t, err)
	require.Contains(t, steps, "count")
	scope := queue.Scope{
		AccountID:  id.Tenant.AccountID,
		EnvID:      id.Tenant.EnvID,
		FunctionID: id.FunctionID,
	}
	items, err := h.q.ItemsByRunID(h.ctx, h.shard, scope, id.RunID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
	require.NoError(t, err)
	require.Equal(t, int64(1), snap.Counter)

	// the retry finishes the run with the committed state.
	resp, err := h.retry(t, id, 1)
	require.NoError(t, err)
	require.Nil(t, resp.Err)
	h.requireRunGone(t, id)
	require.JSONEq(t, `{"n":1}`, h.finishedResult(t))
}

func TestLoopReturnWithStepRunCommit(t *testing.T) {
	h := newLoopHarness(t)
	d := &loopReturnDriver{
		echo: true,
		ops: []*state.GeneratorOpcode{
			{
				ID:   "count",
				Op:   enums.OpcodeStepRun,
				Name: "count",
				Data: json.RawMessage(`{"data":10}`),
			},
			loopReturnOp(t, state.LoopReturn{
				State: json.RawMessage(`{"n":1}`),
			}),
		},
	}
	id, resp, err := h.runLoop(t, d, nil)
	require.NoError(t, err)
	require.Nil(t, resp.Err)
	require.Equal(t, 1, d.calls)

	snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
	require.NoError(t, err)
	require.Equal(t, int64(1), snap.Counter)
	require.JSONEq(t, `{"n":1}`, string(snap.State))
	h.requireRunGone(t, id)
	require.JSONEq(t, `{"n":1}`, h.finishedResult(t))
	require.Len(t, resp.Generator, 2)
	require.Equal(t, enums.OpcodeStepRun, resp.Generator[0].Op)
	require.Equal(t, enums.OpcodeRunComplete, resp.Generator[1].Op)
}

func TestLoopReturnValidation(t *testing.T) {
	bigState, err := json.Marshal(strings.Repeat("x", consts.MaxLoopStateSize))
	require.NoError(t, err)

	tests := []struct {
		name string
		// noLoop runs the function without a loop config.
		noLoop bool
		driver *loopReturnDriver
		want   string
	}{
		{
			name: "two LoopReturns",
			driver: &loopReturnDriver{
				ops: []*state.GeneratorOpcode{
					loopReturnOp(t, state.LoopReturn{
						State:   json.RawMessage(`{}`),
						Version: 0,
					}),
					loopReturnOp(t, state.LoopReturn{
						State:   json.RawMessage(`{}`),
						Version: 0,
					}),
				},
			},
			want: "one LoopReturn",
		},
		{
			name: "Sleep with a LoopReturn",
			driver: &loopReturnDriver{
				ops: []*state.GeneratorOpcode{
					{
						ID:   "nap",
						Op:   enums.OpcodeSleep,
						Name: "nap",
					},
					loopReturnOp(t, state.LoopReturn{
						State:   json.RawMessage(`{}`),
						Version: 0,
					}),
				},
			},
			want: "a LoopReturn may only be returned with step.run outputs, got Sleep",
		},
		{
			name:   "function without Loop",
			noLoop: true,
			driver: &loopReturnDriver{
				ops: []*state.GeneratorOpcode{
					loopReturnOp(t, state.LoopReturn{
						State:   json.RawMessage(`{}`),
						Version: 0,
					}),
				},
			},
			want: "is not a loop",
		},
		{
			name: "subscribe present",
			driver: &loopReturnDriver{
				ops: []*state.GeneratorOpcode{
					loopReturnOp(t, state.LoopReturn{
						State:   json.RawMessage(`{}`),
						Version: 0,
						Subscribe: []state.LoopSubscribe{
							{
								ID:    "s1",
								Event: "app/event",
							},
						},
					}),
				},
			},
			want: "subscriptions are not supported yet",
		},
		{
			name: "unsubscribe present",
			driver: &loopReturnDriver{
				ops: []*state.GeneratorOpcode{
					loopReturnOp(t, state.LoopReturn{
						State:       json.RawMessage(`{}`),
						Version:     0,
						Unsubscribe: []string{"s1"},
					}),
				},
			},
			want: "subscriptions are not supported yet",
		},
		{
			name: "state over the cap",
			driver: &loopReturnDriver{
				ops: []*state.GeneratorOpcode{
					loopReturnOp(t, state.LoopReturn{
						State:   bigState,
						Version: 0,
					}),
				},
			},
			want: fmt.Sprintf("the limit is %d bytes", consts.MaxLoopStateSize),
		},
		{
			name: "wake with a zero duration",
			driver: &loopReturnDriver{
				ops: []*state.GeneratorOpcode{
					loopReturnOp(t, state.LoopReturn{
						State:   json.RawMessage(`{}`),
						Version: 0,
						Wake: &state.LoopWake{
							ID:    "w1",
							After: "0s",
						},
					}),
				},
			},
			want: "invalid duration",
		},
		{
			name: "wake with a negative duration",
			driver: &loopReturnDriver{
				ops: []*state.GeneratorOpcode{
					loopReturnOp(t, state.LoopReturn{
						State:   json.RawMessage(`{}`),
						Version: 0,
						Wake: &state.LoopWake{
							ID:    "w1",
							After: "-5s",
						},
					}),
				},
			},
			want: "invalid duration",
		},
		{
			name: "wake past the limit",
			driver: &loopReturnDriver{
				ops: []*state.GeneratorOpcode{
					loopReturnOp(t, state.LoopReturn{
						State:   json.RawMessage(`{}`),
						Version: 0,
						Wake: &state.LoopWake{
							ID:    "w1",
							After: "169h",
						},
					}),
				},
			},
			want: "the limit is the idle_ttl of 168h0m0s",
		},
		{
			name: "wake with a bad duration",
			driver: &loopReturnDriver{
				ops: []*state.GeneratorOpcode{
					loopReturnOp(t, state.LoopReturn{
						State:   json.RawMessage(`{}`),
						Version: 0,
						Wake: &state.LoopWake{
							ID:    "w1",
							After: "soon",
						},
					}),
				},
			},
			want: "invalid duration",
		},
		{
			name: "return without a version",
			driver: &loopReturnDriver{
				ops: []*state.GeneratorOpcode{
					{
						ID:   "loop-return",
						Op:   enums.OpcodeLoopReturn,
						Data: json.RawMessage(`{"state":{}}`),
					},
				},
			},
			want: "invalid LoopReturn",
		},
		{
			name: "plain 200",
			driver: &loopReturnDriver{
				output: map[string]any{"ok": true},
			},
			want: "loop run ended without a LoopReturn",
		},
		{
			name: "SyncRunComplete from a loop",
			driver: &loopReturnDriver{
				ops: []*state.GeneratorOpcode{
					{
						ID:   "done",
						Op:   enums.OpcodeSyncRunComplete,
						Data: json.RawMessage(`{"data":{"status":200}}`),
					},
				},
			},
			want: "loop run ended without a LoopReturn",
		},
		{
			name: "RunComplete from a loop",
			driver: &loopReturnDriver{
				ops: []*state.GeneratorOpcode{
					{
						ID:   "done",
						Op:   enums.OpcodeRunComplete,
						Data: json.RawMessage(`{"ok":true}`),
					},
				},
			},
			want: "loop run ended without a LoopReturn",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newLoopHarness(t)
			if tt.noLoop {
				h.req.Function.Loop = nil
				h.req.Function.Concurrency = nil
				fn := h.req.Function
				h.e.fl = singletonRetryFunctionLoader{fn: &fn}
			}

			id, resp, err := h.runLoop(t, tt.driver, nil)
			h.requireLoopFailed(t, id, resp, err, tt.want)
			require.Equal(t, 1, tt.driver.calls)

			if tt.noLoop {
				_, err := h.store.Load(h.ctx, h.loopID("user-1"))
				require.ErrorIs(t, err, loops.ErrNotFound)
				return
			}
			snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
			require.NoError(t, err)
			require.Equal(t, int64(0), snap.Counter)
			require.Nil(t, snap.State)
			_, err = h.loadWake(t, "w1")
			require.ErrorIs(t, err, queue.ErrQueueItemNotFound)
		})
	}
}
