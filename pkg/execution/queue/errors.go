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
