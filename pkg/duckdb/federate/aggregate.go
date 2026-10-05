package federate

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/inngest/inngest/pkg/duckdb/schema"
)

// AggFunc is an aggregate whose partials merge exactly: each side computes
// it over its own rows, and DuckDB combines the two.
type AggFunc string

const (
	// AggCount counts rows (Column empty) or a column's non-NULL values.
	// Partials merge by summing.
	AggCount AggFunc = "count"
	// AggMin and AggMax merge by taking the min (max) of the partials.
	AggMin AggFunc = "min"
	AggMax AggFunc = "max"
)

// GroupKey is one GROUP BY key: a column, or a timestamp column bucketed
// into windows of Bucket (a whole number of seconds dividing a day, so both
// stores align windows to the same boundaries).
type GroupKey struct {
	Column string
	Bucket time.Duration
	// As names the key's output column.
	As string
}

// Agg is one aggregate output column.
type Agg struct {
	Func   AggFunc
	Column string
	As     string
}

// AggregateQuery is a federated GROUP BY over one table. Each side
// aggregates its own rows and DuckDB merges the partials, so only partial
// aggregates (one row per group) cross from the buffer instead of its rows.
//
// It relies on the lake and the delta being disjoint, with one row per
// entity on each side: true of spans and events, and of runs where the lake
// holds only final rows and the delta reads each run's latest row (the
// ClickHouse buffer's lake_open_runs). Predicates are the whole filter, and
// both sides apply each exactly.
type AggregateQuery struct {
	AccountID uuid.UUID
	EnvID     uuid.UUID
	// From and To bound the run's queued time (an event's received time),
	// [From, To).
	From, To  time.Time
	Lake      LakeSource
	Watermark time.Time
	Table     Table
	// Predicates is the filter: each on a scalar column (no JSON path), and
	// applied exactly by both sides (a streamer that can't returns
	// ErrNotExact, and the delta's rows are aggregated in DuckDB instead).
	Predicates []Predicate
	GroupBy    []GroupKey
	Aggs       []Agg
	// RowCap bounds the delta: its groups when it aggregates, its rows when
	// it can't and the delta is aggregated in DuckDB instead.
	RowCap int
	// SQL, if set, reads the merged result as the table agg (columns: the
	// keys' and aggs' As names), e.g. to order or limit it. It may not start
	// with WITH. Unset, every merged row is returned.
	SQL  string
	Args []any
}

// AggregateRequest asks an AggregateStreamer for a delta's partial
// aggregates: one row per group, columns the keys then the aggregates, named
// by As. Every predicate must be applied exactly, or the streamer returns
// ErrNotExact.
type AggregateRequest struct {
	DeltaRequest
	GroupBy []GroupKey
	Aggs    []Agg
}

// AggregateStreamer is an optional DeltaStreamer extension: it aggregates a
// delta itself, so only partials are streamed. The executor uses it when
// Streaming is set.
type AggregateStreamer interface {
	StreamAggregate(ctx context.Context, req AggregateRequest) (EncodedReader, error)
}

// ErrNotExact is returned by an AggregateStreamer that can't apply some
// predicate exactly; the executor then aggregates the delta's rows instead.
var ErrNotExact = errors.New("federate: predicate can't be applied exactly")

// aggRelation is the merged result's name, which AggregateQuery.SQL reads.
const aggRelation = "agg"

// Aggregate runs q, aggregating the delta in the buffer when Delta is an
// AggregateStreamer that can apply q's predicates exactly, and over the
// delta's rows in DuckDB otherwise. Both give the same rows.
func (e *Executor) Aggregate(ctx context.Context, q AggregateQuery) (*Rows, error) {
	if err := validateAggregate(q); err != nil {
		return nil, err
	}
	if as, ok := e.Delta.(AggregateStreamer); ok && e.Streaming {
		rows, err := e.aggregatePushed(ctx, as, q)
		if !errors.Is(err, ErrNotExact) {
			return rows, err
		}
		recordAggregateFallback(ctx, q.Table)
	}
	return e.aggregateRows(ctx, q)
}

