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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type DebounceEventData struct {
	Counter int    `json:"counter"`
	Name    string `json:"name"`
}

type DebounceEvent = inngestgo.GenericEvent[DebounceEventData]

func TestDebounceWithSingleKey(t *testing.T) {
	inngestClient, server, registerFuncs := NewSDKHandler(t, "debounce")
	defer server.Close()

	var (
		counter    int32
		calledWith inngestgo.GenericEvent[DebounceEventData]
	)

	period := 5 * time.Second

	at := time.Now()
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "test-sdk",
			Debounce: &inngestgo.ConfigDebounce{
				Key:    "event.data.name",
				Period: period,
			},
		},
		inngestgo.EventTrigger("test/sdk", nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEventData]) (any, error) {
			// We expect that this function is called after at least the debounce period
			// of 5 seconds.
			now := time.Now()
			// NOTE: Debounces are enqueued with 2s of buffer after the last event.
			delta := now.Sub(at)

			require.True(
				t,
				delta >= period,
				"Expected %s, got %s",
				at.Add(period),
				now,
			)

			if atomic.LoadInt32(&counter) == 0 {
				calledWith = input.Event
			}

			name, err := step.Run(ctx, "get name", func(ctx context.Context) (string, error) {
				fmt.Printf("running debounce fn after %s: %s\n", delta, time.Now().Format(time.RFC3339Nano))
				return input.Event.Data.Name, nil
			})
			require.NoError(t, err)

			atomic.AddInt32(&counter, 1)

			return name, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	sendEvent := func(i int) {
		_, err := inngestClient.Send(context.Background(), DebounceEvent{
			Name: "test/sdk",
			Data: DebounceEventData{
				Counter: i,
				Name:    "debounce",
			},
		})
		// Update the last sent time.
		at = time.Now()
		require.NoError(t, err)
	}

	t.Run("It debounces the first function call", func(t *testing.T) {
		// Send any number of events until 1 second before the debounce period.
		cap := time.Now().Add(period).Add(-1 * time.Second)
		i := 0

		for time.Now().Before(cap) {
			i++
			sendEvent(i)
			<-time.After(time.Duration(rand.Int31n(100)) * time.Millisecond)
		}

		fmt.Printf("sent %d debounce events\n", i)

		// Send one more.
		// We have up to 900ms before the debounce period is done;  wait for 400ms
		<-time.After(400 * time.Millisecond)
		sendEvent(999)

		fmt.Printf("waiting for debounce fn: %s\n", time.Now().Format(time.RFC3339Nano))

		require.Eventually(t, func() bool {
			return atomic.LoadInt32(&counter) == 1
		}, period*2, 50*time.Millisecond, time.Now())
		require.EqualValues(t, DebounceEventData{Counter: 999, Name: "debounce"}, calledWith.Data)
	})

	<-time.After(period + time.Second)
	require.EqualValues(t, 1, counter)

	t.Run("It runs the function a second time", func(t *testing.T) {
		sendEvent(1)
		require.Eventually(t, func() bool {
			return atomic.LoadInt32(&counter) == 2
		}, period*2, time.Second)
	})
}

// TestDebounecWithMultipleKeys
func TestDebounecWithMultipleKeys(t *testing.T) {
	inngestClient, server, registerFuncs := NewSDKHandler(t, "debounce")
	defer server.Close()

	var counter int32

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "test-sdk",
			Debounce: &inngestgo.ConfigDebounce{
				Key:    "event.data.name",
				Period: 5 * time.Second,
			},
		},
		inngestgo.EventTrigger("test/sdk", nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			fmt.Println("Debounced function ran", input.Event.Data.Name)
			atomic.AddInt32(&counter, 1)
			return nil, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	n := 5
	for i := 0; i < n; i++ {
		_, err := inngestClient.Send(context.Background(), DebounceEvent{
			Name: "test/sdk",
			Data: DebounceEventData{
				Counter: i,
				Name:    fmt.Sprintf("debounce %d", i),
			},
		})
		require.NoError(t, err)
	}

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&counter) == int32(n)
	}, 10*time.Second, 100*time.Millisecond, "Expected %d, got %d", n, counter)
}

func TestDebounce_OutOfOrderTS(t *testing.T) {
	inngestClient, server, registerFuncs := NewSDKHandler(t, "debounce")
	defer server.Close()

	var counter int32

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "test-out-of-order-debounce-ignored",
			Debounce: &inngestgo.ConfigDebounce{
				Period: 5 * time.Second,
			},
		},
		inngestgo.EventTrigger("test/sdk", nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			fmt.Println("Debounced function ran", input.Event.Data.Name)
			require.Equal(t, "future", input.Event.Data.Name)
			atomic.AddInt32(&counter, 1)
			return nil, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	now := time.Now()
	in_2_s := now.Add(time.Second * 2)

	_, err = inngestClient.Send(context.Background(), DebounceEvent{
		Name: "test/sdk",
		Data: DebounceEventData{
			Name: "future",
		},
		Timestamp: in_2_s.UnixMilli(),
	})
	require.NoError(t, err)

	_, err = inngestClient.Send(context.Background(), DebounceEvent{
		Name: "test/sdk",
		Data: DebounceEventData{
			Name: "now",
		},
		Timestamp: now.UnixMilli(),
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&counter) == 1
	}, 10*time.Second, 100*time.Millisecond, "Expected 1, got %d", counter)
}

func TestDebounce_Timeout(t *testing.T) {
	r := require.New(t)
	inngestClient, server, registerFuncs := NewSDKHandler(t, "debounce")
	defer server.Close()

	start := time.Now()
	period := 5 * time.Second
	max := 10 * time.Second

	var runTime time.Time
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "test-out-of-order-debounce-ignored",
			Debounce: &inngestgo.ConfigDebounce{
				Period:  period,
				Timeout: &max,
			},
		},
		inngestgo.EventTrigger("test/sdk", nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			runTime = time.Now()
			return nil, nil
		},
	)
	r.NoError(err)
	registerFuncs()

	sendEventCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		// Send an events for 20 seconds in a goroutine.
		// This ensures that we wait up to 15s - just past the max - to receive
		// a fn invocation.
		for time.Since(start) < 20*time.Second {
			select {
			case <-sendEventCtx.Done():
				return
			default:
				_, _ = inngestClient.Send(sendEventCtx, DebounceEvent{
					Name: "test/sdk",
					Data: DebounceEventData{
						Name: "debounce",
					},
				})

				// Send more than 1 per second due to a bug we fixed. We had a
				// bug where a debounce was extended past the timeout if an
				// event was received within 1 second before the timeout. This
				// could happen indefinitely if events kept coming in within 1
				// second
				time.Sleep(500 * time.Millisecond)
			}
		}
	}()

	r.EventuallyWithT(func(t *assert.CollectT) {
		r := require.New(t)
		r.NotZero(runTime)

		// Run started within 1 second of the timeout
		timeout := time.Duration(max).Seconds()
		runStarted := runTime.Sub(start).Seconds()
		r.LessOrEqual(
			runStarted,
			timeout+1,
		)
		r.GreaterOrEqual(
			runStarted,
			timeout-1,
		)
	}, 15*time.Second, 100*time.Millisecond)
}
