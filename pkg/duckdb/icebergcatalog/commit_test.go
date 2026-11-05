package icebergcatalog

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	_ "github.com/apache/iceberg-go/catalog/rest"
)

const testSchema = `{"type":"struct","schema-id":0,"fields":[{"id":1,"name":"id","type":"string","required":false}]}`

// backends runs fn against a sqlite-backed and (if reachable) a
// postgres-backed server.
func backends(t *testing.T, fn func(t *testing.T, newServer func(t *testing.T) (*httptest.Server, string))) {
	t.Run("sqlite", func(t *testing.T) {
		fn(t, func(t *testing.T) (*httptest.Server, string) { return newTestServerAt(t, t.TempDir()) })
	})
	t.Run("postgres", func(t *testing.T) {
		requirePostgres(t)
		fn(t, func(t *testing.T) (*httptest.Server, string) {
			db, err := sql.Open("pgx", requirePostgres(t))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = db.Close() })
			wh := filepath.Join(t.TempDir(), "warehouse")
			srv, err := New(context.Background(), db, Options{Warehouse: wh})
			if err != nil {
				t.Fatal(err)
			}
			ts := httptest.NewServer(srv)
			t.Cleanup(ts.Close)
			return ts, wh
		})
	})
}

func createTestTable(t *testing.T) (func(body string) (int, map[string]any), map[string]any) {
	t.Helper()
	return createTestTableOn(t, newTestServer(t))
}

func createTestTableOn(t *testing.T, ts *httptest.Server) (func(body string) (int, map[string]any), map[string]any) {
	t.Helper()
	if status, body := do(t, ts, "POST", "/v1/namespaces", `{"namespace":["app"]}`); status != 200 {
		t.Fatalf("create ns: %d %v", status, body)
	}
	status, body := do(t, ts, "POST", "/v1/namespaces/app/tables", `{"name":"t","schema":`+testSchema+`}`)
	if status != 200 {
		t.Fatalf("create table: %d %v", status, body)
	}
	return func(b string) (int, map[string]any) {
		return do(t, ts, "POST", "/v1/namespaces/app/tables/t", b)
	}, body["metadata"].(map[string]any)
}

// TestRequirements checks every requirement type 409s when it doesn't hold
// and passes when it does.
func TestRequirements(t *testing.T) {
	commit, md := createTestTable(t)
	uuid := md["table-uuid"].(string)
	noop := `{"action":"set-properties","updates":{"a":"b"}}`

	for _, tc := range []struct {
		req  string
		want int
	}{
		{`{"type":"assert-create"}`, 409},
		{`{"type":"assert-table-uuid","uuid":"` + uuid + `"}`, 200},
		{`{"type":"assert-table-uuid","uuid":"00000000-0000-0000-0000-000000000000"}`, 409},
		{`{"type":"assert-ref-snapshot-id","ref":"main","snapshot-id":null}`, 200},
		{`{"type":"assert-ref-snapshot-id","ref":"main","snapshot-id":1}`, 409},
		{`{"type":"assert-last-assigned-field-id","last-assigned-field-id":1}`, 200},
		{`{"type":"assert-last-assigned-field-id","last-assigned-field-id":2}`, 409},
		{`{"type":"assert-current-schema-id","current-schema-id":0}`, 200},
		{`{"type":"assert-current-schema-id","current-schema-id":1}`, 409},
		{`{"type":"assert-last-assigned-partition-id","last-assigned-partition-id":999}`, 200},
		{`{"type":"assert-last-assigned-partition-id","last-assigned-partition-id":1000}`, 409},
		{`{"type":"assert-default-spec-id","default-spec-id":0}`, 200},
		{`{"type":"assert-default-spec-id","default-spec-id":1}`, 409},
		{`{"type":"assert-default-sort-order-id","default-sort-order-id":0}`, 200},
		{`{"type":"assert-default-sort-order-id","default-sort-order-id":1}`, 409},
		{`{"type":"assert-something-else"}`, 400},
		{`{"type":"assert-current-schema-id","current-schema-id":0,"extra":1}`, 400},
	} {
		status, body := commit(`{"requirements":[` + tc.req + `],"updates":[` + noop + `]}`)
		if status != tc.want {
			t.Errorf("%s: got %d, want %d: %v", tc.req, status, tc.want, body)
		}
		if tc.want == 409 && body["error"].(map[string]any)["type"] != "CommitFailedException" {
			t.Errorf("%s: %v", tc.req, body)
		}
	}
}

