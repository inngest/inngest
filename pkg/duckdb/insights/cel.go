package insights

import (
	"context"
	"fmt"

	sq "github.com/doug-martin/goqu/v9"
)

// XXX: Implemented in PR that'll get merged into this

// CELEventFilters converts exprs (a run.ExpressionHandler.EventExprList)
// into SQL filter expressions against runsInputsCELScope (event.*, targeting
// a runs.inputs array element), keeping only the event.* leaf predicates of
// each expression — see celExprsToSQL's doc comment for why a mixed
// expression's output.*/error.* half is dropped here rather than erroring
// or being converted anyway.
func CELEventFilters(ctx context.Context, exprs []string) ([]sq.Expression, error) {
	return nil, fmt.Errorf("CELEventFilters is unimplemented")
}

// CELOutputFilters is CELEventFilters' output.*/error.* counterpart — see
// its doc comment.
func CELOutputFilters(ctx context.Context, exprs []string) ([]sq.Expression, error) {
	return nil, fmt.Errorf("CELOutputFilters is unimplemented")
}

// CELEventTableFilters converts exprs into SQL filter expressions against
// eventsTableCELScope (event.*, targeting inngest.events' own columns
// directly), for CEL search over the events table itself (see
// pkg/cqrs/duckdbquery.GetEventsByExpressions) rather than a run's
// triggering events. There's no output.*/error.* namespace for a bare
// event search, so any such predicate in exprs is silently dropped, same
// as an unrecognized field already is.
func CELEventTableFilters(ctx context.Context, exprs []string) ([]sq.Expression, error) {
	return nil, fmt.Errorf("CELEventTableFilters is unimplemented")
}

// RenderWhereSQL renders goqu filter expressions (as returned by
// CELEventFilters/CELOutputFilters) into a prepared-statement WHERE-clause
// fragment and its positional args, for callers like pkg/cqrs/duckdbquery
// that build queries as plain SQL strings/"?" args rather than through
// goqu end-to-end. Returns "", nil, nil for an empty filter set.
func RenderWhereSQL(filters []sq.Expression) (string, []any, error) {
	if len(filters) == 0 {
		return "", nil, nil
	}

	return "", nil, fmt.Errorf("RenderWhereSQL is unimplemented")
}
