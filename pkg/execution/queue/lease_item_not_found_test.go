package queue

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// goneItemShard reports every item lease as not found: the item was removed
// (for example by a cancellation) after the capacity acquire.
type goneItemShard struct {
	*mockShardForIterator
}

func (goneItemShard) Lease(context.Context, QueueItem, time.Duration, time.Time, ...LeaseOptionFn) (*ulid.ULID, error) {
	return nil, ErrQueueItemNotFound
}

// grantingCapacityManager grants every Acquire and records Releases.
type grantingCapacityManager struct {
	replayed bool
	lease    ulid.ULID

	mu       sync.Mutex
	acquires int
	releases []*constraintapi.CapacityReleaseRequest
}

func (m *grantingCapacityManager) Check(context.Context, *constraintapi.CapacityCheckRequest) (*constraintapi.CapacityCheckResponse, errs.UserError, errs.InternalError) {
	return nil, nil, nil
}

func (m *grantingCapacityManager) Acquire(_ context.Context, req *constraintapi.CapacityAcquireRequest) (*constraintapi.CapacityAcquireResponse, errs.InternalError) {
	m.mu.Lock()
	m.acquires++
	m.mu.Unlock()
	return &constraintapi.CapacityAcquireResponse{
		Leases:                  []constraintapi.CapacityLease{{LeaseID: m.lease, IdempotencyKey: req.IdempotencyKey}},
		OperationIdempotencyHit: m.replayed,
	}, nil
}

func (m *grantingCapacityManager) ExtendLease(context.Context, *constraintapi.CapacityExtendLeaseRequest) (*constraintapi.CapacityExtendLeaseResponse, errs.InternalError) {
	return nil, nil
}

func (m *grantingCapacityManager) Release(_ context.Context, req *constraintapi.CapacityReleaseRequest) (*constraintapi.CapacityReleaseResponse, errs.InternalError) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.releases = append(m.releases, req)
	return &constraintapi.CapacityReleaseResponse{}, nil
}

// When the item lease finds the item gone, capacity acquired for this attempt
// is released; a lease replayed from operation idempotency (possibly held by
// another attempt for the same item) is not.
func TestLeaseItemReleasesCapacityWhenItemGone(t *testing.T) {
	for _, tt := range []struct {
		name        string
		replayed    bool
		wantRelease bool
		disable     bool
		explicitOff bool
		reused      bool
		semaphore   *constraintapi.Semaphore
	}{
		{name: "default off", disable: true},
		{name: "explicitly disabled", disable: true, explicitOff: true},
		{name: "fresh lease is released", wantRelease: true},
		{name: "manual permit preserves recovery", semaphore: &constraintapi.Semaphore{ID: "fn:manual", Weight: 2, Release: constraintapi.SemaphoreReleaseManual}},
		{name: "auto permit is released", wantRelease: true, semaphore: &constraintapi.Semaphore{ID: "app:auto", Weight: 1, Release: constraintapi.SemaphoreReleaseAuto}},
		{name: "replayed lease is kept", replayed: true},
		{name: "lease reused from the item is kept", reused: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			shard := goneItemShard{&mockShardForIterator{name: "test"}}
			registry, err := NewSingleShardRegistry(shard)
			require.NoError(t, err)
			cm := &grantingCapacityManager{replayed: tt.replayed, lease: ulid.MustNew(ulid.Timestamp(time.Now().Add(30*time.Second)), nil)}
			options := []QueueOpt{
				WithCapacityManager(cm),
				WithPartitionConstraintConfigGetter(func(context.Context, PartitionIdentifier) PartitionConstraintConfig {
					return PartitionConstraintConfig{Concurrency: PartitionConcurrency{AccountConcurrency: 1}}
				}),
			}
			if !tt.disable {
				options = append(options, WithReleaseCapacityOnItemGone(true))
			}
			if tt.explicitOff {
				options = append(options, WithReleaseCapacityOnItemGone(false))
			}
			q, err := New(ctx, "test", registry, options...)
			require.NoError(t, err)

			accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
			item := &QueueItem{
				ID:          "item",
				FunctionID:  fnID,
				WorkspaceID: envID,
				Data: Item{Identifier: state.Identifier{
					AccountID: accountID, WorkspaceID: envID, WorkflowID: fnID, RunID: ulid.Make(),
				}},
			}
			if tt.semaphore != nil {
				item.Data.Semaphores = []constraintapi.Semaphore{*tt.semaphore}
			}
			if tt.reused {
				// A lease the item already holds (e.g. from a backlog refill) is
				// reused without an Acquire and must not be released here.
				item.CapacityLease = &CapacityLease{
					LeaseID:    ulid.MustNew(ulid.Timestamp(time.Now().Add(QueueLeaseDuration)), nil),
					IssuedAtMS: time.Now().UnixMilli(),
				}
			}
			res, err := q.LeaseItem(ctx, LeaseItemRequest{Item: item}, func(context.Context, ProcessItem) (DispatchedItem, error) {
				t.Fatal("a missing item must not be dispatched")
				return nil, nil
			})
			require.NoError(t, err)
			require.Equal(t, LeaseItemStatusNotFound, res.Status)

			cm.mu.Lock()
			defer cm.mu.Unlock()
			if tt.reused {
				require.Zero(t, cm.acquires, "a reusable item lease must not be re-acquired")
			}
			if !tt.wantRelease {
				require.Empty(t, cm.releases)
				return
			}
			require.Len(t, cm.releases, 1)
			require.Equal(t, cm.lease, cm.releases[0].LeaseID)
			require.Equal(t, accountID, cm.releases[0].AccountID)
			require.Equal(t, "item:not-found:"+cm.lease.String(), cm.releases[0].IdempotencyKey)
		})
	}
}

