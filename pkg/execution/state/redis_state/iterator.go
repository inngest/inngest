package redis_state

import "time"

type QueueIterOpt func(o *queueIterOpt)

// queueIterOpt provides options to be used for queue iteration operations
type queueIterOpt struct {
	// TODO figure how to embed queueOpOpt into this struct and share the setting overrides
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
