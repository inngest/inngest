package redis_state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getItemIDsFromBacklog is a helper function to peek items from a backlog and extract their IDs
func getItemIDsFromBacklog(ctx context.Context, q osqueue.ShardOperations, backlog *osqueue.QueueBacklog, refillUntil time.Time, limit int64) ([]string, error) {
	res, err := q.BacklogPeek(
		ctx,
		backlog,
		time.Time{},
		refillUntil,
		limit,
		osqueue.WithPeekOptIgnoreCleanup(),
	)
	if err != nil {
		return nil, err
	}

	itemIDs := make([]string, len(res.Items))
	for i, item := range res.Items {
		itemIDs[i] = item.ID
	}
	return itemIDs, nil
}

func TestQueueRefillBacklog(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	_, shard := newQueue(
		t, rc,
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
			return true
		}),
		osqueue.WithClock(clock),
	)
	kg := shard.Client().kg
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

	qi, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
	require.NoError(t, err)

	expectedBacklog := osqueue.ItemBacklog(ctx, item)
	require.NotEmpty(t, expectedBacklog.BacklogID)

	shadowPartition := osqueue.ItemShadowPartition(ctx, item)
	require.NotEmpty(t, shadowPartition.PartitionID)

	t.Run("should find backlog with peek", func(t *testing.T) {
		backlogs, totalCount, err := shard.ShadowPartitionPeek(ctx, &shadowPartition, true, at.Add(time.Minute), 10)
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

		// Get items to refill from backlog
		itemIDs, err := getItemIDsFromBacklog(ctx, shard, &expectedBacklog, clock.Now(), 1000)
		require.NoError(t, err)

		res, err := shard.BacklogRefill(ctx, &expectedBacklog, &shadowPartition, clock.Now(), itemIDs)
		require.NoError(t, err)

		require.Equal(t, 1, len(res.RefilledItems))

		require.False(t, hasMember(t, r, kg.BacklogSet(expectedBacklog.BacklogID), qi.ID))

		require.True(t, hasMember(t, r, kg.PartitionQueueSet(enums.PartitionTypeDefault, fnID.String(), ""), qi.ID))
		require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.PartitionQueueSet(enums.PartitionTypeDefault, fnID.String(), ""), qi.ID)))

		require.Equal(t, at.Unix(), int64(score(t, r, kg.GlobalPartitionIndex(), fnID.String())))
		require.Equal(t, at.Unix(), int64(score(t, r, kg.GlobalAccountIndex(), accountId.String())))
		require.Equal(t, at.Unix(), int64(score(t, r, kg.AccountPartitionIndex(accountId), fnID.String())))
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

		qi, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		expectedBacklog := osqueue.ItemBacklog(ctx, item)
		require.NotEmpty(t, expectedBacklog.BacklogID)

		_, err = r.ZAdd(kg.BacklogSet(expectedBacklog.BacklogID), float64(at.UnixMilli()), "missing-1")
		require.NoError(t, err)

		_, err = r.ZAdd(kg.BacklogSet(expectedBacklog.BacklogID), float64(at.UnixMilli()), "missing-2")
		require.NoError(t, err)

		_, err = r.ZAdd(kg.BacklogSet(expectedBacklog.BacklogID), float64(at.UnixMilli()), "missing-3")
		require.NoError(t, err)

		shadowPartition := osqueue.ItemShadowPartition(ctx, item)
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

		// Get items to refill from backlog
		itemIDs, err := getItemIDsFromBacklog(ctx, shard, &expectedBacklog, clock.Now(), 1000)
		require.NoError(t, err)

		// Peek will not return missing items, but also don't delete them due to WithPeekOptIgnoreCleanup
		require.Len(t, itemIDs, 1)
		// Simulate peek returned missing items
		itemIDs = append(itemIDs, "missing-1", "missing-2", "missing-3")

		res, err := shard.BacklogRefill(ctx, &expectedBacklog, &shadowPartition, clock.Now(), itemIDs)
		require.NoError(t, err)

		require.Equal(t, 1, len(res.RefilledItems))

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

		clock := clockwork.NewFakeClock()

		enqueueToBacklog := true
		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			osqueue.WithRunMode(osqueue.QueueRunMode{
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
			osqueue.WithBacklogRefillLimit(1),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return osqueue.PartitionConstraintConfig{
					Concurrency: osqueue.PartitionConcurrency{
						AccountConcurrency:  123,
						FunctionConcurrency: 45,
						SystemConcurrency:   678,
					},
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

			qi, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
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

		addItem("test3", state.Identifier{
			AccountID:   accountId,
			WorkspaceID: wsID,
			WorkflowID:  fnID,
		}, at.Add(time.Second))

		backlog := osqueue.ItemBacklog(ctx, qi1)
		shadowPart := osqueue.ItemShadowPartition(ctx, qi1)

		refillUntil := at.Add(time.Minute)

		// Get items to refill from backlog
		itemIDs, err := getItemIDsFromBacklog(ctx, shard, &backlog, refillUntil, 1000)
		require.NoError(t, err)
		require.Len(t, itemIDs, 3)

		// Only include first 2 items
		itemIDs = itemIDs[:2]

		res, err := shard.BacklogRefill(ctx, &backlog, &shadowPart, refillUntil, itemIDs)
		require.NoError(t, err)

		require.Equal(t, 3, res.TotalBacklogCount)
		require.Equal(t, 3, res.BacklogCountUntil)
		require.Equal(t, 2, len(res.RefilledItems))
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

		clock := clockwork.NewFakeClock()

		enqueueToBacklog := true
		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			osqueue.WithRunMode(osqueue.QueueRunMode{
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
			osqueue.WithBacklogRefillLimit(500),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return osqueue.PartitionConstraintConfig{
					Concurrency: osqueue.PartitionConcurrency{
						AccountConcurrency:  123,
						FunctionConcurrency: 45,
						SystemConcurrency:   678,
					},
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

			qi, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
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

		backlog := osqueue.ItemBacklog(ctx, qi1)
		shadowPart := osqueue.ItemShadowPartition(ctx, qi1)

		require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), qi1.ID)))
		require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.ShadowPartitionSet(shadowPart.PartitionID), backlog.BacklogID)))
		require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountId.String())), r.Keys())
		require.Equal(t, at.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountId), shadowPart.PartitionID)))

		refillUntil := at.Add(time.Minute)

		// Get items to refill from backlog
		itemIDs, err := getItemIDsFromBacklog(ctx, shard, &backlog, refillUntil, 1000)
		require.NoError(t, err)

		res, err := shard.BacklogRefill(ctx, &backlog, &shadowPart, refillUntil, itemIDs)
		require.NoError(t, err)

		require.Equal(t, 2, res.TotalBacklogCount)
		require.Equal(t, 1, res.BacklogCountUntil)
		require.Equal(t, 1, len(res.RefilledItems))

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

		// Get items to refill from backlog
		itemIDs, err = getItemIDsFromBacklog(ctx, shard, &backlog, refillUntil, 1000)
		require.NoError(t, err)

		res, err = shard.BacklogRefill(ctx, &backlog, &shadowPart, refillUntil, itemIDs)
		require.NoError(t, err)

		require.Equal(t, 1, res.TotalBacklogCount)
		require.Equal(t, 1, res.BacklogCountUntil)
		require.Equal(t, 1, len(res.RefilledItems))

		require.False(t, r.Exists(kg.BacklogSet(backlog.BacklogID)))
		require.False(t, r.Exists(kg.ShadowPartitionSet(shadowPart.PartitionID)))
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

	clock := clockwork.NewFakeClock()

	enqueueToBacklog := false
	_, shard := newQueue(
		t, rc,
		osqueue.WithClock(clock),
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
			return enqueueToBacklog
		}),
	)
	kg := shard.Client().kg

	fnID, accountID, envID := uuid.New(), uuid.New(), uuid.New()
	shadowPart := &osqueue.QueueShadowPartition{
		PartitionID: fnID.String(),
		LeaseID:     nil,
		FunctionID:  &fnID,
		EnvID:       &envID,
		AccountID:   &accountID,
	}

	marshaled, err := json.Marshal(shadowPart)
	require.NoError(t, err)

	t.Run("should not be able to lease missing partition", func(t *testing.T) {
		_, err = shard.ShadowPartitionLease(ctx, shadowPart, osqueue.ShadowPartitionLeaseDuration)
		require.Error(t, err)
		require.ErrorIs(t, err, osqueue.ErrShadowPartitionNotFound)

		r.HSet(kg.ShadowPartitionMeta(), shadowPart.PartitionID, string(marshaled))
	})

	var leaseID *ulid.ULID

	t.Run("first lease should lease shadow partition", func(t *testing.T) {
		dur := osqueue.ShadowPartitionLeaseDuration
		expectedLeaseExpiry := clock.Now().Add(dur)

		leaseID, err = shard.ShadowPartitionLease(ctx, shadowPart, dur)
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		leaseTime := ulid.Time(leaseID.Time())
		require.Equal(t, expectedLeaseExpiry.UnixMilli(), leaseTime.UnixMilli())

		leasedPart := osqueue.QueueShadowPartition{}
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
		_, err = shard.ShadowPartitionLease(ctx, shadowPart, osqueue.ShadowPartitionLeaseDuration)
		require.Error(t, err)
		require.ErrorIs(t, err, osqueue.ErrShadowPartitionAlreadyLeased)
	})

	t.Run("extend lease should work", func(t *testing.T) {
		// Simulate 2s have passed
		clock.Advance(2 * time.Second)
		r.FastForward(2 * time.Second)

		dur := osqueue.ShadowPartitionLeaseDuration
		expectedLeaseExpiry := clock.Now().Add(dur)

		newLeaseID, err := shard.ShadowPartitionExtendLease(ctx, shadowPart, *leaseID, dur)
		require.NoError(t, err)
		require.NotNil(t, newLeaseID)
		leaseID = newLeaseID

		leaseTime := ulid.Time(leaseID.Time())
		require.Equal(t, expectedLeaseExpiry.UnixMilli(), leaseTime.UnixMilli())

		leasedPart := osqueue.QueueShadowPartition{}
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

		err = shard.ShadowPartitionRequeue(ctx, shadowPart, nil)
		require.NoError(t, err)

		leasedPart := osqueue.QueueShadowPartition{}
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
		dur := osqueue.ShadowPartitionLeaseDuration
		expectedLeaseExpiry := now.Add(dur)
		leaseID, err := shard.ShadowPartitionLease(ctx, shadowPart, dur)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))

		err = shard.ShadowPartitionRequeue(ctx, shadowPart, nil)
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
		dur := osqueue.ShadowPartitionLeaseDuration
		expectedLeaseExpiry := now.Add(dur)
		leaseID, err := shard.ShadowPartitionLease(ctx, shadowPart, dur)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))

		forceRequeueAt := time.Now().Add(32 * time.Minute)
		err = shard.ShadowPartitionRequeue(ctx, shadowPart, &forceRequeueAt)
		require.NoError(t, err)

		leasedPart := osqueue.QueueShadowPartition{}
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
		dur := osqueue.ShadowPartitionLeaseDuration
		expectedLeaseExpiry := now.Add(dur)
		leaseID, err := shard.ShadowPartitionLease(ctx, shadowPart, dur)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
		require.Equal(t, expectedLeaseExpiry.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))

		forceRequeueAt := time.Now().Add(32 * time.Minute)
		err = shard.ShadowPartitionRequeue(ctx, shadowPart, &forceRequeueAt)
		require.NoError(t, err)

		leasedPart := osqueue.QueueShadowPartition{}
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

	clock := clockwork.NewFakeClock()

	enqueueToBacklog := true
	q, shard := newQueue(
		t, rc,
		osqueue.WithClock(clock),
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
			return enqueueToBacklog
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

	_, err = shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
	require.NoError(t, err)

	qspc := make(chan osqueue.ShadowPartitionChanMsg, 1)

	err = q.ScanShadowPartitions(ctx, at, qspc)
	require.NoError(t, err)

	select {
	case msg := <-qspc:
		require.Equal(t, fnID, *msg.ShadowPartition.FunctionID)
		require.Equal(t, accountID, *msg.ShadowPartition.AccountID)
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

	clock := clockwork.NewFakeClock()
	queueOpts := []osqueue.QueueOpt{
		osqueue.WithClock(clock),
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
			return true
		}),
		osqueue.WithBacklogRefillLimit(1),
		osqueue.WithRunMode(osqueue.QueueRunMode{
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
		osqueue.WithQueueShadowContinuationLimit(10),
	}

	ctx := context.Background()

	q, shard := newQueue(
		t, rc,
		queueOpts...,
	)

	fnID1, accountID1, envID1 := uuid.New(), uuid.New(), uuid.New()
	fnID2, accountID2, envID2 := uuid.New(), uuid.New(), uuid.New()

	addItem := func(shard osqueue.QueueShard, id string, identifier state.Identifier, at time.Time) osqueue.QueueItem {
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

		qi, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		return qi
	}
	at := clock.Now()

	item1 := addItem(shard, "test1", state.Identifier{
		AccountID:   accountID1,
		WorkspaceID: envID1,
		WorkflowID:  fnID1,
	}, at)

	item2 := addItem(shard, "test2", state.Identifier{
		AccountID:   accountID2,
		WorkspaceID: envID2,
		WorkflowID:  fnID2,
	}, at)

	// we leave some room for multiple partitions as scanShadowPartitions will
	// call both scan continuations and the regular scanner, so the first item
	// is expected to be the continuation, followed by the actual scan run
	qspc := make(chan osqueue.ShadowPartitionChanMsg, 10)

	sp1 := osqueue.ItemShadowPartition(ctx, item1)
	sp2 := osqueue.ItemShadowPartition(ctx, item2)
	require.NotEqual(t, sp1, sp2)

	t.Run("should retrieve using continuation", func(t *testing.T) {
		q.AddShadowContinue(ctx, &sp1, 1)

		cont, ok := q.GetShadowContinuations()[sp1.PartitionID]
		require.True(t, ok)
		require.Equal(t, uint(1), cont.Count)
		require.Equal(t, sp1, *cont.ShadowPart)

		fmt.Println("scanning")

		err = q.ScanShadowPartitions(ctx, at, qspc)
		require.NoError(t, err)

		// check that it's scanned and gone

		_, ok = q.GetShadowContinuations()[sp1.PartitionID]
		require.False(t, ok)
	})

	t.Run("should increase continuations when more items are available", func(t *testing.T) {
		r.FlushAll()

		q, shard := newQueue(
			t, rc,
			queueOpts...,
		)

		addItem(shard, "test1", state.Identifier{
			AccountID:   accountID1,
			WorkspaceID: envID1,
			WorkflowID:  fnID1,
		}, at)

		addItem(shard, "test2", state.Identifier{
			AccountID:   accountID1,
			WorkspaceID: envID1,
			WorkflowID:  fnID1,
		}, at)

		addItem(shard, "test3", state.Identifier{
			AccountID:   accountID2,
			WorkspaceID: envID2,
			WorkflowID:  fnID2,
		}, at)

		l := logger.StdlibLogger(ctx, logger.WithLoggerLevel(logger.LevelTrace))
		ctx := logger.WithStdlib(ctx, l)
		q.AddShadowContinue(ctx, &sp1, 1)

		// Process and refill once
		err := q.ProcessShadowPartition(ctx, &sp1, 1)
		require.NoError(t, err)

		// Expect continuation to be set
		cont, ok := q.GetShadowContinuations()[sp1.PartitionID]
		require.True(t, ok)
		require.Equal(t, uint(2), cont.Count)
		require.Equal(t, sp1, *cont.ShadowPart)

		// Process and refill again, final item in backlog
		err = q.ProcessShadowPartition(ctx, &sp1, 1)
		require.NoError(t, err)

		// Expect continuation to be cleared out
		_, ok = q.GetShadowContinuations()[sp1.PartitionID]
		require.False(t, ok)
	})

	t.Run("should remove continuation on missing shadow partition", func(t *testing.T) {
		r.FlushAll()

		q, shard := newQueue(
			t, rc,
			queueOpts...,
		)
		kg := shard.Client().kg

		addItem(shard, "test1", state.Identifier{
			AccountID:   accountID1,
			WorkspaceID: envID1,
			WorkflowID:  fnID1,
		}, at)

		addItem(shard, "test2", state.Identifier{
			AccountID:   accountID1,
			WorkspaceID: envID1,
			WorkflowID:  fnID1,
		}, at)

		addItem(shard, "test3", state.Identifier{
			AccountID:   accountID2,
			WorkspaceID: envID2,
			WorkflowID:  fnID2,
		}, at)

		// Process and refill once
		err := q.ProcessShadowPartition(ctx, &sp1, 1)
		require.NoError(t, err)

		// Expect continuation to be set
		cont, ok := q.GetShadowContinuations()[sp1.PartitionID]
		require.True(t, ok, cont)
		require.Equal(t, uint(2), cont.Count)
		require.Equal(t, sp1, *cont.ShadowPart)

		// Drop shadow partition
		r.HDel(kg.ShadowPartitionMeta(), sp1.PartitionID)

		// Process and refill again, final item in backlog
		err = q.ProcessShadowPartition(ctx, &sp1, 1)
		require.NoError(t, err)

		// Expect continuation to be cleared out
		_, ok = q.GetShadowContinuations()[sp1.PartitionID]
		require.False(t, ok)
	})

	t.Run("should remove continuation on leased shadow partition", func(t *testing.T) {
		r.FlushAll()

		q.ClearShadowContinuations()
		q, shard := newQueue(
			t, rc,
			queueOpts...,
		)

		addItem(shard, "test1", state.Identifier{
			AccountID:   accountID1,
			WorkspaceID: envID1,
			WorkflowID:  fnID1,
		}, at)

		addItem(shard, "test2", state.Identifier{
			AccountID:   accountID1,
			WorkspaceID: envID1,
			WorkflowID:  fnID1,
		}, at)

		addItem(shard, "test3", state.Identifier{
			AccountID:   accountID2,
			WorkspaceID: envID2,
			WorkflowID:  fnID2,
		}, at)

		// Process and refill once
		err := q.ProcessShadowPartition(ctx, &sp1, 1)
		require.NoError(t, err)

		// Expect continuation to be set
		cont, ok := q.GetShadowContinuations()[sp1.PartitionID]
		require.True(t, ok)
		require.Equal(t, uint(2), cont.Count)
		require.Equal(t, sp1, *cont.ShadowPart)

		// Simulate another process leasing the shadow partition
		spCopy := sp1
		leaseID, err := shard.ShadowPartitionLease(ctx, &spCopy, 3*time.Minute)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		// Process and refill again, final item in backlog
		err = q.ProcessShadowPartition(ctx, &sp1, 1)
		require.NoError(t, err)

		// Expect continuation to be cleared out
		_, ok = q.GetShadowContinuations()[sp1.PartitionID]
		require.False(t, ok)
	})
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

		clock := clockwork.NewFakeClock()

		accountID, wsID, fnID := uuid.New(), uuid.New(), uuid.New()

		enqueueToBacklog := true
		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			osqueue.WithRunMode(osqueue.QueueRunMode{
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
			osqueue.WithBacklogRefillLimit(500),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return osqueue.PartitionConstraintConfig{
					Concurrency: osqueue.PartitionConcurrency{
						AccountConcurrency:  123,
						FunctionConcurrency: 45,
						SystemConcurrency:   678,
					},
				}
			}),
		)
		kg := shard.Client().kg

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

			qi, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
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

		backlog := osqueue.ItemBacklog(ctx, items[0])
		shadowPart := osqueue.ItemShadowPartition(ctx, items[0])

		// Pointer should be earliest item
		require.Equal(t, now.Add(time.Second).UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), items[0].ID)))
		require.Equal(t, now.Add(time.Second).UnixMilli(), int64(score(t, r, kg.ShadowPartitionSet(shadowPart.PartitionID), backlog.BacklogID)))
		require.Equal(t, now.Add(time.Second).UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, now.Add(time.Second).UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
		require.Equal(t, now.Add(time.Second).UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))

		until := now.Add(osqueue.PartitionLookahead)
		peeked, totalUntil, err := shard.ShadowPartitionPeek(ctx, &shadowPart, false, until, 100)
		require.NoError(t, err)

		require.Equal(t, 1, totalUntil)
		require.Len(t, peeked, 1)
		require.Equal(t, backlog, *peeked[0])

		for i := range numItems {
			itemAt := now.Add(time.Duration(i+1) * time.Second)
			refillUntil := itemAt

			// Get items to refill from backlog
			itemIDs, err := getItemIDsFromBacklog(ctx, shard, &backlog, refillUntil, 1000)
			require.NoError(t, err)

			res, err := shard.BacklogRefill(ctx, &backlog, &shadowPart, refillUntil, itemIDs)
			require.NoError(t, err)

			require.Equal(t, numItems-i, res.TotalBacklogCount)
			require.Equal(t, 1, res.BacklogCountUntil)
			require.Equal(t, 1, len(res.RefilledItems))

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

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Second))

		accountID, wsID, fnID := uuid.New(), uuid.New(), uuid.New()

		enqueueToBacklog := true
		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			osqueue.WithRunMode(osqueue.QueueRunMode{
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
			osqueue.WithBacklogRefillLimit(500),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return osqueue.PartitionConstraintConfig{
					Concurrency: osqueue.PartitionConcurrency{
						AccountConcurrency:  123,
						FunctionConcurrency: 45,
						SystemConcurrency:   678,
					},
				}
			}),
		)
		kg := shard.Client().kg

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
		qi, err := shard.EnqueueItem(ctx, item, sleepUntil, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		backlog := osqueue.ItemBacklog(ctx, qi)
		shadowPart := osqueue.ItemShadowPartition(ctx, qi)

		require.Equal(t, sleepUntil.UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID)))
		require.Equal(t, sleepUntil.UnixMilli(), int64(score(t, r, kg.ShadowPartitionSet(shadowPart.PartitionID), backlog.BacklogID)))
		require.Equal(t, sleepUntil.UnixMilli(), int64(score(t, r, kg.GlobalShadowPartitionSet(), shadowPart.PartitionID)))
		require.Equal(t, sleepUntil.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountID), shadowPart.PartitionID)))
		require.Equal(t, sleepUntil.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountID.String())))

		until := now.Add(time.Second)
		peeked, totalUntil, err := shard.ShadowPartitionPeek(ctx, &shadowPart, false, until, 100)
		require.NoError(t, err)

		require.Equal(t, 0, totalUntil)
		require.Len(t, peeked, 0)

		until = now.Add(2 * time.Second)
		peeked, totalUntil, err = shard.ShadowPartitionPeek(ctx, &shadowPart, false, until, 100)
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

	clock := clockwork.NewFakeClock()

	testLifecycles := newTestLifecycleListener()

	cmLifecycles := constraintapi.NewConstraintAPIDebugLifecycles()
	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClock(clock),
		constraintapi.WithClient(rc),
		constraintapi.WithShardName("test"),
		constraintapi.WithLifecycles(cmLifecycles),
	)
	require.NoError(t, err)

	constraints := osqueue.PartitionConstraintConfig{
		FunctionVersion: 1,
		Concurrency: osqueue.PartitionConcurrency{
			AccountConcurrency:  1,
			FunctionConcurrency: 1,
		},
	}

	enqueueToBacklog := true
	q, shard := newQueue(
		t, rc,
		osqueue.WithClock(clock),
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
			return enqueueToBacklog
		}),
		osqueue.WithRunMode(osqueue.QueueRunMode{
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
		osqueue.WithBacklogRefillLimit(100),
		osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
			return constraints
		}),
		osqueue.WithQueueLifecycles(testLifecycles),
		osqueue.WithCapacityManager(cm),
		osqueue.WithAcquireCapacityLeaseOnBacklogRefill(true),
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

		qi, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		return qi
	}
	at := clock.Now()

	itemA1 := addItem("test1", state.Identifier{
		AccountID:   accountID1,
		WorkspaceID: envID1,
		WorkflowID:  fnID1,
	}, at)

	sp1 := osqueue.ItemShadowPartition(ctx, itemA1)
	b1 := osqueue.ItemBacklog(ctx, itemA1)

	require.Equal(t, 1, constraints.Concurrency.FunctionConcurrency)
	require.Equal(t, 1, constraints.Concurrency.AccountConcurrency)

	res, limitingConstraint, err := q.ProcessShadowPartitionBacklog(ctx, &sp1, &b1, at.Add(time.Minute), constraints)
	require.NoError(t, err)
	require.NotNil(t, res)

	// 1 item must have been refilled
	require.Len(t, res.RefilledItems, 1)

	// This was the last unit of account + function concurrency, so we should see function concurrency as the constraint
	require.Equal(t, enums.QueueConstraintFunctionConcurrency, limitingConstraint)

	require.EventuallyWithT(t, func(t *assert.CollectT) {
		testLifecycles.lock.Lock()
		assert.Equal(t, 0, testLifecycles.acctConcurrency[accountID1])
		assert.Equal(t, 1, testLifecycles.fnConcurrency[fnID1])
		assert.Equal(t, 0, testLifecycles.fnConcurrency[fnID2])
		testLifecycles.lock.Unlock()
	}, 1*time.Second, 100*time.Millisecond)

	require.Equal(t, 1, len(cmLifecycles.AcquireCalls))
	cmLifecycles.Reset()

	_ = addItem("test2", state.Identifier{
		AccountID:   accountID1,
		WorkspaceID: envID1,
		WorkflowID:  fnID1,
	}, at)

	res, limitingConstraint, err = q.ProcessShadowPartitionBacklog(ctx, &sp1, &b1, at.Add(time.Minute), constraints)
	require.NoError(t, err)
	require.NotNil(t, res)

	require.Equal(t, enums.QueueConstraintFunctionConcurrency, limitingConstraint)
	require.Len(t, res.RefilledItems, 0)

	require.EventuallyWithT(t, func(t *assert.CollectT) {
		testLifecycles.lock.Lock()
		assert.Equal(t, 1, len(cmLifecycles.AcquireCalls))
		assert.Equal(t, 0, testLifecycles.acctConcurrency[accountID1], "expected account not to be hit")
		assert.Equal(t, 2, testLifecycles.fnConcurrency[fnID1], "expected fn1 to be hit twice", fnID1, testLifecycles.fnConcurrency)
		assert.Equal(t, 0, testLifecycles.fnConcurrency[fnID2], "expected fn2 not to be hit")
		testLifecycles.lock.Unlock()
	}, 1*time.Second, 100*time.Millisecond)

	itemB1 := addItem("test3", state.Identifier{
		WorkflowID:  fnID2,
		AccountID:   accountID1,
		WorkspaceID: envID1,
	}, at)

	b2 := osqueue.ItemBacklog(ctx, itemB1)

	sp2 := osqueue.ItemShadowPartition(ctx, itemB1)

	res, _, err = q.ProcessShadowPartitionBacklog(ctx, &sp2, &b2, at.Add(time.Minute), constraints)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		testLifecycles.lock.Lock()
		assert.Equal(t, 1, testLifecycles.acctConcurrency[accountID1])
		assert.Equal(t, 2, testLifecycles.fnConcurrency[fnID1])
		assert.Equal(t, 0, testLifecycles.fnConcurrency[fnID2])
		testLifecycles.lock.Unlock()
	}, 1*time.Second, 100*time.Millisecond)
}

