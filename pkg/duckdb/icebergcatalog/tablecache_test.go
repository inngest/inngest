package icebergcatalog

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/apache/iceberg-go"
	sqlcat "github.com/apache/iceberg-go/catalog/sql"
	"github.com/apache/iceberg-go/table"
)

func snapshotUpdates(id int, manifestList string) string {
	return fmt.Sprintf(`{"action":"add-snapshot","snapshot":{"snapshot-id":%d,"sequence-number":%d,"timestamp-ms":%d,"manifest-list":%q,"summary":{"operation":"append"},"schema-id":0}},`+
		`{"action":"set-snapshot-ref","ref-name":"main","type":"branch","snapshot-id":%d}`,
		id, id, time.Now().UnixMilli()+int64(id), manifestList, id)
}

func assertMain(id int) string {
	return fmt.Sprintf(`{"type":"assert-ref-snapshot-id","ref":"main","snapshot-id":%d}`, id)
}

// TestCacheNeverServesStale commits behind the server's back (as another
// process sharing the catalog db would) and checks the server's cached
// metadata neither fakes a 409 nor hides a real one, and loads stay fresh.
func TestCacheNeverServesStale(t *testing.T) {
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

	if status, body := do(t, ts, "POST", "/v1/namespaces", `{"namespace":["app"]}`); status != 200 {
		t.Fatalf("create ns: %d %v", status, body)
	}
	schema := `{"type":"struct","schema-id":0,"fields":[{"id":1,"name":"id","type":"string","required":false}]}`
	status, body := do(t, ts, "POST", "/v1/namespaces/app/tables", `{"name":"t","schema":`+schema+`}`)
	if status != 200 {
		t.Fatalf("create table: %d %v", status, body)
	}
	loc := body["metadata"].(map[string]any)["location"].(string)
	commit := func(reqs, updates string) (int, map[string]any) {
		return do(t, ts, "POST", "/v1/namespaces/app/tables/t", `{"requirements":[`+reqs+`],"updates":[`+updates+`]}`)
	}

	// v1 through the server, so it's cached.
	if status, body := commit("", snapshotUpdates(1, loc+"/metadata/snap-1.avro")); status != 200 {
		t.Fatalf("commit 1: %d %v", status, body)
	}

	// v2 straight through iceberg-go, so the cache still has v1.
	var reqs table.Requirements
	var ups table.Updates
	if err := json.Unmarshal([]byte(`[`+assertMain(1)+`]`), &reqs); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`[`+snapshotUpdates(2, loc+"/metadata/snap-2.avro")+`]`), &ups); err != nil {
		t.Fatal(err)
	}
	if _, _, err := cat.CommitTable(t.Context(), table.Identifier{"app", "t"}, reqs, ups); err != nil {
		t.Fatal(err)
	}

	// Asserting the real current snapshot (2) has to pass even though the
	// cache says 1.
	if status, body := commit(assertMain(2), snapshotUpdates(3, loc+"/metadata/snap-3.avro")); status != 200 {
		t.Fatalf("commit on the real current snapshot: %d %v", status, body)
	}
	// And asserting a stale one is still a 409.
	if status, body := commit(assertMain(2), snapshotUpdates(4, loc+"/metadata/snap-4.avro")); status != http.StatusConflict {
		t.Fatalf("commit on a stale snapshot: expected 409, got %d %v", status, body)
	}

	// v4 behind the server's back again, then a load has to see it.
	if err := json.Unmarshal([]byte(`[`+assertMain(3)+`]`), &reqs); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`[`+snapshotUpdates(4, loc+"/metadata/snap-4.avro")+`]`), &ups); err != nil {
		t.Fatal(err)
	}
	if _, _, err := cat.CommitTable(t.Context(), table.Identifier{"app", "t"}, reqs, ups); err != nil {
		t.Fatal(err)
	}
	_, body = do(t, ts, "GET", "/v1/namespaces/app/tables/t", "")
	if got := body["metadata"].(map[string]any)["current-snapshot-id"]; got != float64(4) {
		t.Fatalf("load served stale metadata: current-snapshot-id %v", got)
	}
}

// TestCheckMetadataOnlyNewPaths checks a commit only re-validates the paths
// it adds while the table's roots stay put, and everything once they move.
func TestCheckMetadataOnlyNewPaths(t *testing.T) {
	wh := t.TempDir()
	s := New(nil, Options{Warehouse: wh})
	loc := filepath.Join(wh, "app", "t")
	base, err := table.NewMetadata(iceberg.NewSchema(0), nil, table.UnsortedSortOrder, loc, nil)
	if err != nil {
		t.Fatal(err)
	}
	apply := func(md table.Metadata, updates string) table.Metadata {
		t.Helper()
		var ups table.Updates
		if err := json.Unmarshal([]byte(`[`+updates+`]`), &ups); err != nil {
			t.Fatal(err)
		}
		out, err := table.UpdateTableMetadata(md, ups, "")
		if err != nil {
			t.Fatal(err)
		}
		return out
	}

	// prev carries a path outside the table, as if the rules had tightened
	// since it was committed. It's prev's, so it isn't re-checked.
	prev := apply(base, snapshotUpdates(1, "/tmp/evil/snap-1.avro"))
	if err := s.checkMetadata(prev, nil); err == nil {
		t.Fatal("expected the full check to reject prev")
	}
	md := apply(prev, snapshotUpdates(2, loc+"/metadata/snap-2.avro"))
	if err := s.checkMetadata(md, prev); err != nil {
		t.Fatalf("only the new path should be checked: %v", err)
	}

	// A new path outside is still rejected.
	bad := apply(prev, snapshotUpdates(2, "/tmp/evil/snap-2.avro"))
	if err := s.checkMetadata(bad, prev); err == nil || !strings.Contains(err.Error(), "snap-2") {
		t.Fatalf("expected the new path to be rejected, got %v", err)
	}

	// Moving the table re-checks everything, prev's paths included.
	good := apply(base, snapshotUpdates(1, loc+"/metadata/snap-1.avro"))
	moved := apply(good, `{"action":"set-location","location":"`+filepath.Join(wh, "app", "moved")+`"}`)
	if err := s.checkMetadata(moved, good); err == nil {
		t.Fatal("expected snap-1 to be rejected once it's outside the new location")
	}
}
