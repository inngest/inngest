package redis_state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/inngest/inngest/pkg/util"
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
	GuaranteedCapacityTickTime = 30 * time.Second
	// AccountLeaseTime is how long accounts with guaranteed capacity are leased.
	AccountLeaseTime = 20 * time.Second

	maxAccountLeaseAttempts = 10

	// How many accounts to lease on a single worker when processing guaranteed capacity
	GuaranteedCapacityLeaseLimit = 1
)

func guaranteedCapacityKeyForAccount(accountId uuid.UUID) string {
	gc := GuaranteedCapacity{
		Scope:     enums.GuaranteedCapacityScopeAccount,
		AccountID: accountId,
	}

	return gc.Key()
}

// GuaranteedCapacity represents an account with guaranteed capacity.
type GuaranteedCapacity struct {
	Name string `json:"n,omitempty"`

	// Scope identifies the level of guaranteed capacity, currently we only support account
	Scope enums.GuaranteedCapacityScope `json:"s"`

	// AccountID identifies guaranteed capacity
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

func (g *GuaranteedCapacity) UnmarshalJSON(data []byte) error {
	// Start decoding the base type
	type baseType struct {
		Scope              enums.GuaranteedCapacityScope `json:"s"`
		AccountID          uuid.UUID                     `json:"a"`
		Priority           uint                          `json:"p"`
		GuaranteedCapacity uint                          `json:"gc"`
	}

	// If we encounter an error, always report it
	base := baseType{}
	if err := json.Unmarshal(data, &base); err != nil {
		return err
	}

	g.Scope = base.Scope
	g.AccountID = base.AccountID
	g.Priority = base.Priority
	g.GuaranteedCapacity = base.GuaranteedCapacity

	// Lua's cjson library fails to encode empty arrays, so we need to handle empty objects
	type leasesMap struct {
		Leases map[string]interface{} `json:"leases"`
	}
	if err := json.Unmarshal(data, &leasesMap{}); err == nil {
		return nil
	}

	// If we have an array of leases, decode them
	type leasesArr struct {
		Leases []ulid.ULID `json:"leases"`
	}
	l := leasesArr{}
	if err := json.Unmarshal(data, &l); err != nil {
		return err
	}

	g.Leases = l.Leases

	return nil
}

func (g *GuaranteedCapacity) Key() string {
	switch g.Scope {
	case enums.GuaranteedCapacityScopeAccount:
		return fmt.Sprintf("a:%s", g.AccountID)
	default:
		return fmt.Sprintf("a:%s", g.AccountID)
	}
}

// leasedAccount represents an account leased by a queue.
type leasedAccount struct {
	GuaranteedCapacity GuaranteedCapacity
	Lease              ulid.ULID
}

func (q *queue) getGuaranteedCapacityMap(ctx context.Context) (map[string]GuaranteedCapacity, error) {
	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for getGuaranteedCapacityMap: %s", q.primaryQueueShard.Kind)
	}

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "getGuaranteedCapacityMap"), redis_telemetry.ScopeQueue)

	m, err := q.primaryQueueShard.RedisClient.unshardedRc.Do(ctx, q.primaryQueueShard.RedisClient.unshardedRc.B().Hgetall().Key(q.primaryQueueShard.RedisClient.kg.GuaranteedCapacityMap()).Build()).AsMap()
	if rueidis.IsRedisNil(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error fetching guaranteed capacity map: %w", err)
	}
	guaranteedCapacityMap := map[string]GuaranteedCapacity{}
	for k, v := range m {
		guaranteedCapacity := GuaranteedCapacity{}
		if err := v.DecodeJSON(&guaranteedCapacity); err != nil {
			return nil, fmt.Errorf("error decoding guaranteed capacity: %w", err)
		}
		guaranteedCapacityMap[k] = guaranteedCapacity
	}
	return guaranteedCapacityMap, nil
}

