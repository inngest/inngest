package resolvers

import (
	"context"
	"encoding/json"
	"sort"
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
		ID:        evt.EventID,
		Name:      &evt.EventName,
		CreatedAt: &createdAt,
		Payload:   &payload,
	}, nil
}

// TODO Use a dataloader to retrieve events and fetch individual fields in
// individual resolvers; we shouldn't be mapping any of the fields in this
// query.
func (r *queryResolver) Events(ctx context.Context, query models.EventsQuery) ([]*models.Event, error) {
	evts, err := r.Runner.Events(ctx, "")
	if err != nil {
		return nil, err
	}

	var events []*models.Event

	for _, evt := range evts {
		name := evt.Name

		createdAt := time.UnixMilli(evt.Timestamp)
		if evt.Timestamp == 0 {
			if id, err := ulid.Parse(evt.ID); err == nil {
				createdAt = ulid.Time(id.Time())
			}
		}

		payloadByt, err := json.Marshal(evt.Data)
		if err != nil {
			continue
		}
		payload := string(payloadByt)

		events = append(events, &models.Event{
			ID:        evt.ID,
			Name:      &name,
			CreatedAt: &createdAt,
			Payload:   &payload,
		})
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].ID > events[j].ID
	})

	return events, nil
}
