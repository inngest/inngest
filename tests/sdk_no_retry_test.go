package main

import (
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngestgo"
)

func TestSDKNoRetry(t *testing.T) {
	// 1. Assert that a function is registered with the name of "sdk-step-test"
	// 1. Assert that there's an invocation with no steps.
	evt := inngestgo.Event{
		Name: "tests/no-retry.test",
		Data: map[string]any{
			"hi": true,
		},
		User: map[string]any{},
	}

	fnID := "test-suite-no-retry"
	test := &Test{
		ID:           fnID,
		Name:         "SDK No Retry",
		Description:  ``,
		EventTrigger: evt,
		Timeout:      45 * time.Second,
	}

	test.SetAssertions(
		// All executor requests should have this event.
		test.SetRequestEvent(evt),
		test.SendTrigger(),

		test.Printf("Expecting StepFailed opcode"),

		test.ExpectRequest("Initial request", "step", time.Second),
		// Newer SDKs report a non-retriable error as an OpcodeStepFailed
		// (rather than OpcodeStepError) and carry the serialized error as the
		// step data. The stack is redacted by ExpectGeneratorResponse.
		test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
			Op:          enums.OpcodeStepFailed,
			ID:          "98bf98df193bcce7c33e6bc50927cf2ac21206cb",
			Name:        "first step",
			DisplayName: inngestgo.StrPtr(`first step`),
			Error: &state.UserError{
				Name:    "NonRetriableError",
				Message: "no retry plz",
			},
			Data: []byte(`{"__serialized":true,"name":"NonRetriableError","message":"no retry plz","stack":""}`),
		}}),

		test.Printf("Expecting Try/Catch request"),

		// We should get ANOTHER request which captures this error,
		// allowing for try-catch outside of the function.
		//
		// In this case, the above step is added to the stack with an error type.
		test.AddRequestStack(driver.FunctionStack{
			Stack:   []string{"98bf98df193bcce7c33e6bc50927cf2ac21206cb"},
			Current: 1,
		}),
		test.AddRequestSteps(map[string]any{
			// Data is wrapped.
			"98bf98df193bcce7c33e6bc50927cf2ac21206cb": map[string]any{
				"error": map[string]any{
					"message": "no retry plz",
					"name":    "NonRetriableError",
					"noRetry": true,
					// stack is ignored for now, as it has absolute paths.
				},
			},
		}),

		test.ExpectRequest("Try-catch request", "step", time.Second),
		test.ExpectResponse(200, []byte(`"ok"`)),
	)

	run(t, test)
}
