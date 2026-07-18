package executor

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
)

// saveStepAndEnqueueDiscovery persists an executor-owned step result and resumes
// the workflow. A retry after persistence must still attempt discovery, even
// when the state store reports that the response already exists.
func (e *executor) saveStepAndEnqueueDiscovery(
	ctx context.Context,
	runCtx execution.RunContext,
	gen state.GeneratorOpcode,
	edge queue.PayloadEdge,
	output []byte,
	coalesceKey *string,
) error {
	if delta := runCtx.Metadata().Metrics.SwapMetadataSizeDelta(); delta > 0 {
		ctx = state.WithMetadataSizeDelta(ctx, delta)
	}

	hasPendingSteps, err := e.smv2.SaveStep(ctx, runCtx.Metadata().ID, gen.ID, output)
	if errors.Is(err, state.ErrDuplicateResponse) || errors.Is(err, state.ErrIdempotentResponse) {
		e.log.Warn("step output already persisted; keeping existing output", "error", err, "run_id", runCtx.Metadata().ID.RunID, "step_id", gen.ID)
		err = nil
	}
	if err != nil {
		return err
	}

	if err := runCtx.ReleaseCapacityLease(); err != nil {
		logger.StdlibLogger(ctx).ReportError(err, "could not release capacity lease early")
	}

	groupID := uuid.New().String()
	ctx = state.WithGroupID(ctx, groupID)
	return e.maybeEnqueueDiscoveryStep(ctx, runCtx, gen, edge, groupID, hasPendingSteps, coalesceKey)
}
