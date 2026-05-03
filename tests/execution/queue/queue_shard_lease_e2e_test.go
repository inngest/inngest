package queue

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

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

	shardRegistry, err := queue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	q, err := queue.New(ctx, "test", shardRegistry)
	require.NoError(t, err)

	// Verify primaryQueueShard is set
	require.Equal(t, shard, q.Shard())
}
