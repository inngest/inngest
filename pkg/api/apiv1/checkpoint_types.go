package apiv1

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fatih/structs"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngestgo"
	"github.com/oklog/ulid/v2"
)

// NewAPIRunData represents event data stored and used to create new API-based
// runs.  This is wrapped via CheckpointNewRunRequestr
type NewAPIRunData struct {
	// Domain is the domain that served the incoming request.  This must always
	// include the scheme, ie. "https://" (or "http://" in local dev).
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
	Body []byte `json:"body"`
}

// NewAPIRunRequest represents the entire request payload used to create new
// API-based runs.
type CheckpointNewRunRequest struct {
	// RunID represents the run ID for this request.  This is generated on the
	// client, and embeds the timestamp that the request started
	RunID ulid.ULID `json:"run_id"`

	// Idempotency allows the customization of an idempotency key, allowing us to
	// handle API idempotency using Inngest.
	Idempotency string `json:"idempotency"`

	// Event embeds the key request information which is used as the triggering
	// event for API-based runs.
	Event inngestgo.GenericEvent[NewAPIRunData] `json:"event"`

	// XXX: SDK Version and language??
}

func (r CheckpointNewRunRequest) AppSlug() string {
	return r.Event.Data.Domain
}

// AppID returns a deterministic V1 UUID based off of the environment ID
// and the given app slug.
func (r CheckpointNewRunRequest) AppID(envID uuid.UUID) uuid.UUID {
	return util.DeterministicUUID(append(envID[:], []byte(r.AppSlug())...))
}

func (r CheckpointNewRunRequest) AppURL() string {
	return r.Event.Data.Domain
}

func (r CheckpointNewRunRequest) FnSlug() string {
	if r.Event.Data.Fn != "" {
		return r.Event.Data.Fn
	}
	return fmt.Sprintf("%s %s", r.Event.Data.Method, r.Event.Data.Path)
}

// Fn returns a deterministic V1 UUID based off of the environment ID
// and the given fn slug.
func (r CheckpointNewRunRequest) FnID(appID uuid.UUID) uuid.UUID {
	return util.DeterministicUUID(append(appID[:], []byte(r.FnSlug())...))
}

func (r CheckpointNewRunRequest) Fn(appID uuid.UUID) inngest.Function {
	return inngest.Function{
		ID:              r.FnID(appID),
		ConfigVersion:   1,
		FunctionVersion: 1,
		Name:            r.FnSlug(),
		Slug:            r.FnSlug(),
		Steps: []inngest.Step{
			{
				ID:      "step",
				Name:    r.FnSlug(),
				URI:     r.Event.Data.Domain + r.Event.Data.Path,
				Retries: inngestgo.Ptr(0),
			},
		},
	}
}

func (r CheckpointNewRunRequest) FnConfig(envID uuid.UUID) string {
	fn := r.Fn(envID)
	byt, _ := json.Marshal(fn)
	return string(byt)
}

// APIResult represents the final result of an API function call
type APIResult struct {
	// StatusCode represents the status code for the API result
	StatusCode int `json:"status_code"`
	// Headers represents any response headers sent in the server response
	Headers map[string]string `json:"headers"`
	// Body represents the API response.  This may be nil by default.  It is only
	// captured when you manually specify that you want to track the result.
	Body []byte `json:"body,omitempty"`
	// Duration represents the duration
	Duration time.Duration `json:"duration"`
	// Error represents any error from the API.  This is only for internal errors,
	// eg. when a step permanently fails
	Error string `json:"error,omitempty"`
}

// CheckpointNewRunResponse represents the response payload for a successful run creation.
type CheckpointNewRunResponse struct {
	// FnID represents the ID of the function that the checkpoint run relates to.
	// This is required to be passed back in future step and response checkpoint calls
	// for proper tracking.
	FnID uuid.UUID `json:"fn_id"`
	// AppID represents the ID of the app that the checkpoint run relates to.
	// This is required to be passed back in future step and response checkpoint calls
	// for proper tracking.
	AppID uuid.UUID `json:"app_id"`
	// RunID is the function run ID created for this execution.
	RunID string `json:"run_id"`
}

// runEvent creates a new event.Event from the CheckpointNewRunRequest.  This allows us to
// record the input params for each API-based run as if it were a regular event-driven app.
func runEvent(r CheckpointNewRunRequest) event.Event {
	// TODO: Refactor event.Event to use inngestgo.Event as its base type.
	s := structs.New(r.Event.Data)
	s.TagName = "json"

	evt := event.Event{
		Name:      r.Event.Name,
		Data:      s.Map(),
		Timestamp: r.Event.Timestamp,
		Version:   r.Event.Version,
	}

	if evt.Timestamp == 0 {
		evt.Timestamp = time.Now().UnixMilli()
	}

	if r.Event.ID == nil {
		evt.ID = ulid.MustNew(ulid.Now(), rand.Reader).String()
	} else {
		evt.ID = *r.Event.ID
	}

	return evt
}
