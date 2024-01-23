package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/state"
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
		"first step": "98bf98df193bcce7c33e6bc50927cf2ac21206cb",
	}

	fnID := "test-suite-retry-test"
	test := &Test{
		ID:           fnID,
		Name:         "SDK Retry",
		Description:  ``,
		EventTrigger: evt,
		Timeout:      45 * time.Second,
	}

	test.SetAssertions(
		// All executor requests should have this event.
		test.SetRequestEvent(evt),
		test.SendTrigger(),

		test.ExpectRequest("Initial request", "step", time.Second),
		test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
			Op:          enums.OpcodeStepError,
			ID:          "98bf98df193bcce7c33e6bc50927cf2ac21206cb",
			Name:        "first step",
			DisplayName: inngestgo.StrPtr(`first step`),
			Error: &state.UserError{
				Name:    "Error",
				Message: "broken",
			},
			Data: []byte(`null`),
		}}),

		// We should retry the step successfully.
		test.Printf("Awaiting step retry"),
		test.ExpectRequest("Second request", "step", 45*time.Second, func(r *driver.SDKRequestContext) {
			r.Attempt = 1
		}),
		test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
			Op:          enums.OpcodeStepRun,
			ID:          hashes["first step"],
			Name:        "first step",
			DisplayName: inngestgo.StrPtr("first step"),
			Data:        []byte(`"yes: 2"`),
		}}),
		// Stack is updated
		test.AddRequestStack(driver.FunctionStack{
			Stack:   []string{hashes["first step"]},
			Current: 1,
		}),
		// State is updated with step data
		test.AddRequestSteps(map[string]any{
			hashes["first step"]: map[string]any{"data": "yes: 2"},
		}),

		// Finally, the function should be called and should error once.
		test.Printf("Awaiting function call after step"),
		test.ExpectRequest("Final call", "step", time.Second, func(r *driver.SDKRequestContext) {
			r.Attempt = 0
		}),
		// Expect a 500
		test.ExpectResponseFunc(500, func(byt []byte) error {
			e := map[string]any{}
			err := json.Unmarshal(byt, &e)
			require.NoError(t, err)

			require.Equal(t, "Error", e["name"])
			require.Equal(t, "broken func", e["message"])
			return nil
		}),

		test.Printf("Awaiting function call retry"),
		test.ExpectRequest("Final call", "step", 45*time.Second, func(r *driver.SDKRequestContext) {
			r.Attempt = 1
		}),
		test.ExpectJSONResponse(200, map[string]any{
			"body": "ok",
			"name": "tests/retry.test",
		}),
	)

	run(t, test)
}
