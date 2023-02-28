package redis_state

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

const testPriority = PriorityDefault

func TestQueueEnqueueItem(t *testing.T) {
	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	defer rc.Close()
	q := NewQueue(rc)
	ctx := context.Background()

	start := time.Now().Truncate(time.Second)

	t.Run("It enqueues an item", func(t *testing.T) {
		item, err := q.EnqueueItem(ctx, QueueItem{}, start)
		require.NoError(t, err)
		require.NotEqual(t, item.ID, ulid.ULID{})

		// Ensure that our data is set up correctly.
		found := getQueueItem(t, r, item.ID)
		require.Equal(t, item, found)

		// Ensure the partition is inserted.
		qp := getPartition(t, r, item.WorkflowID)
		require.Equal(t, QueuePartition{
			WorkflowID: item.WorkflowID,
			Priority:   testPriority,
			AtS:        start.Unix(),
		}, qp)
	})

	t.Run("It sets the right item score", func(t *testing.T) {
		start := time.Now()
		item, err := q.EnqueueItem(ctx, QueueItem{}, start)
		require.NoError(t, err)

		requireItemScoreEquals(t, r, item, start)
	})

	t.Run("It enqueues an item in the future", func(t *testing.T) {
		at := time.Now().Add(time.Hour).Truncate(time.Second)
		item, err := q.EnqueueItem(ctx, QueueItem{}, at)
		require.NoError(t, err)

		// Ensure the partition is inserted, and the earliest time is still
		// the start time.
		qp := getPartition(t, r, item.WorkflowID)
		require.Equal(t, QueuePartition{
			WorkflowID: item.WorkflowID,
			Priority:   testPriority,
			AtS:        start.Unix(),
		}, qp)

		// Ensure that the zscore did not change.
		keys, err := r.ZMembers(defaultQueueKey.PartitionIndex())
		require.NoError(t, err)
		require.Equal(t, 1, len(keys))
		score, err := r.ZScore(defaultQueueKey.PartitionIndex(), keys[0])
		require.NoError(t, err)
		require.EqualValues(t, start.Unix(), score)
	})

	t.Run("Updates partition vesting time to earlier times", func(t *testing.T) {
		at := time.Now().Add(-10 * time.Minute).Truncate(time.Second)
		item, err := q.EnqueueItem(ctx, QueueItem{}, at)
		require.NoError(t, err)

		// Ensure the partition is inserted, and the earliest time is updated
		// inside the partition item.
		qp := getPartition(t, r, item.WorkflowID)
		require.Equal(t, QueuePartition{
			WorkflowID: item.WorkflowID,
			Priority:   testPriority,
			AtS:        at.Unix(),
		}, qp)

		// Assert that the zscore was changed to this earliest timestamp.
		keys, err := r.ZMembers(defaultQueueKey.PartitionIndex())
		require.NoError(t, err)
		require.Equal(t, 1, len(keys))
		score, err := r.ZScore(defaultQueueKey.PartitionIndex(), keys[0])
		require.NoError(t, err)
		require.EqualValues(t, at.Unix(), score)
	})

	t.Run("Adding another workflow ID increases partition set", func(t *testing.T) {
		at := time.Now().Truncate(time.Second)
		item, err := q.EnqueueItem(ctx, QueueItem{
			WorkflowID: uuid.New(),
		}, at)
		require.NoError(t, err)

		// Assert that we have two zscores in partition:sorted.
		keys, err := r.ZMembers(defaultQueueKey.PartitionIndex())
		require.NoError(t, err)
		require.Equal(t, 2, len(keys))

		// Ensure the partition is inserted, and the earliest time is updated
		// inside the partition item.
		qp := getPartition(t, r, item.WorkflowID)
		require.Equal(t, QueuePartition{
			WorkflowID: item.WorkflowID,
			Priority:   testPriority,
			AtS:        at.Unix(),
		}, qp)
	})
}

func TestQueueEnqueueItemIdempotency(t *testing.T) {
	dur := 2 * time.Second

	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	defer rc.Close()
	// Set idempotency to a second
	q := NewQueue(rc, WithIdempotencyTTL(dur))
	ctx := context.Background()

	start := time.Now().Truncate(time.Second)

	t.Run("It enqueues an item only once", func(t *testing.T) {
		i := QueueItem{ID: "once"}

		item, err := q.EnqueueItem(ctx, i, start)
		p := QueuePartition{WorkflowID: item.WorkflowID}

		require.NoError(t, err)
		require.Equal(t, hashID(ctx, "once"), item.ID)
		require.NotEqual(t, i.ID, item.ID)
		found := getQueueItem(t, r, item.ID)
		require.Equal(t, item, found)

		// Ensure we can't enqueue again.
		_, err = q.EnqueueItem(ctx, i, start)
		require.Equal(t, ErrQueueItemExists, err)

		// Dequeue
		err = q.Dequeue(ctx, p, item)
		require.NoError(t, err)

		// Ensure we can't enqueue even after dequeue.
		_, err = q.EnqueueItem(ctx, i, start)
		require.Equal(t, ErrQueueItemExists, err)

		// Wait for the idempotency TTL to expire
		r.FastForward(dur)

		item, err = q.EnqueueItem(ctx, i, start)
		require.NoError(t, err)
		require.Equal(t, hashID(ctx, "once"), item.ID)
		require.NotEqual(t, i.ID, item.ID)
		found = getQueueItem(t, r, item.ID)
		require.Equal(t, item, found)
	})
}

