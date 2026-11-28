// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite // import "modernc.org/sqlite"

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"math/bits"
	"net/url"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"modernc.org/libc"
	"modernc.org/libc/sys/types"
	sqlite3 "modernc.org/sqlite/lib"
)

var (
	_ driver.Conn      = (*conn)(nil)
	_ driver.Connector = (*connector)(nil)
	_ driver.Driver    = (*Driver)(nil)
	//lint:ignore SA1019 TODO implement ExecerContext
	_ driver.Execer = (*conn)(nil)
	//lint:ignore SA1019 TODO implement QueryerContext
	_ driver.Queryer                        = (*conn)(nil)
	_ driver.Result                         = (*result)(nil)
	_ driver.Rows                           = (*rows)(nil)
	_ driver.RowsColumnTypeDatabaseTypeName = (*rows)(nil)
	_ driver.RowsColumnTypeLength           = (*rows)(nil)
	_ driver.RowsColumnTypeNullable         = (*rows)(nil)
	_ driver.RowsColumnTypePrecisionScale   = (*rows)(nil)
	_ driver.RowsColumnTypeScanType         = (*rows)(nil)
	_ driver.Stmt                           = (*stmt)(nil)
	_ driver.Tx                             = (*tx)(nil)
	_ error                                 = (*Error)(nil)
)

const (
	driverName              = "sqlite"
	ptrSize                 = unsafe.Sizeof(uintptr(0))
	sqliteLockedSharedcache = sqlite3.SQLITE_LOCKED | (1 << 8)
)

func init() {
	sql.Register(driverName, defaultDriver())
	sqlite3.PatchIssue199() // https://gitlab.com/cznic/sqlite/-/issues/199

}

// Inspired by mattn/go-sqlite3: https://github.com/mattn/go-sqlite3/blob/ab91e934/sqlite3.go#L210-L226
//
// These time.Parse formats handle formats 1 through 7 listed at https://www.sqlite.org/lang_datefunc.html.
var parseTimeFormats = []string{
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02T15:04:05.999999999-07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04",
	"2006-01-02T15:04",
	"2006-01-02",
}

// interruptOnDone sets up a goroutine to interrupt the provided db when the
// context is canceled, and returns a function the caller must defer so it
// doesn't interrupt after the caller finishes.
func interruptOnDone(
	ctx context.Context,
	c *conn,
	done *int32,
) func() {
	if done == nil {
		var d int32
		done = &d
	}
	// donemu prevents a TOCTOU logical race between checking the done flag and
	// calling interrupt in the select statement below.
	var donemu sync.Mutex

	donech := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			// don't call interrupt if we were already done: it indicates that this
			// call to exec is no longer running and we would be interrupting
			// nothing, or even possibly an unrelated later call to exec.
			donemu.Lock()
			if atomic.CompareAndSwapInt32(done, 0, 1) {
				c.interrupt(c.db)
			}
			donemu.Unlock()
		case <-donech:
		}
	}()

	// the caller is expected to defer this function
	return func() {
		// set the done flag so that a context cancellation right after the caller
		// returns doesn't trigger a call to interrupt for some other statement.
		donemu.Lock()
		atomic.StoreInt32(done, 1)
		donemu.Unlock()
		close(donech)
	}
}

func getVFSName(query string) (r string, err error) {
	q, err := url.ParseQuery(query)
	if err != nil {
		return "", err
	}

	for _, v := range q["vfs"] {
		if r != "" && r != v {
			return "", fmt.Errorf("conflicting vfs query parameters: %v", q["vfs"])
		}

		r = v
	}

	return r, nil
}

// applyDQSConfig consults the _dqs DSN query parameter and, when set to a
// false value, disables SQLite's double-quoted string literal compatibility
// quirk on the connection by calling sqlite3_db_config with both
// SQLITE_DBCONFIG_DQS_DDL and SQLITE_DBCONFIG_DQS_DML. Absence or a true
// value leaves SQLite's default (DQS enabled) untouched.
//
// Called from newConn after sqlite3_open_v2 and before applyQueryParams.
// The DBCONFIG_DQS_* flags are required to be set before any statement is
// prepared on the connection, and applyQueryParams runs user-supplied
// PRAGMA statements, so this must come first.
//
// See https://www.sqlite.org/quirks.html#dblquote and
// https://gitlab.com/cznic/sqlite/-/issues/61.
func applyDQSConfig(c *conn, query string) error {
	q, err := url.ParseQuery(query)
	if err != nil {
		return err
	}
	v := q.Get("_dqs")
	if v == "" {
		return nil
	}
	on, err := strconv.ParseBool(v)
	if err != nil {
		return fmt.Errorf("invalid _dqs value %q: %w", v, err)
	}
	if on {
		// _dqs=1 is the SQLite default; nothing to do.
		return nil
	}
	for _, op := range []int32{
		sqlite3.SQLITE_DBCONFIG_DQS_DDL,
		sqlite3.SQLITE_DBCONFIG_DQS_DML,
	} {
		if rc := c.dbConfigBool(op, false); rc != sqlite3.SQLITE_OK {
			return fmt.Errorf("sqlite3_db_config(op=%d, off) returned %d", op, rc)
		}
	}
	return nil
}

// getDefensiveMode validates the _defensive DSN query parameter before
// sqlite3_open_v2 can create or mutate a database. Absence or a false value
// preserves SQLite's default behavior for backwards compatibility.
//
// The parameter is intentionally single-valued. Accepting duplicate values
// would make the security posture depend on url.Values.Get choosing the first
// value, so duplicates fail the connection before any query parameter is
// applied.
func getDefensiveMode(query string) (bool, error) {
	q, err := url.ParseQuery(query)
	if err != nil {
		return false, err
	}
	values, ok := q["_defensive"]
	if !ok {
		return false, nil
	}
	if len(values) != 1 {
		return false, fmt.Errorf("_defensive must be specified exactly once, got %d values", len(values))
	}
	on, err := strconv.ParseBool(values[0])
	if err != nil {
		return false, fmt.Errorf("invalid _defensive value %q: %w", values[0], err)
	}
	return on, nil
}

