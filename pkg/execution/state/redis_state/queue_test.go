package redis_state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/util"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/assert"
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
	runID := ulid.MustNew(uint64(start.UnixMilli()), rand.Reader)

	// What we care about:  Items are promoted IFF the scheduled at time <=
	// time.Now().Add(consts.FutureAtLimit).

	kinds := []string{osqueue.KindEdge, osqueue.KindSleep, osqueue.KindStart, osqueue.KindEdgeError}
	for _, kind := range kinds {

		t.Run(fmt.Sprintf("%s: within promotion timerange", kind), func(t *testing.T) {
			// Enqueue a job now, and ensure that it is fudged and promoted.
			item := osqueue.QueueItem{
				AtMS: time.Now().UnixMilli(),
				Data: osqueue.Item{
					Kind: kind,
					Identifier: state.Identifier{
						RunID: runID,
					},
				},
			}

			actual := item.Score(time.Now())
			require.Equal(t, start.UnixMilli(), actual, kind)
		})

		t.Run(fmt.Sprintf("%s: outside promotion timerange", kind), func(t *testing.T) {
			atMS := time.Now().Add(consts.FutureAtLimit * 2).UnixMilli()
			item := osqueue.QueueItem{
				AtMS: atMS,
				Data: osqueue.Item{
					Kind: kind,
					Identifier: state.Identifier{
						RunID: runID,
					},
				},
			}
			actual := item.Score(time.Now())
			require.Equal(t, atMS, actual, kind)
		})

		t.Run(fmt.Sprintf("%s: with priority factors", kind), func(t *testing.T) {
			atMS := time.Now().UnixMilli()
			item := osqueue.QueueItem{
				AtMS: atMS,
				Data: osqueue.Item{
					Kind: kind,
					Identifier: state.Identifier{
						RunID:          runID,
						PriorityFactor: int64ptr(-60),
					},
				},
			}

			expected := start.Add(60 * time.Second).UnixMilli()
			if kind == osqueue.KindSleep {
				// NOT FUDGED.  Sleeps do not move with fudge factors.
				expected = start.UnixMilli()
			}

			actual := item.Score(time.Now())
			require.Equal(t, expected, actual, kind)
		})

	}

	t.Run("Sleep with priority factor does nothing", func(t *testing.T) {
		// A job enqueued in an hour should always be enqueued in an hour
		// even with a priority factor.
		atMS := time.Now().Add(time.Hour).UnixMilli()
		item := osqueue.QueueItem{
			AtMS: atMS,
			Data: osqueue.Item{
				Kind: osqueue.KindSleep,
				Identifier: state.Identifier{
					RunID:          runID,
					PriorityFactor: int64ptr(-60),
				},
			},
		}

		actual := item.Score(time.Now())
		require.Equal(t, atMS, actual)
	})

	// Non-promotable kinds
	kinds = []string{
		osqueue.KindDebounce,
		osqueue.KindScheduleBatch,
		osqueue.KindQueueMigrate,
		osqueue.KindPauseBlockFlush,
		osqueue.KindJobPromote,
	}
	for _, kind := range kinds {
		t.Run(fmt.Sprintf("%s: within promotion timerange", kind), func(t *testing.T) {
			// Enqueue a job now, and ensure that it is fudged and promoted.
			atMS := time.Now().UnixMilli()
			item := osqueue.QueueItem{
				AtMS: time.Now().UnixMilli(),
				Data: osqueue.Item{
					Kind: kind,
					Identifier: state.Identifier{
						RunID: runID,
					},
				},
			}

			actual := item.Score(time.Now())
			require.Equal(t, atMS, actual, kind)
		})

		t.Run(fmt.Sprintf("%s: outside promotion timerange", kind), func(t *testing.T) {
			atMS := time.Now().Add(consts.FutureAtLimit * 2).UnixMilli()
			item := osqueue.QueueItem{
				AtMS: atMS,
				Data: osqueue.Item{
					Kind: kind,
					Identifier: state.Identifier{
						RunID: runID,
					},
				},
			}
			actual := item.Score(time.Now())
			require.Equal(t, atMS, actual, kind)
		})
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
			qi := &osqueue.QueueItem{}
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

	q := NewQueue(QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName})
	ctx := context.Background()

	start := time.Now().Truncate(time.Second)

	accountId := uuid.New()

	t.Run("It enqueues an item", func(t *testing.T) {
		id := uuid.New()

		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			FunctionID: id,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					AccountID: accountId,
				},
			},
		}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		require.NotEqual(t, item.ID, ulid.Zero)
		require.Equal(t, time.UnixMilli(item.WallTimeMS).Truncate(time.Second), start)

		// Ensure that our data is set up correctly.
		found := getQueueItem(t, r, item.ID)
		require.Equal(t, item, found)

		// Ensure the partition is inserted.
		qp := getDefaultPartition(t, r, item.FunctionID)
		require.Equal(t, accountId.String(), qp.AccountID.String())
		require.Equal(t, QueuePartition{
			ID:         item.FunctionID.String(),
			FunctionID: &item.FunctionID,
			AccountID:  accountId,
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

		// New key queue data structures should not exist with the flag being toggled off
		backlog := q.ItemBacklog(ctx, item)
		require.NotEmpty(t, backlog.BacklogID)

		shadowPartition := q.ItemShadowPartition(ctx, item)
		require.NotEmpty(t, shadowPartition.PartitionID)

		require.False(t, r.Exists(kg.BacklogMeta()))
		require.False(t, r.Exists(kg.BacklogSet(backlog.BacklogID)))
		require.False(t, r.Exists(kg.ShadowPartitionMeta()))
		require.False(t, r.Exists(kg.ShadowPartitionSet(shadowPartition.PartitionID)), r.Keys())
		require.False(t, r.Exists(kg.GlobalShadowPartitionSet()))
	})

	t.Run("It sets the right item score", func(t *testing.T) {
		start := time.Now()

		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		requireItemScoreEquals(t, r, item, start)
	})

	t.Run("It enqueues an item in the future", func(t *testing.T) {
		// Empty the DB.
		r.FlushAll()

		at := time.Now().Add(time.Hour).Truncate(time.Second)

		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			Data: osqueue.Item{
				Identifier: state.Identifier{
					AccountID: accountId,
				},
			},
		}, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		// Ensure the partition is inserted, and the earliest time is still
		// the start time.
		qp := getDefaultPartition(t, r, item.FunctionID)
		require.Equal(t, QueuePartition{
			ID:         item.FunctionID.String(),
			FunctionID: &item.FunctionID,
			AccountID:  accountId,
		}, qp)

		// Ensure that the zscore did not change.
		keys, err := r.ZMembers(q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex())
		require.NoError(t, err)
		require.Equal(t, 1, len(keys))

		score, err := r.ZScore(q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex(), keys[0])
		require.NoError(t, err)
		require.EqualValues(t, at.Unix(), score)

		score, err = r.ZScore(q.primaryQueueShard.RedisClient.kg.AccountPartitionIndex(accountId), keys[0])
		require.NoError(t, err)
		require.EqualValues(t, at.Unix(), score)

		score, err = r.ZScore(q.primaryQueueShard.RedisClient.kg.GlobalAccountIndex(), accountId.String())
		require.NoError(t, err)
		require.EqualValues(t, at.Unix(), score)
	})

	t.Run("Updates partition vesting time to earlier times", func(t *testing.T) {
		now := time.Now()
		at := now.Add(-10 * time.Minute).Truncate(time.Second)

		// Note: This will reuse the existing partition (zero UUID) from the step above
		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			Data: osqueue.Item{
				Identifier: state.Identifier{
					AccountID: accountId,
				},
			},
		}, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		// Ensure the partition is inserted, and the earliest time is updated
		// inside the partition item.
		qp := getDefaultPartition(t, r, item.FunctionID)
		require.Equal(t, QueuePartition{
			ID:         item.FunctionID.String(),
			FunctionID: &item.FunctionID,
			AccountID:  accountId,
		}, qp, "queue partition does not match")

		// Assert that the zscore was changed to this earliest timestamp.
		keys, err := r.ZMembers(q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex())
		require.NoError(t, err)
		require.Equal(t, 1, len(keys))

		score, err := r.ZScore(q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex(), keys[0])
		require.NoError(t, err)
		require.EqualValues(t, now.Unix(), score)

		score, err = r.ZScore(q.primaryQueueShard.RedisClient.kg.AccountPartitionIndex(accountId), keys[0])
		require.NoError(t, err)
		require.NotZero(t, score)
		require.EqualValues(t, now.Unix(), score, r.Dump())

		score, err = r.ZScore(q.primaryQueueShard.RedisClient.kg.GlobalAccountIndex(), accountId.String())
		require.NoError(t, err)
		require.EqualValues(t, now.Unix(), score)
	})

	t.Run("Adding another workflow ID increases partition set", func(t *testing.T) {
		at := time.Now().Truncate(time.Second)

		accountId := uuid.New()

		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			FunctionID: uuid.New(),
			Data: osqueue.Item{
				Identifier: state.Identifier{
					AccountID: accountId,
				},
			},
		}, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		// Assert that we have two zscores in partition:sorted.
		keys, err := r.ZMembers(q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex())
		require.NoError(t, err)
		require.Equal(t, 2, len(keys))

		// Assert that we have one zscore in accounts:$accountId:partition:sorted.
		keys, err = r.ZMembers(q.primaryQueueShard.RedisClient.kg.AccountPartitionIndex(accountId))
		require.NoError(t, err)
		require.Equal(t, 1, len(keys))

		// Ensure the partition is inserted, and the earliest time is updated
		// inside the partition item.
		qp := getDefaultPartition(t, r, item.FunctionID)
		require.Equal(t, QueuePartition{
			ID:         item.FunctionID.String(),
			FunctionID: &item.FunctionID,
			AccountID:  accountId,
		}, qp)
	})

	t.Run("Stores default indexes", func(t *testing.T) {
		at := time.Now().Truncate(time.Second)
		rid := ulid.MustNew(ulid.Now(), rand.Reader)
		_, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			FunctionID: uuid.New(),
			Data: osqueue.Item{
				Kind: osqueue.KindEdge,
				Identifier: state.Identifier{
					RunID: rid,
				},
			},
		}, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		keys, err := r.ZMembers(fmt.Sprintf("{queue}:idx:run:%s", rid))
		require.NoError(t, err)
		require.Equal(t, 1, len(keys))
	})

	t.Run("Custom concurrency key queues", func(t *testing.T) {
		now := time.Now()
		fnID := uuid.New()

		r.FlushAll()

		t.Run("Single custom key, function scope", func(t *testing.T) {
			// Enqueueing an item
			ck := createConcurrencyKey(enums.ConcurrencyScopeFn, fnID, "test", 1)
			_, _, hash, _ := ck.ParseKey() // get the hash of the "test" string / evaluated input.

			qi := osqueue.QueueItem{
				FunctionID: fnID,
				Data: osqueue.Item{
					CustomConcurrencyKeys: []state.CustomConcurrency{ck},
					Identifier: state.Identifier{
						AccountID: accountId,
					},
				},
			}

			_, partitionCustomConcurrencyKey1, _ := q.ItemPartitions(ctx, q.primaryQueueShard, qi)

			// Enqueue always enqueues to the default partitions - enqueueing to key queues has been disabled for now
			customkeyQueuePartition := QueuePartition{
				ID:                         q.primaryQueueShard.RedisClient.kg.PartitionQueueSet(enums.PartitionTypeConcurrencyKey, fnID.String(), hash),
				PartitionType:              int(enums.PartitionTypeConcurrencyKey),
				ConcurrencyScope:           int(enums.ConcurrencyScopeFn),
				FunctionID:                 &fnID,
				AccountID:                  accountId,
				EvaluatedConcurrencyKey:    ck.Key,
				UnevaluatedConcurrencyHash: ck.Hash,
			}

			assert.Equal(t, customkeyQueuePartition, partitionCustomConcurrencyKey1)

			i, err := q.EnqueueItem(ctx, q.primaryQueueShard, qi, now.Add(10*time.Second), osqueue.EnqueueOpts{})
			require.NoError(t, err)

			// There should be 2 partitions - custom key, and the function
			// level limit.
			items, _ := r.HKeys(q.primaryQueueShard.RedisClient.kg.PartitionItem())
			require.Equal(t, 1, len(items))

			// Concurrency key queue should not exist
			require.False(t, r.Exists(q.primaryQueueShard.RedisClient.kg.PartitionQueueSet(enums.PartitionTypeConcurrencyKey, fnID.String(), hash)))

			accountIds := getGlobalAccounts(t, rc)
			require.Equal(t, 1, len(accountIds))
			require.Contains(t, accountIds, accountId.String())

			apIds := getAccountPartitions(t, rc, accountId)
			require.Equal(t, 1, len(apIds), "expected two account partitions", apIds, r.Dump())

			// workflow partition for backwards compatibility
			require.Contains(t, apIds, fnID.String())

			// We enqueue to the function-specific queue for backwards-compatibility reasons
			defaultPartition := getDefaultPartition(t, r, fnID)
			assert.Equal(t, QueuePartition{
				ID:         fnID.String(),
				FunctionID: &fnID,
				AccountID:  accountId,
			}, defaultPartition)

			mem, err := r.ZMembers(defaultPartition.zsetKey(q.primaryQueueShard.RedisClient.kg))
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

			qi := osqueue.QueueItem{
				FunctionID: fnID,
				Data: osqueue.Item{
					CustomConcurrencyKeys: []state.CustomConcurrency{ckA, ckB},
					Identifier: state.Identifier{
						AccountID: accountId,
					},
				},
			}

			partitionFn, partitionCustomConcurrencyKey1, partitionCustomConcurrencyKey2 := q.ItemPartitions(ctx, q.primaryQueueShard, qi)

			// We enqueue to the function-specific queue for backwards-compatibility reasons
			expectedDefaultPartition := QueuePartition{
				ID:         fnID.String(),
				FunctionID: &fnID,
				AccountID:  accountId,
			}
			assert.Equal(t, expectedDefaultPartition, partitionFn)

			keyQueueA := QueuePartition{
				ID:                         q.primaryQueueShard.RedisClient.kg.PartitionQueueSet(enums.PartitionTypeConcurrencyKey, fnID.String(), hashA),
				PartitionType:              int(enums.PartitionTypeConcurrencyKey),
				ConcurrencyScope:           int(enums.ConcurrencyScopeFn),
				FunctionID:                 &fnID,
				AccountID:                  accountId,
				EvaluatedConcurrencyKey:    ckA.Key,
				UnevaluatedConcurrencyHash: ckA.Hash,
			}
			assert.Equal(t, keyQueueA, partitionCustomConcurrencyKey1)

			keyQueueB := QueuePartition{
				ID:                         q.primaryQueueShard.RedisClient.kg.PartitionQueueSet(enums.PartitionTypeConcurrencyKey, fnID.String(), hashB),
				PartitionType:              int(enums.PartitionTypeConcurrencyKey),
				ConcurrencyScope:           int(enums.ConcurrencyScopeFn),
				FunctionID:                 &fnID,
				AccountID:                  accountId,
				EvaluatedConcurrencyKey:    ckB.Key,
				UnevaluatedConcurrencyHash: ckB.Hash,
			}
			assert.Equal(t, keyQueueB, partitionCustomConcurrencyKey2)

			i, err := q.EnqueueItem(ctx, q.primaryQueueShard, qi, now.Add(10*time.Second), osqueue.EnqueueOpts{})
			require.NoError(t, err)

			// just the default partition
			items, _ := r.HKeys(q.primaryQueueShard.RedisClient.kg.PartitionItem())
			require.Equal(t, 1, len(items))
			require.Contains(t, items, expectedDefaultPartition.ID)

			// We do not expect key queues to be enqueued!
			//concurrencyPartitionA := getPartition(t, r, enums.PartitionTypeConcurrencyKey, fnID, hashA) // nb. also asserts that the partition exists
			//require.Equal(t, keyQueueA, concurrencyPartitionA)
			//
			//concurrencyPartitionB := getPartition(t, r, enums.PartitionTypeConcurrencyKey, fnID, hashB) // nb. also asserts that the partition exists
			//require.Equal(t, keyQueueB, concurrencyPartitionB)

			accountIds := getGlobalAccounts(t, rc)
			require.Equal(t, 1, len(accountIds))
			require.Contains(t, accountIds, accountId.String())

			apIds := getAccountPartitions(t, rc, accountId)
			require.Equal(t, 1, len(apIds))
			require.Contains(t, apIds, expectedDefaultPartition.ID)

			assert.True(t, r.Exists(expectedDefaultPartition.zsetKey(q.primaryQueueShard.RedisClient.kg)), "expected default partition to exist")
			defaultPartition := getDefaultPartition(t, r, fnID)
			assert.Equal(t, expectedDefaultPartition, defaultPartition)

			mem, err := r.ZMembers(defaultPartition.zsetKey(q.primaryQueueShard.RedisClient.kg))
			require.NoError(t, err)
			require.Equal(t, 1, len(mem))
			require.Contains(t, mem, i.ID)

			t.Run("Peeking partitions returns the three partitions", func(t *testing.T) {
				parts, err := q.PartitionPeek(ctx, true, time.Now().Add(time.Hour), 10)
				require.NoError(t, err)
				require.Equal(t, 1, len(parts))
				require.Equal(t, expectedDefaultPartition, *parts[0], "Got: %v", spew.Sdump(parts), r.Dump())
			})
		})
	})

	t.Run("Migrates old partitions to add accountId", func(t *testing.T) {
		r.FlushAll()

		id := uuid.MustParse("baac957a-3aa5-4e42-8c1d-f86dee5d58da")
		envId := uuid.MustParse("e8c0aacd-fcb4-4d5a-b78a-7f0528841543")

		oldPartitionSnapshot := "{\"at\":1723814830,\"p\":6,\"wsID\":\"e8c0aacd-fcb4-4d5a-b78a-7f0528841543\",\"wid\":\"baac957a-3aa5-4e42-8c1d-f86dee5d58da\",\"last\":1723814800026,\"forceAtMS\":0,\"off\":false}"

		r.HSet(q.primaryQueueShard.RedisClient.kg.PartitionItem(), id.String(), oldPartitionSnapshot)
		assert.Equal(t, QueuePartition{
			FunctionID: &id,
			EnvID:      &envId,
			// No accountId is present,
			AccountID: uuid.UUID{},
			LeaseID:   nil,
			Last:      1723814800026,
		}, getPartition(t, r, enums.PartitionTypeDefault, id))

		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			FunctionID: id,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					AccountID: accountId,
				},
			},
		}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		require.NotEqual(t, item.ID, ulid.Zero)
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
	q := NewQueue(QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}, WithIdempotencyTTL(dur))
	ctx := context.Background()

	start := time.Now().Truncate(time.Second)

	t.Run("It enqueues an item only once", func(t *testing.T) {
		i := osqueue.QueueItem{ID: "once"}

		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, i, start, osqueue.EnqueueOpts{})

		require.NoError(t, err)
		require.Equal(t, osqueue.HashID(ctx, "once"), item.ID)
		require.NotEqual(t, i.ID, item.ID)
		found := getQueueItem(t, r, item.ID)
		require.Equal(t, item, found)

		// Ensure we can't enqueue again.
		_, err = q.EnqueueItem(ctx, q.primaryQueueShard, i, start, osqueue.EnqueueOpts{})
		require.Equal(t, ErrQueueItemExists, err)

		// Dequeue
		err = q.Dequeue(ctx, q.primaryQueueShard, item)
		require.NoError(t, err)

		// Ensure we can't enqueue even after dequeue.
		_, err = q.EnqueueItem(ctx, q.primaryQueueShard, i, start, osqueue.EnqueueOpts{})
		require.Equal(t, ErrQueueItemExists, err)

		// Wait for the idempotency TTL to expire
		r.FastForward(dur)

		item, err = q.EnqueueItem(ctx, q.primaryQueueShard, i, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		require.Equal(t, osqueue.HashID(ctx, "once"), item.ID)
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

	q := NewQueue(QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName})
	ctx := context.Background()

	enqueue := func(id uuid.UUID, n int) {
		for i := 0; i < n; i++ {
			_, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: id}, time.Now(), osqueue.EnqueueOpts{})
			if err != nil {
				panic(err)
			}
		}
	}

	for i := 0; i < b.N; i++ {
		id := uuid.New()
		enqueue(id, int(DefaultQueuePeekMax))
		items, err := q.Peek(ctx, &QueuePartition{FunctionID: &id}, time.Now(), DefaultQueuePeekMax)
		if err != nil {
			panic(err)
		}
		if len(items) != int(DefaultQueuePeekMax) {
			panic(fmt.Sprintf("expected %d, got %d", DefaultQueuePeekMax, len(items)))
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
		QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName},
		WithAllowQueueNames(customQueueName),
		WithPartitionConstraintConfigGetter(func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
			return PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					AccountConcurrency:  5000,
					SystemConcurrency:   customTestLimit,
					FunctionConcurrency: 1,
				},
			}
		}),
	)
	ctx := context.Background()

	start := time.Now().Truncate(time.Second)

	id := uuid.New()

	qi := osqueue.QueueItem{
		FunctionID: id,
		Data: osqueue.Item{
			Payload:   json.RawMessage("{\"test\":\"payload\"}"),
			QueueName: &customQueueName,
		},
		QueueName: &customQueueName,
	}

	t.Run("It enqueues an item", func(t *testing.T) {
		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		require.NotEqual(t, item.ID, ulid.Zero)
		require.Equal(t, time.UnixMilli(item.WallTimeMS).Truncate(time.Second), start)

		// Ensure that our data is set up correctly.
		found := getQueueItem(t, r, item.ID)
		require.Equal(t, item, found)

		// Ensure the partition is inserted.
		qp := getSystemPartition(t, r, customQueueName)
		require.Equal(t, QueuePartition{
			ID:            customQueueName,
			PartitionType: int(enums.PartitionTypeDefault),
			QueueName:     &customQueueName,
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
		require.Equal(t, 1, availableCapacity)
	})

	t.Run("peeks partition successfully", func(t *testing.T) {
		qp := getSystemPartition(t, r, customQueueName)

		items, err := q.Peek(ctx, &qp, start, 100)
		require.NoError(t, err)
		require.Equal(t, 1, len(items))
		require.Equal(t, qi.Data.Payload, items[0].Data.Payload)
	})

	t.Run("leases partition items while respecting concurrency", func(t *testing.T) {
		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		require.NotEqual(t, item.ID, ulid.Zero)
		require.Equal(t, time.UnixMilli(item.WallTimeMS).Truncate(time.Second), start)

		item2, err := q.EnqueueItem(ctx, q.primaryQueueShard, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		require.NotEqual(t, item.ID, ulid.Zero)
		require.Equal(t, time.UnixMilli(item.WallTimeMS).Truncate(time.Second), start)

		// Ensure that our data is set up correctly.
		found := getQueueItem(t, r, item.ID)
		require.Equal(t, item, found)

		leaseId, err := q.Lease(ctx, item, time.Second, time.Now(), nil)
		require.NoError(t, err)
		require.NotNil(t, leaseId)

		leaseId, err = q.Lease(ctx, item2, time.Second, time.Now(), nil)
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

		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		require.NotEqual(t, item.ID, ulid.Zero)
		require.Equal(t, time.UnixMilli(item.WallTimeMS).Truncate(time.Second), start)

		qp := getSystemPartition(t, r, customQueueName)

		leaseStart := time.Now()
		leaseExpires := q.clock.Now().Add(time.Second)

		itemCountMatches := func(num int) {
			zsetKey := qp.zsetKey(q.primaryQueueShard.RedisClient.kg)
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
				Key(qp.concurrencyKey(q.primaryQueueShard.RedisClient.kg)).
				Min("-inf").
				Max("+inf").
				Build()).AsStrSlice()
			require.NoError(t, err)
			assert.Equal(t, num, len(items), "expected %d items in the concurrency queue", num, r.Dump())
		}

		itemCountMatches(1)
		concurrencyItemCountMatches(0)

		leaseId, err := q.Lease(ctx, item, time.Second, leaseStart, nil)
		require.NoError(t, err)
		require.NotNil(t, leaseId)

		itemCountMatches(0)
		concurrencyItemCountMatches(1)

		// wait til leases are expired
		<-time.After(2 * time.Second)
		require.True(t, time.Now().After(leaseExpires))

		incompatibleConcurrencyIndexItem := q.primaryQueueShard.RedisClient.kg.Concurrency("p", customQueueName)
		compatibleConcurrencyIndexItem := customQueueName

		indexMembers, err := r.ZMembers(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex())
		require.NoError(t, err)
		require.Equal(t, 1, len(indexMembers))
		require.Contains(t, indexMembers, compatibleConcurrencyIndexItem)

		requeued, err := q.Scavenge(ctx, ScavengePeekSize)
		require.NoError(t, err)
		assert.Equal(t, 1, requeued, "expected one item with expired leases to be requeued by scavenge", r.Dump())

		itemCountMatches(1)
		concurrencyItemCountMatches(0)

		indexItems, err := rc.Do(ctx, rc.B().Zcard().Key(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex()).Build()).AsInt64()
		require.NoError(t, err)
		assert.Equal(t, 0, int(indexItems), "expected no items in the concurrency index", r.Dump())

		newConcurrencyQueueItems, err := rc.Do(ctx, rc.B().Zcard().Key(incompatibleConcurrencyIndexItem).Build()).AsInt64()
		require.NoError(t, err)
		assert.Equal(t, 0, int(newConcurrencyQueueItems), "expected no items in the new concurrency queue", r.Dump())

		oldConcurrencyQueueItems, err := rc.Do(ctx, rc.B().Zcard().Key(compatibleConcurrencyIndexItem).Build()).AsInt64()
		require.NoError(t, err)
		assert.Equal(t, 0, int(oldConcurrencyQueueItems), "expected no items in the old concurrency queue", r.Dump())
	})

	t.Run("It enqueues an item to account queues when account id is present", func(t *testing.T) {
		r.FlushAll()

		start := time.Now().Truncate(time.Second)

		// This test case handles account-scoped system partitions

		accountId := uuid.New()

		qi := osqueue.QueueItem{
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

		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		require.NotEqual(t, item.ID, ulid.Zero)
		require.Equal(t, time.UnixMilli(item.WallTimeMS).Truncate(time.Second), start)

		// Ensure that our data is set up correctly.
		found := getQueueItem(t, r, item.ID)
		require.Equal(t, item, found)

		// Ensure the partition is inserted.
		qp := getSystemPartition(t, r, customQueueName)
		require.Equal(t, QueuePartition{
			ID:            customQueueName,
			QueueName:     &customQueueName,
			PartitionType: int(enums.PartitionTypeDefault),
			AccountID:     uuid.Nil,
		}, qp)

		apIds := getAccountPartitions(t, rc, accountId)
		// it should not add system queues to account partitions
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

	q := NewQueue(QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName})
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

		ia, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{ID: "a"}, a, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		ib, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{ID: "b"}, b, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		ic, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{ID: "c"}, c, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		items, err := q.Peek(ctx, &QueuePartition{FunctionID: &workflowID}, time.Now().Add(time.Hour), 10)
		require.NoError(t, err)
		require.EqualValues(t, 3, len(items))
		require.EqualValues(t, []*osqueue.QueueItem{&ia, &ib, &ic}, items)
		require.NotEqualValues(t, []*osqueue.QueueItem{&ib, &ia, &ic}, items)

		id, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{ID: "d"}, d, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		items, err = q.Peek(ctx, &QueuePartition{FunctionID: &workflowID}, time.Now().Add(time.Hour), 10)
		require.NoError(t, err)
		require.EqualValues(t, 4, len(items))
		require.EqualValues(t, []*osqueue.QueueItem{&ia, &ib, &ic, &id}, items)

		t.Run("It should limit the list", func(t *testing.T) {
			items, err = q.Peek(ctx, &QueuePartition{FunctionID: &workflowID}, time.Now().Add(time.Hour), 2)
			require.NoError(t, err)
			require.EqualValues(t, 2, len(items))
			require.EqualValues(t, []*osqueue.QueueItem{&ia, &ib}, items)
		})

		t.Run("It should apply a peek offset", func(t *testing.T) {
			items, err = q.Peek(ctx, &QueuePartition{FunctionID: &workflowID}, time.Now().Add(-1*time.Hour), DefaultQueuePeekMax)
			require.NoError(t, err)
			require.EqualValues(t, 0, len(items))

			items, err = q.Peek(ctx, &QueuePartition{FunctionID: &workflowID}, c, DefaultQueuePeekMax)
			require.NoError(t, err)
			require.EqualValues(t, 3, len(items))
			require.EqualValues(t, []*osqueue.QueueItem{&ia, &ib, &ic}, items)
		})

		t.Run("It should remove any leased items from the list", func(t *testing.T) {
			// Lease step A, and it should be removed.
			_, err := q.Lease(ctx, ia, 50*time.Millisecond, time.Now(), nil)
			require.NoError(t, err)

			items, err = q.Peek(ctx, &QueuePartition{FunctionID: &workflowID}, d, DefaultQueuePeekMax)
			require.NoError(t, err)
			require.EqualValues(t, 3, len(items))
			require.EqualValues(t, []*osqueue.QueueItem{&ib, &ic, &id}, items)
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

			items, err = q.Peek(ctx, &QueuePartition{FunctionID: &workflowID}, d, DefaultQueuePeekMax)
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
			ia.EnqueuedAt = items[0].EnqueuedAt
			require.EqualValues(t, []*osqueue.QueueItem{&ia, &ib, &ic, &id}, items)
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

func TestQueuePartitionPeek(t *testing.T) {
	idA := uuid.New() // low pri
	idB := uuid.New()
	idC := uuid.New()

	accountId := uuid.New()

	newQueueItem := func(id uuid.UUID) osqueue.QueueItem {
		return osqueue.QueueItem{
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

	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	q := NewQueue(
		QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey)},
		WithPartitionPriorityFinder(func(ctx context.Context, p QueuePartition) uint {
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

	enqueue := func(q *queue, now time.Time) {
		atA, atB, atC := now, now.Add(2*time.Second), now.Add(4*time.Second)

		_, err := q.EnqueueItem(ctx, q.primaryQueueShard, newQueueItem(idA), atA, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		_, err = q.EnqueueItem(ctx, q.primaryQueueShard, newQueueItem(idB), atB, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		_, err = q.EnqueueItem(ctx, q.primaryQueueShard, newQueueItem(idC), atC, osqueue.EnqueueOpts{})
		require.NoError(t, err)
	}
	enqueue(q, now)

	t.Run("Sequentially returns partitions in order", func(t *testing.T) {
		items, err := q.PartitionPeek(ctx, true, time.Now().Add(time.Hour), PartitionPeekMax)
		require.NoError(t, err)
		require.Len(t, items, 3)
		require.EqualValues(t, []*QueuePartition{
			{ID: idA.String(), FunctionID: &idA, AccountID: accountId},
			{ID: idB.String(), FunctionID: &idB, AccountID: accountId},
			{ID: idC.String(), FunctionID: &idC, AccountID: accountId},
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
			QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey)},
			WithPartitionPriorityFinder(func(ctx context.Context, p QueuePartition) uint {
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

		enqueue(q, now)

		// This should only select B and C, as id A is ignored.
		items, err := q.PartitionPeek(ctx, true, now.Add(time.Hour), PartitionPeekMax)
		require.NoError(t, err)
		require.Len(t, items, 2)
		require.EqualValues(t, []*QueuePartition{
			{ID: idB.String(), FunctionID: &idB, AccountID: accountId},
			{ID: idC.String(), FunctionID: &idC, AccountID: accountId},
		}, items)

		// Try without sequential scans
		items, err = q.PartitionPeek(ctx, false, now.Add(time.Hour), PartitionPeekMax)
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

		paused := make(map[uuid.UUID]bool)
		q := NewQueue(
			QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey)},
			WithPartitionPriorityFinder(func(_ context.Context, _ QueuePartition) uint {
				return PriorityDefault
			}),
			WithPartitionPausedGetter(func(ctx context.Context, fnID uuid.UUID) PartitionPausedInfo {
				return PartitionPausedInfo{
					Paused: paused[fnID],
				}
			}),
		)
		now := time.Now()
		enqueue(q, now)
		requirePartitionScoreEquals(t, r, &idA, now)

		// Pause A, excluding it from peek:
		paused[idA] = true

		// This should only select B and C, as id A is ignored:
		items, err := q.PartitionPeek(ctx, true, now.Add(time.Hour), PartitionPeekMax)
		require.NoError(t, err)
		require.Len(t, items, 2)
		require.EqualValues(t, []*QueuePartition{
			{ID: idB.String(), FunctionID: &idB, AccountID: accountId},
			{ID: idC.String(), FunctionID: &idC, AccountID: accountId},
		}, items)
		requirePartitionScoreEquals(t, r, &idA, now.Add(PartitionPausedRequeueExtension))

		// After unpausing A, it should be included in the peek:
		paused[idA] = false
		require.NoError(t, q.UnpauseFunction(ctx, q.primaryQueueShard.Name, accountId, idA))

		require.NoError(t, err)
		items, err = q.PartitionPeek(ctx, true, time.Now().Add(time.Hour), PartitionPeekMax)
		require.NoError(t, err)
		require.Len(t, items, 3)
		require.EqualValues(t, []*QueuePartition{
			{ID: idA.String(), FunctionID: &idA, AccountID: accountId},
			{ID: idB.String(), FunctionID: &idB, AccountID: accountId},
			{ID: idC.String(), FunctionID: &idC, AccountID: accountId},
		}, items, r.Dump())
		requirePartitionScoreEquals(t, r, &idA, now)
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
			QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey)},
			WithPartitionPriorityFinder(func(_ context.Context, _ QueuePartition) uint {
				return PriorityDefault
			}),
		)
		enqueue(q, now)

		// Create inconsistency: Delete partition item from partition hash and global partition index but _not_ account partitions
		err = rc.Do(ctx, rc.B().Hdel().Key(q.primaryQueueShard.RedisClient.kg.PartitionItem()).Field(idA.String()).Build()).Error()
		require.NoError(t, err)
		err = rc.Do(ctx, rc.B().Zrem().Key(q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex()).Member(idA.String()).Build()).Error()
		require.NoError(t, err)

		// This should only select B and C, as id A is ignored and cleaned up:
		items, err := q.partitionPeek(ctx, q.primaryQueueShard.RedisClient.kg.AccountPartitionIndex(accountId), true, time.Now().Add(time.Hour), PartitionPeekMax, &accountId)
		require.NoError(t, err)
		require.Len(t, items, 2)
		require.EqualValues(t, []*QueuePartition{
			{ID: idB.String(), AccountID: accountId, FunctionID: &idB},
			{ID: idC.String(), AccountID: accountId, FunctionID: &idC},
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

	var enableKeyQueues bool
	shard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey)}
	q := NewQueue(
		shard,
		WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
			return enableKeyQueues
		}),
	)
	ctx := context.Background()
	idA := uuid.New()
	now := time.Now()
	accountID := uuid.New()

	t.Run("For default items without concurrency settings", func(t *testing.T) {
		qi, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: idA}, now, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		p := QueuePartition{FunctionID: &qi.FunctionID, EnvID: &qi.WorkspaceID}

		t.Run("Uses the next job item's time when requeueing with another job", func(t *testing.T) {
			requirePartitionScoreEquals(t, r, &idA, now)
			next := now.Add(time.Hour)
			err := q.PartitionRequeue(ctx, q.primaryQueueShard, &p, next, false)
			require.NoError(t, err)
			requirePartitionScoreEquals(t, r, &idA, now)
		})

		next := now.Add(5 * time.Second)
		t.Run("It removes any lease when requeueing", func(t *testing.T) {
			_, _, err := q.PartitionLease(ctx, &QueuePartition{FunctionID: &idA}, time.Minute)
			require.NoError(t, err)

			err = q.PartitionRequeue(ctx, q.primaryQueueShard, &p, next, true)
			require.NoError(t, err)
			requirePartitionScoreEquals(t, r, &idA, next)

			loaded := getDefaultPartition(t, r, idA)
			require.Nil(t, loaded.LeaseID)

			// Forcing should set a ForceAtMS field.
			require.NotEmpty(t, loaded.ForceAtMS)

			t.Run("Enqueueing with a force at time should not update the score", func(t *testing.T) {
				loaded := getDefaultPartition(t, r, idA)
				require.NotEmpty(t, loaded.ForceAtMS)

				qi, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: idA}, now, osqueue.EnqueueOpts{})

				loaded = getDefaultPartition(t, r, idA)
				require.NotEmpty(t, loaded.ForceAtMS)

				require.NoError(t, err)
				requirePartitionScoreEquals(t, r, &idA, next)
				requirePartitionScoreEquals(t, r, &idA, time.UnixMilli(loaded.ForceAtMS))

				// Now remove this item, as we don't need it for any future tests.
				err = q.Dequeue(ctx, q.primaryQueueShard, qi)
				require.NoError(t, err)
			})
		})

		t.Run("It returns a partition not found error if deleted", func(t *testing.T) {
			err := q.Dequeue(ctx, q.primaryQueueShard, qi)
			require.NoError(t, err)

			err = q.PartitionRequeue(ctx, q.primaryQueueShard, &p, time.Now().Add(time.Minute), false)
			require.Equal(t, ErrPartitionGarbageCollected, err)

			// ensure gc also drops fn metadata
			require.False(t, r.Exists(q.primaryQueueShard.RedisClient.kg.FnMetadata(*p.FunctionID)))

			err = q.PartitionRequeue(ctx, q.primaryQueueShard, &p, time.Now().Add(time.Minute), false)
			require.Equal(t, ErrPartitionNotFound, err)
		})

		// We no longer delete queues on requeue when the concurrency queue is not empty;  this should happen on a final dequeue.
		t.Run("Does not garbage collect the partition with a non-empty concurrency queue", func(t *testing.T) {
			r.FlushAll()

			now := time.Now()
			next = now.Add(10 * time.Second)

			qi, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: idA}, now, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			requirePartitionScoreEquals(t, r, &idA, now)

			// Move the queue item to the concurrency (in-progress) queue
			_, err = q.Lease(ctx, qi, 10*time.Second, q.clock.Now(), nil)
			require.NoError(t, err)

			next = now.Add(time.Hour)

			// Requeuing cannot gc until queue item finishes processing
			err = q.PartitionRequeue(ctx, q.primaryQueueShard, &p, next, false)
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

			item := osqueue.QueueItem{
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

			fnPart, custom1, _ := q.ItemPartitions(ctx, q.primaryQueueShard, item)

			originalPart := custom1
			require.Equal(t, "{queue}:concurrency:custom:a:4d59bf95-28b6-5423-b1a8-604046826e33:3cwxlkg53rr2c", originalPart.concurrencyKey(q.primaryQueueShard.RedisClient.kg))

			// Originally, this test was designed to run on concurrency key queues. Since we don't enqueue these anymore,
			// p has been changed to the default function partition.
			p := fnPart

			item, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, now, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			t.Run("Uses the next job item's time when requeueing with another job", func(t *testing.T) {
				t.Skip("This test is not applicable to the current system, as we do not update pointers for key queues")
				requireGlobalPartitionScore(t, r, p.zsetKey(q.primaryQueueShard.RedisClient.kg), now)
				next := now.Add(time.Hour)
				err := q.PartitionRequeue(ctx, q.primaryQueueShard, &p, next, false)
				require.NoError(t, err)
				// This should still be now(), as we're not forcing "next" and the earliest job is still now.
				requireGlobalPartitionScore(t, r, p.zsetKey(q.primaryQueueShard.RedisClient.kg), now)
			})

			t.Run("Forces a custom partition with `force` set to true", func(t *testing.T) {
				t.Skip("This test is not applicable to the current system, as we do not update pointers for key queues")
				requireGlobalPartitionScore(t, r, p.zsetKey(q.primaryQueueShard.RedisClient.kg), now)
				next := now.Add(time.Hour)
				err := q.PartitionRequeue(ctx, q.primaryQueueShard, &p, next, true)
				require.NoError(t, err)
				requireGlobalPartitionScore(t, r, p.zsetKey(q.primaryQueueShard.RedisClient.kg), next)
			})

			t.Run("Sets back to next job with force: false", func(t *testing.T) {
				t.Skip("This test is not applicable to the current system, as we do not update pointers for key queues")
				err := q.PartitionRequeue(ctx, q.primaryQueueShard, &p, time.Now(), false)
				require.NoError(t, err)
				requireGlobalPartitionScore(t, r, p.zsetKey(q.primaryQueueShard.RedisClient.kg), now)
			})

			t.Run("It doesn't dequeue the partition with an in-progress job", func(t *testing.T) {
				id, err := q.Lease(ctx, item, 10*time.Second, q.clock.Now(), nil)
				require.NoError(t, err)
				require.NotNil(t, id)

				next := now.Add(time.Minute)

				err = q.PartitionRequeue(ctx, q.primaryQueueShard, &p, next, false)
				require.NoError(t, err)

				// We do not set the global partition score for key queues
				require.False(t, r.Exists(p.zsetKey(q.primaryQueueShard.RedisClient.kg)))

				t.Run("With an empty queue the zset is deleted", func(t *testing.T) {
					err := q.Dequeue(ctx, q.primaryQueueShard, item)
					require.NoError(t, err)
					err = q.PartitionRequeue(ctx, q.primaryQueueShard, &p, next, false)
					require.Error(t, ErrPartitionGarbageCollected, err)
				})
			})
		})
	})

	t.Run("does not clean up if backlog isn't empty", func(t *testing.T) {
		r.FlushAll()

		//
		// Setup: Enqueue 2 items, one to backlog, one to ready queue
		//

		qi, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: idA, Data: osqueue.Item{Identifier: state.Identifier{AccountID: accountID, WorkflowID: idA}}}, now, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		fnReadyQueue := shard.RedisClient.kg.PartitionQueueSet(enums.PartitionTypeDefault, idA.String(), "")

		require.True(t, r.Exists(fnReadyQueue))
		require.True(t, r.Exists(shard.RedisClient.kg.GlobalPartitionIndex()))
		require.True(t, r.Exists(shard.RedisClient.kg.AccountPartitionIndex(accountID)))
		require.True(t, r.Exists(shard.RedisClient.kg.GlobalAccountIndex()))

		enableKeyQueues = true
		qi2, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			FunctionID: idA,
			Data: osqueue.Item{
				Kind: osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID: idA,
					AccountID:  accountID,
				},
			},
		}, now, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		enableKeyQueues = false

		backlog := q.ItemBacklog(ctx, qi2)
		shadowPart := q.ItemShadowPartition(ctx, qi2)

		require.True(t, r.Exists(shard.RedisClient.kg.BacklogSet(backlog.BacklogID)))
		require.True(t, r.Exists(shard.RedisClient.kg.ShadowPartitionSet(backlog.ShadowPartitionID)))
		require.Equal(t, 1, zcard(t, rc, fnReadyQueue))

		p := q.ItemPartition(ctx, shard, qi)
		require.Equal(t, idA.String(), p.ID)
		require.Equal(t, accountID, p.AccountID)

		//
		// Dequeue item from ready queue, only backlog remains
		//

		err = q.Dequeue(ctx, q.primaryQueueShard, qi)
		require.NoError(t, err)

		require.Equal(t, 0, zcard(t, rc, fnReadyQueue))
		require.True(t, r.Exists(shard.RedisClient.kg.GlobalPartitionIndex()))
		require.True(t, r.Exists(shard.RedisClient.kg.AccountPartitionIndex(accountID)))
		require.True(t, r.Exists(shard.RedisClient.kg.GlobalAccountIndex()))

		// do not expect function metadata to be set anymore
		require.False(t, r.Exists(shard.RedisClient.kg.FnMetadata(*p.FunctionID)), r.Keys())

		//
		// PartitionRequeue should drop pointers but not partition metadata
		//

		err = q.PartitionRequeue(ctx, q.primaryQueueShard, &p, now.Add(time.Minute), false)
		require.Equal(t, ErrPartitionGarbageCollected, err)

		require.Equal(t, 0, zcard(t, rc, fnReadyQueue))
		require.False(t, r.Exists(shard.RedisClient.kg.GlobalPartitionIndex()))
		require.False(t, r.Exists(shard.RedisClient.kg.AccountPartitionIndex(accountID)))
		require.False(t, r.Exists(shard.RedisClient.kg.GlobalAccountIndex()))

		// fn metadata still should not exist
		require.False(t, r.Exists(shard.RedisClient.kg.FnMetadata(*p.FunctionID)), r.Keys())

		// ensure gc does not drop partition item yet
		require.True(t, r.Exists(shard.RedisClient.kg.PartitionItem()))
		keys, err := r.HKeys(shard.RedisClient.kg.PartitionItem())
		require.NoError(t, err)
		require.Contains(t, keys, p.FunctionID.String())

		//
		// Drop backlog and have PartitionRequeue clean up remaining data
		//

		// drop backlog
		// Get items to refill from backlog
		itemIDs, err := getItemIDsFromBacklog(ctx, q, &backlog, time.Now().Add(time.Minute), 1000)
		require.NoError(t, err)

		res, err := q.BacklogRefill(ctx, &backlog, &shadowPart, time.Now().Add(time.Minute), itemIDs, PartitionConstraintConfig{})
		require.NoError(t, err)
		require.Equal(t, 1, res.Refilled)

		err = q.Dequeue(ctx, shard, qi2)
		require.NoError(t, err)

		err = q.PartitionRequeue(ctx, shard, &p, now.Add(time.Minute), false)
		require.Equal(t, ErrPartitionGarbageCollected, err)

		require.False(t, r.Exists(shard.RedisClient.kg.FnMetadata(*p.FunctionID)))
		require.False(t, r.Exists(shard.RedisClient.kg.PartitionItem()))
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

	var paused bool

	shard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey)}
	kg := shard.RedisClient.KeyGenerator()
	q := NewQueue(
		shard,
		WithPartitionPriorityFinder(func(_ context.Context, _ QueuePartition) uint {
			return PriorityDefault
		}),
		WithPartitionPausedGetter(func(ctx context.Context, fnID uuid.UUID) PartitionPausedInfo {
			return PartitionPausedInfo{
				Paused: paused,
			}
		}),
	)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	idA := uuid.New()
	_, err = q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: idA}, now, osqueue.EnqueueOpts{})
	require.NoError(t, err)

	paused = true

	peeked, err := q.partitionPeek(ctx, kg.GlobalPartitionIndex(), true, now.Add(5*time.Minute), 100, nil)
	require.NoError(t, err)
	require.Len(t, peeked, 0)

	paused = false

	err = q.UnpauseFunction(ctx, shard.Name, uuid.Nil, idA)
	require.NoError(t, err)

	peeked, err = q.partitionPeek(ctx, kg.GlobalPartitionIndex(), true, now.Add(5*time.Minute), 100, nil)
	require.NoError(t, err)
	require.Len(t, peeked, 1)
	require.Equal(t, idA, *peeked[0].FunctionID)
}

