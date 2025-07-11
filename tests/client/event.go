package client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/99designs/gqlgen/graphql"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/oklog/ulid/v2"
)

type GetEventsOpts struct {
	PageSize int
	Cursor   *string
	Filter   models.EventsFilter
}

func (c *Client) GetEvents(ctx context.Context, opts GetEventsOpts) (*models.EventsConnection, error) {
	c.Helper()

	query := `
		query GetEventsV2($pageSize: Int!, $cursor: String, $startTime: Time!, $endTime: Time, $celQuery: String = null, $eventNames: [String!] = null, $includeInternalEvents: Boolean = true){
		  eventsV2(
			first: $pageSize
			after: $cursor
			filter: {from: $startTime, until: $endTime, query: $celQuery, eventNames: $eventNames, includeInternalEvents: $includeInternalEvents}
		  ) {
			edges {
			  node {
				name
				id
				occurredAt
				receivedAt
				raw
				runs {
				  status
				  id
				  startedAt
				  endedAt
				  function {
					name
					slug
				  }
				}
			  }
			}
			totalCount
			pageInfo {
			  hasNextPage
			  endCursor
			  hasPreviousPage
			  startCursor
			}
		  }
		}
	`

	resp, err := c.DoGQL(ctx, graphql.RawParams{
		Query: query,
		Variables: map[string]any{
			"pageSize":              opts.PageSize,
			"cursor":                opts.Cursor,
			"startTime":             opts.Filter.From,
			"endTime":               opts.Filter.Until,
			"celQuery":              opts.Filter.Query,
			"eventNames":            opts.Filter.EventNames,
			"includeInternalEvents": opts.Filter.IncludeInternalEvents,
		},
	})

	if err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("err with gql: %#v", resp.Errors)
	}

	type response struct {
		EventsV2 *models.EventsConnection `json:"eventsV2"`
	}

	data := &response{}
	if err := json.Unmarshal(resp.Data, data); err != nil {
		return nil, err
	}

	return data.EventsV2, nil
}

func (c *Client) GetEvent(ctx context.Context, id ulid.ULID) (*models.EventV2, error) {
	c.Helper()

	query := `
		query GetEvent($id: ULID!) {
			eventV2(id: $id) {
				id
				idempotencyKey
				name
				occurredAt
				raw
				receivedAt
				runs {
					id
					function {
						name
					}
					status
				}
				version
			}
		}`

	resp, err := c.DoGQL(ctx, graphql.RawParams{
		Query: query,
		Variables: map[string]any{
			"id": id.String(),
		},
	})

	if err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("err with gql: %#v", resp.Errors)
	}

	type response struct {
		EventV2 models.EventV2 `json:"eventV2"`
	}

	data := &response{}
	if err := json.Unmarshal(resp.Data, data); err != nil {
		return nil, err
	}

	return &data.EventV2, nil
}
