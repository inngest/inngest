package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest-cli/pkg/event"
	"github.com/oklog/ulid/v2"
)

type PaginatedEvents struct {
	Page struct {
		Cursor string
	}
	Data []Event
}

type Event struct {
	Name            string
	Description     string
	IntegrationName string
	SchemaSource    string
	VersionCount    *int
	WorkspaceID     *uuid.UUID
	FirstSeen       time.Time

	Versions []EventVersion
}

type EventVersion struct {
	Name    string
	Version string
	CueType string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type ArchivedEvent struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Event     string `json:"event,omitempty"`
	Timestamp string `json:"occurredAt,omitempty"`
	Version   string `json:"version,omitempty"`
}

func (e ArchivedEvent) MarshalToEvent() (*event.Event, error) {
	evt := &event.Event{}

	if err := json.Unmarshal([]byte(e.Event), evt); err != nil {
		return nil, err
	}

	return evt, nil
}

type EventQuery struct {
	Name         *string    `json:"name,omitempty"`
	Prefix       *string    `json:"prefix,omitempty"`
	WorkspaceID  *uuid.UUID `json:"workspaceID,omitempty"`
	SchemaSource *string    `json:"schemaSource,omitempty"`
}

type RecentEventsQuery struct {
	WorkspaceID uuid.UUID
	EventID     ulid.ULID
	EventName   string
	Count       int
}

type Cursor struct {
	PerPage int
	Cursor  *string
}

func (c httpClient) Events(ctx context.Context, query *EventQuery, cursor *Cursor) (*PaginatedEvents, error) {
	gql := `
          query ($query: EventQuery, $perPage: Int, $cursor: String) {
	    events(query: $query) @cursored(cursor: $cursor, perPage: $perPage) {
	      page { cursor }
	      data {
	        name versionCount workspaceID schemaSource integrationName firstSeen
	        versions {
	          name version cueType
	        }
	      }
            }
          }`

	vars := map[string]interface{}{
		"query": query,
	}
	if cursor != nil {
		vars["perPage"] = cursor.PerPage
		vars["cursor"] = cursor.Cursor
	}

	resp, err := c.DoGQL(ctx, Params{Query: gql, Variables: vars})
	if err != nil {
		return nil, err
	}

	type response struct {
		Events PaginatedEvents
	}

	data := &response{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %w", err)
	}

	return &data.Events, nil
}

func (c httpClient) AllEvents(ctx context.Context, query *EventQuery) ([]Event, error) {
	evts := []Event{}

	// Fetch 100 events at a time.
	cursor := &Cursor{
		PerPage: 100,
	}

	// Fetch a maximum of 1000 events.
	max := 10
	for i := 0; i < max; i++ {
		result, err := c.Events(ctx, query, cursor)
		if err != nil {
			return nil, err
		}
		evts = append(evts, result.Data...)
		if len(result.Data) < cursor.PerPage {
			return evts, nil
		}
		cursor.Cursor = &result.Page.Cursor
	}

	return evts, nil
}

func (c httpClient) RecentEvent(ctx context.Context, workspaceID uuid.UUID, eventID ulid.ULID) (*ArchivedEvent, error) {
	query := `
		query RecentEvent($workspaceId: ID!, $archivedEventId: ULID!) {
			workspace(id: $workspaceId) {
				archivedEvent(id: $archivedEventId) {
					id
					name
					event
				}
			}
		}
	`

	type response struct {
		Workspace struct {
			ArchivedEvent *ArchivedEvent
		}
	}

	resp, err := c.DoGQL(ctx, Params{Query: query, Variables: map[string]interface{}{
		"workspaceId":     workspaceID,
		"archivedEventId": eventID.String(),
	}})
	if err != nil {
		return nil, err
	}

	data := &response{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %w", err)
	}

	return data.Workspace.ArchivedEvent, nil

}

func (c httpClient) RecentEvents(ctx context.Context, workspaceID uuid.UUID, triggerName string, count int64) ([]ArchivedEvent, error) {
	query := `
		query RecentEvents($workspaceId: ID!, $name: String!, $count: Int) {
			workspace(id: $workspaceId) {
				event(name: $name) {
					recent(count: $count) {
						id
						name
						event
						occurredAt
						version
					}
				}
			}
		}
	`

	type response struct {
		Workspace struct {
			Event struct {
				Recent []ArchivedEvent
			}
		}
	}

	resp, err := c.DoGQL(ctx, Params{Query: query, Variables: map[string]interface{}{
		"workspaceId": workspaceID,
		"name":        triggerName,
		"count":       count,
	}})
	if err != nil {
		return nil, err
	}

	data := &response{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %w", err)
	}

	return data.Workspace.Event.Recent, nil
}
