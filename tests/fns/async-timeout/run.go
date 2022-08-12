package async_timeout

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

		testdsl.RequireLogFieldsWithin(map[string]any{
			"caller":   "runner",
			"message":  "initializing fn",
			"function": "async-timeout-fn-id",
		}, testdsl.DefaultDuration),

		// The function should run within 10ms.
		testdsl.RequireLogFieldsWithin(map[string]any{
			"caller":   "runner",
			"message":  "initializing fn",
			"function": "async-timeout-fn-id",
		}, testdsl.DefaultDuration),

		// The trigger should run.
		testdsl.RequireTriggerExecution,

		// We shouldn't get anything re. ignoring a pause timeout for 9s.
		testdsl.RequireNoLogFieldsWithin(
			map[string]any{
				"message": "scheduling pause timeout step",
			},
			9*time.Second,
		),
		// Then within the 10th second we should ignore the pause timeout,
		// then run our function
		testdsl.RequireLogFieldsWithin(
			map[string]any{
				"message": "scheduling pause timeout step",
			},
			time.Second,
		),
		testdsl.RequireLogFieldsWithin(
			map[string]any{
				"message": "executing step",
				"step":    "step-1",
			},
			testdsl.DefaultDuration,
		),
	}
}
