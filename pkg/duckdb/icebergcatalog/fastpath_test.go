package icebergcatalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/apache/iceberg-go"
	sqlcat "github.com/apache/iceberg-go/catalog/sql"
	"github.com/apache/iceberg-go/table"
)

// countingCatalog counts the sql catalog loads/commits the server falls
// back to.
type countingCatalog struct {
	*sqlcat.Catalog
	loads, commits atomic.Int64
}

func (c *countingCatalog) LoadTable(ctx context.Context, ident table.Identifier) (*table.Table, error) {
	c.loads.Add(1)
	return c.Catalog.LoadTable(ctx, ident)
}

func (c *countingCatalog) CommitTable(ctx context.Context, ident table.Identifier, reqs []table.Requirement, updates []table.Update) (table.Metadata, string, error) {
	c.commits.Add(1)
	return c.Catalog.CommitTable(ctx, ident, reqs, updates)
}

type fastTable struct {
	t   *testing.T
	ts  *httptest.Server
	srv *Server
	cat *countingCatalog
	ns  []string
	// path is the table's REST path.
	path string
	loc  string
}

// newFastTable is a fast path server w/ one table, `<ns>.t`, created w/
// props.
func newFastTable(t *testing.T, ns []string, props string) *fastTable {
	t.Helper()
	return newFastTableDB(t, ns, props, false)
}

// newFastTableDB is newFastTable, w/ the catalog db on the legacy V0 schema
// (no iceberg_type column) if v0.
func newFastTableDB(t *testing.T, ns []string, props string, v0 bool) *fastTable {
	t.Helper()
	dir := t.TempDir()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if v0 {
		// The JDBC catalog's original DDL, which iceberg-go leaves alone.
		if _, err := db.Exec(`CREATE TABLE iceberg_tables (
			catalog_name VARCHAR(255) NOT NULL,
			table_namespace VARCHAR(255) NOT NULL,
			table_name VARCHAR(255) NOT NULL,
			metadata_location VARCHAR(1000),
			previous_metadata_location VARCHAR(1000),
			PRIMARY KEY (catalog_name, table_namespace, table_name))`); err != nil {
			t.Fatal(err)
		}
	}
	wh := filepath.Join(dir, "warehouse")
	inner, err := sqlcat.NewCatalog("inngest", db, sqlcat.SQLite, iceberg.Properties{"warehouse": wh})
	if err != nil {
		t.Fatal(err)
	}
	ft := &fastTable{t: t, cat: &countingCatalog{Catalog: inner}, ns: ns}
	ft.srv = New(ft.cat, Options{Warehouse: wh, CatalogDB: testCatalogDB(db, wh)})
	ft.ts = httptest.NewServer(ft.srv)
	t.Cleanup(ft.ts.Close)

	nsJSON, _ := json.Marshal(ns)
	if status, body := do(t, ft.ts, "POST", "/v1/namespaces", `{"namespace":`+string(nsJSON)+`}`); status != 200 {
		t.Fatalf("create ns: %d %v", status, body)
	}
	nsPath := url.PathEscape(strings.Join(ns, namespaceSeparator))
	if props == "" {
		props = "{}"
	}
	schema := `{"type":"struct","schema-id":0,"fields":[{"id":1,"name":"id","type":"string","required":false}]}`
	status, body := do(t, ft.ts, "POST", "/v1/namespaces/"+nsPath+"/tables", `{"name":"t","schema":`+schema+`,"properties":`+props+`}`)
	if status != 200 {
		t.Fatalf("create table: %d %v", status, body)
	}
	ft.path = "/v1/namespaces/" + nsPath + "/tables/t"
	ft.loc = body["metadata"].(map[string]any)["location"].(string)
	return ft
}

func (ft *fastTable) ident() table.Identifier {
	return append(table.Identifier{}, append(ft.ns, "t")...)
}

func (ft *fastTable) commit(reqs, updates string) (int, map[string]any) {
	ft.t.Helper()
	return do(ft.t, ft.ts, "POST", ft.path, `{"requirements":[`+reqs+`],"updates":[`+updates+`]}`)
}

