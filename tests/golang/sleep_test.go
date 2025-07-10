package golang

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/logger"
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
	inngestClient, server, registerFuncs := NewSDKHandler(t, "sleep-test")
	defer server.Close()

	// Create our function.
	var (
		started   int32
		startedAt time.Time
		completed int32
		runID     string
	)
	evtName := "test/sleep"

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "test-sleep"},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}

			if atomic.LoadInt32(&started) == 0 {
				// Throw an immediate error
				atomic.AddInt32(&started, 1)
				return nil, inngestgo.RetryAtError(fmt.Errorf("throwing an initial error"), time.Now().Add(5*time.Second))
			}

			if atomic.LoadInt32(&started) == 1 {
				// Set the started time
				atomic.AddInt32(&started, 1)
				startedAt = time.Now()
			}

			step.Sleep(ctx, "nap", 5*time.Second)

			// Ensure any time we're here it's 5 seconds after the sleep.
			require.GreaterOrEqual(t, int(time.Since(startedAt).Seconds()), 5)

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
	require.NoError(t, err)

	// Register the fns via the test SDK harness above.
	registerFuncs()

	_, err = inngestClient.Send(ctx, inngestgo.Event{
		Name: evtName,
		Data: map[string]any{"sleep": "ok"},
	})
	require.NoError(t, err)

	t.Run("expected values", func(t *testing.T) {
		require.Eventually(t, func() bool {
			return atomic.LoadInt32(&completed) == 1
		}, time.Minute, 2*time.Second)
	})

	t.Run("complete", func(t *testing.T) {
		run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{
			Status:         models.FunctionStatusCompleted,
			ChildSpanCount: 3,
			Timeout:        9 * time.Second,
			Interval:       3 * time.Second,
		})
		require.EqualValues(t, 1, atomic.LoadInt32(&completed))

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
				assert.Equal(t, "Attempt 0", exec.Name)
				assert.NotNil(t, exec.OutputID)

				execOutput := c.RunSpanOutput(ctx, *exec.OutputID)
				assert.NotNil(t, execOutput)
				c.ExpectSpanErrorOutput(t, "throwing an initial error", "", execOutput)
			})
		})
	})
}

