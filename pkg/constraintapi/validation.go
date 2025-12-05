package constraintapi

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

const (
	MaximumLeaseLifetime = 3 * time.Hour
	MinimumDuration      = 2 * time.Second
	MaximumDuration      = 1 * time.Minute
	MaximumAmount        = 20
	MaxConstraints       = 10

	MaxIdempotencyKeyLength = 128

	// Max constraints per kind
	MaxRateLimits            = 1
	MaxThrottles             = 1
	MaxCustomConcurrencyKeys = 2
)

func (r *CapacityCheckRequest) Valid() error {
	var errs error

	if r.AccountID == uuid.Nil {
		errs = multierror.Append(errs, fmt.Errorf("missing accountID"))
	}

	if r.EnvID == uuid.Nil {
		errs = multierror.Append(errs, fmt.Errorf("missing envID"))
	}

	// We want to allow check requests without specific functions to check account concurrency, etc.
	// For function-level constraints, we do require the function ID and version, see the constraint checks below
	//
	// if r.FunctionID == uuid.Nil {
	// 	errs = multierror.Append(errs, fmt.Errorf("missing functionID"))
	// }
	//
	// if r.Configuration.FunctionVersion == 0 {
	// 	errs = multierror.Append(errs, fmt.Errorf("missing constraint config workflow version"))
	// }

	// Validate configuration
	if err := r.Configuration.Valid(); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("invalid configuration: %w", err))
	}

	if len(r.Constraints) == 0 {
		errs = multierror.Append(errs, fmt.Errorf("must provide constraints"))
	}

	if len(r.Constraints) > MaxConstraints {
		errs = multierror.Append(errs, fmt.Errorf("exceeds %d maximum constraints", MaxConstraints))
	}

	// Validate individual constraint items
	for i, ci := range r.Constraints {
		if err := ci.Valid(); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("invalid constraint %d: %w", i, err))
		}

		if err := r.Configuration.ValidConstraintUsage(ci); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("invalid constraint usage %d: %w", i, err))
		}

		isFunctionLevelConstraint := ci.IsFunctionLevelConstraint()

		if isFunctionLevelConstraint && r.FunctionID == uuid.Nil {
			errs = multierror.Append(errs, fmt.Errorf("function ID is required for function-level constraints"))
		}

		if isFunctionLevelConstraint && r.Configuration.FunctionVersion == 0 {
			errs = multierror.Append(errs, fmt.Errorf("function version is required for function-level constraints"))
		}
	}

	// NOTE: This validation is only enforced as long as existing constraint state
	// and the new lease-related data are colocated.
	//
	// Once we move all constraint state to a dedicated store, we will be able to
	// mix constraints of different stages.
	var hasRateLimit bool
	var hasQueueConstraint bool
	for _, ci := range r.Constraints {
		if ci.Kind.IsQueueConstraint() {
			hasQueueConstraint = true
		}

		if ci.Kind == ConstraintKindRateLimit {
			hasRateLimit = true
		}
	}

	if hasRateLimit && hasQueueConstraint {
		errs = multierror.Append(errs, fmt.Errorf("cannot mix queue and rate limit constraints for first stage"))
	}

	// Ensure migration identifier is provided
	if hasRateLimit && !r.Migration.IsRateLimit {
		errs = multierror.Append(errs, fmt.Errorf("missing rate limit flag in migration identifier"))
	}

	if hasQueueConstraint && r.Migration.QueueShard == "" {
		errs = multierror.Append(errs, fmt.Errorf("missing queue shard in migration identifier"))
	}

	return errs
}

