package tracing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// newTestDuckLake opens a real duckdb subprocess the way the dev server's
// persisted dual-write does: a DuckLake catalog over the quack transport.
// The pool is capped at 1 conn, so a test can see whether a failed batch
// left that conn unusable (IE stuck in an aborted transaction).
func newTestDuckLake(t *testing.T) *sql.DB {
	t.Helper()
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
	dir := t.TempDir()
	addr := driver.EphemeralQuackAddr
	db, err := driver.Open(t.Context(), driver.Options{
		BinaryPath: binPath,
		StateDir:   dir,
		DBFile:     filepath.Join(dir, "main.duckdb"),
		DuckLake: &driver.DuckLakeOptions{
			CatalogPath: filepath.Join(dir, "catalog.duckdb"),
			DataPath:    filepath.Join(dir, "data"),
		},
		QuackAddr:  &addr,
		QuackConns: 2,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, duckdb.Migrate(t.Context(), db, duckdb.ModeRawLake))
	db.SetMaxOpenConns(1)
	return db
}

// legacyMaterializeRuns is the runs materialization InsertSpans used to do,
// kept verbatim (modulo the projection, which is shared) to compare against:
// insert the spans, then project runs back out of the stored
// run_trace_spans by span_id.
func legacyMaterializeRuns(ctx context.Context, db *sql.DB, spans []Span) error {
	var spanIDs []any
	for _, s := range spans {
		if slices.Contains(runsMaterializingSpanNames, s.Name) {
			spanIDs = append(spanIDs, s.SpanID)
		}
	}
	if len(spanIDs) == 0 {
		return nil
	}
	_, err := db.ExecContext(ctx, legacyRunsSQL(len(spanIDs)), spanIDs...)
	return err
}

func legacyRunsSQL(n int) string {
	q := runsInsertSQL("inngest.run_trace_spans")
	return strings.Replace(q, "\nWHERE ", "\nWHERE span_id IN ("+strings.TrimSuffix(strings.Repeat("?, ", n), ", ")+")\nAND ", 1)
}

func legacyInsertSpans(ctx context.Context, db *sql.DB, spans []Span) error {
	rows := make([][]any, len(spans))
	for i, s := range spans {
		rows[i] = []any{
			s.AccountID, s.EnvID, s.AppID, s.AppName, s.FunctionID, s.FunctionSlug,
			s.RunID, s.RunQueuedAt, s.TraceID, s.SpanID, nullableString(s.ParentSpanID), s.Name,
			s.StartTime, s.EndTime, s.Attributes, nullableJSON(s.Output), nullableJSON(s.Error), nullableJSON(s.Input),
		}
	}
	if err := NewDuckDBInserter(db).insert(ctx, "inngest.run_trace_spans", spanColumns, rows); err != nil {
		return err
	}
	return legacyMaterializeRuns(ctx, db, spans)
}

func runSpan(runID ulid.ULID, spanID, name string, attrs map[string]any) Span {
	byt, _ := json.Marshal(attrs)
	now := time.UnixMilli(1_760_000_000_000).UTC()
	return Span{
		AccountID: batchTestAccountID.String(), EnvID: batchTestEnvID.String(),
		AppID: batchTestAppID.String(), AppName: "test-app",
		FunctionID: batchTestFunctionID.String(), FunctionSlug: "test-fn",
		RunID: runID.String(), RunQueuedAt: ulid.Time(runID.Time()).UTC(),
		TraceID: "trace-" + runID.String(), SpanID: spanID, Name: name,
		StartTime: now, EndTime: now.Add(time.Second),
		Attributes: byt,
		// runs.inputs is NOT NULL, and the listener always sets it on run
		// spans.
		Input: json.RawMessage(`[]`),
	}
}

// spanBatch is a batch covering every column the runs projection derives:
// queued/started/finished run spans (w/ and w/o the optional timing, event,
// session and defer attributes, input/output), plus a step span that must
// not produce a runs row.
func spanBatch(runA, runB ulid.ULID) []Span {
	ms := func(sec int64) int64 { return 1_760_000_000_000 + sec*1000 }

	queued := runSpan(runA, "span-a-run", "executor.run.queued", map[string]any{
		"_inngest.dynamic.status": "Queued",
		"_inngest.scheduled_at":   ms(1),
		"_inngest.event.ids":      []string{"01EVENTA", "01EVENTB"},
		"_inngest.event.sessions": []map[string]string{{"key": "user", "id": "u1"}},
	})
	queued.Input = json.RawMessage(`[{"name":"evt","data":{"n":1}}]`)

	finished := runSpan(runA, "span-a-run", "executor.run", map[string]any{
		"_inngest.dynamic.status":         "Completed",
		"_inngest.scheduled_at":           ms(1),
		"_inngest.started_at":             ms(2),
		"_inngest.ended_at":               ms(5),
		"_inngest.defer.parent_fn_slug":   "parent-fn",
		"_inngest.defer.parent_run_ids":   []string{"01PARENT"},
		"_inngest.function.output_extra": map[string]any{"nested": true},
	})
	finished.Output = json.RawMessage(`{"data":"ok"}`)

	// No scheduled_at (old run state): falls back to queued_at.
	started := runSpan(runB, "span-b-run", "executor.run.started", map[string]any{
		"_inngest.dynamic.status": "Running",
		"_inngest.started_at":     ms(3),
	})

	step := runSpan(runA, "span-a-step", "executor.step", map[string]any{"_inngest.dynamic.status": "Completed"})
	step.ParentSpanID = "span-a-run"
	step.Output = json.RawMessage(`{"step":"out"}`)

	return []Span{queued, step, finished, started}
}

// normalized renders rows transport-independently: jsonlines and quack
// decode the same values into different Go types (float64 vs int64, a UTC
// vs a nil *time.Location), so compare them as JSON w/ times in UTC.
func normalized(t *testing.T, rows []map[string]any) []string {
	t.Helper()
	out := make([]string, len(rows))
	for i, row := range rows {
		norm := make(map[string]any, len(row))
		for k, v := range row {
			if tv, ok := v.(time.Time); ok {
				v = tv.UTC().Format(time.RFC3339Nano)
			}
			norm[k] = v
		}
		byt, err := json.Marshal(norm)
		require.NoError(t, err)
		out[i] = string(byt)
	}
	return out
}

const runsCompareSQL = `SELECT * EXCLUDE (bucket_at) FROM inngest.runs WHERE run_id IN (?, ?) ORDER BY run_id, status`

// TestInsertSpansRunsMatchLegacyMaterialization proves building runs rows
// from the staged batch produces exactly what reading them back out of
// run_trace_spans did, on both transports.
func TestInsertSpansRunsMatchLegacyMaterialization(t *testing.T) {
	runA, runB := ulid.Make(), ulid.Make()
	batch := spanBatch(runA, runB)

	legacyDB, cleanup := newTestDuckDB(t)
	defer cleanup()
	require.NoError(t, legacyInsertSpans(t.Context(), legacyDB, batch))
	want := selectRows(t, legacyDB, runsCompareSQL, runA.String(), runB.String())
	require.Len(t, want, 3, "3 run spans -> 3 runs rows, the step span doesn't count")
	wantSpans := selectRows(t, legacyDB, `SELECT * EXCLUDE (bucket_at) FROM inngest.run_trace_spans ORDER BY span_id, name`)

	memDB, cleanup := newTestDuckDB(t)
	defer cleanup()

	for name, db := range map[string]*sql.DB{"jsonlines memory": memDB, "quack ducklake": newTestDuckLake(t)} {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, NewDuckDBInserter(db).InsertSpans(t.Context(), batch))
			require.Equal(t, normalized(t, want), normalized(t, selectRows(t, db, runsCompareSQL, runA.String(), runB.String())))
			require.Equal(t, normalized(t, wantSpans), normalized(t, selectRows(t, db, `SELECT * EXCLUDE (bucket_at) FROM inngest.run_trace_spans ORDER BY span_id, name`)))

			// A runs row's bucket_at is its span's, as when it was read
			// back from the stored span.
			var n int
			require.NoError(t, db.QueryRowContext(t.Context(), `SELECT count(DISTINCT bucket_at) FROM (
  SELECT bucket_at FROM inngest.runs UNION ALL SELECT bucket_at FROM inngest.run_trace_spans)`).Scan(&n))
			require.Equal(t, 1, n)
		})
	}
}

