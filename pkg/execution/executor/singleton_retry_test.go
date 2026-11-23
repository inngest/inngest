package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/singleton"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	telemetrytrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// Exercise scheduling idempotency with the real singleton, state, and queue Lua.
func TestScheduleSingletonRetryPreservesRun(t *testing.T) {
	for _, mode := range []string{"no_singleton", "cancel_singleton", "cancel_singleton_new_event", "cancel_singleton_custom_key", "cancel_singleton_enqueue_retry", "cancel_singleton_lookup_error", "cancel_singleton_old_event_retry", "cancel_singleton_migration", "skip_singleton_migration"} {
		t.Run(mode, func(t *testing.T) {
			ctx := logger.WithStdlib(context.Background(), logger.VoidLogger())
			mr := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{mr.Addr()}, DisableCache: true})
			require.NoError(t, err)
			t.Cleanup(rc.Close)
			uc := redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
			sc := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
				UnshardedClient: uc, FunctionRunStateClient: rc, BatchClient: rc,
				StateDefaultKey: redis_state.StateDefaultKey, QueueDefaultKey: redis_state.QueueDefaultKey,
				FnRunIsSharded: redis_state.AlwaysShardOnRun,
			})
			mgr, err := redis_state.New(ctx, redis_state.WithShardedClient(sc), redis_state.WithPauseDeleter(redis_state.NewPauseStore(uc)))
			require.NoError(t, err)
			rs := &singletonRetryState{RunService: redis_state.MustRunServiceV2(mgr)}
			shard := redis_state.NewQueueShard("local", redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey))
			registry, err := queue.NewSingleShardRegistry(shard)
			require.NoError(t, err)
			q, err := queue.New(ctx, "singleton-retry", registry)
			require.NoError(t, err)
			req := capScheduleRequest(nil)
			firstRunID := ulid.Make()
			req.RunID = &firstRunID
			if mode == "cancel_singleton_custom_key" {
				customKey := "shared-scheduling-key"
				req.IdempotencyKey = &customKey
			}
			req.Function.Name = "singleton-retry"
			req.Function.Slug = "singleton-retry"
			key := "event.data.postId"
			if mode != "no_singleton" {
				req.Function.Singleton = &inngest.Singleton{Key: &key, Mode: enums.SingletonModeCancel}
			}
			if mode == "skip_singleton_migration" {
				req.Function.Singleton.Mode = enums.SingletonModeSkip
			}
			// Match the post-debounce scheduling call; do not test the debounce timer itself.
			req.PreventDebounce = true
			evt := req.Events[0].(event.InternalEvent)
			evt.Event.Data = map[string]any{"postId": "local-post"}
			req.Events[0] = evt
			lifecycle := &recordingCancelLifecycle{cancelled: make(chan sv2.Metadata, 2)}
			e := &executor{log: logger.VoidLogger(), smv2: rs, queue: q, shards: registry,
				singletonMgr: singleton.New(ctx, registry), fl: singletonRetryFunctionLoader{fn: &req.Function},
				tracerProvider: newRecordingTracerProvider(), conditionalTracer: telemetrytrace.NoopConditionalTracer(),
				lifecycles: []execution.LifecycleListener{lifecycle},
			}
			if mode == "cancel_singleton_enqueue_retry" {
				e.queue = &singletonRetryQueue{Queue: q, failNext: true}
			}
			id, md, err := e.Schedule(ctx, req)
			originalID := sv2.ID{RunID: firstRunID, FunctionID: req.Function.ID, Tenant: sv2.Tenant{AccountID: req.AccountID, EnvID: req.WorkspaceID, AppID: req.AppID}}
			if mode == "cancel_singleton_enqueue_retry" {
				require.ErrorIs(t, err, errSingletonRetryEnqueue)
				id = &firstRunID
			} else {
				require.NoError(t, err)
				require.NotNil(t, md)
				require.Equal(t, originalID, md.ID)
			}
			_, err = rs.LoadMetadata(ctx, originalID)
			require.NoError(t, err)
			scope := queue.Scope{AccountID: req.AccountID, EnvID: req.WorkspaceID, FunctionID: req.Function.ID}
			jobs, err := q.RunJobs(ctx, "local", scope, *id, 10, 0)
			require.NoError(t, err)
			if mode == "cancel_singleton_enqueue_retry" {
				require.Empty(t, jobs)
			} else {
				require.Len(t, jobs, 1)
			}
			if req.Function.Singleton != nil && mode != "cancel_singleton_enqueue_retry" {
				sk, err := singleton.SingletonKey(ctx, req.Function.ID, *req.Function.Singleton, evt.Event.Map())
				require.NoError(t, err)
				owner, err := shard.SingletonGetRunID(ctx, scope, sk)
				require.NoError(t, err)
				require.Equal(t, id, owner)
			}
			t.Logf("first scheduling: original run exists, queued jobs=%d", len(jobs))
			req.RunID = nil
			if mode == "cancel_singleton_migration" || mode == "skip_singleton_migration" {
				require.NoError(t, shard.(queue.SingletonMigrationOperations).BeginSingletonMigration(ctx, scope, "migration"))
				fresh := evt
				fresh.ID = ulid.Make()
				fresh.Event.ID = fresh.ID.String()
				req.Events = []event.TrackedEvent{fresh}
				_, _, err := e.Schedule(ctx, req)
				require.ErrorIs(t, err, queue.ErrSingletonMigrationInProgress)
				require.NotErrorIs(t, err, ErrFunctionSkipped)
				_, err = rs.LoadMetadata(ctx, originalID)
				require.NoError(t, err, "migration retry must not cancel the source run")
				sk, err := singleton.SingletonKey(ctx, req.Function.ID, *req.Function.Singleton, evt.Event.Map())
				require.NoError(t, err)
				owner, err := shard.SingletonGetRunID(ctx, scope, sk)
				require.NoError(t, err)
				require.Equal(t, id, owner)
				return
			}
			if mode == "cancel_singleton_old_event_retry" {
				require.NoError(t, e.Cancel(ctx, originalID, execution.CancelRequest{}))
				select {
				case <-lifecycle.cancelled:
				case <-time.After(time.Second):
					t.Fatal("no original cancellation")
				}
				nextReq := cloneScheduleRequest(req)
				fresh := evt
				fresh.ID = ulid.Make()
				fresh.Event.ID = fresh.ID.String()
				nextReq.Events = []event.TrackedEvent{fresh}
				nextID, nextMD, err := e.Schedule(ctx, nextReq)
				require.NoError(t, err)
				require.NotNil(t, nextMD)
				_, _, err = e.Schedule(ctx, req)
				require.ErrorIs(t, err, ErrFunctionSkippedIdempotency)
				_, err = rs.LoadMetadata(ctx, nextMD.ID)
				require.NoError(t, err)
				replacementJobs, err := q.RunJobs(ctx, "local", scope, *nextID, 10, 0)
				require.NoError(t, err)
				require.Len(t, replacementJobs, 1)
				sk, err := singleton.SingletonKey(ctx, req.Function.ID, *req.Function.Singleton, evt.Event.Map())
				require.NoError(t, err)
				owner, err := shard.SingletonGetRunID(ctx, scope, sk)
				require.NoError(t, err)
				require.Equal(t, nextID, owner, "old event must not release newer run's singleton lock")
				select {
				case <-lifecycle.cancelled:
					t.Fatal("old event canceled replacement")
				default:
				}
				return
			}
			if mode == "cancel_singleton_new_event" || mode == "cancel_singleton_custom_key" {
				fresh := evt
				fresh.ID = ulid.Make()
				fresh.Event.ID = fresh.ID.String()
				req.Events = []event.TrackedEvent{fresh}
			}
			if mode == "cancel_singleton_lookup_error" {
				rs.lookupErr = errors.New("state lookup unavailable")
			}
			retryID, _, retryErr := e.Schedule(ctx, req)
			t.Logf("second scheduling: returned original=%v, error=%v", retryID != nil && *retryID == *id, retryErr)
			_, loadErr := rs.LoadMetadata(ctx, originalID)
			jobs, err = q.RunJobs(ctx, "local", scope, *id, 10, 0)
			require.NoError(t, err)
			if mode == "cancel_singleton_new_event" {
				select {
				case cancelled := <-lifecycle.cancelled:
					require.Equal(t, originalID, cancelled.ID)
				case <-time.After(time.Second):
					t.Fatal("no cancellation lifecycle observed")
				}
				require.ErrorIs(t, loadErr, state.ErrRunNotFound)
				require.Empty(t, jobs)
				require.NoError(t, retryErr)
				require.NotNil(t, retryID)
				require.NotEqual(t, *id, *retryID)
				newID := originalID
				newID.RunID = *retryID
				_, err := rs.LoadMetadata(ctx, newID)
				require.NoError(t, err)
				replacementJobs, err := q.RunJobs(ctx, "local", scope, *retryID, 10, 0)
				require.NoError(t, err)
				require.Len(t, replacementJobs, 1)
				require.Equal(t, []string{"create", "delete", "create"}, rs.operations)
			} else {
				require.NoError(t, loadErr)
				require.Len(t, jobs, 1)
				if mode == "cancel_singleton_enqueue_retry" {
					require.NoError(t, retryErr)
				} else if mode == "cancel_singleton_lookup_error" {
					require.ErrorIs(t, retryErr, rs.lookupErr)
				} else {
					require.ErrorIs(t, retryErr, state.ErrIdentifierExists)
				}
				if mode == "cancel_singleton_lookup_error" {
					require.Equal(t, []string{"create"}, rs.operations)
				} else {
					require.Equal(t, []string{"create", "create"}, rs.operations)
				}
				select {
				case <-lifecycle.cancelled:
					t.Fatal("unexpected cancellation")
				default:
				}
				if req.Function.Singleton != nil {
					sk, err := singleton.SingletonKey(ctx, req.Function.ID, *req.Function.Singleton, evt.Event.Map())
					require.NoError(t, err)
					owner, err := shard.SingletonGetRunID(ctx, scope, sk)
					require.NoError(t, err)
					require.Equal(t, id, owner, "retry must keep the original singleton lock")
				}
			}
		})
	}
}

