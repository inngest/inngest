package redis_state

type QueueIteratorOpt func(o *queueIterOpt)

type queueIterOpt struct {
	allowKeyQueues func() bool
}

func WithQueueItemIteratorAllowKeyQueues(kq func() bool) QueueIteratorOpt {
	return func(o *queueIterOpt) {
		o.allowKeyQueues = kq
	}
}
