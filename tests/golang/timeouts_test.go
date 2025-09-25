package golang

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/tests/client"
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

// TestTimeoutStartEagerCancellation ensures that the Timeouts.Start config works correctly.
//
// In this test, the function has a throttle config of 1 function run every 5 seconds.
// The function has a start timeout of 7 seconds.
// We create 4 function runs. This means that last two function runs should BOTH be cancelled IMMEDIATELY after the timeout.
func TestStartTimeoutEagerCancellation(t *testing.T) {
	ctx := context.Background()
	c := client.New(t)
	inngestClient, server, registerFuncs := NewSDKHandler(t, "eager-cancellation-start")
	defer server.Close()

	trigger := randomSuffix("test/timeouts-start")
	timeoutStart := 7 * time.Second

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "eager-cancellation-start",
			Throttle: &inngestgo.ConfigThrottle{
				Limit:  1,
				Period: 5. * time.Second,
			},
			Timeouts: &inngestgo.ConfigTimeouts{
				Start: &timeoutStart,
			},
		},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			fmt.Println("Running func", *input.Event.ID, input.Event.Data)
			return true, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	for i := 0; i < 4; i++ {
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

	eventsFilter := models.EventsFilter{
		EventNames: []string{trigger},
	}
	res, err := c.GetEvents(ctx, client.GetEventsOpts{
		PageSize: 40,
		Filter:   eventsFilter,
	})
	require.NoError(t, err)
	require.Equal(t, res.TotalCount, 4)
	var runIDs []string
	for _, edge := range res.Edges {
		require.Greater(t, len(edge.Node.Runs), 0)
		runIDs = append(runIDs, edge.Node.Runs[0].ID.String())
	}

	cancelledRuns := 0
	t.Run("trace run should have appropriate data", func(t *testing.T) {
		for _, runID := range runIDs {
			run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{})

			require.NotNil(t, run.Trace)
			if run.Trace.Status == models.RunTraceSpanStatusCancelled.String() {
				cancelledRuns++
			}
		}
		require.Equal(t, cancelledRuns, 2)
	})
}

func TestStartTimeoutEagerCancellationTimeoutRemoved(t *testing.T) {
	require.True(t, false)
}

func TestStartTimeoutEagerCancellationTimeoutIncreased(t *testing.T) {
	require.True(t, false)
}

func TestStartTimeoutEagerCancellationTimeoutDecreased(t *testing.T) {
	require.True(t, false)
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

// TODO: TestTimeoutFinish ensures that the Timeouts.Finish config works correctly.
func TestFinishTimeoutEagerCancellation(t *testing.T) {
	// In this test, a function has two steps which take 10 seconds to run.  The
	// finish timeout is 3 seconds, so the function should be cancelled after the
	// first step.
	t.Run("When steps take too long", func(t *testing.T) {
		inngestClient, server, registerFuncs := NewSDKHandler(t, "concurrency")
		defer server.Close()

		var (
			progressA, progressB, progressC int32
			stepDuration                    = 10
		)

		trigger := randomSuffix("test/timeouts-finish")
		timeoutFinish := 3 * time.Second

		_, err := inngestgo.CreateFunction(
			inngestClient,
			inngestgo.FunctionOpts{
				ID: "timeouts-finish",
				Timeouts: &inngestgo.ConfigTimeouts{
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

		for i := 0; i < 10; i++ {
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

		<-time.After(4 * time.Second)

		eventsFilter := models.EventsFilter{
			EventNames: []string{trigger},
		}
		ctx := context.Background()
		c := client.New(t)
		res, err := c.GetEvents(ctx, client.GetEventsOpts{
			PageSize: 40,
			Filter:   eventsFilter,
		})
		require.NoError(t, err)
		require.Equal(t, res.TotalCount, 10)
		var runIDs []string
		for _, edge := range res.Edges {
			require.Greater(t, len(edge.Node.Runs), 0)
			runIDs = append(runIDs, edge.Node.Runs[0].ID.String())
		}

		cancelledRuns := 0
		t.Run("trace run should have appropriate data", func(t *testing.T) {
			for _, runID := range runIDs {
				run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{})
				require.NotNil(t, run.Trace)

				if run.Trace.Status == models.RunTraceSpanStatusCancelled.String() {
					cancelledRuns++
				}
			}
			require.Equal(t, cancelledRuns, 10)
		})
	})
}

func TestFinishTimeoutEagerCancellationTimeoutRemoved(t *testing.T) {
	require.True(t, false)
}

func TestFinishTimeoutEagerCancellationTimeoutIncreased(t *testing.T) {
	require.True(t, false)
}

func TestFinishTimeoutEagerCancellationTimeoutDecreased(t *testing.T) {
	require.True(t, false)
}
