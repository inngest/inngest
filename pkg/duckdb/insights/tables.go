package insights

// knownColumn is one column of a logicalTable's static schema — the same
// allowlist validate checks input columns against, the source
// buildColumnHints traces output items back to, and colType's the declared
// type inferType reports for a bare reference to it.
type knownColumn struct {
	colType ColumnType
	// description is a short, human-readable explanation of what this
	// column holds -- surfaced by DESCRIBE (describe.go) so a user
	// exploring the schema sees more than a bare name/type pair.
	description string
	// pathHints is every hint this column carries, at any JSON sub-path
	// depth, each keyed by an explicit []PathSegment rather than a single
	// dotted string -- see PathSegment's own doc comment for why. A nil
	// pathHints is a column with no hint of its own and no known JSON
	// sub-paths.
	//
	// An entry with a nil/empty Path (built by rootHint, or hint(h) with
	// no further segments) is the column's own whole-value hint -- what
	// used to be a separate `hint ColumnHint` field, folded in here so
	// every hint this column carries lives in one place. A column hinted
	// only at its own whole value (e.g. run_id) has no other entries at
	// all.
	//
	// Two other path shapes describe an array-valued path explicitly,
	// rather than relying only on resolveItemHint's implicit
	// UNNEST(...)/single-index fallback: {wc} — every element of the
	// array at that point carries this hint (e.g. event_ids' {wc});
	// {wc, seg(c)} — field c of every element of that array carries it
	// (e.g. runs.inputs' {wc, seg("data"), seg("_inngest"),
	// seg("parent_run_id")}).
	pathHints []PathHint
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
			"run_id":       {colType: ColumnTypeString, pathHints: rootHint(HintRunID), description: "The unique ID of the run."},
			"queued_at":    {colType: ColumnTypeDatetime, description: "When the run was queued."},
			"scheduled_at": {colType: ColumnTypeDatetime, description: "When the run is scheduled to start, if it was scheduled for later."},
			"started_at":   {colType: ColumnTypeDatetime, description: "When the run started executing."},
			"ended_at":     {colType: ColumnTypeDatetime, description: "When the run finished (succeeded, failed, or was cancelled)."},
			"app_id":       {colType: ColumnTypeString, pathHints: rootHint(HintAppID), description: "The app that owns the function this run belongs to."},
			"function_id":  {colType: ColumnTypeString, pathHints: rootHint(HintFunctionID), description: "The function that was run."},
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
			// DuckDBToColumnType). The hint describes each element, not the
			// column's own value -- an array of event IDs isn't itself an
			// event ID -- so this declares only "[*]", no "" root entry;
			// the explicit "[*]" pathHints entry declares that per-element
			// hint directly, rather than leaving it to resolveItemHint's
			// implicit UNNEST(...)/single-index fallback alone.
			"event_ids": {colType: ColumnTypeJSON, pathHints: []PathHint{hint(HintEventID, wc)}, description: "The event(s) that triggered this run."},
			// sessions is STRUCT(key VARCHAR, id VARCHAR)[] on disk, not
			// real JSON — see knownColumn.arrayOfStructs. Same "[*]"-only
			// convention as event_ids, just one level up: each element is
			// itself the {key,id} object HintSession describes, not a bare
			// scalar, so there's no separate "[*].id"-only hint either.
			"sessions":                 {colType: ColumnTypeJSON, arrayOfStructs: true, pathHints: []PathHint{hint(HintSession, wc)}, description: "Session identifiers carried by the triggering event(s), if any."},
			"is_deferred":              {colType: ColumnTypeBoolean, description: "Whether this run continues another run that deferred to it, rather than being triggered directly by an event."},
			"defer_parent_function_id": {colType: ColumnTypeString, pathHints: rootHint(HintFunctionID), description: "The function of the run that deferred to this one, if is_deferred is true."},
			"defer_parent_run_ids":     {colType: ColumnTypeJSON, pathHints: []PathHint{hint(HintRunID, wc)}, description: "The run(s) that deferred to this run, if is_deferred is true."},
			// metadata/inngest are the run-scoped metadata rollup joined
			// onto insights_runs: metadata is caller (userland) emitted
			// key->value data, inngest is Inngest's own internal metadata —
			// both are a single merged JSON object per run (kind-namespaced,
			// last-emission-wins via json_merge_patch), not the raw
			// per-emission rows the metadata logical table exposes.
			"metadata": {colType: ColumnTypeJSON, pathHints: nil, description: "Custom metadata your code recorded during the run."},
			"inngest":  {colType: ColumnTypeJSON, pathHints: nil, description: "Metadata Inngest recorded automatically during the run."},
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
			"id":          {colType: ColumnTypeString, pathHints: rootHint(HintEventID), description: "The unique ID of the event."},
			"name":        {colType: ColumnTypeString, description: "The event's name."},
			"data":        {colType: ColumnTypeJSON, pathHints: eventDataHints, description: "The event's payload."},
			"v":           {colType: ColumnTypeString, description: "The event's version, if one was set when it was sent."},
			"ts":          {colType: ColumnTypeDatetime, description: "The timestamp included with the event when it was sent."},
			"meta":        {colType: ColumnTypeJSON, pathHints: eventMetaHints, description: "Additional metadata Inngest recorded about the event."},
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
			"run_id":        {colType: ColumnTypeString, pathHints: rootHint(HintRunID), description: "The run this metadata belongs to."},
			"run_queued_at": {colType: ColumnTypeDatetime, description: "When the run was queued."},
			"span_id":       {colType: ColumnTypeString, description: "Identifies which part of the run (the run itself, a step, or a request) this metadata belongs to."},
			"scope":         {colType: ColumnTypeString, description: "What this metadata was recorded for: the run, a step, a step attempt, or a request."},
			"step_id":       {colType: ColumnTypeString, description: "The step this metadata belongs to, if any."},
			"step_index":    {colType: ColumnTypeNumber, description: "The step's position within the run, if this metadata belongs to a step."},
			"step_attempt":  {colType: ColumnTypeNumber, description: "The attempt number this metadata belongs to, if any."},
			"inngest":       {colType: ColumnTypeJSON, pathHints: nil, description: "Metadata Inngest recorded automatically."},
			"metadata":      {colType: ColumnTypeJSON, pathHints: nil, description: "Custom metadata your code recorded."},
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
			"run_id":         {colType: ColumnTypeString, pathHints: rootHint(HintRunID), description: "The run this event belongs to."},
			"run_queued_at":  {colType: ColumnTypeDatetime, description: "When the run was queued."},
			"app_id":         {colType: ColumnTypeString, pathHints: rootHint(HintAppID), description: "The app that owns the function this run belongs to."},
			"function_id":    {colType: ColumnTypeString, pathHints: rootHint(HintFunctionID), description: "The function this run belongs to."},
			"trace_id":       {colType: ColumnTypeString, description: "Identifies the full execution trace this event belongs to."},
			"span_id":        {colType: ColumnTypeString, description: "The unique ID of this event."},
			"parent_span_id": {colType: ColumnTypeString, description: "The step or event this one is nested under, if any."},
			"name":           {colType: ColumnTypeString, description: "The name given to this trace event."},
			"start_time":     {colType: ColumnTypeDatetime, description: "When this event started."},
			"end_time":       {colType: ColumnTypeDatetime, description: "When this event ended."},
			"attributes":     {colType: ColumnTypeJSON, pathHints: spanAttrPathHints, description: "Additional details recorded with this event."},
			"metadata":       {colType: ColumnTypeJSON, pathHints: nil, description: "Custom metadata your code recorded during the run."},
			"inngest":        {colType: ColumnTypeJSON, pathHints: nil, description: "Metadata Inngest recorded automatically during the run."},
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
// Every key here is ONE literal, flat top-level key of the attributes
// JSON object -- not nested access -- so each is written as a single
// seg() call, dots and all: an OTel attribute name is always flat
// (pkg/duckdb/insights' own CLAUDE.md), confirmed against DuckDB's own
// ->>'/json_extract_string semantics (a bare, non-"$"-prefixed operand is
// one literal key; see queryPath's doc comment). Writing "_inngest.app.
// name" as a single seg("_inngest.app.name") instead of
// seg("_inngest"), seg("app"), seg("name") is what actually encodes that
// fact, rather than leaving it to a reader's guess.
var spanAttrPathHints = []PathHint{
	hint(HintAppID, seg("_inngest.app.name")),
	hint(HintFunctionID, seg("_inngest.function.slug")),
	hint(HintRunID, seg("_inngest.run.id")),
	hint(HintRunID, seg("_inngest.debug.run.id")),
	hint(HintRunID, seg("_inngest.run.replay_original_run_id")),
	hint(HintRunID, seg("_inngest.defer.child_run_id")),
	hint(HintEventID, seg("_inngest.defer.event_id")),
	hint(HintRunID, seg("_inngest.step.invoke.run.id")),
	hint(HintEventID, seg("_inngest.step.invoke.trigger.event.id")),
	hint(HintEventID, seg("_inngest.step.invoke.finish.event.id")),
	hint(HintFunctionID, seg("_inngest.step.invoke.function.id")),
	// event.ids and defer.parent_run_ids are both a StringSliceAttr
	// (*[]string), not a single ULID -- the hint describes each element,
	// not the array value itself, matching runs.event_ids' own wc-only
	// precedent (tables.go) -- no bare (Path-less) entry for either. Each
	// is still one flat key (one seg() call) followed by the wc step, not
	// two structural segments.
	hint(HintEventID, seg("_inngest.event.ids"), wc),
	hint(HintRunID, seg("_inngest.defer.parent_run_ids"), wc),
}

// eventDataHints is events.data's pathHints -- unlike spanAttrPathHints,
// data's "_inngest" key holds a genuinely nested object
// (event.DeferredScheduleMetadata, pkg/event/defer.go, marshaled under
// consts.InngestEventDataPrefix), so each entry here is two real
// structural segments, not one flat dotted key.
var eventDataHints = []PathHint{
	hint(HintFunctionID, seg("_inngest"), seg("fn_slug")),
	hint(HintRunID, seg("_inngest"), seg("parent_run_id")),
	hint(HintAppID, seg("_inngest"), seg("parent_app_name")),
	hint(HintFunctionID, seg("_inngest"), seg("parent_fn_slug")),
}

// eventMetaHints is events.meta's pathHints. sessions is event.EventMeta's
// own "sessions" field (pkg/event/sessions.go) -- a plain
// map[string]string of session key -> id, unlike runs.sessions' STRUCT
// array, but hinted the same way (HintSession) since a UI consuming
// either still needs a key/id pair to build a session link.
var eventMetaHints = []PathHint{
	hint(HintSession, seg("sessions")),
}

// runInputsHints is runs.inputs' pathHints -- each entry starts with wc
// (every element of the inputs array), then real structural nesting
// within that element (a full marshaled event.Event object), matching
// eventDataHints' own "data._inngest.*" shape one level deeper.
var runInputsHints = []PathHint{
	hint(HintRunID, wc, seg("data"), seg("_inngest"), seg("parent_run_id")),
	hint(HintAppID, wc, seg("data"), seg("_inngest"), seg("parent_app_id")),
	hint(HintFunctionID, wc, seg("data"), seg("_inngest"), seg("parent_fn_slug")),
	// id is each triggering event's own internal ULID — the same
	// HintEventID events.id carries.
	hint(HintEventID, wc, seg("id")),
	// meta.sessions mirrors eventMetaHints -- inputs' elements are full
	// marshaled event.Event objects, so meta sits one level deeper here
	// than on events.meta directly (see runInputsHints' own doc comment
	// on tables.go's "inputs" column entry).
	hint(HintSession, wc, seg("meta"), seg("sessions")),
}

func traceSpanTable(name, view, description string, extra map[string]knownColumn) logicalTable {
	order := []string{
		"run_id", "run_queued_at", "app_id",
		"function_id", "span_id", "trace_id", "parent_span_id", "name",
		"start_time", "end_time", "output", "input", "attributes",
	}
	cols := map[string]knownColumn{
		"run_id":         {colType: ColumnTypeString, pathHints: rootHint(HintRunID), description: "The run this step belongs to."},
		"run_queued_at":  {colType: ColumnTypeDatetime, description: "When the run was queued."},
		"app_id":         {colType: ColumnTypeString, pathHints: rootHint(HintAppID), description: "The app that owns the function this run belongs to."},
		"function_id":    {colType: ColumnTypeString, pathHints: rootHint(HintFunctionID), description: "The function this run belongs to."},
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
	cols["metadata"] = knownColumn{colType: ColumnTypeJSON, pathHints: nil, description: "Custom metadata your code recorded during the run."}
	cols["inngest"] = knownColumn{colType: ColumnTypeJSON, pathHints: nil, description: "Metadata Inngest recorded automatically during the run."}
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
