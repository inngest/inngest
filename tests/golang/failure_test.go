package golang

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFunctionFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	c := client.New(t)
	h, server, registerFuncs := NewSDKHandler(t, "fnfail")
	defer server.Close()

	// Create our function.
	var (
		counter int32
		runID   string
	)
	fn := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name:    "test sdk",
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
	h.Register(fn)

	// Register the fns via the test SDK harness above.
	registerFuncs()

	// Trigger the function.
	evt := inngestgo.Event{
		Name: "failure/run",
		Data: map[string]interface{}{
			"foo": "bar",
		},
	}
	_, err := inngestgo.Send(ctx, evt)
	require.NoError(t, err)

	<-time.After(5 * time.Second)

	require.EqualValues(t, counter, 1)

	t.Run("trace run should have appropriate data", func(t *testing.T) {
		<-time.After(3 * time.Second)

		require.Eventually(t, func() bool {
			// function run
			run := c.RunTraces(ctx, runID)
			require.NotNil(t, run)
			require.NotNil(t, run.Trace)
			require.True(t, run.Trace.IsRoot)
			require.Equal(t, 1, len(run.Trace.ChildSpans))
			require.Equal(t, models.RunTraceSpanStatusFailed.String(), run.Trace.Status)
			// output test
			// require.NotNil(t, run.Trace.OutputID)
			// runOutput := c.RunTraceSpanOutput(*run.Trace.OutputID)
			// require.NotNil(t, runOutput)
			// require.Nil(t, runOutput.Data)
			// require.NotNil(t, runOutput.StepError)
			// require.Contains(t, *runOutput.StepError.Stack, "nope!")

			rootSpanID := run.Trace.SpanID

			t.Run("failed run", func(t *testing.T) {
				span := run.Trace.ChildSpans[0]
				assert.Equal(t, "function error", span.Name)
				assert.False(t, span.IsRoot)
				assert.Equal(t, rootSpanID, span.ParentSpanID)
				assert.Equal(t, models.RunTraceSpanStatusFailed.String(), span.Status)
				// assert.NotNil(t, span.OutputID)

				// output test
				// output := c.RunTraceSpanOutput(*failed.OutputID)
				// assert.NotNil(t, output)
				// assert.Nil(t, output.Data)
				// assert.NotNil(t, output.StepError)
				// assert.Contains(t, *output.StepError.Stack, "nope!")
			})

			return true
		}, 10*time.Second, 2*time.Second)
	})
}

