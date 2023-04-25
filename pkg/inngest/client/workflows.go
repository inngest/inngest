package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/internal/cuedefs"
)

// Workflow represents all versions of a single workflow in a workspace.
type Workflow struct {
	ID uuid.UUID `json:"id"`

	// Slug represents the human ID for a workflow, used as the "ID"
	// within a workflow version's configuration.
	Slug string `json:"slug"`
	Name string `json:"name"`

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

	Triggers []WorkflowTrigger `json:"triggers"`
}

type WorkflowTrigger struct {
	EventName *string `json:"eventName"`
	Schedule  *string `json:"schedule"`
}

func (w WorkflowTrigger) String() string {
	if w.EventName != nil {
		return *w.EventName
	}
	if w.Schedule != nil {
		return *w.Schedule
	}
	return ""
}

func (c httpClient) Workflow(ctx context.Context, workspaceID, workflowID uuid.UUID) (*Workflow, error) {
	query := `
	query($workspaceID: ID!, $id: ID!) {
	    workspace(id: $workspaceID) {
	      workflow(id: $id) {
		id name slug
		usage { period range total data { slot count } }
		current { config version description validFrom validTo createdAt updatedAt triggers { eventName schedule }}
		drafts { version description validFrom validTo createdAt updatedAt }
		previous { version description validFrom validTo createdAt updatedAt }
	      }
            }
          }`

	type response struct {
		Workspace struct {
			Workflow *Workflow
		}
	}
	resp, err := c.DoGQL(ctx, Params{Query: query, Variables: map[string]interface{}{"workspaceID": workspaceID, "id": workflowID}})
	if err != nil {
		return nil, err
	}

	data := &response{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling workspaces: %w", err)
	}

	return data.Workspace.Workflow, nil
}

func (c httpClient) LatestWorkflowVersion(ctx context.Context, workspaceID, workflowID uuid.UUID) (*WorkflowVersion, error) {
	query := `
	query($workspaceID: ID!, $id: ID!) {
	    workspace(id: $workspaceID) {
	      workflow(id: $id) {
	        latest {
		  config version description validFrom validTo createdAt updatedAt triggers { eventName schedule }
		}
	      }
            }
          }`

	type response struct {
		Workspace struct {
			Workflow struct {
				Version *WorkflowVersion
			}
		}
	}
	resp, err := c.DoGQL(ctx, Params{Query: query, Variables: map[string]interface{}{
		"workspaceID": workspaceID,
		"id":          workflowID,
	}})
	if err != nil {
		return nil, err
	}
	data := &response{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling workspaces: %w", err)
	}
	return data.Workspace.Workflow.Version, nil
}

func (c httpClient) WorkflowVersion(ctx context.Context, workspaceID, workflowID uuid.UUID, v int) (*WorkflowVersion, error) {
	query := `
	query($workspaceID: ID!, $id: ID!, $version: Int!) {
	    workspace(id: $workspaceID) {
	      workflow(id: $id) {
	        version(id: $version) {
		  config version description validFrom validTo createdAt updatedAt triggers { eventName schedule }
		}
	      }
            }
          }`

	type response struct {
		Workspace struct {
			Workflow struct {
				Version *WorkflowVersion
			}
		}
	}
	resp, err := c.DoGQL(ctx, Params{Query: query, Variables: map[string]interface{}{
		"workspaceID": workspaceID,
		"id":          workflowID,
		"version":     v,
	}})
	if err != nil {
		return nil, err
	}

	data := &response{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling workspaces: %w", err)
	}

	return data.Workspace.Workflow.Version, nil
}

func (c httpClient) Workflows(ctx context.Context, workspaceID uuid.UUID) ([]Workflow, error) {
	query := `
	query($id: ID!, $page: Int) {
	    workspace(id: $id) {
	      workflows @paginated(page: $page) {
		page { page totalPages }
	        data {
		  id name slug
		  usage { period range total data { slot count } }
		  current { config version description validFrom validTo createdAt updatedAt triggers { eventName schedule }}
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
		resp, err := c.DoGQL(ctx, Params{Query: query, Variables: map[string]interface{}{"id": workspaceID, "page": page}})
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

func (c httpClient) DeployWorkflow(ctx context.Context, workspaceID uuid.UUID, config string, live bool) (*WorkflowVersion, error) {
	if _, err := cuedefs.ParseWorkflow(config); err != nil {
		return nil, fmt.Errorf("error parsing workflow: %w", err)
	}

	query := `
	  mutation($input: UpsertWorkflowInput!) {
	    upsertWorkflow(input: $input) {
	      workflow { id name slug }
	      version { config version description validFrom validTo createdAt updatedAt triggers { eventName schedule } }
            }
          }`

	type response struct {
		UpsertWorkflow struct {
			Workflow struct {
				id   uuid.UUID
				Name string
				Slug string
			}
			Version *WorkflowVersion
		}
	}

	resp, err := c.DoGQL(ctx, Params{
		Query: query,
		Variables: map[string]interface{}{
			"input": map[string]interface{}{
				"live":        live,
				"config":      config,
				"workspaceID": workspaceID.String(),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	data := &response{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling workspaces: %w", err)

	}

	return data.UpsertWorkflow.Version, nil
}
