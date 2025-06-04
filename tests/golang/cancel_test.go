package golang

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCancelEvt inngestgo.GenericEvent[any]

func TestEventCancellation(t *testing.T) {
	ctx := context.Background()

	c := client.New(t)
	appName := uuid.New().String()
	inngestClient, server, registerFuncs := NewSDKHandler(t, appName)
	defer server.Close()

	var (
		runCounter   int32
		runCancelled int32
		runID        string
	)

	triggerEvtName := uuid.New().String()
	cancelEvtName := uuid.New().String()

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "test-cancel",
			Cancel: []inngestgo.ConfigCancel{
				{Event: cancelEvtName, If: inngestgo.StrPtr("async.data.cancel == event.data.cancel")},
			},
		},
		inngestgo.EventTrigger(triggerEvtName, nil),
		func(ctx context.Context, input inngestgo.Input[testCancelEvt]) (any, error) {
			_, _ = step.Run(ctx, "do something", func(ctx context.Context) (any, error) {
				runID = input.InputCtx.RunID
				fmt.Println("HELLO")

				atomic.AddInt32(&runCounter, 1)
				return nil, nil
			})

			step.Sleep(ctx, "stop", 30*time.Second)

			_, _ = step.Run(ctx, "should not happen", func(ctx context.Context) (any, error) {
				atomic.AddInt32(&runCounter, 1)
				return nil, nil
			})

			return true, nil
		},
	)
	require.NoError(t, err)

	_, err = inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "handle-cancel"},
		inngestgo.EventTrigger(
			"inngest/function.cancelled",
			inngestgo.StrPtr(fmt.Sprintf(
				"event.data.function_id == '%s-test-cancel'",
				appName,
			)),
		),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			fmt.Println("CANCELLED")

			atomic.AddInt32(&runCancelled, 1)

			return true, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	evt := inngestgo.Event{
		Name: triggerEvtName,
		Data: map[string]any{"cancel": 1},
	}
	_, err = inngestClient.Send(ctx, evt)
	require.NoError(t, err)

	<-time.After(3 * time.Second)

	t.Run("check run", func(t *testing.T) {
		require.Equal(t, int32(1), atomic.LoadInt32(&runCounter))
		require.Equal(t, int32(0), atomic.LoadInt32(&runCancelled))
	})

	t.Run("should cancel run", func(t *testing.T) {
		r := require.New(t)
		_, err = inngestClient.Send(ctx, inngestgo.Event{
			Name: cancelEvtName,
			Data: map[string]any{"cancel": 1},
		})
		require.NoError(t, err)

		r.EventuallyWithT(func(t *assert.CollectT) {
			a := assert.New(t)
			a.Equal(int32(1), atomic.LoadInt32(&runCounter))
			a.Equal(int32(1), atomic.LoadInt32(&runCancelled))
		}, 10*time.Second, 1*time.Second)
	})

	t.Run("trace run should have appropriate data", func(t *testing.T) {
		run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{
			Status:         models.FunctionStatusCancelled,
			Timeout:        10 * time.Second,
			Interval:       500 * time.Millisecond,
			ChildSpanCount: 2,
		})

		require.Equal(t, models.RunTraceSpanStatusCancelled.String(), run.Trace.Status)
	})
}
