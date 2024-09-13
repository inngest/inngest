package redis_state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func init() {
	miniredis.DumpMaxLineLen = 1024
}

const testPriority = PriorityDefault

func TestQueueItemScore(t *testing.T) {
	parse := func(layout, val string) time.Time {
		t, _ := time.Parse(layout, val)
		return t
	}

	start := parse(time.RFC3339, "2023-01-01T12:30:30.000Z")
	old := parse(time.RFC3339, "2022-09-01T12:30:30.000Z")

	tests := []struct {
		name     string
		qi       QueueItem
		expected int64
	}{
		{
			name:     "Current edge queue",
			expected: start.UnixMilli(),
			qi: QueueItem{
				AtMS: start.UnixMilli(),
				Data: osqueue.Item{
					Kind: osqueue.KindEdge,
					Identifier: state.Identifier{
						RunID: ulid.MustNew(uint64(start.UnixMilli()), rand.Reader),
					},
				},
			},
		},
		{
			name:     "Item with old run",
			expected: old.UnixMilli(),
			qi: QueueItem{
				AtMS: start.UnixMilli(),
				Data: osqueue.Item{
					Kind: osqueue.KindEdge,
					Identifier: state.Identifier{
						RunID: ulid.MustNew(uint64(old.UnixMilli()), rand.Reader),
					},
				},
			},
		},
		// Edge cases
		{
			name:     "Item with old run, 2nd attempt",
			expected: start.UnixMilli(),
			qi: QueueItem{
				AtMS: start.UnixMilli(),
				Data: osqueue.Item{
					Kind:    osqueue.KindEdge,
					Attempt: 2,
					Identifier: state.Identifier{
						RunID: ulid.MustNew(uint64(old.UnixMilli()), rand.Reader),
					},
				},
			},
		},
		{
			name:     "Item within leeway",
			expected: start.UnixMilli(),
			qi: QueueItem{
				AtMS: start.UnixMilli(),
				Data: osqueue.Item{
					Kind:    osqueue.KindEdge,
					Attempt: 2,
					Identifier: state.Identifier{
						RunID: ulid.MustNew(uint64(start.UnixMilli()-1_000), rand.Reader),
					},
				},
			},
		},
		{
			name:     "Sleep",
			expected: start.UnixMilli(),
			qi: QueueItem{
				AtMS: start.UnixMilli(),
				Data: osqueue.Item{
					Kind: osqueue.KindSleep,
					Identifier: state.Identifier{
						RunID: ulid.MustNew(uint64(old.UnixMilli()), rand.Reader),
					},
				},
			},
		},
		// PriorityFactor
		{
			name:     "With PriorityFactor of -60",
			expected: old.Add(60 * time.Second).UnixMilli(), // subtract two seconds given factor
			qi: QueueItem{
				AtMS: start.UnixMilli(),
				Data: osqueue.Item{
					Kind: osqueue.KindEdge,
					Identifier: state.Identifier{
						RunID: ulid.MustNew(
							uint64(old.UnixMilli()),
							rand.Reader,
						),
						PriorityFactor: int64ptr(-60),
					},
				},
			},
		},
		{
			name:     "With PriorityFactor of 30",
			expected: old.Add(-30 * time.Second).UnixMilli(), // subtract two seconds given factor
			qi: QueueItem{
				AtMS: start.UnixMilli(),
				Data: osqueue.Item{
					Kind: osqueue.KindEdge,
					Identifier: state.Identifier{
						RunID: ulid.MustNew(
							uint64(old.UnixMilli()),
							rand.Reader,
						),
						PriorityFactor: int64ptr(30),
					},
				},
			},
		},
		{
			name:     "Sleep with PF does nothing",
			expected: start.UnixMilli(),
			qi: QueueItem{
				AtMS: start.UnixMilli(),
				Data: osqueue.Item{
					Kind: osqueue.KindSleep,
					Identifier: state.Identifier{
						RunID: ulid.MustNew(uint64(old.UnixMilli()), rand.Reader),
						// Subtract 2
						PriorityFactor: int64ptr(30),
					},
				},
			},
		},
	}

	for _, item := range tests {
		actual := item.qi.Score(time.Now())
		require.Equal(t, item.expected, actual)
	}
}

func TestQueueItemIsLeased(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		time     time.Time
		expected bool
	}{
		{
			name:     "returns true for leased item",
			time:     now.Add(1 * time.Minute), // 1m later
			expected: true,
		},
		{
			name:     "returns false for item with expired lease",
			time:     now.Add(-1 * time.Minute), // 1m ago
			expected: false,
		},
		{
			name:     "returns false for empty lease ID",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			qi := &QueueItem{}
			if !test.time.IsZero() {
				leaseID, err := ulid.New(ulid.Timestamp(test.time), rand.Reader)
				if err != nil {
					t.Fatalf("failed to create new LeaseID: %v\n", err)
				}
				qi.LeaseID = &leaseID
			}

			require.Equal(t, test.expected, qi.IsLeased(now))
		})
	}
}

func TestQueueEnqueueItem(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	q := NewQueue(NewQueueClient(rc, QueueDefaultKey))
	ctx := context.Background()

	start := time.Now().Truncate(time.Second)

	t.Run("It enqueues an item", func(t *testing.T) {
		item, err := q.EnqueueItem(ctx, QueueItem{}, start)
		require.NoError(t, err)
		require.NotEqual(t, item.ID, ulid.ULID{})
		require.Equal(t, time.UnixMilli(item.WallTimeMS).Truncate(time.Second), start)

		// Ensure that our data is set up correctly.
		found := getQueueItem(t, r, item.ID)
		require.Equal(t, item, found)

		// Ensure the partition is inserted.
		qp := getPartition(t, r, item.WorkflowID)
		require.Equal(t, QueuePartition{
			WorkflowID: item.WorkflowID,
			Priority:   testPriority,
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
		}, qp)

		// Ensure that the zscore did not change.
		keys, err := r.ZMembers(q.u.kg.GlobalPartitionIndex())
		require.NoError(t, err)
		require.Equal(t, 1, len(keys))
		score, err := r.ZScore(q.u.kg.GlobalPartitionIndex(), keys[0])
		require.NoError(t, err)
		require.EqualValues(t, start.Unix(), score)
	})

	t.Run("Updates partition vesting time to earlier times", func(t *testing.T) {

		now := time.Now()
		at := now.Add(-10 * time.Minute).Truncate(time.Second)
		item, err := q.EnqueueItem(ctx, QueueItem{}, at)
		require.NoError(t, err)

		// Ensure the partition is inserted, and the earliest time is updated
		// inside the partition item.
		qp := getPartition(t, r, item.WorkflowID)
		require.Equal(t, QueuePartition{
			WorkflowID: item.WorkflowID,
			Priority:   testPriority,
		}, qp)

		// Assert that the zscore was changed to this earliest timestamp.
		keys, err := r.ZMembers(q.u.kg.GlobalPartitionIndex())
		require.NoError(t, err)
		require.Equal(t, 1, len(keys))
		score, err := r.ZScore(q.u.kg.GlobalPartitionIndex(), keys[0])
		require.NoError(t, err)
		require.EqualValues(t, now.Unix(), score)
	})

	t.Run("Adding another workflow ID increases partition set", func(t *testing.T) {
		at := time.Now().Truncate(time.Second)
		item, err := q.EnqueueItem(ctx, QueueItem{
			WorkflowID: uuid.New(),
		}, at)
		require.NoError(t, err)

		// Assert that we have two zscores in partition:sorted.
		keys, err := r.ZMembers(q.u.kg.GlobalPartitionIndex())
		require.NoError(t, err)
		require.Equal(t, 2, len(keys))

		// Ensure the partition is inserted, and the earliest time is updated
		// inside the partition item.
		qp := getPartition(t, r, item.WorkflowID)
		require.Equal(t, QueuePartition{
			WorkflowID: item.WorkflowID,
			Priority:   testPriority,
		}, qp)
	})

	t.Run("Stores default indexes", func(t *testing.T) {
		at := time.Now().Truncate(time.Second)
		rid := ulid.MustNew(ulid.Now(), rand.Reader)
		_, err := q.EnqueueItem(ctx, QueueItem{
			WorkflowID: uuid.New(),
			Data: osqueue.Item{
				Kind: osqueue.KindEdge,
				Identifier: state.Identifier{
					RunID: rid,
				},
			},
		}, at)
		require.NoError(t, err)

		keys, err := r.ZMembers(fmt.Sprintf("{queue}:idx:run:%s", rid))
		require.NoError(t, err)
		require.Equal(t, 1, len(keys))
	})

	t.Run("Enqueueing to a paused partition does not affect the partition's pause state", func(t *testing.T) {
		now := time.Now()
		workflowId := uuid.New()

		item, err := q.EnqueueItem(ctx, QueueItem{
			WorkflowID: workflowId,
		}, now.Add(10*time.Second))
		require.NoError(t, err)

		err = q.SetFunctionPaused(ctx, item.WorkflowID, true)
		require.NoError(t, err)

		item, err = q.EnqueueItem(ctx, QueueItem{
			WorkflowID: workflowId,
		}, now)
		require.NoError(t, err)

		second := getPartition(t, r, item.WorkflowID)
		require.True(t, second.Paused)

		item, err = q.EnqueueItem(ctx, QueueItem{
			WorkflowID: workflowId,
		}, now.Add(-10*time.Second))
		require.NoError(t, err)

		second = getPartition(t, r, item.WorkflowID)
		require.True(t, second.Paused)
	})
}

