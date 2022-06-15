package testharness

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/pkg/event"
	"github.com/inngest/inngest-cli/pkg/execution/state"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

var (
	w = inngest.Workflow{
		Name: "Test workflow",
		ID:   "shuffling-sphinx-87bd12",
		Triggers: []inngest.Trigger{
			{
				EventTrigger: &inngest.EventTrigger{
					Event: "test/some.event",
				},
			},
		},
		Steps: []inngest.Step{
			{
				ID:       "step-a",
				ClientID: 1,
				Name:     "first step",
				DSN:      "test-step",
			},
			{
				ID:       "step-b",
				ClientID: 2,
				Name:     "second step",
				DSN:      "test-step",
			},
		},
		Edges: []inngest.Edge{
			{
				Incoming: inngest.TriggerName,
				Outgoing: "step-a",
			},
			{
				Incoming: "step-a",
				Outgoing: "step-b",
			},
		},
	}

	input = event.Event{
		Name: "test-event",
		Data: map[string]any{
			"title": "They don't think it be like it is, but it do",
			"data": map[string]any{
				"float": 3.14132,
			},
		},
		User: map[string]any{
			"external_id": "1",
		},
		Version: "1985-01-01",
	}
)

func CheckState(t *testing.T, m state.Manager) {
	t.Helper()
	t.Parallel()

	funcs := map[string]func(t *testing.T, m state.Manager){
		"New":                         checkNew,
		"SaveActionOutput":            checkSaveOutput,
		"SaveActionOutputClearsError": checkSaveOutputClearsError,
		"SaveActionError":             checkSaveError,
		"SavePause":                   checkSavePause,
		"LeasePause":                  checkLeasePause,
		"ConsumePause":                checkConsumePause,
	}
	for name, f := range funcs {
		t.Run(name, func(t *testing.T) { f(t, m) })
	}
}

func checkNew(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	found := s.Workflow()
	require.EqualValues(t, w, found, "Returned workflow does not match input")
	require.EqualValues(t, input.Map(), s.Event(), "Returned event does not match input")

	loaded, err := m.Load(ctx, s.Identifier())
	require.NoError(t, err)

	found = loaded.Workflow()
	require.EqualValues(t, w, found, "Loaded workflow does not match input")
	require.EqualValues(t, input.Map(), loaded.Event(), "Loaded event does not match input")
}

func checkSaveOutput(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	//
	// Save some basic output
	//
	first := map[string]any{
		"first": true,
	}
	next, err := m.SaveActionOutput(ctx, s.Identifier(), w.Steps[0].ID, first)
	require.NoError(t, err)
	require.EqualValues(t, s.Identifier(), next.Identifier())
	require.EqualValues(t, s.Workflow(), next.Workflow())
	require.EqualValues(t, s.Event(), next.Event())
	// Assert that the next state has actions set. for the first step.
	require.Equal(t, 0, len(s.Actions()))
	require.NotEqualValues(t, s.Actions(), next.Actions())
	require.Equal(t, 1, len(next.Actions()))
	require.EqualValues(t, first, next.Actions()[w.Steps[0].ID])
	// Assert that requesting data for the given step ID works as expected.
	loaded, err := next.ActionID(w.Steps[0].ID)
	require.NoError(t, err)
	require.EqualValues(t, first, loaded)
	// And that we have no state for the second step.
	require.Empty(t, next.Actions()[w.Steps[1].ID])

	//
	// Check that saving a subsequent step saves the next output.
	//
	second := map[string]any{
		"another": "yea",
		"lol":     float64(1),
	}
	next, err = m.SaveActionOutput(ctx, s.Identifier(), w.Steps[1].ID, second)
	require.NoError(t, err)
	require.EqualValues(t, s.Identifier(), next.Identifier())
	require.EqualValues(t, s.Workflow(), next.Workflow())
	require.EqualValues(t, s.Event(), next.Event())
	// Assert that the next state has actions set. for the first step.
	require.Equal(t, 0, len(s.Actions()))
	require.NotEqualValues(t, s.Actions(), next.Actions())
	require.Equal(t, 2, len(next.Actions()))
	require.EqualValues(t, first, next.Actions()[w.Steps[0].ID])
	require.EqualValues(t, second, next.Actions()[w.Steps[1].ID])
	// Assert that requesting data for the given step ID works as expected.
	loaded, err = next.ActionID(w.Steps[0].ID)
	require.NoError(t, err)
	require.EqualValues(t, first, loaded)
	loaded, err = next.ActionID(w.Steps[1].ID)
	require.NoError(t, err)
	require.EqualValues(t, second, loaded)

	//
	// Load() the state independently.
	//
	reloaded, err := m.Load(ctx, s.Identifier())
	require.NoError(t, err)
	require.EqualValues(t, next.Identifier(), reloaded.Identifier())
	require.EqualValues(t, next.Workflow(), reloaded.Workflow())
	require.EqualValues(t, next.Event(), reloaded.Event())
	require.EqualValues(t, next.Actions(), reloaded.Actions())
	require.EqualValues(t, next.Errors(), reloaded.Errors())

	// TODO: Assert that we cannot save data to a run that does not exist.
	// TODO: Assert that we cannot save data to a step that doesn't exist.
	// TODO: Assert that we cannot overwrite data.
}

