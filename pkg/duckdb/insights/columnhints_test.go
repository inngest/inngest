package insights

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func hintsFor(t *testing.T, sql string) []ColumnHint {
	t.Helper()
	stmt := mustParse(t, sql)
	scope, err := resolveScope(stmt.From, nil, nil)
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
	// event_ids' hint describes each element (its only pathHints entry is
	// {wc}), not the array value itself -- selecting the whole array
	// bare must not inherit it (same policy as
	// TestBuildColumnHintsWholeArrayOfObjectsGetsNoHint).
	require.Equal(t, []ColumnHint{HintNone}, hintsFor(t, "SELECT event_ids FROM runs"))
}

func TestBuildColumnHintsEventsID(t *testing.T) {
	require.Equal(t, []ColumnHint{HintEventID}, hintsFor(t, "SELECT id FROM events"))
}

func TestBuildColumnHintsEventsNameNoHint(t *testing.T) {
	require.Equal(t, []ColumnHint{HintNone}, hintsFor(t, "SELECT name FROM events"))
}

// TestBuildColumnHintsSessionsArray mirrors TestBuildColumnHintsEventIDsArray:
// runs.sessions' only pathHints entry is {wc} (its elements' hint), so a
// bare column reference to the whole array gets no hint.
func TestBuildColumnHintsSessionsArray(t *testing.T) {
	require.Equal(t, []ColumnHint{HintNone}, hintsFor(t, "SELECT sessions FROM runs"))
}

func TestBuildColumnHintsSessionsUnnest(t *testing.T) {
	require.Equal(t, []ColumnHint{HintSession}, hintsFor(t, "SELECT UNNEST(sessions) FROM runs"))
}

func TestBuildColumnHintsSessionsIndex(t *testing.T) {
	require.Equal(t, []ColumnHint{HintSession}, hintsFor(t, "SELECT sessions[1] FROM runs"))
}

// TestBuildColumnHintsSessionsFieldNoHint confirms a specific struct field
// off one session element (e.g. sessions[1].id) is NOT separately hinted --
// unlike the other ID hints, HintSession describes the whole {key,id}
// object, since a UI needs both fields together to build a session link.
func TestBuildColumnHintsSessionsFieldNoHint(t *testing.T) {
	require.Equal(t, []ColumnHint{HintNone}, hintsFor(t, "SELECT sessions[1].id FROM runs"))
}

func TestBuildColumnHintsEventMetaSessions(t *testing.T) {
	require.Equal(t, []ColumnHint{HintSession}, hintsFor(t, "SELECT meta.sessions FROM events"))
}

func TestBuildColumnHintsEventMetaSessionsJSONArrow(t *testing.T) {
	require.Equal(t, []ColumnHint{HintSession}, hintsFor(t, "SELECT meta ->> 'sessions' FROM events"))
}

// TestBuildColumnHintsRunInputsMetaSessions mirrors
// TestBuildColumnHintsRunInputsParentRunID (data._inngest.parent_run_id):
// inputs' elements are full marshaled event.Event objects, so meta.sessions
// sits one level deeper here than on events.meta directly.
func TestBuildColumnHintsRunInputsMetaSessions(t *testing.T) {
	hints := hintsFor(t, "SELECT inputs ->> '$[*].meta.sessions' FROM runs")
	require.Equal(t, []ColumnHint{HintSession}, hints)
}

func TestBuildColumnHintsNoHintForComputedExpr(t *testing.T) {
	require.Equal(t, []ColumnHint{HintNone}, hintsFor(t, "SELECT UPPER(run_id) FROM runs"))
}

func TestBuildColumnHintsNoHintForAggregate(t *testing.T) {
	require.Equal(t, []ColumnHint{HintNone}, hintsFor(t, "SELECT COUNT(*) FROM runs"))
}

