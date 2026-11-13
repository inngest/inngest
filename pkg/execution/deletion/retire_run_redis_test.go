package deletion

import (
	"context"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// Optional real-engine coverage uses ONLY an explicitly supplied loopback server.
// All data is scoped by random account/function IDs; no FLUSH or global cleanup.
func TestRetirementRedisIntegration(t *testing.T) {
	addr := os.Getenv("RETIREMENT_TEST_REDIS_ADDR")
	if addr == "" {
		addr = miniredis.RunT(t).Addr()
	}
	host, _, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	require.Contains(t, []string{"127.0.0.1", "::1"}, host)
	client, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{addr}, DisableCache: true})
	require.NoError(t, err)
	t.Cleanup(client.Close)
	ctx := context.Background()
	id := sv2.ID{RunID: ulid.Make(), FunctionID: uuid.New(), Tenant: sv2.Tenant{AccountID: uuid.New(), EnvID: uuid.New(), AppID: uuid.New()}}
	expression, bucket := "event.data.app_id", "legacy-app"
	sem := constraintapi.Semaphore{ID: constraintapi.SemaphoreIDFnKey(id.FunctionID, expression), EvaluatedKeyHash: bucket, Weight: 1, Release: constraintapi.SemaphoreReleaseManual}
	usage := (&constraintapi.SemaphoreConstraint{ID: sem.ID, EvaluatedKeyHash: bucket}).UsageKey(id.Tenant.AccountID)
	otherUsage := (&constraintapi.SemaphoreConstraint{ID: constraintapi.SemaphoreIDFnKey(uuid.New(), expression), EvaluatedKeyHash: bucket}).UsageKey(id.Tenant.AccountID)
	require.NoError(t, client.Do(ctx, client.B().Set().Key(usage).Value("1").Build()).Error())
	require.NoError(t, client.Do(ctx, client.B().Set().Key(otherUsage).Value("7").Build()).Error())
	unsharded := redis_state.NewUnshardedClient(client, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	shard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unsharded.Queue())
	item, err := shard.EnqueueItem(ctx, queue.QueueItem{ID: ulid.Make().String(), FunctionID: id.FunctionID, WorkspaceID: id.Tenant.EnvID, Data: queue.Item{Kind: queue.KindStart, Identifier: state.Identifier{RunID: id.RunID, WorkflowID: id.FunctionID, AccountID: id.Tenant.AccountID, WorkspaceID: id.Tenant.EnvID, AppID: id.Tenant.AppID, Semaphores: []constraintapi.Semaphore{sem}}, Semaphores: []constraintapi.Semaphore{sem}}}, time.Now(), queue.EnqueueOpts{})
	require.NoError(t, err)
	failure := errors.New("state deletion unavailable")
	s := &retirementState{absent: true, deleteErr: failure}
	guard := func(ctx context.Context) error {
		return constraintapi.CheckFunctionLeasesDrained(ctx, client, id.Tenant.AccountID, id.FunctionID)
	}
	request := RunRetirement{ID: id, JobID: item.ID, Before: time.Now(), RetiredSemaphoreID: sem.ID, RequireRetiredSemaphore: true}
	_, err = RetireMaintenanceRun(ctx, shard, s, request, guard)
	require.ErrorIs(t, err, failure)
	_, err = shard.LoadQueueItem(ctx, item.ID)
	require.ErrorIs(t, err, queue.ErrQueueItemNotFound)
	// Recreate the shard wrapper: retry evidence must survive outside the
	// previous caller's memory and the already-removed queue job.
	shard = redis_state.NewQueueShard(consts.DefaultQueueShardName, unsharded.Queue())
	checkpoints := shard.(queue.MaintenanceRetirementStore)
	checkpoint, err := checkpoints.LoadMaintenanceRetirement(ctx, item.ID)
	require.NoError(t, err)
	require.NotNil(t, checkpoint)
	s.deleteErr = nil
	_, err = RetireMaintenanceRun(ctx, shard, s, request, guard)
	require.NoError(t, err)
	checkpoint, err = checkpoints.LoadMaintenanceRetirement(ctx, item.ID)
	require.NoError(t, err)
	require.Nil(t, checkpoint)
	value, err := client.Do(ctx, client.B().Get().Key(usage).Build()).ToString()
	require.NoError(t, err)
	require.Equal(t, "1", value, "retirement must not decrement an uncertain legacy permit")
	value, err = client.Do(ctx, client.B().Get().Key(otherUsage).Build()).ToString()
	require.NoError(t, err)
	require.Equal(t, "7", value)
	_, err = RetireMaintenanceRun(ctx, shard, s, request, guard)
	require.NoError(t, err)
	value, err = client.Do(ctx, client.B().Get().Key(usage).Build()).ToString()
	require.NoError(t, err)
	require.Equal(t, "1", value, "idempotent cleanup also leaves the old counter untouched")
}
