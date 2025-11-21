package constraintapi

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/oklog/ulid/v2"
)

type CapacityManager interface {
	Check(ctx context.Context, req *CapacityCheckRequest) (*CapacityCheckResponse, errs.UserError, errs.InternalError)
	Acquire(ctx context.Context, req *CapacityAcquireRequest) (*CapacityAcquireResponse, errs.InternalError)
	ExtendLease(ctx context.Context, req *CapacityExtendLeaseRequest) (*CapacityExtendLeaseResponse, errs.InternalError)
	Release(ctx context.Context, req *CapacityReleaseRequest) (*CapacityReleaseResponse, errs.InternalError)
}

type RolloutKeyGenerator interface {
	KeyInProgressLeasesAccount(accountID uuid.UUID) string
	KeyInProgressLeasesFunction(accountID uuid.UUID, fnID uuid.UUID) string
	KeyInProgressLeasesCustom(accountID uuid.UUID, scope enums.ConcurrencyScope, entityID uuid.UUID, keyExpressionHash, evaluatedKeyHash string) string
	KeyConstraintCheckIdempotency(mi MigrationIdentifier, accountID uuid.UUID, leaseIdempotencyKey string) string
}

type RolloutManager interface {
	CapacityManager
	RolloutKeyGenerator
}

type wrappedManager struct {
	CapacityManager
	keyGenerator
}

func NewRolloutManager(cm CapacityManager, queueStatePrefix string, rateLimitPrefix string) RolloutManager {
	return &wrappedManager{
		keyGenerator: keyGenerator{
			rateLimitKeyPrefix:  rateLimitPrefix,
			queueStateKeyPrefix: queueStatePrefix,
		},
		CapacityManager: cm,
	}
}

// MigrationIdentifier includes hints for the Constraint API which will be removed
// once all constraint state is moved to a dedicated data store
//
// While we can infer the target data store from the contraint, we only send constraints
// during the Acquire call. Sharing the same migration identifier simplifies this.
type MigrationIdentifier struct {
	// IsRateLimit specifies whether the request is linked to a rate limit constraint vs.
	// queue constraints.
	//
	// This is only necessary until constraint state is migrated to a dedicated data store in a later milestone.
	IsRateLimit bool
	QueueShard  string
}

type CapacityCheckRequest struct {
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

	// Constraints describes the constraints that should be checked for a request.
	//
	// This should include _all_ constraints that need to be checked to perform an operation.
	//
	// For example:
	// - To process a queue item, we need to check account, function, and optionally custom concurrency. If throttle is set,
	//   the request should also include throttle capacity.
	//
	// This design assumes that the other side _knows_ the current constraint.
	Constraints []ConstraintItem

	Migration MigrationIdentifier
}

type CapacityCheckResponse struct {
	// AvailableCapacity for given constraints and configuration
	AvailableCapacity int

	// LimitingConstraints contains constraints that
	// ended up reducing the number of leases from the expected Amount.
	LimitingConstraints []ConstraintItem

	// Detailed constraint usage for requested constraints
	Usage []ConstraintUsage

	// FairnessReduction specifies the capacity that was reserved for fairness reasons.
	FairnessReduction int

	RetryAfter time.Time

	internalDebugState checkScriptResponse
}

type CapacityAcquireRequest struct {
	// IdempotencyKey prevents performing the same lease request multiple times.
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

	// Constraints describes the constraints that should be checked for a request.
	//
	// This should include _all_ constraints that need to be checked to perform an operation.
	//
	// For example:
	// - To process a queue item, we need to check account, function, and optionally custom concurrency. If throttle is set,
	//   the request should also include throttle capacity.
	//
	// This design assumes that the other side _knows_ the current constraint.
	Constraints []ConstraintItem

	// Amount specifies upper bound of requested capacity
	//
	// The Constraint API will check the provided constraints and calculate the
	// allowed capacity. This determines the number of created leases.
	Amount int

	// LeaseIdempotencyKeys represent individual idempotency keys to be used in case multiple leases are generated by the Acquire
	// request.
	//
	// This is useful to check the validity of individual leases using another Acquire call, as well as guaranteeing idempotency
	// in case the original lease expired by the time the respective item starts processing.
	LeaseIdempotencyKeys []string

	// LeaseRunIDs represent individual run IDs associated with the leases.
	// This may be empty in case the operation is not related to a run.
	//
	//
	// This may include duplicates: We may be acquiring leases for multiple items of the same run in parallel.
	LeaseRunIDs map[string]ulid.ULID

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

	Migration MigrationIdentifier
}

// CapacityLease represents the tuple of LeaseID <-> IdempotencyKey which identifies the leased resource (event, queue item, etc.).
type CapacityLease struct {
	// LeaseID is set to the time of lease expiry and will change when extended.
	LeaseID ulid.ULID

	// IdempotencyKey represents the resource associated with the lease, e.g. a queue item or event.
	IdempotencyKey string
}

type CapacityAcquireResponse struct {
	// Leases may contain anywhere between 0 and <Amount> IDs.
	//
	// Each lease will be identified by its idempotency key (set in LeaseIdempotencyKeys).
	//
	// Depending on the available constraint capacity, there may be
	// fewer leases than requested.
	Leases []CapacityLease

	// LimitingConstraints contains constraints that
	// ended up reducing the number of leases from the expected Amount.
	LimitingConstraints []ConstraintItem

	// FairnessReduction specifies the capacity that was reserved for fairness reasons.
	FairnessReduction int

	RetryAfter time.Time

	internalDebugState acquireScriptResponse
}

// Debug returns INTERNAL debug information
func (ac *CapacityAcquireResponse) Debug() []string {
	return ac.internalDebugState.Debug
}

type CapacityExtendLeaseRequest struct {
	// IdempotencyKey is the operation idempotency key
	IdempotencyKey string

	AccountID uuid.UUID

	// LeaseID is the current lease ID
	LeaseID ulid.ULID

	Duration time.Duration

	Migration MigrationIdentifier
}

type CapacityExtendLeaseResponse struct {
	// LeaseID is set to the next lease ID. If this is unset, the lease may have already expired.
	LeaseID *ulid.ULID

	internalDebugState extendLeaseScriptResponse
}

type CapacityReleaseRequest struct {
	// IdempotencyKey is the operation idempotency key
	IdempotencyKey string

	AccountID uuid.UUID

	// LeaseID is the current lease ID
	LeaseID ulid.ULID

	Migration MigrationIdentifier
}

type CapacityReleaseResponse struct {
	internalDebugState releaseScriptResponse
}

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

	LeaseLocationCheckpoint
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

type UseConstraintAPIFn func(ctx context.Context, accountID uuid.UUID) (enable bool, fallback bool)
