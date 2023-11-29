package resolvers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/history_reader"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
)

func (r *functionRunResolver) Status(ctx context.Context, obj *models.FunctionRun) (*models.FunctionRunStatus, error) {
	md, err := r.Runner.StateManager().Metadata(ctx, ulid.MustParse(obj.ID))
	if err != nil {
		return nil, fmt.Errorf("Run ID not found: %w", err)
	}
	status := models.FunctionRunStatusRunning
	switch md.Status {
	case enums.RunStatusCompleted:
		status = models.FunctionRunStatusCompleted
	case enums.RunStatusFailed:
		status = models.FunctionRunStatusFailed
	case enums.RunStatusCancelled:
		status = models.FunctionRunStatusCancelled
	}
	return &status, nil
}

func (r *functionRunResolver) PendingSteps(ctx context.Context, obj *models.FunctionRun) (*int, error) {
	md, err := r.Runner.StateManager().Metadata(ctx, ulid.MustParse(obj.ID))
	if err != nil {
		return nil, fmt.Errorf("Run ID not found: %w", err)
	}
	pending, _ := r.Queue.OutstandingJobCount(
		ctx,
		md.Identifier.WorkspaceID,
		md.Identifier.WorkflowID,
		md.Identifier.RunID,
	)
	return &pending, nil
}

func (r *functionRunResolver) Function(ctx context.Context, obj *models.FunctionRun) (*models.Function, error) {
	fn, err := r.Data.GetFunctionByID(ctx, uuid.MustParse(obj.FunctionID))
	if err != nil {
		return nil, err
	}
	return models.MakeFunction(fn)
}

func (r *functionRunResolver) FinishedAt(ctx context.Context, obj *models.FunctionRun) (*time.Time, error) {
	// Mapped in MakeFunctionRun
	return obj.FinishedAt, nil
}

func (r *functionRunResolver) History(
	ctx context.Context,
	obj *models.FunctionRun,
) ([]*history_reader.RunHistory, error) {
	runID, err := ulid.Parse(obj.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid run ID: %w", err)
	}

	// For required UUID fields that don't matter in OSS.
	randomUUID := uuid.New()

	return r.HistoryReader.GetRunHistory(
		ctx,
		runID,
		history_reader.GetRunOpts{
			AccountID: randomUUID,
		},
	)
}

func (r *functionRunResolver) HistoryItemOutput(
	ctx context.Context,
	obj *models.FunctionRun,
	historyID ulid.ULID,
) (*string, error) {
	runID, err := ulid.Parse(obj.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid run ID: %w", err)
	}

	// For required UUID fields that don't matter in OSS.
	randomUUID := uuid.New()

	return r.HistoryReader.GetRunHistoryItemOutput(
		ctx,
		historyID,
		history_reader.GetHistoryOutputOpts{
			AccountID:   randomUUID,
			RunID:       runID,
			WorkflowID:  randomUUID,
			WorkspaceID: randomUUID,
		},
	)
}

func (r *functionRunResolver) Output(ctx context.Context, obj *models.FunctionRun) (*string, error) {
	// Mapped in MakeFunctionRun
	return obj.Output, nil
}

func (r *functionRunResolver) Event(ctx context.Context, obj *models.FunctionRun) (*models.Event, error) {
	eventID, err := ulid.Parse(obj.EventID)
	if err != nil {
		return nil, err
	}

	evt, err := r.Data.GetEventByInternalID(ctx, eventID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(evt.EventData)
	if err != nil {
		return nil, err
	}

	return &models.Event{
		CreatedAt: &evt.ReceivedAt,
		ID:        evt.ID.String(),
		Name:      &evt.EventName,
		Payload:   util.StrPtr(string(payload)),
	}, nil
}

func (r *functionRunResolver) WaitingFor(ctx context.Context, obj *models.FunctionRun) (*models.StepEventWait, error) {
	return nil, nil
}
