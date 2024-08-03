package golang

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/consts"
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
			return "invoked!", nil
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

			return "success", nil
		},
	)

	h.Register(invokedFn, mainFn)
	registerFuncs()

	// Trigger the main function and successfully invoke the other function
	_, err := inngestgo.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)

	t.Run("trace run should have appropriate data", func(t *testing.T) {
		<-time.After(3 * time.Second)

		require.EventuallyWithT(t, func(ct *assert.CollectT) {
			r := require.New(ct)

			run := c.RunTraces(ctx, runID)
			r.NotNil(run)
			r.Equal(models.FunctionStatusCompleted.String(), run.Status)
			r.NotNil(run.Trace)
			r.Equal(1, len(run.Trace.ChildSpans))
			r.True(run.Trace.IsRoot)
			r.Equal(models.RunTraceSpanStatusCompleted.String(), run.Trace.Status)

			// output test
			r.NotNil(run.Trace.OutputID)
			output := c.RunSpanOutput(ctx, *run.Trace.OutputID)
			c.ExpectSpanOutput(ct, "success", output)

			rootSpanID := run.Trace.SpanID

			t.Run("invoke", func(t *testing.T) {
				as := assert.New(ct)

				invoke := run.Trace.ChildSpans[0]
				as.Equal("invoke", invoke.Name)
				as.Equal(0, invoke.Attempts)
				as.Equal(0, len(invoke.ChildSpans))
				as.False(invoke.IsRoot)
				as.Equal(rootSpanID, invoke.ParentSpanID)
				as.Equal(models.StepOpInvoke.String(), invoke.StepOp)

				// output test
				as.NotNil(invoke.OutputID)
				invokeOutput := c.RunSpanOutput(ctx, *invoke.OutputID)
				c.ExpectSpanOutput(ct, "invoked!", invokeOutput)

				var stepInfo models.InvokeStepInfo
				byt, err := json.Marshal(invoke.StepInfo)
				as.NoError(err)
				as.NoError(json.Unmarshal(byt, &stepInfo))

				as.False(*stepInfo.TimedOut)
				as.NotNil(stepInfo.ReturnEventID)
				as.NotNil(stepInfo.RunID)
			})
		}, 10*time.Second, 2*time.Second)
	})
}

