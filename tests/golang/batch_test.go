package golang

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
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

func TestBatchInvoke(t *testing.T) {
	ctx := context.Background()
	h, server, registerFuncs := NewSDKHandler(t, "batchinvoke")
	defer server.Close()

	var (
		counter       int32
		totalEvents   int32
		invokeCounter int32
	)

	batcher := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			ID:   "batcher",
			Name: "test batching",
			BatchEvents: &inngest.EventBatchConfig{
				MaxSize: 3,
				Timeout: "5s",
			},
		},
		inngestgo.EventTrigger("batchinvoke/batch", nil),
		func(ctx context.Context, input inngestgo.Input[BatchEvent]) (any, error) {
			evts := input.Events
			atomic.AddInt32(&counter, 1)
			atomic.AddInt32(&totalEvents, int32(len(evts)))
			return fmt.Sprintf("batched %d events", len(evts)), nil
		},
	)
	caller := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			ID:   "caller",
			Name: "test batching",
		},
		inngestgo.EventTrigger("batchinvoke/caller", nil),
		func(ctx context.Context, input inngestgo.Input[BatchEvent]) (any, error) {
			_, _ = step.Run(ctx, "print", func(ctx context.Context) (any, error) {
				fmt.Println("invoking batched fn")
				return nil, nil
			})
			val, err := step.Invoke[any](ctx, "batch", step.InvokeOpts{
				FunctionId: "batchinvoke-batcher",
				Data: map[string]any{
					"name": "invoke",
				},
			})
			fmt.Println("Invoked batch fn:", val)
			atomic.AddInt32(&invokeCounter, 1)
			require.NoError(t, err)
			return true, nil
		},
	)

	h.Register(batcher, caller)
	registerFuncs()

	t.Run("trigger a batch", func(t *testing.T) {
		// Call invoke twice
		for i := 0; i < 5; i++ {
			_, err := inngestgo.Send(ctx, BatchEvent{
				Name: "batchinvoke/caller",
				Data: BatchEventData{Time: time.Now()},
			})
			require.NoError(t, err)
		}

		// First trigger should be because of batch is full
		<-time.After(2 * time.Second)
		require.EqualValues(t, 1, atomic.LoadInt32(&counter))
		require.EqualValues(t, 3, atomic.LoadInt32(&totalEvents))
		require.EqualValues(t, 3, atomic.LoadInt32(&invokeCounter))

		// Second trigger should be because of the batch timeout
		<-time.After(5 * time.Second)
		require.EqualValues(t, 2, atomic.LoadInt32(&counter))
		require.EqualValues(t, 5, atomic.LoadInt32(&totalEvents))
		require.EqualValues(t, 5, atomic.LoadInt32(&invokeCounter))
	})
}

func TestBatchEventsWithKeys(t *testing.T) {
	type BatchEventDataWithUserId struct {
		Time   time.Time `json:"time"`
		UserId string    `json:"userId"`
	}
	type BatchEventWithKey = inngestgo.GenericEvent[BatchEventDataWithUserId, any]

	ctx := context.Background()
	h, server, registerFuncs := NewSDKHandler(t, "user-notifications")
	defer server.Close()

	var (
		totalEvents int32
	)

	batchInvokedCounter := make(map[string]int32)
	batchEventsCounter := make(map[string]int)
	mut := sync.Mutex{}

	batchKey := "event.data.userId"

	a := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{Name: "batch test", BatchEvents: &inngest.EventBatchConfig{MaxSize: 3, Timeout: "5s", Key: &batchKey}},
		inngestgo.EventTrigger("test/notification.send", nil),
		func(ctx context.Context, input inngestgo.Input[BatchEventWithKey]) (any, error) {
			mut.Lock()
			batchInvokedCounter[input.Events[0].Data.UserId] += 1
			batchEventsCounter[input.Events[0].Data.UserId] += len(input.Events)
			mut.Unlock()
			atomic.AddInt32(&totalEvents, int32(len(input.Events)))
			return true, nil
		},
	)
	h.Register(a)
	registerFuncs()
	// calls pkg/devserver/api.go:256
	// wrong type in pkg/sdk/function.go:11

	t.Run("trigger batch", func(t *testing.T) {
		sequence := []string{"a", "b", "c", "a", "b", "c", "a", "b"}
		for _, userId := range sequence {
			_, err := inngestgo.Send(ctx, BatchEventWithKey{
				Name: "test/notification.send",
				Data: BatchEventDataWithUserId{Time: time.Now(), UserId: userId},
			})
			require.NoError(t, err)
		}

		// First trigger should be because of batch is full
		<-time.After(2 * time.Second)
		require.EqualValues(t, 1, batchInvokedCounter["a"])
		require.EqualValues(t, 3, batchEventsCounter["a"])
		require.EqualValues(t, 1, batchInvokedCounter["b"])
		require.EqualValues(t, 3, batchEventsCounter["b"])
		require.EqualValues(t, 0, batchInvokedCounter["c"])
		require.EqualValues(t, 0, batchEventsCounter["c"])

		require.EqualValues(t, 6, atomic.LoadInt32(&totalEvents))

		<-time.After(5 * time.Second)
		require.EqualValues(t, 1, batchInvokedCounter["a"])
		require.EqualValues(t, 3, batchEventsCounter["a"])
		require.EqualValues(t, 1, batchInvokedCounter["b"])
		require.EqualValues(t, 3, batchEventsCounter["b"])
		require.EqualValues(t, 1, batchInvokedCounter["c"])
		require.EqualValues(t, 2, batchEventsCounter["c"])

		require.EqualValues(t, 8, atomic.LoadInt32(&totalEvents))
	})
}
