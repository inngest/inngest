package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

type cleanupDeleteFailure struct{ sv2.RunService }

func (s cleanupDeleteFailure) Delete(context.Context, sv2.ID, ...sv2.DeleteOption) error {
	return errors.New("injected state delete failure")
}

// Run on a local Redis/Valkey container by setting SEMAPHORE_TEST_REDIS_ADDR;
// CI uses miniredis. All fixtures have unique accounts and queue prefixes, and
// state, queue and constraints are in separate logical databases. No flushes.
func TestVerifiedCancellationRedis(t *testing.T) {
	addr := os.Getenv("SEMAPHORE_TEST_REDIS_ADDR")
	if addr == "" {
		addr = miniredis.RunT(t).Addr()
	}
	host, _, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", host)
	for _, name := range []string{"holder", "missing-input", "missing-metadata", "unknown-owner", "orphan-index", "delete-retry"} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			client := func(db int) rueidis.Client {
				rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{addr}, SelectDB: db, DisableCache: true, ForceSingleClient: true})
				require.NoError(t, err)
				t.Cleanup(rc.Close)
				return rc
			}
			queueRC, stateRC, constraintRC := client(0), client(1), client(2)
			prefix := "verified-cancel-" + uuid.NewString()
			unsharded := redis_state.NewUnshardedClient(queueRC, redis_state.StateDefaultKey, prefix)
			sharded := redis_state.NewShardedClient(redis_state.ShardedClientOpts{UnshardedClient: unsharded, FunctionRunStateClient: stateRC, StateDefaultKey: redis_state.StateDefaultKey, FnRunIsSharded: redis_state.AlwaysShardOnRun, BatchClient: stateRC, QueueDefaultKey: prefix})
			pm := pauses.NewPauseStoreManager(unsharded)
			legacyState, err := redis_state.New(ctx, redis_state.WithShardedClient(sharded), redis_state.WithPauseDeleter(pm))
			require.NoError(t, err)
			runs := redis_state.MustRunServiceV2(legacyState)
			shard := redis_state.NewQueueShard(prefix, unsharded.Queue())
			registry, err := queue.NewSingleShardRegistry(shard)
			require.NoError(t, err)
			q, err := queue.New(ctx, prefix, registry)
			require.NoError(t, err)
			tenant := sv2.Tenant{AccountID: uuid.New(), EnvID: uuid.New(), AppID: uuid.New()}
			id := sv2.ID{RunID: ulid.Make(), FunctionID: uuid.New(), Tenant: tenant}
			sem := constraintapi.Semaphore{ID: constraintapi.SemaphoreIDFnKey(id.FunctionID, "event.data.app_id"), EvaluatedKeyHash: "same-app", Weight: 1, Release: constraintapi.SemaphoreReleaseManual}
			sm := constraintapi.NewRedisSemaphoreManager(constraintRC)
			_, err = sm.SetCapacity(ctx, tenant.AccountID, sem.ID, uuid.NewString(), 2)
			require.NoError(t, err)
			capacity, err := constraintapi.NewRedisCapacityManager(constraintapi.WithClient(constraintRC), constraintapi.WithShardName(prefix), constraintapi.WithManualSemaphoreOwnership())
			require.NoError(t, err)
			acquire := func(run ulid.ULID) {
				t.Helper()
				key := run.String() + "-start"
				res, err := capacity.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
					AccountID: tenant.AccountID, EnvID: tenant.EnvID, FunctionID: id.FunctionID, IdempotencyKey: key, Amount: 1,
					Configuration:        constraintapi.ConstraintConfig{FunctionVersion: 1, Semaphores: []constraintapi.Semaphore{sem}},
					Constraints:          []constraintapi.ConstraintItem{{Kind: constraintapi.ConstraintKindSemaphore, Semaphore: &constraintapi.SemaphoreConstraint{ID: sem.ID, EvaluatedKeyHash: sem.EvaluatedKeyHash, Weight: 1, Release: sem.Release}}},
					LeaseIdempotencyKeys: []string{key}, LeaseRunIDs: map[string]ulid.ULID{key: run}, CurrentTime: time.Now(), Duration: time.Minute, MaximumLifetime: time.Hour,
					Source: constraintapi.LeaseSource{Service: constraintapi.ServiceExecutor, Location: constraintapi.CallerLocationItemLease},
				})
				require.NoError(t, err)
				require.Len(t, res.Leases, 1)
			}
			other := ulid.Make()
			acquire(other)
			if name != "unknown-owner" {
				acquire(id.RunID)
			}
			eventID := ulid.Make()
			md := sv2.Metadata{ID: id, Config: *sv2.InitConfig(&sv2.Config{EventIDs: []ulid.ULID{eventID}, Idempotency: id.RunID.String(), FunctionVersion: 1, RequestVersion: 1, Semaphores: []constraintapi.Semaphore{sem}})}
			_, err = runs.Create(ctx, sv2.CreateState{Metadata: md, Events: []json.RawMessage{json.RawMessage(`{"name":"test/cancel","data":{}}`)}})
			require.NoError(t, err)
			md, err = runs.LoadMetadata(ctx, id)
			require.NoError(t, err)
			var items []queue.QueueItem
			for i := range 3 {
				item, err := shard.EnqueueItem(ctx, queue.QueueItem{ID: fmt.Sprintf("%s-%d", id.RunID, i), FunctionID: id.FunctionID, WorkspaceID: tenant.EnvID, Data: queue.Item{Kind: queue.KindStart, Identifier: sv2.V1FromMetadata(md), Semaphores: md.Config.Semaphores, Attempt: 7, Payload: queue.PayloadEdge{}}}, time.Now().Add(time.Minute), queue.EnqueueOpts{})
				require.NoError(t, err)
				items = append(items, item)
			}
			request := execution.CancelRequest{ForceLifecycleHook: true, RequireCleanupVerification: true}
			for _, item := range items {
				request.KnownQueueItemIDs = append(request.KnownQueueItemIDs, item.ID)
			}
			stateKeys := sharded.FunctionRunState().KeyGenerator()
			if name == "missing-input" {
				require.NoError(t, stateRC.Do(ctx, stateRC.B().Del().Key(stateKeys.Events(ctx, true, id.FunctionID, id.RunID)).Build()).Error())
			}
			if name == "missing-metadata" {
				require.NoError(t, stateRC.Do(ctx, stateRC.B().Del().Key(stateKeys.RunMetadata(ctx, true, id.RunID)).Build()).Error())
			}
			if name == "orphan-index" {
				require.NoError(t, queueRC.Do(ctx, queueRC.B().Zrem().Key(unsharded.Queue().KeyGenerator().RunIndex(id.RunID)).Member(items[0].ID).Build()).Error())
			}
			instance, err := NewExecutor(WithStateManager(runs), WithPauseManager(pm), WithQueue(q), WithShardRegistry(registry), WithFunctionLoader(&failedRunFunctionLoader{slug: "test"}), WithSemaphoreManager(sm), WithLogger(logger.VoidLogger()))
			require.NoError(t, err)
			e := instance.(*executor)
			if name == "delete-retry" {
				e.smv2 = cleanupDeleteFailure{RunService: runs}
			}
			err = e.Cancel(ctx, id, request)
			if name == "holder" || name == "orphan-index" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			for _, item := range items {
				_, err := q.LoadQueueItem(ctx, shard.Name(), item.ID)
				if name == "missing-metadata" || name == "unknown-owner" {
					require.NoError(t, err)
				} else {
					require.ErrorIs(t, err, queue.ErrQueueItemNotFound)
				}
			}
			_, usage, err := sm.GetCapacity(ctx, tenant.AccountID, sem.ID, sem.EvaluatedKeyHash)
			require.NoError(t, err)
			want := int64(1)
			if name == "missing-metadata" {
				want = 2
			}
			require.Equal(t, want, usage, "preserve the other run's actual acquired permit")
			_, mdErr := runs.LoadMetadata(ctx, id)
			if name == "delete-retry" || name == "missing-input" || name == "unknown-owner" {
				require.NoError(t, mdErr, "retain metadata on incomplete cleanup")
			}
			if name == "delete-retry" {
				e.smv2 = runs
				require.NoError(t, e.Cancel(ctx, id, request))
				_, usage, err = sm.GetCapacity(ctx, tenant.AccountID, sem.ID, sem.EvaluatedKeyHash)
				require.NoError(t, err)
				require.Equal(t, int64(1), usage, "retry must not release the other holder")
			}
		})
	}
}
