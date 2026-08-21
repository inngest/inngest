package duckdb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// ErrDisabled is returned once a process has been permanently disabled after
// a failed restart attempt. Per the plan's "one restart attempt; if that also
// fails, permanently disable" policy (see
// docs/plans/006-duckdb-poc-subprocess-dual-write.md, "Subprocess" section),
// callers must treat this as "dual-write disabled for this process's
// lifetime" and stop issuing statements — a disabled process never attempts
// to respawn again, so every subsequent call fails identically.
//
// It is exported so pkg/execution/dualwrite can observe the terminal state
// instead of retrying forever: the very first error that disables the process
// wraps this too, so a caller never has to fail twice to notice.
var ErrDisabled = errors.New("duckdb: subprocess permanently disabled after a failed restart attempt")

// restartHealthTimeout bounds the health check of an exec-triggered restart.
// It is deliberately detached from the caller's ctx — see exec.
const restartHealthTimeout = 10 * time.Second

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
	// duckLake is nil unless the caller opted into DuckLake (Options.DuckLake).
	// When set, every freshly spawned subprocess is re-bootstrapped from it —
	// see bootstrapDuckLakeLocked for why that has to happen per spawn.
	duckLake *DuckLakeOptions

	procCtx    context.Context
	procCancel context.CancelFunc

	mu    sync.Mutex
	cmd   *exec.Cmd
	sess  *session
	stdin io.WriteCloser
	// out is the read end of the subprocess's merged stdout+stderr pipe —
	// see spawnLocked for why they are merged.
	out      *os.File
	started  bool
	disabled bool
}

// startProcess spawns the subprocess and health-checks it before returning,
// so a caller (Connector.Connect / Open) never receives a process handle
// that hasn't proven it can round-trip a query. The ctx passed in only
// bounds the health check; the subprocess's own OS lifetime is governed by
// the process-owned procCtx (see the process doc comment), not this ctx.
func startProcess(ctx context.Context, binaryPath, dbFile string) (*process, error) {
	return startProcessWithDuckLake(ctx, binaryPath, dbFile, nil)
}

