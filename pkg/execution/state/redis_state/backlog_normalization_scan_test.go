package redis_state

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func normalizeTestItem(accountID, fnID, wsID uuid.UUID) osqueue.QueueItem {
	return osqueue.QueueItem{
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
		},
	}
}

func newNormalizeTestQueue(t testing.TB) (*miniredis.Miniredis, *clockwork.FakeClock, queueImpl, RedisQueueShard) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	t.Cleanup(rc.Close)

	clock := clockwork.NewFakeClock()
	q, shard := newQueue(
		t, rc,
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
			return true
		}),
		osqueue.WithClock(clock),
	)
	return r, clock, q, shard
}

// enqueueAndPrepareNormalize enqueues n items for a new function in the account and marks the
// function's only backlog for normalization, returning its shadow partition and backlog.
func enqueueAndPrepareNormalize(t testing.TB, ctx context.Context, clock clockwork.Clock, shard RedisQueueShard, accountID uuid.UUID, n int) (osqueue.QueueShadowPartition, osqueue.QueueBacklog) {
	item := normalizeTestItem(accountID, uuid.New(), uuid.New())
	for i := range n {
		_, err := shard.EnqueueItem(ctx, item, clock.Now().Add(time.Duration(i)*time.Millisecond), osqueue.EnqueueOpts{})
		require.NoError(t, err)
	}

	sp := osqueue.ItemShadowPartition(ctx, item)
	bl := osqueue.ItemBacklog(ctx, item)
	require.NoError(t, shard.BacklogPrepareNormalize(ctx, &bl, &sp))
	return sp, bl
}

// enqueueActivePartitions enqueues one item for each of n new functions in the account, so the
// account has n active shadow partitions with nothing to normalize.
func enqueueActivePartitions(t testing.TB, ctx context.Context, clock clockwork.Clock, shard RedisQueueShard, accountID uuid.UUID, n int) {
	for range n {
		item := normalizeTestItem(accountID, uuid.New(), uuid.New())
		_, err := shard.EnqueueItem(ctx, item, clock.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)
	}
}

func partitionIDs(sps []*osqueue.QueueShadowPartition) []string {
	ids := make([]string, 0, len(sps))
	for _, sp := range sps {
		ids = append(ids, sp.PartitionID)
	}
	return ids
}

