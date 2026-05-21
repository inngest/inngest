package golang

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/tests/client"

	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
	"github.com/stretchr/testify/require"
)

// TestFnConcurrency tests function-level concurrency via semaphores.
// Unlike step concurrency, the limit is held for the ENTIRE run — across
// all steps. Only one run should execute at a time with limit=1.
func TestFnConcurrency(t *testing.T) {
	c := client.New(t)
	c.ResetAll(t)

	inngestClient, server, registerFuncs := NewSDKHandler(t, "fn-concurrency")
	defer server.Close()

	var (
		inProgress, total int32

		numEvents  = 3
		fnDuration = 3
	)

	trigger := "test/fn-concurrency"

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "fn-concurrency-test",
			Concurrency: &inngestgo.ConfigConcurrency{
				Fn: []inngestgo.ConfigFnConcurrency{
					{
						Limit: 1,
					},
				},
			},
			Retries: inngestgo.IntPtr(0),
		},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			fmt.Println("Running fn concurrency test", *input.Event.ID)

			// Step 1
			_, _ = step.Run(ctx, "step-1", func(ctx context.Context) (any, error) {
				// Do this once.
				next := atomic.AddInt32(&inProgress, 1)
				// With fn concurrency limit=1, we should never have more than 1 run active
				require.Less(t, next, int32(2), "fn concurrency violated: more than 1 run active")

				<-time.After(time.Duration(fnDuration/2) * time.Second)
				return "step-1-done", nil
			})

			// Step 2 — the semaphore should still be held from step 1
			_, _ = step.Run(ctx, "step-2", func(ctx context.Context) (any, error) {
				<-time.After(time.Duration(fnDuration/2) * time.Second)
				return "step-2-done", nil
			})

			atomic.AddInt32(&inProgress, -1)
			atomic.AddInt32(&total, 1)
			return true, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	// Send multiple events to trigger concurrent runs
	for i := 0; i < numEvents; i++ {
		_, err := inngestClient.Send(context.Background(), inngestgo.Event{
			Name: trigger,
			Data: map[string]any{
				"test": true,
			},
		})
		require.NoError(t, err)
		<-time.After(20 * time.Millisecond)
	}

	// Eventually the first fn starts
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&inProgress) == 1
	}, 5*time.Second, 100*time.Millisecond, "function should start")

	// During execution, never exceed limit
	totalDuration := time.Duration(numEvents*fnDuration+5) * time.Second
	deadline := time.Now().Add(totalDuration)
	for time.Now().Before(deadline) {
		<-time.After(200 * time.Millisecond)
		require.LessOrEqual(t, atomic.LoadInt32(&inProgress), int32(1),
			"fn concurrency violated: more than 1 run active")
	}

	// All runs should eventually complete
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&total) == int32(numEvents)
	}, 5*time.Second, 100*time.Millisecond, "all runs should complete")
}

// TestFnConcurrency_Key tests function-level concurrency with expression-based keys.
// Each unique key value gets its own semaphore. Events with the same key share a limit;
// events with different keys run independently.
func TestFnConcurrency_Key(t *testing.T) {
	c := client.New(t)
	c.ResetAll(t)

	inngestClient, server, registerFuncs := NewSDKHandler(t, "fn-concurrency-key")
	defer server.Close()

	var (
		// Track in-progress per key value
		inProgressA, inProgressB int32
		totalA, totalB           int32

		fnDuration = 3
	)

	trigger := "test/fn-concurrency-key"

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "fn-concurrency-key-test",
			Concurrency: &inngestgo.ConfigConcurrency{
				Fn: []inngestgo.ConfigFnConcurrency{
					{
						Limit: 1,
						Key:   inngestgo.StrPtr("event.data.customer_id"),
					},
				},
			},
			Retries: inngestgo.IntPtr(0),
		},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			customerID, _ := input.Event.Data["customer_id"].(string)
			fmt.Printf("Running fn concurrency key test customer=%s event=%s\n", customerID, *input.Event.ID)

			_, _ = step.Run(ctx, "work", func(ctx context.Context) (any, error) {
				switch customerID {
				case "A":
					next := atomic.AddInt32(&inProgressA, 1)
					require.Less(t, next, int32(2), "fn concurrency for customer A violated")
					<-time.After(time.Duration(fnDuration) * time.Second)
					atomic.AddInt32(&inProgressA, -1)
				case "B":
					next := atomic.AddInt32(&inProgressB, 1)
					require.Less(t, next, int32(2), "fn concurrency for customer B violated")
					<-time.After(time.Duration(fnDuration) * time.Second)
					atomic.AddInt32(&inProgressB, -1)
				}
				return "done", nil
			})

			switch customerID {
			case "A":
				atomic.AddInt32(&totalA, 1)
			case "B":
				atomic.AddInt32(&totalB, 1)
			}
			return true, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	// Send 2 events for customer A and 2 for customer B
	for _, cid := range []string{"A", "B", "A", "B"} {
		_, err := inngestClient.Send(context.Background(), inngestgo.Event{
			Name: trigger,
			Data: map[string]any{
				"customer_id": cid,
			},
		})
		require.NoError(t, err)
		<-time.After(20 * time.Millisecond)
	}

	// Both customers should start concurrently (different keys = independent semaphores)
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&inProgressA) == 1 && atomic.LoadInt32(&inProgressB) == 1
	}, 5*time.Second, 100*time.Millisecond, "both customers should start concurrently")

	// During execution, neither customer should exceed limit=1
	totalDuration := time.Duration(2*fnDuration+5) * time.Second
	deadline := time.Now().Add(totalDuration)
	for time.Now().Before(deadline) {
		<-time.After(200 * time.Millisecond)
		require.LessOrEqual(t, atomic.LoadInt32(&inProgressA), int32(1),
			"fn concurrency for customer A violated")
		require.LessOrEqual(t, atomic.LoadInt32(&inProgressB), int32(1),
			"fn concurrency for customer B violated")
	}

	// All runs should eventually complete (2 per customer, serialized within each)
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&totalA) == 2 && atomic.LoadInt32(&totalB) == 2
	}, 5*time.Second, 100*time.Millisecond, "all runs should complete")
}