func TestQueuePeek(t *testing.T) {
	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	defer rc.Close()
	q := NewQueue(rc)
	ctx := context.Background()

	// The default blank UUID
	workflowID := uuid.UUID{}

	t.Run("It returns none with no items enqueued", func(t *testing.T) {
		items, err := q.Peek(ctx, workflowID.String(), time.Now().Add(time.Hour), 10)
		require.NoError(t, err)
		require.EqualValues(t, 0, len(items))
	})

	t.Run("It returns an ordered list of items", func(t *testing.T) {
		a := time.Now().Truncate(time.Second)
		b := a.Add(2 * time.Second)
		c := b.Add(2 * time.Second)
		d := c.Add(2 * time.Second)

		ia, err := q.EnqueueItem(ctx, QueueItem{}, a)
		require.NoError(t, err)
		ib, err := q.EnqueueItem(ctx, QueueItem{}, b)
		require.NoError(t, err)
		ic, err := q.EnqueueItem(ctx, QueueItem{}, c)
		require.NoError(t, err)

		items, err := q.Peek(ctx, workflowID.String(), time.Now().Add(time.Hour), 10)
		require.NoError(t, err)
		require.EqualValues(t, 3, len(items))
		require.EqualValues(t, []*QueueItem{&ia, &ib, &ic}, items)
		require.NotEqualValues(t, []*QueueItem{&ib, &ia, &ic}, items)

		id, err := q.EnqueueItem(ctx, QueueItem{}, d)
		require.NoError(t, err)

		items, err = q.Peek(ctx, workflowID.String(), time.Now().Add(time.Hour), 10)
		require.NoError(t, err)
		require.EqualValues(t, 4, len(items))
		require.EqualValues(t, []*QueueItem{&ia, &ib, &ic, &id}, items)

		t.Run("It should limit the list", func(t *testing.T) {
			items, err = q.Peek(ctx, workflowID.String(), time.Now().Add(time.Hour), 2)
			require.NoError(t, err)
			require.EqualValues(t, 2, len(items))
			require.EqualValues(t, []*QueueItem{&ia, &ib}, items)
		})

		t.Run("It should apply a peek offset", func(t *testing.T) {
			items, err = q.Peek(ctx, workflowID.String(), time.Now().Add(-1*time.Hour), QueuePeekMax)
			require.NoError(t, err)
			require.EqualValues(t, 0, len(items))

			items, err = q.Peek(ctx, workflowID.String(), c, QueuePeekMax)
			require.NoError(t, err)
			require.EqualValues(t, 3, len(items))
			require.EqualValues(t, []*QueueItem{&ia, &ib, &ic}, items)
		})

		t.Run("It should remove any leased items from the list", func(t *testing.T) {
			p := QueuePartition{WorkflowID: ia.WorkflowID}

			// Lease step A, and it should be removed.
			_, err := q.Lease(ctx, p, ia, 50*time.Millisecond)
			require.NoError(t, err)

			items, err = q.Peek(ctx, workflowID.String(), d, QueuePeekMax)
			require.NoError(t, err)
			require.EqualValues(t, 3, len(items))
			require.EqualValues(t, []*QueueItem{&ib, &ic, &id}, items)
		})

		t.Run("Expired leases should move back via scavenging", func(t *testing.T) {
			// Run scavenging.
			caught, err := q.Scavenge(ctx)
			require.NoError(t, err)
			require.EqualValues(t, 0, caught)

			// When the lease expires it should re-appear
			<-time.After(55 * time.Millisecond)

			// Run scavenging.
			scavengeAt := time.Now().UnixMilli()
			caught, err = q.Scavenge(ctx)
			require.NoError(t, err)
			require.EqualValues(t, 1, caught)

			items, err = q.Peek(ctx, workflowID.String(), d, QueuePeekMax)
			require.NoError(t, err)
			require.EqualValues(t, 4, len(items))

			require.EqualValues(t, ia.ID, items[0].ID)
			// NOTE: Scavenging requeues items, and so the time will have changed.
			require.GreaterOrEqual(t, items[0].AtMS, scavengeAt)
			require.Greater(t, items[0].AtMS, ia.AtMS)
			ia.LeaseID = nil
			ia.AtMS = items[0].AtMS
			require.EqualValues(t, []*QueueItem{&ia, &ib, &ic, &id}, items)
		})
	})

}

