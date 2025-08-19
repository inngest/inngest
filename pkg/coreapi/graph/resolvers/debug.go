package resolvers

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	loader "github.com/inngest/inngest/pkg/coreapi/graph/loaders"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
)

func (qr *queryResolver) DebugRun(ctx context.Context, query models.DebugRunQuery) (*models.DebugRun, error) {
	fmt.Printf("DebugRun resolver called with: functionSlug=%s, debugRunID=%v, runID=%v\n",
		query.FunctionSlug, query.DebugRunID, query.RunID)

	var debugRunSpan *models.RunTraceSpan
	var runSteps []*models.RunStep
	var err error

	// If debugRunID is provided, we need to find the span with that debug run ID
	if query.DebugRunID != nil {
		debugRunSpan, err = qr.getSpanByDebugRunID(ctx, query.FunctionSlug, *query.DebugRunID)
		if err != nil {
			return nil, err
		}
		runSteps = qr.extractRunSteps(debugRunSpan)
	} else if query.RunID != nil {
		// If runID is provided, get the trace for that specific run
		runID, err := ulid.Parse(*query.RunID)
		if err != nil {
			return nil, fmt.Errorf("invalid runID: %w", err)
		}

		// Get the function to get its ID
		function, err := qr.getFunctionBySlug(ctx, query.FunctionSlug)
		if err != nil {
			return nil, fmt.Errorf("function not found: %w", err)
		}

		// Use the trace loader to get the trace span
		debugRunSpan, err = loader.LoadOne[models.RunTraceSpan](
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
		runSteps = qr.extractRunSteps(debugRunSpan)
	} else {
		return nil, fmt.Errorf("either debugRunID or runID must be provided")
	}

	return &models.DebugRun{
		DebugRun: debugRunSpan,
		RunSteps: runSteps,
	}, nil
}

func (qr *queryResolver) DebugSession(ctx context.Context, query models.DebugSessionQuery) ([]*models.RunTraceSpan, error) {
	// If debugSessionID is provided, find all spans with that debug session ID
	if query.DebugSessionID != nil {
		return qr.getSpansByDebugSessionID(ctx, query.FunctionSlug, *query.DebugSessionID)
	}

	// If runID is provided, get all spans for that run
	if query.RunID != nil {
		runID, err := ulid.Parse(*query.RunID)
		if err != nil {
			return nil, fmt.Errorf("invalid runID: %w", err)
		}

		// Get the function to get its ID
		function, err := qr.getFunctionBySlug(ctx, query.FunctionSlug)
		if err != nil {
			return nil, fmt.Errorf("function not found: %w", err)
		}

		// Get the trace run to get trace ID
		traceRun, err := qr.Data.GetTraceRun(ctx, cqrs.TraceRunIdentifier{
			RunID:      runID,
			FunctionID: function.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("trace run not found: %w", err)
		}

		// Use the trace loader to get the main trace span, then extract children
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

		// Collect all spans (root + children recursively)
		result := []*models.RunTraceSpan{rootSpan}
		result = append(result, qr.collectChildrenSpans(rootSpan)...)

		return result, nil
	}

	return nil, fmt.Errorf("debug session not found")
}

// Helper function to get function by slug
func (qr *queryResolver) getFunctionBySlug(ctx context.Context, functionSlug string) (*cqrs.Function, error) {
	return qr.Data.GetFunctionByExternalID(ctx, consts.DevServerEnvID, "local", functionSlug)
}

// Helper function to collect children spans recursively
func (qr *queryResolver) collectChildrenSpans(span *models.RunTraceSpan) []*models.RunTraceSpan {
	var result []*models.RunTraceSpan
	for _, child := range span.ChildrenSpans {
		result = append(result, child)
		result = append(result, qr.collectChildrenSpans(child)...)
	}
	return result
}

// Helper function to extract run steps from a trace span
func (qr *queryResolver) extractRunSteps(span *models.RunTraceSpan) []*models.RunStep {
	if span == nil {
		return nil
	}

	var steps []*models.RunStep
	qr.collectRunSteps(span, &steps)
	return steps
}

// Helper function to collect run steps recursively
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

// Helper function to get span by debug run ID
func (qr *queryResolver) getSpanByDebugRunID(ctx context.Context, functionSlug, debugRunID string) (*models.RunTraceSpan, error) {
	// Get the function
	function, err := qr.getFunctionBySlug(ctx, functionSlug)
	if err != nil {
		return nil, err
	}

	// For now, we'll need to get recent trace runs and search through their spans
	// This is not optimal but works until we have better queries
	runs, err := qr.Data.GetTraceRuns(ctx, cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			FunctionID: []uuid.UUID{function.ID},
		},
		Items: 100, // Limit to recent runs
	})
	if err != nil {
		return nil, fmt.Errorf("error getting trace runs: %w", err)
	}

	// Search through runs for spans with the debug run ID
	for _, run := range runs {
		runID, err := ulid.Parse(run.RunID)
		if err != nil {
			continue
		}

		// Use the trace loader to get the trace for this run
		rootSpan, err := loader.LoadOne[models.RunTraceSpan](
			ctx,
			loader.FromCtx(ctx).RunTraceLoader,
			&loader.TraceRequestKey{
				TraceRunIdentifier: &cqrs.TraceRunIdentifier{
					FunctionID: function.ID,
					RunID:      runID,
					TraceID:    run.TraceID,
				},
			},
		)
		if err != nil {
			continue // Skip this run if we can't get spans
		}

		// Check if the root span or any children have the debug run ID
		if found := qr.findSpanByDebugRunID(rootSpan, debugRunID); found != nil {
			return found, nil
		}
	}

	return nil, fmt.Errorf("span with debug run ID %s not found", debugRunID)
}