func TestBacklogRefillWithDisabledConstraintChecks(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	constraints := osqueue.PartitionConstraintConfig{
		FunctionVersion: 1,
		Throttle: &osqueue.PartitionThrottle{
			ThrottleKeyExpressionHash: "throttle-key-hash",
			Limit:                     1,
			Burst:                     0,
			Period:                    5,
		},
		Concurrency: osqueue.PartitionConcurrency{
			AccountConcurrency:  10,
			FunctionConcurrency: 5,
		},
	}

	var cm constraintapi.CapacityManager = &testRolloutManager{}

	_, shard := newQueue(
		t, rc,
		osqueue.WithClock(clock),
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
			return true
		}),
		osqueue.WithAcquireCapacityLeaseOnBacklogRefill(true),
		osqueue.WithCapacityManager(cm),
		osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
			return constraints
		}),
	)
	ctx := context.Background()

	accountID := uuid.New()
	fnID := uuid.New()

	qi := osqueue.QueueItem{
		FunctionID: fnID,
		Data: osqueue.Item{
			Kind:    osqueue.KindStart,
			Payload: json.RawMessage("{\"test\":\"payload\"}"),
			Identifier: state.Identifier{
				AccountID:  accountID,
				WorkflowID: fnID,
			},
			Throttle: &osqueue.Throttle{
				KeyExpressionHash: "throttle-key-hash",
				Limit:             1,
				Burst:             0,
				Period:            5,
				Key:               "throttle-key",
			},
		},
	}

	start := time.Now().Truncate(time.Second)

	item1, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
	require.NoError(t, err)
	item3, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
	require.NoError(t, err)

	backlog := osqueue.ItemBacklog(ctx, item1)
	require.NotNil(t, backlog.Throttle)

	shadowPart := osqueue.ItemShadowPartition(ctx, item1)

	res, err := shard.BacklogRefill(ctx, &backlog, &shadowPart, clock.Now().Add(time.Minute), []string{item1.ID})
	require.NoError(t, err)
	require.Equal(t, 1, len(res.RefilledItems))

	res, err = shard.BacklogRefill(
		ctx,
		&backlog,
		&shadowPart,
		clock.Now().Add(time.Minute),
		[]string{item3.ID},
	)
	require.NoError(t, err)
	require.Equal(t, 1, len(res.RefilledItems))
	require.Equal(t, []string{item3.ID}, res.RefilledItems)
}

