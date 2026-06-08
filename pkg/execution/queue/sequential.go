package queue

func NewSequentialRole(opts ...QueueRoleOpt) QueueRole {
	return newQueueRole(QueueRoleSequential, RoleLeaseDuration, RoleLeaseDuration/3, nil, nil, opts...)
}

func includeSequentialRole(o *QueueOptions) bool {
	// Allowlisted workers are intentionally scoped to a subset of queue
	// partitions.  Letting one hold the global sequential lease would make all
	// workers process globally in FIFO mode even though the allowlisted worker
	// cannot scan every queue.
	return o.runMode.Sequential && len(o.AllowQueues) == 0
}

func (q *queueProcessor) isSequential() bool {
	return q.isRoleActive(QueueRoleSequential)
}
