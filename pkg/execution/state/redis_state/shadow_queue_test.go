package redis_state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngestgo"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueueRefillBacklog(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
	kg := defaultShard.RedisClient.kg

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
	ctx := context.Background()

	accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	runID := ulid.MustNew(ulid.Timestamp(clock.Now()), rand.Reader)

	// use future timestamp because scores will be bounded to the present
	at := clock.Now().Add(1 * time.Minute)

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

	qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
	require.NoError(t, err)

	expectedBacklog := q.ItemBacklog(ctx, item)
	require.NotEmpty(t, expectedBacklog.BacklogID)

	shadowPartition := q.ItemShadowPartition(ctx, item)
	require.NotEmpty(t, shadowPartition.PartitionID)

	t.Run("should find backlog with peek", func(t *testing.T) {
		backlogs, totalCount, err := q.ShadowPartitionPeek(ctx, &shadowPartition, true, at.Add(time.Minute), 10)
		require.NoError(t, err)
		require.Equal(t, 1, totalCount)

		require.Len(t, backlogs, 1)

		require.Equal(t, expectedBacklog, *backlogs[0])
	})

	t.Run("should refill from backlog", func(t *testing.T) {
		require.True(t, hasMember(t, r, kg.BacklogSet(expectedBacklog.BacklogID), qi.ID))

		clock.Advance(10 * time.Minute)

		count, err := rc.Do(ctx, rc.B().Zcount().Key(kg.BacklogSet(expectedBacklog.BacklogID)).Min("-inf").Max(fmt.Sprintf("%d", clock.Now().UnixMilli())).Build()).ToInt64()
		require.NoError(t, err)
		require.Equal(t, 1, int(count))

		res, err := q.BacklogRefill(ctx, &expectedBacklog, &shadowPartition, clock.Now(), &PartitionConstraintConfig{
			Concurrency: ShadowPartitionConcurrency{
				AccountConcurrency:  defaultConcurrency,
				FunctionConcurrency: defaultConcurrency,
			},
		})
		require.NoError(t, err)

		require.Equal(t, 1, res.Refilled)
		require.Equal(t, enums.QueueConstraintNotLimited, res.Constraint)

		require.False(t, hasMember(t, r, kg.BacklogSet(expectedBacklog.BacklogID), qi.ID))

		require.True(t, hasMember(t, r, kg.PartitionQueueSet(enums.PartitionTypeDefault, fnID.String(), ""), qi.ID))
		require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.PartitionQueueSet(enums.PartitionTypeDefault, fnID.String(), ""), qi.ID)))

		require.Equal(t, at.Unix(), int64(score(t, r, kg.GlobalPartitionIndex(), fnID.String())))
		require.Equal(t, at.Unix(), int64(score(t, r, kg.GlobalAccountIndex(), accountId.String())))
		require.Equal(t, at.Unix(), int64(score(t, r, kg.AccountPartitionIndex(accountId), fnID.String())))

		// Run indexes should be updated
		{
			itemIsMember, err := r.SIsMember(kg.ActiveSet("run", runID.String()), qi.ID)
			require.NoError(t, err)
			require.True(t, itemIsMember)

			isMember, err := r.SIsMember(kg.ActiveRunsSet("p", fnID.String()), runID.String())
			require.NoError(t, err)
			require.True(t, isMember)
		}
	})

	t.Run("should clean up dangling pointers", func(t *testing.T) {
		r.FlushAll()

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

		expectedBacklog := q.ItemBacklog(ctx, item)
		require.NotEmpty(t, expectedBacklog.BacklogID)

		_, err = r.ZAdd(kg.BacklogSet(expectedBacklog.BacklogID), float64(at.UnixMilli()), "missing-1")
		require.NoError(t, err)

		_, err = r.ZAdd(kg.BacklogSet(expectedBacklog.BacklogID), float64(at.UnixMilli()), "missing-2")
		require.NoError(t, err)

		_, err = r.ZAdd(kg.BacklogSet(expectedBacklog.BacklogID), float64(at.UnixMilli()), "missing-3")
		require.NoError(t, err)

		shadowPartition := q.ItemShadowPartition(ctx, item)
		require.NotEmpty(t, shadowPartition.PartitionID)

		require.True(t, hasMember(t, r, kg.BacklogSet(expectedBacklog.BacklogID), qi.ID))
		require.True(t, hasMember(t, r, kg.BacklogSet(expectedBacklog.BacklogID), "missing-1"))
		require.True(t, hasMember(t, r, kg.BacklogSet(expectedBacklog.BacklogID), "missing-2"))
		require.True(t, hasMember(t, r, kg.BacklogSet(expectedBacklog.BacklogID), "missing-3"))

		clock.Advance(10 * time.Minute)
		r.FastForward(10 * time.Minute)

		count, err := rc.Do(ctx, rc.B().Zcount().Key(kg.BacklogSet(expectedBacklog.BacklogID)).Min("-inf").Max(fmt.Sprintf("%d", clock.Now().UnixMilli())).Build()).ToInt64()
		require.NoError(t, err)
		require.Equal(t, 4, int(count))

		res, err := q.BacklogRefill(ctx, &expectedBacklog, &shadowPartition, clock.Now(), &PartitionConstraintConfig{
			Concurrency: ShadowPartitionConcurrency{
				AccountConcurrency:  defaultConcurrency,
				FunctionConcurrency: defaultConcurrency,
			},
		})
		require.NoError(t, err)

		require.Equal(t, 1, res.Refilled)
		require.Equal(t, enums.QueueConstraintNotLimited, res.Constraint)

		require.False(t, hasMember(t, r, kg.BacklogSet(expectedBacklog.BacklogID), qi.ID))
		require.False(t, r.Exists(kg.BacklogMeta()))

		require.False(t, hasMember(t, r, kg.BacklogSet(expectedBacklog.BacklogID), "missing-1"))
		require.False(t, hasMember(t, r, kg.BacklogSet(expectedBacklog.BacklogID), "missing-2"))
		require.False(t, hasMember(t, r, kg.BacklogSet(expectedBacklog.BacklogID), "missing-3"))

		require.True(t, hasMember(t, r, kg.PartitionQueueSet(enums.PartitionTypeDefault, fnID.String(), ""), qi.ID))
		require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.PartitionQueueSet(enums.PartitionTypeDefault, fnID.String(), ""), qi.ID)))

		kg.ShadowPartitionSet(shadowPartition.PartitionID)
	})

	t.Run("should allow moving as many as max refill if no capacity constraints are configured", func(t *testing.T) {
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		ctx := context.Background()

		defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}

		clock := clockwork.NewFakeClock()

		enqueueToBacklog := true
		q := NewQueue(
			defaultShard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
			WithRunMode(QueueRunMode{
				Sequential:                        true,
				Scavenger:                         true,
				Partition:                         true,
				Account:                           true,
				AccountWeight:                     85,
				ShadowPartition:                   true,
				AccountShadowPartition:            true,
				AccountShadowPartitionWeight:      85,
				ShadowContinuations:               true,
				ShadowContinuationSkipProbability: 0,
				NormalizePartition:                true,
			}),
			WithBacklogRefillLimit(1),
			WithConcurrencyLimitGetter(func(ctx context.Context, p QueuePartition) PartitionConcurrencyLimits {
				return PartitionConcurrencyLimits{
					AccountLimit:   123,
					FunctionLimit:  45,
					CustomKeyLimit: 0,
				}
			}),
			WithCustomConcurrencyKeyLimitRefresher(func(ctx context.Context, i osqueue.QueueItem) []state.CustomConcurrency {
				return i.Data.GetConcurrencyKeys()
			}),
			WithSystemConcurrencyLimitGetter(func(ctx context.Context, p QueuePartition) SystemPartitionConcurrencyLimits {
				return SystemPartitionConcurrencyLimits{
					GlobalLimit:    789,
					PartitionLimit: 678,
				}
			}),
		)

		addItem := func(id string, identifier state.Identifier, at time.Time) osqueue.QueueItem {
			item := osqueue.QueueItem{
				ID:          id,
				FunctionID:  identifier.WorkflowID,
				WorkspaceID: identifier.WorkspaceID,
				Data: osqueue.Item{
					WorkspaceID:           identifier.WorkspaceID,
					Kind:                  osqueue.KindEdge,
					Identifier:            identifier,
					QueueName:             nil,
					Throttle:              nil,
					CustomConcurrencyKeys: nil,
				},
				QueueName: nil,
			}

			qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			return qi
		}
		at := clock.Now()

		qi1 := addItem("test1", state.Identifier{
			AccountID:   accountId,
			WorkspaceID: wsID,
			WorkflowID:  fnID,
		}, at)

		addItem("test2", state.Identifier{
			AccountID:   accountId,
			WorkspaceID: wsID,
			WorkflowID:  fnID,
		}, at)

		backlog := q.ItemBacklog(ctx, qi1)
		shadowPart := q.ItemShadowPartition(ctx, qi1)

		refillUntil := at.Add(time.Minute)

		res, err := q.BacklogRefill(ctx, &backlog, &shadowPart, refillUntil, &PartitionConstraintConfig{
			Concurrency: ShadowPartitionConcurrency{
				AccountConcurrency:  123,
				FunctionConcurrency: 45,
			},
		})
		require.NoError(t, err)

		require.Equal(t, 2, res.TotalBacklogCount)
		require.Equal(t, 2, res.BacklogCountUntil)
		require.Equal(t, 45, res.Capacity) // limit by function concurrency
		require.Equal(t, 1, res.Refill)    // limited by max refill limit of 1
		require.Equal(t, 1, res.Refilled)
		require.Equal(t, enums.QueueConstraintNotLimited, res.Constraint)
	})

	t.Run("should not move future items but adjust pointers", func(t *testing.T) {
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		ctx := context.Background()

		defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}

		clock := clockwork.NewFakeClock()

		enqueueToBacklog := true
		q := NewQueue(
			defaultShard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithEnqueueSystemPartitionsToBacklog(false),
			WithDisableLeaseChecksForSystemQueues(false),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
			WithRunMode(QueueRunMode{
				Sequential:                        true,
				Scavenger:                         true,
				Partition:                         true,
				Account:                           true,
				AccountWeight:                     85,
				ShadowPartition:                   true,
				AccountShadowPartition:            true,
				AccountShadowPartitionWeight:      85,
				ShadowContinuations:               true,
				ShadowContinuationSkipProbability: 0,
				NormalizePartition:                true,
			}),
			WithBacklogRefillLimit(500),
			WithConcurrencyLimitGetter(func(ctx context.Context, p QueuePartition) PartitionConcurrencyLimits {
				return PartitionConcurrencyLimits{
					AccountLimit:   123,
					FunctionLimit:  45,
					CustomKeyLimit: 0,
				}
			}),
			WithCustomConcurrencyKeyLimitRefresher(func(ctx context.Context, i osqueue.QueueItem) []state.CustomConcurrency {
				return i.Data.GetConcurrencyKeys()
			}),
			WithSystemConcurrencyLimitGetter(func(ctx context.Context, p QueuePartition) SystemPartitionConcurrencyLimits {
				return SystemPartitionConcurrencyLimits{
					GlobalLimit:    789,
					PartitionLimit: 678,
				}
			}),
		)

		addItem := func(id string, identifier state.Identifier, at time.Time) osqueue.QueueItem {
			item := osqueue.QueueItem{
				ID:          id,
				FunctionID:  identifier.WorkflowID,
				WorkspaceID: identifier.WorkspaceID,
				Data: osqueue.Item{
					WorkspaceID:           identifier.WorkspaceID,
					Kind:                  osqueue.KindEdge,
					Identifier:            identifier,
					QueueName:             nil,
					Throttle:              nil,
					CustomConcurrencyKeys: nil,
				},
				QueueName: nil,
			}

			qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			return qi
		}
		at := clock.Now()

		qi1 := addItem("test1", state.Identifier{
			AccountID:   accountId,
			WorkspaceID: wsID,
			WorkflowID:  fnID,
		}, at)

		futureAt := at.Add(34 * time.Minute) // random value
		qi2 := addItem("test2", state.Identifier{
			AccountID:   accountId,
			WorkspaceID: wsID,
			WorkflowID:  fnID,
		}, futureAt)

		backlog := q.ItemBacklog(ctx, qi1)
		shadowPart := q.ItemShadowPartition(ctx, qi1)

		require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), qi1.ID)))
		require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.ShadowPartitionSet(shadowPart.PartitionID), backlog.BacklogID)))
		require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountId.String())), r.Keys())
		require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountId), shadowPart.PartitionID)))

		refillUntil := at.Add(time.Minute)

		res, err := q.BacklogRefill(ctx, &backlog, &shadowPart, refillUntil, &PartitionConstraintConfig{
			Concurrency: ShadowPartitionConcurrency{
				AccountConcurrency:  123,
				FunctionConcurrency: 45,
			},
		})
		require.NoError(t, err)

		require.Equal(t, 2, res.TotalBacklogCount)
		require.Equal(t, 1, res.BacklogCountUntil)
		require.Equal(t, 45, res.Capacity) // limit by function concurrency
		require.Equal(t, 1, res.Refill)
		require.Equal(t, 1, res.Refilled)
		require.Equal(t, enums.QueueConstraintNotLimited, res.Constraint)

		require.Equal(t, futureAt.UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), qi2.ID)))
		require.Equal(t, futureAt.UnixMilli(), int64(score(t, r, kg.ShadowPartitionSet(shadowPart.PartitionID), backlog.BacklogID)))
		require.NotEqual(t, at.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, futureAt.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, futureAt.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountId.String())))
		require.Equal(t, futureAt.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountId), shadowPart.PartitionID)))

		toTheFuture := futureAt.Sub(at) + time.Minute
		r.FastForward(toTheFuture)
		clock.Advance(toTheFuture)

		refillUntil = futureAt.Add(time.Minute)

		res, err = q.BacklogRefill(ctx, &backlog, &shadowPart, refillUntil, &PartitionConstraintConfig{
			Concurrency: ShadowPartitionConcurrency{
				AccountConcurrency:  123,
				FunctionConcurrency: 45,
			},
		})
		require.NoError(t, err)

		require.Equal(t, 1, res.TotalBacklogCount)
		require.Equal(t, 1, res.BacklogCountUntil)
		require.Equal(t, 44, res.Capacity) // limit by function concurrency
		require.Equal(t, 1, res.Refill)
		require.Equal(t, 1, res.Refilled)
		require.Equal(t, enums.QueueConstraintNotLimited, res.Constraint)

		require.False(t, r.Exists(kg.BacklogSet(backlog.BacklogID)))
		require.False(t, r.Exists(kg.ShadowPartitionSet(shadowPart.PartitionID)))
	})

	t.Run("should move partition to active check queue when running into concurrency limit", func(t *testing.T) {
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		ctx := context.Background()

		defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}

		clock := clockwork.NewFakeClock()

		enqueueToBacklog := true
		q := NewQueue(
			defaultShard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithEnqueueSystemPartitionsToBacklog(false),
			WithDisableLeaseChecksForSystemQueues(false),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
			WithRunMode(QueueRunMode{
				Sequential:                        true,
				Scavenger:                         true,
				Partition:                         true,
				Account:                           true,
				AccountWeight:                     85,
				ShadowPartition:                   true,
				AccountShadowPartition:            true,
				AccountShadowPartitionWeight:      85,
				ShadowContinuations:               true,
				ShadowContinuationSkipProbability: 0,
				NormalizePartition:                true,
				ActiveChecker:                     true,
			}),
			WithBacklogRefillLimit(500),
			WithConcurrencyLimitGetter(func(ctx context.Context, p QueuePartition) PartitionConcurrencyLimits {
				return PartitionConcurrencyLimits{
					AccountLimit:   123,
					FunctionLimit:  45,
					CustomKeyLimit: 0,
				}
			}),
			WithCustomConcurrencyKeyLimitRefresher(func(ctx context.Context, i osqueue.QueueItem) []state.CustomConcurrency {
				return i.Data.GetConcurrencyKeys()
			}),
			WithSystemConcurrencyLimitGetter(func(ctx context.Context, p QueuePartition) SystemPartitionConcurrencyLimits {
				return SystemPartitionConcurrencyLimits{
					GlobalLimit:    789,
					PartitionLimit: 678,
				}
			}),
			WithActiveSpotCheckProbability(func(ctx context.Context, acctID uuid.UUID) (int, int) {
				return 100, 100
			}),
		)

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

		qi, err := q.EnqueueItem(ctx, defaultShard, item, q.clock.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		fnID2 := uuid.New()

		item2 := osqueue.QueueItem{
			ID:          "test-2",
			FunctionID:  fnID2,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID:  fnID2,
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

		_, err = q.EnqueueItem(ctx, defaultShard, item2, q.clock.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		b := q.ItemBacklog(ctx, qi)
		sp := q.ItemShadowPartition(ctx, qi)

		enqueueToBacklog = true
		res, err := q.BacklogRefill(ctx, &b, &sp, q.clock.Now().Add(10*time.Second), &PartitionConstraintConfig{
			Concurrency: ShadowPartitionConcurrency{
				AccountConcurrency:  1,
				FunctionConcurrency: 1,
			},
		})
		require.NoError(t, err)
		require.Equal(t, 1, res.TotalBacklogCount)
		require.Equal(t, 1, res.Capacity)
		require.Equal(t, 1, res.Refill)
		require.Equal(t, 1, res.Refilled)
		require.Equal(t, enums.QueueConstraintNotLimited, res.Constraint)

		b2 := q.ItemBacklog(ctx, item2)
		sp2 := q.ItemShadowPartition(ctx, item2)

		enqueueToBacklog = true
		res, err = q.BacklogRefill(ctx, &b2, &sp2, q.clock.Now().Add(10*time.Second), &PartitionConstraintConfig{
			Concurrency: ShadowPartitionConcurrency{
				AccountConcurrency:  1,
				FunctionConcurrency: 1,
			},
		})
		require.NoError(t, err)
		require.Equal(t, 1, res.TotalBacklogCount)
		require.Equal(t, 0, res.Capacity)
		require.Equal(t, 0, res.Refill)
		require.Equal(t, 0, res.Refilled)
		require.Equal(t, enums.QueueConstraintAccountConcurrency, res.Constraint)

		require.True(t, r.Exists(kg.BacklogActiveCheckSet()))
		members, err := r.ZMembers(kg.BacklogActiveCheckSet())
		require.NoError(t, err)
		require.Len(t, members, 1)
		require.Equal(t, b2.BacklogID, members[0])
	})
}

func TestQueueShadowPartitionLease(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	ctx := context.Background()

	defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
	kg := defaultShard.RedisClient.kg

	clock := clockwork.NewFakeClock()

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

	fnID, accountID, envID := uuid.New(), uuid.New(), uuid.New()
	shadowPart := &QueueShadowPartition{
		PartitionID: fnID.String(),
		LeaseID:     nil,
		FunctionID:  &fnID,
		EnvID:       &envID,
		AccountID:   &accountID,
		PauseRefill: false,
	}

	marshaled, err := json.Marshal(shadowPart)
	require.NoError(t, err)

	t.Run("should not be able to lease missing partition", func(t *testing.T) {
		_, err = q.ShadowPartitionLease(ctx, shadowPart, ShadowPartitionLeaseDuration)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrShadowPartitionNotFound)

		r.HSet(kg.ShadowPartitionMeta(), shadowPart.PartitionID, string(marshaled))
	})

	var leaseID *ulid.ULID

	t.Run("first lease should lease shadow partition", func(t *testing.T) {
		dur := ShadowPartitionLeaseDuration
		expectedLeaseExpiry := clock.Now().Add(dur)

		leaseID, err = q.ShadowPartitionLease(ctx, shadowPart, dur)
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		leaseTime := ulid.Time(leaseID.Time())
		require.Equal(t, expectedLeaseExpiry.UnixMilli(), leaseTime.UnixMilli())

		leasedPart := QueueShadowPartition{}
		metaStr := r.HGet(kg.ShadowPartitionMeta(), shadowPart.PartitionID)
		require.NoError(t, json.Unmarshal([]byte(metaStr), &leasedPart))

		require.NotNil(t, leasedPart.LeaseID)
		require.Equal(t, *leaseID, *leasedPart.LeaseID)

		// Expect shadow partition to be pushed back
		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))
	})

	t.Run("should not be able to lease again", func(t *testing.T) {
		_, err = q.ShadowPartitionLease(ctx, shadowPart, ShadowPartitionLeaseDuration)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrShadowPartitionAlreadyLeased)
	})

	t.Run("extend lease should work", func(t *testing.T) {
		// Simulate 2s have passed
		clock.Advance(2 * time.Second)
		r.FastForward(2 * time.Second)

		dur := ShadowPartitionLeaseDuration
		expectedLeaseExpiry := clock.Now().Add(dur)

		newLeaseID, err := q.ShadowPartitionExtendLease(ctx, shadowPart, *leaseID, dur)
		require.NoError(t, err)
		require.NotNil(t, newLeaseID)
		leaseID = newLeaseID

		leaseTime := ulid.Time(leaseID.Time())
		require.Equal(t, expectedLeaseExpiry.UnixMilli(), leaseTime.UnixMilli())

		leasedPart := QueueShadowPartition{}
		require.NoError(t, json.Unmarshal([]byte(r.HGet(kg.ShadowPartitionMeta(), shadowPart.PartitionID)), &leasedPart))

		require.NotNil(t, leasedPart.LeaseID)
		require.Equal(t, *leaseID, *leasedPart.LeaseID)

		// Expect shadow partition to be pushed back
		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))
	})

	t.Run("return lease should work", func(t *testing.T) {
		// Simulate 2s have passed
		clock.Advance(2 * time.Second)
		r.FastForward(2 * time.Second)

		// Simulate next backlog item in shadow partition
		nextBacklogAt := clock.Now().Add(3 * time.Hour)
		_, err := r.ZAdd(kg.ShadowPartitionSet(shadowPart.PartitionID), float64(nextBacklogAt.UnixMilli()), "backlog-test")
		require.NoError(t, err)

		err = q.ShadowPartitionRequeue(ctx, shadowPart, nil)
		require.NoError(t, err)

		leasedPart := QueueShadowPartition{}
		require.NoError(t, json.Unmarshal([]byte(r.HGet(kg.ShadowPartitionMeta(), shadowPart.PartitionID)), &leasedPart))

		require.Nil(t, leasedPart.LeaseID)

		// Expect shadow partition to be pushed back
		require.Equal(t, nextBacklogAt.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, nextBacklogAt.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
		require.Equal(t, nextBacklogAt.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))
	})

	t.Run("shadow partition requeue should clean dangling pointers", func(t *testing.T) {
		r.FlushAll()

		r.HSet(kg.ShadowPartitionMeta(), shadowPart.PartitionID, string(marshaled))

		now := clock.Now()
		dur := ShadowPartitionLeaseDuration
		expectedLeaseExpiry := now.Add(dur)
		leaseID, err := q.ShadowPartitionLease(ctx, shadowPart, dur)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))

		err = q.ShadowPartitionRequeue(ctx, shadowPart, nil)
		require.NoError(t, err)

		require.False(t, r.Exists(kg.ShadowPartitionMeta()))

		// Expect pointers to be cleaned up
		require.False(t, r.Exists(kg.GlobalShadowPartitionSet()))
		require.False(t, r.Exists(kg.AccountShadowPartitions(accountID)))
		require.False(t, r.Exists(kg.GlobalAccountShadowPartitions()))
	})

	t.Run("forcing requeue should work if earliest backlog is earlier", func(t *testing.T) {
		r.FlushAll()

		r.HSet(kg.ShadowPartitionMeta(), shadowPart.PartitionID, string(marshaled))

		// Simulate next backlog item in shadow partition
		nextBacklogAt := clock.Now().Add(15 * time.Minute)
		_, err := r.ZAdd(kg.ShadowPartitionSet(shadowPart.PartitionID), float64(nextBacklogAt.UnixMilli()), "backlog-test")
		require.NoError(t, err)

		now := clock.Now()
		dur := ShadowPartitionLeaseDuration
		expectedLeaseExpiry := now.Add(dur)
		leaseID, err := q.ShadowPartitionLease(ctx, shadowPart, dur)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))

		forceRequeueAt := time.Now().Add(32 * time.Minute)
		err = q.ShadowPartitionRequeue(ctx, shadowPart, &forceRequeueAt)
		require.NoError(t, err)

		leasedPart := QueueShadowPartition{}
		require.NoError(t, json.Unmarshal([]byte(r.HGet(kg.ShadowPartitionMeta(), shadowPart.PartitionID)), &leasedPart))

		require.Nil(t, leasedPart.LeaseID)

		// Expect pointers to be aligned with forced time
		require.Equal(t, forceRequeueAt.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, forceRequeueAt.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
		require.Equal(t, forceRequeueAt.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))
	})

	t.Run("forcing requeue should be ignored if earliest backlog is later", func(t *testing.T) {
		r.FlushAll()

		r.HSet(kg.ShadowPartitionMeta(), shadowPart.PartitionID, string(marshaled))

		// Simulate next backlog item in shadow partition
		nextBacklogAt := clock.Now().Add(48 * time.Minute)
		_, err := r.ZAdd(kg.ShadowPartitionSet(shadowPart.PartitionID), float64(nextBacklogAt.UnixMilli()), "backlog-test")
		require.NoError(t, err)

		now := clock.Now()
		dur := ShadowPartitionLeaseDuration
		expectedLeaseExpiry := now.Add(dur)
		leaseID, err := q.ShadowPartitionLease(ctx, shadowPart, dur)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))

		forceRequeueAt := time.Now().Add(32 * time.Minute)
		err = q.ShadowPartitionRequeue(ctx, shadowPart, &forceRequeueAt)
		require.NoError(t, err)

		leasedPart := QueueShadowPartition{}
		require.NoError(t, json.Unmarshal([]byte(r.HGet(kg.ShadowPartitionMeta(), shadowPart.PartitionID)), &leasedPart))

		require.Nil(t, leasedPart.LeaseID)

		// Expect pointers to be aligned with next backlog item
		require.Equal(t, nextBacklogAt.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, nextBacklogAt.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
		require.Equal(t, nextBacklogAt.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))
	})
}

