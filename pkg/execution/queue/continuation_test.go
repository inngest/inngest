package queue

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWatchDispatchedPartitionItemAddsContinuation(t *testing.T) {
	ctx := context.Background()
	fnID := uuid.New()
	q := &queueProcessor{
		QueueOptions:  NewQueueOptions(),
		continues:     map[string]continuation{},
		continuesLock: &sync.Mutex{},
	}
	q.runMode.Continuations = true

	partition := &QueuePartition{
		ID:         fnID.String(),
		FunctionID: &fnID,
		AccountID:  uuid.New(),
	}
	handle := newDispatchedItemHandle()

	q.watchDispatchedPartitionItem(ctx, handle, partition, 1)
	handle.complete(DispatchedItemResult{ScheduledImmediateJob: true})

	require.Eventually(t, func() bool {
		q.continuesLock.Lock()
		defer q.continuesLock.Unlock()

		cont, ok := q.continues[partition.Queue()]
		return ok && cont.count == 2 && cont.partition.ID == partition.ID
	}, time.Second, time.Millisecond)
}

func TestWatchDispatchedPartitionItemSkipsContinuationOnError(t *testing.T) {
	ctx := context.Background()
	fnID := uuid.New()
	q := &queueProcessor{
		QueueOptions:  NewQueueOptions(),
		continues:     map[string]continuation{},
		continuesLock: &sync.Mutex{},
	}
	q.runMode.Continuations = true

	partition := &QueuePartition{
		ID:         fnID.String(),
		FunctionID: &fnID,
		AccountID:  uuid.New(),
	}
	handle := newDispatchedItemHandle()

	q.watchDispatchedPartitionItem(ctx, handle, partition, 1)
	handle.complete(DispatchedItemResult{
		ScheduledImmediateJob: true,
		Err:                   context.Canceled,
	})

	require.Never(t, func() bool {
		q.continuesLock.Lock()
		defer q.continuesLock.Unlock()

		_, ok := q.continues[partition.Queue()]
		return ok
	}, 50*time.Millisecond, time.Millisecond)
}