func TestBacklogRefillSetCapacityLease(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	constraints := osqueue.PartitionConstraintConfig{
		FunctionVersion: 1,
		Concurrency: osqueue.PartitionConcurrency{
			AccountConcurrency:  10,
			FunctionConcurrency: 5,
		},
	}

	q, shard := newQueue(
		t, rc,
		osqueue.WithClock(clock),
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
			return true
		}),
		osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
			return constraints
		}),
	)
	ctx := context.Background()

	accountID := uuid.New()
	fnID := uuid.New()

	qi := osqueue.QueueItem{
		FunctionID: fnID,
		Data: osqueue.Item{
			Kind:    osqueue.KindStart,
			Payload: json.RawMessage("{\"test\":\"payload\"}"),
			Identifier: state.Identifier{
				AccountID:  accountID,
				WorkflowID: fnID,
			},
		},
	}

	start := clock.Now()

	item1, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
	require.NoError(t, err)

	item2, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
	require.NoError(t, err)

	item3, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
	require.NoError(t, err)

	backlog := osqueue.ItemBacklog(ctx, item1)

	shadowPart := osqueue.ItemShadowPartition(ctx, item1)

	capacityLeaseID := ulid.MustNew(ulid.Timestamp(clock.Now()), rand.Reader)
	capacityLeaseID2 := ulid.MustNew(ulid.Timestamp(clock.Now()), rand.Reader)
	capacityLeaseID3 := ulid.MustNew(ulid.Timestamp(clock.Now()), rand.Reader)

	refillItemIDs := []string{item1.ID, item3.ID, item2.ID} // intentionally out of order
	capacityLeaseIDs := []osqueue.CapacityLease{{
		LeaseID: capacityLeaseID,
	}, {
		LeaseID: capacityLeaseID3,
	}, {
		LeaseID: capacityLeaseID2,
	}}

	// Refill once, should work
	res, err := shard.BacklogRefill(
		ctx,
		&backlog,
		&shadowPart,
		clock.Now().Add(time.Minute),
		refillItemIDs,
		osqueue.WithBacklogRefillItemCapacityLeases(capacityLeaseIDs),
	)
	require.NoError(t, err)
	require.Equal(t, 3, len(res.RefilledItems))

	loaded, err := q.ItemByID(ctx, shard, item1.ID)
	require.NoError(t, err)
	require.Equal(t, loaded.ID, item1.ID)
	require.NotNil(t, loaded.CapacityLease)
	require.Equal(t, capacityLeaseID, loaded.CapacityLease.LeaseID)

	loaded, err = q.ItemByID(ctx, shard, item2.ID)
	require.NoError(t, err)
	require.Equal(t, loaded.ID, item2.ID)
	require.NotNil(t, loaded.CapacityLease)
	require.Equal(t, capacityLeaseID2, loaded.CapacityLease.LeaseID)

	loaded, err = q.ItemByID(ctx, shard, item3.ID)
	require.NoError(t, err)
	require.Equal(t, loaded.ID, item3.ID)
	require.NotNil(t, loaded.CapacityLease)
	require.Equal(t, capacityLeaseID3, loaded.CapacityLease.LeaseID)
}

