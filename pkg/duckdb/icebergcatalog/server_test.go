package icebergcatalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	"github.com/apache/iceberg-go/catalog/catalogtest"
	_ "github.com/apache/iceberg-go/catalog/rest"
	sqlcat "github.com/apache/iceberg-go/catalog/sql"
	_ "modernc.org/sqlite"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	wh := filepath.Join(dir, "warehouse")
	cat, err := sqlcat.NewCatalog("inngest", db, sqlcat.SQLite, iceberg.Properties{"warehouse": wh})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New(cat, Options{Warehouse: wh, CatalogDB: testCatalogDB(db, wh)}))
	t.Cleanup(ts.Close)
	return ts
}

// testCatalogDB is the CatalogDB for a sqlite catalog the tests create.
func testCatalogDB(db *sql.DB, wh string) *CatalogDB {
	return &CatalogDB{DB: db, Dialect: sqlcat.SQLite, Name: "inngest", Properties: iceberg.Properties{"warehouse": wh}}
}

func do(t *testing.T, ts *httptest.Server, method, path, body string) (int, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest(method, ts.URL+path, strings.NewReader(body))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func TestErrorMapping(t *testing.T) {
	ts := newTestServer(t)

	status, body := do(t, ts, "GET", "/v1/namespaces/nope/tables/x", "")
	if status != 404 || body["error"].(map[string]any)["type"] != "NoSuchTableException" {
		t.Fatalf("missing table: %d %v", status, body)
	}

	if status, body := do(t, ts, "POST", "/v1/namespaces", `{"namespace":["app"]}`); status != 200 {
		t.Fatalf("create ns: %d %v", status, body)
	}
	if status, _ := do(t, ts, "POST", "/v1/namespaces", `{"namespace":["app"]}`); status != 409 {
		t.Fatalf("dup ns: %d", status)
	}

	schema := `{"type":"struct","schema-id":0,"fields":[{"id":1,"name":"id","type":"string","required":false}]}`
	if status, body := do(t, ts, "POST", "/v1/namespaces/app/tables", `{"name":"t","schema":`+schema+`}`); status != 200 {
		t.Fatalf("create table: %d %v", status, body)
	}

	// Wrong UUID -> CommitFailedException / 409.
	status, body = do(t, ts, "POST", "/v1/namespaces/app/tables/t",
		`{"requirements":[{"type":"assert-table-uuid","uuid":"00000000-0000-0000-0000-000000000000"}],"updates":[{"action":"set-properties","updates":{"a":"b"}}]}`)
	if status != 409 || body["error"].(map[string]any)["type"] != "CommitFailedException" {
		t.Fatalf("bad uuid commit: %d %v", status, body)
	}

	if status, _ := do(t, ts, "DELETE", "/v1/namespaces/app", ""); status != 409 {
		t.Fatalf("drop non-empty ns: %d", status)
	}
}

// TestRESTConformance runs iceberg-go's catalog conformance suite through
// its own REST client against our server.
func TestRESTConformance(t *testing.T) {
	catalogtest.RunCatalogTests(t, catalogtest.Config{
		SupportsNamespaceProperties: true,
		NewCatalog: func(t *testing.T) catalog.Catalog {
			ts := newTestServer(t)
			cat, err := catalog.Load(context.Background(), "test", iceberg.Properties{"type": "rest", "uri": ts.URL})
			if err != nil {
				t.Fatal(err)
			}
			return cat
		},
	})
}

// TestRequestLogUsesRoute: the request log names the matched route, never
// the raw URI the client sent.
func TestRequestLogUsesRoute(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	wh := filepath.Join(dir, "warehouse")
	cat, err := sqlcat.NewCatalog("inngest", db, sqlcat.SQLite, iceberg.Properties{"warehouse": wh})
	if err != nil {
		t.Fatal(err)
	}
	var logs strings.Builder
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ts := httptest.NewServer(New(cat, Options{Warehouse: wh, Logger: logger}))
	t.Cleanup(ts.Close)

	do(t, ts, "GET", "/v1/namespaces/evil%0Ainjected/tables/x?q=secret", "")
	do(t, ts, "GET", "/nope/secret", "")
	out := logs.String()
	if !strings.Contains(out, `route="GET /v1/namespaces/{ns}/tables/{table}"`) || !strings.Contains(out, "route=unmatched") {
		t.Fatalf("want routes in the log, got %q", out)
	}
	for _, leak := range []string{"evil", "injected", "secret"} {
		if strings.Contains(out, leak) {
			t.Fatalf("log has %q from the request: %q", leak, out)
		}
	}
}

func TestReadJSONMaxBytes(t *testing.T) {
	big := strings.NewReader(`{"namespace":["` + strings.Repeat("a", maxRequestBodyBytes+1) + `"]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/namespaces", big)
	rec := httptest.NewRecorder()
	var dst createNamespaceRequest
	if readJSON(rec, req, &dst) {
		t.Fatal("expected readJSON to reject oversized body")
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d: %s", rec.Code, rec.Body.String())
	}
}
