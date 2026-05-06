package registration

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngest/pkg/syscode"
)

// ProcessOpts configures function processing.
type ProcessOpts struct {
	AccountID     uuid.UUID
	EnvironmentID uuid.UUID
	AppID         uuid.UUID

	// IdempotencyKey is an optional string (eg. request ID) that is used
	// for accessing the semaphore manager when setting semaphore capacity.
	IdempotencyKey string

	// UseDeterministicIDs determines whether we use determinisctic IDs for
	// fn IDs during processing.  This is required for OSS versions.
	UseDeterministicIDs bool
}

// ProcessResult contains the output of ProcessFunctions.
type ProcessResult struct {
	// opts captures the opts used during processing
	opts ProcessOpts

	// Functions contains validated, enriched functions ready for DB storage.
	Functions []inngest.DeployedFunction
}

// SetSemaphoreCapacity iterates the processed functions and sets semaphore capacity
// for any fn-scoped concurrency limits. This should be called after DB storage.
func (r *ProcessResult) SetSemaphoreCapacity(ctx context.Context, sm constraintapi.SemaphoreManager) {
	if sm == nil {
		return
	}
	for _, df := range r.Functions {
		fn := df.Function
		if fn.Concurrency == nil {
			continue
		}
		for _, fc := range fn.Concurrency.Fn {
			// NOTE: We only set capacity for fn level scopes - app level scopes (used in worker-level concurrency)
			// are mutated whenever workers come online.
			if fc.EffectiveScope() != inngest.FnConcurrencyScopeFn {
				continue
			}
			var semID string
			if fc.Key != nil {
				semID = constraintapi.SemaphoreIDFnKey(df.ID, *fc.Key)
			} else {
				semID = constraintapi.SemaphoreIDFn(df.ID)
			}

			// add the idempotency key to the fn ID.  this resets after the semaphore idempotency period,
			// eg 20 seconds, but ensures simultaneous deploys still update the sem.
			ik := fmt.Sprintf("%s-%s", r.opts.IdempotencyKey, df.ID.String())
			_ = sm.SetCapacity(ctx, df.AccountID, semID, ik, int64(fc.Limit))
		}
	}
}

// ProcessFunctions parses, validates, and enriches functions from a register request.
// This is the single entry point for function registration logic shared between
// devserver and cloud.
func ProcessFunctions(ctx context.Context, req sdk.RegisterRequest, opts ProcessOpts) (*ProcessResult, error) {

	// Collect all errors for reporting.
	var errs error

	result := &ProcessResult{
		opts:      opts,
		Functions: make([]inngest.DeployedFunction, 0, len(req.Functions)),
	}

	for _, sdkFn := range req.Functions {
		fn, err := sdkFn.Function()
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}

		if err := fn.Validate(ctx); err != nil {
			errs = multierror.Append(errs, err)
			continue
		}

		if opts.UseDeterministicIDs {
			fn.ID = fn.DeterministicUUID()
		}

		// Inject app semaphore for connect apps. Runs AFTER Validate()
		// so the app-scoped semaphore bypasses user-facing validation.
		if req.IsConnect() {
			if fn.Concurrency == nil {
				fn.Concurrency = &inngest.ConcurrencyLimits{}
			}
			fn.Concurrency.Fn = append(fn.Concurrency.Fn, inngest.FnConcurrency{
				ID:    constraintapi.SemaphoreIDApp(opts.AppID),
				Scope: inngest.FnConcurrencyScopeApp,
			})
		}

		result.Functions = append(result.Functions, inngest.DeployedFunction{
			ID:            fn.ID,
			Slug:          fn.Slug,
			Function:      *fn,
			AccountID:     opts.AccountID,
			EnvironmentID: opts.EnvironmentID,
			AppID:         opts.AppID,
		})
	}

	if errs != nil {
		data := syscode.DataMultiErr{}
		data.Append(errs)
		return nil, &syscode.Error{
			Code: syscode.CodeConfigInvalid,
			Data: data,
		}
	}

	if len(req.Functions) == 0 {
		return result, sdk.ErrNoFunctions
	}
	return result, nil
}
