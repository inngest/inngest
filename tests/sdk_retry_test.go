package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
)

func TestSDKRetry(t *testing.T) {
	// 1. Assert that a function is registered with the name of "sdk-step-test"
	// 1. Assert that there's an invocation with no steps.
	evt := inngestgo.Event{
		Name: "tests/retry.test",
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
		"first step": "ffd46aab701259a8c1e39bcd9adeaff6fa752340",
	}

	fnID := "test-suite-sdk-retry-test"
	test := &Test{
		Name:        "SDK Retry",
		Description: ``,
		Function: inngest.Function{
			Name: "SDK Retry Test",
			Slug: fnID,
			Triggers: []inngest.Trigger{
				{
					EventTrigger: &inngest.EventTrigger{
						Event: evt.Name,
					},
				},
			},
			Steps: []inngest.Step{
				{
					ID:   "step",
					Name: "step",
					URI:  stepURL(fnID, "step"),
				},
			},
		},
		EventTrigger: evt,
		Timeout:      45 * time.Second,
	}

	test.SetAssertions(
		// All executor requests should have this event.
		test.SetRequestEvent(evt),
		// And the executor should start its requests with this context.
		test.SetRequestContext(SDKCtx{
			FnID:   inngest.DeterministicUUID(test.Function).String(),
			StepID: "step",
			Stack: driver.FunctionStack{
				Current: 0,
			},
		}),

		test.SendTrigger(),

		test.ExpectRequest("Initial request", "step", time.Second),
		// Expect a 500
		test.ExpectResponseFunc(500, func(byt []byte) error {
			// This should be a string, because the SDK double-serializes
			// step errors (right now)
			var str string
			err := json.Unmarshal(byt, &str)
			require.NoError(t, err)

			e := map[string]any{}
			err = json.Unmarshal([]byte(str), &e)
			require.NoError(t, err)

			require.Equal(t, "Error", e["name"])
			require.Equal(t, "broken", e["message"])
			return nil
		}),

		test.ExpectRequest("Second request", "step", 45*time.Second),
		test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
			Op:   enums.OpcodeStep,
			ID:   hashes["first step"],
			Name: "first step",
			Data: []byte(`"yes: 2"`),
		}}),
		// Stack is updated
		test.AddRequestStack(driver.FunctionStack{
			Stack:   []string{hashes["first step"]},
			Current: 1,
		}),
		// State is updated with step data
		test.AddRequestSteps(map[string]any{
			hashes["first step"]: "yes: 2",
		}),

		//

		// Finally, the function should be called and should error once.
		test.ExpectRequest("Final call", "step", time.Second),
		// Expect a 500
		test.ExpectResponseFunc(500, func(byt []byte) error {
			// This should be a string, because the SDK double-serializes
			// step errors (right now)
			var str string
			err := json.Unmarshal(byt, &str)
			require.NoError(t, err)

			e := map[string]any{}
			err = json.Unmarshal([]byte(str), &e)
			require.NoError(t, err)

			require.Equal(t, "Error", e["name"])
			require.Equal(t, "broken func", e["message"])
			return nil
		}),
		test.ExpectRequest("Final call", "step", 45*time.Second),

		test.ExpectJSONResponse(200, map[string]any{
			"body": "ok",
			"name": "tests/retry.test",
		}),
	)

	run(t, test)
}
