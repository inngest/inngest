package resolvers

import (
	"context"
	"fmt"

	loader "github.com/inngest/inngest/pkg/coreapi/graph/loaders"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
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

func (r *functionRunV2Resolver) Defers(ctx context.Context, fn *models.FunctionRunV2) ([]*models.RunDefer, error) {
	defers, err := r.Data.GetRunDefers(ctx, fn.ID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving run defers: %w", err)
	}

	out := make([]*models.RunDefer, 0, len(defers))
	for _, d := range defers {
		var input *string
		if len(d.Input) > 0 {
			s := string(d.Input)
			input = &s
		}

		run, err := partialFunctionRunV2(d.Run)
		if err != nil {
			return nil, err
		}

		out = append(out, &models.RunDefer{
			ID:     d.ID,
			FnSlug: d.FnSlug,
			Status: models.RunDeferStatus(d.Status),
			Input:  input,
			Run:    run,
		})
	}
	return out, nil
}

func (r *functionRunV2Resolver) DeferredFrom(ctx context.Context, fn *models.FunctionRunV2) (*models.RunDeferredFrom, error) {
	df, err := r.Data.GetRunDeferredFrom(ctx, fn.ID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving deferred-from linkage: %w", err)
	}
	if df == nil {
		return nil, nil
	}

	parent, err := partialFunctionRunV2(df.ParentRun)
	if err != nil {
		return nil, err
	}

	return &models.RunDeferredFrom{
		ParentRunID:  df.ParentRunID,
		ParentFnSlug: df.ParentFnSlug,
		ParentRun:    parent,
	}, nil
}

// partialFunctionRunV2 maps a cqrs.FunctionRun to a partially-populated
// FunctionRunV2 with only ID and Status. cqrs.FunctionRun lacks the fields
// required to fully populate FunctionRunV2 (TraceID, AppID, etc.); consumers
// needing more should query the run by id.
func partialFunctionRunV2(r *cqrs.FunctionRun) (*models.FunctionRunV2, error) {
	if r == nil {
		return nil, nil
	}
	status, err := models.ToFunctionRunStatus(r.Status)
	if err != nil {
		return nil, fmt.Errorf("error parsing run status: %w", err)
	}
	return &models.FunctionRunV2{
		ID:     r.RunID,
		Status: status,
	}, nil
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
