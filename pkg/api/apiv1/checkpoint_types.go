package apiv1

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fatih/structs"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngestgo"
	"github.com/oklog/ulid/v2"
)

// CheckpointMetrics represents base IDs used within checkpoint metrics.
type CheckpointMetrics struct {
	AccountID uuid.UUID
	EnvID     uuid.UUID
	AppID     uuid.UUID
	FnID      uuid.UUID
}

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

	// Steps represent optional steps sent when creating the new run.  Sometimes,
	// the SDK may execute the run entirely and want to create run and accounting
	// in the same step.
	Steps []state.GeneratorOpcode `json:"steps"`

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

func (r CheckpointNewRunRequest) URL() string {
	return r.Event.Data.Domain + r.Event.Data.Path
}

func (r CheckpointNewRunRequest) Fn(appID uuid.UUID) inngest.Function {
	// NOTE: We don't use r.Event.Data.Path, because that could contain IDs
	// inside the URL (eg /v1/users/:id).
	//
	// This makes reusming sync functions harder.  For each request, we
	// must store the URL in the run metadata.
	uri := r.Event.Data.Domain

	return inngest.Function{
		ID:              r.FnID(appID),
		ConfigVersion:   1,
		FunctionVersion: 1,
		Name:            r.FnSlug(),
		Slug:            r.FnSlug(),
		Driver: inngest.FunctionDriver{
			URI: uri,
			Metadata: map[string]any{
				"type":   "sync", // This is a sync function
				"method": r.Event.Data.Method,
			},
		},
		Steps: []inngest.Step{
			{
				ID:      "step",
				Name:    r.FnSlug(),
				URI:     uri,
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
	// Token is the token that can be used to view the run output for redirects.
	Token string `json:"token,omitempty"`
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

type checkpointAsyncSteps struct {
	RunID ulid.ULID `json:"run_id"`
	FnID  uuid.UUID `json:"fn_id"`
	// QueueItemRef represents the queue item ID that's currently leased while
	// executing the SDK.
	QueueItemRef string                  `json:"qi_id"`
	Steps        []state.GeneratorOpcode `json:"steps"`
}

type checkpointSyncSteps struct {
	RunID ulid.ULID               `json:"run_id"`
	FnID  uuid.UUID               `json:"fn_id"`
	AppID uuid.UUID               `json:"app_id"`
	Steps []state.GeneratorOpcode `json:"steps"`
}
