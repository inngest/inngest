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
		"first":       "34b1246022be1f725e62d3bd9699f8ceef4f7801",
		"sleep":       "564638b655aca430db7a71bdb32b666c3988d31f",
		"second step": "97d696dd9f212cd6ed88993f61d21183c0691c2e",
	}

	fnID := "test-suite-sdk-step-test"
	test := Test{
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

		Assertions: []HTTPAssertion{
			// First we're only passed the event and context.
			ExecutorRequest{
				Event: evt,
				Ctx: SDKCtx{
					FnID:   fnID,
					StepID: "step",
				},
			},
			SDKResponse{
				// And we reply with step data.
				Status: 206,
				Data: []state.GeneratorOpcode{
					{
						Op:   enums.OpcodeStepPlanned,
						ID:   hashes["first"],
						Name: "test step",
					},
				},
			},

			// We should re-invoke the function to call the step.
			ExecutorRequest{
				Event: evt,
				Ctx: SDKCtx{
					FnID:   fnID,
					StepID: "step",
				},
				QueryStepID: hashes["first"],
			},
			SDKResponse{
				// The step is called with data.
				Status: 206,
				Data: []state.GeneratorOpcode{
					{
						Op:   enums.OpcodeStep,
						ID:   hashes["first"],
						Name: "test step",
						Data: []byte(`"first step"`),
					},
				},
			},

			// After the step is called, we should re-invoke the function with the hash
			// as the executor request.
			ExecutorRequest{
				Event: evt,
				Ctx: SDKCtx{
					FnID:   fnID,
					StepID: "step",
					Stack: driver.FunctionStack{
						Stack:   []string{hashes["first"]},
						Current: 1,
					},
				},
				Steps: map[string]any{
					hashes["first"]: "first step",
				},
				QueryStepID: "step",
			},
			SDKResponse{
				// The SDK then sleeps
				Status: 206,
				Data: []state.GeneratorOpcode{
					{
						Op:   enums.OpcodeSleep,
						Name: "2s",
						ID:   hashes["sleep"],
					},
				},
			},

			// After sleeping, we re-invoke the function and plan the second step.
			ExecutorRequest{
				Event: evt,
				Ctx: SDKCtx{
					FnID:   fnID,
					StepID: "step",
					Stack: driver.FunctionStack{
						Stack:   []string{hashes["first"], hashes["sleep"]},
						Current: 2,
					},
				},
				Steps: map[string]any{
					hashes["first"]: "first step",
					hashes["sleep"]: nil,
				},
				QueryStepID: "step",
			},
			SDKResponse{
				Status: 206,
				Data: []state.GeneratorOpcode{
					{
						Op:   enums.OpcodeStepPlanned,
						Name: "second step",
						ID:   hashes["second step"],
					},
				},
			},

			// The second step is then called.
			ExecutorRequest{
				Event: evt,
				Ctx: SDKCtx{
					FnID:   fnID,
					StepID: "step",
					Stack: driver.FunctionStack{
						Stack:   []string{hashes["first"], hashes["sleep"]},
						Current: 2,
					},
				},
				Steps: map[string]any{
					hashes["first"]: "first step",
					hashes["sleep"]: nil,
				},
				// The query ID must be the hash of the second step.
				QueryStepID: hashes["second step"],
			},
			SDKResponse{
				Status: 206,
				Data: []state.GeneratorOpcode{
					{
						Op:   enums.OpcodeStep,
						Name: "second step",
						ID:   hashes["second step"],
						Data: json.RawMessage(`{"first":"first step","second":true}`),
					},
				},
			},

			// The SDK is finally called and returns
			ExecutorRequest{
				Event: evt,
				Ctx: SDKCtx{
					FnID:   fnID,
					StepID: "step",
					Stack: driver.FunctionStack{
						Stack: []string{
							hashes["first"],
							hashes["sleep"],
							hashes["second step"],
						},
						Current: 3,
					},
				},
				Steps: map[string]any{
					hashes["first"]: "first step",
					hashes["sleep"]: nil,
					hashes["second step"]: map[string]any{
						"first":  "first step",
						"second": true,
					},
				},
				QueryStepID: "step",
			},
			SDKResponse{
				Status: 200,
				Data: map[string]any{
					"body": "ok",
					"name": "tests/step.test",
				},
			},
		},

		Timeout: 10 * time.Second,
	}

	run(t, test)
}
