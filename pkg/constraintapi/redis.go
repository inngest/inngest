package constraintapi

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
)

const (
	MaximumAllowedRequestDelay = time.Second
)

type redisCapacityManager struct {
	client rueidis.Client
	clock  clockwork.Clock

	rateLimitKeyPrefix  string
	queueStateKeyPrefix string

	numScavengerShards int
}

type redisCapacityManagerOption func(m *redisCapacityManager)

func WithClient(client rueidis.Client) redisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.client = client
	}
}

func WithClock(clock clockwork.Clock) redisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.clock = clock
	}
}

func WithRateLimitKeyPrefix(prefix string) redisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.rateLimitKeyPrefix = prefix
	}
}

func WithQueueStateKeyPrefix(prefix string) redisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.queueStateKeyPrefix = prefix
	}
}

func WithNumScavengerShards(numShards int) redisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.numScavengerShards = numShards
	}
}

func NewRedisCapacityManager(
	options ...redisCapacityManagerOption,
) (*redisCapacityManager, error) {
	manager := &redisCapacityManager{}

	for _, rcmo := range options {
		rcmo(manager)
	}

	if manager.client == nil {
		return nil, fmt.Errorf("missing client")
	}

	if manager.clock == nil {
		manager.clock = clockwork.NewRealClock()
	}

	return manager, nil
}

/*
*
* Key structure:
*
* Global:
*
* - {prefix}:css:<number> Sharded scavenger entrypoints
*
* Per Account:
*
* Lease management:
* 	- {prefix}:{account}:reqh: {idempotencyKey} -> request json: Request details (config, constraints, allowed capacity, metadata)
* 	- {prefix}:{account}:reql:{idempotencyKey} -> ULID: current lease ID for the idempotency key
* 		- Expires with the lease expiry
* 	- {prefix}:{account}:leaseq -> sorted set of idempotencyKey tied to the expiry
*
* - {prefix}:{account}:ik:op:{operation}:{idempotencyKey} -> return whether the operation recently succeeded
*
 */

// scavengerShard represents the top-level sharded sorted set containing individual accounts
func (r *redisCapacityManager) scavengerShard(prefix string, shard int) string {
	return fmt.Sprintf("{%s}:css:%d", prefix, shard)
}

// accountLeases represents active leases for the account
func (r *redisCapacityManager) accountLeases(prefix string, accountID uuid.UUID) string {
	return fmt.Sprintf("{%s}:%s:leaseq", prefix, accountID)
}

// requestsHash returns the key to the hash storing request information
func (r *redisCapacityManager) requestsHash(prefix string, accountID uuid.UUID) string {
	return fmt.Sprintf("{%s}:%s:reqh", prefix, accountID)
}

// requestLease returns the key to the mapping key connecting an idempotency key to its current lease ID
func (r *redisCapacityManager) requestLease(prefix string, accountID uuid.UUID, idempotencyKey string) string {
	return fmt.Sprintf("{%s}:%s:reql:%s", prefix, accountID, idempotencyKey)
}

func (r *redisCapacityManager) operationIdempotencyKey(prefix string, accountID uuid.UUID, operation, idempotencyKey string) string {
	return fmt.Sprintf("{%s}:%s:ik:op:%s:%s", prefix, accountID, operation, idempotencyKey)
}

// keyPrefix returns the Lua key prefix for the first stage of the Constraint API.
//
// Since we are colocating lease data with the existing state, we will have to use the
// same Redis hash tag to avoid Lua errors and inconsistencies on the old and new scripts.
//
// This is essentially required for backward- and forward-compatibility.
func (r *redisCapacityManager) keyPrefix(
	constraints []ConstraintItem,
) (string, error) {
	var hasRateLimit bool
	var hasQueueConstraint bool

	for _, ci := range constraints {
		switch ci.Kind {
		case ConstraintKindConcurrency, ConstraintKindThrottle:
			hasQueueConstraint = true
		case ConstraintKindRateLimit:
			hasRateLimit = true
		default:
		}
	}

	if hasRateLimit && hasQueueConstraint {
		return "", fmt.Errorf("mixed constraints are not allowed during the first stage")
	}

	if hasRateLimit {
		return r.rateLimitKeyPrefix, nil
	}

	return r.queueStateKeyPrefix, nil
}

