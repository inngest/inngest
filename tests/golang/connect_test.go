package golang

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/connect/rest"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndToEnd(t *testing.T) {
	os.Setenv("INNGEST_EVENT_KEY", "abc123")
	os.Setenv("INNGEST_SIGNING_KEY", "signkey-test-12345678")
	os.Setenv("INNGEST_SIGNING_KEY_FALLBACK", "signkey-test-00000000")

	type ConnectEvent = inngestgo.GenericEvent[any, any]
	ctx := context.Background()
	c := client.New(t)
	h := NewSDKConnectHandler(t, "connect")

	var (
		counter int32
		runID   string
	)

	connectCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	a := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{Name: "connect test"},
		inngestgo.EventTrigger("test/connect", nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}

			atomic.AddInt32(&counter, 1)
			return "connect done", nil
		},
	)
	h.Register(a)

	go func() {
		err := h.Connect(connectCtx)
		if err != nil {
			// This is expected
			if errors.Is(err, context.Canceled) {
				return
			}

			// This error may happen but should be fixed before releasing
			// TODO Why is the reader attempting to read from a closed connection?
			if errors.Is(err, net.ErrClosed) {
				return
			}

			require.NoError(t, err)
		}
	}()

	var workerGroupID string
	t.Run("verify connection is established", func(t *testing.T) {
		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			a := assert.New(collect)

			resp, err := http.Get("http://127.0.0.1:8289/v0/envs/dev/conns")
			a.NoError(err)

			var reply rest.ShowConnsReply
			err = json.NewDecoder(resp.Body).Decode(&reply)
			a.NoError(err)

			a.Equal(1, len(reply.Data))

			if len(reply.Data) > 0 {
				workerGroupID = reply.Data[0].GroupId
			}
		}, 5*time.Second, 500*time.Millisecond)
	})

	// Check if the SDK is synced
	t.Run("verify the worker is synced", func(t *testing.T) {
		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			a := assert.New(collect)

			endpoint := fmt.Sprintf("http://127.0.0.1:8289/v0/envs/dev/groups/%s", workerGroupID)
			resp, err := http.Get(endpoint)
			a.NoError(err)

			var reply rest.ShowWorkerGroupReply
			a.NoError(json.NewDecoder(resp.Body).Decode(&reply))

			a.True(reply.Data.Synced)
		}, 5*time.Second, 500*time.Millisecond)
	})

	t.Run("trigger function", func(t *testing.T) {
		_, err := inngestgo.Send(ctx, ConnectEvent{
			Name: "test/connect",
			Data: map[string]interface{}{},
		})
		require.NoError(t, err)

		<-time.After(2 * time.Second)
		require.EqualValues(t, 1, atomic.LoadInt32(&counter))

		cancel()
	})

	// Check span tree
	t.Run("trace run should have appropriate data", func(t *testing.T) {
		run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{Status: models.FunctionStatusCompleted})

		require.NotNil(t, run.Trace)
		require.True(t, run.Trace.IsRoot)
		require.Equal(t, 0, len(run.Trace.ChildSpans))
		require.Equal(t, models.RunTraceSpanStatusCompleted.String(), run.Trace.Status)
		// output test
		require.NotNil(t, run.Trace.OutputID)
		output := c.RunSpanOutput(ctx, *run.Trace.OutputID)
		c.ExpectSpanOutput(t, "connect done", output)
	})

}
