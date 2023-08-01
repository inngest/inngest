package testharness

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	mrand "math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	// n100 is a workflow created during init() which has 100 steps and edges.
	n100 = inngest.Function{
		Name: "Test workflow",
		ID:   uuid.NewSHA1(uuid.NameSpaceOID, []byte("Test workflow")),
	}

	w = inngest.Function{
		ID:   uuid.NewSHA1(uuid.NameSpaceOID, []byte("Test workflow")),
		Name: "Test workflow",
		Triggers: []inngest.Trigger{
			{
				EventTrigger: &inngest.EventTrigger{
					Event: "test/some.event",
				},
			},
		},
		Steps: []inngest.Step{
			{
				ID:   "step-a",
				Name: "first step",
				URI:  "http://www.example.com/api/inngest",
			},
			{
				ID:   "step-b",
				Name: "second step",
				URI:  "http://www.example.com/api/inngest",
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

func FunctionLoader() state.FunctionLoader {
	return loader{}
}

type loader struct{}

func (loader) LoadFunction(ctx context.Context, identifier state.Identifier) (*inngest.Function, error) {
	if identifier.WorkflowID == w.ID {
		return &w, nil
	}
	if identifier.WorkflowID == n100.ID {
		return &n100, nil
	}
	return nil, fmt.Errorf("workflow not found: %s", identifier.WorkflowID)
}

func init() {
	// Copy the workflow and make 1000 scheduled steps.
	for i := 1; i <= 100; i++ {
		n100.Steps = append(n100.Steps, inngest.Step{
			ID:   fmt.Sprintf("step-%d", i),
			Name: fmt.Sprintf("Step %d", i),
			URI:  "http://www.example.com/api/inngest",
		})
		n100.Edges = append(n100.Edges, inngest.Edge{
			Incoming: inngest.TriggerName,
			Outgoing: fmt.Sprintf("step-%d", i),
		})
	}
}

type Generator func() (sm state.Manager, cleanup func())

func CheckState(t *testing.T, gen Generator) {
	t.Helper()

	funcs := map[string]func(t *testing.T, m state.Manager){
		"New":                                checkNew,
		"New/StepData":                       checkNew_stepdata,
		"Scheduled":                          checkScheduled,
		"SaveResponse/Output":                checkSaveResponse_output,
		"SaveResponse/Error":                 checkSaveResponse_error,
		"SaveResponse/OutputOverwritesError": checkSaveResponse_outputOverwritesError,
		"SaveResponse/Concurrent":            checkSaveResponse_concurrent,
		"SaveResponse/Stack":                 checkSaveResponse_stack,
		"SavePause":                          checkSavePause,
		"LeasePause":                         checkLeasePause,
		"ConsumePause":                       checkConsumePause,
		"ConsumePause/WithData":              checkConsumePauseWithData,
		"ConsumePause/WithData/StackIndex":   checkConsumePauseWithDataIndex,
		"ConsumePause/WithEmptyData":         checkConsumePauseWithEmptyData,
		"ConsumePause/WithEmptyDataKey":      checkConsumePauseWithEmptyDataKey,
		"PausesByEvent/Empty":                checkPausesByEvent_empty,
		"PausesByEvent/Single":               checkPausesByEvent_single,
		"PausesByEvent/Multiple":             checkPausesByEvent_multi,
		"PausesByEvent/ConcurrentCursors":    checkPausesByEvent_concurrent,
		"PausesByEvent/Consumed":             checkPausesByEvent_consumed,
		"PauseByStep":                        checkPausesByStep,
		"PauseByID":                          checkPauseByID,
		"Idempotency":                        checkIdempotency,
		"SetStatus":                          checkSetStatus,
		"Cancel":                             checkCancel,
		"Cancel/AlreadyCompleted":            checkCancel_completed,
		"Cancel/AlreadyCancelled":            checkCancel_cancelled,
		"Finalized/Status":                   checkFinalizedStatus,
		"Log/FunctionLog":                    checkLogs,
	}
	for name, f := range funcs {
		t.Run(name, func(t *testing.T) {
			t.Helper()
			m, cleanup := gen()
			f(t, m)
			cleanup()
		})
	}
}

func checkNew(t *testing.T, m state.Manager) {
	ctx := context.Background()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	id := state.Identifier{
		WorkflowID:      w.ID,
		WorkflowVersion: w.FunctionVersion,
		RunID:           runID,
		Key:             runID.String(),
	}

	evt := input.Map()
	batch := []map[string]any{input.Map()}

	init := state.Input{
		Identifier:     id,
		EventBatchData: batch,
		Context: map[string]any{
			"some": "data",
			"true": true,
		},
	}

	s, err := m.New(ctx, init)
	require.NoError(t, err)

	found := s.Function()
	require.EqualValues(t, w, found, "Returned workflow does not match input")
	require.EqualValues(t, evt, s.Event(), "Returned event does not match input")
	require.EqualValues(t, batch, s.Events(), "Returned events does not match input")
	require.EqualValues(t, enums.RunStatusRunning, s.Metadata().Status, "Status is not Running")
	require.EqualValues(t, init.Context, s.Metadata().Context, "Metadata context not saved")
	require.EqualValues(t, id, s.Metadata().Identifier, "Metadata didn't save Identifier")

	loaded, err := m.Load(ctx, s.RunID())
	require.NoError(t, err)

	found = loaded.Function()
	require.EqualValues(t, w, found, "Loaded workflow does not match input")
	require.EqualValues(t, input.Map(), loaded.Event(), "Loaded event does not match input")

	metadata := loaded.Metadata()
	require.Equal(t, 1, metadata.Pending, "New should set pending count to 1")
}

// checkNew_stepdata ensures that state stores can be initialized with
// predetermined step data.
func checkNew_stepdata(t *testing.T, m state.Manager) {
	ctx := context.Background()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	id := state.Identifier{
		WorkflowID: w.ID,
		RunID:      runID,
		Key:        runID.String(),
	}

	evt := input.Map()
	batch := []map[string]any{evt}

	init := state.Input{
		Identifier:     id,
		EventBatchData: batch,
		Steps: map[string]any{
			"step-a": map[string]any{
				"result": "predetermined",
			},
		},
	}

	s, err := m.New(ctx, init)
	require.NoError(t, err)

	found := s.Function()
	require.EqualValues(t, w, found, "Returned workflow does not match input")
	require.EqualValues(t, evt, s.Event(), "Returned event does not match input")
	require.EqualValues(t, batch, s.Events(), "Returned events does not match input")

	loaded, err := m.Load(ctx, s.RunID())
	require.NoError(t, err)

	found = loaded.Function()
	require.EqualValues(t, w, found, "Loaded workflow does not match input")
	require.EqualValues(t, input.Map(), loaded.Event(), "Loaded event does not match input")

	metadata := loaded.Metadata()
	require.Equal(t, 1, metadata.Pending, "New should set pending count to 1")

	data := loaded.Actions()
	require.Equal(t, 1, len(data), "New should store predetermined step data")
	require.Equal(t, init.Steps["step-a"], data["step-a"], "New should store predetermined step data")
}

func checkScheduled(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	wg := &sync.WaitGroup{}
	for i := 0; i < len(n100.Steps); i++ {
		n := i
		wg.Add(1)
		go func() {
			err := m.Scheduled(ctx, s.Identifier(), n100.Steps[n].ID, 0, nil)
			wg.Done()
			require.NoError(t, err)
		}()
	}

	wg.Wait()

	// Load the state again.
	loaded, err := m.Load(ctx, s.RunID())
	require.NoError(t, err)
	// The trigger is always pending, and we do not account for this when
	// initializing in setup()
	require.EqualValues(t, len(n100.Steps)+1, loaded.Metadata().Pending, "Scheduling 100 steps concurrently should return the correct pending count via metadata")
}

// checkSaveResponse_output checks the basics of saving output from a response.
//
// This asserts that the state store records output for the given step, by saving
// output for two independent responses.
func checkSaveResponse_output(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	r := state.DriverResponse{
		Step: w.Steps[0],
		Output: map[string]interface{}{
			"status": float64(200),
			"body": map[string]any{
				"ok": true,
			},
		},
	}

	_, err := m.SaveResponse(ctx, s.Identifier(), r, 0)
	require.NoError(t, err)

	next, err := m.Load(ctx, s.Identifier().RunID)
	require.NoError(t, err)
	require.NotNil(t, next)

	// Ensure basics haven't changed.
	require.EqualValues(t, s.Identifier(), next.Identifier())
	require.EqualValues(t, s.Function(), next.Function())
	require.EqualValues(t, s.Event(), next.Event())

	// Assert that the next state has actions set. for the first step.
	require.Equal(t, 0, len(s.Actions()))
	require.NotEqualValues(t, s.Actions(), next.Actions())
	require.Equal(t, 1, len(next.Actions()))
	require.EqualValues(t, r.Output, next.Actions()[w.Steps[0].ID])

	// Assert that requesting data for the given step ID works as expected.
	loaded, err := next.ActionID(w.Steps[0].ID)
	require.NoError(t, err)
	require.EqualValues(t, r.Output, loaded)

	// And that we have no state for the second step.
	require.Empty(t, next.Actions()[w.Steps[1].ID])
	require.Equal(t, 1, next.Metadata().Pending)

	//
	// Check that saving a subsequent step saves the next output,
	// as the second attempt.
	//
	r2 := state.DriverResponse{
		Step: w.Steps[1],
		Output: map[string]interface{}{
			"status": float64(200),
			"body": map[string]any{
				"another": "yea",
				"lol":     float64(1),
			},
		},
	}

	_, err = m.SaveResponse(ctx, s.Identifier(), r2, 1)
	require.NoError(t, err)

	next, err = m.Load(ctx, s.Identifier().RunID)
	require.NoError(t, err)
	require.NotNil(t, next)

	require.EqualValues(t, s.Identifier(), next.Identifier())
	require.EqualValues(t, s.Function(), next.Function())
	require.EqualValues(t, s.Event(), next.Event())
	// Assert that the next state has actions set. for the first step.
	require.Equal(t, 0, len(s.Actions()))
	require.NotEqualValues(t, s.Actions(), next.Actions())
	require.Equal(t, 2, len(next.Actions()))
	require.EqualValues(t, r.Output, next.Actions()[w.Steps[0].ID])
	require.EqualValues(t, r2.Output, next.Actions()[w.Steps[1].ID])
	// Assert that requesting data for the given step ID works as expected.
	loaded, err = next.ActionID(w.Steps[0].ID)
	require.NoError(t, err)
	require.EqualValues(t, r.Output, loaded)
	loaded, err = next.ActionID(w.Steps[1].ID)
	require.NoError(t, err)
	require.EqualValues(t, r2.Output, loaded)
	// Output shouldn't be finalized until edges are added via the runner.
	require.Equal(t, 1, next.Metadata().Pending)

	err = m.Finalized(ctx, s.Identifier(), w.Steps[0].ID, 0)
	require.NoError(t, err)
	err = m.Finalized(ctx, s.Identifier(), w.Steps[1].ID, 0)
	require.NoError(t, err)

	//
	// Load() the state independently.
	//
	reloaded, err := m.Load(ctx, s.RunID())
	require.NoError(t, err)
	require.EqualValues(t, next.Identifier(), reloaded.Identifier())
	require.EqualValues(t, next.Function(), reloaded.Function())
	require.EqualValues(t, next.Event(), reloaded.Event())
	require.EqualValues(t, next.Actions(), reloaded.Actions())
	require.EqualValues(t, next.Errors(), reloaded.Errors())

	// Check metadata:  we must have two finalized steps.
	require.Equal(t, -1, reloaded.Metadata().Pending)
}

func checkSaveResponse_error(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)
	_, err := m.Load(ctx, s.RunID())
	require.NoError(t, err)

	r := state.DriverResponse{
		Step: w.Steps[0],
		Err:  strptr("an absolutely terrible yet intermittent, non-final, retryable error"),
	}
	require.True(t, r.Retryable())
	require.False(t, r.Final())

	_, err = m.SaveResponse(ctx, s.Identifier(), r, 0)
	require.NoError(t, err)

	next, err := m.Load(ctx, s.Identifier().RunID)
	require.NoError(t, err)
	require.NotNil(t, next)

	require.Nil(t, next.Actions()[r.Step.ID])
	require.NotNil(t, r.Err)
	require.Contains(t, next.Errors()[r.Step.ID].Error(), *r.Err)

	// Only the trigger, which was not yet complete.
	require.Equal(t, 1, next.Metadata().Pending)

	// Overwriting the error by setting as final should work and should
	// finalize the error.
	r.SetFinal()

	require.False(t, r.Retryable())
	require.True(t, r.Final())

	_, err = m.SaveResponse(ctx, s.Identifier(), r, 1)
	require.NoError(t, err)

	finalized, err := m.Load(ctx, s.Identifier().RunID)
	require.NoError(t, err)
	require.NotNil(t, next)

	require.Nil(t, finalized.Actions()[r.Step.ID])
	require.NotNil(t, r.Err)
	require.Contains(t, finalized.Errors()[r.Step.ID].Error(), *r.Err)

	// Next stores an outdated reference
	require.Equal(t, 1, next.Metadata().Pending)
	// But finalized should increaase finalized count.
	require.Equal(t, 0, finalized.Metadata().Pending, "finalized error does not decrease pending count")
	require.Equal(t, enums.RunStatusFailed, finalized.Metadata().Status, "finalized error does not set status to failed")
}

func checkSaveResponse_outputOverwritesError(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	stepErr := strptr("an absolutely terrible yet intermittent, non-final, retryable error")
	r := state.DriverResponse{
		Step: w.Steps[0],
		Err:  stepErr,
	}
	require.True(t, r.Retryable())
	require.False(t, r.Final())

	_, err := m.SaveResponse(ctx, s.Identifier(), r, 0)
	require.NoError(t, err)

	next, err := m.Load(ctx, s.Identifier().RunID)
	require.NoError(t, err)
	require.NotNil(t, next)

	require.Nil(t, next.Actions()[r.Step.ID])
	require.NotNil(t, r.Err)
	require.Contains(t, next.Errors()[r.Step.ID].Error(), *r.Err)

	// This is not final
	require.Equal(t, 1, next.Metadata().Pending)

	r.Err = nil
	r.Output = map[string]interface{}{
		"u wot": "m8",
	}
	require.False(t, r.Final())

	_, err = m.SaveResponse(ctx, s.Identifier(), r, 1)
	require.NoError(t, err)

	finalized, err := m.Load(ctx, s.Identifier().RunID)
	require.NoError(t, err)
	require.NotNil(t, finalized)

	require.Equal(t, r.Output, finalized.Actions()[r.Step.ID])
	// The error is still stored.
	require.NotNil(t, stepErr)
	require.Contains(t, next.Errors()[r.Step.ID].Error(), *stepErr)
	// Saving output should not finalize.
	require.Equal(t, 1, finalized.Metadata().Pending)

	err = m.Finalized(ctx, s.Identifier(), w.Steps[0].ID, 0)
	require.NoError(t, err)

	reloaded, err := m.Load(ctx, s.RunID())
	require.NoError(t, err)
	require.Equal(t, 0, reloaded.Metadata().Pending)
}

func checkSaveResponse_concurrent(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)
	id := s.Identifier()

	wg := &sync.WaitGroup{}
	for i := 0; i < len(n100.Steps); i++ {
		n := i
		wg.Add(2)

		go func() {
			err := m.Scheduled(ctx, s.Identifier(), n100.Steps[n].ID, 0, nil)
			wg.Done()
			require.NoError(t, err)
		}()

		go func() {
			defer wg.Done()
			r := state.DriverResponse{
				Step: n100.Steps[n],
				Output: map[string]interface{}{
					"status": float64(200),
					"body": map[string]any{
						"n": n,
					},
				},
			}
			_, err := m.SaveResponse(ctx, id, r, mrand.Intn(3))
			require.NoError(t, err)
			err = m.Finalized(ctx, s.Identifier(), n100.Steps[n].ID, 0)
			require.NoError(t, err)
		}()

	}
	wg.Wait()

	loaded, err := m.Load(ctx, s.RunID())
	require.NoError(t, err)
	require.EqualValues(t, 1, loaded.Metadata().Pending, "scheduling and finalizing concurrently should end with 1 (the trigger)")
}

func checkSaveResponse_stack(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	t.Run("It modifies the stack with step output", func(t *testing.T) {
		r := state.DriverResponse{
			Step: w.Steps[0],
			Output: map[string]interface{}{
				"status": float64(200),
				"body": map[string]any{
					"ok": true,
				},
			},
		}
		idx, err := m.SaveResponse(ctx, s.Identifier(), r, 0)
		require.NoError(t, err)
		require.EqualValues(t, 1, idx)

		next, err := m.Load(ctx, s.Identifier().RunID)
		require.NoError(t, err)

		stack := next.Stack()
		require.EqualValues(t, 1, len(stack))
		require.Equal(t, []string{w.Steps[0].ID}, stack)
	})

	t.Run("It doesn't amend the stack with temporary non-final errors", func(t *testing.T) {
		r := state.DriverResponse{
			Step: w.Steps[1],
			Err:  strptr("an absolutely terrible yet intermittent, non-final, retryable error"),
		}
		require.True(t, r.Retryable())
		require.False(t, r.Final())

		idx, err := m.SaveResponse(ctx, s.Identifier(), r, 0)
		require.NoError(t, err)
		// The stack should still be 1 from the previous step.
		require.EqualValues(t, 1, idx)
		next, err := m.Load(ctx, s.Identifier().RunID)
		require.NoError(t, err)
		stack := next.Stack()
		require.EqualValues(t, 1, len(stack))
		require.Equal(t, []string{w.Steps[0].ID}, stack)
	})

	t.Run("It modifies the stack with a final error", func(t *testing.T) {
		r := state.DriverResponse{
			Step: w.Steps[1],
			Err:  strptr("a permanent error"),
		}
		r.SetFinal()
		require.False(t, r.Retryable())
		require.True(t, r.Final())

		idx, err := m.SaveResponse(ctx, s.Identifier(), r, 0)
		require.NoError(t, err)
		require.EqualValues(t, 2, idx)

		next, err := m.Load(ctx, s.Identifier().RunID)
		require.NoError(t, err)

		stack := next.Stack()
		require.EqualValues(t, 2, len(stack))
		require.Equal(t, []string{w.Steps[0].ID, w.Steps[1].ID}, stack)
	})
}

func checkSavePause(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	history, err := m.History(ctx, s.RunID())
	require.NoError(t, err)
	require.Equal(t, 1, len(history))

	// Save a pause.
	pause := state.Pause{
		ID:         uuid.New(),
		Identifier: s.Identifier(),
		Outgoing:   inngest.TriggerName,
		Incoming:   w.Steps[0].ID,
		Expires:    state.Time(time.Now().Add(5 * time.Second)),
	}
	err = m.SavePause(ctx, pause)
	require.NoError(t, err)

	t.Run("It stores history", func(t *testing.T) {
		history, err := m.History(ctx, s.RunID())
		require.NoError(t, err)
		require.Equal(t, 2, len(history))
		require.Equal(t, enums.HistoryTypeStepWaiting, history[len(history)-1].Type)
	})

	// XXX: Saving a pause with a past expiry is a noop.
}

func checkLeasePause(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	// Leasing a non-existent pause should error.
	err := m.LeasePause(ctx, uuid.New())
	assert.Equal(t, state.ErrPauseNotFound, err, "leasing a non-existent pause should return state.ErrPauseNotFound")

	// Save a pause.
	pause := state.Pause{
		ID:         uuid.New(),
		Identifier: s.Identifier(),
		Outgoing:   inngest.TriggerName,
		Incoming:   w.Steps[0].ID,
		Expires:    state.Time(time.Now().Add(state.PauseLeaseDuration * 3).UTC()),
	}
	err = m.SavePause(ctx, pause)
	require.NoError(t, err)

	now := time.Now()

	var errors int32
	var wg sync.WaitGroup

	tick := time.Now().Add(2 * time.Second).Truncate(time.Second)

	// Leasing the pause should work once over 50 parallel attempts
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			// Only one of these should work.
			<-time.After(time.Until(tick))
			err := m.LeasePause(ctx, pause.ID)
			if err != nil {
				atomic.AddInt32(&errors, 1)
			}
			wg.Done()
		}()
	}

	wg.Wait()
	require.EqualValues(t, int32(99), errors)

	// Fetch the pause and ensure it's formatted appropriately
	fetched, err := m.PauseByID(ctx, pause.ID)
	require.Nil(t, err)
	require.Equal(t, pause.Expires.Time().Truncate(time.Millisecond), fetched.Expires.Time().Truncate(time.Millisecond))
	require.Equal(t, pause.Identifier, fetched.Identifier)
	require.Equal(t, pause.Outgoing, fetched.Outgoing)
	require.Equal(t, pause.Incoming, fetched.Incoming)

	// And we should not be able to re-lease the pause until the pause lease duration is up.
	for time.Now().Before(now.Add(state.PauseLeaseDuration - (5 * time.Millisecond))) {
		err = m.LeasePause(ctx, pause.ID)
		require.NotNil(t, err, "Re-leasing a pause with a valid lease should error")
		require.Error(t, state.ErrPauseLeased, err)
		<-time.After(state.PauseLeaseDuration / 50)
	}

	<-time.After(state.PauseLeaseDuration)

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
		Expires:    state.Time(time.Now().Add(10 * time.Millisecond)),
	}
	<-time.After(15 * time.Millisecond)
	err = m.LeasePause(ctx, pause.ID)
	require.NotNil(t, err, "Leasing an expired pause should fail")
	require.Error(t, state.ErrPauseNotFound, err, "Leasing an expired pause should fail with ErrPauseNotFound")
}

