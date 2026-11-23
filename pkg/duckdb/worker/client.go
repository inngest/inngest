package worker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// SpawnConfig configures a managed child inngest-duckdb-worker subprocess.
type SpawnConfig struct {
	// WorkerBinary is the path to the inngest-duckdb-worker executable.
	WorkerBinary string
	// SocketPath is the Unix domain socket path the worker will listen on.
	SocketPath string
	// LibPath is an optional path to libduckdb; if empty, the worker lazily
	// downloads it into StateDir.
	LibPath string
	// DBPath is the DuckDB database path ("" or ":memory:" for in-memory).
	DBPath string
	// StateDir is the directory used for lazy libduckdb downloads when LibPath is empty.
	StateDir string
	// AllowUnsignedExtensions passes --allow-unsigned-extensions to the worker.
	AllowUnsignedExtensions bool
	// StartupTimeout bounds how long Spawn waits for the socket to become ready.
	// Defaults to 2m when LibPath is empty (allowing a cold libduckdb download)
	// and 15s when LibPath is provided.
	StartupTimeout time.Duration
	// AuthToken, when non-empty, is passed to the worker via environment
	// variable and sent on every RPC request.
	AuthToken string
	// Env holds extra KEY=VALUE environment variables appended to the minimal
	// child environment.
	Env []string
}

// Client communicates with a DuckDB UDS worker in either external sidecar mode
// (Dial) or managed subprocess mode (Spawn). It is 100% pure Go with zero
// purego or libduckdb dependencies.
type Client struct {
	socketPath string
	authToken  string

	// Managed subprocess fields (nil in external sidecar mode).
	cmd    *exec.Cmd
	stderr *bytes.Buffer
	exitCh chan error
	ready  atomic.Bool

	mu     sync.Mutex
	closed bool
}

// Dial connects to an already-running external worker (Unix socket path,
// unix:///path, tcp://host:port, or host:port) and verifies liveness with Ping.
func Dial(ctx context.Context, addr string) (*Client, error) {
	return DialWithToken(ctx, addr, "")
}

// DialWithToken connects to an already-running external worker using authToken
// for RPC authentication and verifies liveness with Ping.
func DialWithToken(ctx context.Context, addr, authToken string) (*Client, error) {
	if addr == "" {
		return nil, errors.New("worker: address is empty")
	}
	c := &Client{socketPath: addr, authToken: authToken}
	if _, err := c.Ping(ctx); err != nil {
		return nil, fmt.Errorf("worker: dialing %q: %w", addr, err)
	}
	return c, nil
}

// Spawn starts a managed duckdb-worker subprocess with a minimal environment,
// waits until the listener responds to Ping, and returns a Client that tears
// down the subprocess on Close.
func Spawn(ctx context.Context, cfg SpawnConfig) (*Client, error) {
	if cfg.WorkerBinary == "" {
		return nil, errors.New("worker: WorkerBinary is required")
	}
	if cfg.SocketPath == "" {
		return nil, errors.New("worker: SocketPath is required")
	}
	timeout := cfg.StartupTimeout
	if timeout <= 0 {
		if cfg.LibPath == "" {
			timeout = 2 * time.Minute
		} else {
			timeout = 15 * time.Second
		}
	}

	network, address := ParseAddress(cfg.SocketPath)
	var args []string
	if network == "unix" {
		_ = os.Remove(address)
		args = []string{"--socket", address}
	} else {
		args = []string{"--listen", cfg.SocketPath}
	}
	if cfg.LibPath != "" {
		args = append(args, "--lib", cfg.LibPath)
	}
	if cfg.DBPath != "" {
		args = append(args, "--db", cfg.DBPath)
	}
	if cfg.StateDir != "" {
		args = append(args, "--state-dir", cfg.StateDir)
	}
	if cfg.AllowUnsignedExtensions {
		args = append(args, "--allow-unsigned-extensions")
	}

	extraEnv := cfg.Env
	if cfg.AuthToken != "" {
		extraEnv = append(append([]string(nil), cfg.Env...), "INNGEST_DUCKDB_WORKER_AUTH_TOKEN="+cfg.AuthToken)
	}

	var stderr bytes.Buffer
	cmd := exec.Command(cfg.WorkerBinary, args...)
	cmd.Env = minimalChildEnv(extraEnv)
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("worker: starting %q: %w", cfg.WorkerBinary, err)
	}

	exitCh := make(chan error, 1)
	go func() {
		exitCh <- cmd.Wait()
		close(exitCh)
	}()

	c := &Client{
		socketPath: cfg.SocketPath,
		authToken:  cfg.AuthToken,
		cmd:        cmd,
		stderr:     &stderr,
		exitCh:     exitCh,
	}

	readyCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case err := <-exitCh:
			return nil, fmt.Errorf("worker: process exited before ready (err=%v, stderr=%q)", err, strings.TrimSpace(stderr.String()))
		case <-readyCtx.Done():
			_ = c.Close()
			return nil, fmt.Errorf("worker: timed out waiting for socket %q (stderr=%q): %w",
				cfg.SocketPath, strings.TrimSpace(stderr.String()), readyCtx.Err())
		case <-ticker.C:
			pingCtx, pingCancel := context.WithTimeout(readyCtx, 200*time.Millisecond)
			resp, err := c.pingInfo(pingCtx)
			pingCancel()
			if err == nil && resp.PID == cmd.Process.Pid {
				c.ready.Store(true)
				return c, nil
			}
		}
	}
}