func (ft *fastTable) mustCommit(reqs, updates string) map[string]any {
	ft.t.Helper()
	status, body := ft.commit(reqs, updates)
	if status != 200 {
		ft.t.Fatalf("commit: %d %v", status, body)
	}
	return body
}

// counts is (sql catalog loads, sql catalog commits) since the last call.
func (ft *fastTable) counts() (int64, int64) {
	return ft.cat.loads.Swap(0), ft.cat.commits.Swap(0)
}

// stored is what the sql catalog has for the table, bypassing the server.
func (ft *fastTable) stored() *table.Table {
	ft.t.Helper()
	tbl, err := ft.cat.Catalog.LoadTable(ft.t.Context(), ft.ident())
	if err != nil {
		ft.t.Fatal(err)
	}
	return tbl
}

// commitBehindBack commits straight through iceberg-go, as another server
// sharing the catalog db would.
func (ft *fastTable) commitBehindBack(reqs, updates string) {
	ft.t.Helper()
	var rs table.Requirements
	var us table.Updates
	if err := json.Unmarshal([]byte(`[`+reqs+`]`), &rs); err != nil {
		ft.t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`[`+updates+`]`), &us); err != nil {
		ft.t.Fatal(err)
	}
	if _, _, err := ft.cat.Catalog.CommitTable(context.Background(), ft.ident(), rs, us); err != nil {
		ft.t.Fatal(err)
	}
}

func snapLoc(ft *fastTable, id int) string {
	return ft.loc + fmt.Sprintf("/metadata/snap-%d.avro", id)
}

func TestFastPathLoadAndCommit(t *testing.T) {
	for name, tc := range map[string]struct {
		ns []string
		v0 bool
	}{
		"single level": {ns: []string{"app"}},
		"multi level":  {ns: []string{"app", "sub"}},
		// Stored under the sql catalog's encoded key.
		"dotted":    {ns: []string{"app", "s.ub"}},
		"v0 schema": {ns: []string{"app"}, v0: true},
	} {
		t.Run(name, func(t *testing.T) {
			ft := newFastTableDB(t, tc.ns, "", tc.v0)
			if v0, err := ft.srv.fast.isV0(t.Context()); err != nil || v0 != tc.v0 {
				t.Fatalf("isV0: %v %v", v0, err)
			}
			// Nothing cached yet, so the first commit goes through the sql
			// catalog.
			ft.mustCommit("", snapshotUpdates(1, snapLoc(ft, 1)))
			if loads, commits := ft.counts(); commits != 1 || loads == 0 {
				t.Fatalf("expected the first commit to fall back, got %d loads %d commits", loads, commits)
			}

			// After that commits and loads skip it.
			body := ft.mustCommit(assertMain(1), snapshotUpdates(2, snapLoc(ft, 2)))
			status, loaded := do(t, ft.ts, "GET", ft.path, "")
			if status != 200 {
				t.Fatalf("load: %d %v", status, loaded)
			}
			if loads, commits := ft.counts(); loads != 0 || commits != 0 {
				t.Fatalf("expected the fast path, got %d loads %d commits", loads, commits)
			}

			// And what they did is what's stored.
			stored := ft.stored()
			if stored.MetadataLocation() != body["metadata-location"] || stored.MetadataLocation() != loaded["metadata-location"] {
				t.Fatalf("stored %s, committed %v, loaded %v", stored.MetadataLocation(), body["metadata-location"], loaded["metadata-location"])
			}
			if got := stored.Metadata().CurrentSnapshot().SnapshotID; got != 2 {
				t.Fatalf("stored current snapshot %d", got)
			}
			if got := loaded["metadata"].(map[string]any)["current-snapshot-id"]; got != float64(2) {
				t.Fatalf("loaded current snapshot %v", got)
			}
		})
	}
}

