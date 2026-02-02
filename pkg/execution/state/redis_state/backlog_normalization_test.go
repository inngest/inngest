package redis_state

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

func TestBacklogNormalizationLease(t *testing.T) {
	ctx := context.Background()

	_, rc := initRedis(t)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	enqueueToBacklog := false
	_, shard := newQueue(
		t, rc,
		osqueue.WithClock(clock),
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
			return enqueueToBacklog
		}),
	)

	fnID, accountID, envID := uuid.New(), uuid.New(), uuid.New()
	shadowPart := &osqueue.QueueShadowPartition{
		PartitionID: fnID.String(),
		LeaseID:     nil,
		FunctionID:  &fnID,
		EnvID:       &envID,
		AccountID:   &accountID,
	}

	backlog := &osqueue.QueueBacklog{
		BacklogID:         "yolo",
		ShadowPartitionID: shadowPart.PartitionID,
		Throttle: &osqueue.BacklogThrottle{
			ThrottleKey:               "something",
			ThrottleKeyRawValue:       "somethingelse",
			ThrottleKeyExpressionHash: "hash",
		},
	}

	// should lease successfully
	err := shard.LeaseBacklogForNormalization(ctx, backlog)
	require.NoError(t, err)

	// another attempt should fail
	err = shard.LeaseBacklogForNormalization(ctx, backlog)
	require.ErrorIs(t, err, osqueue.ErrBacklogAlreadyLeasedForNormalization)
}

func TestExtendBacklogNormalizationLease(t *testing.T) {
	ctx := context.Background()

	r, rc := initRedis(t)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	enqueueToBacklog := false
	_, shard := newQueue(
		t, rc,
		osqueue.WithClock(clock),
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
			return enqueueToBacklog
		}),
	)

	fnID, accountID, envID := uuid.New(), uuid.New(), uuid.New()
	shadowPart := &osqueue.QueueShadowPartition{
		PartitionID: fnID.String(),
		LeaseID:     nil,
		FunctionID:  &fnID,
		EnvID:       &envID,
		AccountID:   &accountID,
	}

	backlog := &osqueue.QueueBacklog{
		BacklogID:         "yolo",
		ShadowPartitionID: shadowPart.PartitionID,
		Throttle: &osqueue.BacklogThrottle{
			ThrottleKey:               "something",
			ThrottleKeyRawValue:       "somethingelse",
			ThrottleKeyExpressionHash: "hash",
		},
	}

	// attempt to extend without a lease will fail
	err := shard.ExtendBacklogNormalizationLease(ctx, clock.Now(), backlog)
	require.ErrorIs(t, err, osqueue.ErrBacklogNormalizationLeaseExpired)

	// lease the backlog first
	err = shard.LeaseBacklogForNormalization(ctx, backlog)
	require.NoError(t, err)

	// should succeed
	err = shard.ExtendBacklogNormalizationLease(ctx, clock.Now(), backlog)
	require.NoError(t, err)

	clock.Advance(2 * osqueue.BacklogNormalizeLeaseDuration)
	r.FastForward(2 * osqueue.BacklogNormalizeLeaseDuration)

	// expect lease to be expired again
	err = shard.ExtendBacklogNormalizationLease(ctx, clock.Now(), backlog)
	require.ErrorIs(t, err, osqueue.ErrBacklogNormalizationLeaseExpired)
}

