package state_store

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func newStateManager(t *testing.T) state.Manager {
	t.Helper()
	mr := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:       []string{mr.Addr()},
		DisableCache:      true,
		ForceSingleClient: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { rc.Close() })

	unsharded := redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	sharded := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
		UnshardedClient:        unsharded,
		FunctionRunStateClient: rc,
		BatchClient:            rc,
		StateDefaultKey:        redis_state.StateDefaultKey,
		QueueDefaultKey:        redis_state.QueueDefaultKey,
		FnRunIsSharded:         redis_state.NeverShardOnRun,
	})

	m, err := redis_state.New(context.Background(), redis_state.WithShardedClient(sharded))
	require.NoError(t, err)
	return m
}

func createRun(t *testing.T, ctx context.Context, m state.Manager) state.State {
	t.Helper()
	s, err := m.New(ctx, state.Input{
		Identifier: state.Identifier{
			WorkflowID: uuid.New(),
			RunID:      ulid.MustNew(ulid.Now(), rand.Reader),
			Key:        uuid.NewString(),
		},
	})
	require.NoError(t, err)
	return s
}

func pauseFor(id state.Identifier, dataKey string) state.Pause {
	return state.Pause{
		ID: uuid.New(),
		Identifier: state.PauseIdentifier{
			RunID:      id.RunID,
			FunctionID: id.WorkflowID,
			AccountID:  id.AccountID,
		},
		DataKey: dataKey,
	}
}

