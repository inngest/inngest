package queue

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestQueueRoleRequiresRoleLease(t *testing.T) {
	ctx := context.Background()
	role := &testQueueRole{
		name:          "blocked-role",
		leaseDuration: 300 * time.Millisecond,
		runInterval:   20 * time.Millisecond,
	}

	_, rc, q, shard := newRoleQueue(t, role, osqueue.QueueRunMode{})
	defer rc.Close()

	leaseID, err := shard.RoleLease(ctx, role.Name(), role.LeaseDuration())
	require.NoError(t, err)
	require.NotNil(t, leaseID)

	runCtx, cancel := context.WithCancel(ctx)
	done := runQueueForRoleTest(runCtx, q, nil)
	defer func() {
		cancel()
		requireQueueStopped(t, done)
	}()

	require.Never(t, func() bool {
		return role.runCount.Load() > 0
	}, 100*time.Millisecond, 5*time.Millisecond)
	require.Nil(t, activeRoleStatus(q.Queue(), role.Name()))
}

func TestQueueRoleRenewKeepsRoleActive(t *testing.T) {
	ctx := context.Background()
	role := &testQueueRole{
		name:          "renewed-role",
		leaseDuration: 150 * time.Millisecond,
		runInterval:   25 * time.Millisecond,
	}

	_, rc, q, _ := newRoleQueue(t, role, osqueue.QueueRunMode{})
	defer rc.Close()

	runCtx, cancel := context.WithCancel(ctx)
	done := runQueueForRoleTest(runCtx, q, nil)
	defer func() {
		cancel()
		requireQueueStopped(t, done)
	}()

	require.Eventually(t, func() bool {
		return activeRoleStatus(q.Queue(), role.Name()) != nil && role.runCount.Load() > 0
	}, time.Second, 5*time.Millisecond)

	firstLease := activeRoleStatus(q.Queue(), role.Name()).LeaseID

	require.Eventually(t, func() bool {
		status := activeRoleStatus(q.Queue(), role.Name())
		return status != nil && status.LeaseID != firstLease
	}, time.Second, 5*time.Millisecond)
	require.Greater(t, role.runCount.Load(), int32(1))
}

func TestQueueRoleStopsRunningAfterLeaseLoss(t *testing.T) {
	ctx := context.Background()
	role := &testQueueRole{
		name:          "lost-role",
		leaseDuration: 200 * time.Millisecond,
		runInterval:   20 * time.Millisecond,
	}

	r, rc, q, _ := newRoleQueue(t, role, osqueue.QueueRunMode{})
	defer rc.Close()

	runCtx, cancel := context.WithCancel(ctx)
	done := runQueueForRoleTest(runCtx, q, nil)
	defer func() {
		cancel()
		requireQueueStopped(t, done)
	}()

	require.Eventually(t, func() bool {
		return activeRoleStatus(q.Queue(), role.Name()) != nil && role.runCount.Load() > 0
	}, time.Second, 5*time.Millisecond)

	otherLease := ulid.MustNew(ulid.Timestamp(time.Now().Add(time.Second)), rand.Reader)
	require.NoError(t, r.Set(roleLeaseKey(role.Name()), otherLease.String()))

	require.Eventually(t, func() bool {
		return activeRoleStatus(q.Queue(), role.Name()) == nil
	}, time.Second, 5*time.Millisecond)

	afterLoss := role.runCount.Load()
	require.Never(t, func() bool {
		return role.runCount.Load() > afterLoss
	}, 75*time.Millisecond, 5*time.Millisecond)
}

func TestQueueRoleCanExcludeScanning(t *testing.T) {
	ctx := context.Background()
	role := &testQueueRole{
		name:             "exclusive-role",
		leaseDuration:    300 * time.Millisecond,
		runInterval:      50 * time.Millisecond,
		excludesScanning: true,
	}

	_, rc, q, _ := newRoleQueue(t, role, osqueue.QueueRunMode{
		Partition: true,
	})
	defer rc.Close()

	var processed atomic.Int32
	runCtx, cancel := context.WithCancel(ctx)
	done := runQueueForRoleTest(runCtx, q, func(ctx context.Context, info osqueue.RunInfo, item osqueue.Item) (osqueue.RunResult, error) {
		processed.Add(1)
		return osqueue.RunResult{}, nil
	})
	defer func() {
		cancel()
		requireQueueStopped(t, done)
	}()

	require.Eventually(t, func() bool {
		return activeRoleStatus(q.Queue(), role.Name()) != nil
	}, time.Second, 5*time.Millisecond)

	item := roleTestItem()
	require.NoError(t, q.Queue().Enqueue(ctx, item, time.Now(), osqueue.EnqueueOpts{}))

	require.Never(t, func() bool {
		return processed.Load() > 0
	}, 100*time.Millisecond, 5*time.Millisecond)
}

type testQueueRole struct {
	name             string
	leaseDuration    time.Duration
	runInterval      time.Duration
	excludesScanning bool
	runCount         atomic.Int32
}

func (r *testQueueRole) Name() string {
	return r.name
}

func (r *testQueueRole) LeaseDuration() time.Duration {
	return r.leaseDuration
}

func (r *testQueueRole) RunInterval() time.Duration {
	return r.runInterval
}

func (r *testQueueRole) ExcludesScanning() bool {
	return r.excludesScanning
}

func (r *testQueueRole) Run(ctx context.Context, shard osqueue.QueueShard) error {
	r.runCount.Add(1)
	return nil
}

func (r *testQueueRole) OnLeaseTick(ctx context.Context, shard osqueue.QueueShard) {
}

func newRoleQueue(t *testing.T, role osqueue.QueueRole, runMode osqueue.QueueRunMode) (*miniredis.Miniredis, rueidis.Client, osqueue.QueueProcessor, osqueue.QueueShard) {
	t.Helper()

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	options := []osqueue.QueueOpt{
		osqueue.WithQueueRoles(role),
		osqueue.WithRunMode(runMode),
		osqueue.WithPollTick(10 * time.Millisecond),
	}

	queueClient := redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey)
	shard := redis_state.NewQueueShard("test", queueClient, options...)
	registry, err := osqueue.NewSingleShardRegistry(shard)
	require.NoError(t, err)

	q, err := osqueue.New(context.Background(), "test", registry, options...)
	require.NoError(t, err)

	return r, rc, q, shard
}

func runQueueForRoleTest(ctx context.Context, q osqueue.QueueProcessor, f osqueue.RunFunc) chan struct{} {
	if f == nil {
		f = func(ctx context.Context, info osqueue.RunInfo, item osqueue.Item) (osqueue.RunResult, error) {
			return osqueue.RunResult{}, nil
		}
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = q.Run(ctx, f)
	}()
	return done
}

func requireQueueStopped(t *testing.T, done chan struct{}) {
	t.Helper()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("queue did not stop")
	}
}

func activeRoleStatus(q osqueue.RoleStatusReader, roleName string) *osqueue.QueueRoleStatus {
	for _, role := range q.ActiveRoles() {
		if role.Name == roleName {
			copied := role
			return &copied
		}
	}
	return nil
}

func roleLeaseKey(roleName string) string {
	return fmt.Sprintf("{%s}:queue:%s", redis_state.QueueDefaultKey, roleName)
}

func roleTestItem() osqueue.Item {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	return osqueue.Item{
		Kind: osqueue.KindStart,
		Identifier: state.Identifier{
			AccountID:   accountID,
			WorkspaceID: envID,
			WorkflowID:  fnID,
			RunID:       ulid.MustNew(ulid.Now(), rand.Reader),
		},
	}
}
