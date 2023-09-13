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

func TestDebounce(t *testing.T) {
	h, server, registerFuncs := NewSDKHandler(t)
	defer server.Close()

	var (
		counter    int32
		calledWith DebounceEvent
	)

	at := time.Now()
	a := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "test sdk",
			Debounce: &inngestgo.Debounce{
				Key:    "event.data.name",
				Period: 10 * time.Second,
			},
		},
		inngestgo.EventTrigger("test/sdk"),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {

			// We expect that this function is called after at least the debounce period
			// of 5 seconds.
			now := time.Now()
			require.True(
				t,
				now.After(at.Add(10*time.Second)),
				"Expected %s, got %s",
				at.Add(10*time.Second),
				now,
			)

			if atomic.LoadInt32(&counter) == 0 {
				calledWith = input.Event
			}

			name := step.Run(ctx, "get name", func(ctx context.Context) (string, error) {
				fmt.Println("Running function")
				return input.Event.Data.Name, nil
			})

			atomic.AddInt32(&counter, 1)

			return name, nil
		},
	)
	h.Register(a)
	registerFuncs()

	for i := 0; i < 5; i++ {
		_, err := inngestgo.Send(context.Background(), DebounceEvent{
			Name: "test/sdk",
			Data: DebounceEventData{
				Counter: i,
				Name:    "debounce",
			},
		})
		require.NoError(t, err)

		i := rand.Int31n(1000)
		<-time.After(time.Duration(i) * time.Millisecond)
	}

	<-time.After(8 * time.Second)
	at = time.Now()
	// Send one more.
	_, err := inngestgo.Send(context.Background(), DebounceEvent{
		Name: "test/sdk",
		Data: DebounceEventData{
			Counter: 999,
			Name:    "debounce",
		},
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&counter) == 1
	}, 17*time.Second, time.Second)
	require.EqualValues(t, DebounceEventData{Counter: 999, Name: "debounce"}, calledWith.Data)

	<-time.After(4 * time.Second)
	require.EqualValues(t, 1, counter)
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
		inngestgo.EventTrigger("test/sdk"),
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
