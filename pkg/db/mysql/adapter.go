// Package mysql provides a stub db.Adapter for MySQL.
//
// This is a placeholder for future MySQL support. It is not yet implemented;
// calling any method will panic.
package mysql

import (
	"context"
	"database/sql"

	"github.com/inngest/inngest/pkg/db"
)

var (
	_ db.Adapter   = (*Adapter)(nil)
	_ db.TxAdapter = (*TxAdapter)(nil)
)

// Adapter is a stub MySQL adapter. All query methods panic with "not implemented".
type Adapter struct {
	conn *sql.DB
}

// New creates a new MySQL stub adapter.
func New(conn *sql.DB) *Adapter {
	return &Adapter{conn: conn}
}

func (a *Adapter) Dialect() db.Dialect       { return db.DialectMySQL }
func (a *Adapter) Q() db.Querier             { panic("mysql: not implemented") }
func (a *Adapter) Helpers() db.DialectHelpers { panic("mysql: not implemented") }
func (a *Adapter) Close() error              { return nil }

func (a *Adapter) WithTx(ctx context.Context) (db.TxAdapter, error) {
	tx, err := a.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &TxAdapter{Adapter: *a, tx: tx}, nil
}

// TxAdapter is a stub MySQL transaction adapter.
type TxAdapter struct {
	Adapter
	tx *sql.Tx
}

func (t *TxAdapter) Commit(ctx context.Context) error  { return t.tx.Commit() }
func (t *TxAdapter) Rollback(ctx context.Context) error { return t.tx.Rollback() }
