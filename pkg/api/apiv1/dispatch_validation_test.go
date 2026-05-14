package apiv1

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/executor/queueref"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev1 "github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	state "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/flags"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

const testShardName = "test"

func TestDispatchValidation(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		for _, validationEnabled := range []bool{false, true} {
			t.Run(fmt.Sprintf("validation_enabled=%v", validationEnabled), func(t *testing.T) {
				r := require.New(t)
				ctx := silenceLogger(context.Background())
				shard, options := newTestShard(t)
				qi := enqueueTestItem(t, ctx, shard)
				q := newQueueProcessor(t, ctx, shard, options)

				// Create data that simulates what the SDK would send in a
				// checkpoint request.
				body := newCheckpointAsyncBody(qi)

				runState := newTestStateStore(t)
				createRunMetadata(t, ctx, runState, qi)

				api := newTestCheckpointAPI(t, q, runState, authForItem(qi), validationEnabled)

				rec := postAsyncCheckpoint(t, ctx, api, body)
				r.Equal(http.StatusOK, rec.Code, "response body: %s", rec.Body.String())

				// Ensure step processing resulted in a StateStore update.
				steps, err := runState.LoadSteps(ctx, idForItem(qi))
				r.NoError(err)
				r.Contains(steps, "step-1")
				r.JSONEq(`{"data":{"result":"ok"}}`, string(steps["step-1"]))
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		// This is testing our stale dispatch logic. We added this test and its
		// corresponding prod code logic to solve an issue where HTTP timeouts
		// and checkpointing would lead to duplicate execution. When processing
		// a checkpoint request, we return a specific error when the underlying
		// "dispatch" (a.k.a. queue item attempt) no longer exists. This error
		// ultimately tells the SDK "you're done bro, interrupt and stop
		// executing".

		for _, validationEnabled := range []bool{false, true} {
			t.Run(fmt.Sprintf("validation_enabled=%v", validationEnabled), func(t *testing.T) {
				r := require.New(t)
				ctx := silenceLogger(context.Background())
				shard, options := newTestShard(t)
				qi := enqueueTestItem(t, ctx, shard)

				// Capture the RequestID the SDK would echo for the original
				// dispatch (before re-enqueueing bumps the generation).
				staleRequestID := driver.DispatchRequestID(
					qi.Data.Identifier.RunID,
					qi.ID,
					qi.GenerationID,
				).String()

				// Requeue so we bump GenerationID. Any checkpoint that arrives with
				// staleRequestID is now stale.
				r.NoError(shard.Requeue(ctx, qi, time.Now()))

				q := newQueueProcessor(t, ctx, shard, options)

				// Create data that simulates what the SDK would send in a
				// checkpoint request.
				body := newCheckpointAsyncBody(qi)
				body.RequestID = staleRequestID

				runState := newTestStateStore(t)
				createRunMetadata(t, ctx, runState, qi)

				api := newTestCheckpointAPI(t, q, runState, authForItem(qi), validationEnabled)

				rec := postAsyncCheckpoint(t, ctx, api, body)
				if validationEnabled {
					r.Equal(http.StatusConflict, rec.Code, "response body: %s", rec.Body.String())
				} else {
					r.Equal(http.StatusOK, rec.Code, "response body: %s", rec.Body.String())
				}

				// The step must be saved regardless of validation outcome: when the
				// SDK echoes a stale RequestID it means execution actually
				// completed, so we still need to update state.
				steps, err := runState.LoadSteps(ctx, idForItem(qi))
				r.NoError(err)
				r.Contains(steps, "step-1")
				r.JSONEq(`{"data":{"result":"ok"}}`, string(steps["step-1"]))
			})
		}
	})

	t.Run("missing request ID", func(t *testing.T) {
		// When the request ID is empty (e.g. old SDK), dispatch validation is
		// skipped.

		r := require.New(t)
		ctx := silenceLogger(context.Background())
		shard, options := newTestShard(t)
		qi := enqueueTestItem(t, ctx, shard)

		r.NoError(shard.Requeue(ctx, qi, time.Now()))

		q := newQueueProcessor(t, ctx, shard, options)

		body := newCheckpointAsyncBody(qi)
		body.RequestID = ""

		runState := newTestStateStore(t)
		createRunMetadata(t, ctx, runState, qi)

		api := newTestCheckpointAPI(t, q, runState, authForItem(qi), true)

		rec := postAsyncCheckpoint(t, ctx, api, body)
		r.Equal(http.StatusOK, rec.Code, "response body: %s", rec.Body.String())

		steps, err := runState.LoadSteps(ctx, idForItem(qi))
		r.NoError(err)
		r.Contains(steps, "step-1")
		r.JSONEq(`{"data":{"result":"ok"}}`, string(steps["step-1"]))
	})

	t.Run("skip validation when checkpoint request was fast", func(t *testing.T) {
		// Validation is skipped when the checkpoint request came in shortly
		// after the request started. This is an optimization to reduce the
		// number of queue operations.

		r := require.New(t)
		ctx := silenceLogger(context.Background())
		shard, options := newTestShard(t)
		qi := enqueueTestItem(t, ctx, shard)

		staleRequestID := driver.DispatchRequestID(
			qi.Data.Identifier.RunID,
			qi.ID,
			qi.GenerationID,
		).String()

		r.NoError(shard.Requeue(ctx, qi, time.Now()))

		q := newQueueProcessor(t, ctx, shard, options)

		body := newCheckpointAsyncBody(qi)
		body.RequestID = staleRequestID

		// Sumulate the checkpoint request coming in shortly after the request
		// started.
		body.RequestStartedAt = time.Now().UnixMilli()

		runState := newTestStateStore(t)
		createRunMetadata(t, ctx, runState, qi)

		api := newTestCheckpointAPI(t, q, runState, authForItem(qi), true)

		rec := postAsyncCheckpoint(t, ctx, api, body)
		r.Equal(http.StatusOK, rec.Code, "response body: %s", rec.Body.String())

		steps, err := runState.LoadSteps(ctx, idForItem(qi))
		r.NoError(err)
		r.Contains(steps, "step-1")
		r.JSONEq(`{"data":{"result":"ok"}}`, string(steps["step-1"]))
	})
}

// newTestShard sets up a miniredis-backed queue shard for tests
func newTestShard(t *testing.T) (queue.QueueShard, []queue.QueueOpt) {
	t.Helper()

	mr := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{mr.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	t.Cleanup(rc.Close)

	clock := clockwork.NewFakeClock()
	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClient(rc),
		constraintapi.WithShardName(testShardName),
		constraintapi.WithClock(clock),
	)
	require.NoError(t, err)

	options := []queue.QueueOpt{
		queue.WithClock(clock),
		queue.WithCapacityManager(cm),
		queue.WithPartitionConstraintConfigGetter(func(
			_ context.Context,
			_ queue.PartitionIdentifier,
		) queue.PartitionConstraintConfig {
			return queue.PartitionConstraintConfig{
				FunctionVersion: 1,
				Concurrency: queue.PartitionConcurrency{
					SystemConcurrency:   consts.DefaultConcurrencyLimit,
					AccountConcurrency:  consts.DefaultConcurrencyLimit,
					FunctionConcurrency: consts.DefaultConcurrencyLimit,
				},
			}
		}),
	}

	queueClient := redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey)
	shard := redis_state.NewQueueShard(testShardName, queueClient, options...)
	return shard, options
}

