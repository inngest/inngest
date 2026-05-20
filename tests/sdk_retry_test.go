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

	fnID := "test-suite-retry-test"
	test := &Test{
		ID:           fnID,
		Name:         "SDK Retry",
		Description:  ``,
		EventTrigger: evt,
		Timeout:      3 * time.Minute,
	}

	test.SetAssertions(
		// All executor requests should have this event.
		test.SetRequestEvent(evt),
		test.SendTrigger(),

		test.ExpectRequest("Initial request", "step", 5*time.Second),
		test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
			Op:          enums.OpcodeStepError,
			ID:          "98bf98df193bcce7c33e6bc50927cf2ac21206cb",
			Name:        "first step",
			DisplayName: inngestgo.StrPtr(`first step`),
			Error: &state.UserError{
				Name:    "Error",
				Message: "broken",
			},
			Data:    []byte(`null`),
			Opts:    map[string]any{},
			Userland: &struct {
				ID    string `json:"id"`
				Index int    `json:"index,omitempty"`
			}{ID: "first step"},
		}}),

		// We should retry the step successfully. In v4 with immediate execution,
		// the step succeeds and the function continues inline. The function then
		// throws, so we get a 500 error response.
		test.Printf("Awaiting step retry"),
		test.ExpectRequest("Second request", "step", 90*time.Second, func(r *driver.SDKRequestContext) {
			r.Attempt = 1
		}),
		test.ExpectResponseFunc(500, func(byt []byte) error {
			e := map[string]any{}
			err := json.Unmarshal(byt, &e)
			require.NoError(t, err)

			require.Equal(t, "Error", e["name"])
			require.Equal(t, "broken func", e["message"])
			return nil
		}),

		// Server retries the function. It persists the step result from the
		// previous execution (even though the function failed) and includes it
		// in the next request's stack/steps.
		test.Printf("Awaiting function call retry"),
		test.AddRequestStack(driver.FunctionStack{
			Stack:   []string{"98bf98df193bcce7c33e6bc50927cf2ac21206cb"},
			Current: 0,
		}),
		test.AddRequestSteps(map[string]any{
			"98bf98df193bcce7c33e6bc50927cf2ac21206cb": map[string]any{
				"data": "yes",
			},
		}),
		test.ExpectRequest("Final call", "step", 90*time.Second, func(r *driver.SDKRequestContext) {
			r.Attempt = 2
		}),
		test.ExpectRunCompleteResponse(map[string]any{
			"body": "ok",
			"name": "tests/retry.test",
		}),
	)

	run(t, test)
}