func TestQueueSetFunctionMigrate(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	t.Run("with default shard", func(t *testing.T) {
		shard := QueueShard{Name: "default", Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey)}
		kg := shard.RedisClient.kg
		q := NewQueue(
			shard,
			WithPartitionPriorityFinder(func(ctx context.Context, part QueuePartition) uint {
				return PriorityDefault
			}),
		)
		ctx := context.Background()

		acctID := uuid.New()
		now := time.Now().Truncate(time.Second)
		fnID := uuid.New()
		id := state.Identifier{AccountID: acctID, WorkflowID: fnID}
		_, err = q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: fnID, Data: osqueue.Item{Identifier: id}}, now, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		lockUntil := now.Add(10 * time.Minute)
		err = q.SetFunctionMigrate(ctx, "default", fnID, &lockUntil)
		require.NoError(t, err)

		require.True(t, r.Exists(kg.QueueMigrationLock(fnID)))
		lockValue, err := r.Get(kg.QueueMigrationLock(fnID))
		require.NoError(t, err)
		require.Equal(t, lockUntil, ulid.MustParse(lockValue).Timestamp())

		// disable migration flag
		err = q.SetFunctionMigrate(ctx, "default", fnID, nil)
		require.NoError(t, err)

		require.False(t, r.Exists(kg.QueueMigrationLock(fnID)))
	})

	t.Run("with key queues", func(t *testing.T) {
		shard := QueueShard{Name: "default", Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey)}
		q := NewQueue(
			shard,
			WithPartitionPriorityFinder(func(ctx context.Context, part QueuePartition) uint {
				return PriorityDefault
			}),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		)
		ctx := context.Background()

		acctID := uuid.New()
		now := time.Now().Truncate(time.Second)
		fnID := uuid.New()
		id := state.Identifier{AccountID: acctID, WorkflowID: fnID}
		_, err = q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: fnID, Data: osqueue.Item{Identifier: id}}, now, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		getShadowPartition := func() QueueShadowPartition {
			var sp QueueShadowPartition

			str, err := rc.Do(ctx, rc.B().Hget().Key(q.primaryQueueShard.RedisClient.kg.ShadowPartitionMeta()).Field(fnID.String()).Build()).ToString()
			require.NoError(t, err)

			require.NoError(t, json.Unmarshal([]byte(str), &sp))
			return sp
		}

		sp := getShadowPartition()
		require.Equal(t, fnID.String(), sp.PartitionID)

		lockedUntil, err := q.isMigrationLocked(ctx, shard, fnID)
		require.NoError(t, err)
		require.Nil(t, lockedUntil)

		lockUntil := now.Add(10 * time.Minute)
		err = q.SetFunctionMigrate(ctx, "default", fnID, &lockUntil)
		require.NoError(t, err)

		lockedUntil, err = q.isMigrationLocked(ctx, shard, fnID)
		require.NoError(t, err)
		require.NotNil(t, lockedUntil)
		require.Equal(t, lockUntil, *lockedUntil)

		// disable migration flag
		err = q.SetFunctionMigrate(ctx, "default", fnID, nil)
		require.NoError(t, err)

		lockedUntil, err = q.isMigrationLocked(ctx, shard, fnID)
		require.NoError(t, err)
		require.Nil(t, lockedUntil)
	})

	t.Run("with other shards", func(t *testing.T) {
		other := miniredis.RunT(t)
		rc2, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{other.Addr()}, DisableCache: true})
		require.NoError(t, err)
		defer rc2.Close()

		yoloShard := QueueShard{Name: "yolo", Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc2, QueueDefaultKey)}
		defaultShard := QueueShard{Name: "default", Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey)}

		q := NewQueue(
			defaultShard,
			WithQueueShardClients(map[string]QueueShard{
				"yolo":                       yoloShard,
				consts.DefaultQueueShardName: defaultShard,
			}),
			WithPartitionPriorityFinder(func(ctx context.Context, part QueuePartition) uint {
				return PriorityDefault
			}),
		)

		ctx := context.Background()
		acctID := uuid.New()
		now := time.Now().Truncate(time.Second)
		fnID := uuid.New()
		id := state.Identifier{AccountID: acctID, WorkflowID: fnID}
		_, err = q.EnqueueItem(ctx, yoloShard, osqueue.QueueItem{FunctionID: fnID, Data: osqueue.Item{Identifier: id}}, now, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		lockUntil := now.Add(10 * time.Minute)
		err = q.SetFunctionMigrate(ctx, "yolo", fnID, &lockUntil)
		require.NoError(t, err)

		// should not find it in the default shard
		lockedUntil, err := q.isMigrationLocked(ctx, defaultShard, fnID)
		require.NoError(t, err)
		require.Nil(t, lockedUntil)

		// should find metadata in the other shard
		lockedUntil, err = q.isMigrationLocked(ctx, yoloShard, fnID)
		require.NoError(t, err)
		require.Equal(t, lockUntil, *lockedUntil)
	})
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
		QueueShard{Kindstring(enums.QueueShardKindRedis,, RedisClientNewQueueClient(rc, QueueDefaultKey)
		WithPartitionPriorityFinder(func(_ context.Context, _ QueuePartition) uint {
			return priority
		}),
	)
	ctx := context.Background()

	_, err = q.EnqueueItem(ctx,QueueShard{Nameconsts.DefaultQueueShardName,Kindstring(enums.QueueShardKindRedis},RedisClientq.primaryQueueShard.RedisClient, osqueue.QueueItem{FunctionID: idA}, now)
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

	q := NewQueue(QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey)})
	q.ppf = func(ctx context.Context, p QueuePartition) uint {
		return PriorityMin
	}
	q.partitionConstraintConfigGetter = func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
		return PartitionConstraintConfig{
			Concurrency: PartitionConcurrency{
				AccountConcurrency:  100,
				FunctionConcurrency: 100,
			},
		}
	}
	q.itemIndexer = QueueItemIndexerFunc
	q.clock = clockwork.NewRealClock()

	wsA := uuid.New()

	t.Run("Failure cases", func(t *testing.T) {
		t.Run("It fails with a non-existent job ID for an existing partition", func(t *testing.T) {
			r.FlushDB()

			jid := "yeee"
			item := osqueue.QueueItem{
				ID:          jid,
				FunctionID:  wsA,
				WorkspaceID: wsA,
			}
			_, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, time.Now().Add(time.Second), osqueue.EnqueueOpts{})
			require.NoError(t, err)

			err = q.RequeueByJobID(ctx, q.primaryQueueShard, "no bruv", time.Now().Add(5*time.Second))
			require.NotNil(t, err)
		})

		t.Run("It fails if the job is leased", func(t *testing.T) {
			r.FlushDB()

			jid := "leased"
			item := osqueue.QueueItem{
				ID:          jid,
				FunctionID:  wsA,
				WorkspaceID: wsA,
			}

			item, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, time.Now().Add(time.Second), osqueue.EnqueueOpts{})
			require.NoError(t, err)

			partitions, err := q.PartitionPeek(ctx, true, time.Now().Add(5*time.Second), 10)
			require.NoError(t, err)
			require.Equal(t, 1, len(partitions))

			// Lease
			lid, err := q.Lease(ctx, item, time.Second*10, time.Now(), nil)
			require.NoError(t, err)
			require.NotNil(t, lid)

			err = q.RequeueByJobID(ctx, q.primaryQueueShard, jid, time.Now().Add(5*time.Second))
			require.NotNil(t, err)
		})
	})

	t.Run("It requeues the job", func(t *testing.T) {
		r.FlushDB()

		jid := "requeue-plz"
		at := time.Now().Add(time.Second).Truncate(time.Millisecond)
		item := osqueue.QueueItem{
			ID:          jid,
			FunctionID:  wsA,
			WorkspaceID: wsA,
			AtMS:        at.UnixMilli(),
		}
		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, at, osqueue.EnqueueOpts{})
		require.Equal(t, time.UnixMilli(item.WallTimeMS), at)
		require.NoError(t, err)

		// Find all functions
		parts, err := q.PartitionPeek(ctx, true, at.Add(time.Hour), 10)
		require.NoError(t, err)
		require.Equal(t, 1, len(parts))

		// Requeue the function for 5 seconds in the future.
		next := at.Add(5 * time.Second)
		err = q.RequeueByJobID(ctx, q.primaryQueueShard, jid, next)
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

			score, err := r.ZScore(q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex(), wsA.String())
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
			item := osqueue.QueueItem{
				FunctionID:  wsA,
				WorkspaceID: wsA,
				AtMS:        next.UnixMilli(),
			}
			_, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, next, osqueue.EnqueueOpts{})
			require.NoError(t, err)
		}

		target := time.Now().Add(10 * time.Second)
		jid := "requeue-plz"
		item := osqueue.QueueItem{
			ID:          jid,
			FunctionID:  wsA,
			WorkspaceID: wsA,
			AtMS:        target.UnixMilli(),
		}
		_, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, target, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		parts, err := q.PartitionPeek(ctx, true, at.Add(time.Hour), 10)
		require.NoError(t, err)
		require.Equal(t, 1, len(parts))

		t.Run("The earliest time is 'at' for the partition", func(t *testing.T) {
			score, err := r.ZScore(q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex(), wsA.String())
			require.NoError(t, err)
			require.EqualValues(t, at.Unix(), int64(score), r.Dump())
		})

		next := target.Add(5 * time.Second)
		err = q.RequeueByJobID(ctx, q.primaryQueueShard, jid, next)
		require.Nil(t, err, r.Dump())

		t.Run("The earliest time is still 'at' for the partition after requeueing", func(t *testing.T) {
			score, err := r.ZScore(q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex(), wsA.String())
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
			item := osqueue.QueueItem{
				FunctionID:  wsA,
				WorkspaceID: wsA,
				AtMS:        next.UnixMilli(),
			}
			_, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, next, osqueue.EnqueueOpts{})
			require.NoError(t, err)
		}

		target := time.Now().Add(1 * time.Second)
		jid := "requeue-plz"
		item := osqueue.QueueItem{
			ID:          jid,
			FunctionID:  wsA,
			WorkspaceID: wsA,
			AtMS:        target.UnixMilli(),
		}
		_, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, target, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		parts, err := q.PartitionPeek(ctx, true, at.Add(time.Hour), 10)
		require.NoError(t, err)
		require.Equal(t, 1, len(parts))

		t.Run("The earliest time is 'target' for the partition", func(t *testing.T) {
			score, err := r.ZScore(q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex(), wsA.String())
			require.NoError(t, err)
			require.EqualValues(t, target.Unix(), int64(score), r.Dump())
		})

		next := target.Add(5 * time.Second)
		err = q.RequeueByJobID(ctx, q.primaryQueueShard, jid, next)
		require.Nil(t, err, r.Dump())

		t.Run("The earliest time is 'next' for the partition after requeueing", func(t *testing.T) {
			score, err := r.ZScore(q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex(), wsA.String())
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

	qc := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey)}
	q := queue{
		primaryQueueShard: qc,
		ppf: func(ctx context.Context, p QueuePartition) uint {
			return PriorityMin
		},
		clock: clockwork.NewRealClock(),
	}

	var leaseID *ulid.ULID

	t.Run("It claims sequential leases", func(t *testing.T) {
		now := time.Now()
		dur := 500 * time.Millisecond
		leaseID, err = q.ConfigLease(ctx, q.primaryQueueShard.RedisClient.kg.Sequential(), dur)
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		require.WithinDuration(t, now.Add(dur), ulid.Time(leaseID.Time()), 5*time.Millisecond)
	})

	t.Run("It doesn't allow leasing without an existing lease ID", func(t *testing.T) {
		id, err := q.ConfigLease(ctx, q.primaryQueueShard.RedisClient.kg.Sequential(), time.Second)
		require.Equal(t, ErrConfigAlreadyLeased, err)
		require.Nil(t, id)
	})

	t.Run("It doesn't allow leasing with an invalid lease ID", func(t *testing.T) {
		newULID := ulid.MustNew(ulid.Now(), rnd)
		id, err := q.ConfigLease(ctx, q.primaryQueueShard.RedisClient.kg.Sequential(), time.Second, &newULID)
		require.Equal(t, ErrConfigAlreadyLeased, err)
		require.Nil(t, id)
	})

	t.Run("It extends the lease with a valid lease ID", func(t *testing.T) {
		require.NotNil(t, leaseID)

		now := time.Now()
		dur := 50 * time.Millisecond
		leaseID, err = q.ConfigLease(ctx, q.primaryQueueShard.RedisClient.kg.Sequential(), dur, leaseID)
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		require.WithinDuration(t, now.Add(dur), ulid.Time(leaseID.Time()), 5*time.Millisecond)
	})

	t.Run("It allows leasing when the current lease is expired", func(t *testing.T) {
		<-time.After(100 * time.Millisecond)

		now := time.Now()
		dur := 50 * time.Millisecond
		leaseID, err = q.ConfigLease(ctx, q.primaryQueueShard.RedisClient.kg.Sequential(), dur)
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		require.WithinDuration(t, now.Add(dur), ulid.Time(leaseID.Time()), 5*time.Millisecond)
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
	q := NewQueue(
		QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey)}, WithClock(clock),
	)

	idA, idB := uuid.New(), uuid.New()

	r := require.New(t)

	t.Run("Without bursts", func(t *testing.T) {
		throttle := &osqueue.Throttle{
			Key:    "some-key",
			Limit:  1,
			Period: 5, // Admit one every 5 seconds
			Burst:  0, // No burst.
		}

		q.partitionConstraintConfigGetter = func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
			return PartitionConstraintConfig{
				Throttle: &PartitionThrottle{
					Limit:                     1,
					Period:                    5,
					Burst:                     0,
					ThrottleKeyExpressionHash: util.XXHash(throttle.Key),
				},
			}
		}

		aa, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			FunctionID: idA,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					WorkflowID: idA,
				},
				Throttle: throttle,
			},
		}, clock.Now(), osqueue.EnqueueOpts{})
		r.NoError(err)

		ab, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			FunctionID: idA,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					WorkflowID: idA,
				},
				Throttle: throttle,
			},
		}, clock.Now().Add(time.Second), osqueue.EnqueueOpts{})
		r.NoError(err)

		// Leasing A should succeed, then B should fail.
		partitions, err := q.PartitionPeek(ctx, true, clock.Now().Add(5*time.Second), 5)
		r.NoError(err)
		r.EqualValues(1, len(partitions))

		// clock.Advance(10 * time.Millisecond)

		t.Run("Leasing a first item succeeds", func(t *testing.T) {
			leaseA, err := q.Lease(ctx, aa, 10*time.Second, clock.Now(), nil)
			r.NoError(err, "leasing throttled queue item with capacity failed")
			r.NotNil(leaseA)
		})

		// clock.Advance(10 * time.Millisecond)

		t.Run("Attempting to lease another throttled key immediately fails", func(t *testing.T) {
			leaseB, err := q.Lease(ctx, ab, 10*time.Second, clock.Now(), nil)
			r.NotNil(err, "leasing throttled queue item without capacity didn't error")
			r.Nil(leaseB)
		})

		// clock.Advance(10 * time.Millisecond)

		t.Run("Leasing another function succeeds", func(t *testing.T) {
			q.partitionConstraintConfigGetter = func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return PartitionConstraintConfig{
					Throttle: &PartitionThrottle{
						Limit:                     1,
						Period:                    5,
						Burst:                     0,
						ThrottleKeyExpressionHash: util.XXHash("another-key"),
					},
				}
			}
			ba, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
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
			}, clock.Now().Add(time.Second), osqueue.EnqueueOpts{})
			r.NoError(err)
			lease, err := q.Lease(ctx, ba, 10*time.Second, clock.Now(), nil)
			r.Nil(err, "leasing throttled queue item without capacity didn't error")
			r.NotNil(lease)
		})

		// clock.Advance(10 * time.Millisecond)

		t.Run("Leasing after the period succeeds", func(t *testing.T) {
			clock.Advance(time.Duration(throttle.Period)*time.Second + time.Second)

			leaseB, err := q.Lease(ctx, ab, 10*time.Second, clock.Now(), nil)
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

		q.partitionConstraintConfigGetter = func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
			return PartitionConstraintConfig{
				Throttle: &PartitionThrottle{
					ThrottleKeyExpressionHash: util.XXHash("burst-plz"),
					Limit:                     1,
					Period:                    10,
					Burst:                     3,
				},
			}
		}

		items := []osqueue.QueueItem{}
		for i := 0; i <= 20; i++ {
			item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
				FunctionID: idA,
				Data: osqueue.Item{
					Identifier: state.Identifier{WorkflowID: idA},
					Throttle:   throttle,
				},
			}, clock.Now(), osqueue.EnqueueOpts{})
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
				lease, err := q.Lease(ctx, items[i], 2*time.Second, clock.Now(), nil)
				r.NoError(err, "leasing throttled queue item with capacity failed")
				r.NotNil(lease)
				idx++
			}
		})

		t.Run("Leasing the 4th time fails", func(t *testing.T) {
			lease, err := q.Lease(ctx, items[idx], 1*time.Second, clock.Now(), nil)
			r.NotNil(err, "leasing throttled queue item without capacity didn't error")
			r.ErrorContains(err, ErrQueueItemThrottled.Error())
			r.Nil(lease)
		})

		t.Run("After 10s, we can re-lease once as bursting is done.", func(t *testing.T) {
			clock.Advance(time.Duration(throttle.Period)*time.Second + time.Second)

			lease, err := q.Lease(ctx, items[idx], 2*time.Second, clock.Now(), nil)
			r.NoError(err, "leasing throttled queue item with capacity failed")
			r.NotNil(lease)

			idx++

			// It should fail, as bursting is done.
			lease, err = q.Lease(ctx, items[idx], 1*time.Second, clock.Now(), nil)
			r.NotNil(err, "leasing throttled queue item without capacity didn't error")
			r.ErrorContains(err, ErrQueueItemThrottled.Error())
			r.Nil(lease)
		})

		t.Run("After another 40s, we can burst again", func(t *testing.T) {
			clock.Advance(time.Duration(throttle.Period*4) * time.Second)

			for i := 0; i < 3; i++ {
				lease, err := q.Lease(ctx, items[i], 2*time.Second, clock.Now(), nil)
				r.NoError(err, "leasing throttled queue item with capacity failed")
				r.NotNil(lease)
				idx++
			}
		})
	})
}

