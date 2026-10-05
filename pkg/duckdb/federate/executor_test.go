package federate

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// fakeStreamer serves canned deltas and applies pushed predicates the way a
// real streamer would, so tests can check the views stay correct under
// pushdown. It only understands status predicates, which is enough to
// exercise the mutable-column case.
type fakeStreamer struct {
	// rows hold each delta row's values by column name; columns a row
	// doesn't set are NULL.
	rows map[Table][]map[string]any
	reqs []DeltaRequest
}

func (f *fakeStreamer) Stream(_ context.Context, req DeltaRequest) (BatchReader, error) {
	f.reqs = append(f.reqs, req)
	var kept []map[string]any
	for _, row := range f.rows[req.Table] {
		if !matches(row["status"], req.Predicates) {
			continue
		}
		kept = append(kept, row)
	}
	cols := make([][]any, len(req.Columns))
	for _, row := range kept {
		for i, c := range req.Columns {
			cols[i] = append(cols[i], row[c.Name])
		}
	}
	for i := range cols {
		if cols[i] == nil {
			cols[i] = []any{}
		}
	}
	return NewSliceReader(req.Columns, &Batch{Schema: req.Columns, Columns: cols}), nil
}

func matches(status any, preds []Predicate) bool {
	for _, p := range preds {
		if p.Column != "status" {
			continue
		}
		switch p.Op {
		case OpEq:
			if status != p.Value {
				return false
			}
		case OpIn:
			if !slices.Contains(p.Value.([]any), status) {
				return false
			}
		}
	}
	return true
}

type fixture struct {
	db                    *sql.DB
	account, env, app, fn uuid.UUID
	from, to, base        time.Time
	streamer              *fakeStreamer
	exec                  *Executor
}

func newFixture(t *testing.T) *fixture {
	db := newTestDB(t)
	base := time.Now().UTC().Truncate(time.Millisecond).Add(-2 * time.Hour)
	f := &fixture{
		db: db, account: uuid.New(), env: uuid.New(), app: uuid.New(), fn: uuid.New(),
		from: base.Add(-time.Hour), to: base.Add(time.Hour), base: base,
		streamer: &fakeStreamer{rows: map[Table][]map[string]any{}},
	}
	f.exec = &Executor{DB: db, Delta: f.streamer, Ingester: QuackIngester{}}
	return f
}

// forModes runs fn once per delta ingestion mode: TEMP tables, and streamed
// MATERIALIZED CTEs.
func forModes(t *testing.T, fn func(t *testing.T, f *fixture)) {
	for _, streaming := range []bool{false, true} {
		name := "temp"
		if streaming {
			name = "streaming"
		}
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			f.exec.Streaming = streaming
			fn(t, f)
		})
	}
}

// startedAt is when a run row says the run started: never for a queued row,
// as the listener writes them.
func startedAt(status string, queuedAt time.Time) any {
	if status == "Queued" {
		return nil
	}
	return queuedAt
}

// watermark is the W the fixture's queries read with: lake rows sit at or
// below it (lakeBucketAt), and the fake delta ignores it.
func (f *fixture) watermark() time.Time { return f.base }

// lakeBucketAt is when the fixture's lake rows reached the buffer: before W,
// so the lake side of every federated read includes them.
func (f *fixture) lakeBucketAt() time.Time { return f.base.Add(-time.Minute) }

func (f *fixture) lakeRun(t *testing.T, account uuid.UUID, runID, status string, queuedAt time.Time, ended *time.Time) {
	t.Helper()
	_, err := f.db.ExecContext(context.Background(), `INSERT INTO inngest.runs
		(account_id, env_id, run_id, queued_at, scheduled_at, started_at, ended_at, app_id, app_name, function_id, function_slug,
		 status, attributes, inputs, output, bucket_at)
		VALUES (?::UUID, ?::UUID, ?, ?, ?, ?, ?, ?::UUID, 'app', ?::UUID, 'fn', ?, '{}'::JSON::VARIANT, '[]'::JSON::VARIANT, NULL, ?)`,
		account.String(), f.env.String(), runID, queuedAt, queuedAt, startedAt(status, queuedAt), ended, f.app.String(), f.fn.String(), status, f.lakeBucketAt())
	require.NoError(t, err)
}

