package executor

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/cqrs/base_cqrs"
	dbsqlite "github.com/inngest/inngest/pkg/db/sqlite"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// countingTracerProvider wraps a TracerProvider and counts span creations by name.
// Resume() creates exactly one meta.SpanNameStepDiscovery span per enqueued
// discovery, so this counter doubles as a discovery-enqueue counter.
type countingTracerProvider struct {
	tracing.TracerProvider

	mu     sync.Mutex
	counts map[string]int
}

func newCountingTracerProvider(inner tracing.TracerProvider) *countingTracerProvider {
	return &countingTracerProvider{TracerProvider: inner, counts: map[string]int{}}
}

func (c *countingTracerProvider) CreateDroppableSpan(ctx context.Context, name string, opts *tracing.CreateSpanOptions) (*tracing.DroppableSpan, error) {
	c.mu.Lock()
	c.counts[name]++
	c.mu.Unlock()
	return c.TracerProvider.CreateDroppableSpan(ctx, name, opts)
}

func (c *countingTracerProvider) CreateSpan(ctx context.Context, name string, opts *tracing.CreateSpanOptions) (*meta.SpanReference, error) {
	c.mu.Lock()
	c.counts[name]++
	c.mu.Unlock()
	return c.TracerProvider.CreateSpan(ctx, name, opts)
}

func (c *countingTracerProvider) count(name string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counts[name]
}