// startProcessWithDuckLake is startProcess with the opt-in DuckLake bootstrap.
// duckLake may be nil, which is exactly the pre-DuckLake behaviour.
func startProcessWithDuckLake(ctx context.Context, binaryPath, dbFile string, duckLake *DuckLakeOptions) (*process, error) {
	procCtx, procCancel := context.WithCancel(context.Background())
	p := &process{
		binaryPath: binaryPath,
		dbFile:     dbFile,
		duckLake:   duckLake,
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

	p.mu.Lock()
	err = p.initSessionLocked(ctx)
	p.mu.Unlock()
	if err != nil {
		_ = p.close(ctx)
		// Match the spawn-failure path above: nothing else will ever cancel
		// procCtx, since no *process is returned for Connector.Close to
		// reach.
		procCancel()
		return nil, fmt.Errorf("duckdb: subprocess failed initial startup: %w", err)
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

	outR, outW, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("duckdb: creating output pipe: %w", err)
	}

	// stdout and stderr are deliberately the *same* pipe. The DuckDB CLI
	// reports SQL errors (constraint violations, type/conversion failures,
	// schema drift) only on stderr, while still completing the eofMarker
	// round trip on stdout as if nothing went wrong — verified empirically.
	// With two independent pipes there is no way to know whether a stderr
	// line belongs to the statement just executed or to the next one, since
	// the relative arrival order of two pipes is not observable, so the
	// driver had no choice but to report success for rejected statements.
	//
	// Merging them yields one totally-ordered stream: the CLI flushes stdout
	// at every statement boundary (so nothing from a previous statement is
	// still buffered), which means an error written while statement N ran
	// always lands before the marker line statement N's canary emits.
	// session.exec parses each line as a JSON result row and attributes
	// anything unparseable to the in-flight statement, logging it and
	// failing the statement if it is error output. Diagnostics therefore
	// still reach the main process's logger, just from the session rather
	// than from a dedicated stderr goroutine.
	cmd.Stdout = outW
	cmd.Stderr = outW

	if err := cmd.Start(); err != nil {
		_ = outR.Close()
		_ = outW.Close()
		return fmt.Errorf("duckdb: starting subprocess: %w", err)
	}

	// The child now holds the only remaining writer. The parent's copy must
	// be closed or the reader never observes EOF when the subprocess exits,
	// and a dead subprocess would hang exec instead of erroring.
	_ = outW.Close()

	p.cmd = cmd
	p.stdin = stdin
	p.out = outR
	p.sess = newSession(stdin, outR)
	p.started = true
	return nil
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

// initSessionLocked assumes mu is already held. It runs everything a *freshly
// spawned* subprocess needs before it can be handed to a caller: prove it can
// round-trip a query, then re-establish any DuckLake attachment.
//
// Both are per-spawn concerns, which is why they are paired here and why both
// spawn paths (startProcessWithDuckLake and restartLocked) go through this one
// function — a crash-triggered restart that only health-checked would come
// back looking perfectly healthy with no lake catalog attached at all.
//
// The bootstrap deliberately lives here rather than inside healthCheckLocked:
// healthCheck is also called repeatedly against an *already running* session
// (tests do it, and nothing stops a caller from doing it), and re-running
// ATTACH on a session that already has the catalog is a hard error from DuckDB
// ("Binder Error: Failed to attach database: database with name \"lake\"
// already exists"), which session.exec would correctly surface as a failed
// statement. Keying the bootstrap to spawn instead of to health keeps
// healthCheck a pure, repeatable liveness probe.
func (p *process) initSessionLocked(ctx context.Context) error {
	if err := p.healthCheckLocked(ctx); err != nil {
		return err
	}
	return p.bootstrapDuckLakeLocked(ctx)
}

// bootstrapDuckLakeLocked assumes mu is already held. It is a no-op unless the
// caller opted into DuckLake, so every existing caller is unaffected.
//
// A duckdb subprocess starts with a completely fresh, unattached session every
// single time it spawns: extensions it loaded are gone, and so is the ATTACHed
// lake catalog. Nothing about the DuckLake state is carried in the main
// database file, so this must run after *every* spawn — initial start and
// crash-triggered restart alike — or a restarted subprocess would pass its
// health check and then fail every lake.* statement with a Catalog Error.
//
// The statements are the verified DuckLake sequence:
//
//	INSTALL ducklake;                            -- no-op once cached
//	LOAD ducklake;
//	ATTACH IF NOT EXISTS 'ducklake:<catalog>' AS lake (DATA_PATH '<data>/');
//
// None of the three produce any output on the merged stdout+stderr stream when
// they succeed (verified against duckdb v1.5.5 over this exact transport), so
// they neither add phantom rows to session.exec's result nor emit anything that
// reportDiagnostics could misclassify as an error. Failures do print the usual
// "<Kind> Error: " diagnostics, which session.exec turns into
// errStatementFailed — surfaced here as a real error, never swallowed.
//
// IF NOT EXISTS makes the ATTACH idempotent so a re-bootstrap of a session
// that already has the catalog is harmless rather than a hard Binder Error.
func (p *process) bootstrapDuckLakeLocked(ctx context.Context) error {
	if p.duckLake == nil {
		return nil
	}
	opts := *p.duckLake

	if opts.CatalogPath == "" {
		return fmt.Errorf("duckdb: DuckLake enabled but CatalogPath is empty")
	}
	if opts.DataPath == "" {
		return fmt.Errorf("duckdb: DuckLake enabled but DataPath is empty")
	}

	// DuckLake requires the data directory to exist before ATTACH runs.
	// MkdirAll is idempotent, so re-running it on every respawn is free.
	if err := os.MkdirAll(opts.DataPath, 0o755); err != nil {
		return fmt.Errorf("duckdb: creating DuckLake data path %q: %w", opts.DataPath, err)
	}

	// DATA_PATH is interpreted as a directory only when it ends in a
	// separator; the paths themselves are quoted through the same escaping the
	// literal encoder uses, since this transport has no parameter binding.
	catalogLiteral, err := encodeLiteral("ducklake:" + opts.CatalogPath)
	if err != nil {
		return fmt.Errorf("duckdb: encoding DuckLake catalog path: %w", err)
	}
	dataLiteral, err := encodeLiteral(strings.TrimSuffix(opts.DataPath, "/") + "/")
	if err != nil {
		return fmt.Errorf("duckdb: encoding DuckLake data path: %w", err)
	}

	stmts := []string{
		"INSTALL ducklake;",
		"LOAD ducklake;",
		fmt.Sprintf("ATTACH IF NOT EXISTS %s AS %s (DATA_PATH %s);", catalogLiteral, DuckLakeAlias, dataLiteral),
	}
	for _, stmt := range stmts {
		if _, err := p.sess.exec(ctx, stmt); err != nil {
			return fmt.Errorf("duckdb: DuckLake bootstrap failed on %q: %w", stmt, err)
		}
	}
	return nil
}

// restartLocked assumes mu is already held. It closes the current
// subprocess, spawns a fresh one, and re-initializes it (health check plus any
// DuckLake bootstrap) before declaring the restart successful — a
// respawned-but-unresponsive, or respawned-but-unattached, process is treated
// the same as a failed restart.
func (p *process) restartLocked(ctx context.Context) error {
	_ = p.closeLocked(ctx)
	if err := p.spawnLocked(); err != nil {
		return err
	}
	return p.initSessionLocked(ctx)
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
// statement against the supervised subprocess. It classifies whatever
// sess.exec returns into the three cases that need different handling:
//
//   - errStatementFailed: DuckDB rejected the statement (constraint, type, or
//     schema error). The subprocess is fine; return the error as-is. Session
//     errors used to be assumed to be transport-only, which is why these were
//     invisible before session.exec learned to correlate the subprocess's
//     stderr with the statement in flight (see rows.go).
//   - errSessionDesynced: the caller's ctx was cancelled mid-statement, so
//     the session can no longer trust its own framing. Respawn to resync,
//     but do not retry under a context that is already done.
//   - anything else: the pipe broke or the subprocess died. Attempt exactly
//     one restart+health-check and retry the statement once.
//
// If the one restart attempt itself fails, the process is permanently
// disabled: every call from then on (including the one that discovered it)
// fails with ErrDisabled, so callers can stop rather than repeatedly trying
// to respawn a binary that may no longer be there. The whole operation runs
// under mu so a concurrent Close (e.g. via database/sql tearing down the pool
// while this call is in flight) can never observe or produce a half-restarted
// process.
func (p *process) exec(ctx context.Context, sqlText string) ([]map[string]any, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.disabled {
		return nil, ErrDisabled
	}

	rows, err := p.sess.exec(ctx, sqlText)
	if err == nil {
		return rows, nil
	}

	// A statement DuckDB itself rejected is not a transport failure: the
	// subprocess is healthy and an identical retry would fail identically,
	// so surface it to the caller untouched.
	if errors.Is(err, errStatementFailed) {
		return nil, err
	}

	// Everything else means the pipe broke, the subprocess died, or the
	// caller's ctx was cancelled mid-statement (leaving the session
	// protocol-desynced). All three are only recoverable by respawning.
	//
	// The restart's own health check runs under a context detached from ctx:
	// in the cancelled-mid-statement case ctx is already dead, and a health
	// check failing purely because of that would wrongly disable dual-write
	// for the rest of the process's lifetime.
	rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), restartHealthTimeout)
	defer cancel()
	if restartErr := p.restartLocked(rctx); restartErr != nil {
		p.disabled = true
		return nil, fmt.Errorf("%w (exec error: %v; restart error: %v)", ErrDisabled, err, restartErr)
	}

	if ctx.Err() != nil {
		// The restart resynced the session, but retrying under a context
		// that is already done cannot succeed.
		return nil, err
	}

	return p.sess.exec(ctx, sqlText)
}

// closeLocked assumes mu is already held.
func (p *process) closeLocked(ctx context.Context) error {
	if !p.started {
		return nil
	}
	p.started = false

	// Retire the session's reader goroutine first so it can never block
	// handing a line to an exec that will never run again.
	if p.sess != nil {
		p.sess.close()
	}

	_ = p.stdin.Close()

	done := make(chan error, 1)
	go func() { done <- p.cmd.Wait() }()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		_ = p.cmd.Process.Kill()
		<-done
	}

	// The merged output pipe is ours (os.Pipe, not cmd.StdoutPipe), so
	// cmd.Wait does not close it. Closing the read end also unblocks a
	// reader goroutine still parked in a pipe read.
	if p.out != nil {
		_ = p.out.Close()
	}
	return nil
}

