package state

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/event"
	statev1 "github.com/inngest/inngest/pkg/execution/state"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/trace"
)

const (
	cronScheduleKey = "__cron"
	fnslugKey       = "__fnslug"
	traceLinkKey    = "__tracelink"
	debounceKey     = "__debounce"
	evtmapKey       = "__evtmap"
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

	mu sync.Mutex
}

func (c *Config) EventID() ulid.ULID {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.EventIDs) > 0 {
		return c.EventIDs[0]
	}
	return ulid.ULID{}
}

func (c *Config) GetSpanID() (*trace.SpanID, error) {
	fnTrace := c.FunctionTrace()
	if fnTrace != nil {
		if sid := fnTrace.SpanID(); sid.IsValid() {
			return &sid, nil
		}
	}

	// keep this around for backward compatibility purposes
	if c.SpanID != "" {
		sid, err := trace.SpanIDFromHex(c.SpanID)
		return &sid, err
	}
	return nil, fmt.Errorf("invalid span id in run config")
}

func (c *Config) initContext() {
	if c.Context == nil {
		c.Context = map[string]any{}
	}
}

func (c *Config) SetCronSchedule(schedule string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.initContext()
	c.Context[cronScheduleKey] = schedule
}

// CronSchedule retrieves the stored cron schedule information if available
func (c *Config) CronSchedule() *string {
	c.mu.Lock()
	defer c.mu.Unlock()

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

func (c *Config) SetFunctionSlug(slug string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.initContext()
	c.Context[fnslugKey] = slug
}

// FunctionSlug retrieves the stored function slug if available
func (c *Config) FunctionSlug() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Context == nil {
		return ""
	}

	if v, ok := c.Context[fnslugKey]; ok {
		if slug, ok := v.(string); ok {
			return slug
		}
	}

	return ""
}

func (c *Config) SetTraceLink(link string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.initContext()
	c.Context[traceLinkKey] = link
}

func (c *Config) TraceLink() *string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Context == nil {
		return nil
	}

	if v, ok := c.Context[traceLinkKey]; ok {
		if link, ok := v.(string); ok {
			return &link
		}
	}

	return nil
}

func (c *Config) SetDebounceFlag(flag bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.initContext()
	c.Context[debounceKey] = flag
}

func (c *Config) DebounceFlag() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Context == nil {
		return false
	}

	if v, ok := c.Context[debounceKey]; ok {
		if flag, ok := v.(bool); ok {
			return flag
		}
	}

	return false
}

func (c *Config) SetFunctionTrace(carrier *itrace.TraceCarrier) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.initContext()
	c.Context[consts.OtelPropagationKey] = carrier
}

func (c *Config) FunctionTrace() *itrace.TraceCarrier {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Context == nil {
		return nil
	}

	if data, ok := c.Context[consts.OtelPropagationKey]; ok {
		switch v := data.(type) {
		case *itrace.TraceCarrier:
			return v
		default:
			carrier := itrace.NewTraceCarrier()
			if err := carrier.Unmarshal(data); err == nil {
				// reassign it so it doesn't need to do the decoding again
				c.Context[consts.OtelPropagationKey] = carrier
				return carrier
			}

		}

	}
	return nil
}

// SetEventIDMapping creates an event mapping that can be used for referencing
// the events to their internal IDs
//
// - evtID => ULID
func (c *Config) SetEventIDMapping(evts []event.TrackedEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.initContext()

	m := map[string]ulid.ULID{}
	for _, e := range evts {
		evt := e.GetEvent()
		id := e.GetInternalID()
		m[evt.ID] = id
	}
	if byt, err := json.Marshal(m); err == nil {
		// store it as byte string to make it easier to store and retrieve
		c.Context[evtmapKey] = string(byt)
	}
}

func (c *Config) EventIDMapping() map[string]ulid.ULID {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Context == nil {
		return nil
	}

	if v, ok := c.Context[evtmapKey]; ok {
		if s, ok := v.(string); ok {
			var m map[string]ulid.ULID
			if err := json.Unmarshal([]byte(s), &m); err == nil {
				return m
			}
		}
	}

	return nil
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
