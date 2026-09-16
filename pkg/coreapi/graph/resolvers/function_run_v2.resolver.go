package resolvers

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	loader "github.com/inngest/inngest/pkg/coreapi/graph/loaders"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/logger"
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
	defersPtr, err := loader.LoadOneWithString[[]cqrs.RunDefer](ctx, loader.FromCtx(ctx).RunDefersLoader, fn.ID.String())
	if err != nil {
		return nil, fmt.Errorf("error retrieving run defers: %w", err)
	}
	var defers []cqrs.RunDefer
	if defersPtr != nil {
		defers = *defersPtr
	}

	out := make([]*models.RunDefer, 0, len(defers))
	for _, d := range defers {
		status, err := models.ToRunDeferStatus(d.Status)
		if err != nil {
			return nil, fmt.Errorf("error converting defer status: %w", err)
		}
		out = append(out, &models.RunDefer{
			HashedDeferID:   d.HashedDeferID,
			UserlandDeferID: d.UserlandDeferID,
			FnSlug:          d.FnSlug,
			Status:          status,
			RunID:           d.RunID,
		})
	}
	return out, nil
}

func (r *runDeferResolver) Function(ctx context.Context, d *models.RunDefer) (*models.Function, error) {
	fn, err := r.Data.GetFunctionByExternalID(ctx, uuid.Nil, "", d.FnSlug)
	if err != nil {
		return nil, nil
	}
	return models.MakeFunction(fn)
}

func (r *runDeferResolver) Run(ctx context.Context, d *models.RunDefer) (*models.FunctionRunV2, error) {
	if d.RunID == nil {
		return nil, nil
	}
	run, err := r.Data.GetTraceRun(ctx, cqrs.TraceRunIdentifier{RunID: *d.RunID})
	if err != nil {
		logger.StdlibLogger(ctx).Error(
			"failed to get run",
			"error", err,
			"run_id", *d.RunID,
		)
		return nil, errors.New("failed to get run")
	}
	return models.MakeFunctionRunV2(run)
}

func (r *functionRunV2Resolver) Trace(ctx context.Context, fn *models.FunctionRunV2) (*models.RunTraceSpan, error) {
	return loader.LoadOne[models.RunTraceSpan](
		ctx,
		loader.FromCtx(ctx).RunTraceLoader,
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