func TestPeekAccountNormalizePartitions(t *testing.T) {
	ctx := context.Background()

	t.Run("returns only partitions pending normalization", func(t *testing.T) {
		r, clock, _, shard := newNormalizeTestQueue(t)
		kg := shard.Client().kg
		accountID := uuid.New()

		enqueueActivePartitions(t, ctx, clock, shard, accountID, 5)
		sp, _ := enqueueAndPrepareNormalize(t, ctx, clock, shard, accountID, 3)

		// The partition's only backlog moved to the normalize set, so the partition is no
		// longer in the account's shadow partition set. Scanning that set never finds it.
		require.False(t, hasMember(t, r, kg.AccountShadowPartitions(accountID), sp.PartitionID))
		active, err := shard.PeekShadowPartitions(ctx, &accountID, true, osqueue.ShadowPartitionPeekMax, clock.Now())
		require.NoError(t, err)
		require.Len(t, active, 5)
		require.NotContains(t, partitionIDs(active), sp.PartitionID)

		res, err := shard.PeekAccountNormalizePartitions(ctx, accountID, clock.Now(), osqueue.ShadowPartitionPeekMax)
		require.NoError(t, err)
		require.Equal(t, []string{sp.PartitionID}, partitionIDs(res))

		// Valid pointers are left alone.
		require.True(t, hasMember(t, r, kg.AccountNormalizeSet(accountID), sp.PartitionID))
		require.True(t, hasMember(t, r, kg.GlobalAccountNormalizeSet(), accountID.String()))
	})

	t.Run("drops pointer to partition without metadata", func(t *testing.T) {
		r, clock, _, shard := newNormalizeTestQueue(t)
		kg := shard.Client().kg
		accountID := uuid.New()

		sp, _ := enqueueAndPrepareNormalize(t, ctx, clock, shard, accountID, 1)
		missingPartitionID := uuid.NewString()
		_, err := r.ZAdd(kg.AccountNormalizeSet(accountID), float64(clock.Now().UnixMilli()), missingPartitionID)
		require.NoError(t, err)

		res, err := shard.PeekAccountNormalizePartitions(ctx, accountID, clock.Now(), osqueue.ShadowPartitionPeekMax)
		require.NoError(t, err)
		require.Equal(t, []string{sp.PartitionID}, partitionIDs(res))

		require.False(t, hasMember(t, r, kg.AccountNormalizeSet(accountID), missingPartitionID))
		// The account still has a valid partition to normalize.
		require.True(t, hasMember(t, r, kg.GlobalAccountNormalizeSet(), accountID.String()))
	})

	t.Run("drops account once its last partition pointer is missing", func(t *testing.T) {
		r, clock, _, shard := newNormalizeTestQueue(t)
		kg := shard.Client().kg
		accountID := uuid.New()

		now := float64(clock.Now().UnixMilli())
		_, err := r.ZAdd(kg.AccountNormalizeSet(accountID), now, uuid.NewString())
		require.NoError(t, err)
		_, err = r.ZAdd(kg.GlobalAccountNormalizeSet(), now, accountID.String())
		require.NoError(t, err)

		res, err := shard.PeekAccountNormalizePartitions(ctx, accountID, clock.Now(), osqueue.ShadowPartitionPeekMax)
		require.NoError(t, err)
		require.Empty(t, res)

		require.False(t, r.Exists(kg.AccountNormalizeSet(accountID)))
		require.False(t, hasMember(t, r, kg.GlobalAccountNormalizeSet(), accountID.String()))
	})

	t.Run("drops account with empty normalize set", func(t *testing.T) {
		r, clock, _, shard := newNormalizeTestQueue(t)
		kg := shard.Client().kg
		accountID := uuid.New()

		_, err := r.ZAdd(kg.GlobalAccountNormalizeSet(), float64(clock.Now().UnixMilli()), accountID.String())
		require.NoError(t, err)

		res, err := shard.PeekAccountNormalizePartitions(ctx, accountID, clock.Now(), osqueue.ShadowPartitionPeekMax)
		require.NoError(t, err)
		require.Empty(t, res)
		require.False(t, hasMember(t, r, kg.GlobalAccountNormalizeSet(), accountID.String()))
	})

	t.Run("keeps account with partitions due in the future", func(t *testing.T) {
		r, clock, _, shard := newNormalizeTestQueue(t)
		kg := shard.Client().kg
		accountID := uuid.New()

		sp, _ := enqueueAndPrepareNormalize(t, ctx, clock, shard, accountID, 1)

		res, err := shard.PeekAccountNormalizePartitions(ctx, accountID, clock.Now().Add(-time.Minute), osqueue.ShadowPartitionPeekMax)
		require.NoError(t, err)
		require.Empty(t, res)

		require.True(t, hasMember(t, r, kg.AccountNormalizeSet(accountID), sp.PartitionID))
		require.True(t, hasMember(t, r, kg.GlobalAccountNormalizeSet(), accountID.String()))
	})

	t.Run("rejects limit above max", func(t *testing.T) {
		_, clock, _, shard := newNormalizeTestQueue(t)

		_, err := shard.PeekAccountNormalizePartitions(ctx, uuid.New(), clock.Now(), osqueue.ShadowPartitionPeekMax+1)
		require.ErrorIs(t, err, osqueue.ErrShadowPartitionPeekMaxExceedsLimits)
	})
}

