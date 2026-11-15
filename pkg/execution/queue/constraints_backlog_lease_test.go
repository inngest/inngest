package queue

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// A backlog capacity lease does not cover an item's semaphores, so dispatch
// acquires a new lease.  The backlog lease must be released at that point,
// otherwise it holds the function concurrency slot until it expires.
func TestItemLeaseConstraintCheckReleasesBacklogLeaseWithSemaphores(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	accountID, workspaceID, functionID, appID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	appSemaphore := constraintapi.Semaphore{ID: "app:" + appID.String(), Weight: 1}

	manager := &archivedWorkspaceCapacityManager{}
	registry, err := NewSingleShardRegistry(&mockShardForIterator{name: "test-shard"})
	require.NoError(t, err)
	processor, err := New(ctx, "test", registry, WithCapacityManager(manager))
	require.NoError(t, err)

	backlogLease := &CapacityLease{
		LeaseID:    ulid.MustNew(ulid.Timestamp(now.Add(2*QueueLeaseDuration)), nil),
		IssuedAtMS: now.UnixMilli(),
	}
	result, err := processor.ItemLeaseConstraintCheck(
		ctx,
		&QueueShadowPartition{
			PartitionID: functionID.String(),
			AccountID:   &accountID,
			EnvID:       &workspaceID,
			FunctionID:  &functionID,
		},
		&QueueBacklog{ShadowPartitionID: functionID.String()},
		PartitionConstraintConfig{
			Concurrency: PartitionConcurrency{
				AccountConcurrency:  NoConcurrencyLimit,
				FunctionConcurrency: 20,
			},
		},
		&QueueItem{
			ID:            "item-1",
			FunctionID:    functionID,
			WorkspaceID:   workspaceID,
			CapacityLease: backlogLease,
			Data: Item{
				Identifier: state.Identifier{
					AccountID:   accountID,
					WorkspaceID: workspaceID,
					WorkflowID:  functionID,
					AppID:       appID,
					RunID:       ulid.Make(),
				},
				Semaphores: []constraintapi.Semaphore{appSemaphore},
			},
		},
		now,
	)
	require.NoError(t, err)
	require.NotNil(t, result.CapacityLease)
	require.NotEqual(t, backlogLease.LeaseID, result.CapacityLease.LeaseID)

	// The backlog lease is released under its own idempotency key.
	require.Len(t, manager.releaseRequests, 1)
	require.Equal(t, backlogLease.LeaseID, manager.releaseRequests[0].LeaseID)
	require.Equal(t, "item-1:backlog", manager.releaseRequests[0].IdempotencyKey)

	// The new lease holds function concurrency as well as the semaphore, as the
	// released backlog lease no longer does.
	require.Len(t, manager.acquireRequests, 1)
	var kinds []constraintapi.ConstraintKind
	for _, c := range manager.acquireRequests[0].Constraints {
		kinds = append(kinds, c.Kind)
	}
	require.Contains(t, kinds, constraintapi.ConstraintKindConcurrency)
	require.Contains(t, kinds, constraintapi.ConstraintKindSemaphore)
}

type backlogLeaseTestManager struct {
	archivedWorkspaceCapacityManager
	exhaustAcquire bool
	extendRequests []*constraintapi.CapacityExtendLeaseRequest
}

func (m *backlogLeaseTestManager) Acquire(ctx context.Context, req *constraintapi.CapacityAcquireRequest) (*constraintapi.CapacityAcquireResponse, errs.InternalError) {
	if m.exhaustAcquire {
		m.acquireRequests = append(m.acquireRequests, req)
		return &constraintapi.CapacityAcquireResponse{}, nil
	}
	return m.archivedWorkspaceCapacityManager.Acquire(ctx, req)
}

func (m *backlogLeaseTestManager) ExtendLease(_ context.Context, req *constraintapi.CapacityExtendLeaseRequest) (*constraintapi.CapacityExtendLeaseResponse, errs.InternalError) {
	m.extendRequests = append(m.extendRequests, req)
	// the lease was already released: no new lease ID
	return &constraintapi.CapacityExtendLeaseResponse{}, nil
}

