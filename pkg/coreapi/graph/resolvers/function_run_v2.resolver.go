package resolvers

import (
	"context"
	"fmt"

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
				HashedDeferID:   d.HashedDeferID,
				UserlandDeferID: d.UserlandDeferID,
				FnSlug:          d.FnSlug,
				Status:          status,
				RunID:           d.RunID,
			})
		}
	}
	return out, nil
}

func (r *functionRunV2Resolver) DeferredFrom(ctx context.Context, fn *models.FunctionRunV2) ([]*models.RunDeferredFrom, error) {
	dfs, err := loader.LoadOneWithString[[]cqrs.RunDeferredFrom](ctx, loader.FromCtx(ctx).RunDeferredFromLoader, fn.ID.String())
	if err != nil {
		return nil, fmt.Errorf("error retrieving deferred-from linkage: %w", err)
	}
	if dfs == nil {
		return nil, nil
	}

	out := make([]*models.RunDeferredFrom, 0, len(*dfs))
	for _, df := range *dfs {
		out = append(out, &models.RunDeferredFrom{
			RunID:  df.RunID,
			FnSlug: df.FnSlug,
		})
	}
	return out, nil
}

// RunDefer field resolvers — Function/Run are loaded lazily so list views
// that only render hashed id + status skip the joins.

func (r *runDeferResolver) Function(ctx context.Context, obj *models.RunDefer) (*models.Function, error) {
	if obj.FnSlug == "" {
		return nil, nil
	}
	fn, err := r.Data.GetFunctionByExternalID(ctx, uuid.Nil, "", obj.FnSlug)
	if err != nil {
		// Fn slugs that don't resolve are valid: a defer is recorded even
		// when its target function was deleted or renamed. Surface nil so
		// the GraphQL field returns null per schema.
		return nil, nil
	}
	return models.MakeFunction(fn)
}

func (r *runDeferResolver) Run(ctx context.Context, d *models.RunDefer) (*models.FunctionRunV2, error) {
	if d.RunID == nil {
		return nil, nil
	}
	runs, err := r.Data.GetTraceRunsByRunIDs(ctx, []ulid.ULID{*d.RunID})
	if err != nil {
		return nil, fmt.Errorf("error retrieving defer child run: %w", err)
	}
	run, ok := runs[*d.RunID]
	if !ok || run == nil {
		return nil, nil
	}
	return models.MakeFunctionRunV2(run)
}

func (r *runDeferredFromResolver) Function(ctx context.Context, obj *models.RunDeferredFrom) (*models.Function, error) {
	if obj.FnSlug == "" {
		return nil, fmt.Errorf("missing fn slug for deferred-from linkage")
	}
	fn, err := r.Data.GetFunctionByExternalID(ctx, uuid.Nil, "", obj.FnSlug)
	if err != nil {
		return nil, fmt.Errorf("error retrieving deferred-from function: %w", err)
	}
	return models.MakeFunction(fn)
}

func (r *runDeferredFromResolver) Run(ctx context.Context, obj *models.RunDeferredFrom) (*models.FunctionRunV2, error) {
	run, err := r.Data.GetTraceRun(ctx, cqrs.TraceRunIdentifier{RunID: obj.RunID})
	if err != nil {
		return nil, nil
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
