package queue

import (
	"time"
)

const (
	PartitionSelectionMax = int64(100)
	PartitionPeekMax      = PartitionSelectionMax * 3
	AccountPeekMax        = int64(30)

	// PartitionLeaseDuration dictates how long a worker holds the lease for
	// a partition.  This gives the worker a right to scan all queue items
	// for that partition to schedule the execution of jobs.
	//
	// Right now, this must be short enough to reduce contention but long enough
	// to account for the latency of peeking QueuePeekMax jobs from Redis.
	PartitionLeaseDuration = 4 * time.Second
	// PartitionRequeueExtension is the length of time that we extend a partition's
	// vesting time when requeueing by default.
	PartitionRequeueExtension = 30 * time.Second

	// PartitionConcurrencyLimitRequeueExtension is the length of time that a partition
	// is requeued if there is no global or partition(function) capacity because of
	// concurrency limits.
	//
	// This is short, as there are still functions that are due to run (ie vesting < now),
	// but long enough to reduce thrash.
	//
	// This means that jobs not started because of concurrency limits incur up to this amount
	// of additional latency.
	//
	// NOTE: This must be greater than PartitionLookahead
	// NOTE: This is the maximum latency introduced into concurrnecy limited partitions in the
	//       worst case.
	PartitionConcurrencyLimitRequeueExtension = 5 * time.Second
	PartitionThrottleLimitRequeueExtension    = 1 * time.Second
	PartitionPausedRequeueExtension           = 5 * time.Minute
	PartitionLookahead                        = time.Second

	ShadowPartitionLeaseDuration  = 4 * time.Second // same as PartitionLeaseDuration
	BacklogNormalizeLeaseDuration = 4 * time.Second // same as PartitionLeaseDuration

	ShadowPartitionRefillCapacityReachedRequeueExtension = 1 * time.Second
	ShadowPartitionRefillPausedRequeueExtension          = 5 * time.Minute
	BacklogDefaultRequeueExtension                       = 2 * time.Second

	// default values
	DefaultQueuePeekMin  int64 = 300
	DefaultQueuePeekMax  int64 = 750
	AbsoluteQueuePeekMax int64 = 5000

	QueuePeekCurrMultiplier int64 = 4 // threshold 25%
	QueuePeekEWMALen        int   = 10
	QueueLeaseDuration            = 30 * time.Second
	ConfigLeaseDuration           = 10 * time.Second
	ConfigLeaseMax                = 20 * time.Second

	PriorityMax     uint = 0
	PriorityDefault uint = 5
	PriorityMin     uint = 9

	// FunctionStartScoreBufferTime is the grace period used to compare function start
	// times to edge enqueue times.
	FunctionStartScoreBufferTime = 10 * time.Second
)

const (
	AbsoluteShadowPartitionPeekMax int64 = 10 * ShadowPartitionPeekMaxBacklogs

	ShadowPartitionAccountPeekMax  = int64(30)
	ShadowPartitionPeekMax         = int64(300) // same as PartitionPeekMax for now
	ShadowPartitionPeekMinBacklogs = int64(10)
	ShadowPartitionPeekMaxBacklogs = int64(100)

	ShadowPartitionRequeueExtendedDuration = 3 * time.Second

	ShadowPartitionLookahead = 2 * PartitionLookahead

	BacklogForceRequeueMaxBackoff = 5 * time.Minute
)