func TestQueueEnqueueItemIdempotency(t *testing.T) {
	dur := 2 * time.Second

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	// Set idempotency to a second
	q := NewQueue(NewQueueClient(rc, QueueDefaultKey), WithIdempotencyTTL(dur))
	ctx := context.Background()

	start := time.Now().Truncate(time.Second)

	t.Run("It enqueues an item only once", func(t *testing.T) {
		i := QueueItem{ID: "once"}

		item, err := q.EnqueueItem(ctx, i, start)
		p := QueuePartition{WorkflowID: item.WorkflowID}

		require.NoError(t, err)
		require.Equal(t, HashID(ctx, "once"), item.ID)
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
		require.Equal(t, HashID(ctx, "once"), item.ID)
		require.NotEqual(t, i.ID, item.ID)
		found = getQueueItem(t, r, item.ID)
		require.Equal(t, item, found)
	})
}

func BenchmarkPeekTiming(b *testing.B) {

	//
	// Setup
	//
	address := os.Getenv("REDIS_ADDR")
	if address == "" {
		r, err := miniredis.Run()
		if err != nil {
			panic(err)
		}
		address = r.Addr()
		defer r.Close()
		fmt.Println("using miniredis")
	}
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{address},
		DisableCache: true,
	})
	if err != nil {
		panic(err)
	}
	defer rc.Close()

	//
	// Tests
	//

	// Enqueue 500 items into one queue.

	q := NewQueue(NewQueueClient(rc, QueueDefaultKey))
	ctx := context.Background()

	enqueue := func(id uuid.UUID, n int) {
		for i := 0; i < n; i++ {
			_, err := q.EnqueueItem(ctx, QueueItem{WorkflowID: id}, time.Now())
			if err != nil {
				panic(err)
			}
		}
	}

	for i := 0; i < b.N; i++ {
		id := uuid.New()
		enqueue(id, int(QueuePeekMax))
		items, err := q.Peek(ctx, id.String(), time.Now(), QueuePeekMax)
		if err != nil {
			panic(err)
		}
		if len(items) != int(QueuePeekMax) {
			panic(fmt.Sprintf("expected %d, got %d", QueuePeekMax, len(items)))
		}
	}
}

func TestQueuePeek(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	q := NewQueue(NewQueueClient(rc, QueueDefaultKey))
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

		ia, err := q.EnqueueItem(ctx, QueueItem{ID: "a"}, a)
		require.NoError(t, err)
		ib, err := q.EnqueueItem(ctx, QueueItem{ID: "b"}, b)
		require.NoError(t, err)
		ic, err := q.EnqueueItem(ctx, QueueItem{ID: "c"}, c)
		require.NoError(t, err)

		items, err := q.Peek(ctx, workflowID.String(), time.Now().Add(time.Hour), 10)
		require.NoError(t, err)
		require.EqualValues(t, 3, len(items))
		require.EqualValues(t, []*QueueItem{&ia, &ib, &ic}, items)
		require.NotEqualValues(t, []*QueueItem{&ib, &ia, &ic}, items)

		id, err := q.EnqueueItem(ctx, QueueItem{ID: "d"}, d)
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
			_, err := q.Lease(ctx, p, ia, 50*time.Millisecond, time.Now(), nil)
			require.NoError(t, err)

			items, err = q.Peek(ctx, workflowID.String(), d, QueuePeekMax)
			require.NoError(t, err)
			require.EqualValues(t, 3, len(items))
			require.EqualValues(t, []*QueueItem{&ib, &ic, &id}, items)
		})

		t.Run("Expired leases should move back via scavenging", func(t *testing.T) {
			// Run scavenging.
			caught, err := q.Scavenge(ctx, ScavengePeekSize)
			require.NoError(t, err)
			require.EqualValues(t, 0, caught)

			// When the lease expires it should re-appear
			<-time.After(55 * time.Millisecond)

			// Run scavenging.
			scavengeAt := time.Now().UnixMilli()
			caught, err = q.Scavenge(ctx, ScavengePeekSize)
			require.NoError(t, err)
			require.EqualValues(t, 1, caught)

			items, err = q.Peek(ctx, workflowID.String(), d, QueuePeekMax)
			require.NoError(t, err)
			require.EqualValues(t, 4, len(items))

			// Ignore items earlies peek time.
			for _, i := range items {
				if i.EarliestPeekTime != 0 {
					i.EarliestPeekTime = 0
				}
			}

			require.EqualValues(t, ia.ID, items[0].ID)
			// NOTE: Scavenging requeues items, and so the time will have changed.
			require.GreaterOrEqual(t, items[0].AtMS, scavengeAt)
			require.Greater(t, items[0].AtMS, ia.AtMS)
			ia.LeaseID = nil
			ia.AtMS = items[0].AtMS
			ia.WallTimeMS = items[0].WallTimeMS
			require.EqualValues(t, []*QueueItem{&ia, &ib, &ic, &id}, items)
		})

		t.Run("Random scavenge offset should work", func(t *testing.T) {
			// When count is within limits, do not apply offset
			require.Equal(t, int64(0), q.randomScavengeOffset(1, 1, 1))
			require.Equal(t, int64(0), q.randomScavengeOffset(1, 2, 3))

			// Some random fixtures to verify we stay within the range
			require.Equal(t, int64(2), q.randomScavengeOffset(1, 4, 1))
			require.Equal(t, int64(3), q.randomScavengeOffset(2, 4, 1))
			require.Equal(t, int64(1), q.randomScavengeOffset(3, 4, 1))
			require.Equal(t, int64(2), q.randomScavengeOffset(4, 4, 1))
			require.Equal(t, int64(0), q.randomScavengeOffset(5, 4, 1))
		})

		t.Run("Backcompat: Scavenger should gracefully handle invalid items and continue processing", func(t *testing.T) {
			r.FlushAll()

			start := time.Now().Truncate(time.Second)

			fnIdA, fnIdB, fnIdC := uuid.New(), uuid.New(), uuid.New()

			ia, err := q.EnqueueItem(ctx, QueueItem{ID: "a", WorkflowID: fnIdA}, start)
			require.NoError(t, err)
			pA := QueuePartition{WorkflowID: ia.WorkflowID}

			ib, err := q.EnqueueItem(ctx, QueueItem{ID: "b", WorkflowID: fnIdB}, start)
			require.NoError(t, err)
			pB := QueuePartition{WorkflowID: ib.WorkflowID}

			ic, err := q.EnqueueItem(ctx, QueueItem{ID: "c", WorkflowID: fnIdC}, start)
			require.NoError(t, err)
			pC := QueuePartition{WorkflowID: ic.WorkflowID}

			now := time.Now()

			_, err = q.Lease(ctx, pA, ia, 50*time.Millisecond, now, nil)
			require.NoError(t, err)
			_, err = q.Lease(ctx, pB, ib, 50*time.Millisecond, now, nil)
			require.NoError(t, err)
			_, err = q.Lease(ctx, pC, ic, 50*time.Millisecond, now, nil)
			require.NoError(t, err)

			// Run scavenging.
			caught, err := q.Scavenge(ctx, ScavengePeekSize)
			require.NoError(t, err)
			require.EqualValues(t, 0, caught)

			// When the lease expires it should re-appear
			<-time.After(100 * time.Millisecond)

			// Run scavenging.
			caught, err = q.Scavenge(ctx, 1)
			require.NoError(t, err)
			require.EqualValues(t, 1, caught)

			require.True(t, r.Exists(q.u.kg.ConcurrencyIndex()), r.Dump())
			concurrencyIndexMembers, err := r.ZMembers(q.u.kg.ConcurrencyIndex())
			require.NoError(t, err)
			require.Equal(t, 2, len(concurrencyIndexMembers))

			var aExists, bExists, cExists bool
			for _, member := range concurrencyIndexMembers {
				if member == fnIdA.String() {
					aExists = true
				} else if member == fnIdB.String() {
					bExists = true
				} else if member == fnIdC.String() {
					cExists = true
				}

				concurrencyKey := q.u.kg.Concurrency("p", member)
				require.True(t, r.Exists(concurrencyKey), r.Dump())
				inProgress, err := r.ZMembers(concurrencyKey)
				require.NoError(t, err)
				require.Equal(t, 1, len(inProgress))
			}

			// At least one may not exist anymore
			require.False(t, aExists && bExists && cExists)

			// Ensure the concurrency queue is empty but the partition queue has exactly one item (scavenge worked)
			if !aExists {
				require.False(t, r.Exists(q.u.kg.Concurrency("p", fnIdA.String())))
				partMem, err := r.ZMembers(q.u.kg.QueueIndex(fnIdA.String()))
				require.NoError(t, err)
				require.Equal(t, 1, len(partMem))
			}

			if !bExists {
				require.False(t, r.Exists(q.u.kg.Concurrency("p", fnIdB.String())))
				partMem, err := r.ZMembers(q.u.kg.QueueIndex(fnIdB.String()))
				require.NoError(t, err)
				require.Equal(t, 1, len(partMem))
			}

			if !cExists {
				require.False(t, r.Exists(q.u.kg.Concurrency("p", fnIdC.String())))
				partMem, err := r.ZMembers(q.u.kg.QueueIndex(fnIdC.String()))
				require.NoError(t, err)
				require.Equal(t, 1, len(partMem))
			}
		})
	})
}

