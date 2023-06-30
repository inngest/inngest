package duckdb

import "context"

type tx struct {
	c *conn
}

func (t *tx) Commit() error {
	if t.c == nil || !t.c.tx {
		panic("database/sql/driver: misuse of duckdb driver: extra Commit")
	}

	t.c.tx = false
	_, err := t.c.ExecContext(context.Background(), "COMMIT TRANSACTION", nil)
	t.c = nil

	return err
}

func (t *tx) Rollback() error {
	if t.c == nil || !t.c.tx {
		panic("database/sql/driver: misuse of duckdb driver: extra Rollback")
	}

	t.c.tx = false
	_, err := t.c.ExecContext(context.Background(), "ROLLBACK", nil)
	t.c = nil

	return err
}
