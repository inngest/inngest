package resolvers

import (
	"context"
	"encoding/json"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
)

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

			events = append(events, models.FunctionEvent{
				Type:      &t,
				CreatedAt: &h.CreatedAt,
				Output:    &output,
			})
		} else {
			t := stepEventEnum(h.Type)

			events = append(events, models.StepEvent{
				Type:      &t,
				CreatedAt: &h.CreatedAt,
				Output:    &output,
			})
		}
	}

	return events, nil
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
	case enums.HistoryTypeStepSleeping:
		return models.StepEventTypeSleeping
	}

	return models.StepEventTypeScheduled
}
