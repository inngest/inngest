// Copyright 2025 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite // import "modernc.org/sqlite"

import (
	"database/sql/driver"
	"fmt"
	"sync"

	sqlite3 "modernc.org/sqlite/lib"
	"modernc.org/sqlite/vtab"
)

// Driver implements database/sql/driver.Driver.
//
// Registration functions and methods must be called before the first call to
// Open. The methods are safe to call concurrently with each other, so a
// *Driver can be handed out for several packages to fill from their init
// functions; the ordering requirement against Open still stands.
//
// Most code has no use for this type. sql.Open("sqlite", dsn) and
// [NewConnector] both go through the driver this package registers as
// "sqlite", which carries everything registered with [RegisterFunction],
// [RegisterScalarFunction], [RegisterDeterministicScalarFunction],
// [RegisterCollationUtf8], [RegisterConnectionHook] and
// [vtab.RegisterModule].
//
// A Driver a caller constructs is not equivalent to that one. It starts out
// empty, and the package-level registration functions always apply to the
// registered driver, never to a constructed one. Connections it opens
// therefore run without the package-level functions and collations -- and
// where such a registration overrides a SQLite built-in of the same name, they
// run with SQLite's built-in in force instead. Virtual table modules are the
// one exception: those registered through the package-level path are held
// process-globally and reach every Driver.
//
// A constructed Driver is filled in through its own methods, each of which
// registers on that Driver alone: [Driver.RegisterFunction],
// [Driver.RegisterScalarFunction],
// [Driver.RegisterDeterministicScalarFunction],
// [Driver.RegisterCollationUtf8], [Driver.RegisterConnectionHook] and
// [Driver.RegisterModule]. The zero Driver is ready to use; there is no
// constructor. What it registers stays on it, so two constructed Drivers do
// not see each other's registrations and neither leaks into the package-level
// driver. Modules it registers itself are installed in addition to the
// process-global ones, not instead of them, and where both registered the
// same name the package-level implementation wins on its connections.
//
// Constructing one is supported for the private-registration pattern: a driver
// registered under a name of its own with sql.Register, so that its functions,
// collations, modules and connection hooks apply to its own connections rather
// than to every connection in the process. Prefer sql.Open or NewConnector for
// anything else.
type Driver struct {
	// mu guards the registration state below against concurrent
	// registrations. Open reads that state without it, which the ordering
	// requirement in the type documentation makes safe.
	mu sync.Mutex
	// user defined functions that are added to every new connection on Open
	udfs map[string]*userDefinedFunction
	// collations that are added to every new connection on Open
	collations map[string]*collation
	// connection hooks are called after a connection is opened
	connectionHooks []ConnectionHookFn
	// modules holds registered virtual table modules that should be added to
	// every new connection on Open.
	modules map[string]vtab.Module
}

var d = &Driver{
	udfs:            make(map[string]*userDefinedFunction, 0),
	collations:      make(map[string]*collation, 0),
	connectionHooks: make([]ConnectionHookFn, 0),
	modules:         make(map[string]vtab.Module, 0),
}

func defaultDriver() *Driver { return d }

