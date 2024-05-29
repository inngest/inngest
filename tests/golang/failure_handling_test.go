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

func TestFunctionFailureHandlingWithRateLimit(t *testing.T) {
	ctx := context.Background()
	h, server, registerFuncs := NewSDKHandler(t, "failed-rate-limit")
	defer server.Close()

	evtName := "fail/rate-limit"

	var failed, handled int32
	fun := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name:      "failed",
			RateLimit: &inngestgo.RateLimit{Limit: 1, Period: 24 * time.Hour, Key: inngestgo.StrPtr("event.data.number")},
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			atomic.AddInt32(&failed, 1)
			return nil, inngestgo.NoRetryError(fmt.Errorf("failed"))
		},
	)
	// mimic a `onFailure` handler, with the original function defining rate limits
	fail := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name:      "failed-failure",
			RateLimit: &inngestgo.RateLimit{Limit: 1, Period: 24 * time.Hour, Key: inngestgo.StrPtr("event.data.number")},
		},
		inngestgo.EventTrigger("inngest/function.failed", inngestgo.StrPtr(`event.data.function_id == "failed-rate-limit-failed"`)),
		func(ctx context.Context, input inngestgo.Input[inngestgo.GenericEvent[map[string]any, any]]) (any, error) {
			atomic.AddInt32(&handled, 1)
			return "handled", nil
		},
	)
	h.Register(fun, fail)
	registerFuncs()

	_, err := inngestgo.Send(ctx, inngestgo.Event{
		Name: evtName,
		Data: map[string]any{"number": 10},
	})
	require.NoError(t, err)

	<-time.After(5 * time.Second)

	require.Eventually(t, func() bool { return atomic.LoadInt32(&failed) == 1 }, 10*time.Second, time.Second)
	require.Equal(t, int32(1), atomic.LoadInt32(&handled))

	// send another, it should be rate limited
	_, err = inngestgo.Send(ctx, inngestgo.Event{
		Name: evtName,
		Data: map[string]any{"number": 10},
	})
	require.NoError(t, err)
	<-time.After(2 * time.Second)

	require.Eventually(t, func() bool { return atomic.LoadInt32(&failed) == 1 }, 10*time.Second, time.Second)
	require.Equal(t, int32(1), atomic.LoadInt32(&handled))

	// send a different payload
	_, err = inngestgo.Send(ctx, inngestgo.Event{
		Name: evtName,
		Data: map[string]any{"number": 1},
	})
	require.NoError(t, err)
	<-time.After(5 * time.Second)

	require.Eventually(t, func() bool { return atomic.LoadInt32(&failed) == 2 }, 10*time.Second, time.Second)
	require.Equal(t, int32(2), atomic.LoadInt32(&handled))
}
