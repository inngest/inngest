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

func TestBacklogRefillConstraintCheckMissingAccountReturnsEmptyResult(t *testing.T) {
	ctx := context.Background()
	accountID := uuid.New()
	envID := uuid.New()
	fnID := uuid.New()
	appID := uuid.New()

	shard := &mockShardForIterator{name: "test-shard"}
	registry, err := NewSingleShardRegistry(shard)
	require.NoError(t, err)

	q, err := New(
		ctx,
		"test",
		registry,
		WithCapacityManager(backlogRefillMissingAccountCapacityManager{}),
		WithAcquireCapacityLeaseOnBacklogRefill(true),
	)
	require.NoError(t, err)

	res, err := q.BacklogRefillConstraintCheck(
		ctx,
		&QueueShadowPartition{
			PartitionID: fnID.String(),
			AccountID:   &accountID,
			EnvID:       &envID,
			FunctionID:  &fnID,
		},
		&QueueBacklog{
			ShadowPartitionID: fnID.String(),
		},
		PartitionConstraintConfig{
			Concurrency: PartitionConcurrency{
				AccountConcurrency: 1,
			},
		},
		[]*QueueItem{
			{
				ID: "item-1",
				Data: Item{
					Identifier: state.Identifier{
						RunID: ulid.Make(),
						AppID: appID,
					},
				},
			},
		},
		"op-1",
		time.Now(),
	)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Empty(t, res.ItemsToRefill)
}

type backlogRefillMissingAccountCapacityManager struct{}

func (backlogRefillMissingAccountCapacityManager) Check(context.Context, *constraintapi.CapacityCheckRequest) (*constraintapi.CapacityCheckResponse, errs.UserError, errs.InternalError) {
	return nil, nil, nil
}

func (backlogRefillMissingAccountCapacityManager) Acquire(context.Context, *constraintapi.CapacityAcquireRequest) (*constraintapi.CapacityAcquireResponse, errs.InternalError) {
	return nil, wrappedConstraintAPIInternalError{err: constraintapi.ErrAccountNotFound}
}

func (backlogRefillMissingAccountCapacityManager) ExtendLease(context.Context, *constraintapi.CapacityExtendLeaseRequest) (*constraintapi.CapacityExtendLeaseResponse, errs.InternalError) {
	return nil, nil
}

func (backlogRefillMissingAccountCapacityManager) Release(context.Context, *constraintapi.CapacityReleaseRequest) (*constraintapi.CapacityReleaseResponse, errs.InternalError) {
	return nil, nil
}

type wrappedConstraintAPIInternalError struct {
	err error
}

func (e wrappedConstraintAPIInternalError) Error() string {
	return e.err.Error()
}

func (e wrappedConstraintAPIInternalError) Unwrap() error {
	return e.err
}

func (wrappedConstraintAPIInternalError) ErrorCode() int {
	return 0
}

func (wrappedConstraintAPIInternalError) Retryable() bool {
	return false
}

func (wrappedConstraintAPIInternalError) RetryAfter() time.Duration {
	return 0
}

func (wrappedConstraintAPIInternalError) InternalError() {}