func TestQueueBacklogPrepareNormalize(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	_, shard := newQueue(
		t, rc,
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
			return true
		}),
		osqueue.WithClock(clock),
	)
	kg := shard.Client().kg
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

	t.Run("should garbage-collect empty backlog pointer", func(t *testing.T) {
		r.FlushAll()

		qi, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		expectedBacklog := osqueue.ItemBacklog(ctx, item)
		require.NotEmpty(t, expectedBacklog.BacklogID)
		require.NotEmpty(t, r.HGet(kg.BacklogMeta(), expectedBacklog.BacklogID))

		shadowPartition := osqueue.ItemShadowPartition(ctx, item)
		require.NotEmpty(t, shadowPartition.PartitionID)

		// expect backlog in shadow partition
		require.True(t, hasMember(t, r, kg.ShadowPartitionSet(shadowPartition.PartitionID), expectedBacklog.BacklogID))

		// remove item from backlog
		_, err = r.ZRem(kg.BacklogSet(expectedBacklog.BacklogID), qi.ID)
		require.NoError(t, err)
		require.False(t, r.Exists(kg.BacklogSet(expectedBacklog.BacklogID)))

		// still expect backlog in shadow partition
		require.True(t, hasMember(t, r, kg.ShadowPartitionSet(shadowPartition.PartitionID), expectedBacklog.BacklogID))

		err = shard.BacklogPrepareNormalize(ctx, &expectedBacklog, &shadowPartition)
		require.Error(t, err)
		require.ErrorIs(t, err, osqueue.ErrBacklogGarbageCollected)

		require.False(t, hasMember(t, r, kg.GlobalAccountNormalizeSet(), accountId.String()))
		require.False(t, hasMember(t, r, kg.AccountNormalizeSet(accountId), fnID.String()))
		require.False(t, hasMember(t, r, kg.PartitionNormalizeSet(fnID.String()), expectedBacklog.BacklogID))

		require.False(t, r.Exists(kg.BacklogSet(expectedBacklog.BacklogID)))
		require.Empty(t, r.HGet(kg.BacklogMeta(), expectedBacklog.BacklogID))

		// no longer expect backlog in shadow partition set
		require.False(t, hasMember(t, r, kg.ShadowPartitionSet(shadowPartition.PartitionID), expectedBacklog.BacklogID))
	})

	t.Run("should move backlog to normalization set", func(t *testing.T) {
		r.FlushAll()

		_, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		expectedBacklog := osqueue.ItemBacklog(ctx, item)
		require.NotEmpty(t, expectedBacklog.BacklogID)

		shadowPartition := osqueue.ItemShadowPartition(ctx, item)
		require.NotEmpty(t, shadowPartition.PartitionID)
		err = shard.BacklogPrepareNormalize(ctx, &expectedBacklog, &shadowPartition)
		require.NoError(t, err)

		require.True(t, hasMember(t, r, kg.GlobalAccountNormalizeSet(), accountId.String()))
		require.True(t, hasMember(t, r, kg.AccountNormalizeSet(accountId), fnID.String()))
		require.True(t, hasMember(t, r, kg.PartitionNormalizeSet(fnID.String()), expectedBacklog.BacklogID))

		expectedTime := clock.Now().UnixMilli()

		require.Equal(t, expectedTime, int64(score(t, r, kg.GlobalAccountNormalizeSet(), accountId.String())))
		require.Equal(t, expectedTime, int64(score(t, r, kg.AccountNormalizeSet(accountId), fnID.String())))
		require.Equal(t, expectedTime, int64(score(t, r, kg.PartitionNormalizeSet(fnID.String()), expectedBacklog.BacklogID)))
	})
}

