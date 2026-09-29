package insights_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/insights"
	"github.com/stretchr/testify/require"
)

// TestCheckRenderedSQLAcceptsEveryGoldenFixture pins that the post-render
// guard never rejects anything Transpile itself produces: every golden
// fixture that transpiles must pass it.
func TestCheckRenderedSQLAcceptsEveryGoldenFixture(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	files, err := filepath.Glob(filepath.Join("testdata", "queries", "*.sql"))
	require.NoError(t, err)
	require.NotEmpty(t, files)

	checked := 0
	for _, f := range files {
		src, err := os.ReadFile(f)
		require.NoError(t, err)
		tr, err := insights.Transpile(string(src), uuid.New(), uuid.New())
		if err != nil {
			continue // a rejection fixture
		}
		require.NoError(t, insights.CheckRenderedSQL(context.Background(), db, tr.SQL), f)
		checked++
	}
	require.Greater(t, checked, 30)
}

func TestCheckRenderedSQLAcceptsTranspiledShapes(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	for _, q := range []string{
		"SELECT count(*) FROM runs",
		"SELECT count(*) OVER (PARTITION BY app_id), sum(1) FILTER (WHERE true) FROM runs",
		"SELECT [1, 2], {'a': 1}, queued_at + INTERVAL 1 DAY, [x FOR x IN [1, 2] IF x > 1] FROM runs",
		"SELECT status SIMILAR TO 'C.*', status ILIKE 'c%', status GLOB 'C*', status NOT LIKE 'x' FROM runs",
		"WITH c AS (SELECT run_id FROM runs) SELECT * FROM c WHERE run_id IN (SELECT run_id FROM c)",
		"SELECT r.run_id FROM runs r JOIN events e ON e.id = r.run_id UNION ALL SELECT run_id FROM steps",
		"SELECT (SELECT max(queued_at) FROM runs) AS m, strftime(now(), '%Y') AS y",
	} {
		tr, err := insights.Transpile(q, uuid.New(), uuid.New())
		require.NoError(t, err, q)
		require.NoError(t, insights.CheckRenderedSQL(context.Background(), db, tr.SQL), q)
	}
}

// TestCheckRenderedSQLRejectsWhatTranspileNeverProduces covers hand-written
// SQL standing in for a renderer gap: each would reach DuckDB if it slipped
// past validate.
func TestCheckRenderedSQLRejectsWhatTranspileNeverProduces(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	for q, want := range map[string]string{
		"SELECT getenv('HOME')":                                     `function "getenv"`,
		"SELECT content FROM read_text('/etc/hosts')":               `function "read_text"`,
		"SELECT * FROM query('SELECT * FROM inngest.runs')":         `function "query"`,
		"SELECT current_setting('enable_external_access')":          `function "current_setting"`,
		"SELECT * FROM inngest.runs":                                `table "runs"`,
		"SELECT * FROM runs":                                        `table "runs"`,
		"SELECT * FROM inngest.insights_runs(?, ?) r, inngest.runs": `table "runs"`,
		"SELECT inngest.insights_runs(?, ?)":                        `function "inngest.insights_runs"`,
		"SELECT main.abs(1)":                                        `function "main.abs"`,
		"SELECT 1; SELECT 2":                                        "2 statements",
		"SELECT * FROM (SELECT * FROM glob('/etc/*'))":              `function "glob"`,
		"SELECT (SELECT getenv('USER'))":                            `function "getenv"`,
		"SELECT * FROM range(10)":                                   `table function "range"`,
		"SUMMARIZE inngest.runs":                                    "",
	} {
		err := insights.CheckRenderedSQL(context.Background(), db, q)
		require.Error(t, err, q)
		if want != "" {
			require.ErrorContains(t, err, want, q)
		}
	}
}

// TestExecuteSandboxBlocksFileAndEnvAccess proves that even with the
// post-render guard off — i.e. a query that got past every check this
// package has — DuckDB itself refuses file and environment access on the
// sandboxed handle (driver.Options.RestrictExternalAccess, as
// devserver's setupDualWrite opens it).
func TestExecuteSandboxBlocksFileAndEnvAccess(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	secret := filepath.Join(t.TempDir(), "secret.txt")
	require.NoError(t, os.WriteFile(secret, []byte("secret"), 0o600))

	queries := []string{
		"SELECT getenv('HOME') AS v;",
		"SELECT content FROM read_text('" + secret + "');",
		"SELECT * FROM read_csv('" + secret + "');",
		"SELECT * FROM glob('" + filepath.Dir(secret) + "/*');",
	}

	// With the guard on, Execute rejects each before DuckDB runs it.
	for _, q := range queries {
		_, err := insights.Execute(context.Background(), db, &insights.TranspileResult{SQL: q})
		var eerr *insights.ExecutionError
		require.ErrorAs(t, err, &eerr, q)
		require.Contains(t, err.Error(), "not allowed", q)
	}

	// With it off, DuckDB's own sandbox still refuses.
	insights.DisableRenderedSQLGuard(t)
	for _, q := range queries {
		_, err := insights.Execute(context.Background(), db, &insights.TranspileResult{SQL: q})
		require.Error(t, err, q)
		require.True(t, strings.Contains(err.Error(), "disabled through configuration") ||
			strings.Contains(err.Error(), "disabled by configuration"), "%s: %v", q, err)
	}
}
