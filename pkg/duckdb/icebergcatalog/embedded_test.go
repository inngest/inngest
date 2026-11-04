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
	"testing"
	"time"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	"github.com/apache/iceberg-go/catalog/catalogtest"
	_ "github.com/apache/iceberg-go/catalog/rest"
)

func TestOpenSQLite(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	cfg := Config{
		SQLitePath: filepath.Join(dir, "catalog.db"),
		Warehouse:  filepath.Join(dir, "warehouse"),
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

// defaultPostgresTestURI matches the postgres the repo's e2e CI and
// pkg/duckdb/driver's postgres tests use. ICEBERG_POSTGRES_TEST_URI
// overrides it.
const defaultPostgresTestURI = "postgres://postgres:postgres@localhost:5432/inngest_test?sslmode=disable"

// requirePostgres returns a URI scoped (via search_path) to a fresh schema
// on a reachable postgres, or skips.
func requirePostgres(t *testing.T) string {
	t.Helper()
	uri := defaultPostgresTestURI
	if env, ok := os.LookupEnv("ICEBERG_POSTGRES_TEST_URI"); ok {
		uri = env
	}
	db, err := sql.Open("pgx", uri)
	if err != nil {
		t.Skipf("skipping: opening postgres %q: %v", uri, err)
	}
	defer func() { _ = db.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Skipf("skipping: postgres not reachable at %q (set ICEBERG_POSTGRES_TEST_URI): %v", uri, err)
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
				PostgresURI: requirePostgres(t),
				Warehouse:   filepath.Join(t.TempDir(), "warehouse"),
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
