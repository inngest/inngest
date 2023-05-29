package consts

import "time"

const (
	// DefaultRetryCount is used when no retry count for a step is specified.
	DefaultRetryCount = 3

	MaxFunctionTimeout = 2 * time.Hour

	// MaxBodySize is the maximum payload size read on any HTTP response.
	MaxBodySize = 1024 * 1024 * 4

	FunctionIdempotencyPeriod = 24 * time.Hour
)
