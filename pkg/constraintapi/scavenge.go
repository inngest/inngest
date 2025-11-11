package constraintapi

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/oklog/ulid/v2"
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

	eg.Go(func() error {
		return r.scavengePrefix(ctx, false, r.queueStateKeyPrefix, o)
	})
	eg.Go(func() error {
		return r.scavengePrefix(ctx, true, r.rateLimitKeyPrefix, o)
	})

	if err := eg.Wait(); err != nil {
		return errs.Wrap(0, false, "failed to scavenge: %w", err)
	}

	return nil
}

func (r *redisCapacityManager) scavengePrefix(ctx context.Context, isRateLimit bool, keyPrefix string, o *scavengerOptions) error {
	// TODO: Pick random shard
	scavengerShard := 0 // scavengerShard placeholder

	// TODO: Peek accounts
	peekedAccounts, err := r.peekScavengerShard(ctx, keyPrefix, scavengerShard, o.accountsPeekSize)
	if err != nil {
		return fmt.Errorf("could not peek accounts to scavenge expired leases: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(o.concurrency)

	for _, accountID := range peekedAccounts {
		eg.Go(func() error {
			err := r.scavengeAccount(ctx, isRateLimit, accountID, o.leasesPeekSize)
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

func (r *redisCapacityManager) peekScavengerShard(ctx context.Context, keyPrefix string, scavengerShard, peekSize int) ([]uuid.UUID, error) {
	// TODO: Implement scavenger shard peeking
	return nil, nil
}

func (r *redisCapacityManager) scavengeAccount(ctx context.Context, isRateLimit bool, accountID uuid.UUID, peekSize int) error {
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
			Migration: MigrationIdentifier{
				IsRateLimit: isRateLimit,
			},
		})
		if err != nil {
			return fmt.Errorf("could not scavenge expired lease: %w", err)
		}
	}

	return nil
}
