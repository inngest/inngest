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
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/trace"
)

const (
	cronScheduleKey   = "__cron"
	fnslugKey         = "__fnslug"
	traceLinkKey      = "__tracelink"
	debounceKey       = "__debounce"
	evtmapKey         = "__evtmap"
	debugSessionIDKey = "__debug_session_id"
	debugRunIDKey     = "__debug_run_id"
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

func V1FromMetadata(md Metadata) statev1.Identifier {
	return statev1.Identifier{
		RunID:                 md.ID.RunID,
		WorkflowID:            md.ID.FunctionID,
		WorkflowVersion:       md.Config.FunctionVersion,
		WorkspaceID:           md.ID.Tenant.EnvID,
		AccountID:             md.ID.Tenant.AccountID,
		EventID:               md.Config.EventID(),
		EventIDs:              md.Config.EventIDs,
		BatchID:               md.Config.BatchID,
		CustomConcurrencyKeys: md.Config.CustomConcurrencyKeys,
		PriorityFactor:        md.Config.PriorityFactor,
		OriginalRunID:         md.Config.OriginalRunID,
	}
}

// NewPauseIdentifier crease a PauseIdentifier from an ID
func NewPauseIdentifier(id ID) statev1.PauseIdentifier {
	return statev1.PauseIdentifier{
		RunID:      id.RunID,
		FunctionID: id.FunctionID,
		AccountID:  id.Tenant.AccountID,
	}
}

// IDFromPause creates an ID from a pause.
func IDFromPause(p statev1.Pause) ID {
	return ID{
		RunID:      p.Identifier.RunID,
		FunctionID: p.Identifier.FunctionID,
		Tenant: Tenant{
			AccountID: p.Identifier.AccountID,
			EnvID:     p.WorkspaceID,
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

func (m Metadata) ShouldCoalesceParallelism(resp *statev1.DriverResponse) bool {
	reqVersion := m.Config.RequestVersion
	if reqVersion == -1 {
		reqVersion = resp.RequestVersion
	}

	return reqVersion >= 2
}

// IdempotencyKey returns a unique key for this run, intended to used as part of
// an idempotency key when enqueuing steps.
//
// NOT to be used for run-level idempotency keys; this is exclusively used for
// step-level idepotency. For that purpose, use `Identifier.IdempotencyKey()`.
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

func InitConfig(c *Config) *Config {
	if c.mu == nil {
		c.mu = &sync.RWMutex{}
	}
	return c
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
	// HasAI indicates if the function has AI steps
	HasAI bool
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

	mu *sync.RWMutex
}

func (c *Config) EventID() ulid.ULID {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.EventIDs) > 0 {
		return c.EventIDs[0]
	}
	return ulid.Zero
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
	defer c.mu.Unlock()
	c.mu.Lock()

	c.initContext()
	c.Context[cronScheduleKey] = schedule
}

// CronSchedule retrieves the stored cron schedule information if available
func (c *Config) CronSchedule() *string {
	c.mu.RLock()
	defer c.mu.RUnlock()

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
	c.mu.RLock()
	defer c.mu.RUnlock()

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
	c.mu.RLock()
	defer c.mu.RUnlock()

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
	c.mu.RLock()
	defer c.mu.RUnlock()

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

func (c *Config) NewSetFunctionTrace(carrier *meta.SpanReference) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.initContext()
	c.Context[meta.PropagationKey] = *carrier
}

// RootSpanFromConfig is deprecated.  Use tracing.RunSpanRefFromMetadata.
func (c *Config) RootSpanFromConfig() *meta.SpanReference {
	if c == nil || c.mu == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.Context == nil {
		return nil
	}

	if raw, ok := c.Context[meta.PropagationKey]; ok {
		if data, err := json.Marshal(raw); err == nil {
			var meta meta.SpanReference
			if err := json.Unmarshal(data, &meta); err == nil {
				return &meta
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
	c.mu.RLock()
	defer c.mu.RUnlock()

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

func (c *Config) SetDebugSessionID(debugSessionID ulid.ULID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.initContext()
	c.Context[debugSessionIDKey] = debugSessionID.String()
}

// DebugSessionID retrieves the stored debug session ID if available
func (c *Config) DebugSessionID() *ulid.ULID {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.Context == nil {
		return nil
	}

	if v, ok := c.Context[debugSessionIDKey]; ok {
		if s, ok := v.(string); ok {
			if id, err := ulid.Parse(s); err == nil {
				return &id
			}
		}
	}

	return nil
}

func (c *Config) SetDebugRunID(debugRunID ulid.ULID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.initContext()
	c.Context[debugRunIDKey] = debugRunID.String()
}

// DebugRunID retrieves the stored debug run ID if available
func (c *Config) DebugRunID() *ulid.ULID {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.Context == nil {
		return nil
	}

	if v, ok := c.Context[debugRunIDKey]; ok {
		if s, ok := v.(string); ok {
			if id, err := ulid.Parse(s); err == nil {
				return &id
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
	HasAI          bool
}

type CustomConcurrency = statev1.CustomConcurrency
