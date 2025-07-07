package resolvers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
)

func (qr *queryResolver) EventsV2(ctx context.Context, first int, after *string, filter models.EventsFilter) (*models.EventsConnection, error) {
	// TODO limit first

	opts := &cqrs.WorkspaceEventsOpts{
		Limit: first,
		Names: filter.EventNames,
	}

	cursor := &EventsV2ConnectionCursor{}
	if after != nil && *after != "" {
		if err := cursor.Decode(*after); err != nil {
			return nil, fmt.Errorf("error decoding eventsv2 cursor: %w", err)
		}
		if cursor.ID != "" {
			parsed, err := ulid.Parse(cursor.ID)
			if err != nil {
				return nil, fmt.Errorf("error parsing eventsv2 cursor: %w", err)
			}
			opts.Cursor = &parsed
		}
	}

	opts.Oldest = filter.From

	opts.Newest = time.Now() // TODO: this is slightly problematic for total count as user pages through results
	if filter.Until != nil {
		opts.Newest = *filter.Until
	}

	events, err := qr.Data.GetEvents(ctx, consts.DevServerAccountID, consts.DevServerEnvID, opts)
	if err != nil {
		return nil, err
	}

	eventEdges := []*models.EventsEdge{}
	for _, e := range events {

		raw, err := marshalRaw(e)
		if err != nil {
			return nil, err
		}

		eventV2 := models.EventV2{
			EnvID:          e.WorkspaceID,
			ID:             e.InternalID(),
			IdempotencyKey: &e.EventID,
			Name:           e.EventName,
			OccurredAt:     time.UnixMilli(e.EventTS),
			Raw:            raw,
			ReceivedAt:     e.ReceivedAt,
			Runs:           []*models.FunctionRunV2{}, // TODO
			Version:        &e.EventVersion,
		}

		if e.SourceID != nil {
			eventV2.Source = &models.EventSource{
				ID:         e.SourceID.String(),
				Name:       &e.Source,
				SourceKind: "TODO",
			}
		}

		cursorByt, err := json.Marshal(EventsV2ConnectionCursor{ID: eventV2.ID.String()})
		if err != nil {
			return nil, err
		}

		eventEdges = append(eventEdges, &models.EventsEdge{
			Node:   &eventV2,
			Cursor: base64.StdEncoding.EncodeToString(cursorByt),
		})
	}

	totalCount, err := qr.Data.GetEventsCount(ctx, consts.DevServerAccountID, consts.DevServerEnvID, opts)
	if err != nil {
		return nil, err
	}

	// somewhat inaccurate if the last page has exactly the number of entries requested,
	// but the next page will just be a page with 0 and hasNextPage = false
	hasNextPage := len(eventEdges) == first

	var startCursor, endCursor *string
	if len(eventEdges) > 0 {
		startCursor = &(eventEdges[0].Cursor)
		endCursor = &(eventEdges[len(eventEdges)-1].Cursor)
	}

	return &models.EventsConnection{
		Edges: eventEdges,
		PageInfo: &models.PageInfo{
			HasNextPage:     hasNextPage,
			HasPreviousPage: after != nil, // accurate as long as clients always send null cursor for first page
			StartCursor:     startCursor,
			EndCursor:       endCursor,
		},
		TotalCount: int(totalCount),
	}, nil
}

func marshalRaw(e *cqrs.Event) (string, error) {
	data := e.EventData
	if data == nil {
		data = make(map[string]any)
	}

	var version *string
	if len(e.EventVersion) > 0 {
		version = &e.EventVersion
	}

	id := e.InternalID().String()
	if len(e.EventID) > 0 {
		id = e.EventID
	}

	byt, err := json.Marshal(map[string]any{
		"data": data,
		"id":   id,
		"name": e.EventName,
		"ts":   e.EventTS,
		"v":    version,
	})
	if err != nil {
		return "", err
	}
	return string(byt), nil

}

func (c *EventsV2ConnectionCursor) Decode(val string) error {
	byt, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		return err
	}
	return json.Unmarshal(byt, c)
}

type EventsV2ConnectionCursor struct {
	ID string
}
