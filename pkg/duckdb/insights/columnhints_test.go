package insights

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func hintsFor(t *testing.T, sql string) []ColumnHint {
	t.Helper()
	stmt := mustParse(t, sql)
	scope, err := resolveScope(stmt.From, nil)
	require.NoError(t, err)
	return buildColumnHints(stmt, scope, nil)
}

func TestBuildColumnHintsBareColumn(t *testing.T) {
	require.Equal(t, []ColumnHint{HintRunID}, hintsFor(t, "SELECT run_id FROM runs"))
}

func TestBuildColumnHintsQualifiedColumn(t *testing.T) {
	require.Equal(t, []ColumnHint{HintAppID}, hintsFor(t, "SELECT runs.app_id FROM runs"))
}

func TestBuildColumnHintsAliasedColumn(t *testing.T) {
	require.Equal(t, []ColumnHint{HintFunctionID}, hintsFor(t, "SELECT function_id AS fn FROM runs"))
}

func TestBuildColumnHintsEventIDsArray(t *testing.T) {
	require.Equal(t, []ColumnHint{HintEventID}, hintsFor(t, "SELECT event_ids FROM runs"))
}

func TestBuildColumnHintsEventsID(t *testing.T) {
	require.Equal(t, []ColumnHint{HintEventID}, hintsFor(t, "SELECT id FROM events"))
}

func TestBuildColumnHintsEventsNameNoHint(t *testing.T) {
	require.Equal(t, []ColumnHint{HintNone}, hintsFor(t, "SELECT name FROM events"))
}

func TestBuildColumnHintsNoHintForComputedExpr(t *testing.T) {
	require.Equal(t, []ColumnHint{HintNone}, hintsFor(t, "SELECT UPPER(run_id) FROM runs"))
}

func TestBuildColumnHintsNoHintForAggregate(t *testing.T) {
	require.Equal(t, []ColumnHint{HintNone}, hintsFor(t, "SELECT COUNT(*) FROM runs"))
}

func TestBuildColumnHintsNoHintForAmbiguousJoin(t *testing.T) {
	// run_id exists on both runs and extended_trace_spans -- a bare
	// reference in a joined query can't resolve to exactly one.
	hints := hintsFor(t, "SELECT run_id FROM runs JOIN extended_trace_spans ON runs.run_id = extended_trace_spans.run_id")
	require.Equal(t, []ColumnHint{HintNone}, hints[:1])
}

func TestBuildColumnHintsStarExpandsInOrder(t *testing.T) {
	hints := hintsFor(t, "SELECT * FROM runs")
	tbl := logicalTables["runs"]
	require.Len(t, hints, len(tbl.columnOrder))
	for i, col := range tbl.columnOrder {
		require.Equalf(t, tbl.columns[col].hint, hints[i], "column %d (%s)", i, col)
	}
}

func TestBuildColumnHintsStarExcludeShrinksResult(t *testing.T) {
	hints := hintsFor(t, "SELECT * EXCLUDE (run_id, app_id) FROM runs")
	require.Len(t, hints, len(logicalTables["runs"].columnOrder)-2)
}

func TestBuildColumnHintsQualifiedStar(t *testing.T) {
	hints := hintsFor(t, "SELECT runs.* FROM runs JOIN events ON runs.run_id = events.id")
	require.Len(t, hints, len(logicalTables["runs"].columnOrder))
}

func TestBuildColumnHintsSubPathArrowArrow(t *testing.T) {
	hints := hintsFor(t, "SELECT attributes ->> '_inngest.function.slug' FROM extended_trace_spans")
	require.Equal(t, []ColumnHint{HintFunctionID}, hints)
}

