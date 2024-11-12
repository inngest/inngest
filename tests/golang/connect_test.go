package golang

import (
	"context"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
	"os"
	"sync/atomic"
	"testing"
	"time"
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
			require.ErrorIs(t, err, context.Canceled)
		}
	}()

	// Wait until we're connected
	// TODO Read the connection state API to see if socket is connected instead
	<-time.After(4 * time.Second)

	t.Run("trigger function", func(t *testing.T) {
		_, err := inngestgo.Send(ctx, ConnectEvent{
			Name: "test/connect",
			Data: map[string]interface{}{},
		})
		require.NoError(t, err)

		<-time.After(2 * time.Second)
		cancel()

		require.EqualValues(t, 1, atomic.LoadInt32(&counter))
	})

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
