package executor

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/pkg/execution/actionloader"
	"github.com/inngest/inngestctl/pkg/execution/driver"
	"github.com/inngest/inngestctl/pkg/execution/driver/mockdriver"
	"github.com/inngest/inngestctl/pkg/execution/state/inmemory"
	"github.com/oklog/ulid"
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

func TestExecutor(t *testing.T) {
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
				DSN:      "test",
				ClientID: "1",
			},
			{
				DSN:      "test",
				ClientID: "2",
			},
			{
				DSN:      "test",
				ClientID: "3",
			},
			{
				DSN:      "test",
				ClientID: "4",
			},
			{
				DSN:      "test",
				ClientID: "5",
			},
			{
				DSN:      "test",
				ClientID: "6",
			},
			{
				DSN:      "test",
				ClientID: "7",
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

	state, err := sm.New(ctx, w, ulid.MustNew(ulid.Now(), rand.Reader), map[string]interface{}{})
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
	available, err := exec.Execute(ctx, state.Identifier(), inngest.TriggerName)
	assert.NoError(t, err)
	assert.Equal(t, len(driver.Executed), 0)
	assert.Equal(t, len(available), 2)
	assert.ElementsMatch(t, []string{"1", "2"}, available)
	// There should be no state.
	state, err = sm.Load(ctx, state.Identifier())
	require.NoError(t, err)
	assert.Equal(t, 0, len(state.Actions()))

	// Run the first item.
	available, err = exec.Execute(ctx, state.Identifier(), "1")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(driver.Executed))
	assert.Equal(t, 1, len(available))
	assert.ElementsMatch(t, []string{"3"}, available)
	// Ensure we recorded state.
	state, err = sm.Load(ctx, state.Identifier())
	require.NoError(t, err)
	assert.Equal(t, 1, len(state.Actions()))
	assert.Equal(t, 0, len(state.Errors()))

	// Test "scheduled" responses.  The driver should respond with a Scheduled
	// message, which means that the function has begun execution but no further
	// actions are available.
	available, err = exec.Execute(ctx, state.Identifier(), "4")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(driver.Executed), "function not executed")
	assert.Equal(t, 0, len(available), "incorrect number of functions available")
	// No state should be recorded.
	state, err = sm.Load(ctx, state.Identifier())
	require.NoError(t, err)
	assert.Equal(t, 1, len(state.Actions()))
	assert.Equal(t, 0, len(state.Errors()))

	// Test "error" responses
	available, err = exec.Execute(ctx, state.Identifier(), "5")
	assert.Error(t, err)
	assert.Equal(t, 3, len(driver.Executed), "function not executed")
	assert.Equal(t, 0, len(available), "incorrect number of functions available")
	// An error should be recorded.
	state, err = sm.Load(ctx, state.Identifier())
	require.NoError(t, err)
	assert.Equal(t, 2, len(state.Actions()))
	assert.Equal(t, 1, len(state.Errors()))
}
