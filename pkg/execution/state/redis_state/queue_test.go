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

	"github.com/inngest/inngest/pkg/consts"
	"github.com/stretchr/testify/assert"

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

	t.Run("Enqueueing to a paused partition does not affect the partition's pause state", func(t *testing.T) {
		now := time.Now()
		workflowId := uuid.New()
		accountId := uuid.New()

		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			FunctionID: workflowId,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					WorkflowID: workflowId,
					AccountID:  accountId,
				},
			},
		}, now.Add(10*time.Second), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		err = q.SetFunctionPaused(ctx, accountId, item.FunctionID, true)
		require.NoError(t, err)

		item, err = q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			FunctionID: workflowId,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					WorkflowID: workflowId,
					AccountID:  accountId,
				},
			},
		}, now, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		fnMeta, err := getFnMetadata(t, r, item.FunctionID)
		require.NoError(t, err)
		require.True(t, fnMeta.Paused)

		item, err = q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			FunctionID: workflowId,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					WorkflowID: workflowId,
					AccountID:  accountId,
				},
			},
		}, now.Add(-10*time.Second), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		fnMeta, err = getFnMetadata(t, r, item.FunctionID)
		require.NoError(t, err)
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
		WithDisableLeaseChecksForSystemQueues(false),
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

func TestQueueLease(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	queueClient := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
	q := NewQueue(queueClient)
	defaultQueueKey := q.primaryQueueShard.RedisClient.kg

	ctx := context.Background()

	start := time.Now().Truncate(time.Second)

	t.Run("It leases an item", func(t *testing.T) {
		fnID, accountID := uuid.New(), uuid.New()
		runID := ulid.MustNew(ulid.Now(), rand.Reader)

		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					RunID:      runID,
					WorkflowID: fnID,
					AccountID:  accountID,
				},
			},
		}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		p := QueuePartition{
			ID:         fnID.String(),
			FunctionID: &fnID,
			AccountID:  accountID,
		} // Default workflow ID etc

		t.Run("It should exist in the pending partition queue", func(t *testing.T) {
			mem, err := r.ZMembers(p.zsetKey(q.primaryQueueShard.RedisClient.kg))
			require.NoError(t, err)
			require.Equal(t, 1, len(mem))
		})

		now := time.Now()
		leaseExpiry := now.Add(time.Second)
		id, err := q.Lease(ctx, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, id, item.LeaseID)
		require.WithinDuration(t, leaseExpiry, ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		t.Run("It should remove from the pending partition queue", func(t *testing.T) {
			mem, _ := r.ZMembers(p.zsetKey(q.primaryQueueShard.RedisClient.kg))
			require.Empty(t, mem)
		})

		t.Run("It should add the item to the function's in-progress concurrency queue", func(t *testing.T) {
			count, err := q.InProgress(ctx, "p", fnID.String())
			require.NoError(t, err)
			require.EqualValues(t, 1, count, r.Dump())
		})

		t.Run("run indexes are updated", func(t *testing.T) {
			kg := q.primaryQueueShard.RedisClient.kg
			// Run indexes should be updated
			{
				itemIsMember, err := r.SIsMember(kg.ActiveSet("run", runID.String()), item.ID)
				require.NoError(t, err)
				require.True(t, itemIsMember)

				isMember, err := r.SIsMember(kg.ActiveRunsSet("p", fnID.String()), runID.String())
				require.NoError(t, err)
				require.True(t, isMember)

				accountIsMember, err := r.SIsMember(kg.ActiveRunsSet("account", accountID.String()), runID.String())
				require.NoError(t, err)
				require.True(t, accountIsMember)
			}
		})

		t.Run("Scavenge queue is updated", func(t *testing.T) {
			mem, err := r.ZMembers(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex())
			require.NoError(t, err)
			require.Equal(t, 1, len(mem), "scavenge queue should have 1 item", mem)
			require.Contains(t, mem, p.FunctionID.String())

			score, err := r.ZMScore(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex(), p.FunctionID.String())
			require.NoError(t, err)

			require.WithinDuration(t, leaseExpiry, time.UnixMilli(int64(score[0])), 2*time.Millisecond)
		})

		t.Run("Leasing again should fail", func(t *testing.T) {
			for i := 0; i < 50; i++ {
				id, err := q.Lease(ctx, item, time.Second, time.Now(), nil)
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
			id, err := q.Lease(ctx, item, 5*time.Second, time.Now(), nil)
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
			item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			require.Nil(t, item.LeaseID)

			requireItemScoreEquals(t, r, item, start)

			_, err = q.Lease(ctx, item, time.Minute, time.Now(), nil)
			require.NoError(t, err)

			_, err = r.ZScore(q.primaryQueueShard.RedisClient.kg.FnQueueSet(item.FunctionID.String()), item.ID)
			require.Error(t, err, "no such key")
		})

		t.Run("it should not update the partition score to the next item", func(t *testing.T) {
			r.FlushAll()

			timeNow := time.Now().Truncate(time.Second)
			timeNowPlusFiveSeconds := timeNow.Add(time.Second * 5).Truncate(time.Second)

			acctId := uuid.New()

			// Enqueue future item (partition time will be now + 5s)
			item, err = q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
				Data: osqueue.Item{Identifier: state.Identifier{AccountID: acctId}},
			}, timeNowPlusFiveSeconds, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			require.Nil(t, item.LeaseID)

			qp := getDefaultPartition(t, r, uuid.Nil)

			requireItemScoreEquals(t, r, item, timeNowPlusFiveSeconds)
			requirePartitionItemScoreEquals(t, r, q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex(), qp, timeNowPlusFiveSeconds)
			requirePartitionItemScoreEquals(t, r, q.primaryQueueShard.RedisClient.kg.AccountPartitionIndex(acctId), qp, timeNowPlusFiveSeconds)
			requireAccountScoreEquals(t, r, acctId, timeNowPlusFiveSeconds)

			// Enqueue current item (partition time will be moved up to now)
			item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{Data: osqueue.Item{Identifier: state.Identifier{AccountID: acctId}}}, timeNow, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			require.Nil(t, item.LeaseID)

			// We do expect the item score to change!
			requireItemScoreEquals(t, r, item, timeNow)

			requirePartitionItemScoreEquals(t, r, q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex(), qp, timeNow)
			requirePartitionItemScoreEquals(t, r, q.primaryQueueShard.RedisClient.kg.AccountPartitionIndex(acctId), qp, timeNow)
			requireAccountScoreEquals(t, r, acctId, timeNow)

			// Lease item (keeps partition time constant)
			_, err = q.Lease(ctx, item, time.Minute, q.clock.Now(), nil)
			require.NoError(t, err)

			requirePartitionItemScoreEquals(t, r, q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex(), qp, timeNow)
			requirePartitionItemScoreEquals(t, r, q.primaryQueueShard.RedisClient.kg.AccountPartitionIndex(acctId), qp, timeNow)
			requireAccountScoreEquals(t, r, acctId, timeNow)
		})
	})

	// Test default partition-level concurrency limits (not custom)
	t.Run("With partition concurrency limits", func(t *testing.T) {
		r.FlushAll()

		// Only allow a single leased item
		q.partitionConstraintConfigGetter = func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
			return PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					AccountConcurrency:  1,
					FunctionConcurrency: 1,
					SystemConcurrency:   1,
				},
			}
		}

		fnID := uuid.New()
		// Create a new item
		itemA, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: fnID}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		itemB, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: fnID}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		// Use the new item's workflow ID
		p := QueuePartition{ID: itemA.FunctionID.String(), FunctionID: &itemA.FunctionID}

		t.Run("With denylists it does not lease.", func(t *testing.T) {
			list := newLeaseDenyList()
			list.addConcurrency(newKeyError(ErrPartitionConcurrencyLimit, p.Queue()))
			id, err := q.Lease(ctx, itemA, 5*time.Second, time.Now(), list)
			require.NotNil(t, err, "Expcted error leasing denylists")
			require.Nil(t, id, "Expected nil ID with denylists")
			require.ErrorIs(t, err, ErrPartitionConcurrencyLimit)
		})

		t.Run("Leases with capacity", func(t *testing.T) {
			_, err = q.Lease(ctx, itemA, 5*time.Second, time.Now(), nil)
			require.NoError(t, err)
		})

		t.Run("Errors without capacity", func(t *testing.T) {
			id, err := q.Lease(ctx, itemB, 5*time.Second, time.Now(), nil)
			require.Nil(t, id, "Leased item when concurrency limits are reached.\n%s", r.Dump())
			require.Error(t, err)
		})
	})

	// Test default account concurrency limits (not custom)
	t.Run("With account concurrency limits", func(t *testing.T) {
		r.FlushAll()

		// Only allow a single leased item via account limits
		q.partitionConstraintConfigGetter = func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
			return PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					AccountConcurrency:  1,
					FunctionConcurrency: NoConcurrencyLimit,
				},
			}
		}

		acctId := uuid.New()

		// Create a new item
		itemA, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: uuid.New(), Data: osqueue.Item{Identifier: state.Identifier{AccountID: acctId}}}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		itemB, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: uuid.New(), Data: osqueue.Item{Identifier: state.Identifier{AccountID: acctId}}}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		t.Run("Leases with capacity", func(t *testing.T) {
			_, err = q.Lease(ctx, itemA, 5*time.Second, time.Now(), nil)
			require.NoError(t, err)
		})

		t.Run("Errors without capacity", func(t *testing.T) {
			id, err := q.Lease(ctx, itemB, 5*time.Second, time.Now(), nil)
			require.Nil(t, id)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrAccountConcurrencyLimit)
		})
	})

	t.Run("With custom concurrency limits", func(t *testing.T) {
		t.Run("with account keys", func(t *testing.T) {
			r.FlushAll()

			ck := createConcurrencyKey(enums.ConcurrencyScopeAccount, uuid.Nil, "foo", 1)

			// Only allow a single leased item via custom concurrency limits
			q.partitionConstraintConfigGetter = func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return PartitionConstraintConfig{
					Concurrency: PartitionConcurrency{
						AccountConcurrency:  NoConcurrencyLimit,
						FunctionConcurrency: NoConcurrencyLimit,
						CustomConcurrencyKeys: []CustomConcurrencyLimit{
							{
								Scope:               enums.ConcurrencyScopeAccount,
								HashedKeyExpression: ck.Hash,
								Limit:               ck.Limit,
							},
						},
					},
				}
			}

			// Create a new item
			fnA := uuid.New()
			itemA, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
				FunctionID: fnA,
				Data: osqueue.Item{
					CustomConcurrencyKeys: []state.CustomConcurrency{
						ck,
					},
				},
			}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			itemB, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
				FunctionID: uuid.New(),
				Data: osqueue.Item{
					CustomConcurrencyKeys: []state.CustomConcurrency{
						ck,
					},
				},
			}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			t.Run("With denylists it does not lease.", func(t *testing.T) {
				list := newLeaseDenyList()
				list.addConcurrency(newKeyError(ErrConcurrencyLimitCustomKey, ck.Key))
				_, err = q.Lease(ctx, itemA, 5*time.Second, time.Now(), list)
				require.NotNil(t, err)
				require.ErrorIs(t, err, ErrConcurrencyLimitCustomKey)
			})

			t.Run("Leases with capacity", func(t *testing.T) {
				now := time.Now()
				_, err = q.Lease(ctx, itemA, 5*time.Second, now, nil)
				require.NoError(t, err)

				t.Run("Scavenge queue is updated", func(t *testing.T) {
					mem, err := r.ZMembers(queueClient.RedisClient.kg.ConcurrencyIndex())
					require.NoError(t, err, r.Dump())
					require.Equal(t, 1, len(mem), "scavenge queue should have 1 item", mem)
					require.Contains(t, mem, fnA.String())

					score, err := r.ZMScore(queueClient.RedisClient.kg.ConcurrencyIndex(), fnA.String())
					require.NoError(t, err)
					require.Equal(t, float64(now.Add(5*time.Second).UnixMilli()), score[0])
				})
			})

			t.Run("Errors without capacity", func(t *testing.T) {
				id, err := q.Lease(ctx, itemB, 5*time.Second, time.Now(), nil)
				require.Nil(t, id)
				require.Error(t, err)
			})
		})

		t.Run("with function keys", func(t *testing.T) {
			r.FlushAll()

			accountId := uuid.New()
			fnId := uuid.New()

			ck := createConcurrencyKey(enums.ConcurrencyScopeFn, fnId, "foo", 1)
			_, _, keyExprChecksum, err := ck.ParseKey()
			require.NoError(t, err)
			// Only allow a single leased item via custom concurrency limits
			q.partitionConstraintConfigGetter = func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return PartitionConstraintConfig{
					Concurrency: PartitionConcurrency{
						AccountConcurrency:  NoConcurrencyLimit,
						FunctionConcurrency: NoConcurrencyLimit,
						CustomConcurrencyKeys: []CustomConcurrencyLimit{
							{
								Scope:               enums.ConcurrencyScopeFn,
								HashedKeyExpression: ck.Hash,
								Limit:               ck.Limit,
							},
						},
					},
				}
			}

			// Create a new item
			itemA, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
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
			}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			itemB, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
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
			}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			zsetKeyA := q.primaryQueueShard.RedisClient.kg.PartitionQueueSet(enums.PartitionTypeConcurrencyKey, fnId.String(), keyExprChecksum)
			pA := QueuePartition{ID: zsetKeyA, AccountID: accountId, FunctionID: &itemA.FunctionID, PartitionType: int(enums.PartitionTypeConcurrencyKey), EvaluatedConcurrencyKey: ck.Key}

			t.Run("With denylists it does not lease.", func(t *testing.T) {
				list := newLeaseDenyList()
				list.addConcurrency(newKeyError(ErrConcurrencyLimitCustomKey, ck.Key))
				_, err = q.Lease(ctx, itemA, 5*time.Second, time.Now(), list)
				require.NotNil(t, err)
				require.ErrorIs(t, err, ErrConcurrencyLimitCustomKey)
			})

			t.Run("Leases with capacity", func(t *testing.T) {
				// Use the new item's workflow ID
				require.Equal(t, pA.zsetKey(q.primaryQueueShard.RedisClient.kg), zsetKeyA)

				// partition key queue does not exist
				require.False(t, r.Exists(pA.zsetKey(q.primaryQueueShard.RedisClient.kg)), "partition shouldn't have been added by enqueue or lease")
				// require.True(t, r.Exists(zsetKeyA))
				// memPart, err := r.ZMembers(zsetKeyA)
				// require.NoError(t, err)
				// require.Equal(t, 2, len(memPart))
				// require.Contains(t, memPart, itemA.ID)
				// require.Contains(t, memPart, itemB.ID)

				// concurrency key queue does not yet exist
				require.False(t, r.Exists(pA.concurrencyKey(q.primaryQueueShard.RedisClient.kg)))

				_, err = q.Lease(ctx, itemA, 5*time.Second, time.Now(), nil)
				require.NoError(t, err)

				// memPart, err = r.ZMembers(zsetKeyA)
				// require.NoError(t, err)
				// require.Equal(t, 1, len(memPart))
				// require.Contains(t, memPart, itemB.ID)

				require.True(t, r.Exists(pA.concurrencyKey(q.primaryQueueShard.RedisClient.kg)))
				memConcurrency, err := r.ZMembers(pA.concurrencyKey(q.primaryQueueShard.RedisClient.kg))
				require.NoError(t, err)
				require.Equal(t, 1, len(memConcurrency))
				require.Contains(t, memConcurrency, itemA.ID)
			})

			t.Run("Errors without capacity", func(t *testing.T) {
				id, err := q.Lease(ctx, itemB, 5*time.Second, time.Now(), nil)
				require.Nil(t, id)
				require.Error(t, err)
				require.ErrorIs(t, err, ErrConcurrencyLimitCustomKey)
			})
		})

		// this test is the unit variant of TestConcurrency_ScopeFunction_FanOut in cloud
		t.Run("with two distinct functions it processes both", func(t *testing.T) {
			r.FlushAll()

			fnIDA := uuid.New()
			fnIDB := uuid.New()

			ckA := createConcurrencyKey(enums.ConcurrencyScopeFn, fnIDA, "foo", 1)
			_, _, evaluatedKeyChecksumA, err := ckA.ParseKey()
			require.NoError(t, err)

			ckB := createConcurrencyKey(enums.ConcurrencyScopeFn, fnIDB, "foo", 1)
			_, _, evaluatedKeyChecksumB, err := ckB.ParseKey()
			require.NoError(t, err)

			q.partitionConstraintConfigGetter = func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return PartitionConstraintConfig{
					Concurrency: PartitionConcurrency{
						AccountConcurrency:  123_456,
						FunctionConcurrency: 1,
						CustomConcurrencyKeys: []CustomConcurrencyLimit{
							{
								Scope:               enums.ConcurrencyScopeFn,
								HashedKeyExpression: ckA.Hash,
								Limit:               ckA.Limit,
							},
							{
								Scope:               enums.ConcurrencyScopeFn,
								HashedKeyExpression: ckB.Hash,
								Limit:               ckB.Limit,
							},
						},
					},
				}
			}

			// Create a new item
			itemA1, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: fnIDA, Data: osqueue.Item{CustomConcurrencyKeys: []state.CustomConcurrency{ckA}}}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			itemA2, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: fnIDA, Data: osqueue.Item{CustomConcurrencyKeys: []state.CustomConcurrency{ckA}}}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			itemB1, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: fnIDB, Data: osqueue.Item{CustomConcurrencyKeys: []state.CustomConcurrency{ckB}}}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			itemB2, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: fnIDB, Data: osqueue.Item{CustomConcurrencyKeys: []state.CustomConcurrency{ckB}}}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			// Use the new item's workflow ID
			zsetKeyA := q.primaryQueueShard.RedisClient.kg.PartitionQueueSet(enums.PartitionTypeConcurrencyKey, fnIDA.String(), evaluatedKeyChecksumA)

			partitionIsMissingInHash(t, r, enums.PartitionTypeConcurrencyKey, fnIDA, evaluatedKeyChecksumA)

			zsetKeyB := q.primaryQueueShard.RedisClient.kg.PartitionQueueSet(enums.PartitionTypeConcurrencyKey, fnIDB.String(), evaluatedKeyChecksumB)
			partitionIsMissingInHash(t, r, enums.PartitionTypeConcurrencyKey, fnIDB, evaluatedKeyChecksumB)

			// Both key queues do not exist
			require.False(t, r.Exists(zsetKeyA))
			require.False(t, r.Exists(zsetKeyB))

			// Lease item A1 - should work
			_, err = q.Lease(ctx, itemA1, 5*time.Second, time.Now(), nil)
			require.NoError(t, err)

			// Lease item B1 - should work
			_, err = q.Lease(ctx, itemB1, 5*time.Second, time.Now(), nil)
			require.NoError(t, err)

			// Lease item A2 - should fail due to custom concurrency limit
			_, err = q.Lease(ctx, itemA2, 5*time.Second, time.Now(), nil)
			require.ErrorIs(t, err, ErrConcurrencyLimitCustomKey)

			// Lease item B1 - should fail due to custom concurrency limit
			_, err = q.Lease(ctx, itemB2, 5*time.Second, time.Now(), nil)
			require.ErrorIs(t, err, ErrConcurrencyLimitCustomKey)
		})
	})

	t.Run("It should update the global partition index", func(t *testing.T) {
		t.Run("With no concurrency keys", func(t *testing.T) {
			r.FlushAll()

			q.partitionConstraintConfigGetter = func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return PartitionConstraintConfig{}
			}

			// NOTE: We need two items to ensure that this updates.  Leasing an
			// item removes it from the fn queue.
			t.Run("With a single item in the queue hwen leasing, nothing updates", func(t *testing.T) {
				at := time.Now().Truncate(time.Second).Add(time.Second)
				accountId := uuid.New()
				item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
					Data: osqueue.Item{Identifier: state.Identifier{AccountID: accountId}},
				}, at, osqueue.EnqueueOpts{})
				require.NoError(t, err)
				p := QueuePartition{FunctionID: &item.FunctionID}

				score, err := r.ZScore(q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex(), p.Queue())
				require.NoError(t, err)
				require.EqualValues(t, at.Unix(), score, r.Dump())

				score, err = r.ZScore(defaultQueueKey.AccountPartitionIndex(accountId), p.Queue())
				require.NoError(t, err)
				require.EqualValues(t, at.Unix(), score, r.Dump())

				// Nothing should update here, as there's nothing left in the fn queue
				// so nothing happens.
				_, err = q.Lease(ctx, item, 10*time.Second, time.Now(), nil)
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
				itemA, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
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
				}, at, osqueue.EnqueueOpts{})
				require.NoError(t, err)
				itemB, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
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
				}, at, osqueue.EnqueueOpts{})
				require.NoError(t, err)

				defaultPartition := getDefaultPartition(t, r, uuid.Nil)

				// The partition should use a custom ID for the concurrency key.
				_, pa1, pa2 := q.ItemPartitions(ctx, q.primaryQueueShard, itemA)

				_, pb1, pb2 := q.ItemPartitions(ctx, q.primaryQueueShard, itemB)

				require.Equal(t, "{queue}:sorted:c:00000000-0000-0000-0000-000000000000<2gu959eo1zbsi>", pa1.ID)
				require.Equal(t, "{queue}:sorted:c:00000000-0000-0000-0000-000000000000<1x6209w26mx6i>", pa2.ID)
				// Ensure the partitions match for two queue items.
				require.Equal(t, "{queue}:sorted:c:00000000-0000-0000-0000-000000000000<2gu959eo1zbsi>", pb1.ID)
				require.Equal(t, "{queue}:sorted:c:00000000-0000-0000-0000-000000000000<1x6209w26mx6i>", pb2.ID)

				// Since we do not enqueue concurrency queues, we need to check for the default partition score
				score, err := r.ZScore(defaultQueueKey.GlobalPartitionIndex(), defaultPartition.ID)
				require.NoError(t, err)
				require.EqualValues(t, at.Unix(), score, r.Dump())

				// Concurrency queue should be emptyu
				t.Run("Concurrency and scavenge queues are empty", func(t *testing.T) {
					mem, _ := r.ZMembers(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex())
					require.Empty(t, mem, "concurrency queue is not empty")
				})

				// Do the lease.
				_, err = q.Lease(ctx, itemA, 10*time.Second, q.clock.Now(), nil)
				require.NoError(t, err)

				// The queue item is removed from each partition
				t.Run("The queue item is removed from each partition", func(t *testing.T) {
					mem, _ := r.ZMembers(defaultPartition.zsetKey(q.primaryQueueShard.RedisClient.kg))
					require.Equal(t, 1, len(mem), "leased item not removed from first partition", defaultPartition.zsetKey(q.primaryQueueShard.RedisClient.kg))
				})

				t.Run("The scavenger queue is updated with just the default partition", func(t *testing.T) {
					mem, _ := r.ZMembers(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex())
					require.Equal(t, 1, len(mem), "scavenge queue not updated", mem)
					require.NotContains(t, mem, pa1.concurrencyKey(q.primaryQueueShard.RedisClient.kg))
					require.NotContains(t, mem, pa2.concurrencyKey(q.primaryQueueShard.RedisClient.kg))
					require.NotContains(t, mem, defaultPartition.concurrencyKey(q.primaryQueueShard.RedisClient.kg))
					require.Contains(t, mem, defaultPartition.FunctionID.String())
				})

				t.Run("Pointer queues don't update with a single queue item", func(t *testing.T) {
					nextScore, err := r.ZScore(defaultQueueKey.GlobalPartitionIndex(), defaultPartition.Queue())
					require.NoError(t, err)
					require.EqualValues(t, int(score), int(nextScore), "score should not equal previous score")
				})
			})
		})

		t.Run("With more than one item in the fn queue, it uses the next val for the global partition index", func(t *testing.T) {
			r.FlushAll()

			atA := time.Now().Truncate(time.Second).Add(time.Second)
			atB := atA.Add(time.Minute)

			itemA, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{}, atA, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			_, err = q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{}, atB, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			p, _, _ := q.ItemPartitions(ctx, q.primaryQueueShard, itemA)

			score, err := r.ZScore(defaultQueueKey.GlobalPartitionIndex(), p.Queue())
			require.NoError(t, err)
			require.EqualValues(t, atA.Unix(), score)

			// Leasing the item should update the score.
			_, err = q.Lease(ctx, itemA, 10*time.Second, time.Now(), nil)
			require.NoError(t, err)

			nextScore, err := r.ZScore(defaultQueueKey.GlobalPartitionIndex(), p.Queue())
			require.NoError(t, err)
			// lease should match first item, as we don't update pointer scores during lease
			require.EqualValues(t, itemA.AtMS/1000, int(nextScore))
			require.EqualValues(t, int(score), int(nextScore), "score should not equal previous score")
		})
	})

	t.Run("It does nothing for a zero value partition", func(t *testing.T) {
		r.FlushAll()

		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		p := QueuePartition{} // Empty partition

		now := time.Now()
		id, err := q.Lease(ctx, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, id, item.LeaseID)
		require.WithinDuration(t, now.Add(time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		t.Run("It should NOT add the item to the function's in-progress concurrency queue", func(t *testing.T) {
			require.False(t, r.Exists(p.concurrencyKey(q.primaryQueueShard.RedisClient.kg)))
		})
	})

	t.Run("system partitions should be leased properly", func(t *testing.T) {
		r.FlushAll()

		systemQueueName := "system-queue"
		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			QueueName: &systemQueueName,
			Data: osqueue.Item{
				QueueName: &systemQueueName,
			},
		}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		require.True(t, r.Exists("{queue}:queue:sorted:system-queue"))

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		p := getSystemPartition(t, r, systemQueueName)

		now := time.Now()
		id, err := q.Lease(ctx, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		require.False(t, r.Exists("{queue}:queue:sorted:system-queue"))
		require.False(t, r.Exists("{queue}:concurrency:account:system-queue"), r.Dump()) // System queues should not have account concurrency set
		require.True(t, r.Exists("{queue}:concurrency:p:system-queue"))

		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, id, item.LeaseID)
		require.WithinDuration(t, now.Add(time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		require.True(t, r.Exists(p.concurrencyKey(q.primaryQueueShard.RedisClient.kg)), r.Dump())
	})

	t.Run("batch system partitions should be leased properly", func(t *testing.T) {
		r.FlushAll()

		systemQueueName := osqueue.KindScheduleBatch
		qi := osqueue.QueueItem{
			QueueName: &systemQueueName,
			Data: osqueue.Item{
				QueueName: &systemQueueName,
			},
		}

		kg := queueKeyGenerator{
			queueDefaultKey: QueueDefaultKey,
			queueItemKeyGenerator: queueItemKeyGenerator{
				queueDefaultKey: QueueDefaultKey,
			},
		}

		// Sanity check: Ensure partitions are created properly and keys match old system
		fnPart, custom1, custom2 := q.ItemPartitions(ctx, q.primaryQueueShard, qi)
		require.Equal(t, QueuePartition{
			ID:        systemQueueName,
			QueueName: &systemQueueName,
		}, fnPart)
		require.True(t, fnPart.IsSystem())
		require.Equal(t, QueuePartition{}, custom1)
		require.Equal(t, QueuePartition{}, custom2)

		require.Equal(t, "{queue}:queue:sorted:schedule-batch", fnPart.zsetKey(kg))
		require.Equal(t, "{queue}:concurrency:p:schedule-batch", fnPart.concurrencyKey(kg))

		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		require.True(t, r.Exists("{queue}:queue:sorted:schedule-batch"))

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		p := getSystemPartition(t, r, systemQueueName)

		now := time.Now()
		id, err := q.Lease(ctx, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		require.False(t, r.Exists("{queue}:queue:sorted:schedule-batch"))

		// batching uses different rules for concurrency keys
		require.True(t, r.Exists("{queue}:concurrency:p:schedule-batch"))
		require.False(t, r.Exists("{queue}:concurrency:account:schedule-batch"), r.Dump())

		require.False(t, r.Exists("{queue}:concurrency:account:00000000-0000-0000-0000-000000000000"), r.Dump())
		require.False(t, r.Exists("{queue}:concurrency:p:00000000-0000-0000-0000-000000000000"))

		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, id, item.LeaseID)
		require.WithinDuration(t, now.Add(time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		require.True(t, r.Exists(p.concurrencyKey(q.primaryQueueShard.RedisClient.kg)), r.Dump())
	})

	t.Run("leasing key queue should clear backward-compat default partition", func(t *testing.T) {
		r.FlushAll()

		// This is required as not dropping items from all partitions during lease will cause a leftover item to be in the default partition
		// When the item has been processed, and we run Dequeue, this only happens on the key queue, and the default partition retains its pointer even though the queue item is deleted
		// This leads to Peek errors in default partitions, including system partitions (encountered missing queue items in partition queue)

		accountId := uuid.New()

		evaluatedKey := util.ConcurrencyKey(enums.ConcurrencyScopeAccount, accountId, "customer-1")
		ck := state.CustomConcurrency{
			Key:   evaluatedKey,
			Hash:  util.XXHash("event.data.customerId"),
			Limit: 10,
		}

		q.partitionConstraintConfigGetter = func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
			return PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					SystemConcurrency:   0,
					AccountConcurrency:  123,
					FunctionConcurrency: 45,
					CustomConcurrencyKeys: []CustomConcurrencyLimit{
						{
							Scope:               enums.ConcurrencyScopeAccount,
							HashedKeyExpression: ck.Hash,
							Limit:               ck.Limit,
						},
					},
				},
			}
		}

		fnId := uuid.New()
		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			FunctionID: fnId,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					AccountID:  accountId,
					WorkflowID: fnId,
				},
				CustomConcurrencyKeys: []state.CustomConcurrency{
					ck,
				},
			},
		}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		kg := queueKeyGenerator{
			queueDefaultKey: QueueDefaultKey,
			queueItemKeyGenerator: queueItemKeyGenerator{
				queueDefaultKey: QueueDefaultKey,
			},
		}

		defaultPart := getDefaultPartition(t, r, fnId)

		require.True(t, r.Exists(defaultPart.zsetKey(kg)))

		concurrencyKeyQueue := QueuePartition{
			ID:                         kg.PartitionQueueSet(enums.PartitionTypeConcurrencyKey, accountId.String(), util.XXHash("customer-1")),
			PartitionType:              int(enums.PartitionTypeConcurrencyKey),
			ConcurrencyScope:           int(enums.ConcurrencyScopeAccount),
			FunctionID:                 &fnId,
			AccountID:                  accountId,
			EvaluatedConcurrencyKey:    fmt.Sprintf("a:%s:%s", accountId, util.XXHash("customer-1")),
			UnevaluatedConcurrencyHash: util.XXHash("event.data.customerId"),
		}

		// account-scoped custom concurrency queue should not exist
		require.False(t, r.Exists(concurrencyKeyQueue.zsetKey(kg)), evaluatedKey, concurrencyKeyQueue.zsetKey(kg), r.Dump())

		now := time.Now()
		id, err := q.Lease(ctx, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, id, item.LeaseID)
		require.WithinDuration(t, now.Add(time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		require.False(t, r.Exists(defaultPart.zsetKey(kg)))
		require.False(t, r.Exists(concurrencyKeyQueue.zsetKey(kg)), evaluatedKey, concurrencyKeyQueue.zsetKey(kg), r.Dump())

		require.True(t, r.Exists(concurrencyKeyQueue.concurrencyKey(kg)), r.Dump(), concurrencyKeyQueue.concurrencyKey(kg))
		require.True(t, r.Exists(defaultPart.concurrencyKey(kg)), evaluatedKey, concurrencyKeyQueue.concurrencyKey(kg), r.Dump())
		require.True(t, r.Exists(kg.Concurrency("account", accountId.String())))

		err = q.Dequeue(ctx, q.primaryQueueShard, item)
		require.NoError(t, err)

		require.False(t, r.Exists(defaultPart.zsetKey(kg)))
		require.False(t, r.Exists(concurrencyKeyQueue.zsetKey(kg)), evaluatedKey, concurrencyKeyQueue.zsetKey(kg), r.Dump())

		require.False(t, r.Exists(concurrencyKeyQueue.concurrencyKey(kg)), r.Dump())
		require.False(t, r.Exists(defaultPart.concurrencyKey(kg)), evaluatedKey, concurrencyKeyQueue.concurrencyKey(kg), r.Dump())
		require.False(t, r.Exists(kg.Concurrency("account", accountId.String())))
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

	queueClient := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
	q := NewQueue(queueClient)
	ctx := context.Background()

	start := time.Now().Truncate(time.Second)
	t.Run("It leases an item", func(t *testing.T) {
		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		p := q.ItemPartition(ctx, q.primaryQueueShard, item)

		now := time.Now()
		id, err := q.Lease(ctx, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, id, item.LeaseID)
		require.WithinDuration(t, now.Add(time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		now = time.Now()
		nextID, err := q.ExtendLease(ctx, item, *id, 10*time.Second)
		require.NoError(t, err)

		require.False(t, r.Exists(QueuePartition{}.concurrencyKey(q.primaryQueueShard.RedisClient.kg)))

		// Ensure the leased item has the next ID.
		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, nextID, item.LeaseID)
		require.WithinDuration(t, now.Add(10*time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		t.Run("It extends the score of the partition concurrency queue", func(t *testing.T) {
			at := ulid.Time(nextID.Time())
			scores := concurrencyQueueScores(t, r, p.concurrencyKey(q.primaryQueueShard.RedisClient.kg), time.Now())
			require.Len(t, scores, 1)
			// Ensure that the score matches the lease.
			require.Equal(t, at, scores[item.ID], "%s not extended\n%s", p.concurrencyKey(q.primaryQueueShard.RedisClient.kg), r.Dump())
		})

		t.Run("It fails with an invalid lease ID", func(t *testing.T) {
			invalid := ulid.MustNew(ulid.Now(), rnd)
			nextID, err := q.ExtendLease(ctx, item, invalid, 10*time.Second)
			require.EqualValues(t, ErrQueueItemLeaseMismatch, err)
			require.Nil(t, nextID)
		})
	})

	t.Run("It does not extend an unleased item", func(t *testing.T) {
		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		nextID, err := q.ExtendLease(ctx, item, ulid.Zero, 10*time.Second)
		require.EqualValues(t, ErrQueueItemNotLeased, err)
		require.Nil(t, nextID)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)
	})

	t.Run("With custom keys in multiple partitions", func(t *testing.T) {
		r.FlushAll()

		ckA := state.CustomConcurrency{
			Key: util.ConcurrencyKey(
				enums.ConcurrencyScopeAccount,
				uuid.Nil,
				"acct-id",
			),
			Limit: 10,
		}
		ckB := state.CustomConcurrency{
			Key: util.ConcurrencyKey(
				enums.ConcurrencyScopeFn,
				uuid.Nil,
				"fn-id",
			),
			Limit: 5,
		}

		q.partitionConstraintConfigGetter = func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
			return PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					AccountConcurrency:  123,
					FunctionConcurrency: 45,
					CustomConcurrencyKeys: []CustomConcurrencyLimit{
						{
							Scope:               enums.ConcurrencyScopeAccount,
							HashedKeyExpression: ckA.Hash,
							Limit:               ckA.Limit,
						},
						{
							Scope:               enums.ConcurrencyScopeFn,
							HashedKeyExpression: ckB.Hash,
							Limit:               ckB.Limit,
						},
					},
				},
			}
		}

		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			FunctionID: uuid.New(),
			Data: osqueue.Item{
				CustomConcurrencyKeys: []state.CustomConcurrency{
					ckA,
					ckB,
				},
			},
		}, start, osqueue.EnqueueOpts{})
		require.Nil(t, err)

		// First 2 partitions will be custom.
		fnPart, custom1, custom2 := q.ItemPartitions(ctx, q.primaryQueueShard, item)
		require.Equal(t, int(enums.PartitionTypeDefault), fnPart.PartitionType)
		require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom1.PartitionType)
		require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom2.PartitionType)

		// Lease the item.
		id, err := q.Lease(ctx, item, time.Second, q.clock.Now(), nil)
		require.NoError(t, err)
		require.NotNil(t, id)

		score0, err := r.ZMScore(fnPart.concurrencyKey(q.primaryQueueShard.RedisClient.kg), item.ID)
		require.NoError(t, err)
		score1, err := r.ZMScore(custom1.concurrencyKey(q.primaryQueueShard.RedisClient.kg), item.ID)
		require.NoError(t, err)
		require.Equal(t, score0[0], score1[0], "Partition scores should match after leasing")

		t.Run("extending the lease should extend both items in all partition's concurrency queues", func(t *testing.T) {
			id, err = q.ExtendLease(ctx, item, *id, 98712*time.Millisecond)
			require.NoError(t, err)
			require.NotNil(t, id)

			newScore0, err := r.ZMScore(fnPart.concurrencyKey(q.primaryQueueShard.RedisClient.kg), item.ID)
			require.NoError(t, err)
			newScore1, err := r.ZMScore(custom1.concurrencyKey(q.primaryQueueShard.RedisClient.kg), item.ID)
			require.NoError(t, err)

			require.Equal(t, newScore0, newScore1, "Partition scores should match after leasing")
			require.NotEqual(t, int(score0[0]), int(newScore0[0]), "Partition scores should not have been updated: %v", newScore0)
			require.NotEqual(t, score1, newScore1, "Partition scores should have been updated")

			// And, the account-level concurrency queue is updated
			acctScore, err := r.ZMScore(q.primaryQueueShard.RedisClient.kg.Concurrency("account", item.Data.Identifier.AccountID.String()), item.ID)
			require.NoError(t, err)
			require.EqualValues(t, acctScore[0], newScore0[0])
		})

		t.Run("Scavenge queue is updated", func(t *testing.T) {
			mem, err := r.ZMembers(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex())
			require.NoError(t, err)
			require.Equal(t, 1, len(mem), "scavenge queue should have 1 item", mem)
			require.Contains(t, mem, fnPart.ID)
			require.NotContains(t, mem, custom1.concurrencyKey(q.primaryQueueShard.RedisClient.kg))
			require.NotContains(t, mem, custom2.concurrencyKey(q.primaryQueueShard.RedisClient.kg))

			score, err := r.ZMScore(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex(), fnPart.ID)
			require.NoError(t, err)
			require.NotZero(t, score[0])

			id, err = q.ExtendLease(ctx, item, *id, 1238712*time.Millisecond)
			require.NoError(t, err)
			require.NotNil(t, id)

			nextScore, err := r.ZMScore(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex(), fnPart.ID)
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

	queueClient := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
	q := NewQueue(queueClient)
	ctx := context.Background()

	t.Run("It always changes global partition scores", func(t *testing.T) {
		r.FlushAll()

		fnID, acctID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("fn")),
			uuid.NewSHA1(uuid.NameSpaceDNS, []byte("acct"))

		start := time.Now().Truncate(time.Second)

		// Enqueue two items to the same function
		itemA, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
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
		}, start, osqueue.EnqueueOpts{})
		require.Nil(t, err)
		_, err = q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
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
		}, start, osqueue.EnqueueOpts{})
		require.Nil(t, err)

		// First 2 partitions will be custom, third one default
		fnPart, custom1, custom2 := q.ItemPartitions(ctx, q.primaryQueueShard, itemA)
		require.Equal(t, int(enums.PartitionTypeDefault), fnPart.PartitionType)
		require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom1.PartitionType)
		require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom2.PartitionType)

		// Lease the first item, pretending it's in progress.
		_, err = q.Lease(ctx, itemA, 10*time.Second, q.clock.Now(), nil)
		require.NoError(t, err)

		// Note: Originally, this test used the concurrency key queue for testing Dequeue(),
		// but this was changed to the default partition, as we do not enqueue to key queues anymore.
		partitionToDequeue := fnPart

		// Force requeue the partition such that it's pushed forward, pretending there's
		// no capacity.
		err = q.PartitionRequeue(ctx, q.primaryQueueShard, &partitionToDequeue, start.Add(30*time.Minute), true)
		require.NoError(t, err)

		t.Run("Requeueing partitions updates the score", func(t *testing.T) {
			partScoreA, _ := r.ZMScore(q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex(), partitionToDequeue.ID)
			require.EqualValues(t, start.Add(30*time.Minute).Unix(), partScoreA[0])

			partScoreA, _ = r.ZMScore(q.primaryQueueShard.RedisClient.kg.AccountPartitionIndex(acctID), partitionToDequeue.ID)
			require.NotNil(t, partScoreA, "expected partition requeue to update account partition index", r.Dump())
			require.EqualValues(t, start.Add(30*time.Minute).Unix(), partScoreA[0])
		})

		// Dequeue to pull partition back to now
		err = q.Dequeue(ctx, q.primaryQueueShard, itemA)
		require.Nil(t, err)

		t.Run("The outstanding partition scores should reset", func(t *testing.T) {
			partScoreA, _ := r.ZMScore(q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex(), partitionToDequeue.ID)
			require.EqualValues(t, start, time.Unix(int64(partScoreA[0]), 0), r.Dump(), partitionToDequeue, start.UnixMilli())
		})
	})

	t.Run("with concurrency keys", func(t *testing.T) {
		start := time.Now()

		t.Run("with an unleased item", func(t *testing.T) {
			r.FlushAll()
			item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
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
			}, start, osqueue.EnqueueOpts{})
			require.Nil(t, err)

			// First 2 partitions will be custom.
			fnPart, custom1, custom2 := q.ItemPartitions(ctx, q.primaryQueueShard, item)
			require.Equal(t, int(enums.PartitionTypeDefault), fnPart.PartitionType)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom1.PartitionType)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom2.PartitionType)

			err = q.Dequeue(ctx, q.primaryQueueShard, item)
			require.Nil(t, err)

			t.Run("The outstanding partition items should be empty", func(t *testing.T) {
				mem, _ := r.ZMembers(fnPart.zsetKey(q.primaryQueueShard.RedisClient.kg))
				require.Equal(t, 0, len(mem))

				mem, _ = r.ZMembers(custom1.zsetKey(q.primaryQueueShard.RedisClient.kg))
				require.NoError(t, err)
				require.Equal(t, 0, len(mem))
			})
		})

		t.Run("with a leased item", func(t *testing.T) {
			r.FlushAll()
			item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
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
			}, start, osqueue.EnqueueOpts{})
			require.Nil(t, err)

			// First 2 partitions will be custom.
			_, custom1, custom2 := q.ItemPartitions(ctx, q.primaryQueueShard, item)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom1.PartitionType)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom2.PartitionType)

			id, err := q.Lease(ctx, item, 10*time.Second, time.Now(), nil)
			require.NoError(t, err)
			require.NotEmpty(t, id)

			t.Run("The scavenger queue should not yet be empty", func(t *testing.T) {
				mems, err := r.ZMembers(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex())
				require.NoError(t, err)
				require.NotEmpty(t, mems)
			})

			err = q.Dequeue(ctx, q.primaryQueueShard, item)
			require.Nil(t, err)

			t.Run("The outstanding partition items should be empty", func(t *testing.T) {
				mem, _ := r.ZMembers(custom1.zsetKey(q.primaryQueueShard.RedisClient.kg))
				require.Equal(t, 0, len(mem))

				mem, _ = r.ZMembers(custom2.zsetKey(q.primaryQueueShard.RedisClient.kg))
				require.NoError(t, err)
				require.Equal(t, 0, len(mem))
			})

			t.Run("The concurrenty partition items should be empty", func(t *testing.T) {
				mem, _ := r.ZMembers(custom1.concurrencyKey(q.primaryQueueShard.RedisClient.kg))
				require.Equal(t, 0, len(mem))

				mem, _ = r.ZMembers(custom2.concurrencyKey(q.primaryQueueShard.RedisClient.kg))
				require.NoError(t, err)
				require.Equal(t, 0, len(mem))
			})

			t.Run("The scavenger queue should now be empty", func(t *testing.T) {
				mems, _ := r.ZMembers(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex())
				require.Empty(t, mems)
			})
		})
	})

	t.Run("It should remove a queue item", func(t *testing.T) {
		r.FlushAll()

		start := time.Now()

		fnID := uuid.New()
		runID := ulid.MustNew(ulid.Now(), rand.Reader)

		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					RunID:      runID,
					WorkflowID: fnID,
				},
			},
		}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		p := QueuePartition{FunctionID: &item.FunctionID}

		id, err := q.Lease(ctx, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		t.Run("The lease exists in the partition queue", func(t *testing.T) {
			count, err := q.InProgress(ctx, "p", p.FunctionID.String())
			require.NoError(t, err)
			require.EqualValues(t, 1, count, r.Dump())
		})

		err = q.Dequeue(ctx, q.primaryQueueShard, item)
		require.NoError(t, err)

		t.Run("It should remove the item from the queue map", func(t *testing.T) {
			val := r.HGet(q.primaryQueueShard.RedisClient.kg.QueueItem(), id.String())
			require.Empty(t, val)
		})

		t.Run("Extending a lease should fail after dequeue", func(t *testing.T) {
			id, err := q.ExtendLease(ctx, item, *id, time.Minute)
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

		t.Run("run indexes are updated", func(t *testing.T) {
			kg := q.primaryQueueShard.RedisClient.kg
			// Run indexes should be updated

			require.False(t, r.Exists(kg.ActiveSet("run", runID.String())))
			require.False(t, r.Exists(kg.ActiveRunsSet("p", fnID.String())))
		})

		t.Run("It should work if the item is not leased (eg. deletions)", func(t *testing.T) {
			item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			err = q.Dequeue(ctx, q.primaryQueueShard, item)
			require.NoError(t, err)

			val := r.HGet(q.primaryQueueShard.RedisClient.kg.QueueItem(), id.String())
			require.Empty(t, val)
		})

		t.Run("Removes default indexes", func(t *testing.T) {
			at := time.Now().Truncate(time.Second)
			rid := ulid.MustNew(ulid.Now(), rand.Reader)
			item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
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

			err = q.Dequeue(ctx, q.primaryQueueShard, item)
			require.NoError(t, err)

			keys, err = r.ZMembers(fmt.Sprintf("{queue}:idx:run:%s", rid))
			require.NotNil(t, err)
			require.Equal(t, true, strings.Contains(err.Error(), "no such key"))
			require.Equal(t, 0, len(keys))
		})
	})

	t.Run("backcompat: it should not drop previous partition names from concurrency index", func(t *testing.T) {
		// This tests backwards compatibility with the old concurrency index member naming scheme
		r.FlushAll()
		start := time.Now().Truncate(time.Second)

		customQueueName := "custom-queue-name"
		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			FunctionID: uuid.New(),
			Data: osqueue.Item{
				QueueName: &customQueueName,
			},
			QueueName: &customQueueName,
		}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		fnPart := q.ItemPartition(ctx, q.primaryQueueShard, item)

		itemCountMatches := func(num int) {
			zsetKey := fnPart.zsetKey(q.primaryQueueShard.RedisClient.kg)
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
				Key(fnPart.concurrencyKey(q.primaryQueueShard.RedisClient.kg)).
				Min("-inf").
				Max("+inf").
				Build()).AsStrSlice()
			require.NoError(t, err)
			assert.Equal(t, num, len(items), "expected %d items in the concurrency queue", num, r.Dump())
		}

		itemCountMatches(1)
		concurrencyItemCountMatches(0)

		_, err = q.Lease(ctx, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		itemCountMatches(0)
		concurrencyItemCountMatches(1)

		// Ensure the concurrency index is updated.
		mem, err := r.ZMembers(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex())
		require.NoError(t, err)
		assert.Equal(t, 1, len(mem))
		assert.Contains(t, mem[0], fnPart.ID)

		// Dequeue the item.
		err = q.Dequeue(ctx, q.primaryQueueShard, item)
		require.NoError(t, err)

		itemCountMatches(0)
		concurrencyItemCountMatches(0)

		// Ensure the concurrency index is updated.
		numMembers, err := rc.Do(ctx, rc.B().Zcard().Key(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex()).Build()).AsInt64()
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

	q := NewQueue(QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName})
	ctx := context.Background()

	t.Run("Re-enqueuing a leased item should succeed", func(t *testing.T) {
		now := time.Now()

		fnID := uuid.New()
		runID := ulid.MustNew(ulid.Now(), rand.Reader)

		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					RunID:      runID,
					WorkflowID: fnID,
				},
			},
		}, now, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		_, err = q.Lease(ctx, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		// Assert partition index is original
		pi := QueuePartition{FunctionID: &item.FunctionID}
		requirePartitionScoreEquals(t, r, pi.FunctionID, now.Truncate(time.Second))

		requirePartitionInProgress(t, q, item.FunctionID, 1)

		next := now.Add(time.Hour)
		err = q.Requeue(ctx, q.primaryQueueShard, item, next)
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

		t.Run("run indexes are updated on requeue to partition", func(t *testing.T) {
			kg := q.primaryQueueShard.RedisClient.kg

			require.False(t, r.Exists(kg.ActiveRunsSet("p", item.FunctionID.String())))
			require.False(t, r.Exists(kg.ActiveSet("run", runID.String())))
		})

		t.Run("It should not update the partition's earliest time, if later", func(t *testing.T) {
			_, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
				FunctionID: fnID,
				Data: osqueue.Item{
					Identifier: state.Identifier{
						RunID:      runID,
						WorkflowID: fnID,
					},
				},
			}, now, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			requirePartitionScoreEquals(t, r, pi.FunctionID, now)

			next := now.Add(2 * time.Hour)
			err = q.Requeue(ctx, q.primaryQueueShard, item, next)
			require.NoError(t, err)

			requirePartitionScoreEquals(t, r, pi.FunctionID, now)
		})

		t.Run("Updates default indexes", func(t *testing.T) {
			at := time.Now().Truncate(time.Second)
			rid := ulid.MustNew(ulid.Now(), rand.Reader)
			item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
				FunctionID: uuid.New(),
				Data: osqueue.Item{
					Kind: osqueue.KindEdge,
					Identifier: state.Identifier{
						RunID: rid,
					},
				},
			}, at, osqueue.EnqueueOpts{})
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
			err = q.Requeue(ctx, q.primaryQueueShard, item, next)
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
		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, now, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		fnPart, custom1, custom2 := q.ItemPartitions(ctx, q.primaryQueueShard, item)

		// Get all scores
		require.False(t, r.Exists(custom1.zsetKey(q.primaryQueueShard.RedisClient.kg)))
		require.False(t, r.Exists(custom2.zsetKey(q.primaryQueueShard.RedisClient.kg)))
		itemScoreDefault, _ := r.ZMScore(fnPart.zsetKey(q.primaryQueueShard.RedisClient.kg), item.ID)
		partScoreDefault, _ := r.ZMScore(q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex(), fnPart.ID)
		accountPartScore, _ := r.ZMScore(q.primaryQueueShard.RedisClient.kg.AccountPartitionIndex(acctID), fnPart.ID)
		accountScore, _ := r.ZMScore(q.primaryQueueShard.RedisClient.kg.GlobalAccountIndex(), acctID.String())

		require.NotEmpty(t, itemScoreDefault, "Couldn't find item in '%s':\n%s", custom1.zsetKey(q.primaryQueueShard.RedisClient.kg), r.Dump())
		require.NotEmpty(t, partScoreDefault)
		require.Equal(t, partScoreDefault, accountPartScore, "expected account partitions to match global partitions")
		require.Equal(t, accountPartScore[0], accountScore[0], "expected account score to match earliest account partition")

		_, err = q.Lease(ctx, item, time.Second, q.clock.Now(), nil)
		require.NoError(t, err)

		// Requeue
		next := now.Add(time.Hour)
		err = q.Requeue(ctx, q.primaryQueueShard, item, next)
		require.NoError(t, err)

		t.Run("It requeues all partitions", func(t *testing.T) {
			newItemScore, _ := r.ZMScore(fnPart.zsetKey(q.primaryQueueShard.RedisClient.kg), item.ID)
			newPartScore, _ := r.ZMScore(q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex(), fnPart.ID)
			newAccountPartScore, _ := r.ZMScore(q.primaryQueueShard.RedisClient.kg.AccountPartitionIndex(acctID), fnPart.ID)
			newAccountScore, _ := r.ZMScore(q.primaryQueueShard.RedisClient.kg.GlobalAccountIndex(), acctID.String())

			require.NotEqual(t, itemScoreDefault, newItemScore)
			require.NotEqual(t, partScoreDefault, newPartScore)
			require.Equal(t, newPartScore, newAccountPartScore)
			require.Equal(t, newPartScore, newAccountPartScore)
			require.Equal(t, next.Truncate(time.Second).Unix(), int64(newPartScore[0]))
			require.Equal(t, newAccountPartScore[0], newAccountScore[0], "expected account score to match earliest account partition", r.Dump())
			require.EqualValues(t, next.UnixMilli(), int(newItemScore[0]))
			require.EqualValues(t, next.Unix(), int(newPartScore[0]))
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

	q := NewQueue(QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName})
	ctx := context.Background()

	_, err = q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: idA}, atA, osqueue.EnqueueOpts{})
	require.NoError(t, err)
	_, err = q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: idB}, atB, osqueue.EnqueueOpts{})
	require.NoError(t, err)
	_, err = q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: idC}, atC, osqueue.EnqueueOpts{})
	require.NoError(t, err)

	t.Run("Partitions are in order after enqueueing", func(t *testing.T) {
		items, err := q.PartitionPeek(ctx, true, time.Now().Add(time.Hour), PartitionPeekMax)
		require.NoError(t, err)
		require.Len(t, items, 3)
		require.EqualValues(t, []*QueuePartition{
			{ID: idA.String(), FunctionID: &idA, AccountID: uuid.Nil},
			{ID: idB.String(), FunctionID: &idB, AccountID: uuid.Nil},
			{ID: idC.String(), FunctionID: &idC, AccountID: uuid.Nil},
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
				{ID: idB.String(), FunctionID: &idB, AccountID: uuid.Nil},
				{ID: idC.String(), FunctionID: &idC, AccountID: uuid.Nil},
				{
					ID:         idA.String(),
					FunctionID: &idA,
					AccountID:  uuid.Nil,
					Last:       items[2].Last, // Use the leased partition time.
					LeaseID:    leaseID,
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
		q := NewQueue(QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName})
		ctx := context.Background()

		_, err = q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: idA}, atA, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		_, err = q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: idB}, atB, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		_, err = q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: idC}, atC, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		t.Run("Fails to lease a paused partition", func(t *testing.T) {
			// pause fn A's partition:
			err = q.SetFunctionPaused(ctx, uuid.Nil, idA, true)
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
			err = q.SetFunctionPaused(ctx, uuid.Nil, idA, false)
			require.NoError(t, err)

			// attempt to lease the unpaused partition:
			id, capacity, err := q.PartitionLease(ctx, &pA, time.Second*5)
			require.NotNil(t, id)
			require.NoError(t, err)
			require.NotZero(t, capacity)
		})
	})

	t.Run("Partition pausing with key queues", func(t *testing.T) {
		r.FlushAll() // reset everything
		q := NewQueue(
			QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName},
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}))
		ctx := context.Background()

		acctID := uuid.New()
		_, err = q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: idA, Data: osqueue.Item{Identifier: state.Identifier{AccountID: acctID, WorkflowID: idA}}}, atA, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		_, err = q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: idB, Data: osqueue.Item{Identifier: state.Identifier{AccountID: acctID, WorkflowID: idB}}}, atB, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		_, err = q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: idC, Data: osqueue.Item{Identifier: state.Identifier{AccountID: acctID, WorkflowID: idC}}}, atC, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		getShadowPartition := func(fnID uuid.UUID) QueueShadowPartition {
			var sp QueueShadowPartition

			str, err := rc.Do(ctx, rc.B().Hget().Key(q.primaryQueueShard.RedisClient.kg.ShadowPartitionMeta()).Field(fnID.String()).Build()).ToString()
			require.NoError(t, err, r.Dump())

			require.NoError(t, json.Unmarshal([]byte(str), &sp))
			return sp
		}

		t.Run("Fails to lease a paused partition", func(t *testing.T) {
			sp := getShadowPartition(idA)
			require.False(t, sp.PauseRefill)

			// pause fn A's partition:
			err = q.SetFunctionPaused(ctx, uuid.Nil, idA, true)
			require.NoError(t, err)

			sp = getShadowPartition(idA)
			require.True(t, sp.PauseRefill)

			// attempt to lease the paused partition:
			id, capacity, err := q.PartitionLease(ctx, &pA, time.Second*5)
			require.Nil(t, id)
			require.Error(t, err)
			require.Zero(t, capacity)
			require.ErrorIs(t, err, ErrPartitionPaused)
		})

		t.Run("Succeeds to lease a previously paused partition", func(t *testing.T) {
			sp := getShadowPartition(idA)
			require.True(t, sp.PauseRefill)

			// unpause fn A's partition:
			err = q.SetFunctionPaused(ctx, uuid.Nil, idA, false)
			require.NoError(t, err)

			sp = getShadowPartition(idA)
			require.False(t, sp.PauseRefill)

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

		_, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				CustomConcurrencyKeys: []state.CustomConcurrency{ck},
			},
		}, now.Add(10*time.Second), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		defaultPartition := getDefaultPartition(t, r, fnID)

		leaseUntil := now.Add(3 * time.Second)
		leaseID, capacity, err := q.PartitionLease(ctx, &defaultPartition, time.Until(leaseUntil))
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		require.NotZero(t, capacity)
	})

	t.Run("concurrency is checked early", func(t *testing.T) {
		start := time.Now().Truncate(time.Second)

		t.Run("With partition concurrency limits", func(t *testing.T) {
			r.FlushAll()

			// Only allow a single leased item
			q.partitionConstraintConfigGetter = func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return PartitionConstraintConfig{
					Concurrency: PartitionConcurrency{
						AccountConcurrency:  1,
						FunctionConcurrency: 1,
					},
				}
			}

			fnID := uuid.New()
			// Create a new item
			itemA, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: fnID}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			_, err = q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: fnID}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			// Use the new item's workflow ID
			p := QueuePartition{ID: itemA.FunctionID.String(), FunctionID: &itemA.FunctionID}

			t.Run("Leases with capacity", func(t *testing.T) {
				_, err = q.Lease(ctx, itemA, 5*time.Second, time.Now(), nil)
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
			q.partitionConstraintConfigGetter = func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return PartitionConstraintConfig{
					Concurrency: PartitionConcurrency{
						AccountConcurrency:  1,
						FunctionConcurrency: 100,
					},
				}
			}

			acctId := uuid.New()

			// Create a new item
			itemA, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: uuid.New(), Data: osqueue.Item{Identifier: state.Identifier{AccountID: acctId}}}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			_, err = q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: uuid.New(), Data: osqueue.Item{Identifier: state.Identifier{AccountID: acctId}}}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			// Use the new item's workflow ID
			p := QueuePartition{AccountID: acctId, FunctionID: &itemA.FunctionID}

			t.Run("Leases with capacity", func(t *testing.T) {
				_, err = q.Lease(ctx, itemA, 5*time.Second, time.Now(), nil)
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
			ckHash := util.XXHash("key-expr")
			q.partitionConstraintConfigGetter = func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return PartitionConstraintConfig{
					Concurrency: PartitionConcurrency{
						AccountConcurrency:  100,
						FunctionConcurrency: 100,
						CustomConcurrencyKeys: []CustomConcurrencyLimit{
							{
								Scope:               enums.ConcurrencyScopeAccount,
								HashedKeyExpression: ckHash,
								Limit:               1,
							},
						},
					},
				}
			}

			accountId := uuid.New()
			ck := createConcurrencyKey(enums.ConcurrencyScopeAccount, accountId, "foo", 1)

			// Create a new item
			itemA, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
				FunctionID: uuid.New(),
				Data: osqueue.Item{
					Identifier: state.Identifier{AccountID: accountId},
					CustomConcurrencyKeys: []state.CustomConcurrency{
						{
							Key:   ck.Key,
							Hash:  ckHash,
							Limit: 1,
						},
					},
				},
			}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			_, err = q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
				FunctionID: uuid.New(),
				Data: osqueue.Item{
					Identifier: state.Identifier{AccountID: accountId},
					CustomConcurrencyKeys: []state.CustomConcurrency{
						{
							Key:   ck.Key,
							Limit: 1,
						},
					},
				},
			}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			t.Run("Leases with capacity", func(t *testing.T) {
				_, err = q.Lease(ctx, itemA, 5*time.Second, time.Now(), nil)
				require.NoError(t, err)
			})

			t.Run("Partition lease on default fn does not error without capacity", func(t *testing.T) {
				p := QueuePartition{FunctionID: &itemA.FunctionID, AccountID: accountId}

				// Since we don't peek and lease concurrency key queue partitions anymore,
				// we won't check for custom concurrency limits ahead of processing items.
				// Leasing a default partition works even though the concurrency key has no additional capacity.
				leaseId, _, err := q.PartitionLease(ctx, &p, 5*time.Second)
				require.NotNil(t, leaseId, "Expected lease id.\n%s", r.Dump())
				require.NoError(t, err)
			})
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

		q := NewQueue(
			QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey)},
			WithPartitionPriorityFinder(func(_ context.Context, _ QueuePartition) uint {
				return PriorityDefault
			}),
		)
		now := time.Now()
		enqueue(q, now)
		requirePartitionScoreEquals(t, r, &idA, now)

		// Pause A, excluding it from peek:
		err = q.SetFunctionPaused(ctx, uuid.Nil, idA, true)
		require.NoError(t, err)

		// This should only select B and C, as id A is ignored:
		items, err := q.PartitionPeek(ctx, true, now.Add(time.Hour), PartitionPeekMax)
		require.NoError(t, err)
		require.Len(t, items, 2)
		require.EqualValues(t, []*QueuePartition{
			{ID: idB.String(), FunctionID: &idB, AccountID: accountId},
			{ID: idC.String(), FunctionID: &idC, AccountID: accountId},
		}, items)
		requirePartitionScoreEquals(t, r, &idA, now.Add(24*time.Hour))

		// After unpausing A, it should be included in the peek:
		err = q.SetFunctionPaused(ctx, uuid.Nil, idA, false)
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

		t.Run("Requeueing a paused partition does not affect the partition's pause state", func(t *testing.T) {
			_, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{FunctionID: idA}, now, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			_, _, err = q.PartitionLease(ctx, &QueuePartition{FunctionID: &idA}, time.Minute)
			require.NoError(t, err)

			err = q.SetFunctionPaused(ctx, uuid.Nil, idA, true)
			require.NoError(t, err)

			err = q.PartitionRequeue(ctx, q.primaryQueueShard, &p, next, true)
			require.NoError(t, err)

			fnMeta, err := getFnMetadata(t, r, idA)
			require.NoError(t, err)
			require.True(t, fnMeta.Paused)
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

		require.True(t, r.Exists(shard.RedisClient.kg.FnMetadata(*p.FunctionID)), r.Keys())

		//
		// PartitionRequeue should drop pointers but not partition metadata
		//

		err = q.PartitionRequeue(ctx, q.primaryQueueShard, &p, now.Add(time.Minute), false)
		require.Equal(t, ErrPartitionGarbageCollected, err)

		require.Equal(t, 0, zcard(t, rc, fnReadyQueue))
		require.False(t, r.Exists(shard.RedisClient.kg.GlobalPartitionIndex()))
		require.False(t, r.Exists(shard.RedisClient.kg.AccountPartitionIndex(accountID)))
		require.False(t, r.Exists(shard.RedisClient.kg.GlobalAccountIndex()))

		// ensure gc does not drop fn metadata
		require.True(t, r.Exists(shard.RedisClient.kg.FnMetadata(*p.FunctionID)), r.Keys())
		require.True(t, r.Exists(shard.RedisClient.kg.PartitionItem()))
		keys, err := r.HKeys(shard.RedisClient.kg.PartitionItem())
		require.NoError(t, err)
		require.Contains(t, keys, p.FunctionID.String())

		//
		// Drop backlog and have PartitionRequeue clean up remaining data
		//

		// drop backlog
		res, err := q.BacklogRefill(ctx, &backlog, &shadowPart, time.Now().Add(time.Minute), PartitionConstraintConfig{})
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

func TestQueueScavenge(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	q := NewQueue(
		QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName},
	)
	ctx := context.Background()

	id := uuid.New()

	qi := osqueue.QueueItem{
		FunctionID: id,
		Data: osqueue.Item{
			Payload: json.RawMessage("{\"test\":\"payload\"}"),
		},
	}

	t.Run("scavenging removes leftover traces of key queues", func(t *testing.T) {
		r.FlushAll()

		start := time.Now().Truncate(time.Second)

		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		require.NotEqual(t, item.ID, ulid.Zero)
		require.Equal(t, time.UnixMilli(item.WallTimeMS).Truncate(time.Second), start)

		qp := getDefaultPartition(t, r, id)

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

		incompatibleConcurrencyIndexItem := q.primaryQueueShard.RedisClient.kg.Concurrency("p", id.String())
		compatibleConcurrencyIndexItem := id.String()

		indexMembers, err := r.ZMembers(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex())
		require.NoError(t, err)
		require.Equal(t, 1, len(indexMembers))
		require.Contains(t, indexMembers, compatibleConcurrencyIndexItem)

		leftoverData := []string{
			q.primaryQueueShard.RedisClient.kg.Concurrency("p", id.String()),
			"{queue}:concurrency:p:0ffd4629-317c-4f65-8b8f-b30fccfde46f",
			"{queue}:concurrency:custom:f:0ffd4629-317c-4f65-8b8f-b30fccfde46f:1nt4mu0skse4a",
		}
		score := float64(leaseStart.Add(time.Second).UnixMilli())
		for _, leftover := range leftoverData {
			_, err = r.ZAdd(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex(), score, leftover)
			require.NoError(t, err)
		}
		indexMembers, err = r.ZMembers(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex())
		require.NoError(t, err)
		require.Equal(t, 4, len(indexMembers))
		for _, datum := range leftoverData {
			require.Contains(t, indexMembers, datum)
		}

		requeued, err := q.Scavenge(ctx, ScavengePeekSize)
		require.NoError(t, err)
		assert.Equal(t, 1, requeued, "expected one item with expired leases to be requeued by scavenge", r.Dump())

		itemCountMatches(1)
		concurrencyItemCountMatches(0)

		_, err = r.ZMembers(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex())
		require.Error(t, err, r.Dump())
		require.ErrorIs(t, err, miniredis.ErrKeyNotFound)

		newConcurrencyQueueItems, err := rc.Do(ctx, rc.B().Zcard().Key(incompatibleConcurrencyIndexItem).Build()).AsInt64()
		require.NoError(t, err)
		assert.Equal(t, 0, int(newConcurrencyQueueItems), "expected no items in the new concurrency queue", r.Dump())

		oldConcurrencyQueueItems, err := rc.Do(ctx, rc.B().Zcard().Key(compatibleConcurrencyIndexItem).Build()).AsInt64()
		require.NoError(t, err)
		assert.Equal(t, 0, int(oldConcurrencyQueueItems), "expected no items in the old concurrency queue", r.Dump())
	})
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
		q := NewQueue(
			QueueShard{Name: "default", Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey)},
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
		require.False(t, sp.PauseRefill)

		err = q.SetFunctionMigrate(ctx, "default", fnID, true)
		require.NoError(t, err)

		meta, err := getFnMetadata(t, r, fnID)
		require.NoError(t, err)
		require.True(t, meta.Migrate)

		sp = getShadowPartition()
		require.Equal(t, fnID.String(), sp.PartitionID)
		require.True(t, sp.PauseRefill)

		// disable migration flag
		err = q.SetFunctionMigrate(ctx, "default", fnID, false)
		require.NoError(t, err)

		meta, err = getFnMetadata(t, r, fnID)
		require.NoError(t, err)
		require.False(t, meta.Migrate)

		sp = getShadowPartition()
		require.Equal(t, fnID.String(), sp.PartitionID)
		require.False(t, sp.PauseRefill)
	})

	t.Run("with other shards", func(t *testing.T) {
		other := miniredis.RunT(t)
		rc2, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{other.Addr()}, DisableCache: true})
		require.NoError(t, err)
		defer rc2.Close()

		yoloShard := QueueShard{Name: "yolo", Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc2, QueueDefaultKey)}

		q := NewQueue(
			QueueShard{Name: "default", Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey)},
			WithQueueShardClients(map[string]QueueShard{
				"yolo": yoloShard,
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

		err = q.SetFunctionMigrate(ctx, "yolo", fnID, true)
		require.NoError(t, err)

		// should not find it in the default shard
		_, err = getFnMetadata(t, r, fnID)
		require.Error(t, err)
		require.ErrorContains(t, err, "no such key")

		// should find metadata in the other shard
		meta, err := getFnMetadata(t, other, fnID)
		require.NoError(t, err)
		require.True(t, meta.Migrate)
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
	q := NewQueue(QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey)}, WithClock(clock))

	idA, idB := uuid.New(), uuid.New()

	r := require.New(t)

	t.Run("Without bursts", func(t *testing.T) {
		throttle := &osqueue.Throttle{
			Key:    "some-key",
			Limit:  1,
			Period: 5, // Admit one every 5 seconds
			Burst:  0, // No burst.
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
				items, err := q.ItemsByPartition(ctx, shard, partitionID, from, until)
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
			err = q1.SetFunctionMigrate(ctx, shard1Name, fnID, true)
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

func getFnMetadata(t *testing.T, r *miniredis.Miniredis, id uuid.UUID) (*FnMetadata, error) {
	t.Helper()
	kg := &queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
	valJSON, err := r.Get(kg.FnMetadata(id))
	if err != nil {
		return nil, err
	}

	retv := FnMetadata{}
	err = json.Unmarshal([]byte(valJSON), &retv)
	require.NoError(t, err)
	return &retv, nil
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
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return false
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
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return false
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
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return false
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
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return false
			}),
			WithEnqueueSystemPartitionsToBacklog(true),
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

func TestQueueLeaseWithoutValidation(t *testing.T) {
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

		enqueueToBacklog := false
		q := NewQueue(
			defaultShard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		)
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute).Truncate(time.Minute)

		t.Run("should lease item", func(t *testing.T) {
			require.Len(t, r.Keys(), 0)

			item1 := osqueue.QueueItem{
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
				QueueName:    nil,
				RefilledFrom: "fake-backlog",
				RefilledAt:   at.UnixMilli(),
			}

			fnPart := q.ItemPartition(ctx, defaultShard, item1)
			require.Equal(t, int(enums.PartitionTypeDefault), fnPart.PartitionType)

			// for simplicity, this enqueue should go directly to the partition
			enqueueToBacklog = false
			qi, err := q.EnqueueItem(ctx, defaultShard, item1, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			now := q.clock.Now()
			leaseDur := 5 * time.Second
			leaseExpiry := now.Add(leaseDur)

			// simulate having hit a partition concurrency limit in a previous operation,
			// without disabling validation this should cause Lease() to fail
			denies := newLeaseDenyList()
			denies.addConcurrency(newKeyError(ErrPartitionConcurrencyLimit, fnPart.Queue()))

			leaseID, err := q.Lease(ctx, qi, leaseDur, now, denies)
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			backlog := q.ItemBacklog(ctx, item1)
			require.NotEmpty(t, backlog.BacklogID)

			shadowPartition := q.ItemShadowPartition(ctx, item1)
			require.NotEmpty(t, shadowPartition.PartitionID)

			// key queue v2 accounting
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, shadowPartition.accountInProgressKey(kg), qi.ID)))
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, shadowPartition.inProgressKey(kg), qi.ID)))
			require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 1))
			require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 2))
			require.False(t, r.Exists(backlog.customKeyInProgress(kg, 1)))

			// expect classic partition concurrency to include item
			// TODO Do we actually want to update previous accounting?
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.Concurrency("account", accountId.String()), qi.ID)))
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), qi.ID)))
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, fnPart.concurrencyKey(kg), qi.ID)))
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

		enqueueToBacklog := false
		q := NewQueue(
			defaultShard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
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
		at := now.Add(10 * time.Minute).Truncate(time.Minute)

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
				QueueName:    nil,
				RefilledFrom: "fake-backlog",
				RefilledAt:   at.UnixMilli(),
			}

			fnPart, custom1, custom2 := q.ItemPartitions(ctx, defaultShard, item)
			require.NotEmpty(t, fnPart.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), fnPart.PartitionType)
			require.NotEmpty(t, custom1.ID)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom1.PartitionType)
			require.Empty(t, custom2.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), custom2.PartitionType)

			// for simplicity, this enqueue should go directly to the partition
			enqueueToBacklog = false
			qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			now := q.clock.Now()
			leaseDur := 5 * time.Second
			leaseExpiry := now.Add(leaseDur)

			// simulate having hit a partition concurrency limit in a previous operation,
			// without disabling validation this should cause Lease() to fail
			denies := newLeaseDenyList()
			denies.addConcurrency(newKeyError(ErrPartitionConcurrencyLimit, fnPart.Queue()))

			leaseID, err := q.Lease(ctx, qi, leaseDur, now, denies)
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			backlog := q.ItemBacklog(ctx, item)
			require.NotEmpty(t, backlog.BacklogID)

			shadowPartition := q.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			constraints := q.partitionConstraintConfigGetter(ctx, shadowPartition.Identifier())
			require.Len(t, constraints.Concurrency.CustomConcurrencyKeys, 1)

			// key queue v2 accounting
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, shadowPartition.accountInProgressKey(kg), qi.ID)))
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, shadowPartition.inProgressKey(kg), qi.ID)))
			require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope, fnID, unhashedValue)), backlog.customKeyInProgress(kg, 1))
			require.True(t, r.Exists(backlog.customKeyInProgress(kg, 1)))
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, backlog.customKeyInProgress(kg, 1), qi.ID)))
			require.Equal(t, backlog.customKeyInProgress(kg, 1), custom1.concurrencyKey(kg))

			// expect classic partition concurrency to include item
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.Concurrency("account", accountId.String()), qi.ID)))
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), qi.ID)))
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, custom1.concurrencyKey(kg), qi.ID)))
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

		enqueueToBacklog := false
		q := NewQueue(
			defaultShard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
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
			fullKey2 := util.ConcurrencyKey(scope2, wsID, unhashedValue2)

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

			fnPart, custom1, custom2 := q.ItemPartitions(ctx, defaultShard, item)
			require.NotEmpty(t, fnPart.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), fnPart.PartitionType)
			require.NotEmpty(t, custom1.ID)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom1.PartitionType)
			require.NotEmpty(t, custom2.ID)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom2.PartitionType)

			// for simplicity, this enqueue should go directly to the partition
			enqueueToBacklog = false
			qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			now := q.clock.Now()
			leaseDur := 5 * time.Second
			leaseExpiry := now.Add(leaseDur)

			// simulate having hit a partition concurrency limit in a previous operation,
			// without disabling validation this should cause Lease() to fail
			denies := newLeaseDenyList()
			denies.addConcurrency(newKeyError(ErrPartitionConcurrencyLimit, custom2.Queue()))

			leaseID, err := q.Lease(ctx, qi, leaseDur, now, denies)
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			backlog := q.ItemBacklog(ctx, item)
			require.Len(t, backlog.ConcurrencyKeys, 2)

			shadowPartition := q.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			constraints := q.partitionConstraintConfigGetter(ctx, shadowPartition.Identifier())
			require.Len(t, constraints.Concurrency.CustomConcurrencyKeys, 2)

			// key queue v2 accounting
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, shadowPartition.accountInProgressKey(kg), qi.ID)))
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, shadowPartition.inProgressKey(kg), qi.ID)))

			// first key
			require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope1, fnID, unhashedValue1)), backlog.customKeyInProgress(kg, 1))
			require.True(t, r.Exists(backlog.customKeyInProgress(kg, 1)))
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, backlog.customKeyInProgress(kg, 1), qi.ID)))

			// second key
			require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope2, wsID, unhashedValue2)), backlog.customKeyInProgress(kg, 2))
			require.True(t, r.Exists(backlog.customKeyInProgress(kg, 2)))
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, backlog.customKeyInProgress(kg, 2), qi.ID)))

			// expect classic partition concurrency to include item
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.Concurrency("account", accountId.String()), qi.ID)))
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), qi.ID)))
			// first key
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, custom1.concurrencyKey(kg), qi.ID)))
			// second key
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, custom2.concurrencyKey(kg), qi.ID)))
		})
	})

	t.Run("system queues", func(t *testing.T) {
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

		enqueueToBacklog := false
		q := NewQueue(
			defaultShard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		)
		ctx := context.Background()

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute).Truncate(time.Minute)

		t.Run("should lease item", func(t *testing.T) {
			require.Len(t, r.Keys(), 0)

			sysQueueName := osqueue.KindQueueMigrate

			item1 := osqueue.QueueItem{
				ID: "test",
				Data: osqueue.Item{
					Kind:                  osqueue.KindEdge,
					Identifier:            state.Identifier{},
					QueueName:             &sysQueueName,
					Throttle:              nil,
					CustomConcurrencyKeys: nil,
				},
				QueueName: &sysQueueName,
			}

			fnPart := q.ItemPartition(ctx, defaultShard, item1)
			require.Equal(t, int(enums.PartitionTypeDefault), fnPart.PartitionType)
			require.True(t, fnPart.IsSystem())

			// for simplicity, this enqueue should go directly to the partition
			enqueueToBacklog = false
			qi, err := q.EnqueueItem(ctx, defaultShard, item1, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			now := q.clock.Now()
			leaseDur := 5 * time.Second
			leaseExpiry := now.Add(leaseDur)

			// simulate having hit a partition concurrency limit in a previous operation,
			// without disabling validation this should cause Lease() to fail
			denies := newLeaseDenyList()
			denies.addConcurrency(newKeyError(ErrPartitionConcurrencyLimit, fnPart.Queue()))

			leaseID, err := q.Lease(ctx, qi, leaseDur, now, denies)
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			backlog := q.ItemBacklog(ctx, item1)
			require.NotEmpty(t, backlog.BacklogID)

			shadowPartition := q.ItemShadowPartition(ctx, item1)
			require.NotEmpty(t, shadowPartition.PartitionID)

			// key queue v2 accounting
			// should not track account concurrency for system partition
			require.False(t, r.Exists(shadowPartition.accountInProgressKey(kg)))
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, shadowPartition.inProgressKey(kg), qi.ID)))
			require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 1))
			require.False(t, r.Exists(backlog.customKeyInProgress(kg, 1)))

			// expect classic partition concurrency to include item
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.Concurrency("p", sysQueueName), qi.ID)))
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, fnPart.concurrencyKey(kg), qi.ID)))
		})
	})
}

