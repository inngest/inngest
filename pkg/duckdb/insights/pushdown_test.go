package insights

import (
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/federate"
	"github.com/stretchr/testify/require"
)

func TestExtractPushdown(t *testing.T) {
	eq := func(col string, v any) federate.Predicate {
		return federate.Predicate{Column: col, Op: federate.OpEq, Value: v}
	}
	pred := func(col string, op federate.Op, v any) federate.Predicate {
		return federate.Predicate{Column: col, Op: op, Value: v}
	}

	tests := []struct {
		name string
		sql  string
		want Pushdown
	}{
		{
			name: "equality",
			sql:  `SELECT run_id FROM runs WHERE status = 'Running'`,
			want: Pushdown{"runs": {eq("status", "Running")}},
		},
		{
			name: "literal on the left flips the operator; every AND conjunct is pushed",
			sql:  `SELECT run_id FROM runs WHERE 'Running' = status AND app_id <> 'a' AND 'b' < function_id`,
			want: Pushdown{"runs": {eq("status", "Running"), pred("app_id", federate.OpNeq, "a"), pred("function_id", federate.OpGt, "b")}},
		},
		{
			name: "IN, BETWEEN and negative integers",
			sql:  `SELECT run_id FROM metadata WHERE scope IN ('step', 'run') AND step_index BETWEEN 1 AND 3 AND step_attempt > -1`,
			want: Pushdown{"metadata": {
				pred("scope", federate.OpIn, []any{"step", "run"}),
				pred("step_index", federate.OpGte, int64(1)), pred("step_index", federate.OpLte, int64(3)),
				pred("step_attempt", federate.OpGt, int64(-1)),
			}},
		},
		{
			name: "boolean",
			sql:  `SELECT run_id FROM runs WHERE is_deferred = true`,
			want: Pushdown{"runs": {eq("is_deferred", true)}},
		},
		{
			name: "an OR becomes one predicate tree, beside a sibling AND conjunct",
			sql:  `SELECT run_id FROM runs WHERE (status = 'a' OR status = 'b') AND app_id = 'x'`,
			want: Pushdown{"runs": {
				{Op: federate.OpOr, Args: []federate.Predicate{eq("status", "a"), eq("status", "b")}},
				eq("app_id", "x"),
			}},
		},
		{
			name: "NOT, NOT IN, NOT BETWEEN and LIKE become predicates; NULL tests don't",
			sql: `SELECT run_id FROM metadata WHERE NOT (scope = 'a') AND scope NOT IN ('b') AND step_index NOT BETWEEN 1 AND 2
				AND step_id IS NOT NULL AND span_id LIKE 'x%'`,
			want: Pushdown{"metadata": {
				{Op: federate.OpNot, Args: []federate.Predicate{eq("scope", "a")}},
				{Op: federate.OpNot, Args: []federate.Predicate{{Column: "scope", Op: federate.OpIn, Value: []any{"b"}}}},
				{Op: federate.OpNot, Args: []federate.Predicate{{Op: federate.OpAnd, Args: []federate.Predicate{
					{Column: "step_index", Op: federate.OpGte, Value: int64(1)},
					{Column: "step_index", Op: federate.OpLte, Value: int64(2)},
				}}}},
				{Column: "span_id", Op: federate.OpLike, Value: "x%"},
			}},
		},
		{
			name: "literal type must match the column type",
			sql:  `SELECT run_id FROM metadata WHERE step_index = '1' AND step_attempt = 1.5 AND scope = 1`,
			want: Pushdown{},
		},
		{
			name: "datetime columns are not pushed",
			sql:  `SELECT run_id FROM runs WHERE queued_at > '2026-01-01'`,
			want: Pushdown{},
		},
		{
			name: "JSON key text comparisons are extracted (whether they push is decided physically)",
			sql: `SELECT run_id FROM runs WHERE attributes->>'_inngest.function.slug' = 'fn'
				AND json_extract_string(attributes, 'k') IN ('a', 'b') AND attributes->>'$.nested.path' = 'x'
				AND attributes->>'r' > 'x'`,
			want: Pushdown{"runs": {
				{Column: "attributes", Path: "_inngest.function.slug", Op: federate.OpEq, Value: "fn"},
				{Column: "attributes", Path: "k", Op: federate.OpIn, Value: []any{"a", "b"}},
			}},
		},
		{
			name: "column-to-column comparisons are not pushed",
			sql:  `SELECT run_id FROM runs WHERE app_id = function_id`,
			want: Pushdown{},
		},
		{
			name: "join: qualified columns go to their own table",
			sql: `SELECT r.run_id FROM runs r JOIN events e ON e.id = r.run_id
				WHERE r.status = 'Running' AND e.name = 'app/x'`,
			want: Pushdown{"runs": {eq("status", "Running")}, "events": {eq("name", "app/x")}},
		},
		{
			name: "join: an unqualified column unique to one table resolves to it",
			sql:  `SELECT r.run_id FROM runs r JOIN metadata m ON m.run_id = r.run_id WHERE scope = 'step'`,
			want: Pushdown{"metadata": {eq("scope", "step")}},
		},
		{
			name: "outer join: a null-rejecting filter on the null-supplying side is still safe",
			sql:  `SELECT r.run_id FROM runs r LEFT JOIN events e ON e.id = r.run_id WHERE e.name = 'app/x'`,
			want: Pushdown{"events": {eq("name", "app/x")}},
		},
		{
			name: "a table scanned twice pushes nothing (UNION)",
			sql:  `SELECT run_id FROM runs WHERE status = 'a' UNION ALL SELECT run_id FROM runs WHERE status = 'b'`,
			want: Pushdown{},
		},
		{
			name: "a table scanned twice pushes nothing (self-join)",
			sql:  `SELECT a.run_id FROM runs a JOIN runs b ON a.run_id = b.run_id WHERE a.status = 'x'`,
			want: Pushdown{},
		},
		{
			name: "subquery scans count, and their own WHERE is pushed",
			sql:  `SELECT id FROM events WHERE name = 'n' AND id IN (SELECT run_id FROM runs WHERE status = 'Failed')`,
			want: Pushdown{"events": {eq("name", "n")}, "runs": {eq("status", "Failed")}},
		},
		{
			name: "correlated subquery: only its own table's conjuncts",
			sql: `SELECT run_id FROM runs r
				WHERE EXISTS (SELECT 1 FROM events e WHERE e.id = r.run_id AND e.name = 'n')`,
			want: Pushdown{"events": {eq("name", "n")}},
		},
		{
			name: "CTE: the body's filter is pushed, the outer filter on the CTE is not",
			sql:  `WITH r AS (SELECT run_id, status FROM runs WHERE app_id = 'a') SELECT run_id FROM r WHERE status = 'x'`,
			want: Pushdown{"runs": {eq("app_id", "a")}},
		},
		{
			name: "a CTE shadowing a logical table name blocks that table",
			sql:  `WITH runs AS (SELECT id AS run_id FROM events WHERE name = 'n') SELECT run_id FROM runs WHERE run_id = 'x'`,
			want: Pushdown{"events": {eq("name", "n")}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Transpile(tt.sql, uuid.New(), uuid.New())
			require.NoError(t, err)
			require.Equal(t, tt.want, res.Pushdown)
		})
	}
}