// MinimalChildEnv builds a minimal, leak-free environment for the child worker
// process so parent secrets are never inherited.
func MinimalChildEnv(extra []string) []string {
	return minimalChildEnv(extra)
}

func minimalChildEnv(extra []string) []string {
	allow := map[string]struct{}{
		"PATH":          {},
		"HOME":          {},
		"TMPDIR":        {},
		"TEMP":          {},
		"TMP":           {},
		"USER":          {},
		"USERPROFILE":   {},
		"LOCALAPPDATA":  {},
		"SYSTEMROOT":    {},
		"SSL_CERT_FILE": {},
		"SSL_CERT_DIR":  {},
		"LANG":          {},
		"LC_ALL":        {},
	}
	env := os.Environ()
	out := make([]string, 0, len(allow)+len(extra)+1)
	out = append(out, "TZ=UTC")
	for _, kv := range env {
		key, _, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		if _, allowed := allow[strings.ToUpper(key)]; allowed {
			out = append(out, kv)
		}
	}
	out = append(out, extra...)
	return out
}

// PID returns the managed child process PID, or 0 in external sidecar mode.
func (c *Client) PID() int {
	if c.cmd == nil || c.cmd.Process == nil {
		return 0
	}
	return c.cmd.Process.Pid
}

// Alive reports whether a managed worker subprocess is still running. In
// external sidecar mode, Alive always returns true until Close is called.
func (c *Client) Alive() bool {
	if c.exitCh == nil {
		c.mu.Lock()
		defer c.mu.Unlock()
		return !c.closed
	}
	select {
	case <-c.exitCh:
		return false
	default:
		return true
	}
}

// WaitExit returns a channel that closes when the managed worker process exits.
// Returns nil in external sidecar mode.
func (c *Client) WaitExit() <-chan error {
	return c.exitCh
}

func (c *Client) dial(ctx context.Context) (net.Conn, error) {
	var d net.Dialer
	network, address := ParseAddress(c.socketPath)
	return d.DialContext(ctx, network, address)
}

func wrapResponseError(resp Response) error {
	switch resp.ErrorCode {
	case ErrCodeStatement:
		return fmt.Errorf("%w: %s", ErrStatementFailed, resp.Error)
	case ErrCodeStreamSource:
		return fmt.Errorf("%w: %s", ErrStreamSource, resp.Error)
	default:
		return errors.New(resp.Error)
	}
}

