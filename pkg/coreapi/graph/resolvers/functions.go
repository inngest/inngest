package resolvers

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/function"
)

func (r *queryResolver) FunctionRun(ctx context.Context, query models.FunctionRunQuery) (*models.FunctionRun, error) {
	if query.FunctionRunID == "" {
		return nil, fmt.Errorf("function run id is required")
	}

	metadata, err := r.Runner.Runs(ctx, "")
	if err != nil {
		return nil, err
	}

	var targetRun *state.Metadata

	for _, m := range metadata {
		if m.OriginalRunID.String() == query.FunctionRunID {
			targetRun = &m
			break
		}
	}

	if targetRun == nil {
		return nil, nil
	}

	status := models.FunctionRunStatusRunning

	switch targetRun.Status {
	case enums.RunStatusCompleted:
		status = models.FunctionRunStatusCompleted
	case enums.RunStatusFailed:
		status = models.FunctionRunStatusFailed
	case enums.RunStatusCancelled:
		status = models.FunctionRunStatusCancelled
	}

	var startedAt time.Time

	if targetRun.OriginalRunID != nil {
		startedAt = time.UnixMilli(int64(targetRun.OriginalRunID.Time()))
	}

	name := string(targetRun.Name)
	pending := int(targetRun.Pending)

	// Don't let pending be negative for clients
	if pending < 0 {
		pending = 0
	}

	return &models.FunctionRun{
		ID:           targetRun.OriginalRunID.String(),
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

		var startedAt time.Time

		if m.OriginalRunID != nil {
			startedAt = time.UnixMilli(int64(m.OriginalRunID.Time()))
		}

		name := string(m.Name)
		pending := int(m.Pending)

		// Don't let pending be negative for clients
		if pending < 0 {
			pending = 0
		}

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
