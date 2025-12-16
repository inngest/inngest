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
	runs, err := r.Data.GetEventRuns(ctx, obj.ID, consts.DevServerAccountID, consts.DevServerEnvID)
	if err != nil {
		return nil, err
	}

	var pending int
	for _, r := range runs {
		if r.Status == enums.RunStatusRunning {
			pending++
		}
	}

	return &pending, nil
}

func (r *eventResolver) Status(ctx context.Context, obj *models.Event) (*models.EventStatus, error) {
	runs, err := r.Data.GetEventRuns(ctx, obj.ID, consts.DevServerAccountID, consts.DevServerEnvID)
	if err != nil {
		return nil, err
	}

	status := models.EventStatusNoFunctions

	if len(runs) == 0 {
		return &status, nil
	}

	status = models.EventStatusCompleted
	var failedRuns int
	var isRunning bool

	for _, s := range runs {
		if s.Status == enums.RunStatusFailed {
			failedRuns++
			continue
		}

		if s.Status == enums.RunStatusRunning {
			isRunning = true
			continue
		}
	}

	if failedRuns > 0 {
		if failedRuns == len(runs) {
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
	// XXX: This is implemented during a broad refactor;  we should totally implement a query
	// that returns JUST the count for this, so that we don't read and discard data.   I apologize.
	runs, err := r.Data.GetEventRuns(ctx, obj.ID, consts.DevServerAccountID, consts.DevServerEnvID)
	if err != nil {
		return nil, err
	}

	total := len(runs)
	return &total, nil
}
