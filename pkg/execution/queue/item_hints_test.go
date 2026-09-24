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
	item        QueueItem
	validateErr error
	validations atomic.Int32
}

func (s *hintTestShard) ValidateItemForHint(ctx context.Context, item QueueItem) error {
	s.validations.Add(1)
	return s.validateErr
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
	ready := make(chan func(QueueItem) bool, 1)
	q, shard, clock := hintTestQueue(t, func(ctx context.Context, _ QueueShard, offer func(QueueItem) bool) error {
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
	require.True(t, offer(hintItem(shard.item, "first")))
	require.True(t, offer(hintItem(shard.item, "second")))
	require.False(t, offer(hintItem(shard.item, "overflow")))
	clock.BlockUntil(1)
	clock.Advance(time.Millisecond)
	var first *dispatchedItemHandle
	select {
	case first = <-dispatched:
	case <-time.After(time.Second):
		t.Fatal("hint not dispatched")
	}
	require.EqualValues(t, 1, shard.validations.Load(), "overflow and budget-limited hints must not validate or lease")
	require.True(t, offer(hintItem(shard.item, "while-active")))
	require.True(t, offer(hintItem(shard.item, "also-while-active")))
	require.False(t, offer(hintItem(shard.item, "still-full")))
	clock.Advance(time.Millisecond)
	// Let the tick consume this hint while the first job still holds its slot.
	require.Eventually(t, func() bool {
		return offer(hintItem(shard.item, "probe"))
	}, time.Second, time.Millisecond)
	stop()
	q.Semaphore().Release(1) // stand in for normal worker completion
	first.complete(DispatchedItemResult{})
	require.False(t, offer(hintItem(shard.item, "after-shutdown")))
	require.EqualValues(t, 1, shard.validations.Load())
}

func TestItemHintOutcomesReleaseWorkerCapacity(t *testing.T) {
	for _, tc := range []struct {
		name        string
		validateErr error
		kind        string
		dispatchErr error
		immediate   bool
		workerErr   error
	}{
		{name: "missing", validateErr: ErrQueueItemNotFound},
		{name: "eligibility-error", validateErr: errors.New("eligibility failed")},
		{name: "ineligible-kind", kind: KindSleep},
		{name: "dispatch-error", dispatchErr: errors.New("dispatch failed")},
		{name: "completed"},
		{name: "continuation", immediate: true},
		{name: "failed-worker", immediate: true, workerErr: errors.New("worker failed")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q, shard, _ := hintTestQueue(t, nil)
			q.runMode.Continuations = true
			shard.validateErr = tc.validateErr
			if tc.kind != "" {
				shard.item.Data.Kind = tc.kind
			}
			var calls int
			q.processItemHint(t.Context(), shard, hintItem(shard.item, "item"), func(context.Context, ProcessItem) (DispatchedItem, error) {
				calls++
				if tc.dispatchErr != nil {
					return nil, tc.dispatchErr
				}
				q.Semaphore().Release(1)
				return NewCompletedDispatchedItem(DispatchedItemResult{ScheduledImmediateJob: tc.immediate, Err: tc.workerErr}), nil
			})
			require.EqualValues(t, 4, q.Semaphore().Available())
			if tc.validateErr != nil || tc.kind != "" {
				require.Zero(t, calls)
			} else {
				require.Equal(t, 1, calls)
			}
			require.EqualValues(t, 1, shard.validations.Load())
			require.Equal(t, tc.immediate && tc.workerErr == nil, len(q.continues) == 1)
		})
	}
}

func hintItem(item QueueItem, id string) QueueItem {
	item.ID = id
	return item
}
