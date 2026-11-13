package icebergcatalog

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/apache/iceberg-go"
	sqlcat "github.com/apache/iceberg-go/catalog/sql"
)

func TestValidateName(t *testing.T) {
	for _, name := range []string{"", "a/b", `a\b`, "..", "a..b", "a\x00b", "a\nb", "a\x7fb"} {
		if err := validateName(name); err == nil {
			t.Errorf("expected %q to be rejected", name)
		}
	}
	for _, name := range []string{"app", "run_trace_spans", "a.b", "Ünïcode"} {
		if err := validateName(name); err != nil {
			t.Errorf("expected %q to be accepted: %v", name, err)
		}
	}
}

func TestCheckLocation(t *testing.T) {
	dir := t.TempDir()
	wh := filepath.Join(dir, "wh")
	if err := os.MkdirAll(filepath.Join(wh, "app.db"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A symlink inside the warehouse pointing out of it.
	if err := os.Symlink(dir, filepath.Join(wh, "escape")); err != nil {
		t.Fatal(err)
	}

	local := &Server{opts: Options{Warehouse: wh}}
	s3 := &Server{opts: Options{Warehouse: "s3://bucket/wh"}}
	none := &Server{}

	for _, tc := range []struct {
		name string
		srv  *Server
		loc  string
		want string // empty means rejected
	}{
		{"local table", local, filepath.Join(wh, "app.db", "runs"), filepath.Join(wh, "app.db", "runs")},
		{"local warehouse itself", local, wh, wh},
		{"local file url", local, "file://" + filepath.Join(wh, "x"), filepath.Join(wh, "x")},
		{"local double slash", local, wh + "//app.db/runs", filepath.Join(wh, "app.db", "runs")},
		{"local dot segment", local, wh + "/app.db/./runs", ""},
		{"local dotdot inside", local, wh + "/app.db/../app.db/runs", ""},
		{"local dotdot escape", local, wh + "/../other", ""},
		{"local sibling prefix", local, wh + "-evil/runs", ""},
		{"local outside", local, "/etc/passwd", ""},
		{"local relative", local, "app.db/runs", ""},
		{"local symlink escape", local, filepath.Join(wh, "escape", "runs"), ""},
		{"local to s3", local, "s3://bucket/wh/x", ""},
		{"local nul", local, wh + "/a\x00b", ""},
		{"s3 table", s3, "s3://bucket/wh/app.db/runs", "s3://bucket/wh/app.db/runs"},
		{"s3 dotdot inside", s3, "s3://bucket/wh/a/../b", ""},
		{"s3 dotdot escape", s3, "s3://bucket/wh/../other", ""},
		{"s3 sibling prefix", s3, "s3://bucket/wh-evil/x", ""},
		{"s3 other bucket", s3, "s3://evil/wh/x", ""},
		{"s3 other scheme", s3, "gs://bucket/wh/x", ""},
		{"s3 local path", s3, "/bucket/wh/x", ""},
		{"no warehouse", none, "/tmp/x", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.srv.checkLocation(tc.loc)
			if tc.want == "" {
				if err == nil {
					t.Fatalf("expected %q to be rejected, got %q", tc.loc, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected %q to be accepted: %v", tc.loc, err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestRejectsEscapes checks every request that carries a name or a location
// 400s when it'd point outside the warehouse.
func TestRejectsEscapes(t *testing.T) {
	ts := newTestServer(t)
	if status, body := do(t, ts, "POST", "/v1/namespaces", `{"namespace":["app"]}`); status != 200 {
		t.Fatalf("create ns: %d %v", status, body)
	}
	schema := `{"type":"struct","schema-id":0,"fields":[{"id":1,"name":"id","type":"string","required":false}]}`
	if status, body := do(t, ts, "POST", "/v1/namespaces/app/tables", `{"name":"t","schema":`+schema+`}`); status != 200 {
		t.Fatalf("create table: %d %v", status, body)
	}

	for _, tc := range []struct {
		name, method, path, body string
	}{
		{"namespace w/ slash", "POST", "/v1/namespaces", `{"namespace":["a/b"]}`},
		{"namespace w/ dotdot", "POST", "/v1/namespaces", `{"namespace":[".."]}`},
		{"namespace w/ control char", "POST", "/v1/namespaces", `{"namespace":["a\u0001"]}`},
		{"namespace location outside", "POST", "/v1/namespaces", `{"namespace":["b"],"properties":{"location":"/tmp/evil"}}`},
		{"namespace location update", "POST", "/v1/namespaces/app/properties", `{"updates":{"location":"/tmp/evil"}}`},
		{"encoded slash in path ns", "GET", "/v1/namespaces/a%2F..%2Fb/tables", ""},
		{"encoded slash in path table", "GET", "/v1/namespaces/app/tables/a%2Fb", ""},
		{"table name w/ dotdot", "POST", "/v1/namespaces/app/tables", `{"name":"..","schema":` + schema + `}`},
		{"table name w/ backslash", "POST", "/v1/namespaces/app/tables", `{"name":"a\\b","schema":` + schema + `}`},
		{"table location outside", "POST", "/v1/namespaces/app/tables", `{"name":"u","location":"/tmp/evil","schema":` + schema + `}`},
		{"stage-create location outside", "POST", "/v1/namespaces/app/tables", `{"name":"u","stage-create":true,"location":"/tmp/evil","schema":` + schema + `}`},
		{"table data path outside", "POST", "/v1/namespaces/app/tables", `{"name":"u","properties":{"write.data.path":"/tmp/evil"},"schema":` + schema + `}`},
		{"commit set-location", "POST", "/v1/namespaces/app/tables/t", `{"requirements":[],"updates":[{"action":"set-location","location":"/tmp/evil"}]}`},
		{"commit set-properties", "POST", "/v1/namespaces/app/tables/t", `{"requirements":[],"updates":[{"action":"set-properties","updates":{"write.metadata.path":"/tmp/evil"}}]}`},
		{"rename to bad name", "POST", "/v1/tables/rename", `{"source":{"namespace":["app"],"name":"t"},"destination":{"namespace":["app"],"name":"../x"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := do(t, ts, tc.method, tc.path, tc.body)
			if status != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d %v", status, body)
			}
			if typ, _ := body["error"].(map[string]any)["type"].(string); tc.method != "HEAD" && !strings.Contains(typ, "BadRequest") {
				t.Fatalf("expected BadRequestException, got %v", body)
			}
		})
	}

	// The table is untouched.
	status, body := do(t, ts, "GET", "/v1/namespaces/app/tables/t", "")
	if status != 200 {
		t.Fatalf("load table: %d %v", status, body)
	}
	if loc := body["metadata"].(map[string]any)["location"].(string); strings.Contains(loc, "evil") {
		t.Fatalf("location changed: %s", loc)
	}
}

// TestCommitValidation checks commits are validated against iceberg-go's own
// decode and the metadata they'd produce, not a separate parse of the body.
func TestCommitValidation(t *testing.T) {
	ts := newTestServer(t)
	if status, body := do(t, ts, "POST", "/v1/namespaces", `{"namespace":["app"]}`); status != 200 {
		t.Fatalf("create ns: %d %v", status, body)
	}
	schema := `{"type":"struct","schema-id":0,"fields":[{"id":1,"name":"id","type":"string","required":false}]}`
	status, body := do(t, ts, "POST", "/v1/namespaces/app/tables", `{"name":"t","schema":`+schema+`}`)
	if status != 200 {
		t.Fatalf("create table: %d %v", status, body)
	}
	loc := body["metadata"].(map[string]any)["location"].(string)

	snapshot := func(id int, manifestList string) string {
		return fmt.Sprintf(`{"action":"add-snapshot","snapshot":{"snapshot-id":%d,"sequence-number":%d,"timestamp-ms":%d,"manifest-list":%q,"summary":{"operation":"append"},"schema-id":0}}`, id, id, time.Now().UnixMilli()+int64(id), manifestList)
	}
	commit := func(updates ...string) (int, map[string]any) {
		return do(t, ts, "POST", "/v1/namespaces/app/tables/t", `{"requirements":[],"updates":[`+strings.Join(updates, ",")+`]}`)
	}

	// Positive control, so the rejections below aren't just malformed.
	if status, body := commit(snapshot(1, loc+"/metadata/snap-1.avro")); status != 200 {
		t.Fatalf("valid add-snapshot: %d %v", status, body)
	}
	stats := func(path string) string {
		return fmt.Sprintf(`{"action":"set-statistics","snapshot-id":1,"statistics":{"snapshot-id":1,"statistics-path":%q,"file-size-in-bytes":1,"file-footer-size-in-bytes":1,"blob-metadata":[]}}`, path)
	}
	partStats := func(path string) string {
		return fmt.Sprintf(`{"action":"set-partition-statistics","partition-statistics":{"snapshot-id":1,"statistics-path":%q,"file-size-in-bytes":1}}`, path)
	}
	if status, body := commit(stats(loc + "/metadata/1.stats")); status != 200 {
		t.Fatalf("valid set-statistics: %d %v", status, body)
	}
	if status, body := commit(partStats(loc + "/metadata/1.pstats")); status != 200 {
		t.Fatalf("valid set-partition-statistics: %d %v", status, body)
	}
	if status, body := commit(`{"action":"set-properties","updates":{"write.data.path":"` + loc + `/elsewhere"}}`); status != 200 {
		t.Fatalf("valid write.data.path: %d %v", status, body)
	}

	for _, tc := range []struct {
		name    string
		updates []string
	}{
		{"mixed case action", []string{`{"Action":"set-location","location":"/tmp/evil"}`}},
		{"duplicate action keys", []string{`{"action":"set-properties","action":"set-location","location":"/tmp/evil","updates":{}}`}},
		{"case variant duplicate", []string{`{"action":"set-properties","ACTION":"set-location","location":"/tmp/evil","updates":{}}`}},
		{"set-location outside", []string{`{"action":"set-location","location":"/tmp/evil"}`}},
		{"manifest-list outside", []string{snapshot(2, "/tmp/evil/snap-2.avro")}},
		{"manifest-list in another table", []string{snapshot(2, filepath.Dir(loc)+"/other/snap-2.avro")}},
		{"statistics outside", []string{stats("/tmp/evil/1.stats")}},
		{"partition statistics outside", []string{partStats("/tmp/evil/1.pstats")}},
		{"write.data.path outside", []string{`{"action":"set-properties","updates":{"write.data.path":"/tmp/evil"}}`}},
		{"write.metadata.path outside", []string{`{"action":"set-properties","updates":{"write.metadata.path":"/tmp/evil"}}`}},
		{"write.object-storage.path outside", []string{`{"action":"set-properties","updates":{"write.object-storage.path":"/tmp/evil"}}`}},
		{"legacy updated field", []string{`{"action":"set-properties","updated":{"write.data.path":"/tmp/evil"}}`}},
		{"unknown action", []string{`{"action":"do-something"}`}},
		{"disallowed action", []string{`{"action":"remove-schemas","schema-ids":[0]}`}},
		{"bad update after good one", []string{`{"action":"set-properties","updates":{"a":"b"}}`, `{"action":"set-location","location":"/tmp/evil"}`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := commit(tc.updates...)
			if status != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d %v", status, body)
			}
			t.Logf("%v", body["error"].(map[string]any)["message"])
		})
	}

	_, body = do(t, ts, "GET", "/v1/namespaces/app/tables/t", "")
	md := body["metadata"].(map[string]any)
	if md["location"] != loc || strings.Contains(fmt.Sprint(md), "evil") {
		t.Fatalf("metadata changed: %v", md)
	}
}

func TestPurge(t *testing.T) {
	dir := t.TempDir()
	wh := filepath.Join(dir, "warehouse")
	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	cat, err := sqlcat.NewCatalog("inngest", db, sqlcat.SQLite, iceberg.Properties{"warehouse": wh})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New(cat, Options{Warehouse: wh}))
	t.Cleanup(ts.Close)

	schema := `{"type":"struct","schema-id":0,"fields":[{"id":1,"name":"id","type":"string","required":false}]}`
	if status, body := do(t, ts, "POST", "/v1/namespaces", `{"namespace":["app"]}`); status != 200 {
		t.Fatalf("create ns: %d %v", status, body)
	}
	status, body := do(t, ts, "POST", "/v1/namespaces/app/tables", `{"name":"t","schema":`+schema+`}`)
	if status != 200 {
		t.Fatalf("create table: %d %v", status, body)
	}
	loc := body["metadata"].(map[string]any)["location"].(string)
	sibling := filepath.Join(filepath.Dir(loc), "t2")
	if err := os.MkdirAll(sibling, 0o755); err != nil {
		t.Fatal(err)
	}

	if status, body := do(t, ts, "DELETE", "/v1/namespaces/app/tables/t?purgeRequested=true", ""); status != http.StatusNoContent {
		t.Fatalf("purge: %d %v", status, body)
	}
	if _, err := os.Stat(loc); !os.IsNotExist(err) {
		t.Fatalf("table dir still there: %v", err)
	}
	if _, err := os.Stat(sibling); err != nil {
		t.Fatalf("purge touched a sibling: %v", err)
	}

	// A table located at the warehouse root can't be purged.
	if status, body := do(t, ts, "POST", "/v1/namespaces/app/tables", `{"name":"root","location":"`+wh+`","schema":`+schema+`}`); status != 200 {
		t.Fatalf("create root table: %d %v", status, body)
	}
	if status, _ := do(t, ts, "DELETE", "/v1/namespaces/app/tables/root?purgeRequested=true", ""); status != http.StatusBadRequest {
		t.Fatalf("expected root purge to 400, got %d", status)
	}
	if _, err := os.Stat(sibling); err != nil {
		t.Fatalf("warehouse was purged: %v", err)
	}
}
