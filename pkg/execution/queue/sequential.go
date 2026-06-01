package queue

func NewSequentialRole(opts ...QueueRoleOpt) QueueRole {
	return newQueueRole(QueueRoleSequential, RoleLeaseDuration, RoleLeaseDuration/3, nil, opts...)
}
