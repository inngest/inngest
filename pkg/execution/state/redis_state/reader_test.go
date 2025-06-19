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
			num:           500,
			from:          time.Time{},
			until:         clock.Now().Add(time.Minute),
			expectedItems: 500,
		},
		{
			name:          "with interval",
			num:           500,
			from:          time.Time{},
			until:         clock.Now().Add(time.Minute),
			interval:      -1 * time.Second,
			expectedItems: 500,
		},
		{
			name:          "with out of range interval",
			num:           10,
			from:          clock.Now(),
			until:         clock.Now().Add(7 * time.Second).Truncate(time.Second),
			interval:      time.Second,
			expectedItems: 7,
		},
		{
			name:          "with batch size",
			num:           500,
			from:          time.Time{},
			until:         clock.Now().Add(10 * time.Second).Truncate(time.Second),
			interval:      10 * time.Millisecond,
			expectedItems: 500,
			batchSize:     150,
		},
		// With key queues
		{
			name:             "kq - retrieve items in one fetch",
			num:              500,
			from:             clock.Now(),
			until:            clock.Now().Add(time.Minute),
			expectedItems:    500,
			keyQueuesEnabled: true,
		},
		{
			name:             "kq - with interval",
			num:              500,
			from:             time.Time{},
			until:            clock.Now().Add(time.Minute),
			interval:         10 * time.Millisecond,
			expectedItems:    500,
			keyQueuesEnabled: true,
		},
		{
			name:             "kq - with out of range interval",
			num:              10,
			from:             clock.Now(),
			until:            clock.Now().Add(7 * time.Second).Truncate(time.Second),
			interval:         time.Second,
			expectedItems:    7,
			keyQueuesEnabled: true,
		},
		{
			name:             "kq - with batch size",
			num:              500,
			from:             time.Time{},
			until:            clock.Now().Add(10 * time.Second).Truncate(time.Second),
			interval:         -10 * time.Millisecond,
			expectedItems:    500,
			batchSize:        200,
			keyQueuesEnabled: true,
		},
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
				at := clock.Now()
				if !tc.from.IsZero() {
					at = tc.from
				}
				at = at.Add(time.Duration(i) * tc.interval)

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
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()
	defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
	kg := defaultShard.RedisClient.kg

	acctId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	q := NewQueue(
		defaultShard,
		WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
			return true
		}),
		WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
			return false
		}),
		WithClock(clock),
	)

	testcases := []struct {
		name          string
		num           int
		expectedItems int
		interval      time.Duration
		from          time.Time
		until         time.Time
		batchSize     int64
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
		{
			name:          "with batch size",
			num:           10,
			from:          clock.Now(),
			until:         clock.Now().Add(10 * time.Second).Truncate(time.Second),
			interval:      10 * time.Millisecond,
			expectedItems: 10,
			batchSize:     2,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			r.FlushAll()

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

			var backlogID string
			{
				mem, err := r.ZMembers(kg.ShadowPartitionSet(fnID.String()))
				require.NoError(t, err)
				require.Len(t, mem, 1)
				backlogID = mem[0]
			}
			require.NotEmpty(t, backlogID)

			items, err := q.ItemsByBacklog(ctx, defaultShard, backlogID, tc.from, tc.until,
				WithQueueItemIterBatchSize(tc.batchSize),
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