func runBacklogLeaseCheck(t *testing.T, manager constraintapi.CapacityManager, backlog *QueueBacklog, constraints PartitionConstraintConfig, lease *CapacityLease, now time.Time) (*QueueItem, ItemLeaseConstraintCheckResult, error) {
	t.Helper()
	ctx := context.Background()
	accountID, workspaceID, functionID, appID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	registry, err := NewSingleShardRegistry(&mockShardForIterator{name: "test-shard"})
	require.NoError(t, err)
	processor, err := New(ctx, "test", registry, WithCapacityManager(manager))
	require.NoError(t, err)
	backlog.ShadowPartitionID = functionID.String()
	item := &QueueItem{
		ID:            "item-1",
		FunctionID:    functionID,
		WorkspaceID:   workspaceID,
		CapacityLease: lease,
		Data: Item{
			Identifier: state.Identifier{AccountID: accountID, WorkspaceID: workspaceID, WorkflowID: functionID, AppID: appID, RunID: ulid.Make()},
			Semaphores: []constraintapi.Semaphore{{ID: "app:" + appID.String(), Weight: 1}},
		},
	}
	res, err := processor.ItemLeaseConstraintCheck(ctx, &QueueShadowPartition{
		PartitionID: functionID.String(), AccountID: &accountID, EnvID: &workspaceID, FunctionID: &functionID,
	}, backlog, constraints, item, now)
	return item, res, err
}

var throttleBacklogConstraints = PartitionConstraintConfig{
	Concurrency: PartitionConcurrency{AccountConcurrency: NoConcurrencyLimit, FunctionConcurrency: 20},
	Throttle:    &PartitionThrottle{ThrottleKeyExpressionHash: "expr", Limit: 1, Burst: 0, Period: 60},
}

func throttledBacklog() *QueueBacklog {
	return &QueueBacklog{Throttle: &BacklogThrottle{ThrottleKey: "key", ThrottleKeyExpressionHash: "expr"}}
}

// The replacement acquire must not include throttle: the item paid its token
// with the backlog lease, and a throttle constraint cache entry would reject
// the acquire before the GCRA skip applies (acquire cache enabled).
func TestItemLeaseConstraintCheckReplacementAcquireOmitsThrottle(t *testing.T) {
	now := time.Now()
	manager := &backlogLeaseTestManager{}
	lease := &CapacityLease{LeaseID: ulid.MustNew(ulid.Timestamp(now.Add(2*QueueLeaseDuration)), nil), IssuedAtMS: now.UnixMilli()}
	_, res, err := runBacklogLeaseCheck(t, manager, throttledBacklog(), throttleBacklogConstraints, lease, now)
	require.NoError(t, err)
	require.NotNil(t, res.CapacityLease)
	require.Len(t, manager.acquireRequests, 1)
	for _, c := range manager.acquireRequests[0].Constraints {
		require.NotEqual(t, constraintapi.ConstraintKindThrottle, c.Kind)
	}
}

// If the replacement acquire is rejected, the item stays ready.  It must not
// keep pointing at the released backlog lease.
func TestItemLeaseConstraintCheckClearsReleasedBacklogLease(t *testing.T) {
	now := time.Now()
	manager := &backlogLeaseTestManager{exhaustAcquire: true}
	lease := &CapacityLease{LeaseID: ulid.MustNew(ulid.Timestamp(now.Add(2*QueueLeaseDuration)), nil), IssuedAtMS: now.UnixMilli()}
	item, res, err := runBacklogLeaseCheck(t, manager, &QueueBacklog{}, throttleBacklogConstraints, lease, now)
	require.NoError(t, err)
	require.Nil(t, res.CapacityLease)
	require.Len(t, manager.releaseRequests, 1)
	require.Nil(t, item.CapacityLease)
}

// A stale lease that was already released must not fail the check: extending
// it finds nothing, so the check falls through to a fresh acquire.
func TestItemLeaseConstraintCheckStaleLeaseFallsThroughToAcquire(t *testing.T) {
	now := time.Now()
	manager := &backlogLeaseTestManager{}
	// active but too close to expiry to be reused, so an extend is attempted
	lease := &CapacityLease{LeaseID: ulid.MustNew(ulid.Timestamp(now.Add(time.Second)), nil), IssuedAtMS: now.UnixMilli()}
	_, res, err := runBacklogLeaseCheck(t, manager, &QueueBacklog{}, throttleBacklogConstraints, lease, now)
	require.NoError(t, err)
	require.Len(t, manager.extendRequests, 1)
	require.Len(t, manager.acquireRequests, 1)
	require.NotNil(t, res.CapacityLease)
	require.NotEqual(t, lease.LeaseID, res.CapacityLease.LeaseID)
}