func TestBuildColumnHintsSubPathQualifiedArrowArrow(t *testing.T) {
	// A table-qualified base (needed once two joined tables both have an
	// "attributes" column, making a bare reference ambiguous) must resolve
	// the same hint an unqualified reference does.
	hints := hintsFor(t, "SELECT extended_trace_spans.attributes ->> '_inngest.function.slug' FROM runs JOIN extended_trace_spans ON runs.run_id = extended_trace_spans.run_id")
	require.Equal(t, []ColumnHint{HintFunctionID}, hints)
}

func TestBuildColumnHintsSubPathBareArrow(t *testing.T) {
	// See this plan's Global Constraints: a single-Ident LambdaExpr is how
	// the parser represents `attributes -> 'key'`.
	hints := hintsFor(t, "SELECT attributes -> '_inngest.run.id' FROM extended_trace_spans")
	require.Equal(t, []ColumnHint{HintRunID}, hints)
}

func TestBuildColumnHintsSubPathJSONExtractString(t *testing.T) {
	hints := hintsFor(t, "SELECT json_extract_string(attributes, '_inngest.event.ids') FROM extended_trace_spans")
	require.Equal(t, []ColumnHint{HintEventID}, hints)
}

func TestBuildColumnHintsSubPathQuotedDotAccess(t *testing.T) {
	// attributes."_inngest.function.slug" is DuckDB's quoted dot-access
	// syntax -- it parses as a 2-part *parser.Ident (not a jsonPathAccess
	// shape), so it must resolve via the same spanAttrPathHints entry
	// attributes ->> '_inngest.function.slug' does.
	hints := hintsFor(t, `SELECT attributes."_inngest.function.slug" FROM extended_trace_spans`)
	require.Equal(t, []ColumnHint{HintFunctionID}, hints)
}

func TestBuildColumnHintsSubPathEveryKnownKey(t *testing.T) {
	for path, want := range spanAttrPathHints {
		hints := hintsFor(t, "SELECT attributes ->> '"+path+"' FROM extended_trace_spans")
		require.Equalf(t, []ColumnHint{want}, hints, "path %q", path)
	}
}

func TestBuildColumnHintsSubPathUnknownKey(t *testing.T) {
	hints := hintsFor(t, "SELECT attributes ->> 'not.a.real.key' FROM extended_trace_spans")
	require.Equal(t, []ColumnHint{HintNone}, hints)
}

func TestBuildColumnHintsSubPathNonLiteralPath(t *testing.T) {
	hints := hintsFor(t, "SELECT attributes ->> run_id FROM extended_trace_spans")
	require.Equal(t, []ColumnHint{HintNone}, hints)
}

func TestBuildColumnHintsSubPathFunctionAppliedToExtractedValue(t *testing.T) {
	hints := hintsFor(t, "SELECT UPPER(attributes ->> 'function.id') FROM extended_trace_spans")
	require.Equal(t, []ColumnHint{HintNone}, hints)
}

func TestBuildColumnHintsUnnestOfHintedArray(t *testing.T) {
	// _inngest.defer.parent_run_ids is a JSON array of run IDs -- the
	// path hint itself describes each element (like runs.event_ids), and
	// UNNEST(...) just extracts one, so it inherits the same hint.
	hints := hintsFor(t, "SELECT UNNEST(attributes -> '_inngest.defer.parent_run_ids') FROM extended_trace_spans")
	require.Equal(t, []ColumnHint{HintRunID}, hints)
}

func TestBuildColumnHintsSingleIndexOfHintedArray(t *testing.T) {
	hints := hintsFor(t, "SELECT (attributes -> '_inngest.defer.parent_run_ids')[1] FROM extended_trace_spans")
	require.Equal(t, []ColumnHint{HintRunID}, hints)
}

func TestBuildColumnHintsRangeSliceOfHintedArrayGetsNoHint(t *testing.T) {
	// Unlike a single index, a range slice ("[1:2]") doesn't extract one
	// element of a known hinted type -- it's still an array, so this must
	// not inherit the element hint.
	hints := hintsFor(t, "SELECT (attributes -> '_inngest.defer.parent_run_ids')[1:2] FROM extended_trace_spans")
	require.Equal(t, []ColumnHint{HintNone}, hints)
}

