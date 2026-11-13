package golang

import (
	"context"
	"fmt"
	"sync"
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
			Retries: new(0),
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
						Key:   new("event.data.customer_id"),
					},
				},
			},
			Retries: new(0),
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

// TestFnConcurrency_RetriedFirstStep runs a function whose first step fails
// once.  the run keeps its function concurrency slot through the retry, and
// the retry must not take one more.  a retry that waits on the slot of its
// own run never starts, and no run finishes.
func TestFnConcurrency_RetriedFirstStep(t *testing.T) {
	c := client.New(t)
	c.ResetAll(t)

	inngestClient, server, registerFuncs := NewSDKHandler(t, "fn-concurrency-retry")
	defer server.Close()

	var (
		inProgress, total, violations int32
		attempts                      sync.Map

		numEvents = 2
	)

	trigger := "test/fn-concurrency-retry"

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "fn-concurrency-retry-test",
			Concurrency: &inngestgo.ConfigConcurrency{
				Fn: []inngestgo.ConfigFnConcurrency{
					{
						Limit: 1,
					},
				},
			},
			Retries: new(3),
		},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			_, err := step.Run(ctx, "step-1", func(ctx context.Context) (any, error) {
				n, _ := attempts.LoadOrStore(input.InputCtx.RunID, new(int32))
				if atomic.AddInt32(n.(*int32), 1) == 1 {
					return nil, inngestgo.RetryAtError(fmt.Errorf("first attempt fails"), time.Now())
				}

				if atomic.AddInt32(&inProgress, 1) > 1 {
					atomic.AddInt32(&violations, 1)
				}
				<-time.After(time.Second)
				atomic.AddInt32(&inProgress, -1)
				return "step-1-done", nil
			})
			if err != nil {
				return nil, err
			}

			atomic.AddInt32(&total, 1)
			return true, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

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

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&total) == int32(numEvents)
	}, 30*time.Second, 100*time.Millisecond, "every run must finish after its first step retries")
	require.Zero(t, atomic.LoadInt32(&violations), "fn concurrency violated: more than 1 run active")
}

// TestFnConcurrency_CancelWhileQueued cancels a run that waits behind the
// function concurrency limit.  that run never took the slot, so its
// cancellation must not release one.  a release here lets the next run start
// beside the run that is still active.
func TestFnConcurrency_CancelWhileQueued(t *testing.T) {
	c := client.New(t)
	c.ResetAll(t)

	inngestClient, server, registerFuncs := NewSDKHandler(t, "fn-concurrency-cancel")
	defer server.Close()

	var (
		inProgress, violations int32
		started, finished      sync.Map
	)

	trigger := "test/fn-concurrency-cancel"
	cancel := "test/fn-concurrency-cancel.stop"

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "fn-concurrency-cancel-test",
			Concurrency: &inngestgo.ConfigConcurrency{
				Fn: []inngestgo.ConfigFnConcurrency{
					{
						Limit: 1,
					},
				},
			},
			Cancel: []inngestgo.ConfigCancel{
				{Event: cancel, If: new("async.data.id == event.data.id")},
			},
			Retries: new(0),
		},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			id, _ := input.Event.Data["id"].(string)

			_, _ = step.Run(ctx, "work", func(ctx context.Context) (any, error) {
				started.Store(id, true)
				if atomic.AddInt32(&inProgress, 1) > 1 {
					atomic.AddInt32(&violations, 1)
				}
				<-time.After(4 * time.Second)
				atomic.AddInt32(&inProgress, -1)
				return "done", nil
			})

			finished.Store(id, true)
			return true, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	send := func(name, id string) {
		_, err := inngestClient.Send(context.Background(), inngestgo.Event{
			Name: name,
			Data: map[string]any{
				"id": id,
			},
		})
		require.NoError(t, err)
	}
	has := func(m *sync.Map, id string) bool {
		_, ok := m.Load(id)
		return ok
	}

	// run a owns the slot.
	send(trigger, "a")
	require.Eventually(t, func() bool {
		return has(&started, "a")
	}, 5*time.Second, 50*time.Millisecond, "run a should start")

	// run b waits behind a, and is cancelled before it starts.
	send(trigger, "b")
	<-time.After(time.Second)
	send(cancel, "b")
	<-time.After(time.Second)

	// run c must wait for a.
	send(trigger, "c")

	require.Eventually(t, func() bool {
		return has(&finished, "a") && has(&finished, "c")
	}, 20*time.Second, 100*time.Millisecond, "runs a and c should finish")

	require.Zero(t, atomic.LoadInt32(&violations), "fn concurrency violated: run c started beside run a")
	require.False(t, has(&started, "b"), "the cancelled run must not start")
}

