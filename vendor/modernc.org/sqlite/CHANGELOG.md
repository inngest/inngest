# Changelog

 - 2026-02-17 v1.46.1:
     - Ensure connection state is reset if Tx.Commit fails. Previously, errors like SQLITE_BUSY during COMMIT could leave the underlying connection inside a transaction, causing errors when the connection was reused by the database/sql pool. The driver now detects this state and forces a rollback internally.
     - Fixes [GitHub issue #2](https://github.com/modernc-org/sqlite/issues/2), thanks Edoardo Spadolini!
 - 2026-02-17 v1.46.0:
     - Enable ColumnTypeScanType to report time.Time instead of string for TEXT columns declared as DATE, DATETIME, TIME, or TIMESTAMP via a new `_texttotime` URI parameter.
     - See [GitHub pull request #1](https://github.com/modernc-org/sqlite/pull/1), thanks devhaozi!

 - 2026-02-09  v1.45.0:
     - Introduce vtab subpackage (modernc.org/sqlite/vtab) exposing Module, Table, Cursor, and IndexInfo API for Go virtual tables.
     - Wire vtab registration into the driver: vtab.RegisterModule installs modules globally and each new connection calls sqlite3_create_module_v2.
     - Implement vtab trampolines for xCreate/xConnect/xBestIndex/xDisconnect/xDestroy/xOpen/xClose/xFilter/xNext/xEof/xColumn/xRowid.
     - Map SQLite’s sqlite3_index_info into vtab.IndexInfo, including constraints, ORDER BY terms, and constraint usage (ArgIndex → xFilter argv[]).
     - Add an in‑repo dummy vtab module and test (module_test.go) that validates registration, basic scanning, and constraint visibility.
     - See [GitLab merge request #90](https://gitlab.com/cznic/sqlite/-/merge_requests/90), thanks Adrian Witas!

 - 2026-01-19 v1.44.3: Resolves [GitLab issue #243](https://gitlab.com/cznic/sqlite/-/issues/243).

 - 2026-01-18 v1.44.2: Upgrade to  [SQLite 3.51.2](https://sqlite.org/releaselog/3_51_2.html).

 - 2026-01-13 v1.44.0: Upgrade to SQLite 3.51.1.

 - 2025-10-10 v1.39.1: Upgrade to SQLite 3.50.4.

 - 2025-06-09 v1.38.0: Upgrade to SQLite 3.50.1.

 - 2025-02-26 v1.36.0: Upgrade to SQLite 3.49.0.

 - 2024-11-16 v1.34.0: Implement ResetSession and IsValid methods in connection

 - 2024-07-22 v1.31.0: Support windows/386.

 - 2024-06-04 v1.30.0: Upgrade to SQLite 3.46.0, release notes at
   https://sqlite.org/releaselog/3_46_0.html.

 - 2024-02-13 v1.29.0: Upgrade to SQLite 3.45.1, release notes at
   https://sqlite.org/releaselog/3_45_1.html.

 - 2023-12-14: v1.28.0: Add (*Driver).RegisterConnectionHook,
   ConnectionHookFn, ExecQuerierContext, RegisterConnectionHook.

 - 2023-08-03 v1.25.0: enable SQLITE_ENABLE_DBSTAT_VTAB.

 - 2023-07-11 v1.24.0: Add
   (*conn).{Serialize,Deserialize,NewBackup,NewRestore} methods, add Backup
   type.

 - 2023-06-01 v1.23.0: Allow registering aggregate functions.

 - 2023-04-22 v1.22.0: Support linux/s390x.

 - 2023-02-23 v1.21.0: Upgrade to SQLite 3.41.0, release notes at
   https://sqlite.org/releaselog/3_41_0.html.

 - 2022-11-28 v1.20.0: Support linux/ppc64le.

 - 2022-09-16 v1.19.0: Support frebsd/arm64.

 - 2022-07-26 v1.18.0: Add support for Go fs.FS based SQLite virtual
   filesystems, see function New in modernc.org/sqlite/vfs and/or TestVFS in
   all_test.go

 - 2022-04-24 v1.17.0: Support windows/arm64.

 - 2022-04-04 v1.16.0: Support scalar application defined functions written
   in Go. See https://www.sqlite.org/appfunc.html

 - 2022-03-13 v1.15.0: Support linux/riscv64.

 - 2021-11-13 v1.14.0: Support windows/amd64. This target had previously
   only experimental status because of a now resolved memory leak.

 - 2021-09-07 v1.13.0: Support freebsd/amd64.

 - 2021-06-23 v1.11.0: Upgrade to use sqlite 3.36.0, release notes at
   https://www.sqlite.org/releaselog/3_36_0.html.

 - 2021-05-06 v1.10.6: Fixes a memory corruption issue
   (https://gitlab.com/cznic/sqlite/-/issues/53).  Versions since v1.8.6 were
   affected and should be updated to v1.10.6.

 - 2021-03-14 v1.10.0: Update to use sqlite 3.35.0, release notes at
   https://www.sqlite.org/releaselog/3_35_0.html.

 - 2021-03-11 v1.9.0: Support darwin/arm64.

 - 2021-01-08 v1.8.0: Support darwin/amd64.

 - 2020-09-13 v1.7.0: Support linux/arm and linux/arm64.

 - 2020-09-08 v1.6.0: Support linux/386.

 - 2020-09-03 v1.5.0: This project is now completely CGo-free, including
   the Tcl tests.

 - 2020-08-26 v1.4.0: First stable release for linux/amd64.  The
   database/sql driver and its tests are CGo free.  Tests of the translated
   sqlite3.c library still require CGo.

 - 2020-07-26 v1.4.0-beta1: The project has reached beta status while
   supporting linux/amd64 only at the moment. The 'extraquick' Tcl testsuite
   reports

 - 2019-12-28 v1.2.0-alpha.3: Third alpha fixes issue #19.

 - 2019-12-26 v1.1.0-alpha.2: Second alpha release adds support for
   accessing a database concurrently by multiple goroutines and/or processes.
   v1.1.0 is now considered feature-complete. Next planed release should be a
   beta with a proper test suite.

 - 2019-12-18 v1.1.0-alpha.1: First alpha release using the new cc/v3,
   gocc, qbe toolchain. Some primitive tests pass on linux_{amd64,386}. Not
   yet safe for concurrent access by multiple goroutines. Next alpha release
   is planed to arrive before the end of this year.

 - 2017-06-10: Windows/Intel no more uses the VM (thanks Steffen Butzer).

 - 2017-06-05 Linux/Intel no more uses the VM (cznic/virtual).
