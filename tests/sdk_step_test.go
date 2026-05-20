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

		// In v4, the SDK does immediate execution of step.run("first step")
		// inline and continues to the next blocking operation (sleep).
		test.ExpectRequest("Initial request plan", "step", 5*time.Second),
		test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
			Op:          enums.OpcodeSleep,
			ID:          hashes["sleep"],
			Data:        json.RawMessage("null"),
			Name:        "2s",
			DisplayName: inngestgo.StrPtr("for 2s"),
			Opts:        map[string]any{},
			Userland: &struct {
				ID    string `json:"id"`
				Index int    `json:"index,omitempty"`
			}{ID: "for 2s"},
		}}),
		// The server tracks both the inline-executed first step and the sleep
		test.AddRequestStack(driver.FunctionStack{
			Stack:   []string{hashes["first step"], hashes["sleep"]},
			Current: 2,
		}),
		test.AddRequestSteps(map[string]any{
			hashes["first step"]: map[string]any{"data": "first step"},
			hashes["sleep"]:      nil,
		}),

		// After the sleep, the SDK re-executes the function. The first step
		// runs inline again (immediate execution), sleep is memoized, then
		// second step runs inline, and the function returns.
		test.ExpectRequest("Post wait", "step", 3*time.Second),
		test.ExpectRunCompleteResponse(map[string]any{
			"body": "ok",
			"name": "tests/step.test",
		}),
	)

	run(t, test)
}
