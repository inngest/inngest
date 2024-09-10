package golang

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
	"github.com/stretchr/testify/require"
)

func TestWait(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)

	appID := "TestWait" + ulid.MustNew(ulid.Now(), nil).String()
	h, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	// This function will invoke the other function
	runID := ""
	evtName := "wait-event"
	waitEvtName := "resume"
	fn := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "main-fn",
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			runID = input.InputCtx.RunID

			_, _ = step.WaitForEvent[any](
				ctx,
				"wait",
				step.WaitForEventOpts{
					Name:    "dummy",
					Event:   waitEvtName,
					Timeout: 30 * time.Second,
				},
			)

			return "DONE", nil
		},
	)

	h.Register(fn)
	registerFuncs()

	// Trigger the main function
	_, err := inngestgo.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)

	t.Run("in progress wait", func(t *testing.T) {
		run := c.WaitForRunTracesWithTimeout(ctx, t, &runID, models.FunctionStatusRunning, 10*time.Second, 1*time.Second)

		require.NotNil(t, run.Trace)
		require.Equal(t, 1, len(run.Trace.ChildSpans))
		require.Equal(t, models.RunTraceSpanStatusRunning.String(), run.Trace.Status)
		require.Nil(t, run.Trace.OutputID)

		rootSpanID := run.Trace.SpanID

		t.Run("wait step", func(t *testing.T) {
			span := run.Trace.ChildSpans[0]
			assert.Equal(t, "dummy", span.Name)
			assert.Equal(t, 0, span.Attempts)
			assert.Equal(t, rootSpanID, span.ParentSpanID)
			assert.False(t, span.IsRoot)
			assert.Equal(t, 0, len(span.ChildSpans)) // NOTE: should have no child
			assert.Equal(t, models.RunTraceSpanStatusWaiting.String(), span.Status)
			assert.Equal(t, models.StepOpWaitForEvent.String(), span.StepOp)
			assert.Nil(t, span.EndedAt)
			assert.Nil(t, span.OutputID)

			var stepInfo models.WaitForEventStepInfo
			byt, err := json.Marshal(span.StepInfo)
			assert.NoError(t, err)
			assert.NoError(t, json.Unmarshal(byt, &stepInfo))

			assert.Equal(t, waitEvtName, stepInfo.EventName)
			assert.Nil(t, stepInfo.TimedOut)
			assert.Nil(t, stepInfo.FoundEventID)
		})
	})

	<-time.After(10 * time.Second)
	// Trigger the main function
	_, err = inngestgo.Send(ctx, &event.Event{Name: waitEvtName})
	r.NoError(err)

	t.Run("trace run should have appropriate data", func(t *testing.T) {
		run := c.WaitForRunTraces(ctx, t, &runID, models.FunctionStatusCompleted)

		require.Equal(t, models.FunctionStatusCompleted.String(), run.Status)
		require.NotNil(t, run.Trace)
		require.Equal(t, 1, len(run.Trace.ChildSpans))
		require.Equal(t, models.RunTraceSpanStatusCompleted.String(), run.Trace.Status)

		// output test
		require.NotNil(t, run.Trace.OutputID)
		output := c.RunSpanOutput(ctx, *run.Trace.OutputID)
		c.ExpectSpanOutput(t, "DONE", output)

		rootSpanID := run.Trace.SpanID

		t.Run("wait step", func(t *testing.T) {
			span := run.Trace.ChildSpans[0]
			assert.Equal(t, "dummy", span.Name)
			assert.Equal(t, 0, span.Attempts)
			assert.Equal(t, rootSpanID, span.ParentSpanID)
			assert.False(t, span.IsRoot)
			assert.Equal(t, models.RunTraceSpanStatusCompleted.String(), span.Status)
			assert.Equal(t, models.StepOpWaitForEvent.String(), span.StepOp)

			// output test
			assert.NotNil(t, span.OutputID)
			spanOutput := c.RunSpanOutput(ctx, *span.OutputID)
			c.ExpectSpanOutput(t, "resume", spanOutput)

			var stepInfo models.WaitForEventStepInfo
			byt, err := json.Marshal(span.StepInfo)
			assert.NoError(t, err)
			assert.NoError(t, json.Unmarshal(byt, &stepInfo))

			assert.Equal(t, waitEvtName, stepInfo.EventName)
			assert.NotNil(t, stepInfo.TimedOut)
			assert.False(t, *stepInfo.TimedOut)
			assert.NotNil(t, stepInfo.FoundEventID)
			assert.Nil(t, stepInfo.Expression)
		})
	})
}

