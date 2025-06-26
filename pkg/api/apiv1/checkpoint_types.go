package apiv1

import (
	"encoding/json"
	"fmt"

	"github.com/inngest/inngestgo"
	"github.com/oklog/ulid/v2"
)

// NewAPIRunRequest represents the entire request payload used to create new
// API-based runs.
type CheckpointNewRunRequest struct {
	// Seed allows us to construct a deterministic run ID from this data and the
	// event TS.
	Seed string `json:"seed"`

	// TODO
	// Idempotency allows the customization of an idempotency key, allowing us to
	// handle API idempotency using Inngest.
	// Idempotency string `json:"idempotency"`

	// Event embeds the key request information which is used as the triggering
	// event for API-based runs.
	Event inngestgo.GenericEvent[NewAPIRunData] `json:"event"`
}

// NewAPIRunData represents event data stored and used to create new API-based
// runs.
type NewAPIRunData struct {
	// Domain is the domain that served the incoming request.
	Domain string `json:"domain"`
	// Method is the incoming request method.  This is used for RESTful
	// API endpoints.
	Method string `json:"method"`
	// Path is the path for the incoming request.
	Path string `json:"path"` // request path
	// Fn is the optional function slug.  If not present, this is created
	// using a combination of the method and the path: "POST /v1/runs"
	Fn string `json:"fn"`

	// IP is the IP that created the request.
	IP string `json:"ip"` // incoming IP
	// ContentType is the content type for the request.
	ContentType string `json:"content_type"`
	// QueryParams are the query parameters for the request, as a single string
	// without the leading "?".
	//
	// NOTE: This is optional;  we do not require that users store the query params
	// for every request, as this may contain data that users choose not to log.
	QueryParams string `json:"query_params"`
	// Body is the incoming request body.
	//
	// NOTE: This is optional;  we do not require that users store the body for
	// every request, as this may contain data that users choose not to log.
	Body json.RawMessage `json:"body"`
}

func (r CheckpointNewRunRequest) RunID() ulid.ULID {
	// TODO: Seed + TS
	return ulid.ULID{}
}

func (r CheckpointNewRunRequest) FnSlug() string {
	if r.Event.Data.Fn != "" {
		return r.Event.Data.Fn
	}
	return fmt.Sprintf("%s %s", r.Event.Data.Method, r.Event.Data.Path)
}

func (r CheckpointNewRunRequest) AppSlug() string {
	return r.Event.Data.Domain
}

func (r CheckpointNewRunRequest) FnConfig() string {
	// TODO
	return ""
}
