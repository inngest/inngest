package retries_go_test

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
		testdsl.RequireOutputWithin("received message", 500*time.Millisecond),

		// Ensure API publishes event.
		testdsl.RequireLogFields(map[string]any{
			"caller":     "api",
			"event_name": "basic/single-step-retries",
			"message":    "publishing event",
		}),

		// Ensure runner consumes event.
		testdsl.RequireLogFieldsWithin(map[string]any{
			"caller":  "runner",
			"event":   "basic/single-step-retries",
			"message": "received message",
		}, 5*time.Millisecond),
		testdsl.RequireLogFieldsWithin(map[string]any{
			"caller":  "runner",
			"message": "initializing fn",
		}, 5*time.Millisecond),

		// Ensure retries per step are adhered to
		testdsl.RequireStepRetries("step-custom-retries-high", 4),
		testdsl.RequireStepRetries("step-default-retries", 3),
		testdsl.RequireStepRetries("step-custom-retries-low", 2),
		testdsl.RequireStepRetries("step-custom-retries-none", 0),
	}
}
