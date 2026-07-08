package executor

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/stretchr/testify/require"
)

func findEventByName(events []event.Event, name string) *event.Event {
	for i, evt := range events {
		if evt.Name == name {
			return &events[i]
		}
	}
	return nil
}

func deferScheduleFnSlugs(t *testing.T, events []event.Event) []string {
	t.Helper()
	var slugs []string
	for _, evt := range events {
		if evt.Name != consts.FnDeferScheduleName {
			continue
		}
		data := eventDataMap(t, evt)
		inn := data["_inngest"].(map[string]any)
		slugs = append(slugs, inn["fn_slug"].(string))
	}
	return slugs
}

func eventDataMap(t *testing.T, evt event.Event) map[string]any {
	t.Helper()
	r := require.New(t)
	raw, err := json.Marshal(evt.Data)
	r.NoError(err)
	var data map[string]any
	r.NoError(json.Unmarshal(raw, &data))
	return data
}

func TestRunError(t *testing.T) {
	userErr := &state.UserError{Name: "CustomError", Message: "step blew up"}

	t.Run("terminal failure with exhausted retries emits function.failed and keeps AfterRun defer", func(t *testing.T) {
		r := require.New(t)
		infra := newDeferTestInfra(t)
		countingQ := &enqueueCountingQueue{Queue: infra.rq}

		stepID := "step-defer"
		driver := &mockDriverV1{
			t: t,
			response: &state.DriverResponse{
				StatusCode: 206,
				Generator: []*state.GeneratorOpcode{
					{
						Op: enums.OpcodeDeferAdd,
						ID: stepID,
						Opts: map[string]any{
							"fn_slug": "onDefer-score",
							"input":   map[string]any{},
						},
					},
					{
						Op:    enums.OpcodeRunError,
						ID:    "run-error",
						Error: userErr,
					},
				},
			},
		}

		exec := infra.newExecutorWithQueue(t, countingQ, driver)

		var capturedEvents []event.Event
		exec.SetFinalizer(func(ctx context.Context, id statev2.ID, events []event.Event) error {
			capturedEvents = append(capturedEvents, events...)
			return nil
		})

		run := infra.scheduleRun(t, exec)
		countBeforeExecute := countingQ.enqueues

		_, err := exec.Execute(infra.ctx, state.Identifier{
			WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID,
		}, queue.Item{
			WorkspaceID: infra.wsID,
			Kind:        queue.KindStart,
			Attempt:     4,
			Identifier:  state.Identifier{WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID},
			Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: stepID}},
		}, inngest.Edge{Incoming: "$trigger", Outgoing: stepID})
		r.NoError(err)

		enqueuesDuringExecute := countingQ.enqueues - countBeforeExecute
		r.Equal(0, enqueuesDuringExecute,
			"terminal RunError piggybacked with DeferAdd must not enqueue discovery; got %d enqueues", enqueuesDuringExecute)

		r.Equal([]string{"onDefer-score"}, deferScheduleFnSlugs(t, capturedEvents),
			"AfterRun defer must still be scheduled when the run terminates via RunError")

		failedEvt := findEventByName(capturedEvents, event.FnFailedName)
		r.NotNil(failedEvt, "inngest/function.failed must be emitted on terminal RunError")

		data := eventDataMap(t, *failedEvt)
		errData, ok := data["error"].(map[string]any)
		r.True(ok, "error field must be a JSON object, got %T", data["error"])
		r.Equal(userErr.Name, errData["name"])
		r.Equal(userErr.Message, errData["message"])

		inn := data["_inngest"].(map[string]any)
		r.Equal("Failed", inn["status"], "run must finalize as Failed")
	})

	t.Run("no-retry failure on the first attempt is terminal", func(t *testing.T) {
		r := require.New(t)
		infra := newDeferTestInfra(t)
		countingQ := &enqueueCountingQueue{Queue: infra.rq}

		stepID := "step-defer"
		noRetryErr := &state.UserError{Name: "FatalError", Message: "do not retry me", NoRetry: true}
		driver := &mockDriverV1{
			t: t,
			response: &state.DriverResponse{
				StatusCode: 206,
				Generator: []*state.GeneratorOpcode{
					{
						Op: enums.OpcodeDeferAdd,
						ID: stepID,
						Opts: map[string]any{
							"fn_slug": "onDefer-score",
							"input":   map[string]any{},
						},
					},
					{
						Op:    enums.OpcodeRunError,
						ID:    "run-error",
						Error: noRetryErr,
					},
				},
			},
		}

		exec := infra.newExecutorWithQueue(t, countingQ, driver)

		var capturedEvents []event.Event
		exec.SetFinalizer(func(ctx context.Context, id statev2.ID, events []event.Event) error {
			capturedEvents = append(capturedEvents, events...)
			return nil
		})

		run := infra.scheduleRun(t, exec)
		countBeforeExecute := countingQ.enqueues

		_, err := exec.Execute(infra.ctx, state.Identifier{
			WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID,
		}, queue.Item{
			WorkspaceID: infra.wsID,
			Kind:        queue.KindStart,
			Attempt:     0,
			Identifier:  state.Identifier{WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID},
			Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: stepID}},
		}, inngest.Edge{Incoming: "$trigger", Outgoing: stepID})
		r.NoError(err)

		enqueuesDuringExecute := countingQ.enqueues - countBeforeExecute
		r.Equal(0, enqueuesDuringExecute,
			"NoRetry RunError piggybacked with DeferAdd must not enqueue discovery; got %d enqueues", enqueuesDuringExecute)

		r.Equal([]string{"onDefer-score"}, deferScheduleFnSlugs(t, capturedEvents),
			"AfterRun defer must still be scheduled when the run terminates via a NoRetry RunError")

		failedEvt := findEventByName(capturedEvents, event.FnFailedName)
		r.NotNil(failedEvt, "inngest/function.failed must be emitted on the first attempt when NoRetry is set")

		data := eventDataMap(t, *failedEvt)
		errData, ok := data["error"].(map[string]any)
		r.True(ok, "error field must be a JSON object, got %T", data["error"])
		r.Equal(noRetryErr.Name, errData["name"])
		r.Equal(noRetryErr.Message, errData["message"])

		inn := data["_inngest"].(map[string]any)
		r.Equal("Failed", inn["status"], "run must finalize as Failed despite having attempts remaining")
	})

	t.Run("retryable failure persists the defer but does not finalize", func(t *testing.T) {
		r := require.New(t)
		infra := newDeferTestInfra(t)

		stepID := "step-defer"
		driver := &mockDriverV1{
			t: t,
			response: &state.DriverResponse{
				StatusCode: 206,
				Generator: []*state.GeneratorOpcode{
					{
						Op: enums.OpcodeDeferAdd,
						ID: stepID,
						Opts: map[string]any{
							"fn_slug": "onDefer-score",
							"input":   map[string]any{},
						},
					},
					{
						Op:    enums.OpcodeRunError,
						ID:    "run-error",
						Error: userErr,
					},
				},
			},
		}

		exec := infra.newExecutor(t, driver)

		var capturedEvents []event.Event
		exec.SetFinalizer(func(ctx context.Context, id statev2.ID, events []event.Event) error {
			capturedEvents = append(capturedEvents, events...)
			return nil
		})

		run := infra.scheduleRun(t, exec)

		_, err := exec.Execute(infra.ctx, state.Identifier{
			WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID,
		}, queue.Item{
			WorkspaceID: infra.wsID,
			Kind:        queue.KindStart,
			Attempt:     0,
			Identifier:  state.Identifier{WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID},
			Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: stepID}},
		}, inngest.Edge{Incoming: "$trigger", Outgoing: stepID})

		r.Error(err, "a retryable RunError must surface an error so the queue requeues the item")
		r.Contains(err.Error(), userErr.Message)

		var retryable queue.RetryableError
		r.False(errors.As(err, &retryable),
			"a retryable RunError must not be wrapped as queue.NeverRetryError")

		defers, err := infra.smv2.LoadDefers(infra.ctx, run.ID)
		r.NoError(err)
		r.Contains(defers, stepID,
			"the piggybacked DeferAdd must persist even though the host RunError op is retryable")

		r.Nil(findEventByName(capturedEvents, event.FnFailedName),
			"a retryable RunError must not finalize the run or emit function.failed")
	})

	t.Run("parity with a plain (non-generator) rejection", func(t *testing.T) {
		r := require.New(t)

		runErrorEvt, runErrorOutput := runToFailureViaRunError(t, userErr)
		plainRejectionEvt, plainRejectionOutput := runToFailureViaPlainRejection(t, userErr)

		runErrorData := eventDataMap(t, *runErrorEvt)
		plainRejectionData := eventDataMap(t, *plainRejectionEvt)

		r.Equal(runErrorData["error"], plainRejectionData["error"],
			"OpcodeRunError and a plain rejection must carry the same error payload for the same user error")

		runErrorInn := runErrorData["_inngest"].(map[string]any)
		plainRejectionInn := plainRejectionData["_inngest"].(map[string]any)
		r.Equal(runErrorInn["status"], plainRejectionInn["status"])
		r.Equal("Failed", runErrorInn["status"])

		r.NotEmpty(runErrorOutput, "a terminal RunError must record a function output")
		r.Equal(plainRejectionOutput, runErrorOutput,
			"the recorded function output for a RunError run must be byte-identical to an equivalent plain rejection")
	})
}

