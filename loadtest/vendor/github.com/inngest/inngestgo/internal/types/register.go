package types

import "github.com/inngest/inngestgo/internal/fn"

type DeployType string

const (
	DeployTypePing    DeployType = "ping"
	DeployTypeConnect DeployType = "connect"
)

const (
	InBandSyncV1 string = "v1"
	TrustProbeV1 string = "v1"
	ConnectV1    string = "v1"
)

// RegisterRequest represents a new deploy request from SDK-based functions.
// This lets us know that a new deploy has started and that we need to
// upsert function definitions from the newly deployed endpoint.
type RegisterRequest struct {
	// V is the version for this response, which lets us upgrade the SDKs
	// and APIs with backwards compatiblity.
	V string `json:"v"`
	// URL represents the entire URL which hosts the functions, eg.
	// https://www.example.com/api/v1/inngest
	URL string `json:"url"`
	// DeployType represents how this was deployed, eg. via a ping.
	// This allows us to change flows in the future, or support
	// multiple registration flows within a single fetch response.
	DeployType DeployType `json:"deployType"`
	// SDK represents the SDK language and version used for these
	// functions, in the format: "js:v0.1.0"
	SDK string `json:"sdk"`
	// Framework represents the framework used to host these functions.
	// For example, using the JS SDK we support NextJS, Netlify, Express,
	// etc via middleware to initialize the SDK handler.  This lets us
	// gather stats on usage.
	Framework string `json:"framework"`
	// AppName represents a namespaced app name for each deployed function.
	AppName string `json:"appName"`
	// AppVersion represents an optional application version identifier. This should change
	// whenever code within one of your Inngest function or any dependency thereof changes.
	AppVersion string `json:"appVersion,omitempty"`
	// Functions represents all functions hosted within this deploy.
	Functions []fn.SyncConfig `json:"functions"`
	// Headers are fetched from the incoming HTTP request.  They are present
	// on all calls to Inngest from the SDK, and are separate from the RegisterRequest
	// JSON payload to have a single source of truth.
	Headers Headers `json:"headers"`

	// IdempotencyKey is an optional input to deduplicate syncs.
	IdempotencyKey string `json:"idempotencyKey"`

	Capabilities Capabilities `json:"capabilities"`
}

type Headers struct {
	Env      string `json:"env"`
	Platform string `json:"platform"`
}

type Capabilities struct {
	InBandSync string `json:"in_band_sync"`
	TrustProbe string `json:"trust_probe"`
	Connect    string `json:"connect"`
}