// TestCommitSemantics drives snapshots/refs/schemas through the REST API and
// checks the bookkeeping the catalog owns.
func TestCommitSemantics(t *testing.T) {
	commit, md := createTestTable(t)
	loc := md["location"].(string)
	snap := func(id, parent, seq int) string {
		p := ""
		if parent != 0 {
			p = fmt.Sprintf(`"parent-snapshot-id":%d,`, parent)
		}
		return fmt.Sprintf(`{"action":"add-snapshot","snapshot":{"snapshot-id":%d,%s"sequence-number":%d,"timestamp-ms":%d,"manifest-list":"%s/metadata/snap-%d.avro","summary":{"operation":"append"},"schema-id":0}}`,
			id, p, seq, int64(md["last-updated-ms"].(float64))+int64(id), loc, id)
	}
	main := func(id int) string {
		return fmt.Sprintf(`{"action":"set-snapshot-ref","ref-name":"main","type":"branch","snapshot-id":%d}`, id)
	}
	ok := func(body string) map[string]any {
		t.Helper()
		status, out := commit(body)
		if status != 200 {
			t.Fatalf("commit %s: %d %v", body, status, out)
		}
		return out
	}

	// Two snapshots in one commit, main set to both: only the last one
	// stays in the snapshot log (like duckdb's multi-statement txns).
	out := ok(`{"requirements":[{"type":"assert-ref-snapshot-id","ref":"main","snapshot-id":null}],"updates":[` +
		snap(1, 0, 1) + "," + main(1) + "," + snap(2, 1, 2) + "," + main(2) + `]}`)
	m := out["metadata"].(map[string]any)
	if m["current-snapshot-id"].(float64) != 2 || m["last-sequence-number"].(float64) != 2 || len(m["snapshot-log"].([]any)) != 1 {
		t.Fatalf("unexpected metadata: %v", m)
	}
	if len(m["metadata-log"].([]any)) != 1 || !strings.Contains(out["metadata-location"].(string), "/metadata/00001-") {
		t.Fatalf("unexpected metadata log/location: %v %v", m["metadata-log"], out["metadata-location"])
	}

	// Stale assert-ref-snapshot-id loses.
	if status, _ := commit(`{"requirements":[{"type":"assert-ref-snapshot-id","ref":"main","snapshot-id":1}],"updates":[` + snap(3, 1, 3) + `]}`); status != 409 {
		t.Fatalf("stale ref: %d", status)
	}
	// Sequence numbers have to move forward.
	if status, _ := commit(`{"requirements":[],"updates":[` + snap(3, 2, 2) + `]}`); status != 400 {
		t.Fatalf("stale sequence number: %d", status)
	}
	// Can't remove the current snapshot, can remove an old one (and its
	// log entry).
	if status, _ := commit(`{"requirements":[],"updates":[{"action":"remove-snapshots","snapshot-ids":[2]}]}`); status != 400 {
		t.Fatalf("removing current snapshot: %d", status)
	}
	m = ok(`{"requirements":[],"updates":[{"action":"remove-snapshots","snapshot-ids":[1]}]}`)["metadata"].(map[string]any)
	if len(m["snapshots"].([]any)) != 1 {
		t.Fatalf("snapshot not removed: %v", m["snapshots"])
	}

	// add-schema: new id, last-column-id bumps, -1 resolves, re-adding
	// the same schema reuses its id.
	schema1 := `{"type":"struct","schema-id":7,"fields":[{"id":1,"name":"id","type":"string","required":false},{"id":2,"name":"n","type":"long","required":false}]}`
	m = ok(`{"requirements":[{"type":"assert-current-schema-id","current-schema-id":0}],"updates":[{"action":"add-schema","schema":` + schema1 + `},{"action":"set-current-schema","schema-id":-1}]}`)["metadata"].(map[string]any)
	if m["current-schema-id"].(float64) != 7 || m["last-column-id"].(float64) != 2 || len(m["schemas"].([]any)) != 2 {
		t.Fatalf("add-schema: %v", m)
	}
	m = ok(`{"requirements":[],"updates":[{"action":"add-schema","schema":` + strings.Replace(schema1, `"schema-id":7`, `"schema-id":0`, 1) + `},{"action":"set-current-schema","schema-id":-1}]}`)["metadata"].(map[string]any)
	if m["current-schema-id"].(float64) != 7 || len(m["schemas"].([]any)) != 2 {
		t.Fatalf("re-adding a schema should reuse it: %v", m)
	}
	// VARIANT needs v3.
	variant := `{"type":"struct","fields":[{"id":3,"name":"v","type":"variant","required":false}]}`
	if status, _ := commit(`{"requirements":[],"updates":[{"action":"add-schema","schema":` + variant + `}]}`); status != 400 {
		t.Fatalf("variant on v2: %d", status)
	}
	m = ok(`{"requirements":[],"updates":[{"action":"upgrade-format-version","format-version":3},{"action":"add-schema","schema":` + variant + `}]}`)["metadata"].(map[string]any)
	if m["format-version"].(float64) != 3 || m["next-row-id"].(float64) != 0 {
		t.Fatalf("upgrade to v3: %v", m)
	}
	if status, _ := commit(`{"requirements":[],"updates":[{"action":"upgrade-format-version","format-version":2}]}`); status != 400 {
		t.Fatalf("downgrade: %d", status)
	}
	// v3 snapshots need row lineage.
	if status, _ := commit(`{"requirements":[],"updates":[` + snap(4, 2, 4) + `]}`); status != 400 {
		t.Fatalf("v3 snapshot w/o first-row-id: %d", status)
	}
	v3snap := strings.Replace(snap(4, 2, 4), `"schema-id":0`, `"schema-id":0,"first-row-id":0,"added-rows":5`, 1)
	m = ok(`{"requirements":[],"updates":[` + v3snap + `]}`)["metadata"].(map[string]any)
	if m["next-row-id"].(float64) != 5 {
		t.Fatalf("next-row-id: %v", m["next-row-id"])
	}
}

