// +build !go1.9

package ocsql

import (
	"database/sql/driver"
	"errors"
)

// Dummy error for setSpanStatus (does exist as sql.ErrConnDone in 1.9+)
var errConnDone = errors.New("database/sql: connection is already closed")

// ocDriver implements driver.Driver
type ocDriver struct {
	parent  driver.Driver
	options TraceOptions
}

func wrapDriver(d driver.Driver, o TraceOptions) driver.Driver {
	return ocDriver{parent: d, options: o}
}

func wrapConn(c driver.Conn, options TraceOptions) driver.Conn {
	return &ocConn{parent: c, options: options}
}

func wrapStmt(stmt driver.Stmt, query string, options TraceOptions) driver.Stmt {
	s := ocStmt{parent: stmt, query: query, options: options}
	_, hasExeCtx := stmt.(driver.StmtExecContext)
	_, hasQryCtx := stmt.(driver.StmtQueryContext)
	c, hasColCnv := stmt.(driver.ColumnConverter)
	switch {
	case !hasExeCtx && !hasQryCtx && !hasColCnv:
		return struct {
			driver.Stmt
		}{s}
	case !hasExeCtx && hasQryCtx && !hasColCnv:
		return struct {
			driver.Stmt
			driver.StmtQueryContext
		}{s, s}
	case hasExeCtx && !hasQryCtx && !hasColCnv:
		return struct {
			driver.Stmt
			driver.StmtExecContext
		}{s, s}
	case hasExeCtx && hasQryCtx && !hasColCnv:
		return struct {
			driver.Stmt
			driver.StmtExecContext
			driver.StmtQueryContext
		}{s, s, s}
	case !hasExeCtx && !hasQryCtx && hasColCnv:
		return struct {
			driver.Stmt
			driver.ColumnConverter
		}{s, c}
	case !hasExeCtx && hasQryCtx && hasColCnv:
		return struct {
			driver.Stmt
			driver.StmtQueryContext
			driver.ColumnConverter
		}{s, s, c}
	case hasExeCtx && !hasQryCtx && hasColCnv:
		return struct {
			driver.Stmt
			driver.StmtExecContext
			driver.ColumnConverter
		}{s, s, c}
	case hasExeCtx && hasQryCtx && hasColCnv:
		return struct {
			driver.Stmt
			driver.StmtExecContext
			driver.StmtQueryContext
			driver.ColumnConverter
		}{s, s, s, c}
	}
	panic("unreachable")
}
