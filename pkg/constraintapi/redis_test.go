package constraintapi

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestRedisCapacityManager(t *testing.T) {
	r := miniredis.RunT(t)
	ctx := context.Background()

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	cm, err := NewRedisCapacityManager(
		WithClient(rc),
		WithClock(clock),
	)
	require.NoError(t, err)
	require.NotNil(t, cm)

	// The following tests are essential functionality. We also have detailed test for each method,
	// to cover edge cases.

	t.Run("Acquire", func(t *testing.T) {
		resp, err := cm.Acquire(ctx, &CapacityAcquireRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("Check", func(t *testing.T) {
		resp, userErr, internalErr := cm.Check(ctx, &CapacityCheckRequest{})
		require.NoError(t, userErr)
		require.NoError(t, internalErr)
		require.NotNil(t, resp)
	})

	t.Run("Extend", func(t *testing.T) {
		resp, err := cm.ExtendLease(ctx, &CapacityExtendLeaseRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("Release", func(t *testing.T) {
		resp, err := cm.Release(ctx, &CapacityReleaseRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}
