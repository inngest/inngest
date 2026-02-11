package redis_state

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestQueueShardLease(t *testing.T) {
	ctx := context.Background()
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	_, shard := newQueue(t, rc)

	var leaseID *ulid.ULID

	t.Run("cannot claim lease when max leases is 0", func(t *testing.T) {
		leaseID, err = shard.ShardLease(ctx, "shard", 500*time.Millisecond, 0)
		require.Equal(t, osqueue.ErrAllShardsAlreadyLeased, err)
		require.Nil(t, leaseID)
	})

	t.Run("cannot claim lease for greater than allowed duration", func(t *testing.T) {
		leaseID, err = shard.ShardLease(ctx, "shard", 30*time.Second, 1)
		require.Equal(t, osqueue.ErrShardLeaseExceedsLimits, err)
		require.Nil(t, leaseID)
	})

	t.Run("claim a shard lease", func(t *testing.T) {
		now := time.Now()
		dur := 100 * time.Millisecond
		leaseID, err = shard.ShardLease(ctx, "shard", dur, 1)
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		require.WithinDuration(t, now.Add(dur), ulid.Time(leaseID.Time()), 5*time.Millisecond)
	})

	t.Run("cannot extend lease without an existing lease ID", func(t *testing.T) {
		id, err := shard.ShardLease(ctx, "shard", time.Second, 1)
		require.Equal(t, osqueue.ErrAllShardsAlreadyLeased, err)
		require.Nil(t, id)
	})

	t.Run("cannot renew an invalid lease", func(t *testing.T) {
		newULID := ulid.MustNew(ulid.Now(), rnd)
		id, err := shard.ShardLease(ctx, "shard", time.Second, 1, &newULID)
		require.Equal(t, osqueue.ErrShardLeaseNotFound, err)
		require.Nil(t, id)
	})

	t.Run("cannot renew an expired lease", func(t *testing.T) {
		<-time.After(100 * time.Millisecond)

		id, err := shard.ShardLease(ctx, "shard", time.Second, 1, leaseID)
		require.Equal(t, osqueue.ErrShardLeaseExpired, err)
		require.Nil(t, id)
	})

	t.Run("extend an unexpired lease", func(t *testing.T) {
		leaseID, err = shard.ShardLease(ctx, "shard", time.Second, 1)
		require.NotNil(t, leaseID)

		now := time.Now()
		dur := 50 * time.Millisecond
		leaseID, err = shard.ShardLease(ctx, "shard", dur, 1, leaseID)
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		require.WithinDuration(t, now.Add(dur), ulid.Time(leaseID.Time()), 5*time.Millisecond)
	})

	t.Run("can get a new lease after previous one expires", func(t *testing.T) {
		<-time.After(100 * time.Millisecond)

		now := time.Now()
		dur := 50 * time.Millisecond
		leaseID, err = shard.ShardLease(ctx, "shard", dur, 1)
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		require.WithinDuration(t, now.Add(dur), ulid.Time(leaseID.Time()), 5*time.Millisecond)
	})
}