// TestInsertSpansIgnoresSpansAlreadyInLake pins that a batch's runs rows
// come from the batch alone. The legacy span_id IN (...) read also picked up
// every earlier emission of the same span (run spans are seeded by run ID,
// so every queued/started/finished emission of a run shares one span_id) and
// re-inserted it as another runs row.
func TestInsertSpansIgnoresSpansAlreadyInLake(t *testing.T) {
	db := newTestDuckLake(t)
	ins := NewDuckDBInserter(db)

	runA, runB := ulid.Make(), ulid.Make()
	batch := spanBatch(runA, runB)

	// Prior flushes: an earlier emission of runA's run span, and an
	// unrelated run.
	prior := runSpan(runA, "span-a-run", "executor.run.started", map[string]any{"_inngest.dynamic.status": "Running"})
	unrelated := runSpan(ulid.Make(), "span-other", "executor.run", map[string]any{"_inngest.dynamic.status": "Completed"})
	require.NoError(t, ins.InsertSpans(t.Context(), []Span{prior, unrelated}))

	require.NoError(t, ins.InsertSpans(t.Context(), batch))

	// What the legacy read (run after the spans landed) would have pulled
	// for this batch: its 3 run spans plus the prior emission sharing
	// span-a-run, IE one extra runs row.
	var legacyRead int
	require.NoError(t, db.QueryRowContext(t.Context(),
		`SELECT count(*) FROM inngest.run_trace_spans WHERE span_id IN ('span-a-run', 'span-b-run') AND name IN ('executor.run', 'executor.run.queued', 'executor.run.started')`,
	).Scan(&legacyRead))
	require.Equal(t, 4, legacyRead)

	// runA: the prior Running row + the batch's Queued/Completed rows, w/o a
	// second copy of Running.
	got := selectRows(t, db, `SELECT status FROM inngest.runs WHERE run_id = ? ORDER BY status`, runA.String())
	require.Equal(t, []map[string]any{{"status": "Completed"}, {"status": "Queued"}, {"status": "Running"}}, got)

	var runs int
	require.NoError(t, db.QueryRowContext(t.Context(), "SELECT count(*) FROM inngest.runs").Scan(&runs))
	require.Equal(t, 5, runs, "2 prior + 3 from the batch")

	// And the batch's own rows match a lake that never held anything else.
	freshDB, cleanup := newTestDuckDB(t)
	defer cleanup()
	require.NoError(t, NewDuckDBInserter(freshDB).InsertSpans(t.Context(), batch))
	require.Equal(t,
		normalized(t, selectRows(t, freshDB, runsCompareSQL, runA.String(), runB.String())),
		normalized(t, selectRows(t, db, `SELECT * EXCLUDE (bucket_at) FROM inngest.runs WHERE run_id IN (?, ?) AND NOT (run_id = ? AND status = 'Running') ORDER BY run_id, status`, runA.String(), runB.String(), runA.String())),
	)
}

