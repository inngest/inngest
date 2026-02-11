package queue

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestNewQueueRequiresPrimaryShardOrShardGroup(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	_, err := queue.New(ctx, "test", nil, nil, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must pass either primary queue shard or a valid ShardGroup in runMode")
}

func TestNewQueueCreationWithPrimaryShard(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	queueClient := redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey)
	shard := redis_state.NewQueueShard("test-shard", queueClient)

	q, err := queue.New(ctx, "test", shard, nil, nil)
	require.NoError(t, err)

	// Verify primaryQueueShard is set
	require.Equal(t, shard, q.Shard())
}

func TestNewQueueWithNoValidShardsInGroup(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	// One shard belonging to group "A"
	queueClient := redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey)
	shard := redis_state.NewQueueShard("shard-a", queueClient,
		queue.WithShardAssignmentConfig(queue.ShardAssignmentConfig{
			ShardGroup:   "A",
			NumExecutors: 1,
		}),
	)

	queueShards := map[string]queue.QueueShard{
		"shard-a": shard,
	}

	// Runtime expects group "B", but no shards belong to that group
	_, err = queue.New(ctx, "test", nil, queueShards, nil,
		queue.WithRunMode(queue.QueueRunMode{
			ShardGroup: "B",
		}),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "No shards found for configured shard group: B")
}

func TestNewQueueWithShardAssignment(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	groupName := "test-group"

	// Shard A: belongs to the shard group
	queueClientA := redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey)
	shardA := redis_state.NewQueueShard("shard-a", queueClientA,
		queue.WithShardAssignmentConfig(queue.ShardAssignmentConfig{
			ShardGroup:   groupName,
			NumExecutors: 1,
		}),
		queue.WithClock(clock),
	)

	// Shard B: no group config, should never be selected
	queueClientB := redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey)
	shardB := redis_state.NewQueueShard("shard-b", queueClientB,
		queue.WithClock(clock),
	)

	queueShards := map[string]queue.QueueShard{
		"shard-a": shardA,
		"shard-b": shardB,
	}

	q, err := queue.New(ctx, "test", nil, queueShards, nil,
		queue.WithClock(clock),
		queue.WithRunMode(queue.QueueRunMode{
			ShardGroup: groupName,
		}),
	)
	require.NoError(t, err)

	// Primary shard should be nil before Run
	require.Nil(t, q.Shard())

	// Start Run in background; it will block on claimShardLease until the
	// fake clock ticks past ShardLeaseDuration/3.
	go func() {
		_ = q.Run(ctx, func(ctx context.Context, ri queue.RunInfo, i queue.Item) (queue.RunResult, error) {
			return queue.RunResult{}, nil
		})
	}()

	// Advance the fake clock past the shard lease tick interval (ShardLeaseDuration/3 ≈ 3.3s)
	// so claimShardLease fires and acquires the lease.
	require.Eventually(t, func() bool {
		clock.Advance(4 * time.Second)
		r.SetTime(clock.Now())
		return q.Shard() != nil
	}, 5*time.Second, 100*time.Millisecond)

	// The primary shard must be shard-a (the one with matching group config)
	require.Equal(t, "shard-a", q.Shard().Name())
}