// TestFnConcurrency_CancelDuringRetry cancels a run while its first step
// waits for a retry.  the run keeps its function concurrency slot through
// the retry, so the cancellation releases one slot.  a second release lets
// one run more than the limit start.
func TestFnConcurrency_CancelDuringRetry(t *testing.T) {
	c := client.New(t)
	c.ResetAll(t)

	inngestClient, server, registerFuncs := NewSDKHandler(t, "fn-concurrency-cancel-retry")
	defer server.Close()

	var (
		inProgress, violations    int32
		failed, started, finished sync.Map
	)

	trigger := "test/fn-concurrency-cancel-retry"
	cancel := "test/fn-concurrency-cancel-retry.stop"

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "fn-concurrency-cancel-retry-test",
			Concurrency: &inngestgo.ConfigConcurrency{
				Fn: []inngestgo.ConfigFnConcurrency{
					{
						Limit: 2,
					},
				},
			},
			Cancel: []inngestgo.ConfigCancel{
				{Event: cancel, If: new("async.data.id == event.data.id")},
			},
			Retries: new(3),
		},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			id, _ := input.Event.Data["id"].(string)

			_, err := step.Run(ctx, "work", func(ctx context.Context) (any, error) {
				if id == "a" {
					failed.Store(id, true)
					return nil, inngestgo.RetryAtError(fmt.Errorf("run a fails"), time.Now().Add(time.Minute))
				}

				started.Store(id, true)
				if atomic.AddInt32(&inProgress, 1) > 2 {
					atomic.AddInt32(&violations, 1)
				}
				<-time.After(6 * time.Second)
				atomic.AddInt32(&inProgress, -1)
				return "done", nil
			})
			if err != nil {
				return nil, err
			}

			finished.Store(id, true)
			return true, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	send := func(name, id string) {
		_, err := inngestClient.Send(context.Background(), inngestgo.Event{
			Name: name,
			Data: map[string]any{
				"id": id,
			},
		})
		require.NoError(t, err)
	}
	has := func(m *sync.Map, id string) bool {
		_, ok := m.Load(id)
		return ok
	}

	// run x has the first slot for the length of the test.
	send(trigger, "x")
	require.Eventually(t, func() bool {
		return has(&started, "x")
	}, 5*time.Second, 50*time.Millisecond, "run x should start")

	// run a has the second slot, and its first step waits for a retry.
	send(trigger, "a")
	require.Eventually(t, func() bool {
		return has(&failed, "a")
	}, 5*time.Second, 50*time.Millisecond, "run a should fail its first step")

	// run b waits, because run a keeps its slot through the retry.
	send(trigger, "b")
	<-time.After(1500 * time.Millisecond)
	require.False(t, has(&started, "b"), "run b must wait behind run a")

	// the cancellation of run a releases its slot to run b.
	send(cancel, "a")
	require.Eventually(t, func() bool {
		return has(&started, "b")
	}, 5*time.Second, 50*time.Millisecond, "run b should start after run a is cancelled")

	// run c must wait for run x.
	send(trigger, "c")

	require.Eventually(t, func() bool {
		return has(&finished, "x") && has(&finished, "b") && has(&finished, "c")
	}, 30*time.Second, 100*time.Millisecond, "runs x, b and c should finish")

	require.Zero(t, atomic.LoadInt32(&violations), "fn concurrency violated: more than 2 runs active")
}
