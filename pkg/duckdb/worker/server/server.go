package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/inngest/inngest/pkg/duckdb/worker"
	"github.com/inngest/inngest/pkg/duckdb/worker/libduckdb"
)

// Opts configures Serve.
type Opts struct {
	// SocketPath is the Unix domain socket path to listen on (required).
	SocketPath string
	// LibPath is an optional path to libduckdb. If empty, driver.EnsureLibrary
	// lazily downloads and verifies libduckdb into StateDir.
	LibPath string
	// DBPath is the DuckDB database path ("" or ":memory:" for in-memory).
	DBPath string
	// StateDir is the cache directory used when LibPath is empty.
	StateDir string
	// Config holds optional DuckDB database configuration parameters.
	Config map[string]string
}

// Serve loads libduckdb via purego (downloading it lazily if LibPath is empty),
// opens the DuckDB database, and serves the Unix domain socket worker protocol
// until ctx is cancelled or a Shutdown RPC is received.
func Serve(ctx context.Context, opts Opts) error {
	if opts.SocketPath == "" {
		return fmt.Errorf("duckdb worker: SocketPath is required")
	}

	resolvedLib := opts.LibPath
	if resolvedLib == "" {
		dir := opts.StateDir
		if dir == "" {
			cacheDir, err := os.UserCacheDir()
			if err != nil {
				return fmt.Errorf("duckdb worker: resolving user cache dir: %w", err)
			}
			dir = filepath.Join(cacheDir, "inngest", "duckdb")
		}
		var err error
		resolvedLib, err = driver.EnsureLibrary(ctx, dir)
		if err != nil {
			return fmt.Errorf("duckdb worker: ensuring libduckdb: %w", err)
		}
	}

	lib, err := libduckdb.Load(resolvedLib)
	if err != nil {
		return fmt.Errorf("duckdb worker: loading libduckdb: %w", err)
	}
	defer lib.Close()

	db, err := lib.Open(opts.DBPath, opts.Config)
	if err != nil {
		return fmt.Errorf("duckdb worker: opening database: %w", err)
	}
	defer db.Close()

	srv := New(db, lib.Version())
	return srv.ListenAndServe(ctx, opts.SocketPath)
}

type session struct {
	id   string
	conn *libduckdb.Conn

	mu          sync.Mutex
	active      bool
	activeReqID uint64
}

func (sess *session) beginRequest(reqID uint64) {
	sess.mu.Lock()
	sess.active = true
	sess.activeReqID = reqID
	sess.mu.Unlock()
}

func (sess *session) endRequest(reqID uint64) {
	sess.mu.Lock()
	if sess.activeReqID == reqID {
		sess.active = false
		sess.activeReqID = 0
	}
	sess.mu.Unlock()
}

func (sess *session) interrupt(reqID uint64) {
	sess.mu.Lock()
	shouldInterrupt := reqID == 0 || (sess.active && sess.activeReqID == reqID)
	sess.mu.Unlock()
	if shouldInterrupt {
		sess.conn.Interrupt()
	}
}

// Server serves DuckDB queries over a Unix domain socket, maintaining
// concurrent per-session libduckdb connections.
type Server struct {
	db      *libduckdb.DB
	version string

	mu         sync.RWMutex
	sessions   map[string]*session
	conns      map[net.Conn]struct{}
	ln         net.Listener
	socketPath string
	closed     bool

	nextID       atomic.Uint64
	shutdownOnce sync.Once
	shutdownCh   chan struct{}
	closeOnce    sync.Once
	closeDone    chan struct{}
	closeErr     error
	wg           sync.WaitGroup
}

// New constructs a Server backed by db.
func New(db *libduckdb.DB, version string) *Server {
	return &Server{
		db:         db,
		version:    version,
		sessions:   make(map[string]*session),
		conns:      make(map[net.Conn]struct{}),
		shutdownCh: make(chan struct{}),
		closeDone:  make(chan struct{}),
	}
}

// ListenAndServe binds a Unix domain socket at socketPath and serves until ctx
// is cancelled or Shutdown is requested.
func (s *Server) ListenAndServe(ctx context.Context, socketPath string) error {
	sockDir := filepath.Dir(socketPath)
	if err := os.MkdirAll(sockDir, 0o700); err != nil {
		return fmt.Errorf("worker: creating socket directory: %w", err)
	}
	_ = os.Remove(socketPath)

	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("worker: listen unix %q: %w", socketPath, err)
	}
	if err := os.Chmod(socketPath, 0o600); err != nil {
		_ = ln.Close()
		_ = os.Remove(socketPath)
		return fmt.Errorf("worker: chmod unix socket %q: %w", socketPath, err)
	}

	s.mu.Lock()
	s.socketPath = socketPath
	s.mu.Unlock()

	return s.Serve(ctx, ln)
}

// Serve accepts connections on ln until ctx is cancelled, Shutdown is called,
// or Close is called.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		_ = ln.Close()
		return errors.New("worker: server is closed")
	}
	s.ln = ln
	s.mu.Unlock()

	go func() {
		select {
		case <-ctx.Done():
			_ = s.Close()
		case <-s.shutdownCh:
			_ = s.Close()
		}
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if s.isClosed() || errors.Is(err, net.ErrClosed) {
				_ = s.Close()
				return nil
			}
			_ = s.Close()
			return fmt.Errorf("worker: accept: %w", err)
		}
		if !s.trackConn(conn) {
			_ = conn.Close()
			continue
		}
		go func(c net.Conn) {
			defer s.wg.Done()
			defer s.untrackConn(c)
			defer c.Close()
			s.handleConn(c)
		}(conn)
	}
}

