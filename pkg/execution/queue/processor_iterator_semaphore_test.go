package queue

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngest/pkg/util/errs"
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

func TestProcessorIteratorOwnedManualRetryBypassesMemo(t *testing.T) {
	account, other, env, fn := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	enrolled := func(id uuid.UUID) bool { return id == account }
	for _, tc := range []struct {
		name    string
		account uuid.UUID
		policy  func(uuid.UUID) bool
		release constraintapi.SemaphoreReleaseMode
		calls   int
	}{
		{"owned manual retry", account, enrolled, constraintapi.SemaphoreReleaseManual, 2},
		{"legacy account", other, enrolled, constraintapi.SemaphoreReleaseManual, 1},
		{"default disabled", account, nil, constraintapi.SemaphoreReleaseManual, 1},
		{"worker semaphore", account, enrolled, constraintapi.SemaphoreReleaseAuto, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			clock := clockwork.NewFakeClock()
			sem := constraintapi.Semaphore{ID: constraintapi.SemaphoreIDFn(fn), Weight: 1, Release: tc.release}
			denied := &constraintapi.CapacityAcquireResponse{
				ExhaustedConstraints: []constraintapi.ConstraintItem{{Kind: constraintapi.ConstraintKindSemaphore,
					Semaphore: &constraintapi.SemaphoreConstraint{ID: sem.ID, Weight: 1, Release: sem.Release}}},
				RetryAfter: clock.Now().Add(time.Minute),
			}
			manager := &semaphoreMemoCapacityManager{response: denied}
			opts := NewQueueOptions(WithCapacityManager(manager), WithClock(clock), WithManualSemaphoreOwnership(tc.policy))
			processor := &queueProcessor{QueueOptions: opts}
			shadow := &QueueShadowPartition{AccountID: &tc.account, EnvID: &env, FunctionID: &fn}
			iter := &ProcessorIterator{
				Queue:     &mockQueueProcessor{shard: &mockShardForIterator{name: "test"}, opts: opts},
				Partition: &QueuePartition{ID: fn.String(), AccountID: tc.account, EnvID: &env, FunctionID: &fn},
				Leaser: mockQueueItemLeaser{fn: func(ctx context.Context, req LeaseItemRequest, _ DispatchFunc) (LeaseItemResult, error) {
					checked, err := processor.ItemLeaseConstraintCheck(ctx, shadow, &QueueBacklog{}, PartitionConstraintConfig{}, req.Item, clock.Now())
					require.NoError(t, err)
					if checked.CapacityLease != nil {
						return LeaseItemResult{Status: LeaseItemStatusDispatched}, nil
					}
					require.Equal(t, enums.QueueConstraintSemaphore, checked.LimitingConstraint)
					require.Equal(t, denied.RetryAfter, checked.RetryAfter)
					return LeaseItemResult{Status: LeaseItemStatusSemaphoreLimited,
						RetryAfter: checked.RetryAfter, ExhaustedSemaphores: checked.ExhaustedSemaphores}, nil
				}},
			}
			// A new run hits the full key before a later run whose existing
			// admission can be reused. Both go through the real queue boundary.
			require.NoError(t, iter.LeaseItem(ctx, semaphoreTestItem(tc.account, env, fn, clock.Now(), sem)))
			require.Len(t, denied.ExhaustedConstraints, 1, "preserve the actual API exhaustion report")
			manager.response = &constraintapi.CapacityAcquireResponse{Leases: []constraintapi.CapacityLease{{LeaseID: ulid.Make()}}}
			require.NoError(t, iter.LeaseItem(ctx, semaphoreTestItem(tc.account, env, fn, clock.Now(), sem)))
			require.Equal(t, tc.calls, manager.calls)
			require.EqualValues(t, tc.calls-1, iter.CtrSuccess.Load(), "only an owned manual retry bypasses shared memoization")
		})
	}
}

type semaphoreMemoCapacityManager struct {
	constraintapi.CapacityManager
	response *constraintapi.CapacityAcquireResponse
	calls    int
}

func (m *semaphoreMemoCapacityManager) Acquire(context.Context, *constraintapi.CapacityAcquireRequest) (*constraintapi.CapacityAcquireResponse, errs.InternalError) {
	m.calls++
	return m.response, nil
}

func TestOwnedManualRunRefusalDoesNotLeaseOrDispatch(t *testing.T) {
	for _, shared := range []bool{false, true} {
		t.Run(fmt.Sprintf("shared-exhaustion=%t", shared), func(t *testing.T) {
			ctx := t.Context()
			clock := clockwork.NewFakeClock()
			account, env, fn := uuid.New(), uuid.New(), uuid.New()
			sem := constraintapi.Semaphore{ID: constraintapi.SemaphoreIDFn(fn), Weight: 1, Release: constraintapi.SemaphoreReleaseManual}
			limited := []constraintapi.ConstraintItem{{Kind: constraintapi.ConstraintKindSemaphore,
				Semaphore: &constraintapi.SemaphoreConstraint{ID: sem.ID, Weight: 1, Release: sem.Release}}}
			response := &constraintapi.CapacityAcquireResponse{LimitingConstraints: limited, RetryAfter: clock.Now().Add(time.Minute)}
			if shared {
				response.ExhaustedConstraints = limited
			}
			shard := &ownedRefusalTestShard{mockShardForIterator: mockShardForIterator{name: "test"}}
			registry, err := NewSingleShardRegistry(shard)
			require.NoError(t, err)
			processor, err := New(ctx, "test", registry, WithClock(clock), WithNumWorkers(1),
				WithCapacityManager(&semaphoreMemoCapacityManager{response: response}),
				WithManualSemaphoreOwnership(func(uuid.UUID) bool { return true }))
			require.NoError(t, err)
			dispatched := false
			result, err := processor.LeaseItem(ctx, LeaseItemRequest{
				Item: semaphoreTestItem(account, env, fn, clock.Now(), sem), StaticTime: clock.Now(),
			}, func(context.Context, ProcessItem) (DispatchedItem, error) {
				dispatched = true
				return NewCompletedDispatchedItem(DispatchedItemResult{}), nil
			})
			require.NoError(t, err)
			require.Equal(t, LeaseItemStatusSemaphoreLimited, result.Status)
			require.Equal(t, response.RetryAfter, result.RetryAfter)
			require.Empty(t, result.ExhaustedSemaphores, "owned refusal cannot poison shared memos")
			require.Zero(t, shard.leaseCalls)
			require.False(t, dispatched)
		})
	}
}

type ownedRefusalTestShard struct {
	mockShardForIterator
	leaseCalls int
}

func (s *ownedRefusalTestShard) Lease(context.Context, QueueItem, time.Duration, time.Time, ...LeaseOptionFn) (*ulid.ULID, error) {
	s.leaseCalls++
	return nil, errors.New("refused item must not reach shard leasing")
}
