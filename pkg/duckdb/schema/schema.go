// Package schema is the single description of the physical tables the lake
// stores (the OSS DuckDB inngest.runs, inngest.run_trace_spans,
// inngest.events and inngest.run_metadata, which the ClickHouse buffer
// mirrors): each column's type and whether it can
// change over a run's life. The federated executor, the Insights transpiler
// and buffer streamers all read it, so they can't disagree about which
// columns exist or which filters are safe to push.
package schema

import (
	"github.com/inngest/inngest/pkg/tracing/meta"
)

// Type is a column's value kind on the wire between a buffer and DuckDB.
type Type int

const (
	// TypeUUID values are uuid.UUID or its string form.
	TypeUUID Type = iota
	// TypeVarchar values are string.
	TypeVarchar
	// TypeJSON values are string holding raw JSON text. They're JSON
	// everywhere but the lake's own columns, which store that text in a
	// VARCHAR (see jsonText).
	TypeJSON
	// TypeTimestamp values are time.Time (millisecond precision is kept).
	TypeTimestamp
	// TypeBool values are bool.
	TypeBool
	// TypeBigint values are int64 (or any narrower Go integer type).
	TypeBigint
	// TypeVarcharList values are []string.
	TypeVarcharList
)

// DuckDBType is the DuckDB type a value of kind t is ingested as.
func (t Type) DuckDBType() string {
	switch t {
	case TypeUUID:
		return "UUID"
	case TypeJSON:
		return "JSON"
	case TypeTimestamp:
		return "TIMESTAMP WITH TIME ZONE"
	case TypeBool:
		return "BOOLEAN"
	case TypeBigint:
		return "BIGINT"
	case TypeVarcharList:
		return "VARCHAR[]"
	default:
		return "VARCHAR"
	}
}

// TerminalStatuses are the run statuses (the run span's StepStatus, as
// stored in runs.status) a final run row carries: enums.StepStatus.IsEnded's
// set plus Skipped, which IsEnded leaves out but is terminal for a run
// (enums.RunStatus.IsEnded includes it).
var TerminalStatuses = []string{"Completed", "Failed", "Errored", "Cancelled", "TimedOut", "Skipped"}

// Column is one physical column.
type Column struct {
	Name string
	// Type is the value's wire kind.
	Type Type
	// DuckDB is the column's declared type in the lake, when it differs
	// from Type.DuckDBType() (JSON text, STRUCT lists).
	DuckDB string
	// Nullable is false for a column the lake declares NOT NULL.
	Nullable bool
	// StringKeys, for a JSON column, are keys whose values are always
	// strings (span attributes written by string, UUID and ULID attrs), which
	// a buffer can compare exactly; any other key's value may be of any type.
	StringKeys []string
}

// PhysicalType is the column's declared DuckDB type in the lake.
func (c Column) PhysicalType() string {
	if c.DuckDB != "" {
		return c.DuckDB
	}
	return c.Type.DuckDBType()
}

// LogicalType is the type readers see the column as: JSON for JSON text
// (the lake's VARCHAR, which the lake's macros return cast to JSON),
// else PhysicalType.
func (c Column) LogicalType() string {
	if c.Type == TypeJSON && c.DuckDB == jsonText {
		return TypeJSON.DuckDBType()
	}
	return c.PhysicalType()
}

// IsStringKey reports whether key's value is always a string.
func (c Column) IsStringKey(key string) bool {
	for _, k := range c.StringKeys {
		if k == key {
			return true
		}
	}
	return false
}

// Table is one physical table, its columns in declared order.
type Table struct {
	Name    string
	Columns []Column
}

// Column returns the named column.
func (t Table) Column(name string) (Column, bool) {
	for _, c := range t.Columns {
		if c.Name == name {
			return c, true
		}
	}
	return Column{}, false
}

// jsonText is a JSON column's lake type: the JSON text as is, in a VARCHAR
// rather than JSON so an Iceberg lake (which rejects duckdb's JSON type, and
// duckdb doesn't keep the alias on a string column) can use the same types.
// The macros in pkg/db/duckdb's 000002_insights_views.sql return it as JSON
// (LogicalType), casting only where a whole value is returned or embedded:
// the cast is a validating parse per row, so filters and key extractions
// read the text directly.
const jsonText = "VARCHAR"

// attributeKeys are the string-valued span attributes: a buffer compares a
// pushed filter on them directly (any other key only where its value is a
// string).
var attributeKeys = meta.StringAttrKeys()