func checkConsumePause(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	// Consuming a non-existent pause should error.
	err := m.ConsumePause(ctx, uuid.New(), nil)
	require.Equal(t, state.ErrPauseNotFound, err, "Consuming a non-existent pause should return state.ErrPauseNotFound")

	// Save a pause.
	pause := state.Pause{
		ID:         uuid.New(),
		Identifier: s.Identifier(),
		Outgoing:   inngest.TriggerName,
		Incoming:   w.Steps[0].ID,
		StepName:   w.Steps[0].Name,
		Expires:    state.Time(time.Now().Add(state.PauseLeaseDuration * 2)),
	}
	err = m.SavePause(ctx, pause)
	require.NoError(t, err)

	// There now should be 2 items.
	history, err := m.History(ctx, s.RunID())
	require.NoError(t, err)
	require.Equal(t, 2, len(history))
	require.Equal(t, enums.HistoryTypeStepWaiting, history[len(history)-1].Type)

	t.Run("Consuming a pause works", func(t *testing.T) {
		// Add 1ms, ensuring that the step completed history
		// item is always after the pause history item. history is MS precision,
		// and without this there's a small but real chance of flakiness.
		<-time.After(time.Millisecond)
		// Consuming the pause should work.
		err = m.ConsumePause(ctx, pause.ID, nil)
		require.NoError(t, err)
	})

	t.Run("It stores history saying step completed", func(t *testing.T) {
		history, err := m.History(ctx, s.RunID())
		require.NoError(t, err)
		require.Equal(t, 3, len(history))
		last := history[len(history)-1]
		require.Equal(t, enums.HistoryTypeStepCompleted, last.Type)
		hs, ok := last.Data.(state.HistoryStep)
		require.True(t, ok)
		require.Equal(t, w.Steps[0].ID, hs.ID)
		require.Equal(t, w.Steps[0].Name, hs.Name)
	})

	t.Run("Consuming a pause again fails", func(t *testing.T) {
		err = m.ConsumePause(ctx, pause.ID, nil)
		require.NotNil(t, err)
		require.Error(t, state.ErrPauseNotFound, err)
	})

	//
	// Assert that completing a leased pause fails.
	//
	pause = state.Pause{
		ID:         uuid.New(),
		Identifier: s.Identifier(),
		Outgoing:   inngest.TriggerName,
		Incoming:   w.Steps[0].ID,
		Expires:    state.Time(time.Now().Add(10 * time.Millisecond)),
	}
	<-time.After(15 * time.Millisecond)
	err = m.ConsumePause(ctx, pause.ID, nil)
	require.NotNil(t, err, "Consuming an expired pause should error")
	require.Error(t, state.ErrPauseNotFound, err)
}

