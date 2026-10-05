package federate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/inngest/inngest/pkg/duckdb/schema"

	"github.com/google/uuid"
)

// Executor runs queries over lake ∪ delta. Each query holds one connection
// for its whole life: the deltas are ingested into TEMP tables on it (or
// streamed into the query itself, with Streaming), and the query that reads
// them through the views runs on it too.
type Executor struct {
	DB       *sql.DB
	Delta    DeltaStreamer
	Ingester Ingester
	// Streaming feeds each delta straight into the query over quack's
	// SEND_DATA streams, as a MATERIALIZED CTE per delta, instead of
	// ingesting it into a TEMP table first: no DDL, no extra copy, and the
	// query starts while the deltas are still arriving. Requires a quack
	// connection; Ingester is unused.
	Streaming bool
}

// Query is one federated query.
type Query struct {
	AccountID uuid.UUID
	EnvID     uuid.UUID
	// AllTenants reads every tenant's rows, lake and delta, ignoring
	// AccountID/EnvID: Insights' internal raw mode only, never a product
	// query. It's explicit so a zero AccountID can't mean "everyone".
	AllTenants bool
	// From and To bound the run's queued time, [From, To).
	From, To time.Time
	Lake     LakeSource
	// Watermark is W, the split between the stores: lake rows with
	// bucket_at <= W, delta rows with bucket_at > W. Any W works whose lake
	// side has every row up to W (the export frontier or earlier) and whose
	// buffer still has every row after it (no partition above it dropped).
	// A zero W reads everything from the delta.
	Watermark time.Time
	// Tables are the logical tables SQL references.
	Tables []Table
	// Predicates are conjuncts per table that a delta may pre-filter on (SQL
	// must still apply the full predicate). All are forwarded; the
	// DeltaStreamer applies those it can represent and drops the rest.
	//
	// Future work: push GROUP BY with associative aggregates (count, sum,
	// min, max, any; avg as sum and count) into the delta, so it returns partial
	// aggregates DuckDB merges with the lake's instead of raw rows. That
	// needs exact predicates rather than pre-filters, and holds only where
	// lake and delta rows are disjoint (spans, events), not for runs, which
	// collapse across the two.
	Predicates map[Table][]Predicate
	// RowCap bounds each table's delta.
	RowCap int
	// SQL reads the logical tables by name (runs, spans). It may not start
	// with its own WITH clause; wrap it in a subquery instead. Ignored when
	// Render is set.
	SQL  string
	Args []any
	// Render, if set, composes the final query from each table's federated
	// physical source (e.g. the Insights transpiler, passing them to its
	// table macros), instead of wrapping SQL in the default views.
	Render func(sources map[Table]Source) (string, []any, error)
}

// CTEDef is one CTE definition ("name AS (...)") and the args its '?'
// placeholders bind to.
type CTEDef struct {
	SQL  string
	Args []any
}

// Source is one federated physical table, for Query.Render.
type Source struct {
	// Name is the CTE holding the table's lake ∪ delta rows, uncollapsed
	// (SourceName); pass it to a macro taking a physical source.
	Name string
	// Prelude is the CTE definitions Name needs, ending with its own. Place
	// them first in the query's top-level WITH, each once: with Streaming
	// they include the delta's stream scan, which may be read only once and
	// so must not be nested in a CTE DuckDB could inline at several
	// references.
	Prelude []CTEDef
}

// Rows is a query result. Close releases the result, drops the delta TEMP
// tables and returns the connection.
type Rows struct {
	*sql.Rows
	conn   *sql.Conn
	temps  []string
	closed atomic.Bool
}

func (r *Rows) Close() error {
	if !r.closed.CompareAndSwap(false, true) {
		return nil
	}
	err := r.Rows.Close()
	if cerr := releaseConn(r.conn, r.temps); err == nil {
		err = cerr
	}
	return err
}

var querySeq atomic.Int64

