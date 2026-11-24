package server

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/inngest/inngest/pkg/duckdb/worker"
	"github.com/inngest/inngest/pkg/duckdb/worker/libduckdb"
)

// Opts configures Serve.
type Opts struct {
	// ListenAddr is the address to listen on: a Unix domain socket path
	// (/path.sock, unix:///path.sock) or TCP address (:9090, tcp://0.0.0.0:9090).
	// If empty, SocketPath is used.
	ListenAddr string
	// SocketPath is a backward-compatible alias for ListenAddr (Unix socket).
	SocketPath string
	// HealthAddr, when non-empty, serves GET /healthz over HTTP (e.g. ":8080")
	// for remote/k8s deployments. Empty disables the HTTP health server.
	HealthAddr string
	// AuthToken, when non-empty, is required on every RPC request. Required when
	// ListenAddr is a non-loopback TCP address.
	AuthToken string
	// LibPath is an optional path to libduckdb. If empty, driver.EnsureLibrary
	// lazily downloads and verifies libduckdb into StateDir.
	LibPath string
	// DBPath is the DuckDB database path ("" or ":memory:" for in-memory).
	DBPath string
	// StateDir is the cache directory used when LibPath is empty.
	StateDir string
	// AllowUnsignedExtensions enables loading unsigned local extensions.
	AllowUnsignedExtensions bool
	// Config holds optional DuckDB database configuration parameters.
	Config map[string]string
}

// Serve loads libduckdb via purego (downloading it lazily if LibPath is empty),
// opens the DuckDB database, and serves the worker protocol over UDS or TCP
// until ctx is cancelled or a Shutdown RPC is received.
func Serve(ctx context.Context, opts Opts) error {
	addr := opts.ListenAddr
	if addr == "" {
		addr = opts.SocketPath
	}
	if addr == "" {
		return fmt.Errorf("duckdb worker: ListenAddr (or SocketPath) is required")
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

	cfg := make(map[string]string, len(opts.Config)+1)
	for k, v := range opts.Config {
		cfg[k] = v
	}
	if opts.AllowUnsignedExtensions {
		cfg["allow_unsigned_extensions"] = "true"
	}

	db, err := lib.Open(opts.DBPath, cfg)
	if err != nil {
		return fmt.Errorf("duckdb worker: opening database: %w", err)
	}
	defer db.Close()

	srv := New(db, lib.Version())
	srv.SetAuthToken(opts.AuthToken)
	if opts.HealthAddr != "" {
		stopHealth, err := startHealthServer(opts.HealthAddr, srv)
		if err != nil {
			return err
		}
		defer func() { _ = stopHealth() }()
	}
	return srv.ListenAndServe(ctx, addr)
}

func startHealthServer(addr string, srv *Server) (func() error, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("duckdb worker: listen health %q: %w", addr, err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		if srv.isClosed() {
			http.Error(w, "closing", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	httpSrv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		if err := httpSrv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()
	return func() error {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := httpSrv.Shutdown(shutdownCtx)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		return errors.Join(err, <-errCh)
	}, nil
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
	db         *libduckdb.DB
	version    string
	authToken  string
	instanceID string

	mu         sync.RWMutex
	sessions   map[string]*session
	conns      map[net.Conn]struct{}
	ln         net.Listener
	socketPath string
	closed     bool

	registerTFOnce sync.Once
	nextID         atomic.Uint64
	shutdownOnce   sync.Once
	shutdownCh     chan struct{}
	closeOnce    sync.Once
	closeDone    chan struct{}
	closeErr     error
	wg             sync.WaitGroup
}

// New constructs a Server backed by db.
func New(db *libduckdb.DB, version string) *Server {
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	return &Server{
		db:         db,
		version:    version,
		instanceID: hex.EncodeToString(buf[:]),
		sessions:   make(map[string]*session),
		conns:      make(map[net.Conn]struct{}),
		shutdownCh: make(chan struct{}),
		closeDone:  make(chan struct{}),
	}
}

func (s *Server) nextSessionID() string {
	return fmt.Sprintf("s-%s-%d", s.instanceID, s.nextID.Add(1))
}

// SetAuthToken configures a shared secret required on every RPC request.
// Must be called before ListenAndServe/Serve.
func (s *Server) SetAuthToken(token string) {
	s.authToken = token
}

func isLoopbackHost(host string) bool {
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (s *Server) ensureStreamTableFunction(conn *libduckdb.Conn) {
	s.registerTFOnce.Do(func() {
		_ = conn.RegisterStreamTableFunction()
	})
}

// ListenAndServe binds a Unix domain socket or TCP listener at addr and serves
// until ctx is cancelled or Shutdown is requested.
func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	network, address := worker.ParseAddress(addr)
	var (
		ln  net.Listener
		err error
	)
	if network == "unix" {
		sockDir := filepath.Dir(address)
		if err := os.MkdirAll(sockDir, 0o700); err != nil {
			return fmt.Errorf("worker: creating socket directory: %w", err)
		}
		_ = os.Remove(address)

		ln, err = net.Listen("unix", address)
		if err != nil {
			return fmt.Errorf("worker: listen unix %q: %w", address, err)
		}
		if err := os.Chmod(address, 0o600); err != nil {
			_ = ln.Close()
			_ = os.Remove(address)
			return fmt.Errorf("worker: chmod unix socket %q: %w", address, err)
		}

		s.mu.Lock()
		s.socketPath = address
		s.mu.Unlock()
	} else {
		if network == "tcp" && s.authToken == "" {
			host, _, splitErr := net.SplitHostPort(address)
			if splitErr != nil || !isLoopbackHost(host) {
				return fmt.Errorf("worker: refusing to listen on non-loopback TCP address %q without --auth-token", address)
			}
		}
		ln, err = net.Listen(network, address)
		if err != nil {
			return fmt.Errorf("worker: listen %s %q: %w", network, address, err)
		}
	}

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
	if s.authToken != "" && subtle.ConstantTimeCompare([]byte(req.AuthToken), []byte(s.authToken)) != 1 {
		_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindError, Error: "unauthorized"})
		return false
	}
	switch req.Op {
	case worker.OpPing:
		s.mu.RLock()
		active := len(s.sessions)
		s.mu.RUnlock()
		_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindOK, Version: s.version, PID: os.Getpid(), ActiveSessions: active})
		return true

	case worker.OpOpenSession:
		conn, err := s.db.Connect()
		if err != nil {
			_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindError, Error: err.Error()})
			return true
		}
		s.ensureStreamTableFunction(conn)
		id := s.nextSessionID()
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
			_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindError, ErrorCode: worker.ErrCodeStatement, Error: err.Error()})
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
		return s.streamQueryResult(c, sess, req.SQL)

	case worker.OpQueryStream:
		sess := s.getSession(req.SessionID)
		if sess == nil {
			_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindError, Error: fmt.Sprintf("unknown session %q", req.SessionID)})
			return false
		}
		sess.beginRequest(req.RequestID)
		defer sess.endRequest(req.RequestID)
		return s.handleQueryStream(c, sess, req)

	case worker.OpShutdown:
		_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindOK})
		s.triggerShutdown()
		return false

	default:
		_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindError, Error: fmt.Sprintf("unknown op %q", req.Op)})
		return true
	}
}