// TestBuildColumnHints{Max,Min,AnyValue,ArgMax}PreservesArgHint cover
// preservesArgTypeAndHint's family (functionInfo's own doc comment):
// these functions report one of their input rows' own actual,
// unmodified values, so the result carries that value's own hint too --
// same policy as a bare column reference, just reached through an
// aggregate/window call.
func TestBuildColumnHintsMaxPreservesArgHint(t *testing.T) {
	require.Equal(t, []ColumnHint{HintRunID}, hintsFor(t, "SELECT max(run_id) FROM runs"))
}

func TestBuildColumnHintsMinPreservesArgHint(t *testing.T) {
	require.Equal(t, []ColumnHint{HintRunID}, hintsFor(t, "SELECT min(run_id) FROM runs"))
}

func TestBuildColumnHintsAnyValuePreservesArgHint(t *testing.T) {
	require.Equal(t, []ColumnHint{HintRunID}, hintsFor(t, "SELECT any_value(run_id) FROM runs"))
}

// TestBuildColumnHintsArgMaxPreservesRepresentativeArgHint confirms the
// hint is always taken from args[0] (the "arg" to report), not args[1]
// (arg_max's "val" the row is chosen by) -- mirroring preservesArgType's
// own type-inference precedent for the same family.
func TestBuildColumnHintsArgMaxPreservesRepresentativeArgHint(t *testing.T) {
	require.Equal(t, []ColumnHint{HintRunID}, hintsFor(t, "SELECT arg_max(run_id, queued_at) FROM runs"))
}

// TestBuildColumnHintsNestedAggregatesPreserveHint confirms the
// recursion through exprPathHints composes: MAX(FIRST(run_id)) resolves
// FIRST(run_id)'s own hint the same way a bare run_id would, then MAX
// reports that back unmodified too.
func TestBuildColumnHintsNestedAggregatesPreserveHint(t *testing.T) {
	require.Equal(t, []ColumnHint{HintRunID}, hintsFor(t, "SELECT max(first(run_id)) FROM runs"))
}

// TestBuildColumnHints{Greatest,Median}GetNoHint cover the two exclusion
// reasons functionInfo's own doc comment names: GREATEST picks between
// independently-sourced expressions row-by-row rather than accumulating
// one column, and MEDIAN can interpolate a value that was never actually
// present in the input -- both stay plain preservesArgType (type only,
// no hint), unlike MAX/MIN/ANY_VALUE above.
func TestBuildColumnHintsGreatestGetsNoHint(t *testing.T) {
	require.Equal(t, []ColumnHint{HintNone}, hintsFor(t, "SELECT greatest(run_id, app_id) FROM runs"))
}

func TestBuildColumnHintsMedianGetsNoHint(t *testing.T) {
	require.Equal(t, []ColumnHint{HintNone}, hintsFor(t, "SELECT median(run_id) FROM runs"))
}

// TestBuildColumnHints{List,ArrayAgg}WholeValueGetsNoHint: LIST()/
// ARRAY_AGG()'s result is a LIST value, not itself a run_id -- only its
// elements (TestBuildColumnPathHints{List,ArrayAgg}ElementsInheritArgHint,
// below) carry the hint.
func TestBuildColumnHintsListWholeValueGetsNoHint(t *testing.T) {
	require.Equal(t, []ColumnHint{HintNone}, hintsFor(t, "SELECT list(run_id) FROM runs"))
}

func TestBuildColumnHintsArrayAggWholeValueGetsNoHint(t *testing.T) {
	require.Equal(t, []ColumnHint{HintNone}, hintsFor(t, "SELECT array_agg(run_id) FROM runs"))
}

// pathHintsFor mirrors hintsFor, for buildColumnPathHints' fuller
// []PathHint result -- needed to see a non-root (e.g. {wc}) entry
// hintsFor's plain ColumnHint result can't represent.
func pathHintsFor(t *testing.T, sql string) [][]PathHint {
	t.Helper()
	stmt := mustParse(t, sql)
	scope, err := resolveScope(stmt.From, nil, nil)
	require.NoError(t, err)
	return buildColumnPathHints(stmt, scope, nil)
}

