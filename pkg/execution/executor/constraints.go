package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/ratelimit"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
)

const (
	// ScheduleLeaseDuration determines the duration for holding on to the constraint capacity before it is rolled back.
	// This should cover the happy path without requiring lots of extensions while being as short as possible.
	ScheduleLeaseDuration = 20 * time.Second

	ScheduleLeaseExtension = 5 * time.Second
)

func WithConstraints[T any](
	ctx context.Context,
	idempotencyKey string,
	capacityManager constraintapi.CapacityManager,
	useConstraintAPI constraintapi.UseConstraintAPIFn,
	req execution.ScheduleRequest,
	fn func(ctx context.Context) (T, ConstraintAction, error),
) (*T, error) {
	// Cancel context on return
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Perform constraint check to acquire lease
	checkResult, err := CheckConstraints(ctx, capacityManager, useConstraintAPI, req, idempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check constraints: %w", err)
	}

	// If the current action is not allowed, return
	if !checkResult.allowed {
		// TODO: Figure out which constraint was lacking (we only check rate limits so we can assume that)

		// TODO : should we record this?
		return nil, nil
	}

	if checkResult.mustCheck {
		// TODO : check rate limits here
	}

	userCtx, cancel := context.WithCancel(ctx)

	// If no lease was provided, we are not allowed to process
	if checkResult.leaseID == nil {
		// TODO: When does this happen?
		return nil, fmt.Errorf("failed to acquire lease: %w", err)
	}

	leaseID := checkResult.leaseID
	leaseIDLock := sync.Mutex{}

	// TODO: Extend lease while we're processing this function (until we return or commit/rollback)
	go func() {
		for {
			select {
			// Stop extending lease
			case <-ctx.Done():
				// TODO: Should we do a best-effort rollback here lease is still active?
				return
			case <-time.After(ScheduleLeaseExtension):
			}

			leaseIDLock.Lock()
			if leaseID == nil {
				// TODO: Warn here
				cancel()
				return
			}
			lID := *leaseID
			leaseIDLock.Unlock()

			res, _, err := capacityManager.ExtendLease(ctx, &constraintapi.CapacityExtendLeaseRequest{
				IdempotencyKey: idempotencyKey, // TODO: Do we need a new idempotency key here?
				AccountID:      req.AccountID,
				LeaseID:        lID,
			})
			if err != nil {
				// TODO: Log here
				continue
			}

			// If extension did not provide new lease, stop processing
			if res.LeaseID == nil {
				cancel()
				return
			}

			leaseIDLock.Lock()
			leaseID = res.LeaseID
			leaseIDLock.Unlock()
		}
	}()

	// NOTE: How do we properly handle rollbacks/commits?

	// Run user code with lease guarantee
	// NOTE: The passed context will be canceled if the lease expires.
	res, action, err := fn(userCtx)

	if checkResult.leaseID != nil {
		switch action {
		case ConstraintRollback:

			_, userErr, internalErr := capacityManager.Rollback(ctx, &constraintapi.CapacityRollbackRequest{
				AccountID:      req.AccountID,
				LeaseID:        *checkResult.leaseID,
				IdempotencyKey: idempotencyKey,
			})
			if internalErr != nil {
				// TODO Handle internal err
			}

			if userErr != nil {
				// TODO handle user err
			}
		case ConstraintCommit:
			_, userErr, internalErr := capacityManager.Commit(ctx, &constraintapi.CapacityCommitRequest{
				AccountID:      req.AccountID,
				LeaseID:        *checkResult.leaseID,
				IdempotencyKey: idempotencyKey,
			})
			if internalErr != nil {
				// TODO Handle internal err
			}

			if userErr != nil {
				// TODO handle user err
			}

		}
	}

	// TODO Handle error?
	if err != nil {
		return nil, err
	}

	return &res, nil
}

type ConstraintAction int

const (
	ConstraintRollback ConstraintAction = iota
	ConstraintCommit
)

type checkResult struct {
	// allowed determines whether a run can be scheduled
	allowed bool

	// leaseID is the current capacity lease which MUST be committed once done or rolled back on error
	leaseID *ulid.ULID

	// mustCheck instructs the caller to perform constraint checks (rate limit)
	mustCheck bool

	// fallbackIdempotencyKey is the idempotency key that MUST be provided to further constraint checks in case of fallbacks
	fallbackIdempotencyKey string
}

