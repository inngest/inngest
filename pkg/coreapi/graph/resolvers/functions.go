package resolvers

import (
	"context"
	"fmt"
	"sort"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/history_reader"
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

	run, err := r.HistoryReader.GetRun(
		ctx,
		runID,
		history_reader.GetRunOpts{},
	)
	if err != nil {
		return nil, err
	}

	status, err := models.ToFunctionRunStatus(run.Status)
	if err != nil {
		return nil, err
	}

	return &models.FunctionRun{
		ID:         runID.String(),
		FunctionID: run.WorkflowID.String(),
		FinishedAt: run.EndedAt,
		StartedAt:  &run.StartedAt,
		EventID:    run.EventID.String(),
		BatchID:    run.BatchID,
		Status:     &status,
		Output:     run.Output,
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
		status, err := models.ToFunctionRunStatus(m.Status)
		if err != nil {
			return nil, err
		}

		startedAt := ulid.Time(m.Identifier.RunID.Time())

		pending, _ := r.Queue.OutstandingJobCount(
			ctx,
			m.Identifier.WorkspaceID,
			m.Identifier.WorkflowID,
			m.Identifier.RunID,
		)

		runs = append(runs, &models.FunctionRun{
			ID:           m.Identifier.RunID.String(),
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
