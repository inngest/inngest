package executor

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"golang.org/x/sync/errgroup"

	"github.com/inngest/inngest/pkg/event"
	executionpkg "github.com/inngest/inngest/pkg/execution"
	executorpkg "github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestFinalizePublishesFnFinishedOnce(t *testing.T) {
	ctx := context.Background()

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

	pauseMgr := pauses.NewPauseStoreManager(unshardedClient)

	sm, err := redis_state.New(
		ctx,
		redis_state.WithShardedClient(shardedClient),
		redis_state.WithPauseDeleter(pauseMgr),
	)
	require.NoError(t, err)
	smv2 := redis_state.MustRunServiceV2(sm)

	queueShard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue())
	shardRegistry, err := queue.NewSingleShardRegistry(queueShard)
	require.NoError(t, err)

	fnID := uuid.New()
	accountID := uuid.New()
	wsID := uuid.New()
	appID := uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	eventID := ulid.MustNew(ulid.Now(), rand.Reader)

	rawEvent, err := json.Marshal(event.Event{
		ID:   eventID.String(),
		Name: "test/event",
		Data: map[string]any{"ok": true},
	})
	require.NoError(t, err)

	md := statev2.Metadata{
		ID: statev2.ID{
			RunID:      runID,
			FunctionID: fnID,
			Tenant: statev2.Tenant{
				AccountID: accountID,
				EnvID:     wsID,
				AppID:     appID,
			},
		},
		Config: *statev2.InitConfig(&statev2.Config{
			EventIDs:        []ulid.ULID{eventID},
			Idempotency:     fmt.Sprintf("dup-finalize-%s", runID.String()),
			FunctionVersion: 1,
			RequestVersion:  1,
		}),
	}

	_, err = smv2.Create(ctx, statev2.CreateState{
		Metadata: md,
		Events:   []json.RawMessage{rawEvent},
	})
	require.NoError(t, err)

	var (
		mu         sync.Mutex
		calls      int
		finishIDs  []string
		finishRuns []ulid.ULID
	)

	exec, err := executorpkg.NewExecutor(
		executorpkg.WithStateManager(smv2),
		executorpkg.WithPauseManager(pauseMgr),
		executorpkg.WithLogger(logger.StdlibLogger(ctx)),
		executorpkg.WithShardRegistry(shardRegistry),
		executorpkg.WithFinalizer(func(ctx context.Context, id statev2.ID, evts []event.Event) error {
			mu.Lock()
			defer mu.Unlock()

			calls++
			for _, evt := range evts {
				if evt.Name == event.FnFinishedName {
					finishIDs = append(finishIDs, evt.ID)
					finishRuns = append(finishRuns, id.RunID)
				}
			}
			return nil
		}),
	)
	require.NoError(t, err)

	opts := executionpkg.FinalizeOpts{
		Metadata: md,
		Response: executionpkg.FinalizeResponse{
			Type: executionpkg.FinalizeResponseRunComplete,
			RunComplete: statev2.GeneratorOpcode{
				Op:   enums.OpcodeRunComplete,
				Data: json.RawMessage(`{"data":{"ok":true}}`),
			},
		},
		Optional: executionpkg.FinalizeOptional{
			FnSlug:      "test-fn",
			InputEvents: []json.RawMessage{rawEvent},
		},
	}

	eg := errgroup.Group{}
	eg.Go(func() error { return exec.Finalize(ctx, opts) })
	eg.Go(func() error { return exec.Finalize(ctx, opts) })
	require.NoError(t, eg.Wait())

	mu.Lock()
	defer mu.Unlock()

	require.Equal(t, 1, calls, "duplicate finalize should only emit finish effects once")
	require.Len(t, finishIDs, 1, "duplicate finalize should emit one fn.finished event")
	require.Equal(t, runID, finishRuns[0])
}

