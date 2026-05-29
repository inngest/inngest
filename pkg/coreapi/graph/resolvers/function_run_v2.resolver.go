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
			HashedDeferID: d.HashedDeferID,
			DeferID:       d.UserDeferID,
			FnSlug:        d.FnSlug,
			Status:        status,
			RunID:         d.RunID,
		})
	}
	return out, nil
}

// SiblingDefers returns the defers from this run's parent(s), excluding the
// entry that scheduled this run. Lets the UI render "parallel defers" without
// having to fetch the parent's full defer list and filter client-side.
func (r *functionRunV2Resolver) SiblingDefers(ctx context.Context, fn *models.FunctionRunV2) ([]*models.RunDefer, error) {
	df, err := loader.LoadOneWithString[[]cqrs.RunDeferredFrom](ctx, loader.FromCtx(ctx).RunDeferredFromLoader, fn.ID.String())
	if err != nil {
		return nil, fmt.Errorf("error retrieving deferred-from linkage: %w", err)
	}
	if df == nil || len(*df) == 0 {
		return []*models.RunDefer{}, nil
	}

	parentRunIDs := make([]ulid.ULID, 0, len(*df))
	for _, p := range *df {
		parentRunIDs = append(parentRunIDs, p.RunID)
	}

	defersByParent, err := r.Data.GetRunDefers(ctx, parentRunIDs)
	if err != nil {
		return nil, fmt.Errorf("error retrieving sibling defers: %w", err)
	}

	out := []*models.RunDefer{}
	for _, defers := range defersByParent {
		for _, d := range defers {
			// Drop the defer entry that scheduled this run; siblings are
			// peers, not self.
			if d.RunID != nil && *d.RunID == fn.ID {
				continue
			}
			status, err := models.ToRunDeferStatus(d.Status)
			if err != nil {
				return nil, fmt.Errorf("error converting defer status: %w", err)
			}
			out = append(out, &models.RunDefer{
				HashedDeferID: d.HashedDeferID,
				DeferID:       d.UserDeferID,
				FnSlug:        d.FnSlug,
				Status:        status,
				RunID:         d.RunID,
			})
		}
	}
	return out, nil
}

func (r *functionRunV2Resolver) DeferredFrom(ctx context.Context, fn *models.FunctionRunV2) ([]*models.RunDeferredFrom, error) {
	df, err := loader.LoadOneWithString[[]cqrs.RunDeferredFrom](ctx, loader.FromCtx(ctx).RunDeferredFromLoader, fn.ID.String())
	if err != nil {
		return nil, fmt.Errorf("error retrieving deferred-from linkage: %w", err)
	}
	if df == nil {
		return nil, nil
	}

	out := make([]*models.RunDeferredFrom, 0, len(*df))
	for _, parent := range *df {
		out = append(out, &models.RunDeferredFrom{
			RunID:  parent.RunID,
			FnSlug: parent.FnSlug,
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

func (r *runDeferredFromResolver) Function(ctx context.Context, df *models.RunDeferredFrom) (*models.Function, error) {
	fun, err := r.Data.GetFunctionByExternalID(ctx, uuid.Nil, "", df.FnSlug)
	if err != nil {
		return nil, fmt.Errorf("error retrieving deferred-from parent function: %w", err)
	}
	return models.MakeFunction(fun)
}

func (r *runDeferredFromResolver) Run(ctx context.Context, df *models.RunDeferredFrom) (*models.FunctionRunV2, error) {
	run, err := r.Data.GetTraceRun(ctx, cqrs.TraceRunIdentifier{RunID: df.RunID})
	if err != nil {
		logger.StdlibLogger(ctx).Error(
			"failed to get run",
			"error", err,
			"run_id", df.RunID,
		)
		return nil, errors.New("failed to get run")
	}
	return models.MakeFunctionRunV2(run)
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
