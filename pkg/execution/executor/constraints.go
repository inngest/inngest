package executor

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/ratelimit"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
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
	now time.Time,
	capacityManager constraintapi.CapacityManager,
	useConstraintAPI constraintapi.UseConstraintAPIFn,
	req execution.ScheduleRequest,
	idempotencyKey string,
	fn func(
		ctx context.Context,
		// performChecks determines whether constraint checks must be performed
		// This may be false when the Constraint API was used to enforce constraints.
		performChecks bool,
	) (T, error),
) (T, error) {
	l := logger.StdlibLogger(ctx)

	var zero T
	// Cancel context on return
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	start := time.Now()

	// If capacity manager / feature flag are not passed, execute Schedule code
	// with existing constraint checks
	if capacityManager == nil || useConstraintAPI == nil {
		metrics.IncrScheduleConstraintsCheckCounter(ctx, enums.ScheduleConstraintCheckReasonConstraintAPIUninitialized.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return fn(ctx, true)
	}

	// Read feature flag
	enable := useConstraintAPI(ctx, req.AccountID)

	defer func() {
		metrics.HistogramScheduleDuration(ctx, time.Since(start).Milliseconds(), metrics.HistogramOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"constraint_api": enable,
			},
		})
	}()

	if !enable {
		// If feature flag is disabled, execute Schedule code with existing constraint checks
		metrics.IncrScheduleConstraintsCheckCounter(ctx, enums.ScheduleConstraintCheckReasonFeatureFlagDisabled.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return fn(ctx, true)
	}

	constraints, err := getScheduleConstraints(ctx, req)
	if err != nil {
		l.Error("failed to get schedule constraints", "err", err)
		metrics.IncrScheduleConstraintsCheckCounter(ctx, enums.ScheduleConstraintCheckReasonGetConstraintsError.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return zero, fmt.Errorf("could not get constraints for schedule: %w", err)
	}

	// If no rate limits are configured, simply run the function
	if len(constraints) == 0 {
		metrics.IncrScheduleConstraintsCheckCounter(ctx, enums.ScheduleConstraintCheckReasonNoRateLimitConfigured.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return fn(ctx, false)
	}

	// Perform constraint check to acquire lease
	checkResult, err := CheckConstraints(
		ctx,
		now,
		capacityManager,
		useConstraintAPI,
		req,
		idempotencyKey,
		constraints,
	)
	if err != nil {
		l.Error("failed to check constraints", "err", err)
		metrics.IncrScheduleConstraintsCheckCounter(ctx, enums.ScheduleConstraintCheckReasonConstraintAPIError.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return zero, err
	}

	l.Optional(req.AccountID, "schedule").Debug(
		"schedule: got constraints",
		"account_id", req.AccountID,
		"env_id", req.WorkspaceID,
		"app_id", req.AppID,
		"fn_id", req.Function.ID,
		"fn_v", req.Function.FunctionVersion,
		"evt_id", req.Events[0].GetInternalID(),
		"constraints", constraints,
		"results", checkResult,
	)

	// If the current action is not allowed, return
	if !checkResult.allowed {
		metrics.IncrScheduleConstraintsHitCounter(ctx, "rate_limit", metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"constraint_api": true,
			},
		})

		// NOTE: Since Schedule only enforces RateLimit via the Constraint API, we know that
		// we got rate limited if the action is not allowed.
		return zero, ErrFunctionRateLimited
	}

	userCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// If no lease was provided, we are not allowed to process
	if checkResult.leaseID == nil {
		l.ReportError(errors.New("acquire request was allowed but did not return lease ID"), "acquire request was allowed but did not return lease ID")
		metrics.IncrScheduleConstraintsCheckCounter(ctx, enums.ScheduleConstraintCheckReasonMissingLease.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return zero, fmt.Errorf("constraint API did not return lease ID")
	}

	leaseID := checkResult.leaseID
	leaseIDLock := sync.Mutex{}

	source := constraintapi.LeaseSource{
		Service:           constraintapi.ServiceExecutor,
		RunProcessingMode: constraintapi.RunProcessingModeBackground,
		Location:          constraintapi.CallerLocationSchedule,
	}
	if req.RunMode == enums.RunModeSync {
		source.Service = constraintapi.ServiceAPI
		source.RunProcessingMode = constraintapi.RunProcessingModeDurableEndpoint
		source.Location = constraintapi.CallerLocationCheckpoint
	}

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
			operationIempotencyKey := lID.String()

			res, err := capacityManager.ExtendLease(ctx, &constraintapi.CapacityExtendLeaseRequest{
				IdempotencyKey: operationIempotencyKey,
				AccountID:      req.AccountID,
				LeaseID:        lID,
				Duration:       ScheduleLeaseDuration,
				Source:         source,
			})
			if err != nil {
				l.Error("could not extend schedule capacity lease", "err", err, "leaseID", lID, "req", req)
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
			operationIdempotencyKey := lID.String()

			service.Go(func() {
				_, internalErr := capacityManager.Release(context.Background(), &constraintapi.CapacityReleaseRequest{
					AccountID:      req.AccountID,
					LeaseID:        lID,
					IdempotencyKey: operationIdempotencyKey,
					Source:         source,
				})
				if internalErr != nil {
					l.ReportError(internalErr, "failed to release capacity after schedule", logger.WithErrorReportTags(map[string]string{
						"account_id":  req.AccountID.String(),
						"lease_id":    lID.String(),
						"function_id": req.Function.ID.String(),
					}))
				}
			})
		}
	}()

	// Run user code with lease guarantee
	// NOTE: The passed context will be canceled if the lease expires.
	return fn(userCtx, false)
}

type checkResult struct {
	// allowed determines whether a run can be scheduled
	allowed bool

	// leaseID is the current capacity lease which MUST be committed once done or rolled back on error
	leaseID *ulid.ULID
}

func getScheduleConstraints(ctx context.Context, req execution.ScheduleRequest) ([]constraintapi.ConstraintItem, error) {
	var requests []constraintapi.ConstraintItem

	// The only constraint we care about in run scheduling is rate limiting.
	// Throttle + concurrency constraints are checked in the queue.
	if req.Function.RateLimit != nil && !req.PreventRateLimit {
		rateLimitKey, err := ratelimit.RateLimitKey(ctx, req.Function.ID, *req.Function.RateLimit, req.Events[0].GetEvent().Map())
		switch err {
		case ratelimit.ErrNotRateLimited:
			// no rate limit configured, do not return constraints
			return nil, nil
		case nil:
			// enforce rate limit
		default:
			return nil, fmt.Errorf("could not get rate limit key: %w", err)
		}

		var rateLimitKeyExpr string
		if req.Function.RateLimit.Key != nil {
			rateLimitKeyExpr = *req.Function.RateLimit.Key
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
	now time.Time,
	capacityManager constraintapi.CapacityManager,
	useConstraintAPI constraintapi.UseConstraintAPIFn,
	req execution.ScheduleRequest,
	idempotencyKey string,
	constraints []constraintapi.ConstraintItem,
) (checkResult, error) {
	l := logger.StdlibLogger(ctx)

	// NOTE: Schedule may be called from within new-runs or the API
	// In case of the API, we want to ensure the source is properly reflected in constraint checks
	// to enforce fairness between callers
	source := constraintapi.LeaseSource{
		Service:           constraintapi.ServiceExecutor,
		RunProcessingMode: constraintapi.RunProcessingModeBackground,
		Location:          constraintapi.CallerLocationSchedule,
	}
	if req.RunMode == enums.RunModeSync {
		source.Service = constraintapi.ServiceAPI
		source.RunProcessingMode = constraintapi.RunProcessingModeDurableEndpoint
		source.Location = constraintapi.CallerLocationCheckpoint
	}

	// TODO: Fetch account concurrency
	var accountConcurrency int

	configuration, err := queue.ConvertToConstraintConfiguration(accountConcurrency, req.Function)
	if err != nil {
		return checkResult{}, fmt.Errorf("could not create configuration for acquire: %w", err)
	}

	res, internalErr := capacityManager.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
		AccountID:            req.AccountID,
		IdempotencyKey:       idempotencyKey,
		LeaseIdempotencyKeys: []string{idempotencyKey},
		// NOTE: We cannot provide a run ID at this point because
		// we may be retrying a previous Schedule() attempt which
		// already set a run ID. This will only be known after
		// the create state call within schedule().
		// LeaseRunIDs: []ulid.ULID,
		EnvID:             req.WorkspaceID,
		FunctionID:        req.Function.ID,
		Configuration:     configuration,
		Constraints:       constraints,
		Amount:            1,
		CurrentTime:       now,
		Duration:          ScheduleLeaseDuration,
		MaximumLifetime:   5 * time.Minute, // This lease should be short!
		Source:            source,
		BlockingThreshold: 0, // Disable this for now
	})
	if internalErr != nil {
		l.Error("acquiring capacity lease failed", "err", internalErr, "method", "CheckConstraints", "req", req)
		return checkResult{}, fmt.Errorf("could not enforce constraints: %w", internalErr)
	}

	// Rate limited
	allowed := len(res.Leases) == 1
	if !allowed {
		return checkResult{
			allowed: false,
		}, nil
	}

	lease := res.Leases[0]

	return checkResult{
		allowed: true,

		leaseID: &lease.LeaseID,
	}, nil
}