func TestNormalizePointerCleanup(t *testing.T) {
	ctx := context.Background()

	t.Run("partition peek drops pointers when all backlogs are gone", func(t *testing.T) {
		r, clock, _, shard := newNormalizeTestQueue(t)
		kg := shard.Client().kg
		accountID := uuid.New()

		sp, bl := enqueueAndPrepareNormalize(t, ctx, clock, shard, accountID, 1)
		// Backlog metadata disappears, e.g. garbage collected elsewhere.
		r.HDel(kg.BacklogMeta(), bl.BacklogID)

		backlogs, err := shard.ShadowPartitionPeekNormalizeBacklogs(ctx, &sp, osqueue.NormalizePartitionPeekMax)
		require.NoError(t, err)
		require.Empty(t, backlogs)

		require.False(t, hasMember(t, r, kg.PartitionNormalizeSet(sp.PartitionID), bl.BacklogID))
		require.False(t, hasMember(t, r, kg.AccountNormalizeSet(accountID), sp.PartitionID))
		require.False(t, hasMember(t, r, kg.GlobalAccountNormalizeSet(), accountID.String()))
	})

	t.Run("partition peek keeps pointers while backlogs remain", func(t *testing.T) {
		r, clock, _, shard := newNormalizeTestQueue(t)
		kg := shard.Client().kg
		accountID := uuid.New()

		sp, bl := enqueueAndPrepareNormalize(t, ctx, clock, shard, accountID, 1)

		backlogs, err := shard.ShadowPartitionPeekNormalizeBacklogs(ctx, &sp, osqueue.NormalizePartitionPeekMax)
		require.NoError(t, err)
		require.Len(t, backlogs, 1)
		require.Equal(t, bl.BacklogID, backlogs[0].BacklogID)

		require.True(t, hasMember(t, r, kg.AccountNormalizeSet(accountID), sp.PartitionID))
		require.True(t, hasMember(t, r, kg.GlobalAccountNormalizeSet(), accountID.String()))
	})

	t.Run("empty backlog is dropped and partition cleaned up on next peek", func(t *testing.T) {
		r, clock, _, shard := newNormalizeTestQueue(t)
		kg := shard.Client().kg
		accountID := uuid.New()

		sp, bl := enqueueAndPrepareNormalize(t, ctx, clock, shard, accountID, 1)
		// Empty the backlog without going through enqueue, which is the only path that
		// cleaned up normalize pointers before.
		r.Del(kg.BacklogSet(bl.BacklogID))

		res, err := shard.BacklogNormalizePeek(ctx, &bl, osqueue.NormalizeBacklogPeekMax)
		require.NoError(t, err)
		require.Zero(t, res.TotalCount)
		require.False(t, hasMember(t, r, kg.PartitionNormalizeSet(sp.PartitionID), bl.BacklogID))
		// The account is unknown to BacklogNormalizePeek, so the account pointer remains...
		require.True(t, hasMember(t, r, kg.AccountNormalizeSet(accountID), sp.PartitionID))

		// ...until the scanner peeks the partition again.
		backlogs, err := shard.ShadowPartitionPeekNormalizeBacklogs(ctx, &sp, osqueue.NormalizePartitionPeekMax)
		require.NoError(t, err)
		require.Empty(t, backlogs)
		require.False(t, hasMember(t, r, kg.AccountNormalizeSet(accountID), sp.PartitionID))
		require.False(t, hasMember(t, r, kg.GlobalAccountNormalizeSet(), accountID.String()))
	})

	t.Run("full normalization clears all pointers", func(t *testing.T) {
		r, clock, q, shard := newNormalizeTestQueue(t)
		kg := shard.Client().kg
		accountID := uuid.New()

		sp, bl := enqueueAndPrepareNormalize(t, ctx, clock, shard, accountID, 5)

		res, err := shard.PeekAccountNormalizePartitions(ctx, accountID, clock.Now(), osqueue.ShadowPartitionPeekMax)
		require.NoError(t, err)
		require.Equal(t, []string{sp.PartitionID}, partitionIDs(res))

		require.NoError(t, shard.LeaseBacklogForNormalization(ctx, &bl))
		require.NoError(t, q.NormalizeBacklog(ctx, &bl, res[0], osqueue.PartitionConstraintConfig{}))

		require.Equal(t, 0, zcard(t, shard.Client().Client(), kg.BacklogSet(bl.BacklogID)))
		require.False(t, hasMember(t, r, kg.PartitionNormalizeSet(sp.PartitionID), bl.BacklogID))
		require.False(t, hasMember(t, r, kg.AccountNormalizeSet(accountID), sp.PartitionID))
		require.False(t, hasMember(t, r, kg.GlobalAccountNormalizeSet(), accountID.String()))
	})
}

