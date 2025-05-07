package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/inngest/inngest/pkg/cqrs"
)

type GetEventsOpts struct {
	After  *string
	Filter cqrs.EventsFilter
}

func (c *Client) GetEvents(
	ctx context.Context,
	opts ...GetEventsOpts,
) (*cqrs.EventsConnection, error) {
	var o GetEventsOpts
	if len(opts) > 0 {
		o = opts[0]
	}

	query := `
		query Q($after: String, $filter: EventsFilter!) {
			eventsV2(after: $after, filter: $filter) {
				edges {
					cursor
					node {
						id
						idempotencyKey
						name
						occurredAt
						receivedAt
						version
					}
				}
				pageInfo {
					endCursor
					hasNextPage
					startCursor
				}
			}
		}`

	resp, err := c.DoGQL(ctx, graphql.RawParams{
		Query: query,
		Variables: map[string]any{
			"after":  o.After,
			"filter": o.Filter,
		},
	})
	if err != nil {
		return nil, err
	}
	if resp.Errors != nil {
		return nil, resp.Errors
	}

	var data struct {
		EventsV2 cqrs.EventsConnection `json:"eventsV2"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %w", err)
	}

	return &data.EventsV2, nil
}