// Helper function to get spans by debug session ID
func (qr *queryResolver) getSpansByDebugSessionID(ctx context.Context, functionSlug, debugSessionID string) ([]*models.RunTraceSpan, error) {
	// Get the function
	function, err := qr.getFunctionBySlug(ctx, functionSlug)
	if err != nil {
		return nil, err
	}

	// For now, we'll need to get recent trace runs and search through their spans
	runs, err := qr.Data.GetTraceRuns(ctx, cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			FunctionID: []uuid.UUID{function.ID},
		},
		Items: 100, // Limit to recent runs
	})
	if err != nil {
		return nil, fmt.Errorf("error getting trace runs: %w", err)
	}

	var result []*models.RunTraceSpan

	// Search through runs for spans with the debug session ID
	for _, run := range runs {
		runID, err := ulid.Parse(run.RunID)
		if err != nil {
			continue
		}

		// Use the trace loader to get the trace for this run
		rootSpan, err := loader.LoadOne[models.RunTraceSpan](
			ctx,
			loader.FromCtx(ctx).RunTraceLoader,
			&loader.TraceRequestKey{
				TraceRunIdentifier: &cqrs.TraceRunIdentifier{
					FunctionID: function.ID,
					RunID:      runID,
					TraceID:    run.TraceID,
				},
			},
		)
		if err != nil {
			continue // Skip this run if we can't get spans
		}

		// Find all spans with the debug session ID
		found := qr.findSpansByDebugSessionID(rootSpan, debugSessionID)
		result = append(result, found...)
	}

	return result, nil
}

// Helper function to find a span by debug run ID recursively
func (qr *queryResolver) findSpanByDebugRunID(span *models.RunTraceSpan, debugRunID string) *models.RunTraceSpan {
	fmt.Printf("Checking span %s: debugRunID=%v, debugSessionID=%v\n",
		span.SpanID, span.DebugRunID, span.DebugSessionID)

	if span.DebugRunID != nil && *span.DebugRunID == debugRunID {
		fmt.Printf("Found matching debugRunID in span %s\n", span.SpanID)
		return span
	}

	for _, child := range span.ChildrenSpans {
		if found := qr.findSpanByDebugRunID(child, debugRunID); found != nil {
			return found
		}
	}

	return nil
}

// Helper function to find spans by debug session ID recursively
func (qr *queryResolver) findSpansByDebugSessionID(span *models.RunTraceSpan, debugSessionID string) []*models.RunTraceSpan {
	var result []*models.RunTraceSpan

	if span.DebugSessionID != nil && *span.DebugSessionID == debugSessionID {
		result = append(result, span)
	}

	for _, child := range span.ChildrenSpans {
		found := qr.findSpansByDebugSessionID(child, debugSessionID)
		result = append(result, found...)
	}

	return result
}

// Mutation resolver for creating debug sessions
func (mr *mutationResolver) CreateDebugSession(ctx context.Context, input models.CreateDebugSessionInput) (*models.CreateDebugSessionResponse, error) {
	// Generate new ULIDs for debug session and debug run
	debugSessionID := ulid.Make().String()
	debugRunID := ulid.Make().String()

	return &models.CreateDebugSessionResponse{
		DebugSessionID: debugSessionID,
		DebugRunID:     debugRunID,
	}, nil
}
