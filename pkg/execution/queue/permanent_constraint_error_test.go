package queue

import (
	"bytes"
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestDropPermanentlyUnroutableItem(t *testing.T) {
	cause := errors.New("constraint shard missing")
	handlerErr := errors.New("state cleanup failed")
	dequeueErr := errors.New("dequeue failed")

	tests := []struct {
		name             string
		handler          PermanentConstraintErrorHandler
		dequeueErr       error
		wantErr          string
		wantOutcome      string
		wantHandlerCalls int32
		wantDequeueCalls int32
	}{
		{
			name: "cleans up state before dequeueing",
			handler: func(context.Context, QueueItem, error) error {
				return nil
			},
			wantHandlerCalls: 1,
			wantDequeueCalls: 1,
			wantOutcome:      "dropped",
		},
		{
			name:        "requires a handler",
			wantErr:     "handler is not configured",
			wantOutcome: "ignored",
		},
		{
			name: "keeps item when cleanup fails",
			handler: func(context.Context, QueueItem, error) error {
				return handlerErr
			},
			wantErr:          handlerErr.Error(),
			wantHandlerCalls: 1,
			wantOutcome:      "ignored",
		},
		{
			name: "returns dequeue failure for retry",
			handler: func(context.Context, QueueItem, error) error {
				return nil
			},
			dequeueErr:       dequeueErr,
			wantErr:          dequeueErr.Error(),
			wantHandlerCalls: 1,
			wantDequeueCalls: 1,
			wantOutcome:      "ignored",
		},
		{
			name: "missing item is already dropped",
			handler: func(context.Context, QueueItem, error) error {
				return nil
			},
			dequeueErr:       ErrQueueItemNotFound,
			wantHandlerCalls: 1,
			wantDequeueCalls: 1,
			wantOutcome:      "dropped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LOG_HANDLER", "json")
			var logs bytes.Buffer
			ctx := logger.WithStdlib(context.Background(), logger.From(context.Background(), logger.WithLoggerWriter(&logs)))
			shard := &mockShardForIterator{name: "test", dequeueErr: tt.dequeueErr}
			registry, err := NewSingleShardRegistry(shard)
			require.NoError(t, err)

			var handlerCalls int32
			var handler PermanentConstraintErrorHandler
			if tt.handler != nil {
				handler = func(ctx context.Context, item QueueItem, err error) error {
					atomic.AddInt32(&handlerCalls, 1)
					require.ErrorIs(t, err, cause)
					return tt.handler(ctx, item, err)
				}
			}

			q, err := New(ctx, "test", registry, WithPermanentConstraintErrorHandler(handler))
			require.NoError(t, err)

			item := QueueItem{
				ID:         "item",
				FunctionID: uuid.New(),
				Data: Item{
					Identifier: state.Identifier{
						AccountID:   uuid.New(),
						WorkspaceID: uuid.New(),
						WorkflowID:  uuid.New(),
						RunID:       ulid.Make(),
					},
				},
			}
			err = q.dropPermanentlyUnroutableItem(ctx, item, cause)
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
			require.Equal(t, tt.wantHandlerCalls, atomic.LoadInt32(&handlerCalls))
			require.Equal(t, tt.wantDequeueCalls, atomic.LoadInt32(&shard.dequeueCalls))
			if tt.wantOutcome == "" {
				require.NotContains(t, logs.String(), `"outcome":`)
			} else {
				require.Contains(t, logs.String(), `"outcome":"`+tt.wantOutcome+`"`)
			}
		})
	}
}

