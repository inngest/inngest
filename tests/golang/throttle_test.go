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
		inngestClient, server, registerFuncs := NewSDKHandler(t, "concurrency")
		defer server.Close()

		trigger := "test/timeouts-start"

		funcs := 5
		throttlePeriod := 5 * time.Second

		runs := map[string]struct{}{}
		startTimes := []time.Time{}

		_, err := inngestgo.CreateFunction(
			inngestClient,
			inngestgo.FunctionOpts{
				ID: "throttle-test",
				Throttle: &inngestgo.Throttle{
					Limit:  1,
					Period: throttlePeriod,
				},
			},
			inngestgo.EventTrigger(trigger, nil),
			func(ctx context.Context, input inngestgo.Input[inngestgo.GenericEvent[any, any]]) (any, error) {
				fmt.Println(time.Now().Format(time.RFC3339))
				if _, ok := runs[input.InputCtx.RunID]; !ok {
					startTimes = append(startTimes, time.Now())
					runs[input.InputCtx.RunID] = struct{}{}
				}
				return true, nil
			},
		)
		require.NoError(t, err)
		registerFuncs()

		var events []any
		for i := 0; i < funcs; i++ {
			events = append(events, inngestgo.Event{
				Name: trigger,
				Data: map[string]any{"test": true},
			})
		}
		_, err = inngestClient.SendMany(context.Background(), events)
		require.NoError(t, err)

		// Wait for all functions to run
		require.Eventually(t,
			func() bool {
				return len(startTimes) == funcs
			},

			// Add a little extra time to ensure all functions have run
			time.Duration(funcs+1)*5*time.Second,

			time.Second,
		)

		for i := 0; i < funcs; i++ {
			fmt.Println(startTimes[i].Format(time.RFC3339))
			if i == 0 {
				continue
			}

			sincePreviousStart := startTimes[i].Sub(startTimes[i-1])

			// Sometimes runs start a little before the throttle period, but
			// shouldn't be more than 1 second before
			require.GreaterOrEqual(t, sincePreviousStart, throttlePeriod-1*time.Second)

			// Sometimes runs start a little after the throttle period. This
			// fudge factor can be increased if we see it fail in CI
			require.LessOrEqual(t, sincePreviousStart, throttlePeriod+2*time.Second)
		}
	})

	t.Run("Throttling with keys separates values", func(t *testing.T) {
		inngestClient, server, registerFuncs := NewSDKHandler(t, "concurrency")
		defer server.Close()

		trigger := "test/timeouts-start-key"

		var (
			total int32
		)

		_, err := inngestgo.CreateFunction(
			inngestClient,
			inngestgo.FunctionOpts{
				ID: "throttle-test-with-keys",
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
		require.NoError(t, err)
		registerFuncs()

		for i := 0; i < 3; i++ {
			go func(i int) {
				_, err = inngestClient.Send(context.Background(), inngestgo.Event{
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
