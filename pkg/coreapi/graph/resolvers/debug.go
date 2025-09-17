package resolvers

import (
	"context"
	"fmt"

	loader "github.com/inngest/inngest/pkg/coreapi/graph/loaders"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
)

func (qr *queryResolver) DebugRun(ctx context.Context, query models.DebugRunQuery) (*models.DebugRun, error) {
	var debugRunSpan *models.RunTraceSpan
	var originalRunSpan *models.RunTraceSpan

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

	}

	if query.RunID != nil {
		runID, err := ulid.Parse(*query.RunID)
		if err != nil {
			return nil, fmt.Errorf("invalid runID: %w", err)
		}

		traceRun, err := qr.Data.GetTraceRun(ctx, cqrs.TraceRunIdentifier{RunID: runID})
		if err != nil {
			return nil, fmt.Errorf("error retrieving original trace run: %w", err)
		}

		originalRunSpan, err = loader.LoadOne[models.RunTraceSpan](
			ctx,
			loader.FromCtx(ctx).RunTraceLoader,
			&loader.TraceRequestKey{
				TraceRunIdentifier: &cqrs.TraceRunIdentifier{
					FunctionID: traceRun.FunctionID,
					RunID:      runID,
					TraceID:    traceRun.TraceID,
				},
			},
		)
		if err != nil {
			return nil, fmt.Errorf("error getting original trace run span: %w", err)
		}
	}

	return &models.DebugRun{
		DebugRun:    debugRunSpan,
		OriginalRun: originalRunSpan,
	}, nil
}

func (qr *queryResolver) DebugSession(ctx context.Context, query models.DebugSessionQuery) (*models.DebugSession, error) {
	if query.DebugSessionID != nil {
		debugSessionID, err := ulid.Parse(*query.DebugSessionID)
		if err != nil {
			return nil, fmt.Errorf("invalid debugSessionID: %w", err)
		}

		debugSession, err := loader.LoadOne[*models.DebugSession](
			ctx,
			loader.FromCtx(ctx).DebugSessionLoader,
			&loader.DebugSessionRequestKey{
				DebugSessionID: debugSessionID,
			},
		)
		if err != nil {
			return nil, err
		}

		return *debugSession, nil
	}

	return nil, fmt.Errorf("debug session not found")
}

func (r *mutationResolver) CreateDebugSession(ctx context.Context, input models.CreateDebugSessionInput) (*models.CreateDebugSessionResponse, error) {
	return &models.CreateDebugSessionResponse{
		DebugSessionID: ulid.Make(),
		DebugRunID:     ulid.Make(),
	}, nil
}
