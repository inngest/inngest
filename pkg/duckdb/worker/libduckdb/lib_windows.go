//go:build windows

package libduckdb

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"golang.org/x/sys/windows"
)

// Lib holds a dynamically loaded duckdb.dll handle and its registered C API
// function pointers (zero cgo).
type Lib struct {
	mu     sync.Mutex
	handle windows.Handle
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

// Load dynamically opens duckdb.dll at libPath using Windows LoadLibrary and
// binds the C API symbols via purego.RegisterFunc (zero cgo).
func Load(libPath string) (*Lib, error) {
	if libPath == "" {
		return nil, fmt.Errorf("libduckdb: library path is empty")
	}
	handle, err := windows.LoadLibrary(libPath)
	if err != nil {
		return nil, fmt.Errorf("libduckdb: LoadLibrary %q: %w", libPath, err)
	}

	l := &Lib{handle: handle}
	if err := l.registerSymbols(); err != nil {
		_ = windows.FreeLibrary(handle)
		return nil, err
	}
	return l, nil
}

func (l *Lib) registerSymbols() error {
	bindings := []struct {
		fptr any
		name string
	}{
		{&l.openExt, "duckdb_open_ext"},
		{&l.closeDB, "duckdb_close"},
		{&l.createConfig, "duckdb_create_config"},
		{&l.setConfig, "duckdb_set_config"},
		{&l.destroyConfig, "duckdb_destroy_config"},
		{&l.connect, "duckdb_connect"},
		{&l.disconnect, "duckdb_disconnect"},
		{&l.interrupt, "duckdb_interrupt"},
		{&l.query, "duckdb_query"},
		{&l.resultError, "duckdb_result_error"},
		{&l.destroyResult, "duckdb_destroy_result"},
		{&l.columnCount, "duckdb_column_count"},
		{&l.columnName, "duckdb_column_name"},
		{&l.columnType, "duckdb_column_type"},
		{&l.fetchChunk, "duckdb_fetch_chunk"},
		{&l.dataChunkGetSize, "duckdb_data_chunk_get_size"},
		{&l.dataChunkGetColumnCount, "duckdb_data_chunk_get_column_count"},
		{&l.dataChunkGetVector, "duckdb_data_chunk_get_vector"},
		{&l.destroyDataChunk, "duckdb_destroy_data_chunk"},
		{&l.vectorGetData, "duckdb_vector_get_data"},
		{&l.vectorGetValidity, "duckdb_vector_get_validity"},
		{&l.vectorGetColumnType, "duckdb_vector_get_column_type"},
		{&l.destroyLogicalType, "duckdb_destroy_logical_type"},
		{&l.decimalScale, "duckdb_decimal_scale"},
		{&l.decimalInternal, "duckdb_decimal_internal_type"},
		{&l.free, "duckdb_free"},
		{&l.libraryVersion, "duckdb_library_version"},
	}

	for _, b := range bindings {
		addr, err := windows.GetProcAddress(l.handle, b.name)
		if err != nil {
			return fmt.Errorf("libduckdb: GetProcAddress(%q): %w", b.name, err)
		}
		purego.RegisterFunc(b.fptr, addr)
	}
	return nil
}

// Version returns the loaded DuckDB library version string (e.g. "v2.0.0").
func (l *Lib) Version() string {
	if l == nil || l.libraryVersion == nil {
		return ""
	}
	return cString(l.libraryVersion())
}

// Close unloads the DLL handle. Callers must close all open DBs before calling Close.
func (l *Lib) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}
	l.closed = true
	if l.handle != 0 {
		return windows.FreeLibrary(l.handle)
	}
	return nil
}
