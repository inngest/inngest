package resolvers

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/oklog/ulid/v2"
)

func (r *queryResolver) Functions(ctx context.Context) ([]*models.Function, error) {
	all, err := r.Data.GetFunctions(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]*models.Function, len(all))
	for n, i := range all {
		res[n], err = models.MakeFunction(i)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
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

	m := state.Metadata()
	status, err := models.ToFunctionRunStatus(m.Status)
	if err != nil {
		return nil, err
	}

	name := state.Function().Name

	pending, _ := r.Queue.OutstandingJobCount(
		ctx,
		m.Identifier.WorkspaceID,
		m.Identifier.WorkflowID,
		m.Identifier.RunID,
	)

	run, err := r.Data.GetFunctionRun(
		ctx,
		m.Identifier.AccountID,
		m.Identifier.WorkspaceID,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("Run ID not found: %w", err)
	}

	fr := models.MakeFunctionRun(run)
	fr.PendingSteps = &pending
	fr.Name = &name
	fr.Status = &status
	return fr, nil
}