func (r *CapacityAcquireRequest) Valid() error {
	var errs error

	if r.IdempotencyKey == "" {
		errs = multierror.Append(errs, fmt.Errorf("missing idempotency key"))
	}

	if len(r.IdempotencyKey) > MaxIdempotencyKeyLength {
		errs = multierror.Append(errs, fmt.Errorf("idempotency key longer than %d chars", MaxIdempotencyKeyLength))
	}

	if r.AccountID == uuid.Nil {
		errs = multierror.Append(errs, fmt.Errorf("missing accountID"))
	}

	if r.EnvID == uuid.Nil {
		errs = multierror.Append(errs, fmt.Errorf("missing envID"))
	}

	if r.FunctionID == uuid.Nil {
		errs = multierror.Append(errs, fmt.Errorf("missing functionID"))
	}

	if r.Configuration.FunctionVersion == 0 {
		errs = multierror.Append(errs, fmt.Errorf("missing constraint config workflow version"))
	}

	if r.CurrentTime.IsZero() {
		errs = multierror.Append(errs, fmt.Errorf("missing current time"))
	}

	if r.Duration <= MinimumDuration {
		errs = multierror.Append(errs, fmt.Errorf("duration smaller than minimum of %s", MinimumDuration))
	}

	if r.Duration > MaximumDuration {
		errs = multierror.Append(errs, fmt.Errorf("duration exceeds max value of %s", MaximumDuration))
	}

	// NOTE: We do not verify blocking threshold.

	if r.MaximumLifetime <= 0 {
		errs = multierror.Append(errs, fmt.Errorf("missing maximum lifetime"))
	}

	if r.MaximumLifetime > MaximumLeaseLifetime {
		errs = multierror.Append(errs, fmt.Errorf("exceeds maximum lease lifetime of %s", MaximumLeaseLifetime))
	}

	if r.Duration > r.MaximumLifetime {
		errs = multierror.Append(errs, fmt.Errorf("duration cannot be greater than maximum lifetime"))
	}

	if r.Source.Service == ServiceUnknown {
		errs = multierror.Append(errs, fmt.Errorf("missing source service"))
	}

	if r.Source.Location == CallerLocationUnknown {
		errs = multierror.Append(errs, fmt.Errorf("missing source location"))
	}

	if len(r.LeaseIdempotencyKeys) == 0 {
		errs = multierror.Append(errs, fmt.Errorf("missing lease idempotency keys"))
	}

	if r.Amount <= 0 {
		errs = multierror.Append(errs, fmt.Errorf("must request at least one lease"))
	}

	if r.Amount != len(r.LeaseIdempotencyKeys) {
		errs = multierror.Append(errs, fmt.Errorf("must provide as many lease idempotency keys as amount"))
	}

	for i, v := range r.LeaseIdempotencyKeys {
		if len(v) > MaxIdempotencyKeyLength {
			errs = multierror.Append(errs, fmt.Errorf("idempotency key %d longer than %d chars", i, MaxIdempotencyKeyLength))
		}
	}

	if r.Amount > MaximumAmount {
		errs = multierror.Append(errs, fmt.Errorf("must request no more than %d leases", MaximumAmount))
	}

	// Validate configuration
	if err := r.Configuration.Valid(); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("invalid configuration: %w", err))
	}

	if len(r.Constraints) == 0 {
		errs = multierror.Append(errs, fmt.Errorf("must provide constraints"))
	}

	if len(r.Constraints) > MaxConstraints {
		errs = multierror.Append(errs, fmt.Errorf("exceeds %d maximum constraints", MaxConstraints))
	}

	// Validate individual constraint items
	for i, ci := range r.Constraints {
		if err := ci.Valid(); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("invalid constraint %d: %w", i, err))
		}

		if err := r.Configuration.ValidConstraintUsage(ci); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("invalid constraint usage %d: %w", i, err))
		}
	}

	// NOTE: This validation is only enforced as long as existing constraint state
	// and the new lease-related data are colocated.
	//
	// Once we move all constraint state to a dedicated store, we will be able to
	// mix constraints of different stages.
	var hasRateLimit bool
	var hasQueueConstraint bool
	for _, ci := range r.Constraints {
		if ci.Kind.IsQueueConstraint() {
			hasQueueConstraint = true
		}

		if ci.Kind == ConstraintKindRateLimit {
			hasRateLimit = true
		}
	}

	if hasRateLimit && hasQueueConstraint {
		errs = multierror.Append(errs, fmt.Errorf("cannot mix queue and rate limit constraints for first stage"))
	}

	// Ensure migration identifier is provided
	if hasRateLimit && !r.Migration.IsRateLimit {
		errs = multierror.Append(errs, fmt.Errorf("missing rate limit flag in migration identifier"))
	}

	if hasQueueConstraint && r.Migration.QueueShard == "" {
		errs = multierror.Append(errs, fmt.Errorf("missing queue shard in migration identifier"))
	}

	return errs
}