// acquireAccountLease leases an account with guaranteed capacity for the given duration. GuaranteedCapacity can have more than one lease at a time;
// you must provide an index to claim a lease. THe index This prevents multiple workers
// from claiming the same lease index;  if workers A and B see an account with 0 leases and both attempt
// to claim lease "0", only one will succeed.
func (q *queue) acquireAccountLease(ctx context.Context, guaranteedCapacity GuaranteedCapacity, duration time.Duration, leaseIndex int) (*ulid.ULID, error) {
	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for acquireAccountLease: %s", q.primaryQueueShard.Kind)
	}

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "acquireAccountLease"), redis_telemetry.ScopeQueue)

	if guaranteedCapacity.Scope != enums.GuaranteedCapacityScopeAccount {
		return nil, fmt.Errorf("expected account scope, got %s", guaranteedCapacity.Scope)
	}

	now := q.clock.Now()
	leaseID, err := ulid.New(uint64(now.Add(duration).UnixMilli()), rand.Reader)
	if err != nil {
		return nil, err
	}

	keys := []string{q.primaryQueueShard.RedisClient.kg.GuaranteedCapacityMap()}
	args, err := StrSlice([]any{
		now.UnixMilli(),
		guaranteedCapacity.Key(),
		leaseID,
		leaseIndex,
	})
	if err != nil {
		return nil, err
	}

	status, err := scripts["queue/guaranteedCapacityAccountLease"].Exec(
		redis_telemetry.WithScriptName(ctx, "guaranteedCapacityAccountLease"),
		q.primaryQueueShard.RedisClient.unshardedRc,
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
	case int64(-4):
		return nil, errGuaranteedCapacityIndexExceeded
	case int64(0):
		return &leaseID, nil
	default:
		return nil, fmt.Errorf("unknown lease return value: %T(%v)", status, status)
	}
}

func (q *queue) renewAccountLease(ctx context.Context, guaranteedCapacity GuaranteedCapacity, duration time.Duration, leaseID ulid.ULID) (*ulid.ULID, error) {
	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for renewAccountLease: %s", q.primaryQueueShard.Kind)
	}

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "renewAccountLease"), redis_telemetry.ScopeQueue)

	if guaranteedCapacity.Scope != enums.GuaranteedCapacityScopeAccount {
		return nil, fmt.Errorf("expected account scope, got %s", guaranteedCapacity.Scope)
	}

	now := q.clock.Now()
	newLeaseID, err := ulid.New(uint64(now.Add(duration).UnixMilli()), rand.Reader)
	if err != nil {
		return nil, err
	}

	expireArg := "0"
	if duration == 0 {
		expireArg = "1"
	}

	keys := []string{q.primaryQueueShard.RedisClient.kg.GuaranteedCapacityMap()}
	args, err := StrSlice([]any{
		expireArg,
		now.UnixMilli(),
		guaranteedCapacity.Key(),
		leaseID,
		newLeaseID,
	})
	if err != nil {
		return nil, err
	}

	status, err := scripts["queue/guaranteedCapacityRenewAccountLease"].Exec(
		redis_telemetry.WithScriptName(ctx, "guaranteedCapacityRenewAccountLease"),
		q.primaryQueueShard.RedisClient.unshardedRc,
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
		return nil, errGuaranteedCapacityLeaseNotFound
	case int64(0):
		return &newLeaseID, nil
	default:
		return nil, fmt.Errorf("unknown lease renew return value: %T(%v)", status, status)
	}
}

