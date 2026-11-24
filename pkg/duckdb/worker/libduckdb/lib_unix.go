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

	vectorGetData            func(vec uintptr) unsafe.Pointer
	vectorGetValidity        func(vec uintptr) *uint64
	vectorEnsureValidity     func(vec uintptr)
	vectorGetColumnType      func(vec uintptr) uintptr
	vectorAssignStringLen    func(vec uintptr, idx uint64, str unsafe.Pointer, strLen uint64)
	validitySetRowInvalid    func(validity *uint64, row uint64)
	listVectorGetChild       func(vec uintptr) uintptr
	listVectorGetSize        func(vec uintptr) uint64
	structVectorGetChild     func(vec uintptr, idx uint64) uintptr
	arrayVectorGetChild      func(vec uintptr) uintptr

	createLogicalType      func(t int32) uintptr
	destroyLogicalType     func(lt *uintptr)
	getTypeID              func(lt uintptr) int32
	logicalTypeGetAlias    func(lt uintptr) *byte
	decimalWidth           func(lt uintptr) uint8
	decimalScale           func(lt uintptr) uint8
	decimalInternal        func(lt uintptr) int32
	enumInternalType       func(lt uintptr) int32
	enumDictionarySize     func(lt uintptr) uint32
	enumDictionaryValue    func(lt uintptr, idx uint64) *byte
	listTypeChildType      func(lt uintptr) uintptr
	arrayTypeChildType     func(lt uintptr) uintptr
	arrayTypeArraySize     func(lt uintptr) uint64
	mapTypeKeyType         func(lt uintptr) uintptr
	mapTypeValueType       func(lt uintptr) uintptr
	structTypeChildCount   func(lt uintptr) uint64
	structTypeChildName    func(lt uintptr, idx uint64) *byte
	structTypeChildType    func(lt uintptr, idx uint64) uintptr
	unionTypeMemberCount   func(lt uintptr) uint64
	unionTypeMemberName    func(lt uintptr, idx uint64) *byte
	unionTypeMemberType    func(lt uintptr, idx uint64) uintptr

	// Table function API (for streaming Go -> DuckDB relations).
	createTableFunction         func() uintptr
	destroyTableFunction        func(tf *uintptr)
	tableFunctionSetName        func(tf uintptr, name string)
	tableFunctionAddParam       func(tf uintptr, lt uintptr)
	tableFunctionAddNamedParam  func(tf uintptr, name string, lt uintptr)
	tableFunctionSetBind        func(tf uintptr, bind uintptr)
	tableFunctionSetInit        func(tf uintptr, init uintptr)
	tableFunctionSetFunction    func(tf uintptr, fn uintptr)
	registerTableFunction       func(conn uintptr, tf uintptr) int32
	bindGetParam                func(info uintptr, idx uint64) uintptr
	bindAddResultColumn         func(info uintptr, name string, lt uintptr)
	bindSetBindData             func(info uintptr, data uintptr, destroy uintptr)
	bindSetError                func(info uintptr, err string)
	initSetMaxThreads           func(info uintptr, maxThreads uint64)
	functionGetBindData         func(info uintptr) uintptr
	functionSetError            func(info uintptr, err string)
	getVarchar                  func(val uintptr) *byte
	getValueType                func(val uintptr) uintptr
	destroyValue                func(val *uintptr)

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
	reg(&l.columnLogicalType, "duckdb_column_logical_type")

	reg(&l.fetchChunk, "duckdb_fetch_chunk")
	reg(&l.dataChunkGetSize, "duckdb_data_chunk_get_size")
	reg(&l.dataChunkSetSize, "duckdb_data_chunk_set_size")
	reg(&l.dataChunkGetColumnCount, "duckdb_data_chunk_get_column_count")
	reg(&l.dataChunkGetVector, "duckdb_data_chunk_get_vector")
	reg(&l.destroyDataChunk, "duckdb_destroy_data_chunk")

	reg(&l.vectorGetData, "duckdb_vector_get_data")
	reg(&l.vectorGetValidity, "duckdb_vector_get_validity")
	reg(&l.vectorEnsureValidity, "duckdb_vector_ensure_validity_writable")
	reg(&l.vectorGetColumnType, "duckdb_vector_get_column_type")
	reg(&l.vectorAssignStringLen, "duckdb_vector_assign_string_element_len")
	reg(&l.validitySetRowInvalid, "duckdb_validity_set_row_invalid")
	reg(&l.listVectorGetChild, "duckdb_list_vector_get_child")
	reg(&l.listVectorGetSize, "duckdb_list_vector_get_size")
	reg(&l.structVectorGetChild, "duckdb_struct_vector_get_child")
	reg(&l.arrayVectorGetChild, "duckdb_array_vector_get_child")

	reg(&l.createLogicalType, "duckdb_create_logical_type")
	reg(&l.destroyLogicalType, "duckdb_destroy_logical_type")
	reg(&l.getTypeID, "duckdb_get_type_id")
	reg(&l.logicalTypeGetAlias, "duckdb_logical_type_get_alias")
	reg(&l.decimalWidth, "duckdb_decimal_width")
	reg(&l.decimalScale, "duckdb_decimal_scale")
	reg(&l.decimalInternal, "duckdb_decimal_internal_type")
	reg(&l.enumInternalType, "duckdb_enum_internal_type")
	reg(&l.enumDictionarySize, "duckdb_enum_dictionary_size")
	reg(&l.enumDictionaryValue, "duckdb_enum_dictionary_value")
	reg(&l.listTypeChildType, "duckdb_list_type_child_type")
	reg(&l.arrayTypeChildType, "duckdb_array_type_child_type")
	reg(&l.arrayTypeArraySize, "duckdb_array_type_array_size")
	reg(&l.mapTypeKeyType, "duckdb_map_type_key_type")
	reg(&l.mapTypeValueType, "duckdb_map_type_value_type")
	reg(&l.structTypeChildCount, "duckdb_struct_type_child_count")
	reg(&l.structTypeChildName, "duckdb_struct_type_child_name")
	reg(&l.structTypeChildType, "duckdb_struct_type_child_type")
	reg(&l.unionTypeMemberCount, "duckdb_union_type_member_count")
	reg(&l.unionTypeMemberName, "duckdb_union_type_member_name")
	reg(&l.unionTypeMemberType, "duckdb_union_type_member_type")

	reg(&l.createTableFunction, "duckdb_create_table_function")
	reg(&l.destroyTableFunction, "duckdb_destroy_table_function")
	reg(&l.tableFunctionSetName, "duckdb_table_function_set_name")
	reg(&l.tableFunctionAddParam, "duckdb_table_function_add_parameter")
	reg(&l.tableFunctionAddNamedParam, "duckdb_table_function_add_named_parameter")
	reg(&l.tableFunctionSetBind, "duckdb_table_function_set_bind")
	reg(&l.tableFunctionSetInit, "duckdb_table_function_set_init")
	reg(&l.tableFunctionSetFunction, "duckdb_table_function_set_function")
	reg(&l.registerTableFunction, "duckdb_register_table_function")
	reg(&l.bindGetParam, "duckdb_bind_get_parameter")
	reg(&l.bindAddResultColumn, "duckdb_bind_add_result_column")
	reg(&l.bindSetBindData, "duckdb_bind_set_bind_data")
	reg(&l.bindSetError, "duckdb_bind_set_error")
	reg(&l.initSetMaxThreads, "duckdb_init_set_max_threads")
	reg(&l.functionGetBindData, "duckdb_function_get_bind_data")
	reg(&l.functionSetError, "duckdb_function_set_error")
	reg(&l.getVarchar, "duckdb_get_varchar")
	reg(&l.getValueType, "duckdb_get_value_type")
	reg(&l.destroyValue, "duckdb_destroy_value")

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