// finalizeTracer records the finalize span so tests can inspect the function
// output the executor persists at terminal finalization.
type finalizeTracer struct {
	tracing.TracerProvider
	mu      sync.Mutex
	updates []*tracing.UpdateSpanOptions
}

func newFinalizeTracer() *finalizeTracer {
	return &finalizeTracer{TracerProvider: tracing.NewNoopTracerProvider()}
}

func (f *finalizeTracer) UpdateSpan(ctx context.Context, opts *tracing.UpdateSpanOptions) error {
	f.mu.Lock()
	f.updates = append(f.updates, opts)
	f.mu.Unlock()
	return f.TracerProvider.UpdateSpan(ctx, opts)
}

// functionOutput returns the function output recorded on the finalize span.
func (f *finalizeTracer) functionOutput(t *testing.T) string {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.updates {
		if u.Debug == nil || u.Debug.Location != "executor.finalize" || u.Attributes == nil {
			continue
		}
		if out, ok := u.Attributes.Get(meta.Attrs.StepOutput.Key()).(*string); ok && out != nil {
			return *out
		}
	}
	t.Fatal("no finalize span with a function output was recorded")
	return ""
}

func (i *deferTestInfra) newExecutorWithTracer(t *testing.T, driver *mockDriverV1, tp tracing.TracerProvider) execution.Executor {
	t.Helper()
	opts := []executor.ExecutorOpt{
		executor.WithStateManager(i.smv2),
		executor.WithPauseManager(i.pauseMgr),
		executor.WithQueue(i.rq),
		executor.WithLogger(logger.StdlibLogger(i.ctx)),
		executor.WithFunctionLoader(i.loader),
		executor.WithShardRegistry(i.shardRegistry),
		executor.WithTracerProvider(tp),
	}
	if driver != nil {
		opts = append(opts, executor.WithDriverV1(driver))
	}
	exec, err := executor.NewExecutor(opts...)
	require.NoError(t, err)
	return exec
}

