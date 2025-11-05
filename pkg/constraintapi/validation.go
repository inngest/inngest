package constraintapi

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
)

func (r *CapacityAcquireRequest) Valid() error {
	var errs error

	if r.IdempotencyKey == "" {
		errs = multierror.Append(errs, fmt.Errorf("missing idempotency key"))
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

	if r.Duration <= 0 {
		errs = multierror.Append(errs, fmt.Errorf("missing duration"))
	}

	// NOTE: We do not verify blocking threshold.

	if r.MaximumLifetime <= 0 {
		errs = multierror.Append(errs, fmt.Errorf("missing maximum lifetime"))
	}

	if r.Source.Service == ServiceUnknown {
		errs = multierror.Append(errs, fmt.Errorf("missing source service"))
	}

	if r.Source.Location == LeaseLocationUnknown {
		errs = multierror.Append(errs, fmt.Errorf("missing source location"))
	}

	if len(r.LeaseIdempotencyKeys) == 0 {
		errs = multierror.Append(errs, fmt.Errorf("missing lease idempotency keys"))
	}

	// TODO: Validate configuration

	if len(r.Constraints) == 0 {
		errs = multierror.Append(errs, fmt.Errorf("must request capacity"))
	}

	// Validate individual constraint items
	for i, ci := range r.Constraints {
		if err := ci.Valid(); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("invalid constraint %d: %w", i, err))
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

	return errs
}

// Valid validates a ConstraintItem ensuring required fields are present
func (ci ConstraintItem) Valid() error {
	switch ci.Kind {
	case ConstraintKindConcurrency:
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
