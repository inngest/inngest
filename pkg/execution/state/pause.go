package state

import (
	"context"

	"github.com/google/uuid"
	"github.com/inngest/inngest/inngest"
)

// PauseMutater manages creating, leasing, and consuming pauses from a backend implementation.
type PauseMutater interface {
	// SavePause indicates that the traversal of an edge is paused until some future time.
	//
	// The runner which coordinates workflow executions is responsible for managing paused
	// DAG executions.
	SavePause(ctx context.Context, p Pause) error

	// LeasePause allows us to lease the pause until the next step is enqueued, at which point
	// we can 'consume' the pause to remove it.
	//
	// This prevents a failure mode in which we consume the pause but enqueueing the next
	// action fails (eg. due to power loss).
	//
	// If the given pause has been leased within LeasePauseDuration, this should return an
	// ErrPauseLeased error.
	//
	// See https://github.com/inngest/inngest/issues/123 for more info
	LeasePause(ctx context.Context, id uuid.UUID) error

	// ConsumePause consumes a pause by its ID such that it can't be used again and
	// will not be returned from any query.
	//
	// Any data passed when consuming a pause will be stored within function run state
	// for future reference using the pause's DataKey.
	ConsumePause(ctx context.Context, id uuid.UUID, data any) error
}

// PauseGetter allows a runner to return all existing pauses by event or by outgoing ID.  This
// is required to fetch pauses to automatically continue workflows.
type PauseGetter interface {
	// PausesByEvent returns all pauses for a given event, in a given workspace.
	PausesByEvent(ctx context.Context, workspaceID uuid.UUID, eventName string) (PauseIterator, error)

	// PauseByStep returns a specific pause for a given workflow run, from a given step.
	//
	// This is required when continuing a step function from an async step, ie. one that
	// has deferred results which must be continued by resuming the specific pause set
	// up for the given step ID.
	PauseByStep(ctx context.Context, i Identifier, actionID string) (*Pause, error)

	// PauseByID returns a given pause by pause ID.  This must return expired pauses
	// that have not yet been consumed in order to properly handle timeouts.
	//
	// This should not return consumed pauses.
	PauseByID(ctx context.Context, pauseID uuid.UUID) (*Pause, error)
}

// PauseIterator allows the runner to iterate over all pauses returned by a PauseGetter.  This
// ensures that, at scale, all pauses do not need to be loaded into memory.
type PauseIterator interface {
	// Next advances the iterator and returns whether the next call to Val will
	// return a non-nil pause.
	//
	// Next should be called prior to any call to the iterator's Val method, after
	// the iterator has been created.
	//
	// The order of the iterator is unspecified.
	Next(ctx context.Context) bool

	// Val returns the current Pause from the iterator.
	Val(context.Context) *Pause
}

// PauseManager manages mutating and fetching pauses from a backend implementation.
type PauseManager interface {
	PauseMutater
	PauseGetter
}

// Pause allows steps of a function to be paused until some condition in the future.
//
// It pauses a specific workflow run via an Identifier, at a specific step in
// the function as specified by Target.
type Pause struct {
	ID uuid.UUID `json:"id"`
	// WorkspaceID scopes the pause to a specific workspace.
	WorkspaceID uuid.UUID `json:"wsID"`
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
	// Attempt stores the attempt for the current step, if this a pause caused
	// via an async driver.  This lets the executor resume as-is with the current
	// context, ensuring that we retry correctly.
	Attempt int `json:"att,omitempty"`
	// GroupID stores the group ID for this step and history, allowing us to correlate
	// event receives with other history items.
	GroupID string `json:"groupID"`
}

func (p Pause) Edge() inngest.Edge {
	return inngest.Edge{
		Outgoing: p.Outgoing,
		Incoming: p.Incoming,
	}
}
