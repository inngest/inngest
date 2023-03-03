package main

import (
	"testing"
	"time"

	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/function"
	"github.com/inngest/inngestgo"
)

func TestSDKCancelNotReceived(t *testing.T) {
	evt := inngestgo.Event{
		Name: "tests/cancel.test",
		Data: map[string]any{
			"request_id": "123",
		},
		User: map[string]interface{}{},
	}

	fnID := "test-suite-cancel-test"
	abstract := Test{
		Name: "Cancel test",
		Description: `
			This test asserts that steps works across the SDK.  This tests steps and sleeps
			in a serial manner:

			- step.run
			- step.sleep
			- step.run
		`,
		Function: function.Function{
			ID:   fnID,
			Name: "Cancel test",
			Triggers: []function.Trigger{
				{
					EventTrigger: &function.EventTrigger{
						Event: "tests/cancel.test",
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
			Cancel: []function.Cancel{
				{
					Event:   "cancel/please",
					Timeout: strptr("1h"),
					If:      strptr("async.data.request_id == event.data.request_id"),
				},
			},
		},
		EventTrigger: evt,
		Timeout:      20 * time.Second,
	}

	t.Run("Without a cancellation event", func(t *testing.T) {
		copied := abstract
		test := &copied
		test.SetAssertions(
			// All executor requests should have this event.
			test.SetRequestEvent(evt),
			// And the executor should start its requests with this context.
			test.SetRequestContext(SDKCtx{
				FnID:   fnID,
				StepID: "step",
				Stack: driver.FunctionStack{
					Current: 0,
				},
			}),

			test.SendTrigger(),

			// Execute the step again, get a wait
			test.ExpectRequest("Wait step run", "step", time.Second),
			test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
				Op:   enums.OpcodeSleep,
				ID:   "af731ad68b75abe9679cc9fc324a4ad3cd8075a2",
				Name: "10s",
			}}),

			test.After(time.Second),
			test.Send(inngestgo.Event{
				Name: "cancel/please",
				Data: map[string]interface{}{
					// This request ID doesn't match.
					"request_id": "lol no",
				},
			}),

			// Update stack and state
			test.AddRequestStack(driver.FunctionStack{
				Stack:   []string{"af731ad68b75abe9679cc9fc324a4ad3cd8075a2"},
				Current: 1,
			}),
			test.AddRequestSteps(map[string]any{
				"af731ad68b75abe9679cc9fc324a4ad3cd8075a2": nil,
			}),

			test.ExpectRequest("Final request as cancel didn't match", "step", 12*time.Second),
			test.ExpectJSONResponse(200, map[string]any{"name": "tests/cancel.test", "body": "ok"}),
		)
		run(t, test)
	})
}

func TestSDKCancelReceived(t *testing.T) {
	evt := inngestgo.Event{
		Name: "tests/cancel.test",
		Data: map[string]any{
			"request_id": "123",
		},
		User: map[string]interface{}{},
	}

	fnID := "test-suite-cancel-test"
	abstract := Test{
		Name: "Cancel test",
		Description: `
			This test asserts that steps works across the SDK.  This tests steps and sleeps
			in a serial manner:

			- step.run
			- step.sleep
			- step.run
		`,
		Function: function.Function{
			ID:   fnID,
			Name: "Cancel test",
			Triggers: []function.Trigger{
				{
					EventTrigger: &function.EventTrigger{
						Event: "tests/cancel.test",
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
			Cancel: []function.Cancel{
				{
					Event:   "cancel/please",
					Timeout: strptr("1h"),
					If:      strptr("async.data.request_id == event.data.request_id"),
				},
			},
		},
		EventTrigger: evt,
		Timeout:      20 * time.Second,
	}

	t.Run("With a cancellation event", func(t *testing.T) {
		copied := abstract
		test := &copied
		test.SetAssertions(
			// All executor requests should have this event.
			test.SetRequestEvent(evt),
			test.SetRequestSteps(nil),
			// And the executor should start its requests with this context.
			test.SetRequestContext(SDKCtx{
				FnID:   fnID,
				StepID: "step",
				Stack: driver.FunctionStack{
					Current: 0,
				},
			}),

			test.SendTrigger(),

			// Execute the step again, get a wait
			test.ExpectRequest("Wait step run", "step", time.Second),
			test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
				Op:   enums.OpcodeSleep,
				ID:   "af731ad68b75abe9679cc9fc324a4ad3cd8075a2",
				Name: "10s",
			}}),

			test.After(time.Second),
			test.Send(inngestgo.Event{
				Name: "cancel/please",
				Data: map[string]interface{}{
					// This request ID doesn't match.
					"request_id": "123",
				},
			}),
			// Nothing should be called
		)
		run(t, test)
	})
}
