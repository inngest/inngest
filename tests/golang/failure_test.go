package golang

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
)

func TestFunctionFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	c := client.New(t)
	inngestClient, server, registerFuncs := NewSDKHandler(t, "fnfail")
	defer server.Close()

	// Create our function.
	var (
		counter int32
		runID   string
	)
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:      "test-sdk",
			Retries: inngestgo.IntPtr(0),
		},
		inngestgo.EventTrigger("failure/run", nil),
		func(ctx context.Context, input inngestgo.Input[FnRunTestEvt]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}
			atomic.AddInt32(&counter, 1)
			return true, fmt.Errorf("nope!")
		},
	)
	require.NoError(t, err)

	// Register the fns via the test SDK harness above.
	registerFuncs()

	// Trigger the function.
	evt := inngestgo.Event{
		Name: "failure/run",
		Data: map[string]interface{}{
			"foo": "bar",
		},
	}
	_, err = inngestClient.Send(ctx, evt)
	require.NoError(t, err)

	<-time.After(5 * time.Second)

	require.EqualValues(t, counter, 1)

	t.Run("trace run should have appropriate data", func(t *testing.T) {
		run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{Status: models.FunctionStatusFailed, ChildSpanCount: 1})

		require.Equal(t, models.RunTraceSpanStatusFailed.String(), run.Trace.Status)
		// output test
		require.NotNil(t, run.Trace.OutputID)
		runOutput := c.RunSpanOutput(ctx, *run.Trace.OutputID)
		require.NotNil(t, runOutput)
		c.ExpectSpanErrorOutput(t, "nope!", "", runOutput)

		rootSpanID := run.Trace.SpanID

		t.Run("failed run", func(t *testing.T) {
			// The failed execution wrapper now surfaces as a "Finalization"
			// span rather than consts.OtelExecFnErr, and no longer carries
			// its own output (the run root's output, checked above, already
			// covers the final error).
			span, _ := run.Trace.FindStep(t, "Finalization")
			require.False(t, span.IsRoot)
			require.Equal(t, rootSpanID, span.ParentSpanID)
			require.Equal(t, models.RunTraceSpanStatusFailed.String(), span.Status)
			require.Nil(t, span.OutputID)

			t.Run("attempt 0", func(t *testing.T) {
				one, _ := span.FindStep(t, "Attempt 0")
				require.Equal(t, models.RunTraceSpanStatusFailed.String(), one.Status)
				require.NotNil(t, one.OutputID)

				// output test
				output := c.RunSpanOutput(ctx, *one.OutputID)
				require.NotNil(t, output)
				c.ExpectSpanErrorOutput(t, "nope!", "", output)
			})
		})
	})
}

