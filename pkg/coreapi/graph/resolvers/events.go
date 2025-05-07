package resolvers

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

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

// TODO Use a dataloader to retrieve events and fetch individual fields in
// individual resolvers; we shouldn't be mapping any of the fields in this
// query.
func (qr *queryResolver) Events(ctx context.Context, query models.EventsQuery) ([]*models.Event, error) {
	evts, err := qr.Runner.Events(ctx, "")
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

		internalID, err := ulid.Parse(evt.ID)
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

func (qr *queryResolver) EventsV2(
	ctx context.Context,
	first int,
	after *string,
	filter cqrs.EventsFilter,
) (*models.EventsConnection, error) {
	if first < 1 {
		first = defaultPageSize
	} else if first > maxPageSize {
		return nil, fmt.Errorf("first must be less than %d", maxPageSize)
	}

	evts, err := qr.Data.GetEvents(
		ctx,
		cqrs.GetEventsOpts{
			Cursor: after,
			Filter: filter,
			Limit:  first,
		},
	)
	if err != nil {
		return nil, err
	}

	edges := make([]*models.EventsEdge, len(evts))
	for i, evt := range evts {
		var idempotencyKey *string
		if evt.EventID != "" && evt.EventID != evt.ID.String() {
			idempotencyKey = &evt.EventID
		}

		var version *string
		if evt.EventVersion != "" {
			version = &evt.EventVersion
		}

		edges[i] = &models.EventsEdge{
			Cursor: evt.ID.String(),
			Node: &models.EventV2{
				EnvID:          evt.WorkspaceID,
				ID:             evt.ID,
				IdempotencyKey: idempotencyKey,
				Name:           evt.EventName,
				OccurredAt:     evt.ReceivedAt,
				ReceivedAt:     evt.ReceivedAt,
				Version:        version,
			},
		}
	}

	var (
		startCursor *string
		endCursor   *string
	)
	if len(evts) > 0 {
		startCursor = &evts[0].Cursor
		endCursor = &evts[len(evts)-1].Cursor
	}

	return &models.EventsConnection{
		Edges: edges,
		PageInfo: &models.PageInfo{
			EndCursor:       endCursor,
			HasNextPage:     len(evts) == first,
			HasPreviousPage: startCursor != nil,
			StartCursor:     startCursor,
		},
		TotalCount: len(evts),
	}, nil
}
