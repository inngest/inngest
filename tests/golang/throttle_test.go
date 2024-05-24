package golang

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
	"github.com/stretchr/testify/require"
)

func TestThrottle(t *testing.T) {

	t.Run("Basic throttling with a single limit", func(t *testing.T) {
		h, server, registerFuncs := NewSDKHandler(t, "concurrency")
		defer server.Close()

		trigger := "test/timeouts-start"

		var (
			total int32
			funcs = 5
		)

		a := inngestgo.CreateFunction(
			inngestgo.FunctionOpts{
				Name: "throttle test",
				Throttle: &inngestgo.Throttle{
					Limit:  1,
					Period: 5 * time.Second,
				},
			},
			inngestgo.EventTrigger(trigger, nil),
			func(ctx context.Context, input inngestgo.Input[inngestgo.GenericEvent[any, any]]) (any, error) {
				fmt.Println("Throttled function hit")
				atomic.AddInt32(&total, 1)
				return true, nil
			},
		)

		h.Register(a)
		registerFuncs()

		// Run 5 functions
		for i := 0; i < funcs; i++ {
			go func() {
				_, err := inngestgo.Send(context.Background(), inngestgo.Event{
					Name: trigger,
					Data: map[string]any{"test": true},
				})
				require.NoError(t, err)
			}()
		}

		// Wait for the first function to finish, but not long enough for the second function
		// to start.
		<-time.After(time.Second)

		// Ensure that each function finishes after 3 seconds.
		for i := 1; i <= funcs; i++ {
			require.EqualValues(t, i, total)
			<-time.After(5 * time.Second)
		}
	})

	t.Run("Throttling with keys separates values", func(t *testing.T) {
		h, server, registerFuncs := NewSDKHandler(t, "concurrency")
		defer server.Close()

		trigger := "test/timeouts-start-key"

		var (
			total int32
		)

		a := inngestgo.CreateFunction(
			inngestgo.FunctionOpts{
				Name: "throttle test with keys",
				Throttle: &inngestgo.Throttle{
					Key:    inngestgo.StrPtr("event.data.id"),
					Limit:  1,
					Period: 3 * time.Second,
				},
			},
			inngestgo.EventTrigger(trigger, nil),
			func(ctx context.Context, input inngestgo.Input[inngestgo.GenericEvent[any, any]]) (any, error) {
				// Add two steps to ensure steps aren't throttled.
				_, _ = step.Run(ctx, "b", func(ctx context.Context) (any, error) {
					return nil, nil
				})
				_, _ = step.Run(ctx, "b", func(ctx context.Context) (any, error) {
					return nil, nil
				})
				fmt.Println("Throttled function hit: ", input.Event.Data)
				atomic.AddInt32(&total, 1)
				return true, nil
			},
		)

		h.Register(a)
		registerFuncs()

		for i := 0; i < 3; i++ {
			go func(i int) {
				_, err := inngestgo.Send(context.Background(), inngestgo.Event{
					Name: trigger,
					Data: map[string]any{"id": i},
				})
				require.NoError(t, err)
			}(i)
		}

		// Wait for the first function to finish, but not long enough for the second function
		// to start.
		<-time.After(2 * time.Second)

		// Ensure that each function finishes after 3 seconds.
		require.EqualValues(t, 3, total)
	})
}
