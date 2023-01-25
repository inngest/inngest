package executor

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/inngest"
	inmemorydatastore "github.com/inngest/inngest/pkg/coredata/inmemory"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver/mockdriver"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/inmemory"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExecutor(t *testing.T) {
	exec, err := NewExecutor()
	assert.Equal(t, ErrNoStateManager, err)
	assert.Nil(t, exec)

	sm := inmemory.NewStateManager()

	al := inmemorydatastore.NewInMemoryActionLoader()

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

	al := inmemorydatastore.NewInMemoryActionLoader()
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

	id := state.Identifier{
		RunID: ulid.MustNew(ulid.Now(), rand.Reader),
	}

	s, err := sm.New(ctx, state.Input{
		Workflow:   w,
		Identifier: id,
		EventData:  map[string]interface{}{},
	})
	require.Nil(t, err)

	driver := &mockdriver.Mock{
		Responses: map[string]state.DriverResponse{
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
	_, idx, err := exec.Execute(ctx, s.Identifier(), inngest.SourceEdge, 0, 0)
	assert.NoError(t, err)
	assert.Equal(t, len(driver.Executed), 0)
	// assert.Equal(t, len(available), 2)
	// assert.ElementsMatch(t, []string{"1", "2"}, availableIDs(available))
	// There should be no state.
	s, err = sm.Load(ctx, s.RunID())
	require.NoError(t, err)
	assert.Equal(t, 0, len(s.Actions()))

	// Run the first item.
	_, idx, err = exec.Execute(ctx, s.Identifier(), inngest.Edge{Outgoing: inngest.TriggerName, Incoming: "1"}, 0, idx)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(driver.Executed))
	// assert.Equal(t, 1, len(available))
	// assert.ElementsMatch(t, []string{"3"}, availableIDs(available))
	// Ensure we recorded state.
	s, err = sm.Load(ctx, s.RunID())
	require.NoError(t, err)
	assert.Equal(t, 1, len(s.Actions()))
	assert.Equal(t, 0, len(s.Errors()))

	// Test "scheduled" responses.  The driver should respond with a Scheduled
	// message, which means that the function has begun execution but no further
	// actions are available.
	_, idx, err = exec.Execute(ctx, s.Identifier(), inngest.Edge{Outgoing: "3", Incoming: "4"}, 0, idx)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(driver.Executed), "function not executed")
	// assert.Equal(t, 0, len(available), "incorrect number of functions available")
	// No state should be recorded.
	s, err = sm.Load(ctx, s.RunID())
	require.NoError(t, err)
	assert.Equal(t, 1, len(s.Actions()))
	assert.Equal(t, 0, len(s.Errors()))

	// Test "error" responses
	_, idx, err = exec.Execute(ctx, s.Identifier(), inngest.Edge{Outgoing: "3", Incoming: "5"}, 0, idx)
	assert.Error(t, err)
	assert.Equal(t, 3, len(driver.Executed), "function not executed")
	// assert.Equal(t, 0, len(available), "incorrect number of functions available")
	// An error should be recorded.
	s, err = sm.Load(ctx, s.RunID())
	require.NoError(t, err)
	assert.Equal(t, 1, len(s.Actions()))
	assert.Equal(t, 1, len(s.Errors()))
}