// TestConcurrentCommits checks the metadata location CAS: concurrent
// commits either land or 409, and every landed one is in the history.
func TestConcurrentCommits(t *testing.T) {
	backends(t, testConcurrentCommits)
}

func testConcurrentCommits(t *testing.T, newServer func(t *testing.T) (*httptest.Server, string)) {
	ts, _ := newServer(t)
	commit, _ := createTestTableOn(t, ts)
	const n = 16
	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		codes = map[int]int{}
	)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, _ := commit(fmt.Sprintf(`{"requirements":[],"updates":[{"action":"set-properties","updates":{"k%d":"v"}}]}`, i))
			mu.Lock()
			codes[status]++
			mu.Unlock()
		}()
	}
	wg.Wait()
	if codes[200]+codes[409] != n || codes[200] == 0 {
		t.Fatalf("unexpected statuses: %v", codes)
	}
	status, out := commit(`{"requirements":[],"updates":[{"action":"set-properties","updates":{"final":"v"}}]}`)
	if status != 200 {
		t.Fatalf("final commit: %d %v", status, out)
	}
	m := out["metadata"].(map[string]any)
	props := m["properties"].(map[string]any)
	if len(props) != codes[200]+1 || len(m["metadata-log"].([]any)) != codes[200]+1 {
		t.Fatalf("lost a commit: %d landed, props %v, log %d", codes[200], props, len(m["metadata-log"].([]any)))
	}
}

func TestMultiTableTransaction(t *testing.T) {
	ts := newTestServer(t)
	if status, _ := do(t, ts, "POST", "/v1/transactions/commit", `{"table-changes":[]}`); status != http.StatusNotAcceptable {
		t.Fatalf("got %d", status)
	}
}

// TestIcebergGoClient commits through iceberg-go's own REST client + table
// API (schema evolution, properties, and a real parquet append), so a
// second Iceberg implementation round-trips our metadata.
func TestIcebergGoClient(t *testing.T) {
	backends(t, testIcebergGoClient)
}

