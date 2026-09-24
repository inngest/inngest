package queue

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type hintTestShard struct {
	*mockShardForIterator
	item           QueueItem
	leaseErr       error
	leaseCalls     atomic.Int32
	migrationCalls atomic.Int32
	locked         *time.Time
	migrationErr   error
	beforeLease    func()
	leaseNow       time.Time
}

func (s *hintTestShard) Lease(ctx context.Context, item QueueItem, duration time.Duration, now time.Time, opts ...LeaseOptionFn) (*ulid.ULID, error) {
	s.leaseCalls.Add(1)
	if s.beforeLease != nil {
		s.beforeLease()
	}
	s.leaseNow = now
	if s.leaseErr != nil {
		return nil, s.leaseErr
	}
	lease := ulid.MustNew(ulid.Timestamp(now.Add(duration)), nil)
	return &lease, nil
}
func (s *hintTestShard) IsMigrationLocked(context.Context, Scope) (*time.Time, error) {
	s.migrationCalls.Add(1)
	return s.locked, s.migrationErr
}
func (s *hintTestShard) LoadQueueItem(context.Context, string) (*QueueItem, error) {
	panic("hints must not reload the item")
}

func hintTestQueue(t *testing.T, source ItemHintSource) (*queueProcessor, *hintTestShard, *clockwork.FakeClock) {
	t.Helper()
	clock := clockwork.NewFakeClock()
	shard := &hintTestShard{mockShardForIterator: &mockShardForIterator{name: "ss3"}, item: QueueItem{
		AtMS: clock.Now().UnixMilli(), FunctionID: uuid.New(), GenerationID: 1,
		Data: Item{Kind: KindStart, Identifier: state.Identifier{AccountID: uuid.New()}},
	}}
	registry, err := NewSingleShardRegistry(shard)
	require.NoError(t, err)
	q, err := New(t.Context(), "hint-test", registry, WithClock(clock), WithPollTick(time.Millisecond),
		WithNumWorkers(2), WithRunMode(QueueRunMode{Partition: true}),
		WithItemHints(ItemHintOptions{Source: source, BufferSize: 2, MaxActive: 1, AttemptTimeout: time.Second}),
	)
	require.NoError(t, err)
	return q, shard, clock
}

type observedHintCompletion struct {
	*dispatchedItemHandle
	observing chan struct{}
}

func (h *observedHintCompletion) Done() <-chan DispatchedItemResult {
	close(h.observing)
	return h.dispatchedItemHandle.Done()
}

func TestItemHintAttemptBudgetAndWorkerCapacity(t *testing.T) {
	ready := make(chan func(QueueItem) bool, 1)
	q, shard, clock := hintTestQueue(t, func(ctx context.Context, _ QueueShard, offer func(QueueItem) bool) error {
		ready <- offer
		<-ctx.Done()
		return ctx.Err()
	})
	q.runMode.Continuations = true
	entered := make(chan struct{}, 4)
	releaseLease := make(chan struct{})
	shard.beforeLease = func() { entered <- struct{}{}; <-releaseLease }
	dispatched := make(chan *observedHintCompletion, 4)
	stop := q.startItemHints(t.Context(), func(context.Context, ProcessItem) (DispatchedItem, error) {
		h := &observedHintCompletion{newDispatchedItemHandle(), make(chan struct{})}
		dispatched <- h
		return h, nil
	})
	t.Cleanup(stop)
	offer := <-ready
	require.True(t, offer(hintItem(shard.item, "first")))
	require.True(t, offer(hintItem(shard.item, "attempt-budget")))
	require.False(t, offer(hintItem(shard.item, "buffer-full")))
	clock.BlockUntil(1)
	clock.Advance(time.Millisecond)
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("lease not attempted")
	}
	require.Never(t, func() bool { return shard.leaseCalls.Load() > 1 }, 20*time.Millisecond, time.Millisecond)
	close(releaseLease)
	next := func() *observedHintCompletion {
		t.Helper()
		select {
		case h := <-dispatched:
			select {
			case <-h.observing:
			case <-time.After(time.Second):
				t.Fatal("completion not observed")
			}
			return h
		case <-time.After(time.Second):
			t.Fatal("hint not dispatched")
			return nil
		}
	}
	first := next()
	require.EqualValues(t, 1, q.Semaphore().Available())
	// The first worker is still running, but its attempt slot is already free.
	require.True(t, offer(hintItem(shard.item, "second-worker")))
	clock.Advance(time.Millisecond)
	second := next()
	require.Zero(t, q.Semaphore().Available())
	require.True(t, offer(hintItem(shard.item, "worker-full")))
	clock.Advance(time.Millisecond)
	require.Eventually(t, func() bool { return shard.migrationCalls.Load() == 3 }, time.Second, time.Millisecond)
	require.Never(t, func() bool { return shard.leaseCalls.Load() > 2 }, 20*time.Millisecond, time.Millisecond)
	// Failed capacity checks release the attempt too; workers remain authoritative.
	q.Semaphore().Release(1)
	first.complete(DispatchedItemResult{ScheduledImmediateJob: true})
	require.True(t, offer(hintItem(shard.item, "after-worker")))
	clock.Advance(time.Millisecond)
	third := next()
	stop()
	q.Semaphore().Release(2)
	second.complete(DispatchedItemResult{})
	third.complete(DispatchedItemResult{})
	require.False(t, offer(hintItem(shard.item, "after-shutdown")))
	require.EqualValues(t, 3, shard.leaseCalls.Load(), "dropped hints are never replayed")
	require.Len(t, q.continues, 1)
}