// Open returns a new connection to the database. The name is a string in a
// driver-specific format.
//
// Open may return a cached connection (one previously closed), but doing so is
// unnecessary; the sql package maintains a pool of idle connections for
// efficient re-use.
//
// The returned connection is only used by one goroutine at a time.
//
// The name may be a filename, e.g., "/tmp/mydata.sqlite", or a URI, in which
// case it may include a '?' followed by one or more query parameters.
// For example, "file:///tmp/mydata.sqlite?_pragma=foreign_keys(1)&_time_format=sqlite".
// The supported query parameters are:
//
// _pragma: Each value will be run as a "PRAGMA ..." statement (with the PRAGMA
// keyword added for you). May be specified more than once, '&'-separated. For more
// information on supported PRAGMAs see: https://www.sqlite.org/pragma.html
//
// The following shorthand keys set common PRAGMAs for easier DSN compatibility
// when migrating from github.com/mattn/go-sqlite3. Each value is validated
// against the same set github.com/mattn/go-sqlite3 accepts (case-insensitive);
// an unrecognized value fails the connection with an error instead of being
// silently ignored. The keys are applied in a fixed order, independent of the
// order they appear in the DSN: _busy_timeout and _auto_vacuum first (auto_vacuum
// must be set before the database is first written), then the _pragma values,
// then the remaining keys, and _query_only last. Where a shorthand key and a
// _pragma set the same PRAGMA, whichever is applied later in that order wins. If
// a key and its alias are both supplied, the alias (the second name below) wins,
// matching github.com/mattn/go-sqlite3; supplying the alias with an empty value
// therefore suppresses the PRAGMA rather than deferring to the primary key.
// Accepted values:
//
//	_busy_timeout, _timeout   -> PRAGMA busy_timeout   (an integer)
//	_foreign_keys, _fk        -> PRAGMA foreign_keys   (0 1 false true no yes off on)
//	_journal_mode, _journal   -> PRAGMA journal_mode   (DELETE TRUNCATE PERSIST MEMORY WAL OFF)
//	_synchronous, _sync       -> PRAGMA synchronous    (0 OFF 1 NORMAL 2 FULL 3 EXTRA)
//	_auto_vacuum, _vacuum     -> PRAGMA auto_vacuum    (0 NONE 1 FULL 2 INCREMENTAL)
//	_query_only               -> PRAGMA query_only     (0 1 false true no yes off on)
//
// All DSN parameters that can be validated are validated before any of them is
// applied, so a DSN carrying a typo fails without having executed the PRAGMAs
// that precede it -- a rejected DSN does not leave the database converted to WAL
// or with auto_vacuum already set.
//
// Unlike these validated shorthand keys, each _pragma value is executed verbatim
// (with PRAGMA prepended) and is not validated, so a DSN that includes _pragma
// must come from a trusted source. It is also the one case that can still fail
// partway: a bad _pragma is only rejected by SQLite as it runs, after any
// earlier _pragma in the list has taken effect.
//
// _time_format: The name of a format to use when writing time values to the database.
// The currently supported values are (1) "sqlite" for YYYY-MM-DD HH:MM:SS.SSS[+-]HH:MM
// (format 4 from https://www.sqlite.org/lang_datefunc.html#time_values with sub-second
// precision and timezone specifier) and (2) "datetime" for YYYY-MM-DD HH:MM:SS
// (format 3, matching the output of SQLite's datetime() function).
// If this parameter is not specified, then the default String() format will be used.
//
// _time_integer_format: The name of a integer format to use when writing time values.
// By default, the time is stored as string and the format can be set with _time_format
// parameter. If _time_integer_format is set, the time will be stored as an integer and
// the integer value will depend on the integer format.
// If you decide to set both _time_format and _time_integer_format, the time will be
// converted as integer and the _time_format value will be ignored.
// Currently the supported value are "unix","unix_milli", "unix_micro" and "unix_nano",
// which corresponds to seconds, milliseconds, microseconds or nanoseconds
// since unixepoch (1 January 1970 00:00:00 UTC).
//
// _inttotime: Enable conversion of time column (DATE, DATETIME,TIMESTAMP) from integer
// to time if the field contain integer (int64).
//
// _texttotime: Enable ColumnTypeScanType to report time.Time instead of string
// for TEXT columns declared as DATE, DATETIME, TIME, or TIMESTAMP. It also
// best-effort upgrades date-shaped TEXT values from columns SQLite reports with
// an empty declared type (aggregates and expressions such as MAX(d) or
// upper(d), subqueries, and typeless real columns) to time.Time, since the
// declared-type test cannot catch those (#248). When that upgrade fires, a Scan
// into interface{} yields a time.Time where it previously yielded a string, and
// a Scan into *string receives the value reformatted to RFC3339Nano rather than
// the raw stored text. A value that does not parse as a time is delivered
// unchanged as the original string.
//
// _timezone: A timezone to use for all time reads and writes, such as "UTC".
// The value is parsed by time.LoadLocation.
// Writes will convert to the timezone before formatting as a string;
// it does not impact _inttotime integer values, as they always use UTC.
// Reads will interpret timezone-less strings as being in this timezone.
// Values that are in a known timezone, such as a string with a timezone specifier
// or an integer with _inttotime (specified to be in UTC), will be converted to this timezone.
//
// _txlock: The locking behavior to use when beginning a transaction. May be
// "deferred" (the default), "immediate", or "exclusive" (case insensitive). See:
// https://www.sqlite.org/lang_transaction.html#deferred_immediate_and_exclusive_transactions
//
// _dqs: Opt-in toggle for SQLite's double-quoted string literal
// compatibility quirk on the connection. Accepts the values strconv.ParseBool
// understands ("0"/"1", "false"/"true", "f"/"t", case-insensitive). When
// absent or set to a true value, SQLite's built-in behavior is unchanged:
// a double-quoted identifier that fails to resolve is silently
// re-interpreted as a string literal. When set to a false value,
// SQLITE_DBCONFIG_DQS_DDL and SQLITE_DBCONFIG_DQS_DML are both turned
// off via sqlite3_db_config so that mistakes hidden by the legacy
// fallback surface as a parse error instead. See:
// https://www.sqlite.org/quirks.html#dblquote and
// https://gitlab.com/cznic/sqlite/-/issues/61
//
// _defensive: Opt-in toggle for SQLite's defensive connection mode, which
// disables the SQL-level features that let ordinary statements deliberately
// corrupt the database file. Accepts the values strconv.ParseBool understands
// ("0"/"1", "false"/"true", "f"/"t", case-insensitive). When absent or set to
// a false value SQLite's default is unchanged. When set to a true value the
// driver calls sqlite3_db_config(SQLITE_DBCONFIG_DEFENSIVE) immediately after
// sqlite3_open_v2 and before any other parameter is applied, so the PRAGMAs
// this driver runs, the _pragma list, and every statement the caller prepares
// are all subject to it. The parameter is parsed before sqlite3_open_v2, so an
// invalid value fails the connection without creating the database file, and
// it must appear at most once: a repeated _defensive is an error rather than
// letting the first value silently win.
//
// On such a connection PRAGMA writable_schema=ON, PRAGMA journal_mode=OFF and
// PRAGMA schema_version=N become silent no-ops, and writes to a virtual
// table's shadow tables (fts5's _data, _idx and so on) and to sqlite_dbpage
// fail with "table ... may not be modified". Reading those tables, ordinary
// use of the virtual tables that own them, and VACUUM are unaffected.
//
// Because journal_mode=OFF is one of the operations defensive mode suppresses,
// a true _defensive combined with _journal_mode=OFF (or _journal=OFF) is
// rejected rather than silently honouring neither. The unvalidated _pragma
// list is the exception, as always: _pragma=journal_mode(OFF) alongside
// _defensive=1 runs and is silently ignored by SQLite.
//
// Defensive mode is a hardening measure, not a sandbox for hostile database
// files. It is one of several steps SQLite recommends for that purpose (see
// https://www.sqlite.org/security.html); this build compiles with neither
// SQLITE_TRUSTED_SCHEMA=0 nor SQLITE_DQS=0 and the driver exposes no
// authorizer, so _defensive=1 alone does not make opening an untrusted file
// safe. It is also a property of the connection, not of the database: another
// connection to the same file, opened without the parameter, is unrestricted.
// See: https://www.sqlite.org/c3ref/c_dbconfig_defensive.html
//
// _error_rc: Opt-in error-string reporting mode for synthesised errors.
// Accepts the values strconv.ParseBool understands ("0"/"1",
// "false"/"true", "f"/"t", case-insensitive). When absent or set to a
// false value, the legacy "errstr: errmsg (rc)" form is preserved
// byte-for-byte: the canonical sqlite3_errstr(rc) and the connection's
// sqlite3_errmsg(db) are concatenated even when the latter belongs to a
// different operation, which can read as misleading on open-time
// failures such as SQLITE_CANTOPEN reporting "out of memory". When set
// to a true value, the appended errmsg is suppressed if
// sqlite3_extended_errcode(db) is inconsistent with the operation rc
// (full match first, primary code as fallback); in that case the
// canonical errstr(rc) is used alone. The Code() returned by the
// driver's *Error is unchanged in either mode. The parameter is parsed
// before sqlite3_open_v2 so open-time errors are covered. See
// https://gitlab.com/cznic/sqlite/-/issues/230.
//
// vfs: The name of the SQLite VFS to open the database with. Note the absent
// underscore prefix: this is the same parameter SQLite recognizes in a file:
// URI, and its value is passed on as the sqlite3_open_v2 zVfs argument. It
// selects any VFS registered with SQLite, in particular one returned by
// [modernc.org/sqlite/vfs.New], which exposes a Go fs.FS as a read-only VFS.
// When absent or empty the default VFS is used. Supplying the parameter more
// than once with values that differ is an error.
func (d *Driver) Open(name string) (conn driver.Conn, err error) {
	if dmesgs {
		defer func() {
			dmesg("name %q: (driver.Conn %p, err %v)", name, conn, err)
		}()
	}
	c, err := newConn(name)
	if err != nil {
		return nil, err
	}

	for _, udf := range d.udfs {
		if err = c.createFunctionInternal(udf); err != nil {
			c.Close()
			return nil, err
		}
	}
	for _, coll := range d.collations {
		if err = c.createCollationInternal(coll); err != nil {
			c.Close()
			return nil, err
		}
	}
	for _, connHookFn := range d.connectionHooks {
		if err = connHookFn(c, name); err != nil {
			c.Close()
			return nil, fmt.Errorf("connection hook: %w", err)
		}
	}
	// Register any vtab modules with this connection.
	// Note: vtab module registration applies to new connections only. If a
	// module is registered after a connection has been opened, that existing
	// connection will not see the module; open a new connection to use it.
	if err := c.registerModules(d); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

// RegisterConnectionHook registers a function to be called after each connection
// is opened. This is called after all the connection has been set up.
//
// The hook applies only to connections opened by d. To register one on the
// driver this package registers as "sqlite", and so on the connections
// sql.Open and [NewConnector] hand out, use the package-level
// [RegisterConnectionHook].
func (d *Driver) RegisterConnectionHook(fn ConnectionHookFn) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.connectionHooks = append(d.connectionHooks, fn)
}

