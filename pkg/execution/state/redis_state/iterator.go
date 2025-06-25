package redis_state

import "time"

type QueueIterOpt func(o *queueIterOpt)

type queueIterOpt struct {
	batchSize int64
	interval  time.Duration
}

func WithQueueItemIterBatchSize(size int64) QueueIterOpt {
	return func(o *queueIterOpt) {
		if size > 0 {
			o.batchSize = size
		}
	}
}

func WithQueueItemIterInterval(itv time.Duration) QueueIterOpt {
	return func(o *queueIterOpt) {
		o.interval = itv
	}
}
