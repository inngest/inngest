package resolvers

import (
	"context"
	"encoding/json"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

// TODO Duplicate code. Move to field-level resolvers and add dataloaders.
func (r *eventResolver) FunctionRuns(ctx context.Context, obj *models.Event) ([]*models.FunctionRun, error) {
	runs, err := r.Data.GetFunctionRunsFromEvents(ctx, consts.DevServerAccountID, consts.DevServerEnvID, []ulid.ULID{obj.ID})
	if err != nil {
		return nil, err
	}

	var out []*models.FunctionRun

	for _, run := range runs {
		outRun := models.MakeFunctionRun(run)
		out = append(out, outRun)
	}

	return out, nil
}

func (r *eventResolver) PendingRuns(ctx context.Context, obj *models.Event) (*int, error) {
	state, err := r.Runner.Runs(ctx, consts.DevServerAccountID, obj.ID)
	if err != nil {
		return nil, err
	}

	var pending int

	for _, s := range state {
		if s.Metadata().Status == enums.RunStatusRunning {
			pending++
		}
	}

	return &pending, nil
}

func (r *eventResolver) Status(ctx context.Context, obj *models.Event) (*models.EventStatus, error) {
	state, err := r.Runner.Runs(ctx, consts.DevServerAccountID, obj.ID)
	if err != nil {
		return nil, err
	}

	status := models.EventStatusNoFunctions

	if len(state) == 0 {
		return &status, nil
	}

	status = models.EventStatusCompleted
	var failedRuns int
	var isRunning bool

	for _, s := range state {
		m := s.Metadata()
		if m.Status == enums.RunStatusFailed {
			failedRuns++
			continue
		}

		if m.Status == enums.RunStatusRunning {
			isRunning = true
			continue
		}
	}

	if failedRuns > 0 {
		if failedRuns == len(state) {
			status = models.EventStatusFailed
		} else {
			status = models.EventStatusPartiallyFailed
		}
	} else if isRunning {
		status = models.EventStatusRunning
	}

	return &status, nil
}

func (r *eventResolver) Raw(ctx context.Context, obj *models.Event) (*string, error) {
	evt, err := r.Data.GetEventByInternalID(ctx, obj.ID)
	if err != nil {
		return nil, err
	}

	// Marshall the entire event to JSON and return that string.
	byt, err := json.Marshal(evt.Event())
	if err != nil {
		return nil, err
	}

	jsonStr := string(byt)

	return &jsonStr, nil
}

func (r *eventResolver) TotalRuns(ctx context.Context, obj *models.Event) (*int, error) {
	metadata, err := r.Runner.Runs(ctx, consts.DevServerAccountID, obj.ID)
	if err != nil {
		return nil, err
	}

	total := len(metadata)

	return &total, nil
}
