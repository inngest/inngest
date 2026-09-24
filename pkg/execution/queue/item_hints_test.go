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
	"github.com/stretchr/testify/require"
)

type hintTestShard struct {
	*mockShardForIterator
	item    QueueItem
	loadErr error
	loads   atomic.Int32
}

func (s *hintTestShard) LoadReadyItem(ctx context.Context, id string) (*QueueItem, error) {
	s.loads.Add(1)
	item := s.item
	item.ID = id
	return &item, s.loadErr
}

func hintTestQueue(t *testing.T, source ItemHintSource) (*queueProcessor, *hintTestShard, *clockwork.FakeClock) {
	t.Helper()
	clock := clockwork.NewFakeClock()
	shard := &hintTestShard{mockShardForIterator: &mockShardForIterator{name: "ss3"}, item: QueueItem{
		AtMS: clock.Now().UnixMilli(), FunctionID: uuid.New(),
		Data: Item{Kind: KindStart, Identifier: state.Identifier{AccountID: uuid.New()}},
	}}
	registry, err := NewSingleShardRegistry(shard)
	require.NoError(t, err)
	q, err := New(t.Context(), "hint-test", registry, WithClock(clock), WithPollTick(time.Millisecond),
		WithNumWorkers(4), WithRunMode(QueueRunMode{Partition: true}),
		WithItemHints(ItemHintOptions{Source: source, BufferSize: 2, MaxActive: 1, AttemptTimeout: time.Second}),
	)
	require.NoError(t, err)
	return q, shard, clock
}

func TestItemHintAdmissionAndActiveBudget(t *testing.T) {
	ready := make(chan func(string) bool, 1)
	q, shard, clock := hintTestQueue(t, func(ctx context.Context, _ QueueShard, offer func(string) bool) error {
		ready <- offer
		<-ctx.Done()
		return ctx.Err()
	})
	dispatched := make(chan *dispatchedItemHandle, 2)
	stop := q.startItemHints(t.Context(), func(context.Context, ProcessItem) (DispatchedItem, error) {
		handle := newDispatchedItemHandle()
		dispatched <- handle
		return handle, nil
	})
	t.Cleanup(stop)
	offer := <-ready
	require.True(t, offer("first"))
	require.True(t, offer("second"))
	require.False(t, offer("overflow"))
	clock.BlockUntil(1)
	clock.Advance(time.Millisecond)
	var first *dispatchedItemHandle
	select {
	case first = <-dispatched:
	case <-time.After(time.Second):
		t.Fatal("hint not dispatched")
	}
	require.EqualValues(t, 1, shard.loads.Load(), "overflow and budget-limited hints must not read or lease")
	require.True(t, offer("while-active"))
	require.True(t, offer("also-while-active"))
	require.False(t, offer("still-full"))
	clock.Advance(time.Millisecond)
	// Let the tick consume this hint while the first job still holds its slot.
	require.Eventually(t, func() bool {
		return offer("probe")
	}, time.Second, time.Millisecond)
	q.Semaphore().Release(1) // stand in for normal worker completion
	first.complete(DispatchedItemResult{})
	stop()
	require.False(t, offer("after-shutdown"))
	require.EqualValues(t, 1, shard.loads.Load())
}

func TestItemHintOutcomesReleaseWorkerCapacity(t *testing.T) {
	for _, tc := range []struct {
		name        string
		loadErr     error
		kind        string
		dispatchErr error
		immediate   bool
		workerErr   error
	}{
		{name: "missing", loadErr: ErrQueueItemNotFound},
		{name: "read-error", loadErr: errors.New("read failed")},
		{name: "ineligible-kind", kind: KindSleep},
		{name: "dispatch-error", dispatchErr: errors.New("dispatch failed")},
		{name: "completed"},
		{name: "continuation", immediate: true},
		{name: "failed-worker", immediate: true, workerErr: errors.New("worker failed")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q, shard, _ := hintTestQueue(t, nil)
			q.runMode.Continuations = true
			shard.loadErr = tc.loadErr
			if tc.kind != "" {
				shard.item.Data.Kind = tc.kind
			}
			var calls int
			q.processItemHint(t.Context(), shard, "item", func(context.Context, ProcessItem) (DispatchedItem, error) {
				calls++
				if tc.dispatchErr != nil {
					return nil, tc.dispatchErr
				}
				q.Semaphore().Release(1)
				return NewCompletedDispatchedItem(DispatchedItemResult{ScheduledImmediateJob: tc.immediate, Err: tc.workerErr}), nil
			})
			require.EqualValues(t, 4, q.Semaphore().Available())
			if tc.loadErr != nil || tc.kind != "" {
				require.Zero(t, calls)
			} else {
				require.Equal(t, 1, calls)
			}
			require.EqualValues(t, 1, shard.loads.Load())
			require.Equal(t, tc.immediate && tc.workerErr == nil, len(q.continues) == 1)
		})
	}
}
