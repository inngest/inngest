//go:build !darwin && !linux && !windows

package libduckdb

import (
	"fmt"
	"runtime"
	"unsafe"
)

// Lib is a stub on unsupported platforms.
type Lib struct {
	openExt                 func(path string, outDB *uintptr, cfg uintptr, outErr **byte) int32
	closeDB                 func(db *uintptr)
	createConfig            func(outCfg *uintptr) int32
	setConfig               func(cfg uintptr, name, option string) int32
	destroyConfig           func(cfg *uintptr)
	connect                 func(db uintptr, outConn *uintptr) int32
	disconnect              func(conn *uintptr)
	interrupt               func(conn uintptr)
	query                   func(conn uintptr, sql string, outResult *cResult) int32
	resultError             func(res *cResult) *byte
	destroyResult           func(res *cResult)
	columnCount             func(res *cResult) uint64
	columnName              func(res *cResult, col uint64) *byte
	columnType              func(res *cResult, col uint64) int32
	fetchChunk              func(res cResult) uintptr
	dataChunkGetSize        func(chunk uintptr) uint64
	dataChunkGetColumnCount func(chunk uintptr) uint64
	dataChunkGetVector      func(chunk uintptr, col uint64) uintptr
	destroyDataChunk        func(chunk *uintptr)
	vectorGetData           func(vec uintptr) unsafe.Pointer
	vectorGetValidity       func(vec uintptr) *uint64
	vectorGetColumnType     func(vec uintptr) uintptr
	destroyLogicalType      func(lt *uintptr)
	decimalScale            func(lt uintptr) uint8
	decimalInternal         func(lt uintptr) int32
	free                    func(ptr unsafe.Pointer)
	libraryVersion          func() *byte
}

// Load returns an error on platforms without purego dynamic loading support.
func Load(libPath string) (*Lib, error) {
	return nil, fmt.Errorf("libduckdb: purego dynamic loading is not supported on %s/%s", runtime.GOOS, runtime.GOARCH)
}

func (l *Lib) Version() string { return "" }
func (l *Lib) Close() error    { return nil }
