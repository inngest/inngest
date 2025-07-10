package golang

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/tests/client"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
)

func TestConcurrency_ScopeFunction(t *testing.T) {
	c := client.New(t)
	c.ResetAll(t)

	inngestClient, server, registerFuncs := NewSDKHandler(t, "concurrency")
	defer server.Close()

	var (
		inProgress, total int32

		numEvents  = 3
		fnDuration = 5
	)

	trigger := "test/concurrency-fn"

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "fn-concurrency",
			Concurrency: []inngestgo.ConfigStepConcurrency{
				{
					Limit: 1,
				},
			},
		},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			fmt.Println("Running func", *input.Event.ID, input.Event.Data)
			atomic.AddInt32(&total, 1)

			next := atomic.AddInt32(&inProgress, 1)

			// We should never exceed more than one function running
			require.Less(t, next, int32(2))

			<-time.After(time.Duration(fnDuration) * time.Second)

			atomic.AddInt32(&inProgress, -1)
			return true, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	for i := 0; i < numEvents; i++ {
		go func() {
			_, err := inngestClient.Send(context.Background(), inngestgo.Event{
				Name: trigger,
				Data: map[string]any{
					"test": true,
				},
			})
			require.NoError(t, err)
		}()
	}

	// Eventually the fn starts.
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&inProgress) == 1
	}, 2*time.Second, 50*time.Millisecond)

	for i := 0; i < (numEvents*fnDuration)+1; i++ {
		<-time.After(time.Second)
		require.LessOrEqual(t, atomic.LoadInt32(&inProgress), int32(1))
	}

	// Eventually, within 2 seconds of waiting after the total function duration,
	// all tests have started.
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&total) == 3
	}, 2*time.Second, 50*time.Millisecond)
}

// TestConcurrency_ScopeFunction_FanOut tests function limits with two functions,
// both of which should run.
func TestConcurrency_ScopeFunction_FanOut(t *testing.T) {
	inngestClient, server, registerFuncs := NewSDKHandler(t, "concurrency")
	defer server.Close()

	var (
		inProgressA, totalA int32
		inProgressB, totalB int32

		numEvents  = 3
		fnDuration = 5
	)

	trigger := "test/concurrency-acct"

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "acct-concurrency",
			Concurrency: []inngestgo.ConfigStepConcurrency{
				{
					Limit: 1,
					Scope: enums.ConcurrencyScopeFn,
				},
			},
		},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			fmt.Println("Running func", *input.Event.ID, input.Event.Data)
			atomic.AddInt32(&totalA, 1)
			next := atomic.AddInt32(&inProgressA, 1)
			require.Less(t, next, int32(2))
			<-time.After(time.Duration(fnDuration) * time.Second)
			atomic.AddInt32(&inProgressA, -1)
			return true, nil
		},
	)
	require.NoError(t, err)
	_, err = inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "acct-concurrency-v2",
			Concurrency: []inngestgo.ConfigStepConcurrency{
				{
					Limit: 1,
					Scope: enums.ConcurrencyScopeFn,
				},
			},
		},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			fmt.Println("Running func", *input.Event.ID, input.Event.Data)
			atomic.AddInt32(&totalB, 1)
			next := atomic.AddInt32(&inProgressB, 1)
			require.Less(t, next, int32(2))
			<-time.After(time.Duration(fnDuration) * time.Second)
			atomic.AddInt32(&inProgressB, -1)
			return true, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	for i := 0; i < numEvents; i++ {
		go func() {
			_, err := inngestClient.Send(context.Background(), inngestgo.Event{
				Name: trigger,
				Data: map[string]any{
					"test": true,
				},
			})
			require.NoError(t, err)
		}()
	}

	// Eventually the fn starts.
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&inProgressA) == 1 && atomic.LoadInt32(&inProgressB) == 1
	}, 3*time.Second, 50*time.Millisecond)

	for i := 0; i < (numEvents*fnDuration)+1; i++ {
		<-time.After(time.Second)
		require.LessOrEqual(t, atomic.LoadInt32(&inProgressA), int32(1))
		require.LessOrEqual(t, atomic.LoadInt32(&inProgressB), int32(1))
	}

	<-time.After(time.Second)
	require.EqualValues(t, 3, atomic.LoadInt32(&totalA))
	require.EqualValues(t, 3, atomic.LoadInt32(&totalB))
}