func TestFastPathStaleCacheFallsBack(t *testing.T) {
	ft := newFastTable(t, []string{"app"}, "")
	ft.mustCommit("", snapshotUpdates(1, snapLoc(ft, 1)))
	ft.commitBehindBack(assertMain(1), snapshotUpdates(2, snapLoc(ft, 2)))
	ft.counts()

	// The cached version (1) isn't current, so the load goes through the
	// sql catalog and sees 2.
	_, body := do(t, ft.ts, "GET", ft.path, "")
	if got := body["metadata"].(map[string]any)["current-snapshot-id"]; got != float64(2) {
		t.Fatalf("load served stale metadata: current-snapshot-id %v", got)
	}
	if loads, _ := ft.counts(); loads != 1 {
		t.Fatalf("expected the load to fall back, got %d loads", loads)
	}

	// Same for a commit.
	ft.commitBehindBack(assertMain(2), snapshotUpdates(3, snapLoc(ft, 3)))
	ft.mustCommit(assertMain(3), snapshotUpdates(4, snapLoc(ft, 4)))
	if loads, commits := ft.counts(); loads == 0 || commits != 1 {
		t.Fatalf("expected the commit to fall back, got %d loads %d commits", loads, commits)
	}
	if got := ft.stored().Metadata().CurrentSnapshot().SnapshotID; got != 4 {
		t.Fatalf("stored current snapshot %d", got)
	}
}

func TestFastPathCASConflict(t *testing.T) {
	ft := newFastTable(t, []string{"app"}, "")
	ft.mustCommit("", snapshotUpdates(1, snapLoc(ft, 1)))

	// Another writer lands between our check and the swap.
	ft.srv.fast.beforeSwap = func() {
		ft.srv.fast.beforeSwap = nil
		ft.commitBehindBack(assertMain(1), `{"action":"set-properties","updates":{"other":"x"}}`)
	}
	before := metadataJSONs(t, ft.loc)
	status, body := ft.commit(assertMain(1), snapshotUpdates(2, snapLoc(ft, 2)))
	if status != http.StatusConflict || body["error"].(map[string]any)["type"] != "CommitFailedException" {
		t.Fatalf("expected a 409, got %d %v", status, body)
	}
	// Our metadata.json is cleaned up, only the other writer's is new.
	if got := metadataJSONs(t, ft.loc); len(got) != len(before)+1 {
		t.Fatalf("expected only the other writer's metadata.json to be new, got %v (was %v)", got, before)
	}

	// The client reloads and retries, which keeps both.
	ft.mustCommit(assertMain(1), snapshotUpdates(2, snapLoc(ft, 2)))
	md := ft.stored().Metadata()
	if md.CurrentSnapshot().SnapshotID != 2 || md.Properties()["other"] != "x" {
		t.Fatalf("lost a commit: snapshot %d props %v", md.CurrentSnapshot().SnapshotID, md.Properties())
	}
}

func TestFastPathMetadataCompression(t *testing.T) {
	ft := newFastTable(t, []string{"app"}, `{"write.metadata.compression-codec":"gzip"}`)
	ft.mustCommit("", snapshotUpdates(1, snapLoc(ft, 1)))
	body := ft.mustCommit(assertMain(1), snapshotUpdates(2, snapLoc(ft, 2)))
	if loads, commits := ft.counts(); loads != 1 || commits != 1 {
		t.Fatalf("expected only the first commit to fall back, got %d loads %d commits", loads, commits)
	}
	loc := body["metadata-location"].(string)
	if !strings.HasSuffix(loc, ".gz.metadata.json") {
		t.Fatalf("expected a gzipped metadata.json, got %s", loc)
	}
	// The sql catalog can read it back.
	if got := ft.stored().Metadata().CurrentSnapshot().SnapshotID; got != 2 {
		t.Fatalf("stored current snapshot %d", got)
	}
}

