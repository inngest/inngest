package executor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
)

// copyRunState loads the step state and events from a source run and copies
// them into the new run's CreateState. This is used by deferred.start to
// initialize a new run with the completed step outputs of a previous run.
//
// The source run's original events are embedded into the deferred.start
// event's data as `event` and `events`, so the SDK can reconstruct the
// original trigger context.
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
		return fmt.Errorf("error loading source run metadata (runID=%s, accountID=%s, envID=%s): %w",
			sourceID.RunID, sourceID.Tenant.AccountID, sourceID.Tenant.EnvID, err)
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

	// Load the source run's original events and embed them in the
	// deferred.start event's data so the SDK can pass them to the function.
	sourceEvents, err := sm.LoadEvents(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("error loading source run events: %w", err)
	}

	if len(sourceEvents) > 0 && len(newState.Events) > 0 {
		if err := embedOriginalEvents(newState, sourceEvents); err != nil {
			return fmt.Errorf("error embedding original events: %w", err)
		}
	}

	return nil
}

// embedOriginalEvents injects the source run's events into the deferred.start
// event's data as `event` (first event) and `events` (all events).
func embedOriginalEvents(newState *sv2.CreateState, sourceEvents []json.RawMessage) error {
	// Parse the deferred.start event.
	var evt map[string]any
	if err := json.Unmarshal(newState.Events[0], &evt); err != nil {
		return err
	}

	data, _ := evt["data"].(map[string]any)
	if data == nil {
		data = map[string]any{}
	}

	// Decode source events into generic objects.
	decoded := make([]any, len(sourceEvents))
	for i, raw := range sourceEvents {
		var obj any
		if err := json.Unmarshal(raw, &obj); err != nil {
			return err
		}
		decoded[i] = obj
	}

	data["event"] = decoded[0]
	data["events"] = decoded
	evt["data"] = data

	updated, err := json.Marshal(evt)
	if err != nil {
		return err
	}

	newState.Events[0] = updated
	return nil
}
