package golang

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
)

func TestEventIdempotency(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	h, server, registerFuncs := NewSDKHandler(t, "test")
	defer server.Close()

	var counter int32
	h.Register(inngestgo.CreateFunction(
		inngestgo.FunctionOpts{Name: "test"},
		inngestgo.EventTrigger("test", nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			atomic.AddInt32(&counter, 1)
			return nil, nil
		},
	))
	registerFuncs()

	sendEvent := func(id string) {
		_, err := inngestgo.Send(ctx, inngestgo.GenericEvent[any, any]{
			ID:   &id,
			Name: "test",
		})
		r.NoError(err)
	}

	t.Run("same ID", func(t *testing.T) {
		// Only 1 run if multiple events have the same ID

		sendEvent("abc")
		sendEvent("abc")

		r.Eventually(func() bool {
			return atomic.LoadInt32(&counter) == 1
		}, 2*time.Second, time.Second)

		// Wait a little longer to make sure no more runs happen
		<-time.After(100 * time.Millisecond)

		r.Equal(int32(1), atomic.LoadInt32(&counter))
	})

	t.Run("different IDs", func(t *testing.T) {
		// Multiple runs if each event has a different ID

		sendEvent("abc")
		sendEvent("def")

		r.Eventually(func() bool {
			return atomic.LoadInt32(&counter) == 2
		}, 2*time.Second, time.Second)
	})
}