func (c *Client) roundTrip(ctx context.Context, req Request) (Response, error) {
	if req.AuthToken == "" {
		req.AuthToken = c.authToken
	}
	conn, err := c.dial(ctx)
	if err != nil {
		return Response{}, err
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	if err := WriteFrame(conn, req); err != nil {
		return Response{}, err
	}
	var resp Response
	if err := ReadFrame(conn, &resp); err != nil {
		return Response{}, err
	}
	if resp.Kind == KindError {
		return resp, wrapResponseError(resp)
	}
	return resp, nil
}

func (c *Client) pingInfo(ctx context.Context) (Response, error) {
	return c.roundTrip(ctx, Request{Op: OpPing})
}

// Ping checks worker liveness and returns the loaded DuckDB version.
func (c *Client) Ping(ctx context.Context) (string, error) {
	resp, err := c.pingInfo(ctx)
	if err != nil {
		return "", err
	}
	return resp.Version, nil
}

// ActiveSessions returns the number of currently open DuckDB sessions on the worker.
func (c *Client) ActiveSessions(ctx context.Context) (int, error) {
	resp, err := c.roundTrip(ctx, Request{Op: OpPing})
	if err != nil {
		return 0, err
	}
	return resp.ActiveSessions, nil
}

// SessionOption configures OpenSession.
type SessionOption func(*Session)

// WithAbandonedCounter attaches a shared atomic counter incremented while a
// cancelled/abandoned query on this session is still finishing on the worker.
func WithAbandonedCounter(counter *atomic.Int64) SessionOption {
	return func(s *Session) { s.abandonedCounter = counter }
}

// OpenSession creates a new independent DuckDB session (connection) on the worker.
func (c *Client) OpenSession(ctx context.Context, opts ...SessionOption) (*Session, error) {
	resp, err := c.roundTrip(ctx, Request{Op: OpOpenSession})
	if err != nil {
		return nil, err
	}
	s := &Session{
		client: c,
		id:     resp.SessionID,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

// Shutdown asks the worker to exit gracefully.
func (c *Client) Shutdown(ctx context.Context) error {
	_, err := c.roundTrip(ctx, Request{Op: OpShutdown})
	return err
}

// Close closes the client. In managed subprocess mode, it gracefully shuts down
// the child process (falling back to SIGTERM/Kill) and removes the socket.
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	if c.cmd == nil {
		return nil
	}

	if c.ready.Load() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = c.Shutdown(shutCtx)
		cancel()

		select {
		case <-c.exitCh:
		case <-time.After(2 * time.Second):
			terminateProcess(c.cmd.Process)
			select {
			case <-c.exitCh:
			case <-time.After(2 * time.Second):
				if c.cmd.Process != nil {
					_ = c.cmd.Process.Kill()
				}
				<-c.exitCh
			}
		}

		if network, address := ParseAddress(c.socketPath); network == "unix" {
			_ = os.Remove(address)
		}
		return nil
	}

	// Child never confirmed readiness (its PID never answered Ping): terminate
	// only the child process without sending Shutdown to the address or
	// removing a socket another worker might own.
	terminateProcess(c.cmd.Process)
	select {
	case <-c.exitCh:
	case <-time.After(2 * time.Second):
		if c.cmd.Process != nil {
			_ = c.cmd.Process.Kill()
		}
		<-c.exitCh
	}
	return nil
}

// Session represents a single DuckDB connection hosted inside the worker.
// Multiple Sessions on the same Client run queries concurrently.
type Session struct {
	client *Client
	id     string

	nextReqID atomic.Uint64
	abandonedCounter *atomic.Int64
	abandoned        atomic.Bool

	mu        sync.Mutex
	closed    bool
}

// ID returns the server-assigned session identifier.
func (s *Session) ID() string {
	return s.id
}

// Abandoned reports whether a statement on this session was abandoned mid-flight
// due to context cancellation.
func (s *Session) Abandoned() bool {
	return s.abandoned.Load()
}

// Interrupt sends an interrupt RPC to cancel any running query on this session.
func (s *Session) Interrupt(ctx context.Context) error {
	return s.interruptRequest(ctx, 0)
}

func (s *Session) interruptRequest(ctx context.Context, reqID uint64) error {
	_, err := s.client.roundTrip(ctx, Request{
		Op:        OpInterrupt,
		SessionID: s.id,
		RequestID: reqID,
	})
	return err
}

// Close releases the session's DuckDB connection on the worker.
func (s *Session) Close(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()

	_, err := s.client.roundTrip(ctx, Request{
		Op:        OpCloseSession,
		SessionID: s.id,
	})
	return err
}

func (s *Session) startCancelWatcher(ctx context.Context, conn net.Conn, reqID uint64) func() {
	if ctx.Done() == nil {
		return func() {}
	}
	done := make(chan struct{})
	exited := make(chan struct{})
	var triggered atomic.Bool
	go func() {
		defer close(exited)
		select {
		case <-ctx.Done():
			triggered.Store(true)
			s.abandoned.Store(true)
			if s.abandonedCounter != nil {
				s.abandonedCounter.Add(1)
			}
			// Unblock any ReadFrame waiting on the stuck server immediately.
			if conn != nil {
				_ = conn.SetDeadline(time.Now())
			}
			go func() {
				if s.abandonedCounter != nil {
					defer s.abandonedCounter.Add(-1)
				}
				// Interrupt this specific request and close the abandoned
				// session; OpCloseSession waits for the in-flight DuckDB query
				// to finish before returning, keeping abandonedCounter accurate.
				intCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				_ = s.interruptRequest(intCtx, reqID)
				cancel()
				_ = s.Close(context.Background())
			}()
		case <-done:
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			close(done)
			<-exited
			// If ctx ended right as the call returned, ensure abandoned is marked
			// before returning to the caller.
			if !triggered.Load() && ctx.Err() != nil {
				s.abandoned.Store(true)
			}
		})
	}
}

