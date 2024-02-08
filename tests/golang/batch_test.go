package golang

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
)

type BatchEventData struct {
	Time time.Time `json:"time"`
}

type BatchEvent = inngestgo.GenericEvent[BatchEventData, any]

func TestBatchEvents(t *testing.T) {
	ctx := context.Background()
	h, server, registerFuncs := NewSDKHandler(t, "batch")
	defer server.Close()

	var (
		counter     int32
		totalEvents int32
	)

	a := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{Name: "batch test", BatchEvents: &inngest.EventBatchConfig{MaxSize: 5, Timeout: "5s"}},
		inngestgo.EventTrigger("test/batch", nil),
		func(ctx context.Context, input inngestgo.Input[BatchEvent]) (any, error) {
			atomic.AddInt32(&counter, 1)
			atomic.AddInt32(&totalEvents, int32(len(input.Events)))
			return true, nil
		},
	)
	h.Register(a)
	registerFuncs()

	t.Run("trigger batch", func(t *testing.T) {
		for i := 0; i < 8; i++ {
			_, err := inngestgo.Send(ctx, BatchEvent{
				Name: "test/batch",
				Data: BatchEventData{Time: time.Now()},
			})
			require.NoError(t, err)
		}

		// First trigger should be because of batch is full
		<-time.After(2 * time.Second)
		require.EqualValues(t, 1, atomic.LoadInt32(&counter))
		require.EqualValues(t, 5, atomic.LoadInt32(&totalEvents))

		// Second trigger should be because of the batch timeout
		<-time.After(5 * time.Second)
		require.EqualValues(t, 2, atomic.LoadInt32(&counter))
		require.EqualValues(t, 8, atomic.LoadInt32(&totalEvents))
	})
}