func (q *queue) expireAccountLease(ctx context.Context, guaranteedCapacity GuaranteedCapacity, leaseID ulid.ULID) error {
	_, err := q.renewAccountLease(ctx, guaranteedCapacity, 0, leaseID)
	if err != nil {
		return fmt.Errorf("could not renew account lease: %w", err)
	}

	return nil
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

func (q *queue) claimUnleasedGuaranteedCapacity(ctx context.Context, scanTickTime, leaseTickTime time.Duration) {
	scanTick := q.clock.NewTicker(scanTickTime)
	leaseTick := q.clock.NewTicker(leaseTickTime / 2)

	// records whether we're leasing
	var leasing int32

	// records whether we're renewing
	var renewing int32

	for {
		if q.isSequential() {
			// Sequential workers never lease accounts.  They always run in order
			// on the global partition queue.
			// We perform this check every scan iteration to support claiming
			// guaranteed capacity if the sequential lease expires for some reason
			<-scanTick.Chan()
			continue
		}

		select {
		case <-ctx.Done():
			scanTick.Stop()
			leaseTick.Stop()

			// Copy the slice to prevent locking/concurrent access.
			existingLeases := q.getAccountLeases()

			// Attempt to expire all leases ASAP, even if the backing store is single threaded.
			for _, ls := range existingLeases {
				// Expire leases immediately from backing store.
				ls := ls
				go func(ls leasedAccount) {
					q.logger.Debug().Str("account", ls.GuaranteedCapacity.AccountID.String()).Msg("expiring account lease before shutdown")
					err := q.expireAccountLease(context.Background(), ls.GuaranteedCapacity, ls.Lease)
					if err != nil && !errors.Is(err, errGuaranteedCapacityNotFound) {
						q.logger.Error().Err(err).Msg("error expiring account lease")
					}
					q.removeLeasedAccount(ls.GuaranteedCapacity)
				}(ls)
			}

			return
		case <-scanTick.Chan():
			go func() {
				if !atomic.CompareAndSwapInt32(&leasing, 0, 1) {
					// Only one lease can occur at once.
					q.logger.Debug().Msg("already leasing accounts")
					return
				}

				// Always reset the leasing op to zero, allowing us to lease again.
				defer func() { atomic.StoreInt32(&leasing, 0) }()

				// Retry claiming leases until all accounts have been taken.  All operations
				// must succeed, even if it leaves us spinning.  Note that scanAndLeaseUnleasedAccounts filters
				// out unnecessary leases and accounts that have already been leased.
				retry := true
				n := 0
				for retry && n < maxAccountLeaseAttempts {
					n++
					var err error
					retry, err = q.scanAndLeaseUnleasedAccounts(ctx)
					if err != nil {
						q.logger.Error().Err(err).Msg("error scanning and leasing accounts")
						return
					}
					if retry {
						<-q.clock.After(time.Duration(mathRand.Intn(1000)) * time.Millisecond)
					}
				}
			}()
		case <-leaseTick.Chan():
			go func() {
				if !atomic.CompareAndSwapInt32(&renewing, 0, 1) {
					// Only one lease can occur at once.
					q.logger.Debug().Msg("already renewing accounts")
					return
				}

				// Always reset the renewing op to zero, allowing us to lease again.
				defer func() { atomic.StoreInt32(&renewing, 0) }()

				// Copy the slice to prevent locking/concurrent access.
				existingLeases := q.getAccountLeases()

				for _, s := range existingLeases {
					// Attempt to lease all ASAP, even if the backing store is single threaded.
					go func(ls leasedAccount) {
						nextLeaseID, err := q.renewAccountLease(ctx, ls.GuaranteedCapacity, AccountLeaseTime, ls.Lease)
						if err != nil {
							// Renewing a lease should never fail, unless guaranteed capacity was removed.
							// We must stop holding on to the account in any case.
							q.removeLeasedAccount(ls.GuaranteedCapacity)

							// If guaranteed capacity was removed, we can remove the internal lease state
							if errors.Is(err, errGuaranteedCapacityNotFound) {
								return
							}

							// If our lease was stolen, play nice and remove the leased account
							if errors.Is(err, errGuaranteedCapacityLeaseNotFound) {
								q.logger.Warn().Interface("lease", ls).Msg("giving up lease since it was removed in the backing store")
								return
							}

							q.logger.Error().Interface("lease", ls).Err(err).Msg("error renewing account lease")
							return
						}
						q.logger.Debug().Interface("account", ls).Msg("renewed account lease")
						// Update the lease ID so that we have this stored appropriately for
						// the next renewal.
						q.addLeasedAccount(ls.GuaranteedCapacity, *nextLeaseID)
					}(s)
				}
			}()

		}
	}
}

func (q *queue) scanAndLeaseUnleasedAccounts(ctx context.Context) (retry bool, err error) {
	// TODO: Make instances of *queue register worker information when calling
	//       Run().
	//       Fetch this information, and correctly assign workers to guaranteed capacity maps
	//       based on the distribution of items in the queue here.  This lets
	//       us oversubscribe appropriately.
	guaranteedCapacityMap, err := q.getGuaranteedCapacityMap(ctx)
	if err != nil {
		q.logger.Error().Err(err).Msg("error fetching guaranteed capacity map")
		return
	}

	filteredUnleasedAccounts, err := q.filterUnleasedAccounts(guaranteedCapacityMap)
	if err != nil {
		q.logger.Error().Err(err).Msg("error filtering unleased accounts")
		return
	}

	if len(filteredUnleasedAccounts) == 0 {
		return
	}

	// Only lease additional guaranteed capacity if current worker is within limits
	leaseNum := GuaranteedCapacityLeaseLimit - len(q.getAccountLeases())
	if leaseNum <= 0 {
		return
	}

	q.logger.Trace().Msgf("leasing %d accounts", leaseNum)

	for _, guaranteedCapacity := range filteredUnleasedAccounts[0:leaseNum] {
		leaseID, err := q.acquireAccountLease(ctx, guaranteedCapacity, AccountLeaseTime, len(guaranteedCapacity.Leases))
		if err == nil {
			guaranteedCapacity.Leases = append(guaranteedCapacity.Leases, *leaseID)
			// go q.counter(ctx, "queue_account_lease_success_total", 1, map[string]any{
			// 	"shard_name": guaranteedCapacity.Name,
			// })
			q.addLeasedAccount(guaranteedCapacity, *leaseID)
			q.logger.Debug().Interface("guaranteed_capacity", guaranteedCapacity).Str("lease_id", leaseID.String()).Msg("leased account with guaranteed capacity")
			continue
		}

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
		case errGuaranteedCapacityIndexExceeded:
			// This should not have happened; we should have filtered this out.
			q.logger.Error().Interface("guaranteed_capacity", guaranteedCapacity).Err(err).Msg("attempted to lease more than the maximum number of leases")
			continue
		default:
			q.logger.Error().Interface("guaranteed_capacity", guaranteedCapacity).Err(err).Msg("failed to lease account with guaranteed capacity")
			return true, err
		}

		// go q.counter(ctx, "queue_shard_lease_conflict_total", 1, map[string]any{
		// 	"shard_name": guaranteedCapacity.Name,
		// })
	}

	return retry, nil
}