// aggregatePushed streams the delta's partials from the buffer and merges
// them with the lake's.
func (e *Executor) aggregatePushed(ctx context.Context, as AggregateStreamer, q AggregateQuery) (*Rows, error) {
	r, err := as.StreamAggregate(ctx, AggregateRequest{
		DeltaRequest: DeltaRequest{
			AccountID: q.AccountID, EnvID: q.EnvID, Table: q.Table,
			From: q.From, To: q.To, Watermark: q.Watermark,
			Predicates: q.Predicates, RowCap: q.RowCap,
		},
		GroupBy: q.GroupBy, Aggs: q.Aggs,
	})
	if err != nil {
		return nil, err
	}
	defer r.Close()
	if err := sameColumns(r.Schema(), aggregateColumns(q)); err != nil {
		return nil, fmt.Errorf("federate: %s aggregate delta: %w", q.Table, err)
	}
	stream, deltaDef, err := encodedDeltaStream(q.Table, r, q.RowCap, deltaPartials)
	if err != nil {
		return nil, err
	}

	lakeSQL, lakeArgs, err := lakePartials(q)
	if err != nil {
		return nil, err
	}
	merge := fmt.Sprintf("%s AS (SELECT %s FROM (SELECT * FROM __agg_lake UNION ALL BY NAME SELECT * FROM %s) GROUP BY ALL)",
		quoteIdent(aggRelation), mergeList(q), quoteIdent(deltaCTEName(q.Table)))
	sqlText := "WITH " + strings.Join([]string{deltaDef, "__agg_lake AS (" + lakeSQL + ")", merge}, ",\n") + "\n" + outerSQL(q)
	args := append(lakeArgs, q.Args...)

	conn, err := e.DB.Conn(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := conn.ExecContext(ctx, "SET preserve_insertion_order = false;"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("federate: %w", err)
	}
	rows, err := conn.QueryContext(driver.WithQuackStreams(ctx, stream), sqlText, args...)
	if err != nil {
		_ = releaseConn(conn, nil)
		return nil, fmt.Errorf("federate: aggregate: %w", err)
	}
	return &Rows{Rows: rows, conn: conn}, nil
}

// aggregateRows aggregates the federated source (lake ∪ the delta's rows)
// in DuckDB: the delta streams rows, pre-filtered by whichever predicates it
// can push.
func (e *Executor) aggregateRows(ctx context.Context, q AggregateQuery) (*Rows, error) {
	return e.Query(ctx, Query{
		AccountID: q.AccountID, EnvID: q.EnvID, From: q.From, To: q.To,
		Lake: q.Lake, Watermark: q.Watermark, Tables: []Table{q.Table},
		Predicates: map[Table][]Predicate{q.Table: q.Predicates}, RowCap: q.RowCap,
		Render: func(sources map[Table]Source) (string, []any, error) {
			src := sources[q.Table]
			var defs []string
			var args []any
			for _, d := range src.Prelude {
				defs = append(defs, d.SQL)
				args = append(args, d.Args...)
			}
			where, whereArgs, err := predicatesSQL(q.Table, q.Predicates)
			if err != nil {
				return "", nil, err
			}
			sel := make([]string, 0, len(q.GroupBy)+len(q.Aggs))
			sel = append(sel, keyList(q)...)
			for _, a := range q.Aggs {
				sel = append(sel, aggSQL(a.Func, a.Column)+" AS "+quoteIdent(a.As))
			}
			defs = append(defs, fmt.Sprintf("%s AS (SELECT %s FROM %s WHERE %s GROUP BY ALL)",
				quoteIdent(aggRelation), strings.Join(sel, ", "), quoteIdent(src.Name), where))
			args = append(args, whereArgs...)
			return "WITH " + strings.Join(defs, ",\n") + "\n" + outerSQL(q), append(args, q.Args...), nil
		},
	})
}

// lakePartials is the lake side's partials: q's filter and the source's
// tenant, run-time and watermark bounds over the lake table.
func lakePartials(q AggregateQuery) (string, []any, error) {
	phys := Physical[q.Table]
	timeCol := quoteIdent(sourceTimeColumn[q.Table])
	conds := []string{timeCol + " >= ?", timeCol + " < ?", "bucket_at <= ?"}
	args := []any{q.From, q.To, q.Watermark}
	conds = append([]string{"account_id = ?::UUID", "env_id = ?::UUID"}, conds...)
	args = append([]any{q.AccountID.String(), q.EnvID.String()}, args...)
	where, whereArgs, err := predicatesSQL(q.Table, q.Predicates)
	if err != nil {
		return "", nil, err
	}
	conds = append(conds, where)
	args = append(args, whereArgs...)
	sel := keyList(q)
	for _, a := range q.Aggs {
		sel = append(sel, aggSQL(a.Func, a.Column)+" AS "+quoteIdent(a.As))
	}
	return fmt.Sprintf("SELECT %s FROM %s WHERE %s GROUP BY ALL",
		strings.Join(sel, ", "), q.Lake.table(phys.Name), strings.Join(conds, " AND ")), args, nil
}

// keyList renders q's group keys as DuckDB select items.
func keyList(q AggregateQuery) []string {
	out := make([]string, len(q.GroupBy))
	for i, k := range q.GroupBy {
		expr := quoteIdent(k.Column)
		if k.Bucket > 0 {
			expr = fmt.Sprintf("CAST(time_bucket(INTERVAL %d SECOND, %s) AS TIMESTAMP_MS)", int64(k.Bucket/time.Second), expr)
		}
		out[i] = expr + " AS " + quoteIdent(k.As)
	}
	return out
}

// mergeList renders the merge of each side's partials: the keys as they
// are, then each aggregate's merge.
func mergeList(q AggregateQuery) string {
	out := make([]string, 0, len(q.GroupBy)+len(q.Aggs))
	for _, k := range q.GroupBy {
		out = append(out, quoteIdent(k.As))
	}
	for _, a := range q.Aggs {
		col := quoteIdent(a.As)
		var m string
		switch a.Func {
		case AggCount:
			m = "CAST(sum(" + col + ") AS BIGINT)"
		case AggMin:
			m = "min(" + col + ")"
		case AggMax:
			m = "max(" + col + ")"
		}
		out = append(out, m+" AS "+col)
	}
	return strings.Join(out, ", ")
}

func aggSQL(f AggFunc, column string) string {
	switch f {
	case AggCount:
		if column == "" {
			return "CAST(count(*) AS BIGINT)"
		}
		return "CAST(count(" + quoteIdent(column) + ") AS BIGINT)"
	default:
		return string(f) + "(" + quoteIdent(column) + ")"
	}
}

func outerSQL(q AggregateQuery) string {
	if q.SQL == "" {
		return "SELECT * FROM " + quoteIdent(aggRelation)
	}
	return q.SQL
}

// aggregateColumns is the delta partials' schema: the keys, then the
// aggregates.
func aggregateColumns(q AggregateQuery) []Column {
	phys := Physical[q.Table]
	out := make([]Column, 0, len(q.GroupBy)+len(q.Aggs))
	for _, k := range q.GroupBy {
		c, _ := phys.Column(k.Column)
		out = append(out, Column{Name: k.As, Type: c.Type})
	}
	for _, a := range q.Aggs {
		if a.Func == AggCount {
			out = append(out, Column{Name: a.As, Type: schema.TypeBigint})
			continue
		}
		c, _ := phys.Column(a.Column)
		out = append(out, Column{Name: a.As, Type: c.Type})
	}
	return out
}

// predicatesSQL renders preds as one DuckDB condition (TRUE for none).
func predicatesSQL(t Table, preds []Predicate) (string, []any, error) {
	conds := []string{"TRUE"}
	var args []any
	for _, p := range preds {
		c, a := predicateSQL(t, p)
		conds = append(conds, c)
		args = append(args, a...)
	}
	return strings.Join(conds, " AND "), args, nil
}

// predicateSQL renders one predicate (validateAggregate checked its shape).
func predicateSQL(t Table, p Predicate) (string, []any) {
	switch p.Op {
	case OpAnd, OpOr:
		parts := make([]string, len(p.Args))
		var args []any
		for i, a := range p.Args {
			c, ca := predicateSQL(t, a)
			parts[i] = c
			args = append(args, ca...)
		}
		return "(" + strings.Join(parts, " "+string(p.Op)+" ") + ")", args
	case OpNot:
		c, args := predicateSQL(t, p.Args[0])
		return "(NOT " + c + ")", args
	}
	c, _ := Physical[t].Column(p.Column)
	placeholder := "?"
	if c.Type == schema.TypeUUID {
		placeholder = "?::UUID"
	}
	col := quoteIdent(p.Column)
	switch p.Op {
	case OpIn:
		vals, _ := p.Value.([]any)
		ph := make([]string, len(vals))
		args := make([]any, len(vals))
		for i, v := range vals {
			ph[i], args[i] = placeholder, sqlArg(v)
		}
		return fmt.Sprintf("%s IN (%s)", col, strings.Join(ph, ", ")), args
	case OpNeq:
		return fmt.Sprintf("%s <> %s", col, placeholder), []any{sqlArg(p.Value)}
	default:
		return fmt.Sprintf("%s %s %s", col, p.Op, placeholder), []any{sqlArg(p.Value)}
	}
}

// sqlArg binds UUIDs as their text, which the ?::UUID placeholder casts.
func sqlArg(v any) any {
	if u, ok := v.(uuid.UUID); ok {
		return u.String()
	}
	return v
}

// aggregatable are the column types a key, aggregate or predicate may use:
// scalars whose comparisons and grouping agree across the stores.
var aggregatable = map[schema.Type]bool{
	schema.TypeUUID: true, schema.TypeVarchar: true, schema.TypeTimestamp: true,
	schema.TypeBool: true, schema.TypeBigint: true,
}

// outputName is what a key's or aggregate's As may be: a plain identifier,
// since it's rendered into both stores' SQL.
var outputName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func validateAggregate(q AggregateQuery) error {
	phys, ok := Physical[q.Table]
	if !ok {
		return fmt.Errorf("federate: aggregate: unknown table %q", q.Table)
	}
	if len(q.GroupBy)+len(q.Aggs) == 0 {
		return fmt.Errorf("federate: aggregate: no keys or aggregates")
	}
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(q.SQL)), "WITH") {
		return fmt.Errorf("federate: aggregate SQL must not start with WITH; wrap it in a subquery")
	}
	names := map[string]bool{}
	name := func(as string) error {
		if !outputName.MatchString(as) || names[as] {
			return fmt.Errorf("federate: aggregate: output name %q is repeated or not a plain identifier", as)
		}
		names[as] = true
		return nil
	}
	column := func(n string) (schema.Column, error) {
		c, ok := phys.Column(n)
		if !ok {
			return c, fmt.Errorf("federate: aggregate: %s has no column %q", q.Table, n)
		}
		if !aggregatable[c.Type] {
			return c, fmt.Errorf("federate: aggregate: column %q isn't a scalar", n)
		}
		return c, nil
	}
	for _, k := range q.GroupBy {
		c, err := column(k.Column)
		if err != nil {
			return err
		}
		if k.Bucket != 0 {
			if c.Type != schema.TypeTimestamp {
				return fmt.Errorf("federate: aggregate: bucketed key %q isn't a timestamp", k.Column)
			}
			if k.Bucket < time.Second || k.Bucket%time.Second != 0 || (24*time.Hour)%k.Bucket != 0 {
				return fmt.Errorf("federate: aggregate: bucket %s must be whole seconds dividing a day", k.Bucket)
			}
		}
		if err := name(k.As); err != nil {
			return err
		}
	}
	for _, a := range q.Aggs {
		switch a.Func {
		case AggCount:
			if a.Column != "" {
				if _, err := column(a.Column); err != nil {
					return err
				}
			}
		case AggMin, AggMax:
			c, err := column(a.Column)
			if err != nil {
				return err
			}
			if c.Type == schema.TypeUUID {
				// ClickHouse and DuckDB order UUIDs differently, so the
				// sides' partials wouldn't merge to either's answer.
				return fmt.Errorf("federate: aggregate: %s of UUID column %q", a.Func, a.Column)
			}
		default:
			return fmt.Errorf("federate: aggregate: unsupported function %q", a.Func)
		}
		if err := name(a.As); err != nil {
			return err
		}
	}
	var check func(p Predicate) error
	check = func(p Predicate) error {
		if p.IsBool() {
			if len(p.Args) == 0 || (p.Op == OpNot && len(p.Args) != 1) {
				return fmt.Errorf("federate: aggregate: %s needs operands", p.Op)
			}
			for _, a := range p.Args {
				if err := check(a); err != nil {
					return err
				}
			}
			return nil
		}
		c, err := column(p.Column)
		if err != nil {
			return err
		}
		if p.Path != "" {
			return fmt.Errorf("federate: aggregate: predicate on a key of %q isn't supported", p.Column)
		}
		switch p.Op {
		case OpEq, OpNeq:
		case OpLt, OpLte, OpGt, OpGte:
			if c.Type == schema.TypeUUID {
				return fmt.Errorf("federate: aggregate: UUID column %q only takes =, != and IN", p.Column)
			}
		case OpIn:
			if vals, ok := p.Value.([]any); !ok || len(vals) == 0 {
				return fmt.Errorf("federate: aggregate: IN on %q needs a non-empty []any", p.Column)
			}
		case OpLike:
			if _, ok := p.Value.(string); !ok || c.Type != schema.TypeVarchar {
				return fmt.Errorf("federate: aggregate: LIKE on %q needs a string column and pattern", p.Column)
			}
		default:
			return fmt.Errorf("federate: aggregate: unsupported operator %q", p.Op)
		}
		return nil
	}
	for _, p := range q.Predicates {
		if err := check(p); err != nil {
			return err
		}
	}
	return nil
}