// RegisterFunction is like the package-level [RegisterFunction] but registers
// the function on d alone, so it reaches only the connections d opens. See
// [Driver] for when to prefer this over the package-level form.
func (d *Driver) RegisterFunction(
	zFuncName string,
	impl *FunctionImpl,
) (err error) {
	if dmesgs {
		defer func() {
			dmesg("d %p, zFuncName %q, impl %p: err %v", d, zFuncName, impl, err)
		}()
	}
	return d.registerFunction(zFuncName, impl)
}

// MustRegisterFunction is like [Driver.RegisterFunction] but panics on error.
func (d *Driver) MustRegisterFunction(
	zFuncName string,
	impl *FunctionImpl,
) {
	if err := d.RegisterFunction(zFuncName, impl); err != nil {
		panic(err)
	}
}

// RegisterScalarFunction is like the package-level [RegisterScalarFunction]
// but registers the function on d alone, so it reaches only the connections d
// opens. See [Driver] for when to prefer this over the package-level form.
func (d *Driver) RegisterScalarFunction(
	zFuncName string,
	nArg int32,
	xFunc func(ctx *FunctionContext, args []driver.Value) (driver.Value, error),
) (err error) {
	if dmesgs {
		defer func() {
			dmesg("d %p, zFuncName %q, nArg %v, xFunc %p: err %v", d, zFuncName, nArg, xFunc, err)
		}()
	}
	return d.registerFunction(zFuncName, &FunctionImpl{NArgs: nArg, Scalar: xFunc, Deterministic: false})
}