// Cancellation has already removed the item and released the run's manual
// permits when a stale scanner acquires a new lease. Only the scavenger can
// recover that new manual permit: ordinary Release removes its recovery state
// without decrementing manual semaphore usage.
func TestMissingItemPreservesManualPermitScavengerRecovery(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "cleanup_off", true: "cleanup_on"}[enabled], func(t *testing.T) {
			ctx := context.Background()
			r := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
			require.NoError(t, err)
			t.Cleanup(rc.Close)
			clock := clockwork.NewFakeClock()
			cm, err := constraintapi.NewRedisCapacityManager(constraintapi.WithClient(rc), constraintapi.WithClock(clock), constraintapi.WithShardName("missing-manual"))
			require.NoError(t, err)
			registry, err := NewSingleShardRegistry(goneItemShard{&mockShardForIterator{name: "test"}})
			require.NoError(t, err)
			q, err := New(ctx, "test", registry, WithClock(clock), WithCapacityManager(cm), WithReleaseCapacityOnItemGone(enabled), WithPartitionConstraintConfigGetter(func(context.Context, PartitionIdentifier) PartitionConstraintConfig {
				return PartitionConstraintConfig{FunctionVersion: 1, Concurrency: PartitionConcurrency{AccountConcurrency: 10, FunctionConcurrency: 10}}
			}))
			require.NoError(t, err)
			acct, env, fn := uuid.New(), uuid.New(), uuid.New()
			sem := constraintapi.SemaphoreConstraint{ID: "fn:" + fn.String(), EvaluatedKeyHash: "tenant", Weight: 1, Release: constraintapi.SemaphoreReleaseManual}
			require.NoError(t, r.Set(sem.CapacityKey(acct), "1"))
			item := &QueueItem{ID: "cancelled-item", FunctionID: fn, WorkspaceID: env, Data: Item{Identifier: state.Identifier{AccountID: acct, WorkspaceID: env, WorkflowID: fn, WorkflowVersion: 1, RunID: ulid.Make()}, Semaphores: []constraintapi.Semaphore{{ID: sem.ID, EvaluatedKeyHash: sem.EvaluatedKeyHash, Weight: 1, Release: sem.Release}}}}
			result, err := q.LeaseItem(ctx, LeaseItemRequest{Item: item}, func(context.Context, ProcessItem) (DispatchedItem, error) {
				t.Fatal("cancelled item must not dispatch")
				return nil, nil
			})
			require.NoError(t, err)
			require.Equal(t, LeaseItemStatusNotFound, result.Status)
			used, err := r.Get(sem.UsageKey(acct))
			require.NoError(t, err)
			require.Equal(t, "1", used)
			clock.Advance(QueueLeaseDuration + time.Second)
			r.FastForward(QueueLeaseDuration + time.Second)
			r.SetTime(clock.Now())
			reclaimed, err := cm.Scavenge(ctx)
			require.NoError(t, err)
			require.Equal(t, 1, reclaimed.ReclaimedLeases, "missing-item cleanup must retain the manual permit's recovery lease")
			used, err = r.Get(sem.UsageKey(acct))
			require.NoError(t, err)
			require.Equal(t, "0", used, "scavenging must recover the manual slot")
		})
	}
}
