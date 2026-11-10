package queue

import (
	"context"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// a retried start item gives its function concurrency slot back with the
// lease.  a failed requeue leaves the item leased in the queue, and the
// queue scavenger requeues it later, so that case gives the slot back too.
// an item that is gone on requeue belongs to a finalized run, and finalize
// released the slot.
func TestProcessItemForcesSemaphoreReleaseOnRetry(t *testing.T) {
	manual := constraintapi.Semaphore{
		ID:      constraintapi.SemaphoreIDFn(uuid.New()),
		Weight:  1,
		Release: constraintapi.SemaphoreReleaseManual,
	}
	auto := constraintapi.Semaphore{
		ID:      constraintapi.SemaphoreIDApp(uuid.New()),
		Weight:  1,
		Release: constraintapi.SemaphoreReleaseAuto,
	}
	requeueFailed := errors.New("redis unavailable")

	tests := []struct {
		name       string
		semaphores []constraintapi.Semaphore
		requeueErr error
		wantErr    error
		wantForce  bool
	}{
		{
			name:       "requeue succeeds",
			semaphores: []constraintapi.Semaphore{manual},
			wantForce:  true,
		},
		{
			name:       "requeue fails",
			semaphores: []constraintapi.Semaphore{manual},
			requeueErr: requeueFailed,
			wantErr:    requeueFailed,
			wantForce:  true,
		},
		{
			name:       "item gone on requeue",
			semaphores: []constraintapi.Semaphore{manual},
			requeueErr: ErrQueueItemNotFound,
		},
		{
			name:       "no manual semaphore",
			semaphores: []constraintapi.Semaphore{auto},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			shard := &requeueErrorShard{
				mockShardForIterator: &mockShardForIterator{name: "test"},
				requeueErr:           tc.requeueErr,
			}
			registry, err := NewSingleShardRegistry(shard)
			require.NoError(t, err)

			releases := make(chan *constraintapi.CapacityReleaseRequest, 1)
			q, err := New(ctx, "test", registry, WithCapacityManager(releaseRecordingManager{
				releases: releases,
			}))
			require.NoError(t, err)

			capacityLeaseID, err := ulid.New(ulid.Timestamp(time.Now().Add(time.Minute)), rand.Reader)
			require.NoError(t, err)
			itemLeaseID := ulid.Make()
			item := QueueItem{
				ID:         "start-item",
				FunctionID: uuid.New(),
				LeaseID:    &itemLeaseID,
				Data: Item{
					Kind: KindStart,
					Identifier: state.Identifier{
						AccountID:   uuid.New(),
						WorkspaceID: uuid.New(),
						WorkflowID:  uuid.New(),
						RunID:       ulid.Make(),
					},
					Semaphores: tc.semaphores,
				},
			}

			_, err = q.ProcessItem(ctx, ProcessItem{
				I: item,
				CapacityLease: &CapacityLease{
					LeaseID:    capacityLeaseID,
					IssuedAtMS: time.Now().UnixMilli(),
				},
			}, func(context.Context, RunInfo, Item) (RunResult, error) {
				return RunResult{}, errors.New("sdk unreachable")
			})
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}

			select {
			case req := <-releases:
				require.Equal(t, capacityLeaseID, req.LeaseID)
				require.Equal(t, tc.wantForce, req.ForceReleaseSemaphores)
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for the capacity lease release")
			}
		})
	}
}

type requeueErrorShard struct {
	*mockShardForIterator
	requeueErr error
}

func (s *requeueErrorShard) Requeue(context.Context, QueueItem, time.Time, ...RequeueOptionFn) error {
	return s.requeueErr
}

type releaseRecordingManager struct {
	releases chan<- *constraintapi.CapacityReleaseRequest
}

func (m releaseRecordingManager) Check(context.Context, *constraintapi.CapacityCheckRequest) (*constraintapi.CapacityCheckResponse, errs.UserError, errs.InternalError) {
	return nil, nil, nil
}

func (m releaseRecordingManager) Acquire(context.Context, *constraintapi.CapacityAcquireRequest) (*constraintapi.CapacityAcquireResponse, errs.InternalError) {
	return nil, nil
}

func (m releaseRecordingManager) ExtendLease(context.Context, *constraintapi.CapacityExtendLeaseRequest) (*constraintapi.CapacityExtendLeaseResponse, errs.InternalError) {
	return nil, nil
}

func (m releaseRecordingManager) Release(_ context.Context, req *constraintapi.CapacityReleaseRequest) (*constraintapi.CapacityReleaseResponse, errs.InternalError) {
	m.releases <- req
	return &constraintapi.CapacityReleaseResponse{}, nil
}
