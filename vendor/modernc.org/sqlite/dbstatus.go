// Copyright 2026 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite // import "modernc.org/sqlite"

import (
	"unsafe"

	sqlite3 "modernc.org/sqlite/lib"
)

// DBStatusOp identifies a per-connection runtime counter readable through
// [DBStatus.Status]. The values mirror the SQLITE_DBSTATUS_* verbs of the C
// API; the distinct type keeps a counter from a different family (for example
// a file-control or db-config op) from compiling in its place.
//
// See https://www.sqlite.org/c3ref/c_dbstatus_options.html for the per-op
// semantics.
type DBStatusOp int32

// DBStatus* are the operations accepted by [DBStatus.Status]. They report
// their value differently depending on the op:
//
//   - DBStatusLookasideUsed: current is the lookaside memory in use now; high
//     is its high-water mark. The reset flag rebases the high-water mark to
//     current. This is the only op that maintains a high-water mark.
//   - Memory-usage ops (DBStatusCacheUsed, DBStatusSchemaUsed,
//     DBStatusStmtUsed, DBStatusCacheUsedShared): current is the bytes in use
//     now; high is always 0; the reset flag is ignored.
//   - Running-counter ops (DBStatusCacheHit, DBStatusCacheMiss,
//     DBStatusCacheWrite, DBStatusCacheSpill, DBStatusTempbufSpill): current
//     is the cumulative count (bytes spilled, for DBStatusTempbufSpill); high
//     is always 0. The reset flag zeroes current.
//   - Lookaside event ops (DBStatusLookasideHit, DBStatusLookasideMissSize,
//     DBStatusLookasideMissFull): the count is reported in high, not current
//     (current is always 0). The reset flag zeroes high.
//   - DBStatusDeferredFKs: current is 1 if the connection has unresolved
//     deferred foreign-key constraints, else 0; high is always 0; the reset
//     flag is ignored.
const (
	DBStatusLookasideUsed     = DBStatusOp(sqlite3.SQLITE_DBSTATUS_LOOKASIDE_USED)
	DBStatusCacheUsed         = DBStatusOp(sqlite3.SQLITE_DBSTATUS_CACHE_USED)
	DBStatusSchemaUsed        = DBStatusOp(sqlite3.SQLITE_DBSTATUS_SCHEMA_USED)
	DBStatusStmtUsed          = DBStatusOp(sqlite3.SQLITE_DBSTATUS_STMT_USED)
	DBStatusLookasideHit      = DBStatusOp(sqlite3.SQLITE_DBSTATUS_LOOKASIDE_HIT)
	DBStatusLookasideMissSize = DBStatusOp(sqlite3.SQLITE_DBSTATUS_LOOKASIDE_MISS_SIZE)
	DBStatusLookasideMissFull = DBStatusOp(sqlite3.SQLITE_DBSTATUS_LOOKASIDE_MISS_FULL)
	DBStatusCacheHit          = DBStatusOp(sqlite3.SQLITE_DBSTATUS_CACHE_HIT)
	DBStatusCacheMiss         = DBStatusOp(sqlite3.SQLITE_DBSTATUS_CACHE_MISS)
	DBStatusCacheWrite        = DBStatusOp(sqlite3.SQLITE_DBSTATUS_CACHE_WRITE)
	DBStatusDeferredFKs       = DBStatusOp(sqlite3.SQLITE_DBSTATUS_DEFERRED_FKS)
	DBStatusCacheUsedShared   = DBStatusOp(sqlite3.SQLITE_DBSTATUS_CACHE_USED_SHARED)
	DBStatusCacheSpill        = DBStatusOp(sqlite3.SQLITE_DBSTATUS_CACHE_SPILL)
	DBStatusTempbufSpill      = DBStatusOp(sqlite3.SQLITE_DBSTATUS_TEMPBUF_SPILL)
)

// DBStatus exposes sqlite3_db_status, the per-connection runtime counters
// (cache hit/miss/write/spill rates, schema and prepared-statement memory,
// lookaside usage, deferred foreign keys). Reach it through the
// database/sql escape hatch, the same way as [FileControl]:
//
//	err := sqlConn.Raw(func(dc any) error {
//		cur, _, err := dc.(sqlite.DBStatus).Status(sqlite.DBStatusCacheSpill, false)
//		if err != nil {
//			return err
//		}
//		// use cur
//		return nil
//	})
type DBStatus interface {
	// Status returns the current and high-water values of the per-connection
	// counter identified by op. When reset is true the counter is reset after
	// the read; which value the reset affects depends on the op's family, see
	// the DBStatus* constant documentation. The returned error is non-nil only
	// when SQLite rejects op as out of range.
	Status(op DBStatusOp, reset bool) (current, high int, err error)
}

var _ DBStatus = (*conn)(nil)

func (c *conn) Status(op DBStatusOp, reset bool) (current, high int, err error) {
	// Two int32 out-params: pCurrent, pHighwater. sqlite3_db_status writes
	// C int (32-bit) through both, so a single 8-byte buffer holds the pair.
	p := c.tls.Alloc(8)
	defer c.tls.Free(8)

	pCur, pHi := p, p+4
	resetFlag := int32(0)
	if reset {
		resetFlag = 1
	}
	if rc := sqlite3.Xsqlite3_db_status(c.tls, c.db, int32(op), pCur, pHi, resetFlag); rc != sqlite3.SQLITE_OK {
		return 0, 0, c.errstr(rc)
	}

	return int(*(*int32)(unsafe.Pointer(pCur))), int(*(*int32)(unsafe.Pointer(pHi))), nil
}
