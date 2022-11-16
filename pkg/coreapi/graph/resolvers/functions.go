package resolvers

import (
	"context"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/function"
)

type functionRunResolver struct{ *Resolver }

func (r *queryResolver) FunctionRun(ctx context.Context, query models.FunctionRunQuery) (*models.FunctionRun, error) {
	return nil, nil
}

func (r *queryResolver) FunctionRuns(ctx context.Context, query models.FunctionRunsQuery) ([]*models.FunctionRun, error) {
	metadata, err := r.Runner.Runs(ctx)
	if err != nil {
		return nil, err
	}

	if len(metadata) == 0 {
		return nil, nil
	}

	var runs []*models.FunctionRun

	for _, m := range metadata {
		status := models.FunctionRunStatusRunning

		switch m.Status {
		case state.RunStatusCompleted:
			status = models.FunctionRunStatusCompleted
		case state.RunStatusFailed:
			status = models.FunctionRunStatusFailed
		case state.RunStatusCancelled:
			status = models.FunctionRunStatusCancelled
		}

		var startedAt time.Time

		if m.OriginalRunID != nil {
			startedAt = time.UnixMilli(int64(m.OriginalRunID.Time()))
		}

		runs = append(runs, &models.FunctionRun{
			ID:           m.OriginalRunID.String(),
			Status:       &status,
			PendingSteps: &m.Pending,
			StartedAt:    &startedAt,
		})
	}

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

func (r *functionRunResolver) Status(ctx context.Context, as string) (string, error) {
	return as, nil
}