func TestQueueLease(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	queueClient := NewQueueClient(rc, QueueDefaultKey)
	defaultQueueKey := queueClient.kg
	q := NewQueue(queueClient)

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
		id, err := q.Lease(ctx, p, item, time.Second, time.Now(), nil)
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
				id, err := q.Lease(ctx, p, item, time.Second, time.Now(), nil)
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
			id, err := q.Lease(ctx, p, item, 5*time.Second, time.Now(), nil)
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

			_, err = q.Lease(ctx, p, item, time.Minute, time.Now(), nil)
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

		t.Run("With denylists it does not lease.", func(t *testing.T) {
			list := newLeaseDenyList()
			list.addConcurrency(newKeyError(ErrPartitionConcurrencyLimit, p.Queue()))
			_, err = q.Lease(ctx, p, itemA, 5*time.Second, time.Now(), list)
			require.ErrorIs(t, err, ErrPartitionConcurrencyLimit)
		})

		t.Run("Leases with capacity", func(t *testing.T) {
			_, err = q.Lease(ctx, p, itemA, 5*time.Second, time.Now(), nil)
			require.NoError(t, err)
		})
		t.Run("Errors without capacity", func(t *testing.T) {
			id, err := q.Lease(ctx, p, itemB, 5*time.Second, time.Now(), nil)
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
			_, err = q.Lease(ctx, p, itemA, 5*time.Second, time.Now(), nil)
			require.NoError(t, err)
		})
		t.Run("Errors without capacity", func(t *testing.T) {
			id, err := q.Lease(ctx, p, itemB, 5*time.Second, time.Now(), nil)
			require.Nil(t, id)
			require.Error(t, err)
		})
	})

	t.Run("With custom concurrency limits", func(t *testing.T) {
		q.partitionConcurrencyGen = func(ctx context.Context, p QueuePartition) (string, int) {
			return p.Queue(), 100
		}
		q.accountConcurrencyGen = nil
		q.customConcurrencyGen = func(ctx context.Context, i QueueItem) []state.CustomConcurrency {
			return []state.CustomConcurrency{
				{
					Key:   "custom-level-key",
					Limit: 1,
				},
			}
		}

		// Create a new item
		itemA, err := q.EnqueueItem(ctx, QueueItem{WorkflowID: uuid.New()}, start)
		require.NoError(t, err)
		itemB, err := q.EnqueueItem(ctx, QueueItem{WorkflowID: uuid.New()}, start)
		require.NoError(t, err)
		// Use the new item's workflow ID
		p := QueuePartition{WorkflowID: itemA.WorkflowID}

		t.Run("With denylists it does not lease.", func(t *testing.T) {
			list := newLeaseDenyList()
			list.addConcurrency(newKeyError(ErrConcurrencyLimitCustomKey0, "custom-level-key"))
			_, err = q.Lease(ctx, p, itemA, 5*time.Second, time.Now(), list)
			require.ErrorIs(t, err, ErrConcurrencyLimitCustomKey0)
		})

		t.Run("Leases with capacity", func(t *testing.T) {
			_, err = q.Lease(ctx, p, itemA, 5*time.Second, time.Now(), nil)
			require.NoError(t, err)
		})
		t.Run("Errors without capacity", func(t *testing.T) {
			id, err := q.Lease(ctx, p, itemB, 5*time.Second, time.Now(), nil)
			require.Nil(t, id)
			require.Error(t, err)
		})
	})

	t.Run("It should update the global partition index", func(t *testing.T) {
		r.FlushAll()

		// NOTE: We need two items to ensure that this updates.  Leasing an
		// item removes it from the fn queue.
		t.Run("With a single item in the queue hwen leasing, nothing updates", func(t *testing.T) {
			at := time.Now().Truncate(time.Second).Add(time.Second)
			item, err := q.EnqueueItem(ctx, QueueItem{}, at)
			require.NoError(t, err)
			p := QueuePartition{WorkflowID: item.WorkflowID}

			score, err := r.ZScore(defaultQueueKey.GlobalPartitionIndex(), p.Queue())
			require.NoError(t, err)
			require.EqualValues(t, at.Unix(), score)

			// Nothing should update here, as there's nothing left in the fn queue
			// so nothing happens.
			_, err = q.Lease(ctx, p, item, 10*time.Second, time.Now(), nil)
			require.NoError(t, err)

			nextScore, err := r.ZScore(defaultQueueKey.GlobalPartitionIndex(), p.Queue())
			require.NoError(t, err)
			require.EqualValues(t, int(score), int(nextScore), "score should not equal previous score")
		})

		r.FlushAll()

		t.Run("With more than one item in the fn queue, it uses the next val", func(t *testing.T) {
			atA := time.Now().Truncate(time.Second).Add(time.Second)
			atB := atA.Add(time.Minute)

			itemA, err := q.EnqueueItem(ctx, QueueItem{}, atA)
			require.NoError(t, err)
			itemB, err := q.EnqueueItem(ctx, QueueItem{}, atB)
			require.NoError(t, err)
			p := QueuePartition{WorkflowID: itemA.WorkflowID} // same for A+B

			score, err := r.ZScore(defaultQueueKey.GlobalPartitionIndex(), p.Queue())
			require.NoError(t, err)
			require.EqualValues(t, atA.Unix(), score)

			// Leasing the item should update the score.
			_, err = q.Lease(ctx, p, itemA, 10*time.Second, time.Now(), nil)
			require.NoError(t, err)

			nextScore, err := r.ZScore(defaultQueueKey.GlobalPartitionIndex(), p.Queue())
			require.NoError(t, err)
			require.EqualValues(t, itemB.AtMS/1000, nextScore)
			require.NotEqualValues(t, int(score), int(nextScore), "score should not equal previous score")
		})
	})
}

