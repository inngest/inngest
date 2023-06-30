package duckdb

/*
#include <duckdb.h>
*/
import "C"

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"math/big"
	"unsafe"
)

type conn struct {
	con    *C.duckdb_connection
	closed bool
	tx     bool
}

func (c *conn) CheckNamedValue(nv *driver.NamedValue) error {
	switch nv.Value.(type) {
	case *big.Int, Interval:
		return nil
	}
	return driver.ErrSkip
}

func (c *conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if c.closed {
		panic("database/sql/driver: misuse of duckdb driver: ExecContext after Close")
	}

	stmts, size, err := c.extractStmts(query)
	if err != nil {
		return nil, err
	}
	defer C.duckdb_destroy_extracted(&stmts)

	// execute all statements without args, except the last one
	for i := C.idx_t(0); i < size-1; i++ {
		stmt, err := c.prepareExtractedStmt(stmts, i)
		if err != nil {
			return nil, err
		}
		// send nil args to execute statement and ignore result
		_, err = stmt.ExecContext(ctx, nil)
		stmt.Close()
		if err != nil {
			return nil, err
		}
	}

	// prepare and execute last statement with args and return result
	stmt, err := c.prepareExtractedStmt(stmts, size-1)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	return stmt.ExecContext(ctx, args)
}

func (c *conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if c.closed {
		panic("database/sql/driver: misuse of duckdb driver: QueryContext after Close")
	}

	stmts, size, err := c.extractStmts(query)
	if err != nil {
		return nil, err
	}
	defer C.duckdb_destroy_extracted(&stmts)

	// execute all statements without args, except the last one
	for i := C.idx_t(0); i < size-1; i++ {
		stmt, err := c.prepareExtractedStmt(stmts, i)
		if err != nil {
			return nil, err
		}
		// send nil args to execute statement and ignore result (using ExecContext since we're ignoring the result anyway)
		_, err = stmt.ExecContext(ctx, nil)
		stmt.Close()
		if err != nil {
			return nil, err
		}
	}

	// prepare and execute last statement with args and return result
	stmt, err := c.prepareExtractedStmt(stmts, size-1)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, args)
	if err != nil {
		stmt.Close()
		return nil, err
	}

	// we can't close the statement before the query result rows are closed
	stmt.closeOnRowsClose = true
	return rows, err
}

func (c *conn) Prepare(cmd string) (driver.Stmt, error) {
	if c.closed {
		panic("database/sql/driver: misuse of duckdb driver: Prepare after Close")
	}
	return c.prepareStmt(cmd)
}

// Deprecated: Use BeginTx instead.
func (c *conn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *conn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if c.tx {
		panic("database/sql/driver: misuse of duckdb driver: multiple Tx")
	}

	if opts.ReadOnly {
		return nil, errors.New("read-only transactions are not supported")
	}

	switch sql.IsolationLevel(opts.Isolation) {
	case sql.LevelDefault:
	default:
		return nil, errors.New("isolation levels other than default are not supported")
	}

	if _, err := c.ExecContext(ctx, "BEGIN TRANSACTION", nil); err != nil {
		return nil, err
	}

	c.tx = true
	return &tx{c}, nil
}

func (c *conn) Close() error {
	if c.closed {
		panic("database/sql/driver: misuse of duckdb driver: Close of already closed connection")
	}
	c.closed = true

	C.duckdb_disconnect(c.con)

	return nil
}

func (c *conn) prepareStmt(cmd string) (*stmt, error) {
	cmdstr := C.CString(cmd)
	defer C.free(unsafe.Pointer(cmdstr))

	var s C.duckdb_prepared_statement
	if state := C.duckdb_prepare(*c.con, cmdstr, &s); state == C.DuckDBError {
		dbErr := C.GoString(C.duckdb_prepare_error(s))
		C.duckdb_destroy_prepare(&s)
		return nil, errors.New(dbErr)
	}

	return &stmt{c: c, stmt: &s}, nil
}

func (c *conn) extractStmts(query string) (C.duckdb_extracted_statements, C.idx_t, error) {
	cquery := C.CString(query)
	defer C.free(unsafe.Pointer(cquery))

	var stmts C.duckdb_extracted_statements
	stmtsCount := C.duckdb_extract_statements(*c.con, cquery, &stmts)
	if stmtsCount == 0 {
		err := C.GoString(C.duckdb_extract_statements_error(stmts))
		C.duckdb_destroy_extracted(&stmts)
		if err != "" {
			return nil, 0, errors.New(err)
		}
		return nil, 0, errors.New("no statements found")
	}

	return stmts, stmtsCount, nil
}

func (c *conn) prepareExtractedStmt(extractedStmts C.duckdb_extracted_statements, index C.idx_t) (*stmt, error) {
	var s C.duckdb_prepared_statement
	if state := C.duckdb_prepare_extracted_statement(*c.con, extractedStmts, index, &s); state == C.DuckDBError {
		dbErr := C.GoString(C.duckdb_prepare_error(s))
		C.duckdb_destroy_prepare(&s)
		return nil, errors.New(dbErr)
	}

	return &stmt{c: c, stmt: &s}, nil
}
