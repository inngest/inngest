package resolvers

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/history_reader"
	"github.com/oklog/ulid/v2"
)

func (qr *queryResolver) Functions(ctx context.Context) ([]*models.Function, error) {
	all, err := qr.Data.GetFunctions(ctx)
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

func (qr *queryResolver) FunctionRun(ctx context.Context, query models.FunctionRunQuery) (*models.FunctionRun, error) {
	if query.FunctionRunID == "" {
		return nil, fmt.Errorf("function run id is required")
	}

	runID, err := ulid.Parse(query.FunctionRunID)
	if err != nil {
		return nil, fmt.Errorf("invalid run ID: %w", err)
	}

	run, err := qr.HistoryReader.GetRun(
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
