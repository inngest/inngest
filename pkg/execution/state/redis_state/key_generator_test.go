package redis_state

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
)

func TestVerifyKeyGenerator(t *testing.T) {
	ctx := context.Background()

	fakeUlid := ulid.MustNew(ulid.Now(), rand.Reader)
	fakeUuid := uuid.New()

	var legacyKg legacyKeyGenerator = legacyDefaultKeyFunc{
		Prefix: "{estate}",
	}

	legacyDefaultkg := legacyDefaultQueueKeyGenerator{
		Prefix: "{queue}",
	}

	newRunStateKg := runStateKeyGenerator{
		stateDefaultKey: "estate",
	}

	assert.Equal(t, legacyKg.Idempotency(ctx, state.Identifier{RunID: fakeUlid, WorkspaceID: fakeUuid}), newRunStateKg.Idempotency(ctx, false, state.Identifier{RunID: fakeUlid, WorkspaceID: fakeUuid}))

	assert.Equal(t, legacyKg.Stack(ctx, fakeUlid), newRunStateKg.Stack(ctx, false, fakeUlid))

	assert.Equal(t, legacyKg.RunMetadata(ctx, fakeUlid), newRunStateKg.RunMetadata(ctx, false, fakeUlid))
	assert.Equal(
		t,
		legacyKg.Events(ctx, state.Identifier{RunID: fakeUlid, WorkflowID: fakeUuid}),
		newRunStateKg.Events(ctx, false, fakeUuid, fakeUlid),
	)
	assert.Equal(
		t,
		legacyKg.Actions(ctx, state.Identifier{RunID: fakeUlid, WorkflowID: fakeUuid}),
		newRunStateKg.Actions(ctx, false, fakeUuid, fakeUlid),
	)

	var legacyQueueKg legacyQueueKeyGenerator = legacyDefaultkg
	queueItemKg := queueItemKeyGenerator{queueDefaultKey: "queue"}
	assert.Equal(t, legacyQueueKg.QueueItem(), queueItemKg.QueueItem())

	newQueueKg := queueKeyGenerator{queueDefaultKey: "queue", queueItemKeyGenerator: queueItemKg}

	assert.Equal(t, legacyQueueKg.QueueItem(), newQueueKg.QueueItem())

	assert.Equal(t, legacyQueueKg.QueueIndex("id"), newQueueKg.QueueIndex("id"))

	assert.Equal(t, legacyQueueKg.PartitionItem(), newQueueKg.PartitionItem())
	assert.Equal(t, legacyQueueKg.PartitionMeta("id"), newQueueKg.PartitionMeta("id"))
	assert.Equal(t, legacyQueueKg.GlobalPartitionIndex(), newQueueKg.GlobalPartitionIndex())

	assert.Equal(t, legacyQueueKg.ThrottleKey(&osqueue.Throttle{}), newQueueKg.ThrottleKey(&osqueue.Throttle{}))
	assert.Equal(t, legacyQueueKg.Sequential(), newQueueKg.Sequential())
	assert.Equal(t, legacyQueueKg.Scavenger(), newQueueKg.Scavenger())
	assert.Equal(t, legacyQueueKg.Idempotency("key"), newQueueKg.Idempotency("key"))
	assert.Equal(t, legacyQueueKg.Concurrency("prefix", "key"), newQueueKg.Concurrency("prefix", "key"))
	assert.Equal(t, legacyQueueKg.ConcurrencyIndex(), newQueueKg.ConcurrencyIndex())
	assert.Equal(t, legacyQueueKg.RunIndex(fakeUlid), newQueueKg.RunIndex(fakeUlid))
	assert.Equal(t, legacyQueueKg.Status("status", fakeUuid), newQueueKg.Status("status", fakeUuid))
	assert.Equal(t, legacyQueueKg.ConcurrencyFnEWMA(fakeUuid), newQueueKg.ConcurrencyFnEWMA(fakeUuid))

	newPausesKg := pauseKeyGenerator{stateDefaultKey: "estate"}

	assert.Equal(t, legacyKg.PauseID(ctx, fakeUuid), newPausesKg.Pause(ctx, fakeUuid))
	assert.Equal(t, legacyKg.PauseLease(ctx, fakeUuid), newPausesKg.PauseLease(ctx, fakeUuid))
	assert.Equal(t, legacyKg.PauseEvent(ctx, fakeUuid, "key"), newPausesKg.PauseEvent(ctx, fakeUuid, "key"))
	assert.Equal(t, legacyKg.PauseIndex(ctx, "kind", fakeUuid, "event"), newPausesKg.PauseIndex(ctx, "kind", fakeUuid, "event"))
	assert.Equal(t, legacyKg.RunPauses(ctx, fakeUlid), newPausesKg.RunPauses(ctx, fakeUlid))

	newDebounceKg := debounceKeyGenerator{queueDefaultKey: "queue", queueItemKeyGenerator: queueItemKg}
	var legacyDebounceKg legacyDebounceKeyGenerator = legacyDefaultkg

	assert.Equal(t, legacyDebounceKg.QueueItem(), newDebounceKg.QueueItem())
	assert.Equal(t, legacyDebounceKg.DebouncePointer(ctx, fakeUuid, "key"), newDebounceKg.DebouncePointer(ctx, fakeUuid, "key"))
	assert.Equal(t, legacyDebounceKg.Debounce(ctx), newDebounceKg.Debounce(ctx))

	globalKg := globalKeyGenerator{stateDefaultKey: "estate"}

	assert.Equal(t, legacyKg.Invoke(ctx, fakeUuid), globalKg.Invoke(ctx, fakeUuid))
	// No longer used
	// assert.Equal(t, legacyKg.Workflow(ctx, fakeUuid, 1), globalKg.Workflow(ctx, fakeUuid, 1))
}