// TestFastPathConcurrentServers has two servers on one catalog db commit
// concurrently, every commit retried on a 409 like a client would, and
// checks none of them are lost.
func TestFastPathConcurrentServers(t *testing.T) {
	t.Run("sqlite", func(t *testing.T) {
		dir := t.TempDir()
		testConcurrentServers(t, Config{SQLitePath: filepath.Join(dir, "catalog.db"), Warehouse: filepath.Join(dir, "warehouse")})
	})
	t.Run("postgres", func(t *testing.T) {
		testConcurrentServers(t, Config{PostgresURI: requirePostgres(t), Warehouse: filepath.Join(t.TempDir(), "warehouse")})
	})
}

func testConcurrentServers(t *testing.T, cfg Config) {
	ctx := context.Background()
	var servers []*Embedded
	for range 2 {
		e, err := Open(ctx, cfg)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = e.Close() })
		servers = append(servers, e)
	}
	req := func(e *Embedded, method, path, body string) (int, map[string]any) {
		r, _ := http.NewRequestWithContext(ctx, method, e.URL+path, strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+e.Token)
		resp, err := http.DefaultClient.Do(r)
		if err != nil {
			t.Error(err)
			return 0, nil
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		var out map[string]any
		_ = json.Unmarshal(b, &out)
		return resp.StatusCode, out
	}
	if status, body := req(servers[0], "POST", "/v1/namespaces", `{"namespace":["app"]}`); status != 200 {
		t.Fatalf("create ns: %d %v", status, body)
	}
	schema := `{"type":"struct","schema-id":0,"fields":[{"id":1,"name":"id","type":"string","required":false}]}`
	if status, body := req(servers[0], "POST", "/v1/namespaces/app/tables", `{"name":"t","schema":`+schema+`}`); status != 200 {
		t.Fatalf("create table: %d %v", status, body)
	}

	const writers, perWriter = 4, 15
	var wg sync.WaitGroup
	var conflicts atomic.Int64
	for w := range writers {
		e := servers[w%len(servers)]
		wg.Go(func() {
			for i := range perWriter {
				// Every commit applies to whatever base the server has, so
				// a lost update would drop another writer's property.
				body := fmt.Sprintf(`{"requirements":[],"updates":[{"action":"set-properties","updates":{"w%d-%d":"x"}}]}`, w, i)
				for attempt := 0; ; attempt++ {
					status, out := req(e, "POST", "/v1/namespaces/app/tables/t", body)
					if status == 200 {
						break
					}
					if status != http.StatusConflict || attempt > 200 {
						t.Errorf("commit w%d-%d: %d %v", w, i, status, out)
						return
					}
					conflicts.Add(1)
				}
			}
		})
	}
	wg.Wait()
	if t.Failed() {
		return
	}

	for _, e := range servers {
		status, body := req(e, "GET", "/v1/namespaces/app/tables/t", "")
		if status != 200 {
			t.Fatalf("load: %d %v", status, body)
		}
		props := body["metadata"].(map[string]any)["properties"].(map[string]any)
		for w := range writers {
			for i := range perWriter {
				if props[fmt.Sprintf("w%d-%d", w, i)] != "x" {
					t.Fatalf("lost commit w%d-%d (%d conflicts retried)", w, i, conflicts.Load())
				}
			}
		}
	}
	t.Logf("%d conflicts retried", conflicts.Load())
}

func TestParseMetadataVersion(t *testing.T) {
	for loc, want := range map[string]int{
		"file:///wh/app.db/t/metadata/00003-6c0b6e4a-0f3c-4c7e-9a3b-3f1d8a9a0b1c.metadata.json":   3,
		"s3://b/wh/app.db/t/metadata/00012-6c0b6e4a-0f3c-4c7e-9a3b-3f1d8a9a0b1c.gz.metadata.json": 12,
		"/wh/t/metadata/v7.metadata.json":   7,
		"/wh/t/metadata/nope.metadata.json": -1,
		"/wh/t/metadata/":                   -1,
	} {
		if got := parseMetadataVersion(loc); got != want {
			t.Errorf("%s: got %d want %d", loc, got, want)
		}
	}
}
