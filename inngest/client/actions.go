package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type Action struct {
	DSN  string
	Name string
}

func (c httpClient) Actions(ctx context.Context, workspaceID uuid.UUID) error {
	// TODO
	return nil
}

func (c httpClient) CreateAction(ctx context.Context, input string) (*Action, error) {
	query := `
	  mutation CreateAction($config: String!) {
	    createAction(config: $config) {
	      dsn name
            }
          }`

	resp, err := c.DoGQL(ctx, Params{Query: query})
	if err != nil {
		return nil, err
	}

	type response struct {
		CreateAction *Action
	}

	data := &response{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %w", err)
	}

	return data.CreateAction, nil
}
