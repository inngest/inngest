package bun

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

// DropTableQuery builds DROP TABLE statements.
type DropTableQuery struct {
	baseQuery
	cascadeQuery

	ifExists bool
	comment  string
}

var _ Query = (*DropTableQuery)(nil)

// NewDropTableQuery returns a DropTableQuery tied to the provided DB.
func NewDropTableQuery(db *DB) *DropTableQuery {
	q := &DropTableQuery{
		baseQuery: baseQuery{
			db: db,
		},
	}
	return q
}

func (q *DropTableQuery) Conn(db IConn) *DropTableQuery {
	q.setConn(db)
	return q
}

func (q *DropTableQuery) Model(model any) *DropTableQuery {
	q.setModel(model)
	return q
}

func (q *DropTableQuery) Err(err error) *DropTableQuery {
	q.setErr(err)
	return q
}

//------------------------------------------------------------------------------

func (q *DropTableQuery) Table(tables ...string) *DropTableQuery {
	for _, table := range tables {
		q.addTable(schema.UnsafeIdent(table))
	}
	return q
}

func (q *DropTableQuery) TableExpr(query string, args ...any) *DropTableQuery {
	q.addTable(schema.SafeQuery(query, args))
	return q
}

func (q *DropTableQuery) ModelTableExpr(query string, args ...any) *DropTableQuery {
	q.modelTableName = schema.SafeQuery(query, args)
	return q
}

//------------------------------------------------------------------------------

func (q *DropTableQuery) IfExists() *DropTableQuery {
	q.ifExists = true
	return q
}

func (q *DropTableQuery) Cascade() *DropTableQuery {
	q.cascade = true
	return q
}

func (q *DropTableQuery) Restrict() *DropTableQuery {
	q.restrict = true
	return q
}

//------------------------------------------------------------------------------

// Comment adds a comment to the query, wrapped by /* ... */.
func (q *DropTableQuery) Comment(comment string) *DropTableQuery {
	q.comment = comment
	return q
}

//------------------------------------------------------------------------------

func (q *DropTableQuery) Operation() string {
	return "DROP TABLE"
}

func (q *DropTableQuery) AppendQuery(gen schema.QueryGen, b []byte) (_ []byte, err error) {
	if q.err != nil {
		return nil, q.err
	}

	b = appendComment(b, q.comment)

	b = append(b, "DROP TABLE "...)
	if q.ifExists {
		b = append(b, "IF EXISTS "...)
	}

	b, err = q.appendTables(gen, b)
	if err != nil {
		return nil, err
	}

	b = q.appendCascade(gen, b)

	return b, nil
}

//------------------------------------------------------------------------------

func (q *DropTableQuery) Exec(ctx context.Context, dest ...any) (sql.Result, error) {
	if q.table != nil {
		if err := q.beforeDropTableHook(ctx); err != nil {
			return nil, err
		}
	}

	// if a comment is propagated via the context, use it
	setCommentFromContext(ctx, q)

	queryBytes, err := q.AppendQuery(q.db.gen, q.db.makeQueryBytes())
	if err != nil {
		return nil, err
	}

	query := internal.String(queryBytes)

	res, err := q.exec(ctx, q, query)
	if err != nil {
		return nil, err
	}

	if q.table != nil {
		if err := q.afterDropTableHook(ctx); err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (q *DropTableQuery) beforeDropTableHook(ctx context.Context) error {
	if hook, ok := q.table.ZeroIface.(BeforeDropTableHook); ok {
		if err := hook.BeforeDropTable(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

func (q *DropTableQuery) afterDropTableHook(ctx context.Context) error {
	if hook, ok := q.table.ZeroIface.(AfterDropTableHook); ok {
		if err := hook.AfterDropTable(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

// String returns the generated SQL query string. The DropTableQuery instance must not be
// modified during query generation to ensure multiple calls to String() return identical results.
func (q *DropTableQuery) String() string {
	buf, err := q.AppendQuery(q.db.QueryGen(), nil)
	if err != nil {
		panic(err)
	}
	return string(buf)
}
