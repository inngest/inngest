package queue

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"syscall"
)

var ErrQueueItemThrottled = fmt.Errorf("queue item throttled")

// ErrDebounceNotFound is returned by DebounceOperations when the requested
// debounce item or pointer is missing on the shard.
var ErrDebounceNotFound = fmt.Errorf("debounce not found")

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
	ErrRoleAlreadyLeased             = fmt.Errorf("role already leased")
	ErrRoleLeaseExceedsLimits        = fmt.Errorf("role lease duration exceeds the maximum of %d seconds", int(RoleLeaseMax.Seconds()))

	ErrAllShardsAlreadyLeased  = fmt.Errorf("all shards in the group are fully allocated")
	ErrShardLeaseNotFound      = fmt.Errorf("shard lease not found")
	ErrShardLeaseExpired       = fmt.Errorf("cannot renew expired shard lease")
	ErrShardLeaseExceedsLimits = fmt.Errorf("shard lease duration exceeds the maximum of %d seconds", int(ShardLeaseMax.Seconds()))

	ErrPartitionConcurrencyLimit = fmt.Errorf("at partition concurrency limit")
	ErrAccountConcurrencyLimit   = fmt.Errorf("at account concurrency limit")

	// ErrSystemConcurrencyLimit represents a concurrency limit for system partitions
	ErrSystemConcurrencyLimit = fmt.Errorf("at system concurrency limit")

	// ErrConcurrencyLimitCustomKey represents a concurrency limit being hit for *some*, but *not all*
	// jobs in a queue, via custom concurrency keys which are evaluated to a specific string.
	ErrConcurrencyLimitCustomKey = fmt.Errorf("at concurrency limit")

	// ErrSemaphoreLimit represents a semaphore capacity limit for a specific queue item.
	// Unlike partition/account limits, this is per-item (only start jobs carry semaphores),
	// so the iterator should skip the item and continue scanning.
	ErrSemaphoreLimit = fmt.Errorf("at semaphore capacity limit")
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

func ShardLeaseRenewalRetryableError(err error) bool {
	switch {
	case errors.Is(err, ErrShardLeaseNotFound):
		return false
	case errors.Is(err, ErrShardLeaseExpired):
		return false
	}
	return true
}

// IsTransientDBError returns true if the error is a transient database connection
// error that may resolve on retry (e.g. connection reset, refused, EOF).
// This is used to allow the queue scanner and role lease claims to recover from
// brief database unavailability instead of permanently shutting down.
func IsTransientDBError(err error) bool {
	if err == nil {
		return false
	}

	// Network-level errors
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	if errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.EPIPE) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	// String-based detection for wrapped errors that lose type info
	msg := err.Error()
	transientPatterns := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"no such host",
		"i/o timeout",
		"connection timed out",
		"server closed the connection unexpectedly",
		"unexpected EOF",
		"driver: bad connection",
		"sql: database is closed",
	}
	for _, pattern := range transientPatterns {
		if strings.Contains(strings.ToLower(msg), pattern) {
			return true
		}
	}

	return false
}
