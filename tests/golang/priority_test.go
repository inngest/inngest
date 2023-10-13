package golang

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngestgo"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestFunctionPriorityRun(t *testing.T) {
	h, server, registerFuncs := NewSDKHandler(t)
	defer server.Close()

	var (
		counter int32
	)

	type result struct {
		runID    ulid.ULID
		priority string
		at       time.Time
	}

	results := make(chan result)

	a := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name:        "Priority.Run test",
			Concurrency: 1,
			Priority: &inngest.Priority{
				Run: inngestgo.StrPtr(`event.data.priority == "high" ? 5 : 0`),
			},
		},
		inngestgo.EventTrigger("test/sdk"),
		func(ctx context.Context, input inngestgo.Input[inngestgo.GenericEvent[map[string]any, any]]) (any, error) {
			priority, _ := input.Event.Data["priority"].(string)
			results <- result{
				runID:    ulid.MustParse(input.InputCtx.RunID),
				at:       time.Now(),
				priority: priority,
			}
			// Wait 5 seconds before finishing, allowing us to simulate queue backlogs.
			<-time.After(5 * time.Second)
			atomic.AddInt32(&counter, 1)
			return true, nil
		},
	)
	h.Register(a)
	registerFuncs()

	go func() {
		n := 0
		for item := range results {
			switch n {
			case 0:
				fmt.Printf("First function done: %s\n", item.priority)
			case 1:
				fmt.Printf("Second function done: %s\n", item.priority)
				require.Equal(t, "high", item.priority)
			case 2:
				fmt.Printf("Final function done: %s\n", item.priority)
				require.Equal(t, "low", item.priority)
			}
			n += 1
		}
	}()

	// For 3 priorities, run 3 events.  The first priority, "none", should run without blocking.
	priorities := []string{"none", "low", "high"}
	for _, p := range priorities {
		_, err := inngestgo.Send(context.Background(), inngestgo.Event{
			Name: "test/sdk",
			Data: map[string]any{
				"priority": p,
			},
		})
		require.NoError(t, err)
		// The function takes 5s to run, so wait a second ensuring that we run events in
		// order and start processing functions.  By the first event, we have a backlog.
		<-time.After(2 * time.Second)
	}

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&counter) == 3
	}, 20*time.Second, time.Second)
}
