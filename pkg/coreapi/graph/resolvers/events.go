package resolvers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/oklog/ulid/v2"
)

func (r *queryResolver) Event(ctx context.Context, query models.EventQuery) (*models.Event, error) {
	id, _ := ulid.Parse(query.EventID)

	evt, err := r.Data.GetEventByInternalID(ctx, id)
	if err != nil {
		return nil, err
	}

	payloadByt, err := json.Marshal(evt.EventData)
	if err != nil {
		return nil, err
	}
	payload := string(payloadByt)

	createdAt := time.UnixMilli(evt.EventTS)
	if evt.EventTS == 0 {
		createdAt = ulid.Time(evt.ID.Time())
	}

	return &models.Event{
		ID:        evt.InternalID(),
		Name:      &evt.EventName,
		CreatedAt: &createdAt,
		Payload:   &payload,
		Raw:       &payload,
	}, nil
}
