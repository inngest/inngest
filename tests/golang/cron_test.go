package golang

import (
	"context"
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
	ctx := context.Background()

	c := client.New(t)
	r := require.New(t)

	inngestClient, server, registerFuncs := NewSDKHandler(t, "cron")
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
		}, time.Minute, 5*time.Second)
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
