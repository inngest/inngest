package insights

// knownColumn is one column of a logicalTable's static schema — the same
// allowlist validate checks input columns against, the source
// buildColumnHints traces output items back to, and colType's the declared
// type inferType reports for a bare reference to it.
type knownColumn struct {
	colType ColumnType
	hint    ColumnHint
	// pathHints maps a literal JSON path/key string (as extracted via
	// ->>'/dot-path/json_extract_string) to the hint that key carries. nil
	// for non-JSON columns and JSON columns with no known keys yet.
	//
	// Two reserved key forms describe an array-valued path explicitly,
	// rather than relying only on resolveItemHint's implicit UNNEST(...)/
	// single-index fallback: "<path>[*]" — every element of the array at
	// <path> carries this hint (<path> is "" for the column's own
	// top-level value, e.g. event_ids' "[*]"); "<path>[*].c" — field c of
	// every element of the array at <path> carries it (e.g. runs.inputs'
	// "[*].data._inngest.parent_run_id"). <path> matches jsonPathAccess's
	// extracted literal with any leading "$" stripped (normalizeJSONPath).
	pathHints map[string]ColumnHint
	// arrayOfStructs marks a column whose real DuckDB type is
	// STRUCT(...)[] (e.g. runs.sessions), bucketed as ColumnTypeJSON like
	// any other composite type but NOT actually JSON on disk. DuckDB's dot
	// operator only extracts a field from a bare struct/json/map/union
	// value, not a list of them (confirmed empirically), so col.field must
	// be rewritten to the JSONPath wildcard form col ->> '$[*].field' to
	// actually run — rewriteArrayOfStructAccess (rewrite.go) does that.
	arrayOfStructs bool
}

// logicalTable is one of the six logical tables this package exposes. view
// is the physical DuckDB macro name (parameterized by (account_id,
// env_id) — never exposed as columns, only as call arguments) under the
// inngest catalog that remapTables rewrites bare references to.
// columnOrder is this table's columns in the exact order its macro's
// SELECT list projects them — buildColumnHints' StarExpr expansion
// depends on this being exact; it must match the migration SQL.
//
// A logicalTable value doesn't have to come from this static registry — a
// CTE or FROM-clause subquery is exposed as a synthetic logicalTable too
// (deriveTable, derive.go), with view left "" (never rewritten to a macro
// call by remapTables; its own body already was, recursively).
type logicalTable struct {
	name        string
	view        string
	columnOrder []string
	columns     map[string]knownColumn
}

