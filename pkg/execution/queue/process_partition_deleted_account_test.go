package queue

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

func TestProcessPartitionRequeuesDeletedAccount(t *testing.T) {
	ctx := context.Background()
	now := clockwork.NewFakeClock()
	accountID := uuid.New()
	fnID := uuid.New()

	shard := &mockShardForIterator{name: "test-shard"}
	registry, err := NewSingleShardRegistry(shard)
	require.NoError(t, err)

	var checks int32
	q, err := New(
		ctx,
		"test",
		registry,
		WithClock(now),
		WithAccountExists(func(context.Context, uuid.UUID) (bool, error) {
			atomic.AddInt32(&checks, 1)
			return false, nil
		}),
	)
	require.NoError(t, err)

	err = q.ProcessPartition(ctx, &QueuePartition{
		ID:         fnID.String(),
		FunctionID: &fnID,
		AccountID:  accountID,
	}, 0, false, func(_ context.Context, item ProcessItem) error {
		q.workers <- item
		return nil
	})
	require.NoError(t, err)

	require.Equal(t, int32(1), atomic.LoadInt32(&checks))
	require.Equal(t, int32(1), atomic.LoadInt32(&shard.partitionLeaseCount))
	require.Equal(t, int32(1), atomic.LoadInt32(&shard.partitionRequeueCount))
	require.True(t, shard.partitionRequeueForceAt)
	require.Equal(t, now.Now().Add(PartitionDeletedAccountRequeueExtension), shard.partitionRequeueAt)
}

func TestProcessShadowPartitionRequeuesDeletedAccountBeforePeek(t *testing.T) {
	ctx := context.Background()
	now := clockwork.NewFakeClock()
	accountID := uuid.New()
	envID := uuid.New()
	fnID := uuid.New()

	shard := &mockShardForIterator{name: "test-shard"}
	registry, err := NewSingleShardRegistry(shard)
	require.NoError(t, err)

	var checks int32
	q, err := New(
		ctx,
		"test",
		registry,
		WithClock(now),
		WithAccountExists(func(context.Context, uuid.UUID) (bool, error) {
			atomic.AddInt32(&checks, 1)
			return false, nil
		}),
	)
	require.NoError(t, err)

	err = q.ProcessShadowPartition(ctx, &QueueShadowPartition{
		PartitionID: fnID.String(),
		FunctionID:  &fnID,
		AccountID:   &accountID,
		EnvID:       &envID,
	}, 0)
	require.NoError(t, err)

	require.Equal(t, int32(1), atomic.LoadInt32(&checks))
	require.Equal(t, int32(0), atomic.LoadInt32(&shard.shadowPartitionLeaseCount))
	require.Equal(t, int32(1), atomic.LoadInt32(&shard.shadowPartitionRequeueCount))
	require.Equal(t, int32(0), atomic.LoadInt32(&shard.shadowPartitionPeekCount))
	require.Equal(t, int32(0), atomic.LoadInt32(&shard.backlogPeekCount))
	require.NotNil(t, shard.shadowPartitionRequeueAt)
	require.Equal(t, now.Now().Add(PartitionDeletedAccountRequeueExtension), *shard.shadowPartitionRequeueAt)
}