// The staging CTAS clones run_trace_spans' column types w/o reading it.
func TestInsertSpansStagingDoesNotScanLake(t *testing.T) {
	db := newTestDuckLake(t)
	rows := selectRows(t, db, "EXPLAIN SELECT * FROM inngest.run_trace_spans LIMIT 0")
	var plan strings.Builder
	for _, r := range rows {
		fmt.Fprintln(&plan, r["explain_value"])
	}
	require.Contains(t, plan.String(), "Empty Result")
	require.NotContains(t, strings.ToUpper(plan.String()), "SCAN", plan.String())
}

// TestInsertSpansIsAtomicOnDuckLake fails a batch's runs insert (a run span
// w/o a status violates runs.status NOT NULL) and checks the batch's spans
// were rolled back w/ it, and that the conn isn't left in an aborted
// transaction.
func TestInsertSpansIsAtomicOnDuckLake(t *testing.T) {
	db := newTestDuckLake(t)
	ins := NewDuckDBInserter(db)

	runA := ulid.Make()
	bad := []Span{
		runSpan(runA, "span-step", "executor.step", map[string]any{}),
		runSpan(runA, "span-run", "executor.run", map[string]any{}), // no status
	}
	err := ins.InsertSpans(t.Context(), bad)
	require.ErrorIs(t, err, driver.ErrStatementFailed)
	require.Contains(t, err.Error(), "NOT NULL")
	require.False(t, retryableFlushError(err))

	var spans, runs int
	require.NoError(t, db.QueryRowContext(t.Context(), "SELECT count(*) FROM inngest.run_trace_spans").Scan(&spans))
	require.NoError(t, db.QueryRowContext(t.Context(), "SELECT count(*) FROM inngest.runs").Scan(&runs))
	require.Zero(t, spans, "spans must roll back w/ the failed runs insert")
	require.Zero(t, runs)

	// The (only) conn is usable again.
	runB, runC := ulid.Make(), ulid.Make()
	require.NoError(t, ins.InsertSpans(t.Context(), spanBatch(runB, runC)))
	require.NoError(t, db.QueryRowContext(t.Context(), "SELECT count(*) FROM inngest.runs").Scan(&runs))
	require.Equal(t, 3, runs)
}

// jsonlines can't run the batch in a transaction (see
// driver.SupportsTransactions), but a failed batch must still return rather
// than hang.
func TestInsertSpansFailureDoesNotHangJSONLines(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	runA := ulid.Make()
	err := NewDuckDBInserter(db).InsertSpans(ctx, []Span{runSpan(runA, "span-run", "executor.run", map[string]any{})})
	require.ErrorIs(t, err, driver.ErrStatementFailed)
	require.NoError(t, ctx.Err())
}