func TestPreventThrottleBacklogUnfairness(t *testing.T) {
	t.Run("should insert default function backlog to counter unfairness", func(t *testing.T) {
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		ctx := context.Background()
		l := logger.StdlibLogger(ctx, logger.WithLoggerLevel(logger.LevelTrace))
		ctx = logger.WithStdlib(ctx, l)

		clock := clockwork.NewFakeClock()

		constraints := osqueue.PartitionConstraintConfig{
			FunctionVersion: 1,
			Throttle: &osqueue.PartitionThrottle{
				ThrottleKeyExpressionHash: "expr-hash",
				Limit:                     1,
				Period:                    60,
			},
			Concurrency: osqueue.PartitionConcurrency{
				AccountConcurrency:  10,
				FunctionConcurrency: 5,
			},
		}

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return constraints
			}),
			osqueue.WithLogger(l),
		)
		kg := shard.Client().kg

		accountID := uuid.New()
		fnID := uuid.New()

		qi := osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Kind:    osqueue.KindStart,
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{
					AccountID:       accountID,
					WorkflowID:      fnID,
					WorkflowVersion: constraints.FunctionVersion,
				},
				Throttle: &osqueue.Throttle{
					Limit:             1,
					Period:            60,
					KeyExpressionHash: "expr-hash",
					Key:               "key-hash",
				},
			},
		}

		start := clock.Now()

		item1, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		qi2 := osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Kind:    osqueue.KindEdge,
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{
					AccountID:       accountID,
					WorkflowID:      fnID,
					WorkflowVersion: constraints.FunctionVersion,
				},
			},
		}

		item2, err := shard.EnqueueItem(ctx, qi2, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		shadowPart := osqueue.ItemShadowPartition(ctx, item1)

		// This should be the throttle key backlog
		b := osqueue.ItemBacklog(ctx, item1)

		// This should be the "default" function backlog
		b2 := osqueue.ItemBacklog(ctx, item2)

		require.Nil(t, shadowPart.DefaultBacklog(constraints, true))

		// Function backlog should return b2
		require.Equal(t, b2, *shadowPart.DefaultBacklog(constraints, false))

		// Shadow partition set should include both backlog IDs
		require.True(t, r.Exists(kg.ShadowPartitionSet(shadowPart.PartitionID)))

		mem, err := r.ZMembers(kg.ShadowPartitionSet(shadowPart.PartitionID))
		require.NoError(t, err)
		require.Contains(t, mem, b.BacklogID)
		require.Contains(t, mem, b2.BacklogID)

		// Remove "default" function backlog to test the new behavior which should always process this backlog
		_, err = r.ZRem(kg.ShadowPartitionSet(shadowPart.PartitionID), b2.BacklogID)
		require.NoError(t, err)

		require.True(t, b.Start)
		require.NotNil(t, b.Throttle)
		require.False(t, b2.Start)
		require.Nil(t, b2.Throttle)

		require.False(t, r.Exists(kg.PartitionQueueSet(enums.PartitionTypeDefault, fnID.String(), "")))

		err = q.ProcessShadowPartition(ctx, &shadowPart, 0)
		require.NoError(t, err, "expected to refill from both backlogs", r.Dump())

		require.True(t, r.Exists(kg.PartitionQueueSet(enums.PartitionTypeDefault, fnID.String(), "")))

		mem, err = r.ZMembers(kg.PartitionQueueSet(enums.PartitionTypeDefault, fnID.String(), ""))
		require.NoError(t, err)
		require.Contains(t, mem, item1.ID)
		require.Contains(t, mem, item2.ID)
	})

	t.Run("should ensure default function backlog comes first", func(t *testing.T) {
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		ctx := context.Background()
		l := logger.StdlibLogger(ctx, logger.WithLoggerLevel(logger.LevelTrace))
		ctx = logger.WithStdlib(ctx, l)

		clock := clockwork.NewFakeClock()

		constraints := osqueue.PartitionConstraintConfig{
			FunctionVersion: 1,
			Throttle: &osqueue.PartitionThrottle{
				ThrottleKeyExpressionHash: "expr-hash",
				Limit:                     1,
				Period:                    60,
			},
			Concurrency: osqueue.PartitionConcurrency{
				AccountConcurrency:  10,
				FunctionConcurrency: 1, // ensure we can only refill a single item
			},
		}

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return constraints
			}),
			osqueue.WithLogger(l),
		)
		kg := shard.Client().kg

		accountID := uuid.New()
		fnID := uuid.New()

		start := clock.Now()

		qi := osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Kind:    osqueue.KindStart,
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{
					AccountID:       accountID,
					WorkflowID:      fnID,
					WorkflowVersion: constraints.FunctionVersion,
				},
				Throttle: &osqueue.Throttle{
					Limit:             1,
					Period:            60,
					KeyExpressionHash: "expr-hash",
					Key:               fmt.Sprintf("key-hash-%d", 0),
				},
			},
		}

		item1, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		amount := 1000
		for i := range amount {
			qi := osqueue.QueueItem{
				FunctionID: fnID,
				Data: osqueue.Item{
					Kind:    osqueue.KindStart,
					Payload: json.RawMessage("{\"test\":\"payload\"}"),
					Identifier: state.Identifier{
						AccountID:       accountID,
						WorkflowID:      fnID,
						WorkflowVersion: constraints.FunctionVersion,
					},
					Throttle: &osqueue.Throttle{
						Limit:             1,
						Period:            60,
						KeyExpressionHash: "expr-hash",
						Key:               fmt.Sprintf("key-hash-%d", i+1),
					},
				},
			}

			_, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)
		}

		qi2 := osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Kind:    osqueue.KindEdge,
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{
					AccountID:       accountID,
					WorkflowID:      fnID,
					WorkflowVersion: constraints.FunctionVersion,
				},
			},
		}

		// Insert default backlog LATER than other items, but still early enough to get peeked for refilling (+2s)
		// This is to make the test case more extreme: We should always expect the default backlog to be processed,
		// ensuring we continue processing items to wrap up existing runs
		item2, err := shard.EnqueueItem(ctx, qi2, start.Add(time.Second), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		shadowPart := osqueue.ItemShadowPartition(ctx, item1)

		// This should be the throttle key backlog
		b := osqueue.ItemBacklog(ctx, item1)

		// This should be the "default" function backlog
		b2 := osqueue.ItemBacklog(ctx, item2)

		require.Nil(t, shadowPart.DefaultBacklog(constraints, true))

		// Function backlog should return b2
		require.Equal(t, b2, *shadowPart.DefaultBacklog(constraints, false))

		// Shadow partition set should include both backlog IDs
		require.True(t, r.Exists(kg.ShadowPartitionSet(shadowPart.PartitionID)))

		mem, err := r.ZMembers(kg.ShadowPartitionSet(shadowPart.PartitionID))
		require.NoError(t, err)

		// ensure we have 1000 + 2 backlogs
		require.Len(t, mem, amount+2)

		require.Contains(t, mem, b.BacklogID)
		require.Contains(t, mem, b2.BacklogID)

		require.True(t, b.Start)
		require.NotNil(t, b.Throttle)
		require.False(t, b2.Start)
		require.Nil(t, b2.Throttle)

		require.False(t, r.Exists(kg.PartitionQueueSet(enums.PartitionTypeDefault, fnID.String(), "")))

		err = q.ProcessShadowPartition(ctx, &shadowPart, 0)
		require.NoError(t, err, "expected to refill from both backlogs", r.Dump())

		require.True(t, r.Exists(kg.PartitionQueueSet(enums.PartitionTypeDefault, fnID.String(), "")))

		mem, err = r.ZMembers(kg.PartitionQueueSet(enums.PartitionTypeDefault, fnID.String(), ""))
		require.NoError(t, err)
		require.Len(t, mem, 101)
		require.Contains(t, mem, item2.ID)
	})
}

