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
	inngestClient, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	invokedFnName := "invoked-fn"
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:      invokedFnName,
			Retries: inngestgo.IntPtr(0),
		},
		inngestgo.EventTrigger("none", nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			return "invoked!", nil
		},
	)
	r.NoError(err)
	// This function will invoke the other function
	runID := ""
	evtName := "invoke-me"
	_, err = inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "main-fn",
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			runID = input.InputCtx.RunID

			_, err := step.Invoke[any](
				ctx,
				"invoke",
				step.InvokeOpts{FunctionId: appID + "-" + invokedFnName},
			)
			if err != nil {
				return nil, err
			}
			return "success", nil
		},
	)
	r.NoError(err)
	registerFuncs()

	// Trigger the main function and successfully invoke the other function
	_, err = inngestClient.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)

	// Wait a moment for runID to be populated
	<-time.After(2 * time.Second)

	t.Run("trace run should have appropriate data", func(t *testing.T) {
		r := require.New(t)
		run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{Status: models.FunctionStatusCompleted, ChildSpanCount: 1})

		r.NotNil(run.Trace)
		r.True(run.Trace.IsRoot)
		r.Equal(models.RunTraceSpanStatusCompleted.String(), run.Trace.Status)

		// output test
		r.NotNil(run.Trace.OutputID)
		output := c.RunSpanOutput(ctx, *run.Trace.OutputID)
		c.ExpectSpanOutput(t, "success", output)

		rootSpanID := run.Trace.SpanID

		t.Run("invoke", func(t *testing.T) {
			as := assert.New(t)
			invoke := run.Trace.ChildSpans[0]
			as.Equal("invoke", invoke.Name)
			as.Equal(0, invoke.Attempts)
			as.Equal(0, len(invoke.ChildSpans))
			as.False(invoke.IsRoot)
			as.Equal(rootSpanID, invoke.ParentSpanID)
			as.Equal(models.StepOpInvoke.String(), invoke.StepOp)
			as.Equal(models.RunTraceSpanStatusCompleted.String(), invoke.Status)

			// output test
			as.NotNil(invoke.OutputID)
			invokeOutput := c.RunSpanOutput(ctx, *invoke.OutputID)
			c.ExpectSpanOutput(t, "invoked!", invokeOutput)

			var stepInfo models.InvokeStepInfo
			byt, err := json.Marshal(invoke.StepInfo)
			as.NoError(err)
			as.NoError(json.Unmarshal(byt, &stepInfo))

			as.False(*stepInfo.TimedOut)
			as.NotNil(stepInfo.ReturnEventID)
			as.NotNil(stepInfo.RunID)
		})
	})
}

func TestInvokeGroup(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)

	appID := "InvokeGroup-" + ulid.MustNew(ulid.Now(), nil).String()
	inngestClient, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	invokedFnName := "invoked-fn"
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:      invokedFnName,
			Retries: inngestgo.IntPtr(0),
		},
		inngestgo.EventTrigger("none", nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			return "invoked!", nil
		},
	)
	r.NoError(err)
	var (
		started int32
		runID   string
	)

	// This function will invoke the other function
	evtName := "invoke-group-me"
	_, err = inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "main-fn",
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			runID = input.InputCtx.RunID

			if atomic.LoadInt32(&started) == 0 {
				atomic.AddInt32(&started, 1)
				return nil, inngestgo.RetryAtError(fmt.Errorf("initial error"), time.Now().Add(5*time.Second))
			}

			_, err := step.Invoke[any](
				ctx,
				"invoke",
				step.InvokeOpts{FunctionId: appID + "-" + invokedFnName},
			)
			if err != nil {
				return nil, err
			}
			return "success", nil
		},
	)
	r.NoError(err)
	registerFuncs()

	// Trigger the main function and successfully invoke the other function
	_, err = inngestClient.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)

	t.Run("in progress", func(t *testing.T) {
		run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{Status: models.FunctionStatusRunning, ChildSpanCount: 1})
		r := require.New(t)

		r.Nil(run.EndedAt)
		r.Nil(run.Trace.EndedAt)
		r.Equal(models.RunTraceSpanStatusRunning.String(), run.Trace.Status)
		r.Nil(run.Trace.OutputID)

		rootSpanID := run.Trace.SpanID

		as := assert.New(t)

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
			c.ExpectSpanErrorOutput(t, "initial error", "", execOutput)
		})
	})

	t.Run("trace run should have appropriate data", func(t *testing.T) {
		run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{Status: models.FunctionStatusCompleted, ChildSpanCount: 1})

		as := assert.New(t)

		r.True(run.Trace.IsRoot)
		r.Equal(models.RunTraceSpanStatusCompleted.String(), run.Trace.Status)

		// output test
		r.NotNil(run.Trace.OutputID)
		output := c.RunSpanOutput(ctx, *run.Trace.OutputID)
		c.ExpectSpanOutput(t, "success", output)

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
			as.Equal(models.RunTraceSpanStatusCompleted.String(), invoke.Status)

			// output test
			as.NotNil(invoke.OutputID)
			invokeOutput := c.RunSpanOutput(ctx, *invoke.OutputID)
			c.ExpectSpanOutput(t, "invoked!", invokeOutput)

			var stepInfo models.InvokeStepInfo
			byt, err := json.Marshal(invoke.StepInfo)
			as.NoError(err)
			as.NoError(json.Unmarshal(byt, &stepInfo))

			as.False(*stepInfo.TimedOut)
			as.NotNil(stepInfo.ReturnEventID)
			as.NotNil(stepInfo.RunID)
		})
	})
}