func (q *queue) addLeasedAccount(guaranteedCapacity GuaranteedCapacity, lease ulid.ULID) {
	for i, n := range q.accountLeases {
		if n.GuaranteedCapacity.Key() == guaranteedCapacity.Key() {
			// Updated in place.
			q.accountLeaseLock.Lock()
			q.accountLeases[i] = leasedAccount{
				Lease:              lease,
				GuaranteedCapacity: guaranteedCapacity,
			}
			q.accountLeaseLock.Unlock()
			return
		}
	}
	// Not updated in place, so add to the list and return.
	q.accountLeaseLock.Lock()
	q.accountLeases = append(q.accountLeases, leasedAccount{
		Lease:              lease,
		GuaranteedCapacity: guaranteedCapacity,
	})
	q.accountLeaseLock.Unlock()
}

func (q *queue) removeLeasedAccount(guaranteedCapacity GuaranteedCapacity) {
	q.accountLeaseLock.Lock()
	defer q.accountLeaseLock.Unlock()

	if len(q.accountLeases) == 0 {
		return
	}

	filtered := make([]leasedAccount, len(q.accountLeases)-1)
	skipped := 0
	for i, accountLease := range q.accountLeases {
		if accountLease.GuaranteedCapacity.Key() == guaranteedCapacity.Key() {
			skipped += 1
			continue
		}

		filtered[i+skipped] = accountLease
	}

	q.accountLeases = filtered
}

// filterUnleasedAccounts filters guaranteed capacities during assignment, removing any accounts that this worker
// has already leased;  any accounts that have already had their leasing requirements met;
// and priority shuffles guaranteed capacity to lease in a non-deterministic (but prioritized) order.
//
// The returned guaranteed capacities are safe to be leased, and should be attempted in-order.
func (q *queue) filterUnleasedAccounts(guaranteedCapacityMap map[string]GuaranteedCapacity) ([]GuaranteedCapacity, error) {
	if len(guaranteedCapacityMap) == 0 {
		return nil, nil
	}

	// Remove non-account capacity
	for s, capacity := range guaranteedCapacityMap {
		if capacity.Scope != enums.GuaranteedCapacityScopeAccount {
			delete(guaranteedCapacityMap, s)
		}
	}

	// Copy the slice to prevent locking/concurrent access.
	for _, v := range q.getAccountLeases() {
		// Remove any accounts that have already been leased by this worker.
		delete(guaranteedCapacityMap, v.GuaranteedCapacity.Key())
	}

	weights := []float64{}
	shuffleIdx := []GuaranteedCapacity{}
	for _, v := range guaranteedCapacityMap {
		// XXX: Here we can add latency targets, etc.

		// Filter out expired leases; we may be able to take over from another worker.
		validLeases := []ulid.ULID{}
		for _, l := range v.Leases {
			if time.UnixMilli(int64(l.Time())).After(q.clock.Now()) {
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
	result := make([]GuaranteedCapacity, len(weights))
	for n := range result {
		idx, ok := w.Take()
		if !ok && len(result) < len(weights)-1 {
			return result, util.ErrWeightedSampleRead
		}
		result[n] = shuffleIdx[idx]
	}

	return result, nil
}
