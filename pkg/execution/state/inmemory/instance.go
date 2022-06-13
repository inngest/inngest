package inmemory

import (
	"github.com/google/uuid"
	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/pkg/execution/state"
	"github.com/oklog/ulid/v2"
)

// NewStateInstance returns an in-memory state.State implementation with the given data.
//
// The state.State functions return only data - they do not allow for returning errors. This
// means that all state for a run should be loaded ahead of execution instead of just-in-time.
// In practice, this makes error handling simpler as it can only occur in one place.
//
// Because data is loaded ahead of time, most state implementations will require an in-memory
// representation of state.State.
//
// This is safe to use and fulfils that requirement.
func NewStateInstance(
	w inngest.Workflow,
	id state.Identifier,
	event map[string]any,
	actions map[string]map[string]any,
	errors map[string]error,
) state.State {
	return &memstate{
		workflow:   w,
		workflowID: id.WorkflowID,
		runID:      id.RunID,
		event:      event,
		actions:    actions,
		errors:     errors,
	}
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

func (s memstate) Workflow() inngest.Workflow {
	return s.workflow
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
		return nil, state.ErrStepIncomplete
	}
	return data, err
}

func (s memstate) ActionComplete(id string) bool {
	_, hasAction := s.Actions()[id]
	return hasAction
}
