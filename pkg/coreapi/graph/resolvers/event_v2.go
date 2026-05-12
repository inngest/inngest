package resolvers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	loader "github.com/inngest/inngest/pkg/coreapi/graph/loaders"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
)

func (qr *queryResolver) EventV2(ctx context.Context, id ulid.ULID) (*models.EventV2, error) {
	targetLoader := loader.FromCtx(ctx).EventLoader

	event, err := loader.LoadOneWithString[cqrs.Event](
		ctx,
		targetLoader,
		id.String(),
	)

	if err != nil {
		return nil, err
	}

	return cqrsEventToGQLEvent(event), nil
}

func (er eventV2Resolver) Runs(ctx context.Context, obj *models.EventV2) ([]*models.FunctionRunV2, error) {
	// Source runs from the function_runs table (populated for every
	// scheduled run) so that lookups still work with v1 trace writes
	// disabled.
	funcRuns, err := er.Data.GetFunctionRunsFromEvents(ctx, uuid.Nil, uuid.Nil, []ulid.ULID{obj.ID})
	if err != nil {
		return nil, err
	}

	// Batched runs only carry the batch's first event ID in function_runs,
	// so look up batches that contain this event ID and pull their runs in
	// directly.
	batches, batchErr := er.Data.GetEventBatchesByEventID(ctx, obj.ID)
	if batchErr == nil {
		seen := make(map[ulid.ULID]struct{}, len(funcRuns))
		for _, fr := range funcRuns {
			seen[fr.RunID] = struct{}{}
		}
		for _, b := range batches {
			if _, ok := seen[b.RunID]; ok {
				continue
			}
			fr, err := er.Data.GetFunctionRun(ctx, uuid.Nil, uuid.Nil, b.RunID)
			if err != nil {
				continue
			}
			funcRuns = append(funcRuns, fr)
			seen[b.RunID] = struct{}{}
		}
	}

	out := make([]*models.FunctionRunV2, 0, len(funcRuns))
	for _, fr := range funcRuns {
		status := fr.Status
		if status == 0 {
			// FunctionRun has no finish row yet, so the run is still in
			// flight. enums.RunStatusRunning is the running zero value, so
			// this is redundant but keeps intent explicit.
			status = fr.Status
		}

		gqlStatus, err := models.ToFunctionRunStatus(status)
		if err != nil {
			continue
		}

		var (
			started   *time.Time
			ended     *time.Time
			output    *string
			batchTime *time.Time
		)
		if !fr.RunStartedAt.IsZero() {
			started = &fr.RunStartedAt
		}
		if fr.EndedAt != nil {
			ended = fr.EndedAt
		}
		if len(fr.Output) > 0 {
			s := string(fr.Output)
			output = &s
		}
		isBatch := fr.BatchID != nil
		if isBatch {
			ts := ulid.Time(fr.BatchID.Time())
			batchTime = &ts
		}

		out = append(out, &models.FunctionRunV2{
			ID:             fr.RunID,
			FunctionID:     fr.FunctionID,
			QueuedAt:       ulid.Time(fr.RunID.Time()),
			StartedAt:      started,
			EndedAt:        ended,
			Status:         gqlStatus,
			Output:         output,
			IsBatch:        isBatch,
			BatchCreatedAt: batchTime,
			CronSchedule:   fr.Cron,
		})
	}
	return out, nil
}

func (er eventV2Resolver) Raw(ctx context.Context, obj *models.EventV2) (string, error) {
	targetLoader := loader.FromCtx(ctx).EventLoader

	event, err := loader.LoadOneWithString[cqrs.Event](
		ctx,
		targetLoader,
		obj.ID.String(),
	)
	if err != nil {
		return "", err
	}

	raw, err := marshalRaw(event)
	if err != nil {
		return "", err
	}

	return raw, nil
}

func marshalRaw(e *cqrs.Event) (string, error) {
	data := e.EventData
	if data == nil {
		data = make(map[string]any)
	}

	var version *string
	if len(e.EventVersion) > 0 {
		version = &e.EventVersion
	}

	id := e.InternalID().String()
	if len(e.EventID) > 0 {
		id = e.EventID
	}

	byt, err := json.Marshal(map[string]any{
		"data": data,
		"id":   id,
		"name": e.EventName,
		"ts":   e.EventTS,
		"v":    version,
	})
	if err != nil {
		return "", err
	}
	return string(byt), nil
}
