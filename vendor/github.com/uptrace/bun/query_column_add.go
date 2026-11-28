package bun

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

// AddColumnQuery builds ALTER TABLE ... ADD COLUMN statements.
type AddColumnQuery struct {
	baseQuery

	ifNotExists bool
	comment     string
}

var _ Query = (*AddColumnQuery)(nil)

// NewAddColumnQuery creates an AddColumnQuery bound to the provided DB.
func NewAddColumnQuery(db *DB) *AddColumnQuery {
	q := &AddColumnQuery{
		baseQuery: baseQuery{
			db: db,
		},
	}
	return q
}

func (q *AddColumnQuery) Conn(db IConn) *AddColumnQuery {
	q.setConn(db)
	return q
}

func (q *AddColumnQuery) Model(model any) *AddColumnQuery {
	q.setModel(model)
	return q
}

func (q *AddColumnQuery) Err(err error) *AddColumnQuery {
	q.setErr(err)
	return q
}

// Apply calls each function in fns, passing the AddColumnQuery as an argument.
func (q *AddColumnQuery) Apply(fns ...func(*AddColumnQuery) *AddColumnQuery) *AddColumnQuery {
	for _, fn := range fns {
		if fn != nil {
			q = fn(q)
		}
	}
	return q
}

//------------------------------------------------------------------------------

func (q *AddColumnQuery) Table(tables ...string) *AddColumnQuery {
	for _, table := range tables {
		q.addTable(schema.UnsafeIdent(table))
	}
	return q
}

func (q *AddColumnQuery) TableExpr(query string, args ...any) *AddColumnQuery {
	q.addTable(schema.SafeQuery(query, args))
	return q
}

func (q *AddColumnQuery) ModelTableExpr(query string, args ...any) *AddColumnQuery {
	q.modelTableName = schema.SafeQuery(query, args)
	return q
}

//------------------------------------------------------------------------------

func (q *AddColumnQuery) ColumnExpr(query string, args ...any) *AddColumnQuery {
	q.addColumn(schema.SafeQuery(query, args))
	return q
}

func (q *AddColumnQuery) IfNotExists() *AddColumnQuery {
	q.ifNotExists = true
	return q
}

//------------------------------------------------------------------------------

// Comment adds a comment to the query, wrapped by /* ... */.
func (q *AddColumnQuery) Comment(comment string) *AddColumnQuery {
	q.comment = comment
	return q
}

//------------------------------------------------------------------------------

func (q *AddColumnQuery) Operation() string {
	return "ADD COLUMN"
}

func (q *AddColumnQuery) AppendQuery(gen schema.QueryGen, b []byte) (_ []byte, err error) {
	if q.err != nil {
		return nil, q.err
	}

	b = appendComment(b, q.comment)

	if len(q.columns) != 1 {
		return nil, fmt.Errorf("bun: AddColumnQuery requires exactly one column")
	}

	b = append(b, "ALTER TABLE "...)

	b, err = q.appendFirstTable(gen, b)
	if err != nil {
		return nil, err
	}

	b = append(b, " ADD "...)

	if q.ifNotExists {
		b = append(b, "IF NOT EXISTS "...)
	}

	b, err = q.columns[0].AppendQuery(gen, b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

//------------------------------------------------------------------------------

func (q *AddColumnQuery) Exec(ctx context.Context, dest ...any) (sql.Result, error) {
	if q.ifNotExists && !q.hasFeature(feature.AlterColumnExists) {
		return nil, feature.NewNotSupportError(feature.AlterColumnExists)
	}

	// if a comment is propagated via the context, use it
	setCommentFromContext(ctx, q)

	queryBytes, err := q.AppendQuery(q.db.gen, q.db.makeQueryBytes())
	if err != nil {
		return nil, err
	}

	query := internal.String(queryBytes)
	return q.exec(ctx, q, query)
}
