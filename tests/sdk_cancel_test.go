package main

import (
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
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

	hashes := map[string]string{
		"Sleep 10s":       "af731ad68b75abe9679cc9fc324a4ad3cd8075a2",
		"After the sleep": "425e4dc05acdef771b3b59a9ebbaae6377bebfc3",
	}

	retries := 10
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
		Function: inngest.Function{
			Name: "Cancel test",
			Slug: fnID,
			Triggers: []inngest.Trigger{
				{
					EventTrigger: &inngest.EventTrigger{
						Event: "tests/cancel.test",
					},
				},
			},
			Steps: []inngest.Step{
				{
					Name:    "step",
					ID:      "step",
					URI:     stepURL(fnID, "step"),
					Retries: &retries,
				},
			},
			Cancel: []inngest.Cancel{
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
				FnID:   inngest.DeterministicUUID(abstract.Function).String(),
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
				ID:   hashes["Sleep 10s"],
				Name: "10s",
			}}),

			// Send an unrelated event.
			test.After(time.Second),
			test.Send(inngestgo.Event{
				Name: "cancel/please",
				Data: map[string]interface{}{
					// This request ID doesn't match.
					"request_id": "12345",
				},
			}),

			// Update stack and state.  We should now have the sleep
			// item in our stack.
			test.AddRequestStack(driver.FunctionStack{
				Stack:   []string{hashes["Sleep 10s"]},
				Current: 1,
			}),
			test.AddRequestSteps(map[string]any{
				hashes["Sleep 10s"]: nil,
			}),

			// Then, within 10 seconds, we should call the function back.  This should
			// respond with a step.
			test.ExpectRequest("After sleep step", "step", 10*time.Second),
			test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
				Op:   enums.OpcodeStep,
				ID:   hashes["After the sleep"],
				Name: "After the sleep",
				Data: []byte(`"This should be cancelled if a matching cancel event is received"`),
			}}),

			// Update stack and state.  We should now have the step
			// in our stack.
			test.AddRequestStack(driver.FunctionStack{
				Stack:   []string{hashes["After the sleep"]},
				Current: 2,
			}),
			test.AddRequestSteps(map[string]any{
				hashes["After the sleep"]: "This should be cancelled if a matching cancel event is received",
			}),

			test.ExpectRequest("Final request as cancel didn't match", "step", 1*time.Second),
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
			"whatever":   "this doesn't matter my friend",
		},
		User: map[string]interface{}{},
	}

	retries := 10
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
		Function: inngest.Function{
			Name: "Cancel test",
			Slug: fnID,
			Triggers: []inngest.Trigger{
				{
					EventTrigger: &inngest.EventTrigger{
						Event: "tests/cancel.test",
					},
				},
			},
			Steps: []inngest.Step{
				{
					ID:      "step",
					Name:    "step",
					URI:     stepURL(fnID, "step"),
					Retries: &retries,
				},
			},
			Cancel: []inngest.Cancel{
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
			test.SetRequestSteps(map[string]any{}),
			// And the executor should start its requests with this context.
			test.SetRequestContext(SDKCtx{
				FnID:   inngest.DeterministicUUID(abstract.Function).String(),
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
