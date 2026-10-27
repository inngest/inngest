package redis_state

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestMaintenanceRetirementCheckpoint(t *testing.T) {
	r, client := initRedis(t)
	t.Cleanup(client.Close)
	_, shard := newQueue(t, client)
	q := shard.(*queue)
	ctx := context.Background()
	checkpoint := osqueue.MaintenanceRetirement{AccountID: uuid.New(), EnvID: uuid.New(), AppID: uuid.New(), FunctionID: uuid.New(), RunID: ulid.Make(), JobID: "job", BeforeMS: time.Now().UnixMilli(), RetiredSemaphoreID: "retired"}
	loaded, err := q.LoadMaintenanceRetirement(ctx, checkpoint.JobID)
	require.NoError(t, err)
	require.Nil(t, loaded)
	require.NoError(t, q.SaveMaintenanceRetirement(ctx, checkpoint))
	require.NoError(t, q.SaveMaintenanceRetirement(ctx, checkpoint))
	loaded, err = q.LoadMaintenanceRetirement(ctx, checkpoint.JobID)
	require.NoError(t, err)
	require.Equal(t, checkpoint, *loaded)
	other := checkpoint
	other.RunID = ulid.Make()
	require.Error(t, q.SaveMaintenanceRetirement(ctx, other))
	require.Error(t, q.DeleteMaintenanceRetirement(ctx, other))
	loaded, err = q.LoadMaintenanceRetirement(ctx, checkpoint.JobID)
	require.NoError(t, err)
	require.Equal(t, checkpoint, *loaded, "conflicting operations must preserve the original evidence")
	require.NoError(t, q.DeleteMaintenanceRetirement(ctx, checkpoint))
	require.NoError(t, q.DeleteMaintenanceRetirement(ctx, checkpoint))
	loaded, err = q.LoadMaintenanceRetirement(ctx, checkpoint.JobID)
	require.NoError(t, err)
	require.Nil(t, loaded)
	require.NoError(t, r.Set(q.maintenanceRetirementKey(checkpoint.JobID), "not JSON"))
	_, err = q.LoadMaintenanceRetirement(ctx, checkpoint.JobID)
	require.Error(t, err, "malformed checkpoints must fail closed")
}

func TestMaintenanceQueueGuards(t *testing.T) {
	for _, name := range []string{"unleased edge", "leased start", "expired sleep", "capacity holder", "lease races deletion", "changed job races deletion"} {
		t.Run(name, func(t *testing.T) {
			r, client := initRedis(t)
			t.Cleanup(client.Close)
			clock := clockwork.NewFakeClock()
			_, shard := newQueue(t, client, osqueue.WithClock(clock))
			q := shard.(*queue)
			ctx := context.Background()
			fn, account, env, app := uuid.New(), uuid.New(), uuid.New(), uuid.New()
			scope := osqueue.Scope{FunctionID: fn, AccountID: account, EnvID: env}
			kind := osqueue.KindEdge
			if name == "leased start" {
				kind = osqueue.KindStart
			}
			if name == "expired sleep" {
				kind = osqueue.KindSleep
			}
			item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{ID: ulid.Make().String(), FunctionID: fn, WorkspaceID: env, Data: osqueue.Item{Kind: kind, Identifier: state.Identifier{RunID: ulid.Make(), WorkflowID: fn, AccountID: account, WorkspaceID: env, AppID: app}}}, clock.Now(), osqueue.EnqueueOpts{})
			require.NoError(t, err)
			observed, err := shard.LoadQueueItem(ctx, item.ID)
			require.NoError(t, err)
			kg := q.RedisClient.kg
			raw := r.HGet(kg.QueueItem(), item.ID)
			if name == "leased start" || name == "expired sleep" || name == "lease races deletion" {
				_, err := shard.Lease(ctx, item, time.Second, clock.Now())
				require.NoError(t, err)
				if name == "expired sleep" {
					clock.Advance(time.Minute)
				}
			}
			switch name {
			case "capacity holder":
				observed.CapacityLease = &osqueue.CapacityLease{LeaseID: ulid.Make()}
				encoded, err := json.Marshal(observed)
				require.NoError(t, err)
				r.HSet(kg.QueueItem(), item.ID, string(encoded))
			case "changed job races deletion":
				changed := *observed
				changed.GenerationID++
				encoded, err := json.Marshal(changed)
				require.NoError(t, err)
				r.HSet(kg.QueueItem(), item.ID, string(encoded))
			}
			count, err := q.MaintenanceLeaseCount(ctx, scope)
			require.NoError(t, err)
			if name == "leased start" || name == "expired sleep" || name == "lease races deletion" {
				require.EqualValues(t, 1, count)
			} else {
				require.Zero(t, count)
			}
			if name == "unleased edge" {
				statusCount, err := shard.StatusCount(ctx, scope, "in-progress")
				require.NoError(t, err)
				require.EqualValues(t, 1, statusCount, "status index is NOT an execution lease")
				require.NoError(t, q.DequeueMaintenance(ctx, *observed))
				return
			}
			if name == "lease races deletion" || name == "changed job races deletion" {
				// Simulate a mutation AFTER the maintenance reader's final raw read.
				// The exact same script the public maintenance method uses must refuse.
				err = q.dequeue(ctx, *observed, maintenanceDequeueScript(), []string{raw})
			} else {
				err = q.DequeueMaintenance(ctx, *observed)
			}
			require.Error(t, err)
			require.NotEmpty(t, r.HGet(kg.QueueItem(), item.ID))
			countJobs, err := shard.OutstandingJobCount(ctx, scope, item.Data.Identifier.RunID)
			require.NoError(t, err)
			require.Equal(t, 1, countJobs)
		})
	}
}
