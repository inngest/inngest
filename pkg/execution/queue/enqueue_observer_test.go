package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/stretchr/testify/require"
)

type enqueueResultShard struct {
	*mockShardForIterator
	result QueueItem
	err    error
}

func (s enqueueResultShard) EnqueueItem(context.Context, QueueItem, time.Time, EnqueueOpts) (QueueItem, error) {
	return s.result, s.err
}

func TestEnqueueObserverReceivesBackendResult(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "success"}, {name: "duplicate", err: ErrQueueItemExists}, {name: "failure", err: errors.New("enqueue failed")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stored := QueueItem{ID: "final-backend-id", GenerationID: 7, EnqueuedAt: 1234}
			shard := enqueueResultShard{mockShardForIterator: &mockShardForIterator{name: "actual-shard"}, result: stored, err: tc.err}
			registry, err := NewSingleShardRegistry(shard)
			require.NoError(t, err)
			producer := NewProducer(registry)
			calls := 0
			err = producer.Enqueue(t.Context(), Item{Identifier: state.Identifier{WorkflowID: uuid.New(), AccountID: uuid.New()}}, time.Now(), EnqueueOpts{OnEnqueued: func(item QueueItem, name string) {
				calls++
				require.Equal(t, stored, item)
				require.Equal(t, "actual-shard", name)
			}})
			require.ErrorIs(t, err, tc.err)
			if tc.err == nil {
				require.Equal(t, 1, calls)
			} else {
				require.Zero(t, calls)
			}
		})
	}
}
