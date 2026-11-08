// Copyright 2025 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite // import "modernc.org/sqlite"

import (
	"database/sql/driver"

	sqlite3 "modernc.org/sqlite/lib"
)

// Backup object is used to manage progress and cleanup an online backup. It
// is returned by NewBackup or NewRestore.
type Backup struct {
	srcConn *conn   // source database connection
	dstConn *conn   // destination database connection
	pBackup uintptr // sqlite3_backup object pointer
}

// Step will copy up to n pages between the source and destination databases
// specified by the backup object. If n is negative, all remaining source
// pages are copied.
// If it successfully copies n pages and there are still more pages to be
// copied, then the function returns true with no error. If it successfully
// finishes copying all pages from source to destination, then it returns
// false with no error. If an error occurs while running, then an error is
// returned.
func (b *Backup) Step(n int32) (bool, error) {
	rc := sqlite3.Xsqlite3_backup_step(b.srcConn.tls, b.pBackup, n)
	if rc == sqlite3.SQLITE_OK {
		return true, nil
	} else if rc == sqlite3.SQLITE_DONE {
		return false, nil
	} else {
		return false, b.srcConn.errstr(rc)
	}
}

// Finish releases all resources associated with the Backup object. The Backup
// object is invalid and may not be used following a call to Finish.
func (b *Backup) Finish() error {
	rc := sqlite3.Xsqlite3_backup_finish(b.srcConn.tls, b.pBackup)
	b.dstConn.Close()
	if rc == sqlite3.SQLITE_OK {
		return nil
	} else {
		return b.srcConn.errstr(rc)
	}
}

// Remaining returns the number of source-database pages still to be backed
// up at the conclusion of the most recent [Backup.Step] call. The value is
// useful for driving progress UIs that need to estimate how much work is
// left.
//
// If Step has not yet been called on this Backup, or if the most recent
// Step returned false (SQLITE_DONE), Remaining returns 0.
//
// See https://www.sqlite.org/c3ref/backup_finish.html.
func (b *Backup) Remaining() int {
	return int(sqlite3.Xsqlite3_backup_remaining(b.srcConn.tls, b.pBackup))
}

// PageCount returns the total number of pages in the source database at the
// conclusion of the most recent [Backup.Step] call. Pair with [Backup.Remaining]
// to compute progress as a fraction (PageCount - Remaining) / PageCount.
//
// See https://www.sqlite.org/c3ref/backup_finish.html.
func (b *Backup) PageCount() int {
	return int(sqlite3.Xsqlite3_backup_pagecount(b.srcConn.tls, b.pBackup))
}

// Commit releases all resources associated with the Backup object but does not
// close the destination database connection.
//
// The destination database connection is returned to the caller or an error if raised.
// It is the responsibility of the caller to handle the connection closure.
func (b *Backup) Commit() (driver.Conn, error) {
	rc := sqlite3.Xsqlite3_backup_finish(b.srcConn.tls, b.pBackup)

	if rc == sqlite3.SQLITE_OK {
		return b.dstConn, nil
	} else {
		b.dstConn.Close()
		return nil, b.srcConn.errstr(rc)
	}
}