type singletonRetryFunctionLoader struct{ fn *inngest.Function }

func (l singletonRetryFunctionLoader) LoadFunction(context.Context, uuid.UUID, uuid.UUID) (*state.ExecutorFunction, error) {
	return &state.ExecutorFunction{Function: l.fn}, nil
}

type singletonRetryState struct {
	sv2.RunService
	operations []string
	lookupErr  error
}

func (s *singletonRetryState) Create(ctx context.Context, in sv2.CreateState) (sv2.State, error) {
	s.operations = append(s.operations, "create")
	out, err := s.RunService.Create(ctx, in)
	return out, err
}
func (s *singletonRetryState) Delete(ctx context.Context, id sv2.ID, opts ...sv2.DeleteOption) error {
	s.operations = append(s.operations, "delete")
	return s.RunService.Delete(ctx, id, opts...)
}

func (s *singletonRetryState) LookupIdempotency(ctx context.Context, id sv2.ID, key string) (*sv2.IdempotencyEntry, error) {
	if s.lookupErr != nil {
		return nil, s.lookupErr
	}
	return s.RunService.LookupIdempotency(ctx, id, key)
}

var errSingletonRetryEnqueue = errors.New("temporary queue failure")

type singletonRetryQueue struct {
	queue.Queue
	failNext bool
}

func (q *singletonRetryQueue) Enqueue(ctx context.Context, item queue.Item, at time.Time, opts queue.EnqueueOpts) error {
	if q.failNext {
		q.failNext = false
		return errSingletonRetryEnqueue
	}
	return q.Queue.Enqueue(ctx, item, at, opts)
}
