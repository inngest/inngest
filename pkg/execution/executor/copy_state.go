package executor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
)

// copyRunState loads the step state from a source run and copies it into
// the new run's CreateState. This is used by deferred.start to initialize
// a new run with the completed step outputs of a previous run.
func copyRunState(ctx context.Context, sm sv2.RunService, req execution.ScheduleRequest, newState *sv2.CreateState) error {
	if req.CopyStateFrom == nil {
		return nil
	}

	// Load metadata from the source run to get the full state ID (including function ID).
	sourceID := sv2.ID{
		RunID: *req.CopyStateFrom,
		Tenant: sv2.Tenant{
			AccountID: req.AccountID,
			EnvID:     req.WorkspaceID,
		},
	}

	sourceMeta, err := sm.LoadMetadata(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("error loading source run metadata: %w", err)
	}
	sourceID = sourceMeta.ID

	// Load the step execution stack to preserve ordering.
	stack, err := sm.LoadStack(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("error loading source run stack: %w", err)
	}

	// Load all step outputs from the source run.
	stepData, err := sm.LoadSteps(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("error loading source run steps: %w", err)
	}

	// Build MemoizedSteps in stack order so the new run's stack is correct.
	steps := make([]state.MemoizedStep, 0, len(stack))
	for _, stepID := range stack {
		raw, ok := stepData[stepID]
		if !ok {
			continue
		}

		var data any
		if err := json.Unmarshal(raw, &data); err != nil {
			return fmt.Errorf("error unmarshalling step %s: %w", stepID, err)
		}

		steps = append(steps, state.MemoizedStep{
			ID:   stepID,
			Data: data,
		})
	}

	newState.Steps = steps
	return nil
}
