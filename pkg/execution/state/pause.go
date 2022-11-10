package state

import (
	"github.com/google/uuid"
	"github.com/inngest/inngest/inngest"
)

// Pause allows steps of a function to be paused until some condition in the future.
//
// It pauses a specific workflow run via an Identifier, at a specific step in
// the function as specified by Target.
type Pause struct {
	ID uuid.UUID `json:"id"`
	// Identifier is the specific workflow run to resume.  This is required.
	Identifier Identifier `json:"identifier"`
	// Outgoing is the parent step for the pause.
	Outgoing string `json:"outgoing"`
	// Incoming is the step to run after the pause completes.
	Incoming string `json:"incoming"`
	// Expires is a time at which the pause can no longer be consumed.  This
	// gives each pause of a function a TTL.  This is required.
	//
	// NOTE: the pause should remain within the backing state store for
	// some perioud after the expiry time for checking timeout branches:
	//
	// If this pause has its OnTimeout flag set to true, we only traverse
	// the edge if the event *has not* been received.  In order to check
	// this, we enqueue a job that executes on the pause timeout:  if the
	// pause has not yet been consumed we can safely assume the event was
	// not received.  Therefore, we must be able to load the pause for some
	// time after timeout.
	Expires Time `json:"expires"`
	// Event is an optional event that can resume the pause automatically,
	// often paired with an expression.
	Event *string `json:"event"`
	// Expression is an optional expression that must match for the pause
	// to be resumed.
	Expression *string `json:"expression"`
	// ExpressionData _optionally_ stores only the data that we need to evaluate
	// the expression from the event.  This allows us to load pauses from the
	// state store without round trips to fetch the entire function state.  If
	// this is empty and the pause contains an expression, function state will
	// be loaded from the store.
	ExpressionData map[string]any `json:"data"`
	// OnTimeout indicates that this incoming edge should only be ran
	// when the pause times out, if set to true.
	OnTimeout bool `json:"onTimeout"`
	// DataKey is the name of the step to use when adding data to the function
	// run's state after consuming this step.
	//
	// This allows us to create arbitrary "step" names for storing async event
	// data from matching events in async edges, eg. `waitForEvent`.
	//
	// If DataKey is empty and data is provided when consuming a pause, no
	// data will be saved in the function state.
	DataKey string `json:"dataKey,omitempty"`
	// Cancellation indicates whether this pause exists as a cancellation
	// clause for a function.
	//
	// If so, when the matching pause is returned after processing an event
	// the function's status is set to cancelled, preventing any future work.
	Cancel bool `json:"cancel,omitempty"`
}

func (p Pause) Edge() inngest.Edge {
	return inngest.Edge{
		Outgoing: p.Outgoing,
		Incoming: p.Incoming,
	}
}
