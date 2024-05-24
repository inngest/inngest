package state

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	statev1 "github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/trace"
)

const (
	cronScheduleKey = "__cron"
)

type ID struct {
	RunID      ulid.ULID
	FunctionID uuid.UUID
	// Tenant provides tennat information for the run ID.  This is embedded into
	// the identifier as additional fields that may be used in various
	// implementations, but should not be used to reference specific run IDs.
	Tenant Tenant
}

// IDFromV1 returns v2.ID from a statev1.Identifier
func IDFromV1(id statev1.Identifier) ID {
	return ID{
		RunID:      id.RunID,
		FunctionID: id.WorkflowID,
		Tenant: Tenant{
			AppID:     id.AppID,
			EnvID:     id.WorkspaceID,
			AccountID: id.AccountID,
		},
	}
}

// Metadata represets metadata for the run state.
type Metadata struct {
	ID      ID
	Config  Config
	Metrics RunMetrics
	// Stack stores the order of the step IDs as a stack
	Stack []string
}

func (m Metadata) IdempotencyKey() string {
	key := m.Config.Idempotency
	if key == "" {
		key = m.ID.RunID.String()
	}
	return fmt.Sprintf("%s:%s", m.ID.FunctionID, key)
}

// Tenant represents tenant information for the run.
type Tenant struct {
	AppID     uuid.UUID
	EnvID     uuid.UUID
	AccountID uuid.UUID
}

// Config represents run config, stored within metadata.
type Config struct {
	// FunctionSlug stores the function slug.
	FunctionSlug string
	// FunctionVersion stores the version of the function used when the run is
	// scheduled.
	FunctionVersion int
	// SpanID stores the root span ID for the run's trace.
	SpanID string
	// BatchID tracks the batch ID for the function, if the function uses batching.
	BatchID *ulid.ULID
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

func (c *Config) GetSpanID() (*trace.SpanID, error) {
	if c.SpanID != "" {
		sid, err := trace.SpanIDFromHex(c.SpanID)
		return &sid, err
	}
	return nil, fmt.Errorf("invalid span id in run config")
}

func (c *Config) SetCronSchedule(schedule string) {
	if c.Context == nil {
		c.Context = map[string]any{}
	}
	c.Context[cronScheduleKey] = schedule
}

// CronSchedule retrieves the stored cron schedule information if available
func (c *Config) CronSchedule() *string {
	if c.Context == nil {
		return nil
	}

	if v, ok := c.Context[cronScheduleKey]; ok {
		if schedule, ok := v.(string); ok {
			return &schedule
		}
	}

	return nil
}

// FirstEventID returns the first event ID in the list of event IDs.
// If there are no event IDs, it returns an empty ULID.
func (c *Config) FirstEventID() ulid.ULID {
	if len(c.EventIDs) > 0 {
		return c.EventIDs[0]
	}
	return ulid.ULID{}
}

// RunMetrics stores state-level run metrics.
type RunMetrics struct {
	// StateSize stores the total size, in bytes, of all events and step output.
	// This is a counter and always increments.
	StateSize int

	// EventSize stores the size of all events that triggered the function, in bytes
	EventSize int

	// StepCount represents the total number of steps already completed.
	StepCount int

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