func TestQueueBacklogNormalization(t *testing.T) {
	// prep
	r, rc := initRedis(t)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	q, shard := newQueue(
		t, rc,
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
			return true
		}),
		osqueue.WithClock(clock),
	)
	kg := shard.Client().kg
	ctx := context.Background()

	accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	require.Len(t, r.Keys(), 0)

	item := osqueue.QueueItem{
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

	// Create backlog
	for i := range 10 {
		at := clock.Now().Add(time.Duration(i*100) * time.Millisecond)
		_, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)
	}

	//
	//   Test cases
	//

	// Verify backlog is created as expected
	backlog := osqueue.ItemBacklog(ctx, item)
	require.NotEmpty(t, backlog.BacklogID)

	shadowPartition := osqueue.ItemShadowPartition(ctx, item)
	require.NotEmpty(t, shadowPartition.PartitionID)

	constraints := osqueue.PartitionConstraintConfig{}

	// Mark backlog for normalization
	err := shard.BacklogPrepareNormalize(ctx, &backlog, &shadowPartition)
	require.NoError(t, err)
	require.Equal(t, 10, zcard(t, rc, kg.BacklogSet(backlog.BacklogID)))
	require.True(t, hasMember(t, r, kg.GlobalAccountNormalizeSet(), accountId.String()))
	require.True(t, hasMember(t, r, kg.AccountNormalizeSet(accountId), fnID.String()))
	require.True(t, hasMember(t, r, kg.PartitionNormalizeSet(fnID.String()), backlog.BacklogID))

	// Verify normalization
	require.NoError(t, shard.LeaseBacklogForNormalization(ctx, &backlog)) // lease it first

	require.NoError(t, q.NormalizeBacklog(ctx, &backlog, &shadowPartition, constraints))
	require.Equal(t, 0, zcard(t, rc, kg.BacklogSet(backlog.BacklogID)))
	require.False(t, hasMember(t, r, kg.GlobalAccountNormalizeSet(), accountId.String()))
	require.False(t, hasMember(t, r, kg.AccountNormalizeSet(accountId), fnID.String()))
	require.False(t, hasMember(t, r, kg.PartitionNormalizeSet(fnID.String()), backlog.BacklogID))
}

func TestBacklogNormalizeItem(t *testing.T) {
	r, rc := initRedis(t)
	defer r.Close()

	clock := clockwork.NewFakeClock()

	accountID, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	hashedConcurrencyKeyExpr := hashConcurrencyKey("event.data.customerId")
	unhashedValue := "customer1"
	scope := enums.ConcurrencyScopeFn
	entity := fnID
	fullKey := util.ConcurrencyKey(scope, entity, unhashedValue)

	customConc := []state.CustomConcurrency{
		{
			Key:                       fullKey,
			Hash:                      hashedConcurrencyKeyExpr,
			Limit:                     123,
			UnhashedEvaluatedKeyValue: unhashedValue,
		},
	}

	throttleKey := util.XXHash("customer1")
	throttleKeyExpr := util.XXHash("event.data.customerId")

	throttle := &osqueue.Throttle{
		Key:                 throttleKey,
		Limit:               100,
		Burst:               10,
		Period:              int(time.Hour.Seconds()),
		UnhashedThrottleKey: unhashedValue,
		KeyExpressionHash:   throttleKeyExpr,
	}

	latestConstraints := osqueue.PartitionConstraintConfig{}

	q, shard := newQueue(
		t, rc,
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
			return true
		}),
		osqueue.WithNormalizeRefreshItemCustomConcurrencyKeys(func(ctx context.Context, item *osqueue.QueueItem, existingKeys []state.CustomConcurrency, shadowPartition *osqueue.QueueShadowPartition) ([]state.CustomConcurrency, error) {
			return customConc, nil
		}),
		osqueue.WithRefreshItemThrottle(func(ctx context.Context, item *osqueue.QueueItem) (*osqueue.Throttle, error) {
			return throttle, nil
		}),
		osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
			return latestConstraints
		}),
		osqueue.WithClock(clock),
	)
	ctx := context.Background()
	kg := shard.Client().kg

	require.Len(t, r.Keys(), 0)

	item := osqueue.QueueItem{
		FunctionID:  fnID,
		WorkspaceID: wsID,
		Data: osqueue.Item{
			WorkspaceID: wsID,
			Kind:        osqueue.KindStart,
			Identifier: state.Identifier{
				WorkflowID:  fnID,
				AccountID:   accountID,
				WorkspaceID: wsID,
			},
			QueueName:             nil,
			Throttle:              nil,
			CustomConcurrencyKeys: nil,
		},
		QueueName: nil,
	}

	sp := osqueue.ItemShadowPartition(ctx, item)
	sourceBacklog := osqueue.ItemBacklog(ctx, item)

	qi, err := shard.EnqueueItem(ctx, item, clock.Now(), osqueue.EnqueueOpts{})
	require.NoError(t, err)

	require.True(t, r.Exists(kg.BacklogSet(sourceBacklog.BacklogID)))
	require.True(t, hasMember(t, r, kg.ShadowPartitionSet(sp.PartitionID), sourceBacklog.BacklogID))
	require.True(t, hasMember(t, r, kg.BacklogSet(sourceBacklog.BacklogID), qi.ID))

	normalizedItem, err := q.NormalizeItem(ctx, &sp, latestConstraints, &sourceBacklog, qi)
	require.NoError(t, err)

	qi.Data.CustomConcurrencyKeys = customConc
	qi.Data.Throttle = throttle

	require.Equal(t, qi, normalizedItem)
	targetBacklog := osqueue.ItemBacklog(ctx, qi)

	actualBacklog := osqueue.ItemBacklog(ctx, normalizedItem)

	require.Equal(t, targetBacklog, actualBacklog)

	require.True(t, r.Exists(kg.BacklogSet(targetBacklog.BacklogID)))
	require.True(t, hasMember(t, r, kg.ShadowPartitionSet(sp.PartitionID), targetBacklog.BacklogID))
	require.True(t, hasMember(t, r, kg.BacklogSet(targetBacklog.BacklogID), qi.ID))

	require.False(t, r.Exists(kg.BacklogSet(sourceBacklog.BacklogID)))
	require.False(t, hasMember(t, r, kg.ShadowPartitionSet(sp.PartitionID), sourceBacklog.BacklogID), "backlog %s is in shadow partition", sourceBacklog.BacklogID, r.Dump())
	require.False(t, hasMember(t, r, kg.BacklogSet(sourceBacklog.BacklogID), qi.ID))
}

