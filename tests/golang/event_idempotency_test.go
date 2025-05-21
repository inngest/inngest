package golang

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventIdempotency(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)
	inngestClient, server, registerFuncs := NewSDKHandler(t, "test")
	defer server.Close()

	var (
		counter int32
		runID   string
	)
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "test"},
		inngestgo.EventTrigger("test", nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}

			atomic.AddInt32(&counter, 1)
			return nil, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	sendEvent := func(id string) {
		_, err := inngestClient.Send(ctx, inngestgo.Event{
			ID:   &id,
			Name: "test",
		})
		r.NoError(err)
	}

	t.Run("same ID", func(t *testing.T) {
		// Only 1 run if multiple events have the same ID
		for range 5 {
			sendEvent("abc")
		}

		r.Eventually(func() bool {
			return atomic.LoadInt32(&counter) == 1
		}, 2*time.Second, time.Second)

		// Wait a little longer to make sure no more runs happen
		<-time.After(100 * time.Millisecond)

		r.Equal(int32(1), atomic.LoadInt32(&counter))

		t.Run("trace run should have appropriate data", func(t *testing.T) {
			run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{
				Status:         models.FunctionStatusCompleted,
				ChildSpanCount: 1,
			})
			require.False(t, run.IsBatch)
			require.Nil(t, run.BatchCreatedAt)

			t.Run("exec", func(t *testing.T) {
				exec := run.Trace.ChildSpans[0]

				assert.Equal(t, "function success", exec.Name)
				assert.False(t, exec.IsRoot)
			})
		})
	})

	t.Run("different IDs", func(t *testing.T) {
		// Multiple runs if each event has a different ID

		sendEvent("abc")
		sendEvent("def")

		r.Eventually(func() bool {
			return atomic.LoadInt32(&counter) == 2
		}, 2*time.Second, time.Second)
	})
}
