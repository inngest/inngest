package consts

import (
	"time"
)

const (
	// DefaultRetryCount is used when no retry count for a step is specified.
	// Given 4 retries, each step or function is attempted 5 times by default.
	DefaultRetryCount = 4

	// DefaultMaxEventSize represents the maximum size of the event payload we process,
	// currently 512KB.
	DefaultMaxEventSize = 512 * 1024

	// AbsoluteMaxEventSize is the absolute maximum size of the event payload we process.
	AbsoluteMaxEventSize = 3 * 1024 * 1024

	// DefaultMaxStepLimit is the maximum number of steps per function allowed.
	DefaultMaxStepLimit = 1_000

	// AbsoluteMaxStepLimit is the absolute maximium number of steps that an executor can
	// be initialized with.
	AbsoluteMaxStepLimit = 10_000

	// MaxFunctionTimeout represents the longest running function or step allowed within
	// our system.
	MaxFunctionTimeout = 2 * time.Hour

	// MaxStepOutputSize is the maximum size of the output of a step.
	MaxStepOutputSize = 1024 * 1024 * 4 // 4MB

	// MaxStepInputSize is the maximum size of the input of a step.
	MaxStepInputSize = 1024 * 1024 * 4 // 4MB

	// MaxSDKResponseBodySize is the maximum payload size in the response from
	// the SDK.
	MaxSDKResponseBodySize = MaxStepOutputSize + MaxStepInputSize

	// MaxSDKRequestBodySize is the maximum payload size in the request to the
	// SDK.
	MaxSDKRequestBodySize = 1024 * 1024 * 4 // 4MB

	// DefaultMaxStateSizeLimit is the maximum number of bytes of output state per function run allowed.
	DefaultMaxStateSizeLimit = 1024 * 1024 * 32 // 32MB

	// MaxRetries represents the maximum number of retries for a particular function or step
	// possible.
	MaxRetries = 20

	// MaxRetryDuration is the furthest a retry can be scheduled.  If retries are scheduled further
	// than now plus this duration, the retry duration will automatically be lowered to this value.
	MaxRetryDuration = time.Hour * 24

	// MinRetryDuration is the soonest a retry can be scheduled.
	MinRetryDuration = time.Second * 1

	// MinDebouncePeriod is the minimum period of time that can be used to configure a debounce.
	MinDebouncePeriod = time.Second

	// MaxDebouncePeriod is the maximum period of time that can be used to configure a debounce.  This
	// lets users delay functions for up to MaxDebouncePeriod when events are received.
	MaxDebouncePeriod = time.Hour * 24 * 7

	// MaxCancellations represents the max automatic cancellation signals per function
	MaxCancellations = 5

	// MaxConcurrencyLimits limits the max concurrency constraints for a specific function.
	MaxConcurrencyLimits = 2

	// MaxTriggers represents the maximum number of triggers a function can have.
	MaxTriggers = 10

	// MaxBatchTTL represents the maximum amount of duration the batch key will last
	MaxBatchTTL = 10 * time.Minute

	// DefaultConcurrencyLimit is the default concurrency limit applied when not specified
	DefaultConcurrencyLimit = 1_000

	DefaultSystemConcurrencyLimit = 100_000

	// FunctionIdempotencyPeriod determines how long a specific function remains idempotent
	// when using idempotency keys.
	FunctionIdempotencyPeriod = 24 * time.Hour
	// FunctionIdempotencyTombstone indicates the run associated with this idempotency key
	// has already finished
	FunctionIdempotencyTombstone = "-"

	DefaultBatchSizeLimit = 100
	DefaultBatchTimeout   = 60 * time.Second

	// MaxEvents is the maximum number of events we can parse in a single batch.
	MaxEvents = 5_000

	InngestEventDataPrefix = "_inngest"
	// InvokeSlugKey is the data key used to store the fn name when invoking a function
	// via an RPC-like call, abstracting event-driven fanout.
	InvokeFnID          = "fn_id"
	InvokeCorrelationId = "correlation_id"

	// CancelTimeout is the maximum time a cancellation can exist
	CancelTimeout = time.Hour * 24 * 365

	RequestVersionUnknown = -1

	// PriorityFactorMin is the minimum priority factor for any function run, in seconds.
	PriorityFactorMin = int64(-1 * 60 * 60 * 12)
	// PriorityFactorMax is the maximum priority factor for any function run, in seconds.
	// This is set to 12 hours.
	PriorityFactorMax = int64(60 * 60 * 12)
	// FutureQueeueFudgeLimit is the inclusive time range between [now, now() + FutureAtLimit]
	// in which priority factors are taken into account.
	FutureAtLimit = 2 * time.Second

	DefaultQueueContinueLimit = uint(5)

	PauseExpiredDeletionGracePeriod = time.Minute * 20

	DefaultQueueShardName = "default"

	// Minimum number of pauses before using the aggregate pause handler.
	AggregatePauseThreshold = 50

	// QueueContinuationCooldownPeriod is the cooldown period for a continuations, eg.
	// how long we wait after a partition has continued the maximum times.
	// This prevents partitions from greedily acquiring resources in each scan loop.
	QueueContinuationCooldownPeriod = time.Second * 10
	// QueueContinuationMaxPartitions represents the total capacity for partitions
	// that can be continued.
	QueueContinuationMaxPartitions = 50
	// QueueContinuationSkipProbability is the probability of skipping a continuation
	// scan loop.
	QueueContinuationSkipProbability = 0.2

	//
	// Streaming
	//
	MaxStreamingMessageSizeBytes = 1024 * 512 // 512KB
	StreamingChunkSize           = 1024       // 1KB
	MaxStreamingChunks           = 1000       // Allow up to 1000 chunks per stream

	RedisBlockingPoolSize = 10

	ConnectWorkerHeartbeatInterval  = 10 * time.Second
	ConnectGatewayHeartbeatInterval = 5 * time.Second
	ConnectGCThreshold              = 5 * time.Minute

	ConnectWorkerRequestLeaseDuration = 20 * time.Second
	ConnectWorkerRequestGracePeriod   = 5 * time.Second

	// ConnectWorkerNoConcurrencyLimitForRequests is used to indicate that a worker has no capacity limit.
	ConnectWorkerNoConcurrencyLimitForRequests = -1

	KafkaMsgTooLargeError = "MESSAGE_TOO_LARGE"
)

var (
	ConnectWorkerRequestToWorkerMappingTTL  = 6 * ConnectWorkerRequestLeaseDuration  // 2 minutes so in case of gateway failure, we don't lose the mapping for too long
	ConnectWorkerCapacityManagerTTL         = 45 * ConnectWorkerRequestLeaseDuration // 15 minutes
	ConnectWorkerRequestExtendLeaseInterval = ConnectWorkerRequestLeaseDuration / 4
	QueueShadowContinuationCooldownPeriod   = QueueContinuationCooldownPeriod
	QueueShadowContinuationMaxPartitions    = QueueContinuationMaxPartitions
)
