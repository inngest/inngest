package redis_state

type QueueIteratorOpt func(o *queueIterOpt)

type queueIterOpt struct {
	batchSize      int64
	allowKeyQueues func() bool
}

func WithQueueItemIteratorAllowKeyQueues(kq func() bool) QueueIteratorOpt {
	return func(o *queueIterOpt) {
		o.allowKeyQueues = kq
	}
}

func WithQueueItemIterBatchSize(size int64) QueueIteratorOpt {
	return func(o *queueIterOpt) {
		if size > 0 {
			o.batchSize = size
		}
	}
}
