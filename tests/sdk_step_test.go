package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/function"
	"github.com/inngest/inngestgo"
)

func TestSDKSteps(t *testing.T) {
	// 1. Assert that a function is registered with the name of "sdk-step-test"
	// 1. Assert that there's an invocation with no steps.
	evt := inngestgo.Event{
		Name: "tests/step.test",
		Data: map[string]any{
			"steps": map[string]any{
				"ok": "yes",
			},
		},
		User: map[string]any{
			"email": "test@example.com",
		},
	}

	hashes := map[string]string{
		"first step":  "ffd46aab701259a8c1e39bcd9adeaff6fa752340",
		"sleep":       "518add570bed90f1ad3191f40d346d47bd25da83",
		"second step": "555cb806535e79feaa831b5b2a5044f5d243930f",
	}

	fnID := "test-suite-sdk-step-test"
	test := &Test{
		Name: "SDK Steps",
		Description: `
			This test asserts that steps works across the SDK.  This tests steps and sleeps
			in a serial manner:

			- step.run
			- step.sleep
			- step.run
		`,
		Function: function.Function{
			ID:   fnID,
			Name: "SDK Step Test",
			Triggers: []function.Trigger{
				{
					EventTrigger: &function.EventTrigger{
						Event: "tests/step.test",
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
		Timeout:      2 * time.Second,
	}

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

		// Expect to run the first step immediately, no plan included.  The SDK
		// optimizes single steps to run immediately.
		test.ExpectRequest("Initial request plan", "step", time.Second),
		test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
			Op:   enums.OpcodeStep,
			ID:   hashes["first step"],
			Name: "first step",
			Data: []byte(`"first step"`),
		}}),
		// Stack is updated
		test.AddRequestStack(driver.FunctionStack{
			Stack:   []string{hashes["first step"]},
			Current: 1,
		}),
		// State is updated with step data
		test.AddRequestSteps(map[string]any{
			hashes["first step"]: "first step",
		}),

		// Execute the step again, get a wait
		test.ExpectRequest("Wait step run", "step", time.Second),
		test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
			Op:   enums.OpcodeSleep,
			ID:   hashes["sleep"],
			Name: "2s",
		}}),
		// Update stack and state
		test.AddRequestStack(driver.FunctionStack{
			Stack:   []string{hashes["sleep"]},
			Current: 2,
		}),
		test.AddRequestSteps(map[string]any{
			hashes["sleep"]: nil,
		}),

		// After the wait we should re-invoke the request _again_
		test.ExpectRequest("Post wait", "step", 3*time.Second),
		test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
			Op:   enums.OpcodeStep,
			ID:   hashes["second step"],
			Name: "second step",
			Data: json.RawMessage(`{"first":"first step","second":true}`),
		}}),

		// Update state with step data
		test.AddRequestStack(driver.FunctionStack{
			Stack:   []string{hashes["second step"]},
			Current: 3,
		}),
		test.AddRequestSteps(map[string]any{
			hashes["second step"]: map[string]any{
				"first":  "first step",
				"second": true,
			},
		}),

		// Finally, the function should be called and should return a 200
		test.ExpectRequest("Final call", "step", time.Second),
		test.ExpectJSONResponse(200, map[string]any{
			"body": "ok",
			"name": "tests/step.test",
		}),
	)

	run(t, test)
}
