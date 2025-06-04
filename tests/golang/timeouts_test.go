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

// TestTimeoutStart ensures that the Timeouts.Start config works correctly.
//
// In this test, each function takes 5 seconds to run, and a concurrency
// of 1. We create functions with a 3 second start timeout.  This means
// that the second function won't start before the start timeout and
// should be cancelled.
func TestTimeoutStart(t *testing.T) {
	inngestClient, server, registerFuncs := NewSDKHandler(t, "concurrency")
	defer server.Close()

	var (
		total      int32
		fnDuration = 5
	)

	trigger := "test/timeouts-start"
	timeoutStart := 3 * time.Second

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:          "fn-concurrency",
			Concurrency: []inngestgo.ConfigStepConcurrency{{Limit: 1}},
			Timeouts: &inngestgo.ConfigTimeouts{
				Start: &timeoutStart,
			},
		},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			fmt.Println("Running func", *input.Event.ID, input.Event.Data)
			atomic.AddInt32(&total, 1)
			<-time.After(time.Duration(fnDuration) * time.Second)
			return true, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	for i := 0; i < 3; i++ {
		go func() {
			_, err := inngestClient.Send(context.Background(), inngestgo.Event{
				Name: trigger,
				Data: map[string]any{
					"test": true,
				},
			})
			require.NoError(t, err)
		}()
	}

	<-time.After(8 * time.Second)
	require.EqualValues(t, 1, total)

	// XXX: Hit API to ensure runs have been cancelled here alongside testing counts.
}

// TestTimeoutFinish ensures that the Timeouts.Finish config works correctly.
func TestTimeoutFinish(t *testing.T) {
	// In this test, a function has two steps which take 2 seconds to run.  The
	// finish timeout is 3 seconds, so the function should be cancelled after the
	// first step.
	t.Run("When steps take too long", func(t *testing.T) {
		inngestClient, server, registerFuncs := NewSDKHandler(t, "concurrency")
		defer server.Close()

		var (
			progressA, progressB, progressC int32
			stepDuration                    = 2
		)

		trigger := "test/timeouts-finish"
		timeoutStart := 1 * time.Second
		timeoutFinish := 3 * time.Second

		_, err := inngestgo.CreateFunction(
			inngestClient,
			inngestgo.FunctionOpts{
				ID: "timeouts-finish",
				Timeouts: &inngestgo.ConfigTimeouts{
					Start:  &timeoutStart,
					Finish: &timeoutFinish,
				},
			},
			inngestgo.EventTrigger(trigger, nil),
			func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
				fmt.Println("Running func", *input.Event.ID, input.Event.Data)

				_, _ = step.Run(ctx, "a", func(ctx context.Context) (any, error) {
					<-time.After(time.Duration(stepDuration) * time.Second)
					atomic.AddInt32(&progressA, 1)
					return nil, nil
				})

				_, _ = step.Run(ctx, "b", func(ctx context.Context) (any, error) {
					<-time.After(time.Duration(stepDuration) * time.Second)
					atomic.AddInt32(&progressB, 1)
					return nil, nil
				})

				_, _ = step.Run(ctx, "c", func(ctx context.Context) (any, error) {
					<-time.After(time.Duration(stepDuration) * time.Second)
					atomic.AddInt32(&progressC, 1)
					return nil, nil
				})

				return true, nil
			},
		)
		require.NoError(t, err)
		registerFuncs()

		for i := 0; i < 3; i++ {
			go func() {
				_, err := inngestClient.Send(context.Background(), inngestgo.Event{
					Name: trigger,
					Data: map[string]any{
						"test": true,
					},
				})
				require.NoError(t, err)
			}()
		}

		<-time.After(8 * time.Second)
		require.EqualValues(t, 3, progressA)
		require.EqualValues(t, 3, progressB)
		require.EqualValues(t, 0, progressC)

		// XXX: Hit API to ensure runs have been cancelled here alongside testing counts.
	})
}