func TestMigrate(t *testing.T) {
	ctx := context.Background()

	testcases := []struct {
		name     string
		keyQueue bool
	}{
		{
			name: "without key queues",
		},
		{
			name:     "with key queues",
			keyQueue: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			clock := clockwork.NewFakeClock()

			// default redis
			r1 := miniredis.RunT(t)
			rc1, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r1.Addr()}, DisableCache: true})
			require.NoError(t, err)
			defer rc1.Close()

			// other redis
			r2 := miniredis.RunT(t)
			rc2, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r2.Addr()}, DisableCache: true})
			require.NoError(t, err)
			defer rc2.Close()

			shard1Name := "default"
			shard2Name := "yolo"

			shard1 := QueueShard{Name: shard1Name, Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc1, QueueDefaultKey)}
			shard2 := QueueShard{Name: shard2Name, Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc2, QueueDefaultKey)}

			shards := map[string]QueueShard{shard1Name: shard1, shard2Name: shard2}

			q1 := NewQueue(
				shard1,
				WithQueueShardClients(shards),
				WithPartitionPriorityFinder(func(ctx context.Context, part QueuePartition) uint {
					return PriorityDefault
				}),
				WithClock(clock),
				WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
					return tc.keyQueue
				}),
			)

			q2 := NewQueue(
				shard2,
				WithQueueShardClients(shards),
				WithPartitionPriorityFinder(func(ctx context.Context, part QueuePartition) uint {
					return PriorityDefault
				}),
				WithClock(clock),
				WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
					return tc.keyQueue
				}),
			)

			expectItemCountForPartition := func(ctx context.Context, q *queue, shard QueueShard, partitionID uuid.UUID, expected int) {
				var count int

				from := time.Time{}
				until := q.clock.Now().Add(24 * time.Hour * 365)
				items, err := q.ItemsByPartition(ctx, shard, partitionID.String(), from, until)
				require.NoError(t, err)

				for range items {
					count++
				}
				require.Equal(t, expected, count)
			}

			acctID := uuid.New()
			fnID := uuid.New()

			// Enqueue to shard 1
			for range 5 {
				id := state.Identifier{AccountID: acctID, WorkflowID: fnID, EventID: ulid.MustNew(ulid.Now(), rand.Reader), RunID: ulid.MustNew(ulid.Now(), rand.Reader)}
				err := q1.Enqueue(ctx, osqueue.Item{Identifier: id}, clock.Now(), osqueue.EnqueueOpts{})
				require.NoError(t, err)
			}

			// Don't really need it since there are no executors to process the enqueued items
			lockUntil := clock.Now().Add(10 * time.Minute)
			err = q1.SetFunctionMigrate(ctx, shard1Name, fnID, &lockUntil)
			require.NoError(t, err)

			// Verify that there are expected number of items in it
			expectItemCountForPartition(ctx, q1, shard1, fnID, 5)

			// Attempt to migrate from shard1 to shard2
			processed, err := q1.Migrate(ctx, shard1Name, fnID, 10, 0, func(ctx context.Context, qi *osqueue.QueueItem) error {
				return q2.Enqueue(ctx, qi.Data, time.UnixMilli(qi.AtMS), osqueue.EnqueueOpts{PassthroughJobId: true})
			})
			require.NoError(t, err)
			require.Equal(t, int64(5), processed)

			// Verify that shard2 now have all the items
			expectItemCountForPartition(ctx, q2, shard2, fnID, 5)

			// shard1 should no longer have anything
			expectItemCountForPartition(ctx, q1, shard1, fnID, 0)

			clock.Advance(q1.idempotencyTTL + 5*time.Second)
			r1.FastForward(q1.idempotencyTTL + 5*time.Second)

			// Now, move everything back to queue 1
			returned, err := q2.Migrate(ctx, shard2Name, fnID, 10, 0, func(ctx context.Context, qi *osqueue.QueueItem) error {
				return q1.Enqueue(ctx, qi.Data, time.UnixMilli(qi.AtMS), osqueue.EnqueueOpts{PassthroughJobId: true})
			})
			require.NoError(t, err)
			require.Equal(t, int64(5), returned)

			// shard1 should have the queue items again
			expectItemCountForPartition(ctx, q1, shard1, fnID, 5)

			// Verify that shard2 now have nothing
			expectItemCountForPartition(ctx, q2, shard2, fnID, 0)
		})
	}
}

