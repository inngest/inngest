//go:build darwin || linux

package libduckdb

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// Lib holds a dynamically loaded libduckdb shared library handle and its
// registered C API function pointers (zero cgo).
type Lib struct {
	mu     sync.Mutex
	handle uintptr
	closed bool

	openExt       func(path string, outDB *uintptr, cfg uintptr, outErr **byte) int32
	closeDB       func(db *uintptr)
	createConfig  func(outCfg *uintptr) int32
	setConfig     func(cfg uintptr, name, option string) int32
	destroyConfig func(cfg *uintptr)

	connect    func(db uintptr, outConn *uintptr) int32
	disconnect func(conn *uintptr)
	interrupt  func(conn uintptr)

	query         func(conn uintptr, sql string, outResult *cResult) int32
	resultError   func(res *cResult) *byte
	destroyResult func(res *cResult)
	columnCount   func(res *cResult) uint64
	columnName    func(res *cResult, col uint64) *byte
	columnType    func(res *cResult, col uint64) int32

	fetchChunk              func(res cResult) uintptr
	dataChunkGetSize        func(chunk uintptr) uint64
	dataChunkGetColumnCount func(chunk uintptr) uint64
	dataChunkGetVector      func(chunk uintptr, col uint64) uintptr
	destroyDataChunk        func(chunk *uintptr)

	vectorGetData       func(vec uintptr) unsafe.Pointer
	vectorGetValidity   func(vec uintptr) *uint64
	vectorGetColumnType func(vec uintptr) uintptr

	destroyLogicalType func(lt *uintptr)
	decimalScale       func(lt uintptr) uint8
	decimalInternal    func(lt uintptr) int32

	free           func(ptr unsafe.Pointer)
	libraryVersion func() *byte
}

// Load dynamically opens libduckdb at libPath using purego and binds the C API
// symbols required for database, connection, and vector/chunk execution.
func Load(libPath string) (*Lib, error) {
	if libPath == "" {
		return nil, fmt.Errorf("libduckdb: library path is empty")
	}
	handle, err := purego.Dlopen(libPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return nil, fmt.Errorf("libduckdb: dlopen %q: %w", libPath, err)
	}

	l := &Lib{handle: handle}
	if err := l.registerSymbols(); err != nil {
		_ = purego.Dlclose(handle)
		return nil, err
	}
	return l, nil
}

func (l *Lib) registerSymbols() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("libduckdb: registering C symbols: %v", r)
		}
	}()

	reg := func(fptr any, name string) {
		purego.RegisterLibFunc(fptr, l.handle, name)
	}

	reg(&l.openExt, "duckdb_open_ext")
	reg(&l.closeDB, "duckdb_close")
	reg(&l.createConfig, "duckdb_create_config")
	reg(&l.setConfig, "duckdb_set_config")
	reg(&l.destroyConfig, "duckdb_destroy_config")

	reg(&l.connect, "duckdb_connect")
	reg(&l.disconnect, "duckdb_disconnect")
	reg(&l.interrupt, "duckdb_interrupt")

	reg(&l.query, "duckdb_query")
	reg(&l.resultError, "duckdb_result_error")
	reg(&l.destroyResult, "duckdb_destroy_result")
	reg(&l.columnCount, "duckdb_column_count")
	reg(&l.columnName, "duckdb_column_name")
	reg(&l.columnType, "duckdb_column_type")

	reg(&l.fetchChunk, "duckdb_fetch_chunk")
	reg(&l.dataChunkGetSize, "duckdb_data_chunk_get_size")
	reg(&l.dataChunkGetColumnCount, "duckdb_data_chunk_get_column_count")
	reg(&l.dataChunkGetVector, "duckdb_data_chunk_get_vector")
	reg(&l.destroyDataChunk, "duckdb_destroy_data_chunk")

	reg(&l.vectorGetData, "duckdb_vector_get_data")
	reg(&l.vectorGetValidity, "duckdb_vector_get_validity")
	reg(&l.vectorGetColumnType, "duckdb_vector_get_column_type")

	reg(&l.destroyLogicalType, "duckdb_destroy_logical_type")
	reg(&l.decimalScale, "duckdb_decimal_scale")
	reg(&l.decimalInternal, "duckdb_decimal_internal_type")

	reg(&l.free, "duckdb_free")
	reg(&l.libraryVersion, "duckdb_library_version")
	return nil
}

// Version returns the loaded DuckDB library version string (e.g. "v2.0.0").
func (l *Lib) Version() string {
	if l == nil || l.libraryVersion == nil {
		return ""
	}
	return cString(l.libraryVersion())
}

// Close unloads the shared library handle. Callers must close all open DBs
// before calling Close.
func (l *Lib) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}
	l.closed = true
	if l.handle != 0 {
		return purego.Dlclose(l.handle)
	}
	return nil
}
