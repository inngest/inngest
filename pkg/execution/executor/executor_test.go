package executor

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/pkg/execution/actionloader"
	"github.com/inngest/inngest-cli/pkg/execution/driver"
	"github.com/inngest/inngest-cli/pkg/execution/driver/mockdriver"
	"github.com/inngest/inngest-cli/pkg/execution/state"
	"github.com/inngest/inngest-cli/pkg/execution/state/inmemory"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExecutor(t *testing.T) {
	exec, err := NewExecutor()
	assert.Equal(t, ErrNoStateManager, err)
	assert.Nil(t, exec)

	sm := inmemory.NewStateManager()

	al := actionloader.NewMemoryLoader()

	exec, err = NewExecutor(
		WithStateManager(sm),
		WithActionLoader(al),
	)
	assert.Nil(t, err)
	assert.NotNil(t, exec)
}

func TestExecute_state(t *testing.T) {
	ctx := context.Background()
	sm := inmemory.NewStateManager()

	al := actionloader.NewMemoryLoader()
	al.Add(inngest.ActionVersion{
		DSN: "test",
		Runtime: inngest.RuntimeWrapper{
			Runtime: &mockdriver.Mock{},
		},
	})

	w := inngest.Workflow{
		UUID: uuid.New(),
		Steps: []inngest.Step{
			{
				DSN: "test",
				ID:  "1",
			},
			{
				DSN: "test",
				ID:  "2",
			},
			{
				DSN: "test",
				ID:  "3",
			},
			{
				DSN: "test",
				ID:  "4",
			},
			{
				DSN: "test",
				ID:  "5",
			},
			{
				DSN: "test",
				ID:  "6",
			},
			{
				DSN: "test",
				ID:  "7",
			},
		},
		Edges: []inngest.Edge{
			{
				Outgoing: inngest.TriggerName,
				Incoming: "1",
			},
			{
				Outgoing: inngest.TriggerName,
				Incoming: "2",
			},
			{
				Outgoing: "1",
				Incoming: "3",
			},
			{
				Outgoing: "3",
				Incoming: "4",
			},
			{
				Outgoing: "3",
				Incoming: "5",
			},
			{
				Outgoing: "4",
				Incoming: "6",
			},
			{
				Outgoing: "5",
				Incoming: "7",
			},
		},
	}

	s, err := sm.New(ctx, w, ulid.MustNew(ulid.Now(), rand.Reader), map[string]interface{}{})
	require.Nil(t, err)

	driver := &mockdriver.Mock{
		Responses: map[string]driver.Response{
			"1": {Output: map[string]interface{}{"id": 1}},
			"2": {Output: map[string]interface{}{"id": 2}},
			"3": {Output: map[string]interface{}{"id": 3}},
			"4": {Scheduled: true},
			"5": {Err: fmt.Errorf("some error")},
		},
	}

	exec, err := NewExecutor(
		WithStateManager(sm),
		WithActionLoader(al),
		WithRuntimeDrivers(driver),
	)

	require.Nil(t, err)
	require.NotNil(t, exec)

	// Executing the trigger does nothing but validate which descendents from the trigger
	// in the dag can run.
	_, err = exec.Execute(ctx, s.Identifier(), inngest.TriggerName)
	assert.NoError(t, err)
	assert.Equal(t, len(driver.Executed), 0)
	// assert.Equal(t, len(available), 2)
	// assert.ElementsMatch(t, []string{"1", "2"}, availableIDs(available))
	// There should be no state.
	s, err = sm.Load(ctx, s.Identifier())
	require.NoError(t, err)
	assert.Equal(t, 0, len(s.Actions()))

	// Run the first item.
	_, err = exec.Execute(ctx, s.Identifier(), "1")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(driver.Executed))
	// assert.Equal(t, 1, len(available))
	// assert.ElementsMatch(t, []string{"3"}, availableIDs(available))
	// Ensure we recorded state.
	s, err = sm.Load(ctx, s.Identifier())
	require.NoError(t, err)
	assert.Equal(t, 1, len(s.Actions()))
	assert.Equal(t, 0, len(s.Errors()))

	// Test "scheduled" responses.  The driver should respond with a Scheduled
	// message, which means that the function has begun execution but no further
	// actions are available.
	_, err = exec.Execute(ctx, s.Identifier(), "4")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(driver.Executed), "function not executed")
	// assert.Equal(t, 0, len(available), "incorrect number of functions available")
	// No state should be recorded.
	s, err = sm.Load(ctx, s.Identifier())
	require.NoError(t, err)
	assert.Equal(t, 1, len(s.Actions()))
	assert.Equal(t, 0, len(s.Errors()))

	// Test "error" responses
	_, err = exec.Execute(ctx, s.Identifier(), "5")
	assert.Error(t, err)
	assert.Equal(t, 3, len(driver.Executed), "function not executed")
	// assert.Equal(t, 0, len(available), "incorrect number of functions available")
	// An error should be recorded.
	s, err = sm.Load(ctx, s.Identifier())
	require.NoError(t, err)
	assert.Equal(t, 1, len(s.Actions()))
	assert.Equal(t, 1, len(s.Errors()))
}

