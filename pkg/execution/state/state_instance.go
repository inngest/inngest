package state

import (
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
)

// NewStateInstance returns an in-memory State implementation with the given data.
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
	f inngest.Function,
	id Identifier,
	metadata Metadata,
	event map[string]any,
	events map[string]any,
	actions map[string]any,
	errors map[string]error,
	stack []string,
) State {
	return &memstate{
		function:   f,
		identifier: id,
		metadata:   metadata,
		event:      event,
		events:     events,
		actions:    actions,
		errors:     errors,
		stack:      stack,
	}
}

type memstate struct {
	function inngest.Function

	identifier Identifier

	metadata Metadata

	// Event is the root data that triggers the workflow, which is typically
	// an Inngest event.
	event map[string]interface{}

	// Events is the root data that triggers the workflow, which is typically
	// a list of Inngest events.
	events map[string]interface{}

	stack []string

	// Actions stores a map of all output from each individual action
	actions map[string]any

	// errors stores a map of action errors
	errors map[string]error
}

func (s memstate) Metadata() Metadata {
	return s.metadata
}

func (s memstate) Identifier() Identifier {
	return s.identifier
}

func (s memstate) Function() inngest.Function {
	return s.function
}

func (s memstate) WorkflowID() uuid.UUID {
	return s.identifier.WorkflowID
}

func (s memstate) RunID() ulid.ULID {
	return s.identifier.RunID
}

func (s memstate) Stack() []string {
	return s.stack
}

func (s memstate) Event() map[string]interface{} {
	return s.event
}

func (s memstate) Events() map[string]interface{} {
	return s.events
}

func (s memstate) Actions() map[string]any {
	return s.actions
}

func (s memstate) Errors() map[string]error {
	return s.errors
}

func (s memstate) ActionID(id string) (any, error) {
	data, hasAction := s.Actions()[id]
	err, hasError := s.Errors()[id]
	if !hasAction && !hasError {
		return nil, ErrStepIncomplete
	}
	return data, err
}

func (s memstate) ActionComplete(id string) bool {
	_, hasAction := s.Actions()[id]
	return hasAction
}