func checkConsumePauseWithData(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	pauseData := map[string]any{
		"did this work?": true,
	}

	// Consuming a non-existent pause should error.
	err := m.ConsumePause(ctx, uuid.New(), pauseData)
	require.Equal(t, state.ErrPauseNotFound, err, "Consuming a non-existent pause should return state.ErrPauseNotFound")

	// Save a pause.
	pause := state.Pause{
		ID:         uuid.New(),
		Identifier: s.Identifier(),
		Outgoing:   inngest.TriggerName,
		Incoming:   w.Steps[0].ID,
		Expires:    state.Time(time.Now().Add(state.PauseLeaseDuration * 2)),
		DataKey:    "my-pause-data-stored-for-eternity",
	}
	err = m.SavePause(ctx, pause)
	require.NoError(t, err)

	// Consuming the pause should work.
	err = m.ConsumePause(ctx, pause.ID, pauseData)
	require.NoError(t, err)

	err = m.ConsumePause(ctx, pause.ID, pauseData)
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
		Expires:    state.Time(time.Now().Add(10 * time.Millisecond)),
		DataKey:    "my-pause-data-stored-for-eternity",
	}
	<-time.After(15 * time.Millisecond)
	err = m.ConsumePause(ctx, pause.ID, pauseData)
	require.NotNil(t, err, "Consuming an expired pause should error")
	require.Error(t, state.ErrPauseNotFound, err)

	// Load function state and assert we have the pause stored in state.
	reloaded, err := m.Load(ctx, s.RunID())
	require.Nil(t, err)
	require.Equal(t, pauseData, reloaded.Actions()[pause.DataKey], "Pause data was not stored in the state store")
}

