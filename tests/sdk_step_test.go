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
		"first step":  "ffd46aab701259a8c1e39bcd9adeaff6fa752340",
		"sleep":       "518add570bed90f1ad3191f40d346d47bd25da83",
		"second step": "555cb806535e79feaa831b5b2a5044f5d243930f",
	}

	fnID := "test-suite-sdk-step-test"
	test := &Test{
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

		/*
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
		*/
	}

	test.SetAssertions(
		// All executor requests should have this event.
		test.SetRequestEvent(evt),
		// And the executor should start its requests with this context.
		test.SetRequestContext(SDKCtx{
			FnID:   fnID,
			StepID: "step",
			Stack: driver.FunctionStack{
				Current: 0,
			},
		}),

		test.SendTrigger(),

		// Expect the first opcode planning the step
		test.ExpectRequest("Initial request plan", "step", time.Second),
		test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
			Op:   enums.OpcodeStepPlanned,
			ID:   hashes["first step"],
			Name: "first step",
		}}),

		// Rerun the executor, run the step
		test.ExpectRequest("Initial request run", hashes["first step"], time.Second),
		test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
			Op:   enums.OpcodeStep,
			ID:   hashes["first step"],
			Name: "first step",
			Data: []byte(`"first step"`),
		}}),
		// Stack is updated
		test.AddRequestStack(driver.FunctionStack{
			Stack:   []string{hashes["first step"]},
			Current: 1,
		}),
		// State is updated with step data
		test.AddRequestSteps(map[string]any{
			hashes["first step"]: "first step",
		}),

		// Execute the step again, get a wait
		test.ExpectRequest("Wait step run", "step", time.Second),
		test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
			Op:   enums.OpcodeSleep,
			ID:   hashes["sleep"],
			Name: "2s",
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
			Op:   enums.OpcodeStepPlanned,
			ID:   hashes["second step"],
			Name: "second step",
		}}),
		test.ExpectRequest("Post wait", hashes["second step"], time.Second),
		test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
			Op:   enums.OpcodeStep,
			ID:   hashes["second step"],
			Name: "second step",
			Data: json.RawMessage(`{"first":"first step","second":true}`),
		}}),

		// Update state with step data
		test.AddRequestStack(driver.FunctionStack{
			Stack:   []string{hashes["second step"]},
			Current: 3,
		}),
		test.AddRequestSteps(map[string]any{
			hashes["second step"]: map[string]any{
				"first":  "first step",
				"second": true,
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