func checkSaveOutputClearsError(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	//
	// Save an error.
	//
	inputErr := fmt.Errorf("this is temporary, don't sweat it my friend")
	next, err := m.SaveActionError(ctx, s.Identifier(), w.Steps[0].ID, inputErr)
	require.NoError(t, err)
	require.EqualValues(t, s.Identifier(), next.Identifier())
	require.EqualValues(t, s.Workflow(), next.Workflow())
	require.EqualValues(t, s.Event(), next.Event())
	// Assert that the next state has an error and no action
	require.Equal(t, 1, len(next.Errors()))
	require.Equal(t, 0, len(next.Actions()))
	require.EqualValues(t, inputErr, next.Errors()[w.Steps[0].ID])
	require.False(t, next.ActionComplete(w.Steps[0].ID))

	//
	// Assert that saving output to a previously errored function clears
	// the action error.
	//
	output := map[string]any{
		"wut": "the",
		"gosh": map[string]any{
			"darn": "doot",
		},
	}
	next, err = m.SaveActionOutput(ctx, s.Identifier(), w.Steps[0].ID, output)
	require.NoError(t, err)
	require.EqualValues(t, s.Identifier(), next.Identifier())
	require.EqualValues(t, s.Workflow(), next.Workflow())
	require.EqualValues(t, s.Event(), next.Event())
	// Assert that the next state _now_ has an action and no error.
	require.Equal(t, 0, len(next.Errors()))
	require.Equal(t, 1, len(next.Actions()))
	require.Empty(t, next.Errors()[w.Steps[0].ID])
	require.EqualValues(t, output, next.Actions()[w.Steps[0].ID])
	require.True(t, next.ActionComplete(w.Steps[0].ID))
}

func checkSaveError(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	//
	// Save an error
	//
	inputErr := fmt.Errorf("an terrible, unlucky, impossible to debug error. woe betide the SRE who gets this :(")
	next, err := m.SaveActionError(ctx, s.Identifier(), w.Steps[0].ID, inputErr)
	require.NoError(t, err)
	require.EqualValues(t, s.Identifier(), next.Identifier())
	require.EqualValues(t, s.Workflow(), next.Workflow())
	require.EqualValues(t, s.Event(), next.Event())
	// Assert that the next state has actions set. for the first step.
	require.Equal(t, 0, len(s.Actions()))
	require.EqualValues(t, s.Actions(), next.Actions())
	require.Equal(t, 0, len(next.Actions()))
	// Assert that we have an error saved for the first step.
	require.Equal(t, 1, len(next.Errors()))
	require.EqualValues(t, inputErr, next.Errors()[w.Steps[0].ID])
	// Assert that loading this step produces an error.
	output, err := next.ActionID(w.Steps[0].ID)
	require.Empty(t, output)
	require.EqualValues(t, inputErr, err)
	// This action is not complete.
	require.False(t, next.ActionComplete(w.Steps[0].ID))

	//
	// Overwrite the error, as if an action retried.
	//
	inputErr = fmt.Errorf("wow, another one?!")
	next, err = m.SaveActionError(ctx, s.Identifier(), w.Steps[0].ID, inputErr)
	require.NoError(t, err)
	require.EqualValues(t, inputErr, next.Errors()[w.Steps[0].ID])
	require.False(t, next.ActionComplete(w.Steps[0].ID))

	//
	// Save an error to the new action.
	//

	//
	// Load() the state independently.
	//
	reloaded, err := m.Load(ctx, s.Identifier())
	require.NoError(t, err)
	require.EqualValues(t, next.Identifier(), reloaded.Identifier())
	require.EqualValues(t, next.Workflow(), reloaded.Workflow())
	require.EqualValues(t, next.Event(), reloaded.Event())
	require.EqualValues(t, next.Actions(), reloaded.Actions())
	require.EqualValues(t, next.Errors(), reloaded.Errors())

	// Maybe we also want to assert that we can't save an error to an
	// action that has output.
}

