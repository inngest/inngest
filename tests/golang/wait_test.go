package golang

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
	"github.com/stretchr/testify/require"
)

func TestWaitInvalidExpression(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)

	appID := "TestWaitInvalidExpression"
	h, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	// This function will invoke the other function
	runID := ""
	evtName := "my-event"
	fn := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "main-fn",
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			runID = input.InputCtx.RunID

			_, _ = step.WaitForEvent[any](
				ctx,
				"wait",
				step.WaitForEventOpts{
					If:      inngestgo.StrPtr("invalid"),
					Name:    "dummy",
					Timeout: time.Second,
				},
			)

			return nil, nil
		},
	)

	h.Register(fn)
	registerFuncs()

	// Trigger the main function and successfully invoke the other function
	_, err := inngestgo.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)
	c.WaitForRunStatus(ctx, t, "COMPLETED", &runID)
}

func TestWaitInvalidExpressionSyntaxError(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)

	appID := "TestWaitInvalidExpression"
	h, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	// This function will invoke the other function
	runID := ""
	evtName := "my-event"
	fn := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "main-fn",
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			runID = input.InputCtx.RunID

			_, _ = step.WaitForEvent[any](
				ctx,
				"wait",
				step.WaitForEventOpts{
					If:      inngestgo.StrPtr("event.data.userId === async.data.userId"),
					Name:    "test/continue",
					Timeout: time.Second,
				},
			)

			return nil, nil
		},
	)

	h.Register(fn)
	registerFuncs()

	// Trigger the main function and successfully invoke the other function
	_, err := inngestgo.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)
	run := c.WaitForRunStatus(ctx, t, "FAILED", &runID)
	assert.Equal(t, "{\"error\":{\"message\":\"Could not compile expression\",\"name\":\"CompileError\",\"stack\":\"ERROR: \\u003cinput\\u003e:1:21: Syntax error: token recognition error at: '= '\\n | event.data.userId === async.data.userId\\n | ....................^\"}}", run.Output)
}