func TestQueueBacklogNormalizationWithRewrite(t *testing.T) {
	// prep
	r, rc := initRedis(t)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	hashedConcurrencyKeyExpr := hashConcurrencyKey("event.data.customerId")
	unhashedValue := "customer1"
	scope := enums.ConcurrencyScopeFn
	entity := fnID
	fullKey := util.ConcurrencyKey(scope, entity, unhashedValue)

	customConc := []state.CustomConcurrency{
		{
			Key:                       fullKey,
			Hash:                      hashedConcurrencyKeyExpr,
			Limit:                     123,
			UnhashedEvaluatedKeyValue: unhashedValue,
		},
	}

	throttleKey := util.XXHash("customer1")
	throttleKeyExpr := util.XXHash("event.data.customerId")

	throttle := &osqueue.Throttle{
		Key:                 throttleKey,
		Limit:               100,
		Burst:               10,
		Period:              int(time.Hour.Seconds()),
		UnhashedThrottleKey: unhashedValue,
		KeyExpressionHash:   throttleKeyExpr,
	}

	q, shard := newQueue(
		t, rc,
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
			return true
		}),
		osqueue.WithNormalizeRefreshItemCustomConcurrencyKeys(func(ctx context.Context, item *osqueue.QueueItem, existingKeys []state.CustomConcurrency, shadowPartition *osqueue.QueueShadowPartition) ([]state.CustomConcurrency, error) {
			return customConc, nil
		}),
		osqueue.WithRefreshItemThrottle(func(ctx context.Context, item *osqueue.QueueItem) (*osqueue.Throttle, error) {
			return throttle, nil
		}),
		osqueue.WithClock(clock),
	)
	kg := shard.Client().kg
	ctx := context.Background()

	require.Len(t, r.Keys(), 0)

	item := osqueue.QueueItem{
		FunctionID:  fnID,
		WorkspaceID: wsID,
		Data: osqueue.Item{
			WorkspaceID: wsID,
			Kind:        osqueue.KindStart,
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

	item2 := osqueue.QueueItem{
		FunctionID:  fnID,
		WorkspaceID: wsID,
		Data: osqueue.Item{
			WorkspaceID: wsID,
			Kind:        osqueue.KindStart,
			Identifier: state.Identifier{
				WorkflowID:  fnID,
				AccountID:   accountId,
				WorkspaceID: wsID,
			},
			QueueName:             nil,
			Throttle:              throttle,
			CustomConcurrencyKeys: customConc,
		},
		QueueName: nil,
	}

	// Create backlog
	for i := range 10 {
		at := clock.Now().Add(time.Duration(i*100) * time.Millisecond)
		_, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)
	}

	//
	//   Test cases
	//

	// Verify backlog is created as expected
	initialBacklog := osqueue.ItemBacklog(ctx, item)
	require.NotEmpty(t, initialBacklog.BacklogID)
	require.Nil(t, initialBacklog.ConcurrencyKeys)
	require.Nil(t, initialBacklog.Throttle)

	targetBacklog := osqueue.ItemBacklog(ctx, item2)
	require.NotEmpty(t, targetBacklog.BacklogID)
	require.NotNil(t, targetBacklog.ConcurrencyKeys)
	require.NotNil(t, targetBacklog.Throttle)

	shadowPartition := osqueue.ItemShadowPartition(ctx, item)
	require.NotEmpty(t, shadowPartition.PartitionID)

	// Mark backlog for normalization
	err := shard.BacklogPrepareNormalize(ctx, &initialBacklog, &shadowPartition)
	require.NoError(t, err)
	require.Equal(t, 10, zcard(t, rc, kg.BacklogSet(initialBacklog.BacklogID)))
	require.Equal(t, 0, zcard(t, rc, kg.BacklogSet(targetBacklog.BacklogID)))
	require.True(t, hasMember(t, r, kg.GlobalAccountNormalizeSet(), accountId.String()))
	require.True(t, hasMember(t, r, kg.AccountNormalizeSet(accountId), fnID.String()))
	require.True(t, hasMember(t, r, kg.PartitionNormalizeSet(fnID.String()), initialBacklog.BacklogID))

	// Verify normalization
	require.NoError(t, shard.LeaseBacklogForNormalization(ctx, &initialBacklog)) // lease it first

	constraints := osqueue.PartitionConstraintConfig{}

	require.NoError(t, q.NormalizeBacklog(ctx, &initialBacklog, &shadowPartition, constraints))

	require.Equal(t, 0, zcard(t, rc, kg.BacklogSet(initialBacklog.BacklogID)))
	require.Equal(t, 10, zcard(t, rc, kg.BacklogSet(targetBacklog.BacklogID)))

	require.False(t, hasMember(t, r, kg.GlobalAccountNormalizeSet(), accountId.String()))
	require.False(t, hasMember(t, r, kg.AccountNormalizeSet(accountId), fnID.String()))
	require.False(t, hasMember(t, r, kg.PartitionNormalizeSet(fnID.String()), initialBacklog.BacklogID))
}

