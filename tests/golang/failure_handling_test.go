package golang

import (
	"context"
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
	"github.com/stretchr/testify/require"
)

func TestFunctionFailureHandling(t *testing.T) {
	h, server, registerFuncs := NewSDKHandler(t, "fail-app")
	defer server.Close()

	var aCount, bCount int32
	a := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			ID:      "always-fail",
			Name:    "Always fail",
			Retries: inngestgo.IntPtr(0),
		},
		inngestgo.EventTrigger("test/fail", nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			val, err := step.Run(ctx, "1", func(ctx context.Context) (any, error) {
				if rand.Intn(10) == 1 {
					// also randomly panic to assert panics work in the Go SDK
					panic("nope")
				}
				return nil, fmt.Errorf("nope")
			})
			require.Nil(t, val)
			// Return the error from the step, failing the function.
			return nil, err
		},
	)
	b := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{Name: "handle-failures", Retries: inngestgo.IntPtr(0)},
		inngestgo.EventTrigger(
			"inngest/function.finished",
			inngestgo.StrPtr("event.data.function_id == 'fail-app-always-fail'"),
		),
		func(ctx context.Context, input inngestgo.Input[inngestgo.GenericEvent[map[string]any, any]]) (any, error) {
			evt := input.Event

			// Assert that failure handlers are called with valid data.
			require.NotEmpty(t, evt.Data["run_id"])
			require.NotEmpty(t, evt.Data["function_id"])

			error, ok := evt.Data["error"].(map[string]any)
			require.True(t, ok, evt.Data)
			require.NotNil(t, error)
			require.Contains(t, error["error"], "error calling function", evt.Data)
			require.Contains(t, error["error"], "nope", evt.Data)

			atomic.AddInt32(&bCount, 1)
			return true, nil
		},
	)
	h.Register(a, b)
	registerFuncs()

	_, err := inngestgo.Send(context.Background(), inngestgo.Event{
		Name: "test/fail",
		Data: map[string]any{
			"test": true,
			"id":   "1",
		},
	})
	require.NoError(t, err)

	<-time.After(3 * time.Second)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&bCount) == 1
	}, 15*time.Second, time.Second)
	require.EqualValues(t, 0, aCount)
}
