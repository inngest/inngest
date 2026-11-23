package server_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/duckdb/worker"
	"github.com/inngest/inngest/pkg/duckdb/worker/libduckdb"
	"github.com/inngest/inngest/pkg/duckdb/worker/server"
)

func findRealLib(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("DUCKDB_LIB_PATH"); p != "" {
		return p
	}
	const fallback = "/tmp/duckdb-purego-test/libduckdb.dylib"
	if _, err := os.Stat(fallback); err == nil {
		return fallback
	}
	t.Skip("real libduckdb not available; set DUCKDB_LIB_PATH to run worker integration tests")
	return ""
}

func collectAll(t *testing.T, stream *worker.QueryStream) [][]any {
	t.Helper()
	defer stream.Close()
	var all [][]any
	for {
		chunk, err := stream.NextChunk()
		if err != nil {
			t.Fatalf("NextChunk: %v", err)
		}
		if chunk == nil {
			break
		}
		all = append(all, chunk...)
	}
	return all
}

func tempSocketPath(t *testing.T, prefix string) string {
	t.Helper()
	base := ""
	if runtime.GOOS != "windows" {
		base = "/tmp"
	}
	dir, err := os.MkdirTemp(base, prefix)
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return filepath.Join(dir, "w.sock")
}

func TestWorkerExternalSidecarMode(t *testing.T) {
	libPath := findRealLib(t)
	lib, err := libduckdb.Load(libPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	defer lib.Close()

	db, err := lib.Open(":memory:", nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	sockPath := tempSocketPath(t, "dw-*")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := server.New(db, lib.Version())
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.ListenAndServe(ctx, sockPath)
	}()

	var client *worker.Client
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		client, err = worker.Dial(ctx, sockPath)
		if err == nil {
			break
		}
		time.Sleep(15 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("Dial sidecar: %v", err)
	}
	defer client.Close()

	ver, err := client.Ping(ctx)
	if err != nil || ver == "" {
		t.Fatalf("Ping = (%q, %v), want non-empty version", ver, err)
	}

	sess, err := client.OpenSession(ctx)
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	defer sess.Close(ctx)

	if err := sess.Exec(ctx, "CREATE TABLE items (id BIGINT, name VARCHAR)"); err != nil {
		t.Fatalf("Exec CREATE TABLE: %v", err)
	}
	if err := sess.Exec(ctx, "INSERT INTO items VALUES (9007199254740993, 'alpha'), (2, 'this is a long string > 12 bytes')"); err != nil {
		t.Fatalf("Exec INSERT: %v", err)
	}

	stream, err := sess.Query(ctx, `SELECT
		NULL::INTEGER,
		true,
		CAST(-9007199254740993 AS BIGINT),
		CAST(18000000000000000000 AS UBIGINT),
		'short',
		'long varchar exceeding twelve byte inline limit',
		UUID '01234567-89ab-cdef-fedc-ba9876543210',
		TIMESTAMPTZ '2026-11-23 17:30:45.123456+00',
		DATE '2026-11-23',
		'\xCA\xFE\xBA\xBE'::BLOB`)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	rows := collectAll(t, stream)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	r := rows[0]
	if r[0] != nil {
		t.Errorf("null = %#v", r[0])
	}
	if r[1] != true {
		t.Errorf("bool = %#v", r[1])
	}
	if r[2] != int64(-9007199254740993) {
		t.Errorf("bigint = %#v", r[2])
	}
	if r[3] != uint64(18000000000000000000) {
		t.Errorf("ubigint = %#v", r[3])
	}
	if r[4] != "short" {
		t.Errorf("short varchar = %#v", r[4])
	}
	if r[5] != "long varchar exceeding twelve byte inline limit" {
		t.Errorf("long varchar = %#v", r[5])
	}
	if r[6] != "01234567-89ab-cdef-fedc-ba9876543210" {
		t.Errorf("uuid = %#v", r[6])
	}
	wantTS := time.Date(2026, 11, 23, 17, 30, 45, 123456000, time.UTC)
	if ts, ok := r[7].(time.Time); !ok || !ts.Equal(wantTS) {
		t.Errorf("tstz = %#v, want %v", r[7], wantTS)
	}
	wantDate := time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC)
	if d, ok := r[8].(time.Time); !ok || !d.Equal(wantDate) {
		t.Errorf("date = %#v, want %v", r[8], wantDate)
	}
	if b, ok := r[9].([]byte); !ok || !bytes.Equal(b, []byte{0xCA, 0xFE, 0xBA, 0xBE}) {
		t.Errorf("blob = %#v", r[9])
	}

	const n = 4
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			s, err := client.OpenSession(ctx)
			if err != nil {
				errs <- err
				return
			}
			defer s.Close(ctx)
			q, err := s.Query(ctx, fmt.Sprintf("SELECT range + %d FROM range(3000)", id))
			if err != nil {
				errs <- err
				return
			}
			defer q.Close()
			total := 0
			for {
				ch, err := q.NextChunk()
				if err != nil {
					errs <- err
					return
				}
				if ch == nil {
					break
				}
				total += len(ch)
			}
			if total != 3000 {
				errs <- fmt.Errorf("session %d: got %d rows, want 3000", id, total)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}

	cancelCtx, cancelQuery := context.WithTimeout(ctx, 150*time.Millisecond)
	defer cancelQuery()
	_, err = sess.Query(cancelCtx, "SELECT count(*) FROM range(100000000) a, range(1000) b")
	if err == nil {
		t.Fatal("expected cancelled query to fail")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context error on interrupted query, got: %v", err)
	}

	q2, err := sess.Query(ctx, "SELECT 42")
	if err != nil {
		t.Fatalf("Query after interrupt: %v", err)
	}
	got := collectAll(t, q2)
	if len(got) != 1 || got[0][0] != int32(42) {
		t.Fatalf("Query after interrupt = %#v, want [[42]]", got)
	}

	if err := client.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	select {
	case err := <-serveErr:
		if err != nil {
			t.Fatalf("ListenAndServe returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for server to exit after Shutdown")
	}
}

func TestWorkerManagedSubprocessAndCrashRestart(t *testing.T) {
	libPath := findRealLib(t)

	binDir := t.TempDir()
	workerBin := filepath.Join(binDir, "duckdb-worker")
	if runtime.GOOS == "windows" {
		workerBin += ".exe"
	}
	buildCmd := exec.Command("go", "build", "-o", workerBin, "../../../../cmd/duckdb-worker")
	buildCmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("building duckdb-worker: %v\n%s", err, out)
	}

	sockPath := tempSocketPath(t, "dwm-*")

	ctx := context.Background()
	client, err := worker.Spawn(ctx, worker.SpawnConfig{
		WorkerBinary: workerBin,
		SocketPath:   sockPath,
		LibPath:      libPath,
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	ver, err := client.Ping(ctx)
	if err != nil || ver == "" {
		t.Fatalf("Ping managed worker: (%q, %v)", ver, err)
	}
	t.Logf("managed worker PID=%d version=%s", client.PID(), ver)

	sess, err := client.OpenSession(ctx)
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	q, err := sess.Query(ctx, "SELECT 'managed_ok'::VARCHAR, 123::BIGINT")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	rows := collectAll(t, q)
	if len(rows) != 1 || rows[0][0] != "managed_ok" || rows[0][1] != int64(123) {
		t.Fatalf("unexpected rows: %#v", rows)
	}

	// Simulate an unexpected worker crash and verify detection + clean restart.
	proc, err := os.FindProcess(client.PID())
	if err != nil {
		t.Fatalf("FindProcess: %v", err)
	}
	if err := proc.Kill(); err != nil {
		t.Fatalf("Kill worker: %v", err)
	}
	select {
	case <-client.WaitExit():
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for killed worker exit notification")
	}
	if client.Alive() {
		t.Fatal("expected Alive() == false after crash")
	}
	_ = client.Close()

	// Restart a fresh worker on the exact same socket path (must clean up stale socket).
	client2, err := worker.Spawn(ctx, worker.SpawnConfig{
		WorkerBinary: workerBin,
		SocketPath:   sockPath,
		LibPath:      libPath,
	})
	if err != nil {
		t.Fatalf("Respawn after crash: %v", err)
	}
	defer client2.Close()

	sess2, err := client2.OpenSession(ctx)
	if err != nil {
		t.Fatalf("OpenSession after respawn: %v", err)
	}
	defer sess2.Close(ctx)

	q2, err := sess2.Query(ctx, "SELECT 'restarted_ok'::VARCHAR")
	if err != nil {
		t.Fatalf("Query after respawn: %v", err)
	}
	rows2 := collectAll(t, q2)
	if len(rows2) != 1 || rows2[0][0] != "restarted_ok" {
		t.Fatalf("unexpected rows after respawn: %#v", rows2)
	}
}
