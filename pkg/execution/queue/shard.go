package queue

import "github.com/inngest/inngest/pkg/enums"

type QueueShard interface {
	Name() string
	Kind() enums.QueueShardKind
}
