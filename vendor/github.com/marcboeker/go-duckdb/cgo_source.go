//go:build !duckdb_use_lib && (duckdb_from_source || !(darwin || (linux && (amd64 || arm64))))

package duckdb

/*
#cgo CXXFLAGS: -std=c++11 -O3 -DGODUCKDB_FROM_SOURCE -DNDEBUG
#cgo windows CXXFLAGS: -DWIN32 -DDUCKDB_BUILD_LIBRARY
#cgo linux LDFLAGS: -ldl
#cgo windows LDFLAGS: -lws2_32
#include <duckdb.h>
*/
import "C"