func TestBuildColumnHintsUnnestInFromClauseInheritsArgHint(t *testing.T) {
	// The FROM-clause table-function form (scope.go's addTableFunction)
	// resolves the same hint for its synthetic "unnest" column.
	hints := hintsFor(t, "SELECT unnest FROM extended_trace_spans, UNNEST(attributes -> '_inngest.defer.parent_run_ids')")
	require.Equal(t, []ColumnHint{HintRunID}, hints)
}

func TestBuildColumnHintsUnnestOfNonArrayScalarPathGetsNoHint(t *testing.T) {
	// _inngest.function.id is a plain scalar string, not an array -- unlike
	// _inngest.defer.parent_run_ids (which declares an explicit "[*]"
	// entry), nothing declares that UNNEST-ing it yields FUNCTION_ID
	// elements. Must not assume the path's own (whole-value) hint
	// describes an "element" that doesn't exist.
	hints := hintsFor(t, "SELECT UNNEST(attributes -> '_inngest.function.id') FROM extended_trace_spans")
	require.Equal(t, []ColumnHint{HintNone}, hints)
}

func TestBuildColumnHintsSingleIndexOfNonArrayScalarPathGetsNoHint(t *testing.T) {
	hints := hintsFor(t, "SELECT (attributes ->> '_inngest.function.id')[1] FROM extended_trace_spans")
	require.Equal(t, []ColumnHint{HintNone}, hints)
}

func TestBuildColumnHintsUnnestOfColumnWithoutExplicitArrayHintGetsNoHint(t *testing.T) {
	// run_id itself is a hinted column, but that hint describes the bare
	// column's own value, not "each element of an array" -- run_id
	// declares no "[*]" pathHints entry, so UNNEST(run_id) must not assume
	// it means "an array of run IDs."
	hints := hintsFor(t, "SELECT UNNEST(run_id) FROM runs")
	require.Equal(t, []ColumnHint{HintNone}, hints)
}

func TestBuildColumnHintsUnnestInFromClauseOfNonArrayScalarGetsNoHint(t *testing.T) {
	// Same fix applied to the FROM-clause UNNEST form (scope.go's
	// addTableFunction), which used to call resolveItemHint directly on
	// the argument -- the exact same unsound assumption.
	hints := hintsFor(t, "SELECT unnest FROM runs, UNNEST(run_id)")
	require.Equal(t, []ColumnHint{HintNone}, hints)
}

func TestBuildColumnHintsWholeJSONObjectWithHintedSubPathGetsNoHint(t *testing.T) {
	// events.data has a hinted sub-path (_inngest.parent_run_id), but data
	// itself is a JSON *object* -- selecting the whole object must not be
	// tagged with a sub-field's hint. (This already passes today --
	// col.hint is only ever set explicitly per-column, never derived from
	// pathHints -- but is asserted here as an explicit invariant.)
	hints := hintsFor(t, "SELECT data FROM events")
	require.Equal(t, []ColumnHint{HintNone}, hints)
}

func TestBuildColumnHintsWholeArrayOfObjectsGetsNoHint(t *testing.T) {
	// inputs is an array of event *objects* -- some of their fields are
	// hinted ("[*].id", "[*].data.*"), but the objects themselves aren't
	// hinted values, so selecting/unnesting/indexing the whole array must
	// not inherit any of those sub-field hints. No bare "[*]" entry exists
	// in runInputsHints for exactly this reason.
	require.Equal(t, []ColumnHint{HintNone}, hintsFor(t, "SELECT inputs FROM runs"))
	require.Equal(t, []ColumnHint{HintNone}, hintsFor(t, "SELECT UNNEST(inputs) FROM runs"))
	require.Equal(t, []ColumnHint{HintNone}, hintsFor(t, "SELECT inputs[1] FROM runs"))
}