func TestQueueShadowScanner(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	ctx := context.Background()

	defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}

	clock := clockwork.NewFakeClock()

	enqueueToBacklog := true
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

	fnID, accountID, envID := uuid.New(), uuid.New(), uuid.New()

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
			},
			QueueName:             nil,
			Throttle:              nil,
			CustomConcurrencyKeys: nil,
		},
		QueueName: nil,
	}

	at := clock.Now()

	_, err = q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
	require.NoError(t, err)

	qspc := make(chan shadowPartitionChanMsg, 1)

	err = q.scanShadowPartitions(ctx, at, qspc)
	require.NoError(t, err)

	select {
	case msg := <-qspc:
		require.Equal(t, fnID, *msg.sp.FunctionID)
		require.Equal(t, accountID, *msg.sp.AccountID)
	default:
		require.Fail(t, "expected message to be added")
	}
}

func TestQueueShadowScannerContinuations(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	ctx := context.Background()

	defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
	kg := defaultShard.RedisClient.kg

	clock := clockwork.NewFakeClock()

	enqueueToBacklog := true
	q := NewQueue(
		defaultShard,
		WithClock(clock),
		WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
			return enqueueToBacklog
		}),
		WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
			return true
		}),
		WithRunMode(QueueRunMode{
			Sequential:                        true,
			Scavenger:                         true,
			Partition:                         true,
			Account:                           true,
			AccountWeight:                     85,
			ShadowPartition:                   true,
			AccountShadowPartition:            true,
			AccountShadowPartitionWeight:      85,
			ShadowContinuations:               true,
			ShadowContinuationSkipProbability: 0,
			NormalizePartition:                true,
		}),
	)

	fnID1, accountID1, envID1 := uuid.New(), uuid.New(), uuid.New()
	fnID2, accountID2, envID2 := uuid.New(), uuid.New(), uuid.New()

	addItem := func(id string, identifier state.Identifier, at time.Time) osqueue.QueueItem {
		item := osqueue.QueueItem{
			ID:          id,
			FunctionID:  identifier.WorkflowID,
			WorkspaceID: identifier.WorkspaceID,
			Data: osqueue.Item{
				WorkspaceID:           identifier.WorkspaceID,
				Kind:                  osqueue.KindEdge,
				Identifier:            identifier,
				QueueName:             nil,
				Throttle:              nil,
				CustomConcurrencyKeys: nil,
			},
			QueueName: nil,
		}

		qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		return qi
	}
	at := clock.Now()

	item1 := addItem("test1", state.Identifier{
		AccountID:   accountID1,
		WorkspaceID: envID1,
		WorkflowID:  fnID1,
	}, at)

	item2 := addItem("test2", state.Identifier{
		AccountID:   accountID2,
		WorkspaceID: envID2,
		WorkflowID:  fnID2,
	}, at)

	// we leave some room for multiple partitions as scanShadowPartitions will
	// call both scan continuations and the regular scanner, so the first item
	// is expected to be the continuation, followed by the actual scan run
	qspc := make(chan shadowPartitionChanMsg, 10)

	sp1 := q.ItemShadowPartition(ctx, item1)
	sp2 := q.ItemShadowPartition(ctx, item2)
	require.NotEqual(t, sp1, sp2)

	t.Run("should retrieve using continuation", func(t *testing.T) {
		q.addShadowContinue(ctx, &sp1, 1)

		q.shadowContinuesLock.Lock()
		cont, ok := q.shadowContinues[sp1.PartitionID]
		require.True(t, ok)
		require.Equal(t, uint(1), cont.count)
		require.Equal(t, sp1, *cont.shadowPart)
		q.shadowContinuesLock.Unlock()

		fmt.Println("scanning")

		err = q.scanShadowPartitions(ctx, at, qspc)
		require.NoError(t, err)

		fmt.Println("waiting for message")

		// check that it's scanned and gone
		q.shadowContinuesLock.Lock()
		defer q.shadowContinuesLock.Unlock()

		_, ok = q.shadowContinues[sp1.PartitionID]
		require.False(t, ok)
	})

	t.Run("should increase continuations when more items are available", func(t *testing.T) {
		r.FlushAll()

		q.shadowContinuesLock.Lock()
		clear(q.shadowContinues)
		clear(q.shadowContinueCooldown)
		q.shadowContinuesLock.Unlock()

		q.backlogRefillLimit = 1

		addItem("test1", state.Identifier{
			AccountID:   accountID1,
			WorkspaceID: envID1,
			WorkflowID:  fnID1,
		}, at)

		addItem("test2", state.Identifier{
			AccountID:   accountID1,
			WorkspaceID: envID1,
			WorkflowID:  fnID1,
		}, at)

		addItem("test3", state.Identifier{
			AccountID:   accountID2,
			WorkspaceID: envID2,
			WorkflowID:  fnID2,
		}, at)

		q.addShadowContinue(ctx, &sp1, 1)

		// Process and refill once
		err := q.processShadowPartition(ctx, &sp1, 1)
		require.NoError(t, err)

		// Expect continuation to be set
		q.shadowContinuesLock.Lock()
		cont, ok := q.shadowContinues[sp1.PartitionID]
		require.True(t, ok)
		require.Equal(t, uint(2), cont.count)
		require.Equal(t, sp1, *cont.shadowPart)
		q.shadowContinuesLock.Unlock()

		// Process and refill again, final item in backlog
		err = q.processShadowPartition(ctx, &sp1, 1)
		require.NoError(t, err)

		// Expect continuation to be cleared out
		q.shadowContinuesLock.Lock()
		_, ok = q.shadowContinues[sp1.PartitionID]
		require.False(t, ok)
		q.shadowContinuesLock.Unlock()
	})

	t.Run("should remove continuation on missing shadow partition", func(t *testing.T) {
		r.FlushAll()

		q.shadowContinuesLock.Lock()
		clear(q.shadowContinues)
		clear(q.shadowContinueCooldown)
		q.shadowContinuesLock.Unlock()

		q.backlogRefillLimit = 1

		addItem("test1", state.Identifier{
			AccountID:   accountID1,
			WorkspaceID: envID1,
			WorkflowID:  fnID1,
		}, at)

		addItem("test2", state.Identifier{
			AccountID:   accountID1,
			WorkspaceID: envID1,
			WorkflowID:  fnID1,
		}, at)

		addItem("test3", state.Identifier{
			AccountID:   accountID2,
			WorkspaceID: envID2,
			WorkflowID:  fnID2,
		}, at)

		// Process and refill once
		err := q.processShadowPartition(ctx, &sp1, 1)
		require.NoError(t, err)

		// Expect continuation to be set
		q.shadowContinuesLock.Lock()
		cont, ok := q.shadowContinues[sp1.PartitionID]
		require.True(t, ok)
		require.Equal(t, uint(2), cont.count)
		require.Equal(t, sp1, *cont.shadowPart)
		q.shadowContinuesLock.Unlock()

		// Drop shadow partition
		r.HDel(kg.ShadowPartitionMeta(), sp1.PartitionID)

		// Process and refill again, final item in backlog
		err = q.processShadowPartition(ctx, &sp1, 1)
		require.NoError(t, err)

		// Expect continuation to be cleared out
		q.shadowContinuesLock.Lock()
		_, ok = q.shadowContinues[sp1.PartitionID]
		require.False(t, ok)
		q.shadowContinuesLock.Unlock()
	})

	t.Run("should remove continuation on leased shadow partition", func(t *testing.T) {
		r.FlushAll()

		q.shadowContinuesLock.Lock()
		clear(q.shadowContinues)
		clear(q.shadowContinueCooldown)
		q.shadowContinuesLock.Unlock()

		q.backlogRefillLimit = 1

		addItem("test1", state.Identifier{
			AccountID:   accountID1,
			WorkspaceID: envID1,
			WorkflowID:  fnID1,
		}, at)

		addItem("test2", state.Identifier{
			AccountID:   accountID1,
			WorkspaceID: envID1,
			WorkflowID:  fnID1,
		}, at)

		addItem("test3", state.Identifier{
			AccountID:   accountID2,
			WorkspaceID: envID2,
			WorkflowID:  fnID2,
		}, at)

		// Process and refill once
		err := q.processShadowPartition(ctx, &sp1, 1)
		require.NoError(t, err)

		// Expect continuation to be set
		q.shadowContinuesLock.Lock()
		cont, ok := q.shadowContinues[sp1.PartitionID]
		require.True(t, ok)
		require.Equal(t, uint(2), cont.count)
		require.Equal(t, sp1, *cont.shadowPart)
		q.shadowContinuesLock.Unlock()

		// Simulate another process leasing the shadow partition
		spCopy := sp1
		leaseID, err := q.ShadowPartitionLease(ctx, &spCopy, 3*time.Minute)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		// Process and refill again, final item in backlog
		err = q.processShadowPartition(ctx, &sp1, 1)
		require.NoError(t, err)

		// Expect continuation to be cleared out
		q.shadowContinuesLock.Lock()
		_, ok = q.shadowContinues[sp1.PartitionID]
		require.False(t, ok)
		q.shadowContinuesLock.Unlock()
	})
}