func testIcebergGoClient(t *testing.T, newServer func(t *testing.T) (*httptest.Server, string)) {
	ctx := context.Background()
	ts, wh := newServer(t)
	cat, err := catalog.Load(ctx, "test", iceberg.Properties{"type": "rest", "uri": ts.URL})
	if err != nil {
		t.Fatal(err)
	}
	if err := cat.CreateNamespace(ctx, catalog.ToIdentifier("app"), nil); err != nil {
		t.Fatal(err)
	}
	ident := catalog.ToIdentifier("app", "events")
	schema := iceberg.NewSchema(0,
		iceberg.NestedField{ID: 1, Name: "id", Type: iceberg.PrimitiveTypes.Int64, Required: false},
		iceberg.NestedField{ID: 2, Name: "name", Type: iceberg.PrimitiveTypes.String, Required: false},
	)
	tbl, err := cat.CreateTable(ctx, ident, schema, catalog.WithProperties(iceberg.Properties{"owner": "test"}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(tbl.Location(), wh) {
		t.Fatalf("location %q not under %q", tbl.Location(), wh)
	}

	txn := tbl.NewTransaction()
	if err := txn.SetProperties(iceberg.Properties{"k": "v"}); err != nil {
		t.Fatal(err)
	}
	if err := txn.UpdateSchema(true, false).AddColumn([]string{"extra"}, iceberg.PrimitiveTypes.Bool, "", false, nil).Commit(); err != nil {
		t.Fatal(err)
	}
	tbl, err = txn.Commit(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mem := memory.DefaultAllocator
	arrSchema := arrow.NewSchema([]arrow.Field{
		{Name: "id", Type: arrow.PrimitiveTypes.Int64, Nullable: true},
		{Name: "name", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "extra", Type: arrow.FixedWidthTypes.Boolean, Nullable: true},
	}, nil)
	bldr := array.NewRecordBuilder(mem, arrSchema)
	defer bldr.Release()
	bldr.Field(0).(*array.Int64Builder).AppendValues([]int64{1, 2, 3}, nil)
	bldr.Field(1).(*array.StringBuilder).AppendValues([]string{"a", "b", "c"}, nil)
	bldr.Field(2).(*array.BooleanBuilder).AppendValues([]bool{true, false, true}, nil)
	rec := bldr.NewRecordBatch()
	defer rec.Release()
	arrTbl := array.NewTableFromRecords(arrSchema, []arrow.RecordBatch{rec})
	defer arrTbl.Release()
	for range 2 {
		txn = tbl.NewTransaction()
		if err := txn.AppendTable(ctx, arrTbl, 1024, nil); err != nil {
			t.Fatal(err)
		}
		if tbl, err = txn.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}

	tbl, err = cat.LoadTable(ctx, ident)
	if err != nil {
		t.Fatal(err)
	}
	if tbl.Properties()["k"] != "v" || tbl.Properties()["owner"] != "test" {
		t.Fatalf("properties: %v", tbl.Properties())
	}
	if len(tbl.Schema().Fields()) != 3 {
		t.Fatalf("schema: %v", tbl.Schema())
	}
	if n := len(tbl.Metadata().Snapshots()); n != 2 {
		t.Fatalf("expected 2 snapshots, got %d", n)
	}
	got, err := tbl.Scan().ToArrowTable(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer got.Release()
	if got.NumRows() != 6 {
		t.Fatalf("expected 6 rows, got %d", got.NumRows())
	}

	// Rename + drop go through the catalog too.
	to := catalog.ToIdentifier("app", "events2")
	if _, err := cat.RenameTable(ctx, ident, to); err != nil {
		t.Fatal(err)
	}
	if ok, _ := cat.CheckTableExists(ctx, ident); ok {
		t.Fatal("old name still exists")
	}
	if err := cat.DropTable(ctx, to); err != nil {
		t.Fatal(err)
	}
	// Metadata files were all written under the table's metadata dir.
	files, _ := filepath.Glob(filepath.Join(tbl.Location(), "metadata", "*.metadata.json"))
	if len(files) < 4 {
		t.Fatalf("expected >= 4 metadata files, got %v", files)
	}
	if _, err := os.Stat(files[0]); err != nil {
		t.Fatal(err)
	}
}
