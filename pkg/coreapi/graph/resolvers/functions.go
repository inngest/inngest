package resolvers

import (
	"context"
	"fmt"
	"sort"

	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/function"
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

		avs, _, err := fn.Actions(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load function configuration")
		}

		av := avs[0]
		rt, ok := av.Runtime.Runtime.(inngest.RuntimeHTTP)
		if !ok {
			return nil, fmt.Errorf("failed to parse function runtime data")
		}

		functions = append(functions, &models.Function{
			ID:          fn.ID,
			Name:        fn.Name,
			Concurrency: fn.Concurrency,
			Triggers:    triggers,
			URL:         rt.URL,
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
	name := "" // TODO

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
	metadata, err := r.Runner.Runs(ctx, "")
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

		startedAt := ulid.Time(m.Identifier.RunID.Time())

		name := "" // TODO
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

// Deploy a function creating a new function version
func (r *mutationResolver) DeployFunction(ctx context.Context, input models.DeployFunctionInput) (*function.FunctionVersion, error) {
	// Parse function CUE or JSON string - This also validates the function
	f, err := function.Unmarshal(ctx, []byte(input.Config), "")
	if err != nil {
		return nil, err
	}

	// TODO - Move default environment to config
	env := "prod"
	if input.Env != nil {
		env = input.Env.String()
	}
	fv, err := r.APIReadWriter.CreateFunctionVersion(ctx, *f, *input.Live, env)
	if err != nil {
		return nil, err
	}

	config, err := function.MarshalCUE(fv.Function)
	if err != nil {
		return nil, err
	}

	fv.Config = string(config)
	return &fv, nil
}
