package redis_state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/stretchr/testify/assert"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/util"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func init() {
	miniredis.DumpMaxLineLen = 1024
}

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

	accountId := uuid.New()

	t.Run("It enqueues an item", func(t *testing.T) {
		id := uuid.New()

		item, err := q.EnqueueItem(ctx, QueueItem{
			FunctionID: id,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					AccountID: accountId,
				},
			},
		}, start)
		require.NoError(t, err)
		require.NotEqual(t, item.ID, ulid.ULID{})
		require.Equal(t, time.UnixMilli(item.WallTimeMS).Truncate(time.Second), start)

		// Ensure that our data is set up correctly.
		found := getQueueItem(t, r, item.ID)
		require.Equal(t, item, found)

		// Ensure the partition is inserted.
		qp := getDefaultPartition(t, r, item.FunctionID)
		require.Equal(t, accountId.String(), qp.AccountID.String())
		require.Equal(t, QueuePartition{
			ID:               item.FunctionID.String(),
			FunctionID:       &item.FunctionID,
			AccountID:        accountId,
			ConcurrencyLimit: consts.DefaultConcurrencyLimit,
		}, qp)

		// Ensure the account is inserted
		accountIds := getGlobalAccounts(t, rc)
		require.Contains(t, accountIds, accountId.String())

		// Ensure the partition is inserted in account partitions
		partitionIds := getAccountPartitions(t, rc, accountId)
		require.Contains(t, partitionIds, qp.ID)

		// Score of partition in global + account partition indexes should match
		kg := &queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		requirePartitionItemScoreEquals(t, r, kg.GlobalPartitionIndex(), qp, start)
		requirePartitionItemScoreEquals(t, r, kg.AccountPartitionIndex(accountId), qp, start)
		requireAccountScoreEquals(t, r, accountId, start)
	})

	t.Run("It sets the right item score", func(t *testing.T) {
		start := time.Now()

		item, err := q.EnqueueItem(ctx, QueueItem{}, start)
		require.NoError(t, err)

		requireItemScoreEquals(t, r, item, start)
	})

	t.Run("It enqueues an item in the future", func(t *testing.T) {
		// Empty the DB.
		r.FlushAll()

		at := time.Now().Add(time.Hour).Truncate(time.Second)

		item, err := q.EnqueueItem(ctx, QueueItem{
			Data: osqueue.Item{
				Identifier: state.Identifier{
					AccountID: accountId,
				},
			},
		}, at)
		require.NoError(t, err)

		// Ensure the partition is inserted, and the earliest time is still
		// the start time.
		qp := getDefaultPartition(t, r, item.FunctionID)
		require.Equal(t, QueuePartition{
			ID:               item.FunctionID.String(),
			FunctionID:       &item.FunctionID,
			AccountID:        accountId,
			ConcurrencyLimit: consts.DefaultConcurrencyLimit,
		}, qp)

		// Ensure that the zscore did not change.
		keys, err := r.ZMembers(q.u.kg.GlobalPartitionIndex())
		require.NoError(t, err)
		require.Equal(t, 1, len(keys))

		score, err := r.ZScore(q.u.kg.GlobalPartitionIndex(), keys[0])
		require.NoError(t, err)
		require.EqualValues(t, at.Unix(), score)

		score, err = r.ZScore(q.u.kg.AccountPartitionIndex(accountId), keys[0])
		require.NoError(t, err)
		require.EqualValues(t, at.Unix(), score)

		score, err = r.ZScore(q.u.kg.GlobalAccountIndex(), accountId.String())
		require.NoError(t, err)
		require.EqualValues(t, at.Unix(), score)
	})

	t.Run("Updates partition vesting time to earlier times", func(t *testing.T) {
		now := time.Now()
		at := now.Add(-10 * time.Minute).Truncate(time.Second)

		// Note: This will reuse the existing partition (zero UUID) from the step above
		item, err := q.EnqueueItem(ctx, QueueItem{
			Data: osqueue.Item{
				Identifier: state.Identifier{
					AccountID: accountId,
				},
			},
		}, at)
		require.NoError(t, err)

		// Ensure the partition is inserted, and the earliest time is updated
		// inside the partition item.
		qp := getDefaultPartition(t, r, item.FunctionID)
		require.Equal(t, QueuePartition{
			ID:               item.FunctionID.String(),
			FunctionID:       &item.FunctionID,
			AccountID:        accountId,
			ConcurrencyLimit: consts.DefaultConcurrencyLimit,
		}, qp, "queue partition does not match")

		// Assert that the zscore was changed to this earliest timestamp.
		keys, err := r.ZMembers(q.u.kg.GlobalPartitionIndex())
		require.NoError(t, err)
		require.Equal(t, 1, len(keys))

		score, err := r.ZScore(q.u.kg.GlobalPartitionIndex(), keys[0])
		require.NoError(t, err)
		require.EqualValues(t, now.Unix(), score)

		score, err = r.ZScore(q.u.kg.AccountPartitionIndex(accountId), keys[0])
		require.NoError(t, err)
		require.NotZero(t, score)
		require.EqualValues(t, now.Unix(), score, r.Dump())

		score, err = r.ZScore(q.u.kg.GlobalAccountIndex(), accountId.String())
		require.NoError(t, err)
		require.EqualValues(t, now.Unix(), score)
	})

	t.Run("Adding another workflow ID increases partition set", func(t *testing.T) {
		at := time.Now().Truncate(time.Second)

		accountId := uuid.New()

		item, err := q.EnqueueItem(ctx, QueueItem{
			FunctionID: uuid.New(),
			Data: osqueue.Item{
				Identifier: state.Identifier{
					AccountID: accountId,
				},
			},
		}, at)
		require.NoError(t, err)

		// Assert that we have two zscores in partition:sorted.
		keys, err := r.ZMembers(q.u.kg.GlobalPartitionIndex())
		require.NoError(t, err)
		require.Equal(t, 2, len(keys))

		// Assert that we have one zscore in accounts:$accountId:partition:sorted.
		keys, err = r.ZMembers(q.u.kg.AccountPartitionIndex(accountId))
		require.NoError(t, err)
		require.Equal(t, 1, len(keys))

		// Ensure the partition is inserted, and the earliest time is updated
		// inside the partition item.
		qp := getDefaultPartition(t, r, item.FunctionID)
		require.Equal(t, QueuePartition{
			ID:               item.FunctionID.String(),
			FunctionID:       &item.FunctionID,
			AccountID:        accountId,
			ConcurrencyLimit: consts.DefaultConcurrencyLimit,
		}, qp)
	})

	t.Run("Stores default indexes", func(t *testing.T) {
		at := time.Now().Truncate(time.Second)
		rid := ulid.MustNew(ulid.Now(), rand.Reader)
		_, err := q.EnqueueItem(ctx, QueueItem{
			FunctionID: uuid.New(),
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
			FunctionID: workflowId,
		}, now.Add(10*time.Second))
		require.NoError(t, err)

		err = q.SetFunctionPaused(ctx, item.FunctionID, true)
		require.NoError(t, err)

		item, err = q.EnqueueItem(ctx, QueueItem{
			FunctionID: workflowId,
		}, now)
		require.NoError(t, err)

		fnMeta := getFnMetadata(t, r, item.FunctionID)
		require.True(t, fnMeta.Paused)

		item, err = q.EnqueueItem(ctx, QueueItem{
			FunctionID: workflowId,
		}, now.Add(-10*time.Second))
		require.NoError(t, err)

		fnMeta = getFnMetadata(t, r, item.FunctionID)
		require.True(t, fnMeta.Paused)
	})

	t.Run("Custom concurrency key queues", func(t *testing.T) {
		now := time.Now()
		fnID := uuid.New()

		r.FlushAll()

		t.Run("Single custom key, function scope", func(t *testing.T) {
			// Enqueueing an item
			ck := createConcurrencyKey(enums.ConcurrencyScopeFn, fnID, "test", 1)
			_, _, hash, _ := ck.ParseKey() // get the hash of the "test" string / evaluated input.

			qi := QueueItem{
				FunctionID: fnID,
				Data: osqueue.Item{
					CustomConcurrencyKeys: []state.CustomConcurrency{ck},
					Identifier: state.Identifier{
						AccountID: accountId,
					},
				},
			}

			actualItemPartions := q.ItemPartitions(ctx, qi)
			assert.Equal(t, 3, len(actualItemPartions))

			customkeyQueuePartition := QueuePartition{
				ID:               q.u.kg.PartitionQueueSet(enums.PartitionTypeConcurrencyKey, fnID.String(), hash),
				PartitionType:    int(enums.PartitionTypeConcurrencyKey),
				ConcurrencyScope: int(enums.ConcurrencyScopeFn),
				FunctionID:       &fnID,
				AccountID:        accountId,
				ConcurrencyLimit: 1,
				ConcurrencyKey:   ck.Key,
				ConcurrencyHash:  ck.Hash,
			}

			assert.Equal(t, customkeyQueuePartition, actualItemPartions[0])

			i, err := q.EnqueueItem(ctx, qi, now.Add(10*time.Second))
			require.NoError(t, err)

			// There should be 2 partitions - custom key, and the function
			// level limit.
			items, _ := r.HKeys(q.u.kg.PartitionItem())
			require.Equal(t, 2, len(items))

			concurrencyPartition := getPartition(t, r, enums.PartitionTypeConcurrencyKey, fnID, hash) // nb. also asserts that the partition exists
			require.Equal(t, customkeyQueuePartition, concurrencyPartition)

			accountIds := getGlobalAccounts(t, rc)
			require.Equal(t, 1, len(accountIds))
			require.Contains(t, accountIds, accountId.String())

			apIds := getAccountPartitions(t, rc, accountId)
			require.Equal(t, 2, len(apIds), "expected two account partitions", apIds, r.Dump())

			// concurrency key partition
			require.Contains(t, apIds, concurrencyPartition.ID)

			// workflow partition for backwards compatibility
			require.Contains(t, apIds, fnID.String())

			// We enqueue to the function-specific queue for backwards-compatibility reasons
			defaultPartition := getDefaultPartition(t, r, fnID)
			assert.Equal(t, QueuePartition{
				ID:               fnID.String(),
				FunctionID:       &fnID,
				AccountID:        accountId,
				ConcurrencyLimit: consts.DefaultConcurrencyLimit,
			}, defaultPartition)

			mem, err := r.ZMembers(defaultPartition.zsetKey(q.u.kg))
			require.NoError(t, err)
			require.Equal(t, 1, len(mem))
			require.Contains(t, mem, i.ID)
		})

		t.Run("Two keys, function scope", func(t *testing.T) {
			r.FlushAll()

			// Enqueueing an item
			ckA := createConcurrencyKey(enums.ConcurrencyScopeFn, fnID, "test", 1)
			_, _, hashA, _ := ckA.ParseKey() // get the hash of the "test" string / evaluated input.

			ckB := createConcurrencyKey(enums.ConcurrencyScopeFn, fnID, "plz", 2)
			_, _, hashB, _ := ckB.ParseKey() // get the hash of the "test" string / evaluated input.

			qi := QueueItem{
				FunctionID: fnID,
				Data: osqueue.Item{
					CustomConcurrencyKeys: []state.CustomConcurrency{ckA, ckB},
					Identifier: state.Identifier{
						AccountID: accountId,
					}},
			}

			actualItemPartitions := q.ItemPartitions(ctx, qi)
			assert.Equal(t, 3, len(actualItemPartitions))
			keyQueueA := QueuePartition{
				ID:               q.u.kg.PartitionQueueSet(enums.PartitionTypeConcurrencyKey, fnID.String(), hashA),
				PartitionType:    int(enums.PartitionTypeConcurrencyKey),
				ConcurrencyScope: int(enums.ConcurrencyScopeFn),
				FunctionID:       &fnID,
				AccountID:        accountId,
				ConcurrencyLimit: 1,
				ConcurrencyKey:   ckA.Key,
				ConcurrencyHash:  ckA.Hash,
			}
			assert.Equal(t, keyQueueA, actualItemPartitions[0])

			keyQueueB := QueuePartition{
				ID:               q.u.kg.PartitionQueueSet(enums.PartitionTypeConcurrencyKey, fnID.String(), hashB),
				PartitionType:    int(enums.PartitionTypeConcurrencyKey),
				ConcurrencyScope: int(enums.ConcurrencyScopeFn),
				FunctionID:       &fnID,
				AccountID:        accountId,
				ConcurrencyLimit: 2,
				ConcurrencyKey:   ckB.Key,
				ConcurrencyHash:  ckB.Hash,
			}
			assert.Equal(t, keyQueueB, actualItemPartitions[1])

			// We enqueue to the function-specific queue for backwards-compatibility reasons, but
			// we don't return it from ItemPartitions as it's a special case with extra rules for leasing, etc.
			assert.Equal(t, QueuePartition{}, actualItemPartitions[2])

			expectedDefaultPartition := QueuePartition{
				ID:               fnID.String(),
				FunctionID:       &fnID,
				AccountID:        accountId,
				ConcurrencyLimit: consts.DefaultConcurrencyLimit,
			}
			legacyPartition := q.functionPartition(ctx, actualItemPartitions, qi)
			assert.Equal(t, expectedDefaultPartition, legacyPartition)

			i, err := q.EnqueueItem(ctx, qi, now.Add(10*time.Second))
			require.NoError(t, err)

			// 3 partitions (2 custom concurrency keys + 1 default)
			items, _ := r.HKeys(q.u.kg.PartitionItem())
			require.Equal(t, 3, len(items))

			concurrencyPartitionA := getPartition(t, r, enums.PartitionTypeConcurrencyKey, fnID, hashA) // nb. also asserts that the partition exists
			require.Equal(t, keyQueueA, concurrencyPartitionA)

			concurrencyPartitionB := getPartition(t, r, enums.PartitionTypeConcurrencyKey, fnID, hashB) // nb. also asserts that the partition exists
			require.Equal(t, keyQueueB, concurrencyPartitionB)

			accountIds := getGlobalAccounts(t, rc)
			require.Equal(t, 1, len(accountIds))
			require.Contains(t, accountIds, accountId.String())

			apIds := getAccountPartitions(t, rc, accountId)
			require.Equal(t, 3, len(apIds))
			require.Contains(t, apIds, concurrencyPartitionA.ID)
			require.Contains(t, apIds, concurrencyPartitionB.ID)

			require.Contains(t, apIds, expectedDefaultPartition.ID)

			assert.True(t, r.Exists(expectedDefaultPartition.zsetKey(q.u.kg)), "expected default partition to exist")
			defaultPartition := getDefaultPartition(t, r, fnID)
			assert.Equal(t, expectedDefaultPartition, defaultPartition)

			mem, err := r.ZMembers(defaultPartition.zsetKey(q.u.kg))
			require.NoError(t, err)
			require.Equal(t, 1, len(mem))
			require.Contains(t, mem, i.ID)

			t.Run("Peeking partitions returns the three partitions", func(t *testing.T) {
				parts, err := q.PartitionPeek(ctx, true, time.Now().Add(time.Hour), 10)
				require.NoError(t, err)
				require.Equal(t, 3, len(parts))
				require.Equal(t, expectedDefaultPartition, *parts[0], "Got: %v", spew.Sdump(parts), r.Dump())
				require.Equal(t, concurrencyPartitionA, *parts[1], "Got: %v", spew.Sdump(parts), r.Dump())
				require.Equal(t, concurrencyPartitionB, *parts[2], "Got: %v", spew.Sdump(parts), r.Dump())
			})
		})
	})

	t.Run("Migrates old partitions to add accountId", func(t *testing.T) {
		r.FlushAll()

		id := uuid.MustParse("baac957a-3aa5-4e42-8c1d-f86dee5d58da")
		envId := uuid.MustParse("e8c0aacd-fcb4-4d5a-b78a-7f0528841543")

		oldPartitionSnapshot := "{\"at\":1723814830,\"p\":6,\"wsID\":\"e8c0aacd-fcb4-4d5a-b78a-7f0528841543\",\"wid\":\"baac957a-3aa5-4e42-8c1d-f86dee5d58da\",\"last\":1723814800026,\"forceAtMS\":0,\"off\":false}"

		r.HSet(q.u.kg.PartitionItem(), id.String(), oldPartitionSnapshot)
		assert.Equal(t, QueuePartition{
			FunctionID: &id,
			EnvID:      &envId,
			// No accountId is present,
			AccountID: uuid.UUID{},
			LeaseID:   nil,
			Last:      1723814800026,
		}, getPartition(t, r, enums.PartitionTypeDefault, id))

		item, err := q.EnqueueItem(ctx, QueueItem{
			FunctionID: id,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					AccountID: accountId,
				},
			},
		}, start)
		require.NoError(t, err)
		require.NotEqual(t, item.ID, ulid.ULID{})
		require.Equal(t, time.UnixMilli(item.WallTimeMS).Truncate(time.Second), start)

		assert.Equal(t, QueuePartition{
			FunctionID: &id,
			EnvID:      &envId,
			// No accountId is present,
			AccountID: accountId,
			LeaseID:   nil,
			Last:      1723814800026,
		}, getPartition(t, r, enums.PartitionTypeDefault, id), r.Dump())
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
		p := QueuePartition{FunctionID: &item.FunctionID}

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
			_, err := q.EnqueueItem(ctx, QueueItem{FunctionID: id}, time.Now())
			if err != nil {
				panic(err)
			}
		}
	}

	for i := 0; i < b.N; i++ {
		id := uuid.New()
		enqueue(id, int(QueuePeekMax))
		items, err := q.Peek(ctx, &QueuePartition{FunctionID: &id}, time.Now(), QueuePeekMax)
		if err != nil {
			panic(err)
		}
		if len(items) != int(QueuePeekMax) {
			panic(fmt.Sprintf("expected %d, got %d", QueuePeekMax, len(items)))
		}
	}
}

