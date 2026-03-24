package redis_state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestItemsByPartitionOnEmptyPartition(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()

	t.Run("test empty partition", func(t *testing.T) {
		r.FlushAll()

		q, shard := newQueue(
			t, rc,
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithClock(clock),
		)

		_, err := q.ItemsByPartition(ctx, shard, "i-dont-exist", time.Time{}, clock.Now().Add(time.Minute),
			osqueue.WithQueueItemIterBatchSize(150),
		)
		require.Error(t, err)
		require.True(t, errors.Is(err, rueidis.Nil))
	})
}

func TestItemsByPartition(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()

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

			q, shard := newQueue(
				t, rc,
				osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
					return tc.keyQueuesEnabled
				}),
				osqueue.WithClock(clock),
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

				_, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
				require.NoError(t, err)
			}

			items, err := q.ItemsByPartition(ctx, shard, fnID.String(), tc.from, tc.until,
				osqueue.WithQueueItemIterBatchSize(tc.batchSize),
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

func TestItemsByPartitionWithSystemQueue(t *testing.T) {
	_, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()

	acctID, wsID := uuid.New(), uuid.New()

	q, shard := newQueue(
		t, rc,
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
			return false
		}),
		osqueue.WithClock(clock),
	)

	num := 5000

	systemQueueName := "a-system-queue"

	for i := range num {
		at := clock.Now().Add(time.Duration(i) * time.Millisecond)

		item := osqueue.QueueItem{
			ID:          fmt.Sprintf("test%d", i),
			QueueName:   &systemQueueName,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindEdge,
				QueueName:   &systemQueueName,
				Identifier: state.Identifier{
					AccountID:       acctID,
					WorkspaceID:     wsID,
					WorkflowVersion: 1,
				},
			},
		}

		_, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)
	}

	items, err := q.ItemsByPartition(ctx, shard, systemQueueName, time.Time{}, clock.Now().Add(1*time.Hour),
		osqueue.WithQueueItemIterBatchSize(100),
		osqueue.WithQueueItemIterEnableBacklog(false),
	)
	require.NoError(t, err)

	var count int
	for range items {
		count++
	}

	require.Equal(t, num, count)
}

// leaseQueueItem modifies a queue item in Redis to simulate it being leased.
// It reads the item from the hash, sets LeaseID to a future ULID, and writes it back.
func leaseQueueItem(t *testing.T, rc rueidis.Client, kg QueueKeyGenerator, itemID string, leaseExpiry time.Time) {
	t.Helper()
	ctx := context.Background()

	hashKey := kg.QueueItem()

	// Read the current item
	cmd := rc.B().Hget().Key(hashKey).Field(itemID).Build()
	byt, err := rc.Do(ctx, cmd).AsBytes()
	require.NoError(t, err, "failed to read queue item %s", itemID)

	var qi osqueue.QueueItem
	err = json.Unmarshal(byt, &qi)
	require.NoError(t, err, "failed to unmarshal queue item %s", itemID)

	// Set LeaseID to a ULID with a timestamp in the future
	leaseID, err := ulid.New(ulid.Timestamp(leaseExpiry), rand.Reader)
	require.NoError(t, err)
	qi.LeaseID = &leaseID

	// Write back
	updated, err := json.Marshal(qi)
	require.NoError(t, err)

	setCmd := rc.B().Hset().Key(hashKey).FieldValue().FieldValue(itemID, string(updated)).Build()
	err = rc.Do(ctx, setCmd).Error()
	require.NoError(t, err, "failed to update queue item %s", itemID)
}

