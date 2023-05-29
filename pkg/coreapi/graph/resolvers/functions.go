package resolvers

import (
	"context"
	"fmt"
	"sort"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

func (r *queryResolver) Functions(ctx context.Context) ([]*models.Function, error) {
	fns, err := r.APIReadWriter.Functions(ctx)
	if err != nil {
		return nil, err
	}

	var functions []*models.Function
	for _, fn := range fns {
		var triggers []*models.FunctionTrigger

		for _, trigger := range fn.Triggers {
			t := &models.FunctionTrigger{}
			if trigger.EventTrigger != nil {
				t.Type = models.FunctionTriggerTypesEvent
				t.Value = trigger.Event
			}
			if trigger.CronTrigger != nil {
				t.Type = models.FunctionTriggerTypesCron
				t.Value = trigger.Cron
			}
			triggers = append(triggers, t)
		}

		functions = append(functions, &models.Function{
			ID:          fn.Name,
			Name:        fn.Name,
			Concurrency: fn.ConcurrencyLimit(),
			Triggers:    triggers,
			URL:         "",
		})
	}

	return functions, nil
}

func (r *queryResolver) FunctionRun(ctx context.Context, query models.FunctionRunQuery) (*models.FunctionRun, error) {
	if query.FunctionRunID == "" {
		return nil, fmt.Errorf("function run id is required")
	}

	runID, err := ulid.Parse(query.FunctionRunID)
	if err != nil {
		return nil, fmt.Errorf("Invalid run ID: %w", err)
	}

	state, err := r.Runner.StateManager().Load(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("Run ID not found: %w", err)
	}

	status := models.FunctionRunStatusRunning

	switch state.Metadata().Status {
	case enums.RunStatusCompleted:
		status = models.FunctionRunStatusCompleted
	case enums.RunStatusFailed:
		status = models.FunctionRunStatusFailed
	case enums.RunStatusCancelled:
		status = models.FunctionRunStatusCancelled
	}

	startedAt := ulid.Time(runID.Time())
	name := state.Function().Name

	pending := state.Metadata().Pending
	if pending < 0 {
		pending = 0
	}

	return &models.FunctionRun{
		ID:           runID.String(),
		Name:         &name,
		Status:       &status,
		PendingSteps: &pending,
		StartedAt:    &startedAt,
	}, nil
}

func (r *queryResolver) FunctionRuns(ctx context.Context, query models.FunctionRunsQuery) ([]*models.FunctionRun, error) {
	state, err := r.Runner.Runs(ctx, "")
	if err != nil {
		return nil, err
	}

	var runs []*models.FunctionRun

	for _, s := range state {
		m := s.Metadata()
		status := models.FunctionRunStatusRunning

		switch m.Status {
		case enums.RunStatusCompleted:
			status = models.FunctionRunStatusCompleted
		case enums.RunStatusFailed:
			status = models.FunctionRunStatusFailed
		case enums.RunStatusCancelled:
			status = models.FunctionRunStatusCancelled
		}

		startedAt := ulid.Time(m.Identifier.RunID.Time())

		name := s.Function().Name
		pending := int(m.Pending)

		// Don't let pending be negative for clients
		if pending < 0 {
			pending = 0
		}

		runs = append(runs, &models.FunctionRun{
			ID:           m.Identifier.RunID.String(),
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