func runToFailureViaRunError(t *testing.T, userErr *state.UserError) (*event.Event, string) {
	t.Helper()
	r := require.New(t)
	infra := newDeferTestInfra(t)

	stepID := "step-defer"
	driver := &mockDriverV1{
		t: t,
		response: &state.DriverResponse{
			StatusCode: 206,
			Generator: []*state.GeneratorOpcode{
				{
					Op:    enums.OpcodeRunError,
					ID:    "run-error",
					Error: userErr,
				},
			},
		},
	}
	tracer := newFinalizeTracer()
	exec := infra.newExecutorWithTracer(t, driver, tracer)

	var capturedEvents []event.Event
	exec.SetFinalizer(func(ctx context.Context, id statev2.ID, events []event.Event) error {
		capturedEvents = append(capturedEvents, events...)
		return nil
	})

	run := infra.scheduleRun(t, exec)

	_, err := exec.Execute(infra.ctx, state.Identifier{
		WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID,
	}, queue.Item{
		WorkspaceID: infra.wsID,
		Kind:        queue.KindStart,
		Attempt:     4,
		Identifier:  state.Identifier{WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID},
		Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: stepID}},
	}, inngest.Edge{Incoming: "$trigger", Outgoing: stepID})
	r.NoError(err)

	failedEvt := findEventByName(capturedEvents, event.FnFailedName)
	r.NotNil(failedEvt, "inngest/function.failed must be emitted for a terminal RunError")
	return failedEvt, tracer.functionOutput(t)
}

func runToFailureViaPlainRejection(t *testing.T, userErr *state.UserError) (*event.Event, string) {
	t.Helper()
	r := require.New(t)
	infra := newDeferTestInfra(t)

	stepID := "step-defer"
	crashMsg := "sdk crashed"
	output, err := json.Marshal(userErr)
	r.NoError(err)
	driver := &mockDriverV1{
		t: t,
		response: &state.DriverResponse{
			StatusCode: 500,
			Err:        &crashMsg,
			NoRetry:    true,
			UserError:  userErr,
			Output:     json.RawMessage(output),
		},
	}
	tracer := newFinalizeTracer()
	exec := infra.newExecutorWithTracer(t, driver, tracer)

	var capturedEvents []event.Event
	exec.SetFinalizer(func(ctx context.Context, id statev2.ID, events []event.Event) error {
		capturedEvents = append(capturedEvents, events...)
		return nil
	})

	run := infra.scheduleRun(t, exec)

	_, err = exec.Execute(infra.ctx, state.Identifier{
		WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID,
	}, queue.Item{
		WorkspaceID: infra.wsID,
		Kind:        queue.KindStart,
		Attempt:     0,
		Identifier:  state.Identifier{WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID},
		Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: stepID}},
	}, inngest.Edge{Incoming: "$trigger", Outgoing: stepID})
	r.Error(err)
	r.Contains(err.Error(), crashMsg)

	failedEvt := findEventByName(capturedEvents, event.FnFailedName)
	r.NotNil(failedEvt, "inngest/function.failed must be emitted for a terminal plain rejection")
	return failedEvt, tracer.functionOutput(t)
}
