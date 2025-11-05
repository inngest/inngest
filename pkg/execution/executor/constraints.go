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
	"github.com/inngest/inngest/pkg/logger"
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
	capacityManager constraintapi.CapacityManager,
	useConstraintAPI constraintapi.UseConstraintAPIFn,
	req execution.ScheduleRequest,
	idempotencyKey string,
	fn func(
		ctx context.Context,
		// performChecks determines whether constraint checks must be performed
		// This may be false when the Constraint API was used to enforce constraints.
		performChecks bool,
		// fallbackIdempotencyKey may be defined when the Constraint API Acquire request
		// failed (and we don't know if it succeeded on the API)
		fallbackIdempotencyKey string,
	) (T, error),
) (T, error) {
	l := logger.StdlibLogger(ctx)

	var zero T
	// Cancel context on return
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// If capacity manager / feature flag are not passed, execute Schedule code
	// with existing constraint checks
	if capacityManager == nil || useConstraintAPI == nil {
		return fn(ctx, true, "")
	}

	// Read feature flag
	enable, fallback := useConstraintAPI(ctx, req.AccountID)
	if !enable {
		// If feature flag is disabled, execute Schedule code with existing constraint checks

		return fn(ctx, true, "")
	}

	constraints, err := getScheduleConstraints(ctx, req)
	if err != nil {
		l.Error("failed to get schedule constraints", "err", err)
		return fn(ctx, true, "")
	}

	// Perform constraint check to acquire lease
	checkResult, err := CheckConstraints(
		ctx,
		capacityManager,
		useConstraintAPI,
		req,
		idempotencyKey,
		fallback,
		constraints,
	)
	if err != nil {
		l.Error("failed to check constraints", "err", err)
		return fn(ctx, true, "")
	}

	// If the Constraint API didn't successfully return, call the user function and indicate checks should run
	if checkResult.mustCheck {
		return fn(ctx, true, checkResult.fallbackIdempotencyKey)
	}

	// If the current action is not allowed, return
	if !checkResult.allowed {
		// TODO : should we record this?

		// NOTE: Since Schedule only enforces RateLimit via the Constraint API, we know that
		// we got rate limited if the action is not allowed.
		return zero, ErrFunctionRateLimited
	}

	userCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// If no lease was provided, we are not allowed to process
	if checkResult.leaseID == nil {
		// TODO: When does this happen?
		l.Warn("acquire request did not return lease ID")

		// Pretend the API request failed
		return fn(ctx, true, "")
	}

	leaseID := checkResult.leaseID
	leaseIDLock := sync.Mutex{}

	go func() {
		for {
			select {
			// Stop extending lease
			case <-ctx.Done():
				return
			case <-time.After(ScheduleLeaseExtension):
			}

			leaseIDLock.Lock()
			if leaseID == nil {
				l.Warn("no leaseID, canceling context")
				cancel()
				return
			}
			lID := *leaseID
			leaseIDLock.Unlock()

			// Use previous lease as idempotency key
			// This works because each lease is expected to extend once, after which a new lease
			// is generated. This means idempotency can be used for graceful retries.
			idempotencyKey := lID.String()

			res, err := capacityManager.ExtendLease(ctx, &constraintapi.CapacityExtendLeaseRequest{
				IdempotencyKey: idempotencyKey,
				AccountID:      req.AccountID,
				LeaseID:        lID,
			})
			if err != nil {
				l.Error("could not extend schedule capacity lease", "err", err)
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

	defer func() {
		leaseIDLock.Lock()
		defer leaseIDLock.Unlock()

		if leaseID != nil {
			// Release capacity in a non-blocking call.
			//
			// All leases are guaranteed to be released once expired,
			// which means calling Release early is an optimization
			// to hand back capacity as soon as possible, but not strictly
			// required.
			lID := *leaseID

			// Use previous lease as idempotency key
			idempotencyKey := lID.String()

			go func() {
				_, internalErr := capacityManager.Release(context.Background(), &constraintapi.CapacityReleaseRequest{
					AccountID:      req.AccountID,
					LeaseID:        lID,
					IdempotencyKey: idempotencyKey,
				})
				if internalErr != nil {
					l.Error("failed to release capacity", "err", internalErr)
				}
			}()
		}
	}()

	// Run user code with lease guarantee
	// NOTE: The passed context will be canceled if the lease expires.
	return fn(userCtx, false, "")
}

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

func getScheduleConstraints(ctx context.Context, req execution.ScheduleRequest) ([]constraintapi.ConstraintItem, error) {
	var requests []constraintapi.ConstraintItem

	// The only constraint we care about in run scheduling is rate limiting.
	// Throttle + concurrency constraints are checked in the queue.
	if req.Function.RateLimit != nil {
		var rateLimitKeyExpr string
		var rateLimitKey string
		if req.Function.RateLimit.Key != nil {
			rateLimitKeyExpr = *req.Function.RateLimit.Key
			key, err := ratelimit.RateLimitKey(ctx, req.Function.ID, *req.Function.RateLimit, req.Events[0].GetEvent().Map())
			if err != nil {
				return nil, fmt.Errorf("could not get rate limit key: %w", err)
			}
			rateLimitKey = key
		}

		requests = append(requests, constraintapi.ConstraintItem{
			Kind: constraintapi.ConstraintKindRateLimit,
			RateLimit: &constraintapi.RateLimitConstraint{
				Scope:             enums.RateLimitScopeFn,
				KeyExpressionHash: util.XXHash(rateLimitKeyExpr),
				EvaluatedKeyHash:  rateLimitKey,
			},
		})
	}

	return requests, nil
}

func CheckConstraints(
	ctx context.Context,
	capacityManager constraintapi.CapacityManager,
	useConstraintAPI constraintapi.UseConstraintAPIFn,
	req execution.ScheduleRequest,
	idempotencyKey string,
	fallback bool,
	constraints []constraintapi.ConstraintItem,
) (checkResult, error) {
	l := logger.StdlibLogger(ctx)

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

	// TODO: Fetch account concurrency
	var accountConcurrency int

	res, internalErr := capacityManager.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
		AccountID:            req.AccountID,
		IdempotencyKey:       idempotencyKey,
		LeaseIdempotencyKeys: []string{idempotencyKey},
		EnvID:                req.WorkspaceID,
		FunctionID:           req.Function.ID,
		Configuration:        queue.ConvertToConstraintConfiguration(accountConcurrency, req.Function),
		Constraints:          constraints,
		Amount:               1,
		CurrentTime:          time.Now(),
		Duration:             ScheduleLeaseDuration,
		MaximumLifetime:      5 * time.Minute, // This lease should be short!
		Source:               source,
		BlockingThreshold:    0, // Disable this for now
	})
	if internalErr != nil {
		l.Error("acquiring capacity lease failed", "err", internalErr)

		if fallback {
			return checkResult{
				mustCheck:              true,
				fallbackIdempotencyKey: idempotencyKey,
			}, nil
		}
		return checkResult{}, fmt.Errorf("could not enforce constraints: %w", internalErr)
	}

	// Rate limited
	allowed := len(res.Leases) == 1
	if !allowed {
		return checkResult{
			allowed:   false,
			mustCheck: false,
		}, nil
	}

	lease := res.Leases[0]

	return checkResult{
		allowed: true,

		leaseID:                &lease.LeaseID,
		fallbackIdempotencyKey: lease.IdempotencyKey,

		// We already checked constraints
		mustCheck: false,
	}, nil
}
