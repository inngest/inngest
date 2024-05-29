package golang

import (
	"context"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestInvokeRateLimit(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)

	appID := "InvokeRateLimit-" + ulid.MustNew(ulid.Now(), nil).String()
	h, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	// This function will be invoked by the main function
	invokedFnName := "invoked-fn"
	invokedFn := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: invokedFnName,
			RateLimit: &inngestgo.RateLimit{
				Limit:  1,
				Period: 1 * time.Minute,
			},
			Retries: inngestgo.IntPtr(0),
		},
		inngestgo.EventTrigger("none", nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			return nil, nil
		},
	)

	// This function will invoke the other function
	runID := ""
	evtName := "my-event"
	mainFn := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "main-fn",
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			runID = input.InputCtx.RunID

			_, _ = step.Invoke[any](
				ctx,
				"invoke",
				step.InvokeOpts{FunctionId: appID + "-" + invokedFnName})

			return nil, nil
		},
	)

	h.Register(invokedFn, mainFn)
	registerFuncs()

	// Trigger the main function and successfully invoke the other function
	_, err := inngestgo.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)
	c.WaitForRunStatus(ctx, t, "COMPLETED", &runID)

	// Trigger the main function. It'll fail because the invoked function is
	// rate limited
	runID = ""
	_, err = inngestgo.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)
	c.WaitForRunStatus(ctx, t, "FAILED", &runID)
}
