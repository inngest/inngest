package main

import (
	"testing"
	"time"

	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/function"
	"github.com/inngest/inngestgo"
)

func hash(u driver.UnhashedOp) string {
	s, _ := u.Hash()
	return s
}

func TestSDKParallelism(t *testing.T) {
	// 1. Assert that a function is registered with the name of "sdk-step-test"
	// 1. Assert that there's an invocation with no steps.
	evt := inngestgo.Event{
		Name: "tests/parallel.test",
		Data: map[string]any{},
		User: map[string]any{},
	}

	hashes := map[string]string{
		"a": hash(driver.UnhashedOp{Name: "a", Op: enums.OpcodeStepPlanned}),
		"b": hash(driver.UnhashedOp{Name: "b", Op: enums.OpcodeStepPlanned}),
	}

	fnID := "test-suite-sdk-parallel-test"
	test := &Test{
		Name: "SDK Parallel Test",
		Description: `
			This test asserts that parallelism works within the SDK.  It enqueues two steps
			to run concurrently, then a third step which must run after the previous steps
			have both completed.
		`,
		Function: function.Function{
			ID:   fnID,
			Name: "SDK Parallel Test",
			Triggers: []function.Trigger{
				{
					EventTrigger: &function.EventTrigger{
						Event: "tests/parallel.test",
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
		Timeout:      2 * time.Second,
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

		// Expect to run the first step immediately, no plan included.  The SDK
		// optimizes single steps to run immediately.
		test.ExpectRequest("Initial request plan", "step", time.Second),

		test.ExpectGeneratorResponse([]state.GeneratorOpcode{
			{
				Op:   enums.OpcodeStepPlanned,
				ID:   hashes["a"],
				Name: "a",
			},
			{
				Op:   enums.OpcodeStepPlanned,
				ID:   hashes["b"],
				Name: "b",
			},
		}),

		// Expect simultaneous responses.
		test.ExpectParallelSteps(func() []state.GeneratorOpcode {
			return []state.GeneratorOpcode{
				{
					Op:   enums.OpcodeStep,
					ID:   hashes["a"],
					Name: "a",
					Data: []byte(`"a"`),
				},
				{
					Op:   enums.OpcodeStep,
					ID:   hashes["b"],
					Name: "b",
					Data: []byte(`"b"`),
				},
			}
		}, time.Second),

		test.ExpectParallelSteps(func() []state.GeneratorOpcode {
			return []state.GeneratorOpcode{
				{},
				{
					Op: enums.OpcodeStep,
					// The hash of this step is based off of the last parallel step to finish,
					// which is racey.
					ID: hash(driver.UnhashedOp{
						Name:   "c",
						Op:     enums.OpcodeStepPlanned,
						Parent: last(test.requestCtx.Stack.Stack),
					}),
					Name: "c",
					Data: []byte(`"c"`),
				},
			}
		}, time.Second),

		// Finally, the function should be called and should return a 200
		test.ExpectRequest("Final call", "step", time.Second),
		test.ExpectJSONResponse(200, map[string]any{
			"a": "a",
			"b": "b",
			"c": "c",
		}),
	)

	run(t, test)
}

func last(stack []string) *string {
	if len(stack) == 0 {
		return nil
	}
	last := stack[len(stack)-1]
	return &last
}
