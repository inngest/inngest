package redis_state

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"gonum.org/v1/gonum/stat/sampleuv"
	mathRand "math/rand"
	"sync/atomic"
	"time"
)

var (
	// GuaranteedCapacityTickTime is the duration in which we periodically check guaranteed capacity for
	// lease information, etc.
	GuaranteedCapacityTickTime = 15 * time.Second
	// AccountLeaseTime is how long accounts with guaranteed capacity are leased.
	AccountLeaseTime = 10 * time.Second

	maxAccountLeaseAttempts = 10

	// How many accounts to lease on a single worker when processing guaranteed capacity
	GuaranteedCapacityLeaseLimit = 1
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
	guaranteedCapacityMap := map[string]*GuaranteedCapacity{}
	for k, v := range m {
		guaranteedCapacity := &GuaranteedCapacity{}
		if err := v.DecodeJSON(guaranteedCapacity); err != nil {
			return nil, fmt.Errorf("error decoding guaranteed capacity: %w", err)
		}
		guaranteedCapacityMap[k] = guaranteedCapacity
	}
	return guaranteedCapacityMap, nil
}

// leaseAccount leases an account with guaranteed capacity for the given duration. GuaranteedCapacity can have more than one lease at a time;
// you must provide an index to claim a lease. THe index This prevents multiple workers
// from claiming the same lease index;  if workers A and B see an account with 0 leases and both attempt
// to claim lease "0", only one will succeed.
func (q *queue) leaseAccount(ctx context.Context, guaranteedCapacity *GuaranteedCapacity, duration time.Duration, n int) (*ulid.ULID, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "leaseAccount"), redis_telemetry.ScopeQueue)

	now := getNow()
	leaseID, err := ulid.New(uint64(now.Add(duration).UnixMilli()), rand.Reader)
	if err != nil {
		return nil, err
	}

	keys := []string{q.u.kg.GuaranteedCapacityMap()}
	args, err := StrSlice([]any{
		now.UnixMilli(),
		guaranteedCapacity.Name,
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
		return nil, errGuaranteedCapacityNotFound
	case int64(-2):
		return nil, errGuaranteedCapacityIndexLeased
	case int64(-3):
		return nil, errGuaranteedCapacityIndexInvalid
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

func (q *queue) claimUnleasedGuaranteedCapacity(ctx context.Context) {
	if q.gcf == nil {
		// TODO: Inspect denylists and whether this worker is capable of leasing
		// accounts.  Note that accounts should only be created for SDK-based workers;
		// if you use the queue for anything else other than step jobs, the worker
		// cannot lease jobs.  To this point, we should make this opt-in instead of
		// opt-out to prevent errors.
		//
		// For now, you have to provide a guaranteed capacity finder to lease accounts.
		q.logger.Info().Msg("no guaranteed capacity finder;  skipping account claiming")
		return
	}

	scanTick := time.NewTicker(GuaranteedCapacityTickTime)
	leaseTick := time.NewTicker(AccountLeaseTime / 2)

	// records whether we're leasing
	var leasing int32

	for {
		if q.isSequential() {
			// Sequential workers never lease accounts.  They always run in order
			// on the global partition queue.
			<-scanTick.C
			continue
		}

		select {
		case <-ctx.Done():
			// TODO: Remove leases immediately from backing store.
			scanTick.Stop()
			leaseTick.Stop()
			return
		case <-scanTick.C:
			go func() {
				if len(q.accountLeases) >= GuaranteedCapacityLeaseLimit {
					// We've reached the maximum number of leases we can handle.
					return
				}

				if !atomic.CompareAndSwapInt32(&leasing, 0, 1) {
					// Only one lease can occur at once.
					q.logger.Debug().Msg("already leasing accounts")
					return
				}

				// Always reset the leasing op to zero, allowing us to lease again.
				defer func() { atomic.StoreInt32(&leasing, 0) }()

				// Retry claiming leases until all accounts have been taken.  All operations
				// must succeed, even if it leaves us spinning.  Note that scanGuaranteedCapacity filters
				// out unnecessary leases and accounts that have already been leased.
				retry := true
				n := 0
				for retry && n < maxAccountLeaseAttempts {
					n++
					var err error
					retry, err = q.scanGuaranteedCapacity(ctx)
					if err != nil {
						q.logger.Error().Err(err).Msg("error scanning and leasing accounts")
						return
					}
					if retry {
						<-time.After(time.Duration(mathRand.Intn(50)) * time.Millisecond)
					}
				}
			}()
		case <-leaseTick.C:
			// Copy the slice to prevent locking/concurrent access.
			existingLeases := q.getAccountLeases()

			for _, s := range existingLeases {
				// Attempt to lease all ASAP, even if the backing store is single threaded.
				go func(ls leasedAccount) {
					nextLeaseID, err := q.renewAccountLease(ctx, &ls.GuaranteedCapacity, AccountLeaseTime, ls.Lease)
					if err != nil {
						q.logger.Error().Err(err).Msg("error renewing account lease")
						return
					}
					q.logger.Debug().Interface("account", ls).Msg("renewed account lease")
					// Update the lease ID so that we have this stored appropriately for
					// the next renewal.
					q.addLeasedAccount(ctx, &ls.GuaranteedCapacity, *nextLeaseID)
				}(s)
			}
		}
	}
}

func (q *queue) scanGuaranteedCapacity(ctx context.Context) (retry bool, err error) {
	// TODO: Make instances of *queue register worker information when calling
	//       Run().
	//       Fetch this information, and correctly assign workers to guaranteed capacity maps
	//       based on the distribution of items in the queue here.  This lets
	//       us oversubscribe appropriately.
	guaranteedCapacityMap, err := q.getGuaranteedCapacityMap(ctx)
	if err != nil {
		q.logger.Error().Err(err).Msg("error fetching filteredGuaranteedCapacity")
		return
	}

	filteredGuaranteedCapacity, err := q.filterGuaranteedCapacity(ctx, guaranteedCapacityMap)
	if err != nil {
		q.logger.Error().Err(err).Msg("error filtering filteredGuaranteedCapacity")
		return
	}

	if len(filteredGuaranteedCapacity) == 0 {
		return
	}

	leaseNum := GuaranteedCapacityLeaseLimit - len(guaranteedCapacityMap)
	// TODO Do we need to check if some leases have expired locally and remove them?
	//if leaseNum <= 0 {
	//	return
	//}

	for _, guaranteedCapacity := range filteredGuaranteedCapacity[0:leaseNum] {
		leaseID, err := q.leaseAccount(ctx, guaranteedCapacity, AccountLeaseTime, len(guaranteedCapacity.Leases))
		if err == nil {
			// go q.counter(ctx, "queue_account_lease_success_total", 1, map[string]any{
			// 	"shard_name": guaranteedCapacity.Name,
			// })
			q.addLeasedAccount(ctx, guaranteedCapacity, *leaseID)
			q.logger.Debug().Interface("guaranteed_capacity", guaranteedCapacity).Str("lease_id", leaseID.String()).Msg("leased guaranteedCapacity")
		} else {
			q.logger.Debug().Interface("guaranteed_capacity", guaranteedCapacity).Err(err).Msg("failed to lease guaranteedCapacity")
		}

		// go q.counter(ctx, "queue_shard_lease_conflict_total", 1, map[string]any{
		// 	"shard_name": guaranteedCapacity.Name,
		// })

		switch err {
		case errGuaranteedCapacityNotFound:
			// This is okay;  the guaranteedCapacity was removed when trying to lease
			continue
		case errGuaranteedCapacityIndexLeased:
			// This is okay;  another worker grabbed the lease.  No need to retry
			// as another worker grabbed this.
			continue
		case errGuaranteedCapacityIndexInvalid:
			// A lease expired while trying to lease — try again.
			retry = true
		default:
			return true, err
		}
	}

	return retry, nil
}

func (q *queue) addLeasedAccount(ctx context.Context, guaranteedCapacity *GuaranteedCapacity, lease ulid.ULID) {
	for i, n := range q.accountLeases {
		if n.GuaranteedCapacity.Name == guaranteedCapacity.Name {
			// Updated in place.
			q.accountLeaseLock.Lock()
			q.accountLeases[i] = leasedAccount{
				Lease:              lease,
				GuaranteedCapacity: *guaranteedCapacity,
			}
			q.accountLeaseLock.Unlock()
			return
		}
	}
	// Not updated in place, so add to the list and return.
	q.accountLeaseLock.Lock()
	q.accountLeases = append(q.accountLeases, leasedAccount{
		Lease:              lease,
		GuaranteedCapacity: *guaranteedCapacity,
	})
	q.accountLeaseLock.Unlock()
}

// filterGuaranteedCapacity filters guaranteed capacities during assignment, removing any accounts that this worker
// has already leased;  any accounts that have already had their leasing requirements met;
// and priority shuffles guaranteed capacity to lease in a non-deterministic (but prioritized) order.
//
// The returned guaranteed capacities are safe to be leased, and should be attempted in-order.
func (q *queue) filterGuaranteedCapacity(ctx context.Context, guaranteedCapacityMap map[string]*GuaranteedCapacity) ([]*GuaranteedCapacity, error) {
	if len(guaranteedCapacityMap) == 0 {
		return nil, nil
	}

	// Copy the slice to prevent locking/concurrent access.
	for _, v := range q.getAccountLeases() {
		delete(guaranteedCapacityMap, v.GuaranteedCapacity.Name)
	}

	weights := []float64{}
	shuffleIdx := []*GuaranteedCapacity{}
	for _, v := range guaranteedCapacityMap {
		// XXX: Here we can add latency targets, etc.

		validLeases := []ulid.ULID{}
		for _, l := range v.Leases {
			if time.UnixMilli(int64(l.Time())).After(getNow()) {
				validLeases = append(validLeases, l)
			}
		}
		// Replace leases with the # of valid leases.
		v.Leases = validLeases

		if len(validLeases) >= int(v.GuaranteedCapacity) {
			continue
		}

		weights = append(weights, float64(v.Priority))
		shuffleIdx = append(shuffleIdx, v)
	}

	if len(shuffleIdx) == 1 {
		return shuffleIdx, nil
	}

	// Reduce the likelihood of all workers attempting to claim accounts by
	// randomly shuffling.  Note that high priority accounts will still be
	// likely to come first with some contention.
	w := sampleuv.NewWeighted(weights, rnd)
	result := make([]*GuaranteedCapacity, len(weights))
	for n := range result {
		idx, ok := w.Take()
		if !ok && len(result) < len(weights)-1 {
			return result, ErrWeightedSampleRead
		}
		result[n] = shuffleIdx[idx]
	}

	return result, nil
}
