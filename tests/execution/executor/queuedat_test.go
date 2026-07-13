package executor

import (
	"context"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/util/interval"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Regression tests to prevent stale inherited queuedAt on async opcodes
//
// An async opcode (invoke / sleep / waitForEvent / planned step) reported by a
// request whose queue item was enqueued earlier — e.g. because preceding steps
// executed inline in that request under checkpointing — must NOT inherit the
// item's enqueue time as its step span's queuedAt.
//
// We provide Opcode payloads from JS SDK runs. We replay them through the
// executor with a mock driver. We backdate the queue item's EnqueuedAt to set
// the bug's trigger.

// queuedAtScenario replays a captured opcode fixture through the executor and
// returns the persisted span tree plus the timing anchors assertions need.
type queuedAtScenario struct {
	root *cqrs.OtelSpan
	// itemEnqueuedAt is the (backdated) enqueue time of the queue item whose
	// execution reported the opcodes — the value the bug leaked into queuedAt.
	itemEnqueuedAt time.Time
	// executeStart is just before exec.Execute ran; opcode handling (and so
	// the fixed queuedAt) can only happen after this.
	executeStart time.Time
}

func runQueuedAtScenario(t *testing.T, opcodes []*state.GeneratorOpcode) queuedAtScenario {
	t.Helper()

	require.NotEmpty(t, opcodes)

	infra := newExecTestInfra(t, "step")
	ctx := infra.ctx

	mockDriver := &mockDriverV1{
		t: t,
		response: &state.DriverResponse{
			StatusCode:     206,
			RequestVersion: 2,
			Generator:      opcodes,
		},
	}

	exec := infra.newExecutor(t,
		executor.WithInvokeEventHandler(func(ctx context.Context, evt event.TrackedEvent) error { return nil }),
		executor.WithDriverV1(mockDriver),
	)

	run := infra.scheduleRun(t, exec)

	jobs, err := infra.rq.RunJobs(ctx, infra.queueShard.Name(), queue.Scope{
		AccountID:  run.ID.Tenant.AccountID,
		EnvID:      run.ID.Tenant.EnvID,
		FunctionID: run.ID.FunctionID,
	}, run.ID.RunID, 1000, 0)
	require.NoError(t, err)
	require.NotEmpty(t, jobs)

	// The bug's trigger: the request that reports the opcodes was enqueued
	// well before the opcodes are handled (in the wild: the request ran
	// checkpointed sibling steps first; the captured fixtures record ~2s).
	// Backdate far enough that inheriting it is unmistakable.
	itemEnqueuedAt := time.Now().Add(-2 * time.Minute).Truncate(time.Millisecond)

	// Execute with the item Schedule actually enqueued — its Metadata carries
	// the otel propagation carrier that parents the step spans — overriding
	// only its enqueue time.
	rawItem, ok := jobs[0].Raw.(*queue.QueueItem)
	require.True(t, ok, "RunJobs Raw must be a *queue.QueueItem, got %T", jobs[0].Raw)
	item := rawItem.Data
	item.EnqueuedAt = itemEnqueuedAt

	jobCtx := queue.WithJobID(ctx, jobs[0].JobID)
	id := sv2.V1FromMetadata(*run)
	edge := inngest.Edge{Incoming: "$trigger", Outgoing: "step"}

	executeStart := time.Now()
	_, err = exec.Execute(jobCtx, id, item, edge)
	require.NoError(t, err)

	// Span writes flush on the sqlc exporter's batching interval; poll until
	// every opcode's step span is present with a queuedAt.
	var root *cqrs.OtelSpan
	require.Eventually(t, func() bool {
		root, err = infra.dbcqrs.GetSpansByRunID(ctx, run.ID.RunID)
		if err != nil || root == nil {
			return false
		}
		found := 0
		for _, op := range opcodes {
			if span := findStepSpan(root, op.ID); span != nil && span.Attributes != nil && span.Attributes.QueuedAt != nil {
				found++
			}
		}
		return found == len(opcodes)
	}, 10*time.Second, 100*time.Millisecond, "step spans for all fixture opcodes must be persisted with a queuedAt attribute")

	return queuedAtScenario{root: root, itemEnqueuedAt: itemEnqueuedAt, executeStart: executeStart}
}

// findStepSpan walks the span tree for the "executor.step" span whose stepID
// attribute matches the opcode ID.
func findStepSpan(span *cqrs.OtelSpan, stepID string) *cqrs.OtelSpan {
	if span == nil {
		return nil
	}
	if span.Name == meta.SpanNameStep && span.Attributes != nil &&
		span.Attributes.StepID != nil && *span.Attributes.StepID == stepID {
		return span
	}
	for _, child := range span.Children {
		if found := findStepSpan(child, stepID); found != nil {
			return found
		}
	}
	return nil
}

// findSpanByName walks the span tree for the first span with the given name.
func findSpanByName(span *cqrs.OtelSpan, name string) *cqrs.OtelSpan {
	if span == nil {
		return nil
	}
	if span.Name == name {
		return span
	}
	for _, child := range span.Children {
		if found := findSpanByName(child, name); found != nil {
			return found
		}
	}
	return nil
}

// queuedAtFixtureInvoke is the captured single-invoke scenario.
//
//	await step.run("step-before", async () => {
//		await new Promise((r) => setTimeout(r, 2000));
//	});
//	const child = await step.invoke("invoke-child", { // -> the opcode below
//		function: invokedChild, // -> Opts.function_id
//		data: { testCase: "capture/invoke", childSleep: "8s" },
//	});
var queuedAtFixtureInvoke = []*state.GeneratorOpcode{
	{
		Op:   enums.OpcodeInvokeFunction,
		ID:   "7ff123e6431be676220df2842b3ee0fd2b288a86",
		Name: "",
		Opts: map[string]any{
			"function_id": "test-app-invoked-child",
			"payload": map[string]any{
				"data": map[string]any{
					"childSleep": "8s",
					"testCase":   "capture/invoke",
				},
			},
		},
		Data:        nil,
		Error:       nil,
		DisplayName: new("invoke-child"),
		Timing:      interval.Interval{A: 0, B: 0},
	},
}

// queuedAtFixtureSleep is the captured sleep scenario.
//
// Producing JS:
//
//	await step.run("step-before", async () => {
//		await new Promise((r) => setTimeout(r, 2000));
//	});
//	await step.sleep("nap", "5s"); // -> the opcode below
var queuedAtFixtureSleep = []*state.GeneratorOpcode{
	{
		Op:          enums.OpcodeSleep,
		ID:          "c2640f79b4ed481b838ce4ad75330aa3f825d4d9",
		Name:        "5s",
		Opts:        map[string]any{},
		Data:        nil,
		Error:       nil,
		DisplayName: new("nap"),
		Timing:      interval.Interval{A: 0, B: 0},
	},
}

// queuedAtFixtureWaitForEvent is the captured waitForEvent scenario.
//
// Producing JS:
//
//	await step.run("step-before", async () => {
//		await new Promise((r) => setTimeout(r, 2000));
//	});
//	await step.waitForEvent("wait", { // -> the opcode below
//		event: "exe1997-never", // never sent; times out
//		timeout: "5s",
//	});
var queuedAtFixtureWaitForEvent = []*state.GeneratorOpcode{
	{
		Op:   enums.OpcodeWaitForEvent,
		ID:   "daaad336276d15594d0e765f96c17cd746bf4971",
		Name: "exe1997-never",
		Opts: map[string]any{
			"timeout": "5s",
		},
		Data:        nil,
		Error:       nil,
		DisplayName: new("wait"),
		Timing:      interval.Interval{A: 0, B: 0},
	},
}

// TestAsyncOpQueuedAtNotInheritedFromRequest is the stale-inherited-queuedAt
// regression: each async opcode's step span must be stamped queuedAt at
// opcode-handling time, not with the reporting request's (older) enqueue
// time.
func TestAsyncOpQueuedAtNotInheritedFromRequest(t *testing.T) {
	for _, tc := range []struct {
		name    string
		opcodes []*state.GeneratorOpcode
	}{
		{name: "invoke", opcodes: queuedAtFixtureInvoke},
		{name: "sleep", opcodes: queuedAtFixtureSleep},
		{name: "waitforevent", opcodes: queuedAtFixtureWaitForEvent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sc := runQueuedAtScenario(t, tc.opcodes)

			for _, op := range tc.opcodes {
				span := findStepSpan(sc.root, op.ID)
				require.NotNil(t, span, "step span for opcode %s", op.ID)

				queuedAt := span.GetQueuedAtTime()
				// Stored queuedAt is millisecond-granular; truncate the anchor
				// so sub-ms rounding can't flake the comparison.
				assert.False(t,
					queuedAt.Before(sc.executeStart.Truncate(time.Millisecond)),
					"async op queuedAt (%s) must not predate opcode handling (execute started %s); inheriting the request's enqueue time (%s) is a stale-inherited-queuedAt bug",
					queuedAt, sc.executeStart, sc.itemEnqueuedAt)
			}
		})
	}
}

