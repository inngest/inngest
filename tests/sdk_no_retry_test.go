package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
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

	fnID := "test-suite-sdk-no-retry"
	test := &Test{
		Name:        "SDK No Retry",
		Description: ``,
		Function: inngest.Function{
			Name: "SDK No Retry",
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
		test.SetRequestContext(driver.SDKRequestContext{
			FunctionID: inngest.DeterministicUUID(test.Function),
			StepID:     "step",
			Stack: &driver.FunctionStack{
				Current: 0,
			},
		}),

		test.SendTrigger(),

		test.ExpectRequest("Initial request", "step", time.Second),
		// Expect a 500
		test.ExpectResponseFunc(400, func(byt []byte) error {
			// This should be a string, because the SDK double-serializes
			// step errors (right now)
			var str string
			err := json.Unmarshal(byt, &str)
			require.NoError(t, err)

			e := map[string]any{}
			err = json.Unmarshal([]byte(str), &e)
			require.NoError(t, err)

			require.Equal(t, "Error", e["name"])
			require.Equal(t, "no retry plz", e["message"])
			return nil
		}),
	)

	run(t, test)
}
