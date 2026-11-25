package icebergcatalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/apache/iceberg-go"
	sqlcat "github.com/apache/iceberg-go/catalog/sql"
	"github.com/apache/iceberg-go/table"
	"github.com/stretchr/testify/require"
)

type fakeVendor struct {
	mu    sync.Mutex
	calls []string
}

func (f *fakeVendor) VendCredentials(_ context.Context, ident table.Identifier, location string) (iceberg.Properties, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, strings.Join(ident, ".")+"@"+location)
	return iceberg.Properties{"s3.access-key-id": "AK", "s3.session-token": "tok-" + strings.Join(ident, ".")}, nil
}

func (f *fakeVendor) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func TestVendedCredentials(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "catalog.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	wh := filepath.Join(dir, "warehouse")
	cat, err := sqlcat.NewCatalog("inngest", db, sqlcat.SQLite, iceberg.Properties{"warehouse": wh})
	require.NoError(t, err)
	vendor := &fakeVendor{}
	ts := httptest.NewServer(New(cat, Options{Warehouse: wh, CatalogDB: testCatalogDB(db, wh), Credentials: vendor}))
	t.Cleanup(ts.Close)

	req := func(method, path, body string, vended bool) (int, map[string]any) {
		t.Helper()
		r, _ := http.NewRequest(method, ts.URL+path, strings.NewReader(body))
		if vended {
			r.Header.Set(accessDelegationHeader, "vended-credentials")
		}
		resp, err := http.DefaultClient.Do(r)
		require.NoError(t, err)
		defer resp.Body.Close()
		var out map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out
	}
	loc := filepath.Join(wh, "app.db", "t")
	wantCreds := []any{map[string]any{"prefix": loc, "config": map[string]any{"s3.access-key-id": "AK", "s3.session-token": "tok-app.t"}}}

	status, _ := req("POST", "/v1/namespaces", `{"namespace":["app"]}`, false)
	require.Equal(t, 200, status)
	schema := `{"type":"struct","schema-id":0,"fields":[{"id":1,"name":"id","type":"string","required":false}]}`

	status, body := req("POST", "/v1/namespaces/app/tables", `{"name":"t","schema":`+schema+`}`, true)
	require.Equal(t, 200, status, body)
	require.Equal(t, wantCreds, body["storage-credentials"], "create vends")
	require.Equal(t, "tok-app.t", body["config"].(map[string]any)["s3.session-token"])

	// Twice so the second load comes from the table cache.
	for i := range 2 {
		status, body = req("GET", "/v1/namespaces/app/tables/t", "", true)
		require.Equal(t, 200, status, body)
		require.Equal(t, wantCreds, body["storage-credentials"], "load %d vends", i)
	}

	before := vendor.count()
	status, body = req("GET", "/v1/namespaces/app/tables/t", "", false)
	require.Equal(t, 200, status, body)
	require.Nil(t, body["storage-credentials"], "no creds unless asked for")
	require.Empty(t, body["config"])
	require.Equal(t, before, vendor.count())

	status, body = req("GET", "/v1/namespaces/app/tables/t/credentials", "", true)
	require.Equal(t, 200, status, body)
	require.Equal(t, wantCreds, body["storage-credentials"], "refresh vends")
}