func TestSleepFudging(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	//c := client.New(t)
	inngestClient, server, registerFuncs := NewSDKHandler(t, "sleep-fudge-test")
	defer server.Close()

	// Create our function.
	var (
		started                int64
		startedAt              []time.Time
		completed              int64
		runIDA, runIDB, runIDC string
	)
	evtName := "test/sleep-fudge"

	/*
		- Run A, B, C
		- A schedules sleep behind B and C
		- All concurrency of 1
		- A increases a counter
		- B has a really long sleep
			- B checks the counter and sleeps
		- Next job to run shouldn't be C, but A
	*/

	var (
		stepCountersLock sync.Mutex
		stepCounters     = make(map[string][]time.Time)
	)

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "test-sleep-fudge",
			Concurrency: []inngestgo.ConfigStepConcurrency{
				{
					Limit: 1,
				},
			},
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			logger.StdlibLogger(ctx).Info("got request", "run_id", input.InputCtx.RunID)

			if runIDA == "" {
				runIDA = input.InputCtx.RunID
				atomic.AddInt64(&started, 1)
				startedAt = append(startedAt, time.Now())
				logger.StdlibLogger(ctx).Info("started run A")
			} else if runIDB == "" && input.InputCtx.RunID != runIDA {
				runIDB = input.InputCtx.RunID
				atomic.AddInt64(&started, 1)
				startedAt = append(startedAt, time.Now())
				logger.StdlibLogger(ctx).Info("started run B")
			} else if runIDC == "" && input.InputCtx.RunID != runIDA && input.InputCtx.RunID != runIDB {
				runIDC = input.InputCtx.RunID
				atomic.AddInt64(&started, 1)
				startedAt = append(startedAt, time.Now())
				logger.StdlibLogger(ctx).Info("started run C")
			}

			if input.InputCtx.RunID == runIDA {
				stepCountersLock.Lock()
				stepCounters[input.InputCtx.RunID] = append(stepCounters[input.InputCtx.RunID], time.Now())
				stepCountersLock.Unlock()

				logger.StdlibLogger(ctx).Info("sleeping run A")

				step.Sleep(ctx, "nap", 5*time.Second)

				logger.StdlibLogger(ctx).Info("done sleeping run A")
			}

			_, _ = step.Run(ctx, "test", func(ctx context.Context) (any, error) {
				if input.InputCtx.RunID == runIDB {
					stepCountersLock.Lock()
					stepCounters[input.InputCtx.RunID] = append(stepCounters[input.InputCtx.RunID], time.Now())
					stepCountersLock.Unlock()
					logger.StdlibLogger(ctx).Info("sleeping run A")

					step.Sleep(ctx, "nap", 5*time.Second)
				}

				stepCountersLock.Lock()
				stepCounters[input.InputCtx.RunID] = append(stepCounters[input.InputCtx.RunID], time.Now())
				stepCountersLock.Unlock()

				return "done", nil
			})

			stepCountersLock.Lock()
			stepCounters[input.InputCtx.RunID] = append(stepCounters[input.InputCtx.RunID], time.Now())
			stepCountersLock.Unlock()

			logger.StdlibLogger(ctx).Info("completing run", "run", input.InputCtx.RunID)

			atomic.AddInt64(&completed, 1)
			return true, nil
		},
	)
	require.NoError(t, err)

	// Register the fns via the test SDK harness above.
	registerFuncs()

	// Send 3 events for 3 runs
	for i := 0; i < 3; i++ {
		_, err = inngestClient.Send(ctx, inngestgo.Event{
			Name: evtName,
			Data: map[string]any{"sleep": "ok"},
		})
		require.NoError(t, err)
	}

	t.Run("expected values", func(t *testing.T) {
		require.Eventually(t, func() bool {
			return atomic.LoadInt64(&completed) == 3
		}, time.Minute, 2*time.Second)
	})

	t.Run("complete", func(t *testing.T) {
		//run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{
		//	Status:         models.FunctionStatusCompleted,
		//	ChildSpanCount: 3,
		//	Timeout:        9 * time.Second,
		//	Interval:       3 * time.Second,
		//})
		//require.EqualValues(t, 1, atomic.LoadInt32(&completed))
		//
		//require.NotEqual(t, models.RunTraceSpanStatusRunning.String(), run.Trace.Status)
		//
		//// output test
		//require.NotNil(t, run.Trace.OutputID)
		//output := c.RunSpanOutput(ctx, *run.Trace.OutputID)
		//require.NotNil(t, output)
		//c.ExpectSpanOutput(t, "true", output)
		//
		//t.Run("sleep", func(t *testing.T) {
		//	sleep := run.Trace.ChildSpans[0]
		//	assert.Equal(t, models.RunTraceSpanStatusCompleted.String(), sleep.Status)
		//	assert.Equal(t, 2, len(sleep.ChildSpans))
		//	assert.Equal(t, "nap", sleep.Name)
		//	assert.Equal(t, models.StepOpSleep.String(), sleep.StepOp)
		//	assert.Nil(t, sleep.OutputID)
		//
		//	// verify step info
		//	info := &models.SleepStepInfo{}
		//	byt, err := json.Marshal(sleep.StepInfo)
		//	assert.NoError(t, err)
		//	assert.NoError(t, json.Unmarshal(byt, info))
		//
		//	assert.True(t, time.Now().After(info.SleepUntil))
		//
		//	// first is the failed attempt
		//	t.Run("failed execution", func(t *testing.T) {
		//		exec := sleep.ChildSpans[0]
		//		assert.Equal(t, models.RunTraceSpanStatusFailed.String(), exec.Status)
		//		assert.Equal(t, "Attempt 0", exec.Name)
		//		assert.NotNil(t, exec.OutputID)
		//
		//		execOutput := c.RunSpanOutput(ctx, *exec.OutputID)
		//		assert.NotNil(t, execOutput)
		//		c.ExpectSpanErrorOutput(t, "throwing an initial error", "", execOutput)
		//	})
		//})
	})
}