var logicalTables = map[string]logicalTable{
	"runs": {
		name: "runs",
		view: "insights_runs",
		columnOrder: []string{
			"run_id", "queued_at", "scheduled_at",
			"started_at", "ended_at", "app_id", "function_id", "status",
			"event_ids", "sessions",
			"attributes",
			"inputs", "output",
			"is_deferred",
			"defer_parent_function_id", "defer_parent_run_ids",
			"metadata", "inngest",
		},
		columns: map[string]knownColumn{
			"run_id":       {colType: ColumnTypeString, hint: HintRunID},
			"queued_at":    {colType: ColumnTypeDatetime},
			"scheduled_at": {colType: ColumnTypeDatetime},
			"started_at":   {colType: ColumnTypeDatetime},
			"ended_at":     {colType: ColumnTypeDatetime},
			"app_id":       {colType: ColumnTypeString, hint: HintAppID},
			"function_id":  {colType: ColumnTypeString, hint: HintFunctionID},
			"status":       {colType: ColumnTypeString},
			"attributes":   {colType: ColumnTypeJSON, pathHints: spanAttrPathHints},
			// inputs' elements are full marshaled event.Event objects, not
			// flattened event data — event.Event's own payload lives under
			// its "data" field, hence runInputsHints' "data." segment
			// (distinct from events.data's own eventDataHints, one level
			// shallower).
			"inputs": {colType: ColumnTypeJSON, pathHints: runInputsHints},
			"output": {colType: ColumnTypeJSON},
			// event_ids is array-valued (VARCHAR[], so JSON per
			// DuckDBToColumnType); the hint describes each element, not the
			// column's own type. The explicit "[*]" pathHints entry
			// declares that per-element hint directly, rather than leaving
			// it to resolveItemHint's implicit UNNEST(...)/single-index
			// fallback alone.
			"event_ids": {colType: ColumnTypeJSON, hint: HintEventID, pathHints: map[string]ColumnHint{"[*]": HintEventID}},
			// sessions is STRUCT(key VARCHAR, id VARCHAR)[] on disk, not
			// real JSON — see knownColumn.arrayOfStructs.
			"sessions":                 {colType: ColumnTypeJSON, arrayOfStructs: true},
			"is_deferred":              {colType: ColumnTypeBoolean},
			"defer_parent_function_id": {colType: ColumnTypeString, hint: HintFunctionID},
			"defer_parent_run_ids":     {colType: ColumnTypeJSON, hint: HintRunID, pathHints: map[string]ColumnHint{"[*]": HintRunID}},
			// metadata/inngest are the run-scoped metadata rollup joined
			// onto insights_runs: metadata is caller (userland) emitted
			// key->value data, inngest is Inngest's own internal metadata —
			// both are a single merged JSON object per run (kind-namespaced,
			// last-emission-wins via json_merge_patch), not the raw
			// per-emission rows the metadata logical table exposes.
			"metadata": {colType: ColumnTypeJSON, pathHints: map[string]ColumnHint{}},
			"inngest":  {colType: ColumnTypeJSON, pathHints: map[string]ColumnHint{}},
		},
	},
	"events": {
		name: "events",
		view: "insights_events",
		columnOrder: []string{
			"id", "name", "data", "v", "ts", "meta", "received_at",
		},
		columns: map[string]knownColumn{
			// id is insights_events' own alias for the underlying
			// inngest.events table's event_id column — it's the event's
			// ID, so it carries the same HintEventID event_ids (runs) does.
			"id":          {colType: ColumnTypeString, hint: HintEventID},
			"name":        {colType: ColumnTypeString},
			"data":        {colType: ColumnTypeJSON, pathHints: eventDataHints},
			"v":           {colType: ColumnTypeString},
			"ts":          {colType: ColumnTypeDatetime},
			"meta":        {colType: ColumnTypeJSON},
			"received_at": {colType: ColumnTypeDatetime},
		},
	},
	// metadata is backed by the run_metadata_rollup view: one row per
	// (run_id, span_id), with every emission's values merged (by kind,
	// last-emission-wins via json_merge_patch, ordered by created_at) into
	// two JSON objects instead of exposing raw per-emission rows.
	"metadata": {
		name: "metadata",
		view: "insights_metadata",
		columnOrder: []string{
			"run_id", "run_queued_at",
			"span_id", "scope", "step_id", "step_index",
			"step_attempt", "inngest", "metadata",
		},
		columns: map[string]knownColumn{
			"run_id":        {colType: ColumnTypeString, hint: HintRunID},
			"run_queued_at": {colType: ColumnTypeDatetime},
			"span_id":       {colType: ColumnTypeString},
			"scope":         {colType: ColumnTypeString},
			"step_id":       {colType: ColumnTypeString},
			"step_index":    {colType: ColumnTypeNumber},
			"step_attempt":  {colType: ColumnTypeNumber},
			"inngest":       {colType: ColumnTypeJSON, pathHints: map[string]ColumnHint{}},
			"metadata":      {colType: ColumnTypeJSON, pathHints: map[string]ColumnHint{}},
		},
	},
	"extended_trace_spans": traceSpanTable("extended_trace_spans", "insights_extended_trace_spans", nil),
	"steps":                traceSpanTable("steps", "insights_steps", stepColumns()),
	"step_attempts":        traceSpanTable("step_attempts", "insights_step_attempts", stepColumns()),
}

