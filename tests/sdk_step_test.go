package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/state"
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
		"first step":  "98bf98df193bcce7c33e6bc50927cf2ac21206cb",
		"sleep":       "dd44d5dc73e81cfbd3c93d03c50160b0b8dc3d6a",
		"second step": "764e20ec975d4ef820d0f42e6a5833384bd7ee36",
	}

	test := &Test{
		ID:   "test-suite-step-test",
		Name: "SDK Steps",
		Description: `
			This test asserts that steps works across the SDK.  This tests steps and sleeps
			in a serial manner:

			- step.run
			- step.sleep
			- step.run
		`,
		EventTrigger: evt,
		Timeout:      2 * time.Second,
	}

	test.SetAssertions(
		// All executor requests should have this event.
		test.SetRequestEvent(evt),

		test.SendTrigger(),

		// Expect to run the first step immediately, no plan included.  The SDK
		// optimizes single steps to run immediately.
		test.ExpectRequest("Initial request plan", "step", time.Second),
		test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
			Op:          enums.OpcodeStepRun,
			ID:          hashes["first step"],
			Name:        "first step",
			DisplayName: inngestgo.StrPtr("first step"),
			Data:        []byte(`"first step"`),
		}}),
		// Stack is updated
		test.AddRequestStack(driver.FunctionStack{
			Stack:   []string{hashes["first step"]},
			Current: 1,
		}),
		// State is updated with step data
		test.AddRequestSteps(map[string]any{
			hashes["first step"]: map[string]any{"data": "first step"},
		}),

		// Execute the step again, get a wait
		test.ExpectRequest("Wait step run", "step", time.Second),
		test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
			Op:          enums.OpcodeSleep,
			ID:          hashes["sleep"],
			Data:        json.RawMessage("null"),
			Name:        "2s",
			DisplayName: inngestgo.StrPtr("for 2s"),
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
			Op:          enums.OpcodeStepRun,
			ID:          hashes["second step"],
			DisplayName: inngestgo.StrPtr("second step"),
			Name:        "second step",
			Data:        json.RawMessage(`{"first":"first step","second":true}`),
		}}),

		// Update state with step data
		test.AddRequestStack(driver.FunctionStack{
			Stack:   []string{hashes["second step"]},
			Current: 3,
		}),
		test.AddRequestSteps(map[string]any{
			hashes["second step"]: map[string]any{
				"data": map[string]any{
					"first":  "first step",
					"second": true,
				},
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