func TestItemsByPartitionLeasedItems(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()

	acctId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	enqueueItems := func(t *testing.T, shard RedisQueueShard, n int, prefix string, atFn func(i int) time.Time) []string {
		t.Helper()
		ids := make([]string, 0, n)
		for i := range n {
			item := osqueue.QueueItem{
				ID:          fmt.Sprintf("%s-%d", prefix, i),
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
			enqueued, err := shard.EnqueueItem(ctx, item, atFn(i), osqueue.EnqueueOpts{})
			require.NoError(t, err)
			ids = append(ids, enqueued.ID)
		}
		return ids
	}

	countIter := func(items func(yield func(*osqueue.QueueItem) bool)) int {
		var count int
		for range items {
			count++
		}
		return count
	}

	t.Run("should skip leased items but continue iterating remaining items", func(t *testing.T) {
		r.FlushAll()

		q, shard := newQueue(
			t, rc,
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithClock(clock),
		)
		kg := shard.Client().kg

		// Enqueue 10 items spread across 10ms
		ids := enqueueItems(t, shard, 10, "leased", func(i int) time.Time {
			return clock.Now().Add(time.Duration(i) * time.Millisecond)
		})

		// Lease ALL items
		leaseExpiry := clock.Now().Add(10 * time.Minute)
		for _, id := range ids {
			leaseQueueItem(t, rc, kg, id, leaseExpiry)
		}

		items, err := q.ItemsByPartition(ctx, shard, fnID.String(), time.Time{}, clock.Now().Add(time.Minute),
			osqueue.WithQueueItemIterBatchSize(100),
			osqueue.WithQueueItemIterEnableBacklog(false),
		)
		require.NoError(t, err)

		// All items are leased so none should be yielded, but the iterator should
		// NOT exit early — it should recognize that peek returned data (all leased)
		// and that there may be more items beyond this batch. With all items fitting
		// in a single batch and all leased, returning 0 is acceptable.
		count := countIter(items)
		require.Equal(t, 0, count, "all leased items should be skipped")
	})

	t.Run("should iterate past a fully-leased first batch to reach unleased items", func(t *testing.T) {
		r.FlushAll()

		q, shard := newQueue(
			t, rc,
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithClock(clock),
		)
		kg := shard.Client().kg

		batchSize := int64(10)
		leasedCount := int(batchSize) // first batch entirely leased
		unleasedCount := 20           // items beyond the first batch

		ids := enqueueItems(t, shard, leasedCount+unleasedCount, "mixed", func(i int) time.Time {
			return clock.Now().Add(time.Duration(i) * time.Millisecond)
		})

		// Lease only the first batch worth of items
		leaseExpiry := clock.Now().Add(10 * time.Minute)
		for _, id := range ids[:leasedCount] {
			leaseQueueItem(t, rc, kg, id, leaseExpiry)
		}

		items, err := q.ItemsByPartition(ctx, shard, fnID.String(), time.Time{}, clock.Now().Add(time.Minute),
			osqueue.WithQueueItemIterBatchSize(batchSize),
			osqueue.WithQueueItemIterEnableBacklog(false),
		)
		require.NoError(t, err)

		// The iterator must continue past the fully-leased first batch and
		// return all 20 unleased items from subsequent batches.
		count := countIter(items)
		require.Equal(t, unleasedCount, count,
			"iterator should continue past fully-leased batch and return all unleased items")
	})

	t.Run("items at different milliseconds with leased entries across batch boundary", func(t *testing.T) {
		r.FlushAll()

		q, shard := newQueue(
			t, rc,
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithClock(clock),
		)
		kg := shard.Client().kg

		batchSize := int64(5)

		// Enqueue 10 items at distinct milliseconds
		ids := enqueueItems(t, shard, 10, "diff-ms", func(i int) time.Time {
			return clock.Now().Add(time.Duration(i) * time.Millisecond)
		})

		// Lease items 2 and 3 (in the middle of the first batch)
		leaseExpiry := clock.Now().Add(10 * time.Minute)
		leaseQueueItem(t, rc, kg, ids[2], leaseExpiry)
		leaseQueueItem(t, rc, kg, ids[3], leaseExpiry)

		items, err := q.ItemsByPartition(ctx, shard, fnID.String(), time.Time{}, clock.Now().Add(time.Minute),
			osqueue.WithQueueItemIterBatchSize(batchSize),
			osqueue.WithQueueItemIterEnableBacklog(false),
		)
		require.NoError(t, err)

		// 10 items total, 2 leased → 8 should be returned.
		// The iterator should advance past the first batch and pick up items from
		// the second batch even though some items in the first batch were leased.
		count := countIter(items)
		require.Equal(t, 8, count,
			"all unleased items across batches should be returned")
	})
}

// deleteQueueItemFromHash removes a queue item from the hash map only,
// leaving its entry in the sorted set (simulating a race where the item
// was dequeued between ZRANGEBYSCORE and HMGET).
func deleteQueueItemFromHash(t *testing.T, rc rueidis.Client, kg QueueKeyGenerator, itemID string) {
	t.Helper()
	ctx := context.Background()
	cmd := rc.B().Hdel().Key(kg.QueueItem()).Field(itemID).Build()
	err := rc.Do(ctx, cmd).Error()
	require.NoError(t, err, "failed to delete queue item %s from hash", itemID)
}

func TestItemsByPartitionMissingHashItems(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()

	acctId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	enqueueItems := func(t *testing.T, shard RedisQueueShard, n int, prefix string, atFn func(i int) time.Time) []string {
		t.Helper()
		ids := make([]string, 0, n)
		for i := range n {
			item := osqueue.QueueItem{
				ID:          fmt.Sprintf("%s-%d", prefix, i),
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
			enqueued, err := shard.EnqueueItem(ctx, item, atFn(i), osqueue.EnqueueOpts{})
			require.NoError(t, err)
			ids = append(ids, enqueued.ID)
		}
		return ids
	}

	countIter := func(items func(yield func(*osqueue.QueueItem) bool)) int {
		var count int
		for range items {
			count++
		}
		return count
	}

	t.Run("should iterate past items missing from hash to reach remaining items", func(t *testing.T) {
		r.FlushAll()

		q, shard := newQueue(
			t, rc,
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithClock(clock),
		)
		kg := shard.Client().kg

		batchSize := int64(10)
		deletedCount := int(batchSize)
		remainingCount := 15

		ids := enqueueItems(t, shard, deletedCount+remainingCount, "missing-hash", func(i int) time.Time {
			return clock.Now().Add(time.Duration(i) * time.Millisecond)
		})

		// Delete the first batch of items from the hash only, leaving them in the sorted set.
		// This simulates items being dequeued/completed between the ZRANGEBYSCORE and HMGET
		// calls inside peek's Lua script, or items that were cleaned up externally.
		for _, id := range ids[:deletedCount] {
			deleteQueueItemFromHash(t, rc, kg, id)
		}

		items, err := q.ItemsByPartition(ctx, shard, fnID.String(), time.Time{}, clock.Now().Add(time.Minute),
			osqueue.WithQueueItemIterBatchSize(batchSize),
			osqueue.WithQueueItemIterEnableBacklog(false),
		)
		require.NoError(t, err)

		// The first peek returns `batchSize` items from the sorted set, but all are
		// missing from the hash. peek cleans them up (removes from zset) and returns
		// an empty slice. The iterator must NOT exit early — it should recognize that
		// peek found items (even though they were missing) and continue to the next batch.
		count := countIter(items)
		require.Equal(t, remainingCount, count,
			"iterator should continue past missing-hash items and return all remaining items")
	})

	t.Run("should handle all items missing from hash without panicking", func(t *testing.T) {
		r.FlushAll()

		q, shard := newQueue(
			t, rc,
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithClock(clock),
		)
		kg := shard.Client().kg

		ids := enqueueItems(t, shard, 10, "all-missing", func(i int) time.Time {
			return clock.Now().Add(time.Duration(i) * time.Millisecond)
		})

		// Delete ALL items from the hash
		for _, id := range ids {
			deleteQueueItemFromHash(t, rc, kg, id)
		}

		items, err := q.ItemsByPartition(ctx, shard, fnID.String(), time.Time{}, clock.Now().Add(time.Minute),
			osqueue.WithQueueItemIterBatchSize(100),
			osqueue.WithQueueItemIterEnableBacklog(false),
		)
		require.NoError(t, err)

		// All items are missing from hash. peek cleans them up from the zset and
		// returns nothing. After cleanup, the zset is empty, so the next peek also
		// returns nothing. The iterator should gracefully return 0 items.
		count := countIter(items)
		require.Equal(t, 0, count, "all items missing from hash should result in 0 yielded items")
	})
}

func TestItemsByBacklog(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()

	acctId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	q, shard := newQueue(
		t, rc,
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
			return true
		}),
		osqueue.WithClock(clock),
	)
	kg := shard.Client().kg

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

				_, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
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

			items, err := q.ItemsByBacklog(ctx, shard, backlogID, tc.from, tc.until,
				osqueue.WithQueueItemIterBatchSize(tc.batchSize),
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

func TestQueueIterator(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()

	_, shard := newQueue(
		t, rc,
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
			return false // TODO need to add support for key queues
		}),
		osqueue.WithClock(clock),
	)

	acctId, wsID := uuid.New(), uuid.New()

	testcases := []struct {
		name       string
		partitions int
		items      int
	}{
		{
			name:       "one partition",
			partitions: 1,
			items:      100,
		},
		{
			name:       "multiple partitions",
			partitions: 100,
			items:      500,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			r.FlushAll()

			// construct partition IDs
			partitions := make([]uuid.UUID, tc.partitions)
			for i := range tc.partitions {
				partitions[i] = uuid.New()
			}

			for i := range tc.items {
				size := len(partitions)
				fnID := partitions[i%size]

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

				_, err := shard.EnqueueItem(ctx, item, clock.Now(), osqueue.EnqueueOpts{})
				require.NoError(t, err)
			}

			ptCnt, piCnt, err := shard.QueueIterator(ctx, QueueIteratorOpts{})
			require.NoError(t, err)

			require.EqualValues(t, tc.partitions, ptCnt)
			require.EqualValues(t, tc.items, piCnt)
		})
	}
}