func TestBacklogNormalizeItemWithSingleton(t *testing.T) {
	r, rc := initRedis(t)
	defer r.Close()

	clock := clockwork.NewFakeClock()

	accountID, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	hashedConcurrencyKeyExpr := hashConcurrencyKey("event.data.customerId")
	unhashedValue := "customer1"
	scope := enums.ConcurrencyScopeFn
	entity := fnID
	fullKey := util.ConcurrencyKey(scope, entity, unhashedValue)

	customConc := []state.CustomConcurrency{
		{
			Key:                       fullKey,
			Hash:                      hashedConcurrencyKeyExpr,
			Limit:                     123,
			UnhashedEvaluatedKeyValue: unhashedValue,
		},
	}

	throttleKey := util.XXHash("customer1")
	throttleKeyExpr := util.XXHash("event.data.customerId")

	throttle := &osqueue.Throttle{
		Key:                 throttleKey,
		Limit:               100,
		Burst:               10,
		Period:              int(time.Hour.Seconds()),
		UnhashedThrottleKey: unhashedValue,
		KeyExpressionHash:   throttleKeyExpr,
	}

	latestConstraints := osqueue.PartitionConstraintConfig{}

	q, shard := newQueue(
		t, rc,
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
			return true
		}),
		osqueue.WithNormalizeRefreshItemCustomConcurrencyKeys(func(ctx context.Context, item *osqueue.QueueItem, existingKeys []state.CustomConcurrency, shadowPartition *osqueue.QueueShadowPartition) ([]state.CustomConcurrency, error) {
			return customConc, nil
		}),
		osqueue.WithRefreshItemThrottle(func(ctx context.Context, item *osqueue.QueueItem) (*osqueue.Throttle, error) {
			return throttle, nil
		}),
		osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
			return latestConstraints
		}),
		osqueue.WithClock(clock),
	)
	kg := shard.Client().kg
	ctx := context.Background()

	require.Len(t, r.Keys(), 0)

	runID := ulid.MustNew(ulid.Now(), rand.Reader)

	item := osqueue.QueueItem{
		FunctionID:  fnID,
		WorkspaceID: wsID,
		Data: osqueue.Item{
			WorkspaceID: wsID,
			Kind:        osqueue.KindStart,
			Identifier: state.Identifier{
				WorkflowID:  fnID,
				AccountID:   accountID,
				WorkspaceID: wsID,
				RunID:       runID,
			},
			Singleton: &osqueue.Singleton{
				Mode: enums.SingletonModeCancel,
				Key:  "singleton-key",
			},
			QueueName:             nil,
			Throttle:              nil,
			CustomConcurrencyKeys: nil,
		},
		QueueName: nil,
	}

	sp := osqueue.ItemShadowPartition(ctx, item)
	sourceBacklog := osqueue.ItemBacklog(ctx, item)

	qi, err := shard.EnqueueItem(ctx, item, clock.Now(), osqueue.EnqueueOpts{})
	require.NoError(t, err)

	require.True(t, r.Exists(kg.BacklogSet(sourceBacklog.BacklogID)))
	require.True(t, hasMember(t, r, kg.ShadowPartitionSet(sp.PartitionID), sourceBacklog.BacklogID))
	require.True(t, hasMember(t, r, kg.BacklogSet(sourceBacklog.BacklogID), qi.ID))

	normalizedItem, err := q.NormalizeItem(ctx, &sp, latestConstraints, &sourceBacklog, qi)
	require.NoError(t, err)

	qi.Data.CustomConcurrencyKeys = customConc
	qi.Data.Throttle = throttle

	require.Equal(t, qi, normalizedItem)
	targetBacklog := osqueue.ItemBacklog(ctx, qi)

	actualBacklog := osqueue.ItemBacklog(ctx, normalizedItem)

	require.Equal(t, targetBacklog, actualBacklog)

	require.True(t, r.Exists(kg.BacklogSet(targetBacklog.BacklogID)))
	require.True(t, hasMember(t, r, kg.ShadowPartitionSet(sp.PartitionID), targetBacklog.BacklogID))
	require.True(t, hasMember(t, r, kg.BacklogSet(targetBacklog.BacklogID), qi.ID))

	require.False(t, r.Exists(kg.BacklogSet(sourceBacklog.BacklogID)))
	require.False(t, hasMember(t, r, kg.ShadowPartitionSet(sp.PartitionID), sourceBacklog.BacklogID), "backlog %s is in shadow partition", sourceBacklog.BacklogID, r.Dump())
	require.False(t, hasMember(t, r, kg.BacklogSet(sourceBacklog.BacklogID), qi.ID))
}
