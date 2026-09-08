package queue

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/stretchr/testify/require"
)

type requeueRoutingShard struct {
	QueueShard
	name             string
	requeueErr       error
	requeueByIDErr   error
	requeueCalls     int
	requeueByIDCalls int
}

func (s *requeueRoutingShard) Name() string { return s.name }
func (s *requeueRoutingShard) Requeue(context.Context, QueueItem, time.Time, ...RequeueOptionFn) error {
	s.requeueCalls++
	return s.requeueErr
}
func (s *requeueRoutingShard) RequeueByJobID(context.Context, string, time.Time) error {
	s.requeueByIDCalls++
	return s.requeueByIDErr
}

func TestProducerRequeueByJobIDRejectsInvalidScope(t *testing.T) {
	ctx := context.Background()
	shard := &mockShardForIterator{name: "shard-a"}
	registry, err := NewSingleShardRegistry(shard)
	require.NoError(t, err)

	producer, err := New(ctx, "test", registry)
	require.NoError(t, err)

	tests := []struct {
		name    string
		scope   Scope
		wantErr string
	}{
		{
			name:    "missing account ID",
			scope:   Scope{EnvID: uuid.New(), FunctionID: uuid.New()},
			wantErr: "missing account ID",
		},
		{
			name:    "missing env ID",
			scope:   Scope{AccountID: uuid.New(), FunctionID: uuid.New()},
			wantErr: "missing env ID",
		},
		{
			name:    "missing function ID",
			scope:   Scope{AccountID: uuid.New(), EnvID: uuid.New()},
			wantErr: "missing function ID",
		},
		{
			name:    "missing all IDs",
			scope:   Scope{},
			wantErr: "missing account ID",
		},
	}

	for _, isSystem := range []bool{true, false} {
		for _, tt := range tests {
			t.Run(fmt.Sprintf("%s/system=%t", tt.name, isSystem), func(t *testing.T) {
				tt.scope.IsSystem = isSystem
				err := producer.RequeueByJobID(ctx, tt.scope, shard.Name(), "job-id", time.Now())
				require.EqualError(t, err, tt.wantErr)
			})
		}
	}
}

func TestProducerRequeueUsesCurrentShardWithSourceFallback(t *testing.T) {
	functionID := uuid.New()
	item := QueueItem{
		ID:         "job-1",
		FunctionID: functionID,
		Data: Item{Identifier: state.Identifier{
			WorkflowID:  functionID,
			WorkspaceID: uuid.New(),
			AccountID:   uuid.New(),
		}},
	}

	tests := []struct {
		name            string
		destinationErr  error
		wantSourceCalls int
		wantDestCalls   int
	}{
		{name: "destination owns migrated item", wantDestCalls: 1},
		{name: "item not copied yet falls back to source", destinationErr: ErrQueueItemNotFound, wantSourceCalls: 1, wantDestCalls: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &requeueRoutingShard{name: "source"}
			destination := &requeueRoutingShard{name: "destination", requeueErr: tt.destinationErr}
			registry := mustShardRegistry(t,
				map[string]QueueShard{"source": source, "destination": destination},
				WithShardSelector(func(context.Context, Scope, *string) (QueueShard, error) {
					return destination, nil
				}),
			)
			producer := &queueProducer{shards: registry}

			require.NoError(t, producer.Requeue(context.Background(), "source", item, time.Now()))
			require.Equal(t, tt.wantSourceCalls, source.requeueCalls)
			require.Equal(t, tt.wantDestCalls, destination.requeueCalls)
		})
	}
}

func TestProducerRequeueByJobIDUsesCurrentShardWithSourceFallback(t *testing.T) {
	tests := []struct {
		name            string
		destinationErr  error
		wantSourceCalls int
		wantDestCalls   int
	}{
		{name: "destination owns migrated item", wantDestCalls: 1},
		{name: "item not copied yet falls back to source", destinationErr: ErrQueueItemNotFound, wantSourceCalls: 1, wantDestCalls: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &requeueRoutingShard{name: "source"}
			destination := &requeueRoutingShard{name: "destination", requeueByIDErr: tt.destinationErr}
			registry := mustShardRegistry(t,
				map[string]QueueShard{"source": source, "destination": destination},
				WithShardSelector(func(context.Context, Scope, *string) (QueueShard, error) {
					return destination, nil
				}),
			)
			producer := &queueProducer{shards: registry}
			scope := Scope{AccountID: uuid.New(), EnvID: uuid.New(), FunctionID: uuid.New()}

			require.NoError(t, producer.RequeueByJobID(context.Background(), scope, "source", "job-1", time.Now()))
			require.Equal(t, tt.wantSourceCalls, source.requeueByIDCalls)
			require.Equal(t, tt.wantDestCalls, destination.requeueByIDCalls)
		})
	}
}
