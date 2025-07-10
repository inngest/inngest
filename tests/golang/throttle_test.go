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
	tests := []struct {
		name   string
		limit  uint
		burst  uint
		period time.Duration
		minGap time.Duration
		maxGap time.Duration
	}{
		{
			name:   "single limit",
			limit:  1,
			period: 5 * time.Second,
			minGap: 4 * time.Second,
			maxGap: 7 * time.Second,
		},
		{
			name:   "multiple",
			limit:  2,
			period: 5 * time.Second,
			maxGap: 4 * time.Second,
		},
		{
			name:   "single with burst",
			limit:  1,
			burst:  1,
			period: 5 * time.Second,
			maxGap: 7 * time.Second,
		},
		{
			name:   "multiple with burst",
			limit:  2,
			burst:  1,
			period: 5 * time.Second,
			maxGap: 3500 * time.Millisecond,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inngestClient, server, registerFuncs := NewSDKHandler(t, fmt.Sprintf("throttle %s", test.name))
			defer server.Close()

			trigger := "test/timeouts-start"

			funcs := 5
			runs := map[string]struct{}{}
			startTimes := []time.Time{}

			_, err := inngestgo.CreateFunction(
				inngestClient,
				inngestgo.FunctionOpts{
					ID: "throttle-test",
					Throttle: &inngestgo.ConfigThrottle{
						Limit:  test.limit,
						Period: test.period,
						Burst:  test.burst,
					},
				},
				inngestgo.EventTrigger(trigger, nil),
				func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
					fmt.Println(time.Now().Format(time.StampMilli))
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
			for range funcs {
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
				time.Duration(funcs+1)*test.period,
				time.Second,
			)

			for i := range funcs {
				fmt.Println(startTimes[i].Format(time.RFC3339))
				if i == 0 {
					continue
				}

				sincePreviousStart := startTimes[i].Sub(startTimes[i-1])

				// Truncate to a second so that we ignore ~ms gaps.
				sincePreviousStart = sincePreviousStart.Truncate(time.Second)

				// Sometimes runs start a little before the throttle period, but
				// shouldn't be more than 1 second before
				require.GreaterOrEqual(t, sincePreviousStart, test.minGap)

				// Sometimes runs start a little after the throttle period. This
				// fudge factor can be increased if we see it fail in CI
				require.LessOrEqual(t, sincePreviousStart, test.maxGap)
			}
		})
	}

	t.Run("Throttling with keys separates values", func(t *testing.T) {
		inngestClient, server, registerFuncs := NewSDKHandler(t, "concurrency")
		defer server.Close()

		trigger := "test/timeouts-start-key"

		var total int32

		_, err := inngestgo.CreateFunction(
			inngestClient,
			inngestgo.FunctionOpts{
				ID: "throttle-test-with-keys",
				Throttle: &inngestgo.ConfigThrottle{
					Key:    inngestgo.StrPtr("event.data.id"),
					Limit:  1,
					Period: 3 * time.Second,
				},
			},
			inngestgo.EventTrigger(trigger, nil),
			func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
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

		for i := range 3 {
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
