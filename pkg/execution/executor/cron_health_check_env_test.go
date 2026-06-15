package executor

import (
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/stretchr/testify/require"
)

// TestCronHealthCheckEnvID covers the self-hosted cron scheduling regression in
// issue #4387: self-hosted functions are stored without an env ID, so cqrs.Function.EnvID
// is uuid.Nil. The cron health-check re-sync used that zero env ID directly as the cron
// item's workspace ID, which the capacity-lease CheckConstraints validation rejects with
// "missing envID", so self-hosted crons could never schedule and the health check looped
// forever re-syncing them.
func TestCronHealthCheckEnvID(t *testing.T) {
	t.Run("falls back to DevServerEnvID for self-hosted functions with no env ID", func(t *testing.T) {
		// Self-hosted function as returned by GetFunctions: EnvID is the zero UUID.
		fn := &cqrs.Function{EnvID: uuid.Nil}

		got := cronHealthCheckEnvID(fn)

		require.Equal(t, consts.DevServerEnvID, got)
		require.NotEqual(t, uuid.Nil, got, "resolved env ID must be non-nil so constraint validation passes")
	})

	t.Run("nil function falls back to DevServerEnvID", func(t *testing.T) {
		require.Equal(t, consts.DevServerEnvID, cronHealthCheckEnvID(nil))
	})

	t.Run("preserves a real env ID when present (cloud/multi-tenant)", func(t *testing.T) {
		real := uuid.New()
		require.Equal(t, real, cronHealthCheckEnvID(&cqrs.Function{EnvID: real}))
	})

	// Regression guard: prove the resolved env ID actually passes the same constraint
	// validation that was rejecting the request, and that the old zero env ID did not.
	t.Run("resolved env ID passes capacity-lease validation while uuid.Nil fails", func(t *testing.T) {
		fn := &cqrs.Function{EnvID: uuid.Nil}
		resolved := cronHealthCheckEnvID(fn)

		base := func(envID uuid.UUID) *constraintapi.CapacityCheckRequest {
			return &constraintapi.CapacityCheckRequest{
				AccountID: consts.DevServerAccountID,
				EnvID:     envID,
				Constraints: []constraintapi.ConstraintItem{
					{Kind: constraintapi.ConstraintKindConcurrency},
				},
			}
		}

		// Before the fix: the raw zero env ID is rejected with "missing envID".
		err := base(fn.EnvID).Valid()
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing envID")

		// After the fix: the resolved env ID is accepted (no "missing envID").
		err = base(resolved).Valid()
		if err != nil {
			require.NotContains(t, err.Error(), "missing envID")
		}
	})
}