func getQueueItem(t *testing.T, r *miniredis.Miniredis, id string) osqueue.QueueItem {
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
	i := osqueue.QueueItem{}
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
	require.NoError(t, err, r.Dump(), val)
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

func partitionIsMissingInHash(t *testing.T, r *miniredis.Miniredis, pType enums.PartitionType, id uuid.UUID, optionalHash ...string) {
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

	val, err := r.HKeys(kg.PartitionItem())
	require.NoError(t, err)
	require.NotContains(t, val, key, "expected partition to be missing")
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

func requireItemScoreEquals(t *testing.T, r *miniredis.Miniredis, item osqueue.QueueItem, expected time.Time) {
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

func concurrencyQueueScores(t *testing.T, r *miniredis.Miniredis, key string, _ time.Time) map[string]time.Time {
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
	hash := c.EvaluatedKey(context.Background(), scopeID, map[string]any{})

	return state.CustomConcurrency{
		Key:                       hash,
		Limit:                     limit,
		Hash:                      value,
		UnhashedEvaluatedKeyValue: value,
	}
}

func int64ptr(i int64) *int64 { return &i }

func TestQueueEnqueueToBacklog(t *testing.T) {
	t.Run("simple item", func(t *testing.T) {
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
		kg := defaultShard.RedisClient.kg

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Second))
		now := clock.Now()

		q := NewQueue(
			defaultShard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		)
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute)

		t.Run("should enqueue simple item to backlog", func(t *testing.T) {
			require.Len(t, r.Keys(), 0)

			item := osqueue.QueueItem{
				ID:          "test",
				FunctionID:  fnID,
				WorkspaceID: wsID,
				Data: osqueue.Item{
					WorkspaceID: wsID,
					Kind:        osqueue.KindEdge,
					Identifier: state.Identifier{
						WorkflowID:  fnID,
						AccountID:   accountId,
						WorkspaceID: wsID,
					},
					QueueName:             nil,
					Throttle:              nil,
					CustomConcurrencyKeys: nil,
				},
				QueueName: nil,
			}

			qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			backlog := q.ItemBacklog(ctx, item)
			require.NotEmpty(t, backlog.BacklogID)

			marshaledBacklog, err := json.Marshal(backlog)
			require.NoError(t, err)

			shadowPartition := q.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			marshaledShadowPartition, err := json.Marshal(shadowPartition)
			require.NoError(t, err)

			require.True(t, r.Exists(kg.BacklogMeta()))
			require.True(t, r.Exists(kg.BacklogSet(backlog.BacklogID)))
			require.True(t, r.Exists(kg.ShadowPartitionMeta()))
			require.True(t, r.Exists(kg.ShadowPartitionSet(shadowPartition.PartitionID)), r.Keys())
			require.True(t, r.Exists(kg.GlobalShadowPartitionSet()))
			require.True(t, r.Exists(kg.GlobalAccountShadowPartitions()))
			require.True(t, r.Exists(kg.AccountShadowPartitions(accountId)))
			require.Equal(t, string(marshaledBacklog), r.HGet(kg.BacklogMeta(), backlog.BacklogID))
			require.Equal(t, string(marshaledShadowPartition), r.HGet(kg.ShadowPartitionMeta(), shadowPartition.PartitionID))

			require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.ShadowPartitionSet(shadowPartition.PartitionID), backlog.BacklogID)))
			require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPartition.PartitionID)))
			require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID)))

			require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountId.String())))
			require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountId), shadowPartition.PartitionID)))
		})

		t.Run("adding later item should not update scores", func(t *testing.T) {
			newScore := at.Add(5 * time.Minute)

			item := osqueue.QueueItem{
				ID:          "item-2",
				FunctionID:  fnID,
				WorkspaceID: wsID,
				Data: osqueue.Item{
					WorkspaceID: wsID,
					Kind:        osqueue.KindEdge,
					Identifier: state.Identifier{
						WorkflowID:  fnID,
						AccountID:   accountId,
						WorkspaceID: wsID,
					},
					QueueName:             nil,
					Throttle:              nil,
					CustomConcurrencyKeys: nil,
				},
				QueueName: nil,
			}

			qi, err := q.EnqueueItem(ctx, defaultShard, item, newScore, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			backlog := q.ItemBacklog(ctx, item)
			require.NotEmpty(t, backlog.BacklogID)

			shadowPartition := q.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			// pointers should keep earlier score
			require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.ShadowPartitionSet(shadowPartition.PartitionID), backlog.BacklogID)))
			require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPartition.PartitionID)))
			require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountId.String())))
			require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountId), shadowPartition.PartitionID)))

			// item in backlog should have new score
			require.Equal(t, newScore.UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID)))
		})

		t.Run("adding earlier item should pull up pointer scores", func(t *testing.T) {
			newScore := at.Add(-5 * time.Minute)

			item := osqueue.QueueItem{
				ID:          "item-3",
				FunctionID:  fnID,
				WorkspaceID: wsID,
				Data: osqueue.Item{
					WorkspaceID: wsID,
					Kind:        osqueue.KindEdge,
					Identifier: state.Identifier{
						WorkflowID:  fnID,
						AccountID:   accountId,
						WorkspaceID: wsID,
					},
					QueueName:             nil,
					Throttle:              nil,
					CustomConcurrencyKeys: nil,
				},
				QueueName: nil,
			}

			qi, err := q.EnqueueItem(ctx, defaultShard, item, newScore, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			backlog := q.ItemBacklog(ctx, item)
			require.NotEmpty(t, backlog.BacklogID)

			shadowPartition := q.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			// pointers should take on earlier score
			{
				expected := newScore.UnixMilli()
				actual := int64(score(t, r, kg.ShadowPartitionSet(shadowPartition.PartitionID), backlog.BacklogID))

				require.Equal(t, expected, actual, time.UnixMilli(expected).String(), time.UnixMilli(actual).String())
			}
			require.Equal(t, newScore.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPartition.PartitionID)))

			require.Equal(t, newScore.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountId.String())))
			require.Equal(t, newScore.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountId), shadowPartition.PartitionID)))

			// item in backlog should have new score
			require.Equal(t, newScore.UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID)))
		})
	})

	t.Run("single custom concurrency key", func(t *testing.T) {
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
		kg := defaultShard.RedisClient.kg

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Second))
		now := clock.Now()

		q := NewQueue(
			defaultShard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
			WithPartitionConstraintConfigGetter(func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return PartitionConstraintConfig{
					Concurrency: PartitionConcurrency{
						AccountConcurrency:  123,
						FunctionConcurrency: 45,
						SystemConcurrency:   678,
					},
				}
			}),
		)
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute)

		t.Run("should enqueue item to backlog", func(t *testing.T) {
			require.Len(t, r.Keys(), 0)

			hashedConcurrencyKeyExpr := hashConcurrencyKey("event.data.customerId")
			unhashedValue := "customer1"
			scope := enums.ConcurrencyScopeFn
			fullKey := util.ConcurrencyKey(scope, fnID, unhashedValue)

			ckA := state.CustomConcurrency{
				Key:                       fullKey,
				Hash:                      hashedConcurrencyKeyExpr,
				Limit:                     123,
				UnhashedEvaluatedKeyValue: unhashedValue,
			}

			q.partitionConstraintConfigGetter = func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return PartitionConstraintConfig{
					Concurrency: PartitionConcurrency{
						AccountConcurrency:  123,
						FunctionConcurrency: 45,
						CustomConcurrencyKeys: []CustomConcurrencyLimit{
							{
								Scope:               enums.ConcurrencyScopeFn,
								HashedKeyExpression: ckA.Hash,
								Limit:               ckA.Limit,
							},
						},
					},
				}
			}

			item := osqueue.QueueItem{
				ID:          "test",
				FunctionID:  fnID,
				WorkspaceID: wsID,
				Data: osqueue.Item{
					WorkspaceID: wsID,
					Kind:        osqueue.KindEdge,
					Identifier: state.Identifier{
						WorkflowID:  fnID,
						AccountID:   accountId,
						WorkspaceID: wsID,
					},
					QueueName: nil,
					Throttle:  nil,
					CustomConcurrencyKeys: []state.CustomConcurrency{
						ckA,
					},
				},
				QueueName: nil,
			}

			qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			backlog := q.ItemBacklog(ctx, item)
			require.NotEmpty(t, backlog.BacklogID)
			require.Len(t, backlog.ConcurrencyKeys, 1)
			require.NotNil(t, backlog.ConcurrencyKeys[0].Scope)
			require.NotNil(t, backlog.ConcurrencyKeys[0].HashedKeyExpression)

			shadowPartition := q.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			constraints := q.partitionConstraintConfigGetter(ctx, shadowPartition.Identifier())
			require.Len(t, constraints.Concurrency.CustomConcurrencyKeys, 1)

			require.True(t, r.Exists(kg.BacklogMeta()), r.Keys())
			require.True(t, r.Exists(kg.BacklogSet(backlog.BacklogID)))
			require.True(t, r.Exists(kg.ShadowPartitionMeta()))
			require.True(t, r.Exists(kg.ShadowPartitionSet(shadowPartition.PartitionID)), r.Keys())
			require.True(t, r.Exists(kg.GlobalShadowPartitionSet()))
			require.True(t, r.Exists(kg.GlobalAccountShadowPartitions()))
			require.True(t, r.Exists(kg.AccountShadowPartitions(accountId)))

			require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.ShadowPartitionSet(shadowPartition.PartitionID), backlog.BacklogID)))
			require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPartition.PartitionID)))
			require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID)))

			require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountId.String())))
			require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountId), shadowPartition.PartitionID)))
		})
	})

	t.Run("two custom concurrency keys", func(t *testing.T) {
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
		kg := defaultShard.RedisClient.kg

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Second))
		now := clock.Now()

		q := NewQueue(
			defaultShard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
			WithPartitionConstraintConfigGetter(func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return PartitionConstraintConfig{
					Concurrency: PartitionConcurrency{
						AccountConcurrency:  123,
						FunctionConcurrency: 45,
						SystemConcurrency:   678,
					},
				}
			}),
		)
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute)

		t.Run("should enqueue item to backlog", func(t *testing.T) {
			require.Len(t, r.Keys(), 0)

			hashedConcurrencyKeyExpr1 := hashConcurrencyKey("event.data.userId")
			unhashedValue1 := "user1"
			scope1 := enums.ConcurrencyScopeFn
			fullKey1 := util.ConcurrencyKey(scope1, fnID, unhashedValue1)

			hashedConcurrencyKeyExpr2 := hashConcurrencyKey("event.data.orgId")
			unhashedValue2 := "org1"
			scope2 := enums.ConcurrencyScopeEnv
			fullKey2 := util.ConcurrencyKey(scope2, fnID, unhashedValue2)
			ckA := state.CustomConcurrency{
				Key:                       fullKey1,
				Hash:                      hashedConcurrencyKeyExpr1,
				Limit:                     123,
				UnhashedEvaluatedKeyValue: unhashedValue1,
			}
			ckB := state.CustomConcurrency{
				Key:                       fullKey2,
				Hash:                      hashedConcurrencyKeyExpr2,
				Limit:                     234,
				UnhashedEvaluatedKeyValue: unhashedValue2,
			}

			q.partitionConstraintConfigGetter = func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return PartitionConstraintConfig{
					Concurrency: PartitionConcurrency{
						AccountConcurrency:  123,
						FunctionConcurrency: 45,
						CustomConcurrencyKeys: []CustomConcurrencyLimit{
							{
								Scope:               enums.ConcurrencyScopeFn,
								HashedKeyExpression: ckA.Hash,
								Limit:               ckA.Limit,
							},
							{
								Scope:               enums.ConcurrencyScopeEnv,
								HashedKeyExpression: ckB.Hash,
								Limit:               ckB.Limit,
							},
						},
					},
				}
			}

			item := osqueue.QueueItem{
				ID:          "test",
				FunctionID:  fnID,
				WorkspaceID: wsID,
				Data: osqueue.Item{
					WorkspaceID: wsID,
					Kind:        osqueue.KindEdge,
					Identifier: state.Identifier{
						WorkflowID:  fnID,
						AccountID:   accountId,
						WorkspaceID: wsID,
					},
					QueueName: nil,
					Throttle:  nil,
					CustomConcurrencyKeys: []state.CustomConcurrency{
						ckA,
						ckB,
					},
				},
				QueueName: nil,
			}

			qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			backlog := q.ItemBacklog(ctx, item)
			require.Len(t, backlog.ConcurrencyKeys, 2)
			require.NotNil(t, backlog.ConcurrencyKeys[0].Scope)
			require.NotNil(t, backlog.ConcurrencyKeys[0].HashedKeyExpression)
			require.NotNil(t, backlog.ConcurrencyKeys[1].Scope)
			require.NotNil(t, backlog.ConcurrencyKeys[1].HashedKeyExpression)

			marshaledBacklog1, err := json.Marshal(backlog)
			require.NoError(t, err)

			shadowPartition := q.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			constraints := q.partitionConstraintConfigGetter(ctx, shadowPartition.Identifier())
			require.Len(t, constraints.Concurrency.CustomConcurrencyKeys, 2)

			require.True(t, r.Exists(kg.BacklogMeta()), r.Keys())
			require.True(t, r.Exists(kg.BacklogSet(backlog.BacklogID)))
			require.True(t, r.Exists(kg.ShadowPartitionMeta()))
			require.True(t, r.Exists(kg.ShadowPartitionSet(shadowPartition.PartitionID)), r.Keys())
			require.True(t, r.Exists(kg.GlobalShadowPartitionSet()))
			require.Equal(t, string(marshaledBacklog1), r.HGet(kg.BacklogMeta(), backlog.BacklogID))

			require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.ShadowPartitionSet(shadowPartition.PartitionID), backlog.BacklogID)))
			require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPartition.PartitionID)))
			require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID)))
		})
	})

	t.Run("system queues", func(t *testing.T) {
		t.Skip("system queues are never enqueued to backlogs")

		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
		kg := defaultShard.RedisClient.kg

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Second))
		now := clock.Now()

		q := NewQueue(
			defaultShard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
			// WithEnqueueSystemPartitionsToBacklog(true),
		)
		ctx := context.Background()

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute)

		sysQueueName := osqueue.KindQueueMigrate

		t.Run("should enqueue item to backlog", func(t *testing.T) {
			require.Len(t, r.Keys(), 0)

			item := osqueue.QueueItem{
				ID: "test",
				Data: osqueue.Item{
					Kind:                  osqueue.KindQueueMigrate,
					Identifier:            state.Identifier{},
					QueueName:             &sysQueueName,
					Throttle:              nil,
					CustomConcurrencyKeys: nil,
				},
				QueueName: &sysQueueName,
			}

			backlog := q.ItemBacklog(ctx, item)
			require.NotEmpty(t, backlog.BacklogID)

			marshaledBacklog, err := json.Marshal(backlog)
			require.NoError(t, err)

			shadowPartition := q.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			marshaledShadowPartition, err := json.Marshal(shadowPartition)
			require.NoError(t, err)

			qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			require.True(t, r.Exists(kg.BacklogMeta()))
			require.True(t, r.Exists(kg.BacklogSet(backlog.BacklogID)))
			require.True(t, r.Exists(kg.ShadowPartitionMeta()))
			require.True(t, r.Exists(kg.ShadowPartitionSet(shadowPartition.PartitionID)), r.Keys())
			require.True(t, r.Exists(kg.GlobalShadowPartitionSet()))
			require.False(t, r.Exists(kg.GlobalAccountShadowPartitions()))
			require.False(t, r.Exists(kg.AccountShadowPartitions(uuid.Nil)))

			require.Equal(t, string(marshaledBacklog), r.HGet(kg.BacklogMeta(), backlog.BacklogID))
			require.Equal(t, string(marshaledShadowPartition), r.HGet(kg.ShadowPartitionMeta(), shadowPartition.PartitionID))

			require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.ShadowPartitionSet(shadowPartition.PartitionID), backlog.BacklogID)))
			require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPartition.PartitionID)))
			require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID)))
		})
	})
}

