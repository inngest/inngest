package queue

import (
	"context"
	"crypto/rand"
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
	shard := &capacityReleaseNotifierShard{
		mockShardForIterator: &mockShardForIterator{name: "test"},
		events:               events,
	}
	registry, err := NewSingleShardRegistry(shard)
	require.NoError(t, err)

	q, err := New(ctx, "test", registry, WithCapacityManager(capacityReleaseTestManager{events: events}))
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

	_, err = q.ProcessItem(ctx, ProcessItem{
		I: item,
		CapacityLease: &CapacityLease{
			LeaseID:    capacityLeaseID,
			IssuedAtMS: time.Now().UnixMilli(),
		},
	}, func(context.Context, RunInfo, Item) (RunResult, error) {
		return RunResult{}, nil
	})
	require.NoError(t, err)

	require.Equal(t, "dequeue", receiveCapacityReleaseEvent(t, events))
	require.Equal(t, "release", receiveCapacityReleaseEvent(t, events))
	require.Equal(t, "notify", receiveCapacityReleaseEvent(t, events))
	require.True(t, shard.dequeueCapacityLeased)
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
	dequeueCapacityLeased bool
	notifiedItem          QueueItem
}

func (s *capacityReleaseNotifierShard) Dequeue(_ context.Context, _ QueueItem, opts ...DequeueOptionFn) error {
	parsed := DequeueOptions{}
	for _, apply := range opts {
		apply(&parsed)
	}
	s.dequeueCapacityLeased = parsed.CapacityLeased
	s.events <- "dequeue"
	return nil
}

func (s *capacityReleaseNotifierShard) CapacityReleased(_ context.Context, item QueueItem) {
	s.notifiedItem = item
	s.events <- "notify"
}

type capacityReleaseTestManager struct {
	events chan<- string
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
	return &constraintapi.CapacityReleaseResponse{}, nil
}