// legacyShadowPartitionPeekNormalizeBacklogs mirrors ShadowPartitionPeekNormalizeBacklogs before
// pointer cleanup was added, so the benchmark measures the old scan path as it ran.
func legacyShadowPartitionPeekNormalizeBacklogs(ctx context.Context, q *queue, sp *osqueue.QueueShadowPartition) ([]*osqueue.QueueBacklog, error) {
	p := peeker[osqueue.QueueBacklog]{
		q:               q,
		opName:          "ShadowPartitionPeekNormalizeBacklogs",
		keyMetadataHash: q.RedisClient.kg.BacklogMeta(),
		max:             osqueue.NormalizePartitionPeekMax,
		maker: func() *osqueue.QueueBacklog {
			return &osqueue.QueueBacklog{}
		},
		ignoreUntil:            true,
		isMillisecondPrecision: true,
	}
	res, err := p.peek(ctx, q.RedisClient.kg.PartitionNormalizeSet(sp.PartitionID), false, q.Clock.Now(), osqueue.NormalizePartitionPeekMax)
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

// BenchmarkNormalizationScanAccount measures one scanner pass over an account that has one
// partition pending normalization and N other active partitions, which is the shape that kept
// the normalization scanner busy in production.
func BenchmarkNormalizationScanAccount(b *testing.B) {
	ctx := context.Background()

	for _, active := range []int{10, 100, 299} {
		r, clock, _, shard := newNormalizeTestQueue(b)
		q, ok := shard.(*queue)
		require.True(b, ok)

		accountID := uuid.New()
		enqueueActivePartitions(b, ctx, clock, shard, accountID, active)
		enqueueAndPrepareNormalize(b, ctx, clock, shard, accountID, 1)

		b.Run(fmt.Sprintf("shadow_partitions/active=%d", active), func(b *testing.B) {
			start := r.CommandCount()
			b.ResetTimer()
			for b.Loop() {
				sps, err := shard.PeekShadowPartitions(ctx, &accountID, false, osqueue.ShadowPartitionPeekMax, clock.Now())
				require.NoError(b, err)
				found := 0
				for _, sp := range sps {
					backlogs, err := legacyShadowPartitionPeekNormalizeBacklogs(ctx, q, sp)
					require.NoError(b, err)
					found += len(backlogs)
				}
				// The partition that needs normalization is never reached.
				require.Zero(b, found)
			}
			b.ReportMetric(float64(r.CommandCount()-start)/float64(b.N), "redis_cmds/op")
		})

		b.Run(fmt.Sprintf("account_normalize_set/active=%d", active), func(b *testing.B) {
			start := r.CommandCount()
			b.ResetTimer()
			for b.Loop() {
				sps, err := shard.PeekAccountNormalizePartitions(ctx, accountID, clock.Now(), osqueue.ShadowPartitionPeekMax)
				require.NoError(b, err)
				found := 0
				for _, sp := range sps {
					backlogs, err := shard.ShadowPartitionPeekNormalizeBacklogs(ctx, sp, osqueue.NormalizePartitionPeekMax)
					require.NoError(b, err)
					found += len(backlogs)
				}
				require.Equal(b, 1, found)
			}
			b.ReportMetric(float64(r.CommandCount()-start)/float64(b.N), "redis_cmds/op")
		})
	}
}
