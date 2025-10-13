package constraintapi

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
)

func (r *CapacityLeaseRequest) Valid() error {
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

	if r.LatestFunctionVersion == 0 {
		errs = multierror.Append(errs, fmt.Errorf("missing latest workflow version"))
	}

	if r.CurrentTime.IsZero() {
		errs = multierror.Append(errs, fmt.Errorf("missing current time"))
	}

	if r.Duration == 0 {
		errs = multierror.Append(errs, fmt.Errorf("missing duration"))
	}

	// NOTE: We do not verify blocking threshold.

	if r.MaximumLifetime == 0 {
		errs = multierror.Append(errs, fmt.Errorf("missing maximum lifetime"))
	}

	return errs
}
