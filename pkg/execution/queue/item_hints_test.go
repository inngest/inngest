package queue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
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
	kind           enums.QueueShardKind
}

func (s *hintTestShard) Kind() enums.QueueShardKind {
	if s.kind != "" {
		return s.kind
	}
	return enums.QueueShardKindRedis
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

func hintTestQueue(t *testing.T, source ItemHintSource, extra ...QueueOpt) (*queueProcessor, *hintTestShard, *clockwork.FakeClock) {
	t.Helper()
	clock := clockwork.NewFakeClock()
	shard := &hintTestShard{mockShardForIterator: &mockShardForIterator{name: "ss3"}, item: QueueItem{
		AtMS: clock.Now().UnixMilli(), FunctionID: uuid.New(), GenerationID: 1,
		Data: Item{Kind: KindStart, Identifier: state.Identifier{AccountID: uuid.New()}},
	}}
	registry, err := NewSingleShardRegistry(shard)
	require.NoError(t, err)
	opts := []QueueOpt{WithClock(clock), WithPollTick(time.Millisecond),
		WithNumWorkers(2), WithRunMode(QueueRunMode{Partition: true}),
		WithItemHints(ItemHintOptions{Source: source, BufferSize: 2, AttemptTimeout: time.Second}),
	}
	q, err := New(t.Context(), "hint-test", registry, append(opts, extra...)...)
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

func TestItemHintWorkerCapacity(t *testing.T) {
	for _, workers := range []int{2, 32} {
		t.Run(fmt.Sprintf("workers=%d", workers), func(t *testing.T) {
			ready := make(chan func(QueueItem) bool, 1)
			q, shard, clock := hintTestQueue(t, func(ctx context.Context, _ QueueShard, offer func(QueueItem) bool) error {
				ready <- offer
				<-ctx.Done()
				return ctx.Err()
			}, WithNumWorkers(int32(workers)))
			q.itemHints.BufferSize = workers + 3
			q.runMode.Continuations = true
			dispatched := make(chan *observedHintCompletion, workers+3)
			stop := q.startItemHints(t.Context(), func(context.Context, ProcessItem) (DispatchedItem, error) {
				h := &observedHintCompletion{newDispatchedItemHandle(), make(chan struct{})}
				dispatched <- h
				return h, nil
			})
			t.Cleanup(stop)
			offer := <-ready
			for i := range workers + 3 {
				require.True(t, offer(hintItem(shard.item, fmt.Sprintf("item-%d", i))))
			}
			require.False(t, offer(hintItem(shard.item, "buffer-full")))
			clock.BlockUntil(1)
			clock.Advance(time.Millisecond)
			var handles []*observedHintCompletion
			// All available workers can receive hints on one tick, including
			// more than 16, while every previous dispatch is still executing.
			for range workers {
				select {
				case h := <-dispatched:
					handles = append(handles, h)
					select {
					case <-h.observing:
					case <-time.After(time.Second):
						t.Fatal("completion not observed")
					}
				case <-time.After(time.Second):
					t.Fatal("hint did not dispatch despite available workers")
				}
			}
			require.Eventually(t, func() bool { return shard.migrationCalls.Load() == int32(workers+3) }, time.Second, time.Millisecond)
			require.Zero(t, q.Semaphore().Available())
			// Completion observation remains independent of buffer draining.
			handles[0].complete(DispatchedItemResult{ScheduledImmediateJob: true})
			require.Eventually(t, func() bool {
				q.continuesLock.Lock()
				defer q.continuesLock.Unlock()
				return len(q.continues) == 1
			}, time.Second, time.Millisecond)
			stop()
			require.EqualValues(t, workers, shard.leaseCalls.Load(), "normal worker capacity prevents additional backend leases")
			require.Empty(t, dispatched)
			q.Semaphore().Release(int64(workers))
			for _, h := range handles[1:] {
				h.complete(DispatchedItemResult{})
			}
			require.False(t, offer(hintItem(shard.item, "after-shutdown")))
		})
	}
}

func TestItemHintDrainsSnapshotSerially(t *testing.T) {
	ready := make(chan func(QueueItem) bool, 1)
	q, shard, clock := hintTestQueue(t, func(ctx context.Context, _ QueueShard, offer func(QueueItem) bool) error {
		ready <- offer
		<-ctx.Done()
		return ctx.Err()
	})
	entered := make(chan struct{}, 3)
	releaseLease := make(chan struct{})
	release := sync.OnceFunc(func() { close(releaseLease) })
	shard.beforeLease = func() { entered <- struct{}{}; <-releaseLease }
	dispatched := make(chan string, 3)
	stop := q.startItemHints(t.Context(), func(_ context.Context, work ProcessItem) (DispatchedItem, error) {
		dispatched <- work.I.ID
		q.Semaphore().Release(1)
		return NewCompletedDispatchedItem(DispatchedItemResult{}), nil
	})
	t.Cleanup(func() { release(); stop() })
	offer := <-ready
	require.True(t, offer(hintItem(shard.item, "first")))
	require.True(t, offer(hintItem(shard.item, "second")))
	require.False(t, offer(hintItem(shard.item, "buffer-full")))
	clock.BlockUntil(1)
	clock.Advance(time.Millisecond)
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("lease not attempted")
	}
	// A new arrival fits after the first hint is removed but must wait for
	// another tick. The blocked first lease must not fan out other attempts.
	require.True(t, offer(hintItem(shard.item, "next-tick")))
	require.Never(t, func() bool { return shard.leaseCalls.Load() > 1 }, 20*time.Millisecond, time.Millisecond)
	release()
	for _, want := range []string{"first", "second"} {
		select {
		case got := <-dispatched:
			require.Equal(t, want, got)
		case <-time.After(time.Second):
			t.Fatal("snapshot did not drain")
		}
	}
	require.Never(t, func() bool { return shard.leaseCalls.Load() > 2 }, 20*time.Millisecond, time.Millisecond)
	clock.Advance(time.Millisecond)
	select {
	case got := <-dispatched:
		require.Equal(t, "next-tick", got)
	case <-time.After(time.Second):
		t.Fatal("new arrival was not processed on next tick")
	}
	clock.Advance(time.Millisecond)
	require.Never(t, func() bool { return shard.migrationCalls.Load() > 3 }, 20*time.Millisecond, time.Millisecond)
	stop()
	require.EqualValues(t, 3, shard.leaseCalls.Load(), "each admitted hint is attempted once")
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
	item.EnqueuedAt = clock.Now().Add(-100 * time.Millisecond).UnixMilli()
	item.AtMS = clock.Now().Add(2 * time.Second).UnixMilli()
	beforeCount, beforeSum, _ := hintMetrics(t)
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
	count, _, _ := hintMetrics(t)
	require.Equal(t, beforeCount, count, "leasing and waiting must not record a work start")
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
	count, sum, _ := hintMetrics(t)
	require.Equal(t, beforeCount+1, count)
	require.EqualValues(t, 2100, sum-beforeSum, "include enqueue delay and the wait until due")
}

func hintItem(item QueueItem, id string) QueueItem { item.ID = id; return item }