func TestQueueExtendLease(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	queueClient := NewQueueClient(rc, QueueDefaultKey)
	q := NewQueue(queueClient)
	ctx := context.Background()

	start := time.Now().Truncate(time.Second)
	t.Run("It leases an item", func(t *testing.T) {
		item, err := q.EnqueueItem(ctx, QueueItem{}, start)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		p := QueuePartition{WorkflowID: item.WorkflowID}

		now := time.Now()
		id, err := q.Lease(ctx, p, item, time.Second, time.Now(), nil)
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
			scores := concurrencyQueueScores(t, r, queueClient.kg.Concurrency("p", pkey), time.Now())
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

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	queueClient := NewQueueClient(rc, QueueDefaultKey)
	defaultQueueKey := queueClient.kg
	q := NewQueue(queueClient)
	ctx := context.Background()

	t.Run("It should remove a queue item", func(t *testing.T) {
		start := time.Now()

		item, err := q.EnqueueItem(ctx, QueueItem{}, start)
		require.NoError(t, err)

		p := QueuePartition{WorkflowID: item.WorkflowID}

		id, err := q.Lease(ctx, p, item, time.Second, time.Now(), nil)
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

		t.Run("Removes default indexes", func(t *testing.T) {
			at := time.Now().Truncate(time.Second)
			rid := ulid.MustNew(ulid.Now(), rand.Reader)
			item, err := q.EnqueueItem(ctx, QueueItem{
				WorkflowID: uuid.New(),
				Data: osqueue.Item{
					Kind: osqueue.KindEdge,
					Identifier: state.Identifier{
						RunID: rid,
					},
				},
			}, at)
			require.NoError(t, err)

			keys, err := r.ZMembers(fmt.Sprintf("{queue}:idx:run:%s", rid))
			require.NoError(t, err)
			require.Equal(t, 1, len(keys))

			err = q.Dequeue(ctx, p, item)
			require.NoError(t, err)

			keys, err = r.ZMembers(fmt.Sprintf("{queue}:idx:run:%s", rid))
			require.NotNil(t, err)
			require.Equal(t, true, strings.Contains(err.Error(), "no such key"))
			require.Equal(t, 0, len(keys))
		})
	})

}

func TestQueueRequeue(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	q := NewQueue(NewQueueClient(rc, QueueDefaultKey))
	ctx := context.Background()

	t.Run("Re-enqueuing a leased item should succeed", func(t *testing.T) {
		now := time.Now()

		item, err := q.EnqueueItem(ctx, QueueItem{}, now)
		require.NoError(t, err)

		p := QueuePartition{WorkflowID: item.WorkflowID}

		_, err = q.Lease(ctx, p, item, time.Second, time.Now(), nil)
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

		t.Run("Updates default indexes", func(t *testing.T) {
			at := time.Now().Truncate(time.Second)
			rid := ulid.MustNew(ulid.Now(), rand.Reader)
			item, err := q.EnqueueItem(ctx, QueueItem{
				WorkflowID: uuid.New(),
				Data: osqueue.Item{
					Kind: osqueue.KindEdge,
					Identifier: state.Identifier{
						RunID: rid,
					},
				},
			}, at)
			require.NoError(t, err)

			key := fmt.Sprintf("{queue}:idx:run:%s", rid)

			keys, err := r.ZMembers(key)
			require.NoError(t, err)
			require.Equal(t, 1, len(keys))

			// Score for entry should be the first enqueue time.
			scores, err := r.ZMScore(key, keys[0])
			require.NoError(t, err)
			require.EqualValues(t, at.UnixMilli(), scores[0])

			next := now.Add(2 * time.Hour)
			err = q.Requeue(ctx, pi, item, next)
			require.NoError(t, err)

			// Score should be the requeue time.
			scores, err = r.ZMScore(key, keys[0])
			require.NoError(t, err)
			require.EqualValues(t, next.UnixMilli(), scores[0])

			// Still only one member.
			keys, err = r.ZMembers(key)
			require.NoError(t, err)
			require.Equal(t, 1, len(keys))
		})
	})
}

func TestQueuePartitionLease(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	idA, idB, idC := uuid.New(), uuid.New(), uuid.New()
	atA, atB, atC := now, now.Add(time.Second), now.Add(2*time.Second)

	pA := QueuePartition{WorkflowID: idA}

	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	q := NewQueue(NewQueueClient(rc, QueueDefaultKey))
	ctx := context.Background()

	_, err = q.EnqueueItem(ctx, QueueItem{WorkflowID: idA}, atA)
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
			{WorkflowID: idA, Priority: testPriority},
			{WorkflowID: idB, Priority: testPriority},
			{WorkflowID: idC, Priority: testPriority},
		}, items)
	})

	leaseUntil := now.Add(3 * time.Second)

	t.Run("It leases a partition", func(t *testing.T) {
		// Lease the first item now.
		leasedAt := time.Now()
		leaseID, err := q.PartitionLease(ctx, &pA, time.Until(leaseUntil))
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		// Pause so that we can assert that the last lease time was set correctly.
		<-time.After(50 * time.Millisecond)

		t.Run("It updates the partition score", func(t *testing.T) {
			items, err := q.PartitionPeek(ctx, true, now.Add(time.Hour), PartitionPeekMax)

			// Require the lease ID is within 25 MS of the expected value.
			require.WithinDuration(t, leaseUntil, ulid.Time(leaseID.Time()), 25*time.Millisecond)

			require.NoError(t, err)
			require.Len(t, items, 3)
			require.EqualValues(t, []*QueuePartition{
				{WorkflowID: idB, Priority: testPriority},
				{WorkflowID: idC, Priority: testPriority},
				{
					WorkflowID: idA,
					Priority:   testPriority,
					Last:       items[2].Last, // Use the leased partition time.
					LeaseID:    leaseID,
				}, // idA is now last.
			}, items)
			requirePartitionScoreEquals(t, r, idA, leaseUntil)
			// require that the last leased time is within 5ms for tests
			require.WithinDuration(t, leasedAt, time.UnixMilli(items[2].Last), 5*time.Millisecond)
		})

		t.Run("It can't lease an existing partition lease", func(t *testing.T) {
			id, err := q.PartitionLease(ctx, &pA, time.Second*29)
			require.Equal(t, ErrPartitionAlreadyLeased, err)
			require.Nil(t, id)

			// Assert that score didn't change (we added 1 second in the previous test)
			requirePartitionScoreEquals(t, r, idA, leaseUntil)
		})

	})

	t.Run("It allows leasing an expired partition lease", func(t *testing.T) {
		<-time.After(time.Until(leaseUntil))

		requirePartitionScoreEquals(t, r, idA, leaseUntil)

		id, err := q.PartitionLease(ctx, &pA, time.Second*5)
		require.Nil(t, err)
		require.NotNil(t, id)

		requirePartitionScoreEquals(t, r, idA, time.Now().Add(time.Second*5))
	})

	t.Run("Partition pausing", func(t *testing.T) {
		r.FlushAll() // reset everything
		q := NewQueue(NewQueueClient(rc, QueueDefaultKey))
		ctx := context.Background()

		_, err = q.EnqueueItem(ctx, QueueItem{WorkflowID: idA}, atA)
		require.NoError(t, err)
		_, err = q.EnqueueItem(ctx, QueueItem{WorkflowID: idB}, atB)
		require.NoError(t, err)
		_, err = q.EnqueueItem(ctx, QueueItem{WorkflowID: idC}, atC)
		require.NoError(t, err)

		t.Run("Fails to lease a paused partition", func(t *testing.T) {
			// pause fn A's partition:
			err = q.SetFunctionPaused(ctx, idA, true)
			require.NoError(t, err)

			// attempt to lease the paused partition:
			id, err := q.PartitionLease(ctx, &pA, time.Second*5)
			require.Nil(t, id)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrPartitionPaused)
		})

		t.Run("Succeeds to lease a previously paused partition", func(t *testing.T) {
			// unpause fn A's partition:
			err = q.SetFunctionPaused(ctx, idA, false)
			require.NoError(t, err)

			// attempt to lease the unpaused partition:
			id, err := q.PartitionLease(ctx, &pA, time.Second*5)
			require.NotNil(t, id)
			require.NoError(t, err)
		})
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

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	q := NewQueue(
		NewQueueClient(rc, QueueDefaultKey),
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
			{WorkflowID: idA, Priority: PriorityMin},
			{WorkflowID: idB, Priority: PriorityMax},
			{WorkflowID: idC, Priority: PriorityMax},
		}, items)
	})

	t.Run("With a single peek max, it returns the first item if sequential every time", func(t *testing.T) {
		for i := 0; i <= 50; i++ {
			items, err := q.PartitionPeek(ctx, true, time.Now().Add(time.Hour), 1)
			require.NoError(t, err)
			require.Len(t, items, 1)
			require.Equal(t, idA, items[0].WorkflowID)
		}
	})

	t.Run("With a single peek max, it returns random items that are available using offsets", func(t *testing.T) {
		found := map[uuid.UUID]bool{idA: false, idB: false, idC: false}

		for i := 0; i <= 50; i++ {
			items, err := q.PartitionPeek(ctx, false, time.Now().Add(time.Hour), 1)
			require.NoError(t, err)
			require.Len(t, items, 1)
			found[items[0].WorkflowID] = true
			<-time.After(time.Millisecond)
		}

		for id, v := range found {
			require.True(t, v, "PartitionPeek didn't find id '%s' via random offsets", id)
		}
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

		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		q := NewQueue(
			NewQueueClient(rc, QueueDefaultKey),
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
			{WorkflowID: idB, Priority: PriorityMin},
			{WorkflowID: idC, Priority: PriorityMin},
		}, items)

		// Try without sequential scans
		items, err = q.PartitionPeek(ctx, false, time.Now().Add(time.Hour), PartitionPeekMax)
		require.NoError(t, err)
		require.Len(t, items, 2)
	})

	t.Run("Peeking ignores paused partitions", func(t *testing.T) {
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		q := NewQueue(
			NewQueueClient(rc, QueueDefaultKey),
			WithPriorityFinder(func(ctx context.Context, qi QueueItem) uint {
				return PriorityDefault
			}),
		)
		enqueue(q)

		// Pause A, excluding it from peek:
		err = q.SetFunctionPaused(ctx, idA, true)
		require.NoError(t, err)

		// This should only select B and C, as id A is ignored:
		items, err := q.PartitionPeek(ctx, true, time.Now().Add(time.Hour), PartitionPeekMax)
		require.NoError(t, err)
		require.Len(t, items, 2)
		require.EqualValues(t, []*QueuePartition{
			{WorkflowID: idB, Priority: PriorityDefault},
			{WorkflowID: idC, Priority: PriorityDefault},
		}, items)

		// After unpausing A, it should be included in the peek:
		err = q.SetFunctionPaused(ctx, idA, false)
		require.NoError(t, err)
		items, err = q.PartitionPeek(ctx, true, time.Now().Add(time.Hour), PartitionPeekMax)
		require.NoError(t, err)
		require.Len(t, items, 3)
		require.EqualValues(t, []*QueuePartition{
			{WorkflowID: idA, Priority: PriorityDefault},
			{WorkflowID: idB, Priority: PriorityDefault},
			{WorkflowID: idC, Priority: PriorityDefault},
		}, items)
	})
}