func checkConsumePauseWithDataIndex(t *testing.T, m state.Manager) {
	key := "my-pause-data-stored-for-eternity"

	t.Run("it updates the stack with nil data", func(t *testing.T) {
		ctx := context.Background()
		s := setup(t, m)

		// Save a pause.
		pause := state.Pause{
			ID:         uuid.New(),
			Identifier: s.Identifier(),
			Outgoing:   inngest.TriggerName,
			Incoming:   w.Steps[0].ID,
			Expires:    state.Time(time.Now().Add(state.PauseLeaseDuration * 2)),
			DataKey:    key,
		}
		err := m.SavePause(ctx, pause)
		require.NoError(t, err)

		// Consuming the pause should work.
		err = m.ConsumePause(ctx, pause.ID, nil)
		require.NoError(t, err)

		// Load function state and assert we have the pause stored in state.
		reloaded, err := m.Load(ctx, s.RunID())
		require.Nil(t, err)

		require.Equal(t, 1, len(reloaded.Stack()))
		require.Equal(t, key, reloaded.Stack()[0])

		require.Equal(t, 1, len(reloaded.Actions()))
		require.Equal(t, nil, reloaded.Actions()[key])
	})

	t.Run("it updates the stack with actual data", func(t *testing.T) {
		ctx := context.Background()
		s := setup(t, m)

		// Load function state and assert we have the pause stored in state.
		loaded, err := m.Load(ctx, s.RunID())
		require.Nil(t, err)
		require.Equal(t, 0, len(loaded.Stack()))

		// Save a pause.
		pause := state.Pause{
			ID:         uuid.New(),
			Identifier: s.Identifier(),
			Outgoing:   inngest.TriggerName,
			Incoming:   w.Steps[0].ID,
			Expires:    state.Time(time.Now().Add(state.PauseLeaseDuration * 2)),
			DataKey:    key,
		}
		err = m.SavePause(ctx, pause)
		require.NoError(t, err)

		data := map[string]any{"allo": "guvna"}

		// Consuming the pause should work.
		err = m.ConsumePause(ctx, pause.ID, data)
		require.NoError(t, err)

		// Load function state and assert we have the pause stored in state.
		reloaded, err := m.Load(ctx, s.RunID())
		require.Nil(t, err)

		require.Equal(t, 1, len(reloaded.Stack()))
		require.Equal(t, key, reloaded.Stack()[0])

		require.Equal(t, 1, len(reloaded.Actions()))
		require.Equal(t, data, reloaded.Actions()[key])
	})
}

func checkConsumePauseWithEmptyData(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	// Consuming a non-existent pause should error.
	err := m.ConsumePause(ctx, uuid.New(), nil)
	require.Equal(t, state.ErrPauseNotFound, err, "Consuming a non-existent pause should return state.ErrPauseNotFound")

	// Save a pause.
	pause := state.Pause{
		ID:         uuid.New(),
		Identifier: s.Identifier(),
		Outgoing:   inngest.TriggerName,
		Incoming:   w.Steps[0].ID,
		Expires:    state.Time(time.Now().Add(state.PauseLeaseDuration * 2)),
		DataKey:    "my-pause-data-stored-for-eternity",
	}
	err = m.SavePause(ctx, pause)
	require.NoError(t, err)

	// Consuming the pause should work.
	err = m.ConsumePause(ctx, pause.ID, nil)
	require.NoError(t, err)

	err = m.ConsumePause(ctx, pause.ID, nil)
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
		Expires:    state.Time(time.Now().Add(10 * time.Millisecond)),
		DataKey:    "my-pause-data-stored-for-eternity",
	}
	<-time.After(15 * time.Millisecond)
	err = m.ConsumePause(ctx, pause.ID, nil)
	require.NotNil(t, err, "Consuming an expired pause should error")
	require.Error(t, state.ErrPauseNotFound, err)

	// Load function state and assert we have the pause stored in state.
	reloaded, err := m.Load(ctx, s.RunID())
	require.Nil(t, err)
	require.Equal(t, 1, len(reloaded.Actions()), "Pause data should still be stored if data is nil")
}

func checkConsumePauseWithEmptyDataKey(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	pauseData := map[string]any{
		"did this work?": true,
	}

	// Consuming a non-existent pause should error.
	err := m.ConsumePause(ctx, uuid.New(), pauseData)
	require.Equal(t, state.ErrPauseNotFound, err, "Consuming a non-existent pause should return state.ErrPauseNotFound")

	// Save a pause.
	pause := state.Pause{
		ID:         uuid.New(),
		Identifier: s.Identifier(),
		Outgoing:   inngest.TriggerName,
		Incoming:   w.Steps[0].ID,
		Expires:    state.Time(time.Now().Add(state.PauseLeaseDuration * 2)),
	}
	err = m.SavePause(ctx, pause)
	require.NoError(t, err)

	// Consuming the pause should work.
	err = m.ConsumePause(ctx, pause.ID, pauseData)
	require.NoError(t, err)

	err = m.ConsumePause(ctx, pause.ID, pauseData)
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
		Expires:    state.Time(time.Now().Add(10 * time.Millisecond)),
	}
	<-time.After(15 * time.Millisecond)
	err = m.ConsumePause(ctx, pause.ID, pauseData)
	require.NotNil(t, err, "Consuming an expired pause should error")
	require.Error(t, state.ErrPauseNotFound, err)

	// Load function state and assert we have the pause stored in state.
	reloaded, err := m.Load(ctx, s.RunID())
	require.Nil(t, err)
	require.Equal(t, 0, len(reloaded.Actions()), "Pause data was stored in the state store with no data key provided")
}

func checkPausesByEvent_empty(t *testing.T, m state.Manager) {
	ctx := context.Background()

	iter, err := m.PausesByEvent(ctx, uuid.UUID{}, "lol/nothing.my.friend")
	require.NoError(t, err)
	require.NotNil(t, iter)
	require.False(t, iter.Next(ctx))
	require.Nil(t, iter.Val(ctx))

	exists, err := m.EventHasPauses(ctx, uuid.UUID{}, "lol/nothing.my.friend")
	require.NoError(t, err)
	require.False(t, exists)
}

