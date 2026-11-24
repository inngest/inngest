//go:build windows

package libduckdb

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
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

	query             func(conn uintptr, sql string, outResult *cResult) int32
	resultError       func(res *cResult) *byte
	destroyResult     func(res *cResult)
	columnCount       func(res *cResult) uint64
	columnName        func(res *cResult, col uint64) *byte
	columnType        func(res *cResult, col uint64) int32
	columnLogicalType func(res *cResult, col uint64) uintptr

	fetchChunk              func(res cResult) uintptr
	dataChunkGetSize        func(chunk uintptr) uint64
	dataChunkSetSize        func(chunk uintptr, size uint64)
	dataChunkGetColumnCount func(chunk uintptr) uint64
	dataChunkGetVector      func(chunk uintptr, col uint64) uintptr
	destroyDataChunk        func(chunk *uintptr)

	vectorGetData         func(vec uintptr) unsafe.Pointer
	vectorGetValidity     func(vec uintptr) *uint64
	vectorEnsureValidity  func(vec uintptr)
	vectorGetColumnType   func(vec uintptr) uintptr
	vectorAssignStringLen func(vec uintptr, idx uint64, str unsafe.Pointer, strLen uint64)
	validitySetRowInvalid func(validity *uint64, row uint64)
	listVectorGetChild    func(vec uintptr) uintptr
	listVectorGetSize     func(vec uintptr) uint64
	structVectorGetChild  func(vec uintptr, idx uint64) uintptr
	arrayVectorGetChild   func(vec uintptr) uintptr

	createLogicalType    func(t int32) uintptr
	destroyLogicalType   func(lt *uintptr)
	getTypeID            func(lt uintptr) int32
	logicalTypeGetAlias  func(lt uintptr) *byte
	decimalWidth         func(lt uintptr) uint8
	decimalScale         func(lt uintptr) uint8
	decimalInternal      func(lt uintptr) int32
	enumInternalType     func(lt uintptr) int32
	enumDictionarySize   func(lt uintptr) uint32
	enumDictionaryValue  func(lt uintptr, idx uint64) *byte
	listTypeChildType    func(lt uintptr) uintptr
	arrayTypeChildType   func(lt uintptr) uintptr
	arrayTypeArraySize   func(lt uintptr) uint64
	mapTypeKeyType       func(lt uintptr) uintptr
	mapTypeValueType     func(lt uintptr) uintptr
	structTypeChildCount func(lt uintptr) uint64
	structTypeChildName  func(lt uintptr, idx uint64) *byte
	structTypeChildType  func(lt uintptr, idx uint64) uintptr
	unionTypeMemberCount func(lt uintptr) uint64
	unionTypeMemberName  func(lt uintptr, idx uint64) *byte
	unionTypeMemberType  func(lt uintptr, idx uint64) uintptr

	createTableFunction        func() uintptr
	destroyTableFunction       func(tf *uintptr)
	tableFunctionSetName       func(tf uintptr, name string)
	tableFunctionAddParam      func(tf uintptr, lt uintptr)
	tableFunctionAddNamedParam func(tf uintptr, name string, lt uintptr)
	tableFunctionSetBind       func(tf uintptr, bind uintptr)
	tableFunctionSetInit       func(tf uintptr, init uintptr)
	tableFunctionSetFunction   func(tf uintptr, fn uintptr)
	registerTableFunction      func(conn uintptr, tf uintptr) int32
	bindGetParam               func(info uintptr, idx uint64) uintptr
	bindAddResultColumn        func(info uintptr, name string, lt uintptr)
	bindSetBindData            func(info uintptr, data uintptr, destroy uintptr)
	bindSetError               func(info uintptr, err string)
	initSetMaxThreads          func(info uintptr, maxThreads uint64)
	functionGetBindData        func(info uintptr) uintptr
	functionSetError           func(info uintptr, err string)
	getVarchar                 func(val uintptr) *byte
	getValueType               func(val uintptr) uintptr
	destroyValue               func(val *uintptr)

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
		{&l.columnLogicalType, "duckdb_column_logical_type"},
		{&l.dataChunkGetSize, "duckdb_data_chunk_get_size"},
		{&l.dataChunkSetSize, "duckdb_data_chunk_set_size"},
		{&l.dataChunkGetColumnCount, "duckdb_data_chunk_get_column_count"},
		{&l.dataChunkGetVector, "duckdb_data_chunk_get_vector"},
		{&l.destroyDataChunk, "duckdb_destroy_data_chunk"},
		{&l.vectorGetData, "duckdb_vector_get_data"},
		{&l.vectorGetValidity, "duckdb_vector_get_validity"},
		{&l.vectorEnsureValidity, "duckdb_vector_ensure_validity_writable"},
		{&l.vectorGetColumnType, "duckdb_vector_get_column_type"},
		{&l.vectorAssignStringLen, "duckdb_vector_assign_string_element_len"},
		{&l.validitySetRowInvalid, "duckdb_validity_set_row_invalid"},
		{&l.listVectorGetChild, "duckdb_list_vector_get_child"},
		{&l.listVectorGetSize, "duckdb_list_vector_get_size"},
		{&l.structVectorGetChild, "duckdb_struct_vector_get_child"},
		{&l.arrayVectorGetChild, "duckdb_array_vector_get_child"},
		{&l.createLogicalType, "duckdb_create_logical_type"},
		{&l.destroyLogicalType, "duckdb_destroy_logical_type"},
		{&l.getTypeID, "duckdb_get_type_id"},
		{&l.logicalTypeGetAlias, "duckdb_logical_type_get_alias"},
		{&l.decimalWidth, "duckdb_decimal_width"},
		{&l.decimalScale, "duckdb_decimal_scale"},
		{&l.decimalInternal, "duckdb_decimal_internal_type"},
		{&l.enumInternalType, "duckdb_enum_internal_type"},
		{&l.enumDictionarySize, "duckdb_enum_dictionary_size"},
		{&l.enumDictionaryValue, "duckdb_enum_dictionary_value"},
		{&l.listTypeChildType, "duckdb_list_type_child_type"},
		{&l.arrayTypeChildType, "duckdb_array_type_child_type"},
		{&l.arrayTypeArraySize, "duckdb_array_type_array_size"},
		{&l.mapTypeKeyType, "duckdb_map_type_key_type"},
		{&l.mapTypeValueType, "duckdb_map_type_value_type"},
		{&l.structTypeChildCount, "duckdb_struct_type_child_count"},
		{&l.structTypeChildName, "duckdb_struct_type_child_name"},
		{&l.structTypeChildType, "duckdb_struct_type_child_type"},
		{&l.unionTypeMemberCount, "duckdb_union_type_member_count"},
		{&l.unionTypeMemberName, "duckdb_union_type_member_name"},
		{&l.unionTypeMemberType, "duckdb_union_type_member_type"},
		{&l.createTableFunction, "duckdb_create_table_function"},
		{&l.destroyTableFunction, "duckdb_destroy_table_function"},
		{&l.tableFunctionSetName, "duckdb_table_function_set_name"},
		{&l.tableFunctionAddParam, "duckdb_table_function_add_parameter"},
		{&l.tableFunctionAddNamedParam, "duckdb_table_function_add_named_parameter"},
		{&l.tableFunctionSetBind, "duckdb_table_function_set_bind"},
		{&l.tableFunctionSetInit, "duckdb_table_function_set_init"},
		{&l.tableFunctionSetFunction, "duckdb_table_function_set_function"},
		{&l.registerTableFunction, "duckdb_register_table_function"},
		{&l.bindGetParam, "duckdb_bind_get_parameter"},
		{&l.bindAddResultColumn, "duckdb_bind_add_result_column"},
		{&l.bindSetBindData, "duckdb_bind_set_bind_data"},
		{&l.bindSetError, "duckdb_bind_set_error"},
		{&l.initSetMaxThreads, "duckdb_init_set_max_threads"},
		{&l.functionGetBindData, "duckdb_function_get_bind_data"},
		{&l.functionSetError, "duckdb_function_set_error"},
		{&l.getVarchar, "duckdb_get_varchar"},
		{&l.getValueType, "duckdb_get_value_type"},
		{&l.destroyValue, "duckdb_destroy_value"},
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

	// purego.RegisterFunc on Windows rejects struct-by-value arguments.
	// duckdb_fetch_chunk(duckdb_result result) takes a 48-byte struct by value;
	// on Windows x64 ABI structs larger than 8 bytes are passed by hidden
	// pointer in RCX and the returned duckdb_data_chunk pointer is in RAX.
	fetchChunkAddr, err := windows.GetProcAddress(l.handle, "duckdb_fetch_chunk")
	if err != nil {
		return fmt.Errorf("libduckdb: GetProcAddress(%q): %w", "duckdb_fetch_chunk", err)
	}
	l.fetchChunk = func(res cResult) uintptr {
		r1, _, _ := syscall.SyscallN(fetchChunkAddr, uintptr(unsafe.Pointer(&res)))
		runtime.KeepAlive(&res)
		return r1
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
