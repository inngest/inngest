package golang

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type BatchEventData struct {
	Time                   time.Time `json:"time"`
	Num                    int       `json:"num"`
	Promotional            bool      `json:"promotional"`
	NotificationNotMeeting bool      `json:"notificationNotMeeting"`
	NotificationMeeting    bool      `json:"notificationMeeting"`
}

type BatchEvent = inngestgo.GenericEvent[BatchEventData]

func TestBatchEvents(t *testing.T) {
	ctx := context.Background()
	c := client.New(t)
	inngestClient, server, registerFuncs := NewSDKHandler(t, "batch")
	defer server.Close()

	var (
		counter     int32
		totalEvents int32
		runID       string
	)

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "batch-test", BatchEvents: &inngestgo.ConfigBatchEvents{MaxSize: 5, Timeout: 5 * time.Second}},
		inngestgo.EventTrigger("test/batch", nil),
		func(ctx context.Context, input inngestgo.Input[BatchEvent]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}

			atomic.AddInt32(&counter, 1)
			atomic.AddInt32(&totalEvents, int32(len(input.Events)))
			return "batched!!", nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	t.Run("trigger batch", func(t *testing.T) {
		for i := range 8 {
			_, err := inngestClient.Send(ctx, BatchEvent{
				Name: "test/batch",
				Data: BatchEventData{Time: time.Now(), Num: i},
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

	t.Run("trace run should have appropriate data", func(t *testing.T) {
		run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{Status: models.FunctionStatusCompleted})

		require.True(t, run.IsBatch)
		require.NotNil(t, run.BatchCreatedAt)

		require.NotNil(t, run.Trace)
		require.True(t, run.Trace.IsRoot)
		require.Equal(t, models.RunTraceSpanStatusCompleted.String(), run.Trace.Status)
		// output test
		require.NotNil(t, run.Trace.OutputID)
		output := c.RunSpanOutput(ctx, *run.Trace.OutputID)
		c.ExpectSpanOutput(t, "batched!!", output)

		t.Run("trigger", func(t *testing.T) {
			// check trigger
			trigger := c.RunTrigger(ctx, runID)
			assert.NotNil(t, trigger)
			assert.NotNil(t, trigger.EventName)
			assert.Equal(t, "test/batch", *trigger.EventName)
			assert.Equal(t, 5, len(trigger.IDs))
			assert.False(t, trigger.Timestamp.IsZero())
			assert.True(t, trigger.IsBatch)
			assert.NotNil(t, trigger.BatchID)
			assert.Nil(t, trigger.Cron)

			rid := ulid.MustParse(runID)
			assert.True(t, trigger.Timestamp.Before(ulid.Time(rid.Time())))
		})
	})
}

func TestMunsonRepro(t *testing.T) {
	ctx := context.Background()
	inngestClient, server, registerFuncs := NewSDKHandler(t, "batch")
	defer server.Close()

	var (
		counter     int32
		totalEvents int32
		runID       string
	)

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "batch-test", BatchEvents: &inngestgo.ConfigBatchEvents{MaxSize: 10, Timeout: 5 * time.Second}},
		// english translation: only trigger function if all three bool fields are false
		// 	-event.data.promotional
		//  -event.data.notificationNotMeeting
		//  -event.data.notificationMeeting
		inngestgo.EventTrigger("test/batch", inngestgo.StrPtr("!( (has(event.data.promotional) && event.data.promotional) || (has(event.data.notificationNotMeeting) && event.data.notificationNotMeeting) || (has(event.data.notificationMeeting) && event.data.notificationMeeting) )")),
		func(ctx context.Context, input inngestgo.Input[BatchEvent]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}
			atomic.AddInt32(&counter, 1)
			atomic.AddInt32(&totalEvents, int32(len(input.Events)))
			return "batched!!", nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	t.Run("trigger batch", func(t *testing.T) {
		// generate 8 sets of events
		// each set has 8 events with all possible combinations for the three boolean fields.
		//		- Only one of these 8 events will pass the function trigger (false, false, false)
		// each of the 8 sets will have one event that passes the function trigger - so 8/64 events should be valid.
		for i := 0; i < 8; i++ {
			for _, a := range []bool{true, false} {
				for _, b := range []bool{true, false} {
					for _, c := range []bool{true, false} {
						_, err := inngestClient.Send(ctx, BatchEvent{
							Name: "test/batch",
							Data: BatchEventData{Num: i, Promotional: a, NotificationNotMeeting: b, NotificationMeeting: c},
						})
						require.NoError(t, err)
					}
				}
			}
		}

		// Wait for trigger to run
		// after batch timeout s, all 8 events should be scheduled in a single batch (batch limit is 10 events)
		<-time.After(10 * time.Second)
		assert.EqualValues(t, 1, atomic.LoadInt32(&counter))
		assert.EqualValues(t, 8, atomic.LoadInt32(&totalEvents))
	})
}

func TestBatchWithConditionalTrigger(t *testing.T) {
	ctx := context.Background()
	inngestClient, server, registerFuncs := NewSDKHandler(t, "batch")
	defer server.Close()

	var (
		counter     int32
		totalEvents int32
		runID       string
	)

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "batch-test", BatchEvents: &inngestgo.ConfigBatchEvents{MaxSize: 5, Timeout: 5 * time.Second}},
		inngestgo.EventTrigger("test/batch", inngestgo.StrPtr("has(event.data.num) && int(event.data.num) % 2 == 0")),
		func(ctx context.Context, input inngestgo.Input[BatchEvent]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}
			atomic.AddInt32(&counter, 1)
			atomic.AddInt32(&totalEvents, int32(len(input.Events)))
			return "batched!!", nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	t.Run("trigger batch", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			_, err := inngestClient.Send(ctx, BatchEvent{
				Name: "test/batch",
				Data: BatchEventData{Num: i},
			})
			require.NoError(t, err)
		}

		// Wait for trigger to run
		<-time.After(6 * time.Second)
		assert.EqualValues(t, 1, atomic.LoadInt32(&counter))
		assert.EqualValues(t, 5, atomic.LoadInt32(&totalEvents))
	})
}

