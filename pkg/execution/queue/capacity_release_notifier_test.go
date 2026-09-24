package queue

import (
	"context"
	"crypto/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestCapacityLeaseMutationOptions(t *testing.T) {
	dequeueOpts := DequeueOptions{}
	DequeueCapacityLeased()(&dequeueOpts)
	require.True(t, dequeueOpts.CapacityLeased)

	requeueOpts := RequeueOptions{}
	RequeueCapacityLeased()(&requeueOpts)
	require.True(t, requeueOpts.CapacityLeased)
}

func TestProcessItemNotifiesAfterCapacityRelease(t *testing.T) {
	ctx := context.Background()
	events := make(chan string, 3)
	dequeueStarted := make(chan struct{})
	continueDequeue := make(chan struct{})
	releaseReturned := make(chan struct{})
	shard := &capacityReleaseNotifierShard{
		mockShardForIterator: &mockShardForIterator{name: "test"},
		events:               events,
		dequeueStarted:       dequeueStarted,
		continueDequeue:      continueDequeue,
	}
	registry, err := NewSingleShardRegistry(shard)
	require.NoError(t, err)

	q, err := New(ctx, "test", registry, WithCapacityManager(capacityReleaseTestManager{
		events:              events,
		beforeReleaseReturn: dequeueStarted,
		releaseReturned:     releaseReturned,
	}))
	require.NoError(t, err)

	capacityLeaseID, err := ulid.New(ulid.Timestamp(time.Now().Add(time.Minute)), rand.Reader)
	require.NoError(t, err)
	itemLeaseID := ulid.Make()
	item := QueueItem{
		ID:         "item",
		FunctionID: uuid.New(),
		LeaseID:    &itemLeaseID,
		Data: Item{
			Identifier: state.Identifier{
				AccountID:   uuid.New(),
				WorkspaceID: uuid.New(),
				WorkflowID:  uuid.New(),
				RunID:       ulid.Make(),
			},
		},
	}

	processErr := make(chan error, 1)
	go func() {
		_, err := q.ProcessItem(ctx, ProcessItem{
			I: item,
			CapacityLease: &CapacityLease{
				LeaseID:    capacityLeaseID,
				IssuedAtMS: time.Now().UnixMilli(),
			},
		}, func(_ context.Context, info RunInfo, _ Item) (RunResult, error) {
			require.NoError(t, info.CapacityLease.Release())
			return RunResult{}, nil
		})
		processErr <- err
	}()

	require.Equal(t, "release", receiveCapacityReleaseEvent(t, events))
	<-releaseReturned
	select {
	case event := <-events:
		require.NotEqual(t, "notify", event, "notification arrived before the blocked dequeue completed")
	case <-time.After(100 * time.Millisecond):
	}
	close(continueDequeue)

	require.Equal(t, "dequeue", receiveCapacityReleaseEvent(t, events))
	require.NoError(t, <-processErr)
	require.Equal(t, "notify", receiveCapacityReleaseEvent(t, events))
	require.True(t, shard.dequeueCapacityLeased)
	require.Equal(t, item.ID, shard.notifiedItem.ID)
}

func TestProcessItemForwardsCapacityLeaseOnRetry(t *testing.T) {
	ctx := context.Background()
	shard := &capacityReleaseNotifierShard{
		mockShardForIterator: &mockShardForIterator{name: "test"},
		events:               make(chan string, 3),
	}
	registry, err := NewSingleShardRegistry(shard)
	require.NoError(t, err)

	q, err := New(ctx, "test", registry, WithCapacityManager(capacityReleaseTestManager{events: shard.events}))
	require.NoError(t, err)

	capacityLeaseID, err := ulid.New(ulid.Timestamp(time.Now().Add(time.Minute)), rand.Reader)
	require.NoError(t, err)
	itemLeaseID := ulid.Make()
	item := QueueItem{
		ID:         "retry-item",
		FunctionID: uuid.New(),
		LeaseID:    &itemLeaseID,
		Data: Item{
			Identifier: state.Identifier{
				AccountID:   uuid.New(),
				WorkspaceID: uuid.New(),
				WorkflowID:  uuid.New(),
				RunID:       ulid.Make(),
			},
		},
	}

	_, err = q.ProcessItem(ctx, ProcessItem{
		I: item,
		CapacityLease: &CapacityLease{
			LeaseID:    capacityLeaseID,
			IssuedAtMS: time.Now().UnixMilli(),
		},
	}, func(context.Context, RunInfo, Item) (RunResult, error) {
		return RunResult{}, AlwaysRetryError(context.DeadlineExceeded)
	})
	require.NoError(t, err)
	require.True(t, shard.requeueCapacityLeased)
}

