package resolvers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/history_reader"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
)

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
	fn, err := r.Data.GetFunctionByInternalUUID(ctx, uuid.UUID{}, uuid.MustParse(obj.FunctionID))
	if err != nil {
		return nil, err
	}
	return models.MakeFunction(fn)
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

func (r *functionRunResolver) Events(ctx context.Context, obj *models.FunctionRun) ([]*models.Event, error) {
	empty := []*models.Event{}
	runID := ulid.MustParse(obj.ID)

	batch, err := r.Data.GetEventBatchByRunID(ctx, runID)
	if err != nil {
		// if an error occur, it likely means there are no batches for this runID
		// attempt to just return a single event, that's similar to the Event resolver
		evt, err := r.Event(ctx, obj)
		if err != nil {
			return empty, nil
		}

		return []*models.Event{evt}, nil
	}

	// retrieve events by IDs
	evts, err := r.Data.GetEventsByInternalIDs(ctx, batch.EventIDs())
	if err != nil {
		return empty, err
	}

	result := make([]*models.Event, len(evts))
	for i, e := range evts {
		payload, err := json.Marshal(e.EventData)
		if err != nil {
			return empty, err
		}

		result[i] = &models.Event{
			ID:        e.ID.String(),
			Name:      &e.EventName,
			CreatedAt: &e.ReceivedAt,
			Payload:   util.StrPtr(string(payload)),
		}
	}

	return result, nil
}

func (r *functionRunResolver) WaitingFor(ctx context.Context, obj *models.FunctionRun) (*models.StepEventWait, error) {
	return nil, nil
}

func (r *functionRunResolver) BatchCreatedAt(ctx context.Context, obj *models.FunctionRun) (*time.Time, error) {
	batch, err := r.Data.GetEventBatchByRunID(ctx, ulid.MustParse(obj.ID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	out := ulid.Time(batch.ID.Time())
	return &out, nil
}
