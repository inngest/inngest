package resolvers

import (
	"context"
	"fmt"

	"github.com/graph-gophers/dataloader"
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
	thunk := loader.FromCtx(ctx).RunDefersLoader.Load(ctx, dataloader.StringKey(fn.ID.String()))
	result, err := thunk()
	if err != nil {
		return nil, fmt.Errorf("error retrieving run defers: %w", err)
	}
	defers, _ := result.([]cqrs.RunDefer)

	out := make([]*models.RunDefer, 0, len(defers))
	for _, d := range defers {
		var input *string
		if len(d.Input) > 0 {
			s := string(d.Input)
			input = &s
		}
		ref, err := runRefFromCQRS(d.Run)
		if err != nil {
			return nil, err
		}
		out = append(out, &models.RunDefer{
			ID:     d.ID,
			FnSlug: d.FnSlug,
			Status: models.RunDeferStatus(d.Status),
			Input:  input,
			Run:    ref,
		})
	}
	return out, nil
}

func (r *functionRunV2Resolver) DeferredFrom(ctx context.Context, fn *models.FunctionRunV2) (*models.RunDeferredFrom, error) {
	df, err := loader.LoadOneWithString[cqrs.RunDeferredFrom](ctx, loader.FromCtx(ctx).RunDeferredFromLoader, fn.ID.String())
	if err != nil {
		return nil, fmt.Errorf("error retrieving deferred-from linkage: %w", err)
	}
	if df == nil {
		return nil, nil
	}

	parent, err := runRefFromCQRS(df.ParentRun)
	if err != nil {
		return nil, err
	}

	return &models.RunDeferredFrom{
		ParentRunID:  df.ParentRunID,
		ParentFnSlug: df.ParentFnSlug,
		ParentRun:    parent,
	}, nil
}

// RunRef intentionally exposes only id and status; consumers wanting more
// should query the run by id.
func runRefFromCQRS(r *cqrs.FunctionRun) (*models.RunRef, error) {
	if r == nil {
		return nil, nil
	}
	status, err := models.ToFunctionRunStatus(r.Status)
	if err != nil {
		return nil, fmt.Errorf("error parsing run status: %w", err)
	}
	return &models.RunRef{
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
