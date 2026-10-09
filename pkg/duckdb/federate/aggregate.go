package federate

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/inngest/inngest/pkg/duckdb/parser"
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
	// Select, if set, reads the merged result as the table agg (columns:
	// the keys' and aggs' As names), e.g. to order or limit it; Args bind
	// its '?' placeholders. Unset, every merged row is returned.
	Select *parser.SelectStatement
	Args   []any
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
// delta itself, so only partials are streamed.
type AggregateStreamer interface {
	StreamAggregate(ctx context.Context, req AggregateRequest) (EncodedReader, error)
}

// ErrNotExact is returned by an AggregateStreamer that can't apply some
// predicate exactly; the executor then aggregates the delta's rows instead.
var ErrNotExact = errors.New("federate: predicate can't be applied exactly")

// aggRelation is the merged result's name, which AggregateQuery.Select reads.
const aggRelation = "agg"

// Aggregate runs q, aggregating the delta in the buffer when Delta is an
// AggregateStreamer that can apply q's predicates exactly, and over the
// delta's rows in DuckDB otherwise. Both give the same rows.
func (e *Executor) Aggregate(ctx context.Context, q AggregateQuery) (*Rows, error) {
	if err := validateAggregate(q); err != nil {
		return nil, err
	}
	if as, ok := e.Delta.(AggregateStreamer); ok {
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

	lake, err := lakePartials(q)
	if err != nil {
		return nil, err
	}
	sides := unionAll(selectStar(tableRef("__agg_lake")), selectStar(tableRef(deltaCTEName(q.Table))), true)
	merge := &parser.SelectStatement{
		Columns: mergeList(q),
		From:    from(&parser.TableSubqueryRef{Select: sides}),
		GroupBy: &parser.GroupByClause{All: true},
	}
	sqlText, err := render(withCTEs(outerSelect(q), deltaDef, cte("__agg_lake", lake), cte(aggRelation, merge)))
	if err != nil {
		return nil, err
	}

	conn, err := e.DB.Conn(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := conn.ExecContext(ctx, "SET preserve_insertion_order = false;"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("federate: %w", err)
	}
	rows, err := conn.QueryContext(driver.WithQuackStreams(ctx, stream), sqlText, q.Args...)
	if err != nil {
		_ = releaseConn(conn)
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
		Render: func(sources map[Table]Source) (*parser.SelectStatement, []any, error) {
			src := sources[q.Table]
			where, err := predicatesExpr(q.Table, q.Predicates)
			if err != nil {
				return nil, nil, err
			}
			agg := &parser.SelectStatement{
				Columns: append(keyList(q), aggList(q)...),
				From:    from(tableRef(src.Name)),
				Where:   where,
				GroupBy: &parser.GroupByClause{All: true},
			}
			return withCTEs(outerSelect(q), append(src.Prelude, cte(aggRelation, agg))...), q.Args, nil
		},
	})
}

// lakePartials is the lake side's partials: q's filter and the source's
// tenant, run-time and watermark bounds over the lake table.
func lakePartials(q AggregateQuery) (*parser.SelectStatement, error) {
	timeCol := colRef(sourceTimeColumn[q.Table])
	where, err := predicatesExpr(q.Table, q.Predicates)
	if err != nil {
		return nil, err
	}
	return &parser.SelectStatement{
		Columns: append(keyList(q), aggList(q)...),
		From:    from(q.Lake.table(Physical[q.Table].Name)),
		Where: and(
			binary("=", colRef("account_id"), uuidLit(q.AccountID.String())),
			binary("=", colRef("env_id"), uuidLit(q.EnvID.String())),
			binary(">=", timeCol, timeLit(q.From)),
			binary("<", timeCol, timeLit(q.To)),
			binary("<=", colRef("bucket_at"), timeLit(q.Watermark)),
			where,
		),
		GroupBy: &parser.GroupByClause{All: true},
	}, nil
}

func call(name string, args ...parser.Expr) parser.Expr {
	return &parser.FunctionExpr{Name: []string{name}, Args: args}
}

// keyList is q's group keys as select items.
func keyList(q AggregateQuery) []*parser.SelectItem {
	out := make([]*parser.SelectItem, len(q.GroupBy))
	for i, k := range q.GroupBy {
		expr := colRef(k.Column)
		if k.Bucket > 0 {
			width := &parser.IntervalExpr{Value: numLit(int64(k.Bucket / time.Second)), Unit: "SECOND"}
			expr = cast(call("time_bucket", width, expr), "TIMESTAMPTZ")
		}
		out[i] = item(expr, k.As)
	}
	return out
}

// aggList is q's aggregates as select items.
func aggList(q AggregateQuery) []*parser.SelectItem {
	out := make([]*parser.SelectItem, len(q.Aggs))
	for i, a := range q.Aggs {
		out[i] = item(aggExpr(a.Func, a.Column), a.As)
	}
	return out
}

// mergeList is the merge of each side's partials: the keys as they are,
// then each aggregate's merge.
func mergeList(q AggregateQuery) []*parser.SelectItem {
	out := make([]*parser.SelectItem, 0, len(q.GroupBy)+len(q.Aggs))
	for _, k := range q.GroupBy {
		out = append(out, item(colRef(k.As), ""))
	}
	for _, a := range q.Aggs {
		col := colRef(a.As)
		var m parser.Expr
		switch a.Func {
		case AggCount:
			m = cast(call("sum", col), "BIGINT")
		case AggMin:
			m = call("min", col)
		case AggMax:
			m = call("max", col)
		}
		out = append(out, item(m, a.As))
	}
	return out
}

func aggExpr(f AggFunc, column string) parser.Expr {
	switch f {
	case AggCount:
		if column == "" {
			return cast(call("count", &parser.StarExpr{}), "BIGINT")
		}
		return cast(call("count", colRef(column)), "BIGINT")
	default:
		return call(string(f), colRef(column))
	}
}

func outerSelect(q AggregateQuery) *parser.SelectStatement {
	if q.Select == nil {
		return selectStar(tableRef(aggRelation))
	}
	return q.Select
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

// predicatesExpr is preds as one DuckDB condition (nil for none).
func predicatesExpr(t Table, preds []Predicate) (parser.Expr, error) {
	var out parser.Expr
	for _, p := range preds {
		c, err := predicateExpr(t, p)
		if err != nil {
			return nil, err
		}
		out = and(out, c)
	}
	return out, nil
}

// predicateExpr is one predicate as a condition (validatePredicates checked
// its shape).
func predicateExpr(t Table, p Predicate) (parser.Expr, error) {
	switch p.Op {
	case OpAnd, OpOr:
		var out parser.Expr
		for _, a := range p.Args {
			c, err := predicateExpr(t, a)
			if err != nil {
				return nil, err
			}
			if out == nil {
				out = c
				continue
			}
			out = binary(string(p.Op), out, c)
		}
		return out, nil
	case OpNot:
		c, err := predicateExpr(t, p.Args[0])
		if err != nil {
			return nil, err
		}
		return &parser.UnaryExpr{Op: "NOT", X: c}, nil
	}
	c, _ := Physical[t].Column(p.Column)
	value := func(v any) (parser.Expr, error) {
		if c.Type == schema.TypeUUID {
			if u, ok := v.(uuid.UUID); ok {
				v = u.String()
			}
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("federate: UUID column %q compared to a %T", p.Column, v)
			}
			return uuidLit(s), nil
		}
		return lit(v)
	}
	col := colRef(p.Column)
	switch p.Op {
	case OpIn:
		vals, _ := p.Value.([]any)
		list := make([]parser.Expr, len(vals))
		for i, v := range vals {
			e, err := value(v)
			if err != nil {
				return nil, err
			}
			list[i] = e
		}
		return &parser.InExpr{X: col, List: list}, nil
	}
	v, err := value(p.Value)
	if err != nil {
		return nil, err
	}
	switch p.Op {
	case OpLike:
		return &parser.LikeExpr{X: col, Pattern: v, Op: "LIKE"}, nil
	case OpNeq:
		return binary("<>", col, v), nil
	default:
		return binary(string(p.Op), col, v), nil
	}
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
	return validatePredicates(q.Table, q.Predicates, "aggregate")
}

// validatePredicates checks preds can be applied exactly on both sides, as
// predicateExpr builds them for the lake: scalar columns, no JSON keys, and
// operators both stores agree on. what names the caller, for errors.
func validatePredicates(t Table, preds []Predicate, what string) error {
	phys := Physical[t]
	var check func(p Predicate) error
	check = func(p Predicate) error {
		if p.IsBool() {
			if len(p.Args) == 0 || (p.Op == OpNot && len(p.Args) != 1) {
				return fmt.Errorf("federate: %s: %s needs operands", what, p.Op)
			}
			for _, a := range p.Args {
				if err := check(a); err != nil {
					return err
				}
			}
			return nil
		}
		c, ok := phys.Column(p.Column)
		if !ok {
			return fmt.Errorf("federate: %s: %s has no column %q", what, t, p.Column)
		}
		if !aggregatable[c.Type] {
			return fmt.Errorf("federate: %s: column %q isn't a scalar", what, p.Column)
		}
		if p.Path != "" {
			return fmt.Errorf("federate: %s: predicate on a key of %q isn't supported", what, p.Column)
		}
		switch p.Op {
		case OpEq, OpNeq:
		case OpLt, OpLte, OpGt, OpGte:
			if c.Type == schema.TypeUUID {
				return fmt.Errorf("federate: %s: UUID column %q only takes =, != and IN", what, p.Column)
			}
		case OpIn:
			if vals, ok := p.Value.([]any); !ok || len(vals) == 0 {
				return fmt.Errorf("federate: %s: IN on %q needs a non-empty []any", what, p.Column)
			}
		case OpLike:
			if _, ok := p.Value.(string); !ok || c.Type != schema.TypeVarchar {
				return fmt.Errorf("federate: %s: LIKE on %q needs a string column and pattern", what, p.Column)
			}
		default:
			return fmt.Errorf("federate: %s: unsupported operator %q", what, p.Op)
		}
		return nil
	}
	for _, p := range preds {
		if err := check(p); err != nil {
			return err
		}
	}
	return nil
}