// enqueueTestItem enqueues a fresh queue item with random IDs and returns the
// resulting item.
func enqueueTestItem(
	t *testing.T,
	ctx context.Context,
	shard queue.QueueShard,
) queue.QueueItem {
	t.Helper()

	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)

	qi, err := shard.EnqueueItem(ctx, queue.QueueItem{
		FunctionID:  fnID,
		WorkspaceID: envID,
		Data: queue.Item{
			WorkspaceID: envID,
			Kind:        queue.KindStart,
			Identifier: statev1.Identifier{
				AccountID:   accountID,
				WorkspaceID: envID,
				WorkflowID:  fnID,
				RunID:       runID,
			},
		},
	}, time.Now(), queue.EnqueueOpts{})
	require.NoError(t, err)
	return qi
}

// newQueueProcessor wraps shard in a real queue.Queue processor so that
// shard-name-aware operations like ResetAttemptsByJobID and LoadQueueItem work.
func newQueueProcessor(
	t *testing.T,
	ctx context.Context,
	shard queue.QueueShard,
	options []queue.QueueOpt,
) queue.Queue {
	t.Helper()

	reg, err := queue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	q, err := queue.New(ctx, testShardName, reg, options...)
	require.NoError(t, err)
	return q
}

// newCheckpointAsyncBody builds the request body the SDK would send for an
// async checkpoint against qi.
func newCheckpointAsyncBody(qi queue.QueueItem) checkpointAsyncSteps {
	// Simulate the checkpoint request arriving some time after the initial
	// request. We're going 1 minute into the future to avoid a "skip fast
	// requests" logic gate
	requestStartedAt := time.Now().Add(time.Minute).UnixMilli()

	return checkpointAsyncSteps{
		RunID:            qi.Data.Identifier.RunID,
		FnID:             qi.Data.Identifier.WorkflowID,
		QueueItemRef:     queueref.QueueRef{qi.ID, testShardName}.String(),
		RequestStartedAt: requestStartedAt,
		Steps: []statev1.GeneratorOpcode{
			{
				Data: json.RawMessage(`{"result": "ok"}`),
				ID:   "step-1",
				Op:   enums.OpcodeStepRun,
			},
		},
	}
}

