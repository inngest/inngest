package queue

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type mockProducer struct {
	called atomic.Bool
}

func (m *mockProducer) Enqueue(context.Context, Item, time.Time, EnqueueOpts) error {
	m.called.Store(true)
	return nil
}

func TestProcessorWithQueueProducerOverridesDefaultProducer(t *testing.T) {
	ctx := context.Background()
	shard := &mockShardForIterator{name: "shard-a"}
	registry, err := NewSingleShardRegistry(shard)
	require.NoError(t, err)

	producer := &mockProducer{}
	q, err := New(ctx, "test", registry, WithQueueProducer(producer))
	require.NoError(t, err)

	err = q.Enqueue(ctx, Item{}, time.Now(), EnqueueOpts{})
	require.NoError(t, err)
	require.True(t, producer.called.Load())
}

func TestProcessorAccountShardReadsResolveByDefault(t *testing.T) {
	ctx := context.Background()
	accountID := uuid.New()
	shardA := &mockShardForIterator{
		name:                 "shard-a",
		partitionBacklogSize: 1,
		outstandingJobCount:  2,
		runningCount:         3,
		statusCount:          4,
	}
	shardB := &mockShardForIterator{
		name:                 "shard-b",
		partitionBacklogSize: 10,
		outstandingJobCount:  20,
		runningCount:         30,
		statusCount:          40,
	}
	registry, err := NewShardRegistry(
		map[string]QueueShard{
			shardA.Name(): shardA,
			shardB.Name(): shardB,
		},
		WithPrimary(shardA),
		WithShardSelector(func(_ context.Context, scope Scope, _ *string) (QueueShard, error) {
			require.Equal(t, accountID, scope.AccountID)
			return shardB, nil
		}),
	)
	require.NoError(t, err)

	q, err := New(ctx, "test", registry)
	require.NoError(t, err)

	scope := Scope{AccountID: accountID, EnvID: uuid.New(), FunctionID: uuid.New()}

	backlogSize, err := q.PartitionBacklogSize(ctx, scope, "partition")
	require.NoError(t, err)
	require.Equal(t, int64(10), backlogSize)

	outstanding, err := q.OutstandingJobCount(ctx, scope, ulid.Make())
	require.NoError(t, err)
	require.Equal(t, 20, outstanding)

	running, err := q.RunningCount(ctx, scope)
	require.NoError(t, err)
	require.Equal(t, int64(30), running)

	status, err := q.StatusCount(ctx, scope, "status")
	require.NoError(t, err)
	require.Equal(t, int64(40), status)

	require.Equal(t, int32(0), atomic.LoadInt32(&shardA.partitionBacklogCalls))
	require.Equal(t, int32(0), atomic.LoadInt32(&shardA.outstandingJobCalls))
	require.Equal(t, int32(0), atomic.LoadInt32(&shardA.runningCountCalls))
	require.Equal(t, int32(0), atomic.LoadInt32(&shardA.statusCountCalls))
	require.Equal(t, int32(1), atomic.LoadInt32(&shardB.partitionBacklogCalls))
	require.Equal(t, int32(1), atomic.LoadInt32(&shardB.outstandingJobCalls))
	require.Equal(t, int32(1), atomic.LoadInt32(&shardB.runningCountCalls))
	require.Equal(t, int32(1), atomic.LoadInt32(&shardB.statusCountCalls))
}

func TestProcessorAccountShardReadsForEachWhenEnabled(t *testing.T) {
	ctx := context.Background()
	accountID := uuid.New()
	shardA := &mockShardForIterator{
		name:                 "shard-a",
		partitionBacklogSize: 1,
		outstandingJobCount:  2,
		runningCount:         3,
		statusCount:          4,
	}
	shardB := &mockShardForIterator{
		name:                 "shard-b",
		partitionBacklogSize: 10,
		outstandingJobCount:  20,
		runningCount:         30,
		statusCount:          40,
	}
	registry, err := NewShardRegistry(
		map[string]QueueShard{
			shardA.Name(): shardA,
			shardB.Name(): shardB,
		},
		WithPrimary(shardA),
		WithShardSelector(func(context.Context, Scope, *string) (QueueShard, error) {
			return shardB, nil
		}),
	)
	require.NoError(t, err)

	q, err := New(ctx, "test", registry, WithAccountShardIterationEnabled(func(_ context.Context, id uuid.UUID) bool {
		require.Equal(t, accountID, id)
		return true
	}))
	require.NoError(t, err)

	scope := Scope{AccountID: accountID, EnvID: uuid.New(), FunctionID: uuid.New()}

	backlogSize, err := q.PartitionBacklogSize(ctx, scope, "partition")
	require.NoError(t, err)
	require.Equal(t, int64(11), backlogSize)

	outstanding, err := q.OutstandingJobCount(ctx, scope, ulid.Make())
	require.NoError(t, err)
	require.Equal(t, 22, outstanding)

	running, err := q.RunningCount(ctx, scope)
	require.NoError(t, err)
	require.Equal(t, int64(33), running)

	status, err := q.StatusCount(ctx, scope, "status")
	require.NoError(t, err)
	require.Equal(t, int64(44), status)

	require.Equal(t, int32(1), atomic.LoadInt32(&shardA.partitionBacklogCalls))
	require.Equal(t, int32(1), atomic.LoadInt32(&shardA.outstandingJobCalls))
	require.Equal(t, int32(1), atomic.LoadInt32(&shardA.runningCountCalls))
	require.Equal(t, int32(1), atomic.LoadInt32(&shardA.statusCountCalls))
	require.Equal(t, int32(1), atomic.LoadInt32(&shardB.partitionBacklogCalls))
	require.Equal(t, int32(1), atomic.LoadInt32(&shardB.outstandingJobCalls))
	require.Equal(t, int32(1), atomic.LoadInt32(&shardB.runningCountCalls))
	require.Equal(t, int32(1), atomic.LoadInt32(&shardB.statusCountCalls))
}
