package redis_state

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/testharness"
	"github.com/stretchr/testify/require"
)

func TestStateHarness(t *testing.T) {
	r := miniredis.RunT(t)
	sm, err := New(
		context.Background(),
		WithConnectOpts(redis.Options{
			Addr: r.Addr(),
			// Make the pool size less than the 100 concurrent items we run,
			// to ensure contention works.
			//
			// NOTE: Sometimes, when running with the race detector,
			// we'll hit an internal 8128 goroutine limit.  See:
			// https://github.com/golang/go/issues/47056
			PoolSize: 75,
		}),
	)
	require.NoError(t, err)

	create := func() (state.Manager, func()) {
		return sm, func() {
			r.FlushAll()
		}
	}

	testharness.CheckState(t, create)
}