func TestItemByID(t *testing.T) {
	_, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()
	acctId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	q1, shard := newQueue(
		t, rc,
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
			return false
		}),
		osqueue.WithClock(clock),
	)

	enqueue := func(ctx context.Context, shard RedisQueueShard) (osqueue.QueueItem, error) {
		item := osqueue.QueueItem{
			ID:          ulid.MustNew(ulid.Now(), rand.Reader).String(),
			FunctionID:  fnID,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					AccountID:   acctId,
					WorkspaceID: wsID,
					WorkflowID:  fnID,
				},
			},
		}

		return shard.EnqueueItem(ctx, item, clock.Now(), osqueue.EnqueueOpts{})
	}

	t.Run("should be able to find the queue item", func(t *testing.T) {
		enqueued, err := enqueue(ctx, shard)
		require.NoError(t, err)

		res, err := q1.ItemByID(ctx, shard, enqueued.ID)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, enqueued.ID, res.ID)
	})

	t.Run("should return not found error if absent", func(t *testing.T) {
		_, err := enqueue(ctx, shard)
		require.NoError(t, err)

		res, err := q1.ItemByID(ctx, shard, "random")
		require.ErrorIs(t, err, osqueue.ErrQueueItemNotFound)
		require.Nil(t, res)
	})
}

