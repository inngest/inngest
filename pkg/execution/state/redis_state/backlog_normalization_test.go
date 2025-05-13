package redis_state

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
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

	defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
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

	backlog := &QueueBacklog{
		BacklogID:         "yolo",
		ShadowPartitionID: shadowPart.PartitionID,
		Throttle: &BacklogThrottle{
			ThrottleKey:               "something",
			ThrottleKeyRawValue:       "somethingelse",
			ThrottleKeyExpressionHash: "hash",
		},
	}

	// should lease successfully
	err := q.leaseBacklogForNormalization(ctx, backlog)
	require.NoError(t, err)

	// another attempt should fail
	err = q.leaseBacklogForNormalization(ctx, backlog)
	require.ErrorIs(t, err, errBacklogAlreadyLeasedForNormalization)
}

func TestExtendBacklogNormalizationLease(t *testing.T) {
	ctx := context.Background()

	r, rc := initRedis(t)
	defer rc.Close()

	defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
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

	backlog := &QueueBacklog{
		BacklogID:         "yolo",
		ShadowPartitionID: shadowPart.PartitionID,
		Throttle: &BacklogThrottle{
			ThrottleKey:               "something",
			ThrottleKeyRawValue:       "somethingelse",
			ThrottleKeyExpressionHash: "hash",
		},
	}

	// attempt to extend without a lease will fail
	err := q.extendBacklogNormalizationLease(ctx, clock.Now(), backlog)
	require.ErrorIs(t, err, errBacklogNormalizationLeaseExpired)

	// lease the backlog first
	err = q.leaseBacklogForNormalization(ctx, backlog)
	require.NoError(t, err)

	// should succeed
	err = q.extendBacklogNormalizationLease(ctx, clock.Now(), backlog)
	require.NoError(t, err)

	clock.Advance(2 * BacklogNormalizeLeaseDuration)
	r.FastForward(2 * BacklogNormalizeLeaseDuration)

	// expect lease to be expired again
	err = q.extendBacklogNormalizationLease(ctx, clock.Now(), backlog)
	require.ErrorIs(t, err, errBacklogNormalizationLeaseExpired)
}

func TestQueueBacklogPrepareNormalize(t *testing.T) {
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

	t.Run("should move backlog to normalization set", func(t *testing.T) {
		r.FlushAll()

		_, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		expectedBacklog := q.ItemBacklog(ctx, item)
		require.NotEmpty(t, expectedBacklog.BacklogID)

		shadowPartition := q.ItemShadowPartition(ctx, item)
		require.NotEmpty(t, shadowPartition.PartitionID)
		backlogCount, shouldNormalizeAsync, err := q.BacklogPrepareNormalize(ctx, &expectedBacklog, &shadowPartition, 1)
		require.NoError(t, err)

		require.True(t, shouldNormalizeAsync)
		require.Equal(t, 1, backlogCount)
		require.True(t, hasMember(t, r, kg.GlobalAccountNormalizeSet(), accountId.String()))
		require.True(t, hasMember(t, r, kg.AccountNormalizeSet(accountId), fnID.String()))
		require.True(t, hasMember(t, r, kg.PartitionNormalizeSet(fnID.String()), expectedBacklog.BacklogID))

		expectedTime := clock.Now().Unix()

		require.Equal(t, expectedTime, int64(score(t, r, kg.GlobalAccountNormalizeSet(), accountId.String())))
		require.Equal(t, expectedTime, int64(score(t, r, kg.AccountNormalizeSet(accountId), fnID.String())))
		require.Equal(t, expectedTime, int64(score(t, r, kg.PartitionNormalizeSet(fnID.String()), expectedBacklog.BacklogID)))
	})

	t.Run("should not move if below limit", func(t *testing.T) {
		r.FlushAll()

		_, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		expectedBacklog := q.ItemBacklog(ctx, item)
		require.NotEmpty(t, expectedBacklog.BacklogID)

		shadowPartition := q.ItemShadowPartition(ctx, item)
		require.NotEmpty(t, shadowPartition.PartitionID)
		backlogCount, shouldNormalizeAsync, err := q.BacklogPrepareNormalize(ctx, &expectedBacklog, &shadowPartition, 5)
		require.NoError(t, err)

		require.False(t, shouldNormalizeAsync)
		require.Equal(t, 1, backlogCount)
		require.False(t, hasMember(t, r, kg.GlobalAccountNormalizeSet(), accountId.String()))
		require.False(t, hasMember(t, r, kg.AccountNormalizeSet(accountId), fnID.String()))
		require.False(t, hasMember(t, r, kg.PartitionNormalizeSet(fnID.String()), expectedBacklog.BacklogID))
	})
}

func TestQueueBacklogNormalization(t *testing.T) {
	// prep
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
		_, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)
	}

	//
	//   Test cases
	//

	// Verify backlog is created as expected
	backlog := q.ItemBacklog(ctx, item)
	require.NotEmpty(t, backlog.BacklogID)

	shadowPartition := q.ItemShadowPartition(ctx, item)
	require.NotEmpty(t, shadowPartition.PartitionID)

	// Mark backlog for normalization
	backlogCount, shouldNormalizeAsync, err := q.BacklogPrepareNormalize(ctx, &backlog, &shadowPartition, 5)
	require.NoError(t, err)
	require.True(t, shouldNormalizeAsync)
	require.Equal(t, 10, backlogCount)
	require.True(t, hasMember(t, r, kg.GlobalAccountNormalizeSet(), accountId.String()))
	require.True(t, hasMember(t, r, kg.AccountNormalizeSet(accountId), fnID.String()))
	require.True(t, hasMember(t, r, kg.PartitionNormalizeSet(fnID.String()), backlog.BacklogID))

	// Verify normalization
	require.NoError(t, q.leaseBacklogForNormalization(ctx, &backlog)) // lease it first

	require.NoError(t, q.normalizeBacklog(ctx, &backlog))
	require.False(t, hasMember(t, r, kg.GlobalAccountNormalizeSet(), accountId.String()))
	require.False(t, hasMember(t, r, kg.AccountNormalizeSet(accountId), fnID.String()))
	require.False(t, hasMember(t, r, kg.PartitionNormalizeSet(fnID.String()), backlog.BacklogID))
}

// TODO
// func TestBacklogNormalizationScanner(t *testing.T) {}
