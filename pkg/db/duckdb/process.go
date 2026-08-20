package duckdb

import (
	"bufio"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/inngest/inngest/pkg/logger"
)

// errProcessDisabled is returned once a process has been permanently disabled
// after a failed restart attempt. Per the plan's "one restart attempt; if
// that also fails, permanently disable" policy (see
// docs/plans/006-duckdb-poc-subprocess-dual-write.md, "Subprocess" section),
// callers (Task 8's dualwrite package) should treat this as "dual-write
// disabled for this process's lifetime" rather than retrying — a disabled
// process never attempts to respawn again.
var errProcessDisabled = errors.New("duckdb: subprocess permanently disabled after a failed restart attempt")

// process supervises one duckdb CLI subprocess: spawn, health check, one
// restart attempt on death, graceful shutdown. Every access to cmd/sess and
// every state transition (spawn/close/restart/disable) happens under mu, so
// exec (the entry point reachable from conn.go) can safely detect a dead
// subprocess, restart it, and retry — even though database/sql may drive
// health checks, queries, and Close from different call paths over the
// process's lifetime.
//
// procCtx/procCancel give the subprocess its own OS-lifetime context,
// independent of whatever per-call ctx triggered a given exec/spawn/restart.
// exec.CommandContext kills its process for the context's entire lifetime,
// not just at start — so spawning with a short-lived, per-request ctx (the
// shape Task 8's dualwrite package uses: one context per batch flush or hook
// call) would kill a freshly-restarted, perfectly healthy subprocess the
// instant that unrelated context ended. procCtx is created once, alongside
// the process, and cancelled only by Connector.Close.
type process struct {
	binaryPath string
	dbFile     string

	procCtx    context.Context
	procCancel context.CancelFunc

	mu       sync.Mutex
	cmd      *exec.Cmd
	sess     *session
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	started  bool
	disabled bool
}

// startProcess spawns the subprocess and health-checks it before returning,
// so a caller (Connector.Connect / Open) never receives a process handle
// that hasn't proven it can round-trip a query. The ctx passed in only
// bounds the health check; the subprocess's own OS lifetime is governed by
// the process-owned procCtx (see the process doc comment), not this ctx.
func startProcess(ctx context.Context, binaryPath, dbFile string) (*process, error) {
	procCtx, procCancel := context.WithCancel(context.Background())
	p := &process{
		binaryPath: binaryPath,
		dbFile:     dbFile,
		procCtx:    procCtx,
		procCancel: procCancel,
	}

	p.mu.Lock()
	err := p.spawnLocked()
	p.mu.Unlock()
	if err != nil {
		procCancel()
		return nil, err
	}

	if err := p.healthCheck(ctx); err != nil {
		_ = p.close(ctx)
		return nil, fmt.Errorf("duckdb: subprocess failed initial health check: %w", err)
	}
	return p, nil
}

// spawnLocked assumes mu is already held. It always starts the subprocess
// under p.procCtx (the process's own long-lived context), never the ctx of
// whichever call (initial start, or a later restart triggered by exec)
// happened to trigger the spawn — see the process doc comment.
func (p *process) spawnLocked() error {
	cmd := exec.CommandContext(p.procCtx, p.binaryPath, p.dbFile, "-jsonlines")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("duckdb: creating stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("duckdb: creating stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("duckdb: creating stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("duckdb: starting subprocess: %w", err)
	}

	go logStderr(p.procCtx, stderr)

	p.cmd = cmd
	p.stdin = stdin
	p.stdout = stdout
	p.stderr = stderr
	p.sess = newSession(stdin, stdout)
	p.started = true
	return nil
}

func logStderr(ctx context.Context, stderr io.Reader) {
	l := logger.StdlibLogger(ctx)
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		l.Warn("duckdb subprocess stderr", "line", scanner.Text())
	}
}

// healthCheckLocked assumes mu is already held.
func (p *process) healthCheckLocked(ctx context.Context) error {
	rows, err := p.sess.exec(ctx, "SELECT 1 AS ok;")
	if err != nil {
		return fmt.Errorf("duckdb: health check failed: %w", err)
	}
	if len(rows) != 1 {
		return fmt.Errorf("duckdb: health check returned %d rows, expected 1", len(rows))
	}
	return nil
}

func (p *process) healthCheck(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.healthCheckLocked(ctx)
}

// restartLocked assumes mu is already held. It closes the current
// subprocess, spawns a fresh one, and health-checks it before declaring the
// restart successful — a respawned-but-unresponsive process is treated the
// same as a failed restart.
func (p *process) restartLocked(ctx context.Context) error {
	_ = p.closeLocked(ctx)
	if err := p.spawnLocked(); err != nil {
		return err
	}
	return p.healthCheckLocked(ctx)
}