func checkSavePause(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	// Save a pause.
	pause := state.Pause{
		ID:         uuid.New(),
		Identifier: s.Identifier(),
		Outgoing:   inngest.TriggerName,
		Incoming:   w.Steps[0].ID,
		Expires:    time.Now().Add(5 * time.Second),
	}
	err := m.SavePause(ctx, pause)
	require.NoError(t, err)

	// XXX: Saving a pause with a past expiry is a noop.
}

func checkLeasePause(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	// Leasing a non-existent pause should error.
	err := m.LeasePause(ctx, uuid.New())
	require.Equal(t, state.ErrPauseNotFound, err, "leasing a non-existent pause should return state.ErrPauseNotFound")

	// Save a pause.
	pause := state.Pause{
		ID:         uuid.New(),
		Identifier: s.Identifier(),
		Outgoing:   inngest.TriggerName,
		Incoming:   w.Steps[0].ID,
		Expires:    time.Now().Add(state.PauseLeaseDuration * 2),
	}
	err = m.SavePause(ctx, pause)
	require.NoError(t, err)

	now := time.Now()

	// Leasing the pause should work.
	err = m.LeasePause(ctx, pause.ID)
	require.NoError(t, err)

	// And we should not be able to re-lease the pause until the pause lease duration is up.
	for time.Now().Before(now.Add(state.PauseLeaseDuration)) {
		err = m.LeasePause(ctx, pause.ID)
		require.NotNil(t, err, "Re-leasing a pause with a valid lease should error")
		require.Error(t, state.ErrPauseLeased, err)
		<-time.After(state.PauseLeaseDuration / 100)
	}

	// And again, once the lease is up, we should be able to lease the pause.
	err = m.LeasePause(ctx, pause.ID)
	require.NoError(t, err)

	//
	// Assert that leasing an expired pause fails.
	//

	pause = state.Pause{
		ID:         uuid.New(),
		Identifier: s.Identifier(),
		Outgoing:   inngest.TriggerName,
		Incoming:   w.Steps[0].ID,
		Expires:    time.Now().Add(10 * time.Millisecond),
	}
	<-time.After(15 * time.Millisecond)
	err = m.LeasePause(ctx, pause.ID)
	require.NotNil(t, err)
	require.Error(t, state.ErrPauseNotFound, err)
}

func checkConsumePause(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	// Consuming a non-existent pause should error.
	err := m.ConsumePause(ctx, uuid.New())
	require.Equal(t, state.ErrPauseNotFound, err, "Consuming a non-existent pause should return state.ErrPauseNotFound")

	// Save a pause.
	pause := state.Pause{
		ID:         uuid.New(),
		Identifier: s.Identifier(),
		Outgoing:   inngest.TriggerName,
		Incoming:   w.Steps[0].ID,
		Expires:    time.Now().Add(state.PauseLeaseDuration * 2),
	}
	err = m.SavePause(ctx, pause)
	require.NoError(t, err)

	// TODO: Do we want to enforce leasing of a pause prior to consuming it?

	// Consuming the pause should work.
	err = m.ConsumePause(ctx, pause.ID)
	require.NoError(t, err)

	err = m.ConsumePause(ctx, pause.ID)
	require.NotNil(t, err)
	require.Error(t, state.ErrPauseNotFound, err)

	//
	// Assert that completing a leased pause fails.
	//
	pause = state.Pause{
		ID:         uuid.New(),
		Identifier: s.Identifier(),
		Outgoing:   inngest.TriggerName,
		Incoming:   w.Steps[0].ID,
		Expires:    time.Now().Add(10 * time.Millisecond),
	}
	<-time.After(15 * time.Millisecond)
	err = m.ConsumePause(ctx, pause.ID)
	require.NotNil(t, err, "Consuming an expired pause should error")
	require.Error(t, state.ErrPauseNotFound, err)
}

func setup(t *testing.T, m state.Manager) state.State {
	ctx := context.Background()
	w.UUID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(w.ID))
	id := ulid.MustNew(ulid.Now(), rand.Reader)

	s, err := m.New(ctx, w, id, input.Map())
	require.NoError(t, err)
	return s
}
