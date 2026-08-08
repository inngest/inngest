package queue

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQueueProducerSelectShardPreservesCanonicalErrors(t *testing.T) {
	resolveErr := errors.New("resolve shard")

	tests := []struct {
		name      string
		shardName string
		selector  shardSelector
		expected  error
	}{
		{
			name:      "forced shard not found",
			shardName: "missing",
			selector:  alwaysShard(nil),
			expected:  ErrQueueShardNotFound,
		},
		{
			name: "selector error",
			selector: func(context.Context, Scope, *string) (QueueShard, error) {
				return nil, resolveErr
			},
			expected: resolveErr,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry := mustShardRegistry(t,
				map[string]QueueShard{"available": newTestShard("available", "")},
				WithShardSelector(test.selector),
			)
			producer := &queueProducer{shards: registry}

			_, err := producer.selectShard(context.Background(), test.shardName, QueueItem{})

			require.ErrorIs(t, err, test.expected)
		})
	}
}
