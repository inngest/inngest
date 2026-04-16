package golang

import (
	"context"
	"encoding/json"
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

type cronTriggerPayload struct {
	Data struct {
		ScheduledAt string `json:"scheduledAt"`
		FireAt      string `json:"fireAt"`
	} `json:"data"`
}

func TestCron(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	c := client.New(t)
	r := require.New(t)

	inngestClient, server, registerFuncs := NewSDKHandler(t, randomSuffix("cron"))
	defer server.Close()

	var (
		counter int32
		runID   string
	)

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "cron-test"},
		inngestgo.CronTrigger("* * * * *"),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}
			atomic.AddInt32(&counter, 1)

			return "schedule done", nil
		},
	)
	r.NoError(err)
	registerFuncs()

	t.Run("cron should run", func(t *testing.T) {
		r.Eventually(func() bool {
			return atomic.LoadInt32(&counter) == 1
		}, 121*time.Second, 5*time.Second)
	})

	t.Run("trace run should have appropriate data", func(t *testing.T) {
		run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{Status: models.FunctionStatusCompleted})

		r.NotNil(run.CronSchedule)
		r.Equal("* * * * *", *run.CronSchedule)

		r.NotNil(run.Trace)
		r.True(run.Trace.IsRoot)
		r.Equal(models.RunTraceSpanStatusCompleted.String(), run.Trace.Status)
		// output test
		r.NotNil(run.Trace.OutputID)
		output := c.RunSpanOutput(ctx, *run.Trace.OutputID)
		c.ExpectSpanOutput(t, "schedule done", output)

		t.Run("trigger", func(t *testing.T) {
			// check trigger
			trigger := c.RunTrigger(ctx, runID)
			assert.NotNil(t, trigger)
			assert.Equal(t, 1, len(trigger.IDs))
			assert.False(t, trigger.Timestamp.IsZero())
			assert.NotNil(t, trigger.Cron)
			assert.Nil(t, trigger.BatchID)
		})
	})
}

func TestCronRemoveCronTrigger(t *testing.T) {
	t.Parallel()

	r := require.New(t)

	inngestClient, server, registerFuncs := NewSDKHandler(t, randomSuffix("remove-cron"))
	defer server.Close()

	var (
		counter int32
		runID   string
	)

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "cron-test"},
		inngestgo.CronTrigger("* * * * *"),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}
			atomic.AddInt32(&counter, 1)

			return "schedule done", nil
		},
	)
	r.NoError(err)
	registerFuncs()

	t.Run("cron should run", func(t *testing.T) {
		r.Eventually(func() bool {
			return atomic.LoadInt32(&counter) == 1
		}, 121*time.Second, 5*time.Second)
	})

	t.Run("re-register function to remove cron", func(t *testing.T) {
		_, err := inngestgo.CreateFunction(
			inngestClient,
			inngestgo.FunctionOpts{ID: "cron-test"},
			inngestgo.EventTrigger("test/ehh", nil),
			func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
				if runID == "" {
					runID = input.InputCtx.RunID
				}
				atomic.AddInt32(&counter, 1)

				return "schedule done", nil
			},
		)
		r.NoError(err)
		registerFuncs()

		time.Sleep(time.Minute)
		r.Equal(int32(1), atomic.LoadInt32(&counter))
	})
}

func TestCronUpdateCronTrigger(t *testing.T) {
	t.Parallel()

	r := require.New(t)

	inngestClient, server, registerFuncs := NewSDKHandler(t, randomSuffix("reduce-cron-frequency"))
	defer server.Close()

	var (
		counter int32
		runID   string
	)

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "cron-test"},
		inngestgo.CronTrigger("*/2 * * * *"),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}
			atomic.AddInt32(&counter, 1)

			return "schedule done", nil
		},
	)
	r.NoError(err)
	registerFuncs()

	t.Run("cron should run", func(t *testing.T) {
		r.Eventually(func() bool {
			return atomic.LoadInt32(&counter) == 1
		}, 241*time.Second, 5*time.Second)
	})

	t.Run("re-register function to reduce cron frequency", func(t *testing.T) {
		_, err := inngestgo.CreateFunction(
			inngestClient,
			inngestgo.FunctionOpts{ID: "cron-test"},
			inngestgo.CronTrigger("* * * * *"),
			func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
				if runID == "" {
					runID = input.InputCtx.RunID
				}
				atomic.AddInt32(&counter, 1)
				fmt.Println("scheduled function ran at", time.Now())

				return "schedule done", nil
			},
		)
		r.NoError(err)
		registerFuncs()

		r.Eventually(func() bool {
			return atomic.LoadInt32(&counter) == 1
		}, 121*time.Second, 5*time.Second)
	})
}