func (f *fixture) deltaRun(runID, status string, queuedAt time.Time, ended *time.Time, output string) {
	var end, out any
	if ended != nil {
		end = *ended
	}
	if output != "" {
		out = output
	}
	f.streamer.rows[TableRuns] = append(f.streamer.rows[TableRuns], map[string]any{
		"account_id": f.account, "env_id": f.env, "run_id": runID, "app_id": f.app, "app_name": "app",
		"function_id": f.fn, "function_slug": "fn", "queued_at": queuedAt, "started_at": startedAt(status, queuedAt), "ended_at": end,
		"status": status, "attributes": `{}`, "inputs": `[]`, "output": out, "event_ids": []string{"evt-1"},
	})
}

func (f *fixture) query(t *testing.T, tables []Table, preds map[Table][]Predicate, sqlText string, args ...any) map[string]string {
	t.Helper()
	rows, err := f.exec.Query(context.Background(), Query{
		AccountID: f.account, EnvID: f.env, From: f.from, To: f.to,
		Lake: LakeSource{Catalog: "inngest"}, Watermark: f.watermark(), Tables: tables, Predicates: preds,
		RowCap: 1000, SQL: sqlText, Args: args,
	})
	require.NoError(t, err)
	defer rows.Close()
	got := map[string]string{}
	for rows.Next() {
		var k, v string
		require.NoError(t, rows.Scan(&k, &v))
		_, dup := got[k]
		require.False(t, dup, "duplicate row for %s", k)
		got[k] = v
	}
	require.NoError(t, rows.Err())
	return got
}

func TestExecutorRunsUnion(t *testing.T) {
	forModes(t, func(t *testing.T, f *fixture) {
		q := f.base
		done := q.Add(time.Minute)

		// A: queued in the lake, finished in the delta (straddles the watermark).
		f.lakeRun(t, f.account, "run-A", "Queued", q, nil)
		f.deltaRun("run-A", "Completed", q, &done, `{"ok":true}`)
		// B: in flight, only in the lake.
		f.lakeRun(t, f.account, "run-B", "Running", q, nil)
		// C: started in the lake, still running in the delta (latest non-final wins).
		f.lakeRun(t, f.account, "run-C", "Queued", q, nil)
		f.deltaRun("run-C", "Running", q, nil, "")
		// D: finished entirely in the lake.
		f.lakeRun(t, f.account, "run-D", "Failed", q, &done)
		// Another tenant's run, and a run outside the time bound: never visible.
		f.lakeRun(t, uuid.New(), "run-other", "Running", q, nil)
		f.lakeRun(t, f.account, "run-old", "Running", f.from.Add(-time.Hour), nil)

		got := f.query(t, []Table{TableRuns}, nil, `SELECT run_id, status FROM runs`)
		require.Equal(t, map[string]string{
			"run-A": "Completed",
			"run-B": "Running",
			"run-C": "Running",
			"run-D": "Failed",
		}, got)
	})
}

// TestExecutorForwardsStatusPredicates covers a mutable-column filter: the
// runs delta filters each run's latest row (the buffer collapses first), and
// the lake holds only final rows, disjoint from the delta's, so a filter for
// in-flight runs is forwarded and exact: a run finished in the delta isn't
// a false match, and the lake's finished runs don't match.
func TestExecutorForwardsStatusPredicates(t *testing.T) {
	forModes(t, func(t *testing.T, f *fixture) {
		q := f.base
		done := q.Add(time.Minute)
		f.lakeRun(t, f.account, "run-C", "Completed", q, &done)
		f.deltaRun("run-A", "Running", q, nil, "")
		f.deltaRun("run-B", "Completed", q, &done, "")

		inFlight := Predicate{Column: "status", Op: OpIn, Value: []any{"Queued", "Running"}}
		appID := Predicate{Column: "app_id", Op: OpEq, Value: f.app.String()}
		got := f.query(t, []Table{TableRuns},
			map[Table][]Predicate{TableRuns: {inFlight, appID}},
			`SELECT run_id, status FROM runs WHERE status IN ('Queued', 'Running')`)
		require.Equal(t, map[string]string{"run-A": "Running"}, got)

		require.Len(t, f.streamer.reqs, 1)
		require.Equal(t, TableRuns, f.streamer.reqs[0].Table)
		require.Equal(t, []Predicate{inFlight, appID}, f.streamer.reqs[0].Predicates)
	})
}