func TestFunctionFailureWithRetries(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	c := client.New(t)
	h, server, registerFuncs := NewSDKHandler(t, "fnfail-retry")
	defer server.Close()

	// Create our function.
	var (
		counter int32
		runID   string
	)
	fn := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name:    "test sdk fail with retry",
			Retries: inngestgo.IntPtr(1),
		},
		inngestgo.EventTrigger("failure/run-retry", nil),
		func(ctx context.Context, input inngestgo.Input[FnRunTestEvt]) (any, error) {
			runID = input.InputCtx.RunID

			atomic.AddInt32(&counter, 1)
			return true, fmt.Errorf("nope!")
		},
	)
	h.Register(fn)

	// Register the fns via the test SDK harness above.
	registerFuncs()

	// Trigger the function.
	evt := inngestgo.Event{
		Name: "failure/run-retry",
		Data: map[string]interface{}{
			"foo": "bar",
		},
	}
	_, err := inngestgo.Send(ctx, evt)
	require.NoError(t, err)

	<-time.After(5 * time.Second)

	require.EqualValues(t, counter, 1)

	t.Run("in progress run", func(t *testing.T) {
		<-time.After(3 * time.Second)

		require.Eventually(t, func() bool {
			// function run
			run := c.RunTraces(ctx, runID)
			require.NotNil(t, run)
			require.NotNil(t, run.Trace)
			require.True(t, run.Trace.IsRoot)
			require.Equal(t, 1, len(run.Trace.ChildSpans))
			require.Equal(t, models.RunTraceSpanStatusRunning.String(), run.Trace.Status)
			require.Nil(t, run.Trace.OutputID)

			rootSpanID := run.Trace.SpanID

			// test first attempt
			t.Run("attempt 1", func(t *testing.T) {
				span := run.Trace.ChildSpans[0]
				assert.Equal(t, "execute", span.Name)
				assert.False(t, span.IsRoot)
				assert.GreaterOrEqual(t, len(span.ChildSpans), 1)
				assert.Equal(t, rootSpanID, span.ParentSpanID)
				assert.Equal(t, models.RunTraceSpanStatusRunning.String(), span.Status)
				assert.Nil(t, span.OutputID)

				t.Run("failed", func(t *testing.T) {
					failed := span.ChildSpans[0]
					assert.Equal(t, "Attempt 1", failed.Name)
					assert.False(t, span.IsRoot)
					assert.Equal(t, models.RunTraceSpanStatusFailed.String(), failed.Status)

					// output test
				})
			})

			return true
		}, 10*time.Second, 2*time.Second)
	})

	t.Run("trace run should have appropriate data", func(t *testing.T) {
		<-time.After(40 * time.Second)

		require.Eventually(t, func() bool {
			// function run
			run := c.RunTraces(ctx, runID)
			require.NotNil(t, run)
			require.NotNil(t, run.Trace)
			require.True(t, run.Trace.IsRoot)
			require.Equal(t, 1, len(run.Trace.ChildSpans))
			require.Equal(t, models.RunTraceSpanStatusFailed.String(), run.Trace.Status)
			// output test
			// require.NotNil(t, run.Trace.OutputID)
			// runOutput := c.RunTraceSpanOutput(*run.Trace.OutputID)
			// require.NotNil(t, runOutput)
			// require.Nil(t, runOutput.Data)
			// require.NotNil(t, runOutput.StepError)
			// require.Contains(t, *runOutput.StepError.Stack, "nope!")

			rootSpanID := run.Trace.SpanID

			// first attempt
			t.Run("failed run", func(t *testing.T) {
				span := run.Trace.ChildSpans[0]
				assert.Equal(t, "execute", span.Name)
				assert.False(t, span.IsRoot)
				assert.Equal(t, rootSpanID, span.ParentSpanID)
				assert.Equal(t, 2, len(span.ChildSpans))
				assert.Equal(t, 2, span.Attempts)
				assert.Equal(t, models.RunTraceSpanStatusFailed.String(), span.Status)
				// assert.NotNil(t, span.OutputID)

				// output test
				// output := c.RunTraceSpanOutput(*failed.OutputID)
				// assert.NotNil(t, output)
				// assert.Nil(t, output.Data)
				// assert.NotNil(t, output.StepError)
				// assert.Contains(t, *output.StepError.Stack, "nope!")

				t.Run("attempt 1", func(t *testing.T) {
					one := span.ChildSpans[0]
					assert.Equal(t, "Attempt 1", one.Name)
					assert.False(t, one.IsRoot)
					assert.Equal(t, rootSpanID, one.ParentSpanID)
					assert.Equal(t, 1, one.Attempts)
					assert.Equal(t, models.RunTraceSpanStatusFailed.String(), one.Status)
					// assert.NotNil(t, span.OutputID)

					// output test
					// output := c.RunTraceSpanOutput(*failed.OutputID)
					// assert.NotNil(t, output)
					// assert.Nil(t, output.Data)
					// assert.NotNil(t, output.StepError)
					// assert.Contains(t, *output.StepError.Stack, "nope!")
				})

				// second attempt
				t.Run("attempt 2", func(t *testing.T) {
					two := span.ChildSpans[1]
					assert.Equal(t, "Attempt 2", two.Name)
					assert.False(t, two.IsRoot)
					assert.Equal(t, rootSpanID, two.ParentSpanID)
					assert.Equal(t, 2, two.Attempts)
					assert.Equal(t, models.RunTraceSpanStatusFailed.String(), two.Status)
					// assert.NotNil(t, span.OutputID)

					// output test
					// output := c.RunTraceSpanOutput(*failed.OutputID)
					// assert.NotNil(t, output)
					// assert.Nil(t, output.Data)
					// assert.NotNil(t, output.StepError)
					// assert.Contains(t, *output.StepError.Stack, "nope!")
				})
			})

			return true
		}, 10*time.Second, 2*time.Second)
	})
}