// applyDefensiveConfig enables SQLite's defensive connection mode after
// sqlite3_open_v2 and before any user-supplied PRAGMA or statement can run.
// See https://www.sqlite.org/c3ref/c_dbconfig_defensive.html.
func applyDefensiveConfig(c *conn, on bool) error {
	if !on {
		return nil
	}
	if rc := c.dbConfigBool(sqlite3.SQLITE_DBCONFIG_DEFENSIVE, true); rc != sqlite3.SQLITE_OK {
		return fmt.Errorf("sqlite3_db_config(SQLITE_DBCONFIG_DEFENSIVE, on) returned %d", rc)
	}
	return nil
}

// getErrorRcMode reads the _error_rc DSN query parameter and returns
// the parsed boolean. Called from newConn before sqlite3_open_v2 so
// open-time failures get the conditional errmsg treatment too: the
// temporary db handle that openV2 may leave behind on failure carries
// a stale errmsg from earlier initialisation, and the legacy
// "errstr: errmsg" form surfaces that as a misleading message.
//
// Absent parameter or false value preserves the SQLite-default error
// reporting byte-for-byte (legacy behavior). A true value switches
// the connection into the conditional mode described on
// errstrForDB. An unparseable value is reported as a descriptive
// error.
//
// See #230.
func getErrorRcMode(query string) (bool, error) {
	q, err := url.ParseQuery(query)
	if err != nil {
		return false, err
	}
	v := q.Get("_error_rc")
	if v == "" {
		return false, nil
	}
	on, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("invalid _error_rc value %q: %w", v, err)
	}
	return on, nil
}

// dsnPick returns the value and key name for a mattn-compatible shorthand DSN
// parameter and its alias. When both are present the alias wins, matching
// github.com/mattn/go-sqlite3. Selection is by presence, not by value, so an
// alias supplied with an empty value ("_foreign_keys=on&_fk=") selects the
// alias and yields an empty value, suppressing the PRAGMA entirely rather than
// falling back to the primary key. That too matches mattn.
func dsnPick(q url.Values, primary, alias string) (key, val string) {
	if _, ok := q[primary]; ok {
		key, val = primary, q.Get(primary)
	}
	if _, ok := q[alias]; ok {
		key, val = alias, q.Get(alias)
	}
	return key, val
}

// dsnBool reports an error unless val is a mattn-compatible boolean DSN value.
func dsnBool(key, val string) error {
	switch strings.ToLower(val) {
	case "0", "no", "false", "off", "1", "yes", "true", "on":
		return nil
	}
	return fmt.Errorf("invalid %s %q, expecting one of: 0 1 false true no yes off on", key, val)
}

// dsnEnum reports an error unless val matches one of allowed, case-insensitively.
func dsnEnum(key, val string, allowed []string) error {
	for _, a := range allowed {
		if strings.EqualFold(val, a) {
			return nil
		}
	}
	return fmt.Errorf("invalid %s %q, expecting one of: %s", key, val, strings.Join(allowed, " "))
}

