package redis_state

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"time"
)

// GuaranteedCapacity represents an account with guaranteed capacity.
type GuaranteedCapacity struct {
	// Guaranteed capacity name, eg. the company name for isolated execution
	Name string `json:"n"`

	AccountID uuid.UUID `json:"a"`

	// Priority represents the priority for this account.
	Priority uint `json:"p"`

	// GuaranteedCapacity represents the minimum number of workers that must
	// always scan this account.  If zero, there is no guaranteed capacity for
	// the account.
	GuaranteedCapacity uint `json:"gc"`

	// Leases stores the lease IDs from the workers which are currently leasing the
	// account.  The workers currently leasing the account are guaranteed to use
	// the account's partition queue as their source of work.
	Leases []ulid.ULID `json:"leases"`
}

// leasedAccount represents an account leased by a queue.
type leasedAccount struct {
	GuaranteedCapacity GuaranteedCapacity
	Lease              ulid.ULID
}

func (q *queue) getGuaranteedCapacityMap(ctx context.Context) (map[string]*GuaranteedCapacity, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "getGuaranteedCapacityMap"), redis_telemetry.ScopeQueue)

	m, err := q.u.unshardedRc.Do(ctx, q.u.unshardedRc.B().Hgetall().Key(q.u.kg.GuaranteedCapacityMap()).Build()).AsMap()
	if rueidis.IsRedisNil(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error fetching guaranteed capacity map: %w", err)
	}
	shards := map[string]*GuaranteedCapacity{}
	for k, v := range m {
		shard := &GuaranteedCapacity{}
		if err := v.DecodeJSON(shard); err != nil {
			return nil, fmt.Errorf("error decoding guaranteed capacity: %w", err)
		}
		shards[k] = shard
	}
	return shards, nil
}

// leaseAccount leases an account with guaranteed capacity for the given duration. GuaranteedCapacity can have more than one lease at a time;
// you must provide an index to claim a lease. THe index This prevents multiple workers
// from claiming the same lease index;  if workers A and B see an account with 0 leases and both attempt
// to claim lease "0", only one will succeed.
func (q *queue) leaseAccount(ctx context.Context, shard *GuaranteedCapacity, duration time.Duration, n int) (*ulid.ULID, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "leaseAccount"), redis_telemetry.ScopeQueue)

	now := getNow()
	leaseID, err := ulid.New(uint64(now.Add(duration).UnixMilli()), rand.Reader)
	if err != nil {
		return nil, err
	}

	keys := []string{q.u.kg.GuaranteedCapacityMap()}
	args, err := StrSlice([]any{
		now.UnixMilli(),
		shard.Name,
		leaseID,
		n,
	})
	if err != nil {
		return nil, err
	}

	status, err := scripts["queue/guaranteedCapacityAccountLease"].Exec(
		redis_telemetry.WithScriptName(ctx, "guaranteedCapacityAccountLease"),
		q.u.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return nil, fmt.Errorf("error leasing item: %w", err)
	}
	switch status {
	case int64(-1):
		return nil, errShardNotFound
	case int64(-2):
		return nil, errShardIndexLeased
	case int64(-3):
		return nil, errShardIndexInvalid
	case int64(0):
		return &leaseID, nil
	default:
		return nil, fmt.Errorf("unknown lease return value: %T(%v)", status, status)
	}
}

func (q *queue) renewAccountLease(ctx context.Context, guaranteedCapacity *GuaranteedCapacity, duration time.Duration, leaseID ulid.ULID) (*ulid.ULID, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "renewAccountLease"), redis_telemetry.ScopeQueue)

	now := getNow()
	newLeaseID, err := ulid.New(uint64(now.Add(duration).UnixMilli()), rand.Reader)
	if err != nil {
		return nil, err
	}

	keys := []string{q.u.kg.GuaranteedCapacityMap()}
	args, err := StrSlice([]any{
		now.UnixMilli(),
		guaranteedCapacity.Name,
		leaseID,
		newLeaseID,
	})
	if err != nil {
		return nil, err
	}

	status, err := scripts["queue/guaranteedCapacityRenewAccountLease"].Exec(
		redis_telemetry.WithScriptName(ctx, "guaranteedCapacityRenewAccountLease"),
		q.u.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return nil, fmt.Errorf("error leasing item: %w", err)
	}
	switch status {
	case int64(-1):
		return nil, fmt.Errorf("guaranteed capacity not found")
	case int64(-2):
		return nil, fmt.Errorf("lease not found")
	case int64(0):
		return &newLeaseID, nil
	default:
		return nil, fmt.Errorf("unknown lease renew return value: %T(%v)", status, status)
	}
}

//nolint:all
func (q *queue) getAccountLeases() []leasedAccount {
	q.accountLeaseLock.Lock()
	existingLeases := make([]leasedAccount, len(q.accountLeases))
	for n, i := range q.accountLeases {
		existingLeases[n] = i
	}
	q.accountLeaseLock.Unlock()
	return existingLeases
}
