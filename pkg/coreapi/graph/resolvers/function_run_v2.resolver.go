package resolvers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	loader "github.com/inngest/inngest/pkg/coreapi/graph/loaders"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
)

func (r *functionRunV2Resolver) App(
	ctx context.Context,
	run *models.FunctionRunV2,
) (*cqrs.App, error) {
	return r.Data.GetAppByID(ctx, run.AppID)
}

func (r *functionRunV2Resolver) Function(ctx context.Context, fn *models.FunctionRunV2) (*models.Function, error) {
	fun, err := r.Data.GetFunctionByInternalUUID(ctx, fn.FunctionID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving function: %w", err)
	}

	return models.MakeFunction(fun)
}

func (r *functionRunV2Resolver) Trace(ctx context.Context, fn *models.FunctionRunV2, preview *bool) (*models.RunTraceSpan, error) {
	targetLoader := loader.FromCtx(ctx).LegacyRunTraceLoader
	if preview != nil && *preview {
		targetLoader = loader.FromCtx(ctx).RunTraceLoader
	}

	return loader.LoadOne[models.RunTraceSpan](
		ctx,
		targetLoader,
		&loader.TraceRequestKey{
			TraceRunIdentifier: &cqrs.TraceRunIdentifier{
				AppID:      fn.AppID,
				FunctionID: fn.FunctionID,
				RunID:      fn.ID,
				TraceID:    fn.TraceID,
			},
		},
	)
}

// DeferredRuns returns all deferred runs created from this run.
func (r *functionRunV2Resolver) DeferredRuns(ctx context.Context, run *models.FunctionRunV2) ([]*models.FunctionRunV2, error) {
	fnRuns, err := r.Data.GetFunctionRunsByOriginalRunID(ctx, run.ID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving deferred runs: %w", err)
	}

	result := []*models.FunctionRunV2{}
	for _, fr := range fnRuns {
		// Look up the corresponding trace run for full data
		traceRun, err := r.Data.GetTraceRun(ctx, cqrs.TraceRunIdentifier{RunID: fr.RunID})
		if err != nil {
			// If trace run doesn't exist yet (run just scheduled), build a minimal node
			status, _ := models.ToFunctionRunStatus(fr.Status)
			var started *time.Time
			if !fr.RunStartedAt.IsZero() {
				started = &fr.RunStartedAt
			}
			result = append(result, &models.FunctionRunV2{
				ID:         fr.RunID,
				AppID:      run.AppID,
				FunctionID: fr.FunctionID,
				QueuedAt:   fr.RunStartedAt,
				StartedAt:  started,
				Status:     status,
				TriggerIDs: []ulid.ULID{},
			})
			continue
		}

		node := traceRunToFunctionRunV2(traceRun)
		if node != nil {
			result = append(result, node)
		}
	}

	return result, nil
}

// DeferGroupName returns the defer group name if this is a deferred run.
func (r *functionRunV2Resolver) DeferGroupName(ctx context.Context, run *models.FunctionRunV2) (*string, error) {
	// Check if this run has an OriginalRunID (meaning it's a child/deferred run)
	fnRun, err := r.Data.GetFunctionRun(ctx, uuid.UUID{}, uuid.UUID{}, run.ID)
	if err != nil {
		return nil, nil
	}
	if fnRun.OriginalRunID == nil {
		return nil, nil
	}

	// Load the run's metadata from state to get the actual defer group name.
	md, err := r.Executor.LoadRunMetadata(ctx, run.ID)
	if err != nil || md == nil {
		// State may have been deleted after finalization. Return nil.
		return nil, nil
	}
	name := md.Config.GetDeferGroupID()
	if name == "" {
		return nil, nil
	}
	return &name, nil
}

// ParentRunID returns the parent run ID if this is a deferred run.
func (r *functionRunV2Resolver) ParentRunID(ctx context.Context, run *models.FunctionRunV2) (*string, error) {
	fnRun, err := r.Data.GetFunctionRun(ctx, uuid.UUID{}, uuid.UUID{}, run.ID)
	if err != nil {
		return nil, nil
	}
	if fnRun.OriginalRunID == nil {
		return nil, nil
	}
	id := fnRun.OriginalRunID.String()
	return &id, nil
}

// traceRunToFunctionRunV2 converts a TraceRun to a FunctionRunV2 model.
func traceRunToFunctionRunV2(tr *cqrs.TraceRun) *models.FunctionRunV2 {
	runID, err := ulid.Parse(tr.RunID)
	if err != nil {
		return nil
	}

	status, err := models.ToFunctionRunStatus(tr.Status)
	if err != nil {
		return nil
	}

	var (
		started  *time.Time
		ended    *time.Time
		sourceID *string
		output   *string
		batchTS  *time.Time
	)

	if tr.StartedAt.UnixMilli() > 0 {
		started = &tr.StartedAt
	}
	if tr.EndedAt.UnixMilli() > 0 {
		ended = &tr.EndedAt
	}
	if tr.SourceID != "" {
		sourceID = &tr.SourceID
	}
	if len(tr.Output) > 0 {
		s := string(tr.Output)
		output = &s
	}
	if tr.BatchID != nil {
		ts := ulid.Time(tr.BatchID.Time())
		batchTS = &ts
	}

	triggerIDs := []ulid.ULID{}
	for _, tid := range tr.TriggerIDs {
		if id, err := ulid.Parse(tid); err == nil {
			triggerIDs = append(triggerIDs, id)
		}
	}

	return &models.FunctionRunV2{
		ID:             runID,
		AppID:          tr.AppID,
		FunctionID:     tr.FunctionID,
		TraceID:        tr.TraceID,
		QueuedAt:       tr.QueuedAt,
		StartedAt:      started,
		EndedAt:        ended,
		Status:         status,
		SourceID:       sourceID,
		TriggerIDs:     triggerIDs,
		IsBatch:        tr.IsBatch,
		BatchCreatedAt: batchTS,
		CronSchedule:   tr.CronSchedule,
		Output:         output,
		HasAi:          tr.HasAI,
	}
}
