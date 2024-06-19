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
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSleep(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	c := client.New(t)
	h, server, registerFuncs := NewSDKHandler(t, "sleep-test")
	defer server.Close()

	// Create our function.
	var (
		started   int32
		startedAt time.Time
		completed int32
		runID     string
	)
	evtName := "test/sleep"

	a := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{Name: "test sleep"},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}

			if atomic.LoadInt32(&started) == 0 {
				// Throw an immediate error
				atomic.AddInt32(&started, 1)
				return nil, inngestgo.RetryAtError(fmt.Errorf("throwing an initial error"), time.Now())
			}

			if atomic.LoadInt32(&started) == 1 {
				// Set the started time
				atomic.AddInt32(&started, 1)
				startedAt = time.Now()
			}

			step.Sleep(ctx, "nap", 10*time.Second)

			// Ensure any time we're here it's 15 seconds after the sleep.
			require.GreaterOrEqual(t, int(time.Since(startedAt).Seconds()), 10)

			_, _ = step.Run(ctx, "test", func(ctx context.Context) (any, error) {
				if input.InputCtx.Attempt == 0 {
					return nil, inngestgo.RetryAtError(fmt.Errorf("throwing a step error"), time.Now())
				}
				return "done", nil
			})

			if input.InputCtx.Attempt == 0 {
				return nil, inngestgo.RetryAtError(fmt.Errorf("throwing a fn error"), time.Now())
			}

			atomic.AddInt32(&completed, 1)
			return true, nil
		},
	)
	h.Register(a)

	// Register the fns via the test SDK harness above.
	registerFuncs()

	_, err := inngestgo.Send(ctx, inngestgo.Event{
		Name: evtName,
		Data: map[string]any{"sleep": "ok"},
	})
	require.NoError(t, err)

	t.Run("in progress sleep", func(t *testing.T) {
		<-time.After(3 * time.Second)

		require.Eventually(t, func() bool {
			run := c.RunTraces(ctx, runID)
			require.NotNil(t, run)
			require.NotNil(t, run.Trace)
			require.True(t, run.Trace.IsRoot)
			require.Equal(t, 1, len(run.Trace.ChildSpans))
			require.Equal(t, models.RunTraceSpanStatusRunning.String(), run.Trace.Status)
			require.Nil(t, run.Trace.OutputID)

			t.Run("sleep", func(t *testing.T) {
				sleep := run.Trace.ChildSpans[0]
				assert.Equal(t, models.RunTraceSpanStatusRunning.String(), sleep.Status)
				assert.Equal(t, "nap", sleep.Name)
				assert.Equal(t, models.StepOpSleep.String(), sleep.StepOp)

				// verify step info
				info := &models.SleepStepInfo{}
				byt, err := json.Marshal(sleep.StepInfo)
				assert.NoError(t, err)
				assert.NoError(t, json.Unmarshal(byt, info))

				assert.True(t, time.Now().Before(info.SleepUntil))
			})

			return true
		}, 10*time.Second, 2*time.Second)
	})

	t.Run("expected values", func(t *testing.T) {
		require.Eventually(t, func() bool {
			return atomic.LoadInt32(&completed) == 1
		}, time.Minute, 2*time.Second)
	})

	t.Run("complete", func(t *testing.T) {
		<-time.After(15 * time.Second)

		require.Eventually(t, func() bool {
			require.EqualValues(t, 1, atomic.LoadInt32(&completed))
			run := c.RunTraces(ctx, runID)
			require.NotNil(t, run)
			require.NotNil(t, run.Trace)
			require.True(t, run.Trace.IsRoot)
			require.Equal(t, 3, len(run.Trace.ChildSpans))
			require.NotEqual(t, models.RunTraceSpanStatusRunning.String(), run.Trace.Status)

			// output test
			require.NotNil(t, run.Trace.OutputID)
			output := c.RunSpanOutput(ctx, *run.Trace.OutputID)
			require.NotNil(t, output)
			c.ExpectSpanOutput(t, "true", output)

			t.Run("sleep", func(t *testing.T) {
				sleep := run.Trace.ChildSpans[0]
				assert.Equal(t, models.RunTraceSpanStatusCompleted.String(), sleep.Status)
				assert.Equal(t, 2, len(sleep.ChildSpans))
				assert.Equal(t, "nap", sleep.Name)
				assert.Equal(t, models.StepOpSleep.String(), sleep.StepOp)
				assert.Nil(t, sleep.OutputID)

				// verify step info
				info := &models.SleepStepInfo{}
				byt, err := json.Marshal(sleep.StepInfo)
				assert.NoError(t, err)
				assert.NoError(t, json.Unmarshal(byt, info))

				assert.True(t, time.Now().After(info.SleepUntil))

				// first is the failed attempt
				t.Run("failed execution", func(t *testing.T) {
					exec := sleep.ChildSpans[0]
					assert.Equal(t, models.RunTraceSpanStatusFailed.String(), exec.Status)
					assert.Equal(t, consts.OtelExecPlaceholder, exec.Name)
					assert.NotNil(t, exec.OutputID)

					execOutput := c.RunSpanOutput(ctx, *exec.OutputID)
					assert.NotNil(t, execOutput)
					c.ExpectSpanErrorOutput(t, "", "throwing an initial error", execOutput)
				})
			})

			return true
		}, 9*time.Second, 3*time.Second)
	})
}
