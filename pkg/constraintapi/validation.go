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

	// TODO: Validate configuration

	if len(r.Constraints) == 0 {
		errs = multierror.Append(errs, fmt.Errorf("must request capacity"))
	}

	// TODO: Validate constraints

	return errs
}
