package icebergcatalog

import (
	"database/sql"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/apache/iceberg-go"
	sqlcat "github.com/apache/iceberg-go/catalog/sql"
)

// retentionTable is a server w/ retention configured and one table app.t
// created w/ props.
type retentionTable struct {
	t   *testing.T
	ts  *httptest.Server
	loc string
	// clock is both the server's now and what commit stamps snapshots w/.
	// iceberg-go rejects snapshots older than the table's last update, so
	// it starts at the real time and only moves forward.
	clock time.Time
}

func (rt *retentionTable) advance(d time.Duration) { rt.clock = rt.clock.Add(d) }

func newRetentionTable(t *testing.T, r Retention, props string) *retentionTable {
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
	rt := &retentionTable{t: t, clock: time.Now().Add(time.Second)}
	srv := New(cat, Options{Warehouse: wh, Retention: r})
	srv.now = func() time.Time { return rt.clock }
	rt.ts = httptest.NewServer(srv)
	t.Cleanup(rt.ts.Close)
	ts := rt.ts

	if status, body := do(t, ts, "POST", "/v1/namespaces", `{"namespace":["app"]}`); status != 200 {
		t.Fatalf("create ns: %d %v", status, body)
	}
	schema := `{"type":"struct","schema-id":0,"fields":[{"id":1,"name":"id","type":"string","required":false}]}`
	if props == "" {
		props = "{}"
	}
	status, body := do(t, ts, "POST", "/v1/namespaces/app/tables", `{"name":"t","schema":`+schema+`,"properties":`+props+`}`)
	if status != 200 {
		t.Fatalf("create table: %d %v", status, body)
	}
	rt.loc = body["metadata"].(map[string]any)["location"].(string)
	return rt
}

// commit adds snapshot id (w/ parent id-1, written 1s after the last one)
// on main, plus any extra updates, and returns the resulting metadata.
func (rt *retentionTable) commit(id int, extra ...string) map[string]any {
	rt.t.Helper()
	rt.advance(time.Second)
	ts := rt.clock
	reqs := ""
	parent := ""
	if id > 1 {
		reqs = assertMain(id - 1)
		parent = fmt.Sprintf(`"parent-snapshot-id":%d,`, id-1)
	}
	// A real manifest list on disk, to check expiry never deletes files.
	ml := filepath.Join(rt.loc, "metadata", fmt.Sprintf("snap-%d.avro", id))
	if err := os.MkdirAll(filepath.Dir(ml), 0o755); err != nil {
		rt.t.Fatal(err)
	}
	if err := os.WriteFile(ml, []byte("x"), 0o644); err != nil {
		rt.t.Fatal(err)
	}
	updates := fmt.Sprintf(`{"action":"add-snapshot","snapshot":{"snapshot-id":%d,%s"sequence-number":%d,"timestamp-ms":%d,"manifest-list":%q,"summary":{"operation":"append"},"schema-id":0}},`+
		`{"action":"set-snapshot-ref","ref-name":"main","type":"branch","snapshot-id":%d}`,
		id, parent, id, ts.UnixMilli(), ml, id)
	for _, u := range extra {
		updates += "," + u
	}
	status, body := do(rt.t, rt.ts, "POST", "/v1/namespaces/app/tables/t", `{"requirements":[`+reqs+`],"updates":[`+updates+`]}`)
	if status != 200 {
		rt.t.Fatalf("commit %d: %d %v", id, status, body)
	}
	return body["metadata"].(map[string]any)
}

func snapshotIDs(md map[string]any) []int {
	var ids []int
	for _, s := range md["snapshots"].([]any) {
		ids = append(ids, int(s.(map[string]any)["snapshot-id"].(float64)))
	}
	slices.Sort(ids)
	return ids
}

func seq(from, to int) []int {
	var out []int
	for i := from; i <= to; i++ {
		out = append(out, i)
	}
	return out
}

func TestRetentionByCountAndAge(t *testing.T) {
	rt := newRetentionTable(t, Retention{MinSnapshotsToKeep: 3, MaxSnapshotAge: time.Hour}, "")

	var md map[string]any
	for i := 1; i <= 10; i++ {
		md = rt.commit(i)
	}
	// All in the age window.
	if got := snapshotIDs(md); !slices.Equal(got, seq(1, 10)) {
		t.Fatalf("expected every snapshot in the age window, got %v", got)
	}

	// Once they age out only the newest 3 are left.
	rt.advance(2 * time.Hour)
	md = rt.commit(11)
	if got := snapshotIDs(md); !slices.Equal(got, seq(9, 11)) {
		t.Fatalf("expected the newest 3 snapshots, got %v", got)
	}

	// Recent ones are kept past the count, the old ones keep going.
	for i := 12; i <= 16; i++ {
		md = rt.commit(i)
	}
	if got := snapshotIDs(md); !slices.Equal(got, seq(11, 16)) {
		t.Fatalf("expected only the age window, got %v", got)
	}

	// Expiry only touches metadata, never files.
	for i := 1; i <= 16; i++ {
		if _, err := os.Stat(filepath.Join(rt.loc, "metadata", fmt.Sprintf("snap-%d.avro", i))); err != nil {
			t.Fatalf("snap-%d.avro: %v", i, err)
		}
	}
}

