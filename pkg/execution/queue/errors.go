package queue

import "fmt"

var ErrQueueItemThrottled = fmt.Errorf("queue item throttled")

func NewKeyError(err error, key string) error {
	return KeyError{
		cause: err,
		key:   key,
	}
}

// KeyError is an error string which represents the custom key used when returning a
// concurrency or throttled error.  The ErrQueueItemThrottled error must wrap this KeyError
// to embed the key directly in the top-level error class.
type KeyError struct {
	key   string
	cause error
}

func (k KeyError) Cause() error {
	return k.cause
}

func (k KeyError) Error() string {
	return k.cause.Error()
}

func (k KeyError) Unwrap() error {
	return k.cause
}

var (
	ErrQueueItemExists               = fmt.Errorf("queue item already exists")
	ErrQueueItemNotFound             = fmt.Errorf("queue item not found")
	ErrQueueItemAlreadyLeased        = fmt.Errorf("queue item already leased")
	ErrQueueItemLeaseMismatch        = fmt.Errorf("item lease does not match")
	ErrQueueItemNotLeased            = fmt.Errorf("queue item is not leased")
	ErrQueuePeekMaxExceedsLimits     = fmt.Errorf("peek exceeded the maximum limit of %d", AbsoluteQueuePeekMax)
	ErrQueueItemSingletonExists      = fmt.Errorf("singleton item already exists")
	ErrPriorityTooLow                = fmt.Errorf("priority is too low")
	ErrPriorityTooHigh               = fmt.Errorf("priority is too high")
	ErrPartitionNotFound             = fmt.Errorf("partition not found")
	ErrPartitionAlreadyLeased        = fmt.Errorf("partition already leased")
	ErrPartitionPeekMaxExceedsLimits = fmt.Errorf("peek exceeded the maximum limit of %d", PartitionPeekMax)
	ErrAccountPeekMaxExceedsLimits   = fmt.Errorf("account peek exceeded the maximum limit of %d", AccountPeekMax)
	ErrPartitionGarbageCollected     = fmt.Errorf("partition garbage collected")
	ErrPartitionPaused               = fmt.Errorf("partition is paused")
	ErrConfigAlreadyLeased           = fmt.Errorf("config scanner already leased")
	ErrConfigLeaseExceedsLimits      = fmt.Errorf("config lease duration exceeds the maximum of %d seconds", int(ConfigLeaseMax.Seconds()))

	ErrAllShardsAlreadyLeased  = fmt.Errorf("all shards in the group are fully allocated")
	ErrShardLeaseNotFound      = fmt.Errorf("all shards in the group are fully allocated")
	ErrShardLeaseExpired       = fmt.Errorf("all shards in the group are fully allocated")
	ErrShardLeaseExceedsLimits = fmt.Errorf("shard lease duration exceeds the maximum of %d seconds", int(ShardLeaseMax.Seconds()))

	ErrPartitionConcurrencyLimit = fmt.Errorf("at partition concurrency limit")
	ErrAccountConcurrencyLimit   = fmt.Errorf("at account concurrency limit")

	// ErrSystemConcurrencyLimit represents a concurrency limit for system partitions
	ErrSystemConcurrencyLimit = fmt.Errorf("at system concurrency limit")

	// ErrConcurrencyLimitCustomKey represents a concurrency limit being hit for *some*, but *not all*
	// jobs in a queue, via custom concurrency keys which are evaluated to a specific string.
	ErrConcurrencyLimitCustomKey = fmt.Errorf("at concurrency limit")
)

var (
	ErrShadowPartitionAlreadyLeased               = fmt.Errorf("shadow partition already leased")
	ErrShadowPartitionLeaseNotFound               = fmt.Errorf("shadow partition lease not found")
	ErrShadowPartitionNotFound                    = fmt.Errorf("shadow partition not found")
	ErrShadowPartitionPaused                      = fmt.Errorf("shadow partition refill is disabled")
	ErrShadowPartitionBacklogPeekMaxExceedsLimits = fmt.Errorf("shadow partition backlog peek exceeded the maximum limit of %d", ShadowPartitionPeekMaxBacklogs)
	ErrShadowPartitionPeekMaxExceedsLimits        = fmt.Errorf("shadow partition peek exceeded the maximum limit of %d", ShadowPartitionPeekMax)
	ErrShadowPartitionAccountPeekMaxExceedsLimits = fmt.Errorf("account peek with shadow partitions exceeded the maximum limit of %d", ShadowPartitionAccountPeekMax)
)

var (
	ErrProcessNoCapacity   = fmt.Errorf("no capacity")
	ErrProcessStopIterator = fmt.Errorf("stop iterator")
)

var (
	ErrBacklogNormalizationLeaseExpired     = fmt.Errorf("backlog normalization lease expired")
	ErrBacklogAlreadyLeasedForNormalization = fmt.Errorf("backlog already leased for normalization")
)

var ErrQueueShardNotFound = fmt.Errorf("could not find queue shard for the specified name")
