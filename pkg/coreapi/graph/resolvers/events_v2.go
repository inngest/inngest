package resolvers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	loader "github.com/inngest/inngest/pkg/coreapi/graph/loaders"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"

	"github.com/graph-gophers/dataloader"
	"github.com/oklog/ulid/v2"
)

const defaultPageSize = 40

func (qr *queryResolver) EventsV2(ctx context.Context, first int, after *string, filter models.EventsFilter) (*models.EventsConnection, error) {
	pageSize := defaultPageSize
	if first > 0 && first <= cqrs.MaxEvents {
		pageSize = first
	}

	opts := &cqrs.WorkspaceEventsOpts{
		Limit: pageSize,
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

	opts.IncludeInternalEvents = filter.IncludeInternalEvents

	opts.Oldest = filter.From

	// If we don't have a cursor, fetch the latest events
	// otherwise this has no impact
	opts.Newest = time.Now()

	if filter.Until != nil {
		opts.Newest = *filter.Until
	}

	events, err := qr.Data.GetEvents(ctx, consts.DevServerAccountID, consts.DevServerEnvID, opts)
	if err != nil {
		return nil, err
	}

	targetLoader := loader.FromCtx(ctx).EventLoader

	eventEdges := []*models.EventsEdge{}
	for _, e := range events {
		targetLoader.Prime(ctx, dataloader.StringKey(e.InternalID().String()), e)

		eventV2 := cqrsEventToGQLEvent(e)

		cursorByt, err := json.Marshal(EventsV2ConnectionCursor{ID: eventV2.ID.String()})
		if err != nil {
			return nil, err
		}

		eventEdges = append(eventEdges, &models.EventsEdge{
			Node:   eventV2,
			Cursor: base64.StdEncoding.EncodeToString(cursorByt),
		})
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
	}, nil
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

func cqrsEventToGQLEvent(e *cqrs.Event) *models.EventV2 {
	eventV2 := models.EventV2{
		EnvID:          e.WorkspaceID,
		ID:             e.InternalID(),
		IdempotencyKey: &e.EventID,
		Name:           e.EventName,
		OccurredAt:     time.UnixMilli(e.EventTS),
		ReceivedAt:     e.ReceivedAt,
		Version:        &e.EventVersion,
	}

	if e.SourceID != nil {
		eventV2.Source = &models.EventSource{
			ID:         e.SourceID.String(),
			Name:       &e.Source,
			SourceKind: "TODO",
		}
	}

	return &eventV2
}
