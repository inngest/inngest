package insights

// knownColumn is one column of a logicalTable's static schema — the same
// allowlist validate checks input columns against, the source
// buildColumnHints traces output items back to, and colType's the declared
// type inferType reports for a bare reference to it.
type knownColumn struct {
	colType ColumnType
	hint    ColumnHint
	// description is a short, human-readable explanation of what this
	// column holds -- surfaced by DESCRIBE (describe.go) so a user
	// exploring the schema sees more than a bare name/type pair.
	description string
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
	description string
	columnOrder []string
	columns     map[string]knownColumn
}

var logicalTables = map[string]logicalTable{
	"runs": {
		name:        "runs",
		view:        "insights_runs",
		description: "Your function runs, one row per run.",
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
			"run_id":       {colType: ColumnTypeString, hint: HintRunID, description: "The unique ID of the run."},
			"queued_at":    {colType: ColumnTypeDatetime, description: "When the run was queued."},
			"scheduled_at": {colType: ColumnTypeDatetime, description: "When the run is scheduled to start, if it was scheduled for later."},
			"started_at":   {colType: ColumnTypeDatetime, description: "When the run started executing."},
			"ended_at":     {colType: ColumnTypeDatetime, description: "When the run finished (succeeded, failed, or was cancelled)."},
			"app_id":       {colType: ColumnTypeString, hint: HintAppID, description: "The app that owns the function this run belongs to."},
			"function_id":  {colType: ColumnTypeString, hint: HintFunctionID, description: "The function that was run."},
			"status":       {colType: ColumnTypeString, description: "The run's current status: Queued, Running, Completed, Failed, or Cancelled."},
			"attributes":   {colType: ColumnTypeJSON, pathHints: spanAttrPathHints, description: "Additional details recorded about the run."},
			// inputs' elements are full marshaled event.Event objects, not
			// flattened event data — event.Event's own payload lives under
			// its "data" field, hence runInputsHints' "data." segment
			// (distinct from events.data's own eventDataHints, one level
			// shallower).
			"inputs": {colType: ColumnTypeJSON, pathHints: runInputsHints, description: "The event(s) that were sent to the function when it ran."},
			"output": {colType: ColumnTypeJSON, description: "The run's final output, once it completes successfully."},
			// event_ids is array-valued (VARCHAR[], so JSON per
			// DuckDBToColumnType); the hint describes each element, not the
			// column's own type. The explicit "[*]" pathHints entry
			// declares that per-element hint directly, rather than leaving
			// it to resolveItemHint's implicit UNNEST(...)/single-index
			// fallback alone.
			"event_ids": {colType: ColumnTypeJSON, hint: HintEventID, pathHints: map[string]ColumnHint{"[*]": HintEventID}, description: "The event(s) that triggered this run."},
			// sessions is STRUCT(key VARCHAR, id VARCHAR)[] on disk, not
			// real JSON — see knownColumn.arrayOfStructs.
			"sessions":                 {colType: ColumnTypeJSON, arrayOfStructs: true, description: "Session identifiers carried by the triggering event(s), if any."},
			"is_deferred":              {colType: ColumnTypeBoolean, description: "Whether this run continues another run that deferred to it, rather than being triggered directly by an event."},
			"defer_parent_function_id": {colType: ColumnTypeString, hint: HintFunctionID, description: "The function of the run that deferred to this one, if is_deferred is true."},
			"defer_parent_run_ids":     {colType: ColumnTypeJSON, hint: HintRunID, pathHints: map[string]ColumnHint{"[*]": HintRunID}, description: "The run(s) that deferred to this run, if is_deferred is true."},
			// metadata/inngest are the run-scoped metadata rollup joined
			// onto insights_runs: metadata is caller (userland) emitted
			// key->value data, inngest is Inngest's own internal metadata —
			// both are a single merged JSON object per run (kind-namespaced,
			// last-emission-wins via json_merge_patch), not the raw
			// per-emission rows the metadata logical table exposes.
			"metadata": {colType: ColumnTypeJSON, pathHints: map[string]ColumnHint{}, description: "Custom metadata your code recorded during the run."},
			"inngest":  {colType: ColumnTypeJSON, pathHints: map[string]ColumnHint{}, description: "Metadata Inngest recorded automatically during the run."},
		},
	},
	"events": {
		name:        "events",
		view:        "insights_events",
		description: "Events sent to Inngest.",
		columnOrder: []string{
			"id", "name", "data", "v", "ts", "meta", "received_at",
		},
		columns: map[string]knownColumn{
			// id is insights_events' own alias for the underlying
			// inngest.events table's event_id column — it's the event's
			// ID, so it carries the same HintEventID event_ids (runs) does.
			"id":          {colType: ColumnTypeString, hint: HintEventID, description: "The unique ID of the event."},
			"name":        {colType: ColumnTypeString, description: "The event's name."},
			"data":        {colType: ColumnTypeJSON, pathHints: eventDataHints, description: "The event's payload."},
			"v":           {colType: ColumnTypeString, description: "The event's version, if one was set when it was sent."},
			"ts":          {colType: ColumnTypeDatetime, description: "The timestamp included with the event when it was sent."},
			"meta":        {colType: ColumnTypeJSON, description: "Additional metadata Inngest recorded about the event."},
			"received_at": {colType: ColumnTypeDatetime, description: "When Inngest received the event."},
		},
	},
	// metadata is backed by the run_metadata_rollup view: one row per
	// (run_id, span_id), with every emission's values merged (by kind,
	// last-emission-wins via json_merge_patch, ordered by created_at) into
	// two JSON objects instead of exposing raw per-emission rows.
	"metadata": {
		name:        "metadata",
		view:        "insights_metadata",
		description: "Custom and automatic metadata recorded during runs.",
		columnOrder: []string{
			"run_id", "run_queued_at",
			"span_id", "scope", "step_id", "step_index",
			"step_attempt", "inngest", "metadata",
		},
		columns: map[string]knownColumn{
			"run_id":        {colType: ColumnTypeString, hint: HintRunID, description: "The run this metadata belongs to."},
			"run_queued_at": {colType: ColumnTypeDatetime, description: "When the run was queued."},
			"span_id":       {colType: ColumnTypeString, description: "Identifies which part of the run (the run itself, a step, or a request) this metadata belongs to."},
			"scope":         {colType: ColumnTypeString, description: "What this metadata was recorded for: the run, a step, a step attempt, or a request."},
			"step_id":       {colType: ColumnTypeString, description: "The step this metadata belongs to, if any."},
			"step_index":    {colType: ColumnTypeNumber, description: "The step's position within the run, if this metadata belongs to a step."},
			"step_attempt":  {colType: ColumnTypeNumber, description: "The attempt number this metadata belongs to, if any."},
			"inngest":       {colType: ColumnTypeJSON, pathHints: map[string]ColumnHint{}, description: "Metadata Inngest recorded automatically."},
			"metadata":      {colType: ColumnTypeJSON, pathHints: map[string]ColumnHint{}, description: "Custom metadata your code recorded."},
		},
	},
	// extended_trace_spans doesn't reuse traceSpanTable: unlike
	// steps/step_attempts, insights_extended_trace_spans (migrations/
	// 000002_insights_views.sql) never selects output/input — otherwise
	// identical column order to traceSpanTable's shared shape.
	"extended_trace_spans": {
		name:        "extended_trace_spans",
		view:        "insights_extended_trace_spans",
		description: "Custom trace events your code recorded during a run.",
		columnOrder: []string{
			"run_id", "run_queued_at", "app_id", "function_id",
			"trace_id", "span_id", "parent_span_id", "name",
			"start_time", "end_time", "attributes", "metadata", "inngest",
		},
		columns: map[string]knownColumn{
			"run_id":         {colType: ColumnTypeString, hint: HintRunID, description: "The run this event belongs to."},
			"run_queued_at":  {colType: ColumnTypeDatetime, description: "When the run was queued."},
			"app_id":         {colType: ColumnTypeString, hint: HintAppID, description: "The app that owns the function this run belongs to."},
			"function_id":    {colType: ColumnTypeString, hint: HintFunctionID, description: "The function this run belongs to."},
			"trace_id":       {colType: ColumnTypeString, description: "Identifies the full execution trace this event belongs to."},
			"span_id":        {colType: ColumnTypeString, description: "The unique ID of this event."},
			"parent_span_id": {colType: ColumnTypeString, description: "The step or event this one is nested under, if any."},
			"name":           {colType: ColumnTypeString, description: "The name given to this trace event."},
			"start_time":     {colType: ColumnTypeDatetime, description: "When this event started."},
			"end_time":       {colType: ColumnTypeDatetime, description: "When this event ended."},
			"attributes":     {colType: ColumnTypeJSON, pathHints: spanAttrPathHints, description: "Additional details recorded with this event."},
			"metadata":       {colType: ColumnTypeJSON, pathHints: map[string]ColumnHint{}, description: "Custom metadata your code recorded during the run."},
			"inngest":        {colType: ColumnTypeJSON, pathHints: map[string]ColumnHint{}, description: "Metadata Inngest recorded automatically during the run."},
		},
	},
	"steps": traceSpanTable("steps", "insights_steps",
		"The steps run during your functions, showing each step's most recent attempt.", stepColumns()),
	"step_attempts": traceSpanTable("step_attempts", "insights_step_attempts",
		"Every attempt of every step run during your functions.", stepColumns()),
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