func TestCronAddCronTrigger(t *testing.T) {
	t.Parallel()

	r := require.New(t)

	inngestClient, server, registerFuncs := NewSDKHandler(t, randomSuffix("add-cron-trigger"))
	defer server.Close()

	var (
		counter int32
		runID   string
	)

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "cron-test"},
		inngestgo.CronTrigger("*/2 * * * *"),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}
			atomic.AddInt32(&counter, 1)

			return "schedule done", nil
		},
	)
	r.NoError(err)
	registerFuncs()

	t.Run("re-register function to add another cron trigger", func(t *testing.T) {
		_, err := inngestgo.CreateFunction(
			inngestClient,
			inngestgo.FunctionOpts{ID: "cron-test"},
			inngestgo.MultipleTriggers{
				inngestgo.CronTrigger("*/2 * * * *"),
				inngestgo.CronTrigger("* * * * *"),
			},
			func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
				if runID == "" {
					runID = input.InputCtx.RunID
				}
				atomic.AddInt32(&counter, 1)
				fmt.Println("scheduled function ran at", time.Now())

				return "schedule done", nil
			},
		)
		r.NoError(err)
		registerFuncs()

		r.Eventually(func() bool {
			return atomic.LoadInt32(&counter) == 2
		}, 241*time.Second, 5*time.Second)
	})
}

// TestCronJitter verifies that a cron function with jitter configured fires
// with both scheduledAt and fireAt in the event data, and that fireAt is
// within the jitter window.
func TestCronJitter(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	r := require.New(t)
	c := client.New(t)

	appID := randomSuffix("cron-jitter")
	inngestClient, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	var (
		counter    int32
		runID      string
		executedAt atomic.Value
	)

	jitterDuration := 30 * time.Second
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "cron-jitter-test"},
		inngestgo.CronTriggerWithJitter("* * * * *", jitterDuration),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}
			executedAt.Store(time.Now().UTC())
			atomic.AddInt32(&counter, 1)
			return "schedule done", nil
		},
	)
	r.NoError(err)
	registerFuncs()

	t.Run("cron should run", func(t *testing.T) {
		r.Eventually(func() bool {
			return atomic.LoadInt32(&counter) == 1
		}, 121*time.Second, 5*time.Second)
	})

	t.Run("trace run should have appropriate data", func(t *testing.T) {
		run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{Status: models.FunctionStatusCompleted})

		r.NotNil(run.CronSchedule)
		r.Equal("* * * * *", *run.CronSchedule)

		t.Run("trigger", func(t *testing.T) {
			trigger := c.RunTrigger(ctx, runID)
			require.NotNil(t, trigger)
			require.NotNil(t, trigger.Cron)
			require.GreaterOrEqual(t, len(trigger.Payloads), 1)

			_, scheduledAt, fireAt := parseCronTriggerPayload(t, trigger.Payloads[0])
			executedAtVal := executedAt.Load()
			require.NotNil(t, executedAtVal, "executedAt should be captured in the function body")
			executedAtTime, ok := executedAtVal.(time.Time)
			require.True(t, ok, "executedAt should be a time.Time")

			// assert that scheduledAt < fireAt <= scheduledAt + jitterDuration,
			// and that the function actually executes no later than fireAt + tolerance
			assert.True(t, fireAt.After(scheduledAt), "fireAt %s should be after scheduledAt %s (jitter should be applied)", fireAt, scheduledAt)
			assert.True(t, fireAt.Before(scheduledAt.Add(jitterDuration)), "fireAt %s should be within %s jitter of scheduledAt %s", fireAt, jitterDuration, scheduledAt)
			assert.True(t, !executedAtTime.Before(scheduledAt), "executedAt %s should not be before scheduledAt %s", executedAtTime, scheduledAt)
			tolerance := 10 * time.Second
			assert.True(t, !executedAtTime.After(fireAt.Add(tolerance)), "executedAt %s should not be too much after fireAt %s (tolerance %s)", executedAtTime, fireAt, tolerance)
		})
	})
}

