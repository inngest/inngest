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
				<-time.After(time.Second)
				fmt.Println("1")
				atomic.AddInt32(&counter, 1)
				return input.Event, nil
			})

			step.Run(ctx, "2", func(ctx context.Context) (string, error) {
				<-time.After(time.Second)
				fmt.Println("2")
				atomic.AddInt32(&counter, 1)
				return "test", nil
			})

			step.Sleep(ctx, "delay", 2*time.Second)

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
		},
	})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&counter) == 3
	}, 10*time.Second, time.Second)
}
