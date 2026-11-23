package worker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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
	// StartupTimeout bounds how long Spawn waits for the socket to become ready.
	// Defaults to 15s.
	StartupTimeout time.Duration
	// Env holds extra KEY=VALUE environment variables appended to the minimal
	// child environment.
	Env []string
}

// Client communicates with a DuckDB UDS worker in either external sidecar mode
// (Dial) or managed subprocess mode (Spawn). It is 100% pure Go with zero
// purego or libduckdb dependencies.
type Client struct {
	socketPath string

	// Managed subprocess fields (nil in external sidecar mode).
	cmd    *exec.Cmd
	stderr *bytes.Buffer
	exitCh chan error

	mu     sync.Mutex
	closed bool
}

// Dial connects to an already-running external sidecar worker at socketPath
// and verifies liveness with a Ping.
func Dial(ctx context.Context, socketPath string) (*Client, error) {
	if socketPath == "" {
		return nil, errors.New("worker: socket path is empty")
	}
	c := &Client{socketPath: socketPath}
	if _, err := c.Ping(ctx); err != nil {
		return nil, fmt.Errorf("worker: dialing %q: %w", socketPath, err)
	}
	return c, nil
}

// Spawn starts a managed duckdb-worker subprocess with a minimal environment,
// waits until the Unix domain socket responds to Ping, and returns a Client
// that tears down the subprocess on Close.
func Spawn(ctx context.Context, cfg SpawnConfig) (*Client, error) {
	if cfg.WorkerBinary == "" {
		return nil, errors.New("worker: WorkerBinary is required")
	}
	if cfg.SocketPath == "" {
		return nil, errors.New("worker: SocketPath is required")
	}
	timeout := cfg.StartupTimeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	_ = os.Remove(cfg.SocketPath)

	args := []string{"--socket", cfg.SocketPath}
	if cfg.LibPath != "" {
		args = append(args, "--lib", cfg.LibPath)
	}
	if cfg.DBPath != "" {
		args = append(args, "--db", cfg.DBPath)
	}
	if cfg.StateDir != "" {
		args = append(args, "--state-dir", cfg.StateDir)
	}

	var stderr bytes.Buffer
	cmd := exec.Command(cfg.WorkerBinary, args...)
	cmd.Env = minimalChildEnv(cfg.Env)
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
			_, err := c.Ping(pingCtx)
			pingCancel()
			if err == nil {
				return c, nil
			}
		}
	}
}

// minimalChildEnv builds a minimal, leak-free environment for the child worker
// process so parent secrets are never inherited.
func minimalChildEnv(extra []string) []string {
	allow := []string{"PATH", "HOME", "TMPDIR", "TEMP", "TMP", "USER", "USERPROFILE", "SYSTEMROOT", "SSL_CERT_FILE", "SSL_CERT_DIR"}
	out := make([]string, 0, len(allow)+len(extra))
	for _, key := range allow {
		if val, ok := os.LookupEnv(key); ok {
			out = append(out, key+"="+val)
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
	return d.DialContext(ctx, "unix", c.socketPath)
}

func (c *Client) roundTrip(ctx context.Context, req Request) (Response, error) {
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
		return resp, errors.New(resp.Error)
	}
	return resp, nil
}

// Ping checks worker liveness and returns the loaded DuckDB version.
func (c *Client) Ping(ctx context.Context) (string, error) {
	resp, err := c.roundTrip(ctx, Request{Op: OpPing})
	if err != nil {
		return "", err
	}
	return resp.Version, nil
}

// OpenSession creates a new independent DuckDB session (connection) on the worker.
func (c *Client) OpenSession(ctx context.Context) (*Session, error) {
	resp, err := c.roundTrip(ctx, Request{Op: OpOpenSession})
	if err != nil {
		return nil, err
	}
	return &Session{
		client: c,
		id:     resp.SessionID,
	}, nil
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

	_ = os.Remove(c.socketPath)
	return nil
}

// Session represents a single DuckDB connection hosted inside the worker.
// Multiple Sessions on the same Client run queries concurrently.
type Session struct {
	client *Client
	id     string

	nextReqID atomic.Uint64
	mu        sync.Mutex
	closed    bool
}

// ID returns the server-assigned session identifier.
func (s *Session) ID() string {
	return s.id
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
	go func() {
		defer close(exited)
		select {
		case <-ctx.Done():
			if conn != nil {
				_ = conn.SetDeadline(time.Now())
			}
			intCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = s.interruptRequest(intCtx, reqID)
			cancel()
		case <-done:
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			close(done)
			<-exited
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
		return errors.New(resp.Error)
	}
	return nil
}

// QueryStream streams result chunks from an active OpQuery RPC.
type QueryStream struct {
	ctx       context.Context
	conn      net.Conn
	cols      []ColumnInfo
	stopWatch func()
	done      bool
}

// Columns returns the result column metadata.
func (q *QueryStream) Columns() []ColumnInfo {
	out := make([]ColumnInfo, len(q.cols))
	copy(out, q.cols)
	return out
}

// NextChunk reads and decodes the next chunk of rows. Returns (nil, nil) at EOF.
func (q *QueryStream) NextChunk() ([][]any, error) {
	if q.done {
		return nil, nil
	}
	var resp Response
	if err := ReadFrame(q.conn, &resp); err != nil {
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
		return nil, errors.New(resp.Error)
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
	if q.conn != nil {
		return q.conn.Close()
	}
	return nil
}

// Query executes sql and returns a streaming QueryStream. If ctx is cancelled
// while the query is executing, the client sends an out-of-band Interrupt RPC
// which invokes duckdb_interrupt on the session's connection.
func (s *Session) Query(ctx context.Context, sql string) (*QueryStream, error) {
	conn, err := s.client.dial(ctx)
	if err != nil {
		return nil, err
	}

	reqID := s.nextReqID.Add(1)
	stopWatch := s.startCancelWatcher(ctx, conn, reqID)

	if err := WriteFrame(conn, Request{
		Op:        OpQuery,
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
	if err := ReadFrame(conn, &meta); err != nil {
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
		return nil, errors.New(meta.Error)
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
	}, nil
}
