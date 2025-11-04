package constraintapi

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
)

type redisCapacityManager struct {
	client rueidis.Client
	clock  clockwork.Clock

	rateLimitKeyPrefix  string
	queueStateKeyPrefix string
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

func (r *redisCapacityManager) leasesHash(prefix string, accountID uuid.UUID) string {
	return fmt.Sprintf("{%s}:%s:leases", prefix, accountID)
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

// Acquire implements CapacityManager.
func (r *redisCapacityManager) Acquire(ctx context.Context, req *CapacityAcquireRequest) (*CapacityAcquireResponse, errs.InternalError) {
	keyPrefix, err := r.keyPrefix(req.Constraints)
	if err != nil {
		return nil, errs.Wrap(0, false, "failed to generate key prefix: %w", err)
	}

	keys := []string{
		r.leasesHash(keyPrefix, req.AccountID),
	}

	args, err := strSlice([]any{})
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
