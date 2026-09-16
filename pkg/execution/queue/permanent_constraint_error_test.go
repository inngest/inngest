package queue

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
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
		},
		{
			name:    "requires a handler",
			wantErr: "handler is not configured",
		},
		{
			name: "keeps item when cleanup fails",
			handler: func(context.Context, QueueItem, error) error {
				return handlerErr
			},
			wantErr:          handlerErr.Error(),
			wantHandlerCalls: 1,
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
		},
		{
			name: "missing item is already dropped",
			handler: func(context.Context, QueueItem, error) error {
				return nil
			},
			dequeueErr:       ErrQueueItemNotFound,
			wantHandlerCalls: 1,
			wantDequeueCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			q, err := New(context.Background(), "test", registry, WithPermanentConstraintErrorHandler(handler))
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
			err = q.dropPermanentlyUnroutableItem(context.Background(), item, cause)
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
			require.Equal(t, tt.wantHandlerCalls, atomic.LoadInt32(&handlerCalls))
			require.Equal(t, tt.wantDequeueCalls, atomic.LoadInt32(&shard.dequeueCalls))
		})
	}
}
