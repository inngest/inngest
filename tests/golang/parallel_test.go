package golang

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/group"
	"github.com/inngest/inngestgo/step"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParallelSteps(t *testing.T) {
	ctx := context.Background()

	c := client.New(t)
	inngestClient, server, registerFuncs := NewSDKHandler(t, "parallel")
	defer server.Close()

	var (
		counter int32
		runID   string
	)

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "concurrent", Concurrency: []inngestgo.ConfigStepConcurrency{
			{Limit: 2, Scope: enums.ConcurrencyScopeFn},
		}},
		inngestgo.EventTrigger("test/parallel", nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
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
	require.NoError(t, err)
	registerFuncs()

	_, err = inngestClient.Send(ctx, inngestgo.Event{
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
			if cspan.StepOp == "" {
				continue
			}
			t.Run(fmt.Sprintf("child: %s", cspan.Name), func(t *testing.T) {
				assert.Equal(t, 0, cspan.Attempts)
				assert.Equal(t, models.StepOpRun.String(), cspan.StepOp)
				assert.Equal(t, models.RunTraceSpanStatusCompleted.String(), cspan.Status)
			})
		}
	})
}

func TestParallelCoalesce(t *testing.T) {
	// Steps are only called once, regardless of whether they're inside or
	// outside of a parallel group. This test:
	// 1. Diverges into 3 parallel steps.
	// 2. Converges into a single step.
	// 3. Diverges again into 2 parallel steps.
	// 4. Converges again into a single step.

	r := require.New(t)
	ctx := context.Background()
	c := client.New(t)
	ic, server, sync := NewSDKHandler(t, randomSuffix("app"))
	defer server.Close()

	eventName := randomSuffix("event")
	var (
		stepA1Counter      int32
		stepA2Counter      int32
		stepA3Counter      int32
		stepBetweenCounter int32
		stepB1Counter      int32
		stepB2Counter      int32
		stepAfterCounter   int32
		runID              string
	)
	_, err := inngestgo.CreateFunction(
		ic,
		inngestgo.FunctionOpts{ID: "fn"},
		inngestgo.EventTrigger(eventName, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}

			res := group.Parallel(ctx,
				func(ctx context.Context) (any, error) {
					return step.Run(ctx, "a1", func(ctx context.Context) (any, error) {
						atomic.AddInt32(&stepA1Counter, 1)
						<-time.After(100 * time.Millisecond)
						return nil, nil
					})
				},
				func(ctx context.Context) (any, error) {
					return step.Run(ctx, "a2", func(ctx context.Context) (any, error) {
						atomic.AddInt32(&stepA2Counter, 1)
						<-time.After(100 * time.Millisecond)
						return nil, nil
					})
				},
				func(ctx context.Context) (any, error) {
					return step.Run(ctx, "a3", func(ctx context.Context) (any, error) {
						atomic.AddInt32(&stepA3Counter, 1)
						<-time.After(100 * time.Millisecond)
						return nil, nil
					})
				},
			)
			err := res.AnyError()
			if err != nil {
				return nil, err
			}

			_, err = step.Run(ctx, "between", func(ctx context.Context) (any, error) {
				atomic.AddInt32(&stepBetweenCounter, 1)
				time.Sleep(100 * time.Millisecond)
				return nil, nil
			})
			if err != nil {
				return nil, err
			}

			res = group.Parallel(ctx,
				func(ctx context.Context) (any, error) {
					return step.Run(ctx, "b1", func(ctx context.Context) (any, error) {
						atomic.AddInt32(&stepB1Counter, 1)
						<-time.After(100 * time.Millisecond)
						return nil, nil
					})
				},
				func(ctx context.Context) (any, error) {
					return step.Run(ctx, "b2", func(ctx context.Context) (any, error) {
						atomic.AddInt32(&stepB2Counter, 1)
						<-time.After(100 * time.Millisecond)
						return nil, nil
					})
				},
			)
			err = res.AnyError()
			if err != nil {
				return nil, err
			}

			_, err = step.Run(ctx, "after", func(ctx context.Context) (any, error) {
				atomic.AddInt32(&stepAfterCounter, 1)
				time.Sleep(100 * time.Millisecond)
				return nil, nil
			})
			if err != nil {
				return nil, err
			}
			return res, nil
		},
	)
	r.NoError(err)
	sync()

	_, err = ic.Send(ctx, inngestgo.Event{Name: eventName})
	r.NoError(err)

	c.WaitForRunStatus(ctx, t, "COMPLETED", &runID)
	r.Equal(int32(1), stepA1Counter)
	r.Equal(int32(1), stepA2Counter)
	r.Equal(int32(1), stepA3Counter)
	r.Equal(int32(1), stepBetweenCounter)
	r.Equal(int32(1), stepAfterCounter)
}
