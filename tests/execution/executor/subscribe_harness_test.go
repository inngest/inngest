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

func (s crashingState) SaveInboxMessage(ctx context.Context, id statev2.ID, msg statev2.InboxMessage) ([]statev2.InboxWake, error) {
	if err := s.crash.fire("before:SaveInboxMessage"); err != nil {
		return nil, err
	}

	wakes, err := s.RunService.SaveInboxMessage(ctx, id, msg)
	if err != nil {
		return wakes, err
	}

	return wakes, s.crash.fire("after:SaveInboxMessage")
}

func (s crashingState) TimeoutInboxWait(ctx context.Context, id statev2.ID, waitID string, data []byte) (*statev2.InboxWake, error) {
	if err := s.crash.fire("before:TimeoutInboxWait"); err != nil {
		return nil, err
	}

	wake, err := s.RunService.TimeoutInboxWait(ctx, id, waitID, data)
	if err != nil {
		return wake, err
	}

	return wake, s.crash.fire("after:TimeoutInboxWait")
}

func (s crashingState) CloseInbox(ctx context.Context, id statev2.ID, closing statev2.InboxClose) ([]statev2.InboxWake, bool, error) {
	if err := s.crash.fire("before:CloseInbox"); err != nil {
		return nil, false, err
	}

	wakes, pending, err := s.RunService.CloseInbox(ctx, id, closing)
	if err != nil {
		return wakes, pending, err
	}

	return wakes, pending, s.crash.fire("after:CloseInbox")
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
	t      *testing.T
	infra  *execTestInfra
	exec   execution.Executor
	crash  *crashPoints
	run    *statev2.Metadata
	driver *mockDriverV1
	job    queue.JobResponse
	item   *queue.QueueItem
	edge   inngest.Edge
}

// newSubscribeHarness starts a run whose first request reports ops, then
// executes that request so the executor has handled them (a WaitForEvent op
// leaves a pause and a timeout job).
func newSubscribeHarness(t *testing.T, ops ...*state.GeneratorOpcode) *subscribeHarness {
	t.Helper()

	return newSubscribeHarnessWith(t, nil, ops...)
}

// newSubscribeHarnessWith is newSubscribeHarness with extra executor options,
// such as lowered step or state size limits.
func newSubscribeHarnessWith(t *testing.T, opts []executor.ExecutorOpt, ops ...*state.GeneratorOpcode) *subscribeHarness {
	t.Helper()

	return newSubscribeHarnessTriggered(t, nil, opts, ops...)
}

// newSubscribeHarnessTriggered is newSubscribeHarnessWith for a run whose
// trigger event carries data.
func newSubscribeHarnessTriggered(t *testing.T, triggerData map[string]any, opts []executor.ExecutorOpt, ops ...*state.GeneratorOpcode) *subscribeHarness {
	t.Helper()

	infra := newExecTestInfra(t, "step")
	infra.triggerData = triggerData
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
		append([]executor.ExecutorOpt{
			executor.WithStateManager(crashingState{RunService: infra.smv2, crash: crash}),
			executor.WithDriverV1(driver),
			// Event matching evaluates pauses through the aggregator, as the Dev
			// Server wires it.
			executor.WithExpressionAggregator(
				expragg.NewAggregator(infra.ctx, 100, 100, infra.pauseMgr, expressions.ExprEvaluator, nil, nil),
			),
		}, opts...)...,
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

	h := &subscribeHarness{
		t:      t,
		infra:  infra,
		exec:   exec,
		crash:  crash,
		run:    run,
		driver: driver,
		job:    jobs[0],
		item:   raw,
		edge:   inngest.Edge{Incoming: "$trigger", Outgoing: "step"},
	}
	h.execute()

	return h
}

// execute runs the run's first request against the driver's current response.
func (h *subscribeHarness) execute() {
	h.t.Helper()

	require.NoError(h.t, h.tryExecute())
}

// tryExecute is execute for a test that expects the request to fail.
func (h *subscribeHarness) tryExecute() error {
	h.t.Helper()

	id := statev2.V1FromMetadata(*h.run)
	_, err := h.exec.Execute(queue.WithJobID(h.infra.ctx, h.job.JobID), id, h.item.Data, h.edge)

	return err
}

// report makes the SDK's next request report ops, as a later request of the
// run would, and has the executor handle them.
func (h *subscribeHarness) report(ops ...*state.GeneratorOpcode) {
	h.t.Helper()

	h.driver.response = &state.DriverResponse{
		StatusCode:     206,
		RequestVersion: 2,
		Generator:      ops,
	}
	h.execute()
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

	return h.deliverID(ulid.MustNew(ulid.Now(), rand.Reader), evt)
}

// deliverID delivers an event under a given internal ID, as a redelivery of
// the same event does.
func (h *subscribeHarness) deliverID(id ulid.ULID, evt event.Event) error {
	h.t.Helper()

	tracked := event.BaseTrackedEvent{
		ID:          id,
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