func TestExecutorSpansUnionLakeAndDelta(t *testing.T) {
	forModes(t, func(t *testing.T, f *fixture) {
		q := f.base
		insertLakeSpan := func(spanID, attrs string) {
			_, err := f.db.ExecContext(context.Background(), `INSERT INTO inngest.run_trace_spans
				(account_id, env_id, run_id, run_queued_at, app_id, app_name, function_id, function_slug, name,
				 start_time, end_time, trace_id, span_id, parent_span_id, attributes, bucket_at)
				VALUES (?::UUID, ?::UUID, 'run-A', ?, ?::UUID, 'app', ?::UUID, 'fn', 'executor.step', ?, ?, 'trace-1', ?, '', ?::JSON::VARIANT, ?)`,
				f.account.String(), f.env.String(), q, f.app.String(), f.fn.String(), q, q, spanID, attrs, f.lakeBucketAt())
			require.NoError(t, err)
		}
		insertLakeSpan("span-1", `{"v":"lake"}`)
		insertLakeSpan("span-2", `{"v":"lake"}`)
		// Spans aren't collapsed: each row is its own emission.
		f.streamer.rows[TableSpans] = []map[string]any{{
			"account_id": f.account, "env_id": f.env, "run_id": "run-A", "run_queued_at": q, "app_id": f.app,
			"app_name": "app", "function_id": f.fn, "function_slug": "fn", "name": "executor.step",
			"start_time": q, "end_time": q, "trace_id": "trace-1", "span_id": "span-3", "parent_span_id": "",
			"attributes": `{"v":"delta"}`,
		}}

		got := f.query(t, []Table{TableSpans}, nil, `SELECT span_id, attributes.v::VARCHAR FROM spans`)
		require.Equal(t, map[string]string{"span-1": "lake", "span-2": "lake", "span-3": "delta"}, got)
	})
}

func TestExecutorEventsUnionLakeAndDelta(t *testing.T) {
	forModes(t, func(t *testing.T, f *fixture) {
		q := f.base
		insertLakeEvent := func(account uuid.UUID, id string, receivedAt time.Time) {
			_, err := f.db.ExecContext(context.Background(), `INSERT INTO inngest.events
				(account_id, env_id, internal_id, received_at, source, event_id, event_name, event_data, event_v, event_ts, event_meta, bucket_at)
				VALUES (?::UUID, ?::UUID, ?, ?, 'api', ?, 'app/x', '{"src":"lake"}'::JSON::VARIANT, '1', ?, '{}'::JSON::VARIANT, ?)`,
				account.String(), f.env.String(), "internal-"+id, receivedAt, id, receivedAt, f.lakeBucketAt())
			require.NoError(t, err)
		}
		insertLakeEvent(f.account, "e1", q)
		insertLakeEvent(uuid.New(), "e-other", q)                   // another tenant
		insertLakeEvent(f.account, "e-old", f.from.Add(-time.Hour)) // outside the received_at bound
		f.streamer.rows[TableEvents] = []map[string]any{{
			"account_id": f.account, "env_id": f.env, "internal_id": "internal-e2", "received_at": q,
			"source": "api", "event_id": "e2", "event_name": "app/x", "event_data": `{"src":"delta"}`,
			"event_v": "1", "event_ts": q, "event_meta": `{}`,
		}}

		got := f.query(t, []Table{TableEvents}, nil, `SELECT event_id, event_data.src::VARCHAR FROM events`)
		require.Equal(t, map[string]string{"e1": "lake", "e2": "delta"}, got)
	})
}

