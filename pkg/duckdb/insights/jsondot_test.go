package insights

import (
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/parser"
	"github.com/stretchr/testify/require"
)

func mustRewriteJSONDotComparison(t *testing.T, sql string) string {
	t.Helper()
	stmt := mustParse(t, sql)
	_, ctes, _, err := validate(stmt, productCatalog)
	require.NoError(t, err)
	_, err = rewriteJSONDotComparison(stmt, ctes, nil)
	require.NoError(t, err)
	return parser.String(stmt)
}

func TestRewriteJSONDotComparison(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{
			name: "string equality",
			in:   "SELECT id FROM events WHERE data.user.id = 'u1'",
			want: `SELECT id FROM events WHERE data ->> '$."user"."id"' = 'u1'`,
		},
		{
			name: "string on the left",
			in:   "SELECT id FROM events WHERE 'u1' <> data.user.id",
			want: `SELECT id FROM events WHERE 'u1' <> data ->> '$."user"."id"'`,
		},
		{
			name: "table qualified",
			in:   "SELECT e.id FROM events e WHERE e.data.user.id = 'u1'",
			want: `SELECT e.id FROM events AS e WHERE e.data ->> '$."user"."id"' = 'u1'`,
		},
		{
			name: "number comparison stays JSON",
			in:   "SELECT id FROM events WHERE data.n > 5 AND data.n BETWEEN 1 AND 9 AND data.n IN (1, 2)",
			want: "SELECT id FROM events WHERE data.n > 5 AND data.n BETWEEN 1 AND 9 AND data.n IN (1, 2)",
		},
		{
			name: "boolean stays JSON",
			in:   "SELECT id FROM events WHERE data.ok = true",
			want: "SELECT id FROM events WHERE data.ok = TRUE",
		},
		{
			name: "projection group by order by stay JSON",
			in:   "SELECT data.user.id AS uid, count(*) AS n FROM events GROUP BY data.user.id ORDER BY data.user.id",
			want: "SELECT data.user.id AS uid, count(*) AS n FROM events GROUP BY data.user.id ORDER BY data.user.id",
		},
		{
			name: "JSON vs JSON stays JSON",
			in:   "SELECT id FROM events WHERE data.a = data.b",
			want: "SELECT id FROM events WHERE data.a = data.b",
		},
		{
			name: "parenthesized dot access",
			in:   "SELECT id FROM events WHERE (data.user).id = 'u1' OR (events.data).name LIKE 'a%' OR ((data).user).id IS NULL",
			want: `SELECT id FROM events WHERE data ->> '$."user"."id"' = 'u1' OR events.data ->> '$."name"' LIKE 'a%' OR data ->> '$."user"."id"' IS NULL`,
		},
		{
			name: "IN list",
			in:   "SELECT id FROM events WHERE data.user.id NOT IN ('u1', 'u2')",
			want: `SELECT id FROM events WHERE data ->> '$."user"."id"' NOT IN ('u1', 'u2')`,
		},
		{
			name: "BETWEEN strings",
			in:   "SELECT id FROM events WHERE data.user.id BETWEEN 'a' AND 'm'",
			want: `SELECT id FROM events WHERE data ->> '$."user"."id"' BETWEEN 'a' AND 'm'`,
		},
		{
			name: "LIKE and ILIKE",
			in:   "SELECT id FROM events WHERE data.user.id LIKE 'u%' OR data.name ILIKE '%x%'",
			want: `SELECT id FROM events WHERE data ->> '$."user"."id"' LIKE 'u%' OR data ->> '$."name"' ILIKE '%x%'`,
		},
		{
			name: "IS NULL",
			in:   "SELECT id FROM events WHERE data.user.id IS NULL OR data.a IS NOT NULL OR data.b NOTNULL",
			want: `SELECT id FROM events WHERE data ->> '$."user"."id"' IS NULL OR data ->> '$."a"' IS NOT NULL OR data ->> '$."b"' IS NOT NULL`,
		},
		{
			name: "IS DISTINCT FROM",
			in:   "SELECT id FROM events WHERE data.user.id IS DISTINCT FROM 'u1'",
			want: `SELECT id FROM events WHERE data ->> '$."user"."id"' IS DISTINCT FROM 'u1'`,
		},
		{
			name: "string column peer",
			in:   "SELECT r.run_id FROM runs r JOIN events e ON e.data.run_id = r.run_id",
			want: `SELECT r.run_id FROM runs AS r INNER JOIN events AS e ON e.data ->> '$."run_id"' = r.run_id`,
		},
		{
			name: "IN string subquery",
			in:   "SELECT id FROM events WHERE data.run_id IN (SELECT run_id FROM runs)",
			want: `SELECT id FROM events WHERE data ->> '$."run_id"' IN (SELECT run_id FROM runs)`,
		},
		{
			name: "scalar string subquery",
			in:   "SELECT id FROM events WHERE data.run_id = (SELECT max(run_id) AS m FROM runs)",
			want: `SELECT id FROM events WHERE data ->> '$."run_id"' = (SELECT max(run_id) AS m FROM runs)`,
		},
		{
			name: "quantified string subquery",
			in:   "SELECT id FROM events WHERE data.run_id = ANY (SELECT run_id FROM runs)",
			want: `SELECT id FROM events WHERE data ->> '$."run_id"' = ANY (SELECT run_id FROM runs)`,
		},
		{
			name: "list comprehension variable untouched",
			in:   "SELECT [s.key FOR s IN sessions IF s.key = 'x'] AS k FROM runs",
			want: "SELECT [s.key FOR s IN sessions IF s.key = 'x'] AS k FROM runs",
		},
		{
			name: "CASE operand",
			in:   "SELECT CASE data.kind WHEN 'a' THEN 1 ELSE 0 END AS k FROM events",
			want: `SELECT CASE data ->> '$."kind"' WHEN 'a' THEN 1 ELSE 0 END AS k FROM events`,
		},
		{
			name: "quoted keys",
			in:   `SELECT id FROM events WHERE data."a.b"."c d"."e""f" = 'x'`,
			want: `SELECT id FROM events WHERE data ->> '$."a.b"."c d"."e\"f"' = 'x'`,
		},
		{
			name: "key with a single quote",
			in:   `SELECT id FROM events WHERE data."it's" = 'x'`,
			want: `SELECT id FROM events WHERE data ->> '$."it''s"' = 'x'`,
		},
		{
			name: "every JSON column",
			in:   "SELECT run_id FROM runs WHERE output.a = 'x' AND attributes.b = 'y' AND inputs.c = 'z' AND metadata.d = 'w'",
			want: `SELECT run_id FROM runs WHERE output ->> '$."a"' = 'x' AND attributes ->> '$."b"' = 'y' AND inputs ->> '$."c"' = 'z' AND metadata ->> '$."d"' = 'w'`,
		},
		{
			name: "CTE pass-through keeps JSON",
			in:   "WITH e AS (SELECT data FROM events) SELECT data.user.id AS uid FROM e WHERE data.user.id = 'u1'",
			want: `WITH e AS (SELECT data FROM events) SELECT data.user.id AS uid FROM e WHERE data ->> '$."user"."id"' = 'u1'`,
		},
		{
			name: "subquery",
			in:   "SELECT run_id FROM runs WHERE EXISTS (SELECT 1 FROM events WHERE data.user.id = 'u1')",
			want: `SELECT run_id FROM runs WHERE EXISTS (SELECT 1 FROM events WHERE data ->> '$."user"."id"' = 'u1')`,
		},
		{
			name: "array of structs untouched",
			in:   "SELECT run_id FROM runs WHERE sessions.key = 'x'",
			want: "SELECT run_id FROM runs WHERE sessions.key = 'x'",
		},
		{
			name: "bare JSON column untouched",
			in:   "SELECT id FROM events WHERE data = '{}'",
			want: "SELECT id FROM events WHERE data = '{}'",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, mustRewriteJSONDotComparison(t, tc.in))
		})
	}
}
