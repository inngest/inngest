package constraintapi

import (
	"context"
	"fmt"
	"hash/crc32"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"golang.org/x/sync/errgroup"
)

const (
	pkgName = "constraintapi.scavenger"

	defaultScavengerAccountsPeekSize = 20
	defaultScavengerLeasesPeekSize   = 20
	defaultScavengerConcurrency      = 20
)

type CapacityLeaseScavenger interface {
	Scavenge(ctx context.Context, options ...scavengerOpt) (*ScavengeResult, errs.InternalError)
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

type ScavengeResult struct {
	TotalExpiredLeasesCount   int
	TotalExpiredAccountsCount int
	TotalAccountsCount        int
	ReclaimedLeases           int
	ScannedAccounts           int
}

// scavengerShard deterministically retrieves a shard number based on numScavengerShards and accountID
func (r *redisCapacityManager) scavengerShard(ctx context.Context, accountID uuid.UUID) int {
	hash := crc32.ChecksumIEEE([]byte(accountID.String()))
	return int(hash) % r.numScavengerShards
}

func (r *redisCapacityManager) Scavenge(ctx context.Context, options ...scavengerOpt) (*ScavengeResult, errs.InternalError) {
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

	result := &ScavengeResult{}
	resLock := sync.Mutex{}

	// Scavenge all queue shards
	for k, v := range r.queueShards {
		eg.Go(func() error {
			res, err := r.scavengePrefix(ctx, MigrationIdentifier{QueueShard: k}, v, r.queueStateKeyPrefix, o, now)
			if err != nil {
				return fmt.Errorf("could not scavenge expired leases for queue shard: %w", err)
			}

			resLock.Lock()
			result.ReclaimedLeases += res.ReclaimedLeases
			result.TotalAccountsCount += res.TotalAccountsCount
			result.TotalExpiredAccountsCount += res.TotalExpiredAccountsCount
			result.TotalExpiredLeasesCount += res.TotalExpiredLeasesCount
			result.ScannedAccounts += res.ScannedAccounts
			resLock.Unlock()
			return nil
		})
	}

	// Scavenge rate limit cluster
	eg.Go(func() error {
		res, err := r.scavengePrefix(ctx, MigrationIdentifier{IsRateLimit: true}, r.rateLimitClient, r.rateLimitKeyPrefix, o, now)
		if err != nil {
			return fmt.Errorf("could not scavenge rate limit: %w", err)
		}

		resLock.Lock()
		result.ReclaimedLeases += res.ReclaimedLeases
		result.TotalAccountsCount += res.TotalAccountsCount
		result.TotalExpiredAccountsCount += res.TotalExpiredAccountsCount
		result.TotalExpiredLeasesCount += res.TotalExpiredLeasesCount
		result.ScannedAccounts += res.ScannedAccounts
		resLock.Unlock()
		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, errs.Wrap(0, false, "failed to scavenge: %w", err)
	}

	return result, nil
}

func (r *redisCapacityManager) scavengePrefix(ctx context.Context, mi MigrationIdentifier, client rueidis.Client, keyPrefix string, o *scavengerOptions, now time.Time) (*ScavengeResult, error) {
	result := &ScavengeResult{}
	resLock := sync.Mutex{}

	// Iterate over shards
	// TODO: We could also do a random shard strategy
	for scavengerShard := range r.numScavengerShards {
		res, err := r.scavengeShard(ctx, mi, client, keyPrefix, scavengerShard, now, o)
		if err != nil {
			return nil, fmt.Errorf("could not scavenge shard %d: %w", scavengerShard, err)
		}

		resLock.Lock()
		result.TotalAccountsCount += res.TotalAccountsCount
		result.TotalExpiredAccountsCount += res.TotalExpiredAccountsCount
		result.ScannedAccounts += res.ScannedAccounts
		result.TotalExpiredLeasesCount += res.TotalExpiredLeasesCount
		result.ReclaimedLeases += res.ReclaimedLeases
		resLock.Unlock()
	}

	return result, nil
}

func (r *redisCapacityManager) scavengeShard(ctx context.Context, mi MigrationIdentifier, client rueidis.Client, keyPrefix string, scavengerShard int, now time.Time, o *scavengerOptions) (*ScavengeResult, error) {
	start := time.Now()
	defer func() {
		metrics.HistogramConstraintAPIScavengerShardProcessDuration(ctx, time.Since(start), metrics.HistogramOpt{
			PkgName: pkgName,
			Tags:    map[string]any{},
		})
	}()

	// Peek accounts
	res, err := r.peekScavengerShard(ctx, keyPrefix, client, scavengerShard, o.accountsPeekSize, now)
	if err != nil {
		return nil, fmt.Errorf("could not peek accounts to scavenge expired leases: %w", err)
	}

	result := &ScavengeResult{
		TotalExpiredAccountsCount: res.expired,
		TotalAccountsCount:        res.total,
		ScannedAccounts:           res.peeked,
	}
	resLock := sync.Mutex{}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(o.concurrency)

	for _, accountID := range res.accountIDs {
		eg.Go(func() error {
			res, err := r.scavengeAccount(
				ctx,
				mi,
				keyPrefix,
				client,
				accountID,
				o.leasesPeekSize,
				now,
			)
			if err != nil {
				return fmt.Errorf("could not scavenge account: %w", err)
			}

			resLock.Lock()
			result.TotalExpiredLeasesCount += res.TotalExpiredLeasesCount
			result.ReclaimedLeases += res.ReclaimedLeases
			resLock.Unlock()

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, fmt.Errorf("could not scavenge accounts: %w", err)
	}

	return result, nil
}

type scavengerShardPeekResult struct {
	total      int
	expired    int
	peeked     int
	accountIDs []uuid.UUID
}

func (r *redisCapacityManager) peekScavengerShard(
	ctx context.Context,
	keyPrefix string,
	client rueidis.Client,
	scavengerShard,
	peekSize int,
	now time.Time,
) (*scavengerShardPeekResult, error) {
	key := r.keyScavengerShard(keyPrefix, scavengerShard)

	cmd := client.
		B().
		Zcard().
		Key(key).
		Build()

	total, err := client.Do(ctx, cmd).ToInt64()
	if err != nil {
		return nil, fmt.Errorf("error peeking total accounts in scavenger shard: %w", err)
	}

	cmd = client.
		B().
		Zcount().
		Key(key).
		Min("-inf").
		// Scores are represented in unix millis
		Max(fmt.Sprintf("%d", now.UnixMilli())).
		Build()

	expired, err := client.Do(ctx, cmd).ToInt64()
	if err != nil {
		return nil, fmt.Errorf("error peeking total accounts with expired leases in scavenger shard: %w", err)
	}

	// Peek all accounts that have a score < now
	cmd = client.
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

	return &scavengerShardPeekResult{
		total:      int(total),
		expired:    int(expired),
		peeked:     len(parsedIDs),
		accountIDs: parsedIDs,
	}, nil
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
		peekSize,
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
) (*ScavengeResult, error) {
	totalCount, peekedLeases, err := r.peekExpiredLeases(
		ctx,
		keyPrefix,
		client,
		accountID,
		peekSize,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("could not peek expired leases: %w", err)
	}

	for _, leaseID := range peekedLeases {
		leaseAge := now.Sub(leaseID.Timestamp())
		metrics.HistogramConstraintAPIScavengerLeaseAge(ctx, leaseAge, metrics.HistogramOpt{
			PkgName: pkgName,
			Tags:    map[string]any{},
		})

		_, err := r.Release(ctx, &CapacityReleaseRequest{
			IdempotencyKey: leaseID.String(),
			AccountID:      accountID,
			LeaseID:        leaseID,
			Migration:      mi,
		})
		if err != nil {
			return nil, fmt.Errorf("could not scavenge expired lease: %w", err)
		}

	}

	return &ScavengeResult{
		TotalExpiredLeasesCount: int(totalCount),
		ReclaimedLeases:         len(peekedLeases),
	}, nil
}

type scavengerService struct {
	cm       CapacityLeaseScavenger
	interval time.Duration
	opt      []scavengerOpt
}

func (s *scavengerService) Name() string {
	return "lease-scavenger"
}

func (s *scavengerService) Pre(ctx context.Context) error {
	return nil
}

func (s *scavengerService) Run(ctx context.Context) error {
	l := logger.StdlibLogger(ctx).With(
		"service", "lease-scavenger",
	)

	t := time.Tick(s.interval)

	for {
		select {
		case <-ctx.Done():
			l.Info("context canceled, stopping scavenger loop")
			return nil
		case <-t:
		}

		res, err := s.cm.Scavenge(ctx, s.opt...)
		if err != nil {
			l.Error("scavenging expired leases failed", "err", err)
			continue
		}

		metrics.IncrConstraintAPIScavengerTotalAccountsCounter(ctx, int64(res.TotalAccountsCount), metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{},
		})
		metrics.IncrConstraintAPIScavengerExpiredAccountsCounter(ctx, int64(res.TotalExpiredAccountsCount), metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{},
		})
		metrics.IncrConstraintAPIScavengerScannedAccountsCounter(ctx, int64(res.ScannedAccounts), metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{},
		})
		metrics.IncrConstraintAPIScavengerTotalExpiredLeasesCounter(ctx, int64(res.TotalExpiredLeasesCount), metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{},
		})
		metrics.IncrConstraintAPIScavengerReclaimedLeasesCounter(ctx, int64(res.ReclaimedLeases), metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{},
		})

		l := l.With(
			"total_accounts", res.TotalAccountsCount,
			"expired_accounts", res.TotalExpiredAccountsCount,
			"scanned_accounts", res.ScannedAccounts,
			"expired_leases", res.TotalExpiredLeasesCount,
			"reclaimed_leases", res.ReclaimedLeases,
		)
		if res.ReclaimedLeases > 0 {
			l.Debug("scavenger tick completed")
		} else {
			l.Trace("scavenger tick completed")
		}
	}
}

func (s *scavengerService) Stop(ctx context.Context) error {
	return nil
}

func NewLeaseScavengerService(
	cm CapacityLeaseScavenger,
	interval time.Duration,
	opts ...scavengerOpt,
) service.Service {
	return &scavengerService{
		cm:       cm,
		opt:      opts,
		interval: interval,
	}
}
