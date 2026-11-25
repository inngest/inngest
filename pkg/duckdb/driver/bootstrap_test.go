package driver

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/duckdb/worker"
	"github.com/stretchr/testify/require"
)

func TestWorkerAuthTokenAndOnBootstrapHook(t *testing.T) {
	base := ""
	if runtime.GOOS != "windows" {
		base = "/tmp"
	}
	dir, err := os.MkdirTemp(base, "dw-auth-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sockPath := filepath.Join(dir, "w.sock")
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	const wantToken = "secret-worker-token-123"
	var sawBootstrapSQL atomic.Int32
	var badAuth atomic.Int32

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				var req worker.Request
				if err := worker.ReadFrame(conn, &req); err != nil {
					return
				}
				if req.AuthToken != wantToken {
					badAuth.Add(1)
					_ = worker.WriteFrame(conn, worker.Response{Kind: worker.KindError, Error: "unauthorized"})
					return
				}
				switch req.Op {
				case worker.OpPing:
					_ = worker.WriteFrame(conn, worker.Response{Kind: worker.KindOK, Version: "v1.0.0", PID: 1})
				case worker.OpOpenSession:
					_ = worker.WriteFrame(conn, worker.Response{Kind: worker.KindOK, SessionID: "s-1"})
				case worker.OpCloseSession:
					_ = worker.WriteFrame(conn, worker.Response{Kind: worker.KindOK})
				case worker.OpQuery:
					if strings.Contains(req.SQL, "CREATE TABLE bootstrap_marker") {
						sawBootstrapSQL.Add(1)
					}
					cols := []worker.ColumnInfo{{Name: "ok", Type: worker.TypeBigInt}}
					_ = worker.WriteFrame(conn, worker.Response{Kind: worker.KindMeta, Columns: cols})
					_ = worker.WriteFrame(conn, worker.Response{Kind: worker.KindChunk, Rows: worker.EncodeRows(cols, [][]any{{int64(1)}})})
					_ = worker.WriteFrame(conn, worker.Response{Kind: worker.KindDone})
				}
			}(c)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Wrong token must fail to open.
	if _, err := Open(ctx, Options{
		WorkerSocket:    sockPath,
		WorkerAuthToken: "wrong-token",
	}); err == nil {
		t.Fatal("expected Open with wrong WorkerAuthToken to fail")
	}

	var hookCalls atomic.Int32
	conn, db, err := OpenConnector(ctx, Options{
		WorkerSocket:    sockPath,
		WorkerAuthToken: wantToken,
		OnBootstrap: func(ctx context.Context, bdb *sql.DB) error {
			hookCalls.Add(1)
			_, err := bdb.ExecContext(ctx, "CREATE TABLE bootstrap_marker (id BIGINT)")
			return err
		},
	})
	if err != nil {
		t.Fatalf("OpenConnector: %v", err)
	}
	defer func() {
		_ = db.Close()
		_ = conn.Close()
	}()

	if got := hookCalls.Load(); got != 1 {
		t.Fatalf("OnBootstrap called %d times, want 1", got)
	}
	if got := sawBootstrapSQL.Load(); got != 1 {
		t.Fatalf("sawBootstrapSQL = %d, want 1", got)
	}

	// Simulate worker restart and verify OnBootstrap runs again cleanly without deadlocking.
	if err := conn.proc.restart(ctx); err != nil {
		t.Fatalf("restart: %v", err)
	}
	if got := hookCalls.Load(); got != 2 {
		t.Fatalf("OnBootstrap called %d times after restart, want 2", got)
	}
	if got := sawBootstrapSQL.Load(); got != 2 {
		t.Fatalf("sawBootstrapSQL = %d after restart, want 2", got)
	}
}

type mockSandboxedSidecar struct {
	sockPath string

	mu                    sync.Mutex
	externalAccessEnabled bool
	lakeAttached          bool
	installLoadCalls      int
	attachCalls           int
	stateDirCalls         int
	allowedDirsCalls      int
	sandboxToggleCalls    int
	blockedAfterSandbox   int
	onBootstrapCalls      int
}

