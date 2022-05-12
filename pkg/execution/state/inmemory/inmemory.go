package inmemory

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/pkg/execution/state"
	"github.com/oklog/ulid"
)

func NewStateManager() state.Manager {
	return &mem{
		state: map[ulid.ULID]state.State{},
		lock:  sync.RWMutex{},
	}
}

type mem struct {
	state map[ulid.ULID]state.State
	lock  sync.RWMutex
}

func (m *mem) New(ctx context.Context, workflow inngest.Workflow, runID ulid.ULID) (state.State, error) {
	state := &memstate{
		workflow:   workflow,
		runID:      runID,
		workflowID: workflow.UUID,
		event:      map[string]interface{}{},
		actions:    map[string]map[string]interface{}{},
		errors:     map[string]error{},
	}

	m.lock.RLock()
	if _, ok := m.state[runID]; ok {
		return nil, fmt.Errorf("run ID already exists: %s", runID)
	}
	m.lock.RUnlock()

	m.lock.Lock()
	m.state[runID] = state
	m.lock.Unlock()

	return state, nil

}

func (m *mem) Load(ctx context.Context, i state.Identifier) (state.State, error) {
	m.lock.RLock()
	s, ok := m.state[i.RunID]
	m.lock.RUnlock()

	if ok {
		return s, nil
	}

	state := &memstate{
		workflowID: i.WorkflowID,
		runID:      i.RunID,
		event:      map[string]interface{}{},
		actions:    map[string]map[string]interface{}{},
		errors:     map[string]error{},
	}

	m.lock.Lock()
	m.state[i.RunID] = state
	m.lock.Unlock()

	return state, nil
}

func (m *mem) SaveActionOutput(ctx context.Context, i state.Identifier, actionID string, data map[string]interface{}) (state.State, error) {
	s, _ := m.Load(ctx, i)

	state := s.(*memstate)
	state.actions[actionID] = data

	m.lock.Lock()
	m.state[i.RunID] = state
	m.lock.Unlock()

	return state, nil
}

func (m *mem) SaveActionError(ctx context.Context, i state.Identifier, actionID string, err error) (state.State, error) {
	s, _ := m.Load(ctx, i)

	fmt.Printf("err: %#v %s;\n", err, actionID)

	state := s.(*memstate)
	state.errors[actionID] = err

	m.lock.Lock()
	m.state[i.RunID] = state
	m.lock.Unlock()

	return state, nil
}

type memstate struct {
	workflow inngest.Workflow

	workflowID uuid.UUID
	runID      ulid.ULID

	// Event is the root data that triggers the workflow, which is typically
	// an Inngest event.
	event map[string]interface{}

	// Actions stores a map of all output from each individual action
	actions map[string]map[string]interface{}

	// errors stores a map of action errors
	errors map[string]error
}

func (s memstate) Identifier() state.Identifier {
	return state.Identifier{
		WorkflowID: s.workflowID,
		RunID:      s.runID,
	}
}

func (s memstate) Workflow() (inngest.Workflow, error) {
	return s.workflow, nil
}

func (s memstate) WorkflowID() uuid.UUID {
	return s.workflowID
}

func (s memstate) RunID() ulid.ULID {
	return s.runID
}

func (s memstate) Event() map[string]interface{} {
	return s.event
}

func (s memstate) Actions() map[string]map[string]interface{} {
	return s.actions
}

func (s memstate) Errors() map[string]error {
	return s.errors
}

func (s memstate) ActionID(id string) (map[string]interface{}, error) {
	data, hasAction := s.Actions()[id]
	err, hasError := s.Errors()[id]
	if !hasAction && !hasError {
		return nil, state.ErrActionIncomplete
	}
	return data, err
}

func (s memstate) ActionComplete(id string) bool {
	_, err := s.ActionID(id)
	return err != state.ErrActionIncomplete
}
