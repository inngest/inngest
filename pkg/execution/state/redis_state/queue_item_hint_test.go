package redis_state

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestItemHintReadiness(t *testing.T) {
	for _, tc := range []struct {
		name                                       string
		backlog, future, paused, migrating, denied bool
	}{
		{name: "ready"}, {name: "backlog", backlog: true},
		{name: "future", future: true}, {name: "paused", paused: true},
		{name: "migrating", migrating: true}, {name: "denied", denied: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
			require.NoError(t, err)
			t.Cleanup(rc.Close)
			clock := clockwork.NewFakeClock()
			fn, acct, env := uuid.New(), uuid.New(), uuid.New()
			opts := []osqueue.QueueOpt{
				osqueue.WithClock(clock),
				osqueue.WithAllowKeyQueues(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) bool { return tc.backlog }),
				osqueue.WithPartitionPausedGetter(func(context.Context, uuid.UUID) osqueue.PartitionPausedInfo {
					return osqueue.PartitionPausedInfo{Paused: tc.paused}
				}),
			}
			if tc.denied {
				opts = append(opts, osqueue.WithDenyQueueNames(fn.String()))
			}
			_, shard := newQueue(t, rc, opts...)
			at := clock.Now()
			if tc.future {
				at = at.Add(time.Hour)
			}
			item, err := shard.EnqueueItem(t.Context(), osqueue.QueueItem{
				ID: "hint-test", AtMS: at.UnixMilli(), FunctionID: fn, WorkspaceID: env,
				Data: osqueue.Item{Kind: osqueue.KindStart, WorkspaceID: env, Identifier: state.Identifier{
					WorkflowID: fn, AccountID: acct, WorkspaceID: env, RunID: ulid.Make(),
				}},
			}, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			if tc.migrating {
				until := clock.Now().Add(time.Minute)
				require.NoError(t, shard.SetFunctionMigrate(t.Context(), osqueue.Scope{FunctionID: fn}, &until))
			}
			loaded, err := shard.(osqueue.ItemHintShard).LoadReadyItem(t.Context(), item.ID)
			if tc.name == "ready" {
				require.NoError(t, err)
				require.Equal(t, item.ID, loaded.ID)
				lease, err := shard.Lease(t.Context(), item, time.Minute, clock.Now(), osqueue.LeaseRequireReady())
				require.NoError(t, err)
				require.NotNil(t, lease)
				_, err = shard.Lease(t.Context(), item, time.Minute, clock.Now(), osqueue.LeaseRequireReady())
				require.ErrorIs(t, err, osqueue.ErrQueueItemAlreadyLeased)
			} else {
				require.ErrorIs(t, err, osqueue.ErrQueueItemNotReady)
			}
			if tc.backlog || tc.future {
				// The mutation guard also rejects a stale or bypassed point read.
				_, err = shard.Lease(t.Context(), item, time.Minute, clock.Now(), osqueue.LeaseRequireReady())
				require.ErrorIs(t, err, osqueue.ErrQueueItemNotReady)
				stored, err := shard.LoadQueueItem(t.Context(), item.ID)
				require.NoError(t, err)
				require.Nil(t, stored.LeaseID)
			}
		})
	}
}

// Suspend discovery to prove that backlog hints use constrained refill before
// leasing. The ordinary worker and completion path remain real.
type hintOnlyRedisShard struct {
	RedisQueueShard
	osqueue.ItemHintShard
	osqueue.ItemHintBacklogShard
}

func (s hintOnlyRedisShard) Run(ctx context.Context, _ osqueue.QueueScannerRuntime) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestItemHintBacklogHandoff(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		name := "refill and execute"
		if blocked {
			name = "shadow lease held"
		}
		t.Run(name, func(t *testing.T) {
			r := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
			require.NoError(t, err)
			t.Cleanup(rc.Close)
			opts := []osqueue.QueueOpt{osqueue.WithAllowKeyQueues(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) bool { return true }),
				osqueue.WithRunMode(osqueue.QueueRunMode{Partition: true}), osqueue.WithPollTick(5 * time.Millisecond), osqueue.WithNumWorkers(2)}
			base := NewQueueShard("ss3", NewQueueClient(rc, "{hints}"), opts...)
			wrapper := hintOnlyRedisShard{RedisQueueShard: base, ItemHintShard: base.(osqueue.ItemHintShard), ItemHintBacklogShard: base.(osqueue.ItemHintBacklogShard)}
			reg, err := osqueue.NewSingleShardRegistry(wrapper)
			require.NoError(t, err)
			offers := make(chan func(string) bool, 1)
			opts = append(opts, osqueue.WithItemHints(osqueue.ItemHintOptions{BufferSize: 2, MaxActive: 1, AttemptTimeout: time.Second, Source: func(ctx context.Context, _ osqueue.QueueShard, offer func(string) bool) error {
				offers <- offer
				<-ctx.Done()
				return nil
			}}))
			proc, err := osqueue.New(t.Context(), "hints", reg, opts...)
			require.NoError(t, err)
			job := "start"
			acct, env, fn := uuid.New(), uuid.New(), uuid.New()
			item := osqueue.Item{Kind: osqueue.KindStart, JobID: &job, WorkspaceID: env, Identifier: state.Identifier{AccountID: acct, WorkspaceID: env, WorkflowID: fn, RunID: ulid.Make()}}
			require.NoError(t, proc.Enqueue(t.Context(), item, time.Now(), osqueue.EnqueueOpts{}))
			id := osqueue.HashID(t.Context(), job)
			stored, backlog, err := wrapper.LoadBacklogItem(t.Context(), id)
			require.NoError(t, err)
			shadow := osqueue.ItemShadowPartition(t.Context(), *stored)
			if blocked {
				_, err = base.ShadowPartitionLease(t.Context(), &shadow, time.Minute)
				require.NoError(t, err)
			}
			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan error, 1)
			executed := make(chan struct{}, 1)
			go func() {
				done <- proc.Run(ctx, func(context.Context, osqueue.RunInfo, osqueue.Item) (osqueue.RunResult, error) {
					executed <- struct{}{}
					return osqueue.RunResult{}, nil
				})
			}()
			defer func() {
				cancel()
				select {
				case <-done:
				case <-time.After(2 * time.Second):
					t.Error("processor did not stop")
				}
			}()
			var offer func(string) bool
			select {
			case offer = <-offers:
			case <-time.After(time.Second):
				t.Fatal("hint receiver not ready")
			}
			require.True(t, offer(id))
			if blocked {
				select {
				case <-executed:
					t.Fatal("hint bypassed the shadow lease")
				case <-time.After(50 * time.Millisecond):
				}
				after, err := base.LoadQueueItem(t.Context(), id)
				require.NoError(t, err)
				require.Nil(t, after.LeaseID)
				count, err := base.BacklogSize(t.Context(), backlog.BacklogID)
				require.NoError(t, err)
				require.EqualValues(t, 1, count)
			} else {
				select {
				case <-executed:
				case <-time.After(2 * time.Second):
					t.Fatal("backlog hint did not dispatch")
				}
				require.Eventually(t, func() bool { _, err := base.LoadQueueItem(t.Context(), id); return err == osqueue.ErrQueueItemNotFound }, time.Second, time.Millisecond)
				count, err := base.BacklogSize(t.Context(), backlog.BacklogID)
				require.NoError(t, err)
				require.Zero(t, count)
			}
		})
	}
}

