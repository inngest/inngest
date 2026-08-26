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
type Templates struct {
	Tenants    []Tenant
	Statuses   []string
	SpanNames  []string
	EventNames []string
	Inputs     []string
	Outputs    []string
	Attributes []string
	EventData  []string
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