func TestConditionalBatching(t *testing.T) {
	ctx := context.Background()
	inngestClient, server, registerFuncs := NewSDKHandler(t, "batch")
	defer server.Close()

	var (
		counter     int32
		totalEvents int32
		runID       string
	)

	_, err := inngestgo.CreateFunction(
		inngestClient,
		// only even values of event.data.num are eligible for batching. Other events are scheduled for execution immediately.
		inngestgo.FunctionOpts{ID: "conditional-batch-test", BatchEvents: &inngestgo.ConfigBatchEvents{MaxSize: 5, Timeout: 5 * time.Second, If: inngestgo.StrPtr("int(event.data.num) % 2 == 0")}},
		inngestgo.EventTrigger("test/batch", nil),
		func(ctx context.Context, input inngestgo.Input[BatchEvent]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}
			atomic.AddInt32(&counter, 1)
			atomic.AddInt32(&totalEvents, int32(len(input.Events)))
			return "batched!!", nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	t.Run("trigger batch", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			_, err := inngestClient.Send(ctx, BatchEvent{
				Name: "test/batch",
				Data: BatchEventData{Num: i},
			})
			require.NoError(t, err)
		}

		// Wait for trigger to run
		// Half of the 10 events match the batching expression and are execited as a batch of 5 events.
		// The other 5 events are executed immediately.
		<-time.After(6 * time.Second)
		assert.EqualValues(t, 6, atomic.LoadInt32(&counter))
		assert.EqualValues(t, 10, atomic.LoadInt32(&totalEvents))
	})
}

