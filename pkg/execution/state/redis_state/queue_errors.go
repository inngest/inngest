package redis_state

import "fmt"

var (
	ErrQueueItemThrottled = fmt.Errorf("queue item throttled")
)

func newKeyError(err error, key string) error {
	return keyError{
		cause: err,
		key:   key,
	}
}

// keyError is an error string which represents the custom key used when returning a
// concurrency or throttled error.  The ErrQueueItemThrottled error must wrap this keyError
// to embed the key directly in the top-level error class.
type keyError struct {
	key   string
	cause error
}

func (k keyError) Cause() error {
	return k.cause
}

func (k keyError) Error() string {
	return k.cause.Error()
}

func (k keyError) Unwrap() error {
	return k.cause
}
