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

type DebounceEventData struct {
	Counter int    `json:"counter"`
	Name    string `json:"name"`
}

type DebounceEvent = inngestgo.GenericEvent[DebounceEventData, any]

func TestDebounceWithSingleKey(t *testing.T) {
	h, server, registerFuncs := NewSDKHandler(t)
	defer server.Close()

	var (
		counter    int32
		calledWith DebounceEvent
	)

	period := 5 * time.Second

	at := time.Now()
	a := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "test sdk",
			Debounce: &inngestgo.Debounce{
				Key:    "event.data.name",
				Period: period,
			},
		},
		inngestgo.EventTrigger("test/sdk", nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
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

			name := step.Run(ctx, "get name", func(ctx context.Context) (string, error) {
				fmt.Printf("running debounce fn after %s: %s\n", delta, time.Now().Format(time.RFC3339Nano))
				return input.Event.Data.Name, nil
			})

			atomic.AddInt32(&counter, 1)

			return name, nil
		},
	)
	h.Register(a)
	registerFuncs()

	sendEvent := func(i int) {
		_, err := inngestgo.Send(context.Background(), DebounceEvent{
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
	h, server, registerFuncs := NewSDKHandler(t)
	defer server.Close()

	var counter int32

	a := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "test sdk",
			Debounce: &inngestgo.Debounce{
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
	h.Register(a)
	registerFuncs()

	n := 5
	for i := 0; i < n; i++ {
		_, err := inngestgo.Send(context.Background(), DebounceEvent{
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