func (s *Server) streamQueryResult(c net.Conn, sess *session, sql string) bool {
	res, err := sess.conn.Query(sql)
	if err != nil {
		_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindError, ErrorCode: worker.ErrCodeStatement, Error: err.Error()})
		return true
	}
	defer res.Close()

	libCols := res.Columns()
	wireCols := make([]worker.ColumnInfo, len(libCols))
	for i, col := range libCols {
		wireCols[i] = worker.ColumnInfo{
			Name:     col.Name,
			Type:     worker.ColumnType(col.Type),
			TypeName: col.TypeName,
		}
	}
	if err := worker.WriteFrame(c, worker.Response{Kind: worker.KindMeta, Columns: wireCols}); err != nil {
		return false
	}

	for {
		rows, err := res.NextChunk()
		if err != nil {
			_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindError, ErrorCode: worker.ErrCodeStatement, Error: err.Error()})
			return true
		}
		if rows == nil {
			break
		}
		if err := worker.WriteFrame(c, worker.Response{
			Kind: worker.KindChunk,
			Rows: worker.EncodeRows(wireCols, rows),
		}); err != nil {
			if errors.Is(err, worker.ErrFrameTooLarge) {
				_ = worker.WriteFrame(c, worker.Response{Kind: worker.KindError, ErrorCode: worker.ErrCodeStatement, Error: err.Error()})
				return true
			}
			return false
		}
	}
	if err := worker.WriteFrame(c, worker.Response{Kind: worker.KindDone}); err != nil {
		return false
	}
	return true
}