// redisRequestState represents the data structure stored for every request
// This is used by subsequent calls to Extend, Release to properly handle the lease lifecycle
//
// NOTE: This does not represent one individual lease but is used by
// all leases generated in the Acquire call.
type redisRequestState struct {
	OperationIdempotencyKey string    `json:"k,omitempty"`
	EnvID                   uuid.UUID `json:"e,omitempty"`
	FunctionID              uuid.UUID `json:"f,omitempty"`

	// SortedConstraints represents the list of constraints
	// included in the request sorted to execute in the expected
	// order. Configuration limits are now embedded directly in each constraint.
	SortedConstraints []SerializedConstraintItem `json:"s"`

	// ConfigVersion represents the function version used for this request
	ConfigVersion int `json:"cv,omitempty"`

	// RequestedAmount represents the Amount field in the Acquire request
	RequestedAmount int `json:"r,omitempty"`

	// GrantedAmount is populated in Lua during Acquire and represents the actual capacity granted to the request (how many leases were generated)
	GrantedAmount int `json:"g,omitempty"`

	// ActiveAmount represents the total number of active leases (where Release was not yet called)
	ActiveAmount int `json:"a,omitempty"`

	// MaximumLifetime is optional and represenst the maximum lifetime for leases generated by this request.
	// This is enforced during ExtendLease.
	MaximumLifetimeMillis int64 `json:"l,omitempty"`
}

func buildRequestState(req *CapacityAcquireRequest) *redisRequestState {
	state := &redisRequestState{
		OperationIdempotencyKey: req.IdempotencyKey,
		EnvID:                   req.EnvID,
		FunctionID:              req.FunctionID,
		RequestedAmount:         req.Amount,
		MaximumLifetimeMillis:   req.MaximumLifetime.Milliseconds(),
		ConfigVersion:           req.Configuration.FunctionVersion,

		// These keys are set during Acquire and Release respectively
		GrantedAmount: 0,
		ActiveAmount:  0,
	}

	// Sort and serialize constraints with embedded configuration limits
	constraints := req.Constraints
	sortConstraints(constraints)

	serialized := make([]SerializedConstraintItem, len(constraints))
	for i := range constraints {
		serialized[i] = constraints[i].ToSerializedConstraintItem(req.Configuration)
	}

	state.SortedConstraints = serialized

	return state
}

// Acquire implements CapacityManager.
func (r *redisCapacityManager) Acquire(ctx context.Context, req *CapacityAcquireRequest) (*CapacityAcquireResponse, errs.InternalError) {
	// TODO: Add metric for this
	// NOTE: This will include request latency (marshaling, network delays),
	// and it might not work for retries, as those retain the same CurrentTime value.
	// TODO: Ensure retries have the updated CurrentTime
	requestLatency := r.clock.Now().Sub(req.CurrentTime)
	if requestLatency > MaximumAllowedRequestDelay {
		// TODO : Set proper error code
		return nil, errs.Wrap(0, false, "exceeded maximum allowed request delay")
	}

	keyPrefix, err := r.keyPrefix(req.Constraints)
	if err != nil {
		return nil, errs.Wrap(0, false, "failed to generate key prefix: %w", err)
	}

	keys := []string{
		// r.leasesHash(keyPrefix, req.AccountID),
		r.operationIdempotencyKey(keyPrefix, req.AccountID, "acq", req.IdempotencyKey),
	}

	requestState := buildRequestState(req)

	args, err := strSlice([]any{
		// This will be marshaled
		requestState,
	})
	if err != nil {
		return nil, errs.Wrap(0, false, "invalid args: %w", err)
	}

	status, err := scripts["acquire"].Exec(ctx, r.client, keys, args).AsInt64()
	if err != nil {
		return nil, errs.Wrap(0, false, "acquire script failed: %w", err)
	}

	switch status {
	default:
		return nil, errs.Wrap(0, false, "unexpected status code %v", status)
	}
}

