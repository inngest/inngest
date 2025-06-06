package redis_state

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

func TestItemsByPartition(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()
	defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
	// kg := defaultShard.RedisClient.kg

	acctId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	testcases := []struct {
		name             string
		num              int
		interval         time.Duration
		from             time.Time
		until            time.Time
		expectedItems    int
		keyQueuesEnabled bool
		batchSize        int64
	}{
		{
			name:          "retrieve items in one fetch",
			num:           10,
			from:          clock.Now(),
			until:         clock.Now().Add(time.Minute),
			expectedItems: 10,
		},
		{
			name:          "with interval",
			num:           10,
			from:          clock.Now(),
			until:         clock.Now().Add(time.Minute),
			interval:      time.Second,
			expectedItems: 10,
		},
		{
			name:          "with out of range interval",
			num:           10,
			from:          clock.Now(),
			until:         clock.Now().Add(7 * time.Second).Truncate(time.Second),
			interval:      time.Second,
			expectedItems: 7,
		},
		// {
		// 	name:             "retrieve items backlogs",
		// 	num:              14,
		// 	interval:         2 * time.Second,
		// 	from:             clock.Now(),
		// 	until:            clock.Now().Add(time.Minute),
		// 	expectedItems:    14,
		// 	keyQueuesEnabled: true,
		// },
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			r.FlushAll()

			q := NewQueue(
				defaultShard,
				WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
					return tc.keyQueuesEnabled
				}),
				WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
					return false
				}),
				WithClock(clock),
			)

			for i := range tc.num {
				at := tc.from.Add(time.Duration(i) * tc.interval)

				item := osqueue.QueueItem{
					ID:          fmt.Sprintf("test%d", i),
					FunctionID:  fnID,
					WorkspaceID: wsID,
					Data: osqueue.Item{
						WorkspaceID: wsID,
						Kind:        osqueue.KindEdge,
						Identifier: state.Identifier{
							AccountID:       acctId,
							WorkspaceID:     wsID,
							WorkflowID:      fnID,
							WorkflowVersion: 1,
						},
					},
				}

				_, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
				require.NoError(t, err)
			}

			items, err := q.ItemsByPartition(ctx, defaultShard, fnID, tc.from, tc.until,
				WithQueueItemIterBatchSize(tc.batchSize),
				WithQueueItemIteratorAllowKeyQueues(func() bool {
					return tc.keyQueuesEnabled
				}),
			)
			require.NoError(t, err)

			var count int
			for range items {
				count++
			}

			require.Equal(t, tc.expectedItems, count)
		})
	}
}

func TestItemsByBacklog(t *testing.T) {

}
