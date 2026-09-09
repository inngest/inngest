package insights

// ColumnHint identifies a result column as a specific kind of ID string (see
// docs/plans/010-duckdb-insights-query-layer.md's Column hints section).
// HintNone (the zero value) means either the column isn't a known
// identifier column, or buildColumnHints couldn't trace it back to exactly
// one — no hint is returned rather than a guessed one.
type ColumnHint int

const (
	HintNone ColumnHint = iota
	HintAppID
	HintFunctionID
	HintRunID
	HintEventID
)

// String is the same label the GQL layer's InsightsColumnHint enum uses
// (resolvers/insights.go's toGQLColumnHint), reused as-is by
// cmd/gen-insights-schema's JSON dump. Returns "" for HintNone -- a caller
// serializing this should omit the field entirely rather than write out an
// empty hint.
func (h ColumnHint) String() string {
	switch h {
	case HintAppID:
		return "APP_ID"
	case HintFunctionID:
		return "FUNCTION_ID"
	case HintRunID:
		return "RUN_ID"
	case HintEventID:
		return "EVENT_ID"
	default:
		return ""
	}
}