func TestQueueLease(t *testing.T) {
	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	defer rc.Close()
	q := NewQueue(rc)
	ctx := context.Background()

	start := time.Now().Truncate(time.Second)
	t.Run("It leases an item", func(t *testing.T) {
		item, err := q.EnqueueItem(ctx, QueueItem{}, start)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		p := QueuePartition{} // Default workflow ID etc

		require.Equal(t, item.Queue(), item.WorkflowID.String())

		now := time.Now()
		id, err := q.Lease(ctx, p, item, time.Second)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, id, item.LeaseID)
		require.WithinDuration(t, now.Add(time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		t.Run("It should add the item to the partition queue", func(t *testing.T) {
			key, _ := q.partitionConcurrencyGen(ctx, p)
			require.EqualValues(t, uuid.UUID{}.String(), key)
			count, err := q.InProgress(ctx, "p", key)
			require.NoError(t, err)
			require.EqualValues(t, 1, count)
		})

		t.Run("Leasing again should fail", func(t *testing.T) {
			for i := 0; i < 50; i++ {
				id, err := q.Lease(ctx, p, item, time.Second)
				require.Equal(t, ErrQueueItemAlreadyLeased, err)
				require.Nil(t, id)
				<-time.After(5 * time.Millisecond)
			}
		})

		t.Run("Leasing an expired lease should succeed", func(t *testing.T) {
			<-time.After(1005 * time.Millisecond)

			// Now expired
			t.Run("After expiry, no items should be in progress", func(t *testing.T) {
				key, _ := q.partitionConcurrencyGen(ctx, p)
				count, err := q.InProgress(ctx, "p", key)
				require.NoError(t, err)
				require.EqualValues(t, 0, count)
			})

			now := time.Now()
			id, err := q.Lease(ctx, p, item, 5*time.Second)
			require.NoError(t, err)
			require.NoError(t, err)

			item = getQueueItem(t, r, item.ID)
			require.NotNil(t, item.LeaseID)
			require.EqualValues(t, id, item.LeaseID)
			require.WithinDuration(t, now.Add(5*time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

			t.Run("Leasing an expired key has one in-progress", func(t *testing.T) {
				key, _ := q.partitionConcurrencyGen(ctx, p)
				count, err := q.InProgress(ctx, "p", key)
				require.NoError(t, err)
				require.EqualValues(t, 1, count)
			})
		})

		t.Run("It should remove the item from the function queue, as this is now in the partition's in-progress concurrency queue", func(t *testing.T) {
			start := time.Now()
			item, err := q.EnqueueItem(ctx, QueueItem{}, start)
			require.NoError(t, err)
			require.Nil(t, item.LeaseID)

			requireItemScoreEquals(t, r, item, start)

			_, err = q.Lease(ctx, p, item, time.Minute)
			require.NoError(t, err)

			_, err = r.ZScore(defaultQueueKey.QueueIndex(item.WorkflowID.String()), item.ID)
			require.Error(t, err, "no such key")
		})
	})

	t.Run("With partition concurrency limits", func(t *testing.T) {
		// Only allow a single leased item
		q.partitionConcurrencyGen = func(ctx context.Context, p QueuePartition) (string, int) {
			return p.Queue(), 1
		}
		q.accountConcurrencyGen = nil

		// Create a new item
		itemA, err := q.EnqueueItem(ctx, QueueItem{WorkflowID: uuid.New()}, start)
		require.NoError(t, err)
		itemB, err := q.EnqueueItem(ctx, QueueItem{WorkflowID: uuid.New()}, start)
		require.NoError(t, err)
		// Use the new item's workflow ID
		p := QueuePartition{WorkflowID: itemA.WorkflowID}

		t.Run("Leases with capacity", func(t *testing.T) {
			_, err = q.Lease(ctx, p, itemA, 5*time.Second)
			require.NoError(t, err)
		})
		t.Run("Errors without capacity", func(t *testing.T) {
			id, err := q.Lease(ctx, p, itemB, 5*time.Second)
			require.Nil(t, id)
			require.Error(t, err)
		})
	})

	t.Run("With account concurrency limits", func(t *testing.T) {
		// Only allow a single leased item
		q.partitionConcurrencyGen = func(ctx context.Context, p QueuePartition) (string, int) {
			return p.Queue(), 100
		}
		q.customConcurrencyGen = nil
		q.accountConcurrencyGen = func(ctx context.Context, i QueueItem) (string, int) {
			return "account-level-key", 1
		}

		// Create a new item
		itemA, err := q.EnqueueItem(ctx, QueueItem{WorkflowID: uuid.New()}, start)
		require.NoError(t, err)
		itemB, err := q.EnqueueItem(ctx, QueueItem{WorkflowID: uuid.New()}, start)
		require.NoError(t, err)
		// Use the new item's workflow ID
		p := QueuePartition{WorkflowID: itemA.WorkflowID}

		t.Run("Leases with capacity", func(t *testing.T) {
			_, err = q.Lease(ctx, p, itemA, 5*time.Second)
			require.NoError(t, err)
		})
		t.Run("Errors without capacity", func(t *testing.T) {
			id, err := q.Lease(ctx, p, itemB, 5*time.Second)
			require.Nil(t, id)
			require.Error(t, err)
		})
	})

	t.Run("With account concurrency limits", func(t *testing.T) {
		// Only allow a single leased item
		q.partitionConcurrencyGen = func(ctx context.Context, p QueuePartition) (string, int) {
			return p.Queue(), 100
		}
		q.accountConcurrencyGen = nil
		q.customConcurrencyGen = func(ctx context.Context, i QueueItem) (string, int) {
			return "custom-level-key", 1
		}

		// Create a new item
		itemA, err := q.EnqueueItem(ctx, QueueItem{WorkflowID: uuid.New()}, start)
		require.NoError(t, err)
		itemB, err := q.EnqueueItem(ctx, QueueItem{WorkflowID: uuid.New()}, start)
		require.NoError(t, err)
		// Use the new item's workflow ID
		p := QueuePartition{WorkflowID: itemA.WorkflowID}

		t.Run("Leases with capacity", func(t *testing.T) {
			_, err = q.Lease(ctx, p, itemA, 5*time.Second)
			require.NoError(t, err)
		})
		t.Run("Errors without capacity", func(t *testing.T) {
			id, err := q.Lease(ctx, p, itemB, 5*time.Second)
			require.Nil(t, id)
			require.Error(t, err)
		})
	})
}

func TestQueueExtendLease(t *testing.T) {
	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	defer rc.Close()
	q := NewQueue(rc)
	ctx := context.Background()

	start := time.Now().Truncate(time.Second)
	t.Run("It leases an item", func(t *testing.T) {
		item, err := q.EnqueueItem(ctx, QueueItem{}, start)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		p := QueuePartition{WorkflowID: item.WorkflowID}

		now := time.Now()
		id, err := q.Lease(ctx, p, item, time.Second)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, id, item.LeaseID)
		require.WithinDuration(t, now.Add(time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		now = time.Now()
		nextID, err := q.ExtendLease(ctx, p, item, *id, 10*time.Second)
		require.NoError(t, err)

		// Ensure the leased item has the next ID.
		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, nextID, item.LeaseID)
		require.WithinDuration(t, now.Add(10*time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		t.Run("It extends the score of the partition concurrency queue", func(t *testing.T) {
			at := ulid.Time(nextID.Time())
			pkey, _ := q.partitionConcurrencyGen(ctx, p)
			scores := concurrencyQueueScores(t, r, q.kg.Concurrency("p", pkey), time.Now())
			require.Len(t, scores, 1)
			// Ensure that the score matches the lease.
			require.Equal(t, at, scores[item.ID])
		})

		t.Run("It fails with an invalid lease ID", func(t *testing.T) {
			invalid := ulid.MustNew(ulid.Now(), rnd)
			nextID, err := q.ExtendLease(ctx, p, item, invalid, 10*time.Second)
			require.EqualValues(t, ErrQueueItemLeaseMismatch, err)
			require.Nil(t, nextID)
		})
	})

	t.Run("It does not extend an unleased item", func(t *testing.T) {
		item, err := q.EnqueueItem(ctx, QueueItem{}, start)
		require.NoError(t, err)

		p := QueuePartition{WorkflowID: item.WorkflowID}

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		nextID, err := q.ExtendLease(ctx, p, item, ulid.ULID{}, 10*time.Second)
		require.EqualValues(t, ErrQueueItemNotLeased, err)
		require.Nil(t, nextID)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)
	})

}

func TestQueueDequeue(t *testing.T) {
	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	defer rc.Close()
	q := NewQueue(rc)
	ctx := context.Background()

	t.Run("It should remove a queue item", func(t *testing.T) {
		start := time.Now()

		item, err := q.EnqueueItem(ctx, QueueItem{}, start)
		require.NoError(t, err)

		p := QueuePartition{WorkflowID: item.WorkflowID}

		id, err := q.Lease(ctx, p, item, time.Second)
		require.NoError(t, err)

		t.Run("The lease exists in the partition queue", func(t *testing.T) {
			key, _ := q.partitionConcurrencyGen(ctx, p)
			require.EqualValues(t, uuid.UUID{}.String(), key)
			count, err := q.InProgress(ctx, "p", key)
			require.NoError(t, err)
			require.EqualValues(t, 1, count)
		})

		err = q.Dequeue(ctx, p, item)
		require.NoError(t, err)

		t.Run("It should remove the item from the queue map", func(t *testing.T) {
			val := r.HGet(defaultQueueKey.QueueItem(), id.String())
			require.Empty(t, val)
		})

		t.Run("Extending a lease should fail after dequeue", func(t *testing.T) {
			id, err := q.ExtendLease(ctx, p, item, *id, time.Minute)
			require.Equal(t, ErrQueueItemNotFound, err)
			require.Nil(t, id)
		})

		t.Run("It should remove the item from the queue index", func(t *testing.T) {
			items, err := q.Peek(ctx, item.Queue(), time.Now().Add(time.Hour), 10)
			require.NoError(t, err)
			require.EqualValues(t, 0, len(items))
		})

		t.Run("It should remove the item from the concurrency partition's queue", func(t *testing.T) {
			key, _ := q.partitionConcurrencyGen(ctx, p)
			require.EqualValues(t, uuid.UUID{}.String(), key)
			count, err := q.InProgress(ctx, "p", key)
			require.NoError(t, err)
			require.EqualValues(t, 0, count)
		})

		t.Run("It should work if the item is not leased (eg. deletions)", func(t *testing.T) {
			item, err := q.EnqueueItem(ctx, QueueItem{}, start)
			require.NoError(t, err)

			err = q.Dequeue(ctx, p, item)
			require.NoError(t, err)

			val := r.HGet(defaultQueueKey.QueueItem(), id.String())
			require.Empty(t, val)
		})
	})
}

func TestQueueRequeue(t *testing.T) {
	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	defer rc.Close()
	q := NewQueue(rc)
	ctx := context.Background()

	t.Run("Re-enqueuing a leased item should succeed", func(t *testing.T) {
		now := time.Now()

		item, err := q.EnqueueItem(ctx, QueueItem{}, now)
		require.NoError(t, err)

		p := QueuePartition{WorkflowID: item.WorkflowID}

		_, err = q.Lease(ctx, p, item, time.Second)
		require.NoError(t, err)

		// Assert partition index is original
		pi := QueuePartition{WorkflowID: item.WorkflowID, Priority: testPriority}
		requirePartitionScoreEquals(t, r, pi.WorkflowID, now.Truncate(time.Second))

		requirePartitionInProgress(t, q, item.WorkflowID, 1)

		next := now.Add(time.Hour)
		err = q.Requeue(ctx, p, item, next)
		require.NoError(t, err)

		t.Run("It should re-enqueue the item with the future time", func(t *testing.T) {
			requireItemScoreEquals(t, r, item, next)
		})

		t.Run("It should always remove the lease from the re-enqueued item", func(t *testing.T) {
			fetched := getQueueItem(t, r, item.ID)
			require.Nil(t, fetched.LeaseID)
		})

		t.Run("It should decrease the in-progress count", func(t *testing.T) {
			requirePartitionInProgress(t, q, item.WorkflowID, 0)
		})

		t.Run("It should update the partition's earliest time, if earliest", func(t *testing.T) {
			// Assert partition index is updated, as there's only one item here.
			requirePartitionScoreEquals(t, r, pi.WorkflowID, next)
		})

		t.Run("It should not update the partition's earliest time, if later", func(t *testing.T) {
			_, err := q.EnqueueItem(ctx, QueueItem{}, now)
			require.NoError(t, err)

			requirePartitionScoreEquals(t, r, pi.WorkflowID, now)

			next := now.Add(2 * time.Hour)
			err = q.Requeue(ctx, pi, item, next)
			require.NoError(t, err)

			requirePartitionScoreEquals(t, r, pi.WorkflowID, now)
		})
	})
}

func TestQueuePartitionLease(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	idA, idB, idC := uuid.New(), uuid.New(), uuid.New()
	atA, atB, atC := now, now.Add(time.Second), now.Add(2*time.Second)

	pA := QueuePartition{WorkflowID: idA}

	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	defer rc.Close()
	q := NewQueue(rc)
	ctx := context.Background()

	_, err := q.EnqueueItem(ctx, QueueItem{WorkflowID: idA}, atA)
	require.NoError(t, err)
	_, err = q.EnqueueItem(ctx, QueueItem{WorkflowID: idB}, atB)
	require.NoError(t, err)
	_, err = q.EnqueueItem(ctx, QueueItem{WorkflowID: idC}, atC)
	require.NoError(t, err)

	t.Run("Partitions are in order after enqueueing", func(t *testing.T) {
		items, err := q.PartitionPeek(ctx, true, time.Now().Add(time.Hour), PartitionPeekMax)
		require.NoError(t, err)
		require.Len(t, items, 3)
		require.EqualValues(t, []*QueuePartition{
			{WorkflowID: idA, Priority: testPriority, AtS: atA.Unix()},
			{WorkflowID: idB, Priority: testPriority, AtS: atB.Unix()},
			{WorkflowID: idC, Priority: testPriority, AtS: atC.Unix()},
		}, items)
	})

	leaseUntil := now.Add(3 * time.Second)

	t.Run("It leases a partition", func(t *testing.T) {
		// Lease the first item
		leaseID, _, err := q.PartitionLease(ctx, pA, time.Until(leaseUntil))
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		t.Run("It updates the partition score", func(t *testing.T) {
			items, err := q.PartitionPeek(ctx, true, now.Add(time.Hour), PartitionPeekMax)

			// Require the lease ID is within 25 MS of the expected value.
			require.WithinDuration(t, leaseUntil, ulid.Time(leaseID.Time()), 25*time.Millisecond)

			require.NoError(t, err)
			require.Len(t, items, 3)
			require.EqualValues(t, []*QueuePartition{
				{WorkflowID: idB, Priority: testPriority, AtS: atB.Unix()},
				{WorkflowID: idC, Priority: testPriority, AtS: atC.Unix()},
				{
					WorkflowID: idA,
					Priority:   testPriority,
					AtS:        ulid.Time(leaseID.Time()).Unix(),
					Last:       time.Now().Unix(),
					LeaseID:    leaseID,
				}, // idA is now last.
			}, items)
			requirePartitionScoreEquals(t, r, idA, leaseUntil)
		})

		t.Run("It can't lease an existing partition lease", func(t *testing.T) {
			id, _, err := q.PartitionLease(ctx, pA, time.Second*29)
			require.Equal(t, ErrPartitionAlreadyLeased, err)
			require.Nil(t, id)

			// Assert that score didn't change (we added 1 second in the previous test)
			requirePartitionScoreEquals(t, r, idA, leaseUntil)
		})

	})

	t.Run("It allows leasing an expired partition lease", func(t *testing.T) {
		<-time.After(time.Until(leaseUntil))

		requirePartitionScoreEquals(t, r, idA, leaseUntil)

		id, _, err := q.PartitionLease(ctx, pA, time.Second*5)
		require.Nil(t, err)
		require.NotNil(t, id)

		requirePartitionScoreEquals(t, r, idA, time.Now().Add(time.Second*5))
	})

	// TODO: Capacity checks
}

func TestQueuePartitionPeek(t *testing.T) {
	idA := uuid.New() // low pri
	idB := uuid.New()
	idC := uuid.New()

	newQueueItem := func(id uuid.UUID) QueueItem {
		return QueueItem{
			WorkflowID: id,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					WorkflowID: id,
				},
			},
		}
	}

	now := time.Now().Truncate(time.Second).UTC()
	atA, atB, atC := now, now.Add(time.Second), now.Add(2*time.Second)

	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	defer rc.Close()
	q := NewQueue(
		rc,
		WithPriorityFinder(func(ctx context.Context, qi QueueItem) uint {
			switch qi.Data.Identifier.WorkflowID {
			case idB, idC:
				return PriorityMax
			default:
				return PriorityMin // Sorry A
			}
		}),
	)
	ctx := context.Background()

	enqueue := func(q *queue) {
		_, err := q.EnqueueItem(ctx, newQueueItem(idA), atA)
		require.NoError(t, err)
		_, err = q.EnqueueItem(ctx, newQueueItem(idB), atB)
		require.NoError(t, err)
		_, err = q.EnqueueItem(ctx, newQueueItem(idC), atC)
		require.NoError(t, err)
	}
	enqueue(q)

	t.Run("Sequentially returns indexes in order", func(t *testing.T) {
		items, err := q.PartitionPeek(ctx, true, time.Now().Add(time.Hour), PartitionPeekMax)
		require.NoError(t, err)
		require.Len(t, items, 3)
		require.EqualValues(t, []*QueuePartition{
			{WorkflowID: idA, Priority: PriorityMin, AtS: atA.Unix()},
			{WorkflowID: idB, Priority: PriorityMax, AtS: atB.Unix()},
			{WorkflowID: idC, Priority: PriorityMax, AtS: atC.Unix()},
		}, items)
	})

	t.Run("Random returns items randomly using weighted sample", func(t *testing.T) {
		a, b, c := 0, 0, 0
		for i := 0; i <= 1000; i++ {
			items, err := q.PartitionPeek(ctx, false, time.Now().Add(time.Hour), PartitionPeekMax)
			require.NoError(t, err)
			require.Len(t, items, 3)
			switch items[0].WorkflowID {
			case idA:
				a++
			case idB:
				b++
			case idC:
				c++
			default:
				t.Fatal()
			}
		}
		// Statistically this is going to fail at some point, but we want to ensure randomness
		// will return low priority items less.
		require.GreaterOrEqual(t, a, 1) // A may be called low-digit times.
		require.Less(t, a, 250)         // But less than 1/4 (it's 1 in 10, statistically)
		require.Greater(t, c, 300)
		require.Greater(t, b, 300)
	})

	t.Run("It ignores partitions with denylists", func(t *testing.T) {
		r := miniredis.RunT(t)
		rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
		defer rc.Close()
		q := NewQueue(
			rc,
			WithPriorityFinder(func(ctx context.Context, qi QueueItem) uint {
				switch qi.Data.Identifier.WorkflowID {
				case idA:
					// A is max priority, so likely to come up.
					return PriorityMax
				default:
					return PriorityMin
				}
			}),
			// Ignore A
			WithDenyQueueNames(idA.String()),
		)

		enqueue(q)

		// This should only select B and C, as id A is ignored.
		items, err := q.PartitionPeek(ctx, true, time.Now().Add(time.Hour), PartitionPeekMax)
		require.NoError(t, err)
		require.Len(t, items, 2)
		require.EqualValues(t, []*QueuePartition{
			{WorkflowID: idB, Priority: PriorityMin, AtS: atB.Unix()},
			{WorkflowID: idC, Priority: PriorityMin, AtS: atC.Unix()},
		}, items)

		// Try without sequential scans
		items, err = q.PartitionPeek(ctx, false, time.Now().Add(time.Hour), PartitionPeekMax)
		require.NoError(t, err)
		require.Len(t, items, 2)
	})
}

func TestQueuePartitionRequeue(t *testing.T) {
	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	defer rc.Close()
	q := NewQueue(rc)
	ctx := context.Background()
	idA := uuid.New()
	now := time.Now()

	qi, err := q.EnqueueItem(ctx, QueueItem{WorkflowID: idA}, now)
	require.NoError(t, err)

	p := QueuePartition{WorkflowID: qi.WorkflowID}

	t.Run("Uses the next job item's time when requeueing with another job", func(t *testing.T) {
		requirePartitionScoreEquals(t, r, idA, now)
		next := now.Add(time.Hour)
		err := q.PartitionRequeue(ctx, idA.String(), next, false)
		require.NoError(t, err)
		requirePartitionScoreEquals(t, r, idA, now)
	})

	next := now.Add(5 * time.Second)
	t.Run("It removes any lease when requeueing", func(t *testing.T) {

		_, _, err := q.PartitionLease(ctx, QueuePartition{WorkflowID: idA}, time.Minute)
		require.NoError(t, err)

		err = q.PartitionRequeue(ctx, idA.String(), next, true)
		require.NoError(t, err)
		requirePartitionScoreEquals(t, r, idA, next)

		loaded := getPartition(t, r, idA)
		require.Nil(t, loaded.LeaseID)
	})

	t.Run("Deletes the partition with an empty queue and a leased job", func(t *testing.T) {
		p := QueuePartition{WorkflowID: qi.WorkflowID}

		requirePartitionScoreEquals(t, r, idA, next)

		// Leasing the only job available moves the job into the concurrency queue,
		// so the partition should be empty. when requeeing.
		_, err := q.Lease(ctx, p, qi, 10*time.Second)
		require.NoError(t, err)

		requirePartitionScoreEquals(t, r, idA, next)

		next := now.Add(time.Hour)
		err = q.PartitionRequeue(ctx, idA.String(), next, false)
		require.Error(t, ErrPartitionGarbageCollected, err)
	})

	t.Run("It returns a partition not found error if deleted", func(t *testing.T) {
		err := q.Dequeue(ctx, p, qi)
		require.NoError(t, err)
		err = q.PartitionRequeue(ctx, idA.String(), time.Now().Add(time.Minute), false)
		require.Equal(t, ErrPartitionNotFound, err)
	})
}

func TestQueuePartitionReprioritize(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	idA := uuid.New()

	priority := PriorityMin
	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	defer rc.Close()
	q := NewQueue(
		rc,
		WithPriorityFinder(func(ctx context.Context, item QueueItem) uint {
			return priority
		}),
	)
	ctx := context.Background()

	_, err := q.EnqueueItem(ctx, QueueItem{WorkflowID: idA}, now)
	require.NoError(t, err)

	first := getPartition(t, r, idA)
	require.Equal(t, first.Priority, PriorityMin)

	t.Run("It updates priority", func(t *testing.T) {
		priority = PriorityMax
		err = q.PartitionReprioritize(ctx, idA.String(), PriorityMax)
		require.NoError(t, err)
		second := getPartition(t, r, idA)
		require.Equal(t, second.Priority, PriorityMax)
	})

	t.Run("It doesn't accept min priorities", func(t *testing.T) {
		err = q.PartitionReprioritize(ctx, idA.String(), PriorityMin+1)
		require.Equal(t, ErrPriorityTooLow, err)
	})
}

func TestQueueLeaseSequential(t *testing.T) {
	ctx := context.Background()
	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	q := queue{
		kg: defaultQueueKey,
		r:  rc,
		pf: func(ctx context.Context, item QueueItem) uint {
			return PriorityMin
		},
	}

	var (
		leaseID *ulid.ULID
		err     error
	)

	t.Run("It claims sequential leases", func(t *testing.T) {
		now := time.Now()
		dur := 500 * time.Millisecond
		leaseID, err = q.ConfigLease(ctx, q.kg.Sequential(), dur)
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		require.WithinDuration(t, now.Add(dur), ulid.Time(leaseID.Time()), 5*time.Millisecond)
	})

	t.Run("It doesn't allow leasing without an existing lease ID", func(t *testing.T) {
		id, err := q.ConfigLease(ctx, q.kg.Sequential(), time.Second)
		require.Equal(t, ErrConfigAlreadyLeased, err)
		require.Nil(t, id)
	})

	t.Run("It doesn't allow leasing with an invalid lease ID", func(t *testing.T) {
		newULID := ulid.MustNew(ulid.Now(), rnd)
		id, err := q.ConfigLease(ctx, q.kg.Sequential(), time.Second, &newULID)
		require.Equal(t, ErrConfigAlreadyLeased, err)
		require.Nil(t, id)
	})

	t.Run("It extends the lease with a valid lease ID", func(t *testing.T) {
		require.NotNil(t, leaseID)

		now := time.Now()
		dur := 50 * time.Millisecond
		leaseID, err = q.ConfigLease(ctx, q.kg.Sequential(), dur, leaseID)
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		require.WithinDuration(t, now.Add(dur), ulid.Time(leaseID.Time()), 5*time.Millisecond)
	})

	t.Run("It allows leasing when the current lease is expired", func(t *testing.T) {
		<-time.After(100 * time.Millisecond)

		now := time.Now()
		dur := 50 * time.Millisecond
		leaseID, err = q.ConfigLease(ctx, q.kg.Sequential(), dur)
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		require.WithinDuration(t, now.Add(dur), ulid.Time(leaseID.Time()), 5*time.Millisecond)
	})
}

func getQueueItem(t *testing.T, r *miniredis.Miniredis, id string) QueueItem {
	t.Helper()
	// Ensure that our data is set up correctly.
	val := r.HGet(defaultQueueKey.QueueItem(), id)
	require.NotEmpty(t, val)
	i := QueueItem{}
	err := json.Unmarshal([]byte(val), &i)
	require.NoError(t, err)
	return i
}

func requirePartitionInProgress(t *testing.T, q *queue, workflowID uuid.UUID, count int) {
	t.Helper()
	actual, err := q.InProgress(context.Background(), "p", workflowID.String())
	require.NoError(t, err)
	require.EqualValues(t, count, actual)
}

func getPartition(t *testing.T, r *miniredis.Miniredis, id uuid.UUID) QueuePartition {
	t.Helper()
	val := r.HGet(defaultQueueKey.PartitionItem(), id.String())
	qp := QueuePartition{}
	err := json.Unmarshal([]byte(val), &qp)
	require.NoError(t, err)
	return qp
}

func requireItemScoreEquals(t *testing.T, r *miniredis.Miniredis, item QueueItem, expected time.Time) {
	t.Helper()
	score, err := r.ZScore(defaultQueueKey.QueueIndex(item.WorkflowID.String()), item.ID)
	parsed := time.UnixMilli(int64(score))
	require.NoError(t, err)
	require.WithinDuration(t, expected.Truncate(time.Millisecond), parsed, 15*time.Millisecond)
}

func requirePartitionScoreEquals(t *testing.T, r *miniredis.Miniredis, wid uuid.UUID, expected time.Time) {
	t.Helper()
	score, err := r.ZScore(defaultQueueKey.PartitionIndex(), wid.String())
	parsed := time.Unix(int64(score), 0)
	require.NoError(t, err)
	require.WithinDuration(t, expected.Truncate(time.Second), parsed, time.Millisecond)
}

func concurrencyQueueScores(t *testing.T, r *miniredis.Miniredis, key string, from time.Time) map[string]time.Time {
	t.Helper()
	members, err := r.ZMembers(key)
	require.NoError(t, err)
	scores := map[string]time.Time{}
	for _, item := range members {
		score, err := r.ZScore(key, item)
		require.NoError(t, err)
		scores[item] = time.UnixMilli(int64(score))
	}
	return scores
}
