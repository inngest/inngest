package redis_state

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func BenchmarkKeyQueues(b *testing.B) {
	fmt.Println("benchmarking with", b.N)

	r := miniredis.RunT(b)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(b, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	defaultShard := RedisQueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}

	q := NewQueue(
		defaultShard,
		WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
			return true
		}),
		WithKindToQueueMapping(map[string]string{
			osqueue.KindPause:           osqueue.KindPause,
			osqueue.KindDebounce:        osqueue.KindDebounce,
			osqueue.KindQueueMigrate:    osqueue.KindQueueMigrate,
			osqueue.KindPauseBlockFlush: osqueue.KindPauseBlockFlush,
			osqueue.KindScheduleBatch:   osqueue.KindScheduleBatch,
		}),
		WithBacklogRefillLimit(10),
		WithClock(clock),
		WithRunMode(QueueRunMode{
			Sequential:                        true,
			Scavenger:                         true,
			Partition:                         true,
			Account:                           true,
			AccountWeight:                     80,
			Continuations:                     true,
			ShadowPartition:                   true,
			AccountShadowPartition:            true,
			AccountShadowPartitionWeight:      80,
			ShadowContinuations:               true,
			ShadowContinuationSkipProbability: consts.QueueContinuationSkipProbability,
			NormalizePartition:                true,
		}),
		WithShardSelector(func(ctx context.Context, accountId uuid.UUID, queueName *string) (RedisQueueShard, error) {
			return defaultShard, nil
		}),
	)
	ctx := context.Background()

	accountID, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	var counter int64

	withTimeout, cancelWithTimeout := context.WithTimeout(ctx, 1*time.Minute)
	defer cancelWithTimeout()

	withCancelAfterDone, cancelAfterDone := context.WithCancel(withTimeout)
	defer cancelAfterDone()

	eg := errgroup.Group{}
	eg.Go(func() error {
		err = q.Run(withCancelAfterDone, func(ctx context.Context, info osqueue.RunInfo, item osqueue.Item) (osqueue.RunResult, error) {
			current := int(atomic.LoadInt64(&counter))
			fmt.Println("current: ", current)
			if current >= b.N {
				cancelAfterDone()
			}

			atomic.AddInt64(&counter, 1)

			return osqueue.RunResult{}, nil
		})
		require.NoError(b, withTimeout.Err())

		return nil
	})

	for i := 0; i < b.N; i++ {
		err = q.Enqueue(ctx, osqueue.Item{
			WorkspaceID: wsID,
			Kind:        osqueue.KindEdge,
			Identifier: state.Identifier{
				WorkflowID:      fnID,
				WorkflowVersion: 1,
				AccountID:       accountID,
				WorkspaceID:     wsID,
			},
		}, q.clock.Now(), osqueue.EnqueueOpts{})
		require.NoError(b, err)
	}

	require.NoError(b, eg.Wait())
}
