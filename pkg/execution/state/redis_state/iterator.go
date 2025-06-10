package redis_state

type QueueIterOpt func(o *queueIterOpt)

type queueIterOpt struct {
	batchSize      int64
	allowKeyQueues func() bool
}

func WithQueueItemIterAllowKeyQueues(kq func() bool) QueueIterOpt {
	return func(o *queueIterOpt) {
		o.allowKeyQueues = kq
	}
}

func WithQueueItemIterBatchSize(size int64) QueueIterOpt {
	return func(o *queueIterOpt) {
		if size > 0 {
			o.batchSize = size
		}
	}
}
