package executor

import (
	"context"

	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
)

// maybeResetForceStepPlan resets ForceStepPlan (disableImmediateExecution) to
// false when parallelism has ended for a v2 execution, re-enabling checkpointing.
// This is a no-op for v1 executions or when ForceStepPlan is already false.
func (e *executor) maybeResetForceStepPlan(ctx context.Context, md *sv2.Metadata) error {
	if md.Config.RequestVersion < 2 || !md.Config.ForceStepPlan {
		return nil
	}

	return e.smv2.UpdateMetadata(ctx, md.ID, sv2.MutableConfig{
		ForceStepPlan:  false,
		RequestVersion: md.Config.RequestVersion,
		StartedAt:      md.Config.StartedAt,
		HasAI:          md.Config.HasAI,
	})
}
