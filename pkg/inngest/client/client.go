package client

import (
	"context"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/inngest/inngest/inngest"
	"github.com/oklog/ulid/v2"
)

const (
	InngestCloudAPI = "https://api.inngest.com"
)

// Client implements all functionality necessary to communicate with
// the inngest server.
type Client interface {
	Credentials() []byte
	IsCloudAPI() bool

	Login(ctx context.Context, email, password string) ([]byte, error)

	StartDeviceLogin(ctx context.Context, clientID uuid.UUID) (*StartDeviceLoginResponse, error)
	PollDeviceLogin(ctx context.Context, clientID uuid.UUID, deviceCode uuid.UUID) (*DeviceLoginResponse, error)

	Account(ctx context.Context) (*Account, error)
	Workspaces(ctx context.Context) ([]Workspace, error)

	AllEvents(ctx context.Context, query *EventQuery) ([]Event, error)
	Events(ctx context.Context, query *EventQuery, cursor *Cursor) (*PaginatedEvents, error)
	// Fetch a single event from the event store with ID `eventId`.
	RecentEvent(ctx context.Context, workspaceID uuid.UUID, eventID ulid.ULID) (*ArchivedEvent, error)
	// Fetch `count` latest `triggerName` events from the event store.
	RecentEvents(ctx context.Context, workspaceID uuid.UUID, triggerName string, count int64) ([]ArchivedEvent, error)

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

	// DeployFunction deploys a function for a given environment. Live determines if the function is a draft or live.
	DeployFunction(ctx context.Context, config string, env string, live bool) (*FunctionVersion, error)

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
	api := InngestCloudAPI
	if os.Getenv("INNGEST_API") != "" {
		api = os.Getenv("INNGEST_API")
	}
	// XXX: this enables us to use different queries for the self hosted API & the Cloud API
	// until we meet full compatibility
	isCloudAPI := true
	if api != InngestCloudAPI && api != "http://localhost:8090" {
		isCloudAPI = false
	}

	c := &httpClient{
		Client:     http.DefaultClient,
		api:        api,
		isCloudAPI: isCloudAPI,
		ingest:     "https://inn.gs",
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

	api        string
	isCloudAPI bool
	ingest     string
	creds      []byte
}

func (c httpClient) Credentials() []byte {
	return c.creds
}

func (c httpClient) IsCloudAPI() bool {
	return c.isCloudAPI
}