func CheckConstraints(
	ctx context.Context,
	capacityManager constraintapi.CapacityManager,
	useConstraintAPI constraintapi.UseConstraintAPIFn,
	req execution.ScheduleRequest,
	idempotencyKey string,
) (checkResult, error) {
	if capacityManager == nil {
		return checkResult{
			mustCheck: true,
		}, nil
	}

	if useConstraintAPI == nil {
		return checkResult{
			mustCheck: true,
		}, nil
	}

	// Read feature flag
	enable, fallback := useConstraintAPI(ctx, req.AccountID)
	if !enable {
		return checkResult{
			mustCheck: true,
		}, nil
	}

	// TODO: Handle missing idempotency key

	// TODO: Fetch account concurrency
	var accountConcurrency int

	// NOTE: Schedule may be called from within new-runs or the API
	// In case of the API, we want to ensure the source is properly reflected in constraint checks
	// to enforce fairness between callers
	source := constraintapi.LeaseSource{
		Service:           constraintapi.ServiceExecutor,
		RunProcessingMode: constraintapi.RunProcessingModeBackground,
		Location:          constraintapi.LeaseLocationScheduleRun,
	}
	if req.RunMode == enums.RunModeSync {
		source.Service = constraintapi.ServiceAPI
		source.RunProcessingMode = constraintapi.RunProcessingModeSync
		source.Location = constraintapi.LeaseLocationCheckpoint
	}

	var requests []constraintapi.ConstraintCapacityItem

	// The only constraint we care about in run scheduling is rate limiting. Throttle + concurrency constraints are checked in the queue.
	if req.Function.RateLimit != nil {
		var rateLimitKeyExpr string
		var rateLimitKey string
		if req.Function.RateLimit.Key != nil {
			rateLimitKeyExpr = *req.Function.RateLimit.Key
			key, err := ratelimit.RateLimitKey(ctx, req.Function.ID, *req.Function.RateLimit, req.Events[0].GetEvent().Map())
			if err != nil {
				// TODO: Handle error
			}
			rateLimitKey = key
		}

		requests = append(requests, constraintapi.ConstraintCapacityItem{
			Kind: constraintapi.CapacityKindRateLimit,
			RateLimit: &constraintapi.RateLimitCapacity{
				Scope:             enums.RateLimitScopeFn,
				KeyExpressionHash: util.XXHash(rateLimitKeyExpr),
				EvaluatedKeyHash:  rateLimitKey,
			},
			Amount: 1,
		})
	}

	// TODO: If no constraints defined, do not call up constraint API

	res, userErr, internalErr := capacityManager.Lease(ctx, &constraintapi.CapacityLeaseRequest{
		AccountID:         req.AccountID,
		IdempotencyKey:    idempotencyKey,
		EnvID:             req.WorkspaceID,
		FunctionID:        req.Function.ID,
		Configuration:     queue.ConvertToConstraintConfiguration(accountConcurrency, req.Function),
		RequestedCapacity: requests,
		CurrentTime:       time.Now(),
		Duration:          ScheduleLeaseDuration,
		MaximumLifetime:   5 * time.Minute, // This lease should be short!
		Source:            source,
		BlockingThreshold: 0, // Disable this for now
	})
	if internalErr != nil {
		if fallback {
			// TODO: Handle fallback and supply res.FallbackIdempotencyKey
			return checkResult{}, fmt.Errorf("fallback not implemented")
		}
		return checkResult{}, fmt.Errorf("could not enforce constraints: %w", internalErr)
	}

	if userErr != nil {
		if fallback {
			// TODO: Handle fallback and supply res.FallbackIdempotencyKey
			return checkResult{}, fmt.Errorf("fallback not implemented")
		}
		return checkResult{}, fmt.Errorf("could not enforce constraints: %w", userErr)
	}

	// TODO: Do we need to add more fine-grained checks here?
	allowed := len(res.InsufficientCapacity) == 0

	return checkResult{
		allowed: allowed,

		leaseID:                res.LeaseID,
		fallbackIdempotencyKey: res.FallbackIdempotencyKey,

		// We already checked constraints
		mustCheck: false,
	}, nil
}
