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

func TestFunctionSteps(t *testing.T) {
	h, server, registerFuncs := NewSDKHandler(t)
	defer server.Close()

	var (
		counter int32
	)

	a := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{Name: "test sdk"},
		inngestgo.EventTrigger("test/sdk", nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			step.Run(ctx, "1", func(ctx context.Context) (any, error) {
				fmt.Println("1")
				atomic.AddInt32(&counter, 1)
				return input.Event, nil
			})

			step.Run(ctx, "2", func(ctx context.Context) (string, error) {
				fmt.Println("2")
				atomic.AddInt32(&counter, 1)
				return "test", nil
			})

			step.Sleep(ctx, "delay", 2*time.Second)

			_, err := step.WaitForEvent[any](ctx, "wait", step.WaitForEventOpts{
				Name:    "step name",
				Event:   "api/new.event",
				Timeout: time.Minute,
			})
			if err == step.ErrEventNotReceived {
				panic("no event found")
			}

			// Wait for an event with an expression
			_, err = step.WaitForEvent[any](ctx, "wait", step.WaitForEventOpts{
				Name:    "step name",
				Event:   "api/new.event",
				If:      inngestgo.StrPtr(`async.data.ok == "yes" && async.data.id == event.data.id`),
				Timeout: time.Minute,
			})
			if err == step.ErrEventNotReceived {
				panic("no event found")
			}

			fmt.Println("3")
			atomic.AddInt32(&counter, 1)
			return true, nil
		},
	)
	h.Register(a)
	registerFuncs()

	_, err := inngestgo.Send(context.Background(), inngestgo.Event{
		Name: "test/sdk",
		Data: map[string]any{
			"test": true,
			"id":   "1",
		},
	})
	require.NoError(t, err)

	<-time.After(3 * time.Second)

	// Send the first event to trigger the wait.
	_, err = inngestgo.Send(context.Background(), inngestgo.Event{
		Name: "api/new.event",
		Data: map[string]any{
			"test": true,
		},
	})
	require.NoError(t, err)

	<-time.After(time.Second)

	// And the second event to trigger the next wait.
	_, err = inngestgo.Send(context.Background(), inngestgo.Event{
		Name: "api/new.event",
		Data: map[string]any{
			"ok": "yes",
			"id": "1",
		},
	})
	require.NoError(t, err)

	<-time.After(time.Second)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&counter) == 3
	}, 15*time.Second, time.Second)
}
