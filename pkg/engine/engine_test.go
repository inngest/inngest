package engine

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/pkg/event"
	"github.com/inngest/inngestctl/pkg/execution/driver"
	"github.com/inngest/inngestctl/pkg/execution/driver/mockdriver"
	"github.com/inngest/inngestctl/pkg/execution/executor"
	"github.com/inngest/inngestctl/pkg/function"
	"github.com/inngest/inngestctl/pkg/logger"
	"github.com/stretchr/testify/require"
)

// TestEngine_async asserst that the engine coordinates events between the runner, executor, and
// state manager to successfully pause workflows until specific events are received.
func TestEngine_async(t *testing.T) {
	ctx := context.Background()
	buf := &bytes.Buffer{}

	e, err := New(Options{
		Logger: logger.Buffered(buf),
	})
	require.NoError(t, err)

	err = e.SetFunctions(ctx, []*function.Function{
		{
			Name: "test fn",
			ID:   "test-fn",
			Triggers: []function.Trigger{
				{
					EventTrigger: &function.EventTrigger{
						Event: "test/new.event",
					},
				},
			},
			Steps: map[string]function.Step{
				"first": {
					Name: "first",
					Runtime: inngest.RuntimeWrapper{
						Runtime: &mockdriver.Mock{},
					},
					After: []function.After{
						{
							Step: inngest.TriggerName,
						},
					},
				},
				"wait-for-evt": {
					Name: "wait-for-evt",
					Runtime: inngest.RuntimeWrapper{
						Runtime: &mockdriver.Mock{},
					},
					After: []function.After{
						{
							Step: inngest.TriggerName,
							Async: &inngest.AsyncEdgeMetadata{
								Event: "test/continue",
								TTL:   "5s",
								Match: strptr("async.data.continue == 'yes'"),
							},
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	// Update the executor to use a mock driver.
	driver := &mockdriver.Mock{
		Responses: map[string]driver.Response{
			"first":        {Output: map[string]interface{}{"ok": true}},
			"wait-for-evt": {Output: map[string]interface{}{"ok": true}},
		},
	}
	exec, err := executor.NewExecutor(
		executor.WithStateManager(e.sm),
		executor.WithActionLoader(e.al),
		executor.WithRuntimeDrivers(
			driver,
		),
	)
	e.setExecutor(exec)
	require.NoError(t, err, "couldn't set mock driver")

	// 1.
	// Send an event that does nothing, and assert nothing runs.
	err = e.HandleEvent(ctx, &event.Event{
		Name: "test/random.walk.down.the.street",
		Data: map[string]interface{}{
			"test": true,
		},
	})
	require.NoError(t, err)
	<-time.After(50 * time.Millisecond)
	require.EqualValues(t, 0, len(driver.Executed))

	// 2.
	// HandleEvent should create a new execution when an event matches
	// the trigger.
	err = e.HandleEvent(ctx, &event.Event{
		Name: "test/new.event",
		Data: map[string]interface{}{
			"test": true,
		},
	})
	require.NoError(t, err)

	// Eventually the first step should execute.
	require.Eventually(t, func() bool {
		return len(driver.Executed) == 1
	}, 50*time.Millisecond, 10*time.Millisecond)
	// Assert that the first step ran.
	require.Equal(t, "first", driver.Executed["first"].Name)
	// And we should have a pause.
	require.Eventually(t, func() bool {
		return len(e.sm.Pauses()) == 1
	}, 50*time.Millisecond, 10*time.Millisecond)

	// 3.
	// Once we have the pause, we can send another event.  This shouldn't continue
	// the stopped function as the expression doesn't match.
	err = e.HandleEvent(ctx, &event.Event{
		Name: "test/continue",
		Data: map[string]interface{}{
			"continue": "no",
		},
	})
	require.NoError(t, err)
	<-time.After(50 * time.Millisecond)
	require.EqualValues(t, 1, len(driver.Executed))
	require.EqualValues(t, 1, len(e.sm.Pauses()))

	// 4.
	// Finally, assert that sending an event which matches the pause conditions
	// starts the workflow from the stopped edge.
	err = e.HandleEvent(ctx, &event.Event{
		Name: "test/continue",
		Data: map[string]interface{}{
			"continue": "yes",
		},
	})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		return len(driver.Executed) == 2
	}, 50*time.Millisecond, 10*time.Millisecond)
	require.Equal(t, "wait-for-evt", driver.Executed["wait-for-evt"].Name)
}

func strptr(s string) *string {
	return &s
}