func startMockSandboxedSidecar(t *testing.T, externalAccessEnabled, lakeAttached bool) *mockSandboxedSidecar {
	t.Helper()
	base := ""
	if runtime.GOOS != "windows" {
		base = "/tmp"
	}
	dir, err := os.MkdirTemp(base, "dw-sandbox-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	sockPath := filepath.Join(dir, "w.sock")
	ln, err := net.Listen("unix", sockPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	s := &mockSandboxedSidecar{
		sockPath:              sockPath,
		externalAccessEnabled: externalAccessEnabled,
		lakeAttached:          lakeAttached,
	}

	var sessSeq atomic.Int64
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				var req worker.Request
				if err := worker.ReadFrame(conn, &req); err != nil {
					return
				}
				switch req.Op {
				case worker.OpPing:
					_ = worker.WriteFrame(conn, worker.Response{Kind: worker.KindOK, Version: "v1.0.0", PID: 1})
				case worker.OpOpenSession:
					id := fmt.Sprintf("s-%d", sessSeq.Add(1))
					_ = worker.WriteFrame(conn, worker.Response{Kind: worker.KindOK, SessionID: id})
				case worker.OpCloseSession:
					_ = worker.WriteFrame(conn, worker.Response{Kind: worker.KindOK})
				case worker.OpQuery:
					s.handleQuery(conn, req.SQL)
				}
			}(c)
		}
	}()

	return s
}