func TestFederatedPushdownIsPhysical(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want map[federate.Table][]federate.Predicate
	}{
		{
			name: "logical columns map to the physical columns their macro passes through",
			sql:  `SELECT run_id FROM runs WHERE app_id = 'my-app' AND function_id IN ('a', 'b') AND run_id = 'r1'`,
			want: map[federate.Table][]federate.Predicate{federate.TableRuns: {
				{Column: "app_name", Op: federate.OpEq, Value: "my-app"},
				{Column: "function_slug", Op: federate.OpIn, Value: []any{"a", "b"}},
				{Column: "run_id", Op: federate.OpEq, Value: "r1"},
			}},
		},
		{
			name: "every status and attribute key filter is forwarded (the runs delta filters after its collapse)",
			sql: `SELECT run_id FROM runs WHERE status = 'Running' AND status = 'Failed' AND attributes->>'_inngest.dynamic.status' = 'Failed'
				AND attributes->>'_inngest.queued_at' = '1' AND attributes->>'_inngest.function.slug' = 'fn'`,
			want: map[federate.Table][]federate.Predicate{federate.TableRuns: {
				{Column: "status", Op: federate.OpEq, Value: "Running"},
				{Column: "status", Op: federate.OpEq, Value: "Failed"},
				{Column: "attributes", Path: "_inngest.dynamic.status", Op: federate.OpEq, Value: "Failed"},
				{Column: "attributes", Path: "_inngest.queued_at", Op: federate.OpEq, Value: "1"},
				{Column: "attributes", Path: "_inngest.function.slug", Op: federate.OpEq, Value: "fn"},
			}},
		},
		{
			name: "extended_trace_spans federates as the spans table",
			sql:  `SELECT span_id FROM extended_trace_spans WHERE trace_id = 't' AND app_id = 'my-app'`,
			want: map[federate.Table][]federate.Predicate{federate.TableSpans: {
				{Column: "trace_id", Op: federate.OpEq, Value: "t"},
				{Column: "app_name", Op: federate.OpEq, Value: "my-app"},
			}},
		},
		{
			name: "events federate; every event column is immutable",
			sql:  `SELECT id FROM events WHERE name = 'app/x' AND id IN ('e1', 'e2')`,
			want: map[federate.Table][]federate.Predicate{federate.TableEvents: {
				{Column: "event_name", Op: federate.OpEq, Value: "app/x"},
				{Column: "event_id", Op: federate.OpIn, Value: []any{"e1", "e2"}},
			}},
		},
		{
			name: "tables that aren't federated contribute nothing",
			sql:  `SELECT run_id FROM metadata WHERE scope = 'step'`,
			want: map[federate.Table][]federate.Predicate{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Transpile(tt.sql, uuid.New(), uuid.New())
			require.NoError(t, err)
			_, preds, _ := res.Federated()
			require.Equal(t, tt.want, preds)
		})
	}
}

// TestPhysicalColumnsExist guards the logical→physical mapping against the
// shared schema.
func TestPhysicalColumnsExist(t *testing.T) {
	for raw, sources := range federatedSources {
		cat := map[bool]catalog{false: productCatalog, true: rawCatalog}[raw]
		for logical, fs := range sources {
			for name, col := range cat.tables[logical].columns {
				if col.physical == "" {
					continue
				}
				_, ok := federate.Physical[fs.table].Column(col.physical)
				require.True(t, ok, "%s.%s maps to unknown physical column %q", logical, name, col.physical)
			}
		}
	}
}