// newTestCheckpointAPI builds a CheckpointAPI wired up against the given
// queue, state store, and auth, with the dispatch-validation flag toggled.
func newTestCheckpointAPI(
	t *testing.T,
	q queue.Queue,
	runState state.RunService,
	auth apiv1auth.V1Auth,
	validationEnabled bool,
) CheckpointAPI {
	t.Helper()
	return NewCheckpointAPI(Opts{
		AuthFinder: func(_ context.Context) (apiv1auth.V1Auth, error) {
			return auth, nil
		},
		Queue:          q,
		State:          runState,
		TracerProvider: tracing.NewNoopTracerProvider(),
		CheckpointOpts: CheckpointAPIOpts{
			AllowAsyncDispatchValidation: flags.NewBoolFlag(func(_ context.Context, _ uuid.UUID) bool {
				return validationEnabled
			}),
		},
	})
}

// postAsyncCheckpoint marshals body, posts it to api.CheckpointAsyncSteps, and
// returns the recorded response. ctx is attached to the request so that any
// context values (e.g. a silenced logger via silenceLogger) propagate into the
// handler.
func postAsyncCheckpoint(t *testing.T, ctx context.Context, api CheckpointAPI, body checkpointAsyncSteps) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/async", bytes.NewReader(raw)).WithContext(ctx)
	rec := httptest.NewRecorder()
	api.CheckpointAsyncSteps(rec, req)
	return rec
}

// testAuth implements apiv1auth.V1Auth.
type testAuth struct {
	accountID, envID uuid.UUID
}

func (a testAuth) AccountID() uuid.UUID   { return a.accountID }
func (a testAuth) WorkspaceID() uuid.UUID { return a.envID }

// authForItem returns the V1Auth that the handler should see for a request
// targeting qi. AccountID and EnvID must match the queue item so that the
// checkpointer's computed state.ID lines up with the metadata we created.
func authForItem(qi queue.QueueItem) apiv1auth.V1Auth {
	return testAuth{
		accountID: qi.Data.Identifier.AccountID,
		envID:     qi.Data.Identifier.WorkspaceID,
	}
}

// idForItem returns the v2 state.ID the checkpointer will compute for a
// checkpoint targeting qi (under authForItem). Use it to seed metadata and to
// read steps back out.
func idForItem(qi queue.QueueItem) state.ID {
	return state.ID{
		RunID:      qi.Data.Identifier.RunID,
		FunctionID: qi.Data.Identifier.WorkflowID,
		Tenant: state.Tenant{
			AccountID: qi.Data.Identifier.AccountID,
			EnvID:     qi.Data.Identifier.WorkspaceID,
		},
	}
}

// createRunMetadata creates run metadata for qi so the checkpoint handler's
// LoadMetadata call succeeds.
func createRunMetadata(t *testing.T, ctx context.Context, runState state.RunService, qi queue.QueueItem) {
	t.Helper()
	md := state.Metadata{ID: idForItem(qi)}
	state.InitConfig(&md.Config)
	_, err := runState.Create(ctx, state.CreateState{Metadata: md})
	require.NoError(t, err)
}

// newTestStateStore stands up a miniredis-backed state.RunService. It uses its
// own miniredis instance. The state store and the queue don't need to share
// Redis, and keeping them separate avoids accidental keyspace collisions.
func newTestStateStore(t *testing.T) state.RunService {
	t.Helper()

	mr := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{mr.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	t.Cleanup(rc.Close)

	u := redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	s := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
		UnshardedClient:        u,
		FunctionRunStateClient: rc,
		BatchClient:            rc,
		StateDefaultKey:        redis_state.StateDefaultKey,
		QueueDefaultKey:        redis_state.QueueDefaultKey,
		FnRunIsSharded:         redis_state.AlwaysShardOnRun,
	})
	mgr, err := redis_state.New(context.Background(), redis_state.WithShardedClient(s))
	require.NoError(t, err)
	return redis_state.MustRunServiceV2(mgr)
}

func silenceLogger(ctx context.Context) context.Context {
	return logger.WithStdlib(
		ctx,
		logger.StdlibLogger(ctx, logger.WithLoggerLevel(logger.LevelEmergency)),
	)
}