// applyQueryParams validates and applies the DSN query parameters. defensive
// is the _defensive value newConn parsed and already acted on; the validation
// phase below needs it to tell whether a parameter it is about to accept can
// still take effect on the connection.
func applyQueryParams(c *conn, query string, defensive bool) error {
	q, err := url.ParseQuery(query)
	if err != nil {
		return err
	}

	// Validation phase. Everything that can be rejected is rejected here, before
	// the apply phase below executes a single statement. PRAGMA journal_mode and
	// auto_vacuum are persistent changes to the database file, so validating
	// lazily as each key is applied would let a typo in a later parameter fail
	// the connection only after the file had already been converted -- a failed
	// Open must not leave the database half-configured. Assignments to c are
	// exempt from that concern: newConn closes and discards the connection when
	// this returns an error, so they cannot outlive the failure.
	//
	// Each shorthand value is validated against the same set
	// github.com/mattn/go-sqlite3 accepts (case-insensitive). When a key and its
	// alias are both present the alias wins, matching mattn. See the Driver
	// documentation in driver.go for the full apply order and precedence.
	//
	// _pragma values are the one exception: they are executed verbatim and
	// cannot be checked here, so a malformed _pragma can still fail partway.
	busyKey, busyTimeout := dsnPick(q, "_busy_timeout", "_timeout")
	if busyTimeout != "" {
		if _, err := strconv.ParseInt(busyTimeout, 10, 64); err != nil {
			return fmt.Errorf("invalid %s %q: %w", busyKey, busyTimeout, err)
		}
	}

	autoVacuumKey, autoVacuum := dsnPick(q, "_auto_vacuum", "_vacuum")
	if autoVacuum != "" {
		if err := dsnEnum(autoVacuumKey, autoVacuum, []string{"0", "NONE", "1", "FULL", "2", "INCREMENTAL"}); err != nil {
			return err
		}
	}

	foreignKeysKey, foreignKeys := dsnPick(q, "_foreign_keys", "_fk")
	if foreignKeys != "" {
		if err := dsnBool(foreignKeysKey, foreignKeys); err != nil {
			return err
		}
	}

	journalModeKey, journalMode := dsnPick(q, "_journal_mode", "_journal")
	if journalMode != "" {
		if err := dsnEnum(journalModeKey, journalMode, []string{"DELETE", "TRUNCATE", "PERSIST", "MEMORY", "WAL", "OFF"}); err != nil {
			return err
		}
		// PRAGMA journal_mode=OFF is one of the operations defensive mode
		// suppresses, and SQLite suppresses it silently: the statement
		// succeeds and reports the unchanged mode. Accepting the
		// combination would mean honouring neither parameter without
		// telling anyone, so reject it here, alongside the other checks
		// that run before a single statement is executed.
		if defensive && strings.EqualFold(journalMode, "OFF") {
			return fmt.Errorf("conflicting DSN parameters: %s=%s cannot take effect under _defensive", journalModeKey, journalMode)
		}
	}

	synchronousKey, synchronous := dsnPick(q, "_synchronous", "_sync")
	if synchronous != "" {
		if err := dsnEnum(synchronousKey, synchronous, []string{"0", "OFF", "1", "NORMAL", "2", "FULL", "3", "EXTRA"}); err != nil {
			return err
		}
	}

	queryOnly := q.Get("_query_only")
	if queryOnly != "" {
		if err := dsnBool("_query_only", queryOnly); err != nil {
			return err
		}
	}

	if v := q.Get("_time_format"); v != "" {
		f, ok := writeTimeFormats[v]
		if !ok {
			return fmt.Errorf("unknown _time_format %q", v)
		}
		c.writeTimeFormat = f
	}
	if v := q.Get("_time_integer_format"); v != "" {
		switch v {
		case "unix":
		case "unix_milli":
		case "unix_micro":
		case "unix_nano":
		default:
			return fmt.Errorf("unknown _time_integer_format %q", v)
		}
		c.integerTimeFormat = v
	}

	if v := q.Get("_timezone"); v != "" {
		loc, err := time.LoadLocation(v)
		if err != nil {
			return fmt.Errorf("unknown _timezone %q: %w", v, err)
		}
		c.loc = loc
	}

	if v := q.Get("_txlock"); v != "" {
		lower := strings.ToLower(v)
		if lower != "deferred" && lower != "immediate" && lower != "exclusive" {
			return fmt.Errorf("unknown _txlock %q", v)
		}
		c.beginMode = v
	}

	if v := q.Get("_inttotime"); v != "" {
		onoff, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("unknown _inttotime %q, must be 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False",
				v)
		}
		c.intToTime = onoff
	}

	if v := q.Get("_texttotime"); v != "" {
		onoff, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("unknown _texttotime %q, must be 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False",
				v)
		}
		c.textToTime = onoff
	}

	// Apply phase. The order here is the documented one and is independent of
	// the order the keys appear in the DSN.
	//
	// Busy timeout must be one of the first PRAGMAs set from query params as some
	// that make changes to the database might otherwise unexpectedly fail with
	// SQLITE_BUSY.
	if busyTimeout != "" {
		if _, err := c.exec(context.Background(), "pragma busy_timeout = "+busyTimeout, nil); err != nil {
			return err
		}
	}

	// auto_vacuum must be applied while the database is still new: a journal_mode
	// change or the first table materialises page 1 and locks the setting in, so
	// it runs before the _pragma list and the other shorthand keys.
	if autoVacuum != "" {
		if _, err := c.exec(context.Background(), "pragma auto_vacuum = "+autoVacuum, nil); err != nil {
			return err
		}
	}

	var a []string
	for _, v := range q["_pragma"] {
		a = append(a, v)
	}
	// Push 'busy_timeout' first, the rest in lexicographic order, case insenstive.
	// See https://gitlab.com/cznic/sqlite/-/issues/198#note_2233423463 for
	// discussion.
	sort.Slice(a, func(i, j int) bool {
		x, y := strings.TrimSpace(strings.ToLower(a[i])), strings.TrimSpace(strings.ToLower(a[j]))
		if strings.HasPrefix(x, "busy_timeout") {
			return true
		}
		if strings.HasPrefix(y, "busy_timeout") {
			return false
		}

		return x < y
	})
	for _, v := range a {
		cmd := "pragma " + v
		_, err := c.exec(context.Background(), cmd, nil)
		if err != nil {
			return err
		}
	}

	if foreignKeys != "" {
		if _, err := c.exec(context.Background(), "pragma foreign_keys = "+foreignKeys, nil); err != nil {
			return err
		}
	}

	if journalMode != "" {
		if _, err := c.exec(context.Background(), "pragma journal_mode = "+journalMode, nil); err != nil {
			return err
		}
	}

	if synchronous != "" {
		if _, err := c.exec(context.Background(), "pragma synchronous = "+synchronous, nil); err != nil {
			return err
		}
	}

	// query_only is applied last: it makes the connection read-only, so it must
	// not precede the write-capable pragmas above (notably auto_vacuum).
	if queryOnly != "" {
		if _, err := c.exec(context.Background(), "pragma query_only = "+queryOnly, nil); err != nil {
			return err
		}
	}

	return nil
}

func unlockNotify(t *libc.TLS, ppArg uintptr, nArg int32) {
	for i := int32(0); i < nArg; i++ {
		mu := *(*uintptr)(unsafe.Pointer(ppArg))
		(*mutex)(unsafe.Pointer(mu)).Unlock()
		ppArg += ptrSize
	}
}