func checkPausesByEvent_single(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	evtA := "event/a"
	evtB := "event/...b"
	wsA := uuid.New()
	wsB := uuid.New()

	// Save a pause.
	pause := state.Pause{
		ID:          uuid.New(),
		WorkspaceID: wsA,
		Identifier:  s.Identifier(),
		Outgoing:    inngest.TriggerName,
		Incoming:    w.Steps[0].ID,
		Expires:     state.Time(time.Now().Add(state.PauseLeaseDuration * 2).Truncate(time.Millisecond).UTC()),
		Event:       &evtA,
	}
	err := m.SavePause(ctx, pause)
	require.NoError(t, err)

	// Save an unrelated pause to another event in the same workspace
	unusedA := state.Pause{
		ID:          uuid.New(),
		WorkspaceID: wsA,
		Identifier:  s.Identifier(),
		Outgoing:    inngest.TriggerName,
		Incoming:    w.Steps[0].ID,
		Expires:     state.Time(time.Now().Add(state.PauseLeaseDuration * 2).Truncate(time.Millisecond).UTC()),
		Event:       &evtB,
	}
	err = m.SavePause(ctx, unusedA)
	require.NoError(t, err)

	// Save an unrelated pause to the same event in a different workspace
	unusedB := state.Pause{
		ID:          uuid.New(),
		WorkspaceID: wsB,
		Identifier:  s.Identifier(),
		Outgoing:    inngest.TriggerName,
		Incoming:    w.Steps[0].ID,
		Expires:     state.Time(time.Now().Add(state.PauseLeaseDuration * 2).Truncate(time.Millisecond).UTC()),
		Event:       &evtA,
	}
	err = m.SavePause(ctx, unusedB)
	require.NoError(t, err)

	exists, err := m.EventHasPauses(ctx, wsA, evtA)
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = m.EventHasPauses(ctx, wsB, evtA)
	require.NoError(t, err)
	require.True(t, exists)

	iter, err := m.PausesByEvent(ctx, wsA, evtA)
	require.NoError(t, err)
	require.NotNil(t, iter)
	require.True(t, iter.Next(ctx))
	require.EqualValues(t, &pause, iter.Val(ctx))
	require.False(t, iter.Next(ctx))
}

func checkPausesByEvent_multi(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	evtA := "event/a-multi"
	evtB := "event/unused-plz"

	// Save many a pause.
	pauses := []state.Pause{}
	for i := 0; i <= 2000; i++ {
		p := state.Pause{
			ID:         uuid.New(),
			Identifier: s.Identifier(),
			Outgoing:   inngest.TriggerName,
			Incoming:   w.Steps[0].ID,
			Expires:    state.Time(time.Now().Add(time.Duration(i+1) * time.Minute).Truncate(time.Millisecond).UTC()),
			Event:      &evtA,
		}
		err := m.SavePause(ctx, p)
		require.NoError(t, err)
		pauses = append(pauses, p)
	}

	// Save an unrelated pause to another event.
	unused := state.Pause{
		ID:         uuid.New(),
		Identifier: s.Identifier(),
		Outgoing:   "plz-dont",
		Incoming:   w.Steps[0].ID,
		Expires:    state.Time(time.Now().Add(state.PauseLeaseDuration * 2)),
		Event:      &evtB,
	}
	err := m.SavePause(ctx, unused)
	require.NoError(t, err)

	iter, err := m.PausesByEvent(ctx, uuid.UUID{}, evtA)
	require.NoError(t, err)
	require.NotNil(t, iter)

	seen := []string{}
	n := 0
	for iter.Next(ctx) {
		result := iter.Val(ctx)
		require.NotNil(t, result, "Nil pause returned from iterator")

		found := false
		for _, existing := range pauses {
			if existing.ID == result.ID {
				found = true
				break
			}
		}

		byt, _ := json.MarshalIndent(result, "", "  ")
		require.True(t, found, "iterator returned pause not in event set:\n%v", string(byt))
		// Some iterators may return the same item multiple times (eg. Redis).
		// Record the items that were seen.
		seen = append(seen, result.ID.String())
		n++
	}

	// Sanity check number of seen items.
	require.GreaterOrEqual(t, len(pauses), n, "didn't iterate through all matching pauses")
	require.GreaterOrEqual(t, len(pauses), len(seen))
	// Ensure
	require.GreaterOrEqual(t, n, len(pauses)-1, "Iterator must have returned the correct number of pauses for matching events")
	// Don't get excessive...
	require.LessOrEqual(t, n, len(pauses)+2, "Iterator returned too many duplicate items.")

	// Ensure that all IDs were returned.
	for _, p := range pauses {
		require.Contains(t, seen, p.ID.String(), "Iterator did not return all pause IDs for multiple events")
	}

	// Ensure we didn't get the unrelated event.
	require.NotContains(t, seen, unused.ID)
}

func checkPausesByEvent_concurrent(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	// Create many pauses, then multiple iterators.
	evtA := "event/a-multi"
	pauses := []state.Pause{}
	for i := 0; i <= 2000; i++ {
		p := state.Pause{
			ID:         uuid.New(),
			Identifier: s.Identifier(),
			Outgoing:   inngest.TriggerName,
			Incoming:   w.Steps[0].ID,
			Expires:    state.Time(time.Now().Add(time.Duration(i+1) * time.Minute).Truncate(time.Millisecond).UTC()),
			Event:      &evtA,
		}
		err := m.SavePause(ctx, p)
		require.NoError(t, err)
		pauses = append(pauses, p)
	}

	iterA, err := m.PausesByEvent(ctx, uuid.UUID{}, evtA)
	require.NoError(t, err)
	require.NotNil(t, iterA)

	// Consume 50% of the iterator.
	seenA := []string{}
	a := 0
	for a <= (len(pauses)/2) && iterA.Next(ctx) {
		result := iterA.Val(ctx)
		found := false
		for _, existing := range pauses {
			if existing.ID == result.ID {
				found = true
				break
			}
		}
		require.True(t, found, "iterator returned pause not in event set")
		seenA = append(seenA, result.ID.String())
		a++
	}

	// Create a new iterator and consume it all.
	iterB, err := m.PausesByEvent(ctx, uuid.UUID{}, evtA)
	require.NoError(t, err)
	require.NotNil(t, iterB)
	seenB := []string{}
	b := 0
	for iterB.Next(ctx) {
		result := iterB.Val(ctx)
		found := false
		for _, existing := range pauses {
			if existing.ID == result.ID {
				found = true
				break
			}
		}
		require.True(t, found, "iterator returned pause not in event set")
		seenB = append(seenB, result.ID.String())
		b++
	}

	// Consume the rest of A.
	for iterA.Next(ctx) {
		result := iterA.Val(ctx)
		found := false
		for _, existing := range pauses {
			if existing.ID == result.ID {
				found = true
				break
			}
		}
		require.True(t, found, "iterator returned pause not in event set")
		seenA = append(seenA, result.ID.String())
		a++
	}

	// Sanity check number of seen items.
	require.GreaterOrEqual(t, len(pauses), a, "didn't iterate through all of the first concurrent iterator")
	require.GreaterOrEqual(t, len(pauses), b, "didn't iterate through all of the second concurrent iterator")
	require.GreaterOrEqual(t, len(pauses), len(seenA))
	require.GreaterOrEqual(t, len(pauses), len(seenB))
	require.GreaterOrEqual(t, a, len(pauses)-1, "Iterator must have returned the correct number of pauses for matching events")
	require.GreaterOrEqual(t, b, len(pauses)-1, "Iterator must have returned the correct number of pauses for matching events")

	// Ensure that all IDs were returned.
	for _, p := range pauses {
		require.Contains(t, seenA, p.ID.String(), "Iterator A did not return all pause IDs for multiple events")
		require.Contains(t, seenB, p.ID.String(), "Iterator B did not return all pause IDs for multiple events")
	}
}