func (p *process) close(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closeLocked(ctx)
}

// DuckLakeAlias is the catalog name the DuckLake bootstrap attaches under, so
// callers address DuckLake-backed tables as lake.<table>. It is fixed for this
// POC; making it configurable is future work.
const DuckLakeAlias = "lake"

// DuckLakeOptions opts a process into DuckLake. It is exploratory groundwork:
// nothing in the existing Migrate / dual-write path sets it yet.
//
// Both fields are required when this struct is used. Presence of the struct
// itself is the enable switch (Options.DuckLake == nil means "disabled, behave
// exactly as before"), so there is no separate boolean to keep in sync.
type DuckLakeOptions struct {
	// CatalogPath is the path to the DuckLake metadata catalog file, attached
	// as 'ducklake:<CatalogPath>'. It is created by DuckDB on first attach and
	// is distinct from Options.DBFile: the main database can be ":memory:"
	// while the lake still persists to disk.
	CatalogPath string
	// DataPath is the directory DuckLake writes its Parquet data files into.
	// It is created (os.MkdirAll) before ATTACH runs, because DuckLake
	// requires it to exist. A trailing separator is added if absent, since
	// DuckLake only treats DATA_PATH as a directory when it ends in one.
	DataPath string
}

// Options configures Open.
type Options struct {
	// BinaryPath is the path to the duckdb executable. Leave empty to resolve
	// "duckdb" from PATH.
	BinaryPath string
	// DBFile is the path to the .duckdb catalog file, or ":memory:".
	DBFile string
	// DuckLake, when non-nil, installs and loads the ducklake extension and
	// attaches a DuckLake catalog as DuckLakeAlias after every successful
	// health check of a freshly spawned subprocess — including the respawn
	// after a crash, since a new subprocess starts with an unattached session.
	// Leave nil (the zero value) to disable DuckLake entirely; that path is
	// byte-for-byte the previous behaviour.
	DuckLake *DuckLakeOptions
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
		p, err := startProcessWithDuckLake(ctx, binPath, c.opts.DBFile, c.opts.DuckLake)
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
