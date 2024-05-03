package state

import (
	"time"

	"github.com/google/uuid"
	statev1 "github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
)

type ID struct {
	RunID      ulid.ULID
	FunctionID uuid.UUID
}

// Metadata represets metadata for the run state.
type Metadata struct {
	ID      ID
	Tenant  Tenant
	Config  Config
	Metrics RunMetrics
}

// Tenant represents tenant information for the run.
type Tenant struct {
	AppID     uuid.UUID
	EnvID     uuid.UUID
	AccountID uuid.UUID
}

// Config represents run config, stored within metadata.
type Config struct {
	// SpanID stores the root span ID for the run's trace.
	SpanID string
	// StartedAt stores the time that the first step started.  This allows us to
	// track wall time for `timeout.finish` configuration.
	StartedAt time.Time
	// EventIDs represents the IDs of the event(s) that trirgger the function.
	EventIDs []ulid.ULID
	// RequestVersion represents the executor request versioning/hashing style
	// used to manage state.
	//
	// TS v3, Go, Rust, Elixir, and Java all use the same hashing style (1).
	//
	// TS v1 + v2 use a unique hashing style (0) which cannot be transferred
	// to other languages.
	//
	// This lets us send the hashing style to SDKs so that we can execute in
	// the correct format with backcompat guarantees built in.
	//
	// NOTE: We can only know this the first time an SDK is responding to a step.
	RequestVersion int
	// Idempotency represents an optional idempotency key.  This must be an
	// xxhash64 hashed string.
	Idempotency string
	// ReplayID stores the ID of the replay, if this identifier belongs to a replay.
	ReplayID *uuid.UUID
	// OriginalRunID stores the ID of the original run, for a one-off replay.
	OriginalRunID *ulid.ULID
	// PriorityFactor is the overall priority factor for this particular function
	// run.  This allows individual runs to take precedence within the same queue.
	// The higher the number (up to consts.PriorityFactorMax), the higher priority
	// this run has.  All next steps will use this as the factor when scheduling
	// future edge jobs (on their first attempt).
	PriorityFactor *int64
	// CustomConcurrencyKeys stores custom concurrency keys for this function run.  This
	// allows us to use custom concurrency keys for each job when processing steps for
	// the function, with cached expression results.
	CustomConcurrencyKeys []CustomConcurrency
	// ForceStepPlan forces SDKs, where supported, to submit step plan opcodes prior
	// to running steps.
	ForceStepPlan bool
	// Context allows storing arbitrary context for a run.
	Context map[string]any
}

// RunMetrics stores state-level run metrics.
type RunMetrics struct {
	// StateSize stores the total size, in bytes, of all events and step output.
	// This is a counter and always increments.
	StateSize int

	// TODO

	// Waits stores the current number of in-progress waits for the run.
	// This is in effect a gauge.
	// Waits int

	// Sleeps stores the current number of in-progress sleeps for the run.
	// This is in effect a gauge.
	// Sleeps int

	// Steps stores the current number of in-progress or pending steps for the run.
	// This is usually 1, but may be > 1 in the case of step parallelism.
	// Steps int
}

// MutableConfig represents mutable config options.
type MutableConfig struct {
	StartedAt      time.Time
	RequestVersion int
	ForceStepPlan  bool
}

type CustomConcurrency = statev1.CustomConcurrency

/*
type CustomConcurrency struct {
	// Key represents the actual evaluated concurrency key.
	Key string `json:"k"`
	// Hash represents the hash of the concurrency expression - unevaluated -
	// as defined in the function.  This lets us look up the latest concurrency
	// values as defined in the most recent version of the function and use
	// these concurrency values.  Without this, it's impossible to adjust concurrency
	// for in-progress functions.
	Hash string `json:"h"`
	// Limit represents the limit at the time the function started.  If the concurrency
	// key is removed from the fn definition, this pre-computed value will be used instead.
	//
	// NOTE: If the value is removed from the last deployed function we could also disregard
	// this concurrency key.
	Limit int `json:"l"`
}
*/