func (s *mockSandboxedSidecar) handleQuery(conn net.Conn, sqlText string) {
	q := strings.TrimSpace(sqlText)
	upper := strings.ToUpper(q)

	writeStmtErr := func(msg string) {
		_ = worker.WriteFrame(conn, worker.Response{
			Kind:      worker.KindError,
			ErrorCode: worker.ErrCodeStatement,
			Error:     msg,
		})
	}
	writeRows := func(cols []worker.ColumnInfo, rows [][]any) {
		_ = worker.WriteFrame(conn, worker.Response{Kind: worker.KindMeta, Columns: cols})
		if len(rows) > 0 {
			_ = worker.WriteFrame(conn, worker.Response{Kind: worker.KindChunk, Rows: worker.EncodeRows(cols, rows)})
		}
		_ = worker.WriteFrame(conn, worker.Response{Kind: worker.KindDone})
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	switch {
	case strings.Contains(q, "current_setting('enable_external_access')"):
		cols := []worker.ColumnInfo{{Name: "v", Type: worker.TypeBoolean}}
		writeRows(cols, [][]any{{s.externalAccessEnabled}})
		return
	case strings.Contains(q, "duckdb_databases()"):
		cols := []worker.ColumnInfo{{Name: "attached", Type: worker.TypeBigInt}}
		if s.lakeAttached {
			writeRows(cols, [][]any{{int64(1)}})
		} else {
			writeRows(cols, nil)
		}
		return
	case strings.HasPrefix(upper, "INSTALL ") || strings.HasPrefix(upper, "LOAD "):
		s.installLoadCalls++
		if !s.externalAccessEnabled {
			s.blockedAfterSandbox++
			writeStmtErr("Permission Error: file system operations are disabled by configuration")
			return
		}
	case strings.HasPrefix(upper, "ATTACH "):
		s.attachCalls++
		if !s.externalAccessEnabled {
			s.blockedAfterSandbox++
			writeStmtErr("Permission Error: file system operations are disabled by configuration")
			return
		}
		s.lakeAttached = true
	case strings.HasPrefix(upper, "SET HOME_DIRECTORY=") ||
		strings.HasPrefix(upper, "SET EXTENSION_DIRECTORIES=") ||
		strings.HasPrefix(upper, "SET TEMP_DIRECTORY="):
		s.stateDirCalls++
		if !s.externalAccessEnabled {
			s.blockedAfterSandbox++
			writeStmtErr("Permission Error: cannot change directory setting when enable_external_access is disabled")
			return
		}
	case strings.HasPrefix(upper, "SET ALLOWED_DIRECTORIES="):
		s.allowedDirsCalls++
		if !s.externalAccessEnabled {
			s.blockedAfterSandbox++
			writeStmtErr("Permission Error: cannot change allowed_directories when enable_external_access is disabled")
			return
		}
	case upper == "SET ENABLE_EXTERNAL_ACCESS=FALSE;":
		s.sandboxToggleCalls++
		s.externalAccessEnabled = false
	case strings.Contains(q, "CREATE TABLE IF NOT EXISTS bootstrap_marker"):
		s.onBootstrapCalls++
	}

	cols := []worker.ColumnInfo{{Name: "ok", Type: worker.TypeBigInt}}
	writeRows(cols, [][]any{{int64(1)}})
}

// TestSandboxedSidecarReopenAndReconnect proves that once a shared external
// worker has been bootstrapped with Iceberg and RestrictExternalAccess, later
// OpenConnector calls and reconnects reuse the completed setup instead of
// repeating INSTALL/LOAD/ATTACH/SET allowed_directories that the sandbox blocks.
func TestSandboxedSidecarReopenAndReconnect(t *testing.T) {
	sidecar := startMockSandboxedSidecar(t, true, false)

	var namespaceHits atomic.Int32
	catalogSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		namespaceHits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(catalogSrv.Close)

	stateDir := t.TempDir()
	opts := Options{
		WorkerSocket:           sidecar.sockPath,
		StateDir:               stateDir,
		Iceberg:                &IcebergOptions{URI: catalogSrv.URL, Warehouse: "inngest", Token: "tok"},
		RestrictExternalAccess: true,
		ResourceLimits:         &ResourceLimits{MemoryLimit: "512MiB", Threads: 2},
		OnBootstrap: func(ctx context.Context, db *sql.DB) error {
			_, err := db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS bootstrap_marker (id BIGINT);")
			return err
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First open bootstraps state dir, Iceberg, resource limits, and enables sandbox.
	c1, db1, err := OpenConnector(ctx, opts)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db1.Close()
		_ = c1.Close()
	})

	sidecar.mu.Lock()
	require.False(t, sidecar.externalAccessEnabled)
	require.True(t, sidecar.lakeAttached)
	require.Equal(t, 4, sidecar.installLoadCalls)
	require.Equal(t, 1, sidecar.attachCalls)
	require.Equal(t, 3, sidecar.stateDirCalls)
	require.Equal(t, 1, sidecar.allowedDirsCalls)
	require.Equal(t, 1, sidecar.sandboxToggleCalls)
	require.Equal(t, 0, sidecar.blockedAfterSandbox)
	require.Equal(t, 1, sidecar.onBootstrapCalls)
	sidecar.mu.Unlock()

	// Reopening a new Connector against the same sandboxed sidecar must succeed
	// without repeating sandbox-blocked statements or hitting the namespace API.
	c2, db2, err := OpenConnector(ctx, opts)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db2.Close()
		_ = c2.Close()
	})

	// Reconnecting an existing process (restart) against the sandboxed sidecar must also succeed.
	require.NoError(t, c2.proc.restart(ctx))

	// Concurrent reopens against the sandboxed sidecar must be race-free.
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, db, err := OpenConnector(ctx, opts)
			if err != nil {
				errs <- err
				return
			}
			_ = db.Close()
			_ = c.Close()
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	sidecar.mu.Lock()
	require.Equal(t, 4, sidecar.installLoadCalls, "must not repeat INSTALL/LOAD after sandbox")
	require.Equal(t, 1, sidecar.attachCalls, "must not repeat ATTACH after sandbox")
	require.Equal(t, 3, sidecar.stateDirCalls, "must not repeat state dir SETs after sandbox")
	require.Equal(t, 1, sidecar.allowedDirsCalls, "must not repeat SET allowed_directories after sandbox")
	require.Equal(t, 1, sidecar.sandboxToggleCalls, "must not repeat SET enable_external_access after sandbox")
	require.Equal(t, 0, sidecar.blockedAfterSandbox, "must not issue statements blocked by sandbox")
	require.Equal(t, 7, sidecar.onBootstrapCalls, "OnBootstrap must still run on every open/reconnect")
	sidecar.mu.Unlock()
	require.Zero(t, namespaceHits.Load(), "must not trigger Iceberg namespace retry when already sandboxed")
}

// TestSandboxedSidecarFailsWhenIcebergNotAttached proves that if a shared
// sidecar is already sandboxed without LakeAlias attached, opening with Iceberg
// fails clearly instead of silently skipping Iceberg or retrying blocked SQL.
func TestSandboxedSidecarFailsWhenIcebergNotAttached(t *testing.T) {
	sidecar := startMockSandboxedSidecar(t, false, false)

	var namespaceHits atomic.Int32
	catalogSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		namespaceHits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(catalogSrv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _, err := OpenConnector(ctx, Options{
		WorkerSocket:           sidecar.sockPath,
		Iceberg:                &IcebergOptions{URI: catalogSrv.URL, Warehouse: "inngest"},
		RestrictExternalAccess: true,
	})
	require.Error(t, err)
	require.ErrorContains(t, err, `catalog "inngest" is not attached`)

	sidecar.mu.Lock()
	require.Zero(t, sidecar.installLoadCalls)
	require.Zero(t, sidecar.attachCalls)
	require.Zero(t, sidecar.blockedAfterSandbox)
	sidecar.mu.Unlock()
	require.Zero(t, namespaceHits.Load())
}