func TestItemHintBacklogPreservesCapacity(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
	require.NoError(t, err)
	t.Cleanup(rc.Close)
	cm, err := constraintapi.NewRedisCapacityManager(constraintapi.WithClient(rc), constraintapi.WithShardName("hints"))
	require.NoError(t, err)
	opts := []osqueue.QueueOpt{
		osqueue.WithAllowKeyQueues(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) bool { return true }),
		osqueue.WithCapacityManager(cm), osqueue.WithAcquireCapacityLeaseOnBacklogRefill(true),
		osqueue.WithPartitionConstraintConfigGetter(func(context.Context, osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
			return osqueue.PartitionConstraintConfig{FunctionVersion: 1, Concurrency: osqueue.PartitionConcurrency{AccountConcurrency: 1, FunctionConcurrency: 1}}
		}),
		osqueue.WithRunMode(osqueue.QueueRunMode{Partition: true}), osqueue.WithPollTick(5 * time.Millisecond), osqueue.WithNumWorkers(2),
	}
	base := NewQueueShard("ss3", NewQueueClient(rc, "{capacity-hints}"), opts...)
	reg, err := osqueue.NewSingleShardRegistry(hintOnlyRedisShard{base, base.(osqueue.ItemHintShard), base.(osqueue.ItemHintBacklogShard)})
	require.NoError(t, err)
	offers := make(chan func(string) bool, 1)
	opts = append(opts, osqueue.WithItemHints(osqueue.ItemHintOptions{BufferSize: 4, MaxActive: 2, AttemptTimeout: time.Second, Source: func(ctx context.Context, _ osqueue.QueueShard, offer func(string) bool) error {
		offers <- offer
		<-ctx.Done()
		return nil
	}}))
	proc, err := osqueue.New(t.Context(), "hints", reg, opts...)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	entered := make(chan string, 2)
	release := make(chan struct{})
	go func() {
		done <- proc.Run(ctx, func(_ context.Context, _ osqueue.RunInfo, item osqueue.Item) (osqueue.RunResult, error) {
			entered <- *item.JobID
			<-release
			return osqueue.RunResult{}, nil
		})
	}()
	defer func() {
		close(release)
		cancel()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("processor did not stop")
		}
	}()
	offer := <-offers
	acct, env, fn := uuid.New(), uuid.New(), uuid.New()
	enqueue := func(job string) string {
		item := osqueue.Item{Kind: osqueue.KindStart, JobID: &job, WorkspaceID: env, Identifier: state.Identifier{AccountID: acct, WorkspaceID: env, WorkflowID: fn, RunID: ulid.Make()}}
		require.NoError(t, proc.Enqueue(t.Context(), item, time.Now(), osqueue.EnqueueOpts{}))
		return osqueue.HashID(t.Context(), job)
	}
	first := enqueue("first")
	require.True(t, offer(first))
	select {
	case id := <-entered:
		require.Equal(t, first, id)
	case <-time.After(2 * time.Second):
		t.Fatal("first hint did not dispatch")
	}
	second := enqueue("second")
	require.True(t, offer(second))
	select {
	case <-entered:
		t.Fatal("hint bypassed function/account concurrency")
	case <-time.After(100 * time.Millisecond):
	}
	stored, err := base.LoadQueueItem(t.Context(), second)
	require.NoError(t, err)
	require.Nil(t, stored.LeaseID)
	require.Empty(t, stored.CapacityLease)
	backlog := osqueue.ItemBacklog(t.Context(), *stored)
	count, err := base.BacklogSize(t.Context(), backlog.BacklogID)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
}
