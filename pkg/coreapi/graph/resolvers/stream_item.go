package resolvers

import (
	"context"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/oklog/ulid/v2"
)

func (r *streamItemResolver) InBatch(ctx context.Context, q *models.StreamItem) (bool, error) {
	eventID, err := ulid.Parse(q.ID)
	if err != nil {
		return false, err
	}

	batches, err := r.Data.GetEventBatchesByEventID(ctx, eventID)
	if err != nil {
		return false, err
	}

	return len(batches) > 0, nil
}