// FunctionImpl describes an [application-defined SQL function]. If Scalar is
// set, it is treated as a scalar function; otherwise, it is treated as an
// aggregate function using MakeAggregate.
//
// [application-defined SQL function]: https://sqlite.org/appfunc.html
type FunctionImpl struct {
	// NArgs is the required number of arguments that the function accepts.
	// If NArgs is negative, then the function is variadic.
	NArgs int32

	// If Deterministic is true, the function must always give the same
	// output when the input parameters are the same. This enables functions
	// to be used in additional contexts like the WHERE clause of partial
	// indexes and enables additional optimizations.
	//
	// See https://sqlite.org/c3ref/c_deterministic.html#sqlitedeterministic
	// for more details.
	Deterministic bool

	// Scalar is called when a scalar function is invoked in SQL. The
	// argument Values are not valid past the return of the function.
	Scalar func(ctx *FunctionContext, args []driver.Value) (driver.Value, error)

	// MakeAggregate is called at the beginning of each evaluation of an
	// aggregate function.
	MakeAggregate func(ctx FunctionContext) (AggregateFunction, error)

	// VolatileArgs is an opt-in performance flag that eliminates the per-call
	// allocation of string and []byte argument bodies. When true, the driver
	// passes argument strings and byte slices as zero-copy views into
	// SQLite-owned memory instead of Go-allocated copies.
	//
	// Setting this is unsafe unless the user-provided Scalar / Step /
	// WindowInverse callbacks treat string and []byte arguments as strictly
	// transient: they must not be retained past the return of the call,
	// neither directly (storing the slice or string in a struct field, map,
	// channel, or outer-scope variable) nor indirectly (passing them to
	// something that captures them, including most concurrency primitives).
	//
	// Retaining a volatile argument produces silent data corruption: SQLite
	// reuses the underlying buffer for the next row, so on a later read every
	// retained value will appear to hold the contents of the most recent row.
	// The Go race detector cannot catch this because UDF execution is
	// sequential on a single goroutine; the corruption is deterministic and
	// invisible to -race.
	//
	// As a guard against accidental capture, callbacks that must retain
	// values across rows should copy:
	//
	//	saved := append([]byte(nil), args[0].([]byte)...) // []byte
	//	saved := string(append([]byte(nil), args[0].(string)...)) // string, no aliasing
	//
	// When in doubt, leave VolatileArgs at its default (false) — the driver
	// already pools the argument-slice header (issue #226), so the per-row
	// overhead with VolatileArgs=false is one make([]byte) per BLOB column
	// and one libc.GoString per TEXT column, not a fresh slice header.
	//
	// Similarly, do not re-enter SQLite on the same connection while a
	// volatile argument is in scope. A nested Query/Exec on the same conn
	// can cause SQLite to reuse the underlying value buffers, so a volatile
	// string or []byte read before the nested call may alias different
	// bytes after it returns.
	//
	// VolatileArgs has no effect on integer, float, time, or NULL arguments.
	VolatileArgs bool
}

// An AggregateFunction is an invocation of an aggregate or window function. See
// the documentation for [aggregate function callbacks] and [application-defined
// window functions] for an overview.
//
// [aggregate function callbacks]: https://www.sqlite.org/appfunc.html#the_aggregate_function_callbacks
// [application-defined window functions]: https://www.sqlite.org/windowfunctions.html#user_defined_aggregate_window_functions
type AggregateFunction interface {
	// Step is called for each row of an aggregate function's SQL
	// invocation. The argument Values are not valid past the return of the
	// function. When the aggregate was registered with
	// [FunctionImpl.VolatileArgs] set to true, string and []byte arguments
	// in rowArgs are zero-copy views into SQLite-owned memory and retaining
	// them produces silent data corruption — see [FunctionImpl.VolatileArgs]
	// for the full safety contract.
	Step(ctx *FunctionContext, rowArgs []driver.Value) error

	// WindowInverse is called to remove the oldest presently aggregated
	// result of Step from the current window. The arguments are those
	// passed to Step for the row being removed. The argument Values are not
	// valid past the return of the function. The same
	// [FunctionImpl.VolatileArgs] caveat applies as for Step.
	WindowInverse(ctx *FunctionContext, rowArgs []driver.Value) error

	// WindowValue is called to get the current value of an aggregate
	// function. This is used to return the final value of the function,
	// whether it is used as a window function or not.
	WindowValue(ctx *FunctionContext) (driver.Value, error)

	// Final is called after all of the aggregate function's input rows have
	// been stepped through. No other methods will be called on the
	// AggregateFunction after calling Final. WindowValue returns the value
	// from the function.
	Final(ctx *FunctionContext)
}

type collation struct {
	zName uintptr
	pApp  uintptr
	enc   int32
}

// RegisterCollationUtf8 makes a Go function available as a collation named zName.
// impl receives two UTF-8 strings: left and right.
// The result needs to be:
//
// - 0 if left == right
// - 1 if left < right
// - +1 if left > right
//
// impl must always return the same result given the same inputs.
// Additionally, it must have the following properties for all strings A, B and C:
// - if A==B, then B==A
// - if A==B and B==C, then A==C
// - if A<B, then B>A
// - if A<B and B<C, then A<C.
//
// The new collation will be available to all new connections the driver
// registered as "sqlite" opens after executing RegisterCollationUtf8.
func RegisterCollationUtf8(
	zName string,
	impl func(left, right string) int,
) error {
	return d.registerCollation(zName, impl, sqlite3.SQLITE_UTF8)
}

// MustRegisterCollationUtf8 is like RegisterCollationUtf8 but panics on error.
func MustRegisterCollationUtf8(
	zName string,
	impl func(left, right string) int,
) {
	if err := RegisterCollationUtf8(zName, impl); err != nil {
		panic(err)
	}
}

func (d *Driver) registerCollation(
	zName string,
	impl func(left, right string) int,
	enc int32,
) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.collations == nil {
		d.collations = map[string]*collation{}
	}
	if _, ok := d.collations[zName]; ok {
		return fmt.Errorf("a collation %q is already registered", zName)
	}

	// dont free, collations registered on the driver live as long as the program
	name, err := libc.CString(zName)
	if err != nil {
		return err
	}

	xCollations.mu.Lock()
	id := xCollations.ids.next()
	xCollations.m[id] = impl
	xCollations.mu.Unlock()

	d.collations[zName] = &collation{
		zName: name,
		pApp:  id,
		enc:   enc,
	}

	return nil
}

type ExecQuerierContext interface {
	driver.ExecerContext
	driver.QueryerContext
}

type HookRegisterer interface {
	RegisterPreUpdateHook(PreUpdateHookFn)
	RegisterCommitHook(CommitHookFn)
	RegisterRollbackHook(RollbackHookFn)
}

// ConnectionHookFn function type for a connection hook on the Driver. Connection
// hooks are called after the connection has been set up.
type ConnectionHookFn func(
	conn ExecQuerierContext,
	dsn string,
) error