// queuedAtFixtureParallelInvoke is the captured parallel two-invoke scenario.
//
// Producing JS
// - no preceding step: both invokes arrive as one opcode group
// on the run's first request
//
//	await Promise.all([
//		step.invoke("invoke-a", { // -> opcode [0]
//			function: invokedChild, // -> Opts.function_id
//			data: { testCase: "capture/parallel-invoke", childSleep: "5s" },
//		}),
//		step.invoke("invoke-b", { // -> opcode [1]
//			function: invokedChild,
//			data: { testCase: "capture/parallel-invoke", childSleep: "5s" },
//		}),
//	]);
var queuedAtFixtureParallelInvoke = []*state.GeneratorOpcode{
	{
		Op:   enums.OpcodeInvokeFunction,
		ID:   "017fd6c24a110fb73efa99fc93fc99a591775192",
		Name: "",
		Opts: map[string]any{
			"function_id": "test-app-invoked-child",
			"payload": map[string]any{
				"data": map[string]any{
					"childSleep": "5s",
					"testCase":   "capture/parallel-invoke",
				},
			},
		},
		Data:        nil,
		Error:       nil,
		DisplayName: new("invoke-a"),
		Timing:      interval.Interval{A: 0, B: 0},
	},
	{
		Op:   enums.OpcodeInvokeFunction,
		ID:   "98f8c99b8a4344641386da72ded706eacedfaf0b",
		Name: "",
		Opts: map[string]any{
			"function_id": "test-app-invoked-child",
			"payload": map[string]any{
				"data": map[string]any{
					"childSleep": "5s",
					"testCase":   "capture/parallel-invoke",
				},
			},
		},
		Data:        nil,
		Error:       nil,
		DisplayName: new("invoke-b"),
		Timing:      interval.Interval{A: 0, B: 0},
	},
}