// TestCronJitterRemovalAppliesToCurrentOccurrence verifies that removing jitter
// before the next cron boundary causes the current occurrence to use the
// canonical boundary rather than the previously registered jitter.
func TestCronJitterRemovalAppliesToCurrentOccurrence(t *testing.T) {
	// Note - this test intentionally does not run in parallel to avoid contention with other tests.
	ctx := context.Background()

	r := require.New(t)
	c := client.New(t)

	// ensures 20s before the next minute to both register and update cron configurations
	waitForSafeCronWindow(t, 20*time.Second)

	appID := randomSuffix("cron-jitter-update")
	inngestClient, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	var (
		counter    int32
		runID      string
		executedAt atomic.Value
	)

	// Register with 50s jitter first.
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "cron-jitter-test"},
		inngestgo.CronTriggerWithJitter("* * * * *", 50*time.Second),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}
			executedAt.Store(time.Now().UTC())
			atomic.AddInt32(&counter, 1)
			return "schedule done", nil
		},
	)
	r.NoError(err)
	registerFuncs()

	// Re-register without jitter. Since handleCron() reads jitter from the
	// live config, removing jitter should cause the current occurrence to fire
	// at the canonical boundary.
	_, err = inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "cron-jitter-test"},
		inngestgo.CronTrigger("* * * * *"),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}
			executedAt.Store(time.Now().UTC())
			atomic.AddInt32(&counter, 1)
			return "schedule done", nil
		},
	)
	r.NoError(err)
	registerFuncs()

	r.Eventually(func() bool {
		return atomic.LoadInt32(&counter) == 1
	}, 121*time.Second, 5*time.Second)

	run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{Status: models.FunctionStatusCompleted})
	r.NotNil(run.CronSchedule)
	r.Equal("* * * * *", *run.CronSchedule)

	trigger := c.RunTrigger(ctx, runID)
	require.NotNil(t, trigger)
	require.NotNil(t, trigger.Cron)
	require.GreaterOrEqual(t, len(trigger.Payloads), 1)

	_, scheduledAt, fireAt := parseCronTriggerPayload(t, trigger.Payloads[0])
	executedAtVal := executedAt.Load()
	require.NotNil(t, executedAtVal, "executedAt should be captured in the function body")
	executedAtTime, ok := executedAtVal.(time.Time)
	require.True(t, ok, "executedAt should be a time.Time")

	assert.True(t, !fireAt.Before(scheduledAt), "fireAt %s should not be before scheduledAt %s", fireAt, scheduledAt)
	assert.True(t, fireAt.Equal(scheduledAt), "fireAt %s should equal scheduledAt %s once jitter is removed", fireAt, scheduledAt)
	assert.True(t, !executedAtTime.Before(scheduledAt), "executedAt %s should not be before scheduledAt %s", executedAtTime, scheduledAt)
	tolerance := 10 * time.Second
	assert.True(t, !executedAtTime.After(fireAt.Add(tolerance)), "executedAt %s should not be too much after fireAt %s (tolerance %s)", executedAtTime, fireAt, tolerance)
}

// parseCronTriggerPayload is a helper function to parse the payload of a cron trigger
// and extract the scheduledAt and fireAt times for assertions in tests.
func parseCronTriggerPayload(t *testing.T, raw string) (cronTriggerPayload, time.Time, time.Time) {
	t.Helper()

	var payload cronTriggerPayload
	err := json.Unmarshal([]byte(raw), &payload)
	require.NoError(t, err)
	require.NotEmpty(t, payload.Data.ScheduledAt, "scheduledAt should be present in event data")
	require.NotEmpty(t, payload.Data.FireAt, "fireAt should be present in event data")

	scheduledAt, err := time.Parse(time.RFC3339, payload.Data.ScheduledAt)
	require.NoError(t, err)
	fireAt, err := time.Parse(time.RFC3339, payload.Data.FireAt)
	require.NoError(t, err)

	return payload, scheduledAt, fireAt
}

// waitForSafeCronWindow ensures we have enough "minLead" seconds before the next minute
// boundary to both register and update cron configurations within the same interval
func waitForSafeCronWindow(t *testing.T, minLead time.Duration) {
	t.Helper()

	now := time.Now().UTC()
	nextBoundary := now.Truncate(time.Minute).Add(time.Minute)
	untilBoundary := time.Until(nextBoundary)
	if untilBoundary < minLead {
		time.Sleep(untilBoundary + 2*time.Second)
	}
}
