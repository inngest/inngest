package constraintapi

import (
	"context"
	"fmt"

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

	// Scavenge all queue shards
	for k, v := range r.queueShards {
		eg.Go(func() error {
			return r.scavengePrefix(ctx, MigrationIdentifier{QueueShard: k}, v, r.queueStateKeyPrefix, o)
		})
	}

	// Scavenge rate limit cluster
	eg.Go(func() error {
		return r.scavengePrefix(ctx, MigrationIdentifier{IsRateLimit: true}, r.rateLimitClient, r.rateLimitKeyPrefix, o)
	})

	if err := eg.Wait(); err != nil {
		return errs.Wrap(0, false, "failed to scavenge: %w", err)
	}

	return nil
}

func (r *redisCapacityManager) scavengePrefix(ctx context.Context, mi MigrationIdentifier, client rueidis.Client, keyPrefix string, o *scavengerOptions) error {
	// TODO: Pick random shard
	scavengerShard := 0

	// Peek accounts
	peekedAccounts, err := r.peekScavengerShard(ctx, keyPrefix, client, scavengerShard, o.accountsPeekSize)
	if err != nil {
		return fmt.Errorf("could not peek accounts to scavenge expired leases: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(o.concurrency)

	for _, accountID := range peekedAccounts {
		eg.Go(func() error {
			err := r.scavengeAccount(ctx, mi, accountID, o.leasesPeekSize)
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

func (r *redisCapacityManager) peekScavengerShard(ctx context.Context, keyPrefix string, client rueidis.Client, scavengerShard, peekSize int) ([]uuid.UUID, error) {
	key := r.keyScavengerShard(keyPrefix, scavengerShard)

	// Scores are represented in unix millis
	nowMS := r.clock.Now().UnixMilli()
	now := fmt.Sprintf("%d", nowMS)

	// Peek all accounts that have a score < now
	cmd := client.
		B().
		Zrange().
		Key(key).
		Min("-inf").
		Max(now).
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

type peekedLease struct {
	LeaseIdempotencyKey string
	LeaseID             ulid.ULID
}

func peekExpiredLeases(ctx context.Context, keyPrefix string, client rueidis.Client, accountID uuid.UUID, peekSize int) ([]peekedLease, error) {
}

func (r *redisCapacityManager) scavengeAccount(ctx context.Context, mi MigrationIdentifier, accountID uuid.UUID, peekSize int) error {
	// TODO: Peek lease idempotency key + lease ID
	peekedLeaseIdempotencyKeys := []struct {
		leaseIdempotencyKey string
		leaseID             ulid.ULID
	}{}

	for _, v := range peekedLeaseIdempotencyKeys {
		_, err := r.Release(ctx, &CapacityReleaseRequest{
			IdempotencyKey:      v.leaseID.String(),
			AccountID:           accountID,
			LeaseIdempotencyKey: v.leaseIdempotencyKey,
			LeaseID:             v.leaseID,
			Migration:           mi,
		})
		if err != nil {
			return fmt.Errorf("could not scavenge expired lease: %w", err)
		}
	}

	return nil
}