func TestItemExists(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()
	acctId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	q, shard := newQueue(
		t, rc,
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
			return false
		}),
		osqueue.WithClock(clock),
	)

	enqueue := func(ctx context.Context, shard RedisQueueShard, jobID string) (osqueue.QueueItem, error) {
		item := osqueue.QueueItem{
			ID:          jobID,
			FunctionID:  fnID,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					AccountID:   acctId,
					WorkspaceID: wsID,
					WorkflowID:  fnID,
				},
			},
		}

		return shard.EnqueueItem(ctx, item, clock.Now(), osqueue.EnqueueOpts{})
	}

	t.Run("should return true when item exists", func(t *testing.T) {
		r.FlushAll()

		jobID := ulid.MustNew(ulid.Now(), rand.Reader).String()
		enqueued, err := enqueue(ctx, shard, jobID)
		require.NoError(t, err)

		exists, err := q.ItemExists(ctx, shard, enqueued.ID)
		require.NoError(t, err)
		require.True(t, exists, "item should exist")
	})

	t.Run("should return false when item does not exist", func(t *testing.T) {
		r.FlushAll()

		nonExistentJobID := ulid.MustNew(ulid.Now(), rand.Reader).String()

		exists, err := q.ItemExists(ctx, shard, nonExistentJobID)
		require.NoError(t, err)
		require.False(t, exists, "item should not exist")
	})

	t.Run("should return false after item is dequeued", func(t *testing.T) {
		r.FlushAll()

		jobID := ulid.MustNew(ulid.Now(), rand.Reader).String()
		enqueued, err := enqueue(ctx, shard, jobID)
		require.NoError(t, err)

		// Verify it exists
		exists, err := q.ItemExists(ctx, shard, enqueued.ID)
		require.NoError(t, err)
		require.True(t, exists)

		// Dequeue the item
		err = q.Dequeue(ctx, shard, enqueued)
		require.NoError(t, err)

		// Should no longer exist
		exists, err = q.ItemExists(ctx, shard, enqueued.ID)
		require.NoError(t, err)
		require.False(t, exists, "dequeued item should not exist")
	})
}
