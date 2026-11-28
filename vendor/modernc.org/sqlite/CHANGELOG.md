# Changelog

 - 2026-08-19 v1.57.0:
     - Add an opt-in `_defensive` DSN query parameter that turns on SQLite's defensive mode for the connection, disabling the SQL-level features that let ordinary statements deliberately corrupt the database file. When `_defensive=1` (or any `strconv.ParseBool` true value) is supplied, the driver calls `sqlite3_db_config` with `SQLITE_DBCONFIG_DEFENSIVE` immediately after `sqlite3_open_v2` and before every other parameter is applied, so the PRAGMAs the driver itself runs, the `_pragma` list, and every statement the caller prepares are all subject to it. On such a connection `PRAGMA writable_schema=ON`, `PRAGMA journal_mode=OFF` and `PRAGMA schema_version=N` become silent no-ops, and writes to a virtual table's shadow tables (fts5's `_data`, `_idx` and so on) and to `sqlite_dbpage` fail with "table ... may not be modified"; reading those tables, ordinary use of the virtual tables that own them, and `VACUUM` are unaffected. The flag has no PRAGMA equivalent, so `sqlite3_db_config` — and therefore a DSN parameter — is the only way to reach it short of dropping to `modernc.org/sqlite/lib`. The value is parsed before `sqlite3_open_v2`, so an invalid one fails the connection without creating the database file, and the parameter must appear at most once: a repeated `_defensive` is an error rather than letting the first value silently win. Absence of the parameter, or `_defensive=0`, leaves SQLite's default behavior unchanged; existing DSNs continue to work byte-for-byte. Two limits are worth stating plainly, since the name invites more confidence than the flag earns. Defensive mode is a hardening measure, not a sandbox for hostile database files: it is one of several steps [SQLite recommends](https://www.sqlite.org/security.html) for that purpose, and this build compiles with neither `SQLITE_TRUSTED_SCHEMA=0` nor `SQLITE_DQS=0` and exposes no authorizer. And it is a property of the connection, not of the database file — a second handle opened on the same file without the parameter is unrestricted.
     - Reject the one DSN combination defensive mode would otherwise swallow in silence. `_defensive=1` together with `_journal_mode=OFF` (or `_journal=OFF`) now fails the connection instead of opening one in which neither parameter was honoured: SQLite turns `PRAGMA journal_mode=OFF` into a no-op that still reports success, so the driver would have accepted the mode, executed it, and left the journal untouched without telling anyone. The check runs in the validation phase introduced in v1.55.0, before any statement executes, so a rejected DSN cannot leave the database half-configured. `_pragma` remains the exception it has always been: `_pragma=journal_mode(OFF)` alongside `_defensive=1` still runs and is still silently ignored by SQLite. Only DSNs using `_defensive` can be affected, and that parameter is new, so no DSN that opened before changes behavior.
     - See [GitHub pull request #6](https://github.com/modernc-org/sqlite/pull/6), thanks wsman!
     - Ship the sqlite-vec license notice this module has been missing. `modernc.org/sqlite/vec` has bundled the transpiled [sqlite-vec](https://github.com/asg017/sqlite-vec) sources since v1.47.0, but the module carried only its own BSD-3-Clause `LICENSE` and the public-domain SQLite notice. sqlite-vec is Copyright (c) 2024 Alex Garcia, dual-licensed Apache-2.0 OR MIT and used here under MIT, whose terms require the copyright and permission notice to accompany substantial portions of the software — which 2.8 MB of transpiled `vec/` plainly is. The notice now ships as `LICENSE-SQLITE_VEC` in the module root, byte-identical to the `LICENSE-MIT` in the upstream v0.1.9 archive and named after the file `modernc.org/libsqlite_vec` extracts it into. Attribution was never absent — `vec`'s package documentation has named the extension, pinned the version and linked upstream — but the license text itself was, and the omission was ours: `vendor_libs/main.go` copied the per-target transpiles and nothing else. It now copies the notice alongside them and fails the vendoring run if it cannot, so a `make vendor` can no longer quietly drop it. The `vec` package documentation gained a License section recording that the package is under a different license from the rest of this module.
     - **The SQLite notice is renamed from `SQLITE-LICENSE` to `LICENSE-SQLITE`**; update any direct links to it. Its contents are unchanged and SQLite remains public domain. The name now matches both the new `LICENSE-SQLITE_VEC` beside it and the `LICENSE-<upstream>` convention every other modernc.org repository follows, but it is more than cosmetic: `go mod vendor` selects the files it copies into a downstream `vendor/` tree by matching each name against a fixed list of prefixes — `LICENSE` among them — so a name merely *ending* in `LICENSE` was never propagated. Both bundled notices now travel with the code into vendored builds, which is where the MIT terms on `vec/` keep applying. No code changes; no behavior changes.
     - Let a caller-constructed `Driver` register its own functions, collations and virtual table modules. `Driver` has always held four categories of registration state, but only `RegisterConnectionHook` could put anything on a constructed one: functions and collations were reachable through the package-level API alone, and modules through the package-level driver only, which left the `modules` field written and read through that instance and so process-global state wearing a per-instance field. `Driver` now has `RegisterFunction`, `RegisterScalarFunction`, `RegisterDeterministicScalarFunction`, `RegisterCollationUtf8` and `RegisterModule`, plus `Must*` variants of the first four, each registering on that `Driver` alone; the methods are safe to call concurrently, and the zero `Driver` is ready to use as-is. `vtab.RegisterModule` also honours its `db` argument now: a non-nil `db` registers on the driver backing it when that driver implements the new `vtab.ModuleRegisterer`, while a nil `db` keeps targeting the driver this package registers as `sqlite`. One existing pattern changes behavior, narrowly and loudly: `vtab.RegisterModule(db, ...)` where `db` was opened on a caller-constructed `Driver` used to discard the `db` argument and land on the `sqlite` driver, reaching every connection in the process; it now lands on the constructed driver alone, so a `sql.Open("sqlite")` connection that used to resolve such a module gets `no such module` instead. The same pattern is also the one way an existing program could hold one module name on both a constructed `Driver` and the package-level one: there the first of the two registrations used to win and the second was refused as already registered, whereas now the package-level implementation wins on the constructed `Driver`'s connections regardless of the order they ran in. Reaching that case at all means the program ignored an error the older version returned. Two smaller deviations round out the list: `Driver.RegisterModule` reports no error for such a collision, and `vtab.RegisterModule` now validates its name and module arguments before the not-implemented check, so a call with an empty name that returned `vtab: RegisterModule not wired into engine` outside this driver returns `vtab: module name must be non-empty` instead. Everything else is additive against v1.56.0: the package-level registration functions target the same driver they always did, connections still receive every module registered through the package-level path whichever `Driver` opened them, and a `db` opened on the `sqlite` driver resolves to that same driver. The isolating change discussed in [GitLab issue #254](https://gitlab.com/cznic/sqlite/-/issues/254) is deliberately not made here.
     - See [GitLab merge request #135](https://gitlab.com/cznic/sqlite/-/merge_requests/135), thanks Ian Chechin!
     - Promote `freebsd/386`, `freebsd/arm` and `netbsd/amd64` to fully supported platforms. All three are now listed in the package documentation's platform table, which had carried seventeen entries while this module shipped, cross-built and tested twenty. They arrived as **experimental** in v1.53.0 — `netbsd/amd64` reviving a port that had been broken for years, `freebsd/386` replacing a stale, effectively untested SQLite 3.41 transpile, and `freebsd/arm` entirely new — and were deliberately kept out of that table until they had accumulated real-world exposure, with promotion promised once "a period of broader real-world testing … elapses without surprises". That period has elapsed: all three have been in the builder test matrix and in `make build_all_targets` since v1.53.0, all three pass the full test suite on this release's commit alongside the seventeen platforms already listed, and no open issue reports a defect in any of them. The two `netbsd/amd64` build failures filed before the revival, [GitLab issue #202](https://gitlab.com/cznic/sqlite/-/issues/202) and [GitLab issue #234](https://gitlab.com/cznic/sqlite/-/issues/234), no longer reproduce at this commit: `Xsqlite3_is_interrupted` is present in the sources that target selects, and the `mu.enter`/`mu.leave` symbols that broke the build are gone. Documentation only — the transpiled sources under `lib/` are byte-for-byte what v1.56.0 shipped, and nothing about how these targets behave changes.

 - 2026-08-03 v1.56.0:
     - Re-vendor the transpiled SQLite sources, picking up `modernc.org/libsqlite3`'s fix for an upstream **data-corruption bug in SQLite 3.53.3's journal rollback**. The SQLite version is unchanged at [3.53.3](https://sqlite.org/releaselog/3_53_3.html); what changes is that the amalgamation is now patched before it is transpiled. 3.53.3 reworked `readSuperJournal()` to return the super-journal name through a `char**` out-parameter, and `pager_playback()` now tests that pointer where it used to test `zSuper[0]`. A crash during the commit of a multi-database (ATTACH) transaction can leave the super-journal name and its checksum zeroed while the name length and the trailing magic survive; the checksum is a plain byte sum, so an all-zero name still validates and `readSuperJournal()` hands back a non-NULL pointer to an empty string. `pager_playback()` then calls `sqlite3OsAccess(pVfs, "", SQLITE_ACCESS_EXISTS)`, gets ENOENT, and deletes the hot journal without playing it back — leaving the database corrupted. This is not a transpilation artifact: a plain gcc build of the stock 3.53.3 amalgamation fails on the same bytes while 3.53.2 recovers them, and it is what has been making upstream's own `test/crash.test` fail intermittently, in roughly 2% of runs, on every platform. The patch restores the pre-3.53.3 behaviour of reporting a `(nul)` super-journal name and will be dropped once upstream ships its own fix. Every supported target carries it.
     - Two targets change beyond that patch. On `linux/s390x` the regenerated transpile allocates C bit-fields MSB-first, as the big-endian platform ABI requires, rather than LSB-first; this comes from `modernc.org/cc/v4` v4.29.1 and touches bit-field accesses throughout the SQLite core, s390x being this module's only big-endian target. On `linux/riscv64` the transpile was regenerated on a host running GCC 11.4.0 where the previous one used GCC 13.3.0, which drops a handful of unexported compiler-predefined macro constants (the `__FLT16_*` family, `__DBL_IS_IEC_60559__` and friends) and changes the `COMPILER=gcc-13.3.0` entry `PRAGMA compile_options` reports to `COMPILER=gcc-11.4.0`; no SQLite code generation differs. Every other target's generated code is byte-identical to v1.55.0 apart from the journal-rollback patch above.
     - Bump the pinned `modernc.org/libc` to v1.74.4, and the remaining dependencies to their current releases. v1.74.2 and v1.74.3 are retracted upstream — a `freeaddrinfo` lock leak that deadlocks name resolution — and v1.74.4 is the fix. As always, downstream modules must pin the exact `modernc.org/libc` version this module's `go.mod` pins (see [GitLab issue #177](https://gitlab.com/cznic/sqlite/-/issues/177)).
     - Documentation sweep. `openbsd/amd64` and `openbsd/arm64` join the supported platforms table in the package documentation: both have been in the builder test matrix since January and are cross-built by `make build_all_targets`, but had never been listed. The `vfs` DSN query parameter — which names a VFS registered with SQLite, such as one returned by `vfs.New` — is now documented alongside the other DSN parameters on `Driver.Open`. The "Debug and development versions" section no longer describes a `GO_GENERATE` environment variable and a `go generate` that this repository has not had since `generator.go` moved to `modernc.org/libsqlite3`; it now points at that repository and `make vendor` instead, and the stale `//go:generate` directive naming the removed file is dropped with it. `modernc.org/sqlite/vec` and `modernc.org/sqlite/vfs` gained the package doc comments they were missing, so both finally carry a synopsis on pkg.go.dev. Documentation only; no behavior changes.
     - Add `NewConnector`, returning a `database/sql/driver.Connector` for use with `sql.OpenDB`. It opens the same connections `sql.Open("sqlite", dsn)` does, from the same registered driver, so every function, collation, connection hook and virtual table module registered through this package applies to them. It exists for callers that need to interpose on the physical connections `database/sql` opens — tracing, metrics, connection-scoped setup — which `sql.Open` gives no access to: such a caller can embed the returned `Connector`, override `Connect`, and pass its own wrapper to `sql.OpenDB`. Previously the only way to reach the registered driver was the `db, _ := sql.Open("sqlite", ""); drv := db.Driver(); db.Close()` idiom, which works only because `sql.Open` does not connect and this driver does not implement `driver.DriverContext`; and the only way to get a wrapper into a `*sql.DB` was `sql.Register`, which is process-global, panics on a name it has already seen, and cannot be undone, so a library had to invent a unique driver name per configuration. `sql.OpenDB` registers nothing. Constructing a `&sqlite.Driver{}` is not an alternative — its fields are unexported, so it carries none of the registrations. `NewConnector` checks the DSN only as far as it can without opening a database — a query string that does not parse, and conflicting `vfs` parameters; everything else continues to be validated when the connection is opened, so an unknown parameter or an out-of-range value is reported by `Connect` rather than at construction. Nothing about the existing `sql.Open` path changes: `*Driver` deliberately still does not implement `driver.DriverContext`, so `sql.Open` remains lazy and DSN errors continue to surface where they always have. A runnable sample is in `examples/connector`. Resolves [GitLab issue #253](https://gitlab.com/cznic/sqlite/-/issues/253), thanks Alessandro Segala (@ItalyPaleAle)!
     - Document that a caller-constructed `sqlite.Driver` is not the driver this package registers as `"sqlite"`. Its fields are unexported, so it starts with no functions, collations or connection hooks and the only way to give it any is its own `RegisterConnectionHook` method; the package-level `Register*` functions always apply to the registered driver. Connections such a `Driver` opens therefore run without the package-level functions and collations — and because a registered function silently replaces a SQLite built-in of the same name, a `Driver` you construct can evaluate `upper(x)`, `date(x)` and the like differently from one opened through `sql.Open`. Virtual table modules are the one exception: they are held process-globally and reach every `Driver`. Constructing one remains supported for the private-hook pattern — a driver registered under a name of its own with `sql.Register` so its connection hooks apply only to its own connections — and is otherwise best avoided in favour of `sql.Open` or `NewConnector`. Documentation only; no behavior changes.

 - 2026-07-20 v1.55.0:
     - Add `github.com/mattn/go-sqlite3`-compatible shorthand DSN query parameters to ease migration from that driver: `_busy_timeout`/`_timeout`, `_foreign_keys`/`_fk`, `_journal_mode`/`_journal`, `_synchronous`/`_sync`, `_auto_vacuum`/`_vacuum`, and `_query_only`, each setting the correspondingly named PRAGMA. Values are validated against the same set `mattn/go-sqlite3` accepts (case-insensitive) and an unrecognized value fails the connection with an error, so a typo such as `_synchronous=fu1l` or `_foreign_keys=yes_please` is reported rather than silently downgrading durability or dropping foreign-key enforcement. The keys are applied in a fixed order independent of their order in the DSN — `_busy_timeout` and `_auto_vacuum` before any `_pragma` values (`auto_vacuum` must be set before the database is first written), the rest after, and `_query_only` last — and where a key and its alias are both supplied the alias wins, matching `mattn/go-sqlite3`; selection is by presence rather than by value, so supplying the alias empty (`_foreign_keys=on&_fk=`) suppresses the PRAGMA rather than deferring to the primary key, again matching that driver. Behavior change to note: prior releases ignored these keys entirely, so a DSN carried over from a `mattn/go-sqlite3` setup changes in two ways. A recognized key that previously did nothing now takes effect — `_foreign_keys=on` begins enforcing constraints against data that may already violate them, `_journal_mode=wal` persistently converts the database file, and `_query_only=1` makes the connection read-only. And a value outside the accepted set now fails the connection with an error where the same DSN previously opened successfully — for example a duration-style `_busy_timeout=5s` or `_timeout=5000ms`, neither of which is the integer that key requires. Review such DSNs before upgrading. `_pragma` is unchanged and no pre-existing parameter changes meaning, though see the following entry for a change in when all of them are validated.
     - See [GitLab merge request #134](https://gitlab.com/cznic/sqlite/-/merge_requests/134), thanks Toni Spets (@beeper-hifi) and Ian Chechin!
     - Validate every DSN query parameter before applying any of them. Parameters were previously checked as each was reached, so a DSN whose later parameter was rejected had already executed the PRAGMAs ahead of it. Because `PRAGMA journal_mode` and `PRAGMA auto_vacuum` are persistent changes to the database file, a DSN such as `file:x.db?_journal_mode=wal&_synchronous=bogus` failed the connection and yet left `x.db` converted to WAL. A failed `Open` now leaves the database as it found it. This covers the pre-existing `_txlock`, `_timezone`, `_time_format`, `_time_integer_format`, `_inttotime` and `_texttotime` parameters as well as the shorthand keys above: all of them were validated only after the `_pragma` list had already run, so the same DSN shape — a valid `_pragma=journal_mode=wal` alongside a misspelled `_txlock` — converted the file before reporting the error. Only the values accepted for each parameter are unchanged; a DSN that opened successfully before still opens, and one that failed still fails with the same error. `_pragma` remains the sole exception, since its values are executed verbatim and cannot be checked in advance: a malformed `_pragma` is still rejected by SQLite as it runs, after any earlier `_pragma` in the list has taken effect.

 - 2026-07-15 v1.54.0:
     - Upgrade to [SQLite 3.53.3](https://sqlite.org/releaselog/3_53_3.html). This also bumps the pinned `modernc.org/libc` to v1.74.1; as always, downstream modules must pin the exact same `modernc.org/libc` version this module's `go.mod` pins (see [GitLab issue #177](https://gitlab.com/cznic/sqlite/-/issues/177)).
     - Under the opt-in `_texttotime` DSN parameter, best-effort parse date-shaped TEXT values from columns SQLite reports with an empty declared type — aggregates and expressions over a date column (`MAX(d)`, `COALESCE(d, ...)`, `upper(d)`, `d || ''`), subqueries, and typeless real columns (`CREATE TABLE t(x)`) — into `time.Time`, instead of delivering them as a raw string that `Scan` cannot store into a `*time.Time`. The existing declared `DATE`/`DATETIME`/`TIME`/`TIMESTAMP` path is unchanged; this only adds the empty-decltype case. The conversion is strictly best-effort: a value that does not parse as a time falls through to the original string, so no `Scan` that worked before can newly fail. `ColumnTypeScanType` continues to report `string` for empty-decltype columns, since the declared type cannot prove the column is temporal. Without `_texttotime` the behavior is byte-for-byte unchanged. Resolves [GitLab issue #248](https://gitlab.com/cznic/sqlite/-/issues/248).
     - See [GitLab merge request #133](https://gitlab.com/cznic/sqlite/-/merge_requests/133), thanks Ian Chechin!

 - 2026-06-21 v1.53.0:
     - Add **experimental** `netbsd/amd64` support, resolving the long-standing build break in [GitLab issue #246](https://gitlab.com/cznic/sqlite/-/issues/246). This target is intentionally **not yet listed among the supported platforms** in the package documentation: the port had been broken for years and is only now revived, and there is as yet no real-world experience running it under production workloads. Green CI is not the same as battle-tested — so while the full test suite (including the `pcache` and `vec` packages and the `-race` concurrency test) passes on NetBSD 10.1 / Go 1.26.3, and the entire upstream toolchain (`libc`, `cc`, `ccgo`, `libz`, `libtcl8.6`, `libsqlite3`, `libsqlite_vec`) is green on the NetBSD CI builder, the target is offered for evaluation only. If you run NetBSD, please exercise it with your own workloads and report back via #246; the intent is to promote it to a fully supported platform after a period of broader real-world testing (on the order of a month) elapses without surprises.
     - Implementation notes: the previously shipped `lib/sqlite_netbsd_amd64.go` was a stale old-generator transpile that no longer compiled (the `mu.enter`/`mu.leave` break in #246); it is replaced by a fresh new-generator transpile consistent with every other platform, and `modernc.org/sqlite/vec` (sqlite-vec) is vendored and auto-registers on netbsd. Correct operation requires the matching pinned `modernc.org/libc`, which carries two NetBSD-specific fixes found during this work: the `mmap(2)` `PAD`-argument ABI (without it, concurrent WAL access faults with SIGBUS in the WAL-index shared memory) and a working `abort(3)` (the prior stub left SQLite's crash-recovery `writecrash` test unable to terminate by signal). As usual, downstream modules must pin the exact `modernc.org/libc` version this module's `go.mod` pins.
     - See [GitLab merge request #82](https://gitlab.com/cznic/sqlite/-/merge_requests/82), thanks Leonardo Taccari (@iamleot) and Thomas Klausner (@_wiz_)!
     - Add **experimental** `freebsd/386` and `freebsd/arm` support. As with the `netbsd/amd64` target above, these two 32-bit FreeBSD ports are intentionally **not yet listed among the supported platforms** in the package documentation: `freebsd/386` previously shipped a stale, effectively untested SQLite 3.41 transpile, and `freebsd/arm` is entirely new, so neither has real-world production mileage yet. Both are now freshly transpiled at SQLite 3.53.2 consistent with every other platform, build cleanly, and pass the full test suite (core, WAL/concurrency, and the `vec` package) on the FreeBSD CI builders; they are offered for evaluation only. If you run 32-bit FreeBSD, please exercise these targets with your own workloads and report back — the intent is to promote `freebsd/386`, `freebsd/arm`, and `netbsd/amd64` to fully supported platforms in a future release cycle, once a period of broader real-world testing elapses without surprises.
     - Implementation notes: correct operation on `freebsd/arm` requires the matching pinned `modernc.org/libc` (v1.73.4), which fixes the per-arch `mmap(2)` `off_t` encoding for 32-bit FreeBSD; without it the WAL shared-memory mapping faults with SIGBUS under concurrent access, the same class of bug found on the netbsd port. As usual, downstream modules must pin the exact `modernc.org/libc` version this module's `go.mod` pins.
     - See [GitLab merge request #119](https://gitlab.com/cznic/sqlite/-/merge_requests/119), thanks Olivier Cochard-Labbé (@ocochard)!
     - Add a Go-facing wrapper for `SQLITE_CONFIG_PCACHE2`. `PageCache` is the factory and `Cache` the per-database instance, both idiomatic Go interfaces; `Page` exposes the raw `Buf` and `Extra` pointers that SQLite reads through the C pcache contract. `RegisterPageCache` and `MustRegisterPageCache` install the module process-globally before the first `sql.Open`; subsequent Open calls are gated through a one-shot `Xsqlite3_config(SQLITE_CONFIG_PCACHE2)` so a too-late Register returns `ErrPageCacheTooLate` rather than silently falling through to the built-in pcache1. The binding owns the `sqlite3_pcache_page` stub and re-consults the implementation on every Fetch, reusing the stub only when the returned `Page` value is unchanged, which keeps a bounded/evicting purgeable cache safe by construction.
     - See [GitLab merge request #126](https://gitlab.com/cznic/sqlite/-/merge_requests/126), thanks Ian Chechin!
     - Add `modernc.org/sqlite/pcache`, the reference page-cache implementation that accompanies the #126 `SQLITE_CONFIG_PCACHE2` wrapper. `pcache.New` returns a `*Pool` satisfying the `PageCache` interface; register it once with `sqlite.MustRegisterPageCache(pcache.New())` and every connection opened afterwards draws its pages from it. Each `Pool.Create` mints a fresh per-database `Cache`: a bounded, LRU-evicting page store that honours the `PRAGMA cache_size` soft cap and releases the least-recently-unpinned page when it must make room. Page memory — the `Buf` and `Extra` buffers SQLite reads through — is allocated with `libc.Xmalloc`/`libc.Xcalloc` and therefore lives off the Go heap, which keeps SQLite's interior pointer arithmetic on the page extras from tripping the race detector's checkptr enforcement. `Pool.Stats` reports aggregate lifetime counters (hits, misses, allocs, evictions, rekeys, truncates, caches) across every cache a Pool has created, so hit/miss/eviction behaviour is observable without instrumenting individual caches. Cross-connection page sharing is out of scope for now; each `Create` returns an independent per-database cache.
     - Validated end-to-end against the #126 stress workload (`cache_size=16`, 4000 BLOB rows with DELETE and `incremental_vacuum`, `integrity_check` clean under `-race`) and benchmarked for the memory-utilization goal tracked in [GitLab issue #204](https://gitlab.com/cznic/sqlite/-/issues/204).
     - See [GitLab merge request #127](https://gitlab.com/cznic/sqlite/-/merge_requests/127), thanks Ian Chechin!
     - Tighten the `modernc.org/sqlite/pcache` reference implementation per cznic's !127 review follow-ups. Adds `Stats.EasyRefusals`, a per-Pool counter for the cases where `FetchCreateEasy` returns nil at cap; SQLite reacts to a refusal by spilling dirty pages and retrying with `FetchCreateForce`, so the new field is a direct proxy for the I/O pressure the strict Easy contract imposes vs pcache1's recycle-without-spill behavior. `BenchmarkPoolEvictionChurn` was reworked to drive a rotating-residue DELETE (`k % 3 = i % 3`) and re-insert a matching batch each cycle so the spill pressure recurs and `easy-refusals/op` scales with `b.N` instead of capping at the seed's one-time first-cycle cost; both existing benchmarks now report `easy-refusals/op` alongside the page-allocs/evictions metrics. `Stats.Evictions` documentation was tightened to match the actual behavior (counts LRU eviction, `Unpin(discard=true)`, `Shrink` releases, and `Unpin(discard=false)` trimming back to target after a `FetchCreateForce` overcommit; bulk frees from `Truncate`, `Rekey` collisions, and `Destroy` are not counted). The `TestPoolRoundTripIntegrity` comment claiming the workload exercises `xRekey` ~15 times has been corrected; the SQL surface does not reliably emit xRekey here, and that codepath is covered by the unit tests instead.
     - See [GitLab merge request #130](https://gitlab.com/cznic/sqlite/-/merge_requests/130), thanks Ian Chechin!
     - Make `modernc.org/sqlite/pcache` `-race`-clean under SQLite's `cache=shared` mode. The pool already runs correctly under shared-cache because every callback into a given `Cache` is serialised internally by SQLite's `sqlite3BtreeEnter` on the `BtShared` mutex; verified empirically with a lock-free in-flight probe (max-in-flight = 1 on the canonical two-connection workload, 4 on a positive control with goroutines hitting the cache directly). However the Go race detector does not recognise SQLite's libc mutex as a happens-before edge and reports false-positive races on `Fetch` vs `Unpin` reads/writes of the per-cache state, which surfaces as `DATA RACE` failures for any user who registers the pool and runs their suite under `-race`. A `sync.Mutex` on the `cache` type is now taken on every public method (`SetSize`, `PageCount`, `Fetch`, `Unpin`, `Rekey`, `Truncate`, `Destroy`, `Shrink`), always. On the common non-shared-cache path the lock is uncontended (one atomic CAS per Lock/Unlock pair, negligible next to the SQLite work it bookends); on the shared-cache path it just rubber-stamps the order SQLite's `BtShared` mutex already established. A new `e2e_test.go` `TestSharedCacheTwoConns_Integrity` drives two `sql.Conn` against the same `cache=shared` URI with concurrent writers and asserts `PRAGMA integrity_check = ok` under `-race`; passes cleanly with the lock, would surface the false-positive without it. Design notes live in `pcache/sharing.go`.
     - See [GitLab merge request #131](https://gitlab.com/cznic/sqlite/-/merge_requests/131), thanks Ian Chechin!
     - Add a Go wrapper for `sqlite3_db_status`, the per-connection runtime counters (cache hit/miss/write/spill rates, schema and prepared-statement memory, lookaside usage, deferred foreign keys). `DBStatus` is an interface implemented by the driver connection and reached through the `database/sql` escape hatch `(*sql.Conn).Raw()`, mirroring the existing `FileControl` surface; `DBStatusOp` is a distinct typed enum of the `SQLITE_DBSTATUS_*` verbs so a counter from a different op family will not compile in its place. `Status(op, reset)` returns the `(current, high)` pair and optionally resets the counter. This also lets `modernc.org/sqlite/pcache` measure real I/O instead of the `EasyRefusals` proxy: the new `BenchmarkPoolSpillIO` reads the pager-level `SQLITE_DBSTATUS_CACHE_SPILL`/`_CACHE_WRITE` counters, which the pager maintains identically for pcache1 and the pool, making the pcache1-vs-pool comparison cznic raised on the !127 review a genuine apples-to-apples measurement. On the rotating-residue eviction-churn workload at `cache_size=16` the pool spills ~3.5x more than pcache1 (cache-spill/op 31.96 vs 8.96) for ~3% more page writes (cache-write/op 450 vs 436) at identical hit/miss, quantifying the I/O cost of the strict Easy contract that `EasyRefusals` only proxied.
     - See [GitLab merge request #132](https://gitlab.com/cznic/sqlite/-/merge_requests/132), thanks Ian Chechin!
     - Add an opt-in `_dqs` DSN query parameter that disables SQLite's double-quoted string literal compatibility quirk on a per-connection basis. When `_dqs=0` (or any `strconv.ParseBool` false value) is supplied, the driver calls `sqlite3_db_config` with `SQLITE_DBCONFIG_DQS_DDL` and `SQLITE_DBCONFIG_DQS_DML` set to off before any statement is prepared, so a double-quoted identifier that fails to resolve raises a parse error instead of silently falling back to a string literal. Absence of the parameter, or `_dqs=1`, leaves SQLite's default behavior unchanged; existing DSNs continue to work byte-for-byte. Resolves [GitLab issue #61](https://gitlab.com/cznic/sqlite/-/issues/61).
     - See [GitLab merge request #128](https://gitlab.com/cznic/sqlite/-/merge_requests/128), thanks Ian Chechin!
     - Add an opt-in `_error_rc` DSN query parameter for clearer error reporting on open-time failures. When `_error_rc=1` (or any `strconv.ParseBool` true value) is supplied, error strings synthesised from a `(rc, db)` pair only append `sqlite3_errmsg(db)` when `sqlite3_extended_errcode(db)` is consistent with the operation rc (full match first, primary code `&0xff` as fallback). On mismatch the canonical `sqlite3_errstr(rc)` is used alone, so an open-time `SQLITE_CANTOPEN` no longer carries the temporary handle's stale "out of memory" errmsg. Absence of the parameter, or `_error_rc=0`, preserves the legacy "errstr: errmsg" form byte-for-byte; existing callers that parse error strings are unaffected. The driver's `*Error.Code()` returns the same SQLite result code in both modes. Parsed before `sqlite3_open_v2` so open-time errors are covered. Resolves [GitLab issue #230](https://gitlab.com/cznic/sqlite/-/issues/230).
     - See [GitLab merge request #129](https://gitlab.com/cznic/sqlite/-/merge_requests/129), thanks Ian Chechin!

 - 2026-06-06 v1.52.0:
     -  Upgrade to [SQLite 3.53.2](https://sqlite.org/releaselog/3_53_2.html).
     - Add `Backup.Remaining` and `Backup.PageCount`, thin wrappers around the existing `sqlite3_backup_remaining` and `sqlite3_backup_pagecount` C symbols. Together they expose the per-`Step` progress counters that the underlying backup object already maintains, enabling progress reporting during online backups without dropping to `modernc.org/sqlite/lib` directly.
     - See [GitLab merge request #122](https://gitlab.com/cznic/sqlite/-/merge_requests/122), thanks Ian Chechin!
     - Drop the redundant second copy in `(*conn).columnText`, the path that backs every `Rows.Scan` into a Go `string` for a TEXT column. The value's bytes are still copied once out of SQLite-owned memory into a fresh Go buffer; that buffer is then reinterpreted as the result string with `unsafe.String` rather than copied a second time by the implicit `string([]byte)` conversion. This removes one allocation per TEXT value per row and roughly halves the bytes allocated on that path; on the new `BenchmarkColumnTextScan` cases it is ~13–20% faster for payloads of 256 B and larger, with no measurable change for very short strings. Purely internal: no API or behavioral change, and the returned string never aliases SQLite's buffer.
     - See [GitLab merge request #123](https://gitlab.com/cznic/sqlite/-/merge_requests/123), thanks Ian Chechin!
     - Cache each result column's declared type once per result set in `newRows` instead of recomputing it on every row. The TEXT branch of `Rows.Next` calls `ColumnTypeDatabaseTypeName` for every TEXT column on every row (independent of any DSN flag), which previously did a `libc.GoString` + `strings.ToUpper` each time; that lookup is now a single index into a cached, pre-uppercased `[]string`, and `ColumnTypeScanType` reads the same cache and drops its per-call `strings.ToLower`. The declared type is fixed for the lifetime of a prepared statement, so the C round-trip is paid once per column rather than once per column per row, removing exactly 1 alloc + 8 B per TEXT column per row from the `Next` hot path. The new `BenchmarkTextToTimeScan` cases show ~7% faster on a 1000-row DATETIME SELECT under `_texttotime=1`. Purely internal: `ColumnTypeDatabaseTypeName` and `ColumnTypeScanType` return identical values, no API or behavioral change.
     - See [GitLab merge request #124](https://gitlab.com/cznic/sqlite/-/merge_requests/124), thanks Ian Chechin!
     - Cache, per result column, the `parseTimeFormats` index that first parsed a TEXT-stored DATE/DATETIME/TIMESTAMP value, and try that format first on later rows instead of re-walking the list from the top. `(*conn).parseTime` previously ran `time.Parse` down the format list on every such row; for the canonical SQLite TEXT datetime format every row paid two failed `time.Parse` attempts — each allocating a `*time.ParseError` — before the match. On a 1000-row DATETIME TEXT SELECT this cuts ~50% of allocs/op and ~57% of B/op and is ~37% faster. The fall-through chain is preserved exactly: the seven formats are mutually exclusive, so the cached hint can never select a different match than the in-order scan, and the parsed `driver.Value` is identical to before. Purely internal: no API or behavioral change.
     - See [GitLab merge request #125](https://gitlab.com/cznic/sqlite/-/merge_requests/125), thanks Ian Chechin!

 - 2026-05-28 v1.51.0:
     - Pool the `[]driver.Value` slice passed to scalar/aggregate UDF callbacks and to vtab `Filter`/`Insert`/`Update` callbacks, eliminating the dominant per-row allocation on UDF-heavy queries. Benchmarks on a 1000-row, 3-arg noop scalar UDF show ~40% fewer bytes/op and ~15% fewer allocs/op.
     - Document the matching "arguments are not valid past return" contract on `vtab.Cursor.Filter` and `vtab.Updater.Insert`/`Update`, consistent with the existing rule for `FunctionImpl.Scalar` / `AggregateFunction.Step` / `WindowInverse`.
     - Resolves [GitLab issue #226](https://gitlab.com/cznic/sqlite/-/issues/226). See [GitLab merge request #114](https://gitlab.com/cznic/sqlite/-/merge_requests/114), thanks Ian Chechin!
     - Add `FileControl.FileControlDataVersion`, a wrapper around `SQLITE_FCNTL_DATA_VERSION` for observing pager-cache data-version changes, including those made on the same connection. Useful as a primitive for application-level cache invalidation.
     - Exposed via the idiomatic `database/sql` escape hatch `(*sql.Conn).Raw()`, consistent with the existing `FileControlPersistWAL`.                                                                    
     - See [GitLab merge request #115](https://gitlab.com/cznic/sqlite/-/merge_requests/115), thanks Ian Chechin!
     - Fix a regression where in-memory connections (`:memory:`, `file::memory:`, shared-cache memory URIs) were discarded by `database/sql` after a context-cancelled query, taking the entire in-memory store with them. The fix for #198 had added an `sqlite3_is_interrupted` check to the connection validator that mistakenly applied to in-memory connections too, re-introducing the bug originally fixed by !74. File-backed connections keep the existing behaviour and are still discarded after an interrupt.
     - Resolves [GitLab issue #196](https://gitlab.com/cznic/sqlite/-/issues/196). See [GitLab merge request #116](https://gitlab.com/cznic/sqlite/-/merge_requests/116), thanks Ian Chechin!
     - Add an opt-in `FunctionImpl.VolatileArgs` flag that hands TEXT and BLOB arguments to scalar and aggregate UDF callbacks as zero-copy views (`unsafe.String`/`unsafe.Slice`) over SQLite's own value buffers, eliminating the per-argument `libc.GoString`/`make([]byte)` copy that the #226 slice-pooling left as the remaining per-row allocation. On the same 1000-row, 3-arg (INTEGER/TEXT/BLOB) noop scalar UDF this removes a further ~35% of allocs/op and ~11% of bytes/op on top of #226.
     - The views are valid only for the duration of the callback and must not be retained past return or across rows; a callback that needs to keep a value must copy it. With `VolatileArgs` unset (the default) arguments keep the existing copied, caller-owned semantics, so the flag is fully backward compatible; it has no effect on integer, float, time, or NULL arguments.
     - See [GitLab merge request #120](https://gitlab.com/cznic/sqlite/-/merge_requests/120), thanks Ian Chechin!
     - Extend the opt-in `VolatileArgs` zero-copy TEXT/BLOB argument access from #120 to the virtual-table `Cursor.Filter` (`xFilter`) and `Updater.Insert`/`Update` (`xUpdate`) callbacks. A `vtab.Module` opts in by implementing the new optional `vtab.VolatileArgsOpter` interface (`VolatileArgs() bool`); the flag is read once at module registration and shared by every table created from it. On a vtab call carrying one TEXT and one BLOB argument this removes 2 allocs/op (one `libc.GoString`, one `make([]byte)`) on each of the Filter and Update paths.
     - The same safety contract as #120 applies: the views are valid only for the duration of the callback and must not be retained past return or across rows; a callback that needs to keep a value must copy it. Modules that do not implement `VolatileArgsOpter` (the default for all existing modules) are byte-for-byte unchanged, and the flag has no effect on integer, float, time, or NULL arguments.
     - See [GitLab merge request #121](https://gitlab.com/cznic/sqlite/-/merge_requests/121), thanks Ian Chechin!

 - 2026-05-10 v1.50.1:
     - Upgrade to [SQLite 3.53.1](https://sqlite.org/releaselog/3_53_1.html).

 - 2026-04-24 v1.50.0:
     - Upgrade to sqlite-vec [v0.1.9](https://github.com/asg017/sqlite-vec/releases/tag/v0.1.9).
     - Introduce `ColumnInfo`, enabling dynamic query builders and ORMs to retrieve underlying SQLite C-API metadata (`OriginName`, `TableName`, `DatabaseName`, and `DeclType`).
     - This feature is exposed via the idiomatic `database/sql` escape hatch `(*sql.Conn).Raw()`, avoiding custom statement handles and keeping the standard library workflow intact.
     - See [GitLab merge request #113](https://gitlab.com/cznic/sqlite/-/merge_requests/113), thanks Josh Bleecher Snyder!
 
 - 2026-04-17 v1.49.0: Upgrade to [SQLite 3.53.0](https://sqlite.org/releaselog/3_53_0.html).
     - Added `-DSQLITE_ENABLE_DBPAGE_VTAB` to the transpilation. See ["The SQLITE_DBPAGE Virtual Table"](https://www.sqlite.org/dbpage.html) for details.

 - 2026-04-06 v1.48.2:
     - Fix ABI mapping mismatch in the pre-update hook trampoline that caused silent truncation of large 64-bit RowIDs.
     - Ensure the Go trampoline signature correctly aligns with the public `sqlite3_preupdate_hook` C API, preventing data corruption for high-entropy keys (e.g., Snowflake IDs).
     - See [GitLab merge request #98](https://gitlab.com/cznic/sqlite/-/merge_requests/98), thanks Josh Bleecher Snyder!
     - Fix the memory allocator used in `(*conn).Deserialize`.
     - Replace `tls.Alloc` with `sqlite3_malloc64` to prevent internal allocator corruption. This ensures the buffer is safely owned by SQLite, which may resize or free it due to the `SQLITE_DESERIALIZE_RESIZEABLE` and `SQLITE_DESERIALIZE_FREEONCLOSE` flags.
     - Prevent a memory leak by properly freeing the allocated buffer if fetching the main database name fails before handing ownership to SQLite.
     - See [GitLab merge request #100](https://gitlab.com/cznic/sqlite/-/merge_requests/100), thanks Josh Bleecher Snyder!
     - Fix `(*conn).Deserialize` to explicitly reject `nil` or empty byte slices.
     - Prevent silent database disconnection and connection pool corruption caused by SQLite's default behavior when `sqlite3_deserialize` receives a 0-length buffer.
     - See [GitLab merge request #101](https://gitlab.com/cznic/sqlite/-/merge_requests/101), thanks Josh Bleecher Snyder!
     - Fix `commitHookTrampoline` and `rollbackHookTrampoline` signatures by removing the unused `pCsr` parameter.
     - Aligns internal hook callbacks accurately with the underlying SQLite C API, cleaning up the code to prevent potential future confusion or bugs.
     - See [GitLab merge request #102](https://gitlab.com/cznic/sqlite/-/merge_requests/102), thanks Josh Bleecher Snyder!
     - Fix `checkptr` instrumentation failures during `go test -race` when registering and using virtual tables (`vtab`).
     - Allocate `sqlite3_module` instances using the C allocator (`libc.Xcalloc`) instead of the Go heap. This ensures transpiled C code can safely perform pointer operations on the struct without tripping Go's pointer checks.
     - See [GitLab merge request #103](https://gitlab.com/cznic/sqlite/-/merge_requests/103), thanks Josh Bleecher Snyder!
     - Fix data race on `mutex.id` in the `mutexTry` non-recursive path.
     - Ensure consistent atomic writes (`atomic.StoreInt32`) to prevent data races with atomic loads in `mutexHeld` and `mutexNotheld` during concurrent execution.
     - See [GitLab merge request #104](https://gitlab.com/cznic/sqlite/-/merge_requests/104), thanks Josh Bleecher Snyder!
     - Fix resource leak in `(*Backup).Commit` where the destination connection was not closed on error.
     - Ensure `dstConn` is properly closed when `sqlite3_backup_finish` fails, preventing file descriptor, TLS, and memory leaks.
     - See [GitLab merge request #105](https://gitlab.com/cznic/sqlite/-/merge_requests/105), thanks Josh Bleecher Snyder!
     - Fix `Exec` to fully drain rows when encountering `SQLITE_ROW`, preventing silent data loss in DML statements.
     - Previously, `Exec` aborted after the first row, meaning `INSERT`, `UPDATE`, or `DELETE` statements with a `RETURNING` clause would fail to process subsequent rows. The execution path now correctly loops until `SQLITE_DONE` and properly respects context cancellations during the drain loop, fully aligning with native C `sqlite3_exec` semantics.
     - See [GitLab merge request #106](https://gitlab.com/cznic/sqlite/-/merge_requests/106), thanks Josh Bleecher Snyder!
     - Fix "Shadowed err value (stmt.go)".
     - See [GitLab issue #249](https://gitlab.com/cznic/sqlite/-/work_items/249), thanks Emrecan BATI!
     - Fix silent omission of virtual table savepoint callbacks by correctly setting the sqlite3_module version.
     - See [GitLab merge request #107](https://gitlab.com/cznic/sqlite/-/merge_requests/107), thanks Josh Bleecher Snyder!
     - Fix `vfsRead` to properly handle partial and fragmented reads from `io.Reader`.
     - Replace `f.Read` with `io.ReadFull` to ensure the buffer is fully populated, preventing premature `SQLITE_IOERR_SHORT_READ` errors on valid mid-stream partial reads. Unread tail bytes at EOF are now efficiently zero-filled using the built-in `clear` function.
     - See [GitLab merge request #108](https://gitlab.com/cznic/sqlite/-/merge_requests/108), thanks Josh Bleecher Snyder!
     - Refactor internal error formatting to safely handle uninitialized or closed database pointers.
     - Prevent a misleading "out of memory" error message when an operation fails and the underlying SQLite database handle is `NULL` (`db == 0`).
     - See [GitLab merge request #109](https://gitlab.com/cznic/sqlite/-/merge_requests/109), thanks Josh Bleecher Snyder!
     - Fix error handling in database backup and restore initialization (`sqlite3_backup_init`).
     - Ensure error codes and messages are accurately read from the destination database handle rather than hardcoding the source or remote handle. This prevents swallowed errors or mismatched "not an error" messages when a backup or restore operation fails to start.
     - See [GitLab merge request #111](https://gitlab.com/cznic/sqlite/-/merge_requests/111), thanks Josh Bleecher Snyder!
     - Fix database handle and C-heap memory leaks when `sqlite3_open_v2` fails.
     - Ensure `sqlite3_close_v2` is called on the partially allocated database handle during a failed open, and explicitly close `libc.TLS` in `newConn` to prevent resource leakage.
     - Prevent misleading "out of memory" error messages on failed connections by correctly extracting the exact error string from the allocated handle before it is closed.
     - See [GitLab merge request #112](https://gitlab.com/cznic/sqlite/-/merge_requests/112), thanks Josh Bleecher Snyder!

 - 2026-04-03 v1.48.1:
     - Fix memory leaks and double-free vulnerabilities in the multi-statement query execution path.
     - Ensure bind-parameter allocations are reliably freed via strict ownership transfer if an error occurs mid-loop or if multiple statements bind parameters.
     - Fix a resource leak where a subsequent statement's error could orphan a previously generated `rows` object without closing it, leaking the prepared statement handle.
     - See [GitLab merge request #96](https://gitlab.com/cznic/sqlite/-/merge_requests/96), thanks Josh Bleecher Snyder!

 - 2026-03-27 v1.48.0:
     - Add `_timezone` DSN query parameter to apply IANA timezones (e.g., "America/New_York") to both reads and writes.
     - Writes will convert `time.Time` values to the target timezone before formatting as a string.
     - Reads will interpret timezone-less strings as being in the target timezone.
     - Does not impact `_inttotime` integer values, which will always safely evaluate as UTC.
     - Add support for `_time_format=datetime` URI parameter to format `time.Time` values identically to SQLite's native `datetime()` function and `CURRENT_TIMESTAMP` (`YYYY-MM-DD HH:MM:SS`).
     - See [GitLab merge request #94](https://gitlab.com/cznic/sqlite/-/merge_requests/94) and [GitLab merge request #95](https://gitlab.com/cznic/sqlite/-/merge_requests/95), thanks Josh Bleecher Snyder!

 - 2026-03-17 v1.47.0: Add CGO-free version of the vector extensions from https://github.com/asg017/sqlite-vec. See `vec_test.go` for example usage. From the GitHub project page:
     - **Important:** sqlite-vec is a pre-v1, so expect breaking changes!
     - Store and query float, int8, and binary vectors in vec0 virtual tables
     - Written in pure C, no dependencies, runs anywhere SQLite runs (Linux/MacOS/Windows, in the browser with WASM, Raspberry Pis, etc.)
     - Store non-vector data in metadata, auxiliary, or partition key columns
     - See [GitLab merge request #93](https://gitlab.com/cznic/sqlite/-/merge_requests/93), thanks Zhenghao Zhang!

 - 2026-03-16 v1.46.2: Upgrade to  [SQLite 3.51.3](https://sqlite.org/releaselog/3_51_3.html).

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
