package golang

import (
	"context"
	"errors"
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
	inngestClient, server, registerFuncs := NewSDKHandler(t, "retry-test")
	defer server.Close()

	// Create our function.
	var (
		counter     int32
		stepRetried int32
		fnRetried   int32
		runID       string
	)

	evtName := "test/retry"

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "test-retry"},
		inngestgo.EventTrigger(evtName, nil),
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
	require.NoError(t, err)

	// Register the fns via the test SDK harness above.
	registerFuncs()

	_, err = inngestClient.Send(ctx, inngestgo.Event{
		Name: evtName,
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
		run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{
			Status:         models.FunctionStatusCompleted,
			ChildSpanCount: 2,
			Timeout:        15 * time.Second,
			Interval:       3 * time.Second,
		})

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
			assert.Equal(t, 2, step.Attempts)
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
					attempt := i
					switch attempt {
					case 0:
						assert.Equal(t, fmt.Sprintf("Attempt %d", attempt), span.Name)
						assert.Equal(t, models.RunTraceSpanStatusFailed.String(), span.Status)
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
					attempt := i
					switch attempt {
					case 0:
						assert.Equal(t, fmt.Sprintf("Attempt %d", attempt), span.Name)
						assert.Equal(t, models.RunTraceSpanStatusFailed.String(), span.Status)
					// last
					case 1:
						assert.Equal(t, fmt.Sprintf("Attempt %d", attempt), span.Name)
						assert.Equal(t, models.RunTraceSpanStatusCompleted.String(), span.Status)
					}
				})
			}
		})
	})
}

func TestMaxRetries(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	ctx := context.Background()

	c := client.New(t)
	ic, server, sync := NewSDKHandler(t, randomSuffix("app"))
	defer server.Close()

	var attempt int
	var runID string
	evtName := "event"
	_, err := inngestgo.CreateFunction(
		ic,
		inngestgo.FunctionOpts{
			ID:      "fn",
			Retries: inngestgo.IntPtr(consts.MaxRetries),
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			runID = input.InputCtx.RunID
			return step.Run(ctx, "a", func(ctx context.Context) (any, error) {
				attempt = input.InputCtx.Attempt
				return nil, inngestgo.RetryAtError(errors.New("oh no"), time.Now())
			})
		},
	)
	r.NoError(err)
	sync()

	_, err = ic.Send(ctx, inngestgo.Event{Name: evtName})
	r.NoError(err)

	c.WaitForRunStatus(ctx, t,
		models.FunctionStatusFailed.String(),
		&runID,
		client.WaitForRunStatusOpts{
			Timeout: 30 * time.Second,
		},
	)
	r.EqualValues(20, attempt)
}
