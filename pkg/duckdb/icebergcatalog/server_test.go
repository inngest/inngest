package icebergcatalog

import (
	"context"
	"database/sql"
	"encoding/json"
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
	ts := httptest.NewServer(New(cat, Options{Warehouse: wh}))
	t.Cleanup(ts.Close)
	return ts
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