func TestQueueRequeueToBacklog(t *testing.T) {
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

		enqueueToBacklog := false
		q := NewQueue(
			defaultShard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		)
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		runID := ulid.MustNew(ulid.Now(), rand.Reader)

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute)

		t.Run("should requeue item to backlog", func(t *testing.T) {
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
						RunID:       runID,
					},
					QueueName:             nil,
					Throttle:              nil,
					CustomConcurrencyKeys: nil,
				},
				QueueName: nil,
			}

			// directly enqueue to partition
			enqueueToBacklog = false
			qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			// put item in progress, this is tested separately
			now := q.clock.Now()
			leaseDur := 5 * time.Second
			leaseExpires := now.Add(leaseDur)
			leaseID, err := q.Lease(ctx, qi, leaseDur, now, nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			backlog := q.ItemBacklog(ctx, item)
			require.NotEmpty(t, backlog.BacklogID)

			shadowPartition := q.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			constraints := q.partitionConstraintConfigGetter(ctx, shadowPartition.Identifier())
			require.Len(t, constraints.Concurrency.CustomConcurrencyKeys, 0)

			fnPart, custom1, custom2 := q.ItemPartitions(ctx, defaultShard, item)
			require.NotEmpty(t, fnPart.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), fnPart.PartitionType)
			require.Empty(t, custom1.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), custom1.PartitionType)
			require.Empty(t, custom2.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), custom2.PartitionType)

			require.False(t, hasMember(t, r, fnPart.zsetKey(kg), qi.ID))

			// expect key queue accounting to contain item in in-progress
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartition.inProgressKey(kg), qi.ID)))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartition.accountInProgressKey(kg), qi.ID)))

			// no active set for default partition since this uses the in progress key
			require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 1))
			require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 2))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.True(t, hasMember(t, r, fnPart.concurrencyKey(kg), qi.ID))
			require.True(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.True(t, hasMember(t, r, kg.Concurrency("p", fnPart.Queue()), qi.ID))

			itemIsMember, err := r.SIsMember(kg.ActiveSet("run", runID.String()), qi.ID)
			require.NoError(t, err)
			require.True(t, itemIsMember)

			isMember, err := r.SIsMember(kg.ActiveRunsSet("p", fnID.String()), runID.String())
			require.NoError(t, err)
			require.True(t, isMember)

			requeueFor := at.Add(30 * time.Minute).Truncate(time.Minute)

			require.False(t, r.Exists(kg.GlobalAccountShadowPartitions()))
			require.False(t, r.Exists(kg.AccountShadowPartitions(accountId)))

			err = q.Requeue(ctx, defaultShard, qi, requeueFor)
			require.NoError(t, err)

			// expect item to be requeued to backlog
			require.Equal(t, requeueFor.UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID)))
			require.True(t, r.Exists(kg.GlobalAccountShadowPartitions()))
			require.True(t, r.Exists(kg.AccountShadowPartitions(accountId)))

			require.Equal(t, requeueFor.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountId.String())))
			require.Equal(t, requeueFor.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountId), shadowPartition.PartitionID)))

			// expect key queue accounting to be updated
			remainingMembers, _ := r.ZMembers(shadowPartition.inProgressKey(kg))

			require.False(t, hasMember(t, r, shadowPartition.inProgressKey(kg), qi.ID), remainingMembers)
			require.False(t, hasMember(t, r, shadowPartition.accountInProgressKey(kg), qi.ID))

			require.False(t, r.Exists(kg.ActiveSet("run", runID.String())))
			require.False(t, r.Exists(kg.ActiveRunsSet("p", fnID.String())))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.False(t, hasMember(t, r, fnPart.concurrencyKey(kg), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("p", fnPart.Queue()), qi.ID))

			// item must not be in classic backlog
			require.False(t, hasMember(t, r, fnPart.zsetKey(kg), qi.ID), r.Keys())
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

		enqueueToBacklog := false
		q := NewQueue(
			defaultShard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		)
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute)

		t.Run("should requeue item to backlog", func(t *testing.T) {
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

			// directly enqueue to partition
			enqueueToBacklog = false
			qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			// sanity check: empty key should never be stored
			require.False(t, r.Exists(kg.Concurrency("", "")))

			// put item in progress, this is tested separately
			now := q.clock.Now().Truncate(time.Minute)
			leaseDur := 5 * time.Second
			leaseExpires := now.Add(leaseDur)
			leaseID, err := q.Lease(ctx, qi, leaseDur, now, nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)
			require.Equal(t, leaseExpires, ulid.Time(leaseID.Time()), now)

			backlog := q.ItemBacklog(ctx, item)
			require.NotEmpty(t, backlog.BacklogID)
			require.Equal(t, enums.ConcurrencyScopeFn, backlog.ConcurrencyKeys[0].Scope)
			require.NotEmpty(t, backlog.ConcurrencyKeys[0].HashedKeyExpression)
			require.NotEmpty(t, backlog.ConcurrencyKeys[0].EntityID)

			shadowPartition := q.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			constraints := q.partitionConstraintConfigGetter(ctx, shadowPartition.Identifier())
			require.Len(t, constraints.Concurrency.CustomConcurrencyKeys, 1)

			fnPart, custom1, custom2 := q.ItemPartitions(ctx, defaultShard, item)
			require.NotEmpty(t, custom1.ID)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom1.PartitionType)
			require.NotEmpty(t, fnPart.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), fnPart.PartitionType)
			require.Empty(t, custom2.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), custom2.PartitionType)

			// expect key queue accounting to contain item in in-progress
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartition.inProgressKey(kg), qi.ID)))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartition.accountInProgressKey(kg), qi.ID)))

			// 1 active set for custom concurrency key
			require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope, fnID, unhashedValue)), backlog.customKeyInProgress(kg, 1))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, backlog.customKeyInProgress(kg, 1), qi.ID)))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, custom1.concurrencyKey(kg), qi.ID)))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, kg.Concurrency("account", accountId.String()), qi.ID)))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), qi.ID)), r.Keys())
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, kg.Concurrency("custom", fullKey), qi.ID)))

			// sanity check: empty key should never be stored
			require.False(t, r.Exists(kg.Concurrency("", "")))

			requeueFor := at.Add(30 * time.Minute).Truncate(time.Minute)

			err = q.Requeue(ctx, defaultShard, qi, requeueFor)
			require.NoError(t, err)

			// sanity check: empty key should never be stored
			require.False(t, r.Exists(kg.Concurrency("", "")))

			// expect item to be requeued to backlog
			require.Equal(t, requeueFor.UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID)))

			// expect key queue accounting to be updated
			remainingMembers, _ := r.ZMembers(shadowPartition.inProgressKey(kg))

			require.False(t, hasMember(t, r, shadowPartition.inProgressKey(kg), qi.ID), remainingMembers)
			require.False(t, hasMember(t, r, shadowPartition.accountInProgressKey(kg), qi.ID))
			require.False(t, hasMember(t, r, backlog.customKeyInProgress(kg, 1), qi.ID))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.False(t, hasMember(t, r, custom1.concurrencyKey(kg), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("p", fnPart.Queue()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("custom", fullKey), qi.ID))

			// item must not be in classic backlog
			require.False(t, hasMember(t, r, fnPart.zsetKey(kg), qi.ID))
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

		enqueueToBacklog := false
		q := NewQueue(
			defaultShard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		)
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute)

		t.Run("should requeue item to backlog", func(t *testing.T) {
			require.Len(t, r.Keys(), 0)

			hashedConcurrencyKeyExpr1 := hashConcurrencyKey("event.data.userId")
			unhashedValue1 := "user1"
			scope1 := enums.ConcurrencyScopeFn
			fullKey1 := util.ConcurrencyKey(scope1, fnID, unhashedValue1)

			hashedConcurrencyKeyExpr2 := hashConcurrencyKey("event.data.orgId")
			unhashedValue2 := "org1"
			scope2 := enums.ConcurrencyScopeEnv
			fullKey2 := util.ConcurrencyKey(scope2, wsID, unhashedValue2)

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

			// directly enqueue to partition
			enqueueToBacklog = false
			qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			// sanity check: empty key should never be stored
			require.False(t, r.Exists(kg.Concurrency("", "")))

			// put item in progress, this is tested separately
			now := q.clock.Now().Truncate(time.Minute)
			leaseDur := 5 * time.Second
			leaseExpires := now.Add(leaseDur)
			leaseID, err := q.Lease(ctx, qi, leaseDur, now, nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)
			require.Equal(t, leaseExpires, ulid.Time(leaseID.Time()), now)

			backlog := q.ItemBacklog(ctx, item)
			require.Len(t, backlog.ConcurrencyKeys, 2)
			require.NotEmpty(t, backlog.ConcurrencyKeys[0].HashedKeyExpression)
			require.Equal(t, enums.ConcurrencyScopeFn, backlog.ConcurrencyKeys[0].Scope)
			require.NotEmpty(t, backlog.ConcurrencyKeys[0].EntityID)

			require.NotEmpty(t, backlog.ConcurrencyKeys[1].HashedKeyExpression)
			require.NotEmpty(t, backlog.ConcurrencyKeys[1].Scope)
			require.NotEmpty(t, backlog.ConcurrencyKeys[1].EntityID)

			shadowPartition := q.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			constraints := q.partitionConstraintConfigGetter(ctx, shadowPartition.Identifier())
			require.Len(t, constraints.Concurrency.CustomConcurrencyKeys, 2)

			fnPart, custom1, custom2 := q.ItemPartitions(ctx, defaultShard, item)
			require.NotEmpty(t, custom1.ID)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom1.PartitionType)
			require.NotEmpty(t, custom2.ID)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom2.PartitionType)
			require.NotEmpty(t, fnPart.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), fnPart.PartitionType)

			// expect key queue accounting to contain item in in-progress
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartition.inProgressKey(kg), qi.ID)))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartition.accountInProgressKey(kg), qi.ID)))

			// 2 active set for custom concurrency keys

			// first key
			require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope1, fnID, unhashedValue1)), backlog.customKeyInProgress(kg, 1))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, backlog.customKeyInProgress(kg, 1), qi.ID)))

			require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope2, wsID, unhashedValue2)), backlog.customKeyInProgress(kg, 2))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, backlog.customKeyInProgress(kg, 2), qi.ID)))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, custom1.concurrencyKey(kg), qi.ID)))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, kg.Concurrency("account", accountId.String()), qi.ID)))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), qi.ID)), r.Keys())

			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, kg.Concurrency("custom", fullKey1), qi.ID)))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, kg.Concurrency("custom", fullKey2), qi.ID)))

			// sanity check: empty key should never be stored
			require.False(t, r.Exists(kg.Concurrency("", "")))

			requeueFor := at.Add(30 * time.Minute).Truncate(time.Minute)

			err = q.Requeue(ctx, defaultShard, qi, requeueFor)
			require.NoError(t, err)

			// sanity check: empty key should never be stored
			require.False(t, r.Exists(kg.Concurrency("", "")))

			// expect item to be requeued to backlog
			require.Equal(t, requeueFor.UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID)))

			// expect key queue accounting to be updated
			remainingMembers, _ := r.ZMembers(shadowPartition.inProgressKey(kg))

			require.False(t, hasMember(t, r, shadowPartition.inProgressKey(kg), qi.ID), remainingMembers)
			require.False(t, hasMember(t, r, shadowPartition.accountInProgressKey(kg), qi.ID))

			require.False(t, hasMember(t, r, backlog.customKeyInProgress(kg, 1), qi.ID))
			require.False(t, hasMember(t, r, backlog.customKeyInProgress(kg, 2), qi.ID))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.False(t, hasMember(t, r, custom1.concurrencyKey(kg), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("p", fnPart.Queue()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("custom", fullKey1), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("custom", fullKey2), qi.ID))

			// item must not be in classic backlog
			require.False(t, hasMember(t, r, fnPart.zsetKey(kg), qi.ID))
		})
	})

	t.Run("system queues", func(t *testing.T) {
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

		enqueueToBacklog := false
		q := NewQueue(
			defaultShard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
			WithEnqueueSystemPartitionsToBacklog(true),
		)
		ctx := context.Background()

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute)

		sysQueueName := osqueue.KindQueueMigrate

		t.Run("should requeue item to backlog", func(t *testing.T) {
			require.Len(t, r.Keys(), 0)

			item := osqueue.QueueItem{
				ID: "test",
				Data: osqueue.Item{
					Kind:                  osqueue.KindEdge,
					Identifier:            state.Identifier{},
					QueueName:             &sysQueueName,
					Throttle:              nil,
					CustomConcurrencyKeys: nil,
				},
				QueueName: &sysQueueName,
			}

			// directly enqueue to partition
			enqueueToBacklog = false
			qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			// put item in progress, this is tested separately
			now := q.clock.Now()
			leaseDur := 5 * time.Second
			leaseExpires := now.Add(leaseDur)
			leaseID, err := q.Lease(ctx, qi, leaseDur, now, nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			backlog := q.ItemBacklog(ctx, item)
			require.NotEmpty(t, backlog.BacklogID)

			shadowPartition := q.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			constraints := q.partitionConstraintConfigGetter(ctx, shadowPartition.Identifier())
			require.Len(t, constraints.Concurrency.CustomConcurrencyKeys, 0)

			fnPart, custom1, custom2 := q.ItemPartitions(ctx, defaultShard, item)
			require.NotEmpty(t, fnPart.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), fnPart.PartitionType)
			require.True(t, fnPart.IsSystem())
			require.Empty(t, custom1.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), custom1.PartitionType)
			require.Empty(t, custom2.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), custom2.PartitionType)

			require.False(t, hasMember(t, r, fnPart.zsetKey(kg), qi.ID))

			// expect key queue accounting to contain item in in-progress
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartition.inProgressKey(kg), qi.ID)))

			require.Equal(t, kg.Concurrency("account", ""), shadowPartition.accountInProgressKey(kg))
			require.False(t, r.Exists(shadowPartition.accountInProgressKey(kg)))

			// no active set for default partition since this uses the in progress key
			require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 1))
			require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 2))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.True(t, hasMember(t, r, fnPart.concurrencyKey(kg), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("account", fnPart.Queue()), qi.ID)) // pseudo-limit for system qeueus
			require.True(t, hasMember(t, r, kg.Concurrency("p", fnPart.Queue()), qi.ID))

			requeueFor := at.Add(30 * time.Minute).Truncate(time.Minute)

			err = q.Requeue(ctx, defaultShard, qi, requeueFor)
			require.NoError(t, err)

			// expect item to be requeued to backlog
			require.Equal(t, requeueFor.UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID)))

			// expect key queue accounting to be updated
			remainingMembers, _ := r.ZMembers(shadowPartition.inProgressKey(kg))

			require.False(t, hasMember(t, r, shadowPartition.inProgressKey(kg), qi.ID), remainingMembers)
			require.False(t, hasMember(t, r, shadowPartition.accountInProgressKey(kg), qi.ID))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.False(t, hasMember(t, r, fnPart.concurrencyKey(kg), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("account", fnPart.Queue()), qi.ID)) // pseudo-limit for system queues
			require.False(t, hasMember(t, r, kg.Concurrency("p", fnPart.Queue()), qi.ID))

			// item must not be in classic backlog
			require.False(t, hasMember(t, r, fnPart.zsetKey(kg), qi.ID))
		})
	})

	t.Run("don't update run indexes if another queue item is active", func(t *testing.T) {
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

		enqueueToBacklog := false
		q := NewQueue(
			defaultShard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		)
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		runID := ulid.MustNew(ulid.Now(), rand.Reader)

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute)

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
					RunID:       runID,
				},
				QueueName:             nil,
				Throttle:              nil,
				CustomConcurrencyKeys: nil,
			},
			QueueName: nil,
		}

		//
		// Add two queue items to in progress
		//

		// directly enqueue to partition
		enqueueToBacklog = false
		qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		item.ID = ""
		qi2, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		enqueueToBacklog = true

		// put item in progress, this is tested separately
		leaseDur := 5 * time.Second
		leaseID, err := q.Lease(ctx, qi, leaseDur, now, nil)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		leaseID, err = q.Lease(ctx, qi2, leaseDur, now, nil)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		backlog := q.ItemBacklog(ctx, item)
		require.NotEmpty(t, backlog.BacklogID)

		shadowPartition := q.ItemShadowPartition(ctx, item)
		require.NotEmpty(t, shadowPartition.PartitionID)

		constraints := q.partitionConstraintConfigGetter(ctx, shadowPartition.Identifier())
		require.Len(t, constraints.Concurrency.CustomConcurrencyKeys, 0)

		itemIsMember, err := r.SIsMember(kg.ActiveSet("run", runID.String()), qi.ID)
		require.NoError(t, err)
		require.True(t, itemIsMember)

		itemIsMember, err = r.SIsMember(kg.ActiveSet("run", runID.String()), qi2.ID)
		require.NoError(t, err)
		require.True(t, itemIsMember)

		isMember, err := r.SIsMember(kg.ActiveRunsSet("p", fnID.String()), runID.String())
		require.NoError(t, err)
		require.True(t, isMember)

		//
		// Requeue first active item, expect active run items to be updated (decreased by 1)
		// but run still has another active item
		//

		requeueFor := at.Add(30 * time.Minute).Truncate(time.Minute)

		err = q.Requeue(ctx, defaultShard, qi, requeueFor)
		require.NoError(t, err)

		itemIsMember, err = r.SIsMember(kg.ActiveSet("run", runID.String()), qi.ID)
		require.NoError(t, err)
		require.False(t, itemIsMember)

		itemIsMember, err = r.SIsMember(kg.ActiveSet("run", runID.String()), qi2.ID)
		require.NoError(t, err)
		require.True(t, itemIsMember)

		isMember, err = r.SIsMember(kg.ActiveRunsSet("p", fnID.String()), runID.String())
		require.NoError(t, err)
		require.True(t, isMember)

		//
		// Requeue final active item, expect indexes to be cleared out
		//

		err = q.Requeue(ctx, defaultShard, qi2, requeueFor.Add(time.Hour))
		require.NoError(t, err)

		runSetExists := r.Exists(kg.ActiveSet("run", runID.String()))
		require.False(t, runSetExists)
		require.False(t, r.Exists(kg.ActiveRunsSet("p", fnID.String())))
	})

	t.Run("item without throttle key expression should be backfilled", func(t *testing.T) {
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

		oldThrottle := &osqueue.Throttle{
			Key:                 util.XXHash("old"),
			Limit:               10,
			Period:              60,
			UnhashedThrottleKey: "old",
			// Test: Do not store expression hash yet!
			// KeyExpressionHash:   util.XXHash("old-hash"),
		}
		newThrottle := &osqueue.Throttle{
			Key:                 util.XXHash("new"),
			Limit:               10,
			Period:              60,
			UnhashedThrottleKey: "new",
			KeyExpressionHash:   util.XXHash("new-hash"),
		}

		enqueueToBacklog := false
		var refreshCalled bool
		q := NewQueue(
			defaultShard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
			WithRefreshItemThrottle(func(ctx context.Context, item *osqueue.QueueItem) (*osqueue.Throttle, error) {
				refreshCalled = true
				return newThrottle, nil
			}),
		)
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		runID := ulid.MustNew(ulid.Now(), rand.Reader)

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute)

		t.Run("should requeue item to backlog", func(t *testing.T) {
			require.Len(t, r.Keys(), 0)

			item := osqueue.QueueItem{
				ID:          "test",
				FunctionID:  fnID,
				WorkspaceID: wsID,
				Data: osqueue.Item{
					WorkspaceID: wsID,
					Kind:        osqueue.KindStart,
					Identifier: state.Identifier{
						WorkflowID:  fnID,
						AccountID:   accountId,
						WorkspaceID: wsID,
						RunID:       runID,
					},
					QueueName:             nil,
					Throttle:              oldThrottle,
					CustomConcurrencyKeys: nil,
				},
				QueueName: nil,
			}

			oldBacklog := q.ItemBacklog(ctx, item)

			// directly enqueue to partition
			enqueueToBacklog = false
			qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			require.False(t, refreshCalled)

			require.False(t, hasMember(t, r, kg.BacklogSet(oldBacklog.BacklogID), qi.ID))

			shadowPartition := q.ItemShadowPartition(ctx, item)

			fnPart := q.ItemPartition(ctx, defaultShard, item)

			require.True(t, hasMember(t, r, fnPart.zsetKey(kg), qi.ID), r.Keys())

			requeueFor := at.Add(30 * time.Minute).Truncate(time.Minute)

			err = q.Requeue(ctx, defaultShard, qi, requeueFor)
			require.NoError(t, err)

			item.Data.Throttle = newThrottle
			newBacklog := q.ItemBacklog(ctx, item)

			require.True(t, refreshCalled)

			require.False(t, hasMember(t, r, fnPart.zsetKey(kg), qi.ID), r.Keys())
			require.False(t, hasMember(t, r, kg.BacklogSet(oldBacklog.BacklogID), qi.ID))
			require.True(t, hasMember(t, r, kg.BacklogSet(newBacklog.BacklogID), qi.ID))

			require.Equal(t, requeueFor.UnixMilli(), int64(score(t, r, kg.BacklogSet(newBacklog.BacklogID), qi.ID)))
			require.True(t, r.Exists(kg.GlobalAccountShadowPartitions()))
			require.True(t, r.Exists(kg.AccountShadowPartitions(accountId)))

			require.Equal(t, requeueFor.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountId.String())))
			require.Equal(t, requeueFor.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountId), shadowPartition.PartitionID)))

			var requeuedItem osqueue.QueueItem
			queueItemStr := r.HGet(kg.QueueItem(), qi.ID)
			require.NotEmpty(t, queueItemStr)
			require.NoError(t, json.Unmarshal([]byte(queueItemStr), &requeuedItem))

			require.NotNil(t, requeuedItem.Data.Throttle, queueItemStr)
			require.Equal(t, newThrottle.KeyExpressionHash, requeuedItem.Data.Throttle.KeyExpressionHash)
		})
	})

	t.Run("requeue should remove item from ready queue", func(t *testing.T) {
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

		enqueueToBacklog := false
		q := NewQueue(
			defaultShard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		)
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		runID := ulid.MustNew(ulid.Now(), rand.Reader)

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute)

		t.Run("should requeue item to backlog", func(t *testing.T) {
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
						RunID:       runID,
					},
					QueueName:             nil,
					Throttle:              nil,
					CustomConcurrencyKeys: nil,
				},
				QueueName: nil,
			}

			// directly enqueue to partition
			enqueueToBacklog = false
			qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			backlog := q.ItemBacklog(ctx, item)
			require.NotEmpty(t, backlog.BacklogID)

			shadowPartition := q.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			constraints := q.partitionConstraintConfigGetter(ctx, shadowPartition.Identifier())
			require.Len(t, constraints.Concurrency.CustomConcurrencyKeys, 0)

			fnPart, custom1, custom2 := q.ItemPartitions(ctx, defaultShard, item)
			require.NotEmpty(t, fnPart.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), fnPart.PartitionType)
			require.Empty(t, custom1.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), custom1.PartitionType)
			require.Empty(t, custom2.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), custom2.PartitionType)

			require.True(t, hasMember(t, r, fnPart.zsetKey(kg), qi.ID))

			requeueAt := q.clock.Now()

			err = q.Requeue(ctx, defaultShard, qi, requeueAt)
			require.NoError(t, err)

			// expect item to be requeued to backlog
			require.Equal(t, requeueAt.UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID)))
			require.True(t, r.Exists(kg.GlobalAccountShadowPartitions()))
			require.True(t, r.Exists(kg.AccountShadowPartitions(accountId)))

			require.Equal(t, requeueAt.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountId.String())))
			require.Equal(t, requeueAt.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountId), shadowPartition.PartitionID)))

			require.False(t, hasMember(t, r, fnPart.zsetKey(kg), qi.ID), r.Keys())
		})
	})
}