func TestRetentionKeepsRefTargets(t *testing.T) {
	rt := newRetentionTable(t, Retention{MinSnapshotsToKeep: 1, MaxSnapshotAge: time.Minute}, "")
	rt.commit(1)
	rt.commit(2, `{"action":"set-snapshot-ref","ref-name":"pinned","type":"tag","snapshot-id":1}`)
	for i := 3; i <= 6; i++ {
		rt.commit(i)
	}
	rt.advance(time.Hour)
	md := rt.commit(7)
	if got := snapshotIDs(md); !slices.Equal(got, []int{1, 7}) {
		t.Fatalf("expected the tagged + current snapshot, got %v", got)
	}
	if refs := md["refs"].(map[string]any); refs["pinned"] == nil {
		t.Fatalf("tag was dropped: %v", refs)
	}

	// A commit that moves a ref onto an old snapshot keeps that one too.
	rt = newRetentionTable(t, Retention{MinSnapshotsToKeep: 1, MaxSnapshotAge: time.Minute}, "")
	for i := 1; i <= 4; i++ {
		rt.commit(i)
	}
	rt.advance(time.Hour)
	md = rt.commit(5, `{"action":"set-snapshot-ref","ref-name":"pinned","type":"tag","snapshot-id":2}`)
	if got := snapshotIDs(md); !slices.Equal(got, []int{2, 5}) {
		t.Fatalf("expected the newly tagged + current snapshot, got %v", got)
	}
}

func TestRetentionTableProperties(t *testing.T) {
	// The table keeps 5 and expires anything older than 0ms, over a server
	// default that would keep everything.
	rt := newRetentionTable(t, Retention{MinSnapshotsToKeep: 100, MaxSnapshotAge: 24 * time.Hour},
		`{"history.expire.min-snapshots-to-keep":"5","history.expire.max-snapshot-age-ms":"0"}`)
	var md map[string]any
	for i := 1; i <= 9; i++ {
		md = rt.commit(i)
	}
	if got := snapshotIDs(md); !slices.Equal(got, seq(5, 9)) {
		t.Fatalf("expected the table's own retention, got %v", got)
	}
}

func TestRetentionDisabled(t *testing.T) {
	rt := newRetentionTable(t, Retention{Disabled: true, MinSnapshotsToKeep: 1, MaxSnapshotAge: time.Millisecond}, "")
	var md map[string]any
	for i := 1; i <= 5; i++ {
		md = rt.commit(i)
	}
	if got := snapshotIDs(md); !slices.Equal(got, seq(1, 5)) {
		t.Fatalf("expected no expiry, got %v", got)
	}
}

func metadataJSONs(t *testing.T, loc string) []string {
	t.Helper()
	ents, err := os.ReadDir(filepath.Join(loc, "metadata"))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range ents {
		if strings.HasSuffix(e.Name(), ".metadata.json") {
			out = append(out, filepath.Join(loc, "metadata", e.Name()))
		}
	}
	return out
}

func TestDeleteOldMetadata(t *testing.T) {
	rt := newRetentionTable(t, Retention{}, `{"write.metadata.previous-versions-max":"3"}`)
	// Something else's file that happens to look like one of ours, which
	// was never on the metadata log.
	stray := filepath.Join(rt.loc, "metadata", "00000-stray.metadata.json")
	if err := os.WriteFile(stray, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	var md map[string]any
	for i := 1; i <= 10; i++ {
		md = rt.commit(i)
	}
	keep := map[string]bool{}
	for _, e := range md["metadata-log"].([]any) {
		keep[strings.TrimPrefix(e.(map[string]any)["metadata-file"].(string), "file://")] = true
	}
	if len(keep) != 3 {
		t.Fatalf("expected a 3 entry metadata log, got %v", keep)
	}
	_, body := do(t, rt.ts, "GET", "/v1/namespaces/app/tables/t", "")
	keep[strings.TrimPrefix(body["metadata-location"].(string), "file://")] = true
	keep[stray] = true

	got := metadataJSONs(t, rt.loc)
	if len(got) != len(keep) {
		t.Fatalf("expected only the log + current + stray metadata files (%d), got %d: %v", len(keep), len(got), got)
	}
	for _, f := range got {
		if !keep[f] {
			t.Fatalf("unexpected metadata file left: %s", f)
		}
	}

	// A table can opt out.
	rt = newRetentionTable(t, Retention{}, `{"write.metadata.previous-versions-max":"1","write.metadata.delete-after-commit.enabled":"false"}`)
	for i := 1; i <= 5; i++ {
		rt.commit(i)
	}
	if got := metadataJSONs(t, rt.loc); len(got) != 6 {
		t.Fatalf("expected every metadata file w/ delete-after-commit off, got %d", len(got))
	}
}