func TestRefillConstraints(t *testing.T) {
	fnID1, accountID1, envID1 := uuid.New(), uuid.New(), uuid.New()

	type knobs struct {
		maxRefill              int
		danglingItemsInBacklog int

		accountConcurrencyLimit  int
		functionConcurrencyLimit int

		throttle              *osqueue.Throttle
		customConcurrencyKey1 *state.CustomConcurrency
		customConcurrencyKey2 *state.CustomConcurrency
		isStartItem           bool
	}

	type expected struct {
		result            BacklogRefillResult
		itemsInBacklog    int
		itemsInReadyQueue int
		retryAt           time.Duration
	}

	type currentValues struct {
		itemsInBacklog int

		accountActive               int
		functionActive              int
		customConcurrencyKey1Active int
		customConcurrencyKey2Active int

		throttleUsageWithinPeriod int
	}

	ck1 := createConcurrencyKey(enums.ConcurrencyScopeFn, fnID1, "bruno", 5)

	ck2 := createConcurrencyKey(enums.ConcurrencyScopeEnv, envID1, "inngest", 10)

	throttleKey := "bruno"
	throttle := &osqueue.Throttle{
		Key:                 util.XXHash(throttleKey),
		Limit:               100,
		Burst:               10,
		Period:              int((10 * time.Hour).Seconds()),
		KeyExpressionHash:   util.XXHash("event.data.userID"),
		UnhashedThrottleKey: throttleKey,
	}

	tests := []struct {
		name          string
		currentValues currentValues
		knobs         knobs
		expected      expected
	}{
		{
			name: "simple item",
			currentValues: currentValues{
				itemsInBacklog: 1,
			},
			knobs: knobs{
				maxRefill:                1,
				accountConcurrencyLimit:  20,
				functionConcurrencyLimit: 10,
			},
			expected: expected{
				result: BacklogRefillResult{
					Constraint:        enums.QueueConstraintNotLimited,
					TotalBacklogCount: 1,
					BacklogCountUntil: 1,
					Capacity:          10,
					Refill:            1,
					Refilled:          1,
				},
				itemsInBacklog:    0,
				itemsInReadyQueue: 1,
			},
		},
		// Function limits
		{
			name: "function limits disallow",
			currentValues: currentValues{
				itemsInBacklog: 40,
				functionActive: 10,
			},
			knobs: knobs{
				maxRefill:                50,
				accountConcurrencyLimit:  20,
				functionConcurrencyLimit: 10,
			},
			expected: expected{
				result: BacklogRefillResult{
					Constraint:        enums.QueueConstraintFunctionConcurrency,
					TotalBacklogCount: 40,
					BacklogCountUntil: 40,
					Capacity:          0,
					Refill:            0,
					Refilled:          0,
				},
				itemsInBacklog:    40,
				itemsInReadyQueue: 0,
			},
		},
		{
			name: "function limits disallow already exceeding",
			currentValues: currentValues{
				itemsInBacklog: 40,
				functionActive: 30, // 30 running but only 10 allowed
			},
			knobs: knobs{
				maxRefill:                50,
				accountConcurrencyLimit:  20,
				functionConcurrencyLimit: 10,
			},
			expected: expected{
				result: BacklogRefillResult{
					Constraint:        enums.QueueConstraintFunctionConcurrency,
					TotalBacklogCount: 40,
					BacklogCountUntil: 40,
					Capacity:          0, // would be -20 but can't go negative
					Refill:            0,
					Refilled:          0,
				},
				itemsInBacklog:    40,
				itemsInReadyQueue: 0,
			},
		},
		{
			name: "function limits allow",
			currentValues: currentValues{
				itemsInBacklog: 40,
				functionActive: 9,
			},
			knobs: knobs{
				maxRefill:                50,
				accountConcurrencyLimit:  20,
				functionConcurrencyLimit: 10,
			},
			expected: expected{
				result: BacklogRefillResult{
					Constraint:        enums.QueueConstraintNotLimited,
					TotalBacklogCount: 40,
					BacklogCountUntil: 40,
					Capacity:          1,
					Refill:            1,
					Refilled:          1,
				},
				itemsInBacklog:    39,
				itemsInReadyQueue: 1,
			},
		},
		// Account limits
		{
			name: "account limits disallow",
			currentValues: currentValues{
				itemsInBacklog: 40,
				accountActive:  20,
			},
			knobs: knobs{
				maxRefill:                50,
				accountConcurrencyLimit:  20,
				functionConcurrencyLimit: 10,
			},
			expected: expected{
				result: BacklogRefillResult{
					Constraint:        enums.QueueConstraintAccountConcurrency,
					TotalBacklogCount: 40,
					BacklogCountUntil: 40,
					Capacity:          0,
					Refill:            0,
					Refilled:          0,
				},
				itemsInBacklog:    40,
				itemsInReadyQueue: 0,
			},
		},
		{
			name: "account limits allow",
			currentValues: currentValues{
				itemsInBacklog: 40,
				accountActive:  19,
			},
			knobs: knobs{
				maxRefill:                50,
				accountConcurrencyLimit:  20,
				functionConcurrencyLimit: 10,
			},
			expected: expected{
				result: BacklogRefillResult{
					Constraint:        enums.QueueConstraintNotLimited,
					TotalBacklogCount: 40,
					BacklogCountUntil: 40,
					Capacity:          1,
					Refill:            1,
					Refilled:          1,
				},
				itemsInBacklog:    39,
				itemsInReadyQueue: 1,
			},
		},
		// Single custom concurrency key limits
		{
			name: "single custom concurrency key limits allow",
			knobs: knobs{
				maxRefill:                50,
				accountConcurrencyLimit:  30,
				functionConcurrencyLimit: 20,
				customConcurrencyKey1:    &ck1,
			},
			currentValues: currentValues{
				itemsInBacklog:              40,
				accountActive:               21,
				functionActive:              11,
				customConcurrencyKey1Active: 2,
			},
			expected: expected{
				result: BacklogRefillResult{
					Constraint:        enums.QueueConstraintNotLimited,
					TotalBacklogCount: 40,
					BacklogCountUntil: 40,
					Capacity:          3,
					Refill:            3,
					Refilled:          3,
				},
				itemsInBacklog:    37,
				itemsInReadyQueue: 3,
			},
		},
		{
			name: "single custom concurrency key limits disallow",
			currentValues: currentValues{
				itemsInBacklog:              40,
				accountActive:               20,
				functionActive:              10,
				customConcurrencyKey1Active: 5,
			},
			knobs: knobs{
				maxRefill:                50,
				accountConcurrencyLimit:  30,
				functionConcurrencyLimit: 20,
				customConcurrencyKey1:    &ck1,
			},
			expected: expected{
				result: BacklogRefillResult{
					Constraint:        enums.QueueConstraintCustomConcurrencyKey1,
					TotalBacklogCount: 40,
					BacklogCountUntil: 40,
					Capacity:          0,
					Refill:            0,
					Refilled:          0,
				},
				itemsInBacklog:    40,
				itemsInReadyQueue: 0,
			},
		},
		// Dual custom concurrency key limits
		{
			name: "dual custom concurrency key limits allow",
			currentValues: currentValues{
				itemsInBacklog:              40,
				accountActive:               20, // 20 out of 30
				functionActive:              10, // 10 out of 20
				customConcurrencyKey1Active: 2,  // 2 out of 5
				customConcurrencyKey2Active: 8,  // 8 out of 10
			},
			knobs: knobs{
				maxRefill:                50,
				accountConcurrencyLimit:  30,
				functionConcurrencyLimit: 20,
				customConcurrencyKey1:    &ck1,
				customConcurrencyKey2:    &ck2,
			},
			expected: expected{
				result: BacklogRefillResult{
					Constraint:        enums.QueueConstraintNotLimited,
					TotalBacklogCount: 40,
					BacklogCountUntil: 40,
					Capacity:          2,
					Refill:            2,
					Refilled:          2,
				},
				itemsInBacklog:    38,
				itemsInReadyQueue: 2,
			},
		},
		{
			name: "dual custom concurrency key limits disallow",
			currentValues: currentValues{
				itemsInBacklog:              40,
				accountActive:               20, // 20 out of 30
				functionActive:              10, // 10 out of 20
				customConcurrencyKey1Active: 3,  // 3 out of 5
				customConcurrencyKey2Active: 10, // 10 out of 10
			},
			knobs: knobs{
				maxRefill:                50,
				accountConcurrencyLimit:  30,
				functionConcurrencyLimit: 20,
				customConcurrencyKey1:    &ck1,
				customConcurrencyKey2:    &ck2,
			},
			expected: expected{
				result: BacklogRefillResult{
					Constraint:        enums.QueueConstraintCustomConcurrencyKey2,
					TotalBacklogCount: 40,
					BacklogCountUntil: 40,
					Capacity:          0,
					Refill:            0,
					Refilled:          0,
				},
				itemsInBacklog:    40,
				itemsInReadyQueue: 0,
			},
		},
		// Should adjust by ready queue
		{
			name: "adjust by ready queue",
			knobs: knobs{
				maxRefill:                50,
				accountConcurrencyLimit:  100,
				functionConcurrencyLimit: 20,
			},
			currentValues: currentValues{
				itemsInBacklog: 40,
				accountActive:  25, // 25 out of 100
				functionActive: 15, // 15 out of 20
			},
			expected: expected{
				result: BacklogRefillResult{
					Constraint:        enums.QueueConstraintNotLimited,
					TotalBacklogCount: 40,
					BacklogCountUntil: 40,
					Capacity:          5,
					Refill:            5,
					Refilled:          5,
				},
				itemsInBacklog:    35,
				itemsInReadyQueue: 5,
			},
		},
		{
			name: "with 'full' ready queue, no items will be refilled",
			knobs: knobs{
				maxRefill:                50,
				accountConcurrencyLimit:  100,
				functionConcurrencyLimit: 20,
			},
			currentValues: currentValues{
				itemsInBacklog: 40,
				// these items could be for any key, but we assume the ready queue will be cleared out asap
				accountActive:  30, // 30 out of 100
				functionActive: 20, // 20 out of 20
			},
			expected: expected{
				result: BacklogRefillResult{
					Constraint:        enums.QueueConstraintFunctionConcurrency,
					TotalBacklogCount: 40,
					BacklogCountUntil: 40,
					Capacity:          0,
					Refill:            0,
					Refilled:          0,
				},
				itemsInBacklog:    40,
				itemsInReadyQueue: 0,
			},
		},
		{
			name: "move entire backlog, if possible",
			knobs: knobs{
				maxRefill:                50,
				accountConcurrencyLimit:  100,
				functionConcurrencyLimit: 100,
			},
			currentValues: currentValues{
				itemsInBacklog: 40,
				accountActive:  25, // 25 out of 100
			},
			expected: expected{
				result: BacklogRefillResult{
					Constraint:        enums.QueueConstraintNotLimited,
					TotalBacklogCount: 40, // would move 40
					BacklogCountUntil: 40,
					Capacity:          75,
					Refill:            40,
					Refilled:          40,
				},
				itemsInBacklog:    0,
				itemsInReadyQueue: 40,
			},
		},
		// Throttle allow
		{
			name: "throttle allow",
			currentValues: currentValues{
				itemsInBacklog:            10,
				throttleUsageWithinPeriod: 20,
			},
			knobs: knobs{
				throttle:    throttle,
				isStartItem: true,
			},
			expected: expected{
				itemsInBacklog:    0,
				itemsInReadyQueue: 10,
				result: BacklogRefillResult{
					Constraint:        enums.QueueConstraintNotLimited,
					TotalBacklogCount: 10,
					BacklogCountUntil: 10,
					Capacity:          90,
					Refill:            10,
					Refilled:          10,
				},
			},
		},
		// Throttle deny
		{
			name: "throttle deny",
			currentValues: currentValues{
				itemsInBacklog:            10,
				throttleUsageWithinPeriod: 110,
			},
			knobs: knobs{
				throttle:    throttle,
				isStartItem: true,
			},
			expected: expected{
				itemsInBacklog:    10,
				itemsInReadyQueue: 0,
				result: BacklogRefillResult{
					Constraint:        enums.QueueConstraintThrottle,
					TotalBacklogCount: 10,
					BacklogCountUntil: 10,
					Capacity:          0,
					Refill:            0,
					Refilled:          0,
				},
				retryAt: 6 * time.Minute, // expect GCRA to allow the next item after 6m
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			r := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{
				InitAddress:  []string{r.Addr()},
				DisableCache: true,
			})
			require.NoError(t, err)
			defer rc.Close()

			ctx := context.Background()

			defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
			kg := defaultShard.RedisClient.kg

			clock := clockwork.NewFakeClock()

			testLifecycles := newTestLifecycleListener()

			enqueueToBacklog := true
			q := NewQueue(
				defaultShard,
				WithClock(clock),
				WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
					return enqueueToBacklog
				}),
				WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
					return true
				}),
				WithRunMode(QueueRunMode{
					Sequential:                        true,
					Scavenger:                         true,
					Partition:                         true,
					Account:                           true,
					AccountWeight:                     85,
					ShadowPartition:                   true,
					AccountShadowPartition:            true,
					AccountShadowPartitionWeight:      85,
					ShadowContinuations:               true,
					ShadowContinuationSkipProbability: 0,
					NormalizePartition:                true,
				}),
				WithBacklogRefillLimit(int64(testCase.knobs.maxRefill)),
				WithConcurrencyLimitGetter(func(ctx context.Context, p QueuePartition) PartitionConcurrencyLimits {
					return PartitionConcurrencyLimits{
						AccountLimit:   testCase.knobs.accountConcurrencyLimit,
						FunctionLimit:  testCase.knobs.functionConcurrencyLimit,
						CustomKeyLimit: 0,
					}
				}),
				WithCustomConcurrencyKeyLimitRefresher(func(ctx context.Context, i osqueue.QueueItem) []state.CustomConcurrency {
					return i.Data.GetConcurrencyKeys()
				}),
				WithSystemConcurrencyLimitGetter(func(ctx context.Context, p QueuePartition) SystemPartitionConcurrencyLimits {
					return SystemPartitionConcurrencyLimits{
						GlobalLimit:    789,
						PartitionLimit: 678,
					}
				}),
				WithQueueLifecycles(testLifecycles),
			)

			addItem := func(id string, identifier state.Identifier, at time.Time) osqueue.QueueItem {
				kind := osqueue.KindEdge
				if testCase.knobs.isStartItem {
					kind = osqueue.KindStart
				}

				var customConc []state.CustomConcurrency
				if testCase.knobs.customConcurrencyKey1 != nil {
					customConc = append(customConc, *testCase.knobs.customConcurrencyKey1)
				}

				if testCase.knobs.customConcurrencyKey2 != nil {
					customConc = append(customConc, *testCase.knobs.customConcurrencyKey2)
				}

				item := osqueue.QueueItem{
					ID:          id,
					FunctionID:  identifier.WorkflowID,
					WorkspaceID: identifier.WorkspaceID,
					Data: osqueue.Item{
						WorkspaceID:           identifier.WorkspaceID,
						Kind:                  kind,
						Identifier:            identifier,
						QueueName:             nil,
						Throttle:              testCase.knobs.throttle,
						CustomConcurrencyKeys: customConc,
					},
					QueueName: nil,
				}

				qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
				require.NoError(t, err)

				return qi
			}
			at := clock.Now()

			// Prepare backlog
			qi1 := addItem("test0", state.Identifier{
				AccountID:   accountID1,
				WorkspaceID: envID1,
				WorkflowID:  fnID1,
			}, at)

			if testCase.currentValues.itemsInBacklog > 1 {
				for i := 1; i < testCase.currentValues.itemsInBacklog; i++ {
					addItem(fmt.Sprintf("test%d", i), state.Identifier{
						AccountID:   accountID1,
						WorkspaceID: envID1,
						WorkflowID:  fnID1,
					}, at)
				}
			}

			backlog := q.ItemBacklog(ctx, qi1)
			shadowPart := q.ItemShadowPartition(ctx, qi1)

			if testCase.knobs.danglingItemsInBacklog > 0 {
				for i := 1; i <= testCase.knobs.danglingItemsInBacklog; i++ {
					_, err = r.ZAdd(kg.BacklogSet(backlog.BacklogID), float64(at.UnixMilli()), fmt.Sprintf("dangling%d", i))
					require.NoError(t, err)
				}
			}

			if testCase.currentValues.accountActive > 0 {
				for i := 1; i <= testCase.currentValues.accountActive; i++ {
					key := kg.ActiveSet("account", accountID1.String())
					_, err = r.SAdd(key, fmt.Sprintf("item%d", i))
					require.NoError(t, err)
				}
			}

			if testCase.currentValues.functionActive > 0 {
				for i := 1; i <= testCase.currentValues.functionActive; i++ {
					key := kg.ActiveSet("p", fnID1.String())
					_, err = r.SAdd(key, fmt.Sprintf("item%d", i))
					require.NoError(t, err)
				}
			}

			if testCase.currentValues.customConcurrencyKey1Active > 0 {
				for i := 1; i <= testCase.currentValues.customConcurrencyKey1Active; i++ {
					key := kg.ActiveSet("custom", testCase.knobs.customConcurrencyKey1.Key)
					_, err = r.SAdd(key, fmt.Sprintf("item%d", i))
					require.NoError(t, err)
				}
			}

			if testCase.currentValues.customConcurrencyKey2Active > 0 {
				for i := 1; i <= testCase.currentValues.customConcurrencyKey2Active; i++ {
					key := kg.ActiveSet("custom", testCase.knobs.customConcurrencyKey2.Key)
					_, err = r.SAdd(key, fmt.Sprintf("item%d", i))
					require.NoError(t, err)
				}
			}

			testThrottle := testCase.knobs.throttle
			if testThrottle != nil {
				runGCRAScript := func(t *testing.T, rc rueidis.Client, key string, now time.Time, period time.Duration, limit, burst, capacity int) (int, time.Time) {
					nowMS := now.UnixMilli()
					args, err := StrSlice([]any{
						key,
						nowMS,
						limit,
						burst,
						period.Milliseconds(),
						capacity,
					})
					require.NoError(t, err)

					res, err := scripts["test/gcra_capacity"].Exec(t.Context(), rc, []string{}, args).ToAny()
					require.NoError(t, err)

					capacityAndRetry, ok := res.([]any)
					require.True(t, ok)

					statusOrCapacity, ok := capacityAndRetry[0].(int64)
					require.True(t, ok)

					var retryAt time.Time
					retryAtMillis, ok := capacityAndRetry[1].(int64)
					require.True(t, ok)

					if retryAtMillis > nowMS {
						retryAt = time.UnixMilli(retryAtMillis)
					}

					switch statusOrCapacity {
					case -1:
						return 0, retryAt
					default:
						return int(statusOrCapacity), retryAt
					}
				}

				// Reduce throttle capacity
				runGCRAScript(
					t,
					rc,
					testThrottle.Key,
					at,
					time.Duration(testThrottle.Period)*time.Second,
					testThrottle.Limit,
					testThrottle.Burst,
					testCase.currentValues.throttleUsageWithinPeriod,
				)
			}

			refillUntil := at.Add(time.Minute)

			logKeyValues := func() {
				fmt.Println("all keys:")
				fmt.Println(r.Dump())
			}

			logKeyValues()

			constraints := &PartitionConstraintConfig{
				Concurrency: ShadowPartitionConcurrency{
					AccountConcurrency:  testCase.knobs.accountConcurrencyLimit,
					FunctionConcurrency: testCase.knobs.functionConcurrencyLimit,
				},
			}

			if testCase.knobs.customConcurrencyKey1 != nil {
				scope, _, _, _ := testCase.knobs.customConcurrencyKey1.ParseKey()
				constraints.Concurrency.CustomConcurrencyKeys = append(constraints.Concurrency.CustomConcurrencyKeys,
					CustomConcurrencyLimit{
						Mode:                enums.ConcurrencyModeStep,
						Scope:               scope,
						HashedKeyExpression: testCase.knobs.customConcurrencyKey1.Hash,
						Limit:               testCase.knobs.customConcurrencyKey1.Limit,
					})
			}

			if testCase.knobs.customConcurrencyKey2 != nil {
				scope, _, _, _ := testCase.knobs.customConcurrencyKey2.ParseKey()
				constraints.Concurrency.CustomConcurrencyKeys = append(constraints.Concurrency.CustomConcurrencyKeys,
					CustomConcurrencyLimit{
						Mode:                enums.ConcurrencyModeStep,
						Scope:               scope,
						HashedKeyExpression: testCase.knobs.customConcurrencyKey2.Hash,
						Limit:               testCase.knobs.customConcurrencyKey2.Limit,
					})
			}

			if testCase.knobs.throttle != nil {
				constraints.Throttle = &ShadowPartitionThrottle{
					ThrottleKeyExpressionHash: testCase.knobs.throttle.KeyExpressionHash,
					Limit:                     testCase.knobs.throttle.Limit,
					Burst:                     testCase.knobs.throttle.Burst,
					Period:                    testCase.knobs.throttle.Period,
				}
			}

			res, _, err := q.processShadowPartitionBacklog(ctx, &shadowPart, &backlog, refillUntil, constraints)
			require.NoError(t, err)

			logKeyValues()

			itemsInBacklog, err := rc.Do(ctx, rc.B().Zcount().Key(kg.BacklogSet(backlog.BacklogID)).Min("-inf").Max(fmt.Sprintf("%d", refillUntil.UnixMilli())).Build()).ToInt64()
			require.NoError(t, err)

			itemsInReadyQueue, err := rc.Do(ctx, rc.B().Zcount().Key(kg.PartitionQueueSet(enums.PartitionTypeDefault, shadowPart.PartitionID, "")).Min("-inf").Max(fmt.Sprintf("%d", refillUntil.UnixMilli())).Build()).ToInt64()
			require.NoError(t, err)

			// we do not test refilled items
			require.Equal(t, testCase.expected.result.Refilled, len(res.RefilledItems))
			res.RefilledItems = nil

			if !res.RetryAt.IsZero() {
				require.Greater(t, testCase.expected.retryAt.Milliseconds(), int64(0))
				diff := clock.Now().Add(testCase.expected.retryAt)

				require.WithinDuration(t, diff, res.RetryAt, 10*time.Second)

				res.RetryAt = time.Time{}
			}

			require.Equal(t, testCase.expected.result, *res, "result comparison failed", res, itemsInBacklog, itemsInReadyQueue)

			require.Equal(t, int64(testCase.expected.itemsInBacklog), itemsInBacklog)
			require.Equal(t, int64(testCase.expected.itemsInReadyQueue), itemsInReadyQueue)

			testLifecycles.lock.Lock()
			switch res.Constraint {
			case enums.QueueConstraintAccountConcurrency:
				require.Equal(t, 1, testLifecycles.acctConcurrency[accountID1])
			case enums.QueueConstraintFunctionConcurrency:
				require.Equal(t, 1, testLifecycles.fnConcurrency[fnID1])
			case enums.QueueConstraintCustomConcurrencyKey1:
				require.Equal(t, 1, testLifecycles.ckConcurrency[ck1.Key])
			case enums.QueueConstraintCustomConcurrencyKey2:
				require.Equal(t, 1, testLifecycles.ckConcurrency[ck2.Key])
			default:
			}
			testLifecycles.lock.Unlock()
		})
	}
}