func TestQueueDequeueUpdateAccounting(t *testing.T) {
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

		enqueueToBacklog := false
		q := NewQueue(
			defaultShard,
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		)
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		// use future timestamp because scores will be bounded to the present
		at := time.Now().Add(10 * time.Minute)

		t.Run("should dequeue item and update accounting", func(t *testing.T) {
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

			// directly enqueue to partition
			enqueueToBacklog = false
			qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			// put item in progress, this is tested separately
			now := q.clock.Now()
			leaseDur := 5 * time.Second
			leaseExpires := now.Add(leaseDur)
			leaseID, err := q.Lease(ctx, qi, leaseDur, now, nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			backlog := q.ItemBacklog(ctx, item)
			require.NotEmpty(t, backlog.BacklogID)

			shadowPartition := q.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			constraints := q.partitionConstraintConfigGetter(ctx, shadowPartition.Identifier())
			require.Len(t, constraints.Concurrency.CustomConcurrencyKeys, 0)

			fnPart, custom1, custom2 := q.ItemPartitions(ctx, defaultShard, item)
			require.NotEmpty(t, fnPart.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), fnPart.PartitionType)
			require.Empty(t, custom1.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), custom1.PartitionType)
			require.Empty(t, custom2.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), custom2.PartitionType)

			// expect key queue accounting to contain item in in-progress
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartition.inProgressKey(kg), qi.ID)))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartition.accountInProgressKey(kg), qi.ID)))

			// no active set for default partition since this uses the in progress key
			require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 1))
			require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 2))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.True(t, hasMember(t, r, fnPart.concurrencyKey(kg), qi.ID))
			require.True(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.True(t, hasMember(t, r, kg.Concurrency("p", fnPart.Queue()), qi.ID))

			err = q.Dequeue(ctx, defaultShard, qi)
			require.NoError(t, err)

			// expect item not to be requeued
			require.False(t, hasMember(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID))

			// expect key queue accounting to be updated
			require.False(t, hasMember(t, r, shadowPartition.inProgressKey(kg), qi.ID))
			require.False(t, hasMember(t, r, shadowPartition.accountInProgressKey(kg), qi.ID))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.False(t, hasMember(t, r, fnPart.concurrencyKey(kg), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("p", fnPart.Queue()), qi.ID))

			// item must not be in classic backlog
			require.False(t, hasMember(t, r, fnPart.zsetKey(kg), qi.ID))
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

		enqueueToBacklog := false
		q := NewQueue(
			defaultShard,
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		)
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		// use future timestamp because scores will be bounded to the present
		at := time.Now().Add(10 * time.Minute)

		t.Run("should dequeue item and update accounting", func(t *testing.T) {
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

			// directly enqueue to partition
			enqueueToBacklog = false
			qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			// put item in progress, this is tested separately
			now := q.clock.Now()
			leaseDur := 5 * time.Second
			leaseExpires := now.Add(leaseDur)
			leaseID, err := q.Lease(ctx, qi, leaseDur, now, nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			backlog := q.ItemBacklog(ctx, item)
			require.NotEmpty(t, backlog.BacklogID)

			shadowPartition := q.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			constraints := q.partitionConstraintConfigGetter(ctx, shadowPartition.Identifier())
			require.Len(t, constraints.Concurrency.CustomConcurrencyKeys, 1)

			fnPart, custom1, custom2 := q.ItemPartitions(ctx, defaultShard, item)
			require.NotEmpty(t, fnPart.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), fnPart.PartitionType)
			require.NotEmpty(t, custom1.ID)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom1.PartitionType)
			require.Empty(t, custom2.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), custom2.PartitionType)

			// expect key queue accounting to contain item in in-progress
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartition.inProgressKey(kg), qi.ID)))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartition.accountInProgressKey(kg), qi.ID)))

			// 1 active set for custom concurrency key
			require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope, fnID, unhashedValue)), backlog.customKeyInProgress(kg, 1))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, backlog.customKeyInProgress(kg, 1), qi.ID)))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.True(t, hasMember(t, r, custom1.concurrencyKey(kg), qi.ID))
			require.True(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.True(t, hasMember(t, r, kg.Concurrency("p", fnID.String()), qi.ID), r.Keys())
			require.True(t, hasMember(t, r, kg.Concurrency("custom", fullKey), qi.ID))

			err = q.Dequeue(ctx, defaultShard, qi)
			require.NoError(t, err)

			// expect item not to be requeued
			require.False(t, hasMember(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID))

			// expect key queue accounting to be updated
			require.False(t, hasMember(t, r, shadowPartition.inProgressKey(kg), qi.ID))
			require.False(t, hasMember(t, r, shadowPartition.accountInProgressKey(kg), qi.ID))
			require.False(t, hasMember(t, r, backlog.customKeyInProgress(kg, 1), qi.ID))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.False(t, hasMember(t, r, custom1.concurrencyKey(kg), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("p", fnID.String()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("custom", fullKey), qi.ID))

			// item must not be in classic backlog
			require.False(t, hasMember(t, r, fnPart.zsetKey(kg), qi.ID))
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

		enqueueToBacklog := false
		q := NewQueue(
			defaultShard,
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		)
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		// use future timestamp because scores will be bounded to the present
		at := time.Now().Add(10 * time.Minute)

		t.Run("should dequeue item and update accounting", func(t *testing.T) {
			require.Len(t, r.Keys(), 0)

			hashedConcurrencyKeyExpr1 := hashConcurrencyKey("event.data.userId")
			unhashedValue1 := "user1"
			scope1 := enums.ConcurrencyScopeFn
			fullKey1 := util.ConcurrencyKey(scope1, fnID, unhashedValue1)

			hashedConcurrencyKeyExpr2 := hashConcurrencyKey("event.data.orgId")
			unhashedValue2 := "org1"
			scope2 := enums.ConcurrencyScopeEnv
			fullKey2 := util.ConcurrencyKey(scope2, wsID, unhashedValue2)

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

			// directly enqueue to partition
			enqueueToBacklog = false
			qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			// put item in progress, this is tested separately
			now := q.clock.Now()
			leaseDur := 5 * time.Second
			leaseExpires := now.Add(leaseDur)
			leaseID, err := q.Lease(ctx, qi, leaseDur, now, nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			backlog := q.ItemBacklog(ctx, item)
			require.Len(t, backlog.ConcurrencyKeys, 2)

			shadowPartition := q.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			constraints := q.partitionConstraintConfigGetter(ctx, shadowPartition.Identifier())
			require.Len(t, constraints.Concurrency.CustomConcurrencyKeys, 2)

			fnPart, custom1, custom2 := q.ItemPartitions(ctx, defaultShard, item)
			require.NotEmpty(t, fnPart.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), fnPart.PartitionType)
			require.NotEmpty(t, custom1.ID)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom1.PartitionType)
			require.NotEmpty(t, custom2.ID)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom2.PartitionType)

			// expect key queue accounting to contain item in in-progress
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartition.inProgressKey(kg), qi.ID)))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartition.accountInProgressKey(kg), qi.ID)))

			// 2 active set for custom concurrency keys
			require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope1, fnID, unhashedValue1)), backlog.customKeyInProgress(kg, 1))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, backlog.customKeyInProgress(kg, 1), qi.ID)))

			require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope2, wsID, unhashedValue2)), backlog.customKeyInProgress(kg, 2))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, backlog.customKeyInProgress(kg, 2), qi.ID)))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.True(t, hasMember(t, r, custom1.concurrencyKey(kg), qi.ID))
			require.True(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.True(t, hasMember(t, r, kg.Concurrency("p", fnID.String()), qi.ID), r.Keys())
			require.True(t, hasMember(t, r, kg.Concurrency("custom", fullKey1), qi.ID))
			require.True(t, hasMember(t, r, kg.Concurrency("custom", fullKey2), qi.ID))

			err = q.Dequeue(ctx, defaultShard, qi)
			require.NoError(t, err)

			// expect item not to be requeued
			require.False(t, hasMember(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID))

			// expect key queue accounting to be updated
			require.False(t, hasMember(t, r, shadowPartition.inProgressKey(kg), qi.ID))
			require.False(t, hasMember(t, r, shadowPartition.accountInProgressKey(kg), qi.ID))
			require.False(t, hasMember(t, r, backlog.customKeyInProgress(kg, 1), qi.ID))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.False(t, hasMember(t, r, custom1.concurrencyKey(kg), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("p", fnID.String()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("custom", fullKey1), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("custom", fullKey2), qi.ID))

			// item must not be in classic backlog
			require.False(t, hasMember(t, r, fnPart.zsetKey(kg), qi.ID))
		})
	})

	t.Run("system queues", func(t *testing.T) {
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
		q := NewQueue(
			defaultShard,
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		)
		ctx := context.Background()

		sysQueueName := osqueue.KindQueueMigrate

		// use future timestamp because scores will be bounded to the present
		at := time.Now().Add(10 * time.Minute)

		t.Run("should dequeue item and update accounting", func(t *testing.T) {
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

			// directly enqueue to partition
			enqueueToBacklog = false
			qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			// put item in progress, this is tested separately
			now := q.clock.Now()
			leaseDur := 5 * time.Second
			leaseExpires := now.Add(leaseDur)
			leaseID, err := q.Lease(ctx, qi, leaseDur, now, nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			backlog := q.ItemBacklog(ctx, item)
			require.NotEmpty(t, backlog.BacklogID)

			shadowPartition := q.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			constraints := q.partitionConstraintConfigGetter(ctx, shadowPartition.Identifier())
			require.Len(t, constraints.Concurrency.CustomConcurrencyKeys, 0)

			fnPart, custom1, custom2 := q.ItemPartitions(ctx, defaultShard, item)
			require.NotEmpty(t, fnPart.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), fnPart.PartitionType)
			require.True(t, fnPart.IsSystem())
			require.Empty(t, custom1.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), custom1.PartitionType)
			require.Empty(t, custom2.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), custom2.PartitionType)

			// expect key queue accounting to contain item in in-progress
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartition.inProgressKey(kg), qi.ID)))

			require.Equal(t, kg.Concurrency("account", ""), shadowPartition.accountInProgressKey(kg))
			require.False(t, r.Exists(shadowPartition.accountInProgressKey(kg)))

			// no active set for default partition since this uses the in progress key
			require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 1))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.True(t, hasMember(t, r, fnPart.concurrencyKey(kg), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("account", fnPart.Queue()), qi.ID)) // pseudo-limit for system queue
			require.True(t, hasMember(t, r, kg.Concurrency("p", fnPart.Queue()), qi.ID))

			err = q.Dequeue(ctx, defaultShard, qi)
			require.NoError(t, err)

			// expect item not to be requeued
			require.False(t, hasMember(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID))

			// expect key queue accounting to be updated
			require.False(t, hasMember(t, r, shadowPartition.inProgressKey(kg), qi.ID))
			require.False(t, hasMember(t, r, shadowPartition.accountInProgressKey(kg), qi.ID))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.False(t, hasMember(t, r, fnPart.concurrencyKey(kg), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("account", fnPart.Queue()), qi.ID)) // pseudo-limit for system queue
			require.False(t, hasMember(t, r, kg.Concurrency("p", fnPart.Queue()), qi.ID))

			// item must not be in classic backlog
			require.False(t, hasMember(t, r, fnPart.zsetKey(kg), qi.ID))
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
		WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
			return false
		}),
		WithDisableLeaseChecksForSystemQueues(false),
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
			res, err := q.BacklogRefill(ctx, &backlog, &shadowPart, refillUntil, PartitionConstraintConfig{
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
			res, err := q.BacklogRefill(ctx, &backlog, &shadowPart, refillUntil, PartitionConstraintConfig{
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
			res, err := q.BacklogRefill(ctx, &backlog, &shadowPart, refillUntil, PartitionConstraintConfig{
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