func TestExecute_Generator(t *testing.T) {
	ctx := context.Background()
	sm := inmemory.NewStateManager()

	al := inmemorydatastore.NewInMemoryActionLoader()
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
				ID:  "step",
			},
		},
		Edges: []inngest.Edge{
			{
				Outgoing: inngest.TriggerName,
				Incoming: "step",
			},
		},
	}

	id := state.Identifier{WorkflowID: w.UUID, RunID: ulid.MustNew(ulid.Now(), rand.Reader)}

	s, err := sm.New(ctx, state.Input{
		Workflow:   w,
		Identifier: id,
		EventData:  map[string]interface{}{},
	})
	require.Nil(t, err)

	responses := map[string]state.DriverResponse{
		"step": {
			Output: map[string]interface{}{"id": 1},
			Generator: []*state.GeneratorOpcode{
				{
					Op: enums.OpcodeStep,
					ID: "1:some-id",
					// JSON encoded string
					Data: []byte(`"hello"`),
				},
			},
		},
	}
	driver := &mockdriver.Mock{
		Responses: responses,
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
	_, idx, err := exec.Execute(ctx, s.Identifier(), inngest.SourceEdge, 0, 0)
	assert.NoError(t, err)
	assert.Equal(t, len(driver.Executed), 0)

	// Execute the first generator step.
	_, idx, err = exec.Execute(ctx, s.Identifier(), w.Edges[0], 0, idx)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(driver.Executed))
	// Ensure we recorded state.
	s, err = sm.Load(ctx, s.RunID())
	require.NoError(t, err)
	output := s.Actions()
	assert.Equal(t, 1, len(output))
	assert.Equal(t, "hello", output["1:some-id"], "Data should be unmarshalled JSON")

	// Update the responses map with another generator.
	data := map[string]any{
		"u wot": "m8",
		"ok":    []any{true, false},
	}
	byt, _ := json.Marshal(data)
	driver.Responses = map[string]state.DriverResponse{
		"step": {
			Output: map[string]interface{}{"id": 1},
			Generator: []*state.GeneratorOpcode{
				{
					Op:   enums.OpcodeStep,
					ID:   "2:another-id",
					Data: byt,
				},
			},
		},
	}

	// Execute the second generator step.
	_, _, err = exec.Execute(ctx, s.Identifier(), w.Edges[0], 0, idx)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(driver.Executed))
	// Ensure we recorded state.
	s, err = sm.Load(ctx, s.RunID())
	require.NoError(t, err)
	output = s.Actions()
	assert.Equal(t, 2, len(output))
	assert.Equal(t, "hello", output["1:some-id"], "Data should be unmarshalled JSON")
	assert.Equal(t, data, output["2:another-id"], "Data should be unmarshalled JSON")
}

// TestExecute_edge_expressions asserts that we execute expressions using the correct
// data, calculating edge expressions appropriately.
func TestExecute_edge_expressions(t *testing.T) {
	ctx := context.Background()
	sm := inmemory.NewStateManager()

	al := inmemorydatastore.NewInMemoryActionLoader()
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

	id := state.Identifier{
		RunID: ulid.MustNew(ulid.Now(), rand.Reader),
	}

	s, err := sm.New(ctx, state.Input{
		Workflow:   w,
		Identifier: id,
		EventData: map[string]interface{}{
			"data": map[string]interface{}{
				"run":    true,
				"string": "yes",
			},
		},
	})
	require.Nil(t, err)

	driver := &mockdriver.Mock{
		Responses: map[string]state.DriverResponse{
			"run-step-trigger": {
				Step: inngest.Step{
					DSN: "test",
					ID:  "run-step-trigger",
				},
				Output: map[string]interface{}{
					"ok":   true,
					"step": "run-step-trigger",
					"pi":   3.141,
				},
			},
			"run-step-child": {
				Step: inngest.Step{
					DSN: "test",
					ID:  "run-step-child",
				},
				Output: map[string]interface{}{"ok": true, "step": "run-step-child"},
			},
		},
	}

	exec, err := NewExecutor(
		WithStateManager(sm),
		WithActionLoader(al),
		WithRuntimeDrivers(driver),
	)
	require.NoError(t, err)

	_, idx, err := exec.Execute(ctx, s.Identifier(), inngest.SourceEdge, 0, 0)
	require.NoError(t, err)
	require.Equal(t, len(driver.Executed), 0)

	s, err = sm.Load(ctx, s.RunID())
	require.NoError(t, err)
	edges, err := state.DefaultEdgeEvaluator.AvailableChildren(ctx, s, inngest.TriggerName)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"run-step-trigger"}, availableIDs(edges))

	// As we haven't ran the step called run-step-trigger, we should have no children available.
	edges, err = state.DefaultEdgeEvaluator.AvailableChildren(ctx, s, "run-step-trigger")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{}, availableIDs(edges))

	// Run the next step.
	response, _, err := exec.Execute(ctx, s.Identifier(), w.Edges[0], 0, idx)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(driver.Executed))
	assert.NoError(t, response.Err)
	assert.EqualValues(t, *response, driver.Responses["run-step-trigger"])

	s, err = sm.Load(ctx, s.RunID())
	require.NoError(t, err)
	edges, err = state.DefaultEdgeEvaluator.AvailableChildren(ctx, s, "run-step-trigger")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"run-step-child"}, availableIDs(edges))
}

func availableIDs(edges []state.AvailableEdge) []string {
	strs := make([]string, len(edges))
	for n, e := range edges {
		strs[n] = e.Incoming
	}
	return strs
}
