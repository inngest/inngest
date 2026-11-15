package queue

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution/state"
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
