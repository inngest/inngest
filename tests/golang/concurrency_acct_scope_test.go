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

func TestConcurrency_ScopeAccount(t *testing.T) {
	h, server, registerFuncs := NewSDKHandler(t)
	defer server.Close()

	var (
		inProgress, total int32

		numEvents  = 3
		fnDuration = 5
	)

	trigger := "test/concurrency-acct"

	handler := func(ctx context.Context, input inngestgo.Input[inngestgo.GenericEvent[any, any]]) (any, error) {
		fmt.Println("Running func", *input.Event.ID, input.Event.Data)
		atomic.AddInt32(&total, 1)

		next := atomic.AddInt32(&inProgress, 1)
		// We should never exceed more than one function running
		require.Less(t, next, int32(2))
		<-time.After(time.Duration(fnDuration) * time.Second)
		atomic.AddInt32(&inProgress, -1)
		return true, nil
	}

	a := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "acct concurrency",
			Concurrency: []inngest.Concurrency{
				{
					Limit: 1,
					Scope: enums.ConcurrencyScopeAccount,
					Key:   inngestgo.StrPtr("'global'"),
				},
			},
		},
		inngestgo.EventTrigger(trigger),
		handler,
	)
	b := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "acct concurrency v2",
			Concurrency: []inngest.Concurrency{
				{
					Limit: 1,
					Scope: enums.ConcurrencyScopeAccount,
					Key:   inngestgo.StrPtr("'global'"),
				},
			},
		},
		inngestgo.EventTrigger(trigger),
		handler,
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
		return atomic.LoadInt32(&inProgress) == 1
	}, 2*time.Second, 50*time.Millisecond)

	for i := 0; i < ((numEvents*2)*fnDuration)+5; i++ {
		<-time.After(time.Second)
		require.LessOrEqual(t, atomic.LoadInt32(&inProgress), int32(1))
	}

	require.EqualValues(t, 6, atomic.LoadInt32(&total))
}