func TestQueuePartitionRequeue(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	q := NewQueue(NewQueueClient(rc, QueueDefaultKey))
	ctx := context.Background()
	idA := uuid.New()
	now := time.Now()

	qi, err := q.EnqueueItem(ctx, QueueItem{WorkflowID: idA}, now)
	require.NoError(t, err)

	p := QueuePartition{WorkflowID: qi.WorkflowID, WorkspaceID: qi.WorkspaceID}

	t.Run("Uses the next job item's time when requeueing with another job", func(t *testing.T) {
		requirePartitionScoreEquals(t, r, idA, now)
		next := now.Add(time.Hour)
		err := q.PartitionRequeue(ctx, &p, next, false)
		require.NoError(t, err)
		requirePartitionScoreEquals(t, r, idA, now)
	})

	next := now.Add(5 * time.Second)
	t.Run("It removes any lease when requeueing", func(t *testing.T) {

		_, err := q.PartitionLease(ctx, &QueuePartition{WorkflowID: idA}, time.Minute)
		require.NoError(t, err)

		err = q.PartitionRequeue(ctx, &p, next, true)
		require.NoError(t, err)
		requirePartitionScoreEquals(t, r, idA, next)

		loaded := getPartition(t, r, idA)
		require.Nil(t, loaded.LeaseID)

		// Forcing should set a ForceAtMS field.
		require.NotEmpty(t, loaded.ForceAtMS)

		t.Run("Enqueueing with a force at time should not update the score", func(t *testing.T) {
			loaded := getPartition(t, r, idA)
			require.NotEmpty(t, loaded.ForceAtMS)

			qi, err := q.EnqueueItem(ctx, QueueItem{WorkflowID: idA}, now)

			loaded = getPartition(t, r, idA)
			require.NotEmpty(t, loaded.ForceAtMS)

			require.NoError(t, err)
			requirePartitionScoreEquals(t, r, idA, next)
			requirePartitionScoreEquals(t, r, idA, time.UnixMilli(loaded.ForceAtMS))

			// Now remove this item, as we dont need it for any future tests.
			err = q.Dequeue(ctx, p, qi)
			require.NoError(t, err)
		})
	})

	t.Run("Deletes the partition with an empty queue and a leased job", func(t *testing.T) {
		requirePartitionScoreEquals(t, r, idA, next)

		// Leasing the only job available moves the job into the concurrency queue,
		// so the partition should be empty. when requeeing.
		_, err := q.Lease(ctx, p, qi, 10*time.Second, time.Now(), nil)
		require.NoError(t, err)

		requirePartitionScoreEquals(t, r, idA, next)

		next := now.Add(time.Hour)
		err = q.PartitionRequeue(ctx, &p, next, false)
		require.Error(t, ErrPartitionGarbageCollected, err)

		loaded := getPartition(t, r, idA)

		// This should unset the force at field.
		require.Empty(t, loaded.ForceAtMS)
	})

	t.Run("It returns a partition not found error if deleted", func(t *testing.T) {
		err := q.Dequeue(ctx, p, qi)
		require.NoError(t, err)
		err = q.PartitionRequeue(ctx, &p, time.Now().Add(time.Minute), false)
		require.Equal(t, ErrPartitionGarbageCollected, err)
		err = q.PartitionRequeue(ctx, &p, time.Now().Add(time.Minute), false)
		require.Equal(t, ErrPartitionNotFound, err)
	})

	t.Run("Requeueing a paused partition does not affect the partition's pause state", func(t *testing.T) {
		_, err := q.EnqueueItem(ctx, QueueItem{WorkflowID: idA}, now)
		require.NoError(t, err)

		_, err = q.PartitionLease(ctx, &QueuePartition{WorkflowID: idA}, time.Minute)
		require.NoError(t, err)

		err = q.SetFunctionPaused(ctx, idA, true)
		require.NoError(t, err)

		err = q.PartitionRequeue(ctx, &p, next, true)
		require.NoError(t, err)

		loaded := getPartition(t, r, idA)
		require.True(t, loaded.Paused)
	})
}

func TestQueuePartitionPause(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	q := NewQueue(
		NewQueueClient(rc, QueueDefaultKey),
		WithPriorityFinder(func(ctx context.Context, item QueueItem) uint {
			return PriorityDefault
		}),
	)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	idA := uuid.New()
	_, err = q.EnqueueItem(ctx, QueueItem{WorkflowID: idA}, now)
	require.NoError(t, err)

	err = q.SetFunctionPaused(ctx, idA, true)
	require.NoError(t, err)

	loaded := getPartition(t, r, idA)
	require.True(t, loaded.Paused)

	err = q.SetFunctionPaused(ctx, idA, false)
	require.NoError(t, err)

	loaded = getPartition(t, r, idA)
	require.False(t, loaded.Paused)
}

func TestQueuePartitionReprioritize(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	idA := uuid.New()

	priority := PriorityMin
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	defer rc.Close()
	q := NewQueue(
		NewQueueClient(rc, QueueDefaultKey),
		WithPriorityFinder(func(ctx context.Context, item QueueItem) uint {
			return priority
		}),
	)
	ctx := context.Background()

	_, err = q.EnqueueItem(ctx, QueueItem{WorkflowID: idA}, now)
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

	t.Run("Changing priority does not affect the partition's pause state", func(t *testing.T) {
		err = q.SetFunctionPaused(ctx, idA, true)
		require.NoError(t, err)

		err = q.PartitionReprioritize(ctx, idA.String(), PriorityDefault)
		require.NoError(t, err)

		second := getPartition(t, r, idA)
		require.True(t, second.Paused)
	})
}

func TestQueueRequeueByJobID(t *testing.T) {
	ctx := context.Background()
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	q := queue{
		u: NewQueueClient(rc, QueueDefaultKey),
		pf: func(ctx context.Context, item QueueItem) uint {
			return PriorityMin
		},
		partitionConcurrencyGen: func(ctx context.Context, p QueuePartition) (string, int) {
			return p.Queue(), 100
		},
		itemIndexer: QueueItemIndexerFunc,
		clock:       clockwork.NewRealClock(),
	}

	wsA, wsB := uuid.New(), uuid.New()

	t.Run("Failure cases", func(t *testing.T) {

		t.Run("It fails with a non-existent partition and job ID", func(t *testing.T) {
			err := q.RequeueByJobID(ctx, "foo", "bar", time.Now().Add(5*time.Second))
			require.NotNil(t, err)
		})

		t.Run("It fails with a non-existent job ID for an existing partition", func(t *testing.T) {
			r.FlushDB()

			jid := "yeee"
			item := QueueItem{
				ID:          jid,
				WorkflowID:  wsA,
				WorkspaceID: wsA,
			}
			_, err := q.EnqueueItem(ctx, item, time.Now().Add(time.Second))
			require.NoError(t, err)

			err = q.RequeueByJobID(ctx, wsA.String(), "no bruv", time.Now().Add(5*time.Second))
			require.NotNil(t, err)
		})

		t.Run("It fails with a non-existent partition but an existing job ID", func(t *testing.T) {
			r.FlushDB()

			jid := "another"
			item := QueueItem{
				ID:          jid,
				WorkflowID:  wsA,
				WorkspaceID: wsA,
			}

			_, err := q.EnqueueItem(ctx, item, time.Now().Add(time.Second))
			require.NoError(t, err)

			err = q.RequeueByJobID(ctx, wsB.String(), jid, time.Now().Add(5*time.Second))
			require.NotNil(t, err)
		})

		t.Run("It fails if the job is leased", func(t *testing.T) {
			r.FlushDB()

			jid := "leased"
			item := QueueItem{
				ID:          jid,
				WorkflowID:  wsA,
				WorkspaceID: wsA,
			}

			item, err := q.EnqueueItem(ctx, item, time.Now().Add(time.Second))
			require.NoError(t, err)

			partitions, err := q.PartitionPeek(ctx, true, time.Now().Add(5*time.Second), 10)
			require.NoError(t, err)
			require.Equal(t, 1, len(partitions))

			// Lease
			lid, err := q.Lease(ctx, *partitions[0], item, time.Second*10, time.Now(), nil)
			require.NoError(t, err)
			require.NotNil(t, lid)

			err = q.RequeueByJobID(ctx, wsB.String(), jid, time.Now().Add(5*time.Second))
			require.NotNil(t, err)
		})
	})

	t.Run("It requeues the job", func(t *testing.T) {
		r.FlushDB()

		jid := "requeue-plz"
		at := time.Now().Add(time.Second).Truncate(time.Millisecond)
		item := QueueItem{
			ID:          jid,
			WorkflowID:  wsA,
			WorkspaceID: wsA,
			AtMS:        at.UnixMilli(),
		}
		item, err := q.EnqueueItem(ctx, item, at)
		require.Equal(t, time.UnixMilli(item.WallTimeMS), at)
		require.NoError(t, err)

		// Find all functions
		parts, err := q.PartitionPeek(ctx, true, at.Add(time.Hour), 10)
		require.NoError(t, err)
		require.Equal(t, 1, len(parts))

		// Requeue the function for 5 seconds in the future.
		next := at.Add(5 * time.Second)
		err = q.RequeueByJobID(ctx, wsA.String(), jid, next)
		require.Nil(t, err, r.Dump())

		t.Run("It updates the queue's At time", func(t *testing.T) {
			found, err := q.Peek(ctx, wsA.String(), at.Add(10*time.Second), 5)
			require.NoError(t, err)
			require.Equal(t, 1, len(found))
			require.NotEqual(t, item.AtMS, found[0].AtMS)
			require.Equal(t, next.UnixMilli(), found[0].AtMS)

			require.Equal(t, time.UnixMilli(found[0].WallTimeMS), next)
		})

		t.Run("Requeueing updates the fn's score in the global partition index", func(t *testing.T) {
			// We've already requeued the item, for 5 seconds in the future.
			// The function pointer in the global queue should be 5 seconds ahead.
			fnPtrsAfterRequeue, err := q.PartitionPeek(ctx, true, at.Add(time.Hour), 10)
			require.NoError(t, err)
			require.Equal(t, 1, len(fnPtrsAfterRequeue))

			score, err := r.ZScore(q.u.kg.GlobalPartitionIndex(), wsA.String())
			require.NoError(t, err)

			// The score should have updated.
			require.EqualValues(t, next.Unix(), int64(score), r.Dump())
		})
	})

	t.Run("It requeues the 5th job to a later time", func(t *testing.T) {
		r.FlushDB()

		at := time.Now()
		for i := 0; i < 4; i++ {
			next := at.Add(time.Duration(i) * time.Second)
			item := QueueItem{
				WorkflowID:  wsA,
				WorkspaceID: wsA,
				AtMS:        next.UnixMilli(),
			}
			_, err := q.EnqueueItem(ctx, item, next)
			require.NoError(t, err)
		}

		target := time.Now().Add(10 * time.Second)
		jid := "requeue-plz"
		item := QueueItem{
			ID:          jid,
			WorkflowID:  wsA,
			WorkspaceID: wsA,
			AtMS:        target.UnixMilli(),
		}
		_, err := q.EnqueueItem(ctx, item, target)
		require.NoError(t, err)

		parts, err := q.PartitionPeek(ctx, true, at.Add(time.Hour), 10)
		require.NoError(t, err)
		require.Equal(t, 1, len(parts))

		t.Run("The earliest time is 'at' for the partition", func(t *testing.T) {
			score, err := r.ZScore(q.u.kg.GlobalPartitionIndex(), wsA.String())
			require.NoError(t, err)
			require.EqualValues(t, at.Unix(), int64(score), r.Dump())
		})

		next := target.Add(5 * time.Second)
		err = q.RequeueByJobID(ctx, wsA.String(), jid, next)
		require.Nil(t, err, r.Dump())

		t.Run("The earliest time is still 'at' for the partition after requeueing", func(t *testing.T) {
			score, err := r.ZScore(q.u.kg.GlobalPartitionIndex(), wsA.String())
			require.NoError(t, err)
			require.EqualValues(t, at.Unix(), int64(score), r.Dump())
		})

		t.Run("It updates the queue's At time", func(t *testing.T) {
			found, err := q.Peek(ctx, wsA.String(), at.Add(30*time.Second), 5)
			require.NoError(t, err)
			require.Equal(t, 5, len(found))
			require.Equal(t, at.UnixMilli(), found[0].AtMS, "First job shouldn't change")
			require.Equal(t, target.Add(5*time.Second).UnixMilli(), found[4].AtMS, "Target job didnt change")
		})
	})

	t.Run("It requeues the 1st job to a later time", func(t *testing.T) {
		r.FlushDB()

		at := time.Now().Add(10 * time.Second)
		for i := 0; i < 4; i++ {
			next := at.Add(time.Duration(i) * time.Second)
			item := QueueItem{
				WorkflowID:  wsA,
				WorkspaceID: wsA,
				AtMS:        next.UnixMilli(),
			}
			_, err := q.EnqueueItem(ctx, item, next)
			require.NoError(t, err)
		}

		target := time.Now().Add(1 * time.Second)
		jid := "requeue-plz"
		item := QueueItem{
			ID:          jid,
			WorkflowID:  wsA,
			WorkspaceID: wsA,
			AtMS:        target.UnixMilli(),
		}
		_, err := q.EnqueueItem(ctx, item, target)
		require.NoError(t, err)

		parts, err := q.PartitionPeek(ctx, true, at.Add(time.Hour), 10)
		require.NoError(t, err)
		require.Equal(t, 1, len(parts))

		t.Run("The earliest time is 'target' for the partition", func(t *testing.T) {
			score, err := r.ZScore(q.u.kg.GlobalPartitionIndex(), wsA.String())
			require.NoError(t, err)
			require.EqualValues(t, target.Unix(), int64(score), r.Dump())
		})

		next := target.Add(5 * time.Second)
		err = q.RequeueByJobID(ctx, wsA.String(), jid, next)
		require.Nil(t, err, r.Dump())

		t.Run("The earliest time is 'next' for the partition after requeueing", func(t *testing.T) {
			score, err := r.ZScore(q.u.kg.GlobalPartitionIndex(), wsA.String())
			require.NoError(t, err)
			require.EqualValues(t, next.Unix(), int64(score), r.Dump())
		})
	})
}

