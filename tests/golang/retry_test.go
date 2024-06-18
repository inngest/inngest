package golang

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetry(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	c := client.New(t)
	h, server, registerFuncs := NewSDKHandler(t, "retry-test")
	defer server.Close()

	// Create our function.
	var (
		counter     int32
		stepRetried int32
		fnRetried   int32
		runID       string
	)

	fn := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{Name: "test retry"},
		inngestgo.EventTrigger("test/executor-retry", nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}

			_, _ = step.Run(ctx, "Log input, increase counter", func(ctx context.Context) (string, error) {
				fmt.Println("step called")
				// If this is the first run, throw an error.
				res := atomic.AddInt32(&counter, 1)
				switch res {
				case 1:
					// First attempt
					fmt.Println("First retry, failing")
					return "1", inngestgo.RetryAtError(fmt.Errorf("step err"), time.Now())
				case 2:
					// Second attempt, first retry
					fmt.Println("Second retry, failing")
					atomic.AddInt32(&stepRetried, 1)
					return "2", inngestgo.RetryAtError(fmt.Errorf("second step err"), time.Now())
				case 3:
					fmt.Println("Final retry, completing")
					atomic.AddInt32(&stepRetried, 1)
				}
				return "retry", nil
			})

			res := atomic.AddInt32(&counter, 1)
			switch res {
			case 4:
				fmt.Println("Failing after step")
				// First attempt of fn, as step blocks this call
				return "fn error", inngestgo.RetryAtError(fmt.Errorf("fn err"), time.Now())
			case 5:
				// Second attempt of fn
				atomic.AddInt32(&fnRetried, 1)
			}
			fmt.Println("finishing")
			return "done", nil
		},
	)
	h.Register(fn)

	// Register the fns via the test SDK harness above.
	registerFuncs()

	_, err := inngestgo.Send(ctx, inngestgo.Event{
		Name: "test/executor-retry",
		Data: map[string]interface{}{
			"name": "retry",
		},
	})
	require.NoError(t, err)

	t.Run("expected values", func(t *testing.T) {
		require.Eventually(t, func() bool {
			return atomic.LoadInt32(&counter) == 5
		}, 10*time.Second, 2*time.Second)

		require.EqualValues(t, 2, stepRetried, "Step should have retried twice")
		require.EqualValues(t, 1, fnRetried, "Fn should have retried")
	})

	t.Run("trace run should have the appropriate data", func(t *testing.T) {
		<-time.After(3 * time.Second)

		require.Eventually(t, func() bool {
			run := c.RunTraces(ctx, runID)
			require.NotNil(t, run)
			require.NotNil(t, run.Trace)
			require.True(t, run.Trace.IsRoot)
			require.Equal(t, 2, len(run.Trace.ChildSpans))

			// output test
			require.NotNil(t, run.Trace.OutputID)
			runOutput := c.RunSpanOutput(ctx, *run.Trace.OutputID)
			require.NotNil(t, runOutput)
			require.NotNil(t, runOutput.Data)
			require.Contains(t, *runOutput.Data, "done")

			rootSpanID := run.Trace.SpanID

			t.Run("step retries", func(t *testing.T) {
				step := run.Trace.ChildSpans[0]
				assert.Equal(t, "Log input, increase counter", step.Name)
				assert.Equal(t, 3, step.Attempts)
				assert.Equal(t, rootSpanID, step.ParentSpanID)
				assert.Equal(t, 3, len(step.ChildSpans))
				assert.Equal(t, models.RunTraceSpanStatusCompleted.String(), step.Status)

				assert.NotNil(t, step.OutputID)
				output := c.RunSpanOutput(ctx, *step.OutputID)
				assert.NotNil(t, output)
				assert.NotNil(t, output.Data)
				assert.Contains(t, *output.Data, "retry")

				for i, span := range step.ChildSpans {
					testName := fmt.Sprintf("step retry %d", i)
					t.Run(testName, func(t *testing.T) {
						attempt := i + 1
						switch attempt {
						case 1:
							assert.Equal(t, fmt.Sprintf("Attempt %d", attempt), span.Name)
							assert.Equal(t, models.RunTraceSpanStatusFailed.String(), span.Status)
						case 2:
							assert.Equal(t, fmt.Sprintf("Attempt %d", attempt), span.Name)
							assert.Equal(t, models.RunTraceSpanStatusFailed.String(), span.Status)
						// last
						case 3:
							assert.Equal(t, fmt.Sprintf("Attempt %d", attempt), span.Name)
							assert.Equal(t, models.RunTraceSpanStatusCompleted.String(), span.Status)
						}
					})
				}
			})

			t.Run("function retries", func(t *testing.T) {
				exec := run.Trace.ChildSpans[1]
				assert.Equal(t, consts.OtelExecFnOk, exec.Name)
				assert.Equal(t, rootSpanID, exec.ParentSpanID)
				assert.Equal(t, 2, len(exec.ChildSpans))
				assert.Equal(t, models.RunTraceSpanStatusCompleted.String(), exec.Status)

				assert.NotNil(t, exec.OutputID)
				output := c.RunSpanOutput(ctx, *exec.OutputID)
				assert.NotNil(t, output)
				assert.NotNil(t, output.Data)
				assert.Contains(t, *output.Data, "done")

				for i, span := range exec.ChildSpans {
					testName := fmt.Sprintf("fn retry %d", i)
					t.Run(testName, func(t *testing.T) {
						attempt := i + 1
						switch attempt {
						case 1:
							assert.Equal(t, fmt.Sprintf("Attempt %d", attempt), span.Name)
							assert.Equal(t, models.RunTraceSpanStatusFailed.String(), span.Status)
						// last
						case 2:
							assert.Equal(t, fmt.Sprintf("Attempt %d", attempt), span.Name)
							assert.Equal(t, models.RunTraceSpanStatusCompleted.String(), span.Status)
						}
					})
				}
			})

			return true
		}, 15*time.Second, 3*time.Second)
	})

}