// FunctionContext represents the context user defined functions execute in.
// Fields and/or methods of this type may get addedd in the future.
type FunctionContext struct {
	tls *libc.TLS
	ctx uintptr
}

const sqliteValPtrSize = unsafe.Sizeof(&sqlite3.Sqlite3_value{})

// RegisterFunction registers a function named zFuncName with nArg arguments.
// Passing -1 for nArg indicates the function is variadic. The FunctionImpl
// determines whether the function is deterministic or not, and whether it is a
// scalar function (when Scalar is defined) or an aggregate function (when
// Scalar is not defined and MakeAggregate is defined).
//
// The new function will be available to all new connections the driver
// registered as "sqlite" opens after executing RegisterFunction.
func RegisterFunction(
	zFuncName string,
	impl *FunctionImpl,
) error {
	return d.registerFunction(zFuncName, impl)
}

// MustRegisterFunction is like RegisterFunction but panics on error.
func MustRegisterFunction(
	zFuncName string,
	impl *FunctionImpl,
) {
	if err := RegisterFunction(zFuncName, impl); err != nil {
		panic(err)
	}
}

// RegisterScalarFunction registers a scalar function named zFuncName with nArg
// arguments. Passing -1 for nArg indicates the function is variadic.
//
// The new function will be available to all new connections the driver
// registered as "sqlite" opens after executing RegisterScalarFunction.
func RegisterScalarFunction(
	zFuncName string,
	nArg int32,
	xFunc func(ctx *FunctionContext, args []driver.Value) (driver.Value, error),
) (err error) {
	if dmesgs {
		defer func() {
			dmesg("zFuncName %q, nArg %v, xFunc %p: err %v", zFuncName, nArg, xFunc, err)
		}()
	}
	return d.registerFunction(zFuncName, &FunctionImpl{NArgs: nArg, Scalar: xFunc, Deterministic: false})
}

// MustRegisterScalarFunction is like RegisterScalarFunction but panics on
// error.
func MustRegisterScalarFunction(
	zFuncName string,
	nArg int32,
	xFunc func(ctx *FunctionContext, args []driver.Value) (driver.Value, error),
) {
	if dmesgs {
		dmesg("zFuncName %q, nArg %v, xFunc %p", zFuncName, nArg, xFunc)
	}
	if err := RegisterScalarFunction(zFuncName, nArg, xFunc); err != nil {
		panic(err)
	}
}

// MustRegisterDeterministicScalarFunction is like
// RegisterDeterministicScalarFunction but panics on error.
func MustRegisterDeterministicScalarFunction(
	zFuncName string,
	nArg int32,
	xFunc func(ctx *FunctionContext, args []driver.Value) (driver.Value, error),
) {
	if dmesgs {
		dmesg("zFuncName %q, nArg %v, xFunc %p", zFuncName, nArg, xFunc)
	}
	if err := RegisterDeterministicScalarFunction(zFuncName, nArg, xFunc); err != nil {
		panic(err)
	}
}

// RegisterDeterministicScalarFunction registers a deterministic scalar
// function named zFuncName with nArg arguments. Passing -1 for nArg indicates
// the function is variadic. A deterministic function means that the function
// always gives the same output when the input parameters are the same.
//
// The new function will be available to all new connections the driver
// registered as "sqlite" opens after executing
// RegisterDeterministicScalarFunction.
func RegisterDeterministicScalarFunction(
	zFuncName string,
	nArg int32,
	xFunc func(ctx *FunctionContext, args []driver.Value) (driver.Value, error),
) (err error) {
	if dmesgs {
		defer func() {
			dmesg("zFuncName %q, nArg %v, xFunc %p: err %v", zFuncName, nArg, xFunc, err)
		}()
	}
	return d.registerFunction(zFuncName, &FunctionImpl{NArgs: nArg, Scalar: xFunc, Deterministic: true})
}

func (d *Driver) registerFunction(
	zFuncName string,
	impl *FunctionImpl,
) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.udfs == nil {
		d.udfs = map[string]*userDefinedFunction{}
	}
	if _, ok := d.udfs[zFuncName]; ok {
		return fmt.Errorf("a function named %q is already registered", zFuncName)
	}

	// dont free, functions registered on the driver live as long as the program
	name, err := libc.CString(zFuncName)
	if err != nil {
		return err
	}

	var textrep int32 = sqlite3.SQLITE_UTF8

	if impl.Deterministic {
		textrep |= sqlite3.SQLITE_DETERMINISTIC
	}

	udf := &userDefinedFunction{
		zFuncName: name,
		nArg:      impl.NArgs,
		eTextRep:  textrep,
	}

	if impl.Scalar != nil {
		xFuncs.mu.Lock()
		id := xFuncs.ids.next()
		xFuncs.m[id] = xFuncEntry{fn: impl.Scalar, volatile: impl.VolatileArgs}
		xFuncs.mu.Unlock()

		udf.scalar = true
		udf.pApp = id
	} else {
		xAggregateFactories.mu.Lock()
		id := xAggregateFactories.ids.next()
		xAggregateFactories.m[id] = xAggregateFactoryEntry{factory: impl.MakeAggregate, volatile: impl.VolatileArgs}
		xAggregateFactories.mu.Unlock()

		udf.pApp = id
	}

	d.udfs[zFuncName] = udf

	return nil
}

// RegisterConnectionHook registers a function to be called after each
// connection the driver registered as "sqlite" opens. This is called after all
// the connection has been set up. Use [Driver.RegisterConnectionHook] to hook
// the connections of a caller-constructed Driver instead.
func RegisterConnectionHook(fn ConnectionHookFn) {
	d.RegisterConnectionHook(fn)
}