// restart attempts one respawn after a detected failure. Exposed for tests;
// exec (below) is the production entry point that drives this automatically
// on a detected crash.
func (p *process) restart(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.restartLocked(ctx)
}

// exec is the entry point conn.go (ExecContext/QueryContext) calls to run a
// statement against the supervised subprocess. It is the integration point
// for finding #3/#4 of the task-5 review: a session-level error from
// sess.exec is never a plain SQL error — DuckDB CLI reports SQL errors to
// stderr and still completes the eofMarker round trip on stdout (verified
// empirically), so an error here means the pipe broke or the subprocess
// died. On such an error, exec attempts exactly one restart+health-check and
// retries the statement once; if the restart itself fails, the process is
// permanently disabled so every subsequent call fails fast with
// errProcessDisabled instead of repeatedly trying to respawn a binary that
// may no longer be there. The whole operation runs under mu so a concurrent
// Close (e.g. via database/sql tearing down the pool while this call is in
// flight) can never observe or produce a half-restarted process.
func (p *process) exec(ctx context.Context, sqlText string) ([]map[string]any, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.disabled {
		return nil, errProcessDisabled
	}

	rows, err := p.sess.exec(ctx, sqlText)
	if err == nil {
		return rows, nil
	}

	if restartErr := p.restartLocked(ctx); restartErr != nil {
		p.disabled = true
		return nil, fmt.Errorf("duckdb: subprocess restart failed after exec error (%w): %w", err, restartErr)
	}

	return p.sess.exec(ctx, sqlText)
}

// closeLocked assumes mu is already held.
func (p *process) closeLocked(ctx context.Context) error {
	if !p.started {
		return nil
	}
	p.started = false

	_ = p.stdin.Close()

	done := make(chan error, 1)
	go func() { done <- p.cmd.Wait() }()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		_ = p.cmd.Process.Kill()
		<-done
	}
	return nil
}

func (p *process) close(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closeLocked(ctx)
}

// Options configures Open.
type Options struct {
	// BinaryPath is the path to the duckdb executable. Leave empty to resolve
	// "duckdb" from PATH.
	BinaryPath string
	// DBFile is the path to the .duckdb catalog file, or ":memory:".
	DBFile string
}

// Connector implements database/sql/driver.Connector over one supervised
// duckdb subprocess. It also implements io.Closer: database/sql calls
// Close() on a Connector that implements it when *sql.DB.Close() runs, which
// is how the supervised subprocess actually gets torn down (see Close
// below) — Connector.Connect alone has no shutdown hook.
type Connector struct {
	opts Options
	proc *process
}

func (c *Connector) Connect(ctx context.Context) (driver.Conn, error) {
	if c.proc == nil {
		binPath := c.opts.BinaryPath
		if binPath == "" {
			resolved, err := exec.LookPath("duckdb")
			if err != nil {
				return nil, fmt.Errorf("duckdb: binary not found on PATH: %w", err)
			}
			binPath = resolved
		}
		p, err := startProcess(ctx, binPath, c.opts.DBFile)
		if err != nil {
			return nil, err
		}
		c.proc = p
	}
	return &conn{sess: c.proc}, nil
}

func (c *Connector) Driver() driver.Driver { return &Driver{} }

// Close implements io.Closer so *sql.DB.Close() tears down the supervised
// subprocess (stdin close → bounded Wait → Kill fallback, per
// process.close), then cancels the process's own procCtx so the subprocess
// cannot outlive the connector even if the graceful teardown above somehow
// left it running. io.Closer.Close takes no context, so a fixed grace
// period is used here instead of propagating a caller context.
func (c *Connector) Close() error {
	if c.proc == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := c.proc.close(ctx)
	c.proc.procCancel()
	return err
}

// Open returns a *sql.DB backed by a single supervised duckdb subprocess,
// with SetMaxOpenConns(1) already applied to reflect the POC's single
// serialized session — see docs/plans/006-duckdb-poc-subprocess-dual-write.md.
// The returned *sql.DB's Close method terminates the subprocess (Connector
// implements io.Closer); the subprocess is health-checked before Open
// returns (startProcess).
func Open(ctx context.Context, opts Options) (*sql.DB, error) {
	c := &Connector{opts: opts}
	if _, err := c.Connect(ctx); err != nil {
		return nil, err
	}
	db := sql.OpenDB(c)
	db.SetMaxOpenConns(1)
	return db, nil
}