// Exec executes a SQL statement on the session, interrupting DuckDB if ctx is cancelled.
func (s *Session) Exec(ctx context.Context, sql string) error {
	conn, err := s.client.dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	reqID := s.nextReqID.Add(1)
	stopWatch := s.startCancelWatcher(ctx, conn, reqID)
	defer stopWatch()

	if err := WriteFrame(conn, Request{
		Op:        OpExec,
		AuthToken: s.client.authToken,
		SessionID: s.id,
		RequestID: reqID,
		SQL:       sql,
	}); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}

	var resp Response
	if err := ReadFrame(conn, &resp); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	if resp.Kind == KindError {
		if ctx.Err() != nil {
			return fmt.Errorf("%w: %s", ctx.Err(), resp.Error)
		}
		return wrapResponseError(resp)
	}
	return nil
}

// QueryStream streams result chunks from an active OpQuery or OpQueryStream RPC.
type QueryStream struct {
	ctx       context.Context
	conn      net.Conn
	cols      []ColumnInfo
	stopWatch func()
	cleanup   func()
	bytesRead int64
	done      bool
}

// Columns returns the result column metadata.
func (q *QueryStream) Columns() []ColumnInfo {
	out := make([]ColumnInfo, len(q.cols))
	copy(out, q.cols)
	return out
}

// BytesRead returns the cumulative wire payload bytes read so far.
func (q *QueryStream) BytesRead() int64 {
	return q.bytesRead
}

// NextChunk reads and decodes the next chunk of rows. Returns (nil, nil) at EOF.
func (q *QueryStream) NextChunk() ([][]any, error) {
	if q.done {
		return nil, nil
	}
	var resp Response
	n, err := ReadFrameWithSize(q.conn, &resp)
	q.bytesRead += n
	if err != nil {
		q.Close()
		if q.ctx.Err() != nil {
			return nil, q.ctx.Err()
		}
		return nil, err
	}
	switch resp.Kind {
	case KindChunk:
		return DecodeRows(q.cols, resp.Rows)
	case KindDone:
		q.Close()
		return nil, nil
	case KindError:
		q.Close()
		if q.ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %s", q.ctx.Err(), resp.Error)
		}
		return nil, wrapResponseError(resp)
	default:
		q.Close()
		return nil, fmt.Errorf("worker: unexpected response kind %q", resp.Kind)
	}
}

// Close closes the underlying stream connection and stops the cancellation watcher.
func (q *QueryStream) Close() error {
	if q.done {
		return nil
	}
	q.done = true
	if q.stopWatch != nil {
		q.stopWatch()
	}
	var err error
	if q.conn != nil {
		err = q.conn.Close()
	}
	if q.cleanup != nil {
		q.cleanup()
	}
	return err
}

