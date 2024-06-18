package golang

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvoke(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)

	appID := "Invoke-" + ulid.MustNew(ulid.Now(), nil).String()
	h, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	invokedFnName := "invoked-fn"
	invokedFn := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name:    invokedFnName,
			Retries: inngestgo.IntPtr(0),
		},
		inngestgo.EventTrigger("none", nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			return "hello", nil
		},
	)

	// This function will invoke the other function
	runID := ""
	evtName := "invoke-me"
	mainFn := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "main-fn",
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			runID = input.InputCtx.RunID

			_, _ = step.Invoke[any](
				ctx,
				"invoke",
				step.InvokeOpts{FunctionId: appID + "-" + invokedFnName},
			)

			return nil, nil
		},
	)

	h.Register(invokedFn, mainFn)
	registerFuncs()

	// Trigger the main function and successfully invoke the other function
	_, err := inngestgo.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)

	t.Run("trace run should have appropriate data", func(t *testing.T) {
		<-time.After(3 * time.Second)

		require.Eventually(t, func() bool {
			run := c.RunTraces(ctx, runID)
			require.NotNil(t, run)
			require.Equal(t, models.FunctionStatusCompleted.String(), run.Status)
			require.NotNil(t, run.Trace)
			require.Equal(t, 1, len(run.Trace.ChildSpans))
			require.True(t, run.Trace.IsRoot)
			require.Equal(t, models.RunTraceSpanStatusCompleted.String(), run.Trace.Status)

			rootSpanID := run.Trace.SpanID
			// TODO: output test

			invoke := run.Trace.ChildSpans[0]
			assert.Equal(t, "invoke", invoke.Name)
			assert.Equal(t, 0, invoke.Attempts)
			assert.False(t, invoke.IsRoot)
			assert.Equal(t, rootSpanID, invoke.ParentSpanID)
			assert.Equal(t, models.StepOpInvoke.String(), invoke.StepOp)

			// TODO: output test

			var stepInfo models.InvokeStepInfo
			byt, err := json.Marshal(invoke.StepInfo)
			assert.NoError(t, err)
			assert.NoError(t, json.Unmarshal(byt, &stepInfo))

			assert.False(t, *stepInfo.TimedOut)
			assert.NotNil(t, stepInfo.ReturnEventID)
			assert.NotNil(t, stepInfo.RunID)

			return true
		}, 10*time.Second, 2*time.Second)
	})
}

func TestInvokeTimeout(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)

	appID := "InvokeTimeout-" + ulid.MustNew(ulid.Now(), nil).String()
	h, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	invokedFnName := "invoked-fn"
	invokedFn := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name:    invokedFnName,
			Retries: inngestgo.IntPtr(0),
		},
		inngestgo.EventTrigger("none", nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			step.Sleep(ctx, "sleep", 5*time.Second)

			return nil, nil
		},
	)

	// This function will invoke the other function
	runID := ""
	evtName := "my-event"
	mainFn := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "main-fn",
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			runID = input.InputCtx.RunID

			_, _ = step.Invoke[any](
				ctx,
				"invoke",
				step.InvokeOpts{FunctionId: appID + "-" + invokedFnName, Timeout: 1 * time.Second},
			)

			return nil, nil
		},
	)

	h.Register(invokedFn, mainFn)
	registerFuncs()

	// Trigger the main function and successfully invoke the other function
	_, err := inngestgo.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)

	// The invoke target times out and should fail the main run
	c.WaitForRunStatus(ctx, t, "FAILED", &runID)
}

func TestInvokeRateLimit(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)

	appID := "InvokeRateLimit-" + ulid.MustNew(ulid.Now(), nil).String()
	h, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	// This function will be invoked by the main function
	invokedFnName := "invoked-fn"
	invokedFn := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: invokedFnName,
			RateLimit: &inngestgo.RateLimit{
				Limit:  1,
				Period: 1 * time.Minute,
			},
			Retries: inngestgo.IntPtr(0),
		},
		inngestgo.EventTrigger("none", nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			return nil, nil
		},
	)

	// This function will invoke the other function
	runID := ""
	evtName := "my-event"
	mainFn := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "main-fn",
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			runID = input.InputCtx.RunID

			_, _ = step.Invoke[any](
				ctx,
				"invoke",
				step.InvokeOpts{FunctionId: appID + "-" + invokedFnName})

			return nil, nil
		},
	)

	h.Register(invokedFn, mainFn)
	registerFuncs()

	// Trigger the main function and successfully invoke the other function
	_, err := inngestgo.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)
	c.WaitForRunStatus(ctx, t, "COMPLETED", &runID)

	// Trigger the main function. It'll fail because the invoked function is
	// rate limited
	runID = ""
	_, err = inngestgo.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)
	c.WaitForRunStatus(ctx, t, "FAILED", &runID)
}
