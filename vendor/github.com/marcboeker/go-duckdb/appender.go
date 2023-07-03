package duckdb

/*
#include <duckdb.h>
*/
import "C"

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"time"
	"unsafe"
)

// Appender holds the duckdb appender. It allows to load bulk data into a DuckDB database.
type Appender struct {
	c        *conn
	schema   string
	table    string
	appender *C.duckdb_appender
	closed   bool
}

// NewAppenderFromConn returns a new Appender from a DuckDB driver connection.
func NewAppenderFromConn(driverConn driver.Conn, schema string, table string) (*Appender, error) {
	dbConn, ok := driverConn.(*conn)
	if !ok {
		return nil, fmt.Errorf("not a duckdb driver connection")
	}

	if dbConn.closed {
		panic("database/sql/driver: misuse of duckdb driver: Appender after Close")
	}

	var schemastr *(C.char)
	if schema != "" {
		schemastr = C.CString(schema)
		defer C.free(unsafe.Pointer(schemastr))
	}

	tablestr := C.CString(table)
	defer C.free(unsafe.Pointer(tablestr))

	var a C.duckdb_appender
	if state := C.duckdb_appender_create(*dbConn.con, schemastr, tablestr, &a); state == C.DuckDBError {
		return nil, fmt.Errorf("can't create appender")
	}

	return &Appender{c: dbConn, schema: schema, table: table, appender: &a}, nil
}

// Error returns the last duckdb appender error.
func (a *Appender) Error() error {
	dbErr := C.GoString(C.duckdb_appender_error(*a.appender))
	return errors.New(dbErr)
}

// Flush the appender to the underlying table and clear the internal cache.
func (a *Appender) Flush() error {
	if state := C.duckdb_appender_flush(*a.appender); state == C.DuckDBError {
		dbErr := C.GoString(C.duckdb_appender_error(*a.appender))
		return errors.New(dbErr)
	}
	return nil
}

// Closes closes the appender.
func (a *Appender) Close() error {
	if a.closed {
		panic("database/sql/driver: misuse of duckdb driver: double Close of Appender")
	}

	a.closed = true

	if state := C.duckdb_appender_destroy(a.appender); state == C.DuckDBError {
		dbErr := C.GoString(C.duckdb_appender_error(*a.appender))
		return errors.New(dbErr)
	}
	return nil
}

// AppendRow loads a row of values into the appender. The values are provided as separate arguments.
func (a *Appender) AppendRow(args ...driver.Value) error {
	return a.AppendRowArray(args)
}

// AppendRowArray loads a row of values into the appender. The values are provided as an array.
func (a *Appender) AppendRowArray(args []driver.Value) error {
	if a.closed {
		panic("database/sql/driver: misuse of duckdb driver: use of closed Appender")
	}

	for i, v := range args {
		if v == nil {
			if rv := C.duckdb_append_null(*a.appender); rv == C.DuckDBError {
				return fmt.Errorf("couldn't append parameter %d", i)
			}
			continue
		}

		var rv C.duckdb_state
		switch v := v.(type) {
		case uint8:
			rv = C.duckdb_append_uint8(*a.appender, C.uint8_t(v))
		case int8:
			rv = C.duckdb_append_int8(*a.appender, C.int8_t(v))
		case uint16:
			rv = C.duckdb_append_uint16(*a.appender, C.uint16_t(v))
		case int16:
			rv = C.duckdb_append_int16(*a.appender, C.int16_t(v))
		case uint32:
			rv = C.duckdb_append_uint32(*a.appender, C.uint32_t(v))
		case int32:
			rv = C.duckdb_append_int32(*a.appender, C.int32_t(v))
		case uint64:
			rv = C.duckdb_append_uint64(*a.appender, C.uint64_t(v))
		case int64:
			rv = C.duckdb_append_int64(*a.appender, C.int64_t(v))
		case uint:
			rv = C.duckdb_append_uint64(*a.appender, C.uint64_t(v))
		case int:
			rv = C.duckdb_append_int64(*a.appender, C.int64_t(v))
		case float32:
			rv = C.duckdb_append_float(*a.appender, C.float(v))
		case float64:
			rv = C.duckdb_append_double(*a.appender, C.double(v))
		case bool:
			rv = C.duckdb_append_bool(*a.appender, C.bool(v))
		case []byte:
			rv = C.duckdb_append_blob(*a.appender, unsafe.Pointer(&v[0]), C.uint64_t(len(v)))
		case string:
			str := C.CString(v)
			rv = C.duckdb_append_varchar(*a.appender, str)
			C.free(unsafe.Pointer(str))
		case time.Time:
			var dt C.duckdb_timestamp
			dt.micros = C.int64_t(v.UTC().UnixMicro())
			rv = C.duckdb_append_timestamp(*a.appender, dt)

		default:
			return fmt.Errorf("couldn't append unsupported parameter %d (type %T)", i, v)
		}
		if rv == C.DuckDBError {
			dbErr := C.GoString(C.duckdb_appender_error(*a.appender))
			return fmt.Errorf("couldn't append parameter %d (type %T): %s", i, v, dbErr)
		}
	}

	if state := C.duckdb_appender_end_row(*a.appender); state == C.DuckDBError {
		dbErr := C.GoString(C.duckdb_appender_error(*a.appender))
		return errors.New(dbErr)
	}

	return nil
}

var errCouldNotAppend = errors.New("could not append parameter")
