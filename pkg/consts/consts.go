package consts

import "time"

const (
	// DefaultRetryCount is used when no retry count for a step is specified.
	DefaultRetryCount = 3

	MaxFunctionTimeout = 2 * time.Hour

	// MaxBodySize is the maximum payload size read on any HTTP response.
	MaxBodySize = 1024 * 1024 * 4 // 4MB

	FunctionIdempotencyPeriod = 24 * time.Hour

	MaxBatchSize    = 100
	MaxBatchTimeout = 60 * time.Second

	// InvokeEventName is the event name used to invoke specific functions via an
	// API.  Note that invoking functions still sends an event in the usual manner.
	InvokeEventName = "inngest/function.invoked"
	// InvokeSlugKey is the data key used to store the fn name when invoking a function
	// via an RPC-like call, abstracting event-driven fanout.
	InvokeSlugKey = "_inngest_fn"
)
