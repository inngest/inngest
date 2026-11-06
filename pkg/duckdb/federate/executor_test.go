package federate

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/parser"
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
	f.exec = &Executor{DB: db, Delta: f.streamer}
	return f
}

// withFixture runs fn with a fresh fixture.
func withFixture(t *testing.T, fn func(t *testing.T, f *fixture)) { fn(t, newFixture(t)) }

// sel parses a test query.
func sel(sqlText string) *parser.SelectStatement {
	stmt, err := parser.ParseString(sqlText)
	if err != nil {
		panic(err)
	}
	return stmt
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

// deltaBucketAt is when the fixture's delta rows reached the buffer: after
// W, so the delta side keeps them.
func (f *fixture) deltaBucketAt() time.Time { return f.base.Add(time.Minute) }

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
		"bucket_at": f.deltaBucketAt(),
	})
}

func (f *fixture) query(t *testing.T, tables []Table, preds map[Table][]Predicate, sqlText string, args ...any) map[string]string {
	t.Helper()
	rows, err := f.exec.Query(context.Background(), Query{
		AccountID: f.account, EnvID: f.env, From: f.from, To: f.to,
		Lake: LakeSource{Catalog: "inngest"}, Watermark: f.watermark(), Tables: tables, Predicates: preds,
		RowCap: 1000, RunsCollapsed: true, Select: sel(sqlText), Args: args,
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

// TestExecutorQueryShapes checks the executor's CTEs compose w/ any query:
// one reading no table, one w/ its own WITH, and a pushed predicate whose
// value needs quoting.
func TestExecutorQueryShapes(t *testing.T) {
	withFixture(t, func(t *testing.T, f *fixture) {
		done := f.base.Add(time.Minute)
		f.lakeRun(t, f.account, "run-D", "Failed", f.base, &done)
		f.deltaRun("run-A", "it's done", f.base, &done, "")

		tests := []struct {
			name   string
			tables []Table
			preds  map[Table][]Predicate
			sql    string
			want   map[string]string
		}{
			{"no table", nil, nil, `SELECT 'a', 'b'`, map[string]string{"a": "b"}},
			{"its own WITH", []Table{TableRuns}, nil,
				`WITH r AS (SELECT run_id, status FROM runs) SELECT run_id, status FROM r`,
				map[string]string{"run-A": "it's done", "run-D": "Failed"}},
			{"a quoted predicate value", []Table{TableRuns},
				map[Table][]Predicate{TableRuns: {{Column: "status", Op: OpEq, Value: "it's done"}}},
				`SELECT run_id, status FROM runs WHERE status = 'it''s done'`,
				map[string]string{"run-A": "it's done"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				require.Equal(t, tt.want, f.query(t, tt.tables, tt.preds, tt.sql))
			})
		}
	})
}

func TestExecutorRunsUnion(t *testing.T) {
	withFixture(t, func(t *testing.T, f *fixture) {
		q := f.base
		done := q.Add(time.Minute)

		// A flat lake: each finished run's final row, at or below W.
		f.lakeRun(t, f.account, "run-D", "Failed", q, &done)
		// Another tenant's run, and a run outside the time bound: never visible.
		f.lakeRun(t, uuid.New(), "run-other", "Completed", q, &done)
		f.lakeRun(t, f.account, "run-old", "Completed", f.from.Add(-time.Hour), &done)
		// The delta: each other run's latest row.
		f.deltaRun("run-A", "Completed", q, &done, `{"ok":true}`)
		f.deltaRun("run-C", "Running", q, nil, "")
		// E: still open, bucketed at or below W: the runs delta isn't cut at
		// the watermark.
		f.deltaRun("run-E", "Running", q, nil, "")
		f.streamer.rows[TableRuns][len(f.streamer.rows[TableRuns])-1]["bucket_at"] = f.lakeBucketAt()
		// Another tenant's delta run: the delta side re-applies the tenant.
		f.deltaRun("run-other-delta", "Running", q, nil, "")
		f.streamer.rows[TableRuns][len(f.streamer.rows[TableRuns])-1]["account_id"] = uuid.New()

		got := f.query(t, []Table{TableRuns}, nil, `SELECT run_id, status FROM runs`)
		require.Equal(t, map[string]string{
			"run-A": "Completed",
			"run-C": "Running",
			"run-D": "Failed",
			"run-E": "Running",
		}, got)
	})
}

// TestExecutorRejectsRunsWithoutRunsCollapsed checks a runs query needs a
// flat lake: over runs straddling W, forwarded run predicates and a top-n's
// lake cut would return wrong rows.
func TestExecutorRejectsRunsWithoutRunsCollapsed(t *testing.T) {
	withFixture(t, func(t *testing.T, f *fixture) {
		query := func(tables ...Table) error {
			rows, err := f.exec.Query(context.Background(), Query{
				AccountID: f.account, EnvID: f.env, From: f.from, To: f.to,
				Lake: LakeSource{Catalog: "inngest"}, Watermark: f.watermark(), Tables: tables,
				RowCap: 1000, Select: sel(`SELECT 1 AS a, 1 AS b`),
			})
			if err == nil {
				_ = rows.Close()
			}
			return err
		}
		require.ErrorIs(t, query(TableRuns), ErrRunsNotCollapsed)
		require.ErrorIs(t, query(TableSpans, TableRuns), ErrRunsNotCollapsed)
		require.Empty(t, f.streamer.reqs, "a rejected query streams no delta")
		require.NoError(t, query(TableSpans, TableEvents))
	})
}

// TestExecutorRunsCollapsedSkipsRawRuns checks RunsCollapsed reads runs as
// the plain union: a lake and delta that are one row per run, disjoint (as
// cloud's are), come back as is, and the delta is asked for just the
// columns the query and the delta side's bounds read, none for raw_runs'
// collapse.
func TestExecutorRunsCollapsedSkipsRawRuns(t *testing.T) {
	withFixture(t, func(t *testing.T, f *fixture) {
		q := f.base
		done := q.Add(time.Minute)
		f.lakeRun(t, f.account, "run-A", "Completed", q, &done)
		f.deltaRun("run-B", "Running", q, nil, "")
		f.deltaRun("run-C", "Failed", q, &done, "")

		rows, err := f.exec.Query(context.Background(), Query{
			AccountID: f.account, EnvID: f.env, From: f.from, To: f.to,
			Lake: LakeSource{Catalog: "inngest"}, Watermark: f.watermark(), Tables: []Table{TableRuns},
			RowCap: 1000, RunsCollapsed: true, Select: sel(`SELECT run_id, status FROM runs`),
		})
		require.NoError(t, err)
		defer rows.Close()
		got := map[string]string{}
		for rows.Next() {
			var k, v string
			require.NoError(t, rows.Scan(&k, &v))
			got[k] = v
		}
		require.NoError(t, rows.Err())
		require.Equal(t, map[string]string{"run-A": "Completed", "run-B": "Running", "run-C": "Failed"}, got)

		require.Len(t, f.streamer.reqs, 1)
		var cols []string
		for _, c := range f.streamer.reqs[0].Columns {
			cols = append(cols, c.Name)
		}
		require.ElementsMatch(t, []string{"run_id", "status", "account_id", "env_id", "queued_at"}, cols)
	})
}

// TestExecutorForwardsStatusPredicates covers a mutable-column filter: the
// runs delta filters each run's latest row (the buffer collapses first), and
// the lake holds only final rows, disjoint from the delta's, so a filter for
// in-flight runs is forwarded and exact: a run finished in the delta isn't
// a false match, and the lake's finished runs don't match.
func TestExecutorForwardsStatusPredicates(t *testing.T) {
	withFixture(t, func(t *testing.T, f *fixture) {
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
	withFixture(t, func(t *testing.T, f *fixture) {
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
		deltaSpan := func(spanID string, account uuid.UUID, queuedAt, bucketAt time.Time) map[string]any {
			return map[string]any{
				"account_id": account, "env_id": f.env, "run_id": "run-A", "run_queued_at": queuedAt, "app_id": f.app,
				"app_name": "app", "function_id": f.fn, "function_slug": "fn", "name": "executor.step",
				"start_time": q, "end_time": q, "trace_id": "trace-1", "span_id": spanID, "parent_span_id": "",
				"attributes": `{"v":"delta"}`, "bucket_at": bucketAt,
			}
		}
		// The delta side re-applies the tenant, time and watermark bounds, so
		// a streamer returning too much can't leak or double-count rows.
		f.streamer.rows[TableSpans] = []map[string]any{
			deltaSpan("span-3", f.account, q, f.deltaBucketAt()),
			deltaSpan("span-other-tenant", uuid.New(), q, f.deltaBucketAt()),
			deltaSpan("span-old", f.account, f.from.Add(-time.Hour), f.deltaBucketAt()),
			deltaSpan("span-1", f.account, q, f.lakeBucketAt()), // already in the lake
		}

		got := f.query(t, []Table{TableSpans}, nil, `SELECT span_id, attributes.v::VARCHAR FROM spans`)
		require.Equal(t, map[string]string{"span-1": "lake", "span-2": "lake", "span-3": "delta"}, got)
	})
}

func TestExecutorEventsUnionLakeAndDelta(t *testing.T) {
	withFixture(t, func(t *testing.T, f *fixture) {
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
			"event_v": "1", "event_ts": q, "event_meta": `{}`, "bucket_at": f.deltaBucketAt(),
		}}

		got := f.query(t, []Table{TableEvents}, nil, `SELECT event_id, event_data.src::VARCHAR FROM events`)
		require.Equal(t, map[string]string{"e1": "lake", "e2": "delta"}, got)
	})
}

func TestExecutorRejectsOverCap(t *testing.T) {
	withFixture(t, func(t *testing.T, f *fixture) {
		q := f.base
		for i := range 3 {
			f.deltaRun("run-"+string(rune('a'+i)), "Running", q, nil, "")
		}
		_, err := f.exec.Query(context.Background(), Query{
			AccountID: f.account, EnvID: f.env, From: f.from, To: f.to,
			Lake: LakeSource{Catalog: "inngest"}, Tables: []Table{TableRuns}, RowCap: 2, RunsCollapsed: true,
			Select: sel(`SELECT run_id FROM runs`),
		})
		require.True(t, errors.Is(err, ErrRowCapExceeded), "got %v", err)

		rows, err := f.exec.Query(context.Background(), Query{
			AccountID: f.account, EnvID: f.env, From: f.from, To: f.to,
			Lake: LakeSource{Catalog: "inngest"}, Tables: []Table{TableRuns}, RowCap: 10, RunsCollapsed: true,
			Select: sel(`SELECT run_id FROM runs`),
		})
		require.NoError(t, err, "the connection is usable after a failed query")
		require.NoError(t, rows.Close())
	})
}

// TestExecutorStreamsOnlyColumnsTheQueryReads checks the delta is requested
// with just the columns the planned query reads: the query's own, those its
// filters read, and what the views need (raw_runs' collapse for runs).
func TestExecutorStreamsOnlyColumnsTheQueryReads(t *testing.T) {
	withFixture(t, func(t *testing.T, f *fixture) {
		names := func(cols []Column) []string {
			var out []string
			for _, c := range cols {
				out = append(out, c.Name)
			}
			return out
		}

		// The delta side's own bounds read the tenant, time and watermark
		// columns.
		bounds := []string{"account_id", "env_id", "run_queued_at", "bucket_at"}
		f.query(t, []Table{TableSpans}, nil, `SELECT span_id, span_id FROM spans`)
		require.Len(t, f.streamer.reqs, 1)
		require.ElementsMatch(t, append([]string{"span_id"}, bounds...), names(f.streamer.reqs[0].Columns))

		// A filter's column is read too, even when the filter is pushed into
		// the scan rather than projected.
		f.streamer.reqs = nil
		f.query(t, []Table{TableSpans}, nil, `SELECT span_id, span_id FROM spans WHERE name = 'executor.step'`)
		require.Len(t, f.streamer.reqs, 1)
		require.ElementsMatch(t, append([]string{"span_id", "name"}, bounds...), names(f.streamer.reqs[0].Columns))

		// Every scan of the table counts: a self-join reads the union.
		f.streamer.reqs = nil
		f.query(t, []Table{TableSpans}, nil, `SELECT a.span_id, b.name FROM spans a JOIN spans b ON a.parent_span_id = b.span_id`)
		require.Len(t, f.streamer.reqs, 1)
		require.ElementsMatch(t, append([]string{"span_id", "parent_span_id", "name"}, bounds...), names(f.streamer.reqs[0].Columns))

		// A JSON key the planner pushes into the scan (projected as
		// attributes.<key>) still reads its column.
		f.streamer.reqs = nil
		f.query(t, []Table{TableSpans}, nil, `SELECT attributes."_inngest.step.userland.id"::VARCHAR FROM spans`)
		require.Len(t, f.streamer.reqs, 1)
		require.ElementsMatch(t, append([]string{"attributes"}, bounds...), names(f.streamer.reqs[0].Columns))

		f.streamer.reqs = nil
		f.query(t, []Table{TableRuns}, nil, `SELECT run_id, status FROM runs`)
		require.Len(t, f.streamer.reqs, 1)
		// Runs aren't collapsed (RunsCollapsed): the query's columns and the
		// delta side's bounds (no watermark for runs).
		require.ElementsMatch(t, []string{"run_id", "status", "account_id", "env_id", "queued_at"}, names(f.streamer.reqs[0].Columns))

		// A count reads no column of the plain union, so only the delta
		// side's bounds stream.
		f.streamer.reqs = nil
		f.query(t, []Table{TableSpans}, nil, `SELECT count(*)::VARCHAR, '' FROM spans`)
		require.Len(t, f.streamer.reqs, 1)
		require.ElementsMatch(t, bounds, names(f.streamer.reqs[0].Columns))
	})
}

// TestExecutorReportsTheQueryError checks a query that fails to bind returns
// its own error, not the streamed deltas' cancellation ("context canceled")
// that its failure causes.
func TestExecutorReportsTheQueryError(t *testing.T) {
	withFixture(t, func(t *testing.T, f *fixture) {
		_, err := f.exec.Query(context.Background(), Query{
			AccountID: f.account, EnvID: f.env, From: f.from, To: f.to,
			Lake: LakeSource{Catalog: "inngest"}, Watermark: f.base,
			Tables: []Table{TableRuns}, RowCap: 100, RunsCollapsed: true,
			Render: func(src map[Table]Source) (*parser.SelectStatement, []any, error) {
				s := src[TableRuns]
				return withCTEs(sel("SELECT no_such_column FROM "+s.Name), s.Prelude...), nil, nil
			},
		})
		require.ErrorContains(t, err, `"no_such_column" not found`)
		require.NotErrorIs(t, err, context.Canceled)
	})
}

// raggedStreamer returns a delta batch whose columns don't line up with the
// requested schema.
type raggedStreamer struct{ drop bool }

func (r raggedStreamer) Stream(_ context.Context, req DeltaRequest) (BatchReader, error) {
	cols := make([][]any, len(req.Columns))
	for i := range cols {
		cols[i] = []any{nil, nil}
	}
	if r.drop {
		cols = cols[:len(cols)-1] // a column short of the schema
	} else {
		cols[0] = cols[0][:1] // one value short in the first column
	}
	return NewSliceReader(req.Columns, &Batch{Schema: req.Columns[:len(cols)], Columns: cols}), nil
}

// TestExecutorRejectsMalformedDeltaBatches checks a batch that doesn't
// match its schema fails the query instead of panicking the stream.
func TestExecutorRejectsMalformedDeltaBatches(t *testing.T) {
	tests := []struct {
		name string
		drop bool
		want string
	}{
		{name: "ragged column", want: "has 2 values, want 1"},
		{name: "missing column", drop: true, want: "-column reader schema"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withFixture(t, func(t *testing.T, f *fixture) {
				f.exec.Delta = raggedStreamer{drop: tt.drop}
				_, err := f.exec.Query(context.Background(), Query{
					AccountID: f.account, EnvID: f.env, From: f.from, To: f.to,
					Lake: LakeSource{Catalog: "inngest"}, Watermark: f.watermark(),
					Tables: []Table{TableSpans}, RowCap: 100, Select: sel(`SELECT span_id FROM spans`),
				})
				require.ErrorContains(t, err, tt.want)
			})
		})
	}
}