func TestFunctionFailureWithRetries(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	c := client.New(t)
	inngestClient, server, registerFuncs := NewSDKHandler(t, "fnfail-retry")
	defer server.Close()

	// Create our function.
	var (
		counter int32
		runID   string
	)
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:      "test-sdk-fail-with-retry",
			Retries: inngestgo.IntPtr(1),
		},
		inngestgo.EventTrigger("failure/run-retry", nil),
		func(ctx context.Context, input inngestgo.Input[FnRunTestEvt]) (any, error) {
			runID = input.InputCtx.RunID

			atomic.AddInt32(&counter, 1)
			return true, fmt.Errorf("nope!")
		},
	)
	require.NoError(t, err)

	// Register the fns via the test SDK harness above.
	registerFuncs()

	// Trigger the function.
	evt := inngestgo.Event{
		Name: "failure/run-retry",
		Data: map[string]interface{}{
			"foo": "bar",
		},
	}
	_, err = inngestClient.Send(ctx, evt)
	require.NoError(t, err)

	<-time.After(5 * time.Second)

	require.EqualValues(t, counter, 1)

	t.Run("in progress run", func(t *testing.T) {
		run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{Status: models.FunctionStatusRunning, ChildSpanCount: 1})

		require.Equal(t, models.RunTraceSpanStatusRunning.String(), run.Trace.Status)
		require.Nil(t, run.Trace.OutputID)

		rootSpanID := run.Trace.SpanID

		// test first attempt
		t.Run("attempt 1", func(t *testing.T) {
			// The in-progress execution wrapper now surfaces as a
			// "Finalization" span rather than the old discovery-span
			// placeholder name ("execute" / consts.OtelExecPlaceholder).
			span, _ := run.Trace.FindStep(t, "Finalization")
			require.False(t, span.IsRoot)
			require.GreaterOrEqual(t, len(span.ChildSpans), 1)
			require.Equal(t, rootSpanID, span.ParentSpanID)
			require.Equal(t, models.RunTraceSpanStatusFailed.String(), span.Status)
			require.Nil(t, span.OutputID)

			t.Run("failed", func(t *testing.T) {
				failed, _ := span.FindStep(t, "Attempt 0")
				require.False(t, span.IsRoot)
				require.Equal(t, models.RunTraceSpanStatusFailed.String(), failed.Status)

				// output test
				require.NotNil(t, failed.OutputID)
				output := c.RunSpanOutput(ctx, *failed.OutputID)
				require.NotNil(t, output)
				c.ExpectSpanErrorOutput(t, "nope!", "", output)
			})
		})
	})

	t.Run("trace run should have appropriate data", func(t *testing.T) {
		run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{Status: models.FunctionStatusFailed, Timeout: 1 * time.Minute, Interval: 5 * time.Second, ChildSpanCount: 1})

		require.Equal(t, models.RunTraceSpanStatusFailed.String(), run.Trace.Status)
		// output test
		require.NotNil(t, run.Trace.OutputID)
		runOutput := c.RunSpanOutput(ctx, *run.Trace.OutputID)
		c.ExpectSpanErrorOutput(t, "nope!", "", runOutput)

		rootSpanID := run.Trace.SpanID

		// first attempt
		t.Run("failed run", func(t *testing.T) {
			// The retrying function's execution wrapper now surfaces as a
			// "Finalization" span (see convertRunSpanToGQL's finalization-group
			// collapsing) rather than the old discovery-span placeholder name
			// (consts.OtelExecPlaceholder). It groups the attempts but no
			// longer carries its own output — that now lives on each attempt
			// (and the sibling executor.nonstep spans), and the run root
			// already covers the final error (checked above).
			span, _ := run.Trace.FindStep(t, "Finalization")
			require.False(t, span.IsRoot)
			require.Equal(t, rootSpanID, span.ParentSpanID)
			require.Equal(t, 2, len(span.ChildSpans))
			require.Equal(t, 1, span.Attempts) // max attempt index across children (0 and 1)
			require.Equal(t, models.RunTraceSpanStatusFailed.String(), span.Status)
			require.Nil(t, span.OutputID)

			t.Run("attempt 0", func(t *testing.T) {
				one, _ := span.FindStep(t, "Attempt 0")
				require.False(t, one.IsRoot)
				// Nested under the Finalization wrapper, not the run root.
				require.Equal(t, span.SpanID, one.ParentSpanID)
				require.Equal(t, 0, one.Attempts)
				require.Equal(t, models.RunTraceSpanStatusFailed.String(), one.Status)
				require.NotNil(t, one.OutputID)

				// output test
				oneOutput := c.RunSpanOutput(ctx, *one.OutputID)
				c.ExpectSpanErrorOutput(t, "nope!", "", oneOutput)
			})

			// second attempt
			t.Run("attempt 1", func(t *testing.T) {
				two, _ := span.FindStep(t, "Attempt 1")
				require.False(t, two.IsRoot)
				require.Equal(t, span.SpanID, two.ParentSpanID)
				require.Equal(t, 1, two.Attempts)
				require.Equal(t, models.RunTraceSpanStatusFailed.String(), two.Status)
				require.NotNil(t, two.OutputID)

				// output test
				twoOutput := c.RunSpanOutput(ctx, *two.OutputID)
				c.ExpectSpanErrorOutput(t, "nope!", "", twoOutput)
			})
		})
	})
}

func TestFunctionResponseTooLargeFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	c := client.New(t)
	inngestClient, server, registerFuncs := NewSDKHandler(t,
		randomSuffix("fail-large-response_output"),
	)
	defer server.Close()

	var runID string
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:      "test-sdk-response-too-large",
			Retries: inngestgo.IntPtr(0),
		},
		inngestgo.EventTrigger("failure/run-response-too-large", nil),
		func(ctx context.Context, input inngestgo.Input[FnRunTestEvt]) (any, error) {
			runID = input.InputCtx.RunID
			return strings.Repeat("A", consts.MaxSDKResponseBodySize*10), nil
		},
	)
	require.NoError(t, err)

	registerFuncs()

	evt := inngestgo.Event{
		Name: "failure/run-response-too-large",
		Data: map[string]interface{}{
			"foo": "bar",
		},
	}
	_, err = inngestClient.Send(ctx, evt)
	require.NoError(t, err)

	t.Run("trace run should fail with output too large error", func(t *testing.T) {
		run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{
			Status:         models.FunctionStatusFailed,
			Timeout:        30 * time.Second,
			Interval:       2 * time.Second,
			ChildSpanCount: 1,
		})

		require.Equal(t, models.RunTraceSpanStatusFailed.String(), run.Trace.Status)
		require.NotNil(t, run.Trace.OutputID)

		rootSpanID := run.Trace.SpanID
		require.NotEmpty(t, rootSpanID)

		t.Run("attempt 1", func(t *testing.T) {
			// The execution wrapper now surfaces as a "Finalization" span
			// rather than consts.OtelExecFnErr ("function error"), and no
			// longer carries its own output.
			span, _ := run.Trace.FindStep(t, "Finalization")
			require.False(t, span.IsRoot)
			require.GreaterOrEqual(t, len(span.ChildSpans), 1)
			require.Equal(t, rootSpanID, span.ParentSpanID)
			require.Equal(t, models.RunTraceSpanStatusFailed.String(), span.Status)
			require.Nil(t, span.OutputID)

			t.Run("failed", func(t *testing.T) {
				failed, _ := span.FindStep(t, "Attempt 0")
				require.False(t, span.IsRoot)
				require.Equal(t, models.RunTraceSpanStatusFailed.String(), failed.Status)

				require.NotNil(t, failed.OutputID)
				output := c.RunSpanOutput(ctx, *failed.OutputID)
				require.NotNil(t, output)
				require.NotNil(t, output.Error)
				// The syscode now surfaces on the error's Name field rather
				// than embedded in Stack.
				require.NotNil(t, output.Error.Name)
				require.Equal(t, syscode.CodeOutputTooLarge, *output.Error.Name)
			})
		})

	})
}

func TestFunctionResponseTooLargeFailureWithRetry(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	c := client.New(t)
	inngestClient, server, registerFuncs := NewSDKHandler(t,
		randomSuffix("fail-large-response_output"),
	)
	defer server.Close()

	var runID string
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:      "test-sdk-response-too-large",
			Retries: inngestgo.IntPtr(1),
		},
		inngestgo.EventTrigger("failure/run-response-too-large-retry", nil),
		func(ctx context.Context, input inngestgo.Input[FnRunTestEvt]) (any, error) {
			runID = input.InputCtx.RunID
			return strings.Repeat("A", consts.MaxSDKResponseBodySize*10), nil
		},
	)
	require.NoError(t, err)

	registerFuncs()

	evt := inngestgo.Event{
		Name: "failure/run-response-too-large-retry",
		Data: map[string]interface{}{
			"foo": "bar",
		},
	}
	_, err = inngestClient.Send(ctx, evt)
	require.NoError(t, err)

	t.Run("in progress run with large response body", func(t *testing.T) {
		run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{Status: models.FunctionStatusRunning, ChildSpanCount: 1})

		require.Equal(t, models.RunTraceSpanStatusRunning.String(), run.Trace.Status)
		require.Nil(t, run.Trace.OutputID)

		rootSpanID := run.Trace.SpanID

		// test first attempt
		t.Run("attempt 1", func(t *testing.T) {
			// The in-progress execution wrapper now surfaces as a
			// "Finalization" span rather than the old discovery-span
			// placeholder name ("execute" / consts.OtelExecPlaceholder).
			span, _ := run.Trace.FindStep(t, "Finalization")
			require.False(t, span.IsRoot)
			require.GreaterOrEqual(t, len(span.ChildSpans), 1)
			require.Equal(t, rootSpanID, span.ParentSpanID)
			require.Equal(t, models.RunTraceSpanStatusFailed.String(), span.Status)
			require.Nil(t, span.OutputID)

			t.Run("failed with output too large error", func(t *testing.T) {
				failed, _ := span.FindStep(t, "Attempt 0")
				require.False(t, span.IsRoot)
				require.Equal(t, models.RunTraceSpanStatusFailed.String(), failed.Status)

				// output test
				require.NotNil(t, failed.OutputID)
				output := c.RunSpanOutput(ctx, *failed.OutputID)
				require.NotNil(t, output)
				require.NotNil(t, output.Error)
				// The syscode now surfaces on the error's Name field rather
				// than embedded in Stack.
				require.NotNil(t, output.Error.Name)
				require.Equal(t, syscode.CodeOutputTooLarge, *output.Error.Name)
			})
		})
	})
}
