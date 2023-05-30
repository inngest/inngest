package consts

import "time"

const (
	// DefaultRetryCount is used when no retry count for a step is specified.
	DefaultRetryCount = 3

	// MaxFunctionTimeout represents the longest running function or step allowed within
	// our system.
	MaxFunctionTimeout = 2 * time.Hour

	// MaxBodySize is the maximum payload size read on any HTTP response.
	MaxBodySize = 1024 * 1024 * 4

	// MaxRetries represents the maximum number of retries for a particular function or step
	// possible.
	MaxRetries = 30

	// MaxRetryDuration is the furthest a retry can be scheduled.  If retries are scheduled further
	// than now plus this duration, the retry duration will automatically be lowered to this value.
	MaxRetryDuration = time.Hour * 24

	// MinRetryDuration is the soonest a retry can be scheduled.
	MinRetryDuration = time.Second * 1

	// FunctionIdempotencyPeriod determines how long a specific function remains idempotent
	// when using idempotency keys.
	FunctionIdempotencyPeriod = 24 * time.Hour
)
