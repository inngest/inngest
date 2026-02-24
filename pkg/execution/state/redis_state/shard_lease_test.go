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

	t.Run("can renew an expired lease", func(t *testing.T) {
		<-time.After(100 * time.Millisecond)

		now := time.Now()
		dur := time.Second
		leaseID, err = shard.ShardLease(ctx, "shard", dur, 1, leaseID)
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		require.WithinDuration(t, now.Add(dur), ulid.Time(leaseID.Time()), 5*time.Millisecond)
	})

	t.Run("extend an unexpired lease", func(t *testing.T) {

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

func TestReleaseShardLease(t *testing.T) {
	ctx := context.Background()
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	_, shard := newQueue(t, rc)

	t.Run("releasing a non-existent lease is a no-op", func(t *testing.T) {
		nonExistent := ulid.MustNew(ulid.Now(), rnd)
		err := shard.ReleaseShardLease(ctx, "shard", nonExistent)
		require.NoError(t, err)
	})

	t.Run("release a valid lease frees the slot", func(t *testing.T) {
		// Claim a lease with maxLeases=1
		leaseID, err := shard.ShardLease(ctx, "shard", 5*time.Second, 1)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		// Slot is full, cannot claim another
		_, err = shard.ShardLease(ctx, "shard", 5*time.Second, 1)
		require.Equal(t, osqueue.ErrAllShardsAlreadyLeased, err)

		// Release the lease
		err = shard.ReleaseShardLease(ctx, "shard", *leaseID)
		require.NoError(t, err)

		// Now a new lease can be claimed
		newLeaseID, err := shard.ShardLease(ctx, "shard", 5*time.Second, 1)
		require.NoError(t, err)
		require.NotNil(t, newLeaseID)

		// Clean up
		err = shard.ReleaseShardLease(ctx, "shard", *newLeaseID)
		require.NoError(t, err)
	})

	t.Run("releasing an already released lease is a no-op", func(t *testing.T) {
		leaseID, err := shard.ShardLease(ctx, "shard", 5*time.Second, 1)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		// Release once
		err = shard.ReleaseShardLease(ctx, "shard", *leaseID)
		require.NoError(t, err)

		// Release again is a no-op
		err = shard.ReleaseShardLease(ctx, "shard", *leaseID)
		require.NoError(t, err)
	})

	t.Run("releasing an expired lease is a no-op", func(t *testing.T) {
		dur := 50 * time.Millisecond
		leaseID, err := shard.ShardLease(ctx, "shard", dur, 1)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		// Wait for it to expire
		<-time.After(100 * time.Millisecond)

		err = shard.ReleaseShardLease(ctx, "shard", *leaseID)
		require.NoError(t, err)
	})

	t.Run("release one of multiple leases", func(t *testing.T) {
		// Claim two leases (maxLeases=2)
		leaseA, err := shard.ShardLease(ctx, "shard-multi", 5*time.Second, 2)
		require.NoError(t, err)
		require.NotNil(t, leaseA)

		leaseB, err := shard.ShardLease(ctx, "shard-multi", 5*time.Second, 2)
		require.NoError(t, err)
		require.NotNil(t, leaseB)

		// All slots full
		_, err = shard.ShardLease(ctx, "shard-multi", 5*time.Second, 2)
		require.Equal(t, osqueue.ErrAllShardsAlreadyLeased, err)

		// Release one lease
		err = shard.ReleaseShardLease(ctx, "shard-multi", *leaseA)
		require.NoError(t, err)

		// Can claim one more again
		leaseC, err := shard.ShardLease(ctx, "shard-multi", 5*time.Second, 2)
		require.NoError(t, err)
		require.NotNil(t, leaseC)

		// Still full
		_, err = shard.ShardLease(ctx, "shard-multi", 5*time.Second, 2)
		require.Equal(t, osqueue.ErrAllShardsAlreadyLeased, err)
	})

	t.Run("release does not affect other keys", func(t *testing.T) {
		leaseA, err := shard.ShardLease(ctx, "shard-a", 5*time.Second, 1)
		require.NoError(t, err)
		require.NotNil(t, leaseA)

		leaseB, err := shard.ShardLease(ctx, "shard-b", 5*time.Second, 1)
		require.NoError(t, err)
		require.NotNil(t, leaseB)

		// Release lease on shard-a
		err = shard.ReleaseShardLease(ctx, "shard-a", *leaseA)
		require.NoError(t, err)

		// shard-b should still be leased (cannot claim another)
		_, err = shard.ShardLease(ctx, "shard-b", 5*time.Second, 1)
		require.Equal(t, osqueue.ErrAllShardsAlreadyLeased, err)

		// shard-a should be free
		newLeaseA, err := shard.ShardLease(ctx, "shard-a", 5*time.Second, 1)
		require.NoError(t, err)
		require.NotNil(t, newLeaseA)
	})
}