func TestReadOnly(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "catalog.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	wh := filepath.Join(dir, "warehouse")
	cat, err := sqlcat.NewCatalog("inngest", db, sqlcat.SQLite, iceberg.Properties{"warehouse": wh})
	require.NoError(t, err)
	rw := httptest.NewServer(New(cat, Options{Warehouse: wh}))
	t.Cleanup(rw.Close)
	ro := httptest.NewServer(New(cat, Options{Warehouse: wh, ReadOnly: true}))
	t.Cleanup(ro.Close)

	status, _ := do(t, rw, "POST", "/v1/namespaces", `{"namespace":["app"]}`)
	require.Equal(t, 200, status)
	schema := `{"type":"struct","schema-id":0,"fields":[{"id":1,"name":"id","type":"string","required":false}]}`
	status, _ = do(t, rw, "POST", "/v1/namespaces/app/tables", `{"name":"t","schema":`+schema+`}`)
	require.Equal(t, 200, status)
	// A table named like the metrics report's last path segment.
	status, _ = do(t, rw, "POST", "/v1/namespaces/app/tables", `{"name":"metrics","schema":`+schema+`}`)
	require.Equal(t, 200, status)

	for _, c := range []struct{ method, path, body string }{
		{"GET", "/v1/namespaces/app/tables/t", ""},
		{"HEAD", "/v1/namespaces/app/tables/t", ""},
		{"GET", "/v1/namespaces", ""},
		{"POST", "/v1/namespaces/app/tables/t/metrics", `{}`},
		{"POST", "/v1/namespaces/app/tables/metrics/metrics", `{}`},
	} {
		status, _ := do(t, ro, c.method, c.path, c.body)
		require.NotEqual(t, 403, status, "%s %s", c.method, c.path)
	}
	for _, c := range []struct{ method, path, body string }{
		{"POST", "/v1/namespaces", `{"namespace":["x"]}`},
		{"POST", "/v1/namespaces/app/tables", `{"name":"u","schema":` + schema + `}`},
		{"POST", "/v1/namespaces/app/tables/t", `{"requirements":[],"updates":[]}`},
		// A commit, not a metrics report.
		{"POST", "/v1/namespaces/app/tables/metrics", `{"requirements":[],"updates":[{"action":"set-properties","updates":{"k":"v"}}]}`},
		{"POST", "/v1/namespaces/app/tables/metrics/", `{"requirements":[],"updates":[]}`},
		{"POST", "/v1/namespaces/metrics", `{}`},
		{"DELETE", "/v1/namespaces/app/tables/t", ""},
		{"POST", "/v1/tables/rename", `{}`},
		{"POST", "/v1/namespaces/app/properties", `{}`},
	} {
		status, body := do(t, ro, c.method, c.path, c.body)
		require.Equal(t, 403, status, "%s %s: %v", c.method, c.path, body)
	}
	tbl, err := cat.LoadTable(t.Context(), table.Identifier{"app", "metrics"})
	require.NoError(t, err)
	require.NotContains(t, tbl.Properties(), "k", "read-only catalog took a commit")
}

// Creds are scoped to the table's location, also for a load served from
// the table cache of a table whose metadata lives elsewhere.
func TestVendedCredentialsMetadataPath(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "catalog.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	wh := filepath.Join(dir, "warehouse")
	cat, err := sqlcat.NewCatalog("inngest", db, sqlcat.SQLite, iceberg.Properties{"warehouse": wh})
	require.NoError(t, err)
	vendor := &fakeVendor{}
	ts := httptest.NewServer(New(cat, Options{Warehouse: wh, CatalogDB: testCatalogDB(db, wh), Credentials: vendor}))
	t.Cleanup(ts.Close)

	req := func(method, path, body string) (int, map[string]any) {
		t.Helper()
		r, _ := http.NewRequest(method, ts.URL+path, strings.NewReader(body))
		r.Header.Set(accessDelegationHeader, "vended-credentials")
		resp, err := http.DefaultClient.Do(r)
		require.NoError(t, err)
		defer resp.Body.Close()
		var out map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out
	}
	loc := filepath.Join(wh, "app.db", "t")
	metaPath := filepath.Join(wh, "meta", "metadata", "t")

	status, _ := req("POST", "/v1/namespaces", `{"namespace":["app"]}`)
	require.Equal(t, 200, status)
	schema := `{"type":"struct","schema-id":0,"fields":[{"id":1,"name":"id","type":"string","required":false}]}`
	props, _ := json.Marshal(map[string]string{"write.metadata.path": metaPath})
	status, body := req("POST", "/v1/namespaces/app/tables", `{"name":"t","schema":`+schema+`,"properties":`+string(props)+`}`)
	require.Equal(t, 200, status, body)
	// A commit, so the current metadata.json is under write.metadata.path.
	status, body = req("POST", "/v1/namespaces/app/tables/t", `{"requirements":[],"updates":[{"action":"set-properties","updates":{"k":"v"}}]}`)
	require.Equal(t, 200, status, body)
	require.Contains(t, body["metadata-location"], metaPath+"/")

	for i := range 3 {
		status, body = req("GET", "/v1/namespaces/app/tables/t", "")
		require.Equal(t, 200, status, body)
		require.Contains(t, body["metadata-location"], metaPath+"/")
		creds := body["storage-credentials"].([]any)
		require.Equal(t, loc, creds[0].(map[string]any)["prefix"], "load %d", i)
	}
	for _, c := range vendor.calls {
		require.Equal(t, "app.t@"+loc, c)
	}
}
