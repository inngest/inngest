package duckdb

/*
#include <duckdb.h>
*/
import "C"

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"math/big"
	"time"
	"unsafe"
)

type stmt struct {
	c                *conn
	stmt             *C.duckdb_prepared_statement
	closeOnRowsClose bool
	closed           bool
	rows             bool
}

func (s *stmt) Close() error {
	if s.rows {
		panic("database/sql/driver: misuse of duckdb driver: Close with active Rows")
	}
	if s.closed {
		panic("database/sql/driver: misuse of duckdb driver: double Close of Stmt")
	}

	s.closed = true
	C.duckdb_destroy_prepare(s.stmt)
	return nil
}

func (s *stmt) NumInput() int {
	if s.closed {
		panic("database/sql/driver: misuse of duckdb driver: NumInput after Close")
	}
	paramCount := C.duckdb_nparams(*s.stmt)
	return int(paramCount)
}

func (s *stmt) start(args []driver.NamedValue) error {
	if s.NumInput() != len(args) {
		return fmt.Errorf("incorrect argument count for command: have %d want %d", len(args), s.NumInput())
	}

	for i, v := range args {
		switch v := v.Value.(type) {
		case bool:
			if rv := C.duckdb_bind_boolean(*s.stmt, C.idx_t(i+1), C.bool(v)); rv == C.DuckDBError {
				return errCouldNotBind
			}
		case int8:
			if rv := C.duckdb_bind_int8(*s.stmt, C.idx_t(i+1), C.int8_t(v)); rv == C.DuckDBError {
				return errCouldNotBind
			}
		case int16:
			if rv := C.duckdb_bind_int16(*s.stmt, C.idx_t(i+1), C.int16_t(v)); rv == C.DuckDBError {
				return errCouldNotBind
			}
		case int32:
			if rv := C.duckdb_bind_int32(*s.stmt, C.idx_t(i+1), C.int32_t(v)); rv == C.DuckDBError {
				return errCouldNotBind
			}
		case int64:
			if rv := C.duckdb_bind_int64(*s.stmt, C.idx_t(i+1), C.int64_t(v)); rv == C.DuckDBError {
				return errCouldNotBind
			}
		case int:
			if rv := C.duckdb_bind_int64(*s.stmt, C.idx_t(i+1), C.int64_t(v)); rv == C.DuckDBError {
				return errCouldNotBind
			}
		case *big.Int:
			val, err := hugeIntFromNative(v)
			if err != nil {
				return err
			}
			if rv := C.duckdb_bind_hugeint(*s.stmt, C.idx_t(i+1), val); rv == C.DuckDBError {
				return errCouldNotBind
			}
		case uint8:
			if rv := C.duckdb_bind_uint8(*s.stmt, C.idx_t(i+1), C.uchar(v)); rv == C.DuckDBError {
				return errCouldNotBind
			}
		case uint16:
			if rv := C.duckdb_bind_uint16(*s.stmt, C.idx_t(i+1), C.uint16_t(v)); rv == C.DuckDBError {
				return errCouldNotBind
			}
		case uint32:
			if rv := C.duckdb_bind_uint32(*s.stmt, C.idx_t(i+1), C.uint32_t(v)); rv == C.DuckDBError {
				return errCouldNotBind
			}
		case uint64:
			if rv := C.duckdb_bind_uint64(*s.stmt, C.idx_t(i+1), C.uint64_t(v)); rv == C.DuckDBError {
				return errCouldNotBind
			}
		case float32:
			if rv := C.duckdb_bind_float(*s.stmt, C.idx_t(i+1), C.float(v)); rv == C.DuckDBError {
				return errCouldNotBind
			}
		case float64:
			if rv := C.duckdb_bind_double(*s.stmt, C.idx_t(i+1), C.double(v)); rv == C.DuckDBError {
				return errCouldNotBind
			}
		case string:
			val := C.CString(v)
			if rv := C.duckdb_bind_varchar(*s.stmt, C.idx_t(i+1), val); rv == C.DuckDBError {
				C.free(unsafe.Pointer(val))
				return errCouldNotBind
			}
			C.free(unsafe.Pointer(val))
		case []byte:
			val := C.CBytes(v)
			l := len(v)
			if rv := C.duckdb_bind_blob(*s.stmt, C.idx_t(i+1), val, C.uint64_t(l)); rv == C.DuckDBError {
				C.free(unsafe.Pointer(val))
				return errCouldNotBind
			}
			C.free(unsafe.Pointer(val))
		case time.Time:
			val := C.duckdb_timestamp{
				micros: C.int64_t(v.UTC().UnixMicro()),
			}
			if rv := C.duckdb_bind_timestamp(*s.stmt, C.idx_t(i+1), val); rv == C.DuckDBError {
				return errCouldNotBind
			}
		case Interval:
			val := C.duckdb_interval{
				months: C.int32_t(v.Months),
				days:   C.int32_t(v.Days),
				micros: C.int64_t(v.Micros),
			}
			if rv := C.duckdb_bind_interval(*s.stmt, C.idx_t(i+1), val); rv == C.DuckDBError {
				return errCouldNotBind
			}
		case nil:
			if rv := C.duckdb_bind_null(*s.stmt, C.idx_t(i+1)); rv == C.DuckDBError {
				return errCouldNotBind
			}
		default:
			return driver.ErrSkip
		}
	}

	return nil
}

