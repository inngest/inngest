package resolvers

import (
	"context"
	"fmt"

	loader "github.com/inngest/inngest/pkg/coreapi/graph/loaders"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/oklog/ulid/v2"
)

func (qr *queryResolver) DebugRun(ctx context.Context, query models.DebugRunQuery) (*models.DebugRun, error) {
	var debugRunSpans []*models.RunTraceSpan

	if query.DebugRunID != nil {
		debugRunID, err := ulid.Parse(*query.DebugRunID)
		if err != nil {
			return nil, fmt.Errorf("invalid debugRunID: %w", err)
		}

		spans, err := loader.LoadOne[[]*models.RunTraceSpan](
			ctx,
			loader.FromCtx(ctx).DebugRunLoader,
			&loader.DebugRunRequestKey{
				DebugRunID: debugRunID,
			},
		)
		if err != nil {
			return nil, err
		}
		if spans != nil {
			debugRunSpans = *spans
		}

	}

	return &models.DebugRun{
		DebugTraces: debugRunSpans,
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