func TestShadowPartitionUpdate(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
	kg := defaultShard.RedisClient.kg

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
	ctx := context.Background()

	accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	type itemConf struct {
		kind            string // osqueue.Kind, default to edge
		throttle        *osqueue.Throttle
		concurrencyKeys []state.CustomConcurrency
	}

	idv2 := sv2.ID{
		FunctionID: fnID,
		Tenant: sv2.Tenant{
			AccountID: accountId,
			EnvID:     wsID,
		},
	}

	// test cases
	tests := []struct {
		name  string
		conf1 itemConf
		conf2 itemConf
	}{
		{
			name: "none to concurrency",
			conf2: itemConf{
				concurrencyKeys: osqueue.GetCustomConcurrencyKeys(
					ctx,
					idv2,
					[]inngest.Concurrency{
						{Limit: 123, Key: util.StrPtr("event.data.customerId")},
					},
					inngestgo.Event{
						Name: "yolo",
						Data: map[string]any{"customerId": 10},
					}.Map(),
				),
			},
		},
		{
			name: "concurrency to none",
			conf1: itemConf{
				concurrencyKeys: osqueue.GetCustomConcurrencyKeys(
					ctx,
					idv2,
					[]inngest.Concurrency{
						{Limit: 123, Key: util.StrPtr("event.data.customerId")},
					},
					inngestgo.Event{
						Name: "yolo",
						Data: map[string]any{"customerId": 10},
					}.Map(),
				),
			},
		},
		//{
		//	name: "change concurrency",
		//	conf1: itemConf{
		//		concurrencyKeys: osqueue.GetCustomConcurrencyKeys(
		//			ctx,
		//			idv2,
		//			[]inngest.Concurrency{
		//				{Limit: 123, Key: util.StrPtr("event.data.customerId")},
		//			},
		//			inngestgo.Event{
		//				Name: "yolo",
		//				Data: map[string]any{"customerId": 10},
		//			}.Map(),
		//		),
		//	},
		//	conf2: itemConf{
		//		concurrencyKeys: osqueue.GetCustomConcurrencyKeys(
		//			ctx,
		//			idv2,
		//			[]inngest.Concurrency{
		//				{Limit: 123, Key: util.StrPtr("event.data.userId")},
		//			},
		//			inngestgo.Event{
		//				Name: "yolo",
		//				Data: map[string]any{"userId": 10},
		//			}.Map(),
		//		),
		//	},
		//},
		{
			name:  "none to throttle",
			conf1: itemConf{kind: osqueue.KindStart},
			conf2: itemConf{
				kind: osqueue.KindStart,
				throttle: osqueue.GetThrottleConfig(
					ctx,
					fnID,
					&inngest.Throttle{
						Limit:  10,
						Period: 30 * time.Second,
						Burst:  2,
						Key:    util.StrPtr("event.data.customerId"),
					},
					inngestgo.Event{
						Name: "hello/world",
						Data: map[string]any{"customerId": 100},
					}.Map(),
				),
			},
		},
		{
			name: "throttle to none",
			conf1: itemConf{
				kind: osqueue.KindStart,
				throttle: osqueue.GetThrottleConfig(
					ctx,
					fnID,
					&inngest.Throttle{
						Limit:  10,
						Period: 30 * time.Second,
						Burst:  2,
						Key:    util.StrPtr("event.data.customerId"),
					},
					inngestgo.Event{
						Name: "hello/world",
						Data: map[string]any{"customerId": 100},
					}.Map(),
				),
			},
		},
		//{
		//	name: "change throttle",
		//	conf1: itemConf{
		//		kind: osqueue.KindStart,
		//		throttle: osqueue.GetThrottleConfig(
		//			ctx,
		//			fnID,
		//			&inngest.Throttle{
		//				Limit:  10,
		//				Period: 30 * time.Second,
		//				Burst:  2,
		//				Key:    util.StrPtr("event.data.customerId"),
		//			},
		//			inngestgo.Event{
		//				Name: "hello/world",
		//				Data: map[string]any{"customerId": 100},
		//			}.Map(),
		//		),
		//	},
		//	conf2: itemConf{
		//		kind: osqueue.KindStart,
		//		throttle: osqueue.GetThrottleConfig(
		//			ctx,
		//			fnID,
		//			&inngest.Throttle{
		//				Limit:  10,
		//				Period: 30 * time.Second,
		//				Burst:  2,
		//				Key:    util.StrPtr("event.data.userId"),
		//			},
		//			inngestgo.Event{
		//				Name: "hello/world",
		//				Data: map[string]any{"userId": 100},
		//			}.Map(),
		//		),
		//	},
		//},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r.FlushAll()
			require.Len(t, r.Keys(), 0)

			// use future timestamp because scores will be bounded to the present
			at := clock.Now().Add(1 * time.Minute)

			//
			// Create initial shadow partition
			//
			kind1 := osqueue.KindEdge
			if tc.conf1.kind != "" {
				kind1 = tc.conf1.kind
			}

			item1 := osqueue.QueueItem{
				ID:          "test",
				FunctionID:  fnID,
				WorkspaceID: wsID,
				Data: osqueue.Item{
					WorkspaceID: wsID,
					Kind:        kind1,
					Identifier: state.Identifier{
						WorkflowID:      fnID,
						AccountID:       accountId,
						WorkspaceID:     wsID,
						WorkflowVersion: 1,
					},
					Throttle:              tc.conf1.throttle,
					CustomConcurrencyKeys: tc.conf1.concurrencyKeys,
				},
			}

			backlog1 := q.ItemBacklog(ctx, item1)
			require.NotEmpty(t, backlog1.BacklogID)
			fmt.Printf("Backlog1: %#v\n", backlog1.Throttle)

			initialShadowPart := q.ItemShadowPartition(ctx, item1)
			require.NotEmpty(t, initialShadowPart.PartitionID)

			if len(tc.conf1.concurrencyKeys) > 0 {
				require.Len(t, backlog1.ConcurrencyKeys, len(tc.conf1.concurrencyKeys))

				hashes := make([]string, len(tc.conf1.concurrencyKeys))
				for i, k := range initialShadowPart.Concurrency.CustomConcurrencyKeys {
					hashes[i] = k.HashedKeyExpression
				}
				for _, k := range backlog1.ConcurrencyKeys {
					require.Contains(t, hashes, k.HashedKeyExpression)
				}
			}
			if tc.conf1.throttle != nil {
				require.NotNil(t, backlog1.Throttle)
				require.Equal(t, initialShadowPart.Throttle.ThrottleKeyExpressionHash, backlog1.Throttle.ThrottleKeyExpressionHash)
			}

			_, err := q.EnqueueItem(ctx, defaultShard, item1, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			// verify shadow partition
			savedPart := QueueShadowPartition{}
			require.NoError(t, json.Unmarshal([]byte(r.HGet(kg.ShadowPartitionMeta(), initialShadowPart.PartitionID)), &savedPart))
			require.Equal(t, initialShadowPart, savedPart)
			require.Equal(t, 1, savedPart.FunctionVersion)

			// verify backlog
			savedBacklog1 := QueueBacklog{}
			require.NoError(t, json.Unmarshal([]byte(r.HGet(kg.BacklogMeta(), backlog1.BacklogID)), &savedBacklog1))
			require.Equal(t, backlog1, savedBacklog1)

			//
			// Test update case
			//
			kind2 := osqueue.KindEdge
			if tc.conf2.throttle != nil {
				kind2 = tc.conf2.kind
			}

			item2 := osqueue.QueueItem{
				ID:          "test2",
				FunctionID:  fnID,
				WorkspaceID: wsID,
				Data: osqueue.Item{
					WorkspaceID: wsID,
					Kind:        kind2,
					Identifier: state.Identifier{
						WorkflowID:      fnID,
						AccountID:       accountId,
						WorkspaceID:     wsID,
						WorkflowVersion: 2,
					},
					Throttle:              tc.conf2.throttle,
					CustomConcurrencyKeys: tc.conf2.concurrencyKeys,
				},
			}

			updatedShadowPart := q.ItemShadowPartition(ctx, item2)
			require.Len(t, updatedShadowPart.Concurrency.CustomConcurrencyKeys, len(tc.conf2.concurrencyKeys))
			require.Equal(t, 2, updatedShadowPart.FunctionVersion)

			backlog2 := q.ItemBacklog(ctx, item2)
			require.NotEmpty(t, backlog2.BacklogID)
			require.NotEqual(t, backlog1, backlog2)

			if len(tc.conf2.concurrencyKeys) > 0 {
				require.Len(t, backlog2.ConcurrencyKeys, len(tc.conf2.concurrencyKeys))

				hashes := make([]string, len(tc.conf2.concurrencyKeys))
				for i, k := range updatedShadowPart.Concurrency.CustomConcurrencyKeys {
					hashes[i] = k.HashedKeyExpression
				}
				for _, k := range backlog2.ConcurrencyKeys {
					require.Contains(t, hashes, k.HashedKeyExpression)
				}
			}
			if tc.conf2.throttle != nil {
				require.NotNil(t, backlog2.Throttle)
				require.Equal(t, updatedShadowPart.Throttle.ThrottleKeyExpressionHash, backlog2.Throttle.ThrottleKeyExpressionHash)
			}

			_, err = q.EnqueueItem(ctx, defaultShard, item2, at.Add(time.Minute), osqueue.EnqueueOpts{})
			require.NoError(t, err)
			fmt.Printf("Backlog2: %#v\n", backlog2.Throttle)

			// verify shadow partition
			savedPart = QueueShadowPartition{}
			require.NoError(t, json.Unmarshal([]byte(r.HGet(kg.ShadowPartitionMeta(), initialShadowPart.PartitionID)), &savedPart))
			require.Equal(t, updatedShadowPart, savedPart)

			// verify backlog
			savedBacklog2 := QueueBacklog{}
			require.NoError(t, json.Unmarshal([]byte(r.HGet(kg.BacklogMeta(), backlog2.BacklogID)), &savedBacklog2))
			require.Equal(t, backlog2, savedBacklog2)

			require.NotEqual(t, updatedShadowPart, initialShadowPart)
			// fmt.Printf("Initial: %#v\n", initialShadowPart.Throttle)
			// fmt.Printf("Updated: %#v\n", updatedShadowPart.Throttle)

			//
			// Ensure shadow partition is not reverted to old version
			//

			item3 := osqueue.QueueItem{
				ID:          "test3",
				FunctionID:  fnID,
				WorkspaceID: wsID,
				Data: osqueue.Item{
					WorkspaceID: wsID,
					Kind:        kind1,
					Identifier: state.Identifier{
						WorkflowID:      fnID,
						AccountID:       accountId,
						WorkspaceID:     wsID,
						WorkflowVersion: 1,
					},
					Throttle:              tc.conf1.throttle,
					CustomConcurrencyKeys: tc.conf1.concurrencyKeys,
				},
			}

			_, err = q.EnqueueItem(ctx, defaultShard, item3, at.Add(2*time.Minute), osqueue.EnqueueOpts{})
			require.NoError(t, err)

			savedPart = QueueShadowPartition{}
			require.NoError(t, json.Unmarshal([]byte(r.HGet(kg.ShadowPartitionMeta(), initialShadowPart.PartitionID)), &savedPart))

			require.Equal(t, updatedShadowPart, savedPart)
			require.Equal(t, 2, savedPart.FunctionVersion)
		})
	}
}

