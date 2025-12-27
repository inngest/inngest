package queue

import "time"

type QueueIterOpt func(o *QueueIterOptions)

type QueueIterOptions struct {
	BatchSize                 int64
	Interval                  time.Duration
	IterateBacklogs           bool
	EnableMillisecondIncrease bool
}

func WithQueueItemIterBatchSize(size int64) QueueIterOpt {
	return func(o *QueueIterOptions) {
		if size > 0 {
			o.BatchSize = size
		}
	}
}

func WithQueueItemIterInterval(itv time.Duration) QueueIterOpt {
	return func(o *QueueIterOptions) {
		o.Interval = itv
	}
}

func WithQueueItemIterEnableBacklog(iterateBacklogs bool) QueueIterOpt {
	return func(o *QueueIterOptions) {
		o.IterateBacklogs = iterateBacklogs
	}
}

func WithQueueItemIterDisableMillisecondIncrease() QueueIterOpt {
	return func(o *QueueIterOptions) {
		o.EnableMillisecondIncrease = false
	}
}