func TestProcessItemRetriesFailedEarlyCapacityReleaseDuringCleanup(t *testing.T) {
	ctx := context.Background()
	events := make(chan string, 5)
	attempts := make(chan int32, 2)
	attemptCount := &atomic.Int32{}
	shard := &capacityReleaseNotifierShard{
		mockShardForIterator: &mockShardForIterator{name: "test"},
		events:               events,
	}
	registry, err := NewSingleShardRegistry(shard)
	require.NoError(t, err)

	q, err := New(ctx, "test", registry, WithCapacityManager(capacityReleaseTestManager{
		events:       events,
		attemptCount: attemptCount,
		attempts:     attempts,
		failFirst:    true,
	}))
	require.NoError(t, err)

	capacityLeaseID, err := ulid.New(ulid.Timestamp(time.Now().Add(time.Minute)), rand.Reader)
	require.NoError(t, err)
	itemLeaseID := ulid.Make()
	item := QueueItem{
		ID:         "release-retry-item",
		FunctionID: uuid.New(),
		LeaseID:    &itemLeaseID,
		Data: Item{Identifier: state.Identifier{
			AccountID:   uuid.New(),
			WorkspaceID: uuid.New(),
			WorkflowID:  uuid.New(),
			RunID:       ulid.Make(),
		}},
	}

	_, err = q.ProcessItem(ctx, ProcessItem{
		I: item,
		CapacityLease: &CapacityLease{
			LeaseID:    capacityLeaseID,
			IssuedAtMS: time.Now().UnixMilli(),
		},
	}, func(_ context.Context, info RunInfo, _ Item) (RunResult, error) {
		require.NoError(t, info.CapacityLease.Release())
		require.Equal(t, int32(1), <-attempts)
		return RunResult{}, nil
	})
	require.NoError(t, err)
	require.Equal(t, int32(2), <-attempts)

	for receiveCapacityReleaseEvent(t, events) != "notify" {
	}
	require.Equal(t, int32(2), attemptCount.Load())
	require.Equal(t, item.ID, shard.notifiedItem.ID)
}

func receiveCapacityReleaseEvent(t *testing.T, events <-chan string) string {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for capacity release event")
		return ""
	}
}

type capacityReleaseNotifierShard struct {
	*mockShardForIterator
	events                chan<- string
	dequeueStarted        chan<- struct{}
	continueDequeue       <-chan struct{}
	dequeueCapacityLeased bool
	requeueCapacityLeased bool
	notifiedItem          QueueItem
}

func (s *capacityReleaseNotifierShard) Dequeue(_ context.Context, _ QueueItem, opts ...DequeueOptionFn) error {
	parsed := DequeueOptions{}
	for _, apply := range opts {
		apply(&parsed)
	}
	s.dequeueCapacityLeased = parsed.CapacityLeased
	if s.dequeueStarted != nil {
		close(s.dequeueStarted)
	}
	if s.continueDequeue != nil {
		<-s.continueDequeue
	}
	s.events <- "dequeue"
	return nil
}

func (s *capacityReleaseNotifierShard) Requeue(_ context.Context, _ QueueItem, _ time.Time, opts ...RequeueOptionFn) error {
	parsed := RequeueOptions{}
	for _, apply := range opts {
		apply(&parsed)
	}
	s.requeueCapacityLeased = parsed.CapacityLeased
	return nil
}

func (s *capacityReleaseNotifierShard) CapacityReleased(_ context.Context, item QueueItem) {
	s.notifiedItem = item
	s.events <- "notify"
}

type capacityReleaseTestManager struct {
	events              chan<- string
	beforeReleaseReturn <-chan struct{}
	releaseReturned     chan<- struct{}
	attemptCount        *atomic.Int32
	attempts            chan<- int32
	failFirst           bool
}

func (m capacityReleaseTestManager) Check(context.Context, *constraintapi.CapacityCheckRequest) (*constraintapi.CapacityCheckResponse, errs.UserError, errs.InternalError) {
	return nil, nil, nil
}

func (m capacityReleaseTestManager) Acquire(context.Context, *constraintapi.CapacityAcquireRequest) (*constraintapi.CapacityAcquireResponse, errs.InternalError) {
	return nil, nil
}

func (m capacityReleaseTestManager) ExtendLease(context.Context, *constraintapi.CapacityExtendLeaseRequest) (*constraintapi.CapacityExtendLeaseResponse, errs.InternalError) {
	return nil, nil
}

func (m capacityReleaseTestManager) Release(context.Context, *constraintapi.CapacityReleaseRequest) (*constraintapi.CapacityReleaseResponse, errs.InternalError) {
	m.events <- "release"
	if m.attemptCount != nil {
		attempt := m.attemptCount.Add(1)
		if m.attempts != nil {
			m.attempts <- attempt
		}
		if m.failFirst && attempt == 1 {
			return nil, errs.Wrap(500, true, "test release failure")
		}
	}
	if m.beforeReleaseReturn != nil {
		<-m.beforeReleaseReturn
	}
	if m.releaseReturned != nil {
		close(m.releaseReturned)
	}
	return &constraintapi.CapacityReleaseResponse{}, nil
}
