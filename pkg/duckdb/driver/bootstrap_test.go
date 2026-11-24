package driver

import (
	"context"
	"database/sql"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/duckdb/worker"
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
