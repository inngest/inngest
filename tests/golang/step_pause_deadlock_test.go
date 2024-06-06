package golang

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
	"github.com/stretchr/testify/require"
)

// Regression test for inngest/inngest#1430
func TestStepPauseDeadlockRegression(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)

	appID := "TestStepPauseDeadlockRegression"
	h, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	runID := ""
	reachedAfter := false
	var afterStepAttempts int32 = 0
	evtName := "my-event"

	invokeeFn := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "invokee-fn",
		},
		inngestgo.EventTrigger("youll-never-guess", nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			return "ok", nil
		},
	)

	fn := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name:    "my-fn",
			Retries: inngestgo.IntPtr(0),
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			runID = input.InputCtx.RunID

			step.Invoke[any](ctx, "invoke-step", step.InvokeOpts{
				FunctionId: fmt.Sprintf("%s-%s", appID, "invokee-fn"),
				Data:       map[string]any{},
			})

			_, err := step.Run(ctx, "after-invoke-step", func(ctx context.Context) (any, error) {
				atomic.AddInt32(&afterStepAttempts, 1)
				return nil, fmt.Errorf("uh oh")
			})

			reachedAfter = true

			return nil, err
		},
	)

	h.Register(invokeeFn, fn)
	registerFuncs()

	// Trigger the main function and successfully invoke the other function
	_, err := inngestgo.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)
	c.WaitForRunStatus(ctx, t, "FAILED", &runID)
	r.Exactly(int32(1), afterStepAttempts, "after step should have been attempted exactly once")
	r.True(reachedAfter)
}
