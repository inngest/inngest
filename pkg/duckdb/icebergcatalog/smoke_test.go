package icebergcatalog

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/apache/iceberg-go"
	sqlcat "github.com/apache/iceberg-go/catalog/sql"
	"github.com/inngest/inngest/pkg/duckdb/driver"
)

// TestDuckDBSmoke starts the catalog on sqlite + a temp dir warehouse and
// drives the purego DuckDB worker against it (skips w/o one, same as the rest
// of pkg/duckdb). It needs network on first run for INSTALL iceberg/httpfs
// unless they're already cached.
//
// INNGEST_ICEBERG_SMOKE_DIR keeps the catalog/warehouse around for poking
// at, INNGEST_ICEBERG_SMOKE_BODIES=1 logs request bodies.
func TestDuckDBSmoke(t *testing.T) {
	bin := driver.RequireDuckDBBinary(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dir := os.Getenv("INNGEST_ICEBERG_SMOKE_DIR") // keep artifacts for poking at
	if dir == "" {
		dir = t.TempDir()
	}
	warehouse := filepath.Join(dir, "warehouse")
	if err := os.MkdirAll(warehouse, 0o755); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "catalog.db")+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	cat, err := sqlcat.NewCatalog("inngest", db, sqlcat.SQLite, iceberg.Properties{"warehouse": warehouse})
	if err != nil {
		t.Fatal(err)
	}

	var (
		mu   sync.Mutex
		reqs []string
	)
	const token = "smoke-token"
	srv := New(cat, Options{
		Warehouse: warehouse,
		Token:     token,
		CatalogDB: testCatalogDB(db, warehouse),
		OnRequest: func(method, path string, status int) {
			mu.Lock()
			defer mu.Unlock()
			reqs = append(reqs, fmt.Sprintf("%s %s -> %d", method, path, status))
		},
	})
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		t.Logf("---- requests (%d) ----\n%s", len(reqs), strings.Join(reqs, "\n"))
	})
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	endpoint := "http://" + l.Addr().String()
	debugBodies := os.Getenv("INNGEST_ICEBERG_SMOKE_BODIES") != ""
	go func() {
		_ = http.Serve(l, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if debugBodies && r.Body != nil {
				b, _ := io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewReader(b))
				t.Logf("BODY %s %s: %s", r.Method, r.URL.Path, b)
			}
			srv.ServeHTTP(w, r)
		}))
	}()
	t.Cleanup(func() { _ = l.Close() })

	openWorker := func() (*sql.DB, error) {
		wdb, err := driver.Open(ctx, driver.Options{
			BinaryPath: bin,
			DBFile:     ":memory:",
		})
		if err != nil {
			return nil, err
		}
		setupStmts := []string{
			"INSTALL iceberg; LOAD iceberg;",
			"INSTALL httpfs; LOAD httpfs;",
			fmt.Sprintf("CREATE OR REPLACE SECRET inngest_iceberg (TYPE iceberg, TOKEN '%s');", token),
			fmt.Sprintf("ATTACH IF NOT EXISTS 'inngest' AS lake (TYPE iceberg, URI '%s', SECRET inngest_iceberg);", endpoint),
		}
		for _, stmt := range setupStmts {
			if _, err := wdb.ExecContext(ctx, stmt); err != nil {
				_ = wdb.Close()
				return nil, err
			}
		}
		return wdb, nil
	}

	execCSV := func(conn *sql.Conn, stmts []string) (string, error) {
		var out strings.Builder
		for _, raw := range stmts {
			s := strings.TrimSpace(raw)
			if s == "" {
				continue
			}
			rows, err := conn.QueryContext(ctx, s)
			if err != nil {
				return out.String(), err
			}
			cols, err := rows.Columns()
			if err != nil {
				_ = rows.Close()
				return out.String(), err
			}
			if len(cols) > 0 {
				out.WriteString(strings.Join(cols, ","))
				out.WriteByte('\n')
				vals := make([]any, len(cols))
				ptrs := make([]any, len(cols))
				for i := range vals {
					ptrs[i] = &vals[i]
				}
				for rows.Next() {
					if err := rows.Scan(ptrs...); err != nil {
						_ = rows.Close()
						return out.String(), err
					}
					parts := make([]string, len(vals))
					for i, v := range vals {
						parts[i] = fmt.Sprint(v)
					}
					out.WriteString(strings.Join(parts, ","))
					out.WriteByte('\n')
				}
				if err := rows.Err(); err != nil {
					_ = rows.Close()
					return out.String(), err
				}
			}
			_ = rows.Close()
		}
		return out.String(), nil
	}

	run := func(name string, stmts ...string) (string, error) {
		t.Helper()
		wdb, err := openWorker()
		if err != nil {
			return "", err
		}
		defer wdb.Close()
		conn, err := wdb.Conn(ctx)
		if err != nil {
			return "", err
		}
		defer conn.Close()
		out, err := execCSV(conn, stmts)
		t.Logf("---- %s ----\n%s", name, out)
		return out, err
	}
	must := func(name string, stmts ...string) string {
		t.Helper()
		out, err := run(name, stmts...)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return out
	}

	must("setup",
		`CREATE SCHEMA IF NOT EXISTS lake.app;`,
		`CREATE TABLE lake.app.runs (
  id VARCHAR,
  ts TIMESTAMPTZ,
  tags VARCHAR[],
  steps STRUCT(name VARCHAR, ms BIGINT)[]
);`,
	)

	// VARIANT needs Iceberg format-version 3, duckdb defaults to 2.
	if out, err := run("variant-v2",
		`CREATE TABLE lake.app.variant_v2 (id INT, v VARIANT);`,
		`INSERT INTO lake.app.variant_v2 VALUES (1, 42::VARIANT);`,
	); err == nil {
		t.Fatalf("expected VARIANT on a v2 table to fail, got:\n%s", out)
	}
	must("variant-v3",
		`CREATE TABLE lake.app.variant_v3 (id INT, v VARIANT) WITH ('format-version' = '3');`,
		`INSERT INTO lake.app.variant_v3 VALUES (1, 42::VARIANT), (2, 'x'::VARIANT), (3, {'a': [1,2]}::VARIANT);`,
	)
	// Fresh process so this reads back through LoadTable.
	if out := must("variant-v3-reread", `SELECT id, variant_typeof(v) AS t FROM lake.app.variant_v3 ORDER BY id;`); !strings.Contains(out, "1,INT32") || !strings.Contains(out, "3,OBJECT") {
		t.Fatalf("unexpected variant read back:\n%s", out)
	}

	// One duckdb txn -> one catalog commit, but a snapshot per statement.
	must("insert-txn",
		`BEGIN;`,
		`INSERT INTO lake.app.runs VALUES ('a', '2026-01-01 00:00:00+00', ['x','y'], [{'name': 's1', 'ms': 10}]);`,
		`INSERT INTO lake.app.runs VALUES ('b', '2026-01-02 00:00:00+00', [], []);`,
		`COMMIT;`,
	)
	must("delete-insert-txn",
		`BEGIN;`,
		`DELETE FROM lake.app.runs WHERE id = 'a';`,
		`INSERT INTO lake.app.runs VALUES ('a', '2026-02-01 00:00:00+00', ['z'], [{'name': 's9', 'ms': 99}]);`,
		`COMMIT;`,
	)
	if out := must("snapshots", `SELECT count(*) AS n FROM iceberg_snapshots('lake.app.runs');`); !strings.Contains(out, "\n4\n") {
		t.Fatalf("expected 4 snapshots (2 per txn):\n%s", out)
	}

	// Time travel to the first snapshot still sees the pre-delete row.
	if out := must("time-travel",
		`SET VARIABLE first_snap = (SELECT snapshot_id FROM iceberg_snapshots('lake.app.runs') ORDER BY sequence_number LIMIT 1);`,
		`SELECT id, strftime(ts AT TIME ZONE 'UTC', '%Y-%m') AS m FROM lake.app.runs AT (VERSION => getvariable('first_snap')) ORDER BY id;`,
	); !strings.Contains(out, "a,2026-01") {
		t.Fatalf("time travel didn't return the original row:\n%s", out)
	}

	// Retention: a table that only keeps 2 snapshots once they're older
	// than the age window (0ms here) gets its older ones expired on
	// commit, and the default window keeps an older snapshot readable.
	must("retention-setup",
		`CREATE TABLE lake.app.expiring (id INT) WITH ('history.expire.min-snapshots-to-keep' = '2', 'history.expire.max-snapshot-age-ms' = '0');`,
	)
	for i := range 6 {
		must(fmt.Sprintf("retention-insert-%d", i),
			fmt.Sprintf(`INSERT INTO lake.app.expiring VALUES (%d);`, i),
			fmt.Sprintf(`INSERT INTO lake.app.runs VALUES ('r%d', now(), [], []);`, i),
		)
	}
	if out := must("retention-expired",
		`SELECT count(*) AS n, sum(id) AS s FROM lake.app.expiring;`,
		`SELECT count(*) AS snaps FROM iceberg_snapshots('lake.app.expiring');`,
	); !strings.Contains(out, "\n6,15\n") || !strings.Contains(out, "snaps\n2\n") {
		t.Fatalf("expected all rows readable w/ only 2 snapshots left:\n%s", out)
	}
	// app.runs is on the defaults, so its first snapshot (well within the
	// age window) still time travels after more commits.
	if out := must("retention-time-travel",
		`SET VARIABLE first_snap = (SELECT snapshot_id FROM iceberg_snapshots('lake.app.runs') ORDER BY sequence_number LIMIT 1);`,
		`SELECT count(*) AS n FROM lake.app.runs AT (VERSION => getvariable('first_snap'));`,
		`SELECT count(*) AS snaps FROM iceberg_snapshots('lake.app.runs');`,
	); !strings.Contains(out, "n\n1\n") || !strings.Contains(out, "snaps\n10\n") {
		t.Fatalf("expected the first snapshot to still be readable:\n%s", out)
	}

	// A transaction that read a table before some commits expired older
	// snapshots keeps reading its own version (duckdb goes back to that
	// version's metadata.json through the metadata log).
	must("held-setup",
		`CREATE TABLE lake.app.held (id INT) WITH ('history.expire.min-snapshots-to-keep' = '1');`,
		`INSERT INTO lake.app.held VALUES (1);`,
	)
	readerDB, err := openWorker()
	if err != nil {
		t.Fatalf("reader open: %v", err)
	}
	defer readerDB.Close()
	readerConn, err := readerDB.Conn(ctx)
	if err != nil {
		t.Fatalf("reader conn: %v", err)
	}
	defer readerConn.Close()
	if out, err := execCSV(readerConn, []string{`BEGIN;`, `SELECT count(*) AS n1 FROM lake.app.held;`}); err != nil || !strings.Contains(out, "n1\n1\n") {
		t.Fatalf("unexpected first read (err=%v):\n%s", err, out)
	}
	for i := range 4 {
		must(fmt.Sprintf("held-insert-%d", i), `INSERT INTO lake.app.held VALUES (1);`)
	}
	if out, err := execCSV(readerConn, []string{`SELECT count(*) AS n2 FROM lake.app.held;`, `COMMIT;`}); err != nil || !strings.Contains(out, "n2\n1\n") {
		t.Fatalf("expected the held transaction to still read its version (err=%v):\n%s", err, out)
	}
	if out := must("held-after",
		`SELECT count(*) AS n FROM lake.app.held;`,
		`SELECT count(*) AS snaps FROM iceberg_snapshots('lake.app.held');`,
	); !strings.Contains(out, "n\n5\n") {
		t.Fatalf("unexpected rows after the held transaction:\n%s", out)
	}

	// The sandbox (see pkg/duckdb/driver/sandbox.go) blocks the REST calls
	// themselves unless the catalog URL is in allowed_directories.
	sandboxed := func(allowed string) []string {
		return []string{
			allowed,
			`SET enable_external_access=false;`,
			`INSERT INTO lake.app.runs VALUES ('d', now(), [], []);`,
			`SELECT count(*) AS n FROM lake.app.runs;`,
		}
	}
	if out, err := run("sandbox-warehouse-only", sandboxed(fmt.Sprintf("SET allowed_directories=['%s/'];", warehouse))...); err == nil {
		t.Fatalf("expected the sandbox to block the catalog w/o the URL allowlisted:\n%s", out)
	}
	must("sandbox-catalog-allowed", sandboxed(fmt.Sprintf("SET allowed_directories=['%s/', '%s/'];", warehouse, endpoint))...)
}
