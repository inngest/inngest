package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// Workspace represents a single workspace within an Inngest account. The pertinent
// fields for the active workspace are marshalled into State.
type Workspace struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Test bool      `json:"test"`
}

func (c httpClient) Workspaces(ctx context.Context) ([]Workspace, error) {
	query := `
          query {
            workspaces {
	      id name test
            }
          }`

	resp, err := c.DoGQL(ctx, Params{Query: query})
	if err != nil {
		return nil, err
	}

	type response struct {
		Workspaces []Workspace
	}

	data := &response{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling workspaces: %w", err)
	}

	return data.Workspaces, nil
}
