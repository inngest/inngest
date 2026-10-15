package executor

import (
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	telemetrytrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestIdempotencyKeyForReplay(t *testing.T) {
	original, first, second := ulid.Make(), ulid.Make(), ulid.Make()
	key := "replay:replay-id:original-run-id"
	req := execution.ScheduleRequest{
		Function:       inngest.Function{ID: uuid.New()},
		OriginalRunID:  &original,
		IdempotencyKey: &key,
	}
	want := fmt.Sprintf("%s-%s", util.XXHash(req.Function.ID.String()), util.XXHash(key))
	require.Equal(t, want, idempotencyKey(req, first))
	require.Equal(t, want, idempotencyKey(req, second))

	req.IdempotencyKey = nil
	for _, runID := range []ulid.ULID{first, second} {
		want := fmt.Sprintf("%s-%s", util.XXHash(req.Function.ID.String()), util.XXHash(runID.String()))
		require.Equal(t, want, idempotencyKey(req, runID))
	}
	require.NotEqual(t, idempotencyKey(req, first), idempotencyKey(req, second))
}

func TestScheduleReplayIdempotency(t *testing.T) {
	ctx := t.Context()
	mr := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{mr.Addr()}, DisableCache: true})
	require.NoError(t, err)
	t.Cleanup(rc.Close)
	unsharded := redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	sharded := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
		UnshardedClient: unsharded, FunctionRunStateClient: rc, BatchClient: rc,
		StateDefaultKey: redis_state.StateDefaultKey, QueueDefaultKey: redis_state.QueueDefaultKey,
		FnRunIsSharded: redis_state.AlwaysShardOnRun,
	})
	mgr, err := redis_state.New(ctx, redis_state.WithShardedClient(sharded), redis_state.WithPauseDeleter(redis_state.NewPauseStore(unsharded)))
	require.NoError(t, err)
	runs := redis_state.MustRunServiceV2(mgr)
	shard := redis_state.NewQueueShard("test", unsharded.Queue())
	registry, err := queue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	e := &executor{
		log: logger.VoidLogger(), smv2: runs,
		queue:          &scheduleFastPathQueue{producer: queue.NewProducer(registry)},
		tracerProvider: tracing.NewNoopTracerProvider(), conditionalTracer: telemetrytrace.NoopConditionalTracer(),
	}
	original := ulid.Make()
	key := "replay:" + uuid.NewString() + ":" + original.String()
	req := execution.ScheduleRequest{
		AccountID: uuid.New(), WorkspaceID: uuid.New(), AppID: uuid.New(),
		Function:      inngest.Function{ID: uuid.New(), FunctionVersion: 1, Name: "Replay idempotency"},
		OriginalRunID: &original, IdempotencyKey: &key,
		Events: []event.TrackedEvent{event.InternalEvent{ID: ulid.Make(), Event: event.Event{
			Name: "test/replay", Timestamp: time.Now().UnixMilli(), Data: map[string]any{},
		}}},
	}
	first, metadata, err := e.Schedule(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, first)
	require.NotNil(t, metadata)
	require.Equal(t, &original, metadata.Config.OriginalRunID)

	second, _, err := e.Schedule(ctx, req)
	require.ErrorIs(t, err, state.ErrIdentifierExists)
	require.Equal(t, first, second, "retry returns the existing run")
	persisted, err := runs.LoadMetadata(ctx, metadata.ID)
	require.NoError(t, err)
	require.Equal(t, *first, persisted.ID.RunID)

	req.IdempotencyKey = nil
	ids := []ulid.ULID{*first}
	for range 2 {
		id, md, err := e.Schedule(ctx, req)
		require.NoError(t, err)
		require.NotContains(t, ids, *id, "each unkeyed rerun creates a new run")
		ids = append(ids, *id)
		persisted, err := runs.LoadMetadata(ctx, md.ID)
		require.NoError(t, err)
		require.Equal(t, *id, persisted.ID.RunID)
		require.Equal(t, &original, persisted.Config.OriginalRunID)
	}
}
