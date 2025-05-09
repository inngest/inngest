package redis_state

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
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
		WithAllowSystemKeyQueues(func(ctx context.Context) bool {
			return true
		}),
		WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
			return false
		}),
		WithDisableSystemQueueLeaseChecks(func(ctx context.Context) bool {
			return false
		}),
		WithClock(clock),
	)
	ctx := context.Background()

	accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

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

		res, err := q.BacklogRefill(ctx, &expectedBacklog, &shadowPartition, clock.Now())
		require.NoError(t, err)

		require.Equal(t, 1, res.Refilled)
		require.Equal(t, enums.QueueConstraintNotLimited, res.Constraint)

		require.False(t, hasMember(t, r, kg.BacklogSet(expectedBacklog.BacklogID), qi.ID))

		require.True(t, hasMember(t, r, kg.PartitionQueueSet(enums.PartitionTypeDefault, fnID.String(), ""), qi.ID))
		require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.PartitionQueueSet(enums.PartitionTypeDefault, fnID.String(), ""), qi.ID)))

		kg.ShadowPartitionSet(shadowPartition.PartitionID)
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

		_, err = r.ZAdd(kg.BacklogSet(expectedBacklog.BacklogID), float64(at.Unix()), "missing-1")
		require.NoError(t, err)

		_, err = r.ZAdd(kg.BacklogSet(expectedBacklog.BacklogID), float64(at.Unix()), "missing-2")
		require.NoError(t, err)

		_, err = r.ZAdd(kg.BacklogSet(expectedBacklog.BacklogID), float64(at.Unix()), "missing-3")
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

		res, err := q.BacklogRefill(ctx, &expectedBacklog, &shadowPartition, clock.Now())
		require.NoError(t, err)

		require.Equal(t, 1, res.Refilled)
		require.Equal(t, enums.QueueConstraintNotLimited, res.Constraint)

		require.False(t, hasMember(t, r, kg.BacklogSet(expectedBacklog.BacklogID), qi.ID))

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
			WithAllowSystemKeyQueues(func(ctx context.Context) bool {
				return enqueueToBacklog
			}),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
			WithDisableSystemQueueLeaseChecks(func(ctx context.Context) bool {
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

		fnID1, accountID1, envID1 := uuid.New(), uuid.New(), uuid.New()

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
			AccountID:   accountID1,
			WorkspaceID: envID1,
			WorkflowID:  fnID1,
		}, at)

		addItem("test2", state.Identifier{
			AccountID:   accountID1,
			WorkspaceID: envID1,
			WorkflowID:  fnID1,
		}, at)

		backlog := q.ItemBacklog(ctx, qi1)
		shadowPart := q.ItemShadowPartition(ctx, qi1)

		refillUntil := at.Add(time.Minute)

		res, err := q.BacklogRefill(ctx, &backlog, &shadowPart, refillUntil)
		require.NoError(t, err)

		require.Equal(t, 2, res.TotalBacklogCount)
		require.Equal(t, 45, res.Capacity) // limit by function concurrency
		require.Equal(t, 1, res.Refill)
		require.Equal(t, 1, res.Refilled)
		require.Equal(t, enums.QueueConstraintNotLimited, res.Constraint)
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
		WithAllowSystemKeyQueues(func(ctx context.Context) bool {
			return enqueueToBacklog
		}),
		WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
			return true
		}),
		WithDisableSystemQueueLeaseChecks(func(ctx context.Context) bool {
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
		require.NoError(t, json.Unmarshal([]byte(r.HGet(kg.ShadowPartitionMeta(), shadowPart.PartitionID)), &leasedPart))

		require.NotNil(t, leasedPart.LeaseID)
		require.Equal(t, *leaseID, *leasedPart.LeaseID)

		// Expect shadow partition to be pushed back
		require.Equal(t, expectedLeaseExpiry.Unix(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.Unix(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.Unix(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))
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
		require.Equal(t, expectedLeaseExpiry.Unix(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.Unix(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.Unix(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))
	})

	t.Run("return lease should work", func(t *testing.T) {
		// Simulate 2s have passed
		clock.Advance(2 * time.Second)
		r.FastForward(2 * time.Second)

		// Simulate next backlog item in shadow partition
		nextBacklogAt := clock.Now().Add(3 * time.Hour)
		_, err := r.ZAdd(kg.ShadowPartitionSet(shadowPart.PartitionID), float64(nextBacklogAt.UnixMilli()), "backlog-test")
		require.NoError(t, err)

		err = q.ShadowPartitionRequeue(ctx, shadowPart, *leaseID, nil)
		require.NoError(t, err)

		leasedPart := QueueShadowPartition{}
		require.NoError(t, json.Unmarshal([]byte(r.HGet(kg.ShadowPartitionMeta(), shadowPart.PartitionID)), &leasedPart))

		require.Nil(t, leasedPart.LeaseID)

		// Expect shadow partition to be pushed back
		require.Equal(t, nextBacklogAt.Unix(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, nextBacklogAt.Unix(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
		require.Equal(t, nextBacklogAt.Unix(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))
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

		require.Equal(t, expectedLeaseExpiry.Unix(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.Unix(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.Unix(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))

		err = q.ShadowPartitionRequeue(ctx, shadowPart, *leaseID, nil)
		require.NoError(t, err)

		leasedPart := QueueShadowPartition{}
		require.NoError(t, json.Unmarshal([]byte(r.HGet(kg.ShadowPartitionMeta(), shadowPart.PartitionID)), &leasedPart))

		require.Nil(t, leasedPart.LeaseID)

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

		require.Equal(t, expectedLeaseExpiry.Unix(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.Unix(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.Unix(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))

		forceRequeueAt := time.Now().Add(32 * time.Minute)
		err = q.ShadowPartitionRequeue(ctx, shadowPart, *leaseID, &forceRequeueAt)
		require.NoError(t, err)

		leasedPart := QueueShadowPartition{}
		require.NoError(t, json.Unmarshal([]byte(r.HGet(kg.ShadowPartitionMeta(), shadowPart.PartitionID)), &leasedPart))

		require.Nil(t, leasedPart.LeaseID)

		// Expect pointers to be aligned with forced time
		require.Equal(t, forceRequeueAt.Unix(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, forceRequeueAt.Unix(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
		require.Equal(t, forceRequeueAt.Unix(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))
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

		require.Equal(t, expectedLeaseExpiry.Unix(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.Unix(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.Unix(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))

		forceRequeueAt := time.Now().Add(32 * time.Minute)
		err = q.ShadowPartitionRequeue(ctx, shadowPart, *leaseID, &forceRequeueAt)
		require.NoError(t, err)

		leasedPart := QueueShadowPartition{}
		require.NoError(t, json.Unmarshal([]byte(r.HGet(kg.ShadowPartitionMeta(), shadowPart.PartitionID)), &leasedPart))

		require.Nil(t, leasedPart.LeaseID)

		// Expect pointers to be aligned with next backlog item
		require.Equal(t, nextBacklogAt.Unix(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, nextBacklogAt.Unix(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
		require.Equal(t, nextBacklogAt.Unix(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))
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
		WithAllowSystemKeyQueues(func(ctx context.Context) bool {
			return enqueueToBacklog
		}),
		WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
			return true
		}),
		WithDisableSystemQueueLeaseChecks(func(ctx context.Context) bool {
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

	clock := clockwork.NewFakeClock()

	enqueueToBacklog := true
	q := NewQueue(
		defaultShard,
		WithClock(clock),
		WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
			return enqueueToBacklog
		}),
		WithAllowSystemKeyQueues(func(ctx context.Context) bool {
			return enqueueToBacklog
		}),
		WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
			return true
		}),
		WithDisableSystemQueueLeaseChecks(func(ctx context.Context) bool {
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

	qspc := make(chan shadowPartitionChanMsg, 1)

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

		select {
		case msg := <-qspc:
			require.Equal(t, sp1, *msg.sp)
			require.Equal(t, uint(1), msg.continuationCount)
		default:
			require.Fail(t, "expected message to be added")
		}
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
}
