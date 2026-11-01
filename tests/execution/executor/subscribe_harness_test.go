package executor

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/expressions/expragg"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// This file is the seam 2 harness of step.subscribe: it drives the executor's
// arrival, timer and wait handlers against Redis and stops a handler between
// two of its writes, so a test can check that a redelivery repairs the gap.
//
// A crash is injected by wrapping the state service or the queue, which are
// plain interfaces: the wrapper does the real write, then fails the call as
// a process death would after the write landed. Nothing in production code
// knows about it.

var errCrash = errors.New("injected crash")

// crashPoints holds the armed crash points. A point is "after:<Method>" or
// "before:<Method>" and fires once, then disarms.
type crashPoints struct {
	mu    sync.Mutex
	armed map[string]bool
}

func newCrashPoints() *crashPoints {
	return &crashPoints{armed: map[string]bool{}}
}

func (c *crashPoints) arm(point string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.armed[point] = true
}

func (c *crashPoints) fire(point string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.armed[point] {
		delete(c.armed, point)

		return errCrash
	}

	return nil
}

// crashingState fails a state write after it was made.
type crashingState struct {
	statev2.RunService
	crash *crashPoints
}

func (s crashingState) ConsumePause(ctx context.Context, p state.Pause, opts state.ConsumePauseOpts) (state.ConsumePauseResult, error) {
	if err := s.crash.fire("before:ConsumePause"); err != nil {
		return state.ConsumePauseResult{}, err
	}

	res, err := s.RunService.ConsumePause(ctx, p, opts)
	if err != nil {
		return res, err
	}

	return res, s.crash.fire("after:ConsumePause")
}

func (s crashingState) SaveStep(ctx context.Context, id statev2.ID, stepID string, data []byte) (bool, error) {
	if err := s.crash.fire("before:SaveStep"); err != nil {
		return false, err
	}

	pending, err := s.RunService.SaveStep(ctx, id, stepID, data)
	if err != nil {
		return pending, err
	}

	return pending, s.crash.fire("after:SaveStep")
}

// crashingQueue fails a queue write around the real one.
type crashingQueue struct {
	queue.Queue
	crash *crashPoints
}

func (q crashingQueue) Enqueue(ctx context.Context, item queue.Item, at time.Time, opts queue.EnqueueOpts) error {
	if err := q.crash.fire("before:Enqueue"); err != nil {
		return err
	}

	if err := q.Queue.Enqueue(ctx, item, at, opts); err != nil {
		return err
	}

	return q.crash.fire("after:Enqueue")
}

// subscribeHarness is a run waiting on pauses, with its executor, state and
// queue in reach, and the crash points that stop the executor's handlers.
type subscribeHarness struct {
	t     *testing.T
	infra *execTestInfra
	exec  execution.Executor
	crash *crashPoints
	run   *statev2.Metadata
}

// newSubscribeHarness starts a run whose first request reports ops, then
// executes that request so the executor has handled them (a WaitForEvent op
// leaves a pause and a timeout job).
func newSubscribeHarness(t *testing.T, ops ...*state.GeneratorOpcode) *subscribeHarness {
	t.Helper()

	infra := newExecTestInfra(t, "step")
	crash := newCrashPoints()
	driver := &mockDriverV1{
		t: t,
		response: &state.DriverResponse{
			StatusCode:     206,
			RequestVersion: 2,
			Generator:      ops,
		},
	}
	exec := infra.newExecutorWithQueue(t,
		crashingQueue{Queue: infra.rq, crash: crash},
		executor.WithStateManager(crashingState{RunService: infra.smv2, crash: crash}),
		executor.WithDriverV1(driver),
		// Event matching evaluates pauses through the aggregator, as the Dev
		// Server wires it.
		executor.WithExpressionAggregator(
			expragg.NewAggregator(infra.ctx, 100, 100, infra.pauseMgr, expressions.ExprEvaluator, nil, nil),
		),
	)
	run := infra.scheduleRun(t, exec)

	jobs, err := infra.rq.RunJobs(infra.ctx, infra.queueShard.Name(), queue.Scope{
		AccountID:  run.ID.Tenant.AccountID,
		EnvID:      run.ID.Tenant.EnvID,
		FunctionID: run.ID.FunctionID,
	}, run.ID.RunID, 1000, 0)
	require.NoError(t, err)
	require.NotEmpty(t, jobs)

	raw, ok := jobs[0].Raw.(*queue.QueueItem)
	require.True(t, ok, "RunJobs Raw must be a *queue.QueueItem, got %T", jobs[0].Raw)

	id := statev2.V1FromMetadata(*run)
	edge := inngest.Edge{Incoming: "$trigger", Outgoing: "step"}
	_, err = exec.Execute(queue.WithJobID(infra.ctx, jobs[0].JobID), id, raw.Data, edge)
	require.NoError(t, err)

	return &subscribeHarness{t: t, infra: infra, exec: exec, crash: crash, run: run}
}