func TestInvokeGroup(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)

	appID := "InvokeGroup-" + ulid.MustNew(ulid.Now(), nil).String()
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
			return "invoked!", nil
		},
	)
	var (
		started int32
		runID   string
	)

	// This function will invoke the other function
	evtName := "invoke-group-me"
	mainFn := inngestgo.CreateFunction(
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

			_, _ = step.Invoke[any](
				ctx,
				"invoke",
				step.InvokeOpts{FunctionId: appID + "-" + invokedFnName},
			)

			return "success", nil
		},
	)

	h.Register(invokedFn, mainFn)
	registerFuncs()

	// Trigger the main function and successfully invoke the other function
	_, err := inngestgo.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)

	t.Run("in progress", func(t *testing.T) {
		<-time.After(3 * time.Second)

		require.EventuallyWithT(t, func(ct *assert.CollectT) {
			r := require.New(ct)

			run := c.RunTraces(ctx, runID)
			r.Nil(run.EndedAt)
			r.Nil(run.Trace.EndedAt)
			r.NotNil(models.FunctionStatusRunning.String(), run.Status)
			r.NotNil(run.Trace)
			r.Equal(1, len(run.Trace.ChildSpans))
			r.Equal(models.RunTraceSpanStatusRunning.String(), run.Trace.Status)
			r.Nil(run.Trace.OutputID)

			rootSpanID := run.Trace.SpanID

			as := assert.New(ct)

			span := run.Trace.ChildSpans[0]
			as.Equal(consts.OtelExecPlaceholder, span.Name)
			as.Equal(0, span.Attempts)
			as.Equal(rootSpanID, span.ParentSpanID)
			as.False(span.IsRoot)
			as.Equal(2, len(span.ChildSpans)) // include queued retry span
			as.Equal(models.RunTraceSpanStatusRunning.String(), span.Status)
			as.Equal("", span.StepOp)
			as.Nil(span.OutputID)

			t.Run("failed", func(t *testing.T) {
				exec := span.ChildSpans[0]
				as.Equal("Attempt 0", exec.Name)
				as.Equal(models.RunTraceSpanStatusFailed.String(), exec.Status)
				as.NotNil(exec.OutputID)

				execOutput := c.RunSpanOutput(ctx, *exec.OutputID)
				as.NotNil(t, execOutput)
				c.ExpectSpanErrorOutput(ct, "", "initial error", execOutput)
			})
		}, 10*time.Second, 2*time.Second)
	})

	t.Run("trace run should have appropriate data", func(t *testing.T) {
		<-time.After(3 * time.Second)

		require.EventuallyWithT(t, func(ct *assert.CollectT) {
			r := require.New(ct)
			as := assert.New(ct)

			run := c.RunTraces(ctx, runID)
			r.NotNil(run)
			r.Equal(models.FunctionStatusCompleted.String(), run.Status)
			r.NotNil(run.Trace)
			r.Equal(1, len(run.Trace.ChildSpans))
			r.True(run.Trace.IsRoot)
			r.Equal(models.RunTraceSpanStatusCompleted.String(), run.Trace.Status)

			// output test
			r.NotNil(run.Trace.OutputID)
			output := c.RunSpanOutput(ctx, *run.Trace.OutputID)
			c.ExpectSpanOutput(ct, "success", output)

			rootSpanID := run.Trace.SpanID

			t.Run("invoke", func(t *testing.T) {
				invoke := run.Trace.ChildSpans[0]
				as.Equal("invoke", invoke.Name)
				as.Equal(0, invoke.Attempts)
				as.False(invoke.IsRoot)
				as.Equal(rootSpanID, invoke.ParentSpanID)
				as.Equal(2, len(invoke.ChildSpans))
				as.Equal(models.StepOpInvoke.String(), invoke.StepOp)
				as.NotNil(invoke.EndedAt)

				// output test
				as.NotNil(invoke.OutputID)
				invokeOutput := c.RunSpanOutput(ctx, *invoke.OutputID)
				c.ExpectSpanOutput(ct, "invoked!", invokeOutput)

				var stepInfo models.InvokeStepInfo
				byt, err := json.Marshal(invoke.StepInfo)
				as.NoError(err)
				as.NoError(json.Unmarshal(byt, &stepInfo))

				as.False(*stepInfo.TimedOut)
				as.NotNil(stepInfo.ReturnEventID)
				as.NotNil(stepInfo.RunID)
			})
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

	t.Run("trace run should have appropriate data", func(t *testing.T) {
		<-time.After(3 * time.Second)
		errMsg := "Timed out waiting for invoked function to complete"

		require.Eventually(t, func() bool {
			run := c.RunTraces(ctx, runID)
			require.NotNil(t, run)
			require.Equal(t, models.FunctionStatusFailed.String(), run.Status)
			require.NotNil(t, run.Trace)
			require.True(t, run.Trace.IsRoot)
			require.Equal(t, models.RunTraceSpanStatusFailed.String(), run.Trace.Status)

			// output test
			require.NotNil(t, run.Trace.OutputID)
			output := c.RunSpanOutput(ctx, *run.Trace.OutputID)
			require.NotNil(t, output)
			// c.ExpectSpanErrorOutput(t, errMsg, "", output)

			rootSpanID := run.Trace.SpanID

			t.Run("invoke", func(t *testing.T) {
				invoke := run.Trace.ChildSpans[0]
				assert.Equal(t, "invoke", invoke.Name)
				assert.Equal(t, 0, invoke.Attempts)
				assert.False(t, invoke.IsRoot)
				assert.Equal(t, rootSpanID, invoke.ParentSpanID)
				assert.Equal(t, models.StepOpInvoke.String(), invoke.StepOp)
				assert.NotNil(t, invoke.EndedAt)

				// output test
				assert.NotNil(t, invoke.OutputID)
				invokeOutput := c.RunSpanOutput(ctx, *invoke.OutputID)
				c.ExpectSpanErrorOutput(t, errMsg, "", invokeOutput)

				var stepInfo models.InvokeStepInfo
				byt, err := json.Marshal(invoke.StepInfo)
				assert.NoError(t, err)
				assert.NoError(t, json.Unmarshal(byt, &stepInfo))

				assert.True(t, *stepInfo.TimedOut)
				assert.Nil(t, stepInfo.ReturnEventID)
				assert.Nil(t, stepInfo.RunID)
			})

			return true
		}, 10*time.Second, 2*time.Second)
	})
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
