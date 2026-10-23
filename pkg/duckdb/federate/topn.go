package federate

import (
	"fmt"
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/schema"
)

// OrderKey is one ORDER BY key: a scalar column, ascending unless Desc,
// NULLs last unless NullsFirst.
type OrderKey struct {
	Column     string
	Desc       bool
	NullsFirst bool
}

// TopN pushes a query's ORDER BY ... LIMIT into one table's federated
// source: the delta keeps only its own first Limit rows by OrderBy, as does
// the lake side for runs (whose collapse DuckDB can't push a top-n
// through), so the source holds a few times Limit rather than every row,
// and the query's own ORDER BY and LIMIT pick the answer. The first Limit rows of
// a union are among the first Limit of each side, so the answer is the
// same, provided:
//
//   - the table's Predicates are the query's whole filter on it, which both
//     sides apply exactly (validatePredicates' rules);
//   - every row the query returns is one source row: it reads the table
//     once, and doesn't group, join or filter it further;
//   - the lake and delta sides are disjoint. Spans and events are; runs
//     are with a delta of each run's latest row that the lake never holds
//     (the ClickHouse buffer's lake_open_runs), so collapsing runs across
//     the sides changes nothing.
//
// The query must still ORDER BY and LIMIT itself. A delta streamer that
// can't apply every predicate exactly streams its delta unlimited, which
// is slower but still right.
type TopN struct {
	OrderBy []OrderKey
	// Limit counts any OFFSET in: the query's LIMIT plus OFFSET.
	Limit int
}

func validateTopN(t Table, tn TopN, preds []Predicate) error {
	phys, ok := Physical[t]
	if !ok {
		return fmt.Errorf("federate: top-n: unknown table %q", t)
	}
	if tn.Limit <= 0 || len(tn.OrderBy) == 0 {
		return fmt.Errorf("federate: top-n: needs a positive limit and an order")
	}
	for _, k := range tn.OrderBy {
		c, ok := phys.Column(k.Column)
		if !ok {
			return fmt.Errorf("federate: top-n: %s has no column %q", t, k.Column)
		}
		if !aggregatable[c.Type] {
			return fmt.Errorf("federate: top-n: column %q isn't a scalar", k.Column)
		}
		if c.Type == schema.TypeUUID {
			// ClickHouse and DuckDB order UUIDs differently.
			return fmt.Errorf("federate: top-n: can't order by UUID column %q", k.Column)
		}
	}
	return validatePredicates(t, preds, "top-n")
}

// orderSQL renders keys as an ORDER BY list (DuckDB and ClickHouse alike).
func orderSQL(keys []OrderKey, quote func(string) string) string {
	parts := make([]string, len(keys))
	for i, k := range keys {
		dir, nulls := "ASC", "NULLS LAST"
		if k.Desc {
			dir = "DESC"
		}
		if k.NullsFirst {
			nulls = "NULLS FIRST"
		}
		parts[i] = quote(k.Column) + " " + dir + " " + nulls
	}
	return strings.Join(parts, ", ")
}
