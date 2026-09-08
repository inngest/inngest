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
