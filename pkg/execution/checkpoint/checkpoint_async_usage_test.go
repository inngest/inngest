package checkpoint

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	state "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// countingMetrics counts OnStepFinished calls.  Calls are synchronous in the async
// checkpoint path, so no locking is needed.
type countingMetrics struct {
	nilCheckpointMetrics
	steps int
}

func (m *countingMetrics) OnStepFinished(context.Context, MetricCardinality, enums.StepStatus) {
	m.steps++
}

// TestCheckpointAsyncSteps_UsageAgainstRedisState checkpoints against the real redis
// state store, so retried checkpoints hit its actual duplicate/idempotent handling.
func TestCheckpointAsyncSteps_UsageAgainstRedisState(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)

	mr, err := miniredis.Run()
	r.NoError(err)
	defer mr.Close()

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{mr.Addr()},
		DisableCache: true,
	})
	r.NoError(err)
	defer rc.Close()

	unshardedClient := redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	shardedClient := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
		UnshardedClient:        unshardedClient,
		FunctionRunStateClient: rc,
		BatchClient:            rc,
		StateDefaultKey:        redis_state.StateDefaultKey,
		QueueDefaultKey:        redis_state.QueueDefaultKey,
		FnRunIsSharded:         redis_state.AlwaysShardOnRun,
	})
	mgr, err := redis_state.New(
		ctx,
		redis_state.WithShardedClient(shardedClient),
		redis_state.WithPauseDeleter(redis_state.NewPauseStore(unshardedClient)),
	)
	r.NoError(err)
	runService := redis_state.MustRunServiceV2(mgr)

	accountID, envID := uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	_, err = runService.Create(ctx, state.CreateState{
		Metadata: state.Metadata{
			ID: state.ID{
				RunID:      runID,
				FunctionID: uuid.New(),
				Tenant: state.Tenant{
					AccountID: accountID,
					EnvID:     envID,
					AppID:     uuid.New(),
				},
			},
			Config: *state.InitConfig(&state.Config{
				Idempotency: "usage-" + runID.String(),
			}),
		},
		Events: []json.RawMessage{json.RawMessage(`{"name":"test/event","data":{}}`)},
	})
	r.NoError(err)

	tracer := &mockTracerProvider{}
	tracer.On("CreateSpan", mock.Anything, meta.SpanNameStep, mock.Anything).Return(&meta.SpanReference{}, nil)
	metrics := &countingMetrics{}

	c := New(Opts{
		State:           runService,
		TracerProvider:  tracer,
		Queue:           &mockQueue{},
		MetricsProvider: metrics,
		Executor:        &mockExecutor{},
	})

	checkpoint := func(id, data string) {
		t.Helper()
		r.NoError(c.CheckpointAsyncSteps(ctx, AsyncCheckpoint{
			RunID:     runID,
			AccountID: accountID,
			EnvID:     envID,
			Steps: []state.GeneratorOpcode{{
				ID:   id,
				Op:   enums.OpcodeStepRun,
				Data: json.RawMessage(data),
			}},
		}))
	}

	checkpoint("step-1", `{"result":"one"}`)
	r.Equal(1, metrics.steps, "first save of a step is counted")

	// The SDK retries checkpoints (eg. after a timeout) w/ the exact same body.
	checkpoint("step-1", `{"result":"one"}`)
	r.Equal(1, metrics.steps, "identical re-checkpoint must not be counted again")

	// Different output for an already saved step is a duplicate and is dropped.
	checkpoint("step-1", `{"result":"changed"}`)
	r.Equal(1, metrics.steps, "duplicate step must not be counted")

	checkpoint("step-2", `{"result":"two"}`)
	r.Equal(2, metrics.steps, "a new step is counted")
}
