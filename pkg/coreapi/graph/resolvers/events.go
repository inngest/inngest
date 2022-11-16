package resolvers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
)

func (r *queryResolver) Event(ctx context.Context, query models.EventQuery) (*models.Event, error) {
	return nil, nil
}

// TODO Use a dataloader to retrieve events and fetch individual fields in
// individual resolvers; we shouldn't be mapping any of the fields in this
// query.
func (r *queryResolver) Events(ctx context.Context, query models.EventsQuery) ([]*models.Event, error) {
	evts, err := r.Runner.Events(ctx)
	if err != nil {
		return nil, err
	}

	var events []*models.Event

	for _, evt := range evts {
		createdAt := time.UnixMilli(evt.Timestamp)

		payloadByt, err := json.Marshal(evt.Data)
		if err != nil {
			continue
		}
		payload := string(payloadByt)

		events = append(events, &models.Event{
			ID:        evt.ID,
			Name:      &evt.Name,
			CreatedAt: &createdAt,
			Payload:   &payload,
		})
	}

	return events, nil
}