// TestExecute_edge_expressions asserts that we execute expressions using the correct
// data, calculating edge expressions appropriately.
func TestExecute_edge_expressions(t *testing.T) {
	ctx := context.Background()
	sm := inmemory.NewStateManager()

	al := actionloader.NewMemoryLoader()
	al.Add(inngest.ActionVersion{
		DSN: "test",
		Runtime: inngest.RuntimeWrapper{
			Runtime: &mockdriver.Mock{},
		},
	})

	w := inngest.Workflow{
		UUID: uuid.New(),
		Steps: []inngest.Step{
			{
				DSN: "test",
				ID:  "run-step-trigger",
			},
			{
				DSN: "test",
				ID:  "dont-run-step-trigger",
			},
			{
				DSN: "test",
				ID:  "run-step-child",
			},
			{
				DSN: "test",
				ID:  "dont-run-step-child",
			},
		},
		Edges: []inngest.Edge{
			{
				Outgoing: inngest.TriggerName,
				Incoming: "run-step-trigger",
				Metadata: &inngest.EdgeMetadata{
					If: "event.data.run == true && event.data.string == 'yes'",
				},
			},
			// This won't be ran, as the expression is invalid
			{
				Outgoing: inngest.TriggerName,
				Incoming: "dont-run-step-trigger",
				Metadata: &inngest.EdgeMetadata{
					If: "event.data.run == false || event.data.string != 'yes'",
				},
			},
			// This should run, using "response" as the output of the first step.  It should
			// also allow hard-coding steps directly.
			{
				Outgoing: "run-step-trigger",
				Incoming: "run-step-child",
				Metadata: &inngest.EdgeMetadata{
					If: "response.ok == true && response.step == 'run-step-trigger' && event.data.run == true && steps['run-step-trigger'].ok == true",
				},
			},
			// This shouldn't run.
			{
				Outgoing: "run-step-trigger",
				Incoming: "run-step-child",
				Metadata: &inngest.EdgeMetadata{
					If: "response.ok != true || response.step != 'run-step-trigger'",
				},
			},
		},
	}

	s, err := sm.New(ctx, w, ulid.MustNew(ulid.Now(), rand.Reader), map[string]interface{}{
		"data": map[string]interface{}{
			"run":    true,
			"string": "yes",
		},
	})
	require.Nil(t, err)

	driver := &mockdriver.Mock{
		Responses: map[string]driver.Response{
			"run-step-trigger": {Output: map[string]interface{}{
				"ok":   true,
				"step": "run-step-trigger",
				"pi":   3.141,
			}},
			"run-step-child": {Output: map[string]interface{}{"ok": true, "step": "run-step-child"}},
		},
	}

	exec, err := NewExecutor(
		WithStateManager(sm),
		WithActionLoader(al),
		WithRuntimeDrivers(driver),
	)
	require.NoError(t, err)

	_, err = exec.Execute(ctx, s.Identifier(), inngest.TriggerName)
	require.NoError(t, err)
	require.Equal(t, len(driver.Executed), 0)

	s, err = sm.Load(ctx, s.Identifier())
	require.NoError(t, err)
	edges, err := state.DefaultEdgeEvaluator.AvailableChildren(ctx, s, inngest.TriggerName)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"run-step-trigger"}, availableIDs(edges))

	// As we haven't ran the step called run-step-trigger, we should have no children available.
	edges, err = state.DefaultEdgeEvaluator.AvailableChildren(ctx, s, "run-step-trigger")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{}, availableIDs(edges))

	// Run the next step.
	response, err := exec.Execute(ctx, s.Identifier(), "run-step-trigger")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(driver.Executed))
	assert.NoError(t, response.Err)
	assert.EqualValues(t, *response, driver.Responses["run-step-trigger"])

	s, err = sm.Load(ctx, s.Identifier())
	require.NoError(t, err)
	edges, err = state.DefaultEdgeEvaluator.AvailableChildren(ctx, s, "run-step-trigger")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"run-step-child"}, availableIDs(edges))
}

func availableIDs(edges []inngest.Edge) []string {
	strs := make([]string, len(edges))
	for n, e := range edges {
		strs[n] = e.Incoming
	}
	return strs
}