func TestFinalizeRetriesFnFinishedAfterPublishFailure(t *testing.T) {
	ctx := context.Background()

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

	pauseMgr := pauses.NewPauseStoreManager(unshardedClient)

	sm, err := redis_state.New(
		ctx,
		redis_state.WithShardedClient(shardedClient),
		redis_state.WithPauseDeleter(pauseMgr),
	)
	require.NoError(t, err)
	smv2 := redis_state.MustRunServiceV2(sm)

	queueShard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue())
	shardRegistry, err := queue.NewSingleShardRegistry(queueShard)
	require.NoError(t, err)

	fnID := uuid.New()
	accountID := uuid.New()
	wsID := uuid.New()
	appID := uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	eventID := ulid.MustNew(ulid.Now(), rand.Reader)

	rawEvent, err := json.Marshal(event.Event{
		ID:   eventID.String(),
		Name: "test/event",
		Data: map[string]any{"ok": true},
	})
	require.NoError(t, err)

	md := statev2.Metadata{
		ID: statev2.ID{
			RunID:      runID,
			FunctionID: fnID,
			Tenant: statev2.Tenant{
				AccountID: accountID,
				EnvID:     wsID,
				AppID:     appID,
			},
		},
		Config: *statev2.InitConfig(&statev2.Config{
			EventIDs:        []ulid.ULID{eventID},
			Idempotency:     fmt.Sprintf("retry-finalize-%s", runID.String()),
			FunctionVersion: 1,
			RequestVersion:  1,
		}),
	}

	_, err = smv2.Create(ctx, statev2.CreateState{
		Metadata: md,
		Events:   []json.RawMessage{rawEvent},
	})
	require.NoError(t, err)

	var (
		mu        sync.Mutex
		calls     int
		finishIDs []string
	)

	exec, err := executorpkg.NewExecutor(
		executorpkg.WithStateManager(smv2),
		executorpkg.WithPauseManager(pauseMgr),
		executorpkg.WithLogger(logger.StdlibLogger(ctx)),
		executorpkg.WithShardRegistry(shardRegistry),
		executorpkg.WithFinalizer(func(ctx context.Context, id statev2.ID, evts []event.Event) error {
			mu.Lock()
			defer mu.Unlock()

			calls++
			if calls == 1 {
				return fmt.Errorf("synthetic publish failure")
			}

			for _, evt := range evts {
				if evt.Name == event.FnFinishedName {
					finishIDs = append(finishIDs, evt.ID)
				}
			}
			return nil
		}),
	)
	require.NoError(t, err)

	opts := executionpkg.FinalizeOpts{
		Metadata: md,
		Response: executionpkg.FinalizeResponse{
			Type: executionpkg.FinalizeResponseRunComplete,
			RunComplete: statev2.GeneratorOpcode{
				Op:   enums.OpcodeRunComplete,
				Data: json.RawMessage(`{"data":{"ok":true}}`),
			},
		},
		Optional: executionpkg.FinalizeOptional{
			FnSlug:      "test-fn",
			InputEvents: []json.RawMessage{rawEvent},
		},
	}

	err = exec.Finalize(ctx, opts)
	require.Error(t, err)

	err = exec.Finalize(ctx, opts)
	require.NoError(t, err)

	err = exec.Finalize(ctx, opts)
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

	require.Equal(t, 2, calls, "failed publish should release the claim so one retry can emit finish effects")
	require.Len(t, finishIDs, 1, "successful retry should still emit fn.finished exactly once")
}

