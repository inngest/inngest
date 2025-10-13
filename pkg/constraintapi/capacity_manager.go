package constraintapi

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

type CapacityCheckRequest struct {
	AccountID uuid.UUID
}

type CapacityCheckResponse struct{}

type ConstraintKind string

const (
	CapacityKindRateLimit   ConstraintKind = "rate_limit"
	CapacityKindConcurrency ConstraintKind = "concurrency"
	CapacityKindThrottle    ConstraintKind = "throttle"
)

type RateLimitCapacity struct {
	Scope enums.RateLimitScope

	KeyExpressionHash string

	EvaluatedKeyHash string
}

type ConcurrencyCapacity struct {
	// Mode specifies whether concurrency is applied to step (default) or function run level
	Mode enums.ConcurrencyMode

	// Scope specifies the concurrency scope, defaults to function
	Scope enums.ConcurrencyScope

	// KeyExpressionHash is the hashed key expression. If this is set, this refers to a custom concurrency key.
	KeyExpressionHash string
	EvaluatedKeyHash  string
}

type ThrottleCapacity struct {
	Scope enums.ThrottleScope

	KeyExpressionHash string
	EvaluatedKeyHash  string
}

type ConstraintCapacityItem struct {
	Kind *ConstraintKind

	// Amount specifies the number of units for a constraint.
	//
	// Examples:
	// 3 units of step-level concurrency allows to run 3 steps (~queue items).
	// 1 unit of rate limit capacity allows to start 1 run. Rejecting causes event to be skipped.
	// 1 unit of throttle capacity allows to start 1 run. Rejecting causes queue to wait and retry.
	Amount int
}

type CapacityLeaseRequest struct {
	// IdempotencyKey prevents performing the same lease request multiple times.
	// If a previous lease request was granted within the lease lifetime, the same lease will be returned.
	IdempotencyKey string

	AccountID uuid.UUID

	// EnvID is used for identifying the function.
	EnvID uuid.UUID

	// FunctionID is used for identifying the function.
	FunctionID uuid.UUID

	// LatestFunctionVersion specifies the latest known function version.
	// If the version on the manager is newer, it will be used.
	// If the version on the manager is outdated (e.g. stale cache), the latest version will be fetched.
	LatestFunctionVersion int

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

type CapacityManager interface {
	Check(ctx context.Context, req *CapacityCheckRequest) (*CapacityCheckResponse, error)
	Lease(ctx context.Context, req *CapacityLeaseRequest) (*CapacityLeaseResponse, error)
	ExtendLease(ctx context.Context, req *CapacityExtendLeaseRequest) (*CapacityExtendLeaseResponse, error)
	Commit(ctx context.Context, req *CapacityCommitRequest) (*CapacityCommitResponse, error)
	Rollback(ctx context.Context, req *CapacityRollbackRequest) (*CapacityRollbackResponse, error)
}
