// Command duckdbseed seeds a DuckDB dual-write database
// (inngest.runs/run_trace_spans/events, per
// pkg/db/duckdb/migrations/000001_baseline.sql) with synthetic test data
// shaped after a real dev database, for exercising pkg/cqrs/duckdbquery and
// the trace UI without running real workloads through `inngest dev`.
package main

import (
	"time"

	"github.com/google/uuid"
)

// Tenant is one observed (or synthesized) account/env/app/function tuple
// that generated runs are attributed to.
type Tenant struct {
	AccountID  uuid.UUID
	EnvID      uuid.UUID
	AppID      uuid.UUID
	FunctionID uuid.UUID
}

// Templates holds the value pools GenerateRuns samples from when building
// synthetic rows — either sampled from a real database (see sample.go) or
// the built-in defaults (DefaultTemplates) when no real data is available.
// JSON-valued fields hold raw JSON text, reused verbatim as payload shapes.
// Traces/EventProfiles are deliberately not flat pools of independently-
// chosen values the way Statuses/Inputs/Outputs are: a run's span tree
// shape and its triggering events are each properties of one real sampled
// run, replayed onto a new one intact (same names/attributes/payloads/
// relative timing) rather than reassembled from unrelated pieces — see
// each type's own doc comment.
type Templates struct {
	Tenants  []Tenant
	Statuses []string
	Inputs   []string
	Outputs  []string
	// Traces are whole sampled runs' span trees (see TraceTemplate).
	Traces []TraceTemplate
	// EventProfiles are whole sampled runs' sets of triggering events (see
	// EventTemplate) — usually one event, but a batch-triggered run can have
	// several, all received close together.
	EventProfiles [][]EventTemplate
}

// SpanTemplate is one sampled span, stripped of everything that ties it to
// the specific sampled run's absolute timing (StartOffset/EndOffset are
// relative to the sampled run's own queued_at, its "t=0" -- the same
// reference point TraceTemplate's StartedOffset/EndedOffset and
// EventTemplate's Offset use, so a whole generated run shifts by adding
// one randomly placed queued_at to every one of these offsets: a single
// random offset per run, never independent jitter per timestamp).
//
// SpanID/ParentSpanID are reused completely verbatim from the sampled
// span, NOT regenerated per replay: a span_id is only ever meaningful
// paired with its own run_id (every read in this system scopes by run_id
// first), and every generated run gets its own fresh run_id, so reusing
// the exact same span_id/parent_span_id text across many generated runs
// causes no collision. This is also what preserves real nesting for free
// -- a userland/extended-trace span nested under another userland span
// (or any depth of real parent-child structure) replays with that exact
// same structure automatically, since the parent_span_id text still
// matches whichever sibling span_id it always matched, without this
// package ever having to resolve or rebuild the tree itself.
type SpanTemplate struct {
	SpanID       string
	ParentSpanID *string
	Name         string
	Attributes   string
	Output       string
	Input        string
	StartOffset  time.Duration
	EndOffset    time.Duration
}

// TraceTemplate is one sampled run's entire flat set of spans plus its own
// queued_at-relative timing, kept intact (not randomized span-by-span,
// per-name/per-attribute, per-duration, or reparented) so a newly
// generated run's trace looks like a real one shifted in time, not an
// independently reassembled fake. Spans carries no hierarchy of its own --
// each SpanTemplate's own ParentSpanID already encodes its real position,
// reused verbatim (see SpanTemplate's doc comment), so replaying a trace
// needs no tree-building step. TraceID is likewise reused verbatim across
// every replay, for the same reason. StartedOffset/EndedOffset are nil
// when the sampled run's own started_at/ended_at was NULL (e.g. still
// queued or running when sampled), in which case the generated run leaves
// them nil too.
type TraceTemplate struct {
	TraceID       string
	Spans         []SpanTemplate
	StartedOffset *time.Duration
	EndedOffset   *time.Duration
}

// EventTemplate is one sampled triggering event, stripped of its ID so it
// can be replayed onto a newly generated run with a fresh internal ID.
// Offset is the event's received_at relative to the sampled run's own
// queued_at (see SpanTemplate's doc comment) — typically negative, since
// an event is received before the run it triggers is queued.
type EventTemplate struct {
	Name   string
	Data   string
	Offset time.Duration
}

// GenerateConfig parameterizes GenerateRuns.
type GenerateConfig struct {
	// RunCount is the number of synthetic runs to generate.
	RunCount int
	// Window is how far back from Now generated runs' queued_at values are
	// spread.
	Window time.Duration
	// Now anchors the generated time range; callers pass time.Now() in
	// production and a fixed time in tests for determinism.
	Now time.Time
}

// RunRow mirrors inngest.runs' columns (pkg/db/duckdb/migrations/000001_baseline.sql).
type RunRow struct {
	AccountID   uuid.UUID
	EnvID       uuid.UUID
	RunID       string
	QueuedAt    time.Time
	ScheduledAt *time.Time
	StartedAt   *time.Time
	EndedAt     *time.Time
	AppID       uuid.UUID
	FunctionID  uuid.UUID
	Status      string
	Inputs      string
	Output      string
	EventIDs    []string
	// Sessions mirrors inngest.runs.sessions — this tool never generates
	// session-tagged runs, so it's always left nil.
	Sessions []SessionPair
}

// SessionPair is one run-level session membership pair, mirroring
// pkg/tracing/meta.EventSession (kept as a local, dependency-free type
// here rather than importing that package, matching this file's existing
// preference for primitive-typed row structs). JSON tags match
// inngest.runs.sessions' STRUCT(key VARCHAR, id VARCHAR)[] field names.
type SessionPair struct {
	Key string `json:"key"`
	ID  string `json:"id"`
}

// SpanRow mirrors inngest.run_trace_spans' columns.
type SpanRow struct {
	AccountID    uuid.UUID
	EnvID        uuid.UUID
	RunID        string
	RunQueuedAt  time.Time
	AppID        uuid.UUID
	FunctionID   uuid.UUID
	Name         string
	StartTime    time.Time
	EndTime      time.Time
	TraceID      string
	SpanID       string
	ParentSpanID *string
	Attributes   string
	Links        string
	Output       string
	Input        string
}

// EventRow mirrors inngest.events' columns.
type EventRow struct {
	AccountID  uuid.UUID
	EnvID      uuid.UUID
	InternalID string
	ReceivedAt time.Time
	Source     string
	SourceID   *string
	EventID    string
	EventName  string
	EventData  string
	EventV     string
	EventTS    time.Time
	EventMeta  string
}

// GeneratedRun bundles one synthetic run together with its span tree and
// the events it was triggered by.
type GeneratedRun struct {
	Run    RunRow
	Spans  []SpanRow
	Events []EventRow
}
