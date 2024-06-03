package client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/99designs/gqlgen/graphql"
)

type Function struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (c *Client) Functions(ctx context.Context) ([]Function, error) {
	c.Helper()

	query := `
		query {
			functions {
				id
				name
			}
		}`

	resp := c.MustDoGQL(ctx, graphql.RawParams{
		Query: query,
	})
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("err with gql: %#v", resp.Errors)
	}

	type response struct {
		Functions []Function `json:"functions"`
	}

	data := &response{}
	if err := json.Unmarshal(resp.Data, data); err != nil {
		return nil, err
	}

	return data.Functions, nil
}