func traceSpanTable(name, view, description string, extra map[string]knownColumn) logicalTable {
	order := []string{
		"run_id", "run_queued_at", "app_id",
		"function_id", "span_id", "trace_id", "parent_span_id", "name",
		"start_time", "end_time", "output", "input", "attributes",
	}
	cols := map[string]knownColumn{
		"run_id":         {colType: ColumnTypeString, hint: HintRunID, description: "The run this step belongs to."},
		"run_queued_at":  {colType: ColumnTypeDatetime, description: "When the run was queued."},
		"app_id":         {colType: ColumnTypeString, hint: HintAppID, description: "The app that owns the function this run belongs to."},
		"function_id":    {colType: ColumnTypeString, hint: HintFunctionID, description: "The function this run belongs to."},
		"span_id":        {colType: ColumnTypeString, description: "The unique ID of this step attempt."},
		"trace_id":       {colType: ColumnTypeString, description: "Identifies the full execution trace this step belongs to."},
		"parent_span_id": {colType: ColumnTypeString, description: "The step or event this one is nested under, if any."},
		"name":           {colType: ColumnTypeString, description: "The internal name recorded for this step."},
		"start_time":     {colType: ColumnTypeDatetime, description: "When this step attempt started."},
		"end_time":       {colType: ColumnTypeDatetime, description: "When this step attempt ended."},
		"output":         {colType: ColumnTypeJSON, description: "The step's output, once this attempt completes successfully."},
		"input":          {colType: ColumnTypeJSON, description: "The input passed to this step attempt."},
		"attributes":     {colType: ColumnTypeJSON, pathHints: spanAttrPathHints, description: "Additional details recorded about this step attempt."},
	}
	for _, extraCol := range extraStepColumnOrder(extra) {
		order = append(order, extraCol)
		cols[extraCol] = extra[extraCol]
	}
	// Both insights_steps and insights_step_attempts (migrations/
	// 000002_insights_views.sql) LEFT JOIN the metadata rollup, exposing
	// the same run-scoped metadata/inngest columns insights_runs does.
	order = append(order, "metadata", "inngest")
	cols["metadata"] = knownColumn{colType: ColumnTypeJSON, pathHints: map[string]ColumnHint{}, description: "Custom metadata your code recorded during the run."}
	cols["inngest"] = knownColumn{colType: ColumnTypeJSON, pathHints: map[string]ColumnHint{}, description: "Metadata Inngest recorded automatically during the run."}
	return logicalTable{name: name, view: view, description: description, columnOrder: order, columns: cols}
}

// stepColumns is the set of typed columns the insights_step_attempts macro
// unpacks out of attributes on top of extended_trace_spans' base columns
// (plus status, promoted the same way cqrs.ApplyExtractedSpanAttributes
// promotes Attributes.DynamicStatus onto OtelSpan.Status). None of these
// carry an identifier hint (step.id etc. are not app/function/run/event
// IDs).
func stepColumns() map[string]knownColumn {
	return map[string]knownColumn{
		"step_id":           {colType: ColumnTypeString, description: "The unique ID of the step, shared across every attempt."},
		"step_index":        {colType: ColumnTypeNumber, description: "The step's position within the run."},
		"step_name":         {colType: ColumnTypeString, description: "The name given to this step in your code."},
		"step_attempt":      {colType: ColumnTypeNumber, description: "The attempt number for this step, starting at 0."},
		"step_max_attempts": {colType: ColumnTypeNumber, description: "The maximum number of attempts configured for this step."},
		"step_type":         {colType: ColumnTypeString, description: "The kind of step this is (e.g. run, sleep, wait for event, invoke)."},
		"status":            {colType: ColumnTypeString, description: "This step attempt's status."},
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

// allowedFunctions moved to functions.go.