// TestParallelPauseBackedOpsCoalesceDiscovery is the EXE-1714 regression.
//
// When a Promise.all of N step.invoke calls returns a single SDK response with
// N OpcodeInvokeFunction ops, HandleGeneratorResponse must call SavePending so
// that the keyStepsPending Set is seeded with all N step IDs. With the set
// seeded, ConsumePause -> SaveResponse -> hasPending decrements per Resume so
// only the final Resume sees hasPending=false and enqueues a discovery — the
// other N-1 resumes coalesce.
//
// Before the fix, hasPlanOp matched only OpcodeStepPlanned, so SavePending was
// skipped, the Set stayed empty, every Resume saw hasPending=false, and each
// pause enqueued its own discovery — fanning out N executions and N root-level
// RunComplete spans instead of 1.
//
// The clean signal is the count of "executor.step.discovery" spans across all
// Resumes: it must be exactly 1 (not N).
func TestParallelPauseBackedOpsCoalesceDiscovery(t *testing.T) {
	ctx := context.Background()

	db, err := base_cqrs.New(ctx, base_cqrs.BaseCQRSOptions{Persist: false})
	require.NoError(t, err)

	adapter := dbsqlite.New(db)
	dbcqrs := base_cqrs.NewCQRS(adapter)
	loader := dbcqrs.(state.FunctionLoader)

	fnID, wsID, appID, aID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	targetFnID := uuid.New()

	fn := inngest.Function{
		ID:              fnID,
		FunctionVersion: 1,
		Name:            "fan-out-fn",
		Slug:            "fan-out-fn",
		Steps: []inngest.Step{
			{ID: "step", Name: "step", URI: "/step"},
		},
	}

	targetFn := inngest.Function{
		ID:              targetFnID,
		FunctionVersion: 1,
		Name:            "target-fn",
		Slug:            "target-fn",
	}

	cfg, err := json.Marshal(fn)
	require.NoError(t, err)
	targetCfg, err := json.Marshal(targetFn)
	require.NoError(t, err)

	_, err = dbcqrs.UpsertApp(ctx, cqrs.UpsertAppParams{ID: appID, Name: "test-app"})
	require.NoError(t, err)
	_, err = dbcqrs.InsertFunction(ctx, cqrs.InsertFunctionParams{
		ID: fnID, AppID: appID, Name: fn.Name, Slug: fn.Slug, Config: string(cfg),
	})
	require.NoError(t, err)
	_, err = dbcqrs.InsertFunction(ctx, cqrs.InsertFunctionParams{
		ID: targetFnID, AppID: appID, Name: targetFn.Name, Slug: targetFn.Slug, Config: string(targetCfg),
	})
	require.NoError(t, err)

	_, shardedRc, err := createInmemoryRedis(t)
	require.NoError(t, err)
	defer shardedRc.Close()

	_, unshardedRc, err := createInmemoryRedis(t)
	require.NoError(t, err)
	defer unshardedRc.Close()

	unshardedClient := redis_state.NewUnshardedClient(unshardedRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	shardedClient := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
		UnshardedClient:        unshardedClient,
		FunctionRunStateClient: shardedRc,
		StateDefaultKey:        redis_state.StateDefaultKey,
		FnRunIsSharded:         redis_state.AlwaysShardOnRun,
		BatchClient:            shardedRc,
		QueueDefaultKey:        redis_state.QueueDefaultKey,
	})

	queueOpts := []queue.QueueOpt{queue.WithIdempotencyTTL(time.Hour)}
	queueShard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue(), queueOpts...)
	shardRegistry, err := queue.NewSingleShardRegistry(queueShard)
	require.NoError(t, err)

	pauseMgr := pauses.NewPauseStoreManager(unshardedClient)

	var sm state.Manager
	sm, err = redis_state.New(ctx,
		redis_state.WithShardedClient(shardedClient),
		redis_state.WithPauseDeleter(pauseMgr),
	)
	require.NoError(t, err)
	smv2 := redis_state.MustRunServiceV2(sm)

	rq, err := queue.New(
		context.Background(),
		"test-queue",
		shardRegistry,
		queueOpts...,
	)
	require.NoError(t, err)

	stepIDs := []string{"invoke-a", "invoke-b", "invoke-c"}
	gen := make([]*state.GeneratorOpcode, len(stepIDs))
	for i, id := range stepIDs {
		gen[i] = &state.GeneratorOpcode{
			Op: enums.OpcodeInvokeFunction,
			ID: id,
			Opts: map[string]any{
				"function_id": targetFnID.String(),
				"payload":     map[string]any{"data": map[string]any{"step": id}},
			},
		}
	}

	// RequestVersion=2 satisfies Metadata.ShouldCoalesceParallelism, which is
	// the second gate (alongside hasPlanOp) for SavePending. Without this the
	// fix has nothing to do.
	mockDriver := &mockDriverV1{
		t: t,
		response: &state.DriverResponse{
			StatusCode:     206,
			RequestVersion: 2,
			Generator:      gen,
		},
	}

	tp := newCountingTracerProvider(tracing.NewOtelTracerProvider(nil, time.Millisecond))

	exec, err := executor.NewExecutor(
		executor.WithStateManager(smv2),
		executor.WithPauseManager(pauseMgr),
		executor.WithQueue(rq),
		executor.WithLogger(logger.StdlibLogger(ctx)),
		executor.WithFunctionLoader(loader),
		executor.WithShardRegistry(shardRegistry),
		executor.WithTracerProvider(tp),
		executor.WithInvokeEventHandler(func(ctx context.Context, evt event.TrackedEvent) error { return nil }),
		executor.WithDriverV1(mockDriver),
	)
	require.NoError(t, err)

	now := time.Now()
	evtID := ulid.MustNew(ulid.Timestamp(now), rand.Reader)

	_, run, err := exec.Schedule(ctx, execution.ScheduleRequest{
		Function:    fn,
		At:          &now,
		AccountID:   aID,
		WorkspaceID: wsID,
		AppID:       appID,
		Events: []event.TrackedEvent{
			event.NewBaseTrackedEventWithID(event.Event{Name: "test/event"}, evtID),
		},
	})
	require.NoError(t, err)

	jobsAfterSchedule, err := rq.RunJobs(
		ctx,
		queueShard.Name(),
		run.ID.Tenant.EnvID,
		run.ID.FunctionID,
		run.ID.RunID,
		1000,
		0,
	)
	require.NoError(t, err)
	require.NotEmpty(t, jobsAfterSchedule)

	jobCtx := queue.WithJobID(ctx, jobsAfterSchedule[0].JobID)
	id := sv2.V1FromMetadata(*run)
	edge := inngest.Edge{Incoming: "$trigger", Outgoing: "step"}
	_, err = exec.Execute(jobCtx, id, queue.Item{
		WorkspaceID: wsID,
		Kind:        queue.KindStart,
		Identifier:  id,
		Payload:     queue.PayloadEdge{Edge: edge},
	}, edge)
	require.NoError(t, err)

	// Schedule + Execute have already produced unrelated discovery spans
	// (initial trigger discovery, etc.). Capture the baseline so the
	// assertion measures only spans created by the resumes below.
	baseline := tp.count(meta.SpanNameStepDiscovery)

	// Pause IDs are derived deterministically by handleGeneratorInvokeFunction
	// as DeterministicSha1UUID(runID + opcode.ID), which lets us look them up
	// without iterating the index.
	idx := pauses.Index{WorkspaceID: wsID, EventName: event.FnFinishedName}
	for _, id := range stepIDs {
		pauseID := inngest.DeterministicSha1UUID(run.ID.RunID.String() + id)
		p, err := pauseMgr.PauseByID(ctx, idx, pauseID)
		require.NoError(t, err, "pause for %q should be written by handleGeneratorInvokeFunction", id)
		require.NotNil(t, p)

		err = exec.Resume(ctx, *p, execution.ResumeRequest{
			With: map[string]any{"data": map[string]any{"resumed": id}},
		})
		require.NoError(t, err)
	}

	// EXE-1714: exactly one discovery span is created across all three resumes
	// (Resume only calls CreateDroppableSpan(SpanNameStepDiscovery) inside the
	// shouldEnqueueDiscovery branch). Without the hasPlanOp fix the delta is
	// len(stepIDs).
	require.Equal(t, 1, tp.count(meta.SpanNameStepDiscovery)-baseline,
		"parallel pause-backed ops must coalesce into a single discovery")
}
