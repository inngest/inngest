package worker

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

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/libduckdb"
)

type session struct {
	id   string
	conn *libduckdb.Conn
}

// Server serves DuckDB queries over a Unix domain socket, maintaining
// concurrent per-session libduckdb connections.
type Server struct {
	db      *libduckdb.DB
	version string

	mu         sync.RWMutex
	sessions   map[string]*session
	ln         net.Listener
	socketPath string
	closed     bool

	nextID       atomic.Uint64
	shutdownOnce sync.Once
	shutdownCh   chan struct{}
	wg           sync.WaitGroup
}

// NewServer constructs a Server backed by db.
func NewServer(db *libduckdb.DB, version string) *Server {
	return &Server{
		db:         db,
		version:    version,
		sessions:   make(map[string]*session),
		shutdownCh: make(chan struct{}),
	}
}

// ListenAndServe binds a Unix domain socket at socketPath and serves until ctx
// is cancelled or Shutdown is requested.
func (s *Server) ListenAndServe(ctx context.Context, socketPath string) error {
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o755); err != nil {
		return fmt.Errorf("worker: creating socket directory: %w", err)
	}
	// Remove any stale socket file left from a prior crash.
	_ = os.Remove(socketPath)

	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("worker: listen unix %q: %w", socketPath, err)
	}
	_ = os.Chmod(socketPath, 0o600)

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
				s.wg.Wait()
				return nil
			}
			return fmt.Errorf("worker: accept: %w", err)
		}
		s.wg.Add(1)
		go func(c net.Conn) {
			defer s.wg.Done()
			defer c.Close()
			s.handleConn(c)
		}(conn)
	}
}

// Close stops accepting new connections, closes all active sessions, and
// removes the Unix socket file.
func (s *Server) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	ln := s.ln
	sock := s.socketPath
	sessions := make([]*session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		sessions = append(sessions, sess)
	}
	s.sessions = make(map[string]*session)
	s.mu.Unlock()

	s.triggerShutdown()

	var firstErr error
	if ln != nil {
		if err := ln.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			firstErr = err
		}
	}
	for _, sess := range sessions {
		sess.conn.Interrupt()
		_ = sess.conn.Close()
	}
	if sock != "" {
		_ = os.Remove(sock)
	}
	return firstErr
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
		var req Request
		if err := readFrame(c, &req); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return
			}
			_ = writeFrame(c, Response{Kind: KindError, Error: err.Error()})
			return
		}
		if !s.dispatch(c, &req) {
			return
		}
	}
}

func (s *Server) dispatch(c net.Conn, req *Request) bool {
	switch req.Op {
	case OpPing:
		_ = writeFrame(c, Response{Kind: KindOK, Version: s.version})
		return true

	case OpOpenSession:
		conn, err := s.db.Connect()
		if err != nil {
			_ = writeFrame(c, Response{Kind: KindError, Error: err.Error()})
			return true
		}
		id := fmt.Sprintf("s-%d", s.nextID.Add(1))
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			_ = conn.Close()
			_ = writeFrame(c, Response{Kind: KindError, Error: "worker: server shutting down"})
			return false
		}
		s.sessions[id] = &session{id: id, conn: conn}
		s.mu.Unlock()
		_ = writeFrame(c, Response{Kind: KindOK, SessionID: id})
		return true

	case OpCloseSession:
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
		_ = writeFrame(c, Response{Kind: KindOK})
		return true

	case OpInterrupt:
		sess := s.getSession(req.SessionID)
		if sess != nil {
			sess.conn.Interrupt()
		}
		_ = writeFrame(c, Response{Kind: KindOK})
		return true

	case OpExec:
		sess := s.getSession(req.SessionID)
		if sess == nil {
			_ = writeFrame(c, Response{Kind: KindError, Error: fmt.Sprintf("unknown session %q", req.SessionID)})
			return true
		}
		if err := sess.conn.Exec(req.SQL); err != nil {
			_ = writeFrame(c, Response{Kind: KindError, Error: err.Error()})
			return true
		}
		_ = writeFrame(c, Response{Kind: KindOK})
		return true

	case OpQuery:
		sess := s.getSession(req.SessionID)
		if sess == nil {
			_ = writeFrame(c, Response{Kind: KindError, Error: fmt.Sprintf("unknown session %q", req.SessionID)})
			return true
		}
		res, err := sess.conn.Query(req.SQL)
		if err != nil {
			_ = writeFrame(c, Response{Kind: KindError, Error: err.Error()})
			return true
		}
		defer res.Close()

		cols := res.Columns()
		wireCols := make([]ColumnWire, len(cols))
		for i, col := range cols {
			wireCols[i] = ColumnWire{Name: col.Name, Type: col.Type}
		}
		if err := writeFrame(c, Response{Kind: KindMeta, Columns: wireCols}); err != nil {
			return false
		}

		for {
			rows, err := res.NextChunk()
			if err != nil {
				_ = writeFrame(c, Response{Kind: KindError, Error: err.Error()})
				return true
			}
			if rows == nil {
				break
			}
			if err := writeFrame(c, Response{
				Kind: KindChunk,
				Rows: encodeRows(cols, rows),
			}); err != nil {
				return false
			}
		}
		if err := writeFrame(c, Response{Kind: KindDone}); err != nil {
			return false
		}
		return true

	case OpShutdown:
		_ = writeFrame(c, Response{Kind: KindOK})
		s.triggerShutdown()
		return false

	default:
		_ = writeFrame(c, Response{Kind: KindError, Error: fmt.Sprintf("unknown op %q", req.Op)})
		return true
	}
}

func (s *Server) getSession(id string) *session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[id]
}