func checkPausesByEvent_consumed(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	evtA := "event/a-consumed"

	// Save many a pause.
	pauses := []state.Pause{}
	for i := 0; i < 2; i++ {
		p := state.Pause{
			ID:         uuid.New(),
			Identifier: s.Identifier(),
			Outgoing:   inngest.TriggerName,
			Incoming:   w.Steps[0].ID,
			Expires:    state.Time(time.Now().Add(time.Duration(i+1) * time.Minute).Truncate(time.Millisecond).UTC()),
			Event:      &evtA,
		}
		err := m.SavePause(ctx, p)
		require.NoError(t, err)
		pauses = append(pauses, p)
	}

	//
	// Ensure that the iteration shows everything at first.
	//
	iter, err := m.PausesByEvent(ctx, uuid.UUID{}, evtA)
	require.NoError(t, err)
	require.NotNil(t, iter)

	seen := []string{}
	n := 0
	for iter.Next(ctx) {
		result := iter.Val(ctx)

		require.NotNil(t, result)
		require.NotNil(t, result.Event)
		byt, _ := json.MarshalIndent(result, "", "  ")
		require.Equal(t, evtA, *result.Event, "iterator returned pause not in event set:\n%v", string(byt))

		// Some iterators may return the same item multiple times (eg. Redis).
		// Record the items that were seen.
		seen = append(seen, result.ID.String())
		n++
	}

	// Sanity check number of seen items.
	require.GreaterOrEqual(t, n, len(pauses), "didn't iterate through all matching pauses")
	require.GreaterOrEqual(t, len(seen), len(pauses))

	// Consume the first pause, and assert that it doesn't show up in
	// an iterator.
	err = m.ConsumePause(ctx, pauses[0].ID, nil)
	require.NoError(t, err)

	iter, err = m.PausesByEvent(ctx, uuid.UUID{}, evtA)
	require.NoError(t, err)
	require.NotNil(t, iter)

	seen = []string{}
	n = 0
	for iter.Next(ctx) {
		result := iter.Val(ctx)

		// This should not be the consumed pause.
		require.NotEqual(t, pauses[0].ID, result.ID, "returned a consumed pause within iterator")

		found := false
		for _, existing := range pauses {
			if existing.ID == result.ID {
				found = true
				break
			}
		}

		byt, _ := json.MarshalIndent(result, "", "  ")
		require.True(t, found, "iterator returned pause not in event set:\n%v", string(byt))
		// Some iterators may return the same item multiple times (eg. Redis).
		// Record the items that were seen.
		seen = append(seen, result.ID.String())
		n++
	}

	// Sanity check number of seen items.
	require.GreaterOrEqual(t, len(pauses)-1, n, "consumed pause returned within iterator")
	require.GreaterOrEqual(t, len(pauses)-1, len(seen))

	t.Run("It consumes a pause for an event with > 1 pause set up", func(t *testing.T) {
		wsID := uuid.New()
		evtA = "consumed/single"

		p1 := state.Pause{
			ID:          uuid.New(),
			WorkspaceID: wsID,
			Identifier:  s.Identifier(),
			Outgoing:    inngest.TriggerName,
			Incoming:    w.Steps[0].ID,
			Expires:     state.Time(time.Now().Add(time.Minute)),
			Event:       &evtA,
		}
		p2 := state.Pause{
			ID:          uuid.New(),
			WorkspaceID: wsID,
			Identifier:  s.Identifier(),
			Outgoing:    inngest.TriggerName,
			Incoming:    w.Steps[0].ID,
			Expires:     state.Time(time.Now().Add(time.Minute).Truncate(time.Second).UTC()),
			Event:       &evtA,
		}
		err := m.SavePause(ctx, p1)
		require.NoError(t, err)
		err = m.SavePause(ctx, p2)
		require.NoError(t, err)

		//
		// Ensure that the iteration shows everything at first.
		//
		iter, err := m.PausesByEvent(ctx, wsID, evtA)
		require.NoError(t, err)
		require.NotNil(t, iter)

		n := 0
		for iter.Next(ctx) {
			n++
		}

		// There should be two pauses.
		require.Equal(t, 2, n)

		err = m.ConsumePause(ctx, p1.ID, map[string]any{"ok": true})
		require.NoError(t, err)

		//
		// Ensure that the iteration shows the last event.
		//
		iter, err = m.PausesByEvent(ctx, wsID, evtA)
		require.NoError(t, err)
		require.NotNil(t, iter)

		n = 0
		for iter.Next(ctx) {
			n++
			val := iter.Val(ctx)
			require.EqualValues(t, p2, *val)
		}

		require.Equal(t, 1, n)
	})

}

func checkPausesByStep(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	// Save a pause.
	pause := state.Pause{
		ID:         uuid.New(),
		Identifier: s.Identifier(),
		Outgoing:   inngest.TriggerName,
		Incoming:   w.Steps[0].ID,
		Expires:    state.Time(time.Now().Add(state.PauseLeaseDuration * 2).Truncate(time.Millisecond).UTC()),
	}
	err := m.SavePause(ctx, pause)
	require.NoError(t, err)

	second := state.Pause{
		ID:         uuid.New(),
		Identifier: s.Identifier(),
		Outgoing:   w.Steps[0].ID,
		Incoming:   w.Steps[1].ID,
		Expires:    state.Time(time.Now().Add(state.PauseLeaseDuration * 2).Truncate(time.Millisecond).UTC()),
	}
	err = m.SavePause(ctx, second)
	require.NoError(t, err)

	found, err := m.PauseByStep(ctx, s.Identifier(), "none")
	require.Nil(t, found)
	require.NotNil(t, err)
	require.Error(t, state.ErrPauseNotFound, err)

	found, err = m.PauseByStep(ctx, s.Identifier(), w.Steps[0].ID)
	require.Nil(t, err)
	require.NotNil(t, found)
	require.EqualValues(t, pause, *found)

	found, err = m.PauseByStep(ctx, s.Identifier(), w.Steps[1].ID)
	require.Nil(t, err)
	require.NotNil(t, found)
	require.EqualValues(t, second, *found)
}

func checkPauseByID(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	// Save a pause.
	pause := state.Pause{
		ID:         uuid.New(),
		Identifier: s.Identifier(),
		Outgoing:   inngest.TriggerName,
		Incoming:   w.Steps[0].ID,
		Expires:    state.Time(time.Now().Add(time.Second * 2).Truncate(time.Millisecond).UTC()),
	}
	err := m.SavePause(ctx, pause)
	require.NoError(t, err)

	found, err := m.PauseByID(ctx, pause.ID)
	require.Nil(t, err)
	require.EqualValues(t, pause, *found)

	<-time.After(time.Second * 3)

	// Still found.
	found, err = m.PauseByID(ctx, pause.ID)
	require.Nil(t, err, "PauseByID should return expired but unconsumed pauses")
	require.EqualValues(t, pause, *found)

	// Consume.
	err = m.ConsumePause(ctx, pause.ID, nil)
	require.Nil(t, err, "Consuming an expired pause should work")

	found, err = m.PauseByID(ctx, pause.ID)
	require.Nil(t, found, "PauseByID should not return consumed pauses")
	require.NotNil(t, err)
	require.Error(t, state.ErrPauseNotFound, err)

	found, err = m.PauseByID(ctx, uuid.New())
	require.Nil(t, found, "PauseByID should not return random IDs")
	require.NotNil(t, err)
	require.Error(t, state.ErrPauseNotFound, err)
}

