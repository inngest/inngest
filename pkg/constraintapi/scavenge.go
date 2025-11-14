package constraintapi

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"golang.org/x/sync/errgroup"
)

const (
	defaultScavengerAccountsPeekSize = 20
	defaultScavengerLeasesPeekSize   = 20
	defaultScavengerConcurrency      = 20
)

type CapacityLeaseScavenger interface {
	Scavenge(ctx context.Context, peekSize int) errs.InternalError
}

type scavengerOptions struct {
	accountsPeekSize int
	leasesPeekSize   int
	concurrency      int
}

type scavengerOpt func(o *scavengerOptions)

func ScavengerConcurrency(concurrency int) scavengerOpt {
	return func(o *scavengerOptions) {
		o.concurrency = concurrency
	}
}

func ScavengerAccountsPeekSize(peekSize int) scavengerOpt {
	return func(o *scavengerOptions) {
		o.accountsPeekSize = peekSize
	}
}

func ScavengerLeasesPeekSize(peekSize int) scavengerOpt {
	return func(o *scavengerOptions) {
		o.leasesPeekSize = peekSize
	}
}

func (r *redisCapacityManager) Scavenge(ctx context.Context, options ...scavengerOpt) errs.InternalError {
	o := &scavengerOptions{}
	for _, so := range options {
		so(o)
	}

	if o.concurrency == 0 {
		o.concurrency = defaultScavengerConcurrency
	}

	if o.accountsPeekSize == 0 {
		o.accountsPeekSize = defaultScavengerAccountsPeekSize
	}

	if o.leasesPeekSize == 0 {
		o.leasesPeekSize = defaultScavengerLeasesPeekSize
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(o.concurrency)

	now := r.clock.Now()

	// Scavenge all queue shards
	for k, v := range r.queueShards {
		eg.Go(func() error {
			return r.scavengePrefix(ctx, MigrationIdentifier{QueueShard: k}, v, r.queueStateKeyPrefix, o, now)
		})
	}

	// Scavenge rate limit cluster
	eg.Go(func() error {
		return r.scavengePrefix(ctx, MigrationIdentifier{IsRateLimit: true}, r.rateLimitClient, r.rateLimitKeyPrefix, o, now)
	})

	if err := eg.Wait(); err != nil {
		return errs.Wrap(0, false, "failed to scavenge: %w", err)
	}

	return nil
}

func (r *redisCapacityManager) scavengePrefix(ctx context.Context, mi MigrationIdentifier, client rueidis.Client, keyPrefix string, o *scavengerOptions, now time.Time) error {
	// TODO: Pick random shard
	scavengerShard := 0

	// Peek accounts
	peekedAccounts, err := r.peekScavengerShard(ctx, keyPrefix, client, scavengerShard, o.accountsPeekSize, now)
	if err != nil {
		return fmt.Errorf("could not peek accounts to scavenge expired leases: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(o.concurrency)

	for _, accountID := range peekedAccounts {
		eg.Go(func() error {
			err := r.scavengeAccount(ctx, mi, keyPrefix, client, accountID, o.leasesPeekSize, now)
			if err != nil {
				return fmt.Errorf("could not scavenge account: %w", err)
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("could not scavenge accounts: %w", err)
	}

	return nil
}

func (r *redisCapacityManager) peekScavengerShard(ctx context.Context, keyPrefix string, client rueidis.Client, scavengerShard, peekSize int, now time.Time) ([]uuid.UUID, error) {
	key := r.keyScavengerShard(keyPrefix, scavengerShard)

	// Peek all accounts that have a score < now
	cmd := client.
		B().
		Zrange().
		Key(key).
		Min("-inf").
		// Scores are represented in unix millis
		Max(fmt.Sprintf("%d", now.UnixMilli())).
		Byscore().
		Limit(0, int64(peekSize)).
		Build()

	accountIDs, err := client.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		return nil, fmt.Errorf("error peeking scavenger shard key: %w", err)
	}

	parsedIDs := make([]uuid.UUID, len(accountIDs))
	for i, v := range accountIDs {
		parsedID, err := uuid.Parse(v)
		if err != nil {
			return nil, fmt.Errorf("invalid uuid: %w", err)
		}

		parsedIDs[i] = parsedID
	}

	return parsedIDs, nil
}

func (r *redisCapacityManager) peekExpiredLeases(
	ctx context.Context,
	keyPrefix string,
	client rueidis.Client,
	accountID uuid.UUID,
	peekSize int,
	now time.Time,
) (int64, []ulid.ULID, error) {
	keys := []string{
		r.keyAccountLeases(keyPrefix, accountID),
	}

	args, err := strSlice([]any{
		now.UnixMilli(),
	})
	if err != nil {
		return 0, nil, fmt.Errorf("could not convert args: %w", err)
	}

	peekRes, err := scripts["peek_expired_leases"].Exec(ctx, client, keys, args).ToAny()
	if err != nil {
		return 0, nil, fmt.Errorf("could not execute peek script: %w", err)
	}

	returnTuple, ok := peekRes.([]any)
	if !ok || len(returnTuple) != 2 {
		return 0, nil, fmt.Errorf("response is not a slice: %w", err)
	}

	totalCount, ok := returnTuple[0].(int64)
	if !ok {
		return 0, nil, fmt.Errorf("missing totalCount in returned tuple")
	}

	rawLeaseIDs, ok := returnTuple[1].([]any)
	if !ok {
		return 0, nil, fmt.Errorf("missing lease IDs in returned tuple")
	}

	leaseIDs := make([]ulid.ULID, len(rawLeaseIDs))
	for i, v := range rawLeaseIDs {
		strVal, ok := v.(string)
		if !ok {
			return 0, nil, fmt.Errorf("returned lease is not a string")
		}

		parsed, err := ulid.Parse(strVal)
		if err != nil {
			return 0, nil, fmt.Errorf("could not parse lease ID: %w", err)
		}
		leaseIDs[i] = parsed
	}

	return totalCount, leaseIDs, nil
}

func (r *redisCapacityManager) scavengeAccount(
	ctx context.Context,
	mi MigrationIdentifier,
	keyPrefix string,
	client rueidis.Client,
	accountID uuid.UUID,
	peekSize int,
	now time.Time,
) error {
	totalCount, peekedLeases, err := r.peekExpiredLeases(ctx, keyPrefix, client, accountID, peekSize, now)
	if err != nil {
		return fmt.Errorf("could not peek expired leases: %w", err)
	}

	// TODO: Report total expired leases count (to optimize scavenger peeks if we're not processing fast enough)
	_ = totalCount

	for _, leaseID := range peekedLeases {
		_, err := r.Release(ctx, &CapacityReleaseRequest{
			IdempotencyKey: leaseID.String(),
			AccountID:      accountID,
			LeaseID:        leaseID,
			Migration:      mi,
		})
		if err != nil {
			return fmt.Errorf("could not scavenge expired lease: %w", err)
		}
	}

	return nil
}
