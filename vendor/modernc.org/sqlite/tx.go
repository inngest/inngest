// Copyright 2025 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite // import "modernc.org/sqlite"

import (
	"context"
	"database/sql/driver"

	"modernc.org/libc"
	sqlite3 "modernc.org/sqlite/lib"
)

type tx struct {
	c *conn
}

func newTx(ctx context.Context, c *conn, opts driver.TxOptions) (*tx, error) {
	r := &tx{c: c}

	sql := "begin"
	if !opts.ReadOnly && c.beginMode != "" {
		sql = "begin " + c.beginMode
	}

	if err := r.exec(ctx, sql); err != nil {
		return nil, err
	}

	return r, nil
}

// Commit implements driver.Tx.
func (t *tx) Commit() (err error) {
	err = t.exec(context.Background(), "commit")
	if err == nil {
		return nil
	}

	// If Commit fails (e.g., SQLITE_BUSY), the connection might still be inside
	// the transaction. database/sql expects the connection to be clean (no active
	// transaction) after Commit returns.
	//
	// We check the low-level autocommit state.  0 means autocommit is disabled =>
	// we are inside a transaction.
	if t.c.db != 0 && sqlite3.Xsqlite3_get_autocommit(t.c.tls, t.c.db) == 0 {
		// Force a rollback to clean up the connection.  We ignore the rollback error
		// because we must return the original Commit error to the user.
		t.exec(context.Background(), "rollback")
	}

	return err
}

// Rollback implements driver.Tx.
func (t *tx) Rollback() (err error) {
	return t.exec(context.Background(), "rollback")
}

func (t *tx) exec(ctx context.Context, sql string) (err error) {
	psql, err := libc.CString(sql)
	if err != nil {
		return err
	}

	defer t.c.free(psql)
	//TODO use t.conn.ExecContext() instead

	if ctx != nil && ctx.Done() != nil {
		defer interruptOnDone(ctx, t.c, nil)()
	}

	if rc := sqlite3.Xsqlite3_exec(t.c.tls, t.c.db, psql, 0, 0, 0); rc != sqlite3.SQLITE_OK {
		return t.c.errstr(rc)
	}

	return nil
}
