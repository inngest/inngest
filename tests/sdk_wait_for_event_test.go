package main

import (
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngestgo"
)

func TestSDKWaitForEvent(t *testing.T) {
	evt := inngestgo.Event{
		Name: "tests/wait.test",
		Data: map[string]any{
			"id": "123",
		},
		User: map[string]interface{}{},
	}

	hashes := map[string]string{
		"wait": "f36cefdaa75e3285f4f5f20054d3f4f305d726a8",
	}

	fnID := "test-suite-wait-for-event-test"
	abstract := Test{
		Name: "Wait for event test",
		Description: `
		`,
		Function: inngest.Function{
			Name: "Wait for event test",
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
					Name: "step",
					ID:   "step",
					URI:  stepURL(fnID, "step"),
				},
			},
		},
		EventTrigger: evt,
		Timeout:      30 * time.Second,
	}

	resumeID := "resume"
	resume := inngestgo.Event{
		ID:   &resumeID,
		Name: "test/resume",
		Data: map[string]interface{}{
			// This request ID doesn't match.
			"id":     evt.Data["id"],
			"resume": true,
		},
		Timestamp: time.Now().UnixMilli(),
	}

	t.Run("With an event during the timeout", func(t *testing.T) {
		copied := abstract
		test := &copied
		test.SetAssertions(
			// All executor requests should have this event.
			test.SetRequestEvent(evt),
			// And the executor should start its requests with this context.
			test.SetRequestContext(SDKCtx{
				FnID:   inngest.DeterministicUUID(abstract.Function).String(),
				StepID: "step",
				Stack: driver.FunctionStack{
					Current: 0,
				},
			}),
			test.SendTrigger(),

			// Execute the step again, get a wait
			test.ExpectRequest("Wait step run", "step", time.Second),
			test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
				Op:   enums.OpcodeWaitForEvent,
				ID:   hashes["wait"],
				Name: "test/resume",
				Opts: map[string]any{
					"if":      "async.data.resume == true && async.data.id == event.data.id",
					"timeout": "10s",
				},
			}}),

			// Send an unrelated event.
			test.After(time.Second),
			test.Send(inngestgo.Event{
				Name: "test/resume",
				Data: map[string]interface{}{
					// This request ID doesn't match.
					"id": "lol what in the world?!",
				},
			}),

			test.After(time.Second),
			test.Send(resume),

			// Update stack and state.  We should now have the sleep
			// item in our stack.
			test.AddRequestStack(driver.FunctionStack{
				Stack:   []string{hashes["wait"]},
				Current: 1,
			}),
			test.AddRequestSteps(map[string]any{
				hashes["wait"]: resume.Map(),
			}),

			// Then, within 10 seconds, we should call the function back.  This should
			// respond with a step.
			test.ExpectRequest("After wait step", "step", 1*time.Second),
			test.ExpectJSONResponse(200, map[string]any{"result": map[string]any{"id": "123", "resume": true}}),
		)

		run(t, test)
	})
}

func TestSDKWaitForEvent_NoEvent(t *testing.T) {
	evt := inngestgo.Event{
		Name: "tests/wait.test",
		Data: map[string]any{
			"id": "123",
		},
		User: map[string]interface{}{},
	}

	hashes := map[string]string{
		"wait": "f36cefdaa75e3285f4f5f20054d3f4f305d726a8",
	}

	fnID := "test-suite-wait-for-event-test"
	abstract := Test{
		Name: "Wait for event test",
		Description: `
		`,
		Function: inngest.Function{
			Name: "Wait for event test",
			Slug: fnID,
			Triggers: []inngest.Trigger{
				{
					EventTrigger: &inngest.EventTrigger{
						Event: evt.Name,
					},
				}},
			Steps: []inngest.Step{
				{
					Name: "step",
					ID:   "step",
					URI:  stepURL(fnID, "step"),
				},
			},
		},
		EventTrigger: evt,
		Timeout:      30 * time.Second,
	}

	resumeID := "resume"
	resume := inngestgo.Event{
		ID:   &resumeID,
		Name: "test/resume",
		Data: map[string]interface{}{
			// This request ID doesn't match.
			"id":     evt.Data["id"],
			"resume": true,
		},
		Timestamp: time.Now().UnixMilli(),
	}

	t.Run("Without an event", func(t *testing.T) {
		copied := abstract
		test := &copied
		test.SetAssertions(
			// All executor requests should have this event.
			test.SetRequestEvent(evt),
			// And the executor should start its requests with this context.
			test.SetRequestContext(SDKCtx{
				FnID:   inngest.DeterministicUUID(abstract.Function).String(),
				StepID: "step",
				Stack: driver.FunctionStack{
					Current: 0,
				},
			}),
			test.SendTrigger(),

			// Execute the step again, get a wait
			test.ExpectRequest("Wait step run", "step", time.Second),
			test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
				Op:   enums.OpcodeWaitForEvent,
				ID:   hashes["wait"],
				Name: "test/resume",
				Opts: map[string]any{
					"if":      "async.data.resume == true && async.data.id == event.data.id",
					"timeout": "10s",
				},
			}}),

			test.After(11*time.Second),
			test.Send(resume),

			// Update stack and state.  We should now have the sleep
			// item in our stack.
			test.AddRequestStack(driver.FunctionStack{
				Stack:   []string{hashes["wait"]},
				Current: 1,
			}),
			test.AddRequestSteps(map[string]any{
				hashes["wait"]: nil,
			}),

			// Then, within 10 seconds, we should call the function back.  This should
			// respond with a step.
			test.ExpectRequest("After wait step", "step", 1*time.Second),
			test.ExpectJSONResponse(200, map[string]any{}),
		)

		run(t, test)
	})
}
