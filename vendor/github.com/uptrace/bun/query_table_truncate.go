package bun

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

// TruncateTableQuery builds TRUNCATE TABLE statements.
type TruncateTableQuery struct {
	baseQuery
	cascadeQuery

	continueIdentity bool
	comment          string
}

var _ Query = (*TruncateTableQuery)(nil)

// NewTruncateTableQuery creates a TruncateTableQuery attached to the given DB.
func NewTruncateTableQuery(db *DB) *TruncateTableQuery {
	q := &TruncateTableQuery{
		baseQuery: baseQuery{
			db: db,
		},
	}
	return q
}

func (q *TruncateTableQuery) Conn(db IConn) *TruncateTableQuery {
	q.setConn(db)
	return q
}

func (q *TruncateTableQuery) Model(model any) *TruncateTableQuery {
	q.setModel(model)
	return q
}

func (q *TruncateTableQuery) Err(err error) *TruncateTableQuery {
	q.setErr(err)
	return q
}

//------------------------------------------------------------------------------

func (q *TruncateTableQuery) Table(tables ...string) *TruncateTableQuery {
	for _, table := range tables {
		q.addTable(schema.UnsafeIdent(table))
	}
	return q
}

func (q *TruncateTableQuery) TableExpr(query string, args ...any) *TruncateTableQuery {
	q.addTable(schema.SafeQuery(query, args))
	return q
}

func (q *TruncateTableQuery) ModelTableExpr(query string, args ...any) *TruncateTableQuery {
	q.modelTableName = schema.SafeQuery(query, args)
	return q
}

//------------------------------------------------------------------------------

func (q *TruncateTableQuery) ContinueIdentity() *TruncateTableQuery {
	q.continueIdentity = true
	return q
}

func (q *TruncateTableQuery) Cascade() *TruncateTableQuery {
	q.cascade = true
	return q
}

func (q *TruncateTableQuery) Restrict() *TruncateTableQuery {
	q.restrict = true
	return q
}

//------------------------------------------------------------------------------

// Comment adds a comment to the query, wrapped by /* ... */.
func (q *TruncateTableQuery) Comment(comment string) *TruncateTableQuery {
	q.comment = comment
	return q
}

//------------------------------------------------------------------------------

func (q *TruncateTableQuery) Operation() string {
	return "TRUNCATE TABLE"
}

func (q *TruncateTableQuery) AppendQuery(
	gen schema.QueryGen, b []byte,
) (_ []byte, err error) {
	if q.err != nil {
		return nil, q.err
	}

	b = appendComment(b, q.comment)

	if !gen.HasFeature(feature.TableTruncate) {
		b = append(b, "DELETE FROM "...)

		b, err = q.appendTables(gen, b)
		if err != nil {
			return nil, err
		}

		return b, nil
	}

	b = append(b, "TRUNCATE TABLE "...)

	b, err = q.appendTables(gen, b)
	if err != nil {
		return nil, err
	}

	if q.db.HasFeature(feature.TableIdentity) {
		if q.continueIdentity {
			b = append(b, " CONTINUE IDENTITY"...)
		} else {
			b = append(b, " RESTART IDENTITY"...)
		}
	}

	b = q.appendCascade(gen, b)

	return b, nil
}

//------------------------------------------------------------------------------

func (q *TruncateTableQuery) Exec(ctx context.Context, dest ...any) (sql.Result, error) {
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

	return res, nil
}
