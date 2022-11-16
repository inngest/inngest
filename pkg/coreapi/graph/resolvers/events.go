package resolvers

import (
	"context"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
)

func (r *queryResolver) EventTimeline(ctx context.Context, query models.EventTimelineQuery) (*models.EventTimeline, error) {
	return nil, nil
}

func (r *queryResolver) Events(ctx context.Context, query models.EventsQuery) ([]*models.Event, error) {
	return nil, nil
}
