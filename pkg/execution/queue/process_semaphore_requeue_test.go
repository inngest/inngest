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

// an item with a manual semaphore gives its function concurrency slot back
// with the capacity lease on every exit, unless the run function calls
// CapacityLease.KeepManualSemaphores.  after that call the run has the slot,
// and a retry of the item no longer carries the manual semaphore.
func TestProcessItemForcesSemaphoreRelease(t *testing.T) {
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
	both := []constraintapi.Semaphore{auto, manual}
	jobFailed := errors.New("sdk unreachable")
	requeueFailed := errors.New("redis unavailable")
	one := 1

	tests := []struct {
		name        string
		semaphores  []constraintapi.Semaphore
		maxAttempts *int
		keep        bool
		runErr      error
		requeueErr  error
		wantErr     error
		wantForce   bool
		// wantRequeued is the semaphore list of the item that is requeued.
		wantRequeued []constraintapi.Semaphore
	}{
		{
			name:         "run not started, job fails, requeue succeeds",
			semaphores:   both,
			runErr:       jobFailed,
			wantForce:    true,
			wantRequeued: both,
		},
		{
			name:         "run not started, job fails, requeue fails",
			semaphores:   both,
			runErr:       jobFailed,
			requeueErr:   requeueFailed,
			wantErr:      requeueFailed,
			wantForce:    true,
			wantRequeued: both,
		},
		{
			name:         "run not started, job fails, item gone on requeue",
			semaphores:   both,
			runErr:       jobFailed,
			requeueErr:   ErrQueueItemNotFound,
			wantForce:    true,
			wantRequeued: both,
		},
		{
			name:        "run not started, job fails on the last attempt",
			semaphores:  both,
			maxAttempts: &one,
			runErr:      jobFailed,
			wantForce:   true,
		},
		{
			name:       "run not started, job succeeds",
			semaphores: both,
			wantForce:  true,
		},
		{
			name:         "run started, job fails, requeue succeeds",
			semaphores:   both,
			keep:         true,
			runErr:       jobFailed,
			wantRequeued: []constraintapi.Semaphore{auto},
		},
		{
			name:         "run started, job fails, requeue fails",
			semaphores:   both,
			keep:         true,
			runErr:       jobFailed,
			requeueErr:   requeueFailed,
			wantErr:      requeueFailed,
			wantForce:    true,
			wantRequeued: []constraintapi.Semaphore{auto},
		},
		{
			name:         "run started, job fails, item gone on requeue",
			semaphores:   both,
			keep:         true,
			runErr:       jobFailed,
			requeueErr:   ErrQueueItemNotFound,
			wantRequeued: []constraintapi.Semaphore{auto},
		},
		{
			name:        "run started, job fails on the last attempt",
			semaphores:  both,
			maxAttempts: &one,
			keep:        true,
			runErr:      jobFailed,
		},
		{
			name:       "run started, job succeeds",
			semaphores: both,
			keep:       true,
		},
		{
			name:         "no manual semaphore, job fails",
			semaphores:   []constraintapi.Semaphore{auto},
			runErr:       jobFailed,
			wantRequeued: []constraintapi.Semaphore{auto},
		},
		{
			name:       "no manual semaphore, job succeeds",
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

			releases := make(chan *constraintapi.CapacityReleaseRequest, 2)
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
					Kind:        KindStart,
					MaxAttempts: tc.maxAttempts,
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
			}, func(_ context.Context, info RunInfo, _ Item) (RunResult, error) {
				if tc.keep {
					info.CapacityLease.KeepManualSemaphores()
				}
				return RunResult{}, tc.runErr
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

			if tc.wantRequeued == nil {
				require.Empty(t, shard.requeued)
				return
			}
			require.Len(t, shard.requeued, 1)
			require.Equal(t, tc.wantRequeued, shard.requeued[0].Data.Semaphores)
		})
	}
}

// the executor releases the capacity lease early once a step is saved.  for
// an item with a manual semaphore that release is a no-op.  an early release
// deletes the lease and keeps the slot, and an error after it then leaves
// nothing for the exit to give back.
func TestProcessItemSkipsEarlyReleaseWithManualSemaphore(t *testing.T) {
	ctx := context.Background()
	shard := &requeueErrorShard{
		mockShardForIterator: &mockShardForIterator{name: "test"},
	}
	registry, err := NewSingleShardRegistry(shard)
	require.NoError(t, err)

	releases := make(chan *constraintapi.CapacityReleaseRequest, 2)
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
			Semaphores: []constraintapi.Semaphore{{
				ID:      constraintapi.SemaphoreIDFn(uuid.New()),
				Weight:  1,
				Release: constraintapi.SemaphoreReleaseManual,
			}},
		},
	}

	releasedEarly := make(chan bool, 1)
	_, err = q.ProcessItem(ctx, ProcessItem{
		I: item,
		CapacityLease: &CapacityLease{
			LeaseID:    capacityLeaseID,
			IssuedAtMS: time.Now().UnixMilli(),
		},
	}, func(_ context.Context, info RunInfo, _ Item) (RunResult, error) {
		require.NoError(t, info.CapacityLease.Release())
		select {
		case <-releases:
			releasedEarly <- true
		case <-time.After(100 * time.Millisecond):
			releasedEarly <- false
		}
		return RunResult{}, errors.New("error enqueueing discovery step")
	})
	require.NoError(t, err)
	require.False(t, <-releasedEarly, "the lease must stay until the job exits")

	select {
	case req := <-releases:
		require.Equal(t, capacityLeaseID, req.LeaseID)
		require.True(t, req.ForceReleaseSemaphores, "the exit must give the slot back")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for the capacity lease release")
	}

	select {
	case <-releases:
		t.Fatal("the capacity lease was released twice")
	case <-time.After(100 * time.Millisecond):
	}
}

type requeueErrorShard struct {
	*mockShardForIterator
	requeueErr error
	requeued   []QueueItem
}

func (s *requeueErrorShard) Requeue(_ context.Context, item QueueItem, _ time.Time, _ ...RequeueOptionFn) error {
	s.requeued = append(s.requeued, item)
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