func TestQueueLeaseSequential(t *testing.T) {
	ctx := context.Background()
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	q := queue{
		u: NewQueueClient(rc, QueueDefaultKey),
		pf: func(ctx context.Context, item QueueItem) uint {
			return PriorityMin
		},
		clock: clockwork.NewRealClock(),
	}

	var (
		leaseID *ulid.ULID
	)

	t.Run("It claims sequential leases", func(t *testing.T) {
		now := time.Now()
		dur := 500 * time.Millisecond
		leaseID, err = q.ConfigLease(ctx, q.u.kg.Sequential(), dur)
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		require.WithinDuration(t, now.Add(dur), ulid.Time(leaseID.Time()), 5*time.Millisecond)
	})

	t.Run("It doesn't allow leasing without an existing lease ID", func(t *testing.T) {
		id, err := q.ConfigLease(ctx, q.u.kg.Sequential(), time.Second)
		require.Equal(t, ErrConfigAlreadyLeased, err)
		require.Nil(t, id)
	})

	t.Run("It doesn't allow leasing with an invalid lease ID", func(t *testing.T) {
		newULID := ulid.MustNew(ulid.Now(), rnd)
		id, err := q.ConfigLease(ctx, q.u.kg.Sequential(), time.Second, &newULID)
		require.Equal(t, ErrConfigAlreadyLeased, err)
		require.Nil(t, id)
	})

	t.Run("It extends the lease with a valid lease ID", func(t *testing.T) {
		require.NotNil(t, leaseID)

		now := time.Now()
		dur := 50 * time.Millisecond
		leaseID, err = q.ConfigLease(ctx, q.u.kg.Sequential(), dur, leaseID)
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		require.WithinDuration(t, now.Add(dur), ulid.Time(leaseID.Time()), 5*time.Millisecond)
	})

	t.Run("It allows leasing when the current lease is expired", func(t *testing.T) {
		<-time.After(100 * time.Millisecond)

		now := time.Now()
		dur := 50 * time.Millisecond
		leaseID, err = q.ConfigLease(ctx, q.u.kg.Sequential(), dur)
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		require.WithinDuration(t, now.Add(dur), ulid.Time(leaseID.Time()), 5*time.Millisecond)
	})
}