func TestShadowPartitionPointerTimings(t *testing.T) {
	t.Run("multiple spaced out items", func(t *testing.T) {
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		ctx := context.Background()

		defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}

		clock := clockwork.NewFakeClock()

		accountID, wsID, fnID := uuid.New(), uuid.New(), uuid.New()

		kg := defaultShard.RedisClient.kg

		enqueueToBacklog := true
		q := NewQueue(
			defaultShard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithEnqueueSystemPartitionsToBacklog(false),
			WithDisableLeaseChecksForSystemQueues(false),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
			WithRunMode(QueueRunMode{
				Sequential:                        true,
				Scavenger:                         true,
				Partition:                         true,
				Account:                           true,
				AccountWeight:                     85,
				ShadowPartition:                   true,
				AccountShadowPartition:            true,
				AccountShadowPartitionWeight:      85,
				ShadowContinuations:               true,
				ShadowContinuationSkipProbability: 0,
				NormalizePartition:                true,
			}),
			WithBacklogRefillLimit(500),
			WithConcurrencyLimitGetter(func(ctx context.Context, p QueuePartition) PartitionConcurrencyLimits {
				return PartitionConcurrencyLimits{
					AccountLimit:   123,
					FunctionLimit:  45,
					CustomKeyLimit: 0,
				}
			}),
			WithCustomConcurrencyKeyLimitRefresher(func(ctx context.Context, i osqueue.QueueItem) []state.CustomConcurrency {
				return i.Data.GetConcurrencyKeys()
			}),
			WithSystemConcurrencyLimitGetter(func(ctx context.Context, p QueuePartition) SystemPartitionConcurrencyLimits {
				return SystemPartitionConcurrencyLimits{
					GlobalLimit:    789,
					PartitionLimit: 678,
				}
			}),
		)

		addItem := func(id string, identifier state.Identifier, at time.Time) osqueue.QueueItem {
			item := osqueue.QueueItem{
				ID:          id,
				FunctionID:  identifier.WorkflowID,
				WorkspaceID: identifier.WorkspaceID,
				Data: osqueue.Item{
					WorkspaceID:           identifier.WorkspaceID,
					Kind:                  osqueue.KindEdge,
					Identifier:            identifier,
					QueueName:             nil,
					Throttle:              nil,
					CustomConcurrencyKeys: nil,
				},
				QueueName: nil,
			}

			qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			return qi
		}

		now := clock.Now().Truncate(time.Second)

		numItems := 20
		items := make([]osqueue.QueueItem, numItems)
		for i := 1; i <= numItems; i++ {
			items[i-1] = addItem(fmt.Sprintf("item%d", i), state.Identifier{
				AccountID:   accountID,
				WorkspaceID: wsID,
				WorkflowID:  fnID,
			}, now.Add(time.Duration(i)*time.Second))
		}

		backlog := q.ItemBacklog(ctx, items[0])
		shadowPart := q.ItemShadowPartition(ctx, items[0])

		// Pointer should be earliest item
		require.Equal(t, now.Add(time.Second).UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), items[0].ID)))
		require.Equal(t, now.Add(time.Second).UnixMilli(), int64(score(t, r, kg.ShadowPartitionSet(shadowPart.PartitionID), backlog.BacklogID)))
		require.Equal(t, now.Add(time.Second).UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, now.Add(time.Second).UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
		require.Equal(t, now.Add(time.Second).UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))

		until := now.Add(PartitionLookahead)
		peeked, totalUntil, err := q.ShadowPartitionPeek(ctx, &shadowPart, false, until, 100)
		require.NoError(t, err)

		require.Equal(t, 1, totalUntil)
		require.Len(t, peeked, 1)
		require.Equal(t, backlog, *peeked[0])

		for i := range numItems {
			itemAt := now.Add(time.Duration(i+1) * time.Second)
			refillUntil := itemAt
			res, err := q.BacklogRefill(ctx, &backlog, &shadowPart, refillUntil, &PartitionConstraintConfig{
				Concurrency: ShadowPartitionConcurrency{
					AccountConcurrency:  123,
					FunctionConcurrency: 45,
				},
			})
			require.NoError(t, err)

			require.Equal(t, numItems-i, res.TotalBacklogCount)
			require.Equal(t, 1, res.BacklogCountUntil)
			require.Equal(t, 1, res.Refill)
			require.Equal(t, 1, res.Refilled)

			if i == numItems-1 {
				break
			}

			// Pointer should be next earliest time
			nextItemAt := now.Add(time.Duration(i+2) * time.Second)
			require.Equal(t, nextItemAt.UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), items[i+1].ID)))
			require.Equal(t, nextItemAt.UnixMilli(), int64(score(t, r, kg.ShadowPartitionSet(shadowPart.PartitionID), backlog.BacklogID)))
			require.Equal(t, nextItemAt.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
			require.Equal(t, nextItemAt.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
			require.Equal(t, nextItemAt.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))
		}

		require.False(t, r.Exists(kg.BacklogSet(backlog.BacklogID)))
		require.False(t, r.Exists(kg.ShadowPartitionSet(shadowPart.PartitionID)))
	})

	t.Run("sleep should work", func(t *testing.T) {
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		ctx := context.Background()

		defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Second))

		accountID, wsID, fnID := uuid.New(), uuid.New(), uuid.New()

		kg := defaultShard.RedisClient.kg

		enqueueToBacklog := true
		q := NewQueue(
			defaultShard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithEnqueueSystemPartitionsToBacklog(false),
			WithDisableLeaseChecksForSystemQueues(false),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
			WithRunMode(QueueRunMode{
				Sequential:                        true,
				Scavenger:                         true,
				Partition:                         true,
				Account:                           true,
				AccountWeight:                     85,
				ShadowPartition:                   true,
				AccountShadowPartition:            true,
				AccountShadowPartitionWeight:      85,
				ShadowContinuations:               true,
				ShadowContinuationSkipProbability: 0,
				NormalizePartition:                true,
			}),
			WithBacklogRefillLimit(500),
			WithConcurrencyLimitGetter(func(ctx context.Context, p QueuePartition) PartitionConcurrencyLimits {
				return PartitionConcurrencyLimits{
					AccountLimit:   123,
					FunctionLimit:  45,
					CustomKeyLimit: 0,
				}
			}),
			WithCustomConcurrencyKeyLimitRefresher(func(ctx context.Context, i osqueue.QueueItem) []state.CustomConcurrency {
				return i.Data.GetConcurrencyKeys()
			}),
			WithSystemConcurrencyLimitGetter(func(ctx context.Context, p QueuePartition) SystemPartitionConcurrencyLimits {
				return SystemPartitionConcurrencyLimits{
					GlobalLimit:    789,
					PartitionLimit: 678,
				}
			}),
		)

		now := clock.Now()

		item := osqueue.QueueItem{
			FunctionID:  fnID,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID:  fnID,
					AccountID:   accountID,
					WorkspaceID: wsID,
				},
				QueueName: nil,
			},
			QueueName: nil,
		}

		sleepUntil := now.Add(2 * time.Second)
		qi, err := q.EnqueueItem(ctx, defaultShard, item, sleepUntil, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		backlog := q.ItemBacklog(ctx, qi)
		shadowPart := q.ItemShadowPartition(ctx, qi)

		require.Equal(t, sleepUntil.UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID)))
		require.Equal(t, sleepUntil.UnixMilli(), int64(score(t, r, kg.ShadowPartitionSet(shadowPart.PartitionID), backlog.BacklogID)))
		require.Equal(t, sleepUntil.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, sleepUntil.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
		require.Equal(t, sleepUntil.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))

		until := now.Add(time.Second)
		peeked, totalUntil, err := q.ShadowPartitionPeek(ctx, &shadowPart, false, until, 100)
		require.NoError(t, err)

		require.Equal(t, 0, totalUntil)
		require.Len(t, peeked, 0)

		until = now.Add(2 * time.Second)
		peeked, totalUntil, err = q.ShadowPartitionPeek(ctx, &shadowPart, false, until, 100)
		require.NoError(t, err)

		require.Equal(t, 1, totalUntil)
		require.Len(t, peeked, 1)
		require.Equal(t, backlog, *peeked[0])
	})
}

