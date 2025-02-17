package golang

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/experimental/group"
	"github.com/inngest/inngestgo/step"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type parallelTestEvt inngestgo.GenericEvent[any, any]

func TestParallelSteps(t *testing.T) {
	ctx := context.Background()

	c := client.New(t)
	h, server, registerFuncs := NewSDKHandler(t, "parallel")
	defer server.Close()

	var (
		counter int32
		runID   string
	)

	a := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{Name: "concurrent", Concurrency: []inngest.Concurrency{
			{Limit: 2, Scope: enums.ConcurrencyScopeFn},
		}},
		inngestgo.EventTrigger("test/parallel", nil),
		func(ctx context.Context, input inngestgo.Input[parallelTestEvt]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}

			res := group.Parallel(ctx,
				func(ctx context.Context) (any, error) {
					return step.Run(ctx, "p1", func(ctx context.Context) (any, error) {
						atomic.AddInt32(&counter, 1)
						<-time.After(5 * time.Second)
						return "p1", nil
					})
				},
				func(ctx context.Context) (any, error) {
					return step.Run(ctx, "p2", func(ctx context.Context) (any, error) {
						atomic.AddInt32(&counter, 1)
						<-time.After(5 * time.Second)
						return "p2", nil
					})
				},
				func(ctx context.Context) (any, error) {
					return step.Run(ctx, "p3", func(ctx context.Context) (any, error) {
						atomic.AddInt32(&counter, 1)
						<-time.After(5 * time.Second)
						return "p3", nil
					})
				},
				func(ctx context.Context) (any, error) {
					return step.Run(ctx, "p4", func(ctx context.Context) (any, error) {
						atomic.AddInt32(&counter, 1)
						<-time.After(5 * time.Second)
						return "p4", nil
					})
				},
			)

			return res, nil
		},
	)

	h.Register(a)
	registerFuncs()

	_, err := inngestgo.Send(ctx, inngestgo.Event{
		Name: "test/parallel",
		Data: map[string]any{"hello": "world"},
	})
	require.NoError(t, err)

	t.Run("verify in-progress", func(t *testing.T) {
		<-time.After(2 * time.Second)
		require.Equal(t, int32(2), atomic.LoadInt32(&counter))

		_ = c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{
			Status:         models.FunctionStatusRunning,
			ChildSpanCount: 2,
			Timeout:        2 * time.Second,
			Interval:       200 * time.Millisecond,
		})
	})

	t.Run("verify completion", func(t *testing.T) {
		<-time.After(10 * time.Second)
		require.Equal(t, int32(4), atomic.LoadInt32(&counter))

		run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{
			Status:         models.FunctionStatusCompleted,
			ChildSpanCount: 4,
			Timeout:        5 * time.Second,
			Interval:       250 * time.Millisecond,
		})

		// check on spans
		for _, cspan := range run.Trace.ChildSpans {
			t.Run(fmt.Sprintf("child: %s", cspan.Name), func(t *testing.T) {
				assert.Equal(t, 0, cspan.Attempts)
				assert.Equal(t, models.StepOpRun.String(), cspan.StepOp)
				assert.Equal(t, models.RunTraceSpanStatusCompleted.String(), cspan.Status)
			})
		}
	})
}