// TestSharding covers the basics of shards;  we assert that function enqueues/dequeues/leasing
// modify shards appropriately, and that partition opeartions also modify the shards.
func TestSharding(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()
	ctx := context.Background()

	shouldShard := true // indicate whether to shard in tests
	shard := &QueueShard{
		Name:               "sharded",
		Priority:           0,
		GuaranteedCapacity: 1,
	}
	sf := func(ctx context.Context, queueName string, wsID uuid.UUID) *QueueShard {
		if !shouldShard {
			return nil
		}
		return shard
	}
	q := NewQueue(NewQueueClient(rc, QueueDefaultKey), WithShardFinder(sf))
	require.NotNil(t, sf(ctx, "", uuid.UUID{}))

	t.Run("QueueItem which shards", func(t *testing.T) {

		// NOTE: Times for shards or global pointers cannot be <= now.
		// Because of this, tests start with the earliest item 1 hour ahead of now so that
		// we can appropriately test enqueueing earlier items adjust pointer times.

		t.Run("Basic enqueue lease dequeue operations", func(t *testing.T) {
			at := time.Now().Truncate(time.Second).Add(time.Hour)
			item, err := q.EnqueueItem(ctx, QueueItem{
				ID: "foo",
			}, at)
			require.NoError(t, err, "sharded enqueue should succeed")
			// The partition, or function queue, for the just-enqueued item.
			p := QueuePartition{WorkflowID: item.WorkflowID, WorkspaceID: item.WorkspaceID}

			t.Run("Enqueueing creates a shard in the shard map", func(t *testing.T) {
				keys, err := r.HKeys(q.u.kg.Shards())
				require.NoError(t, err)
				require.Equal(t, 1, len(keys))

				shardJSON := r.HGet(q.u.kg.Shards(), shard.Name)
				actual := &QueueShard{}
				err = json.Unmarshal([]byte(shardJSON), actual)
				require.NoError(t, err)
				require.EqualValues(t, *shard, *actual)
			})

			t.Run("items exist in the shard partition", func(t *testing.T) {
				ptrs, err := r.ZMembers(q.u.kg.ShardPartitionIndex(shard.Name))
				require.NoError(t, err)
				require.EqualValues(t, 1, len(ptrs))
				// TODO: Ensure ID matches
			})

			t.Run("enqueueing another item in the same shard doesn't duplicate shards or items", func(t *testing.T) {
				_, err := q.EnqueueItem(ctx, QueueItem{}, at.Add(time.Minute))
				require.NoError(t, err)

				keys, err := r.HKeys(q.u.kg.Shards())
				require.NoError(t, err)
				require.Equal(t, 1, len(keys))

				shardJSON := r.HGet(q.u.kg.Shards(), "sharded")
				actual := &QueueShard{}
				err = json.Unmarshal([]byte(shardJSON), actual)
				require.NoError(t, err)
				require.EqualValues(t, *shard, *actual)

				ptrs, err := r.ZMembers(q.u.kg.ShardPartitionIndex(shard.Name))
				require.NoError(t, err)
				require.EqualValues(t, 1, len(ptrs))
			})

			t.Run("leasing the earliest queue item modifies the shard partition", func(t *testing.T) {
				// Check shard partition score changed in the ptr.
				score, err := r.ZScore(q.u.kg.ShardPartitionIndex(shard.Name), p.Queue())
				require.NoError(t, err)
				require.EqualValues(t, at.Unix(), score, "starting score should be enqueue time")

				_, err = q.Lease(ctx, p, item, 5*time.Second, time.Now(), nil)
				require.NoError(t, err)

				// Check shard partition score changed in the ptr.
				nextScore, err := r.ZScore(q.u.kg.ShardPartitionIndex(shard.Name), p.Queue())
				require.NoError(t, err)
				// This is the score of the second item in the queue
				require.EqualValues(t, at.Add(time.Minute).Unix(), int(nextScore), "leasing should use next queue item's score in shard ptr")
			})

			t.Run("requeue modifies the shard partition", func(t *testing.T) {
				err := q.Requeue(ctx, p, item, at.Add(30*time.Second))
				require.NoError(t, err)

				// Check shard partition score changed in the ptr.
				nextScore, err := r.ZScore(q.u.kg.ShardPartitionIndex(shard.Name), p.Queue())
				require.NoError(t, err)
				require.EqualValues(t, at.Add(30*time.Second).Unix(), nextScore, "requeued score should increase")
			})

			t.Run("requeue by job ID modifies the shard partition", func(t *testing.T) {
				err := q.RequeueByJobID(ctx, p.Queue(), "foo", at.Add(45*time.Second))
				require.NoError(t, err)

				// Check shard partition score changed in the ptr.
				nextScore, err := r.ZScore(q.u.kg.ShardPartitionIndex(shard.Name), p.Queue())
				require.NoError(t, err)
				require.EqualValues(t, at.Add(45*time.Second).Unix(), nextScore, "requeued score should increase")
			})

			// NOTE: Dequeue doesn't need to do anything:  a leased job already removes the
			// item from the fn queue and updates the shard pointer;  dequeueing operates on in-progress
			// queues only.

			t.Run("enqueueing earlier items changes the pointer in the shard partition", func(t *testing.T) {
				// enqueue a new item an hour ago
				earlier := at.Add(-1 * time.Hour)
				_, err := q.EnqueueItem(ctx, QueueItem{}, earlier)
				require.NoError(t, err)

				// Check shard partition score changed in the ptr.
				nextScore, err := r.ZScore(q.u.kg.ShardPartitionIndex(shard.Name), p.Queue())
				nextTime := time.Unix(int64(nextScore), 0)
				require.NoError(t, err)
				require.EqualValues(t, earlier.Unix(), nextTime.Unix(), "enqueueing earlier score should rescore")
			})
		})

		t.Run("shards are updated when enqueueing, if already exists", func(t *testing.T) {
			shardJSON := r.HGet(q.u.kg.Shards(), shard.Name)
			first := &QueueShard{}
			err = json.Unmarshal([]byte(shardJSON), first)
			require.NoError(t, err)
			require.EqualValues(t, *shard, *first)

			// Enqueue again with a capacity of 1
			shard.GuaranteedCapacity = shard.GuaranteedCapacity + 1
			_, err = q.EnqueueItem(ctx, QueueItem{}, time.Now())

			shardJSON = r.HGet(q.u.kg.Shards(), shard.Name)
			updated := &QueueShard{}
			err = json.Unmarshal([]byte(shardJSON), updated)
			require.NoError(t, err)
			require.NotEqualValues(t, *first, *updated)
			require.EqualValues(t, *shard, *updated)
		})
	})

	r.FlushAll() // Reset queue.

	t.Run("partitions/function queues", func(t *testing.T) {
		at := time.Now().Truncate(time.Second).Add(time.Second)
		item, err := q.EnqueueItem(ctx, QueueItem{}, at)
		require.NoError(t, err, "sharded enqueue should succeed")
		// The partition, or function queue, for the just-enqueued item.
		p := QueuePartition{WorkflowID: item.WorkflowID, WorkspaceID: item.WorkspaceID}

		t.Run("leasing a partition changes the partition's shard pointer", func(t *testing.T) {
			// The score should be "At" to begin with.
			shardScore, err := r.ZScore(q.u.kg.ShardPartitionIndex(shard.Name), p.Queue())
			shardTime := time.Unix(int64(shardScore), 0)
			require.NoError(t, err)
			require.EqualValues(t, at.Unix(), shardTime.Unix())

			// Lease the function queue for a minute
			_, err = q.PartitionLease(ctx, &p, time.Minute)
			require.NoError(t, err)

			leasedShardScore, err := r.ZScore(q.u.kg.ShardPartitionIndex(shard.Name), p.Queue())
			require.NoError(t, err)
			leasedShardTime := time.Unix(int64(leasedShardScore), 0)

			require.NoError(t, err)
			require.NotEqualValues(t, at.Unix(), leasedShardTime.Unix(), "leasing should update partition score")
			require.WithinDuration(t, at.Add(time.Minute), leasedShardTime, time.Second, "leasing should update partition score")
		})

		t.Run("requeueing a partition changes the partition's shard pointer", func(t *testing.T) {
			// The score not be at - sanity check
			shardScore, err := r.ZScore(q.u.kg.ShardPartitionIndex(shard.Name), p.Queue())
			shardTime := time.Unix(int64(shardScore), 0)
			require.NoError(t, err)
			require.NotEqualValues(t, at.Unix(), shardTime.Unix())

			// Lease the function queue for a minute
			err = q.PartitionRequeue(ctx, &p, at, false)
			require.NoError(t, err)

			// The score should reset to "At"
			shardScore, err = r.ZScore(q.u.kg.ShardPartitionIndex(shard.Name), p.Queue())
			shardTime = time.Unix(int64(shardScore), 0)
			require.NoError(t, err)
			require.EqualValues(t, at.Unix(), shardTime.Unix())

			t.Run("partitions with no items are GCd from the shard during requeue", func(t *testing.T) {
				err := q.Dequeue(ctx, p, item)
				require.NoError(t, err)

				// Lease the function queue for a minute
				err = q.PartitionRequeue(ctx, &p, at, false)
				require.EqualError(t, err, ErrPartitionGarbageCollected.Error())
			})
		})
	})

	r.FlushAll() // Reset queue.

	t.Run("QueueItem which does not shard", func(t *testing.T) {
		t.Run("enqueueing does not modify shards", func(t *testing.T) {
			shouldShard = false

			at := time.Now().Truncate(time.Second).Add(time.Hour)
			_, err := q.EnqueueItem(ctx, QueueItem{}, at)
			require.NoError(t, err, "sharded enqueue should succeed")

			keys, err := r.HKeys(q.u.kg.Shards())
			require.Equal(t, miniredis.ErrKeyNotFound, err)
			require.Equal(t, 0, len(keys))
		})
	})
}

func TestShardLease(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()
	ctx := context.Background()

	sf := func(ctx context.Context, queueName string, wsID uuid.UUID) *QueueShard {
		return &QueueShard{
			Name:               wsID.String(),
			Priority:           0,
			GuaranteedCapacity: 1,
		}
	}
	q := NewQueue(NewQueueClient(rc, QueueDefaultKey), WithShardFinder(sf))

	t.Run("Leasing a non-existent shard fails", func(t *testing.T) {
		shard := sf(ctx, "", uuid.UUID{})
		leaseID, err := q.leaseShard(ctx, shard, 2*time.Second, 1)
		require.Nil(t, leaseID, "Got lease ID: %v", leaseID)
		require.NotNil(t, err)
		require.ErrorContains(t, err, "shard not found")
	})

	// Ensure shards exist
	idA, idB := uuid.New(), uuid.New()
	_, err = q.EnqueueItem(ctx, QueueItem{WorkspaceID: idA}, time.Now())
	require.NoError(t, err)
	_, err = q.EnqueueItem(ctx, QueueItem{WorkspaceID: idB}, time.Now())
	require.NoError(t, err)

	miniredis.DumpMaxLineLen = 1024

	t.Run("Leasing out-of-bounds fails", func(t *testing.T) {
		// At the beginning, no shards have been leased.  Leasing a shard
		// with an index of >= 1 should fail.
		shard := sf(ctx, "", idA)
		leaseID, err := q.leaseShard(ctx, shard, 2*time.Second, 1)
		require.Nil(t, leaseID, "Got lease ID: %v", leaseID)
		require.NotNil(t, err)
		require.ErrorContains(t, err, "lease index is too high")
	})

	t.Run("Leasing a shard works", func(t *testing.T) {
		shard := sf(ctx, "", idA)

		t.Run("Basic lease", func(t *testing.T) {
			leaseID, err := q.leaseShard(ctx, shard, 1*time.Second, 0)
			require.NotNil(t, leaseID, "Didn't get a lease ID for a basic lease")
			require.Nil(t, err)
		})

		t.Run("Leasing a subsequent index works", func(t *testing.T) {
			leaseID, err := q.leaseShard(ctx, shard, 8*time.Second, 1) // Same length as the lease below, after wait
			require.NotNil(t, leaseID, "Didn't get a lease ID for a secondary lease")
			require.Nil(t, err)
		})

		t.Run("Leasing an index with an expired lease works", func(t *testing.T) {
			// In this test, we have two leases — but one expires with the wait.  This first lease
			// is no longer valid, so leasing with an index of (1) should succeed.
			<-time.After(2 * time.Second) // Wait a few seconds so that time.Now() in the call works.
			r.FastForward(2 * time.Second)
			leaseID, err := q.leaseShard(ctx, shard, 10*time.Second, 1)
			require.NotNil(t, leaseID)
			require.Nil(t, err)

			// This leaves us with two valid leases.
		})

		t.Run("Leasing an already leased index fails", func(t *testing.T) {
			leaseID, err := q.leaseShard(ctx, shard, 2*time.Second, 1)
			require.Nil(t, leaseID, "got a lease ID for an existing lease")
			require.NotNil(t, err)
			require.ErrorContains(t, err, "index is already leased")
		})

		t.Run("Leasing a second shard works", func(t *testing.T) {
			// Try another shard name with an index of 0.
			leaseID, err := q.leaseShard(ctx, sf(ctx, "", idB), 2*time.Second, 0)
			require.NotNil(t, leaseID)
			require.Nil(t, err)
		})
	})

	r.FlushAll()

	t.Run("Renewing shard leases", func(t *testing.T) {
		// Ensure that enqueueing succeeds to make the shard.
		_, err = q.EnqueueItem(ctx, QueueItem{WorkspaceID: idA}, time.Now())
		require.Nil(t, err)

		shard := sf(ctx, "", idA)
		leaseID, err := q.leaseShard(ctx, shard, 1*time.Second, 0)
		require.NotNil(t, leaseID, "could not lease shard")
		require.Nil(t, err)

		t.Run("Current leases succeed", func(t *testing.T) {
			leaseID, err = q.renewShardLease(ctx, shard, 2*time.Second, *leaseID)
			require.NotNil(t, leaseID, "did not get a new lease when renewing")
			require.Nil(t, err)
		})

		t.Run("Expired leases fail", func(t *testing.T) {
			<-time.After(3 * time.Second)
			r.FastForward(3 * time.Second)

			leaseID, err := q.renewShardLease(ctx, shard, 2*time.Second, *leaseID)
			require.ErrorContains(t, err, "lease not found")
			require.Nil(t, leaseID)
		})

		t.Run("Invalid lease IDs fail", func(t *testing.T) {
			leaseID, err := q.renewShardLease(ctx, shard, 2*time.Second, ulid.MustNew(ulid.Now(), rand.Reader))
			require.ErrorContains(t, err, "lease not found")
			require.Nil(t, leaseID)
		})
	})
}

