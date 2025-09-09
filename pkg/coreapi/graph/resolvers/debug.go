package resolvers

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/consts"
	loader "github.com/inngest/inngest/pkg/coreapi/graph/loaders"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
)

func (qr *queryResolver) DebugRun(ctx context.Context, query models.DebugRunQuery) (*models.DebugRun, error) {
	var debugRunSpan *models.RunTraceSpan
	var runSteps []*models.RunStep

	if query.DebugRunID != nil {
		debugRunID, err := ulid.Parse(*query.DebugRunID)
		if err != nil {
			return nil, fmt.Errorf("invalid debugRunID: %w", err)
		}

		debugRunSpan, err = loader.LoadOne[models.RunTraceSpan](
			ctx,
			loader.FromCtx(ctx).DebugRunLoader,
			&loader.DebugRunRequestKey{
				DebugRunID: debugRunID,
			},
		)
		if err != nil {
			return nil, err
		}
		runSteps = qr.extractRunSteps(debugRunSpan)
	}

	if runSteps == nil && query.RunID != nil {
		runID, err := ulid.Parse(*query.RunID)
		if err != nil {
			return nil, fmt.Errorf("invalid runID: %w", err)
		}

		function, err := qr.getFunctionBySlug(ctx, query.FunctionSlug)
		if err != nil {
			return nil, fmt.Errorf("function not found: %w", err)
		}

		var runSpan *models.RunTraceSpan

		runSpan, err = loader.LoadOne[models.RunTraceSpan](
			ctx,
			loader.FromCtx(ctx).RunTraceLoader,
			&loader.TraceRequestKey{
				TraceRunIdentifier: &cqrs.TraceRunIdentifier{
					FunctionID: function.ID,
					RunID:      runID,
				},
			},
		)
		if err != nil {
			return nil, err
		}
		runSteps = qr.extractRunSteps(runSpan)
	}

	// TODO: handle step extraction when there is no run id provided, e.g. search for ANY run

	return &models.DebugRun{
		DebugRun: debugRunSpan,
		RunSteps: runSteps,
	}, nil
}

func (qr *queryResolver) DebugSession(ctx context.Context, query models.DebugSessionQuery) ([]*models.RunTraceSpan, error) {
	if query.DebugSessionID != nil {
		debugSessionID, err := ulid.Parse(*query.DebugSessionID)
		if err != nil {
			return nil, fmt.Errorf("invalid debugSessionID: %w", err)
		}

		spans, err := loader.LoadOne[[]*models.RunTraceSpan](
			ctx,
			loader.FromCtx(ctx).DebugSessionLoader,
			&loader.DebugSessionRequestKey{
				DebugSessionID: debugSessionID,
			},
		)
		if err != nil {
			return nil, err
		}

		return *spans, nil
	}

	if query.RunID != nil {
		runID, err := ulid.Parse(*query.RunID)
		if err != nil {
			return nil, fmt.Errorf("invalid runID: %w", err)
		}

		function, err := qr.getFunctionBySlug(ctx, query.FunctionSlug)
		if err != nil {
			return nil, fmt.Errorf("function not found: %w", err)
		}

		traceRun, err := qr.Data.GetTraceRun(ctx, cqrs.TraceRunIdentifier{
			RunID:      runID,
			FunctionID: function.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("trace run not found: %w", err)
		}

		rootSpan, err := loader.LoadOne[models.RunTraceSpan](
			ctx,
			loader.FromCtx(ctx).RunTraceLoader,
			&loader.TraceRequestKey{
				TraceRunIdentifier: &cqrs.TraceRunIdentifier{
					FunctionID: function.ID,
					RunID:      runID,
					TraceID:    traceRun.TraceID,
				},
			},
		)
		if err != nil {
			return nil, fmt.Errorf("error getting trace span: %w", err)
		}

		result := []*models.RunTraceSpan{rootSpan}
		result = append(result, qr.collectChildrenSpans(rootSpan)...)

		return result, nil
	}

	return nil, fmt.Errorf("debug session not found")
}

func (qr *queryResolver) getFunctionBySlug(ctx context.Context, functionSlug string) (*cqrs.Function, error) {
	return qr.Data.GetFunctionByExternalID(ctx, consts.DevServerEnvID, "local", functionSlug)
}

func (qr *queryResolver) collectChildrenSpans(span *models.RunTraceSpan) []*models.RunTraceSpan {
	var result []*models.RunTraceSpan
	for _, child := range span.ChildrenSpans {
		result = append(result, child)
		result = append(result, qr.collectChildrenSpans(child)...)
	}
	return result
}

// Extract run steps from run trace for debugger display
func (qr *queryResolver) extractRunSteps(span *models.RunTraceSpan) []*models.RunStep {
	if span == nil {
		return nil
	}

	var steps []*models.RunStep
	qr.collectRunSteps(span, &steps)
	return steps
}

func (qr *queryResolver) collectRunSteps(span *models.RunTraceSpan, steps *[]*models.RunStep) {
	if span.StepID != nil && *span.StepID != "" {
		*steps = append(*steps, &models.RunStep{
			StepID: *span.StepID,
			Name:   span.Name,
			StepOp: span.StepOp,
		})
	}

	for _, child := range span.ChildrenSpans {
		qr.collectRunSteps(child, steps)
	}
}

func (r *mutationResolver) CreateDebugSession(ctx context.Context, input models.CreateDebugSessionInput) (*models.CreateDebugSessionResponse, error) {
	return &models.CreateDebugSessionResponse{
		DebugSessionID: ulid.Make(),
		DebugRunID:     ulid.Make(),
	}, nil
}