func origin(skip int) string {
	pc, fn, fl, _ := runtime.Caller(skip)
	f := runtime.FuncForPC(pc)
	var fns string
	if f != nil {
		fns = f.Name()
		if x := strings.LastIndex(fns, "."); x > 0 {
			fns = fns[x+1:]
		}
	}
	return fmt.Sprintf("%s:%d:%s", fn, fl, fns)
}

func errorResultFunction(tls *libc.TLS, ctx uintptr) func(error) {
	return func(res error) {
		errmsg, cerr := libc.CString(res.Error())
		if cerr != nil {
			panic(cerr)
		}
		defer libc.Xfree(tls, errmsg)
		sqlite3.Xsqlite3_result_error(tls, ctx, errmsg, -1)
		sqlite3.Xsqlite3_result_error_code(tls, ctx, sqlite3.SQLITE_ERROR)
	}
}

// udfArgsPool reuses []driver.Value slices passed to user-defined functions.
// The driver's contract (documented on FunctionImpl.Scalar and
// AggregateFunction.Step/WindowInverse) states that the args values are not
// valid past the return of the user function, which makes the slice itself
// safe to reuse. See https://gitlab.com/cznic/sqlite/-/issues/226.
var udfArgsPool = sync.Pool{
	New: func() any {
		s := make([]driver.Value, 0, 8)
		return &s
	},
}

// acquireUDFArgs returns a pooled *[]driver.Value with len == n. The caller
// must invoke releaseUDFArgs after the user function has returned.
func acquireUDFArgs(n int) *[]driver.Value {
	sp := udfArgsPool.Get().(*[]driver.Value)
	if cap(*sp) < n {
		*sp = make([]driver.Value, n)
	} else {
		*sp = (*sp)[:n]
	}
	return sp
}

// releaseUDFArgs returns the slice to the pool after clearing each entry so
// any heap references held in the previous invocation can be reclaimed.
func releaseUDFArgs(sp *[]driver.Value) {
	s := *sp
	for i := range s {
		s[i] = nil
	}
	*sp = s[:0]
	udfArgsPool.Put(sp)
}

// functionArgs prepares a []driver.Value for one user-function invocation.
// The returned slice is owned by the driver and must be released via
// releaseUDFArgs once the user function returns.
//
// When volatile is true, SQLITE_TEXT and SQLITE_BLOB arguments are returned as
// zero-copy views into SQLite-owned memory (see [FunctionImpl.VolatileArgs]
// for the user-facing safety contract). When false (the default for all
// existing call sites), text and blob payloads are copied into Go-owned
// memory and stay valid for the lifetime of the slice.
func functionArgs(tls *libc.TLS, argc int32, argv uintptr, volatile bool) *[]driver.Value {
	sp := acquireUDFArgs(int(argc))
	args := *sp
	for i := int32(0); i < argc; i++ {
		valPtr := *(*uintptr)(unsafe.Pointer(argv + uintptr(i)*sqliteValPtrSize))

		switch valType := sqlite3.Xsqlite3_value_type(tls, valPtr); valType {
		case sqlite3.SQLITE_TEXT:
			if volatile {
				p := sqlite3.Xsqlite3_value_text(tls, valPtr)
				n := sqlite3.Xsqlite3_value_bytes(tls, valPtr)
				if p == 0 || n == 0 {
					args[i] = ""
				} else {
					args[i] = unsafe.String((*byte)(unsafe.Pointer(p)), int(n))
				}
			} else {
				args[i] = libc.GoString(sqlite3.Xsqlite3_value_text(tls, valPtr))
			}
		case sqlite3.SQLITE_INTEGER:
			args[i] = sqlite3.Xsqlite3_value_int64(tls, valPtr)
		case sqlite3.SQLITE_FLOAT:
			args[i] = sqlite3.Xsqlite3_value_double(tls, valPtr)
		case sqlite3.SQLITE_NULL:
			args[i] = nil
		case sqlite3.SQLITE_BLOB:
			size := sqlite3.Xsqlite3_value_bytes(tls, valPtr)
			blobPtr := sqlite3.Xsqlite3_value_blob(tls, valPtr)
			if volatile {
				if blobPtr == 0 || size == 0 {
					args[i] = make([]byte, 0)
				} else {
					args[i] = unsafe.Slice((*byte)(unsafe.Pointer(blobPtr)), int(size))
				}
			} else {
				v := make([]byte, size)
				if size != 0 {
					copy(v, (*libc.RawMem)(unsafe.Pointer(blobPtr))[:size:size])
				}
				args[i] = v
			}
		default:
			panic(fmt.Sprintf("unexpected argument type %q passed by sqlite", valType))
		}
	}

	return sp
}

func functionReturnValue(tls *libc.TLS, ctx uintptr, res driver.Value) error {
	switch resTyped := res.(type) {
	case nil:
		sqlite3.Xsqlite3_result_null(tls, ctx)
	case int64:
		sqlite3.Xsqlite3_result_int64(tls, ctx, resTyped)
	case float64:
		sqlite3.Xsqlite3_result_double(tls, ctx, resTyped)
	case bool:
		sqlite3.Xsqlite3_result_int(tls, ctx, libc.Bool32(resTyped))
	case time.Time:
		sqlite3.Xsqlite3_result_int64(tls, ctx, resTyped.Unix())
	case string:
		size := int32(len(resTyped))
		cstr, err := libc.CString(resTyped)
		if err != nil {
			panic(err)
		}
		defer libc.Xfree(tls, cstr)
		sqlite3.Xsqlite3_result_text(tls, ctx, cstr, size, sqlite3.SQLITE_TRANSIENT)
	case []byte:
		size := int32(len(resTyped))
		if size == 0 {
			sqlite3.Xsqlite3_result_zeroblob(tls, ctx, 0)
			return nil
		}
		p := libc.Xmalloc(tls, types.Size_t(size))
		if p == 0 {
			panic(fmt.Sprintf("unable to allocate space for blob: %d", size))
		}
		defer libc.Xfree(tls, p)
		copy((*libc.RawMem)(unsafe.Pointer(p))[:size:size], resTyped)

		sqlite3.Xsqlite3_result_blob(tls, ctx, p, size, sqlite3.SQLITE_TRANSIENT)
	default:
		return fmt.Errorf("function did not return a valid driver.Value: %T", resTyped)
	}

	return nil
}