func TestItemHintLookahead(t *testing.T) {
	for _, offset := range []time.Duration{-time.Second, 0, 2*time.Second - time.Millisecond, 2 * time.Second, 2*time.Second + time.Millisecond, time.Hour} {
		t.Run(offset.String(), func(t *testing.T) {
			q, shard, clock := hintTestQueue(t, nil)
			var pauseCalls int
			q.PartitionPausedGetter = func(context.Context, uuid.UUID) PartitionPausedInfo { pauseCalls++; return PartitionPausedInfo{} }
			item := hintItem(shard.item, "lookahead")
			item.AtMS = clock.Now().Add(offset).UnixMilli()
			calls := 0
			got := q.processItemHint(t.Context(), item, func(_ context.Context, work ProcessItem) (DispatchedItem, error) {
				calls++
				require.Equal(t, item.AtMS, work.I.AtMS)
				require.Equal(t, item.ID, *work.I.Data.JobID)
				require.Equal(t, ulid.Timestamp(clock.Now().Add(QueueLeaseDuration)), work.I.LeaseID.Time())
				q.Semaphore().Release(1)
				return NewCompletedDispatchedItem(DispatchedItemResult{}), nil
			})
			if offset <= 2*time.Second {
				require.NotNil(t, got)
				require.Equal(t, 1, calls)
				require.Equal(t, clock.Now(), shard.leaseNow, "lookahead must not advance lease time")
			} else {
				require.Nil(t, got)
				require.Zero(t, calls)
				require.Zero(t, pauseCalls, "far-future items must be dropped before any backend callback")
				require.Zero(t, shard.migrationCalls.Load())
				require.Zero(t, shard.leaseCalls.Load())
			}
		})
	}
}

