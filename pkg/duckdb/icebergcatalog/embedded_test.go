package icebergcatalog

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	"github.com/apache/iceberg-go/catalog/catalogtest"
	_ "github.com/apache/iceberg-go/catalog/rest"
	"github.com/inngest/inngest/tests/testutil"
)

func TestOpenSQLite(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	cfg := Config{
		DB:        openSQLite(t, filepath.Join(dir, "catalog.db")),
		Dialect:   SQLite,
		Warehouse: filepath.Join(dir, "warehouse"),
	}
	e, err := Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(e.URL, "http://127.0.0.1:") || e.Token == "" {
		t.Fatalf("unexpected url/token: %q %q", e.URL, e.Token)
	}

	// No token -> 401.
	resp, err := http.Get(e.URL + "/v1/config")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 w/o token, got %d", resp.StatusCode)
	}

	load := func(e *Embedded) catalog.Catalog {
		t.Helper()
		cat, err := catalog.Load(ctx, "test", iceberg.Properties{"type": "rest", "uri": e.URL, "token": e.Token})
		if err != nil {
			t.Fatal(err)
		}
		return cat
	}
	if err := load(e).CreateNamespace(ctx, []string{"app"}, nil); err != nil {
		t.Fatal(err)
	}
	if err := e.Close(); err != nil {
		t.Fatal(err)
	}

	// Reopening the same sqlite file keeps the namespace.
	e, err = Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = e.Close() })
	ok, err := load(e).CheckNamespaceExists(ctx, []string{"app"})
	if err != nil || !ok {
		t.Fatalf("namespace didn't survive a restart: %v %v", ok, err)
	}

	if _, err := Open(ctx, Config{Warehouse: dir}); err == nil {
		t.Fatal("expected an error w/o a backing db")
	}
}

// TestOpenRejectsURLCharsInLocalWarehouse checks a local warehouse w/ a
// character that means something in a URL fails Open (and creates
// nothing), rather than every table create failing later.
func TestOpenRejectsURLCharsInLocalWarehouse(t *testing.T) {
	for _, name := range []string{"my#lake", "q?lake", "pct%41lake"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			wh := filepath.Join(dir, name, "warehouse")
			_, err := Open(context.Background(), Config{
				DB:        openSQLite(t, filepath.Join(dir, "catalog.db")),
				Dialect:   SQLite,
				Warehouse: wh,
			})
			if err == nil || !strings.Contains(err.Error(), "can't contain") {
				t.Fatalf("expected a can't contain error, got %v", err)
			}
			if _, err := os.Stat(wh); !os.IsNotExist(err) {
				t.Fatalf("Open created %s: %v", wh, err)
			}
		})
	}
}

// defaultPostgresTestURI matches the postgres the repo's e2e CI and
// pkg/duckdb/driver's postgres tests use. ICEBERG_POSTGRES_TEST_URI
// overrides it.
const defaultPostgresTestURI = "postgres://postgres:postgres@localhost:5432/inngest_test?sslmode=disable"

// sharedPostgres is a postgres container started once per test binary
// under TEST_DATABASE=postgres (the CI postgres job), since each catalog
// conformance case wants its own catalog. testcontainers' reaper removes it
// when the binary exits.
var sharedPostgres struct {
	once sync.Once
	uri  string
	err  error
}

func startSharedPostgres(t *testing.T) (string, error) {
	sharedPostgres.once.Do(func() {
		// t only bounds the start, the container outlives it.
		pc, err := testutil.StartPostgres(t)
		if err != nil {
			sharedPostgres.err = err
			return
		}
		sharedPostgres.uri = pc.URI
	})
	return sharedPostgres.uri, sharedPostgres.err
}

// requirePostgres returns a URI scoped (via search_path) to a fresh schema
// on a reachable postgres, or skips. Under TEST_DATABASE=postgres it starts
// one instead, and then (or w/ ICEBERG_POSTGRES_TEST_URI set) it fails
// rather than skips if postgres isn't usable.
func requirePostgres(t *testing.T) string {
	t.Helper()
	uri := defaultPostgresTestURI
	// Asked for postgres (an explicit URI or the CI postgres job), so it
	// failing is a failure, not a skip.
	required := false
	if env, ok := os.LookupEnv("ICEBERG_POSTGRES_TEST_URI"); ok {
		uri, required = env, true
	} else if os.Getenv("TEST_DATABASE") == "postgres" {
		var err error
		if uri, err = startSharedPostgres(t); err != nil {
			t.Fatalf("starting postgres: %v", err)
		}
		required = true
	}
	skipOrFail := t.Skipf
	if required {
		skipOrFail = t.Fatalf
	}
	db, err := sql.Open("pgx", uri)
	if err != nil {
		skipOrFail("opening postgres %q: %v", uri, err)
	}
	defer func() { _ = db.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		skipOrFail("postgres not reachable at %q (set ICEBERG_POSTGRES_TEST_URI): %v", uri, err)
	}
	schema := fmt.Sprintf("iceberg_test_%d", time.Now().UnixNano())
	if _, err := db.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db, err := sql.Open("pgx", uri)
		if err != nil {
			return
		}
		defer func() { _ = db.Close() }()
		_, _ = db.Exec("DROP SCHEMA " + schema + " CASCADE")
	})
	u, err := url.Parse(uri)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	return u.String()
}

// TestRESTConformancePostgres is TestRESTConformance w/ the catalog backed
// by postgres (IE the devserver's `--postgres-uri` mode).
func TestRESTConformancePostgres(t *testing.T) {
	requirePostgres(t)
	catalogtest.RunCatalogTests(t, catalogtest.Config{
		SupportsNamespaceProperties: true,
		NewCatalog: func(t *testing.T) catalog.Catalog {
			e, err := Open(context.Background(), Config{
				DB:        openPostgres(t, requirePostgres(t)),
				Dialect:   Postgres,
				Warehouse: filepath.Join(t.TempDir(), "warehouse"),
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = e.Close() })
			cat, err := catalog.Load(context.Background(), "test", iceberg.Properties{"type": "rest", "uri": e.URL, "token": e.Token})
			if err != nil {
				t.Fatal(err)
			}
			return cat
		},
	})
}