// The below is all taken from zombiezen.com/go/sqlite. Aggregate functions need
// to maintain state (for instance, the count of values seen so far). We give
// each aggregate function an ID, generated by idGen, and put that in the pApp
// argument to sqlite3_create_function. We track this on the Go side in
// xAggregateFactories.
//
// When (if) the function is called is called by a query, we call the
// MakeAggregate factory function to set it up, and track that in
// xAggregateContext, retrieving it via sqlite3_aggregate_context.
//
// We also need to ensure that, for both aggregate and scalar functions, the
// function pointer we pass to SQLite meets certain rules on the Go side, so
// that the pointer remains valid.
var (
	xFuncs = struct {
		mu  sync.RWMutex
		m   map[uintptr]xFuncEntry
		ids idGen
	}{
		m: make(map[uintptr]xFuncEntry),
	}

	xAggregateFactories = struct {
		mu  sync.RWMutex
		m   map[uintptr]xAggregateFactoryEntry
		ids idGen
	}{
		m: make(map[uintptr]xAggregateFactoryEntry),
	}

	xAggregateContext = struct {
		mu  sync.RWMutex
		m   map[uintptr]xAggregateContextEntry
		ids idGen
	}{
		m: make(map[uintptr]xAggregateContextEntry),
	}

	xCollations = struct {
		mu  sync.RWMutex
		m   map[uintptr]func(string, string) int
		ids idGen
	}{
		m: make(map[uintptr]func(string, string) int),
	}
)

// xFuncEntry pairs a registered scalar function with the VolatileArgs flag
// captured at registration time. Bundled so trampolines can decide whether to
// pass zero-copy or copied argument values without a second map lookup.
type xFuncEntry struct {
	fn       func(*FunctionContext, []driver.Value) (driver.Value, error)
	volatile bool
}

// xAggregateFactoryEntry pairs a registered aggregate factory with its
// VolatileArgs flag, for the same reason as xFuncEntry.
type xAggregateFactoryEntry struct {
	factory  func(FunctionContext) (AggregateFunction, error)
	volatile bool
}

// xAggregateContextEntry holds the AggregateFunction instance for one
// in-flight aggregate evaluation together with the VolatileArgs flag inherited
// from its factory registration. Caching the flag here avoids a second
// xAggregateFactories lookup in the Step / WindowInverse trampolines.
type xAggregateContextEntry struct {
	fn       AggregateFunction
	volatile bool
}

type idGen struct {
	bitset []uint64
}

func (gen *idGen) next() uintptr {
	base := uintptr(1)
	for i := 0; i < len(gen.bitset); i, base = i+1, base+64 {
		b := gen.bitset[i]
		if b != 1<<64-1 {
			n := uintptr(bits.TrailingZeros64(^b))
			gen.bitset[i] |= 1 << n
			return base + n
		}
	}
	gen.bitset = append(gen.bitset, 1)
	return base
}

func (gen *idGen) reclaim(id uintptr) {
	bit := id - 1
	gen.bitset[bit/64] &^= 1 << (bit % 64)
}

func makeAggregate(tls *libc.TLS, ctx uintptr) (AggregateFunction, bool, uintptr) {
	goCtx := FunctionContext{tls: tls, ctx: ctx}
	aggCtx := (*uintptr)(unsafe.Pointer(sqlite3.Xsqlite3_aggregate_context(tls, ctx, int32(ptrSize))))
	setErrorResult := errorResultFunction(tls, ctx)
	if aggCtx == nil {
		setErrorResult(errors.New("insufficient memory for aggregate"))
		return nil, false, 0
	}
	if *aggCtx != 0 {
		// Already created.
		xAggregateContext.mu.RLock()
		entry := xAggregateContext.m[*aggCtx]
		xAggregateContext.mu.RUnlock()
		return entry.fn, entry.volatile, *aggCtx
	}

	factoryID := sqlite3.Xsqlite3_user_data(tls, ctx)
	xAggregateFactories.mu.RLock()
	factoryEntry := xAggregateFactories.m[factoryID]
	xAggregateFactories.mu.RUnlock()

	f, err := factoryEntry.factory(goCtx)
	if err != nil {
		setErrorResult(err)
		return nil, false, 0
	}
	if f == nil {
		setErrorResult(errors.New("MakeAggregate function returned nil"))
		return nil, false, 0
	}

	xAggregateContext.mu.Lock()
	*aggCtx = xAggregateContext.ids.next()
	xAggregateContext.m[*aggCtx] = xAggregateContextEntry{fn: f, volatile: factoryEntry.volatile}
	xAggregateContext.mu.Unlock()
	return f, factoryEntry.volatile, *aggCtx
}

// cFuncPointer converts a function defined by a function declaration to a C pointer.
// The result of using cFuncPointer on closures is undefined.
func cFuncPointer[T any](f T) uintptr {
	// This assumes the memory representation described in https://golang.org/s/go11func.
	//
	// cFuncPointer does its conversion by doing the following in order:
	// 1) Create a Go struct containing a pointer to a pointer to
	//    the function. It is assumed that the pointer to the function will be
	//    stored in the read-only data section and thus will not move.
	// 2) Convert the pointer to the Go struct to a pointer to uintptr through
	//    unsafe.Pointer. This is permitted via Rule #1 of unsafe.Pointer.
	// 3) Dereference the pointer to uintptr to obtain the function value as a
	//    uintptr. This is safe as long as function values are passed as pointers.
	return *(*uintptr)(unsafe.Pointer(&struct{ f T }{f}))
}