func TestItemHintEligibilityAndOutcomes(t *testing.T) {
	for _, name := range []string{"missing", "leased", "lease-error", "migration-error", "migrating", "paused", "denied", "allowed-prefix", "denied-prefix", "unlisted", "exclusive-account", "system", "wrong-kind", "retry", "dispatch-error", "completed", "continuation", "failed-worker"} {
		t.Run(name, func(t *testing.T) {
			q, shard, clock := hintTestQueue(t, nil)
			q.runMode.Continuations = true
			item := hintItem(shard.item, "item")
			reject := true
			switch name {
			case "missing":
				shard.leaseErr = ErrQueueItemNotFound
			case "leased":
				shard.leaseErr = ErrQueueItemAlreadyLeased
			case "lease-error":
				shard.leaseErr = errors.New("lease failed")
			case "migration-error":
				shard.migrationErr = errors.New("lookup failed")
			case "migrating":
				until := clock.Now().Add(time.Minute)
				shard.locked = &until
			case "paused":
				q.PartitionPausedGetter = func(context.Context, uuid.UUID) PartitionPausedInfo { return PartitionPausedInfo{Paused: true} }
			case "denied":
				q.DenyQueues = []string{item.FunctionID.String()}
			case "allowed-prefix":
				q.AllowQueues = []string{item.FunctionID.String()[:8] + "*"}
				reject = false
			case "denied-prefix":
				q.AllowQueues = []string{"*"}
				q.DenyQueues = []string{item.FunctionID.String()[:8] + "*"}
			case "unlisted":
				q.AllowQueues = []string{uuid.NewString()}
			case "exclusive-account":
				q.runMode.ExclusiveAccounts = []uuid.UUID{uuid.New()}
			case "system":
				system := "system"
				item.QueueName = &system
			case "wrong-kind":
				item.Data.Kind = KindSleep
			case "retry":
				item.Data.Attempt = 1
			default:
				reject = false
			}
			calls := 0
			got := q.processItemHint(t.Context(), item, func(context.Context, ProcessItem) (DispatchedItem, error) {
				calls++
				if name == "dispatch-error" {
					return nil, errors.New("dispatch failed")
				}
				q.Semaphore().Release(1)
				var workerErr error
				if name == "failed-worker" {
					workerErr = errors.New("worker failed")
				}
				return NewCompletedDispatchedItem(DispatchedItemResult{ScheduledImmediateJob: name == "continuation" || name == "failed-worker", Err: workerErr}), nil
			})
			if got != nil {
				q.observeItemHint(t.Context(), ItemPartition(t.Context(), item), got)
			}
			require.EqualValues(t, 2, q.Semaphore().Available())
			if reject {
				require.Zero(t, calls)
			} else {
				require.Equal(t, 1, calls)
			}
			require.Equal(t, name == "continuation", len(q.continues) == 1)
		})
	}
}

type hintWaitClock struct {
	*clockwork.FakeClock
	waiting chan time.Duration
}

func (c hintWaitClock) After(d time.Duration) <-chan time.Time {
	timer := c.FakeClock.After(d)
	c.waiting <- d
	return timer
}

func TestItemHintLeasesEarlyButWorkerWaits(t *testing.T) {
	q, shard, clock := hintTestQueue(t, nil)
	waiting := make(chan time.Duration, 1)
	q.QueueOptions.Clock = hintWaitClock{clock, waiting}
	item := hintItem(shard.item, "early")
	item.AtMS = clock.Now().Add(2 * time.Second).UnixMilli()
	executed := make(chan time.Time, 1)
	done := make(chan error, 1)
	got := q.processItemHint(t.Context(), item, func(ctx context.Context, work ProcessItem) (DispatchedItem, error) {
		handle := newDispatchedItemHandle()
		go func() {
			_, err := q.ProcessItem(context.WithoutCancel(ctx), work, func(context.Context, RunInfo, Item) (RunResult, error) {
				executed <- clock.Now()
				return RunResult{}, nil
			})
			q.Semaphore().Release(1)
			handle.complete(DispatchedItemResult{Err: err})
			done <- err
		}()
		return handle, nil
	})
	require.NotNil(t, got)
	select {
	case delay := <-waiting:
		require.InDelta(t, 2000, delay.Milliseconds(), 1)
	case <-time.After(time.Second):
		t.Fatal("worker did not wait")
	}
	clock.Advance(1999 * time.Millisecond)
	select {
	case <-executed:
		t.Fatal("worker executed early")
	case <-time.After(20 * time.Millisecond):
	}
	clock.Advance(time.Millisecond)
	select {
	case at := <-executed:
		require.Equal(t, item.AtMS, at.UnixMilli())
	case <-time.After(time.Second):
		t.Fatal("worker did not execute when due")
	}
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("worker did not finish")
	}
}

func hintItem(item QueueItem, id string) QueueItem { item.ID = id; return item }
