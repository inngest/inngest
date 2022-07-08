package redis_state

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/inngest/inngest-cli/pkg/execution/state"
	"github.com/inngest/inngest-cli/pkg/execution/state/testharness"
)

func TestStateHarness(t *testing.T) {
	r := miniredis.RunT(t)
	sm := New(WithConnectOpts(redis.Options{
		Addr: r.Addr(),
		// Make the pool size less than the 100 concurrent items we run,
		// to ensure contention works.
		//
		// NOTE: Sometimes, when running with the race detector,
		// we'll hit an internal 8128 goroutine limit.  See:
		// https://github.com/golang/go/issues/47056
		PoolSize: 75,
	}))

	create := func() (state.Manager, func()) {
		return sm, func() {
			r.FlushAll()
		}
	}

	testharness.CheckState(t, create)
}