func TestConditionalBatchingWithEventTriggerCondition(t *testing.T) {
	ctx := context.Background()
	inngestClient, server, registerFuncs := NewSDKHandler(t, "batch")
	defer server.Close()

	var (
		counter     int32
		totalEvents int32
		runID       string
	)

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "conditional-batch-test-with-event-trigger", BatchEvents: &inngestgo.ConfigBatchEvents{MaxSize: 5, Timeout: 5 * time.Second, If: inngestgo.StrPtr("has(event.data.num) && int(event.data.num) % 2 == 0")}},
		inngestgo.EventTrigger("test/batch", inngestgo.StrPtr("event.data.promotional")),
		func(ctx context.Context, input inngestgo.Input[BatchEvent]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}
			atomic.AddInt32(&counter, 1)
			atomic.AddInt32(&totalEvents, int32(len(input.Events)))
			return "batched!!", nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	t.Run("trigger batch", func(t *testing.T) {
		// send 20 events, 10 with promotional=true and 10 with promotional=false.
		for i := 0; i < 10; i++ {
			for _, promotional := range []bool{true, false} {
				_, err := inngestClient.Send(ctx, BatchEvent{
					Name: "test/batch",
					Data: BatchEventData{Num: i, Promotional: promotional},
				})
				require.NoError(t, err)
			}
		}

		// Wait for trigger to run
		// The 10 events with promotional=false do not run as the event trigger is false.
		// Out of the 10 events with promotional=true, the even `num`s are executed in a single batch, and the other 5 are executed immeditaly
		<-time.After(6 * time.Second)
		assert.EqualValues(t, 6, atomic.LoadInt32(&counter))
		assert.EqualValues(t, 10, atomic.LoadInt32(&totalEvents))
	})
}

func TestBatchInvoke(t *testing.T) {
	ctx := context.Background()
	inngestClient, server, registerFuncs := NewSDKHandler(t, "batchinvoke")
	defer server.Close()

	var (
		counter       int32
		totalEvents   int32
		invokeCounter int32
	)

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:   "batcher",
			Name: "test batching",
			BatchEvents: &inngestgo.ConfigBatchEvents{
				MaxSize: 3,
				Timeout: 5 * time.Second,
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
	require.NoError(t, err)

	_, err = inngestgo.CreateFunction(
		inngestClient,
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
	require.NoError(t, err)

	registerFuncs()

	t.Run("trigger a batch", func(t *testing.T) {
		// Call invoke twice
		for i := 0; i < 5; i++ {
			_, err := inngestClient.Send(ctx, BatchEvent{
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
		<-time.After(6 * time.Second)
		require.EqualValues(t, 2, atomic.LoadInt32(&counter))
		require.EqualValues(t, 5, atomic.LoadInt32(&totalEvents))
		require.EqualValues(t, 5, atomic.LoadInt32(&invokeCounter))
	})
}

func TestBatchKeyEvents(t *testing.T) {
	type BatchEventDataWithUserId struct {
		Time   time.Time `json:"time"`
		UserId string    `json:"userId"`
	}
	type BatchEventWithKey = inngestgo.GenericEvent[BatchEventDataWithUserId]

	ctx := context.Background()
	inngestClient, server, registerFuncs := NewSDKHandler(t, "user-notifications")
	defer server.Close()

	var totalEvents int32

	batchInvokedCounter := make(map[string]int32)
	batchEventsCounter := make(map[string]int)
	mut := sync.Mutex{}

	batchKey := "event.data.userId"

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "batch-test", Name: "batch test", BatchEvents: &inngestgo.ConfigBatchEvents{MaxSize: 3, Timeout: 5 * time.Second, Key: &batchKey}},
		inngestgo.EventTrigger("test/notification.send", nil),
		func(ctx context.Context, input inngestgo.Input[BatchEventDataWithUserId]) (any, error) {
			mut.Lock()
			batchInvokedCounter[input.Events[0].Data.UserId] += 1
			batchEventsCounter[input.Events[0].Data.UserId] += len(input.Events)
			mut.Unlock()
			atomic.AddInt32(&totalEvents, int32(len(input.Events)))
			return true, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	t.Run("trigger batch", func(t *testing.T) {
		sequence := []string{"a", "b", "c", "a", "b", "c", "a", "b"}
		for _, userId := range sequence {
			_, err := inngestClient.Send(ctx, BatchEventWithKey{
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

		require.EqualValues(t, len(sequence), atomic.LoadInt32(&totalEvents))
	})
}
