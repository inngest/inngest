package resolvers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
)

func (r *functionRunResolver) Status(ctx context.Context, obj *models.FunctionRun) (*models.FunctionRunStatus, error) {
	md, err := r.Runner.StateManager().Metadata(ctx, ulid.MustParse(obj.ID))
	if err != nil {
		return nil, fmt.Errorf("Run ID not found: %w", err)
	}
	status := models.FunctionRunStatusRunning
	switch md.Status {
	case enums.RunStatusCompleted:
		status = models.FunctionRunStatusCompleted
	case enums.RunStatusFailed:
		status = models.FunctionRunStatusFailed
	case enums.RunStatusCancelled:
		status = models.FunctionRunStatusCancelled
	}
	return &status, nil
}

func (r *functionRunResolver) PendingSteps(ctx context.Context, obj *models.FunctionRun) (*int, error) {
	md, err := r.Runner.StateManager().Metadata(ctx, ulid.MustParse(obj.ID))
	if err != nil {
		return nil, fmt.Errorf("Run ID not found: %w", err)
	}
	return &md.Pending, nil
}

func (r *functionRunResolver) Function(ctx context.Context, obj *models.FunctionRun) (*models.Function, error) {
	fn, err := r.Data.GetFunctionByID(ctx, uuid.MustParse(obj.FunctionID))
	if err != nil {
		return nil, err
	}
	return models.MakeFunction(fn)
}

func (r *functionRunResolver) FinishedAt(ctx context.Context, obj *models.FunctionRun) (*time.Time, error) {
	f, err := r.Data.GetFunctionRunFinishesByRunIDs(ctx, []ulid.ULID{ulid.MustParse(obj.ID)})
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if len(f) == 1 {
		return &f[0].CreatedAt, nil
	}
	return nil, nil
}

func (r *functionRunResolver) Output(ctx context.Context, obj *models.FunctionRun) (*string, error) {
	f, err := r.Data.GetFunctionRunFinishesByRunIDs(ctx, []ulid.ULID{ulid.MustParse(obj.ID)})
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if len(f) == 1 {
		str := string(f[0].Output)
		return &str, nil
	}
	return nil, nil
}

// TODO Duplicate code. Move to field-level resolvers and add dataloaders.
func (r *functionRunResolver) Timeline(ctx context.Context, obj *models.FunctionRun) ([]models.FunctionRunEvent, error) {
	history, err := r.Runner.History(ctx, state.Identifier{
		RunID: ulid.MustParse(obj.ID),
	})
	if err != nil {
		return nil, err
	}

	var events []models.FunctionRunEvent

	for _, h := range history {
		outputByt, err := json.Marshal(h.Data)
		if err != nil {
			continue
		}
		output := string(outputByt)

		if isFunctionEvent(h.Type) {
			t := functionEventEnum(h.Type)
			createdAt := h.CreatedAt

			events = append(events, models.FunctionEvent{
				Type:      &t,
				CreatedAt: &createdAt,
				Output:    &output,
			})
		} else {
			t := stepEventEnum(h.Type)
			createdAt := h.CreatedAt

			event := models.StepEvent{
				Type:      &t,
				CreatedAt: &createdAt,
				Output:    &output,
			}

			switch h.Type {
			case enums.HistoryTypeStepWaiting:
				if step, ok := h.Data.(state.HistoryStep); ok {
					data, _ := step.Data.(map[string]any)
					byt, _ := json.Marshal(data)
					var stepData state.HistoryStepWaitingData
					_ = json.Unmarshal(byt, &stepData)
					event.WaitingFor = &models.StepEventWait{
						ExpiryTime: stepData.ExpiryTime,
						EventName:  stepData.EventName,
						Expression: stepData.Expression,
					}
					event.Output = nil
				}
			default:
				if stepData, ok := h.Data.(state.HistoryStep); ok {
					event.Name = &stepData.Name
					outputByt, err := json.Marshal(stepData.Data)
					if err != nil {
						continue
					}
					output := string(outputByt)
					event.Output = &output
				}
			}

			events = append(events, event)
		}
	}

	return events, nil
}

func (r *functionRunResolver) Event(ctx context.Context, obj *models.FunctionRun) (*models.Event, error) {
	history, err := r.Runner.History(ctx, state.Identifier{
		RunID: ulid.MustParse(obj.ID),
	})
	if err != nil {
		return nil, err
	}

	if len(history) == 0 {
		return nil, nil
	}

	// Find the function start event, which should contain the triggering event.
	var startEvent *state.History
	for _, h := range history {
		if h.Type == enums.HistoryTypeFunctionStarted {
			startEvent = &h
			break
		}
	}

	if startEvent == nil {
		return nil, nil
	}

	jsonStr, err := json.Marshal(startEvent.Data)
	if err != nil {
		return nil, err
	}

	event := &event.Event{}
	if err := json.Unmarshal(jsonStr, event); err != nil {
		return nil, err
	}

	createdAt := time.UnixMilli(event.Timestamp)
	payloadByt, err := json.Marshal(event.Data)
	if err != nil {
		return nil, err
	}
	payload := string(payloadByt)

	return &models.Event{
		ID:        event.ID,
		Name:      &event.Name,
		CreatedAt: &createdAt,
		Payload:   &payload,
	}, nil
}

func (r *functionRunResolver) WaitingFor(ctx context.Context, obj *models.FunctionRun) (*models.StepEventWait, error) {
	history, err := r.Runner.History(ctx, state.Identifier{
		RunID: ulid.MustParse(obj.ID),
	})
	if err != nil {
		return nil, err
	}

	var wait *models.StepEventWait

	for _, h := range history {
		// If this isn't a waiting event, skip it.
		// We also skip function completed logs, as these are thrown early for SDK functions.
		if h.Type != enums.HistoryTypeStepWaiting && h.Type != enums.HistoryTypeFunctionCompleted {
			wait = nil
			continue
		}
		step, ok := h.Data.(state.HistoryStep)
		if ok {
			data, _ := step.Data.(map[string]any)
			byt, _ := json.Marshal(data)
			var stepData state.HistoryStepWaitingData
			_ = json.Unmarshal(byt, &stepData)
			wait = &models.StepEventWait{
				ExpiryTime: stepData.ExpiryTime,
				EventName:  stepData.EventName,
				Expression: stepData.Expression,
			}
		}
	}

	return wait, nil
}

func isFunctionEvent(h enums.HistoryType) bool {
	return h == enums.HistoryTypeFunctionStarted || h == enums.HistoryTypeFunctionCompleted || h == enums.HistoryTypeFunctionCancelled || h == enums.HistoryTypeFunctionFailed
}

func functionEventEnum(h enums.HistoryType) models.FunctionEventType {
	switch h {
	case enums.HistoryTypeFunctionCompleted:
		return models.FunctionEventTypeCompleted
	case enums.HistoryTypeFunctionCancelled:
		return models.FunctionEventTypeCancelled
	case enums.HistoryTypeFunctionFailed:
		return models.FunctionEventTypeFailed
	}

	return models.FunctionEventTypeStarted
}

func stepEventEnum(h enums.HistoryType) models.StepEventType {
	switch h {
	case enums.HistoryTypeStepStarted:
		return models.StepEventTypeStarted
	case enums.HistoryTypeStepCompleted:
		return models.StepEventTypeCompleted
	case enums.HistoryTypeStepErrored:
		return models.StepEventTypeErrored
	case enums.HistoryTypeStepFailed:
		return models.StepEventTypeFailed
	case enums.HistoryTypeStepWaiting:
		return models.StepEventTypeWaiting
	}

	return models.StepEventTypeScheduled
}
