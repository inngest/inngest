package redis_state

import "github.com/inngest/inngest/pkg/enums"

func (q *queue) isPermittedQueueKind() bool {
	switch q.primaryQueueShard.Kind {
	case string(enums.QueueShardKindRedis):
		return true

	default:
		return false
	}
}
