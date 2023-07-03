package duckdb

/*
#include <duckdb.h>
*/
import "C"

const (
	stringInlineLength = 12
	stringPrefixLength = 4
)

// refer to convert_vector_list in
// duckdb/tools/juliapkg/src/result.jl
type duckdb_list_entry_t struct {
	offset C.idx_t
	length C.idx_t
}

// Refer to
// struct string_t
// duckdb/src/include/duckdb/common/types/string_type.hpp
// duckdb/tools/juliapkg/src/ctypes.jl
// duckdb/tools/juliapkg/src/result.jl
type duckdb_string_t struct {
	length int32
	prefix [stringPrefixLength]byte
	ptr    *C.char
}
