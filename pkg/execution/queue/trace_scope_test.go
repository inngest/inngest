package queue

import (
	"testing"

	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/stretchr/testify/require"
)

func TestTraceScopeFromQueueItemSystemScopeIncludesItemKind(t *testing.T) {
	queueName := "system-queue"
	queueShardName := "queue-shard"
	itemKind := "system-kind"

	scope := TraceScopeFromQueueItem(QueueItem{
		QueueName: &queueName,
		Data: Item{
			Kind: itemKind,
		},
	}, queueShardName)

	systemScope, ok := scope.(itrace.SystemScope)
	require.True(t, ok)
	require.Equal(t, &queueName, systemScope.QueueName)
	require.Equal(t, queueShardName, systemScope.QueueShardName)
	require.Equal(t, itemKind, systemScope.ItemKind)
}

func TestTraceScopeFromQueueItemSystemScopeIncludesItemKindFromDataQueueName(t *testing.T) {
	queueName := "system-queue"
	queueShardName := "queue-shard"
	itemKind := "system-kind"

	scope := TraceScopeFromQueueItem(QueueItem{
		Data: Item{
			Kind:      itemKind,
			QueueName: &queueName,
		},
	}, queueShardName)

	systemScope, ok := scope.(itrace.SystemScope)
	require.True(t, ok)
	require.Equal(t, &queueName, systemScope.QueueName)
	require.Equal(t, queueShardName, systemScope.QueueShardName)
	require.Equal(t, itemKind, systemScope.ItemKind)
}
