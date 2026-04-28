package resolvers

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
)

func (qr *queryResolver) Event(ctx context.Context, query models.EventQuery) (*models.Event, error) {
	id, _ := ulid.Parse(query.EventID)

	evt, err := qr.Data.GetEventByInternalID(ctx, id)
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

	var externalID *string
	if evt.EventID != "" {
		externalID = &evt.EventID
	}

	return &models.Event{
		ID:         evt.InternalID(),
		ExternalID: externalID,
		Name:       &evt.EventName,
		CreatedAt:  &createdAt,
		Payload:    &payload,
	}, nil
}

// deprecated
func (qr *queryResolver) Events(ctx context.Context, query models.EventsQuery) ([]*models.Event, error) {
	opts := &cqrs.WorkspaceEventsOpts{
		Limit: cqrs.MaxEvents,
	}

	loaded, err := qr.Data.GetEvents(ctx, consts.DevServerAccountID, consts.DevServerEnvID, opts)
	if err != nil {
		return nil, err
	}

	var events []*models.Event

	for _, evt := range loaded {
		name := evt.EventName

		createdAt := time.UnixMilli(evt.EventTS)
		if evt.EventTS == 0 {
			if id, err := ulid.Parse(evt.EventID); err == nil {
				createdAt = ulid.Time(id.Time())
			}
		}

		payloadByt, err := json.Marshal(evt.EventData)
		if err != nil {
			continue
		}
		payload := string(payloadByt)

		internalID, err := ulid.Parse(evt.EventID)
		if err != nil {
			continue
		}

		events = append(events, &models.Event{
			ID:        internalID,
			Name:      &name,
			CreatedAt: &createdAt,
			Payload:   &payload,
		})
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].ID.String() > events[j].ID.String()
	})

	return events, nil
}
