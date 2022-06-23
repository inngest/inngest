package redis_state

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/inngest/inngest-cli/pkg/execution/state"
	"github.com/inngest/inngest-cli/pkg/execution/state/testharness"
)

func TestStateHarness(t *testing.T) {
	create := func() state.Manager {
		s := miniredis.RunT(t)
		return New(WithConnectOpts(redis.Options{
			Addr: s.Addr(),
		}))
	}
	testharness.CheckState(t, create)
}
