package golang

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
)

func TestFunctionWithRateLimit(t *testing.T) {
	ctx := context.Background()
	h, server, registerFuncs := NewSDKHandler(t, "rate-limit")
	defer server.Close()

	var counter int32
	fun := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name:      "rate-limit",
			RateLimit: &inngestgo.RateLimit{Limit: 1, Period: 24 * time.Hour, Key: inngestgo.StrPtr("event.data.number")},
		},
		inngestgo.EventTrigger("test/ratelimit", nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			atomic.AddInt32(&counter, 1)
			return true, nil
		},
	)
	h.Register(fun)
	registerFuncs()

	_, err := inngestgo.Send(ctx, inngestgo.Event{
		Name: "test/ratelimit",
		Data: map[string]any{"number": 10},
	})
	require.NoError(t, err)

	<-time.After(5 * time.Second)

	require.Eventually(t, func() bool { return atomic.LoadInt32(&counter) == 1 }, 5*time.Second, time.Second)

	// send another, it should be rate limited
	_, err = inngestgo.Send(ctx, inngestgo.Event{
		Name: "test/ratelimit",
		Data: map[string]any{"number": 10},
	})
	require.NoError(t, err)
	<-time.After(5 * time.Second)

	require.Eventually(t, func() bool { return atomic.LoadInt32(&counter) == 1 }, 5*time.Second, time.Second)

	// send a different payload
	_, err = inngestgo.Send(ctx, inngestgo.Event{
		Name: "test/ratelimit",
		Data: map[string]any{"number": 1},
	})
	require.NoError(t, err)
	<-time.After(5 * time.Second)

	require.Eventually(t, func() bool { return atomic.LoadInt32(&counter) == 2 }, 5*time.Second, time.Second)
}