func (s *Server) trackConn(c net.Conn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	s.conns[c] = struct{}{}
	s.wg.Add(1)
	return true
}

func (s *Server) untrackConn(c net.Conn) {
	s.mu.Lock()
	delete(s.conns, c)
	s.mu.Unlock()
}

// Close stops accepting new connections, waits for active handlers to finish,
// closes all remaining sessions, and removes the Unix socket file.
func (s *Server) Close() error {
	s.closeOnce.Do(func() {
		defer close(s.closeDone)

		s.mu.Lock()
		s.closed = true
		ln := s.ln
		sock := s.socketPath
		conns := make([]net.Conn, 0, len(s.conns))
		for c := range s.conns {
			conns = append(conns, c)
		}
		s.conns = make(map[net.Conn]struct{})
		activeSessions := make([]*session, 0, len(s.sessions))
		for _, sess := range s.sessions {
			activeSessions = append(activeSessions, sess)
		}
		s.mu.Unlock()

		s.triggerShutdown()

		var firstErr error
		if ln != nil {
			if err := ln.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
				firstErr = err
			}
		}
		for _, c := range conns {
			_ = c.Close()
		}
		for _, sess := range activeSessions {
			sess.conn.Interrupt()
		}

		// Wait for all connection handlers to exit before closing DuckDB
		// sessions so no handler is still executing against a closing conn.
		s.wg.Wait()

		s.mu.Lock()
		remaining := make([]*session, 0, len(s.sessions))
		for _, sess := range s.sessions {
			remaining = append(remaining, sess)
		}
		s.sessions = make(map[string]*session)
		s.mu.Unlock()

		for _, sess := range remaining {
			sess.conn.Interrupt()
			_ = sess.conn.Close()
		}
		if sock != "" {
			_ = os.Remove(sock)
		}
		s.closeErr = firstErr
	})

	<-s.closeDone
	return s.closeErr
}

func (s *Server) triggerShutdown() {
	s.shutdownOnce.Do(func() {
		close(s.shutdownCh)
	})
}

func (s *Server) isClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

func (s *Server) handleConn(c net.Conn) {
	for {
		var req worker.Request
		if err := worker.ReadFrame(c, &req); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return
			}
			_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindError, Error: err.Error()})
			return
		}
		if !s.dispatch(c, &req) {
			return
		}
	}
}

func (s *Server) dispatch(c net.Conn, req *worker.Request) bool {
	switch req.Op {
	case worker.OpPing:
		_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindOK, Version: s.version})
		return true

	case worker.OpOpenSession:
		conn, err := s.db.Connect()
		if err != nil {
			_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindError, Error: err.Error()})
			return true
		}
		id := fmt.Sprintf("s-%d", s.nextID.Add(1))
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			_ = conn.Close()
			_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindError, Error: "worker: server shutting down"})
			return false
		}
		s.sessions[id] = &session{id: id, conn: conn}
		s.mu.Unlock()
		_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindOK, SessionID: id})
		return true

	case worker.OpCloseSession:
		s.mu.Lock()
		sess, ok := s.sessions[req.SessionID]
		if ok {
			delete(s.sessions, req.SessionID)
		}
		s.mu.Unlock()
		if ok {
			sess.conn.Interrupt()
			_ = sess.conn.Close()
		}
		_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindOK})
		return true

	case worker.OpInterrupt:
		sess := s.getSession(req.SessionID)
		if sess != nil {
			sess.interrupt(req.RequestID)
		}
		_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindOK})
		return true

	case worker.OpExec:
		sess := s.getSession(req.SessionID)
		if sess == nil {
			_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindError, Error: fmt.Sprintf("unknown session %q", req.SessionID)})
			return true
		}
		sess.beginRequest(req.RequestID)
		err := sess.conn.Exec(req.SQL)
		sess.endRequest(req.RequestID)
		if err != nil {
			_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindError, Error: err.Error()})
			return true
		}
		_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindOK})
		return true

	case worker.OpQuery:
		sess := s.getSession(req.SessionID)
		if sess == nil {
			_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindError, Error: fmt.Sprintf("unknown session %q", req.SessionID)})
			return true
		}
		sess.beginRequest(req.RequestID)
		defer sess.endRequest(req.RequestID)
		res, err := sess.conn.Query(req.SQL)
		if err != nil {
			_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindError, Error: err.Error()})
			return true
		}
		defer res.Close()

		libCols := res.Columns()
		wireCols := make([]worker.ColumnInfo, len(libCols))
		for i, col := range libCols {
			wireCols[i] = worker.ColumnInfo{Name: col.Name, Type: worker.ColumnType(col.Type)}
		}
		if err := worker.WriteFrame(c, worker.Response{Kind: worker.KindMeta, Columns: wireCols}); err != nil {
			return false
		}

		for {
			rows, err := res.NextChunk()
			if err != nil {
				_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindError, Error: err.Error()})
				return true
			}
			if rows == nil {
				break
			}
			if err := worker.WriteFrame(c, worker.Response{
				Kind: worker.KindChunk,
				Rows: worker.EncodeRows(wireCols, rows),
			}); err != nil {
				return false
			}
		}
		if err := worker.WriteFrame(c, worker.Response{Kind: worker.KindDone}); err != nil {
			return false
		}
		return true

	case worker.OpShutdown:
		_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindOK})
		s.triggerShutdown()
		return false

	default:
		_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindError, Error: fmt.Sprintf("unknown op %q", req.Op)})
		return true
	}
}

func (s *Server) getSession(id string) *session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[id]
}
