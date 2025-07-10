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
	inngestClient, server, registerFuncs := NewSDKHandler(t, "rate-limit")
	defer server.Close()

	var counter int32
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:        "rate-limit",
			RateLimit: &inngestgo.ConfigRateLimit{Limit: 1, Period: 24 * time.Hour, Key: inngestgo.StrPtr("event.data.number")},
		},
		inngestgo.EventTrigger("test/ratelimit", nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			atomic.AddInt32(&counter, 1)
			return true, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	_, err = inngestClient.Send(ctx, inngestgo.Event{
		Name: "test/ratelimit",
		Data: map[string]any{"number": 10},
	})
	require.NoError(t, err)

	<-time.After(5 * time.Second)

	require.Eventually(t, func() bool { return atomic.LoadInt32(&counter) == 1 }, 5*time.Second, time.Second)

	// send another, it should be rate limited
	_, err = inngestClient.Send(ctx, inngestgo.Event{
		Name: "test/ratelimit",
		Data: map[string]any{"number": 10},
	})
	require.NoError(t, err)
	<-time.After(5 * time.Second)

	require.Eventually(t, func() bool { return atomic.LoadInt32(&counter) == 1 }, 5*time.Second, time.Second)

	// send a different payload
	_, err = inngestClient.Send(ctx, inngestgo.Event{
		Name: "test/ratelimit",
		Data: map[string]any{"number": 1},
	})
	require.NoError(t, err)
	<-time.After(5 * time.Second)

	require.Eventually(t, func() bool { return atomic.LoadInt32(&counter) == 2 }, 5*time.Second, time.Second)
}
