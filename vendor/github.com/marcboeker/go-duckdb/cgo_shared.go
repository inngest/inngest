//go:build duckdb_use_lib

package duckdb

/*
#cgo LDFLAGS: -lduckdb
#include <duckdb.h>
*/
import "C"
