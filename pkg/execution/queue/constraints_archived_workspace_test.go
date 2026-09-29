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

func TestBypassArchivedWorkspaceAppSemaphore(t *testing.T) {
	accountID := uuid.New()
	workspaceID := uuid.New()
	appSemaphore := constraintapi.Semaphore{ID: "app:" + uuid.NewString()}
	functionSemaphore := constraintapi.Semaphore{ID: "fn:" + uuid.NewString()}

	tests := []struct {
		name          string
		semaphores    []constraintapi.Semaphore
		callbackValue bool
		want          bool
		wantCalls     int
	}{
		{
			name:          "app semaphore and callback enabled",
			semaphores:    []constraintapi.Semaphore{appSemaphore},
			callbackValue: true,
			want:          true,
			wantCalls:     1,
		},
		{
			name:       "app semaphore and callback disabled",
			semaphores: []constraintapi.Semaphore{appSemaphore},
			wantCalls:  1,
		},
		{
			name:       "non-app semaphore does not check workspace",
			semaphores: []constraintapi.Semaphore{functionSemaphore},
		},
		{
			name: "no semaphores does not check workspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			processor := &queueProcessor{QueueOptions: NewQueueOptions(
				WithBypassArchivedWorkspaceAppSemaphore(func(_ context.Context, gotAccountID, gotWorkspaceID uuid.UUID) bool {
					calls++
					require.Equal(t, accountID, gotAccountID)
					require.Equal(t, workspaceID, gotWorkspaceID)
					return tt.callbackValue
				}),
			)}

			got := processor.bypassArchivedWorkspaceAppSemaphore(
				context.Background(),
				accountID,
				workspaceID,
				tt.semaphores,
			)
			require.Equal(t, tt.want, got)
			require.Equal(t, tt.wantCalls, calls)
		})
	}
}

func TestArchivedWorkspaceAppSemaphoreBypassPreservesOtherSemaphores(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	accountID := uuid.New()
	workspaceID := uuid.New()
	functionID := uuid.New()
	appID := uuid.New()
	appSemaphore := constraintapi.Semaphore{ID: "app:" + appID.String(), Weight: 1}
	functionSemaphore := constraintapi.Semaphore{ID: "fn:" + functionID.String(), Weight: 1}

	require.True(t, hasAppSemaphoreConfig([]constraintapi.Semaphore{appSemaphore, functionSemaphore}))
	require.True(t, hasNonAppSemaphore([]constraintapi.Semaphore{appSemaphore, functionSemaphore}))
	require.False(t, hasNonAppSemaphore([]constraintapi.Semaphore{appSemaphore}))

	tests := []struct {
		name          string
		capacityLease *CapacityLease
	}{
		{name: "acquire all item capacity"},
		{
			name: "reuse backlog capacity lease",
			capacityLease: &CapacityLease{
				LeaseID:    ulid.MustNew(ulid.Timestamp(now.Add(2*QueueLeaseDuration)), nil),
				IssuedAtMS: now.UnixMilli(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &archivedWorkspaceCapacityManager{}
			shard := &mockShardForIterator{name: "test-shard"}
			registry, err := NewSingleShardRegistry(shard)
			require.NoError(t, err)
			processor, err := New(
				ctx,
				"test",
				registry,
				WithCapacityManager(manager),
				WithBypassArchivedWorkspaceAppSemaphore(func(context.Context, uuid.UUID, uuid.UUID) bool {
					return true
				}),
			)
			require.NoError(t, err)

			result, err := processor.ItemLeaseConstraintCheck(
				ctx,
				&QueueShadowPartition{
					PartitionID: functionID.String(),
					AccountID:   &accountID,
					EnvID:       &workspaceID,
					FunctionID:  &functionID,
				},
				&QueueBacklog{ShadowPartitionID: functionID.String()},
				PartitionConstraintConfig{},
				&QueueItem{
					ID:            "item-1",
					FunctionID:    functionID,
					WorkspaceID:   workspaceID,
					CapacityLease: tt.capacityLease,
					Data: Item{
						Identifier: state.Identifier{
							AccountID:   accountID,
							WorkspaceID: workspaceID,
							WorkflowID:  functionID,
							AppID:       appID,
							RunID:       ulid.Make(),
						},
						Semaphores: []constraintapi.Semaphore{appSemaphore, functionSemaphore},
					},
				},
				now,
			)
			require.NoError(t, err)
			require.True(t, result.ArchivedWorkspaceAppSemaphoreBypassed)
			require.Len(t, manager.acquireRequests, 1)
			var acquiredSemaphores []constraintapi.SemaphoreConstraint
			for _, constraint := range manager.acquireRequests[0].Constraints {
				if constraint.Kind == constraintapi.ConstraintKindSemaphore {
					require.NotNil(t, constraint.Semaphore)
					acquiredSemaphores = append(acquiredSemaphores, *constraint.Semaphore)
				}
			}
			require.Len(t, acquiredSemaphores, 1)
			require.Equal(t, functionSemaphore.ID, acquiredSemaphores[0].ID)
			require.Equal(t, []constraintapi.Semaphore{functionSemaphore}, manager.acquireRequests[0].Configuration.Semaphores)
		})
	}
}

type archivedWorkspaceCapacityManager struct {
	acquireRequests []*constraintapi.CapacityAcquireRequest
}

func (m *archivedWorkspaceCapacityManager) Check(context.Context, *constraintapi.CapacityCheckRequest) (*constraintapi.CapacityCheckResponse, errs.UserError, errs.InternalError) {
	return nil, nil, nil
}

func (m *archivedWorkspaceCapacityManager) Acquire(_ context.Context, req *constraintapi.CapacityAcquireRequest) (*constraintapi.CapacityAcquireResponse, errs.InternalError) {
	m.acquireRequests = append(m.acquireRequests, req)
	return &constraintapi.CapacityAcquireResponse{
		Leases: []constraintapi.CapacityLease{{
			LeaseID:        ulid.Make(),
			IdempotencyKey: req.IdempotencyKey,
		}},
	}, nil
}

func (m *archivedWorkspaceCapacityManager) ExtendLease(context.Context, *constraintapi.CapacityExtendLeaseRequest) (*constraintapi.CapacityExtendLeaseResponse, errs.InternalError) {
	return nil, nil
}

func (m *archivedWorkspaceCapacityManager) Release(context.Context, *constraintapi.CapacityReleaseRequest) (*constraintapi.CapacityReleaseResponse, errs.InternalError) {
	return nil, nil
}
