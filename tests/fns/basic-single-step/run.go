package basic_test

import (
	"context"
	"time"

	"github.com/inngest/inngest/tests/testdsl"
)

func init() {
	testdsl.Register(Do)
}

func Do(ctx context.Context) testdsl.Chain {
	return testdsl.Chain{
		testdsl.SendTrigger,
		testdsl.RequireReceiveTrigger,

		// Ensure API publishes event.
		testdsl.RequireLogFields(map[string]any{
			"caller":  "api",
			"event":   "basic/single-step",
			"message": "publishing event",
		}),
		// Ensure runner consumes event.
		testdsl.RequireLogFieldsWithin(map[string]any{
			"caller":  "runner",
			"event":   "basic/single-step",
			"message": "received message",
		}, 5*time.Millisecond),
		testdsl.RequireLogFieldsWithin(map[string]any{
			"caller":  "runner",
			"message": "initializing fn",
		}, 5*time.Millisecond),
		testdsl.RequireLogFieldsWithin(map[string]any{
			"caller":  "executor",
			"step":    "basic-step-1",
			"message": "executing step",
		}, time.Second*2),
		testdsl.RequireLogFieldsWithin(map[string]any{
			"caller": "output",
			"output": map[string]any{
				"body":   "basic/single-step",
				"status": 200,
			},
			"message": "step output",
		}, time.Second*2),
		testdsl.RequireNoOutput(`"error"`),
	}
}