func (s *Server) handleQueryStream(c net.Conn, sess *session, req *worker.Request) bool {
	var writeMu sync.Mutex
	var writeClosed bool
	writeResp := func(resp worker.Response, terminal bool) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		if writeClosed {
			return net.ErrClosed
		}
		err := worker.WriteFrame(c, resp)
		if terminal || err != nil {
			writeClosed = true
		}
		return err
	}

	streams := make(map[string]*libduckdb.Stream, len(req.StreamIDs))
	for _, id := range req.StreamIDs {
		st := libduckdb.RegisterStream(id)
		streamID := id
		st.SetOnConsume(func(n int64) {
			_ = writeResp(worker.Response{
				Kind:     worker.KindStreamCredit,
				StreamID: streamID,
				Bytes:    n,
			}, false)
		})
		streams[id] = st
	}
	defer func() {
		for _, id := range req.StreamIDs {
			libduckdb.UnregisterStream(id)
		}
	}()

	var sourceFailed atomic.Bool
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		remaining := len(streams)
		for remaining > 0 {
			var frame worker.Request
			if err := worker.ReadFrame(c, &frame); err != nil {
				for _, st := range streams {
					st.CloseWithError(err)
				}
				return
			}
			st := streams[frame.StreamID]
			switch frame.Op {
			case worker.OpStreamData:
				if st == nil {
					continue
				}
				chunks, err := worker.DecodeBatch(frame.Blob)
				if err != nil {
					for _, other := range streams {
						other.CloseWithError(err)
					}
					sess.conn.Interrupt()
					return
				}
				blobBytes := int64(len(frame.Blob))
				if len(chunks) == 0 {
					if blobBytes > 0 {
						_ = writeResp(worker.Response{
							Kind:     worker.KindStreamCredit,
							StreamID: frame.StreamID,
							Bytes:    blobBytes,
						}, false)
					}
					continue
				}
				perChunkCredit := blobBytes / int64(len(chunks))
				for ci, ch := range chunks {
					libCols := make([]libduckdb.StreamColumn, len(ch.Columns))
					for i, col := range ch.Columns {
						libCols[i] = libduckdb.StreamColumn{
							Kind:   libduckdb.StreamColumnKind(col.Kind),
							Nulls:  col.Nulls,
							UUIDs:  col.UUIDs,
							Int64s: col.Int64s,
							Bools:  col.Bools,
							Bytes:  col.Bytes,
							Ends:   col.Ends,
						}
					}
					credit := perChunkCredit
					if ci == len(chunks)-1 {
						credit = blobBytes - perChunkCredit*int64(len(chunks)-1)
					}
					if err := st.Push(libduckdb.StreamChunk{RowCount: ch.RowCount, Columns: libCols, CreditBytes: credit}); err != nil {
						if errors.Is(err, libduckdb.ErrStreamQueueOverflow) {
							for _, other := range streams {
								other.CloseWithError(err)
							}
							sess.conn.Interrupt()
							return
						}
						break
					}
				}
			case worker.OpStreamEOF:
				if st != nil {
					st.Close()
				}
				remaining--
			case worker.OpStreamError:
				sourceFailed.Store(true)
				srcErr := errors.New(frame.Error)
				for _, other := range streams {
					other.CloseWithError(srcErr)
				}
				sess.conn.Interrupt()
				return
			}
		}
	}()

	res, err := sess.conn.Query(req.SQL)
	if err != nil {
		code := worker.ErrCodeStatement
		if sourceFailed.Load() {
			code = worker.ErrCodeStreamSource
		}
		_ = writeResp(worker.Response{Kind: worker.KindError, ErrorCode: code, Error: err.Error()}, true)
		return false
	}
	defer res.Close()

	libCols := res.Columns()
	wireCols := make([]worker.ColumnInfo, len(libCols))
	for i, col := range libCols {
		wireCols[i] = worker.ColumnInfo{
			Name:     col.Name,
			Type:     worker.ColumnType(col.Type),
			TypeName: col.TypeName,
		}
	}
	if err := writeResp(worker.Response{Kind: worker.KindMeta, Columns: wireCols}, false); err != nil {
		return false
	}

	for {
		rows, err := res.NextChunk()
		if err != nil {
			code := worker.ErrCodeStatement
			if sourceFailed.Load() {
				code = worker.ErrCodeStreamSource
			}
			_ = writeResp(worker.Response{Kind: worker.KindError, ErrorCode: code, Error: err.Error()}, true)
			return false
		}
		if rows == nil {
			break
		}
		if err := writeResp(worker.Response{
			Kind: worker.KindChunk,
			Rows: worker.EncodeRows(wireCols, rows),
		}, false); err != nil {
			if errors.Is(err, worker.ErrFrameTooLarge) {
				writeMu.Lock()
				writeClosed = false
				writeMu.Unlock()
				_ = writeResp(worker.Response{Kind: worker.KindError, ErrorCode: worker.ErrCodeStatement, Error: err.Error()}, true)
				return true
			}
			return false
		}
	}
	_ = writeResp(worker.Response{Kind: worker.KindDone}, true)
	return false
}

func (s *Server) getSession(id string) *session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[id]
}
