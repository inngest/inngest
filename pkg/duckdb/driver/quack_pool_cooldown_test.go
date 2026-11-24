package driver

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/duckdb/worker"
	"github.com/stretchr/testify/require"
)

type fakeWorkerServer struct {
	sockPath string

	mu     sync.Mutex
	ln     net.Listener
	wg     sync.WaitGroup
	sessID atomic.Int64
}

func startFakeWorkerServer(t *testing.T, sockPath string) *fakeWorkerServer {
	t.Helper()
	s := &fakeWorkerServer{sockPath: sockPath}
	require.NoError(t, s.start())
	t.Cleanup(s.stop)
	return s
}

func (s *fakeWorkerServer) start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ln != nil {
		return nil
	}
	_ = os.Remove(s.sockPath)
	ln, err := net.Listen("unix", s.sockPath)
	if err != nil {
		return err
	}
	s.ln = ln
	s.wg.Add(1)
	go s.acceptLoop(ln)
	return nil
}

func (s *fakeWorkerServer) stop() {
	s.mu.Lock()
	ln := s.ln
	s.ln = nil
	s.mu.Unlock()
	if ln != nil {
		_ = ln.Close()
	}
	_ = os.Remove(s.sockPath)
	s.wg.Wait()
}

func (s *fakeWorkerServer) acceptLoop(ln net.Listener) {
	defer s.wg.Done()
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		s.wg.Add(1)
		go func(c net.Conn) {
			defer s.wg.Done()
			defer c.Close()
			s.handleConn(c)
		}(conn)
	}
}

func (s *fakeWorkerServer) handleConn(conn net.Conn) {
	var req worker.Request
	if err := worker.ReadFrame(conn, &req); err != nil {
		return
	}
	switch req.Op {
	case worker.OpPing:
		_ = worker.WriteFrame(conn, worker.Response{Kind: worker.KindOK, Version: "test"})
	case worker.OpOpenSession:
		id := fmt.Sprintf("s-%d", s.sessID.Add(1))
		_ = worker.WriteFrame(conn, worker.Response{Kind: worker.KindOK, SessionID: id})
	case worker.OpCloseSession:
		_ = worker.WriteFrame(conn, worker.Response{Kind: worker.KindOK})
	case worker.OpExec, worker.OpQuery:
		cols := []worker.ColumnInfo{{Name: "ok", Type: worker.TypeInteger}}
		_ = worker.WriteFrame(conn, worker.Response{
			Kind:    worker.KindMeta,
			Columns: cols,
		})
		_ = worker.WriteFrame(conn, worker.Response{
			Kind: worker.KindChunk,
			Rows: worker.EncodeRows(cols, [][]any{{int32(1)}}),
		})
		_ = worker.WriteFrame(conn, worker.Response{Kind: worker.KindDone})
	default:
		_ = worker.WriteFrame(conn, worker.Response{Kind: worker.KindError, Error: "unsupported op"})
	}
}

// TestOpenSharedRecoversAfterCooldownAndStaysClosedAfterClose verifies that:
//  1. When a worker crashes and restart fails into cooldown (clearing workerEP),
//     a pooled-only OpenShared caller recovers automatically once cooldown expires
//     without needing a primary-connection query to kick-start the restart.
//  2. Once Connector.Close is called, pooled-only callers fail fast with
//     errQuackUnavailable and never restart the worker.
func TestOpenSharedRecoversAfterCooldownAndStaysClosedAfterClose(t *testing.T) {
	base := ""
	if runtime.GOOS != "windows" {
		base = "/tmp"
	}
	dir, err := os.MkdirTemp(base, "idw-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	sockPath := filepath.Join(dir, "w.sock")
	srv := startFakeWorkerServer(t, sockPath)

	ctx := t.Context()
	c, primary, err := OpenConnector(ctx, Options{
		WorkerSocket: sockPath,
		DBFile:       ":memory:",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = primary.Close() })

	shared := c.OpenShared(2)
	t.Cleanup(func() { _ = shared.Close() })

	// Warm up a pooled connection on OpenShared.
	pooledConn, err := shared.Conn(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pooledConn.Close() })

	var one int
	require.NoError(t, pooledConn.QueryRowContext(ctx, "SELECT 1 AS ok;").Scan(&one))
	require.Equal(t, 1, one)

	var clockMu sync.Mutex
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c.proc.mu.Lock()
	genInitial := c.proc.quackGen
	c.proc.now = func() time.Time {
		clockMu.Lock()
		defer clockMu.Unlock()
		return now
	}
	c.proc.mu.Unlock()

	// Crash the worker so the restart attempt fails and enters cooldown with workerEP == nil.
	srv.stop()

	err = pooledConn.QueryRowContext(ctx, "SELECT 1 AS ok;").Scan(&one)
	require.ErrorIs(t, err, ErrDisabled)
	require.Nil(t, c.proc.workerEP.Load(), "failed restart should clear workerEP")

	// Still in cooldown at +9s -> pooled-only caller gets ErrDisabled.
	clockMu.Lock()
	now = now.Add(9 * time.Second)
	clockMu.Unlock()
	err = pooledConn.QueryRowContext(ctx, "SELECT 1 AS ok;").Scan(&one)
	require.ErrorIs(t, err, ErrDisabled)

	// Bring the worker back and advance past the 10s cooldown.
	require.NoError(t, srv.start())
	clockMu.Lock()
	now = now.Add(1 * time.Second)
	clockMu.Unlock()

	// Pooled-only caller (existing connection) recovers without any primary-conn traffic.
	require.NoError(t, pooledConn.QueryRowContext(ctx, "SELECT 1 AS ok;").Scan(&one))
	require.Equal(t, 1, one)

	c.proc.mu.Lock()
	require.Equal(t, genInitial+1, c.proc.quackGen)
	require.Equal(t, 0, c.proc.restartFails)
	require.True(t, c.proc.disabledUntil.IsZero())
	c.proc.mu.Unlock()

	// Also verify a brand-new OpenShared pool (no existing conn) recovers after cooldown.
	srv.stop()
	err = pooledConn.QueryRowContext(ctx, "SELECT 1 AS ok;").Scan(&one)
	require.ErrorIs(t, err, ErrDisabled)
	require.NoError(t, srv.start())
	clockMu.Lock()
	now = now.Add(10 * time.Second)
	clockMu.Unlock()

	freshShared := c.OpenShared(2)
	t.Cleanup(func() { _ = freshShared.Close() })
	require.NoError(t, freshShared.QueryRowContext(ctx, "SELECT 1 AS ok;").Scan(&one))
	require.Equal(t, 1, one)

	c.proc.mu.Lock()
	genAfterRecovery := c.proc.quackGen
	c.proc.mu.Unlock()
	require.Equal(t, genInitial+2, genAfterRecovery)

	// Explicit Close must permanently disable recovery even while the worker is reachable.
	require.NoError(t, c.Close())

	err = pooledConn.QueryRowContext(ctx, "SELECT 1 AS ok;").Scan(&one)
	require.ErrorIs(t, err, errQuackUnavailable)

	err = freshShared.QueryRowContext(ctx, "SELECT 1 AS ok;").Scan(&one)
	require.ErrorIs(t, err, errQuackUnavailable)

	c.proc.mu.Lock()
	require.Equal(t, genAfterRecovery, c.proc.quackGen, "explicitly closed process must not restart")
	require.Nil(t, c.proc.workerEP.Load())
	c.proc.mu.Unlock()
}