func TestBuildColumnHintsArrayOfObjectsFieldWithDollarPrefix(t *testing.T) {
	// inputs is a JSON array of triggering-event objects (runs.inputs);
	// runInputsHints' keys apply per-element via the "[*].data." prefix
	// (each element's own payload lives under its "data" field -- see
	// tables.go's "inputs" doc comment). DuckDB's JSONPath operand accepts
	// a leading "$" optionally -- this must resolve the same as without it.
	hints := hintsFor(t, "SELECT inputs ->> '$[*].data._inngest.parent_run_id' FROM runs")
	require.Equal(t, []ColumnHint{HintRunID}, hints)
}

func TestBuildColumnHintsArrayOfObjectsFieldWithoutDollarPrefix(t *testing.T) {
	hints := hintsFor(t, "SELECT inputs ->> '[*].data._inngest.parent_run_id' FROM runs")
	require.Equal(t, []ColumnHint{HintRunID}, hints)
}

func TestBuildColumnHintsArrayOfObjectsUnknownFieldNoHint(t *testing.T) {
	hints := hintsFor(t, "SELECT inputs ->> '$[*].data.not_a_real_key' FROM runs")
	require.Equal(t, []ColumnHint{HintNone}, hints)
}

func TestBuildColumnHintsArrayOfObjectsOwnFieldHint(t *testing.T) {
	// Each inputs[i] is a full triggering-event object with its own "id"
	// (runInputsHints' "[*].id" entry) -- distinct from the nested
	// "data.*" hints above, and not to be confused with events.id's own
	// (unrelated) pathHints-free column-level hint.
	hints := hintsFor(t, "SELECT inputs ->> '$[*].id' FROM runs")
	require.Equal(t, []ColumnHint{HintEventID}, hints)
}

func TestBuildColumnHintsExplicitArrayElementHintTakesPrecedence(t *testing.T) {
	// A column-level hint of HintAppID but an explicit "[*]" pathHints
	// entry of HintRunID -- UNNEST(...) must prefer the explicit
	// per-element declaration (tables.go's "[*]" key convention) over
	// falling back to the column's own (different) whole-column hint.
	scope := &tableScope{entries: []scopeEntry{{alias: "t", table: logicalTable{
		name: "t",
		columns: map[string]knownColumn{
			"arr": {colType: ColumnTypeJSON, hint: HintAppID, pathHints: map[string]ColumnHint{"[*]": HintRunID}},
		},
	}}}}
	stmt := mustParse(t, "SELECT UNNEST(arr)")
	require.Equal(t, HintRunID, resolveItemHint(stmt.Columns[0].Expr, scope))
}

func TestBuildColumnHintsExplicitArrayElementHintOnSubPath(t *testing.T) {
	// _inngest.defer.parent_run_ids is itself a JSON sub-path whose value
	// is an array -- its explicit "<path>[*]" entry must be preferred the
	// same way a top-level column's "[*]" entry is.
	hints := hintsFor(t, "SELECT UNNEST(attributes -> '_inngest.defer.parent_run_ids') FROM extended_trace_spans")
	require.Equal(t, []ColumnHint{HintRunID}, hints)
}

func TestBuildColumnHintsArrayOfStructsFieldHint(t *testing.T) {
	scope := &tableScope{entries: []scopeEntry{{alias: "t", table: logicalTable{
		name: "t",
		columns: map[string]knownColumn{
			"arr": {colType: ColumnTypeJSON, arrayOfStructs: true, pathHints: map[string]ColumnHint{"[*].id": HintRunID}},
		},
	}}}}
	stmt := mustParse(t, "SELECT arr.id")
	require.Equal(t, HintRunID, resolveItemHint(stmt.Columns[0].Expr, scope))
}