func TestFinalizeDeleteFailureDoesNotLoseFinishEffects(t *testing.T) {
	ctx := context.Background()

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

	pauseMgr := pauses.NewPauseStoreManager(unshardedClient)

	sm, err := redis_state.New(
		ctx,
		redis_state.WithShardedClient(shardedClient),
		redis_state.WithPauseDeleter(pauseMgr),
	)
	require.NoError(t, err)
	baseState := redis_state.MustRunServiceV2(sm)
	claimant, ok := baseState.(statev2.FinalizationClaimer)
	require.True(t, ok)

	stateSvc := &deleteFailsOnceRunService{
		RunService: baseState,
		claimant:   claimant,
	}

	queueShard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue())
	shardRegistry, err := queue.NewSingleShardRegistry(queueShard)
	require.NoError(t, err)

	fnID := uuid.New()
	accountID := uuid.New()
	wsID := uuid.New()
	appID := uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	eventID := ulid.MustNew(ulid.Now(), rand.Reader)

	rawEvent, err := json.Marshal(event.Event{
		ID:   eventID.String(),
		Name: "test/event",
		Data: map[string]any{"ok": true},
	})
	require.NoError(t, err)

	md := statev2.Metadata{
		ID: statev2.ID{
			RunID:      runID,
			FunctionID: fnID,
			Tenant: statev2.Tenant{
				AccountID: accountID,
				EnvID:     wsID,
				AppID:     appID,
			},
		},
		Config: *statev2.InitConfig(&statev2.Config{
			EventIDs:        []ulid.ULID{eventID},
			Idempotency:     fmt.Sprintf("delete-failure-finalize-%s", runID.String()),
			FunctionVersion: 1,
			RequestVersion:  1,
		}),
	}

	_, err = stateSvc.Create(ctx, statev2.CreateState{
		Metadata: md,
		Events:   []json.RawMessage{rawEvent},
	})
	require.NoError(t, err)

	var (
		mu        sync.Mutex
		calls     int
		finishIDs []string
	)

	exec, err := executorpkg.NewExecutor(
		executorpkg.WithStateManager(stateSvc),
		executorpkg.WithPauseManager(pauseMgr),
		executorpkg.WithLogger(logger.StdlibLogger(ctx)),
		executorpkg.WithShardRegistry(shardRegistry),
		executorpkg.WithFinalizer(func(ctx context.Context, id statev2.ID, evts []event.Event) error {
			mu.Lock()
			defer mu.Unlock()

			calls++
			for _, evt := range evts {
				if evt.Name == event.FnFinishedName {
					finishIDs = append(finishIDs, evt.ID)
				}
			}
			return nil
		}),
	)
	require.NoError(t, err)

	opts := executionpkg.FinalizeOpts{
		Metadata: md,
		Response: executionpkg.FinalizeResponse{
			Type: executionpkg.FinalizeResponseRunComplete,
			RunComplete: statev2.GeneratorOpcode{
				Op:   enums.OpcodeRunComplete,
				Data: json.RawMessage(`{"data":{"ok":true}}`),
			},
		},
		Optional: executionpkg.FinalizeOptional{
			FnSlug:      "test-fn",
			InputEvents: []json.RawMessage{rawEvent},
		},
	}

	err = exec.Finalize(ctx, opts)
	require.NoError(t, err, "delete failure is best-effort and should not block finish effects")

	exists, err := stateSvc.Exists(ctx, md.ID)
	require.NoError(t, err)
	require.True(t, exists, "first finalize should leave state behind when delete fails")

	err = exec.Finalize(ctx, opts)
	require.NoError(t, err)

	exists, err = stateSvc.Exists(ctx, md.ID)
	require.NoError(t, err)
	require.False(t, exists, "later finalize retry should still be able to clean up state")

	mu.Lock()
	defer mu.Unlock()

	require.Equal(t, 1, calls, "delete retry must not re-emit finish effects")
	require.Len(t, finishIDs, 1, "delete failure must not lose or duplicate fn.finished")
	require.Equal(t, 2, stateSvc.deleteCalls)
}

type deleteFailsOnceRunService struct {
	statev2.RunService
	claimant statev2.FinalizationClaimer

	mu          sync.Mutex
	deleteCalls int
}

func (d *deleteFailsOnceRunService) Delete(ctx context.Context, id statev2.ID) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.deleteCalls++
	if d.deleteCalls == 1 {
		return fmt.Errorf("synthetic delete failure")
	}

	return d.RunService.Delete(ctx, id)
}

func (d *deleteFailsOnceRunService) ClaimFinalization(ctx context.Context, md statev2.Metadata) (bool, error) {
	return d.claimant.ClaimFinalization(ctx, md)
}

func (d *deleteFailsOnceRunService) ReleaseFinalization(ctx context.Context, md statev2.Metadata) error {
	return d.claimant.ReleaseFinalization(ctx, md)
}
