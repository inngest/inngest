package golang

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"

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

			return nil, nil
		},
	)

	h.Register(fn)
	registerFuncs()

	// Trigger the main function
	_, err := inngestgo.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)

	t.Run("in progress wait", func(t *testing.T) {
		<-time.After(3 * time.Second)

		require.Eventually(t, func() bool {
			run := c.RunTraces(ctx, runID)
			require.NotNil(t, models.FunctionStatusRunning.String(), run.Status)
			require.NotNil(t, run.Trace)
			require.Equal(t, 1, len(run.Trace.ChildSpans))
			require.Equal(t, models.RunTraceSpanStatusRunning.String(), run.Trace.Status)

			rootSpanID := run.Trace.SpanID

			t.Run("wait step", func(t *testing.T) {
				span := run.Trace.ChildSpans[0]
				assert.Equal(t, "dummy", span.Name)
				assert.Equal(t, 0, span.Attempts)
				assert.Equal(t, rootSpanID, span.ParentSpanID)
				assert.False(t, span.IsRoot)
				assert.Equal(t, models.RunTraceSpanStatusWaiting.String(), span.Status)
				assert.Equal(t, models.StepOpWaitForEvent.String(), span.StepOp)

				var stepInfo models.WaitForEventStepInfo
				byt, err := json.Marshal(span.StepInfo)
				assert.NoError(t, err)
				assert.NoError(t, json.Unmarshal(byt, &stepInfo))

				assert.Equal(t, waitEvtName, stepInfo.EventName)
				assert.Nil(t, stepInfo.TimedOut)
				assert.Nil(t, stepInfo.FoundEventID)
			})

			return true
		}, 4*time.Second, 1*time.Second)
	})

	<-time.After(3 * time.Second)
	// Trigger the main function
	_, err = inngestgo.Send(ctx, &event.Event{Name: waitEvtName})
	r.NoError(err)

	t.Run("trace run should have appropriate data", func(t *testing.T) {
		<-time.After(3 * time.Second)

		require.Eventually(t, func() bool {
			run := c.RunTraces(ctx, runID)
			require.NotNil(t, run)
			require.Equal(t, models.FunctionStatusCompleted.String(), run.Status)
			require.NotNil(t, run.Trace)
			require.Equal(t, 1, len(run.Trace.ChildSpans))
			require.Equal(t, models.RunTraceSpanStatusCompleted.String(), run.Trace.Status)

			rootSpanID := run.Trace.SpanID
			// TODO: output test

			t.Run("wait step", func(t *testing.T) {
				span := run.Trace.ChildSpans[0]
				assert.Equal(t, "dummy", span.Name)
				assert.Equal(t, 0, span.Attempts)
				assert.Equal(t, rootSpanID, span.ParentSpanID)
				assert.False(t, span.IsRoot)
				assert.Equal(t, models.RunTraceSpanStatusCompleted.String(), span.Status)
				assert.Equal(t, models.StepOpWaitForEvent.String(), span.StepOp)

				// TODO: output test

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

			return true
		}, 10*time.Second, 2*time.Second)
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