func TestConsumePauseIdempotencyByData(t *testing.T) {
	ctx := context.Background()

	t.Run("first consume writes data correctly", func(t *testing.T) {
		m := newStateManager(t)
		s := createRun(t, ctx, m)
		pause := pauseFor(s.Identifier(), "step-1")
		data := map[string]any{"id": "evt-123", "name": "test"}

		res, err := m.ConsumePause(ctx, pause, state.ConsumePauseOpts{
			IdempotencyKey: "evt-123",
			Data:           data,
		})
		require.NoError(t, err)
		require.True(t, res.DidConsume)

		reloaded, err := m.Load(ctx, s.Identifier().AccountID, s.RunID())
		require.NoError(t, err)
		require.Equal(t, data, reloaded.Actions()["step-1"])
		require.Equal(t, []string{"step-1"}, reloaded.Stack())
	})

	t.Run("same event retrying does not double-write", func(t *testing.T) {
		m := newStateManager(t)
		s := createRun(t, ctx, m)
		pause := pauseFor(s.Identifier(), "step-1")
		data := map[string]any{"id": "evt-456", "name": "test"}
		opts := state.ConsumePauseOpts{IdempotencyKey: "evt-456", Data: data}

		res, err := m.ConsumePause(ctx, pause, opts)
		require.NoError(t, err)
		require.True(t, res.DidConsume)

		// Retry — the caller needs DidConsume=true to re-enqueue
		res, err = m.ConsumePause(ctx, pause, opts)
		require.NoError(t, err)
		require.True(t, res.DidConsume)

		reloaded, err := m.Load(ctx, s.Identifier().AccountID, s.RunID())
		require.NoError(t, err)
		require.Equal(t, data, reloaded.Actions()["step-1"])
		require.Equal(t, []string{"step-1"}, reloaded.Stack(), "stack should not have duplicates")
	})

	t.Run("same event retrying with pending steps", func(t *testing.T) {
		m := newStateManager(t)
		s := createRun(t, ctx, m)

		// Save a step response first so step-2 is pending after we consume step-1
		_, err := m.SaveResponse(ctx, s.Identifier(), "step-2", "null")
		require.NoError(t, err)

		pause := pauseFor(s.Identifier(), "step-1")
		data := map[string]any{"id": "evt-retry", "name": "test"}
		opts := state.ConsumePauseOpts{IdempotencyKey: "evt-retry", Data: data}

		res, err := m.ConsumePause(ctx, pause, opts)
		require.NoError(t, err)
		require.True(t, res.DidConsume)

		// Retry
		res, err = m.ConsumePause(ctx, pause, opts)
		require.NoError(t, err)
		require.True(t, res.DidConsume)

		reloaded, err := m.Load(ctx, s.Identifier().AccountID, s.RunID())
		require.NoError(t, err)
		require.Contains(t, reloaded.Stack(), "step-1")
		// step-1 should appear only once
		count := 0
		for _, s := range reloaded.Stack() {
			if s == "step-1" {
				count++
			}
		}
		require.Equal(t, 1, count, "stack should not have duplicates")
	})

	// Signals include the signal ID in the payload.
	t.Run("signal retry does not double-write", func(t *testing.T) {
		m := newStateManager(t)
		s := createRun(t, ctx, m)
		pause := pauseFor(s.Identifier(), "signal-step")
		data := map[string]any{
			"data": map[string]any{
				"signal": "my-signal-id",
				"data":   map[string]any{"hello": "world"},
			},
		}
		opts := state.ConsumePauseOpts{IdempotencyKey: "my-signal-id", Data: data}

		res, err := m.ConsumePause(ctx, pause, opts)
		require.NoError(t, err)
		require.True(t, res.DidConsume)

		// Retry after transient failure
		res, err = m.ConsumePause(ctx, pause, opts)
		require.NoError(t, err)
		require.True(t, res.DidConsume)

		reloaded, err := m.Load(ctx, s.Identifier().AccountID, s.RunID())
		require.NoError(t, err)
		require.Equal(t, data, reloaded.Actions()["signal-step"])
		require.Equal(t, []string{"signal-step"}, reloaded.Stack())
	})

	// Invoke finish can have a nil result, producing {"data":null}.
	t.Run("invoke retry with nil result does not double-write", func(t *testing.T) {
		m := newStateManager(t)
		s := createRun(t, ctx, m)
		pause := pauseFor(s.Identifier(), "invoke-step")
		data := map[string]any{"data": nil}
		opts := state.ConsumePauseOpts{IdempotencyKey: "run-123.invoke-step", Data: data}

		res, err := m.ConsumePause(ctx, pause, opts)
		require.NoError(t, err)
		require.True(t, res.DidConsume)

		// Retry after transient failure
		res, err = m.ConsumePause(ctx, pause, opts)
		require.NoError(t, err)
		require.True(t, res.DidConsume)

		reloaded, err := m.Load(ctx, s.Identifier().AccountID, s.RunID())
		require.NoError(t, err)
		require.Equal(t, data, reloaded.Actions()["invoke-step"])
		require.Equal(t, []string{"invoke-step"}, reloaded.Stack())
	})

	// Invoke finish with actual result data.
	t.Run("invoke retry with result data does not double-write", func(t *testing.T) {
		m := newStateManager(t)
		s := createRun(t, ctx, m)
		pause := pauseFor(s.Identifier(), "invoke-step")
		data := map[string]any{"data": map[string]any{"result": "ok", "count": float64(42)}}
		opts := state.ConsumePauseOpts{IdempotencyKey: "run-456.invoke-step", Data: data}

		res, err := m.ConsumePause(ctx, pause, opts)
		require.NoError(t, err)
		require.True(t, res.DidConsume)

		res, err = m.ConsumePause(ctx, pause, opts)
		require.NoError(t, err)
		require.True(t, res.DidConsume)

		reloaded, err := m.Load(ctx, s.Identifier().AccountID, s.RunID())
		require.NoError(t, err)
		require.Equal(t, data, reloaded.Actions()["invoke-step"])
		require.Equal(t, []string{"invoke-step"}, reloaded.Stack())
	})

	t.Run("different event with different payload is rejected", func(t *testing.T) {
		m := newStateManager(t)
		s := createRun(t, ctx, m)
		pause := pauseFor(s.Identifier(), "step-1")

		firstData := map[string]any{"id": "evt-aaa", "name": "resume"}
		res, err := m.ConsumePause(ctx, pause, state.ConsumePauseOpts{
			IdempotencyKey: "evt-aaa",
			Data:           firstData,
		})
		require.NoError(t, err)
		require.True(t, res.DidConsume)

		// Different event
		secondData := map[string]any{"id": "evt-bbb", "name": "resume"}
		res, err = m.ConsumePause(ctx, pause, state.ConsumePauseOpts{
			IdempotencyKey: "evt-bbb",
			Data:           secondData,
		})
		require.NoError(t, err)
		require.False(t, res.DidConsume)

		// Original data preserved
		reloaded, err := m.Load(ctx, s.Identifier().AccountID, s.RunID())
		require.NoError(t, err)
		require.Equal(t, firstData, reloaded.Actions()["step-1"])
	})
}