func funcTrampoline(tls *libc.TLS, ctx uintptr, argc int32, argv uintptr) {
	id := sqlite3.Xsqlite3_user_data(tls, ctx)
	xFuncs.mu.RLock()
	entry := xFuncs.m[id]
	xFuncs.mu.RUnlock()

	setErrorResult := errorResultFunction(tls, ctx)
	sp := functionArgs(tls, argc, argv, entry.volatile)
	defer releaseUDFArgs(sp)
	res, err := entry.fn(&FunctionContext{}, *sp)

	if err != nil {
		setErrorResult(err)
		return
	}

	err = functionReturnValue(tls, ctx, res)
	if err != nil {
		setErrorResult(err)
	}
}

// sqlite3AllocCString allocates a NUL-terminated copy of s using SQLite's
// memory allocator (sqlite3_malloc). The caller must arrange for SQLite to
// free the returned pointer via sqlite3_free.
func sqlite3AllocCString(tls *libc.TLS, s string) uintptr {
	n := len(s) + 1
	p := sqlite3.Xsqlite3_malloc(tls, int32(n))
	if p == 0 {
		return 0
	}
	mem := (*libc.RawMem)(unsafe.Pointer(p))[:n:n]
	copy(mem, []byte(s))
	mem[n-1] = 0
	return p
}

func stepTrampoline(tls *libc.TLS, ctx uintptr, argc int32, argv uintptr) {
	impl, volatile, _ := makeAggregate(tls, ctx)
	if impl == nil {
		return
	}

	setErrorResult := errorResultFunction(tls, ctx)
	sp := functionArgs(tls, argc, argv, volatile)
	defer releaseUDFArgs(sp)
	err := impl.Step(&FunctionContext{}, *sp)
	if err != nil {
		setErrorResult(err)
	}
}

func inverseTrampoline(tls *libc.TLS, ctx uintptr, argc int32, argv uintptr) {
	impl, volatile, _ := makeAggregate(tls, ctx)
	if impl == nil {
		return
	}

	setErrorResult := errorResultFunction(tls, ctx)
	sp := functionArgs(tls, argc, argv, volatile)
	defer releaseUDFArgs(sp)
	err := impl.WindowInverse(&FunctionContext{}, *sp)
	if err != nil {
		setErrorResult(err)
	}
}

func valueTrampoline(tls *libc.TLS, ctx uintptr) {
	impl, _, _ := makeAggregate(tls, ctx)
	if impl == nil {
		return
	}

	setErrorResult := errorResultFunction(tls, ctx)
	res, err := impl.WindowValue(&FunctionContext{})
	if err != nil {
		setErrorResult(err)
	} else {
		err = functionReturnValue(tls, ctx, res)
		if err != nil {
			setErrorResult(err)
		}
	}
}

func finalTrampoline(tls *libc.TLS, ctx uintptr) {
	impl, _, id := makeAggregate(tls, ctx)
	if impl == nil {
		return
	}

	setErrorResult := errorResultFunction(tls, ctx)
	res, err := impl.WindowValue(&FunctionContext{})
	if err != nil {
		setErrorResult(err)
	} else {
		err = functionReturnValue(tls, ctx, res)
		if err != nil {
			setErrorResult(err)
		}
	}
	impl.Final(&FunctionContext{})

	xAggregateContext.mu.Lock()
	defer xAggregateContext.mu.Unlock()
	delete(xAggregateContext.m, id)
	xAggregateContext.ids.reclaim(id)
}

func collationTrampoline(tls *libc.TLS, pApp uintptr, nLeft int32, zLeft uintptr, nRight int32, zRight uintptr) int32 {
	xCollations.mu.RLock()
	xCollation := xCollations.m[pApp]
	xCollations.mu.RUnlock()

	left := string(libc.GoBytes(zLeft, int(nLeft)))
	right := string(libc.GoBytes(zRight, int(nRight)))

	// res is of type int, which can be 64-bit wide
	// Since we just need to know if the value is positive, negative, or zero, we can ensure it's -1, 0, +1
	res := xCollation(left, right)
	switch {
	case res < 0:
		return -1
	case res == 0:
		return 0
	case res > 0:
		return 1
	default:
		// Should never hit here, make the compiler happy
		return 0
	}
}

// Limit calls sqlite3_limit, see the docs at
// https://www.sqlite.org/c3ref/limit.html for details.
//
// To get a sql.Conn from a *sql.DB, use (*sql.DB).Conn().  Limits are bound to
// the particular instance of 'c', so getting a new connection only to pass it
// to Limit is possibly not useful above querying what are the various
// configured default values.
func Limit(c *sql.Conn, id int, newVal int) (r int, err error) {
	err = c.Raw(func(driverConn any) error {
		switch dc := driverConn.(type) {
		case *conn:
			r = dc.limit(id, newVal)
			return nil
		default:
			return fmt.Errorf("unexpected driverConn type: %T", driverConn)
		}
	})
	return r, err

}

// ColumnInfo describes one output column of a prepared SQL statement.
//
// See https://www.sqlite.org/c3ref/column_database_name.html for more details.
type ColumnInfo struct {
	// Name is the column's name, from sqlite3_column_name.
	Name string
	// DeclType is the declared column type, from sqlite3_column_decltype.
	DeclType string
	// DatabaseName is the name of the source database, from sqlite3_column_database_name.
	// Empty if the column does not resolve to an unambiguous reference to a single database column,
	// such as expressions, function calls, or constants.
	DatabaseName string
	// TableName is the name of the source table, from sqlite3_column_table_name.
	// Empty if the column does not resolve to an unambiguous reference to a single database column,
	// such as expressions, function calls, or constants.
	TableName string
	// OriginName is the name of the source column name, from sqlite3_column_origin_name.
	// Empty if the column does not resolve to an unambiguous reference to a single database column,
	// such as expressions, function calls, or constants.
	OriginName string
}
