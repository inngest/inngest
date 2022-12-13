package redis_state

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
)

var (
	defaultQueueKey = DefaultQueueKeyGenerator{}
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
	RunMetadata(ctx context.Context, runID ulid.ULID) string

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

	// PauseEvent returns the key used to store data for loading pauses by events.
	PauseEvent(context.Context, uuid.UUID, string) string

	// PauseStep returns the prefix of the key used within PauseStep.  This lets us
	// iterate through all pauses for a given identifier
	PauseStepPrefix(context.Context, state.Identifier) string

	// PauseStep returns the key used to store a pause ID by the run ID and step ID.
	PauseStep(context.Context, state.Identifier, string) string

	// History returns the key used to store a log entry for run hisotry
	History(ctx context.Context, runID ulid.ULID) string
}

type DefaultKeyFunc struct {
	Prefix string
}

func (d DefaultKeyFunc) Idempotency(ctx context.Context, id state.Identifier) string {
	return fmt.Sprintf("%s:key:%s", d.Prefix, id.IdempotencyKey())
}

func (d DefaultKeyFunc) RunMetadata(ctx context.Context, runID ulid.ULID) string {
	return fmt.Sprintf("%s:metadata:%s", d.Prefix, runID)
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

func (d DefaultKeyFunc) PauseEvent(ctx context.Context, workspaceID uuid.UUID, event string) string {
	return fmt.Sprintf("%s:pause-events:%s:%s", d.Prefix, workspaceID, event)
}

func (d DefaultKeyFunc) PauseStepPrefix(ctx context.Context, id state.Identifier) string {
	return fmt.Sprintf("%s:pause-steps:%s", d.Prefix, id.RunID)
}

func (d DefaultKeyFunc) PauseStep(ctx context.Context, id state.Identifier, step string) string {
	prefix := d.PauseStepPrefix(ctx, id)
	return fmt.Sprintf("%s-%s", prefix, step)
}

func (d DefaultKeyFunc) History(ctx context.Context, runID ulid.ULID) string {
	return fmt.Sprintf("%s:history:%s", d.Prefix, runID)
}

type QueueKeyGenerator interface {
	// QueueItem returns the key for the hash containing all items within a
	// queue for a function.
	QueueItem() string
	// QueueIndex returns the key containing the sorted zset for a function
	// queue.
	QueueIndex(id string) string
	// PartitionItem returns the key for the hash containing all partition items.
	PartitionItem() string
	// PartitionIndex returns the sorted set for the partition queue.
	PartitionIndex() string
	// PartitionMeta returns the key to store metadata for partitions, eg.
	// the number of items enqueued, number in progress, etc.
	PartitionMeta(id string) string
	// Sequential returns the key which allows a worker to claim sequential processing
	// of the partitions.
	Sequential() string
	// Idempotency stores the map for storing idempotency keys in redis
	Idempotency(key string) string
}

type DefaultQueueKeyGenerator struct {
	Prefix string
}

func (d DefaultQueueKeyGenerator) QueueItem() string {
	return fmt.Sprintf("%s:queue:item", d.Prefix)
}

func (d DefaultQueueKeyGenerator) QueueIndex(id string) string {
	return fmt.Sprintf("%s:queue:sorted:%s", d.Prefix, id)
}

func (d DefaultQueueKeyGenerator) PartitionItem() string {
	return fmt.Sprintf("%s:partition:item", d.Prefix)
}

func (d DefaultQueueKeyGenerator) PartitionIndex() string {
	return fmt.Sprintf("%s:partition:sorted", d.Prefix)
}

func (d DefaultQueueKeyGenerator) PartitionMeta(id string) string {
	return fmt.Sprintf("%s:partition:meta:%s", d.Prefix, id)
}

func (d DefaultQueueKeyGenerator) Sequential() string {
	return fmt.Sprintf("%s:queue:sequential", d.Prefix)
}

func (d DefaultQueueKeyGenerator) Idempotency(key string) string {
	return fmt.Sprintf("%s:queue:seen:%s", d.Prefix, key)
}
