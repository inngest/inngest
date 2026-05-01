package golang

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/tests/client"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
)

func TestConcurrency_ScopeAccount(t *testing.T) {
	c := client.New(t)
	c.ResetAll(t)

	inngestClient, server, registerFuncs := NewSDKHandler(t, "concurrency")
	defer server.Close()

	var (
		inProgress, total, completed int32

		numEvents  = 3
		fnDuration = 2
	)

	trigger := "test/concurrency-acct"

	handler := func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
		fmt.Println("Running func", *input.Event.ID, input.Event.Data, time.Now().Format(time.RFC3339))
		atomic.AddInt32(&total, 1)

		next := atomic.AddInt32(&inProgress, 1)
		// We should never exceed more than one function running
		require.Less(t, next, int32(2))
		<-time.After(time.Duration(fnDuration) * time.Second)
		atomic.AddInt32(&inProgress, -1)
		atomic.AddInt32(&completed, 1)
		return true, nil
	}

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "acct-concurrency",
			Concurrency: &inngestgo.ConfigConcurrency{
				Step: []inngestgo.ConfigStepConcurrency{
					{
						Limit: 1,
						Scope: enums.ConcurrencyScopeAccount,
						Key:   inngestgo.StrPtr("'global'"),
					},
				},
			},
		},
		inngestgo.EventTrigger(trigger, nil),
		handler,
	)
	require.NoError(t, err)
	_, err = inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "acct-concurrency-v2",
			Concurrency: &inngestgo.ConfigConcurrency{
				Step: []inngestgo.ConfigStepConcurrency{
					{
						Limit: 1,
						Scope: enums.ConcurrencyScopeAccount,
						Key:   inngestgo.StrPtr("'global'"),
					},
				},
			},
		},
		inngestgo.EventTrigger(trigger, nil),
		handler,
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

	deadline := time.Now().Add(time.Duration(numEvents*2*fnDuration)*time.Second + queue.PartitionConcurrencyLimitRequeueExtension*time.Duration(numEvents*2) + 5*time.Second)
	for time.Now().Before(deadline) {
		require.LessOrEqual(t, atomic.LoadInt32(&inProgress), int32(1))
		if atomic.LoadInt32(&completed) == int32(numEvents*2) {
			break
		}
		<-time.After(200 * time.Millisecond)
	}

	require.EqualValues(t, numEvents*2, atomic.LoadInt32(&total))
	require.EqualValues(t, numEvents*2, atomic.LoadInt32(&completed))
}