func TestWaitGroup(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)

	appID := "TestWaitGroup" + ulid.MustNew(ulid.Now(), nil).String()
	h, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	var started int32

	// This function will invoke the other function
	runID := ""
	evtName := "wait-group"
	waitEvtName := "resume-group"

	fn := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "main-fn",
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			runID = input.InputCtx.RunID

			if atomic.LoadInt32(&started) == 0 {
				atomic.AddInt32(&started, 1)
				return nil, inngestgo.RetryAtError(fmt.Errorf("initial error"), time.Now().Add(5*time.Second))
			}

			_, _ = step.WaitForEvent[any](
				ctx,
				"wait",
				step.WaitForEventOpts{
					Name:    "dummy",
					Event:   waitEvtName,
					Timeout: 30 * time.Second,
				},
			)

			return "DONE", nil
		},
	)

	h.Register(fn)
	registerFuncs()

	// Trigger the main function
	_, err := inngestgo.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)

	t.Run("in progress wait", func(t *testing.T) {
		run := c.WaitForRunTraces(ctx, t, &runID, models.FunctionStatusRunning)

		require.NotNil(t, run.Trace)
		require.Equal(t, 1, len(run.Trace.ChildSpans))
		require.Equal(t, models.RunTraceSpanStatusRunning.String(), run.Trace.Status)
		require.Nil(t, run.Trace.OutputID)

		rootSpanID := run.Trace.SpanID

		span := run.Trace.ChildSpans[0]
		assert.Equal(t, consts.OtelExecPlaceholder, span.Name)
		assert.Equal(t, 0, span.Attempts)
		assert.Equal(t, rootSpanID, span.ParentSpanID)
		assert.False(t, span.IsRoot)
		assert.Equal(t, 2, len(span.ChildSpans)) // include queued retry span
		assert.Equal(t, models.RunTraceSpanStatusRunning.String(), span.Status)
		assert.Equal(t, "", span.StepOp)
		assert.Nil(t, span.OutputID)

		t.Run("failed", func(t *testing.T) {
			exec := span.ChildSpans[0]
			assert.Equal(t, "Attempt 0", exec.Name)
			assert.Equal(t, models.RunTraceSpanStatusFailed.String(), exec.Status)
			assert.NotNil(t, exec.OutputID)

			execOutput := c.RunSpanOutput(ctx, *exec.OutputID)
			assert.NotNil(t, execOutput)
			c.ExpectSpanErrorOutput(t, "", "initial error", execOutput)
		})

	})

	<-time.After(3 * time.Second)
	// Trigger the main function
	_, err = inngestgo.Send(ctx, &event.Event{Name: waitEvtName})
	r.NoError(err)

	t.Run("trace run should have appropriate data", func(t *testing.T) {
		run := c.WaitForRunTraces(ctx, t, &runID, models.FunctionStatusCompleted)

		require.Equal(t, models.FunctionStatusCompleted.String(), run.Status)
		require.NotNil(t, run.Trace)
		require.Equal(t, 1, len(run.Trace.ChildSpans))
		require.Equal(t, models.RunTraceSpanStatusCompleted.String(), run.Trace.Status)

		// output test
		require.NotNil(t, run.Trace.OutputID)
		output := c.RunSpanOutput(ctx, *run.Trace.OutputID)
		c.ExpectSpanOutput(t, "DONE", output)

		rootSpanID := run.Trace.SpanID

		t.Run("wait step", func(t *testing.T) {
			span := run.Trace.ChildSpans[0]
			assert.Equal(t, "dummy", span.Name)
			assert.Equal(t, 0, span.Attempts)
			assert.Equal(t, rootSpanID, span.ParentSpanID)
			assert.False(t, span.IsRoot)
			assert.Equal(t, 2, len(span.ChildSpans))
			assert.Equal(t, models.RunTraceSpanStatusCompleted.String(), span.Status)
			assert.Equal(t, models.StepOpWaitForEvent.String(), span.StepOp)

			// output test
			assert.NotNil(t, span.OutputID)
			spanOutput := c.RunSpanOutput(ctx, *span.OutputID)
			c.ExpectSpanOutput(t, waitEvtName, spanOutput)

			var stepInfo models.WaitForEventStepInfo
			byt, err := json.Marshal(span.StepInfo)
			assert.NoError(t, err)
			assert.NoError(t, json.Unmarshal(byt, &stepInfo))

			assert.Equal(t, waitEvtName, stepInfo.EventName)
			assert.NotNil(t, stepInfo.TimedOut)
			assert.False(t, *stepInfo.TimedOut)
			assert.NotNil(t, stepInfo.FoundEventID)
			assert.Nil(t, stepInfo.Expression)
		})
	})
}

func TestWaitInvalidExpression(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)

	appID := "TestWaitInvalidExpression" + ulid.MustNew(ulid.Now(), nil).String()
	h, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	// This function will invoke the other function
	runID := ""
	evtName := "my-event"
	fn := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "main-fn",
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			runID = input.InputCtx.RunID

			_, _ = step.WaitForEvent[any](
				ctx,
				"wait",
				step.WaitForEventOpts{
					If:      inngestgo.StrPtr("invalid"),
					Name:    "dummy",
					Timeout: time.Second,
				},
			)

			return nil, nil
		},
	)

	h.Register(fn)
	registerFuncs()

	// Trigger the main function and successfully invoke the other function
	_, err := inngestgo.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)
	c.WaitForRunStatus(ctx, t, "COMPLETED", &runID)
}

func TestWaitInvalidExpressionSyntaxError(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)

	appID := "TestWaitInvalidExpression" + ulid.MustNew(ulid.Now(), nil).String()
	h, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	// This function will invoke the other function
	runID := ""
	evtName := "my-event"
	fn := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "main-fn",
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			runID = input.InputCtx.RunID

			_, _ = step.WaitForEvent[any](
				ctx,
				"wait",
				step.WaitForEventOpts{
					If:      inngestgo.StrPtr("event.data.userId === async.data.userId"),
					Name:    "test/continue",
					Timeout: time.Second,
				},
			)

			return nil, nil
		},
	)

	h.Register(fn)
	registerFuncs()

	// Trigger the main function and successfully invoke the other function
	_, err := inngestgo.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)
	run := c.WaitForRunStatus(ctx, t, "FAILED", &runID)
	assert.Equal(t, "{\"error\":{\"error\":\"CompileError: Could not compile expression\",\"name\":\"CompileError\",\"message\":\"Could not compile expression\",\"stack\":\"ERROR: \\u003cinput\\u003e:1:21: Syntax error: token recognition error at: '= '\\n | event.data.userId === async.data.userId\\n | ....................^\"}}", run.Output)
}