// Check implements CapacityManager.
func (r *redisCapacityManager) Check(ctx context.Context, req *CapacityCheckRequest) (*CapacityCheckResponse, errs.UserError, errs.InternalError) {
	keys := []string{}

	args, err := strSlice([]any{})
	if err != nil {
		return nil, nil, errs.Wrap(0, false, "invalid args: %w", err)
	}

	status, err := scripts["check"].Exec(ctx, r.client, keys, args).AsInt64()
	if err != nil {
		return nil, nil, errs.Wrap(0, false, "check script failed: %w", err)
	}

	switch status {
	default:
		return nil, nil, errs.Wrap(0, false, "unexpected status code %v", status)
	}
}

// ExtendLease implements CapacityManager.
func (r *redisCapacityManager) ExtendLease(ctx context.Context, req *CapacityExtendLeaseRequest) (*CapacityExtendLeaseResponse, errs.InternalError) {
	keys := []string{}

	args, err := strSlice([]any{})
	if err != nil {
		return nil, errs.Wrap(0, false, "invalid args: %w", err)
	}

	status, err := scripts["extend"].Exec(ctx, r.client, keys, args).AsInt64()
	if err != nil {
		return nil, errs.Wrap(0, false, "extend script failed: %w", err)
	}

	switch status {
	default:
		return nil, errs.Wrap(0, false, "unexpected status code %v", status)
	}
}

// Release implements CapacityManager.
func (r *redisCapacityManager) Release(ctx context.Context, req *CapacityReleaseRequest) (*CapacityReleaseResponse, errs.InternalError) {
	keys := []string{}

	args, err := strSlice([]any{})
	if err != nil {
		return nil, errs.Wrap(0, false, "invalid args: %w", err)
	}

	status, err := scripts["release"].Exec(ctx, r.client, keys, args).AsInt64()
	if err != nil {
		return nil, errs.Wrap(0, false, "release script failed: %w", err)
	}

	switch status {
	default:
		return nil, errs.Wrap(0, false, "unexpected status code %v", status)
	}
}

/*
*	local requestData
*	local latestConfig
*
* handleIdempotency()
*		... might return if request was successfully completed within idempotency TTL
*
*	local reserved = {
*
*
*	}
*
*
* if requestData.rateLimit {
*			local remainingCapacity = (limit(latestConfig, "accountConcurrency") - current("accountConcurrency"))
*		reserved["accountConcurrency"] = remainingCapacity
*
	* }
*
*
* if requestData.throttle {
*			local remainingCapacity = (limit(latestConfig, "accountConcurrency") - current("accountConcurrency"))
*		reserved["accountConcurrency"] = remainingCapacity
*
	* }
*
* if requestData.accountConcurrency {
*			local remainingCapacity = (limit(latestConfig, "accountConcurrency") - current("accountConcurrency"))
*		reserved["accountConcurrency"] = remainingCapacity
*
	* }
*
*
* if requestData.functionConcurrency {
*			local remainingCapacity = (limit(latestConfig, "accountConcurrency") - current("accountConcurrency"))
*		reserved["accountConcurrency"] = remainingCapacity
*
	* }
*
*
* if requestData.customConcurrency {
*			local remainingCapacity = (limit(latestConfig, "accountConcurrency") - current("accountConcurrency"))
*		reserved["accountConcurrency"] = remainingCapacity
*
	* }
*
*
*
* if isEmpty(reserved) {
*		// return "no capacity left, please wait for a bit"
* }
*
* -- At this point, we know that the request reserved _some_ capacity
*
*	redis.call("ZADD", leasesSet, leaseID, leaseExpiry)
*
*	redis.call("HSET", leasesHash, leaseID, requestData)
*	redis.call("HSET", leaseReserved, leaseID, reserved )
*
*	return {
*		leaseID,
*		reserved, -- client should know how much capacity was _actually_ reserved
*
*	}
*
*
*/