// Query streams and ingests the deltas q needs, then runs q.SQL over the
// logical views on the same connection.
func (e *Executor) Query(ctx context.Context, q Query) (*Rows, error) {
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(q.SQL)), "WITH") {
		return nil, fmt.Errorf("federate: query SQL must not start with WITH; wrap it in a subquery")
	}

	conn, err := e.DB.Conn(ctx)
	if err != nil {
		return nil, err
	}
	// The federated sources UNION ALL the lake with the delta; keeping
	// insertion order runs that union, and the delta's casts to VARIANT,
	// on one thread (about 10x slower on JSON-heavy deltas). Results carry
	// no order beyond the query's own ORDER BY, so drop it for this
	// connection; releaseConn resets it before the connection is pooled.
	if _, err := conn.ExecContext(ctx, "SET preserve_insertion_order = false;"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("federate: %w", err)
	}

	needed := deltaTables(q.Tables)
	seq := querySeq.Add(1)
	var temps []string
	var closers []func() error
	closeReaders := func() {
		for _, c := range closers {
			_ = c()
		}
		closers = nil
	}
	fail := func(err error) (*Rows, error) {
		closeReaders()
		_ = releaseConn(conn, temps)
		return nil, err
	}

	// Only stream the delta columns the query reads.
	used, err := deltaColumnsUsed(ctx, conn, q, needed, seq)
	if err != nil {
		return fail(err)
	}

	deltaRel := map[Table]string{}
	deltaCols := map[Table]map[string]bool{}
	var streams []driver.QuackStream
	preludes := map[Table]string{}
	for _, t := range needed {
		cols := used[t]
		if cols == nil {
			cols = DeltaColumns(t)
		}
		deltaCols[t] = columnSet(cols)
		req := DeltaRequest{
			AccountID: q.AccountID, EnvID: q.EnvID, AllTenants: q.AllTenants, Table: t,
			Columns: cols, From: q.From, To: q.To,
			Watermark: q.Watermark, RowCap: q.RowCap,
		}
		req.Predicates = q.Predicates[t]
		if es, ok := e.Delta.(EncodedDeltaStreamer); ok && e.Streaming {
			r, err := es.StreamEncoded(ctx, req)
			if err != nil {
				return fail(fmt.Errorf("federate: streaming %s delta: %w", t, err))
			}
			closers = append(closers, r.Close)
			if err := sameColumns(r.Schema(), req.Columns); err != nil {
				return fail(fmt.Errorf("federate: %s delta: %w", t, err))
			}
			st, def, err := encodedDeltaStream(t, r, q.RowCap, deltaRows)
			if err != nil {
				return fail(err)
			}
			streams = append(streams, st)
			preludes[t] = def
			deltaRel[t] = deltaCTEName(t)
			continue
		}
		r, err := e.Delta.Stream(ctx, req)
		if err != nil {
			return fail(fmt.Errorf("federate: streaming %s delta: %w", t, err))
		}
		if err := sameColumns(r.Schema(), req.Columns); err != nil {
			_ = r.Close()
			return fail(fmt.Errorf("federate: %s delta: %w", t, err))
		}
		if e.Streaming {
			st, def, err := deltaStream(t, r, q.RowCap)
			if err != nil {
				_ = r.Close()
				return fail(err)
			}
			closers = append(closers, r.Close)
			streams = append(streams, st)
			preludes[t] = def
			deltaRel[t] = deltaCTEName(t)
			continue
		}
		name := fmt.Sprintf("__delta_%s_%d", t, seq)
		temps = append(temps, name)
		n, err := e.Ingester.Ingest(ctx, conn, name, r, q.RowCap)
		if err != nil {
			if errors.Is(err, ErrRowCapExceeded) {
				recordRowCapExceeded(ctx, t, deltaRows)
			}
			return fail(err)
		}
		count := &deltaCount{table: t, kind: deltaRows, total: int(n)}
		count.done(ctx)
		deltaRel[t] = name
	}

	sqlText, args, err := assemble(q, needed, deltaRel, preludes, deltaCols)
	if err != nil {
		return fail(err)
	}
	qctx := ctx
	if len(streams) > 0 {
		qctx = driver.WithQuackStreams(ctx, streams...)
	}
	rows, err := conn.QueryContext(qctx, sqlText, args...)
	// QueryContext has consumed every stream by the time it returns.
	closeReaders()
	if err != nil {
		return fail(fmt.Errorf("federate: query: %w", err))
	}
	return &Rows{Rows: rows, conn: conn, temps: temps}, nil
}

// assemble renders the final query over the federated sources: each table's
// source CTE (lake ∪ the relation in deltaRel, with any streamed delta's own
// CTE first), then either q.Render's query or the plain views and q.SQL.
func assemble(q Query, needed []Table, deltaRel map[Table]string, preludes map[Table]string, deltaCols map[Table]map[string]bool) (string, []any, error) {
	sources := make(map[Table]Source, len(needed))
	var preludeSQL []string
	var args []any
	for _, t := range needed {
		src, err := sourceCTE(t, sourceSpec{
			AccountID: q.AccountID, EnvID: q.EnvID, AllTenants: q.AllTenants, From: q.From, To: q.To,
			Lake: q.Lake, Watermark: q.Watermark, Delta: deltaRel[t], DeltaColumns: deltaCols[t],
		})
		if err != nil {
			return "", nil, err
		}
		var prelude []CTEDef
		if def, ok := preludes[t]; ok {
			prelude = append(prelude, CTEDef{SQL: def})
		}
		prelude = append(prelude, src)
		sources[t] = Source{Name: SourceName(t), Prelude: prelude}
		for _, d := range prelude {
			preludeSQL = append(preludeSQL, d.SQL)
			args = append(args, d.Args...)
		}
	}
	if q.Render != nil {
		sqlText, rargs, err := q.Render(sources)
		if err != nil {
			return "", nil, fmt.Errorf("federate: render: %w", err)
		}
		return sqlText, rargs, nil
	}
	views := append([]string{}, preludeSQL...)
	for _, t := range needed {
		v, err := rawView(t, q.Lake)
		if err != nil {
			return "", nil, err
		}
		views = append(views, v)
	}
	sqlText := "WITH " + strings.Join(views, ",\n") + "\nSELECT * FROM (\n" + q.SQL + "\n)"
	return sqlText, append(args, q.Args...), nil
}

// deltaTables returns the delta tables the logical tables need, in a stable
// order.
func deltaTables(tables []Table) []Table {
	var out []Table
	seen := map[Table]bool{}
	add := func(t Table) {
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	for _, t := range tables {
		add(t)
	}
	return out
}

// Physical is the physical table each logical delta table is stored as.
var Physical = map[Table]schema.Table{TableRuns: schema.Runs, TableSpans: schema.Spans, TableEvents: schema.Events}

func sameColumns(got, want []Column) error {
	if len(got) != len(want) {
		return fmt.Errorf("schema has %d columns, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			return fmt.Errorf("column %d is %v, want %v", i, got[i], want[i])
		}
	}
	return nil
}

// releaseConn drops the query's TEMP tables, restores the session setting
// Query changed, and returns the connection to the pool.
func releaseConn(conn *sql.Conn, temps []string) error {
	dropTemps(conn, temps)
	_, _ = conn.ExecContext(context.Background(), "RESET preserve_insertion_order;")
	return conn.Close()
}

func dropTemps(conn *sql.Conn, temps []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, name := range temps {
		_, _ = conn.ExecContext(ctx, "DROP TABLE IF EXISTS temp.main."+quoteIdent(name)+";")
	}
}
