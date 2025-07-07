package client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/99designs/gqlgen/graphql"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
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
