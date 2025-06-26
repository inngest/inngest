package checkpoint

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/inngest/inngestgo"
)

// NewAPIRunRequest represents the entire request payload used to create new
// API-based runs.
type NewRunRequest struct {
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
	Body json.RawMessage `json:"request"`
}

func (r NewRunRequest) FnSlug() string {
	if r.Event.Data.Fn != "" {
		return r.Event.Data.Fn
	}
	return fmt.Sprintf("%s %s", r.Event.Data.Method, r.Event.Data.Path)
}

func (r NewRunRequest) AppSlug() string {
	return r.Event.Data.Domain
}

// NewAPI creates a new API for checkpointing API-based runs.
func NewAPI() http.Handler {
	return nil
}

type api struct{}

func (a api) NewAPIRun(w http.ResponseWriter, r *http.Request) {
	// TODO: Create the app, if it doesn't exist.
	// TODO: Create the function, if it doesn't exist.
}

func (a api) APISteps(w http.ResponseWriter, r *http.Request) {
	// checkpoint those steps by writing to state.
}

func (a api) APIResponse(w http.ResponseWriter, r *http.Request) {
	// fialioze the run by storing the response
}
