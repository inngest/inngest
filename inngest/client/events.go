package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
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

type EventQuery struct {
	Name         *string    `json:"name,omitempty"`
	Prefix       *string    `json:"prefix,omitempty"`
	WorkspaceID  *uuid.UUID `json:"workspaceID,omitempty"`
	SchemaSource *string    `json:"schemaSource,omitempty"`
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
