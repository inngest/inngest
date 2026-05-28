package queue

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestEnqueueUsesRedisShardEnqueueItemForAnyShardKind(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(1_700_000_000, 0).UTC()
	at := now.Add(5 * time.Second)

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	redisShard := redis_state.NewQueueShard(
		"custom",
		redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey),
		queue.WithClock(clockwork.NewFakeClockAt(now)),
	)
	shard := customKindRedisShard{
		RedisQueueShard: redisShard,
		kind:            enums.QueueShardKind("custom"),
	}
	shards, err := queue.NewShardRegistry(
		map[string]queue.QueueShard{shard.Name(): shard},
		queue.WithShardSelector(alwaysSelect(shard)),
		queue.WithPrimary(shard),
	)
	require.NoError(t, err)
	q, err := queue.New(
		ctx,
		"test-queue",
		shards,
		queue.WithClock(clockwork.NewFakeClockAt(now)),
	)
	require.NoError(t, err)

	workspaceID := uuid.New()
	workflowID := uuid.New()
	jobID := "enqueue-any-shard-kind"
	err = q.Enqueue(ctx, queue.Item{
		JobID:       &jobID,
		WorkspaceID: workspaceID,
		Kind:        queue.KindCron,
		Identifier: state.Identifier{
			AccountID:   uuid.New(),
			WorkspaceID: workspaceID,
			WorkflowID:  workflowID,
		},
	}, at, queue.EnqueueOpts{})
	require.NoError(t, err)

	loaded, err := q.LoadQueueItem(ctx, shard.Name(), queue.HashID(ctx, jobID))
	require.NoError(t, err)
	require.Equal(t, queue.HashID(ctx, jobID), loaded.ID)
	require.Equal(t, queue.KindCron, loaded.Data.Kind)
	require.Equal(t, workspaceID, loaded.WorkspaceID)
	require.Equal(t, workflowID, loaded.FunctionID)
	require.Equal(t, at.UnixMilli(), loaded.AtMS)
	require.NotNil(t, loaded.Data.JobID)
	require.Equal(t, queue.HashID(ctx, jobID), *loaded.Data.JobID)
}

type customKindRedisShard struct {
	redis_state.RedisQueueShard
	kind enums.QueueShardKind
}

func (c customKindRedisShard) Kind() enums.QueueShardKind {
	return c.kind
}
