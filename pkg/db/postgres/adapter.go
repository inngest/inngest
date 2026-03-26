// Package postgres implements the db.Adapter interface for PostgreSQL databases.
package postgres

import (
	"context"
	"database/sql"

	sqlc "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/postgres"
	"github.com/inngest/inngest/pkg/db"
	"github.com/inngest/inngest/pkg/db/driverhelp"
)

var (
	_ db.Adapter   = (*Adapter)(nil)
	_ db.TxAdapter = (*TxAdapter)(nil)
)

// Adapter implements db.Adapter for PostgreSQL.
type Adapter struct {
	conn *sql.DB
	q    *pgQuerier
	h    *helpers
}

// New creates a new PostgreSQL adapter wrapping the given database connection.
func New(conn *sql.DB) *Adapter {
	return &Adapter{
		conn: conn,
		q:    &pgQuerier{q: sqlc.New(conn)},
		h:    &helpers{},
	}
}

func (a *Adapter) Dialect() db.Dialect              { return db.DialectPostgres }
func (a *Adapter) Q() db.Querier                    { return a.q }
func (a *Adapter) Helpers() driverhelp.DialectHelpers { return a.h }
func (a *Adapter) Close() error                     { return nil }

func (a *Adapter) WithTx(ctx context.Context) (db.TxAdapter, error) {
	tx, err := a.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &TxAdapter{
		Adapter: Adapter{
			conn: a.conn,
			q:    &pgQuerier{q: sqlc.New(tx)},
			h:    a.h,
		},
		tx: tx,
	}, nil
}

// TxAdapter wraps Adapter with transaction commit/rollback.
type TxAdapter struct {
	Adapter
	tx *sql.Tx
}

func (t *TxAdapter) Commit(ctx context.Context) error  { return t.tx.Commit() }
func (t *TxAdapter) Rollback(ctx context.Context) error { return t.tx.Rollback() }
