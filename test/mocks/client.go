package mocks

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/inngest/client"
)

func NewMockClient() client.Client {
	return &MockClient{
		actions: make(map[string]*inngest.Action),
	}
}

type MockClient struct {
	actions map[string]*inngest.Action
}

func (mc *MockClient) Credentials() []byte { return nil }

func (mc *MockClient) Login(ctx context.Context, email, password string) ([]byte, error) {
	return nil, nil
}
func (mc *MockClient) Account(ctx context.Context) (*client.Account, error)       { return nil, nil }
func (mc *MockClient) Workspaces(ctx context.Context) ([]client.Workspace, error) { return nil, nil }

func (mc *MockClient) AllEvents(ctx context.Context, query *client.EventQuery) ([]client.Event, error) {
	return nil, nil
}
func (mc *MockClient) Events(ctx context.Context, query *client.EventQuery, cursor *client.Cursor) (*client.PaginatedEvents, error) {
	return nil, nil
}

// Workflows lists all workflows in a given workspace
func (mc *MockClient) Workflows(ctx context.Context, workspaceID uuid.UUID) ([]client.Workflow, error) {
	return nil, nil
}

// Workflow returns a specific workflow by ID
func (mc *MockClient) Workflow(ctx context.Context, workspaceID, workflowID uuid.UUID) (*client.Workflow, error) {
	return nil, nil
}

// WorkflowVersion returns a specific workflow version for a given workflow
func (mc *MockClient) WorkflowVersion(ctx context.Context, workspaceID, workflowID uuid.UUID, version int) (*client.WorkflowVersion, error) {
	return nil, nil
}

// LatestWorkflowVersion returns the latest workflow version by modification date for a given workflow
func (mc *MockClient) LatestWorkflowVersion(ctx context.Context, workspaceID, workflowID uuid.UUID) (*client.WorkflowVersion, error) {
	return nil, nil
}

// DeployWorflow idempotently deploys a workflow, by default as a draft.  Set live to true to deploy as the live version.
func (mc *MockClient) DeployWorkflow(ctx context.Context, workspaceID uuid.UUID, config string, live bool) (*client.WorkflowVersion, error) {
	return nil, nil
}

func (mc *MockClient) Actions(ctx context.Context, includePublic bool) ([]*client.Action, error) {
	return nil, nil
}
func (mc *MockClient) UpdateActionVersion(ctx context.Context, v client.ActionVersionQualifier, enabled bool) (*client.ActionVersion, error) {
	return nil, nil
}
func (mc *MockClient) CreateAction(ctx context.Context, config string) (*client.Action, error) {
	return nil, nil
}

// Action returns a specific action based on DSN
func (mc *MockClient) Action(ctx context.Context, dsn string) (*client.Action, error) {
	return nil, errors.New("action not found")
}
