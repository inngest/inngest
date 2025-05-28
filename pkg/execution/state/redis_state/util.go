package redis_state

import "github.com/inngest/inngest/pkg/enums"

func (q *queue) isQueueShardKindAllowed(kind string) bool {
	switch kind {
	case string(enums.QueueShardKindRedis):
		return true
	default:
		return false
	}
}