func TestQueueSystemPartitions(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	customQueueName := "custom"
	customTestLimit := 1

	q := NewQueue(
		NewQueueClient(rc, QueueDefaultKey),
		WithAllowQueueNames(customQueueName),
		WithSystemConcurrencyLimitGetter(
			func(ctx context.Context, p QueuePartition) int {
				return customTestLimit
			}),
		WithConcurrencyLimitGetter(func(ctx context.Context, p QueuePartition) PartitionConcurrencyLimits {
			return PartitionConcurrencyLimits{5000, 5000, 5000}
		}),
	)
	ctx := context.Background()

	start := time.Now().Truncate(time.Second)

	id := uuid.New()

	qi := QueueItem{
		FunctionID: id,
		Data: osqueue.Item{
			Payload:   json.RawMessage("{\"test\":\"payload\"}"),
			QueueName: &customQueueName,
		},
		QueueName: &customQueueName,
	}

	t.Run("It enqueues an item", func(t *testing.T) {
		item, err := q.EnqueueItem(ctx, qi, start)
		require.NoError(t, err)
		require.NotEqual(t, item.ID, ulid.ULID{})
		require.Equal(t, time.UnixMilli(item.WallTimeMS).Truncate(time.Second), start)

		// Ensure that our data is set up correctly.
		found := getQueueItem(t, r, item.ID)
		require.Equal(t, item, found)

		// Ensure the partition is inserted.
		qp := getSystemPartition(t, r, customQueueName)
		require.Equal(t, QueuePartition{
			ID:               customQueueName,
			PartitionType:    int(enums.PartitionTypeDefault),
			QueueName:        &customQueueName,
			ConcurrencyLimit: customTestLimit,
		}, qp)

		apIds := getAccountPartitions(t, rc, uuid.Nil)
		require.Empty(t, apIds)
		require.NotContains(t, apIds, qp.ID)
	})

	t.Run("peeks correct partition", func(t *testing.T) {
		qp := getSystemPartition(t, r, customQueueName)

		partitions, err := q.PartitionPeek(ctx, true, start, 100)
		require.NoError(t, err)
		require.Equal(t, 1, len(partitions))
		require.Equal(t, qp, *partitions[0])

		items, err := q.Peek(ctx, &qp, start, 100)
		require.NoError(t, err)
		require.Equal(t, 1, len(items))
	})

	t.Run("leases correct partition", func(t *testing.T) {
		qp := getSystemPartition(t, r, customQueueName)

		leaseId, availableCapacity, err := q.PartitionLease(ctx, &qp, time.Second)
		require.NoError(t, err)
		require.NotNil(t, leaseId)
		require.Equal(t, 5000, availableCapacity)
	})

	t.Run("peeks partition successfully", func(t *testing.T) {
		qp := getSystemPartition(t, r, customQueueName)

		items, err := q.Peek(ctx, &qp, start, 100)
		require.NoError(t, err)
		require.Equal(t, 1, len(items))
		require.Equal(t, qi.Data.Payload, items[0].Data.Payload)
	})

	t.Run("leases partition items while respecting concurrency", func(t *testing.T) {
		qp := getSystemPartition(t, r, customQueueName)

		item, err := q.EnqueueItem(ctx, qi, start)
		require.NoError(t, err)
		require.NotEqual(t, item.ID, ulid.ULID{})
		require.Equal(t, time.UnixMilli(item.WallTimeMS).Truncate(time.Second), start)

		item2, err := q.EnqueueItem(ctx, qi, start)
		require.NoError(t, err)
		require.NotEqual(t, item.ID, ulid.ULID{})
		require.Equal(t, time.UnixMilli(item.WallTimeMS).Truncate(time.Second), start)

		// Ensure that our data is set up correctly.
		found := getQueueItem(t, r, item.ID)
		require.Equal(t, item, found)

		leaseId, err := q.Lease(ctx, qp, item, time.Second, time.Now(), nil)
		require.NoError(t, err)
		require.NotNil(t, leaseId)

		leaseId, err = q.Lease(ctx, qp, item2, time.Second, time.Now(), nil)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrSystemConcurrencyLimit)
		require.Nil(t, leaseId)
	})

	t.Run("scavenges partition items with expired leases", func(t *testing.T) {
		// wait til leases are expired
		<-time.After(2 * time.Second)

		requeued, err := q.Scavenge(ctx, ScavengePeekSize)
		require.NoError(t, err)
		assert.Equal(t, 1, requeued, "expected one item with expired leases to be requeued by scavenge", r.Dump())
	})

	t.Run("backcompat: scavenges previous partition items with expired leases", func(t *testing.T) {
		r.FlushAll()

		start := time.Now().Truncate(time.Second)

		item, err := q.EnqueueItem(ctx, qi, start)
		require.NoError(t, err)
		require.NotEqual(t, item.ID, ulid.ULID{})
		require.Equal(t, time.UnixMilli(item.WallTimeMS).Truncate(time.Second), start)

		qp := getSystemPartition(t, r, customQueueName)

		leaseStart := time.Now()
		leaseExpires := q.clock.Now().Add(time.Second)

		itemCountMatches := func(num int) {
			zsetKey := qp.zsetKey(q.u.kg)
			items, err := rc.Do(ctx, rc.B().
				Zrangebyscore().
				Key(zsetKey).
				Min("-inf").
				Max("+inf").
				Build()).AsStrSlice()
			require.NoError(t, err)
			assert.Equal(t, num, len(items), "expected %d items in the queue %q", num, zsetKey, r.Dump())
		}

		concurrencyItemCountMatches := func(num int) {
			items, err := rc.Do(ctx, rc.B().
				Zrangebyscore().
				Key(qp.concurrencyKey(q.u.kg)).
				Min("-inf").
				Max("+inf").
				Build()).AsStrSlice()
			require.NoError(t, err)
			assert.Equal(t, num, len(items), "expected %d items in the concurrency queue", num, r.Dump())
		}

		itemCountMatches(1)
		concurrencyItemCountMatches(0)

		leaseId, err := q.Lease(ctx, qp, item, time.Second, leaseStart, nil)
		require.NoError(t, err)
		require.NotNil(t, leaseId)

		itemCountMatches(0)
		concurrencyItemCountMatches(1)

		// wait til leases are expired
		<-time.After(2 * time.Second)
		require.True(t, time.Now().After(leaseExpires))

		newConcurrencyIndexItem := q.u.kg.Concurrency("p", customQueueName)
		oldConcurrencyIndexItem := customQueueName

		removed, err := rc.Do(ctx, rc.B().Zrem().Key(q.u.kg.ConcurrencyIndex()).Member(newConcurrencyIndexItem).Build()).AsInt64()
		require.NoError(t, err)
		assert.Equal(t, int64(1), removed, "expected one previous item to be removed")

		err = rc.Do(ctx, rc.B().Zadd().Key(q.u.kg.ConcurrencyIndex()).ScoreMember().ScoreMember(float64(leaseExpires.UnixMilli()), oldConcurrencyIndexItem).Build()).Error()
		require.NoError(t, err)

		requeued, err := q.Scavenge(ctx, ScavengePeekSize)
		require.NoError(t, err)
		assert.Equal(t, 1, requeued, "expected one item with expired leases to be requeued by scavenge", r.Dump())

		itemCountMatches(1)
		concurrencyItemCountMatches(0)

		indexItems, err := rc.Do(ctx, rc.B().Zcard().Key(q.u.kg.ConcurrencyIndex()).Build()).AsInt64()
		require.NoError(t, err)
		assert.Equal(t, 0, int(indexItems), "expected no items in the concurrency index", r.Dump())

		newConcurrencyQueueItems, err := rc.Do(ctx, rc.B().Zcard().Key(newConcurrencyIndexItem).Build()).AsInt64()
		require.NoError(t, err)
		assert.Equal(t, 0, int(newConcurrencyQueueItems), "expected no items in the new concurrency queue", r.Dump())

		oldConcurrencyQueueItems, err := rc.Do(ctx, rc.B().Zcard().Key(oldConcurrencyIndexItem).Build()).AsInt64()
		require.NoError(t, err)
		assert.Equal(t, 0, int(oldConcurrencyQueueItems), "expected no items in the old concurrency queue", r.Dump())
	})

	t.Run("It enqueues an item to account queues when account id is present", func(t *testing.T) {
		r.FlushAll()

		start := time.Now().Truncate(time.Second)

		// This test case handles account-scoped system partitions

		accountId := uuid.New()

		qi := QueueItem{
			FunctionID: id,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					AccountID: accountId,
				},
				Payload:   json.RawMessage("{\"test\":\"payload\"}"),
				QueueName: &customQueueName,
			},
			QueueName: &customQueueName,
		}

		item, err := q.EnqueueItem(ctx, qi, start)
		require.NoError(t, err)
		require.NotEqual(t, item.ID, ulid.ULID{})
		require.Equal(t, time.UnixMilli(item.WallTimeMS).Truncate(time.Second), start)

		// Ensure that our data is set up correctly.
		found := getQueueItem(t, r, item.ID)
		require.Equal(t, item, found)

		// Ensure the partition is inserted.
		qp := getSystemPartition(t, r, customQueueName)
		require.Equal(t, QueuePartition{
			ID:               customQueueName,
			QueueName:        &customQueueName,
			PartitionType:    int(enums.PartitionTypeDefault),
			ConcurrencyLimit: customTestLimit,
			// We do not store the accountId for system partitions
			AccountID: uuid.Nil,
		}, qp)

		apIds := getAccountPartitions(t, rc, accountId)
		require.Equal(t, 1, len(apIds))
		require.Contains(t, apIds, qp.ID)
	})

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
		items, err := q.Peek(ctx, &QueuePartition{FunctionID: &workflowID}, time.Now().Add(time.Hour), 10)
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

		items, err := q.Peek(ctx, &QueuePartition{FunctionID: &workflowID}, time.Now().Add(time.Hour), 10)
		require.NoError(t, err)
		require.EqualValues(t, 3, len(items))
		require.EqualValues(t, []*QueueItem{&ia, &ib, &ic}, items)
		require.NotEqualValues(t, []*QueueItem{&ib, &ia, &ic}, items)

		id, err := q.EnqueueItem(ctx, QueueItem{ID: "d"}, d)
		require.NoError(t, err)

		items, err = q.Peek(ctx, &QueuePartition{FunctionID: &workflowID}, time.Now().Add(time.Hour), 10)
		require.NoError(t, err)
		require.EqualValues(t, 4, len(items))
		require.EqualValues(t, []*QueueItem{&ia, &ib, &ic, &id}, items)

		t.Run("It should limit the list", func(t *testing.T) {
			items, err = q.Peek(ctx, &QueuePartition{FunctionID: &workflowID}, time.Now().Add(time.Hour), 2)
			require.NoError(t, err)
			require.EqualValues(t, 2, len(items))
			require.EqualValues(t, []*QueueItem{&ia, &ib}, items)
		})

		t.Run("It should apply a peek offset", func(t *testing.T) {
			items, err = q.Peek(ctx, &QueuePartition{FunctionID: &workflowID}, time.Now().Add(-1*time.Hour), QueuePeekMax)
			require.NoError(t, err)
			require.EqualValues(t, 0, len(items))

			items, err = q.Peek(ctx, &QueuePartition{FunctionID: &workflowID}, c, QueuePeekMax)
			require.NoError(t, err)
			require.EqualValues(t, 3, len(items))
			require.EqualValues(t, []*QueueItem{&ia, &ib, &ic}, items)
		})

		t.Run("It should remove any leased items from the list", func(t *testing.T) {
			p := QueuePartition{FunctionID: &ia.FunctionID}

			// Lease step A, and it should be removed.
			_, err := q.Lease(ctx, p, ia, 50*time.Millisecond, time.Now(), nil)
			require.NoError(t, err)

			items, err = q.Peek(ctx, &QueuePartition{FunctionID: &workflowID}, d, QueuePeekMax)
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
			require.EqualValues(t, 1, caught, "Items not found during scavenge\n%s", r.Dump())

			items, err = q.Peek(ctx, &QueuePartition{FunctionID: &workflowID}, d, QueuePeekMax)
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
	q := NewQueue(queueClient)
	defaultQueueKey := q.u.kg

	ctx := context.Background()

	start := time.Now().Truncate(time.Second)

	t.Run("It leases an item", func(t *testing.T) {
		item, err := q.EnqueueItem(ctx, QueueItem{}, start)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		nilUUID := uuid.UUID{}
		p := QueuePartition{
			FunctionID: &nilUUID,
		} // Default workflow ID etc

		t.Run("It should exist in the pending partition queue", func(t *testing.T) {
			mem, err := r.ZMembers(p.zsetKey(q.u.kg))
			require.NoError(t, err)
			require.Equal(t, 1, len(mem))
		})

		now := time.Now()
		id, err := q.Lease(ctx, p, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, id, item.LeaseID)
		require.WithinDuration(t, now.Add(time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		t.Run("It should remove from the pending partition queue", func(t *testing.T) {
			mem, _ := r.ZMembers(p.zsetKey(q.u.kg))
			require.Empty(t, mem)
		})

		t.Run("It should add the item to the function's in-progress concurrency queue", func(t *testing.T) {
			count, err := q.InProgress(ctx, "p", uuid.UUID{}.String())
			require.NoError(t, err)
			require.EqualValues(t, 1, count, r.Dump())
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
				count, err := q.InProgress(ctx, "p", p.FunctionID.String())
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
				count, err := q.InProgress(ctx, "p", p.FunctionID.String())
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

			_, err = r.ZScore(q.u.kg.FnQueueSet(item.FunctionID.String()), item.ID)
			require.Error(t, err, "no such key")
		})

		t.Run("it should update the partition score to the next item", func(t *testing.T) {
			r.FlushAll()

			timeNow := time.Now().Truncate(time.Second)
			timeNowPlusFiveSeconds := timeNow.Add(time.Second * 5).Truncate(time.Second)

			acctId := uuid.New()

			// Enqueue future item (partition time will be now + 5s)
			item, err = q.EnqueueItem(ctx, QueueItem{
				Data: osqueue.Item{Identifier: state.Identifier{AccountID: acctId}},
			}, timeNowPlusFiveSeconds)
			require.NoError(t, err)
			require.Nil(t, item.LeaseID)

			qp := getDefaultPartition(t, r, uuid.Nil)

			requireItemScoreEquals(t, r, item, timeNowPlusFiveSeconds)
			requirePartitionItemScoreEquals(t, r, q.u.kg.GlobalPartitionIndex(), qp, timeNowPlusFiveSeconds)
			requirePartitionItemScoreEquals(t, r, q.u.kg.AccountPartitionIndex(acctId), qp, timeNowPlusFiveSeconds)
			requireAccountScoreEquals(t, r, acctId, timeNowPlusFiveSeconds)

			// Enqueue current item (partition time will be moved up to now)
			item, err := q.EnqueueItem(ctx, QueueItem{Data: osqueue.Item{Identifier: state.Identifier{AccountID: acctId}}}, timeNow)
			require.NoError(t, err)
			require.Nil(t, item.LeaseID)

			requireItemScoreEquals(t, r, item, timeNow)

			requirePartitionItemScoreEquals(t, r, q.u.kg.GlobalPartitionIndex(), qp, timeNow)
			requirePartitionItemScoreEquals(t, r, q.u.kg.AccountPartitionIndex(acctId), qp, timeNow)
			requireAccountScoreEquals(t, r, acctId, timeNow)

			// Lease item (moves partition time back to now + 5s)
			_, err = q.Lease(ctx, p, item, time.Minute, q.clock.Now(), nil)
			require.NoError(t, err)

			requirePartitionItemScoreEquals(t, r, q.u.kg.GlobalPartitionIndex(), qp, timeNowPlusFiveSeconds)
			requirePartitionItemScoreEquals(t, r, q.u.kg.AccountPartitionIndex(acctId), qp, timeNowPlusFiveSeconds)
			requireAccountScoreEquals(t, r, acctId, timeNowPlusFiveSeconds)
		})
	})

	// Test default partition-level concurrency limits (not custom)
	t.Run("With partition concurrency limits", func(t *testing.T) {
		r.FlushAll()

		// Only allow a single leased item
		q.concurrencyLimitGetter = func(ctx context.Context, p QueuePartition) PartitionConcurrencyLimits {
			return PartitionConcurrencyLimits{1, 1, 1}
		}

		fnID := uuid.New()
		// Create a new item
		itemA, err := q.EnqueueItem(ctx, QueueItem{FunctionID: fnID}, start)
		require.NoError(t, err)
		itemB, err := q.EnqueueItem(ctx, QueueItem{FunctionID: fnID}, start)
		require.NoError(t, err)
		// Use the new item's workflow ID
		p := QueuePartition{ID: itemA.FunctionID.String(), FunctionID: &itemA.FunctionID}

		t.Run("With denylists it does not lease.", func(t *testing.T) {
			list := newLeaseDenyList()
			list.addConcurrency(newKeyError(ErrPartitionConcurrencyLimit, p.Queue()))
			id, err := q.Lease(ctx, p, itemA, 5*time.Second, time.Now(), list)
			require.NotNil(t, err, "Expcted error leasing denylists")
			require.Nil(t, id, "Expected nil ID with denylists")
			require.ErrorIs(t, err, ErrPartitionConcurrencyLimit)
		})

		t.Run("Leases with capacity", func(t *testing.T) {
			_, err = q.Lease(ctx, p, itemA, 5*time.Second, time.Now(), nil)
			require.NoError(t, err)
		})

		t.Run("Errors without capacity", func(t *testing.T) {
			id, err := q.Lease(ctx, p, itemB, 5*time.Second, time.Now(), nil)
			require.Nil(t, id, "Leased item when concurrency limits are reached.\n%s", r.Dump())
			require.Error(t, err)
		})
	})

	// Test default account concurrency limits (not custom)
	t.Run("With account concurrency limits", func(t *testing.T) {
		r.FlushAll()

		// Only allow a single leased item via account limits
		q.concurrencyLimitGetter = func(ctx context.Context, p QueuePartition) PartitionConcurrencyLimits {
			return PartitionConcurrencyLimits{
				AccountLimit:   1,
				FunctionLimit:  NoConcurrencyLimit,
				CustomKeyLimit: NoConcurrencyLimit,
			}
		}

		acctId := uuid.New()

		// Create a new item
		itemA, err := q.EnqueueItem(ctx, QueueItem{FunctionID: uuid.New(), Data: osqueue.Item{Identifier: state.Identifier{AccountID: acctId}}}, start)
		require.NoError(t, err)
		itemB, err := q.EnqueueItem(ctx, QueueItem{FunctionID: uuid.New(), Data: osqueue.Item{Identifier: state.Identifier{AccountID: acctId}}}, start)
		require.NoError(t, err)
		// Use the new item's workflow ID
		p := QueuePartition{AccountID: acctId, FunctionID: &itemA.FunctionID}

		t.Run("Leases with capacity", func(t *testing.T) {
			_, err = q.Lease(ctx, p, itemA, 5*time.Second, time.Now(), nil)
			require.NoError(t, err)
		})

		t.Run("Errors without capacity", func(t *testing.T) {
			id, err := q.Lease(ctx, p, itemB, 5*time.Second, time.Now(), nil)
			require.Nil(t, id)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrAccountConcurrencyLimit)
		})
	})

	t.Run("With custom concurrency limits", func(t *testing.T) {
		t.Run("with account keys", func(t *testing.T) {
			r.FlushAll()
			// Only allow a single leased item via custom concurrency limits
			q.concurrencyLimitGetter = func(ctx context.Context, p QueuePartition) PartitionConcurrencyLimits {
				return PartitionConcurrencyLimits{
					AccountLimit:   NoConcurrencyLimit,
					FunctionLimit:  NoConcurrencyLimit,
					CustomKeyLimit: 1,
				}
			}

			ck := createConcurrencyKey(enums.ConcurrencyScopeAccount, uuid.Nil, "foo", 1)

			// Create a new item
			itemA, err := q.EnqueueItem(ctx, QueueItem{
				FunctionID: uuid.New(),
				Data: osqueue.Item{
					CustomConcurrencyKeys: []state.CustomConcurrency{
						{
							Key:   ck.Key,
							Limit: 1,
						},
					},
				},
			}, start)
			require.NoError(t, err)

			itemB, err := q.EnqueueItem(ctx, QueueItem{
				FunctionID: uuid.New(),
				Data: osqueue.Item{
					CustomConcurrencyKeys: []state.CustomConcurrency{
						{
							Key:   ck.Key,
							Limit: 1,
						},
					},
				},
			}, start)
			require.NoError(t, err)

			// Use the new item's workflow ID
			p := QueuePartition{FunctionID: &itemA.FunctionID}

			t.Run("With denylists it does not lease.", func(t *testing.T) {
				list := newLeaseDenyList()
				list.addConcurrency(newKeyError(ErrConcurrencyLimitCustomKey, ck.Key))
				_, err = q.Lease(ctx, p, itemA, 5*time.Second, time.Now(), list)
				require.NotNil(t, err)
				require.ErrorIs(t, err, ErrConcurrencyLimitCustomKey)
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

		t.Run("with function keys", func(t *testing.T) {
			r.FlushAll()

			accountId := uuid.New()
			fnId := uuid.New()

			// Only allow a single leased item via custom concurrency limits
			q.concurrencyLimitGetter = func(ctx context.Context, p QueuePartition) PartitionConcurrencyLimits {
				return PartitionConcurrencyLimits{
					AccountLimit:   NoConcurrencyLimit,
					FunctionLimit:  NoConcurrencyLimit,
					CustomKeyLimit: 1,
				}
			}

			ck := createConcurrencyKey(enums.ConcurrencyScopeFn, fnId, "foo", 1)
			_, _, keyExprChecksum, err := ck.ParseKey()
			require.NoError(t, err)

			// Create a new item
			itemA, err := q.EnqueueItem(ctx, QueueItem{
				FunctionID: fnId,
				Data: osqueue.Item{
					CustomConcurrencyKeys: []state.CustomConcurrency{
						{
							Key:   ck.Key,
							Limit: 1,
						},
					},
					Identifier: state.Identifier{
						AccountID: accountId,
					},
				},
			}, start)
			require.NoError(t, err)

			itemB, err := q.EnqueueItem(ctx, QueueItem{
				FunctionID: fnId,
				Data: osqueue.Item{
					CustomConcurrencyKeys: []state.CustomConcurrency{
						{
							Key:   ck.Key,
							Limit: 1,
						},
					},
					Identifier: state.Identifier{
						AccountID: accountId,
					},
				},
			}, start)
			require.NoError(t, err)

			// Use the new item's workflow ID
			p := getPartition(t, r, enums.PartitionTypeConcurrencyKey, fnId, keyExprChecksum)

			t.Run("With denylists it does not lease.", func(t *testing.T) {
				list := newLeaseDenyList()
				list.addConcurrency(newKeyError(ErrConcurrencyLimitCustomKey, ck.Key))
				_, err = q.Lease(ctx, p, itemA, 5*time.Second, time.Now(), list)
				require.NotNil(t, err)
				require.ErrorIs(t, err, ErrConcurrencyLimitCustomKey)
			})

			t.Run("Leases with capacity", func(t *testing.T) {
				// Use the new item's workflow ID
				zsetKeyA := q.u.kg.PartitionQueueSet(enums.PartitionTypeConcurrencyKey, fnId.String(), keyExprChecksum)
				pA := QueuePartition{ID: zsetKeyA, AccountID: accountId, FunctionID: &itemA.FunctionID, PartitionType: int(enums.PartitionTypeConcurrencyKey), ConcurrencyKey: ck.Key, ConcurrencyLimit: 1}
				require.Equal(t, pA.zsetKey(q.u.kg), zsetKeyA)
				require.Equal(t, pA, getPartition(t, r, enums.PartitionTypeConcurrencyKey, fnId, keyExprChecksum))

				memPart, err := r.ZMembers(zsetKeyA)
				require.NoError(t, err)
				require.Equal(t, 2, len(memPart))
				require.Contains(t, memPart, itemA.ID)
				require.Contains(t, memPart, itemB.ID)

				// concurrency key queue does not yet exist
				require.False(t, r.Exists(pA.concurrencyKey(q.u.kg)))

				// partition key queue exists
				require.True(t, r.Exists(zsetKeyA))

				_, err = q.Lease(ctx, p, itemA, 5*time.Second, time.Now(), nil)
				require.NoError(t, err)

				memPart, err = r.ZMembers(zsetKeyA)
				require.NoError(t, err)
				require.Equal(t, 1, len(memPart))
				require.Contains(t, memPart, itemB.ID)

				require.True(t, r.Exists(pA.concurrencyKey(q.u.kg)))
				memConcurrency, err := r.ZMembers(pA.concurrencyKey(q.u.kg))
				require.NoError(t, err)
				require.Equal(t, 1, len(memConcurrency))
				require.Contains(t, memConcurrency, itemA.ID)
			})

			t.Run("Errors without capacity", func(t *testing.T) {
				id, err := q.Lease(ctx, p, itemB, 5*time.Second, time.Now(), nil)
				require.Nil(t, id)
				require.Error(t, err)
				require.ErrorIs(t, err, ErrConcurrencyLimitCustomKey)
			})
		})

		// this test is the unit variant of TestConcurrency_ScopeFunction_FanOut in cloud
		t.Run("with two distinct functions it processes both", func(t *testing.T) {
			r.FlushAll()

			q.concurrencyLimitGetter = func(ctx context.Context, p QueuePartition) PartitionConcurrencyLimits {
				return PartitionConcurrencyLimits{
					FunctionLimit:  1,
					AccountLimit:   123_456,
					CustomKeyLimit: 234_567,
				}
			}

			fnIDA := uuid.New()
			fnIDB := uuid.New()

			ckA := createConcurrencyKey(enums.ConcurrencyScopeFn, fnIDA, "foo", 1)
			_, _, evaluatedKeyChecksumA, err := ckA.ParseKey()
			require.NoError(t, err)

			ckB := createConcurrencyKey(enums.ConcurrencyScopeFn, fnIDB, "foo", 1)
			_, _, evaluatedKeyChecksumB, err := ckB.ParseKey()
			require.NoError(t, err)

			// Create a new item
			itemA1, err := q.EnqueueItem(ctx, QueueItem{FunctionID: fnIDA, Data: osqueue.Item{CustomConcurrencyKeys: []state.CustomConcurrency{ckA}}}, start)
			require.NoError(t, err)
			itemA2, err := q.EnqueueItem(ctx, QueueItem{FunctionID: fnIDA, Data: osqueue.Item{CustomConcurrencyKeys: []state.CustomConcurrency{ckA}}}, start)
			require.NoError(t, err)
			itemB1, err := q.EnqueueItem(ctx, QueueItem{FunctionID: fnIDB, Data: osqueue.Item{CustomConcurrencyKeys: []state.CustomConcurrency{ckB}}}, start)
			require.NoError(t, err)
			itemB2, err := q.EnqueueItem(ctx, QueueItem{FunctionID: fnIDB, Data: osqueue.Item{CustomConcurrencyKeys: []state.CustomConcurrency{ckB}}}, start)
			require.NoError(t, err)

			// Use the new item's workflow ID
			zsetKeyA := q.u.kg.PartitionQueueSet(enums.PartitionTypeConcurrencyKey, fnIDA.String(), evaluatedKeyChecksumA)
			pA := QueuePartition{ID: zsetKeyA, FunctionID: &itemA1.FunctionID, PartitionType: int(enums.PartitionTypeConcurrencyKey), ConcurrencyKey: ckA.Key, ConcurrencyLimit: 1, ConcurrencyHash: ckA.Hash}

			require.Equal(t, pA, getPartition(t, r, enums.PartitionTypeConcurrencyKey, fnIDA, evaluatedKeyChecksumA))

			zsetKeyB := q.u.kg.PartitionQueueSet(enums.PartitionTypeConcurrencyKey, fnIDB.String(), evaluatedKeyChecksumB)
			pB := QueuePartition{ID: zsetKeyB, FunctionID: &itemB1.FunctionID, PartitionType: int(enums.PartitionTypeConcurrencyKey), ConcurrencyKey: ckB.Key, ConcurrencyLimit: 1, ConcurrencyHash: ckB.Hash}
			require.Equal(t, pB, getPartition(t, r, enums.PartitionTypeConcurrencyKey, fnIDB, evaluatedKeyChecksumB))

			// Both key queues exist
			require.True(t, r.Exists(zsetKeyA))
			require.True(t, r.Exists(zsetKeyB))

			// Lease item A1 - should work
			_, err = q.Lease(ctx, pA, itemA1, 5*time.Second, time.Now(), nil)
			require.NoError(t, err)

			// Lease item B1 - should work
			_, err = q.Lease(ctx, pB, itemB1, 5*time.Second, time.Now(), nil)
			require.NoError(t, err)

			// Lease item A2 - should fail due to custom concurrency limit
			_, err = q.Lease(ctx, pA, itemA2, 5*time.Second, time.Now(), nil)
			require.ErrorIs(t, err, ErrConcurrencyLimitCustomKey)

			// Lease item B1 - should fail due to custom concurrency limit
			_, err = q.Lease(ctx, pB, itemB2, 5*time.Second, time.Now(), nil)
			require.ErrorIs(t, err, ErrConcurrencyLimitCustomKey)
		})
	})

	t.Run("It should update the global partition index", func(t *testing.T) {
		t.Run("With no concurrency keys", func(t *testing.T) {
			r.FlushAll()
			q.customConcurrencyLimitRefresher = func(ctx context.Context, i QueueItem) []state.CustomConcurrency {
				return nil
			}

			// NOTE: We need two items to ensure that this updates.  Leasing an
			// item removes it from the fn queue.
			t.Run("With a single item in the queue hwen leasing, nothing updates", func(t *testing.T) {
				at := time.Now().Truncate(time.Second).Add(time.Second)
				accountId := uuid.New()
				item, err := q.EnqueueItem(ctx, QueueItem{
					Data: osqueue.Item{Identifier: state.Identifier{AccountID: accountId}},
				}, at)
				require.NoError(t, err)
				p := QueuePartition{FunctionID: &item.FunctionID}

				score, err := r.ZScore(q.u.kg.GlobalPartitionIndex(), p.Queue())
				require.NoError(t, err)
				require.EqualValues(t, at.Unix(), score, r.Dump())

				score, err = r.ZScore(defaultQueueKey.AccountPartitionIndex(accountId), p.Queue())
				require.NoError(t, err)
				require.EqualValues(t, at.Unix(), score, r.Dump())

				// Nothing should update here, as there's nothing left in the fn queue
				// so nothing happens.
				_, err = q.Lease(ctx, p, item, 10*time.Second, time.Now(), nil)
				require.NoError(t, err)

				nextScore, err := r.ZScore(defaultQueueKey.GlobalPartitionIndex(), p.Queue())
				require.NoError(t, err)
				require.EqualValues(t, int(score), int(nextScore), "score should not equal previous score")

				nextScore, err = r.ZScore(defaultQueueKey.AccountPartitionIndex(accountId), p.Queue())
				require.NoError(t, err)
				require.EqualValues(t, int(score), int(nextScore), "account score should not equal previous score")
			})
		})

		t.Run("With custom concurrency keys", func(t *testing.T) {
			r.FlushAll()

			t.Run("It moves items from each concurrency queue", func(t *testing.T) {
				at := time.Now().Truncate(time.Second).Add(time.Second)
				itemA, err := q.EnqueueItem(ctx, QueueItem{
					Data: osqueue.Item{
						CustomConcurrencyKeys: []state.CustomConcurrency{
							{
								Key: util.ConcurrencyKey(
									enums.ConcurrencyScopeAccount,
									uuid.Nil,
									"acct-id",
								),
								Limit: 10,
							},
							{
								Key: util.ConcurrencyKey(
									enums.ConcurrencyScopeFn,
									uuid.Nil,
									"fn-id",
								),
								Limit: 5,
							},
						},
					},
				}, at)
				require.NoError(t, err)
				itemB, err := q.EnqueueItem(ctx, QueueItem{
					Data: osqueue.Item{
						CustomConcurrencyKeys: []state.CustomConcurrency{
							{
								Key: util.ConcurrencyKey(
									enums.ConcurrencyScopeAccount,
									uuid.Nil,
									"acct-id",
								),
								Limit: 10,
							},
							{
								Key: util.ConcurrencyKey(
									enums.ConcurrencyScopeFn,
									uuid.Nil,
									"fn-id",
								),
								Limit: 5,
							},
						},
					},
				}, at)
				require.NoError(t, err)

				defaultPartition := getDefaultPartition(t, r, uuid.Nil)

				// The partition should use a custom ID for the concurrency key.
				pa1 := q.ItemPartitions(ctx, itemA)[0]
				pa2 := q.ItemPartitions(ctx, itemA)[1]
				pb1 := q.ItemPartitions(ctx, itemB)[0]
				pb2 := q.ItemPartitions(ctx, itemB)[1]

				require.Equal(t, "{queue}:sorted:c:00000000-0000-0000-0000-000000000000<2gu959eo1zbsi>", pa1.ID)
				require.Equal(t, "{queue}:sorted:c:00000000-0000-0000-0000-000000000000<1x6209w26mx6i>", pa2.ID)
				// Ensure the partitions match for two queue items.
				require.Equal(t, "{queue}:sorted:c:00000000-0000-0000-0000-000000000000<2gu959eo1zbsi>", pb1.ID)
				require.Equal(t, "{queue}:sorted:c:00000000-0000-0000-0000-000000000000<1x6209w26mx6i>", pb2.ID)

				score, err := r.ZScore(defaultQueueKey.GlobalPartitionIndex(), pa2.ID)
				require.NoError(t, err)
				require.EqualValues(t, at.Unix(), score, r.Dump())

				// Concurrency queue should be emptyu
				t.Run("Concurrency and scavenge queues are empty", func(t *testing.T) {
					mem, _ := r.ZMembers(q.u.kg.ConcurrencyIndex())
					require.Empty(t, mem, "concurrency queue is not empty")
				})

				// Do the lease.
				_, err = q.Lease(ctx, pa1, itemA, 10*time.Second, q.clock.Now(), nil)
				require.NoError(t, err)

				// The queue item is removed from each partition
				t.Run("The queue item is removed from each partition", func(t *testing.T) {
					mem, _ := r.ZMembers(pa1.zsetKey(q.u.kg))
					require.Equal(t, 1, len(mem), "leased item not removed from first partition", pa1.zsetKey(q.u.kg))

					mem, _ = r.ZMembers(pa2.zsetKey(q.u.kg))
					require.Equal(t, 1, len(mem), "leased item not removed from second partition", pa2.zsetKey(q.u.kg))
				})

				t.Run("The scavenger queue is updated with all queue items", func(t *testing.T) {
					mem, _ := r.ZMembers(q.u.kg.ConcurrencyIndex())
					require.Equal(t, 3, len(mem), "scavenge queue not updated", mem)
					require.Contains(t, mem, pa1.concurrencyKey(q.u.kg))
					require.Contains(t, mem, pa2.concurrencyKey(q.u.kg))
					require.NotContains(t, mem, defaultPartition.concurrencyKey(q.u.kg))
					require.Contains(t, mem, defaultPartition.FunctionID.String())
				})

				t.Run("Pointer queues don't update with a single tqueue item", func(t *testing.T) {
					nextScore, err := r.ZScore(defaultQueueKey.GlobalPartitionIndex(), pa1.Queue())
					require.NoError(t, err)
					require.EqualValues(t, int(score), int(nextScore), "score should not equal previous score")

					nextScore, err = r.ZScore(defaultQueueKey.GlobalPartitionIndex(), pa2.Queue())
					require.NoError(t, err)
					require.EqualValues(t, int(score), int(nextScore), "score should not equal previous score")

				})
			})
		})

		t.Run("With more than one item in the fn queue, it uses the next val for the global partition index", func(t *testing.T) {
			r.FlushAll()

			atA := time.Now().Truncate(time.Second).Add(time.Second)
			atB := atA.Add(time.Minute)

			itemA, err := q.EnqueueItem(ctx, QueueItem{}, atA)
			require.NoError(t, err)
			itemB, err := q.EnqueueItem(ctx, QueueItem{}, atB)
			require.NoError(t, err)

			p := q.ItemPartitions(ctx, itemA)[0]

			score, err := r.ZScore(defaultQueueKey.GlobalPartitionIndex(), p.Queue())
			require.NoError(t, err)
			require.EqualValues(t, atA.Unix(), score)

			// Leasing the item should update the score.
			_, err = q.Lease(ctx, p, itemA, 10*time.Second, time.Now(), nil)
			require.NoError(t, err)

			nextScore, err := r.ZScore(defaultQueueKey.GlobalPartitionIndex(), p.Queue())
			require.NoError(t, err)
			require.EqualValues(t, itemB.AtMS/1000, int(nextScore))
			require.NotEqualValues(t, int(score), int(nextScore), "score should not equal previous score")
		})
	})

	t.Run("It does nothing for a zero value partition", func(t *testing.T) {
		r.FlushAll()

		item, err := q.EnqueueItem(ctx, QueueItem{}, start)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		p := QueuePartition{} // Empty partition

		now := time.Now()
		id, err := q.Lease(ctx, p, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, id, item.LeaseID)
		require.WithinDuration(t, now.Add(time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		t.Run("It should NOT add the item to the function's in-progress concurrency queue", func(t *testing.T) {
			require.False(t, r.Exists(p.concurrencyKey(q.u.kg)))
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

		p := q.ItemPartitions(ctx, item)[0]

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

		require.False(t, r.Exists(QueuePartition{}.concurrencyKey(q.u.kg)))

		// Ensure the leased item has the next ID.
		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, nextID, item.LeaseID)
		require.WithinDuration(t, now.Add(10*time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		t.Run("It extends the score of the partition concurrency queue", func(t *testing.T) {
			at := ulid.Time(nextID.Time())
			scores := concurrencyQueueScores(t, r, p.concurrencyKey(q.u.kg), time.Now())
			require.Len(t, scores, 1)
			// Ensure that the score matches the lease.
			require.Equal(t, at, scores[item.ID], "%s not extended\n%s", p.concurrencyKey(q.u.kg), r.Dump())
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

		p := QueuePartition{FunctionID: &item.FunctionID}

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		nextID, err := q.ExtendLease(ctx, p, item, ulid.ULID{}, 10*time.Second)
		require.EqualValues(t, ErrQueueItemNotLeased, err)
		require.Nil(t, nextID)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)
	})

	t.Run("With custom keys in multiple partitions", func(t *testing.T) {
		r.FlushAll()

		item, err := q.EnqueueItem(ctx, QueueItem{
			FunctionID: uuid.New(),
			Data: osqueue.Item{
				CustomConcurrencyKeys: []state.CustomConcurrency{
					{
						Key: util.ConcurrencyKey(
							enums.ConcurrencyScopeAccount,
							uuid.Nil,
							"acct-id",
						),
						Limit: 10,
					},
					{
						Key: util.ConcurrencyKey(
							enums.ConcurrencyScopeFn,
							uuid.Nil,
							"fn-id",
						),
						Limit: 5,
					},
				},
			},
		}, start)
		require.Nil(t, err)

		// First 2 partitions will be custom.
		parts := q.ItemPartitions(ctx, item)
		require.Equal(t, int(enums.PartitionTypeConcurrencyKey), parts[0].PartitionType)
		require.Equal(t, int(enums.PartitionTypeConcurrencyKey), parts[1].PartitionType)

		// Lease the item.
		id, err := q.Lease(ctx, QueuePartition{}, item, time.Second, q.clock.Now(), nil)
		require.NoError(t, err)
		require.NotNil(t, id)

		score0, err := r.ZMScore(parts[0].concurrencyKey(q.u.kg), item.ID)
		require.NoError(t, err)
		score1, err := r.ZMScore(parts[1].concurrencyKey(q.u.kg), item.ID)
		require.NoError(t, err)
		require.Equal(t, score0[0], score1[0], "Partition scores should match after leasing")

		t.Run("extending the lease should extend both items in all partition's concurrency queues", func(t *testing.T) {
			id, err = q.ExtendLease(ctx, QueuePartition{}, item, *id, 98712*time.Millisecond)
			require.NoError(t, err)
			require.NotNil(t, id)

			newScore0, err := r.ZMScore(parts[0].concurrencyKey(q.u.kg), item.ID)
			require.NoError(t, err)
			newScore1, err := r.ZMScore(parts[1].concurrencyKey(q.u.kg), item.ID)
			require.NoError(t, err)

			require.Equal(t, newScore0, newScore1, "Partition scores should match after leasing")
			require.NotEqual(t, int(score0[0]), int(newScore0[0]), "Partition scores should have been updated: %v", newScore0)
			require.NotEqual(t, score1, newScore1, "Partition scores should have been updated")

			// And, the account-level concurrency queue is updated
			acctScore, err := r.ZMScore(q.u.kg.Concurrency("account", item.Data.Identifier.AccountID.String()), item.ID)
			require.NoError(t, err)
			require.EqualValues(t, acctScore[0], newScore0[0])
		})

		t.Run("Scavenge queue is updated", func(t *testing.T) {
			score, err := r.ZMScore(q.u.kg.ConcurrencyIndex(), parts[0].concurrencyKey(q.u.kg))
			require.NoError(t, err)
			require.NotZero(t, score[0])

			id, err = q.ExtendLease(ctx, QueuePartition{}, item, *id, 1238712*time.Millisecond)
			require.NoError(t, err)
			require.NotNil(t, id)

			nextScore, err := r.ZMScore(q.u.kg.ConcurrencyIndex(), parts[0].concurrencyKey(q.u.kg))
			require.NoError(t, err)

			require.NotEqual(t, score[0], nextScore[0])
		})
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
	q := NewQueue(queueClient)
	ctx := context.Background()

	t.Run("It always changes global partition scores", func(t *testing.T) {
		r.FlushAll()

		fnID, acctID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("fn")),
			uuid.NewSHA1(uuid.NameSpaceDNS, []byte("acct"))

		start := time.Now().Truncate(time.Second)
		itemA, err := q.EnqueueItem(ctx, QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					AccountID: acctID,
				},
				CustomConcurrencyKeys: []state.CustomConcurrency{
					{
						Key: util.ConcurrencyKey(
							enums.ConcurrencyScopeAccount,
							acctID,
							"acct-id",
						),
						Limit: 10,
					},
					{
						Key: util.ConcurrencyKey(
							enums.ConcurrencyScopeFn,
							fnID,
							"fn-id",
						),
						Limit: 5,
					},
				},
			},
		}, start)
		require.Nil(t, err)
		_, err = q.EnqueueItem(ctx, QueueItem{
			FunctionID: uuid.New(),
			Data: osqueue.Item{
				Identifier: state.Identifier{
					AccountID: acctID,
				},
				CustomConcurrencyKeys: []state.CustomConcurrency{
					{
						Key: util.ConcurrencyKey(
							enums.ConcurrencyScopeAccount,
							acctID,
							"acct-id",
						),
						Limit: 10,
					},
					{
						Key: util.ConcurrencyKey(
							enums.ConcurrencyScopeFn,
							fnID,
							"fn-id",
						),
						Limit: 5,
					},
				},
			},
		}, start)
		require.Nil(t, err)

		// First 2 partitions will be custom.
		parts := q.ItemPartitions(ctx, itemA)
		require.Equal(t, int(enums.PartitionTypeConcurrencyKey), parts[0].PartitionType)
		require.Equal(t, int(enums.PartitionTypeConcurrencyKey), parts[1].PartitionType)

		// Lease the first item, pretending it's in progress.
		_, err = q.Lease(ctx, QueuePartition{}, itemA, 10*time.Second, q.clock.Now(), nil)
		require.NoError(t, err)

		// Force requeue the next partition such that it's pushed forward, pretending there's
		// no capacity.
		err = q.PartitionRequeue(ctx, &parts[0], start.Add(30*time.Minute), true)
		require.NoError(t, err)
		err = q.PartitionRequeue(ctx, &parts[1], start.Add(30*time.Minute), true)
		require.NoError(t, err)

		t.Run("Requeueing partitions updates the score", func(t *testing.T) {
			partScoreA, _ := r.ZMScore(q.u.kg.GlobalPartitionIndex(), parts[0].ID)
			partScoreB, _ := r.ZMScore(q.u.kg.GlobalPartitionIndex(), parts[1].ID)
			require.EqualValues(t, start.Add(30*time.Minute).Unix(), partScoreA[0])
			require.EqualValues(t, start.Add(30*time.Minute).Unix(), partScoreB[0])

			partScoreA, _ = r.ZMScore(q.u.kg.AccountPartitionIndex(acctID), parts[0].ID)
			partScoreB, _ = r.ZMScore(q.u.kg.AccountPartitionIndex(acctID), parts[1].ID)
			require.NotNil(t, partScoreA, "expected partition requeue to update account partition index", r.Dump())
			require.NotNil(t, partScoreB)
			require.EqualValues(t, start.Add(30*time.Minute).Unix(), partScoreA[0])
			require.EqualValues(t, start.Add(30*time.Minute).Unix(), partScoreB[0])
		})

		err = q.Dequeue(ctx, QueuePartition{}, itemA)
		require.Nil(t, err)

		t.Run("The outstanding partition scores should reset", func(t *testing.T) {
			partScoreA, _ := r.ZMScore(q.u.kg.GlobalPartitionIndex(), parts[0].ID)
			partScoreB, _ := r.ZMScore(q.u.kg.GlobalPartitionIndex(), parts[1].ID)
			require.EqualValues(t, start, time.Unix(int64(partScoreA[0]), 0), r.Dump())
			require.EqualValues(t, start, time.Unix(int64(partScoreB[0]), 0))

			partScoreA, _ = r.ZMScore(q.u.kg.AccountPartitionIndex(acctID), parts[0].ID)
			partScoreB, _ = r.ZMScore(q.u.kg.AccountPartitionIndex(acctID), parts[1].ID)
			require.EqualValues(t, start, time.Unix(int64(partScoreA[0]), 0), r.Dump())
			require.EqualValues(t, start, time.Unix(int64(partScoreB[0]), 0))
		})
	})

	t.Run("with concurrency keys", func(t *testing.T) {
		start := time.Now()

		t.Run("with an unleased item", func(t *testing.T) {
			r.FlushAll()
			item, err := q.EnqueueItem(ctx, QueueItem{
				FunctionID: uuid.New(),
				Data: osqueue.Item{
					CustomConcurrencyKeys: []state.CustomConcurrency{
						{
							Key: util.ConcurrencyKey(
								enums.ConcurrencyScopeAccount,
								uuid.Nil,
								"acct-id",
							),
							Limit: 10,
						},
						{
							Key: util.ConcurrencyKey(
								enums.ConcurrencyScopeFn,
								uuid.Nil,
								"fn-id",
							),
							Limit: 5,
						},
					},
				},
			}, start)
			require.Nil(t, err)

			// First 2 partitions will be custom.
			parts := q.ItemPartitions(ctx, item)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), parts[0].PartitionType)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), parts[1].PartitionType)

			err = q.Dequeue(ctx, QueuePartition{}, item)
			require.Nil(t, err)

			t.Run("The outstanding partition items should be empty", func(t *testing.T) {
				mem, _ := r.ZMembers(parts[0].zsetKey(q.u.kg))
				require.Equal(t, 0, len(mem))

				mem, _ = r.ZMembers(parts[1].zsetKey(q.u.kg))
				require.NoError(t, err)
				require.Equal(t, 0, len(mem))
			})
		})

		t.Run("with a leased item", func(t *testing.T) {
			r.FlushAll()
			item, err := q.EnqueueItem(ctx, QueueItem{
				FunctionID: uuid.New(),
				Data: osqueue.Item{
					CustomConcurrencyKeys: []state.CustomConcurrency{
						{
							Key: util.ConcurrencyKey(
								enums.ConcurrencyScopeAccount,
								uuid.Nil,
								"acct-id",
							),
							Limit: 10,
						},
						{
							Key: util.ConcurrencyKey(
								enums.ConcurrencyScopeFn,
								uuid.Nil,
								"fn-id",
							),
							Limit: 5,
						},
					},
				},
			}, start)
			require.Nil(t, err)

			// First 2 partitions will be custom.
			parts := q.ItemPartitions(ctx, item)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), parts[0].PartitionType)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), parts[1].PartitionType)

			id, err := q.Lease(ctx, QueuePartition{}, item, 10*time.Second, time.Now(), nil)
			require.NoError(t, err)
			require.NotEmpty(t, id)

			t.Run("The scavenger queue should not yet be empty", func(t *testing.T) {
				mems, err := r.ZMembers(q.u.kg.ConcurrencyIndex())
				require.NoError(t, err)
				require.NotEmpty(t, mems)
			})

			err = q.Dequeue(ctx, QueuePartition{}, item)
			require.Nil(t, err)

			t.Run("The outstanding partition items should be empty", func(t *testing.T) {
				mem, _ := r.ZMembers(parts[0].zsetKey(q.u.kg))
				require.Equal(t, 0, len(mem))

				mem, _ = r.ZMembers(parts[1].zsetKey(q.u.kg))
				require.NoError(t, err)
				require.Equal(t, 0, len(mem))
			})

			t.Run("The concurrenty partition items should be empty", func(t *testing.T) {
				mem, _ := r.ZMembers(parts[0].concurrencyKey(q.u.kg))
				require.Equal(t, 0, len(mem))

				mem, _ = r.ZMembers(parts[1].concurrencyKey(q.u.kg))
				require.NoError(t, err)
				require.Equal(t, 0, len(mem))
			})

			t.Run("The scavenger queue should now be empty", func(t *testing.T) {
				mems, _ := r.ZMembers(q.u.kg.ConcurrencyIndex())
				require.Empty(t, mems)
			})
		})
	})

	t.Run("It should remove a queue item", func(t *testing.T) {
		r.FlushAll()

		start := time.Now()

		item, err := q.EnqueueItem(ctx, QueueItem{}, start)
		require.NoError(t, err)

		p := QueuePartition{FunctionID: &item.FunctionID}

		id, err := q.Lease(ctx, p, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		t.Run("The lease exists in the partition queue", func(t *testing.T) {
			count, err := q.InProgress(ctx, "p", p.FunctionID.String())
			require.NoError(t, err)
			require.EqualValues(t, 1, count, r.Dump())
		})

		err = q.Dequeue(ctx, p, item)
		require.NoError(t, err)

		t.Run("It should remove the item from the queue map", func(t *testing.T) {
			val := r.HGet(q.u.kg.QueueItem(), id.String())
			require.Empty(t, val)
		})

		t.Run("Extending a lease should fail after dequeue", func(t *testing.T) {
			id, err := q.ExtendLease(ctx, p, item, *id, time.Minute)
			require.Equal(t, ErrQueueItemNotFound, err)
			require.Nil(t, id)
		})

		t.Run("It should remove the item from the queue index", func(t *testing.T) {
			items, err := q.Peek(ctx, &p, time.Now().Add(time.Hour), 10)
			require.NoError(t, err)
			require.EqualValues(t, 0, len(items))
		})

		t.Run("It should remove the item from the concurrency partition's queue", func(t *testing.T) {
			count, err := q.InProgress(ctx, "p", p.FunctionID.String())
			require.NoError(t, err)
			require.EqualValues(t, 0, count)
		})

		t.Run("It should work if the item is not leased (eg. deletions)", func(t *testing.T) {
			item, err := q.EnqueueItem(ctx, QueueItem{}, start)
			require.NoError(t, err)

			err = q.Dequeue(ctx, p, item)
			require.NoError(t, err)

			val := r.HGet(q.u.kg.QueueItem(), id.String())
			require.Empty(t, val)
		})

		t.Run("Removes default indexes", func(t *testing.T) {
			at := time.Now().Truncate(time.Second)
			rid := ulid.MustNew(ulid.Now(), rand.Reader)
			item, err := q.EnqueueItem(ctx, QueueItem{
				FunctionID: uuid.New(),
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

	t.Run("backcompat: it should drop previous partition names from concurrency index", func(t *testing.T) {
		// This tests backwards compatibility with the old concurrency index member naming scheme
		r.FlushAll()
		start := time.Now().Truncate(time.Second)

		customQueueName := "custom-queue-name"
		item, err := q.EnqueueItem(ctx, QueueItem{
			FunctionID: uuid.New(),
			Data: osqueue.Item{
				QueueName: &customQueueName,
			},
			QueueName: &customQueueName,
		}, start)
		require.NoError(t, err)
		parts := q.ItemPartitions(ctx, item)

		itemCountMatches := func(num int) {
			zsetKey := parts[0].zsetKey(q.u.kg)
			items, err := rc.Do(ctx, rc.B().
				Zrangebyscore().
				Key(zsetKey).
				Min("-inf").
				Max("+inf").
				Build()).AsStrSlice()
			require.NoError(t, err)
			assert.Equal(t, num, len(items), "expected %d items in the queue %q", num, zsetKey, r.Dump())
		}

		concurrencyItemCountMatches := func(num int) {
			items, err := rc.Do(ctx, rc.B().
				Zrangebyscore().
				Key(parts[0].concurrencyKey(q.u.kg)).
				Min("-inf").
				Max("+inf").
				Build()).AsStrSlice()
			require.NoError(t, err)
			assert.Equal(t, num, len(items), "expected %d items in the concurrency queue", num, r.Dump())
		}

		itemCountMatches(1)
		concurrencyItemCountMatches(0)

		_, err = q.Lease(ctx, parts[0], item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		itemCountMatches(0)
		concurrencyItemCountMatches(1)

		leaseExpiry := time.Now().Add(time.Second)

		// Ensure the concurrency index is updated.
		mem, err := r.ZMembers(q.u.kg.ConcurrencyIndex())
		require.NoError(t, err)
		assert.Equal(t, 1, len(mem))
		assert.Contains(t, mem[0], parts[0].concurrencyKey(q.u.kg))

		// Rename the member to the old format
		removed, err := r.ZRem(q.u.kg.ConcurrencyIndex(), parts[0].concurrencyKey(q.u.kg))
		require.NoError(t, err)
		assert.True(t, removed)

		added, err := r.ZAdd(q.u.kg.ConcurrencyIndex(), float64(leaseExpiry.UnixMilli()), customQueueName)
		require.NoError(t, err)
		assert.True(t, added)

		// Dequeue the item.
		err = q.Dequeue(ctx, parts[0], item)
		require.NoError(t, err)

		itemCountMatches(0)
		concurrencyItemCountMatches(0)

		// Ensure the concurrency index is updated.
		numMembers, err := rc.Do(ctx, rc.B().Zcard().Key(q.u.kg.ConcurrencyIndex()).Build()).AsInt64()
		require.NoError(t, err, r.Dump())
		assert.Equal(t, int64(0), numMembers, "concurrency index should be empty", mem)
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

		p := QueuePartition{FunctionID: &item.FunctionID}

		_, err = q.Lease(ctx, p, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		// Assert partition index is original
		pi := QueuePartition{FunctionID: &item.FunctionID}
		requirePartitionScoreEquals(t, r, pi.FunctionID, now.Truncate(time.Second))

		requirePartitionInProgress(t, q, item.FunctionID, 1)

		next := now.Add(time.Hour)
		err = q.Requeue(ctx, item, next)
		require.NoError(t, err)

		t.Run("It should re-enqueue the item with the future time", func(t *testing.T) {
			requireItemScoreEquals(t, r, item, next)
		})

		t.Run("It should always remove the lease from the re-enqueued item", func(t *testing.T) {
			fetched := getQueueItem(t, r, item.ID)
			require.Nil(t, fetched.LeaseID)
		})

		t.Run("It should decrease the in-progress count", func(t *testing.T) {
			requirePartitionInProgress(t, q, item.FunctionID, 0)
		})

		t.Run("It should update the partition's earliest time, if earliest", func(t *testing.T) {
			// Assert partition index is updated, as there's only one item here.
			requirePartitionScoreEquals(t, r, pi.FunctionID, next)
		})

		t.Run("It should not update the partition's earliest time, if later", func(t *testing.T) {
			_, err := q.EnqueueItem(ctx, QueueItem{}, now)
			require.NoError(t, err)

			requirePartitionScoreEquals(t, r, pi.FunctionID, now)

			next := now.Add(2 * time.Hour)
			err = q.Requeue(ctx, item, next)
			require.NoError(t, err)

			requirePartitionScoreEquals(t, r, pi.FunctionID, now)
		})

		t.Run("Updates default indexes", func(t *testing.T) {
			at := time.Now().Truncate(time.Second)
			rid := ulid.MustNew(ulid.Now(), rand.Reader)
			item, err := q.EnqueueItem(ctx, QueueItem{
				FunctionID: uuid.New(),
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
			err = q.Requeue(ctx, item, next)
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

	t.Run("For a queue item with concurrency keys it requeues all partitions", func(t *testing.T) {
		r.FlushAll()

		fnID, acctID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("fn")),
			uuid.NewSHA1(uuid.NameSpaceDNS, []byte("acct"))

		now := time.Now()
		item := QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					AccountID: acctID,
				},
				CustomConcurrencyKeys: []state.CustomConcurrency{
					{
						Key: util.ConcurrencyKey(
							enums.ConcurrencyScopeAccount,
							acctID,
							"test-plz",
						),
						Limit: 5,
					},
					{
						Key: util.ConcurrencyKey(
							enums.ConcurrencyScopeFn,
							fnID,
							"another-id",
						),
						Limit: 2,
					},
				},
			},
		}
		item, err := q.EnqueueItem(ctx, item, now)
		require.NoError(t, err)

		parts := q.ItemPartitions(ctx, item)

		// Get all scores
		itemScoreA, _ := r.ZMScore(parts[0].zsetKey(q.u.kg), item.ID)
		itemScoreB, _ := r.ZMScore(parts[1].zsetKey(q.u.kg), item.ID)
		partScoreA, _ := r.ZMScore(q.u.kg.GlobalPartitionIndex(), parts[0].ID)
		partScoreB, _ := r.ZMScore(q.u.kg.GlobalPartitionIndex(), parts[1].ID)
		accountPartScoreA, _ := r.ZMScore(q.u.kg.AccountPartitionIndex(acctID), parts[0].ID)
		accountPartScoreB, _ := r.ZMScore(q.u.kg.AccountPartitionIndex(acctID), parts[1].ID)
		accountScore, _ := r.ZMScore(q.u.kg.GlobalAccountIndex(), acctID.String())

		require.NotEmpty(t, itemScoreA, "Couldn't find item in '%s':\n%s", parts[0].zsetKey(q.u.kg), r.Dump())
		require.NotEmpty(t, itemScoreB, "Couldn't find item in '%s':\n%s", parts[1].zsetKey(q.u.kg), r.Dump())
		require.NotEmpty(t, partScoreA)
		require.NotEmpty(t, partScoreB)
		require.Equal(t, partScoreA, accountPartScoreA, "expected account partitions to match global partitions")
		require.Equal(t, partScoreB, accountPartScoreB, "expected account partitions to match global partitions")
		require.Equal(t, accountPartScoreA[0], accountScore[0], "expected account score to match earliest account partition")

		_, err = q.Lease(ctx, QueuePartition{}, item, time.Second, q.clock.Now(), nil)
		require.NoError(t, err)

		// Requeue
		next := now.Add(time.Hour)
		err = q.Requeue(ctx, item, next)
		require.NoError(t, err)

		t.Run("It requeues all partitions", func(t *testing.T) {
			newItemScoreA, _ := r.ZMScore(parts[0].zsetKey(q.u.kg), item.ID)
			newItemScoreB, _ := r.ZMScore(parts[1].zsetKey(q.u.kg), item.ID)
			newPartScoreA, _ := r.ZMScore(q.u.kg.GlobalPartitionIndex(), parts[0].ID)
			newPartScoreB, _ := r.ZMScore(q.u.kg.GlobalPartitionIndex(), parts[1].ID)
			newAccountPartScoreA, _ := r.ZMScore(q.u.kg.AccountPartitionIndex(acctID), parts[0].ID)
			newAccountPartScoreB, _ := r.ZMScore(q.u.kg.AccountPartitionIndex(acctID), parts[1].ID)
			newAccountScore, _ := r.ZMScore(q.u.kg.GlobalAccountIndex(), acctID.String())

			require.NotEqual(t, itemScoreA, newItemScoreA)
			require.NotEqual(t, itemScoreB, newItemScoreB)
			require.NotEqual(t, partScoreA, newPartScoreA)
			require.NotEqual(t, partScoreB, newPartScoreB)
			require.Equal(t, newPartScoreA, newAccountPartScoreA)
			require.Equal(t, newPartScoreB, newAccountPartScoreB)
			require.Equal(t, next.Truncate(time.Second).Unix(), int64(newPartScoreA[0]))
			require.Equal(t, newAccountPartScoreA[0], newAccountScore[0], "expected account score to match earliest account partition", r.Dump())

			require.Equal(t, newItemScoreA, newItemScoreB)
			require.EqualValues(t, next.UnixMilli(), int(newItemScoreA[0]))
			require.EqualValues(t, next.Unix(), int(newPartScoreA[0]))
		})
	})
}

func TestQueuePartitionLease(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	idA, idB, idC := uuid.New(), uuid.New(), uuid.New()
	atA, atB, atC := now, now.Add(time.Second), now.Add(2*time.Second)

	pA := QueuePartition{ID: idA.String(), FunctionID: &idA}

	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	q := NewQueue(NewQueueClient(rc, QueueDefaultKey))
	ctx := context.Background()

	_, err = q.EnqueueItem(ctx, QueueItem{FunctionID: idA}, atA)
	require.NoError(t, err)
	_, err = q.EnqueueItem(ctx, QueueItem{FunctionID: idB}, atB)
	require.NoError(t, err)
	_, err = q.EnqueueItem(ctx, QueueItem{FunctionID: idC}, atC)
	require.NoError(t, err)

	t.Run("Partitions are in order after enqueueing", func(t *testing.T) {
		items, err := q.PartitionPeek(ctx, true, time.Now().Add(time.Hour), PartitionPeekMax)
		require.NoError(t, err)
		require.Len(t, items, 3)
		require.EqualValues(t, []*QueuePartition{
			{ID: idA.String(), FunctionID: &idA, AccountID: uuid.Nil, ConcurrencyLimit: consts.DefaultConcurrencyLimit},
			{ID: idB.String(), FunctionID: &idB, AccountID: uuid.Nil, ConcurrencyLimit: consts.DefaultConcurrencyLimit},
			{ID: idC.String(), FunctionID: &idC, AccountID: uuid.Nil, ConcurrencyLimit: consts.DefaultConcurrencyLimit},
		}, items)
	})

	leaseUntil := now.Add(3 * time.Second)

	t.Run("It leases a partition", func(t *testing.T) {
		// Lease the first item now.
		leasedAt := time.Now()
		leaseID, capacity, err := q.PartitionLease(ctx, &pA, time.Until(leaseUntil))
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		require.NotZero(t, capacity)

		// Pause so that we can assert that the last lease time was set correctly.
		<-time.After(50 * time.Millisecond)

		t.Run("It updates the partition score", func(t *testing.T) {
			items, err := q.PartitionPeek(ctx, true, now.Add(time.Hour), PartitionPeekMax)

			// Require the lease ID is within 25 MS of the expected value.
			require.WithinDuration(t, leaseUntil, ulid.Time(leaseID.Time()), 25*time.Millisecond)

			require.NoError(t, err)
			require.Len(t, items, 3)
			require.EqualValues(t, []*QueuePartition{
				{ID: idB.String(), FunctionID: &idB, AccountID: uuid.Nil, ConcurrencyLimit: consts.DefaultConcurrencyLimit},
				{ID: idC.String(), FunctionID: &idC, AccountID: uuid.Nil, ConcurrencyLimit: consts.DefaultConcurrencyLimit},
				{
					ID:               idA.String(),
					FunctionID:       &idA,
					AccountID:        uuid.Nil,
					Last:             items[2].Last, // Use the leased partition time.
					LeaseID:          leaseID,
					ConcurrencyLimit: consts.DefaultConcurrencyLimit,
				}, // idA is now last.
			}, items)
			requirePartitionScoreEquals(t, r, &idA, leaseUntil)
			// require that the last leased time is within 5ms for tests
			require.WithinDuration(t, leasedAt, time.UnixMilli(items[2].Last), 5*time.Millisecond)
		})

		t.Run("It can't lease an existing partition lease", func(t *testing.T) {
			id, capacity, err := q.PartitionLease(ctx, &pA, time.Second*29)
			require.Equal(t, ErrPartitionAlreadyLeased, err)
			require.Nil(t, id)
			require.Zero(t, capacity)

			// Assert that score didn't change (we added 1 second in the previous test)
			requirePartitionScoreEquals(t, r, &idA, leaseUntil)
		})

	})

	t.Run("It allows leasing an expired partition lease", func(t *testing.T) {
		<-time.After(time.Until(leaseUntil))

		requirePartitionScoreEquals(t, r, &idA, leaseUntil)

		id, capacity, err := q.PartitionLease(ctx, &pA, time.Second*5)
		require.Nil(t, err)
		require.NotNil(t, id)
		require.NotZero(t, capacity)

		requirePartitionScoreEquals(t, r, &idA, time.Now().Add(time.Second*5))
	})

	t.Run("Partition pausing", func(t *testing.T) {
		r.FlushAll() // reset everything
		q := NewQueue(NewQueueClient(rc, QueueDefaultKey))
		ctx := context.Background()

		_, err = q.EnqueueItem(ctx, QueueItem{FunctionID: idA}, atA)
		require.NoError(t, err)
		_, err = q.EnqueueItem(ctx, QueueItem{FunctionID: idB}, atB)
		require.NoError(t, err)
		_, err = q.EnqueueItem(ctx, QueueItem{FunctionID: idC}, atC)
		require.NoError(t, err)

		t.Run("Fails to lease a paused partition", func(t *testing.T) {
			// pause fn A's partition:
			err = q.SetFunctionPaused(ctx, idA, true)
			require.NoError(t, err)

			// attempt to lease the paused partition:
			id, capacity, err := q.PartitionLease(ctx, &pA, time.Second*5)
			require.Nil(t, id)
			require.Error(t, err)
			require.Zero(t, capacity)
			require.ErrorIs(t, err, ErrPartitionPaused)
		})

		t.Run("Succeeds to lease a previously paused partition", func(t *testing.T) {
			// unpause fn A's partition:
			err = q.SetFunctionPaused(ctx, idA, false)
			require.NoError(t, err)

			// attempt to lease the unpaused partition:
			id, capacity, err := q.PartitionLease(ctx, &pA, time.Second*5)
			require.NotNil(t, id)
			require.NoError(t, err)
			require.NotZero(t, capacity)
		})
	})

	t.Run("With key partitions", func(t *testing.T) {
		fnID := uuid.New()

		// Enqueueing an item
		ck := createConcurrencyKey(enums.ConcurrencyScopeFn, fnID, "test", 1)
		_, _, hash, _ := ck.ParseKey() // get the hash of the "test" string / evaluated input.

		_, err := q.EnqueueItem(ctx, QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				CustomConcurrencyKeys: []state.CustomConcurrency{ck},
			},
		}, now.Add(10*time.Second))
		require.NoError(t, err)

		p := QueuePartition{
			ID:               q.u.kg.PartitionQueueSet(enums.PartitionTypeConcurrencyKey, fnID.String(), hash),
			FunctionID:       &fnID,
			PartitionType:    int(enums.PartitionTypeConcurrencyKey),
			ConcurrencyScope: int(enums.ConcurrencyScopeFn),
		}

		leaseUntil := now.Add(3 * time.Second)
		leaseID, capacity, err := q.PartitionLease(ctx, &p, time.Until(leaseUntil))
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		require.NotZero(t, capacity)
	})

	t.Run("concurrency is checked early", func(t *testing.T) {
		start := time.Now().Truncate(time.Second)

		t.Run("With partition concurrency limits", func(t *testing.T) {
			r.FlushAll()

			// Only allow a single leased item
			q.concurrencyLimitGetter = func(ctx context.Context, p QueuePartition) PartitionConcurrencyLimits {
				return PartitionConcurrencyLimits{1, 1, 1}
			}

			fnID := uuid.New()
			// Create a new item
			itemA, err := q.EnqueueItem(ctx, QueueItem{FunctionID: fnID}, start)
			require.NoError(t, err)
			_, err = q.EnqueueItem(ctx, QueueItem{FunctionID: fnID}, start)
			require.NoError(t, err)
			// Use the new item's workflow ID
			p := QueuePartition{ID: itemA.FunctionID.String(), FunctionID: &itemA.FunctionID}

			t.Run("Leases with capacity", func(t *testing.T) {
				_, err = q.Lease(ctx, p, itemA, 5*time.Second, time.Now(), nil)
				require.NoError(t, err)
			})

			t.Run("Partition lease errors without capacity", func(t *testing.T) {
				leaseId, _, err := q.PartitionLease(ctx, &p, 5*time.Second)
				require.Nil(t, leaseId, "No lease id when leasing fails.\n%s", r.Dump())
				require.Error(t, err)
				require.ErrorIs(t, err, ErrPartitionConcurrencyLimit)
			})
		})

		t.Run("With account concurrency limits", func(t *testing.T) {
			r.FlushAll()

			// Only allow a single leased item via account limits
			q.concurrencyLimitGetter = func(ctx context.Context, p QueuePartition) PartitionConcurrencyLimits {
				return PartitionConcurrencyLimits{
					AccountLimit:   1,
					FunctionLimit:  100,
					CustomKeyLimit: NoConcurrencyLimit,
				}
			}

			acctId := uuid.New()

			// Create a new item
			itemA, err := q.EnqueueItem(ctx, QueueItem{FunctionID: uuid.New(), Data: osqueue.Item{Identifier: state.Identifier{AccountID: acctId}}}, start)
			require.NoError(t, err)

			_, err = q.EnqueueItem(ctx, QueueItem{FunctionID: uuid.New(), Data: osqueue.Item{Identifier: state.Identifier{AccountID: acctId}}}, start)
			require.NoError(t, err)

			// Use the new item's workflow ID
			p := QueuePartition{AccountID: acctId, FunctionID: &itemA.FunctionID}

			t.Run("Leases with capacity", func(t *testing.T) {
				_, err = q.Lease(ctx, p, itemA, 5*time.Second, time.Now(), nil)
				require.NoError(t, err)
			})

			t.Run("Partition lease errors without capacity", func(t *testing.T) {
				leaseId, _, err := q.PartitionLease(ctx, &p, 5*time.Second)
				require.Nil(t, leaseId, "No lease id when leasing fails.\n%s", r.Dump())
				require.Error(t, err)
				require.ErrorIs(t, err, ErrAccountConcurrencyLimit)
			})
		})

		t.Run("With custom concurrency limits", func(t *testing.T) {
			r.FlushAll()
			// Only allow a single leased item via account limits
			q.concurrencyLimitGetter = func(ctx context.Context, p QueuePartition) PartitionConcurrencyLimits {
				return PartitionConcurrencyLimits{
					AccountLimit:   100,
					FunctionLimit:  100,
					CustomKeyLimit: 1,
				}
			}

			ck := createConcurrencyKey(enums.ConcurrencyScopeAccount, uuid.Nil, "foo", 1)

			// Create a new item
			itemA, err := q.EnqueueItem(ctx, QueueItem{
				FunctionID: uuid.New(),
				Data: osqueue.Item{
					CustomConcurrencyKeys: []state.CustomConcurrency{
						{
							Key:   ck.Key,
							Limit: 1,
						},
					},
				},
			}, start)
			require.NoError(t, err)

			_, err = q.EnqueueItem(ctx, QueueItem{
				FunctionID: uuid.New(),
				Data: osqueue.Item{
					CustomConcurrencyKeys: []state.CustomConcurrency{
						{
							Key:   ck.Key,
							Limit: 1,
						},
					},
				},
			}, start)
			require.NoError(t, err)

			// Use the new item's workflow ID
			p := QueuePartition{FunctionID: &itemA.FunctionID}

			t.Run("Leases with capacity", func(t *testing.T) {
				_, err = q.Lease(ctx, p, itemA, 5*time.Second, time.Now(), nil)
				require.NoError(t, err)
			})

			t.Run("Partition lease errors without capacity", func(t *testing.T) {
				_, _, hash, _ := ck.ParseKey()
				qp := getPartition(t, r, enums.PartitionTypeConcurrencyKey, uuid.Nil, hash)

				leaseId, _, err := q.PartitionLease(ctx, &qp, 5*time.Second)
				require.Nil(t, leaseId, "No lease id when leasing fails.\n%s", r.Dump())
				require.Error(t, err)
				require.ErrorIs(t, err, ErrConcurrencyLimitCustomKey)
			})
		})
	})
}

func TestQueuePartitionPeek(t *testing.T) {
	idA := uuid.New() // low pri
	idB := uuid.New()
	idC := uuid.New()

	accountId := uuid.New()

	newQueueItem := func(id uuid.UUID) QueueItem {
		return QueueItem{
			FunctionID: id,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					WorkflowID: id,
					AccountID:  accountId,
				},
			},
		}
	}

	now := time.Now().Truncate(time.Second).UTC()
	atA, atB, atC := now, now.Add(2*time.Second), now.Add(4*time.Second)

	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	q := NewQueue(
		NewQueueClient(rc, QueueDefaultKey),
		WithPriorityFinder(func(ctx context.Context, p QueuePartition) uint {
			if p.FunctionID == nil {
				return PriorityMin
			}
			switch *p.FunctionID {
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

	t.Run("Sequentially returns partitions in order", func(t *testing.T) {
		items, err := q.PartitionPeek(ctx, true, time.Now().Add(time.Hour), PartitionPeekMax)
		require.NoError(t, err)
		require.Len(t, items, 3)
		require.EqualValues(t, []*QueuePartition{
			{ID: idA.String(), FunctionID: &idA, AccountID: accountId, ConcurrencyLimit: consts.DefaultConcurrencyLimit},
			{ID: idB.String(), FunctionID: &idB, AccountID: accountId, ConcurrencyLimit: consts.DefaultConcurrencyLimit},
			{ID: idC.String(), FunctionID: &idC, AccountID: accountId, ConcurrencyLimit: consts.DefaultConcurrencyLimit},
		}, items)
	})

	t.Run("With a single peek max, it returns the first item if sequential every time", func(t *testing.T) {
		for i := 0; i <= 50; i++ {
			items, err := q.PartitionPeek(ctx, true, time.Now().Add(time.Hour), 1)
			require.NoError(t, err)
			require.Len(t, items, 1)
			require.Equal(t, &idA, items[0].FunctionID)
		}
	})

	t.Run("With a single peek max, it returns random items that are available using offsets", func(t *testing.T) {
		found := map[uuid.UUID]bool{idA: false, idB: false, idC: false}

		for i := 0; i <= 50; i++ {
			items, err := q.PartitionPeek(ctx, false, time.Now().Add(time.Hour), 1)
			require.NoError(t, err)
			require.Len(t, items, 1)
			found[*items[0].FunctionID] = true
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
			switch *items[0].FunctionID {
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
			WithPriorityFinder(func(ctx context.Context, p QueuePartition) uint {
				if p.FunctionID == nil {
					return PriorityMin
				}
				switch *p.FunctionID {
				case idA:
					return PriorityMax
				default:
					return PriorityMin // Sorry A
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
			{ID: idB.String(), FunctionID: &idB, AccountID: accountId, ConcurrencyLimit: consts.DefaultConcurrencyLimit},
			{ID: idC.String(), FunctionID: &idC, AccountID: accountId, ConcurrencyLimit: consts.DefaultConcurrencyLimit},
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
			WithPriorityFinder(func(_ context.Context, _ QueuePartition) uint {
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
			{ID: idB.String(), FunctionID: &idB, AccountID: accountId, ConcurrencyLimit: consts.DefaultConcurrencyLimit},
			{ID: idC.String(), FunctionID: &idC, AccountID: accountId, ConcurrencyLimit: consts.DefaultConcurrencyLimit},
		}, items)

		// After unpausing A, it should be included in the peek:
		err = q.SetFunctionPaused(ctx, idA, false)
		require.NoError(t, err)
		items, err = q.PartitionPeek(ctx, true, time.Now().Add(time.Hour), PartitionPeekMax)
		require.NoError(t, err)
		require.Len(t, items, 3)
		require.EqualValues(t, []*QueuePartition{
			{ID: idA.String(), FunctionID: &idA, AccountID: accountId, ConcurrencyLimit: consts.DefaultConcurrencyLimit},
			{ID: idB.String(), FunctionID: &idB, AccountID: accountId, ConcurrencyLimit: consts.DefaultConcurrencyLimit},
			{ID: idC.String(), FunctionID: &idC, AccountID: accountId, ConcurrencyLimit: consts.DefaultConcurrencyLimit},
		}, items, r.Dump())
	})

	t.Run("Cleans up missing partitions in account queue", func(t *testing.T) {
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		q := NewQueue(
			NewQueueClient(rc, QueueDefaultKey),
			WithPriorityFinder(func(_ context.Context, _ QueuePartition) uint {
				return PriorityDefault
			}),
		)
		enqueue(q)

		// Create inconsistency: Delete partition item from partition hash and global partition index but _not_ account partitions
		err = rc.Do(ctx, rc.B().Hdel().Key(q.u.kg.PartitionItem()).Field(idA.String()).Build()).Error()
		require.NoError(t, err)
		err = rc.Do(ctx, rc.B().Zrem().Key(q.u.kg.GlobalPartitionIndex()).Member(idA.String()).Build()).Error()
		require.NoError(t, err)

		// This should only select B and C, as id A is ignored and cleaned up:
		items, err := q.partitionPeek(ctx, q.u.kg.AccountPartitionIndex(accountId), true, time.Now().Add(time.Hour), PartitionPeekMax, &accountId)
		require.NoError(t, err)
		require.Len(t, items, 2)
		require.EqualValues(t, []*QueuePartition{
			{ID: idB.String(), AccountID: accountId, FunctionID: &idB, ConcurrencyLimit: consts.DefaultConcurrencyLimit},
			{ID: idC.String(), AccountID: accountId, FunctionID: &idC, ConcurrencyLimit: consts.DefaultConcurrencyLimit},
		}, items)

		// Ensure the partition is removed from the account queue
		apIds := getAccountPartitions(t, rc, accountId)
		assert.Equal(t, 2, len(apIds))
		assert.NotContains(t, apIds, idA.String())
		assert.Contains(t, apIds, idB.String())
		assert.Contains(t, apIds, idC.String())
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

	t.Run("For default items without concurrency settings", func(t *testing.T) {
		qi, err := q.EnqueueItem(ctx, QueueItem{FunctionID: idA}, now)
		require.NoError(t, err)

		p := QueuePartition{FunctionID: &qi.FunctionID, EnvID: &qi.WorkspaceID}

		t.Run("Uses the next job item's time when requeueing with another job", func(t *testing.T) {
			requirePartitionScoreEquals(t, r, &idA, now)
			next := now.Add(time.Hour)
			err := q.PartitionRequeue(ctx, &p, next, false)
			require.NoError(t, err)
			requirePartitionScoreEquals(t, r, &idA, now)
		})

		next := now.Add(5 * time.Second)
		t.Run("It removes any lease when requeueing", func(t *testing.T) {
			_, _, err := q.PartitionLease(ctx, &QueuePartition{FunctionID: &idA}, time.Minute)
			require.NoError(t, err)

			err = q.PartitionRequeue(ctx, &p, next, true)
			require.NoError(t, err)
			requirePartitionScoreEquals(t, r, &idA, next)

			loaded := getDefaultPartition(t, r, idA)
			require.Nil(t, loaded.LeaseID)

			// Forcing should set a ForceAtMS field.
			require.NotEmpty(t, loaded.ForceAtMS)

			t.Run("Enqueueing with a force at time should not update the score", func(t *testing.T) {
				loaded := getDefaultPartition(t, r, idA)
				require.NotEmpty(t, loaded.ForceAtMS)

				qi, err := q.EnqueueItem(ctx, QueueItem{FunctionID: idA}, now)

				loaded = getDefaultPartition(t, r, idA)
				require.NotEmpty(t, loaded.ForceAtMS)

				require.NoError(t, err)
				requirePartitionScoreEquals(t, r, &idA, next)
				requirePartitionScoreEquals(t, r, &idA, time.UnixMilli(loaded.ForceAtMS))

				// Now remove this item, as we dont need it for any future tests.
				err = q.Dequeue(ctx, p, qi)
				require.NoError(t, err)
			})
		})

		t.Run("It returns a partition not found error if deleted", func(t *testing.T) {
			err := q.Dequeue(ctx, p, qi)
			require.NoError(t, err)

			err = q.PartitionRequeue(ctx, &p, time.Now().Add(time.Minute), false)
			require.Equal(t, ErrPartitionGarbageCollected, err)

			// ensure gc also drops fn metadata
			require.False(t, r.Exists(q.u.kg.FnMetadata(*p.FunctionID)))

			err = q.PartitionRequeue(ctx, &p, time.Now().Add(time.Minute), false)
			require.Equal(t, ErrPartitionNotFound, err)
		})

		t.Run("Requeueing a paused partition does not affect the partition's pause state", func(t *testing.T) {
			_, err := q.EnqueueItem(ctx, QueueItem{FunctionID: idA}, now)
			require.NoError(t, err)

			_, _, err = q.PartitionLease(ctx, &QueuePartition{FunctionID: &idA}, time.Minute)
			require.NoError(t, err)

			err = q.SetFunctionPaused(ctx, idA, true)
			require.NoError(t, err)

			err = q.PartitionRequeue(ctx, &p, next, true)
			require.NoError(t, err)

			fnMeta := getFnMetadata(t, r, idA)
			require.True(t, fnMeta.Paused)
		})

		// We no longer delete queues on requeue when the concurrency queue is not empty;  this should happen on a final dequeue.
		t.Run("Does not garbage collect the partition with a non-empty concurrency queue", func(t *testing.T) {
			r.FlushAll()

			now := time.Now()
			next = now.Add(10 * time.Second)

			qi, err := q.EnqueueItem(ctx, QueueItem{FunctionID: idA}, now)
			require.NoError(t, err)

			requirePartitionScoreEquals(t, r, &idA, now)

			// Move the queue item to the concurrency (in-progress) queue
			_, err = q.Lease(ctx, p, qi, 10*time.Second, q.clock.Now(), nil)
			require.NoError(t, err)

			next = now.Add(time.Hour)

			// Requeuing cannot gc until queue item finishes processing
			err = q.PartitionRequeue(ctx, &p, next, false)
			require.NoError(t, err)

			// So the partition metadata should still exist
			loaded := getDefaultPartition(t, r, idA)
			require.Equal(t, &idA, loaded.FunctionID)
		})
	})

	t.Run("Custom concurrency keys", func(t *testing.T) {
		t.Run("For account-scoped partition keys", func(t *testing.T) {
			r.FlushAll()

			fnID, acctID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("fn")), uuid.NewSHA1(uuid.NameSpaceDNS, []byte("acct"))

			item := QueueItem{
				FunctionID: fnID,
				Data: osqueue.Item{
					Identifier: state.Identifier{
						AccountID: acctID,
					},
					CustomConcurrencyKeys: []state.CustomConcurrency{
						{
							Key: util.ConcurrencyKey(
								enums.ConcurrencyScopeAccount,
								acctID,
								"test-plz",
							),
							Limit: 1,
						},
					},
				},
			}

			p := q.ItemPartitions(ctx, item)[0]

			require.Equal(t, "{queue}:concurrency:custom:a:4d59bf95-28b6-5423-b1a8-604046826e33:3cwxlkg53rr2c", p.concurrencyKey(q.u.kg))

			item, err := q.EnqueueItem(ctx, item, now)
			require.NoError(t, err)

			t.Run("Uses the next job item's time when requeueing with another job", func(t *testing.T) {
				requireGlobalPartitionScore(t, r, p.zsetKey(q.u.kg), now)
				next := now.Add(time.Hour)
				err := q.PartitionRequeue(ctx, &p, next, false)
				require.NoError(t, err)
				// This should still be now(), as we're not forcing "next" and the earliest job is still now.
				requireGlobalPartitionScore(t, r, p.zsetKey(q.u.kg), now)
			})

			t.Run("Forces a custom partition with `force` set to true", func(t *testing.T) {
				requireGlobalPartitionScore(t, r, p.zsetKey(q.u.kg), now)
				next := now.Add(time.Hour)
				err := q.PartitionRequeue(ctx, &p, next, true)
				require.NoError(t, err)
				requireGlobalPartitionScore(t, r, p.zsetKey(q.u.kg), next)
			})

			t.Run("Sets back to next job with force: false", func(t *testing.T) {
				err := q.PartitionRequeue(ctx, &p, time.Now(), false)
				require.NoError(t, err)
				requireGlobalPartitionScore(t, r, p.zsetKey(q.u.kg), now)
			})

			t.Run("It doesn't dequeue the partition with an in-progress job", func(t *testing.T) {
				id, err := q.Lease(ctx, p, item, 10*time.Second, q.clock.Now(), nil)
				require.NoError(t, err)
				require.NotNil(t, id)

				next := now.Add(time.Minute)

				err = q.PartitionRequeue(ctx, &p, next, false)
				require.NoError(t, err)
				requireGlobalPartitionScore(t, r, p.zsetKey(q.u.kg), next)

				t.Run("With an empty queue the zset is deleted", func(t *testing.T) {
					err := q.Dequeue(ctx, p, item)
					require.NoError(t, err)
					err = q.PartitionRequeue(ctx, &p, next, false)
					require.Error(t, ErrPartitionGarbageCollected, err)
				})
			})
		})
	})
}

func TestQueueFunctionPause(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	q := NewQueue(
		NewQueueClient(rc, QueueDefaultKey),
		WithPriorityFinder(func(_ context.Context, _ QueuePartition) uint {
			return PriorityDefault
		}),
	)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	idA := uuid.New()
	_, err = q.EnqueueItem(ctx, QueueItem{FunctionID: idA}, now)
	require.NoError(t, err)

	err = q.SetFunctionPaused(ctx, idA, true)
	require.NoError(t, err)

	fnMeta := getFnMetadata(t, r, idA)
	require.True(t, fnMeta.Paused)

	err = q.SetFunctionPaused(ctx, idA, false)
	require.NoError(t, err)

	fnMeta = getFnMetadata(t, r, idA)
	require.False(t, fnMeta.Paused)
}

/*
TODO
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
		WithPriorityFinder(func(_ context.Context, _ QueuePartition) uint {
			return priority
		}),
	)
	ctx := context.Background()

	_, err = q.EnqueueItem(ctx, QueueItem{FunctionID: idA}, now)
	require.NoError(t, err)

	first := getDefaultPartition(t, r, idA)
	require.Equal(t, first.Priority, PriorityMin)

	t.Run("It updates priority", func(t *testing.T) {
		priority = PriorityMax
		err = q.PartitionReprioritize(ctx, idA.String(), PriorityMax)
		require.NoError(t, err)
		second := getDefaultPartition(t, r, idA)
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

		fnMeta := getFnMetadata(t, r, idA)
		require.True(t, fnMeta.Paused)
	})
}
*/

func TestQueueRequeueByJobID(t *testing.T) {
	ctx := context.Background()
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	q := NewQueue(NewQueueClient(rc, QueueDefaultKey))
	q.pf = func(ctx context.Context, p QueuePartition) uint {
		return PriorityMin
	}
	q.concurrencyLimitGetter = func(ctx context.Context, p QueuePartition) PartitionConcurrencyLimits {
		return PartitionConcurrencyLimits{100, 100, 100}
	}
	q.itemIndexer = QueueItemIndexerFunc
	q.clock = clockwork.NewRealClock()

	wsA := uuid.New()

	t.Run("Failure cases", func(t *testing.T) {

		t.Run("It fails with a non-existent job ID for an existing partition", func(t *testing.T) {
			r.FlushDB()

			jid := "yeee"
			item := QueueItem{
				ID:          jid,
				FunctionID:  wsA,
				WorkspaceID: wsA,
			}
			_, err := q.EnqueueItem(ctx, item, time.Now().Add(time.Second))
			require.NoError(t, err)

			err = q.RequeueByJobID(ctx, "no bruv", time.Now().Add(5*time.Second))
			require.NotNil(t, err)
		})

		t.Run("It fails if the job is leased", func(t *testing.T) {
			r.FlushDB()

			jid := "leased"
			item := QueueItem{
				ID:          jid,
				FunctionID:  wsA,
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

			err = q.RequeueByJobID(ctx, jid, time.Now().Add(5*time.Second))
			require.NotNil(t, err)
		})
	})

	t.Run("It requeues the job", func(t *testing.T) {
		r.FlushDB()

		jid := "requeue-plz"
		at := time.Now().Add(time.Second).Truncate(time.Millisecond)
		item := QueueItem{
			ID:          jid,
			FunctionID:  wsA,
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
		err = q.RequeueByJobID(ctx, jid, next)
		require.Nil(t, err, r.Dump())

		t.Run("It updates the queue's At time", func(t *testing.T) {
			found, err := q.Peek(ctx, &QueuePartition{FunctionID: &wsA}, at.Add(10*time.Second), 5)
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
				FunctionID:  wsA,
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
			FunctionID:  wsA,
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
		err = q.RequeueByJobID(ctx, jid, next)
		require.Nil(t, err, r.Dump())

		t.Run("The earliest time is still 'at' for the partition after requeueing", func(t *testing.T) {
			score, err := r.ZScore(q.u.kg.GlobalPartitionIndex(), wsA.String())
			require.NoError(t, err)
			require.EqualValues(t, at.Unix(), int64(score), r.Dump())
		})

		t.Run("It updates the queue's At time", func(t *testing.T) {
			found, err := q.Peek(ctx, &QueuePartition{FunctionID: &wsA}, at.Add(30*time.Second), 5)
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
				FunctionID:  wsA,
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
			FunctionID:  wsA,
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
		err = q.RequeueByJobID(ctx, jid, next)
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
		pf: func(ctx context.Context, p QueuePartition) uint {
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

// TestGuaranteedCapacity covers the basics of guaranteed capacity;  we assert that function enqueues
// upsert guaranteed capacity appropriately, and that leasing accounts works
func TestGuaranteedCapacity(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()
	ctx := context.Background()

	accountId := uuid.New()
	enableGuaranteedCapacity := true // indicate whether to enable guaranteed capacity in tests
	guaranteedCapacity := &GuaranteedCapacity{
		Scope:              enums.GuaranteedCapacityScopeAccount,
		AccountID:          accountId,
		GuaranteedCapacity: 1,
	}

	sf := func(ctx context.Context, accountId uuid.UUID) *GuaranteedCapacity {
		if !enableGuaranteedCapacity {
			return nil
		}
		return guaranteedCapacity
	}
	q := NewQueue(
		NewQueueClient(rc, QueueDefaultKey),
		WithRunMode(QueueRunMode{
			Account:            true,
			GuaranteedCapacity: true,
		}),
		WithGuaranteedCapacityFinder(sf),
	)
	require.NotNil(t, sf(ctx, accountId))

	t.Run("QueueItem with guaranteed capacity", func(t *testing.T) {

		// NOTE: Times for guaranteed capacity or global pointers cannot be <= now.
		// Because of this, tests start with the earliest item 1 hour ahead of now so that
		// we can appropriately test enqueueing earlier items adjust pointer times.

		t.Run("Basic enqueue", func(t *testing.T) {
			at := time.Now().Truncate(time.Second).Add(time.Hour)
			_, err := q.EnqueueItem(ctx, QueueItem{
				ID: "foo",
				Data: osqueue.Item{
					Identifier: state.Identifier{
						AccountID: accountId,
					},
				},
			}, at)
			require.NoError(t, err, "guaranteed capacity enqueue should succeed")

			t.Run("Enqueueing creates an item in the guaranteed capacity map", func(t *testing.T) {
				keys, err := r.HKeys(q.u.kg.GuaranteedCapacityMap())
				require.NoError(t, err)
				require.Equal(t, 1, len(keys))

				serialized := r.HGet(q.u.kg.GuaranteedCapacityMap(), guaranteedCapacity.Key())
				actual := &GuaranteedCapacity{}
				err = json.Unmarshal([]byte(serialized), actual)
				require.NoError(t, err)
				require.EqualValues(t, *guaranteedCapacity, *actual)
			})

			t.Run("enqueueing another item in the same account doesn't duplicate guaranteed capacity item", func(t *testing.T) {
				_, err := q.EnqueueItem(ctx, QueueItem{
					Data: osqueue.Item{
						Identifier: state.Identifier{
							AccountID: accountId,
						},
					},
				}, at.Add(time.Minute))
				require.NoError(t, err)

				keys, err := r.HKeys(q.u.kg.GuaranteedCapacityMap())
				require.NoError(t, err)
				require.Equal(t, 1, len(keys))

				serialized := r.HGet(q.u.kg.GuaranteedCapacityMap(), guaranteedCapacity.Key())
				actual := &GuaranteedCapacity{}
				err = json.Unmarshal([]byte(serialized), actual)
				require.NoError(t, err)
				require.EqualValues(t, *guaranteedCapacity, *actual)
			})
		})

		t.Run("guaranteed capacity is updated when enqueueing, if already exists", func(t *testing.T) {
			serialized := r.HGet(q.u.kg.GuaranteedCapacityMap(), guaranteedCapacity.Key())
			first := &GuaranteedCapacity{}
			err = json.Unmarshal([]byte(serialized), first)
			require.NoError(t, err)
			require.EqualValues(t, *guaranteedCapacity, *first)

			// Enqueue again with a capacity of 1
			guaranteedCapacity.GuaranteedCapacity = guaranteedCapacity.GuaranteedCapacity + 1
			_, err = q.EnqueueItem(ctx, QueueItem{
				Data: osqueue.Item{
					Identifier: state.Identifier{
						AccountID: accountId,
					},
				},
			}, time.Now())

			serialized = r.HGet(q.u.kg.GuaranteedCapacityMap(), guaranteedCapacity.Key())
			updated := &GuaranteedCapacity{}
			err = json.Unmarshal([]byte(serialized), updated)
			require.NoError(t, err)
			require.NotEqualValues(t, *first, *updated)
			require.EqualValues(t, *guaranteedCapacity, *updated)
		})

		t.Run("disabled guaranteed capacity is removed when enqueueing, if already exists", func(t *testing.T) {
			serialized := r.HGet(q.u.kg.GuaranteedCapacityMap(), guaranteedCapacity.Key())
			first := &GuaranteedCapacity{}
			err = json.Unmarshal([]byte(serialized), first)
			require.NoError(t, err)
			require.EqualValues(t, *guaranteedCapacity, *first)

			exists, err := rc.Do(ctx, rc.B().Hexists().Key(q.u.kg.GuaranteedCapacityMap()).Field(guaranteedCapacity.Key()).Build()).AsBool()
			require.NoError(t, err)
			require.True(t, exists)

			enableGuaranteedCapacity = false
			_, err = q.EnqueueItem(ctx, QueueItem{
				Data: osqueue.Item{
					Identifier: state.Identifier{
						AccountID: accountId,
					},
				},
			}, time.Now())
			require.NoError(t, err)

			exists, err = rc.Do(ctx, rc.B().Hexists().Key(q.u.kg.GuaranteedCapacityMap()).Field(guaranteedCapacity.Key()).Build()).AsBool()
			require.NoError(t, err)
			require.False(t, exists, r.Dump())
		})
	})
}

func TestAccountLease(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()
	ctx := context.Background()

	sf := func(ctx context.Context, accountId uuid.UUID) *GuaranteedCapacity {
		return &GuaranteedCapacity{
			Scope:              enums.GuaranteedCapacityScopeAccount,
			AccountID:          accountId,
			GuaranteedCapacity: 1,
		}
	}
	q := NewQueue(NewQueueClient(rc, QueueDefaultKey), WithGuaranteedCapacityFinder(sf))

	t.Run("Leasing an account without guaranteed capacity fails", func(t *testing.T) {
		shard := sf(ctx, uuid.UUID{})
		leaseID, err := q.leaseAccount(ctx, shard, 2*time.Second, 1)
		require.Nil(t, leaseID, "Got lease ID: %v", leaseID)
		require.NotNil(t, err)
		require.ErrorContains(t, err, "guaranteed capacity not found")
	})

	// Ensure guaranteed capacity exists
	idA, idB := uuid.New(), uuid.New()

	_, err = q.EnqueueItem(ctx, QueueItem{Data: osqueue.Item{Identifier: state.Identifier{AccountID: idA}}}, time.Now())
	require.NoError(t, err)
	exists, err := rc.Do(ctx, rc.B().Hexists().Key(q.u.kg.GuaranteedCapacityMap()).Field(GuaranteedCapacity{AccountID: idA}.Key()).Build()).AsBool()
	require.NoError(t, err)
	require.True(t, exists, r.Dump())

	_, err = q.EnqueueItem(ctx, QueueItem{Data: osqueue.Item{Identifier: state.Identifier{AccountID: idB}}}, time.Now())
	require.NoError(t, err)
	exists, err = rc.Do(ctx, rc.B().Hexists().Key(q.u.kg.GuaranteedCapacityMap()).Field(GuaranteedCapacity{AccountID: idB}.Key()).Build()).AsBool()
	require.NoError(t, err)
	require.True(t, exists, r.Dump())

	miniredis.DumpMaxLineLen = 1024

	t.Run("Leasing out-of-bounds fails", func(t *testing.T) {
		// At the beginning, no shards have been leased.  Leasing a shard
		// with an index of >= 1 should fail.
		guaranteedCapacity := sf(ctx, idA)
		leaseID, err := q.leaseAccount(ctx, guaranteedCapacity, 2*time.Second, 1)
		require.Nil(t, leaseID, "Got lease ID: %v", leaseID)
		require.NotNil(t, err)
		require.ErrorContains(t, err, "lease index is too high", r.Dump())
	})

	t.Run("Leasing an account works", func(t *testing.T) {
		shard := sf(ctx, idA)

		t.Run("Basic lease", func(t *testing.T) {
			leaseID, err := q.leaseAccount(ctx, shard, 1*time.Second, 0)
			require.NotNil(t, leaseID, "Didn't get a lease ID for a basic lease")
			require.Nil(t, err)
		})

		t.Run("Leasing a subsequent index works", func(t *testing.T) {
			leaseID, err := q.leaseAccount(ctx, shard, 8*time.Second, 1) // Same length as the lease below, after wait
			require.NotNil(t, leaseID, "Didn't get a lease ID for a secondary lease")
			require.Nil(t, err)
		})

		t.Run("Leasing an index with an expired lease works", func(t *testing.T) {
			// In this test, we have two leases — but one expires with the wait.  This first lease
			// is no longer valid, so leasing with an index of (1) should succeed.
			<-time.After(2 * time.Second) // Wait a few seconds so that time.Now() in the call works.
			r.FastForward(2 * time.Second)
			leaseID, err := q.leaseAccount(ctx, shard, 10*time.Second, 1)
			require.NotNil(t, leaseID)
			require.Nil(t, err)

			// This leaves us with two valid leases.
		})

		t.Run("Leasing an already leased index fails", func(t *testing.T) {
			leaseID, err := q.leaseAccount(ctx, shard, 2*time.Second, 1)
			require.Nil(t, leaseID, "got a lease ID for an existing lease")
			require.NotNil(t, err)
			require.ErrorContains(t, err, "index is already leased")
		})

		t.Run("Leasing a second account works", func(t *testing.T) {
			// Try another shard name with an index of 0.
			leaseID, err := q.leaseAccount(ctx, sf(ctx, idB), 2*time.Second, 0)
			require.NotNil(t, leaseID)
			require.Nil(t, err)
		})
	})

	r.FlushAll()

	t.Run("Renewing account leases", func(t *testing.T) {
		// Ensure that enqueueing succeeds to make the shard.
		_, err = q.EnqueueItem(ctx, QueueItem{WorkspaceID: idA, Data: osqueue.Item{Identifier: state.Identifier{AccountID: idA}}}, time.Now())
		require.Nil(t, err)

		guaranteedCapacity := sf(ctx, idA)
		leaseID, err := q.leaseAccount(ctx, guaranteedCapacity, 1*time.Second, 0)
		require.NotNil(t, leaseID, "could not lease account", r.Dump())
		require.Nil(t, err)

		t.Run("Current leases succeed", func(t *testing.T) {
			leaseID, err = q.renewAccountLease(ctx, guaranteedCapacity, 2*time.Second, *leaseID)
			require.NotNil(t, leaseID, "did not get a new lease when renewing", r.Dump())
			require.Nil(t, err)
		})

		t.Run("Expired leases fail", func(t *testing.T) {
			<-time.After(3 * time.Second)
			r.FastForward(3 * time.Second)

			leaseID, err := q.renewAccountLease(ctx, guaranteedCapacity, 2*time.Second, *leaseID)
			require.ErrorContains(t, err, "lease not found")
			require.Nil(t, leaseID)
		})

		t.Run("Invalid lease IDs fail", func(t *testing.T) {
			leaseID, err := q.renewAccountLease(ctx, guaranteedCapacity, 2*time.Second, ulid.MustNew(ulid.Now(), rand.Reader))
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
			FunctionID: idA,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					WorkflowID: idA,
				},
				Throttle: throttle,
			},
		}, clock.Now())
		r.NoError(err)

		ab, err := q.EnqueueItem(ctx, QueueItem{
			FunctionID: idA,
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
				FunctionID: idB,
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
				FunctionID: idA,
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

func getDefaultPartition(t *testing.T, r *miniredis.Miniredis, id uuid.UUID) QueuePartition {
	t.Helper()
	kg := &queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
	val := r.HGet(kg.PartitionItem(), id.String())
	qp := QueuePartition{}
	err := json.Unmarshal([]byte(val), &qp)
	require.NoError(t, err)
	return qp
}

func getGlobalAccounts(t *testing.T, rc rueidis.Client) []string {
	t.Helper()

	kg := &queueKeyGenerator{queueDefaultKey: QueueDefaultKey}

	resp := rc.Do(context.Background(), rc.
		B().
		Zrangebyscore().
		Key(kg.GlobalAccountIndex()).
		Min("0").
		Max("+inf").
		Build(),
	)
	require.NoError(t, resp.Error())

	strSlice, err := resp.AsStrSlice()
	require.NoError(t, err)

	return strSlice
}

func getAccountPartitions(t *testing.T, rc rueidis.Client, accountId uuid.UUID) []string {
	t.Helper()

	kg := &queueKeyGenerator{queueDefaultKey: QueueDefaultKey}

	resp := rc.Do(context.Background(), rc.
		B().
		Zrangebyscore().
		Key(kg.AccountPartitionIndex(accountId)).
		Min("0").
		Max("+inf").
		Build(),
	)
	require.NoError(t, resp.Error())

	strSlice, err := resp.AsStrSlice()
	require.NoError(t, err)

	return strSlice
}

func getSystemPartition(t *testing.T, r *miniredis.Miniredis, name string) QueuePartition {
	t.Helper()
	kg := &queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
	val := r.HGet(kg.PartitionItem(), name)
	require.NotEmpty(t, val, "expected item to be set", r.Dump())
	qp := QueuePartition{}
	err := json.Unmarshal([]byte(val), &qp)
	require.NoError(t, err, "expected item to be valid json")
	require.True(t, qp.IsSystem())
	return qp
}

func getPartition(t *testing.T, r *miniredis.Miniredis, pType enums.PartitionType, id uuid.UUID, optionalHash ...string) QueuePartition {
	t.Helper()
	hash := ""
	if len(optionalHash) > 0 {
		hash = optionalHash[0]
	}
	kg := &queueKeyGenerator{queueDefaultKey: QueueDefaultKey}

	key := kg.PartitionQueueSet(pType, id.String(), hash)
	if pType == enums.PartitionTypeDefault {
		key = id.String()
	}

	val := r.HGet(kg.PartitionItem(), key)

	items, _ := r.HKeys(kg.PartitionItem())

	require.NotEmpty(t, val, "couldn't find partition in map with key:\n--> %s\nhave:\n%v", key, strings.Join(items, "\n"))
	qp := QueuePartition{}
	err := json.Unmarshal([]byte(val), &qp)
	require.NoError(t, err)
	return qp
}

func getFnMetadata(t *testing.T, r *miniredis.Miniredis, id uuid.UUID) FnMetadata {
	t.Helper()
	kg := &queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
	valJSON, err := r.Get(kg.FnMetadata(id))
	require.NoError(t, err)
	retv := FnMetadata{}
	err = json.Unmarshal([]byte(valJSON), &retv)
	require.NoError(t, err)
	return retv
}

func requireItemScoreEquals(t *testing.T, r *miniredis.Miniredis, item QueueItem, expected time.Time) {
	t.Helper()
	kg := &queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
	score, err := r.ZScore(kg.FnQueueSet(item.FunctionID.String()), item.ID)
	parsed := time.UnixMilli(int64(score))
	require.NoError(t, err)
	require.WithinDuration(t, expected.Truncate(time.Millisecond), parsed, 15*time.Millisecond)
}

func requirePartitionItemScoreEquals(t *testing.T, r *miniredis.Miniredis, keyPartitionIndex string, qp QueuePartition, expected time.Time) {
	t.Helper()
	score, err := r.ZScore(keyPartitionIndex, qp.ID)
	require.NotZero(t, score, r.Dump(), qp.ID)

	parsed := time.Unix(int64(score), 0) // score is in seconds :)
	require.NoError(t, err)
	require.WithinDuration(t, expected.Truncate(time.Millisecond), parsed, 15*time.Millisecond, r.Dump())
}

// requireGlobalPartitionScore is used to check scores for any partition, including custom partitions.
func requireGlobalPartitionScore(t *testing.T, r *miniredis.Miniredis, id string, expected time.Time) {
	t.Helper()
	kg := &queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
	score, err := r.ZScore(kg.GlobalPartitionIndex(), id)
	parsed := time.Unix(int64(score), 0)
	require.NoError(t, err)
	require.WithinDuration(t, expected.Truncate(time.Second), parsed, time.Millisecond, r.Dump())
}

// requireAccountScoreEquals is used to check scores for any account
func requireAccountScoreEquals(t *testing.T, r *miniredis.Miniredis, accountId uuid.UUID, expected time.Time) {
	t.Helper()
	kg := &queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
	score, err := r.ZScore(kg.GlobalAccountIndex(), accountId.String())
	parsed := time.Unix(int64(score), 0)
	require.NoError(t, err)
	require.WithinDuration(t, expected.Truncate(time.Second), parsed, time.Millisecond, r.Dump())
}

// requirePartitionScoreEquals is used to check scores for fn partitions (queues for function IDs)
func requirePartitionScoreEquals(t *testing.T, r *miniredis.Miniredis, wid *uuid.UUID, expected time.Time) {
	t.Helper()
	requireGlobalPartitionScore(t, r, wid.String(), expected)
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

func createConcurrencyKey(scope enums.ConcurrencyScope, scopeID uuid.UUID, value string, limit int) state.CustomConcurrency {
	// Users always define concurrency on the funciton level.  We then evaluate these "keys", eg:
	//
	// concurrency: [
	//   {
	//     "key": "event.data.user_id",
	//     "limit": 10
	//   }
	// ]
	//
	// This replicates that logic.

	// Evaluate expects that value is either `event.data.user_id` - a JSON path - or a quoted string.
	// Always quote for these tests.
	value = strconv.Quote(value)

	c := inngest.Concurrency{
		Key:   &value,
		Scope: scope,
	}
	hash := c.Evaluate(context.Background(), scopeID, map[string]any{})

	return state.CustomConcurrency{
		Key:   hash,
		Limit: limit,
		Hash:  value,
	}
}

func int64ptr(i int64) *int64 { return &i }
