package redis_state

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/testharness"
	"github.com/oklog/ulid/v2"
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

func BenchmarkNew(b *testing.B) {
	r := miniredis.RunT(b)
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
	require.NoError(b, err)

	id := state.Identifier{
		WorkflowID: uuid.New(),
	}
	init := state.Input{
		Identifier: id,
		Workflow:   inngest.Workflow{},
		EventData: event.Event{
			Name: "test-event",
			Data: map[string]any{
				"title": "They don't think it be like it is, but it do",
				"data": map[string]any{
					"float": 3.14132,
				},
			},
			User: map[string]any{
				"external_id": "1",
			},
			Version: "1985-01-01",
		}.Map(),
	}

	ctx := context.Background()
	for n := 0; n < b.N; n++ {
		init.Identifier.RunID = ulid.MustNew(ulid.Now(), rand.Reader)
		_, err := sm.New(ctx, init)
		require.NoError(b, err)
	}

}