func TestLeaseItemPermanentConstraintError(t *testing.T) {
	permanentErr := constraintapi.ErrConstraintShardNotFound
	transientErr := errors.New("constraint api unavailable")
	cleanupErr := errors.New("state cleanup failed")

	tests := []struct {
		name             string
		constraintErr    error
		configureHandler bool
		handlerErr       error
		dequeueErr       error
		wantStatus       LeaseItemStatus
		wantErr          error
		wantHandlerCalls int32
		wantDequeueCalls int32
		wantOutcome      string
	}{
		{
			name:             "permanent error is cleaned up and dropped",
			constraintErr:    permanentErr,
			configureHandler: true,
			wantStatus:       LeaseItemStatusDropped,
			wantHandlerCalls: 1,
			wantDequeueCalls: 1,
			wantOutcome:      "dropped",
		},
		{
			name:          "permanent error retries without handler",
			constraintErr: permanentErr,
			wantStatus:    LeaseItemStatusNone,
			wantErr:       ErrProcessStopIterator,
			wantOutcome:   "ignored",
		},
		{
			name:             "cleanup failure retains item for retry",
			constraintErr:    permanentErr,
			configureHandler: true,
			handlerErr:       cleanupErr,
			wantStatus:       LeaseItemStatusNone,
			wantErr:          ErrProcessStopIterator,
			wantHandlerCalls: 1,
			wantOutcome:      "ignored",
		},
		{
			name:             "dequeue failure retains item for retry",
			constraintErr:    permanentErr,
			configureHandler: true,
			dequeueErr:       errors.New("queue unavailable"),
			wantStatus:       LeaseItemStatusNone,
			wantErr:          ErrProcessStopIterator,
			wantHandlerCalls: 1,
			wantDequeueCalls: 1,
			wantOutcome:      "ignored",
		},
		{
			name:             "transient error retains item for retry",
			constraintErr:    transientErr,
			configureHandler: true,
			wantStatus:       LeaseItemStatusNone,
			wantErr:          ErrProcessStopIterator,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LOG_HANDLER", "json")
			var logs bytes.Buffer
			ctx := logger.WithStdlib(context.Background(), logger.From(context.Background(), logger.WithLoggerWriter(&logs)))
			shard := &mockShardForIterator{name: "test", dequeueErr: tt.dequeueErr}
			registry, err := NewSingleShardRegistry(shard)
			require.NoError(t, err)

			var handlerCalls int32
			opts := []QueueOpt{
				WithCapacityManager(permanentConstraintErrorCapacityManager{cause: tt.constraintErr}),
				WithPartitionConstraintConfigGetter(func(context.Context, PartitionIdentifier) PartitionConstraintConfig {
					return PartitionConstraintConfig{
						Concurrency: PartitionConcurrency{AccountConcurrency: 1},
					}
				}),
			}
			if tt.configureHandler {
				opts = append(opts, WithPermanentConstraintErrorHandler(func(context.Context, QueueItem, error) error {
					atomic.AddInt32(&handlerCalls, 1)
					return tt.handlerErr
				}))
			}

			q, err := New(ctx, "test", registry, opts...)
			require.NoError(t, err)

			accountID := uuid.New()
			envID := uuid.New()
			functionID := uuid.New()
			item := &QueueItem{
				ID:          "item",
				FunctionID:  functionID,
				WorkspaceID: envID,
				Data: Item{
					Identifier: state.Identifier{
						AccountID:   accountID,
						WorkspaceID: envID,
						WorkflowID:  functionID,
						RunID:       ulid.Make(),
					},
				},
			}

			result, err := q.LeaseItem(ctx, LeaseItemRequest{Item: item}, func(context.Context, ProcessItem) (DispatchedItem, error) {
				t.Fatal("dispatch must not be called when the constraint check fails")
				return nil, nil
			})
			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tt.wantErr)
			}
			require.Equal(t, tt.wantStatus, result.Status)
			require.Equal(t, tt.wantHandlerCalls, atomic.LoadInt32(&handlerCalls))
			require.Equal(t, tt.wantDequeueCalls, atomic.LoadInt32(&shard.dequeueCalls))
			if tt.wantOutcome == "" {
				require.NotContains(t, logs.String(), `"outcome":`)
			} else {
				require.Contains(t, logs.String(), `"outcome":"`+tt.wantOutcome+`"`)
			}
		})
	}
}

type permanentConstraintErrorCapacityManager struct {
	cause error
}

func (m permanentConstraintErrorCapacityManager) Check(context.Context, *constraintapi.CapacityCheckRequest) (*constraintapi.CapacityCheckResponse, errs.UserError, errs.InternalError) {
	return nil, nil, nil
}

func (m permanentConstraintErrorCapacityManager) Acquire(context.Context, *constraintapi.CapacityAcquireRequest) (*constraintapi.CapacityAcquireResponse, errs.InternalError) {
	return nil, wrappedConstraintAPIInternalError{err: m.cause}
}

func (m permanentConstraintErrorCapacityManager) ExtendLease(context.Context, *constraintapi.CapacityExtendLeaseRequest) (*constraintapi.CapacityExtendLeaseResponse, errs.InternalError) {
	return nil, nil
}

func (m permanentConstraintErrorCapacityManager) Release(context.Context, *constraintapi.CapacityReleaseRequest) (*constraintapi.CapacityReleaseResponse, errs.InternalError) {
	return nil, nil
}