func checkIdempotency(t *testing.T, m state.Manager) {
	ctx := context.Background()

	// Create 100 new functions concurrently.
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	id := state.Identifier{
		WorkflowID: w.ID,
		RunID:      runID,
		Key:        runID.String(),
	}
	data := input.Map()
	batch := []map[string]any{data}

	var errCount int32
	var okCount int32

	tick := time.Now().Add(2 * time.Second).Truncate(time.Second)

	wg := &sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		copiedID := id

		wg.Add(1)
		go func() {
			<-time.After(time.Until(tick))
			// Create a new Run ID each time
			copiedID.RunID = ulid.MustNew(ulid.Now(), rand.Reader)

			init := state.Input{
				Identifier:     copiedID,
				EventBatchData: batch,
			}

			_, err := m.New(ctx, init)
			if err == nil {
				atomic.AddInt32(&okCount, 1)
			} else {
				atomic.AddInt32(&errCount, 1)
				assert.ErrorIs(t, err, state.ErrIdentifierExists)
			}

			wg.Done()
		}()
	}

	wg.Wait()
	assert.Equal(t, int32(1), atomic.LoadInt32(&okCount), "Must have saved the run ID once")
	assert.Equal(t, int32(99), atomic.LoadInt32(&errCount), "Must have errored 99 times when the run ID exists")
}

func checkSetStatus(t *testing.T, m state.Manager) {
	ctx := context.Background()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	id := state.Identifier{
		WorkflowID: w.ID,
		RunID:      runID,
		Key:        runID.String(),
	}

	init := state.Input{
		Identifier:     id,
		EventBatchData: []map[string]any{input.Map()},
	}

	s, err := m.New(ctx, init)
	require.NoError(t, err)
	require.EqualValues(t, enums.RunStatusRunning, s.Metadata().Status, "Status is not Running")

	// Add time so that the history ticks a millisecond
	<-time.After(time.Millisecond)

	err = m.SetStatus(ctx, s.Identifier(), enums.RunStatusOverflowed)
	require.NoError(t, err)

	reloaded, err := m.Load(ctx, s.RunID())
	require.NoError(t, err)
	require.EqualValues(t, enums.RunStatusOverflowed, reloaded.Metadata().Status, "Status is not Overflowed")

	history, err := m.History(ctx, s.RunID())
	require.NoError(t, err)
	require.Equal(t, enums.HistoryTypeFunctionStatusUpdated, history[len(history)-1].Type)
}

func checkCancel(t *testing.T, m state.Manager) {
	ctx := context.Background()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	id := state.Identifier{
		WorkflowID: w.ID,
		RunID:      runID,
		Key:        runID.String(),
	}

	init := state.Input{
		Identifier:     id,
		EventBatchData: []map[string]any{input.Map()},
	}

	s, err := m.New(ctx, init)
	require.NoError(t, err)
	require.EqualValues(t, enums.RunStatusRunning, s.Metadata().Status, "Status is not Running")

	// Add time so that the history ticks a millisecond
	<-time.After(time.Millisecond)

	err = m.Cancel(ctx, s.Identifier())
	require.NoError(t, err)

	reloaded, err := m.Load(ctx, s.RunID())
	require.NoError(t, err)
	require.EqualValues(t, enums.RunStatusCancelled, reloaded.Metadata().Status, "Status is not Cancelled")

	history, err := m.History(ctx, s.RunID())
	require.NoError(t, err)
	require.Equal(t, enums.HistoryTypeFunctionCancelled, history[len(history)-1].Type)
}

func checkCancel_cancelled(t *testing.T, m state.Manager) {
	ctx := context.Background()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	id := state.Identifier{
		WorkflowID: w.ID,
		RunID:      runID,
		Key:        runID.String(),
	}
	init := state.Input{
		Identifier:     id,
		EventBatchData: []map[string]any{input.Map()},
	}

	s, err := m.New(ctx, init)
	require.NoError(t, err)
	require.EqualValues(t, enums.RunStatusRunning, s.Metadata().Status, "Status is not Running")

	// Add time so that the history ticks a millisecond
	<-time.After(time.Millisecond)

	err = m.Cancel(ctx, s.Identifier())
	require.NoError(t, err)
	reloaded, err := m.Load(ctx, s.RunID())
	require.NoError(t, err)
	require.EqualValues(t, enums.RunStatusCancelled, reloaded.Metadata().Status, "Status is not Cancelled")

	err = m.Cancel(ctx, s.Identifier())
	require.Equal(t, err, state.ErrFunctionCancelled)
}

func checkCancel_completed(t *testing.T, m state.Manager) {
	ctx := context.Background()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	id := state.Identifier{
		WorkflowID: w.ID,
		RunID:      runID,
		Key:        runID.String(),
	}
	init := state.Input{
		Identifier:     id,
		EventBatchData: []map[string]any{input.Map()},
	}

	s, err := m.New(ctx, init)
	require.NoError(t, err)
	require.EqualValues(t, enums.RunStatusRunning, s.Metadata().Status, "Status is not Running")

	// Add time so that the history ticks a millisecond
	<-time.After(time.Millisecond)

	err = m.Finalized(ctx, s.Identifier(), w.Steps[0].ID, 0)
	require.NoError(t, err)

	s, err = m.Load(ctx, s.RunID())
	require.NoError(t, err)
	require.EqualValues(t, enums.RunStatusCompleted, s.Metadata().Status, "Status is not Complete after finalizing")

	// Add time so that the history ticks a millisecond
	<-time.After(time.Millisecond)

	err = m.Cancel(ctx, s.Identifier())
	require.Equal(t, err, state.ErrFunctionComplete)

	s, err = m.Load(ctx, s.RunID())
	require.NoError(t, err)
	require.EqualValues(t, enums.RunStatusCompleted, s.Metadata().Status, "Status is not Complete after finalizing")
}

func checkFinalizedStatus(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)
	loaded, err := m.Load(ctx, s.RunID())
	require.NoError(t, err)
	require.Equal(t, enums.RunStatusRunning, loaded.Metadata().Status, fmt.Sprintf("expected status to be %d", enums.RunStatusRunning))

	// Finalize, reducing count to 0 which should set status to complete.
	err = m.Finalized(ctx, s.Identifier(), inngest.TriggerName, 0)
	require.NoError(t, err)

	loaded, err = m.Load(ctx, s.RunID())
	require.NoError(t, err)
	require.Equal(t, enums.RunStatusCompleted, loaded.Metadata().Status, "Finalizing step setting pending to 0 should set status to enums.RunStatusComplete")
	require.Equal(t, 0, loaded.Metadata().Pending)

	t.Run("It doesn't overwrite cancelled statuses, etc", func(t *testing.T) {
		s = setup(t, m)

		err = m.Cancel(ctx, s.Identifier())
		require.NoError(t, err)

		loaded, err := m.Load(ctx, s.RunID())
		require.NoError(t, err)
		require.Equal(t, enums.RunStatusCancelled, loaded.Metadata().Status)

		err = m.Finalized(ctx, s.Identifier(), inngest.TriggerName, 0)
		require.NoError(t, err)

		loaded, err = m.Load(ctx, s.RunID())
		require.NoError(t, err)
		require.Equal(t, enums.RunStatusCancelled, loaded.Metadata().Status)
	})

	t.Run("It allows Failed statuses which write to history", func(t *testing.T) {
		s = setup(t, m)

		err = m.Finalized(ctx, s.Identifier(), inngest.TriggerName, 0, enums.RunStatusFailed)
		require.NoError(t, err)

		loaded, err := m.Load(ctx, s.RunID())
		require.NoError(t, err)
		require.Equal(t, enums.RunStatusFailed, loaded.Metadata().Status)

		history, err := m.History(ctx, s.RunID())
		require.NoError(t, err)
		require.Equal(t, enums.HistoryTypeFunctionFailed, history[len(history)-1].Type)
	})
}

