package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Workflow represents all versions of a single workflow in a workspace.
type Workflow struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`

	Usage Usage `json:"usage"`

	Current  *WorkflowVersion  `json:"current"`
	Drafts   []WorkflowVersion `json:"drafts"`
	Previous []WorkflowVersion `json:"previous"`
}

type WorkflowVersion struct {
	Version     int    `json:"version"`
	Description string `json:"description"`
	Config      string `json:"config"`

	ValidFrom *time.Time `json:"validFrom"`
	ValidTo   *time.Time `json:"validTo"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (c httpClient) Workflows(ctx context.Context, workflowID uuid.UUID) ([]Workflow, error) {
	query := `
	query($id: ID!, $page: Int) {
	    workspace(id: $id) {
	      workflows @paginated(page: $page) {
		page { page totalPages }
	        data {
		  id name 
		  usage { period range total data { slot count } }
		  current { config version description validFrom validTo createdAt updatedAt }
		  drafts { version description validFrom validTo createdAt updatedAt }
		  previous { version description validFrom validTo createdAt updatedAt }
	        }
	      }
            }
          }`

	workflows := []Workflow{}

	type response struct {
		Workspace struct {
			Workflows struct {
				Page struct {
					Page       int
					TotalPages int
				}
				Data []Workflow
			}
		}
	}

	page := 0

	all := false
	for !all {
		resp, err := c.DoGQL(ctx, Params{Query: query, Variables: map[string]interface{}{"id": workflowID, "page": page}})
		if err != nil {
			return nil, err
		}

		data := &response{}
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return nil, fmt.Errorf("error unmarshalling workspaces: %w", err)

		}

		workflows = append(workflows, data.Workspace.Workflows.Data...)

		if data.Workspace.Workflows.Page.Page == data.Workspace.Workflows.Page.TotalPages {
			all = true
		}

		page += 1
	}

	return workflows, nil
}
