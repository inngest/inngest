package constraintapi

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/oklog/ulid/v2"
)

type CapacityManager interface {
	Check(ctx context.Context, req *CapacityCheckRequest) (*CapacityCheckResponse, errs.UserError, errs.InternalError)
	Lease(ctx context.Context, req *CapacityLeaseRequest) (*CapacityLeaseResponse, errs.UserError, errs.InternalError)
	ExtendLease(ctx context.Context, req *CapacityExtendLeaseRequest) (*CapacityExtendLeaseResponse, errs.UserError, errs.InternalError)
	Commit(ctx context.Context, req *CapacityCommitRequest) (*CapacityCommitResponse, errs.UserError, errs.InternalError)
	Rollback(ctx context.Context, req *CapacityRollbackRequest) (*CapacityRollbackResponse, errs.UserError, errs.InternalError)
}

type CapacityCheckRequest struct {
	AccountID uuid.UUID
}

type CapacityCheckResponse struct{}

type CapacityLeaseRequest struct {
	// IdempotencyKey prevents performing the same lease request multiple times.
	// If a previous lease request was granted within the lease lifetime, the same lease will be returned.
	IdempotencyKey string

	AccountID uuid.UUID

	// EnvID is used for identifying the function.
	EnvID uuid.UUID

	// FunctionID is used for identifying the function.
	FunctionID uuid.UUID

	// Configuration represents the latest known constraint configuration (a subset of the function config).
	//
	// The server _may_ reject calls if it has recently seen a newer configuration. This is expected for a short
	// period after updating the configuration (as executors independently refresh the in-memory cache), but old
	// configurations should not be used for an extended time.
	Configuration ConstraintConfig

	// RequestedCapacity describes the constraints that should be checked for a request.
	//
	// This should include _all_ constraints that need to be checked to perform an operation.
	//
	// For example:
	// - To process a queue item, we need to check account, function, and optionally custom concurrency. If throttle is set,
	//   the request should also include throttle capacity.
	//
	// This design assumes that the other side _knows_ the current constraint.
	RequestedCapacity []ConstraintCapacityItem

	// CurrentTime specifies the current time on the calling side. If this drifts too far from the manager, the request will be
	// rejected. For generating the lease expiry, we will use the current time on the manager side.
	//
	// This is a cheap check to prevent clock skew. We instrument the skew and will set a reasonable threshold over time.
	CurrentTime time.Time

	// Duration specifies the lease duration. This may be capped by the manager.
	Duration time.Duration

	// MaximumLifetime specifies the maximum lifetime for a lease.
	// If the caller attempts to extend a lease past this duration, the request will be rejected.
	MaximumLifetime time.Duration

	// BlockingThreshold optionally allows the server to hold the request up to the specific Duration
	// in case capacity is likely to be available within the duration.
	//
	// Setting this may reduce roundtrip-time.
	BlockingThreshold time.Duration

	// Source includes information on the calling service and processing mode for instrumentation purposes and to enforce fairness/avoid starvation.
	Source LeaseSource
}

type CapacityLeaseResponse struct {
	LeaseID *ulid.ULID

	ReservedCapacity []ConstraintCapacityItem

	InsufficientCapacity []ConstraintCapacityItem

	RetryAfter time.Time
}

type CapacityExtendLeaseRequest struct {
	IdempotencyKey string

	AccountID uuid.UUID
	LeaseID   ulid.ULID

	Duration time.Duration
}

type CapacityExtendLeaseResponse struct {
	LeaseID *ulid.ULID
}

type CapacityCommitRequest struct {
	IdempotencyKey string

	AccountID uuid.UUID
	LeaseID   ulid.ULID
}

type CapacityCommitResponse struct{}

type CapacityRollbackRequest struct {
	IdempotencyKey string

	AccountID uuid.UUID
	LeaseID   ulid.ULID
}

type CapacityRollbackResponse struct{}

type RunProcessingMode int

const (
	// RunProcessingModeBackground is used for regular (async) run scheduling and execution.
	RunProcessingModeBackground RunProcessingMode = iota
	// RunProcessingModeSync is used for requests sent by the Checkpointing API/Project Zero.
	RunProcessingModeSync
)

type LeaseLocation int

const (
	LeaseLocationUnknown LeaseLocation = iota

	// LeaseLocationScheduleRun is hit before scheduling a run
	LeaseLocationScheduleRun

	// LeaseLocationPartitionLease is hit before leasing a partition
	LeaseLocationPartitionLease

	// LeaseLocationItemLease is hit before leasing a queue item
	LeaseLocationItemLease
)

type LeaseService int

const (
	ServiceUnknown LeaseService = iota
	ServiceNewRuns
	ServiceExecutor
	ServiceAPI
)

type LeaseSource struct {
	// Service refers to the origin service (new-runs, api, executor)
	Service LeaseService

	// Location refers to the lifecycle step requiring constraint checks
	Location LeaseLocation

	RunProcessingMode RunProcessingMode
}