// Valid validates a ConstraintItem ensuring required fields are present
func (ci ConstraintItem) Valid() error {
	switch ci.Kind {
	case ConstraintKindConcurrency:
		// TODO: Implement run level concurrency and remove this validation
		if ci.Concurrency != nil && ci.Concurrency.Mode == enums.ConcurrencyModeRun {
			return fmt.Errorf("run level concurrency is not implemented yet")
		}
		if ci.Concurrency != nil && ci.Concurrency.InProgressItemKey == "" {
			return fmt.Errorf("concurrency constraint must specify InProgressItemKey")
		}
	case ConstraintKindThrottle:
		if ci.Throttle != nil && ci.Throttle.EvaluatedKeyHash == "" {
			return fmt.Errorf("throttle constraint must include EvaluatedKeyHash")
		}
	case ConstraintKindRateLimit:
		if ci.RateLimit != nil && ci.RateLimit.EvaluatedKeyHash == "" {
			return fmt.Errorf("rate limit constraint must include EvaluatedKeyHash")
		}
	}
	return nil
}

// Valid validates a ConstraintItem ensuring required fields are present
func (cc ConstraintConfig) Valid() error {
	var errs error

	// We want to allow check requests without specific functions to check account concurrency, etc.
	// if cc.FunctionVersion == 0 {
	// 	errs = multierror.Append(errs, fmt.Errorf("missing function version"))
	// }

	if len(cc.Concurrency.CustomConcurrencyKeys) > MaxCustomConcurrencyKeys {
		errs = multierror.Append(errs, fmt.Errorf("exceeded maximum of %d custom concurrency keys", MaxCustomConcurrencyKeys))
	}

	if len(cc.RateLimit) > MaxRateLimits {
		errs = multierror.Append(errs, fmt.Errorf("exceeded maximum of %d rate limits", MaxRateLimits))
	}

	if len(cc.Throttle) > MaxThrottles {
		errs = multierror.Append(errs, fmt.Errorf("exceeded maximum of %d throttles", MaxThrottles))
	}

	return errs
}

func (cc ConstraintConfig) ValidConstraintUsage(ci ConstraintItem) error {
	switch ci.Kind {
	case ConstraintKindConcurrency:
		if ci.Concurrency != nil && ci.Concurrency.EvaluatedKeyHash != "" {
			var found bool
			for _, ckey := range cc.Concurrency.CustomConcurrencyKeys {
				if ckey.Scope == ci.Concurrency.Scope && ckey.Mode == ci.Concurrency.Mode && ckey.KeyExpressionHash == ci.Concurrency.KeyExpressionHash {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("unknown custom concurrency key")
			}
		}

	case ConstraintKindThrottle:
		if ci.Throttle != nil {
			var found bool
			for _, t := range cc.Throttle {
				if t.Scope == ci.Throttle.Scope && t.KeyExpressionHash == ci.Throttle.KeyExpressionHash {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("unknown throttle constraint")
			}
		}

	case ConstraintKindRateLimit:
		if ci.RateLimit != nil {
			var found bool
			for _, r := range cc.RateLimit {
				if r.Scope == ci.RateLimit.Scope && r.KeyExpressionHash == ci.RateLimit.KeyExpressionHash {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("unknown rate limit constraint")
			}
		}
	}
	return nil
}

func (r *CapacityExtendLeaseRequest) Valid() error {
	var errs error

	if r.IdempotencyKey == "" {
		errs = multierror.Append(errs, fmt.Errorf("missing idempotency key"))
	}

	if len(r.IdempotencyKey) > MaxIdempotencyKeyLength {
		errs = multierror.Append(errs, fmt.Errorf("idempotency key longer than %d chars", MaxIdempotencyKeyLength))
	}

	if r.AccountID == uuid.Nil {
		errs = multierror.Append(errs, fmt.Errorf("missing accountID"))
	}

	if r.LeaseID == ulid.Zero {
		errs = multierror.Append(errs, fmt.Errorf("missing lease ID"))
	}

	if r.Duration <= 0 {
		errs = multierror.Append(errs, fmt.Errorf("invalid duration: must be positive"))
	}

	return errs
}

func (r *CapacityReleaseRequest) Valid() error {
	var errs error

	if r.IdempotencyKey == "" {
		errs = multierror.Append(errs, fmt.Errorf("missing idempotency key"))
	}

	if len(r.IdempotencyKey) > MaxIdempotencyKeyLength {
		errs = multierror.Append(errs, fmt.Errorf("idempotency key longer than %d chars", MaxIdempotencyKeyLength))
	}

	if r.AccountID == uuid.Nil {
		errs = multierror.Append(errs, fmt.Errorf("missing accountID"))
	}

	if r.LeaseID == ulid.Zero {
		errs = multierror.Append(errs, fmt.Errorf("missing lease ID"))
	}

	return errs
}