// queuedAtFixtureParallelStepPlanned is the captured parallel two-planned-steps scenario.
//
// Producing JS
// - no preceding step: parallel step.run calls are first reported as planned
// steps, one opcode group on the run's first request
//
//	await Promise.all([
//		step.run("step-a", async () => "a"), // -> opcode [0] (OpcodeStepPlanned)
//		step.run("step-b", async () => "b"), // -> opcode [1]
//	]);
var queuedAtFixtureParallelStepPlanned = []*state.GeneratorOpcode{
	{
		Op:          enums.OpcodeStepPlanned,
		ID:          "1419522417cff8c6ddd205cc9882d16411490c09",
		Name:        "step-a",
		Opts:        map[string]any{},
		Data:        nil,
		Error:       nil,
		DisplayName: new("step-a"),
		Timing:      interval.Interval{A: 0, B: 0},
	},
	{
		Op:          enums.OpcodeStepPlanned,
		ID:          "fe0ff3a9f16e8030660e5c6c446c04439143a0d1",
		Name:        "step-b",
		Opts:        map[string]any{},
		Data:        nil,
		Error:       nil,
		DisplayName: new("step-b"),
		Timing:      interval.Interval{A: 0, B: 0},
	},
}

// TestParallelOpcodesShareQueuedAt pins the fan-out mitigation and nets the
// stale-inherited-queuedAt bug for the parallel handlers. It asserts BOTH
// properties:
//   - (a) siblings must not inherit the reporting request's enqueue time
//     (netting the stepPlanned handler that the single-opcode test does not
//     exercise); and
//   - (b) opcodes arriving in one SDK response share a single queuedAt so the
//     trace order of parallel siblings stays deterministic under the UI's
//     stable sort.
func TestParallelOpcodesShareQueuedAt(t *testing.T) {
	for _, tc := range []struct {
		name    string
		opcodes []*state.GeneratorOpcode
	}{
		{name: "parallel_invoke", opcodes: queuedAtFixtureParallelInvoke},
		{name: "parallel_stepplanned", opcodes: queuedAtFixtureParallelStepPlanned},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sc := runQueuedAtScenario(t, tc.opcodes)
			require.GreaterOrEqual(t, len(tc.opcodes), 2, "parallel fixture must carry multiple opcodes")

			var first time.Time
			for i, op := range tc.opcodes {
				span := findStepSpan(sc.root, op.ID)
				require.NotNil(t, span, "step span for opcode %s", op.ID)
				queuedAt := span.GetQueuedAtTime()

				// Stale-inherited-queuedAt non-inheritance net for the
				// stepPlanned and parallel-invoke handlers (the single-opcode
				// test covers invoke/sleep/waitForEvent). Applies to every sibling,
				// including index 0: pre-fix all siblings inherit the SAME
				// stale time, so the share-queuedAt check alone would pass.
				assert.False(t, queuedAt.Before(sc.executeStart.Truncate(time.Millisecond)),
					"parallel op %s queuedAt (%s) must not predate opcode handling (execute started %s); inheriting the request's enqueue time (%s) is a stale-inherited-queuedAt bug",
					op.ID, queuedAt, sc.executeStart, sc.itemEnqueuedAt)

				if i == 0 {
					first = queuedAt
					continue
				}
				assert.True(t, first.Equal(queuedAt),
					"parallel opcode %s queuedAt (%s) must equal its first sibling's (%s): opcodes in one response share one queuedAt (sibling order is goroutine-scheduling-dependent otherwise)",
					op.ID, queuedAt, first)
			}
		})
	}
}

// TestExecutionSpanKeepsItemEnqueuedAt pins the path the
// stale-inherited-queuedAt fix must NOT change: the execution (attempt)
// span's queuedAt is the queue item's real enqueue time — that is a genuine
// queue wait, unlike the async-op spans'.
func TestExecutionSpanKeepsItemEnqueuedAt(t *testing.T) {
	sc := runQueuedAtScenario(t, queuedAtFixtureInvoke)

	execSpan := findSpanByName(sc.root, meta.SpanNameExecution)
	require.NotNil(t, execSpan, "execution span must exist")
	require.NotNil(t, execSpan.Attributes)
	require.NotNil(t, execSpan.Attributes.QueuedAt)

	assert.Equal(t, sc.itemEnqueuedAt.UnixMilli(), execSpan.Attributes.QueuedAt.UnixMilli(),
		"execution span queuedAt must remain the queue item's enqueue time")
}