func TestInvokeTimeout(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)

	appID := "InvokeTimeout-" + ulid.MustNew(ulid.Now(), nil).String()
	inngestClient, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	invokedFnName := "invoked-fn"
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:      invokedFnName,
			Retries: inngestgo.IntPtr(0),
		},
		inngestgo.EventTrigger("none", nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			step.Sleep(ctx, "sleep", 5*time.Second)
			return nil, nil
		},
	)
	r.NoError(err)
	// This function will invoke the other function
	runID := ""
	evtName := "my-event"
	_, err = inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "main-fn",
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			runID = input.InputCtx.RunID

			_, err := step.Invoke[any](
				ctx,
				"invoke",
				step.InvokeOpts{FunctionId: appID + "-" + invokedFnName, Timeout: 1 * time.Second},
			)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	)
	r.NoError(err)
	registerFuncs()

	// Trigger the main function and successfully invoke the other function
	_, err = inngestClient.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)

	// The invoke target times out and should fail the main run
	c.WaitForRunStatus(ctx, t, "FAILED", &runID)

	t.Run("trace run should have appropriate data", func(t *testing.T) {
		errMsg := "Timed out waiting for invoked function to complete"
		run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{Status: models.FunctionStatusFailed})

		require.NotNil(t, run.Trace)
		require.True(t, run.Trace.IsRoot)
		require.Equal(t, models.RunTraceSpanStatusFailed.String(), run.Trace.Status)

		// output test
		require.NotNil(t, run.Trace.OutputID)
		output := c.RunSpanOutput(ctx, *run.Trace.OutputID)
		require.NotNil(t, output)

		rootSpanID := run.Trace.SpanID

		t.Run("invoke", func(t *testing.T) {
			r := require.New(t)
			invoke := run.Trace.ChildSpans[0]
			r.Equal("invoke", invoke.Name)
			r.Equal(0, invoke.Attempts)
			r.False(invoke.IsRoot)
			r.Equal(rootSpanID, invoke.ParentSpanID)
			r.Equal(models.StepOpInvoke.String(), invoke.StepOp)
			r.Equal(models.RunTraceSpanStatusFailed.String(), invoke.Status)
			r.NotNil(invoke.EndedAt)

			// output test
			assert.NotNil(t, invoke.OutputID)
			r.EventuallyWithT(func(t *assert.CollectT) {
				invokeOutput := c.RunSpanOutput(ctx, *invoke.OutputID)
				c.ExpectSpanErrorOutput(t, errMsg, "", invokeOutput)
			}, 10*time.Second, 1*time.Second)

			var stepInfo models.InvokeStepInfo
			byt, err := json.Marshal(invoke.StepInfo)
			r.NoError(err)
			r.NoError(json.Unmarshal(byt, &stepInfo))

			r.True(*stepInfo.TimedOut)
			r.Nil(stepInfo.ReturnEventID)
			r.Nil(stepInfo.RunID)
		})
	})
}

func TestInvokeRateLimit(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)

	appID := "InvokeRateLimit-" + ulid.MustNew(ulid.Now(), nil).String()
	inngestClient, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	// This function will be invoked by the main function
	invokedFnName := "invoked-fn"
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: invokedFnName,
			RateLimit: &inngestgo.ConfigRateLimit{
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
	r.NoError(err)
	// This function will invoke the other function
	runID := ""
	evtName := "my-event"
	_, err = inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "main-fn",
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			runID = input.InputCtx.RunID

			_, err := step.Invoke[any](
				ctx,
				"invoke",
				step.InvokeOpts{FunctionId: appID + "-" + invokedFnName})
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	)
	r.NoError(err)
	registerFuncs()

	// Trigger the main function and successfully invoke the other function
	_, err = inngestClient.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)

	// Wait a moment for runID to be populated
	<-time.After(2 * time.Second)

	c.WaitForRunStatus(ctx, t, "COMPLETED", &runID)

	// Trigger the main function. It'll fail because the invoked function is
	// rate limited
	runID = ""
	_, err = inngestClient.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)
	c.WaitForRunStatus(ctx, t, "FAILED", &runID)
}
