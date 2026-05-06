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

func TestSDKCancelNotReceived(t *testing.T) {
	evt := inngestgo.Event{
		Name: "tests/cancel.test",
		Data: map[string]any{
			"request_id": "123",
		},
		User: map[string]interface{}{},
	}

	hashes := map[string]string{
		"Sleep 10s":       "c3ca5f787365eae0dea86250e27d476406956478",
		"After the sleep": "dcd448548befa33b66c7a4927d1eac75f6d18107",
	}

	abstract := Test{
		ID:   "test-suite-cancel-test",
		Name: "Cancel test",
		Description: `
			This test asserts that steps works across the SDK.  This tests steps and sleeps
			in a serial manner:

			- step.run
			- step.sleep
			- step.run
		`,
		EventTrigger: evt,
		Timeout:      20 * time.Second,
	}

	t.Run("Without a cancellation event", func(t *testing.T) {
		copied := abstract
		test := &copied
		test.SetAssertions(
			// All executor requests should have this event.
			test.SetRequestEvent(evt),
			test.SendTrigger(),

			// Execute the step again, get a wait
			test.ExpectRequest("Wait step run", "step", time.Second),
			test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
				Op:          enums.OpcodeSleep,
				ID:          hashes["Sleep 10s"],
				Name:        "10s",
				DisplayName: inngestgo.StrPtr("sleep"),
				Data:        json.RawMessage("null"),
				Opts:        map[string]any{},
				Userland: &struct {
					ID    string `json:"id"`
					Index int    `json:"index,omitempty"`
				}{ID: "sleep"},
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

			// In v4, after sleep the SDK does immediate execution of step.run
			// and returns the final result directly.
			test.ExpectRequest("After sleep step", "step", 10*time.Second),
			test.ExpectRunCompleteResponse(map[string]any{"name": "tests/cancel.test", "body": "ok"}),
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

	abstract := Test{
		ID:   "test-suite-cancel-test",
		Name: "Cancel test",
		Description: `
			This test asserts that steps works across the SDK.  This tests steps and sleeps
			in a serial manner:

			- step.run
			- step.sleep
			- step.run
		`,
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
			test.SetRequestContext(driver.SDKRequestContext{
				StepID: "step",
				Stack: &driver.FunctionStack{
					Current: 0,
				},
			}),

			test.SendTrigger(),

			// Execute the step again, get a wait
			test.ExpectRequest("Wait step run", "step", time.Second),
			test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
				Op:          enums.OpcodeSleep,
				ID:          "c3ca5f787365eae0dea86250e27d476406956478",
				Name:        "10s",
				DisplayName: inngestgo.StrPtr("sleep"),
				Data:        json.RawMessage("null"),
				Opts:        map[string]any{},
				Userland: &struct {
					ID    string `json:"id"`
					Index int    `json:"index,omitempty"`
				}{ID: "sleep"},
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
