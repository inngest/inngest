// Copyright 2026 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite // import "modernc.org/sqlite"

import (
	"context"
	"database/sql/driver"
	"strings"
)

// NewConnector returns a driver.Connector that opens connections to dsn using
// the driver this package registers as "sqlite" -- the one carrying every
// function, collation, connection hook and virtual table module registered
// through RegisterFunction, RegisterScalarFunction,
// RegisterDeterministicScalarFunction, RegisterCollationUtf8,
// RegisterConnectionHook and vtab.RegisterModule. The dsn syntax and the
// supported query parameters are documented on Driver.Open.
//
// The returned value is intended for sql.OpenDB:
//
//	c, err := sqlite.NewConnector("file:app.db?_pragma=foreign_keys(1)")
//	if err != nil {
//		return err
//	}
//	db := sql.OpenDB(c)
//	defer db.Close()
//
// For opening a database this is equivalent to sql.Open("sqlite", dsn). It
// exists for callers that need to interpose on the physical connections
// database/sql opens -- tracing, metrics, or connection-scoped setup. Such a
// caller can embed the returned Connector, override Connect, and pass its own
// wrapper to sql.OpenDB:
//
//	type tracer struct{ driver.Connector }
//
//	func (t tracer) Connect(ctx context.Context) (driver.Conn, error) {
//		conn, err := t.Connector.Connect(ctx)
//		// ... wrap conn ...
//		return conn, err
//	}
//
//	base, err := sqlite.NewConnector(dsn)
//	if err != nil {
//		return err
//	}
//	db := sql.OpenDB(tracer{base})
//
// Reaching the same driver through sql.Open requires sql.Register, which is
// process-global, rejects a name it has already seen with a panic, and offers
// no way to undo a registration; a library doing the above would have to
// invent a unique name per configuration. sql.OpenDB registers nothing.
//
// The dsn is checked here only as far as it can be without opening a database:
// a query string that does not parse, such as one carrying an invalid
// percent-escape, and conflicting vfs parameters are reported immediately.
// Everything else is validated when the connection is opened, so an unknown
// parameter or an out-of-range value is reported by Connect, and hence by the
// first use of the sql.DB, rather than by NewConnector.
//
// The returned Connector is safe for concurrent use; database/sql calls
// Connect from multiple goroutines as it grows the pool. As with sql.Open, it
// does not itself open a connection.
func NewConnector(dsn string) (driver.Connector, error) {
	// getVFSName is the first thing newConn does with the query string: it
	// parses it, which rejects a malformed query, and it rejects conflicting
	// vfs parameters. Calling that same function rather than repeating its
	// parse here keeps what NewConnector rejects aligned with what opening a
	// connection rejects, by construction.
	if _, err := getVFSName(dsnQuery(dsn)); err != nil {
		return nil, err
	}

	return &connector{dsn: dsn}, nil
}

// connector implements driver.Connector on top of the package-level driver. It
// holds no mutable state, so the concurrent Connect calls database/sql makes
// need no synchronization here.
type connector struct {
	dsn string
}

// Connect implements driver.Connector.
//
// ctx is honored only up to the point the open begins: sqlite3_open_v2 is a
// blocking call with no cancellation hook, so a cancellation arriving once it
// is under way takes effect no earlier than the first statement run on the
// returned connection.
func (c *connector) Connect(ctx context.Context) (driver.Conn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return d.Open(c.dsn)
}

// Driver implements driver.Connector. It returns the driver registered as
// "sqlite", so (*sql.DB).Driver() reports the same value for a database opened
// through this Connector as for one opened with sql.Open("sqlite", dsn).
func (c *connector) Driver() driver.Driver { return d }

// dsnQuery returns the query-parameter portion of dsn, or "" when it has none.
// It must agree with the split newConn performs, which likewise ignores a '?'
// in the first position; TestConnectorDSNSplitMatchesOpen guards that.
func dsnQuery(dsn string) string {
	if pos := strings.IndexRune(dsn, '?'); pos >= 1 {
		return dsn[pos+1:]
	}

	return ""
}
