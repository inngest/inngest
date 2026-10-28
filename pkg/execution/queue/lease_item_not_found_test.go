package queue

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/oklog/ulid/v2"
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
	}{
		{name: "default off", disable: true},
		{name: "explicitly disabled", disable: true, explicitOff: true},
		{name: "fresh lease is released", wantRelease: true},
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
