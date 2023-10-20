package golang

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
)

func TestConcurrency_ScopeFunction(t *testing.T) {
	h, server, registerFuncs := NewSDKHandler(t)
	defer server.Close()

	var (
		inProgress, total int32

		numEvents  = 3
		fnDuration = 5
	)

	trigger := "test/concurrency-fn"

	a := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "fn concurrency",
			Concurrency: []inngest.Concurrency{
				{
					Limit: 1,
				},
			},
		},
		inngestgo.EventTrigger(trigger),
		func(ctx context.Context, input inngestgo.Input[inngestgo.GenericEvent[any, any]]) (any, error) {
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
	h.Register(a)
	registerFuncs()

	for i := 0; i < numEvents; i++ {
		go func() {
			_, err := inngestgo.Send(context.Background(), inngestgo.Event{
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

	require.EqualValues(t, 3, atomic.LoadInt32(&total))
}

// TestConcurrency_ScopeFunction_FanOut tests function limits with two functions,
// both of which should run.
func TestConcurrency_ScopeFunction_FanOut(t *testing.T) {
	h, server, registerFuncs := NewSDKHandler(t)
	defer server.Close()

	var (
		inProgressA, totalA int32
		inProgressB, totalB int32

		numEvents  = 3
		fnDuration = 5
	)

	trigger := "test/concurrency-acct"

	a := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "acct concurrency",
			Concurrency: []inngest.Concurrency{
				{
					Limit: 1,
					Scope: enums.ConcurrencyScopeFn,
				},
			},
		},
		inngestgo.EventTrigger(trigger),
		func(ctx context.Context, input inngestgo.Input[inngestgo.GenericEvent[any, any]]) (any, error) {
			fmt.Println("Running func", *input.Event.ID, input.Event.Data)
			atomic.AddInt32(&totalA, 1)
			next := atomic.AddInt32(&inProgressA, 1)
			require.Less(t, next, int32(2))
			<-time.After(time.Duration(fnDuration) * time.Second)
			atomic.AddInt32(&inProgressA, -1)
			return true, nil
		},
	)
	b := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "acct concurrency v2",
			Concurrency: []inngest.Concurrency{
				{
					Limit: 1,
					Scope: enums.ConcurrencyScopeFn,
				},
			},
		},
		inngestgo.EventTrigger(trigger),
		func(ctx context.Context, input inngestgo.Input[inngestgo.GenericEvent[any, any]]) (any, error) {
			fmt.Println("Running func", *input.Event.ID, input.Event.Data)
			atomic.AddInt32(&totalB, 1)
			next := atomic.AddInt32(&inProgressB, 1)
			require.Less(t, next, int32(2))
			<-time.After(time.Duration(fnDuration) * time.Second)
			atomic.AddInt32(&inProgressB, -1)
			return true, nil
		},
	)

	h.Register(a, b)
	registerFuncs()

	for i := 0; i < numEvents; i++ {
		go func() {
			_, err := inngestgo.Send(context.Background(), inngestgo.Event{
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
	}, 2*time.Second, 50*time.Millisecond)

	for i := 0; i < (numEvents*fnDuration)+1; i++ {
		<-time.After(time.Second)
		require.LessOrEqual(t, atomic.LoadInt32(&inProgressA), int32(1))
		require.LessOrEqual(t, atomic.LoadInt32(&inProgressB), int32(1))
	}

	require.EqualValues(t, 3, atomic.LoadInt32(&totalA))
	require.EqualValues(t, 3, atomic.LoadInt32(&totalB))
}

// TestConcurrency_ScopeFunction_Key asserts that keys in function concurrency work as expected.
func TestConcurrency_ScopeFunction_Key(t *testing.T) {
	h, server, registerFuncs := NewSDKHandler(t)
	defer server.Close()

	var (
		inProgress, total int32

		numEvents  = 3
		fnDuration = 5
	)

	trigger := "test/concurrency-key-fn"

	a := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "fn concurrency",
			Concurrency: []inngest.Concurrency{
				{
					Limit: 1,
					Key:   inngestgo.StrPtr("event.data.num"),
				},
			},
		},
		inngestgo.EventTrigger(trigger),
		func(ctx context.Context, input inngestgo.Input[inngestgo.GenericEvent[any, any]]) (any, error) {
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
	h.Register(a)
	registerFuncs()

	// Send events 0, 1, 2.  Because the "num" is a concurrency key and 0 is already
	// running, it should not run immediately.
	for i := 0; i < numEvents; i++ {
		int := i
		go func() {
			_, err := inngestgo.Send(context.Background(), inngestgo.Event{
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
		_, err := inngestgo.Send(context.Background(), inngestgo.Event{
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
	h, server, registerFuncs := NewSDKHandler(t)
	defer server.Close()

	var (
		inProgress, total int32

		limit = 2

		numEvents  = 3
		fnDuration = 5
	)

	trigger := "test/concurrency-key-fn-mixed"

	a := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "fn concurrency",
			Concurrency: []inngest.Concurrency{
				{
					Limit: limit,
				},
				{
					Limit: 1,
					Key:   inngestgo.StrPtr("event.data.num"),
				},
			},
		},
		inngestgo.EventTrigger(trigger),
		func(ctx context.Context, input inngestgo.Input[inngestgo.GenericEvent[any, any]]) (any, error) {
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
	h.Register(a)
	registerFuncs()

	// Send events 0, 1, 2.  Because the "num" is a concurrency key and 0 is already
	// running, it should not run immediately.
	for i := 0; i < numEvents; i++ {
		int := i
		go func() {
			_, err := inngestgo.Send(context.Background(), inngestgo.Event{
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
		_, err := inngestgo.Send(context.Background(), inngestgo.Event{
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
	}, 2*time.Second, 50*time.Millisecond)

	expectedTime := time.Duration((numEvents+1)*fnDuration) * time.Second

	for i := int64(0); i < expectedTime.Milliseconds(); i++ {
		<-time.After(time.Millisecond)
		// We should never have more than 3 functions running.
		require.LessOrEqual(t, atomic.LoadInt32(&inProgress), int32(limit))
	}

	require.EqualValues(t, numEvents+1, atomic.LoadInt32(&total))
}