func checkLogs(t *testing.T, m state.Manager) {
	t.Helper()

	ctx := context.Background()
	s := setup(t, m)

	t.Run("New() stores HistoryTypeFunctionStarted", func(t *testing.T) {
		history, err := m.History(ctx, s.RunID())

		require.NoError(t, err)
		require.Equal(t, 1, len(history))
		require.Equal(t, enums.HistoryTypeFunctionStarted, history[0].Type)
	})

	t.Run("Started() stores HistoryTypeStepStarted with attempt information", func(t *testing.T) {
		s = setup(t, m)
		<-time.After(time.Millisecond)

		err := m.Started(ctx, s.Identifier(), w.Steps[0].ID, 2)
		require.NoError(t, err)

		history, err := m.History(ctx, s.RunID())
		require.NoError(t, err)
		require.Equal(t, 2, len(history))

		started := history[1]

		require.Equal(t, enums.HistoryTypeStepStarted.String(), started.Type.String())
		require.NotNil(t, started.Data)

		stepdata, ok := started.Data.(state.HistoryStep)
		require.True(t, ok, "step data is %T instead of state.HistoryStep", started.Data)
		require.Equal(t, 2, stepdata.Attempt)
	})

	t.Run("SaveResponse() with a non-final error stores HistoryTypeStepErrored", func(t *testing.T) {
		s = setup(t, m)
		<-time.After(time.Millisecond)

		r := state.DriverResponse{
			Step: w.Steps[0],
			Err:  strptr("lol"),
		}
		_, err := m.SaveResponse(ctx, s.Identifier(), r, 2)
		require.NoError(t, err)

		history, err := m.History(ctx, s.RunID())
		require.NoError(t, err)
		require.Equal(t, 2, len(history))

		errored := history[1]

		require.Equal(t, enums.HistoryTypeStepErrored.String(), errored.Type.String())
		require.NotNil(t, errored.Data)

		stepdata, ok := errored.Data.(state.HistoryStep)
		require.True(t, ok, "step data is %T instead of state.HistoryStep (%v)", errored.Data, errored.Data)
		require.Equal(t, 2, stepdata.Attempt)
		require.Equal(t, w.Steps[0].ID, stepdata.ID)
		require.Equal(t, w.Steps[0].Name, stepdata.Name)
	})

	t.Run("SaveResponse() with a final error stores HistoryTypeStepFailed and HistoryTypeFunctionFailed", func(t *testing.T) {
		s = setup(t, m)
		<-time.After(time.Millisecond)

		r := state.DriverResponse{
			Step: w.Steps[0],
			Err:  strptr("lol"),
		}
		r.SetFinal()

		_, err := m.SaveResponse(ctx, s.Identifier(), r, 2)
		require.NoError(t, err)

		history, err := m.History(ctx, s.RunID())
		require.NoError(t, err)
		require.Equal(t, 3, len(history))

		historyStep := history[1]
		historyFunc := history[2]

		require.Equal(t, enums.HistoryTypeStepFailed.String(), historyStep.Type.String())
		require.Equal(t, enums.HistoryTypeFunctionFailed.String(), historyFunc.Type.String())

		stepdata, ok := historyStep.Data.(state.HistoryStep)
		require.True(t, ok, "step data is %T instead of state.HistoryStep (%v)", historyStep.Data, historyStep.Data)
		require.Equal(t, 2, stepdata.Attempt)
		require.Equal(t, w.Steps[0].ID, stepdata.ID)
		require.Equal(t, w.Steps[0].Name, stepdata.Name)
	})

	t.Run("SaveResponse() with step data stores HistoryTypeStepCompleted", func(t *testing.T) {
		s = setup(t, m)
		<-time.After(time.Millisecond)

		r := state.DriverResponse{
			Step:   w.Steps[0],
			Output: map[string]any{"ok": true},
		}
		_, err := m.SaveResponse(ctx, s.Identifier(), r, 2)
		require.NoError(t, err)

		history, err := m.History(ctx, s.RunID())
		require.NoError(t, err)
		require.Equal(t, 2, len(history))

		historyStep := history[1]

		require.Equal(t, enums.HistoryTypeStepCompleted.String(), historyStep.Type.String())

		stepdata, ok := historyStep.Data.(state.HistoryStep)
		require.True(t, ok, "step data is %T instead of state.HistoryStep (%v)", historyStep.Data, historyStep.Data)
		require.Equal(t, 2, stepdata.Attempt)
		require.Equal(t, w.Steps[0].ID, stepdata.ID)
		require.Equal(t, w.Steps[0].Name, stepdata.Name)
	})

	t.Run("Scheduled() stores HistoryTypeStepScheduled", func(t *testing.T) {
		s = setup(t, m)
		<-time.After(time.Millisecond)

		err := m.Scheduled(ctx, s.Identifier(), w.Steps[0].ID, 2, nil)
		require.NoError(t, err)

		history, err := m.History(ctx, s.RunID())
		require.NoError(t, err)
		require.Equal(t, 2, len(history))

		historyStep := history[1]
		require.Equal(t, enums.HistoryTypeStepScheduled.String(), historyStep.Type.String())

		stepdata, ok := historyStep.Data.(state.HistoryStep)
		require.True(t, ok, "step data is %T instead of state.HistoryStep (%v)", historyStep.Data, historyStep.Data)
		require.Equal(t, 2, stepdata.Attempt)
		require.Equal(t, w.Steps[0].ID, stepdata.ID)
		// TODO
		// require.Equal(t, w.Steps[0].Name, stepdata.Name)
	})

	t.Run("Finalized() does nothing with a >0 pending count", func(t *testing.T) {
		s = setup(t, m)
		err := m.Scheduled(ctx, s.Identifier(), w.Steps[0].ID, 2, nil)
		require.NoError(t, err)

		reloaded, err := m.Load(ctx, s.RunID())
		require.NoError(t, err)
		require.EqualValues(t, 2, reloaded.Metadata().Pending)

		err = m.Finalized(ctx, s.Identifier(), w.Steps[0].ID, 2)
		require.NoError(t, err)

		history, err := m.History(ctx, s.RunID())
		require.NoError(t, err)
		require.Equal(t, 2, len(history))
	})

	t.Run("Finalized() stores HistoryTypeFunctionComplete if pending count transitions to 0", func(t *testing.T) {
		s = setup(t, m)
		<-time.After(time.Millisecond)

		require.EqualValues(t, 1, s.Metadata().Pending)

		err := m.Finalized(ctx, s.Identifier(), w.Steps[0].ID, 2)
		require.NoError(t, err)

		history, err := m.History(ctx, s.RunID())
		require.NoError(t, err)
		require.Equal(t, 2, len(history))

		historyStep := history[1]
		require.Equal(t, enums.HistoryTypeFunctionCompleted.String(), historyStep.Type.String())
	})
}

// TODO: Optimization - when finalizing steps, we should delete all pauses when the counter is set to 0
/*
func checkFinalizedDeletesPauses(t *testing.T, m state.Manager) {
	ctx := context.Background()
	s := setup(t, m)

	// Create a pause.
	evt := "event/a"
	pause := state.Pause{
		ID:         uuid.New(),
		Identifier: s.Identifier(),
		Outgoing:   inngest.TriggerName,
		Incoming:   w.Steps[0].ID,
		Expires:    state.Time(time.Now().Add(time.Minute)),
		Event:      &evt,
	}
	err := m.SavePause(ctx, pause)
	require.NoError(t, err)

	iter, err := m.PausesByEvent(ctx, evt)
	require.NoError(t, err)
	require.NotNil(t, iter)
	require.True(t, iter.Next(ctx))
	require.EqualValues(t, &pause, iter.Val(ctx))

	found, err := m.PauseByID(ctx, pause.ID)
	require.Nil(t, err)
	require.EqualValues(t, pause, *found)

	// Finalize, reducing count to 0 which should delete all active pauses for this identifier
	err = m.Finalized(ctx, s.Identifier(), inngest.TriggerName)
	require.NoError(t, err)

	// Pause should be deleted.
	iter, err = m.PausesByEvent(ctx, evt)
	require.NoError(t, err)
	require.Nil(t, iter)
	found, err = m.PauseByID(ctx, pause.ID)
	require.Equal(t, state.ErrPauseNotFound, err)
	require.Nil(t, found)
}
*/

func setup(t *testing.T, m state.Manager) state.State {
	ctx := context.Background()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	id := state.Identifier{
		WorkflowID: w.ID,
		RunID:      runID,
		Key:        runID.String(),
	}

	init := state.Input{
		Identifier:     id,
		EventBatchData: []map[string]any{input.Map()},
	}

	s, err := m.New(ctx, init)
	require.NoError(t, err)
	// Add a millisecond so that this history item always comes first.  There
	// are some race conditions here, as history items are MS precision.
	<-time.After(time.Millisecond)

	return s
}

func strptr(s string) *string {
	return &s
}