// spanAttrPathHints is attributes' pathHints, shared by
// extended_trace_spans/steps/step_attempts, keys confirmed against
// pkg/tracing/meta/attributes.go's Attrs map. Every key carries the
// "_inngest." prefix every *Attr constructor applies (pkg/tracing/meta/
// serializers.go's withPrefix, pkg/tracing/meta/consts.go's AttrKeyPrefix)
// before a span's raw OTel attributes ever see it — a new key here needs
// the prefix or it silently reads NULL.
var spanAttrPathHints = map[string]ColumnHint{
	"_inngest.app.name":                     HintAppID,
	"_inngest.function.slug":                HintFunctionID,
	"_inngest.run.id":                       HintRunID,
	"_inngest.debug.run.id":                 HintRunID,
	"_inngest.run.replay_original_run_id":   HintRunID,
	"_inngest.defer.child_run_id":           HintRunID,
	"_inngest.defer.event_id":               HintEventID,
	"_inngest.step.invoke.run.id":           HintRunID,
	"_inngest.event.ids":                    HintEventID,
	"_inngest.step.invoke.trigger.event.id": HintEventID,
	"_inngest.step.invoke.finish.event.id":  HintEventID,
	"_inngest.step.invoke.function.id":      HintFunctionID,
	// parent_run_ids is a StringSliceAttr (*[]string), not a single ULID —
	// the hint describes each element, matching runs.event_ids' precedent.
	// The explicit "<path>[*]" entry below declares that per-element hint
	// directly.
	"_inngest.defer.parent_run_ids":    HintRunID,
	"_inngest.defer.parent_run_ids[*]": HintRunID,
}

var eventDataHints = map[string]ColumnHint{
	"_inngest.fn_slug":         HintFunctionID,
	"_inngest.parent_run_id":   HintRunID,
	"_inngest.parent_app_name": HintAppID,
	"_inngest.parent_fn_slug":  HintFunctionID,
}

var runInputsHints = map[string]ColumnHint{
	"[*].data._inngest.parent_run_id":  HintRunID,
	"[*].data._inngest.parent_app_id":  HintAppID,
	"[*].data._inngest.parent_fn_slug": HintFunctionID,
	// id is each triggering event's own internal ULID — the same
	// HintEventID events.id carries.
	"[*].id": HintEventID,
}

func traceSpanTable(name, view string, extra map[string]knownColumn) logicalTable {
	order := []string{
		"run_id", "run_queued_at", "app_id",
		"function_id", "span_id", "trace_id", "parent_span_id", "name",
		"start_time", "end_time", "output", "input", "attributes",
	}
	cols := map[string]knownColumn{
		"run_id":         {colType: ColumnTypeString, hint: HintRunID},
		"run_queued_at":  {colType: ColumnTypeDatetime},
		"app_id":         {colType: ColumnTypeString, hint: HintAppID},
		"function_id":    {colType: ColumnTypeString, hint: HintFunctionID},
		"span_id":        {colType: ColumnTypeString},
		"trace_id":       {colType: ColumnTypeString},
		"parent_span_id": {colType: ColumnTypeString},
		"name":           {colType: ColumnTypeString},
		"start_time":     {colType: ColumnTypeDatetime},
		"end_time":       {colType: ColumnTypeDatetime},
		"output":         {colType: ColumnTypeJSON},
		"input":          {colType: ColumnTypeJSON},
		"attributes":     {colType: ColumnTypeJSON, pathHints: spanAttrPathHints},
	}
	for _, extraCol := range extraStepColumnOrder(extra) {
		order = append(order, extraCol)
		cols[extraCol] = extra[extraCol]
	}
	return logicalTable{name: name, view: view, columnOrder: order, columns: cols}
}

// stepColumns is the set of typed columns the insights_step_attempts macro
// unpacks out of attributes on top of extended_trace_spans' base columns
// (plus status, promoted the same way cqrs.ApplyExtractedSpanAttributes
// promotes Attributes.DynamicStatus onto OtelSpan.Status). None of these
// carry an identifier hint (step.id etc. are not app/function/run/event
// IDs).
func stepColumns() map[string]knownColumn {
	return map[string]knownColumn{
		"step_id":           {colType: ColumnTypeString},
		"step_index":        {colType: ColumnTypeNumber},
		"step_name":         {colType: ColumnTypeString},
		"step_attempt":      {colType: ColumnTypeNumber},
		"step_max_attempts": {colType: ColumnTypeNumber},
		"step_type":         {colType: ColumnTypeString},
		"status":            {colType: ColumnTypeString},
	}
}

