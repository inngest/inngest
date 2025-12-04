package redis_state

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestQueueItemProcessWithConstraintChecks(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	shard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}

	q := NewQueue(
		shard,
		WithClock(clock),
		WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
			return true
		}),
	)
	ctx := context.Background()

	accountID := uuid.New()
	fnID := uuid.New()

	item := osqueue.QueueItem{
		FunctionID: fnID,
		Data: osqueue.Item{
			Payload: json.RawMessage("{\"test\":\"payload\"}"),
			Identifier: state.Identifier{
				AccountID:  accountID,
				WorkflowID: fnID,
			},
		},
	}

	start := clock.Now()

	qi, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, start, osqueue.EnqueueOpts{})
	require.NoError(t, err)

	p := q.ItemPartition(ctx, shard, qi)

	err = q.process(ctx, processItem{
		I:                        qi,
		P:                        p,
		disableConstraintUpdates: false,
		capacityLeaseID:          nil,
	}, func(ctx context.Context, ri osqueue.RunInfo, i osqueue.Item) (osqueue.RunResult, error) {
		return osqueue.RunResult{}, nil
	})
	require.NoError(t, err)
}

func TestQueueProcessorPreLeaseWithConstraintAPI(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	shard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}

	q := NewQueue(
		shard,
		WithClock(clock),
		WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
			return true
		}),
	)
	ctx := context.Background()

	accountID := uuid.New()
	fnID := uuid.New()

	item := osqueue.QueueItem{
		FunctionID: fnID,
		Data: osqueue.Item{
			Payload: json.RawMessage("{\"test\":\"payload\"}"),
			Identifier: state.Identifier{
				AccountID:  accountID,
				WorkflowID: fnID,
			},
		},
	}

	start := clock.Now()

	qi, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, start, osqueue.EnqueueOpts{})
	require.NoError(t, err)

	p := q.ItemPartition(ctx, shard, qi)

	iter := processor{
		partition:            &p,
		items:                []*osqueue.QueueItem{&qi},
		partitionContinueCtr: 0,
		queue:                q,
		denies:               newLeaseDenyList(),
		staticTime:           q.clock.Now(),
		parallel:             false,
	}

	err = iter.process(ctx, &qi)
	require.NoError(t, err)
}