// Query executes sql and returns a streaming QueryStream.
func (s *Session) Query(ctx context.Context, sql string) (*QueryStream, error) {
	conn, err := s.client.dial(ctx)
	if err != nil {
		return nil, err
	}

	reqID := s.nextReqID.Add(1)
	stopWatch := s.startCancelWatcher(ctx, conn, reqID)

	if err := WriteFrame(conn, Request{
		Op:        OpQuery,
		AuthToken: s.client.authToken,
		SessionID: s.id,
		RequestID: reqID,
		SQL:       sql,
	}); err != nil {
		stopWatch()
		_ = conn.Close()
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}

	var meta Response
	n, err := ReadFrameWithSize(conn, &meta)
	if err != nil {
		stopWatch()
		_ = conn.Close()
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}
	if meta.Kind == KindError {
		stopWatch()
		_ = conn.Close()
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %s", ctx.Err(), meta.Error)
		}
		return nil, wrapResponseError(meta)
	}
	if meta.Kind != KindMeta {
		stopWatch()
		_ = conn.Close()
		return nil, fmt.Errorf("worker: expected meta frame, got %q", meta.Kind)
	}

	cols := make([]ColumnInfo, len(meta.Columns))
	copy(cols, meta.Columns)

	return &QueryStream{
		ctx:       ctx,
		conn:      conn,
		cols:      cols,
		stopWatch: stopWatch,
		bytesRead: n,
	}, nil
}

// QueryWithStreams executes sql while concurrently feeding all streams over the
// full-duplex UDS connection into the worker's `scan_data_from_quack_client`
// table function.
func (s *Session) QueryWithStreams(ctx context.Context, sql string, streams []SendStream) (*QueryStream, error) {
	conn, err := s.client.dial(ctx)
	if err != nil {
		return nil, err
	}

	streamIDs := make([]string, len(streams))
	for i, st := range streams {
		streamIDs[i] = st.ID
	}

	var writeMu sync.Mutex
	writeFrame := func(req Request) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return WriteFrame(conn, req)
	}

	reqID := s.nextReqID.Add(1)
	if err := writeFrame(Request{
		Op:        OpQueryStream,
		AuthToken: s.client.authToken,
		SessionID: s.id,
		RequestID: reqID,
		SQL:       sql,
		StreamIDs: streamIDs,
	}); err != nil {
		_ = conn.Close()
		return nil, err
	}

	stopWatch := s.startCancelWatcher(ctx, conn, reqID)
	feedCtx, feedCancel := context.WithCancel(ctx)
	var cleanupCancel atomic.Bool

	var errMu sync.Mutex
	var firstSourceErr error
	var firstWriteErr error
	recordSourceErr := func(e error) {
		if cleanupCancel.Load() && errors.Is(e, context.Canceled) {
			return
		}
		errMu.Lock()
		if firstSourceErr == nil {
			firstSourceErr = e
		}
		errMu.Unlock()
	}
	recordWriteErr := func(e error) {
		if cleanupCancel.Load() {
			return
		}
		errMu.Lock()
		if firstWriteErr == nil {
			firstWriteErr = e
		}
		errMu.Unlock()
	}
	getErrors := func() (sourceErr, writeErr error) {
		errMu.Lock()
		defer errMu.Unlock()
		return firstSourceErr, firstWriteErr
	}

	var wg sync.WaitGroup
	for _, st := range streams {
		wg.Add(1)
		go func(st SendStream) {
			defer wg.Done()
			for {
				var blob []byte
				var err error
				if st.NextEncoded != nil {
					blob, _, err = st.NextEncoded(feedCtx)
				} else if st.Next != nil {
					var rows [][]any
					rows, err = st.Next(feedCtx)
					if err == nil && len(rows) > 0 {
						blob, _, err = EncodeRowsBatch(st.Columns, rows)
					}
				} else {
					err = io.EOF
				}
				if errors.Is(err, io.EOF) {
					if werr := writeFrame(Request{Op: OpStreamEOF, StreamID: st.ID}); werr != nil {
						recordWriteErr(werr)
						feedCancel()
						_ = conn.Close()
					}
					return
				}
				if err != nil {
					if cleanupCancel.Load() && errors.Is(err, context.Canceled) {
						return
					}
					recordSourceErr(err)
					feedCancel()
					if werr := writeFrame(Request{Op: OpStreamError, StreamID: st.ID, Error: err.Error()}); werr != nil {
						recordWriteErr(werr)
						_ = conn.Close()
					}
					return
				}
				if len(blob) == 0 {
					continue
				}
				if werr := writeFrame(Request{Op: OpStreamData, StreamID: st.ID, Blob: blob}); werr != nil {
					recordWriteErr(werr)
					feedCancel()
					_ = conn.Close()
					return
				}
			}
		}(st)
	}

	cleanup := func() {
		cleanupCancel.Store(true)
		feedCancel()
		wg.Wait()
	}

	var meta Response
	n, err := ReadFrameWithSize(conn, &meta)
	if err != nil {
		stopWatch()
		_ = conn.Close()
		cleanup()
		srcErr, wErr := getErrors()
		if srcErr != nil {
			return nil, fmt.Errorf("%w: %w", ErrStreamSource, srcErr)
		}
		if wErr != nil {
			return nil, wErr
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}
	if meta.Kind == KindError {
		stopWatch()
		_ = conn.Close()
		cleanup()
		srcErr, wErr := getErrors()
		if srcErr != nil {
			return nil, fmt.Errorf("%w: %w", ErrStreamSource, srcErr)
		}
		if wErr != nil {
			return nil, wErr
		}
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %s", ctx.Err(), meta.Error)
		}
		return nil, wrapResponseError(meta)
	}
	if meta.Kind != KindMeta {
		stopWatch()
		_ = conn.Close()
		cleanup()
		return nil, fmt.Errorf("worker: expected meta frame, got %q", meta.Kind)
	}

	cols := make([]ColumnInfo, len(meta.Columns))
	copy(cols, meta.Columns)

	return &QueryStream{
		ctx:       ctx,
		conn:      conn,
		cols:      cols,
		stopWatch: stopWatch,
		cleanup:   cleanup,
		bytesRead: n,
	}, nil
}

