package redis_state

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
)

// KeyFunc returns a unique string based off of given data, which is used
// as the key for data stored in redis for workflows, events, actions, and
// errors.
type KeyGenerator interface {
	// Workflow returns the key for the current workflow ID and version.
	Workflow(ctx context.Context, workflowID uuid.UUID, version int) string

	// Idempotency stores the idempotency key for atomic lookup.
	Idempotency(context.Context, state.Identifier) string

	// RunMetadata stores state regarding the current run identifier, such
	// as the workflow version, the time the run started, etc.
	RunMetadata(context.Context, state.Identifier) string

	// Event returns the key used to store the specific event for the
	// given workflow run.
	Event(context.Context, state.Identifier) string

	// Actions returns the key used to store the action response map used
	// for given workflow run - ie. the results for individual steps.
	Actions(context.Context, state.Identifier) string

	// Errors returns the key used to store the error hash map used
	// for given workflow run.
	Errors(context.Context, state.Identifier) string

	// PauseLease stores the key which references a pause's lease.
	//
	// This is stored independently as we may store more than one copy of a pause
	// for easy iteration.
	PauseLease(context.Context, uuid.UUID) string

	// PauseID returns the key used to store an individual pause from its ID.
	PauseID(context.Context, uuid.UUID) string

	// PauseEvent returns the key used to store data for
	PauseEvent(context.Context, string) string

	// PauseStep returns the prefix of the key used within PauseStep.  This lets us
	// iterate through all pauses for a given identifier
	PauseStepPrefix(context.Context, state.Identifier) string

	// PauseStep returns the key used to store a pause ID by the run ID and step ID.
	PauseStep(context.Context, state.Identifier, string) string

	// Log returns the key used to store a log entry for run hisotry
	Log(context.Context, state.Identifier) string
}

type DefaultKeyFunc struct {
	Prefix string
}

func (d DefaultKeyFunc) Idempotency(ctx context.Context, id state.Identifier) string {
	return fmt.Sprintf("%s:key:%s", d.Prefix, id.IdempotencyKey())
}

func (d DefaultKeyFunc) RunMetadata(ctx context.Context, id state.Identifier) string {
	return fmt.Sprintf("%s:metadata:%s", d.Prefix, id.RunID)
}

func (d DefaultKeyFunc) Workflow(ctx context.Context, id uuid.UUID, version int) string {
	return fmt.Sprintf("%s:workflows:%s-%d", d.Prefix, id, version)
}

func (d DefaultKeyFunc) Event(ctx context.Context, id state.Identifier) string {
	return fmt.Sprintf("%s:events:%s:%s", d.Prefix, id.WorkflowID, id.RunID)
}

func (d DefaultKeyFunc) Actions(ctx context.Context, id state.Identifier) string {
	return fmt.Sprintf("%s:actions:%s:%s", d.Prefix, id.WorkflowID, id.RunID)
}

func (d DefaultKeyFunc) Errors(ctx context.Context, id state.Identifier) string {
	return fmt.Sprintf("%s:errors:%s:%s", d.Prefix, id.WorkflowID, id.RunID)
}

func (d DefaultKeyFunc) PauseID(ctx context.Context, id uuid.UUID) string {
	return fmt.Sprintf("%s:pauses:%s", d.Prefix, id.String())
}

func (d DefaultKeyFunc) PauseLease(ctx context.Context, id uuid.UUID) string {
	return fmt.Sprintf("%s:pause-lease:%s", d.Prefix, id.String())
}

func (d DefaultKeyFunc) PauseEvent(ctx context.Context, event string) string {
	return fmt.Sprintf("%s:pause-events:%s", d.Prefix, event)
}

func (d DefaultKeyFunc) PauseStepPrefix(ctx context.Context, id state.Identifier) string {
	return fmt.Sprintf("%s:pause-steps:%s", d.Prefix, id.RunID)
}

func (d DefaultKeyFunc) PauseStep(ctx context.Context, id state.Identifier, step string) string {
	prefix := d.PauseStepPrefix(ctx, id)
	return fmt.Sprintf("%s-%s", prefix, step)
}

func (d DefaultKeyFunc) Log(ctx context.Context, id state.Identifier) string {
	return fmt.Sprintf("%s:history:%s", d.Prefix, id.RunID)
}
