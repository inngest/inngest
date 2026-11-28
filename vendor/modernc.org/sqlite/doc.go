// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sqlite is a sql/database driver using a CGo-free port of the C
// SQLite3 library.
//
// SQLite is an in-process implementation of a self-contained, serverless,
// zero-configuration, transactional SQL database engine.
//
// # Pluggable page cache
//
// The package exposes a Go-facing wrapper for SQLite's
// SQLITE_CONFIG_PCACHE2 mechanism. Applications can supply their own
// page cache implementation by registering a [PageCache] before the
// first [sql.Open] via [RegisterPageCache]. See the docstrings
// on [PageCache], [Cache], and [Page] for the contract; the binding
// owns the sqlite3_pcache_page stub on behalf of the implementation
// and re-consults Cache.Fetch on every SQLite request, so a bounded
// and evicting purgeable cache works as the C contract intends.
//
// # Fragile modernc.org/libc dependency
//
// When you import this package you should use in your go.mod file the exact
// same version of modernc.org/libc as seen in the go.mod file of this
// repository.
//
// See the discussion at https://gitlab.com/cznic/sqlite/-/issues/177 for more details.
//
// # Thanks
//
// This project is sponsored by Schleibinger Geräte Teubert u. Greim GmbH by
// allowing one of the maintainers to work on it also in office hours.
//
// # Supported platforms and architectures
//
// These combinations of GOOS and GOARCH are currently supported
//
//	OS      Arch    SQLite version
//	------------------------------
//	darwin	amd64   3.53.3
//	darwin	arm64   3.53.3
//	freebsd	386     3.53.3
//	freebsd	amd64   3.53.3
//	freebsd	arm     3.53.3
//	freebsd	arm64   3.53.3
//	linux	386     3.53.3
//	linux	amd64   3.53.3
//	linux	arm     3.53.3
//	linux	arm64   3.53.3
//	linux	loong64 3.53.3
//	linux	ppc64le 3.53.3
//	linux	riscv64 3.53.3
//	linux	s390x   3.53.3
//	netbsd	amd64   3.53.3
//	openbsd	amd64   3.53.3
//	openbsd	arm64   3.53.3
//	windows	386     3.53.3
//	windows	amd64   3.53.3
//	windows	arm64   3.53.3
//
// # Benchmarks
//
// [The SQLite Drivers Benchmarks Game]
//
// # Builders
//
// Builder results available at:
//
// https://modern-c.appspot.com/-/builder/?importpath=modernc.org%2fsqlite
//
// # Connecting to a database
//
// To access a Sqlite database do something like
//
//	import (
//		"database/sql"
//
//		_ "modernc.org/sqlite"
//	)
//
//	...
//
//
//	db, err := sql.Open("sqlite", dsnURI)
//
//	...
//
// [NewConnector] is an alternative entry point returning a
// [driver.Connector] for use with [sql.OpenDB]. It opens the same
// connections sql.Open does, from the same driver, and exists for callers that
// need to interpose on them -- tracing, metrics, or connection-scoped setup --
// which sql.Open gives no access to. See its docstring for an example.
//
// # Debug and development versions
//
// The transpiled SQLite sources under lib/, and the sqlite-vec sources under
// vec/, are not generated in this repository. They are produced by
// modernc.org/libsqlite3 and modernc.org/libsqlite_vec respectively, which own
// the transpilation and the SQLite compile-time options it uses, and are
// copied here by
//
//	$ make vendor
//
// which reads them from checkouts of those two repositories placed next to
// this one. To build a debug or otherwise modified version, adjust the
// compile-time options in modernc.org/libsqlite3, regenerate there with 'make
// generate', and vendor the result here.
//
// # Hacking
//
// This is an example of how to use the debug logs in modernc.org/libc when hunting a bug.
//
//	0:jnml@e5-1650:~/src/modernc.org/sqlite$ git status
//	On branch master
//	Your branch is up to date with 'origin/master'.
//
//	nothing to commit, working tree clean
//	0:jnml@e5-1650:~/src/modernc.org/sqlite$ git log -1
//	commit df33b8d15107f3cc777799c0fe105f74ef499e62 (HEAD -> master, tag: v1.21.1, origin/master, origin/HEAD, wips, ok)
//	Author: Jan Mercl <0xjnml@gmail.com>
//	Date:   Mon Mar 27 16:18:28 2023 +0200
//
//	    upgrade to SQLite 3.41.2
//	0:jnml@e5-1650:~/src/modernc.org/sqlite$ rm -f /tmp/libc.log ; go test -v -tags=libc.dmesg -run TestScalar ; ls -l /tmp/libc.log
//	test binary compiled for linux/amd64
//	=== RUN   TestScalar
//	--- PASS: TestScalar (0.09s)
//	PASS
//	ok  modernc.org/sqlite 0.128s
//	-rw-r--r-- 1 jnml jnml 76 Apr  6 11:22 /tmp/libc.log
//	0:jnml@e5-1650:~/src/modernc.org/sqlite$ cat /tmp/libc.log
//	[10723 sqlite.test] 2023-04-06 11:22:48.288066057 +0200 CEST m=+0.000707150
//	0:jnml@e5-1650:~/src/modernc.org/sqlite$
//
// The /tmp/libc.log file is created as requested. No useful messages there because none are enabled in libc. Let's try to enable Xwrite as an example.
//
//	0:jnml@e5-1650:~/src/modernc.org/libc$ git status
//	On branch master
//	Your branch is up to date with 'origin/master'.
//
//	Changes not staged for commit:
//	  (use "git add <file>..." to update what will be committed)
//	  (use "git restore <file>..." to discard changes in working directory)
//	modified:   libc_linux.go
//
//	no changes added to commit (use "git add" and/or "git commit -a")
//	0:jnml@e5-1650:~/src/modernc.org/libc$ git log -1
//	commit 1e22c18cf2de8aa86d5b19b165f354f99c70479c (HEAD -> master, tag: v1.22.3, origin/master, origin/HEAD)
//	Author: Jan Mercl <0xjnml@gmail.com>
//	Date:   Wed Feb 22 20:27:45 2023 +0100
//
//	    support sqlite 3.41 on linux targets
//	0:jnml@e5-1650:~/src/modernc.org/libc$ git diff
//	diff --git a/libc_linux.go b/libc_linux.go
//	index 1c2f482..ac1f08d 100644
//	--- a/libc_linux.go
//	+++ b/libc_linux.go
//	@@ -332,19 +332,19 @@ func Xwrite(t *TLS, fd int32, buf uintptr, count types.Size_t) types.Ssize_t {
//	                var n uintptr
//	                switch n, _, err = unix.Syscall(unix.SYS_WRITE, uintptr(fd), buf, uintptr(count)); err {
//	                case 0:
//	-                       // if dmesgs {
//	-                       //      // dmesg("%v: %d %#x: %#x\n%s", origin(1), fd, count, n, hex.Dump(GoBytes(buf, int(n))))
//	-                       //      dmesg("%v: %d %#x: %#x", origin(1), fd, count, n)
//	-                       // }
//	+                       if dmesgs {
//	+                               // dmesg("%v: %d %#x: %#x\n%s", origin(1), fd, count, n, hex.Dump(GoBytes(buf, int(n))))
//	+                               dmesg("%v: %d %#x: %#x", origin(1), fd, count, n)
//	+                       }
//	                        return types.Ssize_t(n)
//	                case errno.EAGAIN:
//	                        // nop
//	                }
//	        }
//
//	-       // if dmesgs {
//	-       //      dmesg("%v: fd %v, count %#x: %v", origin(1), fd, count, err)
//	-       // }
//	+       if dmesgs {
//	+               dmesg("%v: fd %v, count %#x: %v", origin(1), fd, count, err)
//	+       }
//	        t.setErrno(err)
//	        return -1
//	 }
//	0:jnml@e5-1650:~/src/modernc.org/libc$
//
// We need to tell the Go build system to use our local, patched/debug libc.
// 'make work' sets up a go.work covering this and the sibling repositories;
// by hand it is:
//
//	0:jnml@e5-1650:~/src/modernc.org/sqlite$ go work use $(go env GOPATH)/src/modernc.org/libc
//	0:jnml@e5-1650:~/src/modernc.org/sqlite$ go work use .
//
// And run the test again:
//
//	0:jnml@e5-1650:~/src/modernc.org/sqlite$ rm -f /tmp/libc.log ; go test -v -tags=libc.dmesg -run TestScalar ; ls -l /tmp/libc.log
//	test binary compiled for linux/amd64
//	=== RUN   TestScalar
//	--- PASS: TestScalar (0.26s)
//	PASS
//	ok   modernc.org/sqlite 0.285s
//	-rw-r--r-- 1 jnml jnml 918 Apr  6 11:29 /tmp/libc.log
//	0:jnml@e5-1650:~/src/modernc.org/sqlite$ cat /tmp/libc.log
//	[11910 sqlite.test] 2023-04-06 11:29:13.143589542 +0200 CEST m=+0.000689270
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 8 0x200: 0x200
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 8 0xc: 0xc
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 7 0x1000: 0x1000
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 7 0x1000: 0x1000
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 8 0x200: 0x200
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 8 0x4: 0x4
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 8 0x1000: 0x1000
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 8 0x4: 0x4
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 8 0x4: 0x4
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 8 0x1000: 0x1000
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 8 0x4: 0x4
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 8 0xc: 0xc
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 7 0x1000: 0x1000
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 7 0x1000: 0x1000
//	0:jnml@e5-1650:~/src/modernc.org/sqlite$
//
// # Sqlite documentation
//
// See https://sqlite.org/docs.html
//
// [The SQLite Drivers Benchmarks Game]: https://pkg.go.dev/modernc.org/sqlite-bench#readme-tl-dr-scorecard
package sqlite // import "modernc.org/sqlite"