// Runs is inngest.runs: one row per run-level span (queued, started, final),
// collapsed to one per run. A buffer serves each run's latest row only, so
// it can apply a filter on any column exactly.
var Runs = Table{Name: "runs", Columns: []Column{
	{Name: "account_id", Type: TypeUUID},
	{Name: "env_id", Type: TypeUUID},
	{Name: "run_id", Type: TypeVarchar},
	{Name: "queued_at", Type: TypeTimestamp},
	// Fixed at schedule time.
	{Name: "scheduled_at", Type: TypeTimestamp},
	{Name: "started_at", Type: TypeTimestamp, Nullable: true},
	{Name: "ended_at", Type: TypeTimestamp, Nullable: true},
	{Name: "app_id", Type: TypeUUID},
	{Name: "app_name", Type: TypeVarchar},
	{Name: "function_id", Type: TypeUUID},
	{Name: "function_slug", Type: TypeVarchar},
	{Name: "status", Type: TypeVarchar},
	{Name: "attributes", Type: TypeJSON, DuckDB: jsonText, StringKeys: attributeKeys},
	// The run's triggering events: fixed at schedule time, but an array, so
	// no key filter means the same thing in both stores.
	{Name: "inputs", Type: TypeJSON, DuckDB: jsonText},
	// Set once, by the final row.
	{Name: "output", Type: TypeJSON, Nullable: true, DuckDB: jsonText},
	{Name: "event_ids", Type: TypeVarcharList, Nullable: true},
	{Name: "sessions", Type: TypeJSON, Nullable: true, DuckDB: `STRUCT("key" VARCHAR, id VARCHAR)[]`},
	{Name: "is_deferred", Type: TypeBool},
	{Name: "defer_parent_fn_slug", Type: TypeVarchar, Nullable: true},
	{Name: "defer_parent_run_ids", Type: TypeVarcharList, Nullable: true},
	{Name: "trace_id", Type: TypeVarchar, Nullable: true},
	// The federation split (lake bucket_at <= W, delta > W); not a query
	// column, so nothing is pushed.
	{Name: "bucket_at", Type: TypeTimestamp},
}}

// Spans is inngest.run_trace_spans: one row per span emission, never
// collapsed, so a filter is safe on any column (input is array-valued, so
// it takes no key filters).
var Spans = Table{Name: "run_trace_spans", Columns: []Column{
	{Name: "account_id", Type: TypeUUID},
	{Name: "env_id", Type: TypeUUID},
	{Name: "run_id", Type: TypeVarchar},
	{Name: "run_queued_at", Type: TypeTimestamp},
	{Name: "app_id", Type: TypeUUID},
	{Name: "app_name", Type: TypeVarchar},
	{Name: "function_id", Type: TypeUUID},
	{Name: "function_slug", Type: TypeVarchar},
	{Name: "name", Type: TypeVarchar},
	{Name: "start_time", Type: TypeTimestamp},
	{Name: "end_time", Type: TypeTimestamp},
	{Name: "trace_id", Type: TypeVarchar},
	{Name: "span_id", Type: TypeVarchar},
	{Name: "parent_span_id", Type: TypeVarchar, Nullable: true},
	{Name: "attributes", Type: TypeJSON, DuckDB: jsonText, StringKeys: attributeKeys},
	{Name: "output", Type: TypeJSON, Nullable: true, DuckDB: jsonText},
	{Name: "input", Type: TypeJSON, Nullable: true, DuckDB: jsonText},
	// The federation split (lake bucket_at <= W, delta > W); not a query
	// column, so nothing is pushed.
	{Name: "bucket_at", Type: TypeTimestamp},
}}

// Events is inngest.events: one row per received event, never re-written or
// collapsed, so a filter is safe on any column.
var Events = Table{Name: "events", Columns: []Column{
	{Name: "account_id", Type: TypeUUID},
	{Name: "env_id", Type: TypeUUID},
	{Name: "internal_id", Type: TypeVarchar},
	{Name: "received_at", Type: TypeTimestamp},
	{Name: "source", Type: TypeVarchar},
	{Name: "source_id", Type: TypeVarchar, Nullable: true},
	{Name: "event_id", Type: TypeVarchar},
	{Name: "event_name", Type: TypeVarchar},
	{Name: "event_data", Type: TypeJSON, DuckDB: jsonText},
	{Name: "event_v", Type: TypeVarchar},
	{Name: "event_ts", Type: TypeTimestamp},
	{Name: "event_meta", Type: TypeJSON, DuckDB: jsonText},
	// The federation split (lake bucket_at <= W, delta > W); not a query
	// column, so nothing is pushed.
	{Name: "bucket_at", Type: TypeTimestamp},
}}

// RunMetadata is inngest.run_metadata: one row per metadata emission, never
// re-written. Readers roll a span's emissions up to the latest per kind
// (inngest.run_metadata_rollup), so only a filter on the rollup's group key
// (or a column constant within it) is safe to push.
var RunMetadata = Table{Name: "run_metadata", Columns: []Column{
	{Name: "account_id", Type: TypeUUID},
	{Name: "env_id", Type: TypeUUID},
	{Name: "run_id", Type: TypeVarchar},
	{Name: "run_queued_at", Type: TypeTimestamp},
	{Name: "span_id", Type: TypeVarchar},
	{Name: "scope", Type: TypeVarchar},
	{Name: "step_id", Type: TypeVarchar, Nullable: true},
	{Name: "step_index", Type: TypeBigint, Nullable: true, DuckDB: "INTEGER"},
	{Name: "step_attempt", Type: TypeBigint, Nullable: true, DuckDB: "INTEGER"},
	{Name: "kind", Type: TypeVarchar},
	{Name: "is_user", Type: TypeBool},
	{Name: "values", Type: TypeJSON, DuckDB: jsonText},
	{Name: "created_at", Type: TypeTimestamp},
	// The federation split (lake bucket_at <= W, delta > W); not a query
	// column, so nothing is pushed.
	{Name: "bucket_at", Type: TypeTimestamp},
}}