func TestBuildColumnHintsArrayOfStructsFieldHintTableQualified(t *testing.T) {
	scope := &tableScope{entries: []scopeEntry{{alias: "t", table: logicalTable{
		name: "t",
		columns: map[string]knownColumn{
			"arr": {colType: ColumnTypeJSON, arrayOfStructs: true, pathHints: map[string]ColumnHint{"[*].id": HintRunID}},
		},
	}}}}
	stmt := mustParse(t, "SELECT t.arr.id")
	require.Equal(t, HintRunID, resolveItemHint(stmt.Columns[0].Expr, scope))
}

func TestBuildColumnHintsArrayOfStructsFieldNoHintForUnknownField(t *testing.T) {
	// sessions is a real arrayOfStructs column (runs.sessions) whose fields
	// (key/id) are caller-defined session tags, not internal identifiers
	// (see pkg/event/sessions.go) -- this must resolve to HintNone, not a
	// guess, confirming the new lookup doesn't hint an undeclared field.
	hints := hintsFor(t, "SELECT sessions.id FROM runs")
	require.Equal(t, []ColumnHint{HintNone}, hints)
}

func TestBuildColumnHintsUnionAgreeingHints(t *testing.T) {
	// Both sides select run_id first -> RUN_ID on both; app_id/function_id
	// -> APP_ID/FUNCTION_ID on both.
	hints := hintsFor(t, "SELECT run_id, app_id FROM runs UNION SELECT run_id, function_id FROM extended_trace_spans")
	require.Equal(t, []ColumnHint{HintRunID, HintNone}, hints)
}

func TestBuildColumnHintsUnionMatchingSecondColumn(t *testing.T) {
	hints := hintsFor(t, "SELECT run_id, app_id FROM runs UNION SELECT run_id, app_id FROM extended_trace_spans")
	require.Equal(t, []ColumnHint{HintRunID, HintAppID}, hints)
}

func TestBuildColumnHintsUnionNoHintOnEitherSide(t *testing.T) {
	hints := hintsFor(t, "SELECT COUNT(*) FROM runs UNION SELECT COUNT(*) FROM extended_trace_spans")
	require.Equal(t, []ColumnHint{HintNone}, hints)
}

func TestBuildColumnHintsMultiWayUnion(t *testing.T) {
	hints := hintsFor(t, "SELECT run_id FROM runs UNION SELECT run_id FROM extended_trace_spans UNION SELECT run_id FROM steps")
	require.Equal(t, []ColumnHint{HintRunID}, hints)
}

func TestBuildColumnHintsExplicitArrayElementHintSingleIndexTakesPrecedence(t *testing.T) {
	// Single-index counterpart to TestBuildColumnHintsExplicitArrayElementHintTakesPrecedence
	// above -- proves resolveArrayElementHint's explicit-entry check applies
	// to the "[n]" path too, not just UNNEST(...).
	scope := &tableScope{entries: []scopeEntry{{alias: "t", table: logicalTable{
		name: "t",
		columns: map[string]knownColumn{
			"arr": {colType: ColumnTypeJSON, hint: HintAppID, pathHints: map[string]ColumnHint{"[*]": HintRunID}},
		},
	}}}}
	stmt := mustParse(t, "SELECT arr[1]")
	require.Equal(t, HintRunID, resolveItemHint(stmt.Columns[0].Expr, scope))
}

func TestBuildColumnHintsEventIDsExplicitHintUnnest(t *testing.T) {
	// event_ids now declares its element hint explicitly (pathHints["[*]"])
	// rather than relying solely on resolveArrayElementHint's implicit
	// fallback -- this must still resolve the same EVENT_ID hint as before.
	hints := hintsFor(t, "SELECT UNNEST(event_ids) FROM runs")
	require.Equal(t, []ColumnHint{HintEventID}, hints)
}

func TestBuildColumnHintsEventIDsExplicitHintSingleIndex(t *testing.T) {
	hints := hintsFor(t, "SELECT event_ids[1] FROM runs")
	require.Equal(t, []ColumnHint{HintEventID}, hints)
}
