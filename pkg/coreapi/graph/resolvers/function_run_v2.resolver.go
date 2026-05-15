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
		runV2, err := models.MakeFunctionRunV2(d.Run)
		if err != nil {
			return nil, fmt.Errorf("error converting defer child run: %w", err)
		}
		status, err := models.ToRunDeferStatus(d.Status)
		if err != nil {
			return nil, fmt.Errorf("error converting defer status: %w", err)
		}
		out = append(out, &models.RunDefer{
			ID:          d.HashedDeferID,
			UserDeferID: d.UserDeferID,
			FnSlug:      d.FnSlug,
			Status:      status,
			Run:         runV2,
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

	parentV2, err := models.MakeFunctionRunV2(df.ParentRun)
	if err != nil {
		return nil, fmt.Errorf("error converting deferred-from parent run: %w", err)
	}

	return &models.RunDeferredFrom{
		ParentRunID: df.ParentRunID,
		ParentRun:   parentV2,
	}, nil
}

// RunType reports whether a run was scheduled as a deferred (child) run or as
// a primary run, derived from the run_defers linkage. The lookup goes through
// RunDeferredFromLoader so requests for many runs in the same query batch into
// a single backend call.
func (r *functionRunV2Resolver) RunType(ctx context.Context, fn *models.FunctionRunV2) (models.RunType, error) {
	df, err := loader.LoadOneWithString[cqrs.RunDeferredFrom](ctx, loader.FromCtx(ctx).RunDeferredFromLoader, fn.ID.String())
	if err != nil {
		return "", fmt.Errorf("error retrieving deferred-from linkage: %w", err)
	}
	if df != nil {
		return models.RunTypeDefer, nil
	}
	return models.RunTypePrimary, nil
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