// TestConcurrency_ScopeFunction_Key asserts that keys in function concurrency work as expected.
func TestConcurrency_ScopeFunction_Key(t *testing.T) {
	inngestClient, server, registerFuncs := NewSDKHandler(t, "concurrency")
	defer server.Close()

	var (
		inProgress, total int32

		numEvents  = 3
		fnDuration = 5
	)

	trigger := "test/concurrency-key-fn"

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "fn-concurrency",
			Concurrency: []inngestgo.ConfigStepConcurrency{
				{
					Limit: 1,
					Key:   inngestgo.StrPtr("event.data.num"),
				},
			},
		},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			fmt.Println("Running func", *input.Event.ID, input.Event.Data)
			atomic.AddInt32(&total, 1)

			next := atomic.AddInt32(&inProgress, 1)
			// We should never exceed more than three events running
			require.LessOrEqual(t, next, int32(numEvents))

			<-time.After(time.Duration(fnDuration) * time.Second)

			atomic.AddInt32(&inProgress, -1)
			return true, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	// Send events 0, 1, 2.  Because the "num" is a concurrency key and 0 is already
	// running, it should not run immediately.
	for i := 0; i < numEvents; i++ {
		int := i
		go func() {
			_, err := inngestClient.Send(context.Background(), inngestgo.Event{
				Name: trigger,
				Data: map[string]any{
					"num": int,
				},
			})
			require.NoError(t, err)
		}()
	}

	// Send a dupe 0 event.
	go func() {
		_, err := inngestClient.Send(context.Background(), inngestgo.Event{
			Name: trigger,
			Data: map[string]any{
				"num": 0,
			},
		})
		require.NoError(t, err)
	}()

	// Eventually the fn starts.
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&inProgress) == 3
	}, 2*time.Second, 50*time.Millisecond)

	expectedTime := time.Duration((numEvents+1)*fnDuration) * time.Second

	for i := int64(0); i < expectedTime.Milliseconds(); i++ {
		<-time.After(time.Millisecond)
		// We should never have more than 3 functions running.
		require.LessOrEqual(t, atomic.LoadInt32(&inProgress), int32(numEvents))
	}

	require.EqualValues(t, numEvents+1, atomic.LoadInt32(&total))
}

// TestConcurrency_ScopeFunction_Key_Fn asserts that keys in function concurrency work as expected,
// when mixed with function concurrency overall.
func TestConcurrency_ScopeFunction_Key_Fn(t *testing.T) {
	inngestClient, server, registerFuncs := NewSDKHandler(t, "concurrency")
	defer server.Close()

	var (
		inProgress, total int32

		limit = 2

		numEvents  = 3
		fnDuration = 5
	)

	trigger := "test/concurrency-key-fn-mixed"

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "multiple-fn-concurrency",
			Concurrency: []inngestgo.ConfigStepConcurrency{
				{
					Limit: limit,
				},
				{
					Limit: 1,
					Key:   inngestgo.StrPtr("event.data.num"),
				},
			},
			Retries: inngestgo.IntPtr(0),
		},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			fmt.Println("Running func", *input.Event.ID, input.Event.Data)
			atomic.AddInt32(&total, 1)

			next := atomic.AddInt32(&inProgress, 1)
			// We should never exceed more than three events running
			require.LessOrEqual(t, next, int32(limit))

			<-time.After(time.Duration(fnDuration) * time.Second)

			atomic.AddInt32(&inProgress, -1)
			return true, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	// Send events 0, 1, 2.  Because the "num" is a concurrency key and 0 is already
	// running, it should not run immediately.
	for i := 0; i < numEvents; i++ {
		int := i
		go func() {
			_, err := inngestClient.Send(context.Background(), inngestgo.Event{
				Name: trigger,
				Data: map[string]any{
					"num": int,
				},
			})
			require.NoError(t, err)
		}()
	}

	// Send a dupe 0 event.
	go func() {
		_, err := inngestClient.Send(context.Background(), inngestgo.Event{
			Name: trigger,
			Data: map[string]any{
				"num": 0,
			},
		})
		require.NoError(t, err)
	}()

	// Eventually the fn starts.
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&inProgress) == int32(limit)
	}, 3*time.Second, 100*time.Millisecond, "Function didn't start")

	expectedTime := time.Duration((numEvents+1)*fnDuration) * time.Second

	for i := int64(0); i < expectedTime.Milliseconds(); i++ {
		<-time.After(time.Millisecond)
		// We should never have more than 3 functions running.
		require.LessOrEqual(t, atomic.LoadInt32(&inProgress), int32(limit))
	}

	require.EqualValues(t, numEvents+1, atomic.LoadInt32(&total))
}
