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
	inngestClient, server, registerFuncs := NewSDKHandler(t, "fail-app")
	defer server.Close()

	var count int32
	_, err := inngestgo.CreateFunction(
		inngestClient,
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
	require.NoError(t, err)
	_, err = inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "handle-failures", Retries: inngestgo.IntPtr(0)},
		inngestgo.EventTrigger(
			"inngest/function.finished",
			inngestgo.StrPtr("event.data.function_id == 'fail-app-always-fail'"),
		),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			evt := input.Event

			// Assert that failure handlers are called with valid data.
			require.NotEmpty(t, evt.Data["run_id"])
			require.NotEmpty(t, evt.Data["function_id"])

			error, ok := evt.Data["error"].(map[string]any)
			require.True(t, ok, evt.Data)
			require.NotNil(t, error)

			switch error["error"] {
			case "NonRetriableError": // step error
				require.Contains(t, error["message"], "unhandled step error: nope", evt.Data)
			default: // panic
				require.Contains(t, error["message"], "function panicked: nope", evt.Data)
			}

			atomic.AddInt32(&count, 1)
			return true, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	_, err = inngestClient.Send(context.Background(), inngestgo.Event{
		Name: "test/fail",
		Data: map[string]any{
			"test": true,
			"id":   "1",
		},
	})
	require.NoError(t, err)

	<-time.After(3 * time.Second)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&count) == 1
	}, 20*time.Second, 5*time.Millisecond)
}

func TestFunctionFailureHandlingWithRateLimit(t *testing.T) {
	ctx := context.Background()
	inngestClient, server, registerFuncs := NewSDKHandler(t, "failed-rate-limit")
	defer server.Close()

	evtName := "fail/rate-limit"

	var failed, handled int32
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:        "failed",
			RateLimit: &inngestgo.ConfigRateLimit{Limit: 1, Period: 24 * time.Hour, Key: inngestgo.StrPtr("event.data.number")},
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			atomic.AddInt32(&failed, 1)
			return nil, inngestgo.NoRetryError(fmt.Errorf("failed"))
		},
	)
	require.NoError(t, err)
	// mimic a `onFailure` handler, with the original function defining rate limits
	_, err = inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:        "failed-failure",
			RateLimit: &inngestgo.ConfigRateLimit{Limit: 1, Period: 24 * time.Hour, Key: inngestgo.StrPtr("event.data.number")},
		},
		inngestgo.EventTrigger("inngest/function.failed", inngestgo.StrPtr(`event.data.function_id == "failed-rate-limit-failed"`)),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			atomic.AddInt32(&handled, 1)
			return "handled", nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	_, err = inngestClient.Send(ctx, inngestgo.Event{
		Name: evtName,
		Data: map[string]any{"number": 10},
	})
	require.NoError(t, err)

	<-time.After(5 * time.Second)

	require.Eventually(t, func() bool { return atomic.LoadInt32(&failed) == 1 }, 10*time.Second, time.Second)
	require.Equal(t, int32(1), atomic.LoadInt32(&handled))

	// send another, it should be rate limited
	_, err = inngestClient.Send(ctx, inngestgo.Event{
		Name: evtName,
		Data: map[string]any{"number": 10},
	})
	require.NoError(t, err)
	<-time.After(2 * time.Second)

	require.Eventually(t, func() bool { return atomic.LoadInt32(&failed) == 1 }, 10*time.Second, time.Second)
	require.Equal(t, int32(1), atomic.LoadInt32(&handled))

	// send a different payload
	_, err = inngestClient.Send(ctx, inngestgo.Event{
		Name: evtName,
		Data: map[string]any{"number": 1},
	})
	require.NoError(t, err)
	<-time.After(5 * time.Second)

	require.Eventually(t, func() bool { return atomic.LoadInt32(&failed) == 2 }, 10*time.Second, time.Second)
	require.Equal(t, int32(2), atomic.LoadInt32(&handled))
}
