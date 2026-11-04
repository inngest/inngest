package bun

import (
	"context"
	"database/sql"
	"errors"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

// MergeQuery builds MERGE statements for dialects that support them.
type MergeQuery struct {
	baseQuery
	returningQuery

	using   schema.QueryWithArgs
	on      schema.QueryWithArgs
	when    []schema.QueryAppender
	comment string
}

var _ Query = (*MergeQuery)(nil)

// NewMergeQuery creates a MergeQuery associated with the provided DB.
func NewMergeQuery(db *DB) *MergeQuery {
	q := &MergeQuery{
		baseQuery: baseQuery{
			db: db,
		},
	}
	if !q.db.HasFeature(feature.Merge) {
		q.setErr(errors.New("bun: merge not supported for current dialect"))
	}
	return q
}

func (q *MergeQuery) Conn(db IConn) *MergeQuery {
	q.setConn(db)
	return q
}

func (q *MergeQuery) Model(model any) *MergeQuery {
	q.setModel(model)
	return q
}

func (q *MergeQuery) Err(err error) *MergeQuery {
	q.setErr(err)
	return q
}

// Apply calls each function in fns, passing the MergeQuery as an argument.
func (q *MergeQuery) Apply(fns ...func(*MergeQuery) *MergeQuery) *MergeQuery {
	for _, fn := range fns {
		if fn != nil {
			q = fn(q)
		}
	}
	return q
}

func (q *MergeQuery) With(name string, query Query) *MergeQuery {
	q.addWith(NewWithQuery(name, query))
	return q
}

func (q *MergeQuery) WithRecursive(name string, query Query) *MergeQuery {
	q.addWith(NewWithQuery(name, query).Recursive())
	return q
}

func (q *MergeQuery) WithQuery(query *WithQuery) *MergeQuery {
	q.addWith(query)
	return q
}

// ------------------------------------------------------------------------------

func (q *MergeQuery) Table(tables ...string) *MergeQuery {
	for _, table := range tables {
		q.addTable(schema.UnsafeIdent(table))
	}
	return q
}

func (q *MergeQuery) TableExpr(query string, args ...any) *MergeQuery {
	q.addTable(schema.SafeQuery(query, args))
	return q
}

func (q *MergeQuery) ModelTableExpr(query string, args ...any) *MergeQuery {
	q.modelTableName = schema.SafeQuery(query, args)
	return q
}

//------------------------------------------------------------------------------

// Returning adds a RETURNING clause to the query.
//
// To suppress the auto-generated RETURNING clause, use `Returning("NULL")`.
// Supported for PostgreSQL 17+ and MSSQL (via OUTPUT clause)
func (q *MergeQuery) Returning(query string, args ...any) *MergeQuery {
	q.addReturning(schema.SafeQuery(query, args))
	return q
}

//------------------------------------------------------------------------------

func (q *MergeQuery) Using(s string, args ...any) *MergeQuery {
	q.using = schema.SafeQuery(s, args)
	return q
}

func (q *MergeQuery) On(s string, args ...any) *MergeQuery {
	q.on = schema.SafeQuery(s, args)
	return q
}

// WhenInsert for when insert clause.
func (q *MergeQuery) WhenInsert(expr string, fn func(q *InsertQuery) *InsertQuery) *MergeQuery {
	sq := NewInsertQuery(q.db)
	// apply the model as default into sub query, since appendColumnsValues required
	if q.model != nil {
		sq = sq.Model(q.model)
	}
	sq = sq.Apply(fn)
	q.when = append(q.when, &whenInsert{expr: expr, query: sq})
	return q
}

// WhenUpdate for when update clause.
func (q *MergeQuery) WhenUpdate(expr string, fn func(q *UpdateQuery) *UpdateQuery) *MergeQuery {
	sq := NewUpdateQuery(q.db)
	// apply the model as default into sub query
	if q.model != nil {
		sq = sq.Model(q.model)
	}
	sq = sq.Apply(fn)
	q.when = append(q.when, &whenUpdate{expr: expr, query: sq})
	return q
}

// WhenDelete for when delete clause.
func (q *MergeQuery) WhenDelete(expr string) *MergeQuery {
	q.when = append(q.when, &whenDelete{expr: expr})
	return q
}

// When for raw expression clause.
func (q *MergeQuery) When(expr string, args ...any) *MergeQuery {
	q.when = append(q.when, schema.SafeQuery(expr, args))
	return q
}

//------------------------------------------------------------------------------

// Comment adds a comment to the query, wrapped by /* ... */.
func (q *MergeQuery) Comment(comment string) *MergeQuery {
	q.comment = comment
	return q
}

//------------------------------------------------------------------------------

func (q *MergeQuery) Operation() string {
	return "MERGE"
}

func (q *MergeQuery) AppendQuery(gen schema.QueryGen, b []byte) (_ []byte, err error) {
	if q.err != nil {
		return nil, q.err
	}

	b = appendComment(b, q.comment)

	gen = formatterWithModel(gen, q)

	b, err = q.appendWith(gen, b)
	if err != nil {
		return nil, err
	}

	b = append(b, "MERGE "...)
	if q.db.dialect.Name() == dialect.PG {
		b = append(b, "INTO "...)
	}

	b, err = q.appendFirstTableWithAlias(gen, b)
	if err != nil {
		return nil, err
	}

	b = append(b, " USING "...)
	b, err = q.using.AppendQuery(gen, b)
	if err != nil {
		return nil, err
	}

	b = append(b, " ON "...)
	b, err = q.on.AppendQuery(gen, b)
	if err != nil {
		return nil, err
	}

	for _, w := range q.when {
		b = append(b, " WHEN "...)
		b, err = w.AppendQuery(gen, b)
		if err != nil {
			return nil, err
		}
	}

	if q.hasFeature(feature.Output) && q.hasReturning() {
		b = append(b, " OUTPUT "...)
		b, err = q.appendOutput(gen, b)
		if err != nil {
			return nil, err
		}
	}

	if q.hasFeature(feature.MergeReturning) && q.hasReturning() {
		b = append(b, " RETURNING "...)
		b, err = q.appendReturning(gen, b)
		if err != nil {
			return nil, err
		}
	}

	// A MERGE statement must be terminated by a semi-colon (;).
	b = append(b, ";"...)

	return b, nil
}

//------------------------------------------------------------------------------

func (q *MergeQuery) Scan(ctx context.Context, dest ...any) error {
	_, err := q.scanOrExec(ctx, dest, true)
	return err
}

func (q *MergeQuery) Exec(ctx context.Context, dest ...any) (sql.Result, error) {
	return q.scanOrExec(ctx, dest, len(dest) > 0)
}

func (q *MergeQuery) scanOrExec(
	ctx context.Context, dest []any, hasDest bool,
) (sql.Result, error) {
	if q.err != nil {
		return nil, q.err
	}

	// Run append model hooks before generating the query.
	if err := q.beforeAppendModel(ctx, q); err != nil {
		return nil, err
	}

	// if a comment is propagated via the context, use it
	setCommentFromContext(ctx, q)

	// Generate the query before checking hasReturning.
	queryBytes, err := q.AppendQuery(q.db.gen, q.db.makeQueryBytes())
	if err != nil {
		return nil, err
	}

	useScan := hasDest || (q.hasReturning() && q.hasFeature(feature.InsertReturning|feature.MergeReturning|feature.Output))
	var model Model

	if useScan {
		var err error
		model, err = q.getModel(dest)
		if err != nil {
			return nil, err
		}
	}

	query := internal.String(queryBytes)
	var res sql.Result

	if useScan {
		res, err = q.scan(ctx, q, query, model, true)
		if err != nil {
			return nil, err
		}
	} else {
		res, err = q.exec(ctx, q, query)
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

// String returns the generated SQL query string. The MergeQuery instance must not be
// modified during query generation to ensure multiple calls to String() return identical results.
func (q *MergeQuery) String() string {
	buf, err := q.AppendQuery(q.db.QueryGen(), nil)
	if err != nil {
		panic(err)
	}
	return string(buf)
}

//------------------------------------------------------------------------------

type whenInsert struct {
	expr  string
	query *InsertQuery
}

func (w *whenInsert) AppendQuery(gen schema.QueryGen, b []byte) (_ []byte, err error) {
	b = append(b, w.expr...)
	if w.query != nil {
		b = append(b, " THEN INSERT"...)
		b, err = w.query.appendColumnsValues(gen, b, true)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

type whenUpdate struct {
	expr  string
	query *UpdateQuery
}

func (w *whenUpdate) AppendQuery(gen schema.QueryGen, b []byte) (_ []byte, err error) {
	b = append(b, w.expr...)
	if w.query != nil {
		b = append(b, " THEN UPDATE SET "...)
		b, err = w.query.appendSet(gen, b)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

type whenDelete struct {
	expr string
}

func (w *whenDelete) AppendQuery(gen schema.QueryGen, b []byte) (_ []byte, err error) {
	b = append(b, w.expr...)
	b = append(b, " THEN DELETE"...)
	return b, nil
}
