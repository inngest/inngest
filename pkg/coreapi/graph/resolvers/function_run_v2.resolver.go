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
	if cache := loader.GetLookupCache(ctx); cache != nil {
		v, err := cache.GetOrLoad("app", run.AppID, func() (interface{}, error) {
			return r.Data.GetAppByID(ctx, run.AppID)
		})
		if err != nil {
			return nil, err
		}
		return v.(*cqrs.App), nil
	}
	return r.Data.GetAppByID(ctx, run.AppID)
}

func (r *functionRunV2Resolver) Function(ctx context.Context, fn *models.FunctionRunV2) (*models.Function, error) {
	if cache := loader.GetLookupCache(ctx); cache != nil {
		v, err := cache.GetOrLoad("func", fn.FunctionID, func() (interface{}, error) {
			fun, err := r.Data.GetFunctionByInternalUUID(ctx, fn.FunctionID)
			if err != nil {
				return nil, fmt.Errorf("error retrieving function: %w", err)
			}
			return models.MakeFunction(fun)
		})
		if err != nil {
			return nil, err
		}
		return v.(*models.Function), nil
	}

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
