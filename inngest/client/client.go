package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/inngest/inngestctl/inngest"
)

// Client implements all functionality necessary to communicate with
// the inngest server.
type Client interface {
	Credentials() []byte

	Login(ctx context.Context, email, password string) ([]byte, error)
	Account(ctx context.Context) (*Account, error)
	Workspaces(ctx context.Context) ([]Workspace, error)

	AllEvents(ctx context.Context, query *EventQuery) ([]Event, error)
	Events(ctx context.Context, query *EventQuery, cursor *Cursor) (*PaginatedEvents, error)

	// Workflows lists all workflows in a given workspace
	Workflows(ctx context.Context, workspaceID uuid.UUID) ([]Workflow, error)
	// Workflow returns a specific workflow by ID
	Workflow(ctx context.Context, workspaceID, workflowID uuid.UUID) (*Workflow, error)
	// WorkflowVersion returns a specific workflow version for a given workflow
	WorkflowVersion(ctx context.Context, workspaceID, workflowID uuid.UUID, version int) (*WorkflowVersion, error)
	// LatestWorkflowVersion returns the latest workflow version by modification date for a given workflow
	LatestWorkflowVersion(ctx context.Context, workspaceID, workflowID uuid.UUID) (*WorkflowVersion, error)
	// DeployWorflow idempotently deploys a workflow, by default as a draft.  Set live to true to deploy as the live version.
	DeployWorkflow(ctx context.Context, workspaceID uuid.UUID, config string, live bool) (*WorkflowVersion, error)

	// Action returns a single action by DSN.  If no version is specified, this will return the latest
	// major/minor version.  If a major version is supplied with no minor version, this will return the
	// latest minor version for the gievn major version.  If both are supplied, this will return the
	// specific version requested.
	Action(ctx context.Context, dsn string, v *inngest.VersionInfo) (*ActionVersion, error)
	// Acitons returns all actions with their latest versions.
	Actions(ctx context.Context, includePublic bool) ([]*Action, error)
	// UpdateActionVersion updates the given action version, enabling or disbaling the action version.
	UpdateActionVersion(ctx context.Context, v ActionVersionQualifier, enabled bool) (*ActionVersion, error)
	// CreateAction creates a new action in your account.
	CreateAction(ctx context.Context, config string) (*Action, error)
}

type ClientOpt func(Client) Client

func New(opts ...ClientOpt) Client {
	api := "https://api.inngest.com"
	if os.Getenv("INNGEST_API") != "" {
		api = os.Getenv("INNGEST_API")
	}

	c := &httpClient{
		Client: http.DefaultClient,
		api:    api,
		ingest: "https://inn.gs",
	}

	for _, o := range opts {
		c = o(c).(*httpClient)
	}

	return c
}

// WithCredentials is used to configure a client with a given API host.
func WithAPI(api string) ClientOpt {
	return func(c Client) Client {
		if api == "" {
			return c
		}

		client := c.(*httpClient)
		client.api = api
		return client
	}
}

// WithCredentials is used to configure a client with a given JWT.
func WithCredentials(creds []byte) ClientOpt {
	return func(c Client) Client {
		client := c.(*httpClient)
		client.creds = creds
		return client
	}
}

// httpClient represents a concrete HTTP implementation of a Client
type httpClient struct {
	*http.Client

	api    string
	ingest string
	creds  []byte
}

func (c httpClient) Credentials() []byte {
	return c.creds
}

func (c httpClient) Login(ctx context.Context, email, password string) ([]byte, error) {
	input := map[string]string{
		"email":    email,
		"password": password,
	}
	buf := jsonBuffer(ctx, input)

	req, err := c.NewRequest(http.MethodPost, "/v1/login", buf)
	if err != nil {
		return nil, fmt.Errorf("error creating login request: %s", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error performing login request: %s", err)
	}
	defer resp.Body.Close()
	byt, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %s", err)
	}

	type response struct {
		Error string
		JWT   string
	}

	r := &response{}
	if err = json.Unmarshal(byt, r); err != nil {
		return nil, fmt.Errorf("invalid json response: %w: \n%s", err, string(byt))
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s", r.Error)
	}

	return []byte(r.JWT), nil
}
