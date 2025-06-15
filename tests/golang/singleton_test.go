package golang

import (
	"context"
	"fmt"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
	"sync/atomic"
	"testing"
	"time"
)

func TestSingletonFunction(t *testing.T) {
	inngestClient, server, registerFuncs := NewSDKHandler(t, "singleton")
	defer server.Close()

	var (
		inProgress, total int32

		numEvents  = 50
		fnDuration = 10
	)

	trigger := "test/singleton-fn"

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:        "fn-singleton",
			Singleton: &inngestgo.FnSingleton{Key: inngestgo.StrPtr("event.data.user.id")},
		},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			fmt.Println("Running func", *input.Event.ID, input.Event.Data)
			atomic.AddInt32(&total, 1)

			next := atomic.AddInt32(&inProgress, 1)

			// We should never exceed more than one function running
			require.Less(t, next, int32(2))

			<-time.After(time.Duration(fnDuration) * time.Second)

			atomic.AddInt32(&inProgress, -1)
			return true, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	for i := 0; i < numEvents; i++ {
		go func() {
			_, err := inngestClient.Send(context.Background(), inngestgo.Event{
				Name: trigger,
				Data: map[string]any{
					"user": map[string]any{"id": 42},
				},
			})
			require.NoError(t, err)
		}()
	}

	// Eventually the fn starts.
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&inProgress) == 1
	}, 2*time.Second, 50*time.Millisecond)

	for i := 0; i < fnDuration; i++ {
		<-time.After(time.Second)
		require.LessOrEqual(t, atomic.LoadInt32(&inProgress), int32(1))
	}

	// The function executed only once, the other two events were ignored
	require.EqualValues(t, 1, atomic.LoadInt32(&total))
}

func TestSingletonFunctionWithKeyResolvingToFalse(t *testing.T) {
	inngestClient, server, registerFuncs := NewSDKHandler(t, "singleton")
	defer server.Close()

	var (
		inProgress, total int32

		numEvents  = 2
		fnDuration = 5
	)

	trigger := "test/singleton-fn"

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:        "fn-singleton",
			Singleton: &inngestgo.FnSingleton{Key: inngestgo.StrPtr("event.data.decision")},
		},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			fmt.Println("Running func", *input.Event.ID, input.Event.Data)
			atomic.AddInt32(&total, 1)

			atomic.AddInt32(&inProgress, 1)

			<-time.After(time.Duration(fnDuration) * time.Second)

			atomic.AddInt32(&inProgress, -1)
			return true, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	for i := 0; i < numEvents; i++ {
		go func() {
			_, err := inngestClient.Send(context.Background(), inngestgo.Event{
				Name: trigger,
				Data: map[string]any{
					"decision": false,
				},
			})
			require.NoError(t, err)
		}()
	}

	// Eventually both fn start because they are not running as a singleton
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&inProgress) == 2
	}, 2*time.Second, 50*time.Millisecond)

	// The function executed twice
	require.EqualValues(t, 2, atomic.LoadInt32(&total))
}
