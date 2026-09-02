package queue

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/util"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func semaphoreTestItem(accountID, envID, fnID uuid.UUID, at time.Time, sems ...constraintapi.Semaphore) *QueueItem {
	return &QueueItem{
		ID:          ulid.Make().String(),
		FunctionID:  fnID,
		WorkspaceID: envID,
		AtMS:        at.UnixMilli(),
		Data: Item{
			Kind: KindEdge,
			Identifier: state.Identifier{
				AccountID:   accountID,
				WorkspaceID: envID,
				WorkflowID:  fnID,
				RunID:       ulid.Make(),
			},
			Semaphores: sems,
		},
	}
}

// semaphoreLeaser mimics queueProcessor.LeaseItem for items whose semaphores are
// in exhausted.  an exhausted app semaphore stops the iterator, an exhausted fn
// semaphore skips the item, anything else is dispatched.
func semaphoreLeaser(exhausted map[string]constraintapi.SemaphoreConstraint, calls *int32) QueueItemLeaser {
	return mockQueueItemLeaser{
		fn: func(ctx context.Context, req LeaseItemRequest, dispatch DispatchFunc) (LeaseItemResult, error) {
			atomic.AddInt32(calls, 1)

			var hit []constraintapi.SemaphoreConstraint
			for _, s := range req.Item.Data.Semaphores {
				if c, ok := exhausted[semaphoreMemoKey(s.ID, s.EvaluatedKeyHash)]; ok {
					hit = append(hit, c)
				}
			}
			if len(hit) > 0 {
				res := LeaseItemResult{Status: LeaseItemStatusSemaphoreLimited, ExhaustedSemaphores: hit}
				if hasAppSemaphore(hit) {
					return res, fmt.Errorf("semaphore hit: %w", ErrProcessNoUserConstraintCapacity)
				}
				return res, nil
			}

			_, err := dispatch(ctx, ProcessItem{I: *req.Item})
			return LeaseItemResult{Status: LeaseItemStatusDispatched}, err
		},
	}
}

func TestProcessorIteratorStopsOnExhaustedAppSemaphore(t *testing.T) {
	ctx := context.Background()

	accountID, envID, fnID, appID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	at := time.Now().Add(-time.Minute).Truncate(time.Millisecond)

	appSem := constraintapi.Semaphore{ID: constraintapi.SemaphoreIDApp(appID), Weight: 1}
	appConstraint := constraintapi.SemaphoreConstraint{ID: appSem.ID, Weight: 1}
	require.Equal(t, constraintapi.SemaphoreKindApp, appConstraint.Kind())

	shard := &mockShardForIterator{name: "test-shard"}
	opts := NewQueueOptions()
	WithQueueItemEarliestPeekTimeEnabled(func(ctx context.Context, shardName string, acctID, gotEnvID, gotFnID uuid.UUID) QueueItemEarliestPeekTimeConfig {
		return QueueItemEarliestPeekTimeConfig{Enabled: true, BulkStampLimit: 100}
	})(opts)

	mockProc := &mockQueueProcessor{
		shard:     shard,
		clock:     clockwork.NewRealClock(),
		sem:       util.NewTrackingSemaphore(10),
		workers:   make(chan ProcessItem, 10),
		shadowMap: make(map[string]ShadowContinuation),
		opts:      opts,
	}

	items := make([]*QueueItem, 5)
	for i := range items {
		items[i] = semaphoreTestItem(accountID, envID, fnID, at, appSem)
	}

	var calls int32
	iter := ProcessorIterator{
		Partition: &QueuePartition{ID: fnID.String(), AccountID: accountID, EnvID: &envID, FunctionID: &fnID},
		Items:     items,
		Queue:     mockProc,
		Leaser: semaphoreLeaser(map[string]constraintapi.SemaphoreConstraint{
			semaphoreMemoKey(appSem.ID, appSem.EvaluatedKeyHash): appConstraint,
		}, &calls),
		Dispatch: func(_ context.Context, item ProcessItem) (DispatchedItem, error) {
			mockProc.workers <- item
			return NewCompletedDispatchedItem(DispatchedItemResult{}), nil
		},
		StaticTime: at.Add(2 * time.Second),
	}

	require.NoError(t, iter.Iterate(ctx))

	// the first item stops the pass.  no other item reaches the leaser.
	require.Equal(t, int32(1), atomic.LoadInt32(&calls))
	require.Equal(t, int32(1), iter.CtrConcurrency.Load())
	require.Equal(t, int32(0), iter.CtrSuccess.Load())
	require.True(t, iter.IsSemaphoreLimitOnly.Load())
	require.True(t, iter.IsRequeuable())

	// the remaining four items are bulk stamped like a concurrency limit.
	require.Equal(t, int32(4), atomic.LoadInt32(&shard.earliestPeekTimeCalls))
}

