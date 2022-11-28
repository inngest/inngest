package redis_state

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestQueueEnqueue(t *testing.T) {
	r := miniredis.RunT(t)
	q := queue{
		r: redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100}),
		pf: func(ctx context.Context, workflowID uuid.UUID) uint {
			return 4
		},
	}
	ctx := context.Background()

	start := time.Now().Truncate(time.Second)

	t.Run("It enqueues an item", func(t *testing.T) {
		item, err := q.Enqueue(ctx, QueueItem{}, start)
		require.NoError(t, err)
		require.NotEqual(t, item.ID, ulid.ULID{})

		// Ensure that our data is set up correctly.
		found := getQueueItem(t, r, item.ID)
		require.Equal(t, item, found)

		// Ensure the partition is inserted.
		val := r.HGet(fmt.Sprintf("partition:item:%s", item.WorkflowID), "item")
		qp := QueuePartition{}
		err = json.Unmarshal([]byte(val), &qp)
		require.NoError(t, err)
		require.Equal(t, QueuePartition{
			QueuePartitionIndex: QueuePartitionIndex{
				WorkflowID: item.WorkflowID,
				Priority:   4,
			},
			Earliest: start,
		}, qp)
	})

	t.Run("It enqueues an item in the future", func(t *testing.T) {
		at := time.Now().Add(time.Hour).Truncate(time.Second)
		item, err := q.Enqueue(ctx, QueueItem{}, at)
		require.NoError(t, err)

		// Ensure the partition is inserted, and the earliest time is still
		// the start time.
		val := r.HGet(fmt.Sprintf("partition:item:%s", item.WorkflowID), "item")
		qp := QueuePartition{}
		err = json.Unmarshal([]byte(val), &qp)
		require.NoError(t, err)
		require.Equal(t, QueuePartition{
			QueuePartitionIndex: QueuePartitionIndex{
				WorkflowID: item.WorkflowID,
				Priority:   4,
			},
			Earliest: start,
		}, qp)

		// Ensure that the zscore did not change.
		keys, err := r.ZMembers("partition:sorted")
		require.NoError(t, err)
		require.Equal(t, 1, len(keys))
		score, err := r.ZScore("partition:sorted", keys[0])
		require.NoError(t, err)
		require.EqualValues(t, start.Unix(), score)
	})

	t.Run("Updates vestimg time to earlier times", func(t *testing.T) {
		at := time.Now().Add(-10 * time.Minute).Truncate(time.Second)
		item, err := q.Enqueue(ctx, QueueItem{}, at)
		require.NoError(t, err)

		// Ensure the partition is inserted, and the earliest time is updated
		// inside the partition item.
		val := r.HGet(fmt.Sprintf("partition:item:%s", item.WorkflowID), "item")
		qp := QueuePartition{}
		err = json.Unmarshal([]byte(val), &qp)
		require.NoError(t, err)
		require.Equal(t, QueuePartition{
			QueuePartitionIndex: QueuePartitionIndex{
				WorkflowID: item.WorkflowID,
				Priority:   4,
			},
			Earliest: at,
		}, qp)

		// Assert that the zscore was changed to this earliest timestamp.
		keys, err := r.ZMembers("partition:sorted")
		require.NoError(t, err)
		require.Equal(t, 1, len(keys))
		score, err := r.ZScore("partition:sorted", keys[0])
		require.NoError(t, err)
		require.EqualValues(t, at.Unix(), score)
	})

	t.Run("Adding another workflow ID increases partition set", func(t *testing.T) {
		at := time.Now().Truncate(time.Second)
		item, err := q.Enqueue(ctx, QueueItem{
			WorkflowID: uuid.New(),
		}, at)
		require.NoError(t, err)

		// Assert that we have two zscores in partition:sorted.
		keys, err := r.ZMembers("partition:sorted")
		require.NoError(t, err)
		require.Equal(t, 2, len(keys))

		// Ensure the partition is inserted, and the earliest time is updated
		// inside the partition item.
		val := r.HGet(fmt.Sprintf("partition:item:%s", item.WorkflowID), "item")
		qp := QueuePartition{}
		err = json.Unmarshal([]byte(val), &qp)
		require.NoError(t, err)
		require.Equal(t, QueuePartition{
			QueuePartitionIndex: QueuePartitionIndex{
				WorkflowID: item.WorkflowID,
				Priority:   4,
			},
			Earliest: at,
		}, qp)
	})
}

func TestQueueLease(t *testing.T) {
	r := miniredis.RunT(t)
	q := queue{
		r: redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100}),
		pf: func(ctx context.Context, workflowID uuid.UUID) uint {
			return 4
		},
	}
	ctx := context.Background()

	start := time.Now().Truncate(time.Second)
	t.Run("It leases an item", func(t *testing.T) {
		item, err := q.Enqueue(ctx, QueueItem{}, start)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		err = q.Lease(ctx, item.WorkflowID, item.ID, time.Second)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.WithinDuration(t, time.Now().Add(time.Second), ulid.Time(item.LeaseID.Time()), 10*time.Millisecond)

		t.Run("Leasing again should fail", func(t *testing.T) {
			for i := 0; i < 50; i++ {
				err := q.Lease(ctx, item.WorkflowID, item.ID, time.Second)
				require.Equal(t, ErrQueueItemAlreadyLeased, err)
				<-time.After(5 * time.Millisecond)
			}
		})

		t.Run("Leasing an expired lease should succeed", func(t *testing.T) {
			<-time.After(1005 * time.Millisecond)
			err := q.Lease(ctx, item.WorkflowID, item.ID, time.Second)
			require.NoError(t, err)

			item = getQueueItem(t, r, item.ID)
			require.NotNil(t, item.LeaseID)
			require.WithinDuration(t, time.Now().Add(time.Second), ulid.Time(item.LeaseID.Time()), 10*time.Millisecond)
		})
	})
}

func getQueueItem(t *testing.T, r *miniredis.Miniredis, id ulid.ULID) QueueItem {
	// Ensure that our data is set up correctly.
	val, err := r.Get(fmt.Sprintf("queue:item:%s", id))
	require.NoError(t, err)
	i := QueueItem{}
	err = json.Unmarshal([]byte(val), &i)
	require.NoError(t, err)
	return i
}
