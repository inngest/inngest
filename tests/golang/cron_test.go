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
		}, 61*time.Second, 5*time.Second)
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
		}, 61*time.Second, 5*time.Second)
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
		}, 121*time.Second, 5*time.Second)
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
		}, 61*time.Second, 5*time.Second)
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
		}, 121*time.Second, 5*time.Second)
	})
}