func TestProcessorIteratorMemoizesExhaustedFnSemaphores(t *testing.T) {
	ctx := context.Background()

	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	at := time.Now().Add(-time.Minute).Truncate(time.Millisecond)

	keyed := constraintapi.SemaphoreIDFnKey(fnID, "event.data.customer")
	semA := constraintapi.Semaphore{ID: keyed, EvaluatedKeyHash: "a", Weight: 1, Release: constraintapi.SemaphoreReleaseManual}
	semB := constraintapi.Semaphore{ID: keyed, EvaluatedKeyHash: "b", Weight: 1, Release: constraintapi.SemaphoreReleaseManual}
	exhaustedA := constraintapi.SemaphoreConstraint{ID: semA.ID, EvaluatedKeyHash: semA.EvaluatedKeyHash, Weight: 1, Release: semA.Release}
	require.Equal(t, constraintapi.SemaphoreKindFnKey, exhaustedA.Kind())

	shard := &mockShardForIterator{name: "test-shard"}
	mockProc := &mockQueueProcessor{
		shard:     shard,
		clock:     clockwork.NewRealClock(),
		sem:       util.NewTrackingSemaphore(10),
		workers:   make(chan ProcessItem, 10),
		shadowMap: make(map[string]ShadowContinuation),
		opts:      NewQueueOptions(),
	}

	items := []*QueueItem{
		semaphoreTestItem(accountID, envID, fnID, at, semA), // exhausted, one round trip
		semaphoreTestItem(accountID, envID, fnID, at),       // no semaphore, dispatched
		semaphoreTestItem(accountID, envID, fnID, at, semA), // memo hit, no round trip
		semaphoreTestItem(accountID, envID, fnID, at, semB), // other key, dispatched
		semaphoreTestItem(accountID, envID, fnID, at, semA), // memo hit, no round trip
	}

	var calls int32
	iter := ProcessorIterator{
		Partition: &QueuePartition{ID: fnID.String(), AccountID: accountID, EnvID: &envID, FunctionID: &fnID},
		Items:     items,
		Queue:     mockProc,
		Leaser: semaphoreLeaser(map[string]constraintapi.SemaphoreConstraint{
			semaphoreMemoKey(semA.ID, semA.EvaluatedKeyHash): exhaustedA,
		}, &calls),
		Dispatch: func(_ context.Context, item ProcessItem) (DispatchedItem, error) {
			mockProc.workers <- item
			return NewCompletedDispatchedItem(DispatchedItemResult{}), nil
		},
		StaticTime: at.Add(2 * time.Second),
	}

	require.NoError(t, iter.Iterate(ctx))

	// items 1, 2 and 4 reach the leaser.  items 3 and 5 are answered by the memo.
	require.Equal(t, int32(3), atomic.LoadInt32(&calls))
	require.Equal(t, int32(3), iter.CtrConcurrency.Load())
	require.Equal(t, int32(2), iter.CtrSuccess.Load())
	require.True(t, iter.IsSemaphoreLimitOnly.Load())
	require.Len(t, mockProc.workers, 2)
}