// Deprecated: Use ExecContext instead.
func (s *stmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.ExecContext(context.Background(), argsToNamedArgs(args))
}

func (s *stmt) ExecContext(ctx context.Context, nargs []driver.NamedValue) (driver.Result, error) {
	res, err := s.execute(ctx, nargs)
	if err != nil {
		return nil, err
	}
	defer C.duckdb_destroy_result(res)

	ra := int64(C.duckdb_value_int64(res, 0, 0))
	return &result{ra}, nil
}

// Deprecated: Use QueryContext instead.
func (s *stmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.QueryContext(context.Background(), argsToNamedArgs(args))
}

func (s *stmt) QueryContext(ctx context.Context, nargs []driver.NamedValue) (driver.Rows, error) {
	res, err := s.execute(ctx, nargs)
	if err != nil {
		return nil, err
	}
	s.rows = true
	return newRowsWithStmt(*res, s), nil
}

// This method executes the query in steps and checks if context is cancelled before executing each step.
// It uses Pending Result Interface C APIs to achieve this. Reference - https://duckdb.org/docs/api/c/api#pending-result-interface
func (s *stmt) execute(ctx context.Context, args []driver.NamedValue) (*C.duckdb_result, error) {
	if s.closed {
		panic("database/sql/driver: misuse of duckdb driver: ExecContext or QueryContext after Close")
	}
	if s.rows {
		panic("database/sql/driver: misuse of duckdb driver: ExecContext or QueryContext with active Rows")
	}

	err := s.start(args)
	if err != nil {
		return nil, err
	}

	var pendingRes C.duckdb_pending_result
	if state := C.duckdb_pending_prepared(*s.stmt, &pendingRes); state == C.DuckDBError {
		dbErr := C.GoString(C.duckdb_pending_error(pendingRes))
		C.duckdb_destroy_pending(&pendingRes)
		return nil, errors.New(dbErr)
	}
	defer C.duckdb_destroy_pending(&pendingRes)

	ready := false
	for !ready {
		select {
		// if context is cancelled or deadline exceeded, don't execute further
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			// continue
		}
		state := C.duckdb_pending_execute_task(pendingRes)
		switch state {
		case C.DUCKDB_PENDING_RESULT_READY:
			// we are done processing the query, now get the results
			ready = true
		case C.DUCKDB_PENDING_ERROR:
			dbErr := C.GoString(C.duckdb_pending_error(pendingRes))
			return nil, errors.New(dbErr)
		case C.DUCKDB_PENDING_RESULT_NOT_READY:
			// we are not done yet, continue to next task
		default:
			panic(fmt.Sprintf("found unknown state while pending execute: %v", state))
		}
	}

	var res C.duckdb_result
	if state := C.duckdb_execute_pending(pendingRes, &res); state == C.DuckDBError {
		dbErr := C.GoString(C.duckdb_result_error(&res))
		C.duckdb_destroy_result(&res)
		return nil, errors.New(dbErr)
	}
	return &res, nil
}

func argsToNamedArgs(values []driver.Value) []driver.NamedValue {
	args := make([]driver.NamedValue, len(values))
	for n, param := range values {
		args[n].Value = param
		args[n].Ordinal = n + 1
	}
	return args
}

var (
	errCouldNotBind = errors.New("could not bind parameter")
)