type testLifecycleListener struct {
	lock            *sync.Mutex
	fnConcurrency   map[uuid.UUID]int
	acctConcurrency map[uuid.UUID]int
	ckConcurrency   map[string]int
}

func newTestLifecycleListener() testLifecycleListener {
	return testLifecycleListener{
		lock:            &sync.Mutex{},
		fnConcurrency:   map[uuid.UUID]int{},
		acctConcurrency: map[uuid.UUID]int{},
		ckConcurrency:   map[string]int{},
	}
}

func (t testLifecycleListener) OnFnConcurrencyLimitReached(_ context.Context, fnID uuid.UUID) {
	t.lock.Lock()
	defer t.lock.Unlock()

	i := t.fnConcurrency[fnID]
	t.fnConcurrency[fnID] = i + 1
}

func (t testLifecycleListener) OnAccountConcurrencyLimitReached(
	_ context.Context,
	acctID uuid.UUID,
	workspaceID *uuid.UUID,
) {
	t.lock.Lock()
	defer t.lock.Unlock()

	i := t.acctConcurrency[acctID]
	t.acctConcurrency[acctID] = i + 1
}

func (t testLifecycleListener) OnCustomKeyConcurrencyLimitReached(_ context.Context, key string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	i := t.ckConcurrency[key]
	t.ckConcurrency[key] = i + 1
}