// extraStepColumnOrder returns extra's keys in the fixed order the
// insights_step_attempts macro SELECT list projects them — map iteration
// order is not used, since buildColumnHints' StarExpr expansion depends on
// columnOrder being exact.
func extraStepColumnOrder(extra map[string]knownColumn) []string {
	if extra == nil {
		return nil
	}
	return []string{
		"step_id", "step_index", "step_attempt", "step_max_attempts",
		"step_type", "status",
	}
}

// allowedFunctions is the conservative starting function allowlist. Keys
// are upper-cased, dot-joined function names. No table-function or
// filesystem/catalog-touching surface (read_csv, ATTACH, etc.) is ever
// included here. functionReturnType (typecheck.go) has a case for every
// key here — if you add a function, add its return-type rule too.
//
// Deliberately excludes COALESCE, NULLIF, TRIM, and SUBSTRING even though
// they're ordinary DuckDB functions: pkg/duckdb/parser doesn't implement
// their grammar production (DuckDB gives these their own special
// keyword-based syntax instead of routing through the generic function-call
// rule), so ParseString errors on them unconditionally — confirmed
// empirically. Including them here would be dead code. SUBSTR/LTRIM/RTRIM
// (this package's plain-function-call equivalents) are unaffected.
var allowedFunctions = map[string]bool{
	// Aggregates
	"ANY_VALUE":         true,
	"ARRAY_AGG":         true,
	"AVG":               true,
	"COUNT":             true,
	"JSON_GROUP_ARRAY":  true,
	"JSON_GROUP_OBJECT": true,
	"LIST":              true,
	"MAX":               true,
	"MIN":               true,
	"STRING_AGG":        true,
	"SUM":               true,

	// Scalar / string
	"ABS":            true,
	"CONCAT":         true,
	"CONTAINS":       true,
	"ENDS_WITH":      true,
	"GREATEST":       true,
	"LEAST":          true,
	"LEN":            true,
	"LENGTH":         true,
	"LOWER":          true,
	"LPAD":           true,
	"LTRIM":          true,
	"REGEXP_EXTRACT": true,
	"REGEXP_MATCHES": true,
	"REGEXP_REPLACE": true,
	"REPEAT":         true,
	"REPLACE":        true,
	"REVERSE":        true,
	"ROUND":          true,
	"RPAD":           true,
	"RTRIM":          true,
	"SPLIT_PART":     true,
	"STARTS_WITH":    true,
	"SUBSTR":         true,
	"UPPER":          true,

	// Date/time
	"AGE":            true,
	"DATE_ADD":       true,
	"DATE_DIFF":      true,
	"DATE_PART":      true,
	"DATE_SUB":       true,
	"DATE_TRUNC":     true,
	"DAY":            true,
	"EPOCH":          true,
	"HOUR":           true,
	"LAST_DAY":       true,
	"MAKE_DATE":      true,
	"MAKE_TIMESTAMP": true,
	"MINUTE":         true,
	"MONTH":          true,
	"NOW":            true,
	"SECOND":         true,
	"STRFTIME":       true,
	"TIMEZONE":       true,
	"YEAR":           true,

	// JSON
	"JSON_ARRAY_LENGTH":   true,
	"JSON_CONTAINS":       true,
	"JSON_EXISTS":         true,
	"JSON_EXTRACT":        true,
	"JSON_EXTRACT_STRING": true,
	"JSON_KEYS":           true,
	"JSON_MERGE_PATCH":    true,
	"JSON_QUOTE":          true,
	"JSON_STRUCTURE":      true,
	"JSON_TYPE":           true,
	"JSON_VALID":          true,
	"TO_JSON":             true,

	// Array / conditional
	"IF":     true,
	"IFNULL": true,
	"NVL":    true,
	"TYPEOF": true,
	"UNNEST": true,
}
