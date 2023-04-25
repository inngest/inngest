package client

import (
	"context"
	"net/http"
	"os"

	"github.com/google/uuid"
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
