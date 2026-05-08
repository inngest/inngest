package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
)

type sdkRequestIDCtxKey struct{}

type sdkJobIDCtxKey struct{}

// DispatchRequestID returns the deterministic ULID the executor stamps on
// outbound SDK requests for a given dispatch. Single source of truth for the
// producer (executor) and the validator (checkpoint package): a drift in
// either the seed format or the entropy derivation would silently break
// fencing for in-flight runs.
func DispatchRequestID(ts time.Time, runID ulid.ULID, generationID int) ulid.ULID {
	return util.MustDeterministicULID(ts, fmt.Appendf(nil, "%s:%d", runID, generationID))
}

// DispatchRequestIDEntropy returns the entropy portion of the dispatch
// RequestID. The validator compares this against the entropy of the
// SDK-echoed RequestID; the dispatch timestamp doesn't participate in
// fencing.
func DispatchRequestIDEntropy(runID ulid.ULID, generationID int) []byte {
	return DispatchRequestID(time.Unix(0, 0), runID, generationID).Entropy()
}

// WithRequestIDs stores the per-outbound request ID and stable job ID for SDK
// driver calls.
func WithRequestIDs(ctx context.Context, requestID, jobID string) context.Context {
	ctx = context.WithValue(ctx, sdkRequestIDCtxKey{}, requestID)
	ctx = context.WithValue(ctx, sdkJobIDCtxKey{}, jobID)
	return ctx
}

// RequestIDFromContext returns the per-outbound SDK request ID.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(sdkRequestIDCtxKey{}).(string)
	return id
}

// JobIDFromContext returns the stable queue item ID for this SDK request.
func JobIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(sdkJobIDCtxKey{}).(string)
	return id
}

type SDKRequest struct {
	Event   map[string]any   `json:"event"`
	Events  []map[string]any `json:"events"`
	Actions map[string]any   `json:"steps"`

	// For the "defer" opcodes
	Defers map[string]SDKDeferEntry `json:"defers"`

	Context *SDKRequestContext `json:"ctx"`
	// Version indicates the version used to manage the SDK request context.
	//
	// A value of -1 means that the function is starting and has no version.
	Version int `json:"version"`

	// DEPRECATED: NOTE: This is moved into SDKRequestContext for V3+/Non-TS SDKs
	UseAPI bool `json:"use_api"`
}

// SDKDeferEntry tells the SDK that a deferred run is already registered for the
// given hashed ID, so it should not re-report `OpcodeDeferAdd` for it.
type SDKDeferEntry struct {
	// Abortable indicates the SDK may emit `OpcodeDeferAbort` for this
	// entry. It signifies that the deferred run hasn't queued yet, so it can be
	// aborted.
	//
	// Note that this is different than run cancellation on an already-queued
	// run. Aborting a deferred run simply will mean no run will ever exist
	// (e.g. no run ID).
	Abortable bool `json:"abortable"`
}

type SDKRequestContext struct {
	// FunctionID is used within entrypoints to SDK-based functions in
	// order to specify the ID of the function to run via RPC.
	FunctionID uuid.UUID `json:"fn_id"`

	// RunID  is the ID of the current
	RunID ulid.ULID `json:"run_id"`

	// Env is the name of the environment that the function is running in.
	// though this is self-discoverable most of the time, for static envs
	// the SDK has no knowledge of the name as it only has a signing key.
	Env string `json:"env"`

	// StepID is used within entrypoints to SDK-based functions in
	// order to specify the step of the function to run via RPC.
	StepID string `json:"step_id"`

	// Attempt is the zero-index attempt number.
	Attempt int `json:"attempt"`

	// MaxAttempts is the maximum number of attempts allowed for this function.
	MaxAttempts int `json:"max_attempts"`

	// Stack represents the function stack at the time of the step invocation.
	Stack *FunctionStack `json:"stack"`

	// QueueItemID is the ID of the queue item and shard, used when checkpointing
	// async functions so that the API knows which queue item to reset.
	QueueItemRef string `json:"qi_id"`

	// RequestID is a unique ID generated for each outbound SDK request.
	RequestID string `json:"request_id,omitempty"`

	// JobID is the stable queue item ID for the current job.
	JobID string `json:"job_id,omitempty"`

	// DisableImmediateExecution is used to tell the SDK whether it should
	// disallow immediate execution of steps as they are found.
	DisableImmediateExecution bool `json:"disable_immediate_execution"`

	// UseAPI tells the SDK to retrieve `Events` and `Actions` data
	// from the API instead of expecting it to be in the request body.
	// This is a way to get around serverless provider's request body
	// size limits.
	UseAPI bool `json:"use_api"`

	// XXX: Pass in opentracing context within ctx.
}

type FunctionStack struct {
	Stack   []string `json:"stack"`
	Current int      `json:"current"`
}

func (m FunctionStack) MarshalJSON() ([]byte, error) {
	if m.Stack == nil {
		m.Stack = make([]string, 0)
	}

	type alias FunctionStack // Avoid infinite recursion
	return json.Marshal(alias(m))
}