// TestQueueRateLimit asserts that the queue respects rate limits when added to a queue item.
func TestQueueRateLimit(t *testing.T) {
	mr := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{mr.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()
	ctx := context.Background()
	clock := clockwork.NewFakeClock()
	q := NewQueue(NewQueueClient(rc, QueueDefaultKey), WithClock(clock))

	idA, idB := uuid.New(), uuid.New()

	r := require.New(t)

	t.Run("Without bursts", func(t *testing.T) {
		throttle := &osqueue.Throttle{
			Key:    "some-key",
			Limit:  1,
			Period: 5, // Admit one every 5 seconds
			Burst:  0, // No burst.
		}

		aa, err := q.EnqueueItem(ctx, QueueItem{
			WorkflowID: idA,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					WorkflowID: idA,
				},
				Throttle: throttle,
			},
		}, clock.Now())
		r.NoError(err)

		ab, err := q.EnqueueItem(ctx, QueueItem{
			WorkflowID: idA,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					WorkflowID: idA,
				},
				Throttle: throttle,
			},
		}, clock.Now().Add(time.Second))
		r.NoError(err)

		// Leasing A should succeed, then B should fail.
		partitions, err := q.PartitionPeek(ctx, true, clock.Now().Add(5*time.Second), 5)
		r.NoError(err)
		r.EqualValues(1, len(partitions))

		// clock.Advance(10 * time.Millisecond)

		t.Run("Leasing a first item succeeds", func(t *testing.T) {
			leaseA, err := q.Lease(ctx, *partitions[0], aa, 10*time.Second, clock.Now(), nil)
			r.NoError(err, "leasing throttled queue item with capacity failed")
			r.NotNil(leaseA)
		})

		// clock.Advance(10 * time.Millisecond)

		t.Run("Attempting to lease another throttled key immediately fails", func(t *testing.T) {
			leaseB, err := q.Lease(ctx, *partitions[0], ab, 10*time.Second, clock.Now(), nil)
			r.NotNil(err, "leasing throttled queue item without capacity didn't error")
			r.Nil(leaseB)
		})

		// clock.Advance(10 * time.Millisecond)

		t.Run("Leasing another function succeeds", func(t *testing.T) {
			ba, err := q.EnqueueItem(ctx, QueueItem{
				WorkflowID: idB,
				Data: osqueue.Item{
					Identifier: state.Identifier{
						WorkflowID: idB,
					},
					Throttle: &osqueue.Throttle{
						Key:    "another-key",
						Limit:  1,
						Period: 5, // Admit one every 5 seconds
						Burst:  0, // No burst.
					},
				},
			}, clock.Now().Add(time.Second))
			r.NoError(err)
			lease, err := q.Lease(ctx, *partitions[0], ba, 10*time.Second, clock.Now(), nil)
			r.Nil(err, "leasing throttled queue item without capacity didn't error")
			r.NotNil(lease)
		})

		// clock.Advance(10 * time.Millisecond)

		t.Run("Leasing after the period succeeds", func(t *testing.T) {
			clock.Advance(time.Duration(throttle.Period)*time.Second + time.Second)

			leaseB, err := q.Lease(ctx, *partitions[0], ab, 10*time.Second, clock.Now(), nil)
			r.Nil(err, "leasing after waiting for throttle should succeed")
			r.NotNil(leaseB)
		})
	})

	mr.FlushAll()
	clock.Advance(10 * time.Second)

	t.Run("With bursts", func(t *testing.T) {
		throttle := &osqueue.Throttle{
			Key:    "burst-plz",
			Limit:  1,
			Period: 10, // Admit one every 10 seconds
			Burst:  3,  // With bursts of 3
		}

		items := []QueueItem{}
		for i := 0; i <= 20; i++ {
			item, err := q.EnqueueItem(ctx, QueueItem{
				WorkflowID: idA,
				Data: osqueue.Item{
					Identifier: state.Identifier{WorkflowID: idA},
					Throttle:   throttle,
				},
			}, clock.Now())
			clock.Advance(1 * time.Millisecond)
			r.NoError(err)
			items = append(items, item)
		}

		// Leasing A should succeed, then B should fail.
		partitions, err := q.PartitionPeek(ctx, true, clock.Now().Add(5*time.Second), 5)
		r.NoError(err)
		r.EqualValues(1, len(partitions))

		idx := 0

		t.Run("Leasing up to bursts succeeds", func(t *testing.T) {
			for i := 0; i < 3; i++ {
				lease, err := q.Lease(ctx, *partitions[0], items[i], 2*time.Second, clock.Now(), nil)
				r.NoError(err, "leasing throttled queue item with capacity failed")
				r.NotNil(lease)
				idx++
			}
		})

		t.Run("Leasing the 4th time fails", func(t *testing.T) {
			lease, err := q.Lease(ctx, *partitions[0], items[idx], 1*time.Second, clock.Now(), nil)
			r.NotNil(err, "leasing throttled queue item without capacity didn't error")
			r.ErrorContains(err, ErrQueueItemThrottled.Error())
			r.Nil(lease)
		})

		t.Run("After 10s, we can re-lease once as bursting is done.", func(t *testing.T) {
			clock.Advance(time.Duration(throttle.Period)*time.Second + time.Second)

			lease, err := q.Lease(ctx, *partitions[0], items[idx], 2*time.Second, clock.Now(), nil)
			r.NoError(err, "leasing throttled queue item with capacity failed")
			r.NotNil(lease)

			idx++

			// It should fail, as bursting is done.
			lease, err = q.Lease(ctx, *partitions[0], items[idx], 1*time.Second, clock.Now(), nil)
			r.NotNil(err, "leasing throttled queue item without capacity didn't error")
			r.ErrorContains(err, ErrQueueItemThrottled.Error())
			r.Nil(lease)
		})

		t.Run("After another 40s, we can burst again", func(t *testing.T) {
			clock.Advance(time.Duration(throttle.Period*4) * time.Second)

			for i := 0; i < 3; i++ {
				lease, err := q.Lease(ctx, *partitions[0], items[i], 2*time.Second, clock.Now(), nil)
				r.NoError(err, "leasing throttled queue item with capacity failed")
				r.NotNil(lease)
				idx++
			}
		})
	})
}

func getQueueItem(t *testing.T, r *miniredis.Miniredis, id string) QueueItem {
	t.Helper()
	kg := &queueKeyGenerator{
		queueDefaultKey: QueueDefaultKey,
		queueItemKeyGenerator: queueItemKeyGenerator{
			queueDefaultKey: QueueDefaultKey,
		},
	}
	// Ensure that our data is set up correctly.
	val := r.HGet(kg.QueueItem(), id)
	require.NotEmpty(t, val)
	i := QueueItem{}
	err := json.Unmarshal([]byte(val), &i)
	i.Data.JobID = &i.ID
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
	kg := &queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
	val := r.HGet(kg.PartitionItem(), id.String())
	qp := QueuePartition{}
	err := json.Unmarshal([]byte(val), &qp)
	require.NoError(t, err)
	return qp
}

func requireItemScoreEquals(t *testing.T, r *miniredis.Miniredis, item QueueItem, expected time.Time) {
	t.Helper()
	kg := &queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
	score, err := r.ZScore(kg.QueueIndex(item.WorkflowID.String()), item.ID)
	parsed := time.UnixMilli(int64(score))
	require.NoError(t, err)
	require.WithinDuration(t, expected.Truncate(time.Millisecond), parsed, 15*time.Millisecond)
}

func requirePartitionScoreEquals(t *testing.T, r *miniredis.Miniredis, wid uuid.UUID, expected time.Time) {
	t.Helper()
	kg := &queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
	score, err := r.ZScore(kg.GlobalPartitionIndex(), wid.String())
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

func TestCheckList(t *testing.T) {
	checks := []struct {
		Check    string
		Expected bool
		Exact    map[string]*struct{}
		Prefix   map[string]*struct{}
	}{
		{
			// with no prefix or match
			"user-created",
			false,
			map[string]*struct{}{"something-else": nil},
			map[string]*struct{}{"user:*": nil},
		},
		{
			// with exact match
			"user-created",
			true,
			map[string]*struct{}{"user-created": nil},
			nil,
		},
		{
			// with prefix
			"user-created",
			true,
			nil,
			map[string]*struct{}{"user": nil},
		},
	}

	for _, item := range checks {
		actual := checkList(item.Check, item.Exact, item.Prefix)
		require.Equal(t, item.Expected, actual)
	}
}

func int64ptr(i int64) *int64 { return &i }