// MustRegisterScalarFunction is like [Driver.RegisterScalarFunction] but
// panics on error.
func (d *Driver) MustRegisterScalarFunction(
	zFuncName string,
	nArg int32,
	xFunc func(ctx *FunctionContext, args []driver.Value) (driver.Value, error),
) {
	if err := d.RegisterScalarFunction(zFuncName, nArg, xFunc); err != nil {
		panic(err)
	}
}

// RegisterDeterministicScalarFunction is like the package-level
// [RegisterDeterministicScalarFunction] but registers the function on d alone,
// so it reaches only the connections d opens. See [Driver] for when to prefer
// this over the package-level form.
func (d *Driver) RegisterDeterministicScalarFunction(
	zFuncName string,
	nArg int32,
	xFunc func(ctx *FunctionContext, args []driver.Value) (driver.Value, error),
) (err error) {
	if dmesgs {
		defer func() {
			dmesg("d %p, zFuncName %q, nArg %v, xFunc %p: err %v", d, zFuncName, nArg, xFunc, err)
		}()
	}
	return d.registerFunction(zFuncName, &FunctionImpl{NArgs: nArg, Scalar: xFunc, Deterministic: true})
}

// MustRegisterDeterministicScalarFunction is like
// [Driver.RegisterDeterministicScalarFunction] but panics on error.
func (d *Driver) MustRegisterDeterministicScalarFunction(
	zFuncName string,
	nArg int32,
	xFunc func(ctx *FunctionContext, args []driver.Value) (driver.Value, error),
) {
	if err := d.RegisterDeterministicScalarFunction(zFuncName, nArg, xFunc); err != nil {
		panic(err)
	}
}

// RegisterCollationUtf8 is like the package-level [RegisterCollationUtf8] but
// registers the collation on d alone, so it reaches only the connections d
// opens. See [Driver] for when to prefer this over the package-level form.
func (d *Driver) RegisterCollationUtf8(
	zName string,
	impl func(left, right string) int,
) (err error) {
	if dmesgs {
		defer func() {
			dmesg("d %p, zName %q, impl %p: err %v", d, zName, impl, err)
		}()
	}
	return d.registerCollation(zName, impl, sqlite3.SQLITE_UTF8)
}

// MustRegisterCollationUtf8 is like [Driver.RegisterCollationUtf8] but panics
// on error.
func (d *Driver) MustRegisterCollationUtf8(
	zName string,
	impl func(left, right string) int,
) {
	if err := d.RegisterCollationUtf8(zName, impl); err != nil {
		panic(err)
	}
}