// waitOp is the op an SDK reports for step.waitForEvent.
func waitOp(id string, eventName string) *state.GeneratorOpcode {
	return &state.GeneratorOpcode{
		Op:   enums.OpcodeWaitForEvent,
		ID:   id,
		Name: id,
		Opts: map[string]any{"event": eventName, "timeout": "1m"},
	}
}

// pause loads the pause the executor wrote for op id.
func (h *subscribeHarness) pause(eventName, opID string) *state.Pause {
	h.t.Helper()

	pauseID := inngest.DeterministicSha1UUID(h.run.ID.RunID.String() + opID)
	p, err := h.infra.pauseMgr.PauseByID(h.infra.ctx, pauses.Index{
		WorkspaceID: h.run.ID.Tenant.EnvID,
		EventName:   eventName,
	}, pauseID)
	if err != nil {
		return nil
	}

	return p
}

// deliver hands an event to the executor the way the event stream does.
func (h *subscribeHarness) deliver(evt event.Event) error {
	h.t.Helper()

	tracked := event.BaseTrackedEvent{
		ID:          ulid.MustNew(ulid.Now(), rand.Reader),
		Event:       evt,
		WorkspaceID: h.run.ID.Tenant.EnvID,
	}
	_, err := h.exec.HandlePauses(h.infra.ctx, tracked)

	return err
}

// steps is every step saved in the run's state, by step id.
func (h *subscribeHarness) steps() map[string]string {
	h.t.Helper()

	saved, err := h.infra.smv2.LoadSteps(h.infra.ctx, h.run.ID)
	require.NoError(h.t, err)

	out := map[string]string{}
	for id, data := range saved {
		out[id] = string(data)
	}

	return out
}

// jobIDs is the job id of every queue item the run has.
func (h *subscribeHarness) jobIDs() []string {
	h.t.Helper()

	jobs, err := h.infra.rq.RunJobs(h.infra.ctx, h.infra.queueShard.Name(), queue.Scope{
		AccountID:  h.run.ID.Tenant.AccountID,
		EnvID:      h.run.ID.Tenant.EnvID,
		FunctionID: h.run.ID.FunctionID,
	}, h.run.ID.RunID, 1000, 0)
	require.NoError(h.t, err)

	ids := make([]string, 0, len(jobs))
	for _, job := range jobs {
		ids = append(ids, job.JobID)
	}

	return ids
}

// discoveryJobID is the id of the job that runs the step after a wait, as the
// queue stores it (hashed). A wait's event path uses "{idempotency key}-{step
// id}-event", the same id on every delivery of the event.
func (h *subscribeHarness) discoveryJobID(p *state.Pause) string {
	return queue.HashID(h.infra.ctx, fmt.Sprintf("%s-%s-event", h.run.IdempotencyKey(), p.DataKey))
}

func timesIn(ids []string, want string) int {
	n := 0
	for _, id := range ids {
		if id == want {
			n++
		}
	}

	return n
}