// Append bulk-inserts rows into schema.table via a streaming INSERT.
func (s *Session) Append(ctx context.Context, schema, table string, cols []StreamColumnKind, rows [][]any) error {
	if len(rows) == 0 {
		return nil
	}
	streamID, err := NewStreamID()
	if err != nil {
		return fmt.Errorf("duckdb: appender: generating stream id: %w", err)
	}
	sqlText, err := BuildInsertStreamSQL(schema, table, cols, streamID)
	if err != nil {
		return err
	}
	return s.execSingleBatchStream(ctx, sqlText, streamID, cols, rows)
}

// Merge bulk-merges rows into schema.table via a streaming MERGE INTO.
func (s *Session) Merge(ctx context.Context, schema, table string, cfg MergeConfig, columns []MergeColumn, rows [][]any) error {
	if len(rows) == 0 {
		return nil
	}
	streamID, err := NewStreamID()
	if err != nil {
		return fmt.Errorf("duckdb: merge appender: generating stream id: %w", err)
	}
	sqlText, err := BuildMergeStreamSQL(schema, table, cfg, columns, streamID)
	if err != nil {
		return err
	}
	kinds := make([]StreamColumnKind, len(columns))
	for i, c := range columns {
		kinds[i] = c.Kind
	}
	return s.execSingleBatchStream(ctx, sqlText, streamID, kinds, rows)
}

func (s *Session) execSingleBatchStream(ctx context.Context, sqlText, streamID string, cols []StreamColumnKind, rows [][]any) error {
	blob, _, err := EncodeRowsBatch(cols, rows)
	if err != nil {
		return err
	}
	sent := false
	qs, err := s.QueryWithStreams(ctx, sqlText, []SendStream{{
		ID:      streamID,
		Columns: cols,
		NextEncoded: func(context.Context) ([]byte, uint64, error) {
			if sent {
				return nil, 0, io.EOF
			}
			sent = true
			return blob, 1, nil
		},
	}})
	if err != nil {
		return err
	}
	defer qs.Close()
	for {
		ch, err := qs.NextChunk()
		if err != nil {
			return err
		}
		if ch == nil {
			return nil
		}
	}
}