func TestExecutorDropsTempTablesAndRejectsOverCap(t *testing.T) {
	forModes(t, func(t *testing.T, f *fixture) {
		q := f.base
		for i := range 3 {
			f.deltaRun("run-"+string(rune('a'+i)), "Running", q, nil, "")
		}
		_, err := f.exec.Query(context.Background(), Query{
			AccountID: f.account, EnvID: f.env, From: f.from, To: f.to,
			Lake: LakeSource{Catalog: "inngest"}, Tables: []Table{TableRuns}, RowCap: 2,
			SQL: `SELECT run_id FROM runs`,
		})
		require.True(t, errors.Is(err, ErrRowCapExceeded), "got %v", err)

		rows, err := f.exec.Query(context.Background(), Query{
			AccountID: f.account, EnvID: f.env, From: f.from, To: f.to,
			Lake: LakeSource{Catalog: "inngest"}, Tables: []Table{TableRuns}, RowCap: 10,
			SQL: `SELECT run_id FROM runs`,
		})
		require.NoError(t, err)
		require.NoError(t, rows.Close())

		var n int
		require.NoError(t, f.db.QueryRowContext(context.Background(),
			`SELECT count(*) FROM duckdb_tables() WHERE temporary AND table_name LIKE '__delta_%'`).Scan(&n))
		require.Zero(t, n, "delta TEMP tables should be dropped when a query closes")
	})
}

// TestExecutorStreamsOnlyColumnsTheQueryReads checks the delta is requested
// with just the columns the planned query reads: the query's own, those its
// filters read, and what the views need (raw_runs' collapse for runs).
func TestExecutorStreamsOnlyColumnsTheQueryReads(t *testing.T) {
	forModes(t, func(t *testing.T, f *fixture) {
		names := func(cols []Column) []string {
			var out []string
			for _, c := range cols {
				out = append(out, c.Name)
			}
			return out
		}

		f.query(t, []Table{TableSpans}, nil, `SELECT span_id, span_id FROM spans`)
		require.Len(t, f.streamer.reqs, 1)
		require.Equal(t, []string{"span_id"}, names(f.streamer.reqs[0].Columns))

		// A filter's column is read too, even when the filter is pushed into
		// the scan rather than projected.
		f.streamer.reqs = nil
		f.query(t, []Table{TableSpans}, nil, `SELECT span_id, span_id FROM spans WHERE name = 'executor.step'`)
		require.Len(t, f.streamer.reqs, 1)
		require.ElementsMatch(t, []string{"span_id", "name"}, names(f.streamer.reqs[0].Columns))

		f.streamer.reqs = nil
		f.query(t, []Table{TableRuns}, nil, `SELECT run_id, status FROM runs`)
		require.Len(t, f.streamer.reqs, 1)
		got := names(f.streamer.reqs[0].Columns)
		for _, c := range []string{"account_id", "env_id", "run_id", "queued_at", "started_at", "ended_at", "status"} {
			require.Contains(t, got, c, "raw_runs' collapse and the query read %s", c)
		}
		require.NotContains(t, got, "attributes")
		require.NotContains(t, got, "output")
	})
}

// TestExecutorReportsTheQueryError checks a query that fails to bind returns
// its own error, not the streamed deltas' cancellation ("context canceled")
// that its failure causes.
func TestExecutorReportsTheQueryError(t *testing.T) {
	forModes(t, func(t *testing.T, f *fixture) {
		_, err := f.exec.Query(context.Background(), Query{
			AccountID: f.account, EnvID: f.env, From: f.from, To: f.to,
			Lake: LakeSource{Catalog: "inngest"}, Watermark: f.base,
			Tables: []Table{TableRuns}, RowCap: 100,
			Render: func(src map[Table]Source) (string, []any, error) {
				s := src[TableRuns]
				var defs []string
				var args []any
				for _, d := range s.Prelude {
					defs = append(defs, d.SQL)
					args = append(args, d.Args...)
				}
				return "WITH " + strings.Join(defs, ", ") + " SELECT no_such_column FROM " + s.Name, args, nil
			},
		})
		require.ErrorContains(t, err, `"no_such_column" not found`)
		require.NotErrorIs(t, err, context.Canceled)
	})
}
