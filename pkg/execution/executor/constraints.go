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
	capacityManager constraintapi.CapacityManager,
	useConstraintAPI constraintapi.UseConstraintAPIFn,
	req execution.ScheduleRequest,
	fn func(ctx context.Context, performChecks bool, fallbackIdempotencyKey string) (T, error),
) (T, error) {
	var zero T
	// Cancel context on return
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// If capacity manager / feature flag are not passed, execute Schedule code
	// with existing constraint checks
	if capacityManager == nil || useConstraintAPI == nil {
		res, err := fn(ctx, false, "")
		return res, err
	}

	// Read feature flag
	enable, fallback := useConstraintAPI(ctx, req.AccountID)
	if !enable {
		// If feature flag is disabled, execute Schedule code with existing constraint checks

		res, err := fn(ctx, false, "")
		return res, err
	}

	constraints, err := getScheduleConstraints(ctx, req)
	if err != nil {
		return zero, fmt.Errorf("could not get schedule constraints: %w", err)
	}

	// Perform constraint check to acquire lease
	checkResult, err := CheckConstraints(
		ctx,
		capacityManager,
		useConstraintAPI,
		req,
		fallback,
		constraints,
	)
	if err != nil {
		return zero, fmt.Errorf("failed to check constraints: %w", err)
	}

	// If the Constraint API didn't successfully return, call the user function and indicate checks should run
	if checkResult.mustCheck {
		res, err := fn(ctx, true, checkResult.fallbackIdempotencyKey)
		return res, err
	}

	// If the current action is not allowed, return
	if !checkResult.allowed {
		// TODO: Figure out which constraint was lacking (we only check rate limits so we can assume that)

		// TODO : should we record this?
		return zero, nil
	}

	userCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// If no lease was provided, we are not allowed to process
	if checkResult.leaseID == nil {
		// TODO: When does this happen?
		return zero, fmt.Errorf("failed to acquire lease: %w", err)
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

			res, err := capacityManager.ExtendLease(ctx, &constraintapi.CapacityExtendLeaseRequest{
				// TODO: Generate idempotency key.
				IdempotencyKey: "",
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

	// Run user code with lease guarantee
	// NOTE: The passed context will be canceled if the lease expires.
	res, err := fn(userCtx, false, "")

	if checkResult.leaseID != nil {
		_, internalErr := capacityManager.Release(ctx, &constraintapi.CapacityReleaseRequest{
			AccountID: req.AccountID,
			LeaseID:   *checkResult.leaseID,
			// TODO: Generate idempotency key
			IdempotencyKey: "",
		})
		if internalErr != nil {
			// TODO Handle internal err
			_ = internalErr
		}

	}

	// TODO Handle error?
	if err != nil {
		return zero, err
	}

	return res, nil
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

func getScheduleConstraints(ctx context.Context, req execution.ScheduleRequest) ([]constraintapi.ConstraintCapacityItem, error) {
	var requests []constraintapi.ConstraintCapacityItem

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

	return requests, nil
}

func CheckConstraints(
	ctx context.Context,
	capacityManager constraintapi.CapacityManager,
	useConstraintAPI constraintapi.UseConstraintAPIFn,
	req execution.ScheduleRequest,
	fallback bool,
	constraints []constraintapi.ConstraintCapacityItem,
) (checkResult, error) {
	// Retrieve idempotency key to acquire lease
	// NOTE: To allow for retries between multiple executors, this should be
	// consistent between calls to CheckConstraints
	var idempotencyKey string
	if req.IdempotencyKey != nil {
		idempotencyKey = *req.IdempotencyKey
	}

	// TODO: Handle missing idempotency key

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
		AccountID:         req.AccountID,
		IdempotencyKey:    idempotencyKey,
		EnvID:             req.WorkspaceID,
		FunctionID:        req.Function.ID,
		Configuration:     queue.ConvertToConstraintConfiguration(accountConcurrency, req.Function),
		RequestedCapacity: constraints,
		CurrentTime:       time.Now(),
		Duration:          ScheduleLeaseDuration,
		MaximumLifetime:   5 * time.Minute, // This lease should be short!
		Source:            source,
		BlockingThreshold: 0, // Disable this for now
	})
	if internalErr != nil {
		if fallback {
			// TODO: Log error
			return checkResult{
				mustCheck:              true,
				fallbackIdempotencyKey: res.FallbackIdempotencyKey,
			}, nil
		}
		return checkResult{}, fmt.Errorf("could not enforce constraints: %w", internalErr)
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
