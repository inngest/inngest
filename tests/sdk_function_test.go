package main

import (
	"testing"
	"time"

	"github.com/inngest/inngestgo"
)

func TestSDKFunctions(t *testing.T) {
	// 1. Assert that a function is registered with the name of "sdk-function-test"
	// 1. Assert that there's an invocation with no steps.
	evt := inngestgo.Event{
		Name: "tests/function.test",
		Data: map[string]any{
			"test": true,
		},
		User: map[string]any{},
	}

	test := &Test{
		ID: "test-suite-simple-fn",
		Description: `
			This test asserts that functions work across SDKs.

			In order for functions to work, the SDK must:
			- Allow functions to be introspected via the GET handler
			- Register functions correctly with the server
			- Handle incoming event triggers
			- Respond with the correct, expected data
		`,
		EventTrigger: evt,
	}

	test.SetAssertions(
		// All executor requests should have this event.
		test.SetRequestEvent(evt),
		// Send trigger.
		test.SendTrigger(),
		test.ExpectRequest("Initial request", "step", time.Second),
		test.ExpectJSONResponse(200, map[string]any{"name": "tests/function.test", "body": "ok"}),
	)

	run(t, test)
}