// TestBuildColumnPathHints{List,ArrayAgg}ElementsInheritArgHint is
// listReturnType's own case: LIST(run_id)/ARRAY_AGG(run_id)'s *elements*
// -- not the LIST value itself, see the WholeValueGetsNoHint tests above
// -- carry run_id's own hint, one array-wildcard segment deeper, exactly
// like runs.event_ids' own declared {wc} pathHints entry (tables.go).
func TestBuildColumnPathHintsListElementsInheritArgHint(t *testing.T) {
	require.Equal(t, [][]PathHint{{hint(HintRunID, wc)}}, pathHintsFor(t, "SELECT list(run_id) FROM runs"))
}

func TestBuildColumnPathHintsArrayAggElementsInheritArgHint(t *testing.T) {
	require.Equal(t, [][]PathHint{{hint(HintRunID, wc)}}, pathHintsFor(t, "SELECT array_agg(run_id) FROM runs"))
}

// TestBuildColumnHintsUnnestOfListStillArrayShapedGetsNoHint:
// list(sessions) aggregates each row's own sessions *array* into an outer
// LIST, so it's genuinely a LIST of arrays (two wc segments deep --
// TestBuildColumnPathHintsListElementsInheritArgHint-style, but wrapping
// an already-array-typed column instead of a scalar one). A single
// UNNEST(...) only strips the outer (aggregation-added) wc, leaving one
// of the original per-row sessions arrays -- still array-shaped, so its
// own root/whole-value hint is None, exactly like a bare `sessions`
// reference itself (TestBuildColumnHintsSessionsArray).
func TestBuildColumnHintsUnnestOfListStillArrayShapedGetsNoHint(t *testing.T) {
	require.Equal(t, []ColumnHint{HintNone}, hintsFor(t, "SELECT UNNEST(list(sessions)) FROM runs"))
}

// TestBuildColumnHintsDoubleUnnestOfListPreservesArgHint: a *second*
// UNNEST strips the remaining (sessions' own) wc, landing on an
// individual session -- proving unnestReturnType's projection composes
// through repeated application, symmetric with
// TestBuildColumnPathHintsListElementsInheritArgHint's own composition
// going the other direction.
func TestBuildColumnHintsDoubleUnnestOfListPreservesArgHint(t *testing.T) {
	require.Equal(t, []ColumnHint{HintSession}, hintsFor(t, "SELECT UNNEST(UNNEST(list(sessions))) FROM runs"))
}

// TestBuildColumnHintsListOfAlreadyArrayColumnDoubleWildcard confirms a
// LIST()/ARRAY_AGG() over a column that's *already* array-typed produces
// a genuinely double-nested pathHints entry (two wc segments deep, one
// for the aggregation and one for inputs' own array-ness), and that a
// JSON sub-path access naming both wildcards explicitly
// ("$[*][*].meta.sessions") resolves against it correctly -- exercising
// resolveColumnPathHints' jsonPathAccess case recursing through a
// FunctionExpr base (list(inputs)) rather than a bare column.
func TestBuildColumnHintsListOfAlreadyArrayColumnDoubleWildcard(t *testing.T) {
	require.Equal(t, []ColumnHint{HintSession},
		hintsFor(t, "SELECT list(inputs) -> '$[*][*].meta.sessions' FROM runs"))
}

// TestBuildColumnHintsUnnestOfDoubleWildcardJSONPathStillResolves proves
// the {wc} sibling projectJSONSubPathHints (columnhints.go) adds for a
// wildcard-collapsed JSON sub-path access survives a further UNNEST --
// confirmed against real DuckDB that "[*][*]..." flattens into a single
// result array rather than nesting, so UNNEST(...) directly reaches an
// individual session in one step, exactly like
// TestBuildColumnHintsListOfAlreadyArrayColumnDoubleWildcard's own
// (non-UNNESTed) root hint says it should.
func TestBuildColumnHintsUnnestOfDoubleWildcardJSONPathStillResolves(t *testing.T) {
	require.Equal(t, []ColumnHint{HintSession},
		hintsFor(t, "SELECT UNNEST(list(inputs) -> '$[*][*].meta.sessions') FROM runs"))
}

