package queue

import (
	"context"
	"crypto/rand"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// a lost item lease is a queue failure.  it requeues the item and keeps the
// attempt count, even on the last attempt.  a dequeue here leaves the run
// with no job and no finalize.
func TestProcessItemLeaseExtensionFailureDoesNotUseAnAttempt(t *testing.T) {
	ctx := context.Background()
	clock := clockwork.NewFakeClock()

	shard := &leaseExtensionErrorShard{
		mockShardForIterator: &mockShardForIterator{name: "test"},
		extendErr:            errors.New("redis unavailable"),
	}
	registry, err := NewSingleShardRegistry(shard)
	require.NoError(t, err)

	q, err := New(ctx, "test", registry, WithClock(clock))
	require.NoError(t, err)

	maxAttempts := 1
	itemLeaseID := ulid.Make()
	item := QueueItem{
		ID:         "last-attempt-item",
		FunctionID: uuid.New(),
		LeaseID:    &itemLeaseID,
		Data: Item{
			Kind:        KindEdge,
			Attempt:     0,
			MaxAttempts: &maxAttempts,
			Identifier: state.Identifier{
				AccountID:   uuid.New(),
				WorkspaceID: uuid.New(),
				WorkflowID:  uuid.New(),
				RunID:       ulid.Make(),
			},
		},
	}

	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := q.ProcessItem(ctx, ProcessItem{
			I: item,
		}, func(ctx context.Context, _ RunInfo, _ Item) (RunResult, error) {
			close(started)
			<-ctx.Done()
			return RunResult{}, ctx.Err()
		})
		done <- err
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for the job to start")
	}

	// tick the lease renewal until the failed extension ends processing.
	deadline := time.After(2 * time.Second)
	for finished := false; !finished; {
		select {
		case err := <-done:
			require.NoError(t, err)
			finished = true
		case <-deadline:
			t.Fatal("timed out waiting for ProcessItem to finish")
		case <-time.After(5 * time.Millisecond):
			clock.Advance(QueueLeaseDuration / 2)
		}
	}

	requeued, ok := shard.requeuedItem()
	require.True(t, ok, "the item must be requeued")
	require.Equal(t, 0, requeued.Data.Attempt, "a lost lease must not use up an attempt")
	require.Equal(t, int32(0), atomic.LoadInt32(&shard.dequeueCalls), "the item must not be dequeued")
}

func TestProcessItemReturnsWhenEveryErrorSenderFails(t *testing.T) {
	ctx := context.Background()
	clock := clockwork.NewFakeClock()
	shard := &heldLeaseExtensionShard{mockShardForIterator: &mockShardForIterator{name: "test"}}
	manager := &failingExtendManager{}
	registry, err := NewSingleShardRegistry(shard)
	require.NoError(t, err)
	q, err := New(ctx, "test", registry, WithClock(clock), WithCapacityManager(manager))
	require.NoError(t, err)

	synctest.Test(t, func(t *testing.T) {
		shard.inFlight, shard.fail = make(chan struct{}), make(chan struct{})
		manager.after = shard.inFlight

		itemLeaseID := ulid.Make()
		started := make(chan struct{})
		done := make(chan error, 1)
		go func() {
			_, err := q.ProcessItem(ctx, ProcessItem{
				I: QueueItem{
					ID:      "item",
					LeaseID: &itemLeaseID,
					Data:    Item{Kind: KindEdge},
				},
				CapacityLease: &CapacityLease{
					LeaseID: ulid.MustNew(ulid.Timestamp(clock.Now().Add(time.Minute)), rand.Reader),
				},
			}, func(ctx context.Context, _ RunInfo, _ Item) (RunResult, error) {
				close(started)
				<-ctx.Done()
				return RunResult{}, ctx.Err()
			})
			done <- err
		}()

		<-started
		clock.Advance(QueueLeaseDuration / 2)
		synctest.Wait()
		close(shard.fail)

		require.NoError(t, <-done)
	})
}

type leaseExtensionErrorShard struct {
	*mockShardForIterator
	extendErr error

	mu       sync.Mutex
	requeued *QueueItem
}

func (s *leaseExtensionErrorShard) ExtendLease(context.Context, QueueItem, ulid.ULID, time.Duration, ...ExtendLeaseOptionFn) (*ulid.ULID, error) {
	return nil, s.extendErr
}

func (s *leaseExtensionErrorShard) Requeue(_ context.Context, i QueueItem, _ time.Time, _ ...RequeueOptionFn) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requeued = &i
	return nil
}

func (s *leaseExtensionErrorShard) requeuedItem() (QueueItem, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.requeued == nil {
		return QueueItem{}, false
	}
	return *s.requeued, true
}

type heldLeaseExtensionShard struct {
	*mockShardForIterator
	inFlight chan struct{}
	fail     chan struct{}
}

func (s *heldLeaseExtensionShard) ExtendLease(context.Context, QueueItem, ulid.ULID, time.Duration, ...ExtendLeaseOptionFn) (*ulid.ULID, error) {
	close(s.inFlight)
	<-s.fail
	return nil, errors.New("redis unavailable")
}

type failingExtendManager struct {
	constraintapi.CapacityManager
	after <-chan struct{}
}

func (m *failingExtendManager) ExtendLease(context.Context, *constraintapi.CapacityExtendLeaseRequest) (*constraintapi.CapacityExtendLeaseResponse, errs.InternalError) {
	<-m.after
	return nil, errs.Wrap(0, true, "redis unavailable")
}

func (*failingExtendManager) Release(context.Context, *constraintapi.CapacityReleaseRequest) (*constraintapi.CapacityReleaseResponse, errs.InternalError) {
	return &constraintapi.CapacityReleaseResponse{}, nil
}
