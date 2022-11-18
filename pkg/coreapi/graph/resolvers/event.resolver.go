package resolvers

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/enums"
)

// TODO Duplicate code. Move to field-level resolvers and add dataloaders.
func (r *eventResolver) FunctionRuns(ctx context.Context, obj *models.Event) ([]*models.FunctionRun, error) {
	metadata, err := r.Runner.Runs(ctx, obj.ID)
	if err != nil {
		return nil, err
	}

	var runs []*models.FunctionRun

	for _, m := range metadata {
		status := models.FunctionRunStatusRunning

		switch m.Status {
		case enums.RunStatusCompleted:
			status = models.FunctionRunStatusCompleted
		case enums.RunStatusFailed:
			status = models.FunctionRunStatusFailed
		case enums.RunStatusCancelled:
			status = models.FunctionRunStatusCancelled
		}

		var startedAt time.Time

		if m.OriginalRunID != nil {
			startedAt = time.UnixMilli(int64(m.OriginalRunID.Time()))
		}

		name := string(m.Name)
		pending := int(m.Pending)

		runs = append(runs, &models.FunctionRun{
			ID:           m.OriginalRunID.String(),
			Name:         &name,
			Status:       &status,
			PendingSteps: &pending,
			StartedAt:    &startedAt,
		})
	}

	sort.Slice(runs, func(i, j int) bool {
		return runs[i].ID > runs[j].ID
	})

	return runs, nil
}

func (r *eventResolver) PendingRuns(ctx context.Context, obj *models.Event) (*int, error) {
	metadata, err := r.Runner.Runs(ctx, obj.ID)
	if err != nil {
		return nil, err
	}

	var pending int

	for _, m := range metadata {
		if m.Status == enums.RunStatusRunning {
			pending++
		}
	}

	return &pending, nil
}

func (r *eventResolver) Status(ctx context.Context, obj *models.Event) (*models.EventStatus, error) {
	metadata, err := r.Runner.Runs(ctx, obj.ID)
	if err != nil {
		return nil, err
	}

	status := models.EventStatusCompleted

	for _, m := range metadata {
		if m.Status == enums.RunStatusFailed {
			status = models.EventStatusFailed
			break
		}

		if m.Status == enums.RunStatusRunning {
			status = models.EventStatusRunning
		}
	}

	return &status, nil
}

func (r *eventResolver) Raw(ctx context.Context, obj *models.Event) (*string, error) {
	evts, err := r.Runner.Events(ctx, obj.ID)
	if err != nil {
		return nil, err
	}

	if len(evts) == 0 {
		return nil, nil
	}

	evt := evts[0]

	// Marshall the entire event to JSON and return that string.
	byt, err := json.Marshal(evt)
	if err != nil {
		return nil, err
	}

	jsonStr := string(byt)

	return &jsonStr, nil
}
