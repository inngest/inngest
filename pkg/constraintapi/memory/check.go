package memory

import (
	"context"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/util/errs"
)

// Check implements constraintapi.CapacityManager.  it reads every counter and
// writes nothing but the check idempotency record.
func (m *Manager) Check(ctx context.Context, req *constraintapi.CapacityCheckRequest) (*constraintapi.CapacityCheckResponse, errs.UserError, errs.InternalError) {
	return nil, nil, errs.Wrap(0, false, "not implemented")
}