func TestConstraintLifecycleReporting(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	ctx := context.Background()

	defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}

	clock := clockwork.NewFakeClock()

	testLifecycles := newTestLifecycleListener()

	enqueueToBacklog := true
	q := NewQueue(
		defaultShard,
		WithClock(clock),
		WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
			return enqueueToBacklog
		}),
		WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
			return true
		}),
		WithRunMode(QueueRunMode{
			Sequential:                        true,
			Scavenger:                         true,
			Partition:                         true,
			Account:                           true,
			AccountWeight:                     85,
			ShadowPartition:                   true,
			AccountShadowPartition:            true,
			AccountShadowPartitionWeight:      85,
			ShadowContinuations:               true,
			ShadowContinuationSkipProbability: 0,
			NormalizePartition:                true,
		}),
		WithBacklogRefillLimit(100),
		WithConcurrencyLimitGetter(func(ctx context.Context, p QueuePartition) PartitionConcurrencyLimits {
			return PartitionConcurrencyLimits{
				AccountLimit:   1,
				FunctionLimit:  1,
				CustomKeyLimit: 0,
			}
		}),
		WithCustomConcurrencyKeyLimitRefresher(func(ctx context.Context, i osqueue.QueueItem) []state.CustomConcurrency {
			return i.Data.GetConcurrencyKeys()
		}),
		WithSystemConcurrencyLimitGetter(func(ctx context.Context, p QueuePartition) SystemPartitionConcurrencyLimits {
			return SystemPartitionConcurrencyLimits{
				GlobalLimit:    789,
				PartitionLimit: 678,
			}
		}),
		WithQueueLifecycles(testLifecycles),
	)

	fnID1, accountID1, envID1 := uuid.New(), uuid.New(), uuid.New()
	fnID2 := uuid.New()

	addItem := func(id string, identifier state.Identifier, at time.Time) osqueue.QueueItem {
		item := osqueue.QueueItem{
			ID:          id,
			FunctionID:  identifier.WorkflowID,
			WorkspaceID: identifier.WorkspaceID,
			Data: osqueue.Item{
				WorkspaceID:           identifier.WorkspaceID,
				Kind:                  osqueue.KindEdge,
				Identifier:            identifier,
				QueueName:             nil,
				Throttle:              nil,
				CustomConcurrencyKeys: nil,
			},
			QueueName: nil,
		}

		qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		return qi
	}
	at := clock.Now()

	constraints := PartitionConstraintConfig{
		Concurrency: ShadowPartitionConcurrency{
			AccountConcurrency:  1,
			FunctionConcurrency: 1,
		},
	}

	itemA1 := addItem("test1", state.Identifier{
		AccountID:   accountID1,
		WorkspaceID: envID1,
		WorkflowID:  fnID1,
	}, at)

	sp1 := q.ItemShadowPartition(ctx, itemA1)
	b1 := q.ItemBacklog(ctx, itemA1)

	require.Equal(t, 1, sp1.Concurrency.FunctionConcurrency)
	require.Equal(t, 1, sp1.Concurrency.AccountConcurrency)

	res, _, err := q.processShadowPartitionBacklog(ctx, &sp1, &b1, at.Add(time.Minute), &constraints)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, enums.QueueConstraintNotLimited, res.Constraint)

	testLifecycles.lock.Lock()
	require.Equal(t, 0, testLifecycles.acctConcurrency[accountID1])
	assert.Equal(t, 0, testLifecycles.fnConcurrency[fnID1])
	assert.Equal(t, 0, testLifecycles.fnConcurrency[fnID2])
	testLifecycles.lock.Unlock()

	_ = addItem("test2", state.Identifier{
		AccountID:   accountID1,
		WorkspaceID: envID1,
		WorkflowID:  fnID1,
	}, at)

	res, _, err = q.processShadowPartitionBacklog(ctx, &sp1, &b1, at.Add(time.Minute), &constraints)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, enums.QueueConstraintFunctionConcurrency, res.Constraint)
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		testLifecycles.lock.Lock()
		assert.Equal(t, 0, testLifecycles.acctConcurrency[accountID1])
		assert.Equal(t, 1, testLifecycles.fnConcurrency[fnID1])
		assert.Equal(t, 0, testLifecycles.fnConcurrency[fnID2])
		testLifecycles.lock.Unlock()
	}, 1*time.Second, 100*time.Millisecond)

	itemB1 := addItem("test3", state.Identifier{
		WorkflowID:  fnID2,
		AccountID:   accountID1,
		WorkspaceID: envID1,
	}, at)

	b2 := q.ItemBacklog(ctx, itemB1)

	sp2 := q.ItemShadowPartition(ctx, itemB1)

	res, _, err = q.processShadowPartitionBacklog(ctx, &sp2, &b2, at.Add(time.Minute), &constraints)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, enums.QueueConstraintAccountConcurrency, res.Constraint)
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		testLifecycles.lock.Lock()
		assert.Equal(t, 1, testLifecycles.acctConcurrency[accountID1])
		assert.Equal(t, 1, testLifecycles.fnConcurrency[fnID1])
		assert.Equal(t, 0, testLifecycles.fnConcurrency[fnID2])
		testLifecycles.lock.Unlock()
	}, 1*time.Second, 100*time.Millisecond)
}
