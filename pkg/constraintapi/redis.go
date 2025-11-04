package constraintapi

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
)

type redisCapacityManager struct {
	client rueidis.Client
	clock  clockwork.Clock
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

// Acquire implements CapacityManager.
func (r *redisCapacityManager) Acquire(ctx context.Context, req *CapacityAcquireRequest) (*CapacityAcquireResponse, errs.InternalError) {
	keys := []string{}

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
