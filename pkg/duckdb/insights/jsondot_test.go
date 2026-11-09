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
			want: `SELECT id FROM events WHERE json_extract_string(data.user.id, '$') = 'u1'`,
		},
		{
			name: "string on the left",
			in:   "SELECT id FROM events WHERE 'u1' <> data.user.id",
			want: `SELECT id FROM events WHERE 'u1' <> json_extract_string(data.user.id, '$')`,
		},
		{
			name: "table qualified",
			in:   "SELECT e.id FROM events e WHERE e.data.user.id = 'u1'",
			want: `SELECT e.id FROM events AS e WHERE json_extract_string(e.data.user.id, '$') = 'u1'`,
		},
		{
			name: "number comparison reads DOUBLE",
			in:   "SELECT id FROM events WHERE data.n > 5 AND data.n BETWEEN 1 AND 9.5 AND data.n IN (1, 2, NULL) AND 6 <> data.n AND CASE data.n WHEN 1 THEN true END",
			want: "SELECT id FROM events WHERE TRY_CAST(data.n AS DOUBLE) > 5 AND TRY_CAST(data.n AS DOUBLE) BETWEEN 1 AND 9.5 AND TRY_CAST(data.n AS DOUBLE) IN (1, 2, NULL) AND 6 <> TRY_CAST(data.n AS DOUBLE) AND CASE TRY_CAST(data.n AS DOUBLE) WHEN 1 THEN TRUE END",
		},
		{
			name: "numeric column and subquery peers",
			in:   "SELECT run_id FROM steps WHERE output.n = step_attempt AND output.m IN (SELECT step_index FROM steps) AND output.k > (SELECT max(step_index) AS m FROM steps)",
			want: "SELECT run_id FROM steps WHERE TRY_CAST(output.n AS DOUBLE) = step_attempt AND TRY_CAST(output.m AS DOUBLE) IN (SELECT step_index FROM steps) AND TRY_CAST(output.k AS DOUBLE) > (SELECT max(step_index) AS m FROM steps)",
		},
		{
			name: "mixed peers: string wins, other types leave it",
			in:   "SELECT id FROM events WHERE data.n IN (6, 'x') AND data.m IN (6, meta) AND data.b = true",
			want: "SELECT id FROM events WHERE json_extract_string(data.n, '$') IN (6, 'x') AND data.m IN (6, meta) AND data.b = TRUE",
		},
		{
			name: "arithmetic and numeric aggregates",
			in:   "SELECT sum(data.n) AS s, avg(data.n) AS a, max(data.n) AS mx, data.n + 1 AS p, data.n * data.m AS t FROM events GROUP BY data.n, data.m",
			want: "SELECT sum(TRY_CAST(data.n AS DOUBLE)) AS s, avg(TRY_CAST(data.n AS DOUBLE)) AS a, max(data.n) AS mx, TRY_CAST(data.n AS DOUBLE) + 1 AS p, data.n * data.m AS t FROM events GROUP BY data.n, data.m",
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
			want: `SELECT id FROM events WHERE json_extract_string((data.user).id, '$') = 'u1' OR json_extract_string((events.data).name, '$') LIKE 'a%' OR json_extract_string((data).user.id, '$') IS NULL`,
		},
		{
			name: "IN list",
			in:   "SELECT id FROM events WHERE data.user.id NOT IN ('u1', 'u2')",
			want: `SELECT id FROM events WHERE json_extract_string(data.user.id, '$') NOT IN ('u1', 'u2')`,
		},
		{
			name: "BETWEEN strings",
			in:   "SELECT id FROM events WHERE data.user.id BETWEEN 'a' AND 'm'",
			want: `SELECT id FROM events WHERE json_extract_string(data.user.id, '$') BETWEEN 'a' AND 'm'`,
		},
		{
			name: "LIKE and ILIKE",
			in:   "SELECT id FROM events WHERE data.user.id LIKE 'u%' OR data.name ILIKE '%x%'",
			want: `SELECT id FROM events WHERE json_extract_string(data.user.id, '$') LIKE 'u%' OR json_extract_string(data.name, '$') ILIKE '%x%'`,
		},
		{
			name: "IS NULL",
			in:   "SELECT id FROM events WHERE data.user.id IS NULL OR data.a IS NOT NULL OR data.b NOTNULL",
			want: `SELECT id FROM events WHERE json_extract_string(data.user.id, '$') IS NULL OR json_extract_string(data.a, '$') IS NOT NULL OR json_extract_string(data.b, '$') IS NOT NULL`,
		},
		{
			name: "IS DISTINCT FROM",
			in:   "SELECT id FROM events WHERE data.user.id IS DISTINCT FROM 'u1'",
			want: `SELECT id FROM events WHERE json_extract_string(data.user.id, '$') IS DISTINCT FROM 'u1'`,
		},
		{
			name: "string column peer",
			in:   "SELECT r.run_id FROM runs r JOIN events e ON e.data.run_id = r.run_id",
			want: `SELECT r.run_id FROM runs AS r INNER JOIN events AS e ON json_extract_string(e.data.run_id, '$') = r.run_id`,
		},
		{
			name: "IN string subquery",
			in:   "SELECT id FROM events WHERE data.run_id IN (SELECT run_id FROM runs)",
			want: `SELECT id FROM events WHERE json_extract_string(data.run_id, '$') IN (SELECT run_id FROM runs)`,
		},
		{
			name: "scalar string subquery",
			in:   "SELECT id FROM events WHERE data.run_id = (SELECT max(run_id) AS m FROM runs)",
			want: `SELECT id FROM events WHERE json_extract_string(data.run_id, '$') = (SELECT max(run_id) AS m FROM runs)`,
		},
		{
			name: "quantified string subquery",
			in:   "SELECT id FROM events WHERE data.run_id = ANY (SELECT run_id FROM runs)",
			want: `SELECT id FROM events WHERE json_extract_string(data.run_id, '$') = ANY (SELECT run_id FROM runs)`,
		},
		{
			name: "list comprehension variable untouched",
			in:   "SELECT [s.key FOR s IN sessions IF s.key = 'x'] AS k FROM runs",
			want: "SELECT [s.key FOR s IN sessions IF s.key = 'x'] AS k FROM runs",
		},
		{
			name: "CASE operand",
			in:   "SELECT CASE data.kind WHEN 'a' THEN 1 ELSE 0 END AS k FROM events",
			want: `SELECT CASE json_extract_string(data.kind, '$') WHEN 'a' THEN 1 ELSE 0 END AS k FROM events`,
		},
		{
			name: "quoted keys",
			in:   `SELECT id FROM events WHERE data."a.b"."c d"."e""f" = 'x'`,
			want: `SELECT id FROM events WHERE json_extract_string(data."a.b"."c d"."e""f", '$') = 'x'`,
		},
		{
			name: "key with a single quote",
			in:   `SELECT id FROM events WHERE data."it's" = 'x'`,
			want: `SELECT id FROM events WHERE json_extract_string(data."it's", '$') = 'x'`,
		},
		{
			name: "every JSON column",
			in:   "SELECT run_id FROM runs WHERE output.a = 'x' AND attributes.b = 'y' AND inputs.c = 'z' AND metadata.d = 'w'",
			want: `SELECT run_id FROM runs WHERE json_extract_string(output.a, '$') = 'x' AND json_extract_string(attributes.b, '$') = 'y' AND json_extract_string(inputs.c, '$') = 'z' AND json_extract_string(metadata.d, '$') = 'w'`,
		},
		{
			name: "CTE pass-through keeps JSON",
			in:   "WITH e AS (SELECT data FROM events) SELECT data.user.id AS uid FROM e WHERE data.user.id = 'u1'",
			want: `WITH e AS (SELECT data FROM events) SELECT data.user.id AS uid FROM e WHERE json_extract_string(data.user.id, '$') = 'u1'`,
		},
		{
			name: "computed JSON base",
			in:   "SELECT id FROM events WHERE json_extract(data, '$.user').id = 'u1' AND (data -> 'user').id = 'u1' AND (data::JSON).name LIKE 'a%'",
			want: `SELECT id FROM events WHERE json_extract_string(json_extract(data, '$.user').id, '$') = 'u1' AND json_extract_string((data -> 'user').id, '$') = 'u1' AND json_extract_string((data::JSON).name, '$') LIKE 'a%'`,
		},
		{
			name: "CTE computed JSON keeps JSON",
			in:   "WITH e AS (SELECT json_extract(data, '$.user') AS u, to_json(name) AS n FROM events) SELECT u.id AS uid FROM e WHERE u.id = 'u1' AND n.x = 'y'",
			want: `WITH e AS (SELECT json_extract(data, '$.user') AS u, to_json(name) AS n FROM events) SELECT u.id AS uid FROM e WHERE json_extract_string(u.id, '$') = 'u1' AND json_extract_string(n.x, '$') = 'y'`,
		},
		{
			name: "CTE wildcard extraction is not JSON",
			in:   "WITH e AS (SELECT json_extract(data, '$[*].user') AS u FROM events) SELECT 1 AS one FROM e WHERE u.id = 'u1'",
			want: "WITH e AS (SELECT json_extract(data, '$[*].user') AS u FROM events) SELECT 1 AS one FROM e WHERE u.id = 'u1'",
		},
		{
			name: "subquery",
			in:   "SELECT run_id FROM runs WHERE EXISTS (SELECT 1 FROM events WHERE data.user.id = 'u1')",
			want: `SELECT run_id FROM runs WHERE EXISTS (SELECT 1 FROM events WHERE json_extract_string(data.user.id, '$') = 'u1')`,
		},
		{
			name: "cast to text",
			in:   "SELECT data.n::VARCHAR AS n, CAST(data.user.id AS TEXT) AS uid, TRY_CAST(data.obj AS STRING) AS o FROM events WHERE data.user.id::VARCHAR = 'u1' GROUP BY CAST(data.user.id AS VARCHAR)",
			want: "SELECT json_extract_string(data.n, '$') AS n, json_extract_string(data.user.id, '$') AS uid, json_extract_string(data.obj, '$') AS o FROM events WHERE json_extract_string(data.user.id, '$') = 'u1' GROUP BY json_extract_string(data.user.id, '$')",
		},
		{
			name: "cast to non-text untouched",
			in:   "SELECT data.n::INTEGER AS n, CAST(data.f AS DOUBLE) AS f, data.ok::BOOLEAN AS ok, data.user::JSON AS u, data.tags::VARCHAR[] AS t, data::VARCHAR AS d FROM events",
			want: "SELECT data.n::INTEGER AS n, data.f::DOUBLE AS f, data.ok::BOOLEAN AS ok, data.user::JSON AS u, data.tags::VARCHAR[] AS t, data::VARCHAR AS d FROM events",
		},
		{
			name: "string functions",
			in:   "SELECT lower(data.user.id) AS l, LENGTH(data.user.id) AS n, data.user.id || '-' || data.name AS c, concat_ws(',', data.a, name) AS w FROM events WHERE starts_with(data.user.id, 'u') AND regexp_matches(data.user.id, '^u') AND left(data.user.id, 1) = 'u'",
			want: "SELECT lower(json_extract_string(data.user.id, '$')) AS l, LENGTH(json_extract_string(data.user.id, '$')) AS n, json_extract_string(data.user.id, '$') || '-' || json_extract_string(data.name, '$') AS c, concat_ws(',', json_extract_string(data.a, '$'), name) AS w FROM events WHERE starts_with(json_extract_string(data.user.id, '$'), 'u') AND regexp_matches(json_extract_string(data.user.id, '$'), '^u') AND left(json_extract_string(data.user.id, '$'), 1) = 'u'",
		},
		{
			name: "other functions and nested args untouched",
			in:   "SELECT coalesce(data.user.id, data.x) AS c, json_extract(data.user, '$.id') AS j, lower(coalesce(data.a, data.b)) AS l FROM events",
			want: "SELECT COALESCE(data.user.id, data.x) AS c, json_extract(data.user, '$.id') AS j, lower(COALESCE(data.a, data.b)) AS l FROM events",
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
