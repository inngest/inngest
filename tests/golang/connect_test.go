package golang

import (
	"context"
	"encoding/json"
	"fmt"
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

	ctx := context.Background()
	c := client.New(t)
	inngestClient := NewSDKConnectHandler(t, "connect")

	var (
		counter int32
		runID   string
	)

	connectCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "connect-test", Retries: inngestgo.IntPtr(0)},
		inngestgo.EventTrigger("test/connect", nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}

			atomic.AddInt32(&counter, 1)
			return "connect done", nil
		},
	)
	require.NoError(t, err)

	t.Run("with connection", func(t *testing.T) {
		wc, err := inngestgo.Connect(connectCtx, inngestgo.ConnectOpts{
			InstanceID: inngestgo.StrPtr("my-worker"),
			Apps:       []inngestgo.Client{inngestClient},
		})
		require.NoError(t, err)

		var workerGroupID string
		t.Run("verify connection is established", func(t *testing.T) {
			require.EventuallyWithT(t, func(collect *assert.CollectT) {
				a := assert.New(collect)

				resp, err := http.Get(fmt.Sprintf("%s/v0/connect/envs/dev/conns", DEV_URL))
				a.NoError(err)

				var reply rest.ShowConnsReply
				err = json.NewDecoder(resp.Body).Decode(&reply)
				a.NoError(err)

				a.Equal(1, len(reply.Data))

				if len(reply.Data) > 0 {
					for _, wgHash := range reply.Data[0].AllWorkerGroups {
						workerGroupID = wgHash
						break
					}
				}
			}, 5*time.Second, 500*time.Millisecond)
		})

		// Check if the SDK is synced
		t.Run("verify the worker is synced", func(t *testing.T) {
			require.EventuallyWithT(t, func(collect *assert.CollectT) {
				a := assert.New(collect)

				endpoint := fmt.Sprintf("%s/v0/connect/envs/dev/groups/%s", DEV_URL, workerGroupID)
				resp, err := http.Get(endpoint)
				a.NoError(err)

				var reply rest.ShowWorkerGroupReply
				a.NoError(json.NewDecoder(resp.Body).Decode(&reply))

				a.True(reply.Data.Synced)
			}, 5*time.Second, 500*time.Millisecond)
		})

		t.Run("trigger function", func(t *testing.T) {
			_, err := inngestClient.Send(ctx, inngestgo.Event{
				Name: "test/connect",
				Data: map[string]interface{}{},
			})
			require.NoError(t, err)

			<-time.After(2 * time.Second)
			require.EqualValues(t, 1, atomic.LoadInt32(&counter))
		})

		// Check span tree
		t.Run("trace run should have appropriate data", func(t *testing.T) {
			run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{Status: models.FunctionStatusCompleted})

			require.NotNil(t, run.Trace)
			require.True(t, run.Trace.IsRoot)
			require.Equal(t, models.RunTraceSpanStatusCompleted.String(), run.Trace.Status)
			// output test
			require.NotNil(t, run.Trace.OutputID)
			output := c.RunSpanOutput(ctx, *run.Trace.OutputID)
			c.ExpectSpanOutput(t, "connect done", output)
		})

		require.NoError(t, wc.Close())
	})

	// Connection is closed — with worker semaphores, the function stays queued
	// until a worker reconnects and restores semaphore capacity.
	t.Run("should remain queued without a connection", func(t *testing.T) {
		atomic.StoreInt32(&counter, 0)

		// Send event while no worker is connected
		eventID, err := inngestClient.Send(ctx, inngestgo.Event{
			Name: "test/connect",
			Data: map[string]interface{}{},
		})
		require.NoError(t, err)

		// Wait for the run to appear
		var queuedRunID string
		require.EventuallyWithT(t, func(a *assert.CollectT) {
			runsForEvent, err := c.RunsByEventID(ctx, eventID)
			if !assert.NoError(a, err) {
				return
			}
			if !assert.Len(a, runsForEvent, 1) {
				return
			}
			queuedRunID = runsForEvent[0].ID
		}, 10*time.Second, 1*time.Second)
		require.NotEmpty(t, queuedRunID)

		// Assert the function has NOT executed for 5 seconds (blocked by semaphore)
		time.Sleep(5 * time.Second)
		require.EqualValues(t, 0, atomic.LoadInt32(&counter))

		// Verify the run is still queued
		run := c.Run(ctx, queuedRunID)
		require.Equal(t, "QUEUED", run.Status)

		// Reconnect the worker — semaphore capacity is restored
		wc2, err := inngestgo.Connect(connectCtx, inngestgo.ConnectOpts{
			InstanceID: inngestgo.StrPtr("my-worker-2"),
			Apps:       []inngestgo.Client{inngestClient},
		})
		require.NoError(t, err)
		defer wc2.Close()

		// The queued function should now execute and complete
		require.EventuallyWithT(t, func(a *assert.CollectT) {
			assert.EqualValues(a, 1, atomic.LoadInt32(&counter))
		}, 15*time.Second, 1*time.Second)

		// Verify the run completed
		run = c.WaitForRunStatus(ctx, t, "COMPLETED", queuedRunID)
		require.Equal(t, "COMPLETED", run.Status)
	})
}
