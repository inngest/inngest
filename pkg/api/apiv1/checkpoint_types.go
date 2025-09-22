package apiv1

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fatih/structs"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/exechttp"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/tracing/meta"
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

func (r CheckpointNewRunRequest) Fn(appID uuid.UUID) inngest.Function {
	uri := r.Event.Data.Domain + r.Event.Data.Path
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

// APIResult represents the final result of an API function call
type APIResult struct {
	// StatusCode represents the status code for the API result
	StatusCode int `json:"status_code"`
	// Headers represents any response headers sent in the server response
	Headers map[string]string `json:"headers"`
	// Body represents the API response.  This may be nil by default.  It is only
	// captured when you manually specify that you want to track the result.
	Body []byte `json:"body,omitempty"`
	// Duration represents the overall time that it took for the API to execute.
	Duration time.Duration `json:"duration"`
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

type checkpointSteps struct {
	RunID ulid.ULID               `json:"run_id"`
	FnID  uuid.UUID               `json:"fn_id"`
	AppID uuid.UUID               `json:"app_id"`
	Steps []state.GeneratorOpcode `json:"steps"`

	// Plus auth data added from auth.
	AccountID uuid.UUID `json:"-"`
	EnvID     uuid.UUID `json:"-"`

	// Optional metadata.
	md *sv2.Metadata
}

// checkpointRunContext implements execution.RunContext for use in checkpoint API calls
type checkpointRunContext struct {
	md         sv2.Metadata
	httpClient exechttp.RequestExecutor
	events     []json.RawMessage

	// Data from queue.Item that we actually need
	groupID         string
	attemptCount    int
	maxAttempts     int
	priorityFactor  *int64
	concurrencyKeys []state.CustomConcurrency
	parallelMode    enums.ParallelMode
}

func (c *checkpointRunContext) Metadata() *sv2.Metadata {
	return &c.md
}

func (c *checkpointRunContext) Events() []json.RawMessage {
	return c.events
}

func (c *checkpointRunContext) HTTPClient() exechttp.RequestExecutor {
	return c.httpClient
}

func (c *checkpointRunContext) GroupID() string {
	return c.groupID
}

func (c *checkpointRunContext) AttemptCount() int {
	return c.attemptCount
}

func (c *checkpointRunContext) MaxAttempts() *int {
	return &c.maxAttempts
}

func (c *checkpointRunContext) ShouldRetry() bool {
	return c.attemptCount < (c.maxAttempts - 1)
}

func (c *checkpointRunContext) IncrementAttempt() {
	c.attemptCount++
}

func (c *checkpointRunContext) PriorityFactor() *int64 {
	// TODO
	return c.priorityFactor
}

func (c *checkpointRunContext) ConcurrencyKeys() []state.CustomConcurrency {
	// TODO
	return c.concurrencyKeys
}

func (c *checkpointRunContext) ParallelMode() enums.ParallelMode {
	// TODO
	return c.parallelMode
}

func (c *checkpointRunContext) LifecycleItem() queue.Item {
	// For checkpoint context, we create a minimal queue.Item for lifecycle events
	// This is the one place we still need to construct a queue.Item, but it's much simpler
	return queue.Item{
		Identifier: state.Identifier{
			WorkspaceID: c.md.ID.Tenant.EnvID,
			AppID:       c.md.ID.Tenant.AppID,
			WorkflowID:  c.md.ID.FunctionID,
			RunID:       c.md.ID.RunID,
		},
		WorkspaceID:           c.md.ID.Tenant.EnvID,
		GroupID:               c.groupID,
		Attempt:               c.attemptCount,
		PriorityFactor:        c.priorityFactor,
		CustomConcurrencyKeys: c.concurrencyKeys,
		ParallelMode:          c.parallelMode,
		Payload:               queue.PayloadEdge{
			// TODO
		},
	}
}

func (c *checkpointRunContext) SetStatusCode(code int) {
	// this is a noop.
}

func (c *checkpointRunContext) UpdateOpcodeError(op *state.GeneratorOpcode, err state.UserError) {
	// TODO: Update the error by storing the opcodes in the checkpoint
	// struct c.
}

func (c *checkpointRunContext) UpdateOpcodeOutput(op *state.GeneratorOpcode, output json.RawMessage) {
	// TODO: Update the output by storing the opcodes in the
	// checkpoint struct c.
}

func (c *checkpointRunContext) SetError(err error) {
	// TODO
}

func (c *checkpointRunContext) ExecutionSpan() *meta.SpanReference {
	// TODO
	return nil
}
