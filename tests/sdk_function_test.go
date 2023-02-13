package main

import (
	"testing"
	"time"

	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/pkg/function"
	"github.com/inngest/inngestgo"
)

func TestSDKFunctions(t *testing.T) {
	// 1. Assert that a function is registered with the name of "sdk-function-test"
	// 1. Assert that there's an invocation with no steps.
	evt := inngestgo.Event{
		Name: "tests/function.test",
		Data: map[string]any{
			"test": true,
		},
		User: map[string]any{},
	}

	fnID := "test-suite-sdk-function-test"
	test := Test{
		Name: "SDK Functions",
		Description: `
			This test asserts that functions work across SDKs.

			In order for functions to work, the SDK must:
			- Allow functions to be introspected via the GET handler
			- Register functions correctly with the server
			- Handle incoming event triggers
			- Respond with the correct, expected data
		`,
		Function: function.Function{
			ID:   fnID,
			Name: "SDK Function Test",
			Triggers: []function.Trigger{
				{
					EventTrigger: &function.EventTrigger{
						Event: "tests/function.test",
					},
				},
			},
			Steps: map[string]function.Step{
				"step": {
					ID:   "step",
					Name: "step",
					Runtime: &inngest.RuntimeWrapper{
						Runtime: &inngest.RuntimeHTTP{
							URL: stepURL(fnID, "step"),
						},
					},
				},
			},
		},
		EventTrigger: evt,

		Assertions: []HTTPAssertion{
			// First we're only passed the event and context.
			ExecutorRequest{
				Event: evt,
				Ctx: SDKCtx{
					FnID:   fnID,
					StepID: "step",
				},
			},
			// And we reply with step data.
			SDKResponse{
				Status: 200,
				Data: map[string]any{
					"body": "ok",
					"name": "tests/function.test",
				},
			},
		},

		Timeout: time.Second,
	}

	run(t, test)
}