// TestBuildColumnHintsUnnestOfSingleWildcardJSONPathStillResolves is the
// same proof one wildcard level down -- inputs itself (not an
// aggregated list(inputs)) already has its own {wc, meta, sessions}
// pathHints entry, so a single "[*]" here is enough.
func TestBuildColumnHintsUnnestOfSingleWildcardJSONPathStillResolves(t *testing.T) {
	require.Equal(t, []ColumnHint{HintSession},
		hintsFor(t, "SELECT UNNEST(inputs ->> '$[*].meta.sessions') FROM runs"))
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
		require.Equalf(t, tbl.columns[col].hint(), hints[i], "column %d (%s)", i, col)
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
	// _inngest.event.ids is itself a JSON array -- unlike a scalar
	// sub-path, extracting its whole value (rather than UNNEST-ing it,
	// TestBuildColumnHintsUnnestOfHintedArray-style) must not inherit its
	// elements' own hint (spanAttrPathHints declares only a {seg(...), wc}
	// entry for it, no whole-value entry -- same policy as
	// runs.event_ids).
	hints := hintsFor(t, "SELECT json_extract_string(attributes, '_inngest.event.ids') FROM extended_trace_spans")
	require.Equal(t, []ColumnHint{HintNone}, hints)
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
	for _, ph := range spanAttrPathHints {
		// A one-segment Path is one flat literal key, reachable directly
		// via attributes ->> '<key>'. A {seg(...), wc} entry (event.ids,
		// defer.parent_run_ids) describes an array's elements, not the
		// key's own whole value -- TestBuildColumnHintsUnnestOfHintedArray
		// and friends already cover those via UNNEST(...) instead.
		if len(ph.Path) != 1 {
			continue
		}
		path := ph.Path[0].Key
		hints := hintsFor(t, "SELECT attributes ->> '"+path+"' FROM extended_trace_spans")
		require.Equalf(t, []ColumnHint{ph.Hint}, hints, "path %q", path)
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
	// Confirmed against DuckDB directly: a JSON path operand only gets
	// structural ("."/"[*]" as real steps) treatment with a leading "$" --
	// '[*].data...' (or '[0]', or any other bare string) is always one
	// literal top-level key, and there is no key by that literal name, so
	// this must resolve to no hint, not the same as the "$"-prefixed form
	// TestBuildColumnHintsArrayOfObjectsFieldWithDollarPrefix covers.
	hints := hintsFor(t, "SELECT inputs ->> '[*].data._inngest.parent_run_id' FROM runs")
	require.Equal(t, []ColumnHint{HintNone}, hints)
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
			"arr": {colType: ColumnTypeJSON, pathHints: []PathHint{hint(HintAppID), hint(HintRunID, wc)}},
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
			"arr": {colType: ColumnTypeJSON, arrayOfStructs: true, pathHints: []PathHint{hint(HintRunID, wc, seg("id"))}},
		},
	}}}}
	stmt := mustParse(t, "SELECT arr.id")
	require.Equal(t, HintRunID, resolveItemHint(stmt.Columns[0].Expr, scope))
}

func TestBuildColumnHintsArrayOfStructsFieldHintTableQualified(t *testing.T) {
	scope := &tableScope{entries: []scopeEntry{{alias: "t", table: logicalTable{
		name: "t",
		columns: map[string]knownColumn{
			"arr": {colType: ColumnTypeJSON, arrayOfStructs: true, pathHints: []PathHint{hint(HintRunID, wc, seg("id"))}},
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
			"arr": {colType: ColumnTypeJSON, pathHints: []PathHint{hint(HintAppID), hint(HintRunID, wc)}},
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