func TestQueueEnqueueItemSingleton(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
	q := NewQueue(defaultShard)

	kg := defaultShard.RedisClient.KeyGenerator()

	ctx := context.Background()

	start := time.Now().Truncate(time.Second)

	t.Run("It enqueues an item only once until the last item in a function run is dequeued", func(t *testing.T) {
		key := "example"
		runId := ulid.MustNew(ulid.Now(), rand.Reader)
		qi1 := osqueue.QueueItem{
			Data: osqueue.Item{
				Kind: osqueue.KindStart,
				Identifier: state.Identifier{
					RunID: runId,
				},
				Singleton: &osqueue.Singleton{
					Key: key,
				},
			},
		}

		qi2 := osqueue.QueueItem{
			Data: osqueue.Item{
				Kind: osqueue.KindEdge,
				Identifier: state.Identifier{
					RunID: runId,
				},
			},
		}

		item1, err := q.EnqueueItem(ctx, q.primaryQueueShard, qi1, start, osqueue.EnqueueOpts{})

		require.NoError(t, err)
		require.NotEqual(t, qi1.ID, item1.ID)
		found := getQueueItem(t, r, item1.ID)
		require.Equal(t, item1, found)

		// Ensure we can't enqueue a start item again having the same singleton key
		_, err = q.EnqueueItem(ctx, q.primaryQueueShard, qi1, start, osqueue.EnqueueOpts{})
		require.Equal(t, ErrQueueItemSingletonExists, err)

		// Ensure we can enqueue other queue items
		item2, err := q.EnqueueItem(ctx, q.primaryQueueShard, qi2, start, osqueue.EnqueueOpts{})

		require.NoError(t, err)
		require.NotEqual(t, qi2.ID, item2.ID)
		found2 := getQueueItem(t, r, item2.ID)
		require.Equal(t, item2, found2)

		// Dequeue the first item
		err = q.Dequeue(ctx, q.primaryQueueShard, item1)
		require.NoError(t, err)

		// Enqueuing a start item should still not be possible
		_, err = q.EnqueueItem(ctx, q.primaryQueueShard, qi1, start, osqueue.EnqueueOpts{})
		require.Equal(t, ErrQueueItemSingletonExists, err)

		// Dequeue the last item
		err = q.Dequeue(ctx, q.primaryQueueShard, item2)
		require.NoError(t, err)

		// Ensure we can enqueue a new start item with the singleton key after the last dequeue.
		item1, err = q.EnqueueItem(ctx, q.primaryQueueShard, qi1, start, osqueue.EnqueueOpts{})

		require.NoError(t, err)
		require.NotEqual(t, qi1.ID, item1.ID)
		newQueueItem := getQueueItem(t, r, item1.ID)
		require.NotEqual(t, found.ID, newQueueItem.ID)
	})

	t.Run("It does not release the singleton when dequeuing if it's locked by a different run", func(t *testing.T) {
		key := "example-cancel"
		start := time.Now().Truncate(time.Second)

		runId1 := ulid.MustNew(ulid.Now(), rand.Reader)
		qi1 := osqueue.QueueItem{
			Data: osqueue.Item{
				Kind: osqueue.KindStart,
				Identifier: state.Identifier{
					RunID: runId1,
				},
				Singleton: &osqueue.Singleton{
					Key: key,
				},
			},
		}

		runId2 := ulid.MustNew(ulid.Now(), rand.Reader)
		qi2 := osqueue.QueueItem{
			Data: osqueue.Item{
				Kind: osqueue.KindStart,
				Identifier: state.Identifier{
					RunID: runId2,
				},
				Singleton: &osqueue.Singleton{
					Key: key,
				},
			},
		}

		item1, err := q.EnqueueItem(ctx, q.primaryQueueShard, qi1, start, osqueue.EnqueueOpts{})

		require.NoError(t, err)
		require.NotEqual(t, qi1.ID, item1.ID)
		found := getQueueItem(t, r, item1.ID)
		require.Equal(t, item1, found)

		// Simulate locking release
		deleted := r.Del(kg.SingletonKey(&osqueue.Singleton{
			Key: key,
		}))

		require.Equal(t, deleted, true)

		start = time.Now().Truncate(time.Second)
		// Enqueue the new run
		item2, err := q.EnqueueItem(ctx, q.primaryQueueShard, qi2, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		require.NotEqual(t, qi1.ID, item2.ID)
		found = getQueueItem(t, r, item2.ID)
		require.Equal(t, item2, found)

		// Dequeue the first item
		err = q.Dequeue(ctx, q.primaryQueueShard, item1)
		require.NoError(t, err)

		singletonRun, err := r.Get(kg.SingletonKey(&osqueue.Singleton{
			Key: key,
		}))

		// Check that the lock isn't released because the first run doesn't own it anymore
		require.NoError(t, err)
		require.Equal(t, runId2.String(), singletonRun)

		// Dequeue the second item
		err = q.Dequeue(ctx, q.primaryQueueShard, item2)
		require.NoError(t, err)

		// Now the lock should be released
		locked := r.Exists(kg.SingletonKey(&osqueue.Singleton{
			Key: key,
		}))
		require.False(t, locked)
	})
}

func TestQueueActiveCounters(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
	kg := defaultShard.RedisClient.kg

	enqueueToBacklog := false

	clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))
	q := NewQueue(
		defaultShard,
		WithClock(clock),
		WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
			return enqueueToBacklog
		}),
	)
	ctx := context.Background()

	scard := func(key string) int {
		if !r.Exists(key) {
			return 0
		}

		val, err := rc.Do(ctx, rc.B().Scard().Key(key).Build()).AsInt64()
		require.NoError(t, err)

		return int(val)
	}

	accountID, fnID, envID := uuid.New(), uuid.New(), uuid.New()

	t.Run("single item", func(t *testing.T) {
		runID := ulid.MustNew(ulid.Timestamp(clock.Now()), rand.Reader)

		item := osqueue.QueueItem{
			ID:          "test",
			FunctionID:  fnID,
			WorkspaceID: envID,
			Data: osqueue.Item{
				WorkspaceID: envID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID:  fnID,
					AccountID:   accountID,
					WorkspaceID: envID,
					RunID:       runID,
				},
				QueueName:             nil,
				Throttle:              nil,
				CustomConcurrencyKeys: nil,
			},
			QueueName: nil,
		}

		at := clock.Now()

		t.Run("from backlog, requeue", func(t *testing.T) {
			r.FlushAll()

			// enqueue to backlog
			enqueueToBacklog = true
			i, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			shadowPart := q.ItemShadowPartition(ctx, item)
			backlog := q.ItemBacklog(ctx, item)

			refillUntil := at.Add(time.Minute)

			require.Equal(t, 0, scard(kg.ActiveRunsSet("p", shadowPart.PartitionID)))
			require.Equal(t, 0, scard(kg.ActiveRunsSet("account", accountID.String())))
			require.Equal(t, 0, scard(kg.RunActiveSet(runID)))
			require.Equal(t, 0, scard(kg.ActiveSet("p", fnID.String())))
			require.Equal(t, 0, scard(kg.ActiveSet("account", accountID.String())))

			require.Empty(t, i.RefilledFrom)
			require.Zero(t, i.RefilledAt)

			// refill
			// Get items to refill from backlog
			itemIDs, err := getItemIDsFromBacklog(ctx, q, &backlog, refillUntil, 1000)
			require.NoError(t, err)

			res, err := q.BacklogRefill(ctx, &backlog, &shadowPart, refillUntil, itemIDs, PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					SystemConcurrency:   consts.DefaultConcurrencyLimit,
					AccountConcurrency:  consts.DefaultConcurrencyLimit,
					FunctionConcurrency: consts.DefaultConcurrencyLimit,
				},
			})
			require.NoError(t, err)

			require.Equal(t, 1, res.Refilled)

			require.Equal(t, 1, scard(kg.ActiveRunsSet("p", shadowPart.PartitionID)))
			require.Equal(t, 1, scard(kg.ActiveRunsSet("account", accountID.String())))
			require.Equal(t, 1, scard(kg.RunActiveSet(runID)))
			require.Equal(t, 1, scard(kg.ActiveSet("p", fnID.String())))
			require.Equal(t, 1, scard(kg.ActiveSet("account", accountID.String())))

			currentItemStr := r.HGet(kg.QueueItem(), i.ID)
			require.NoError(t, json.Unmarshal([]byte(currentItemStr), &i))
			require.Equal(t, backlog.BacklogID, i.RefilledFrom)
			require.Equal(t, clock.Now(), time.UnixMilli(i.RefilledAt))

			// lease
			leaseID, err := q.Lease(ctx, i, 10*time.Second, clock.Now(), nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			require.Equal(t, 1, scard(kg.ActiveRunsSet("p", shadowPart.PartitionID)))
			require.Equal(t, 1, scard(kg.ActiveRunsSet("account", accountID.String())))
			require.Equal(t, 1, scard(kg.RunActiveSet(runID)))
			require.Equal(t, 1, scard(kg.ActiveSet("p", fnID.String())))
			require.Equal(t, 1, scard(kg.ActiveSet("account", accountID.String())))

			// requeue to backlog
			requeueAt := clock.Now().Add(time.Minute)
			enqueueToBacklog = true
			require.NoError(t, q.Requeue(ctx, defaultShard, i, requeueAt))

			require.Equal(t, 0, scard(kg.ActiveRunsSet("p", shadowPart.PartitionID)))
			require.Equal(t, 0, scard(kg.ActiveRunsSet("account", accountID.String())))
			require.Equal(t, 0, scard(kg.RunActiveSet(runID)))
			require.Equal(t, 0, scard(kg.ActiveSet("p", fnID.String())))
			require.Equal(t, 0, scard(kg.ActiveSet("account", accountID.String())))
		})

		t.Run("from backlog, dequeue", func(t *testing.T) {
			r.FlushAll()

			// enqueue to backlog
			enqueueToBacklog = true
			i, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			shadowPart := q.ItemShadowPartition(ctx, item)
			backlog := q.ItemBacklog(ctx, item)

			refillUntil := at.Add(time.Minute)

			require.Equal(t, 0, scard(kg.ActiveRunsSet("p", shadowPart.PartitionID)))
			require.Equal(t, 0, scard(kg.ActiveRunsSet("account", accountID.String())))
			require.Equal(t, 0, scard(kg.RunActiveSet(runID)))
			require.Equal(t, 0, scard(kg.ActiveSet("p", fnID.String())))
			require.Equal(t, 0, scard(kg.ActiveSet("account", accountID.String())))

			require.Empty(t, i.RefilledFrom)
			require.Zero(t, i.RefilledAt)

			// refill
			// Get items to refill from backlog
			itemIDs, err := getItemIDsFromBacklog(ctx, q, &backlog, refillUntil, 1000)
			require.NoError(t, err)

			res, err := q.BacklogRefill(ctx, &backlog, &shadowPart, refillUntil, itemIDs, PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					SystemConcurrency:   consts.DefaultConcurrencyLimit,
					AccountConcurrency:  consts.DefaultConcurrencyLimit,
					FunctionConcurrency: consts.DefaultConcurrencyLimit,
				},
			})
			require.NoError(t, err)

			require.Equal(t, 1, res.Refilled)

			require.Equal(t, 1, scard(kg.ActiveRunsSet("p", shadowPart.PartitionID)))
			require.Equal(t, 1, scard(kg.ActiveRunsSet("account", accountID.String())))
			require.Equal(t, 1, scard(kg.RunActiveSet(runID)))
			require.Equal(t, 1, scard(kg.ActiveSet("p", fnID.String())))
			require.Equal(t, 1, scard(kg.ActiveSet("account", accountID.String())))

			currentItemStr := r.HGet(kg.QueueItem(), i.ID)
			require.NoError(t, json.Unmarshal([]byte(currentItemStr), &i))
			require.Equal(t, backlog.BacklogID, i.RefilledFrom)
			require.Equal(t, clock.Now(), time.UnixMilli(i.RefilledAt))

			// lease
			leaseID, err := q.Lease(ctx, i, 10*time.Second, clock.Now(), nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			require.Equal(t, 1, scard(kg.ActiveRunsSet("p", shadowPart.PartitionID)))
			require.Equal(t, 1, scard(kg.ActiveRunsSet("account", accountID.String())))
			require.Equal(t, 1, scard(kg.RunActiveSet(runID)))
			require.Equal(t, 1, scard(kg.ActiveSet("p", fnID.String())))
			require.Equal(t, 1, scard(kg.ActiveSet("account", accountID.String())))

			// dequeue
			require.NoError(t, q.Dequeue(ctx, defaultShard, i))

			require.Equal(t, 0, scard(kg.ActiveRunsSet("p", shadowPart.PartitionID)))
			require.Equal(t, 0, scard(kg.ActiveRunsSet("account", accountID.String())))
			require.Equal(t, 0, scard(kg.RunActiveSet(runID)))
			require.Equal(t, 0, scard(kg.ActiveSet("p", fnID.String())))
			require.Equal(t, 0, scard(kg.ActiveSet("account", accountID.String())))
		})

		t.Run("from ready queue, requeue", func(t *testing.T) {
			r.FlushAll()

			// enqueue to backlog
			enqueueToBacklog = false
			i, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			shadowPart := q.ItemShadowPartition(ctx, item)

			require.Equal(t, 0, scard(kg.ActiveRunsSet("p", shadowPart.PartitionID)))
			require.Equal(t, 0, scard(kg.ActiveRunsSet("account", accountID.String())))
			require.Equal(t, 0, scard(kg.RunActiveSet(runID)))
			require.Equal(t, 0, scard(kg.ActiveSet("p", fnID.String())))
			require.Equal(t, 0, scard(kg.ActiveSet("account", accountID.String())))

			// lease
			leaseID, err := q.Lease(ctx, i, 10*time.Second, clock.Now(), nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			require.Equal(t, 1, scard(kg.ActiveRunsSet("p", shadowPart.PartitionID)))
			require.Equal(t, 1, scard(kg.ActiveRunsSet("account", accountID.String())))
			require.Equal(t, 1, scard(kg.RunActiveSet(runID)))
			require.Equal(t, 1, scard(kg.ActiveSet("p", fnID.String())))
			require.Equal(t, 1, scard(kg.ActiveSet("account", accountID.String())))

			// requeue to ready partition
			requeueAt := clock.Now().Add(time.Minute)
			enqueueToBacklog = false
			require.NoError(t, q.Requeue(ctx, defaultShard, i, requeueAt))

			require.Equal(t, 0, scard(kg.ActiveRunsSet("p", shadowPart.PartitionID)))
			require.Equal(t, 0, scard(kg.ActiveRunsSet("account", accountID.String())))
			require.Equal(t, 0, scard(kg.RunActiveSet(runID)))
			require.Equal(t, 0, scard(kg.ActiveSet("p", fnID.String())))
			require.Equal(t, 0, scard(kg.ActiveSet("account", accountID.String())))
		})

		t.Run("from ready queue, dequeue", func(t *testing.T) {
			r.FlushAll()

			// enqueue to backlog
			enqueueToBacklog = false
			i, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			shadowPart := q.ItemShadowPartition(ctx, item)

			require.Equal(t, 0, scard(kg.ActiveRunsSet("p", shadowPart.PartitionID)))
			require.Equal(t, 0, scard(kg.ActiveRunsSet("account", accountID.String())))
			require.Equal(t, 0, scard(kg.RunActiveSet(runID)))
			require.Equal(t, 0, scard(kg.ActiveSet("p", fnID.String())))
			require.Equal(t, 0, scard(kg.ActiveSet("account", accountID.String())))

			// lease
			leaseID, err := q.Lease(ctx, i, 10*time.Second, clock.Now(), nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			require.Equal(t, 1, scard(kg.ActiveRunsSet("p", shadowPart.PartitionID)))
			require.Equal(t, 1, scard(kg.ActiveRunsSet("account", accountID.String())))
			require.Equal(t, 1, scard(kg.RunActiveSet(runID)))
			require.Equal(t, 1, scard(kg.ActiveSet("p", fnID.String())))
			require.Equal(t, 1, scard(kg.ActiveSet("account", accountID.String())))

			// dequeue
			require.NoError(t, q.Dequeue(ctx, defaultShard, i))

			require.Equal(t, 0, scard(kg.ActiveRunsSet("p", shadowPart.PartitionID)))
			require.Equal(t, 0, scard(kg.ActiveRunsSet("account", accountID.String())))
			require.Equal(t, 0, scard(kg.RunActiveSet(runID)))
			require.Equal(t, 0, scard(kg.ActiveSet("p", fnID.String())))
			require.Equal(t, 0, scard(kg.ActiveSet("account", accountID.String())))
		})
	})

	t.Run("multiple items", func(t *testing.T) {
		runIDA := ulid.MustNew(ulid.Timestamp(clock.Now()), rand.Reader)

		itemA1 := osqueue.QueueItem{
			FunctionID:  fnID,
			WorkspaceID: envID,
			Data: osqueue.Item{
				WorkspaceID: envID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID:  fnID,
					AccountID:   accountID,
					WorkspaceID: envID,
					RunID:       runIDA,
				},
				QueueName:             nil,
				Throttle:              nil,
				CustomConcurrencyKeys: nil,
			},
			QueueName: nil,
		}

		itemA2 := osqueue.QueueItem{
			FunctionID:  fnID,
			WorkspaceID: envID,
			Data: osqueue.Item{
				WorkspaceID: envID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID:  fnID,
					AccountID:   accountID,
					WorkspaceID: envID,
					RunID:       runIDA,
				},
				QueueName:             nil,
				Throttle:              nil,
				CustomConcurrencyKeys: nil,
			},
			QueueName: nil,
		}

		runIDB := ulid.MustNew(ulid.Timestamp(clock.Now()), rand.Reader)

		itemB1 := osqueue.QueueItem{
			FunctionID:  fnID,
			WorkspaceID: envID,
			Data: osqueue.Item{
				WorkspaceID: envID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID:  fnID,
					AccountID:   accountID,
					WorkspaceID: envID,
					RunID:       runIDB,
				},
				QueueName:             nil,
				Throttle:              nil,
				CustomConcurrencyKeys: nil,
			},
			QueueName: nil,
		}

		itemB2 := osqueue.QueueItem{
			FunctionID:  fnID,
			WorkspaceID: envID,
			Data: osqueue.Item{
				WorkspaceID: envID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID:  fnID,
					AccountID:   accountID,
					WorkspaceID: envID,
					RunID:       runIDB,
				},
				QueueName:             nil,
				Throttle:              nil,
				CustomConcurrencyKeys: nil,
			},
			QueueName: nil,
		}

		at := clock.Now()

		t.Run("from backlog, requeue", func(t *testing.T) {
			r.FlushAll()

			//
			// Enqueue all
			//

			// enqueue to backlog
			enqueueToBacklog = true
			iA1, err := q.EnqueueItem(ctx, defaultShard, itemA1, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			iA2, err := q.EnqueueItem(ctx, defaultShard, itemA2, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			iB1, err := q.EnqueueItem(ctx, defaultShard, itemB1, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			iB2, err := q.EnqueueItem(ctx, defaultShard, itemB2, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			shadowPart := q.ItemShadowPartition(ctx, itemA1)
			backlog := q.ItemBacklog(ctx, itemA1)

			refillUntil := at.Add(time.Minute)

			require.Equal(t, 0, scard(kg.ActiveRunsSet("p", shadowPart.PartitionID)))
			require.Equal(t, 0, scard(kg.ActiveRunsSet("account", accountID.String())))
			require.Equal(t, 0, scard(kg.RunActiveSet(runIDA)))
			require.Equal(t, 0, scard(kg.RunActiveSet(runIDB)))
			require.Equal(t, 0, scard(kg.ActiveSet("p", fnID.String())))
			require.Equal(t, 0, scard(kg.ActiveSet("account", accountID.String())))

			require.Empty(t, iA1.RefilledFrom)
			require.Zero(t, iA1.RefilledAt)

			//
			// Refill all
			//

			// refill
			// Get items to refill from backlog
			itemIDs, err := getItemIDsFromBacklog(ctx, q, &backlog, refillUntil, 1000)
			require.NoError(t, err)

			res, err := q.BacklogRefill(ctx, &backlog, &shadowPart, refillUntil, itemIDs, PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					SystemConcurrency:   consts.DefaultConcurrencyLimit,
					AccountConcurrency:  consts.DefaultConcurrencyLimit,
					FunctionConcurrency: consts.DefaultConcurrencyLimit,
				},
			})
			require.NoError(t, err)

			require.Equal(t, 4, res.Refilled)

			require.Equal(t, 2, scard(kg.ActiveRunsSet("p", shadowPart.PartitionID)))
			require.Equal(t, 2, scard(kg.ActiveRunsSet("account", accountID.String())))
			require.Equal(t, 2, scard(kg.RunActiveSet(runIDA)))
			require.Equal(t, 2, scard(kg.RunActiveSet(runIDB)))
			require.Equal(t, 4, scard(kg.ActiveSet("p", fnID.String())))
			require.Equal(t, 4, scard(kg.ActiveSet("account", accountID.String())))

			//
			// Process A1
			//

			currentItemStr := r.HGet(kg.QueueItem(), iA1.ID)
			require.NoError(t, json.Unmarshal([]byte(currentItemStr), &iA1))
			require.Equal(t, backlog.BacklogID, iA1.RefilledFrom)
			require.Equal(t, clock.Now(), time.UnixMilli(iA1.RefilledAt))

			// lease
			leaseID, err := q.Lease(ctx, iA1, 10*time.Second, clock.Now(), nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			require.Equal(t, 2, scard(kg.ActiveRunsSet("p", shadowPart.PartitionID)))
			require.Equal(t, 2, scard(kg.ActiveRunsSet("account", accountID.String())))
			require.Equal(t, 2, scard(kg.RunActiveSet(runIDA)))
			require.Equal(t, 2, scard(kg.RunActiveSet(runIDB)))
			require.Equal(t, 4, scard(kg.ActiveSet("p", fnID.String())))
			require.Equal(t, 4, scard(kg.ActiveSet("account", accountID.String())))

			// requeue to backlog
			requeueAt := clock.Now().Add(time.Minute)
			enqueueToBacklog = true
			require.NoError(t, q.Requeue(ctx, defaultShard, iA1, requeueAt))

			require.Equal(t, 2, scard(kg.ActiveRunsSet("p", shadowPart.PartitionID)))
			require.Equal(t, 2, scard(kg.ActiveRunsSet("account", accountID.String())))
			require.Equal(t, 1, scard(kg.RunActiveSet(runIDA)))
			require.Equal(t, 2, scard(kg.RunActiveSet(runIDB)))
			require.Equal(t, 3, scard(kg.ActiveSet("p", fnID.String())))
			require.Equal(t, 3, scard(kg.ActiveSet("account", accountID.String())))

			//
			// Process A2
			//

			currentItemStr = r.HGet(kg.QueueItem(), iA2.ID)
			require.NoError(t, json.Unmarshal([]byte(currentItemStr), &iA2))

			// lease
			leaseID, err = q.Lease(ctx, iA2, 10*time.Second, clock.Now(), nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			require.Equal(t, 2, scard(kg.ActiveRunsSet("p", shadowPart.PartitionID)))
			require.Equal(t, 2, scard(kg.ActiveRunsSet("account", accountID.String())))
			require.Equal(t, 1, scard(kg.RunActiveSet(runIDA)))
			require.Equal(t, 2, scard(kg.RunActiveSet(runIDB)))
			require.Equal(t, 3, scard(kg.ActiveSet("p", fnID.String())))
			require.Equal(t, 3, scard(kg.ActiveSet("account", accountID.String())))

			// requeue to backlog
			requeueAt = clock.Now().Add(time.Minute)
			enqueueToBacklog = true
			require.NoError(t, q.Requeue(ctx, defaultShard, iA2, requeueAt))

			require.Equal(t, 1, scard(kg.ActiveRunsSet("p", shadowPart.PartitionID)))
			require.Equal(t, 1, scard(kg.ActiveRunsSet("account", accountID.String())))
			require.Equal(t, 0, scard(kg.RunActiveSet(runIDA)))
			require.Equal(t, 2, scard(kg.RunActiveSet(runIDB)))
			require.Equal(t, 2, scard(kg.ActiveSet("p", fnID.String())))
			require.Equal(t, 2, scard(kg.ActiveSet("account", accountID.String())))

			//
			// Process B1
			//

			currentItemStr = r.HGet(kg.QueueItem(), iB1.ID)
			require.NoError(t, json.Unmarshal([]byte(currentItemStr), &iB1))
			_, err = q.Lease(ctx, iB1, 10*time.Second, clock.Now(), nil)
			require.NoError(t, err)
			require.NoError(t, q.Requeue(ctx, defaultShard, iB1, requeueAt))

			require.Equal(t, 1, scard(kg.ActiveRunsSet("p", shadowPart.PartitionID)))
			require.Equal(t, 1, scard(kg.ActiveRunsSet("account", accountID.String())))
			require.Equal(t, 0, scard(kg.RunActiveSet(runIDA)))
			require.Equal(t, 1, scard(kg.RunActiveSet(runIDB)))
			require.Equal(t, 1, scard(kg.ActiveSet("p", fnID.String())))
			require.Equal(t, 1, scard(kg.ActiveSet("account", accountID.String())))

			//
			// Process B2
			//

			currentItemStr = r.HGet(kg.QueueItem(), iB2.ID)
			require.NoError(t, json.Unmarshal([]byte(currentItemStr), &iB2))
			_, err = q.Lease(ctx, iB2, 10*time.Second, clock.Now(), nil)
			require.NoError(t, err)
			require.NoError(t, q.Requeue(ctx, defaultShard, iB2, requeueAt))

			require.Equal(t, 0, scard(kg.ActiveRunsSet("p", shadowPart.PartitionID)))
			require.Equal(t, 0, scard(kg.ActiveRunsSet("account", accountID.String())))
			require.Equal(t, 0, scard(kg.RunActiveSet(runIDA)))
			require.Equal(t, 0, scard(kg.RunActiveSet(runIDB)))
			require.Equal(t, 0, scard(kg.ActiveSet("p", fnID.String())))
			require.Equal(t, 0, scard(kg.ActiveSet("account", accountID.String())))
		})
	})
}

func score(t *testing.T, r *miniredis.Miniredis, key string, member string) float64 {
	require.True(t, r.Exists(key), r.Keys())

	score, err := r.ZScore(key, member)
	require.NoError(t, err)

	return score
}

func hasMember(t *testing.T, r *miniredis.Miniredis, key string, member string) bool {
	if !r.Exists(key) {
		return false
	}

	members, err := r.ZMembers(key)
	require.NoError(t, err)

	for _, s := range members {
		if s == member {
			return true
		}
	}
	return false
}

func zcard(t *testing.T, rc rueidis.Client, key string) int {
	cmd := rc.B().Zcard().Key(key).Build()
	num, err := rc.Do(context.Background(), cmd).ToInt64()
	if rueidis.IsRedisNil(err) {
		return 0
	}
	require.NoError(t, err)

	return int(num)
}

func TestInvalidScoreOnRefill(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}

	constraints := PartitionConstraintConfig{
		Concurrency: PartitionConcurrency{
			AccountConcurrency:  100,
			FunctionConcurrency: 20,
		},
	}
	clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))
	q := NewQueue(
		defaultShard,
		WithClock(clock),
		WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
			return true
		}),
		WithPartitionConstraintConfigGetter(func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
			return constraints
		}),
	)
	ctx := context.Background()

	accountID, fnID, envID := uuid.New(), uuid.New(), uuid.New()

	runID := ulid.MustNew(ulid.Timestamp(clock.Now()), rand.Reader)

	item1 := osqueue.QueueItem{
		ID:          "test",
		FunctionID:  fnID,
		WorkspaceID: envID,
		Data: osqueue.Item{
			WorkspaceID: envID,
			Kind:        osqueue.KindEdge,
			Identifier: state.Identifier{
				WorkflowID:  fnID,
				AccountID:   accountID,
				WorkspaceID: envID,
				RunID:       runID,
			},
			QueueName:             nil,
			Throttle:              nil,
			CustomConcurrencyKeys: nil,
		},
		QueueName: nil,
	}

	item2 := osqueue.QueueItem{
		ID:          "test2",
		FunctionID:  fnID,
		WorkspaceID: envID,
		Data: osqueue.Item{
			WorkspaceID: envID,
			Kind:        osqueue.KindEdge,
			Identifier: state.Identifier{
				WorkflowID:  fnID,
				AccountID:   accountID,
				WorkspaceID: envID,
				RunID:       runID,
			},
			QueueName:             nil,
			Throttle:              nil,
			CustomConcurrencyKeys: nil,
		},
		QueueName: nil,
	}

	qi, err := q.EnqueueItem(ctx, defaultShard, item1, clock.Now(), osqueue.EnqueueOpts{})
	require.NoError(t, err)

	qi2, err := q.EnqueueItem(ctx, defaultShard, item2, clock.Now(), osqueue.EnqueueOpts{})
	require.NoError(t, err)

	backlog := q.ItemBacklog(ctx, qi)
	sp := q.ItemShadowPartition(ctx, qi)

	removed, err := r.ZRem(
		defaultShard.RedisClient.kg.BacklogSet(backlog.BacklogID),
		qi.ID,
	)
	require.NoError(t, err)
	require.True(t, removed)

	res, err := q.BacklogRefill(
		ctx,
		&backlog,
		&sp,
		clock.Now().Add(time.Minute),
		[]string{
			qi.ID,
			qi2.ID,
		},
		constraints,
	)
	require.NoError(t, err)

	require.Equal(t, 1, res.Refilled)
	require.Equal(t, qi2.ID, res.RefilledItems[0])
}
