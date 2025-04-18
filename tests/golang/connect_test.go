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

	// Connection is closed
	t.Run("should fail without healthy connection", func(t *testing.T) {
		// Reset counter
		atomic.StoreInt32(&counter, 0)

		eventId, err := inngestClient.Send(ctx, inngestgo.Event{
			Name: "test/connect",
			Data: map[string]interface{}{},
		})
		require.NoError(t, err)

		var failedRunId string
		require.EventuallyWithT(t, func(a *assert.CollectT) {
			runsForEvent, err := c.RunsByEventID(ctx, eventId)
			if !assert.NoError(a, err) {
				return
			}
			if !assert.Len(a, runsForEvent, 1) {
				return
			}

			failedRunId = runsForEvent[0].ID
		}, 10*time.Second, 1*time.Second)
		require.EqualValues(t, 0, atomic.LoadInt32(&counter))
		require.NotEmpty(t, failedRunId)

		run := c.WaitForRunTraces(ctx, t, &failedRunId, client.WaitForRunTracesOptions{Status: models.FunctionStatusFailed})

		require.NotNil(t, run.Trace)
		require.True(t, run.Trace.IsRoot)
		require.Equal(t, models.RunTraceSpanStatusFailed.String(), run.Trace.Status)
		require.Equal(t, 1, len(run.Trace.ChildSpans))
		require.Equal(t, models.RunTraceSpanStatusFailed.String(), run.Trace.ChildSpans[0].Status)
		// output test
		require.NotNil(t, run.Trace.OutputID)
		output := c.RunSpanOutput(ctx, *run.Trace.OutputID)

		errorMsg := "{\"error\":{\"error\":\"connect_no_healthy_connection: Could not find a healthy connection\",\"name\":\"connect_no_healthy_connection\",\"message\":\"Could not find a healthy connection\"}}"

		require.NotNil(t, output.Error.Stack)
		require.Equal(t, errorMsg, *output.Error.Stack)

		r2 := c.Run(ctx, failedRunId)
		require.Equal(t, errorMsg, r2.Output)
	})
}
