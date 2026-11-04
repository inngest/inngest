package driver

import (
	"context"
	"crypto/rand"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/jsonlines"
	"github.com/inngest/inngest/pkg/duckdb/driver/internal/quack"
	"github.com/inngest/inngest/pkg/duckdb/driver/internal/result"
	"github.com/inngest/inngest/pkg/logger"
)

// ErrDisabled is returned once a process has been permanently disabled after
// a failed restart attempt. Per the "one restart attempt; if that also
// fails, permanently disable" policy, callers must treat this as
// "dual-write disabled for this process's lifetime" and stop issuing
// statements — a disabled process never attempts to respawn again, so every
// subsequent call fails identically.
//
// It is exported so pkg/duckdb/tracing can observe the terminal state
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
// shape pkg/duckdb/tracing uses: one context per batch flush or hook
// call) would kill a freshly-restarted, perfectly healthy subprocess the
// instant that unrelated context ended. procCtx is created once, alongside
// the process, and cancelled only by Connector.Close.
type process struct {
	binaryPath string
	dbFile     string
	// iceberg is nil unless the caller opted into an Iceberg lake
	// (Options.Iceberg). When set, every freshly spawned subprocess is
	// re-bootstrapped from it — see bootstrapIcebergLocked for why that has
	// to happen per spawn.
	iceberg   *IcebergOptions
	quackAddr *string
	// stateDir, when non-empty, isolates the subprocess's DuckDB extension
	// cache and home directory under <stateDir>/duckdb — see
	// bootstrapStateDirLocked. Left empty, behavior is byte-for-byte
	// unchanged: the subprocess uses whatever the ambient environment's HOME
	// already points at.
	stateDir string
	// allowUnsigned, when true, starts the subprocess with -unsigned so it
	// will LOAD an unsigned (e.g. locally self-built) extension file — see
	// Options.AllowUnsignedExtensions.
	allowUnsigned bool
	// localExtensionPaths overrides name-based INSTALL/LOAD for specific
	// extensions with a direct local-file LOAD — see installOrLoadStmts and
	// Options.LocalExtensionPaths.
	localExtensionPaths map[string]string
	// quackServeToken, when non-empty, fixes startQuackLocked's quack_serve
	// auth token instead of generating a random one per spawn — see
	// Options.QuackServeToken.
	quackServeToken string
	// restrictExternalAccess, when true, turns off DuckDB's external access
	// at the end of every spawn's bootstrap — see
	// Options.RestrictExternalAccess and bootstrapSandboxLocked.
	restrictExternalAccess bool
	// resourceLimits, when non-nil, caps DuckDB's memory/threads/temp
	// spill at the end of every spawn's bootstrap -- see
	// Options.ResourceLimits and bootstrapResourceLimitsLocked.
	resourceLimits *ResourceLimits
	// abandoned counts statements, on any of this process's quack sessions,
	// that were abandoned mid-flight and that the server may still be
	// running (see quack.WithAbandonedCounter). maxAbandoned caps it for
	// gated pooled connections (see pooledQuackConn.gated); <= 0 is
	// DefaultMaxAbandonedStatements.
	abandoned    atomic.Int64
	maxAbandoned int

	procCtx    context.Context
	procCancel context.CancelFunc

	mu  sync.Mutex
	cmd *exec.Cmd
	// sess is the currently active sqlExecer: the jsonlines session
	// (internal/jsonlines) spawnLocked always creates first, or — once
	// startQuackLocked's bootstrap succeeds, when quackAddr is set — the
	// quack.Session it swaps in. Every exec/healthCheck call after that swap
	// goes over quack instead; the jsonlines pipe stays open (never
	// written to again) purely so the CLI doesn't exit, since closing stdin
	// would kill the quack listener along with the rest of the process.
	sess sqlExecer
	// quackEP is the live quack listener's endpoint, published by
	// startQuackLocked and cleared by closeLocked. It is atomic rather than
	// guarded by mu so pooled connections (pooledQuackConn) can check it on
	// every statement without queueing behind whatever the primary
	// connection is running under mu. A nil value means no listener is up
	// (not yet bootstrapped, mid-restart, disabled, or closed).
	quackEP atomic.Pointer[quackEndpoint]
	// quackGen counts successful quack bootstraps (initial start plus every
	// restart); it is copied into each quackEndpoint so a pooled connection
	// can tell its session belongs to a dead subprocess. Guarded by mu.
	quackGen uint64
	// redactor scrubs quack tokens and catalog URIs out of bootstrap errors
	// and subprocess diagnostics — see secretRedactor.
	redactor *secretRedactor
	stdin    io.WriteCloser
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
	return startProcessWithLake(ctx, binaryPath, dbFile, nil, nil)
}

// processOption configures optional per-spawn behavior beyond
// startProcessWithLake's required parameters — currently just state-dir
// isolation (withStateDir). Added as a variadic functional option so the
// many existing call sites (every test spawning a process directly) don't
// need updating for a knob only Connector.Connect uses.
type processOption func(*process)

// withStateDir sets stateDir on a process being constructed — see
// process.stateDir's doc comment.
func withStateDir(stateDir string) processOption {
	return func(p *process) { p.stateDir = stateDir }
}

// withAllowUnsignedExtensions sets allowUnsigned on a process being
// constructed — see process.allowUnsigned's doc comment.
func withAllowUnsignedExtensions(allow bool) processOption {
	return func(p *process) { p.allowUnsigned = allow }
}

// withLocalExtensionPaths sets localExtensionPaths on a process being
// constructed — see process.localExtensionPaths's doc comment.
func withLocalExtensionPaths(paths map[string]string) processOption {
	return func(p *process) { p.localExtensionPaths = paths }
}

// withQuackServeToken sets quackServeToken on a process being constructed —
// see process.quackServeToken's doc comment.
func withQuackServeToken(token string) processOption {
	return func(p *process) { p.quackServeToken = token }
}

// startProcessWithLake is startProcess with the opt-in Iceberg lake
// bootstrap. iceberg may be nil, which attaches no lake at all.
func startProcessWithLake(ctx context.Context, binaryPath, dbFile string, iceberg *IcebergOptions, quackAddr *string, opts ...processOption) (*process, error) {
	procCtx, procCancel := context.WithCancel(context.Background())
	p := &process{
		binaryPath: binaryPath,
		dbFile:     dbFile,
		iceberg:    iceberg,
		quackAddr:  quackAddr,
		procCtx:    procCtx,
		procCancel: procCancel,
		redactor:   &secretRedactor{},
	}
	for _, opt := range opts {
		opt(p)
	}
	p.redactor.add(p.quackServeToken)
	if iceberg != nil {
		p.redactor.add(iceberg.Token)
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
	args := []string{p.dbFile, "-jsonlines"}
	if p.allowUnsigned {
		args = append(args, "-unsigned")
	}
	cmd := exec.CommandContext(p.procCtx, p.binaryPath, args...)

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
	sess := jsonlines.NewSession(stdin, outR)
	sess.Redact = p.redactor.redact
	p.sess = sess
	p.started = true
	return nil
}

// healthCheckLocked assumes mu is already held.
func (p *process) healthCheckLocked(ctx context.Context) error {
	_, rows, err := p.sess.Exec(ctx, "SELECT 1 AS ok;")
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
// round-trip a query, then re-establish any lake attachment.
//
// Both are per-spawn concerns, which is why they are paired here and why both
// spawn paths (startProcessWithLake and restartLocked) go through this one
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

	if err := p.bootstrapTimeZoneLocked(ctx); err != nil {
		return err
	}

	if err := p.bootstrapStateDirLocked(ctx); err != nil {
		return err
	}

	if err := p.bootstrapIcebergLocked(ctx); err != nil {
		return err
	}

	if err := p.startQuackLocked(ctx); err != nil {
		return err
	}

	if err := p.bootstrapResourceLimitsLocked(ctx); err != nil {
		return err
	}

	// Last: it forbids the INSTALL/LOAD/ATTACH the phases above rely on.
	if err := p.bootstrapSandboxLocked(ctx); err != nil {
		return err
	}

	return nil
}

// startQuackLocked bootstraps a quack listener inside the freshly spawned
// subprocess over the jsonlines control channel (p.sess, still the jsonlines
// session at this point — see spawnLocked), then swaps p.sess to a
// quack.Session pointed at the listener so every subsequent exec/healthCheck
// call goes over quack instead. It is a no-op unless the caller opted into
// quack (Options.QuackAddr).
//
// Note this changes what QuackAddr means from earlier exploration: it used
// to start a quack listener purely as a side channel while jsonlines stayed
// the real transport (and ATTACH against it didn't work reliably). This
// drives quack directly over HTTP instead of via ATTACH, which sidesteps
// that issue — quack becomes the actual data-plane transport once
// bootstrapped, not just an additional listener.
func (p *process) startQuackLocked(ctx context.Context) error {
	if p.quackAddr == nil {
		return nil
	}
	l := logger.StdlibLogger(ctx)

	quackAddrLiteral, err := encodeLiteral("quack:" + *p.quackAddr)
	if err != nil {
		return fmt.Errorf("duckdb: encoding quack address: %w", err)
	}
	token := p.quackServeToken
	if token == "" {
		token, err = generateQuackToken()
		if err != nil {
			return fmt.Errorf("duckdb: generating quack auth token: %w", err)
		}
		p.redactor.add(token)
	}
	tokenLiteral, err := encodeLiteral(token)
	if err != nil {
		return fmt.Errorf("duckdb: encoding quack token: %w", err)
	}

	var bootstrapStmts []string
	// httpfs is not otherwise part of this driver's bootstrap; it is only
	// ever relevant here, as an optional prerequisite for quack_serve on a
	// self-built binary. Production never sets a "httpfs" entry (there is
	// nothing to build it from), so this is a no-op everywhere else: quack
	// on an ordinary, fully-featured DuckDB build already has a real crypto
	// engine registered without it. See LocalExtensionPaths's doc comment.
	if httpfsPath, ok := p.localExtensionPaths["httpfs"]; ok && httpfsPath != "" {
		httpfsStmts, err := installOrLoadStmts("httpfs", p.localExtensionPaths)
		if err != nil {
			return err
		}
		bootstrapStmts = append(bootstrapStmts, httpfsStmts...)
	}
	quackStmts, err := installOrLoadStmts("quack", p.localExtensionPaths)
	if err != nil {
		return err
	}
	bootstrapStmts = append(bootstrapStmts, quackStmts...)
	if err := p.bootstrapExecLocked(ctx, "quack", bootstrapStmts); err != nil {
		return err
	}

	// The CLI's own "listen_url" is authoritative: with an ephemeral
	// QuackAddr (port 0) it carries the port the kernel actually assigned.
	serveStmt := fmt.Sprintf("CALL quack_serve(%s, token = %s);", quackAddrLiteral, tokenLiteral)
	_, rows, err := p.sess.Exec(ctx, serveStmt)
	if err != nil {
		return fmt.Errorf("duckdb: quack bootstrap failed on %s: %w", stmtSummary(serveStmt), p.redactor.redactErr(err))
	}
	if len(rows) != 1 {
		return fmt.Errorf("duckdb: quack_serve returned %d rows, expected 1", len(rows))
	}
	listenURL, ok := rows[0].Get("listen_url").(string)
	if !ok || listenURL == "" {
		// Column names only: the row also carries auth_token.
		return fmt.Errorf("duckdb: quack_serve response missing listen_url (columns: %v)", rows[0].Names())
	}

	quackSess, err := quack.NewSession(ctx, listenURL, token, quack.WithAbandonedCounter(&p.abandoned))
	if err != nil {
		return fmt.Errorf("duckdb: connecting to quack listener at %s: %w", listenURL, err)
	}
	p.sess = quackSess
	p.quackGen++
	p.quackEP.Store(&quackEndpoint{listenURL: listenURL, token: token, gen: p.quackGen})

	l.Info("duckdb: quack transport active",
		"transport", "quack",
		"listen_url", listenURL,
		"connection_id", quackSess.ConnectionID(),
		"generation", p.quackGen,
	)
	return nil
}

// bootstrapExecLocked assumes mu is already held. It runs one bootstrap
// phase's statements over the current session, reporting a failure by
// statement summary (stmtSummary) and redacted engine error rather than the
// full statement, since bootstrap SQL carries tokens and catalog URIs inline.
func (p *process) bootstrapExecLocked(ctx context.Context, phase string, stmts []string) error {
	for _, stmt := range stmts {
		if _, _, err := p.sess.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("duckdb: %s bootstrap failed on %s: %w", phase, stmtSummary(stmt), p.redactor.redactErr(err))
		}
	}
	return nil
}

// openQuackConn hands out a new pooled connection (pooledQuackConn) against
// the running quack listener, beyond the primary one in sess. Unlike every
// call through p.exec, its statements bypass p.mu, so it genuinely executes
// concurrently with the primary connection and with other pooled
// connections. Used only when Options.QuackConns > 1 (see
// Connector.Connect).
//
// A pooled connection does not snapshot the listener: it re-reads quackEP
// before each statement and transparently re-handshakes when the
// subprocess has been restarted underneath it, and a transport failure on
// it drives the same one-restart-then-disable policy the primary connection
// uses (see recoverQuackLocked). That matters because database/sql may hand
// any caller — dual-write's own writes included — one of these rather than
// the primary connection.
func (p *process) openQuackConn(ctx context.Context) (sqlExecer, error) {
	return p.openPooledQuackConn(ctx, false)
}

// openGatedQuackConn is openQuackConn for a pooled connection whose
// handshakes are gated on p's abandoned statements (see
// pooledQuackConn.gated), so it fails fast w/ ErrTooManyAbandoned rather
// than opening a session while too much abandoned work is still running.
func (p *process) openGatedQuackConn(ctx context.Context) (sqlExecer, error) {
	return p.openPooledQuackConn(ctx, true)
}

func (p *process) openPooledQuackConn(ctx context.Context, gated bool) (sqlExecer, error) {
	c := &pooledQuackConn{p: p, gated: gated}
	// Handshake eagerly so a bad listener fails Connect, not the first
	// statement.
	if _, err := c.currentSession(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

// currentQuackEndpoint returns the live quack listener's endpoint. When none
// is published it waits on mu, which an in-flight restart holds, and
// re-checks — so a pooled connection racing a restart gets the new listener
// rather than a spurious error.
func (p *process) currentQuackEndpoint() (*quackEndpoint, error) {
	if ep := p.quackEP.Load(); ep != nil {
		return ep, nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.disabled {
		return nil, ErrDisabled
	}
	if ep := p.quackEP.Load(); ep != nil {
		return ep, nil
	}
	return nil, errQuackUnavailable
}

// recoverQuack is a pooled connection's counterpart to runWithRestartLocked's
// restart path: cause is a transport failure observed on a session from
// generation gen. If the subprocess has already been restarted since (by
// the primary connection or another pooled one), there is nothing to do —
// the caller just re-handshakes. Otherwise this connection performs the
// restart itself, so concurrent failures from one crash restart it once.
func (p *process) recoverQuack(ctx context.Context, gen uint64, cause error) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.disabled {
		return ErrDisabled
	}
	if p.quackGen != gen {
		logger.StdlibLogger(ctx).Debug("duckdb: pooled quack session is stale; subprocess already restarted",
			"transport", "quack", "stale_generation", gen, "generation", p.quackGen)
		return nil
	}
	return p.restartAfterFailureLocked(ctx, cause)
}

// currentQuackSession returns the primary connection's current transport if
// it's quack, for callers (quack_append.go's NewQuackAppender) that need the
// real *quack.Session rather than the sqlExecer interface — AppendRequest has
// no jsonlines equivalent, so there's nothing to abstract over. Like
// openQuackConn, this bypasses p.exec's restart-on-crash handling: a crash
// mid-append surfaces as a plain error, not a transparent retry. Returns an
// error if the primary connection is still jsonlines (Options.QuackAddr
// unset, or the quack bootstrap hasn't swapped p.sess yet). An abandoned
// session is replaced first, like before any other statement.
func (p *process) currentQuackSession(ctx context.Context) (*quack.Session, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.sess.(*quack.Session); !ok {
		return nil, fmt.Errorf("duckdb: quack appender requires a quack-transport connection (Options.QuackAddr)")
	}
	if err := p.replaceAbandonedQuackSessionLocked(ctx); err != nil {
		return nil, err
	}
	return p.sess.(*quack.Session), nil
}

// generateQuackToken returns a random hex auth token for one quack_serve
// bootstrap. Generated fresh per spawn (initial start and every
// crash-triggered restart alike) rather than a fixed shared secret, since
// the listener is bound to loopback but any local process could otherwise
// guess a hardcoded value.
func generateQuackToken() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// bootstrapTimeZoneLocked assumes mu is already held. It pins the
// subprocess's TimeZone to UTC on every spawn, since DuckDB otherwise
// defaults to the host's local zone. Every time column is TIMESTAMPTZ (see
// literal.go's timestampLayout), and the session TimeZone decides how those
// render, how a naive TIMESTAMP compares against them, and which calendar
// date_trunc/time_bucket/::DATE use. TimeZone is a GLOBAL setting, so this
// also covers every quack connection to this process.
func (p *process) bootstrapTimeZoneLocked(ctx context.Context) error {
	return p.bootstrapExecLocked(ctx, "time zone", []string{"SET GLOBAL TimeZone = 'UTC';"})
}

// bootstrapStateDirLocked assumes mu is already held. It is a no-op unless
// the caller set Options.StateDir, so every existing caller (tests,
// cmd/duckdbseed) is unaffected.
//
// A duckdb subprocess derives its extension cache and other home-directory
// state from the ambient HOME by default, which would otherwise be shared
// (and potentially polluted) across every instance running on a machine.
// Pointing both at <stateDir>/duckdb isolates that state the same way
// Options.DBFile/IcebergOptions already isolate the catalog and lake data.
//
// This must run before bootstrapIcebergLocked/startQuackLocked's INSTALL
// statements below, on every spawn (initial start and crash-triggered
// restart alike, since a fresh subprocess starts with none of this set), so
// those installs land in the isolated directory rather than DuckDB's
// default.
func (p *process) bootstrapStateDirLocked(ctx context.Context) error {
	if p.stateDir == "" {
		return nil
	}
	homeDir := filepath.Join(p.stateDir, "duckdb")
	extDir := filepath.Join(homeDir, "extensions")
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		return fmt.Errorf("duckdb: creating extension directory %q: %w", extDir, err)
	}

	homeLiteral, err := encodeLiteral(homeDir)
	if err != nil {
		return fmt.Errorf("duckdb: encoding home directory: %w", err)
	}
	extLiteral, err := encodeLiteral(extDir)
	if err != nil {
		return fmt.Errorf("duckdb: encoding extension directory: %w", err)
	}

	stmts := []string{
		fmt.Sprintf("SET home_directory=%s;", homeLiteral),
		// extension_directory (singular) is deprecated in favor of the
		// list-based extension_directories -- the singular form still works
		// but its CLI deprecation warning can land on the same output line
		// as a canary marker with no separating newline (see
		// stripANSISGR's doc comment in rows.go), so the plural form avoids
		// that risk at the source rather than relying solely on the parser
		// tolerating it.
		fmt.Sprintf("SET extension_directories=[%s];", extLiteral),
	}
	return p.bootstrapExecLocked(ctx, "state dir", stmts)
}

// bootstrapIcebergLocked assumes mu is already held. It is a no-op unless
// the caller opted into an Iceberg lake.
//
// A duckdb subprocess starts with a completely fresh, unattached session every
// single time it spawns: extensions it loaded are gone, and so is the ATTACHed
// lake catalog. Nothing about the lake attach is carried in the main database
// file, so this must run after *every* spawn — initial start and
// crash-triggered restart alike — or a restarted subprocess would pass its
// health check and then fail every inngest.* statement with a Catalog Error.
func (p *process) bootstrapIcebergLocked(ctx context.Context) error {
	if p.iceberg == nil {
		return nil
	}
	stmts, err := icebergBootstrapStmts(*p.iceberg, p.localExtensionPaths)
	if err != nil {
		return err
	}
	return p.bootstrapExecLocked(ctx, "Iceberg", stmts)
}

// icebergBootstrapStmts builds the statements bootstrapIcebergLocked runs to
// attach an Iceberg REST catalog as LakeAlias, and validates opts up front.
// It is a pure function (no I/O, no subprocess) so it can be unit tested
// w/o a live duckdb subprocess or catalog.
//
// An Iceberg catalog has no default schema, so `inngest.<table>` only
// resolves w/ ATTACH's DEFAULT_SCHEMA, which in turn fails unless that
// namespace already exists. So this attaches once w/o it, creates
// LakeNamespace if missing, and re-attaches w/ it.
//
// Verified against the pinned v2.0.0 CLI (see
// pkg/duckdb/icebergcatalog/smoke_test.go): `ENDPOINT` still works but
// warns that it's deprecated in favor of `URI`, httpfs is needed for the
// REST calls, and a TOKEN secret sends `Authorization: Bearer <token>`.
func icebergBootstrapStmts(opts IcebergOptions, localExtensionPaths map[string]string) ([]string, error) {
	if opts.URI == "" {
		return nil, fmt.Errorf("duckdb: Iceberg enabled but URI is empty")
	}
	if opts.Warehouse == "" {
		return nil, fmt.Errorf("duckdb: Iceberg enabled but Warehouse is empty")
	}

	var stmts []string
	for _, ext := range []string{"httpfs", "iceberg"} {
		s, err := installOrLoadStmts(ext, localExtensionPaths)
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, s...)
	}

	uriLit, err := encodeLiteral(strings.TrimSuffix(opts.URI, "/"))
	if err != nil {
		return nil, fmt.Errorf("duckdb: encoding Iceberg URI: %w", err)
	}
	whLit, err := encodeLiteral(opts.Warehouse)
	if err != nil {
		return nil, fmt.Errorf("duckdb: encoding Iceberg warehouse: %w", err)
	}
	nsLit, err := encodeLiteral(LakeNamespace)
	if err != nil {
		return nil, fmt.Errorf("duckdb: encoding Iceberg namespace: %w", err)
	}

	attachOpts := "TYPE iceberg, URI " + uriLit
	if opts.Token != "" {
		tokenLit, err := encodeLiteral(opts.Token)
		if err != nil {
			return nil, fmt.Errorf("duckdb: encoding Iceberg token: %w", err)
		}
		// CREATE OR REPLACE so a crash-triggered restart re-running this
		// doesn't fail on "already exists".
		stmts = append(stmts, fmt.Sprintf("CREATE OR REPLACE SECRET %s (TYPE iceberg, TOKEN %s);", icebergSecretName, tokenLit))
		attachOpts += ", SECRET " + icebergSecretName
	} else {
		attachOpts += ", AUTHORIZATION_TYPE 'none'"
	}

	return append(stmts,
		fmt.Sprintf("ATTACH IF NOT EXISTS %s AS %s (%s);", whLit, LakeAlias, attachOpts),
		fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s.%s;", LakeAlias, LakeNamespace),
		fmt.Sprintf("DETACH %s;", LakeAlias),
		fmt.Sprintf("ATTACH %s AS %s (%s, DEFAULT_SCHEMA %s);", whLit, LakeAlias, attachOpts, nsLit),
	), nil
}

// icebergSecretName is the duckdb secret holding the catalog's bearer token.
const icebergSecretName = "inngest_iceberg"

// installOrLoadStmts returns the statements that make extName available in
// the current session: ordinarily the name-based "INSTALL x; LOAD x;" pair,
// which resolves against DuckDB's extension repository. When
// localExtensionPaths has an entry for extName, that resolution is skipped
// entirely in favor of loading the given file directly (LOAD '<path>';) —
// needed for a self-built extension from an arbitrary dev commit, which the
// repository never has (and never could have) a matching published build
// for. A local build is typically unsigned, so the caller must also start
// the subprocess with Options.AllowUnsignedExtensions for the LOAD to
// succeed.
func installOrLoadStmts(extName string, localExtensionPaths map[string]string) ([]string, error) {
	if path, ok := localExtensionPaths[extName]; ok && path != "" {
		literal, err := encodeLiteral(path)
		if err != nil {
			return nil, fmt.Errorf("duckdb: encoding local extension path for %q: %w", extName, err)
		}
		return []string{fmt.Sprintf("LOAD %s;", literal)}, nil
	}
	return []string{fmt.Sprintf("INSTALL %s;", extName), fmt.Sprintf("LOAD %s;", extName)}, nil
}

// restartLocked assumes mu is already held. It closes the current
// subprocess, spawns a fresh one, and re-initializes it (health check plus any
// lake bootstrap) before declaring the restart successful — a
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

// Exec is the entry point conn.go's ExecContext calls to run a statement
// against the supervised subprocess. See runWithRestartLocked for the
// crash/restart classification shared with query below.
func (p *process) Exec(ctx context.Context, sqlText string) (cols []string, rows []result.Row, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.disabled {
		return nil, nil, ErrDisabled
	}

	runErr := p.runWithRestartLocked(ctx, func() error {
		var e error
		cols, rows, e = p.sess.Exec(ctx, sqlText)
		return e
	})
	if runErr != nil {
		return nil, nil, runErr
	}
	return cols, rows, nil
}

// Query is exec's counterpart for conn.go's QueryContext, additionally
// carrying the result's column types through — see sqlExecer's doc comment
// (conn.go) and internal/jsonlines' Session.Query / internal/quack's
// Session.Query for what each transport does to produce them.
func (p *process) Query(ctx context.Context, sqlText string) (cols []string, types []string, rows []result.Row, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.disabled {
		return nil, nil, nil, ErrDisabled
	}

	runErr := p.runWithRestartLocked(ctx, func() error {
		var e error
		cols, types, rows, e = p.sess.Query(ctx, sqlText)
		return e
	})
	if runErr != nil {
		return nil, nil, nil, runErr
	}
	return cols, types, rows, nil
}

// QueryWithStreams is Query for a statement reading quack streams (see
// WithQuackStreams), fed on p's current quack session. A stream's source
// can't be replayed, so a transport failure restarts the subprocess as
// usual but isn't retried. A failing source (quack.ErrStreamSource) fails
// only this statement, w/o a restart.
func (p *process) QueryWithStreams(ctx context.Context, sqlText string, streams []quack.SendStream) (cols []string, types []string, rows []result.Row, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.disabled {
		return nil, nil, nil, ErrDisabled
	}

	query := onceOnly(func(sess *quack.Session) error {
		var e error
		cols, types, rows, e = sess.QueryWithStreams(ctx, sqlText, streams)
		return e
	})
	runErr := p.runWithRestartLocked(ctx, func() error {
		sess, ok := p.sess.(*quack.Session)
		if !ok {
			return errQuackStreamsTransport
		}
		return query(sess)
	})
	if runErr != nil {
		return nil, nil, nil, runErr
	}
	return cols, types, rows, nil
}

// onceOnly wraps fn so a retry returns fn's first error instead of running
// it again, for statements whose inputs (quack streams) are consumed.
func onceOnly(fn func(*quack.Session) error) func(*quack.Session) error {
	var ran bool
	var first error
	return func(sess *quack.Session) error {
		if ran {
			return first
		}
		ran = true
		first = fn(sess)
		return first
	}
}

// runWithRestartLocked assumes mu is already held (see exec/query above). It
// runs fn once and classifies whatever it returns into the three cases that
// need different handling:
//
//   - result.ErrStatementFailed: DuckDB rejected the statement (constraint, type, or
//     schema error). The subprocess is fine; return the error as-is. Session
//     errors used to be assumed to be transport-only, which is why these were
//     invisible before session.exec learned to correlate the subprocess's
//     stderr with the statement in flight (see rows.go).
//   - result.ErrSessionDesynced: the caller's ctx was cancelled mid-statement, so
//     the session can no longer trust its own framing. Respawn to resync,
//     but do not retry under a context that is already done.
//   - anything else: the pipe broke or the subprocess died. Attempt exactly
//     one restart+health-check and retry fn once.
//
// If the one restart attempt itself fails, the process is permanently
// disabled: every call from then on (including the one that discovered it)
// fails with ErrDisabled, so callers can stop rather than repeatedly trying
// to respawn a binary that may no longer be there. The whole operation runs
// under mu (held by the caller) so a concurrent Close (e.g. via database/sql
// tearing down the pool while this call is in flight) can never observe or
// produce a half-restarted process.
func (p *process) runWithRestartLocked(ctx context.Context, fn func() error) error {
	err := p.replaceAbandonedQuackSessionLocked(ctx)
	if err == nil {
		err = fn()
	}
	if err == nil {
		return nil
	}

	// A statement DuckDB itself rejected is not a transport failure: the
	// subprocess is healthy and an identical retry would fail identically,
	// so surface it to the caller untouched.
	if errors.Is(err, result.ErrStatementFailed) {
		return err
	}

	// Likewise a quack stream's own source failing (a producer error, the
	// federate row cap, ...) only fails this statement: restarting would
	// tear down every other in-flight statement for nothing.
	if errors.Is(err, quack.ErrStreamSource) {
		return err
	}

	// A session left mid-statement-desynced (jsonlines only — see
	// result.ErrSessionDesynced's doc comment) can never be trusted again no
	// matter what ctx says: the subprocess's own stdout stream still has
	// the abandoned statement's output queued on it, which would
	// misattribute to whatever statement runs next unless the whole
	// subprocess respawns. Everything below this assumes err does NOT wrap
	// result.ErrSessionDesynced.
	if !errors.Is(err, result.ErrSessionDesynced) && ctx.Err() != nil {
		// The caller's own context ended (cancelled or timed out), and the
		// transport in use doesn't leave anything behind that a restart
		// would fix: quack's HTTP transport is a self-contained request per
		// statement, so an aborted request doesn't desync anything for any
		// other caller. quack.Session.Query's own watchForCancel already
		// sent the server a real CancelRequest for this statement the moment
		// ctx ended (verified empirically to actually stop the abandoned
		// query's CPU usage server-side for an interruptible statement, see
		// encodeCancelRequest's doc comment) and marked the session
		// Abandoned, so the next statement runs on a fresh quack connection
		// (replaceAbandonedQuackSessionLocked) rather than queueing behind
		// the abandoned one or catching its late interrupt.
		// Restarting the whole subprocess would actively hurt a caller
		// sharing this connection with another workload (dual-write, say):
		// every other in-flight or queued statement gets torn down for a
		// failure that was never the subprocess's fault. It would also be
		// pure waste — the ctx.Err() check further down already skips the
		// retry once the context is done, so today's restart-then-give-up
		// pays the full cost of a restart for zero benefit in exactly this
		// case.
		logger.StdlibLogger(ctx).Warn("duckdb: statement failed because its context ended; surfacing the error without restarting the subprocess", "error", err)
		return err
	}

	if restartErr := p.restartAfterFailureLocked(ctx, err); restartErr != nil {
		return restartErr
	}

	if ctx.Err() != nil {
		// The restart resynced the session, but retrying under a context
		// that is already done cannot succeed.
		return err
	}

	return fn()
}

// replaceAbandonedQuackSessionLocked assumes mu is already held. If p.sess
// is a quack.Session whose last statement was abandoned mid-flight (see
// quack.Session.Abandoned), it handshakes a replacement against the live
// listener, so the next statement neither queues behind the abandoned one
// server-side nor catches its late interrupt. A failed handshake is
// returned like any other transport failure, so runWithRestartLocked's
// usual restart policy applies to it.
func (p *process) replaceAbandonedQuackSessionLocked(ctx context.Context) error {
	old, ok := p.sess.(*quack.Session)
	if !ok || !old.Abandoned() {
		return nil
	}
	ep := p.quackEP.Load()
	if ep == nil {
		return errQuackUnavailable
	}
	sess, err := quack.NewSession(ctx, ep.listenURL, ep.token, quack.WithAbandonedCounter(&p.abandoned))
	if err != nil {
		return fmt.Errorf("duckdb: replacing abandoned quack session: %w", err)
	}
	logger.StdlibLogger(ctx).Debug("duckdb: replaced quack session after an abandoned statement",
		"transport", "quack", "abandoned_connection_id", old.ConnectionID(), "connection_id", sess.ConnectionID())
	p.sess = sess
	return nil
}

// restartAfterFailureLocked assumes mu is already held. err is the failure
// that triggered it: the pipe broke, the subprocess died, or (for jsonlines
// specifically) the caller's ctx was cancelled mid-statement, leaving that
// session protocol-desynced. All are only recoverable by respawning. It
// returns nil once a restart succeeds, or an ErrDisabled-wrapped error after
// permanently disabling the process if the one restart attempt fails.
func (p *process) restartAfterFailureLocked(ctx context.Context, err error) error {
	transport := "jsonlines"
	if _, ok := p.sess.(*quack.Session); ok {
		transport = "quack"
	}
	l := logger.StdlibLogger(ctx).With("transport", transport, "generation", p.quackGen)
	if errors.Is(err, result.ErrSessionDesynced) {
		l.Warn("duckdb: session desynced by a cancelled context; respawning subprocess to resync", "error", err)
	} else {
		l.Warn("duckdb: subprocess crashed; attempting one restart", "error", err)
	}

	// The restart's own health check runs under a context detached from ctx:
	// in the cancelled-mid-statement case ctx is already dead, and a health
	// check failing purely because of that would wrongly disable dual-write
	// for the rest of the process's lifetime.
	rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), restartHealthTimeout)
	defer cancel()
	if restartErr := p.restartLocked(rctx); restartErr != nil {
		p.disabled = true
		l.Error("duckdb: subprocess restart failed; permanently disabling dual-write for this process", "exec_error", err, "restart_error", restartErr)
		return fmt.Errorf("%w (exec error: %v; restart error: %v)", ErrDisabled, err, restartErr)
	}
	l.Info("duckdb: subprocess restarted successfully after a crash", "new_generation", p.quackGen)
	return nil
}

// closeLocked assumes mu is already held.
func (p *process) closeLocked(ctx context.Context) error {
	if !p.started {
		return nil
	}
	p.started = false
	// Unpublish the listener first so pooled connections stop handing out
	// sessions against a subprocess that is about to go away.
	p.quackEP.Store(nil)

	// Retire the jsonlines session's reader goroutine first so it can never
	// block handing a line to an exec that will never run again. A
	// quack.Session (see startQuackLocked) has no such goroutine — it's a
	// plain HTTP client — so this is a no-op once the transport has been
	// swapped.
	if closer, ok := p.sess.(interface{ Close() }); ok {
		closer.Close()
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

// EphemeralQuackAddr asks quack_serve to bind a kernel-assigned loopback
// port. The port actually bound is read back from quack_serve's listen_url,
// so there is no window between choosing a port and binding it for another
// process to claim it.
const EphemeralQuackAddr = "127.0.0.1:0"

// LakeAlias is the catalog name the lake attaches under, so callers address
// lake tables as inngest.<table>. W/o a lake (the in-memory mode) it's a
// plain schema of the same name, so the same names work either way.
const LakeAlias = "inngest"

// LakeNamespace is the Iceberg namespace the lake tables live in, attached
// as LakeAlias's default schema (so inngest.<table> is
// inngest.<LakeNamespace>.<table>).
const LakeNamespace = "main"

// IcebergOptions opts a process into an Iceberg lake: an Iceberg REST
// catalog attached as LakeAlias. Locally that's pkg/duckdb/icebergcatalog
// (an embedded catalog on 127.0.0.1), but any REST catalog w/ a static
// token works the same. Presence of the struct is the enable switch
// (Options.Iceberg == nil means no lake).
type IcebergOptions struct {
	// URI is the REST catalog base URL, e.g. `http://127.0.0.1:41234` (no
	// `/v1`).
	URI string
	// Warehouse is the ATTACH target, which duckdb sends as
	// `GET /v1/config?warehouse=<Warehouse>`. The embedded catalog ignores
	// it, a multi-warehouse catalog (Lakekeeper) uses it to pick one.
	Warehouse string
	// Token is a static bearer token. Empty attaches w/
	// AUTHORIZATION_TYPE 'none'.
	Token string
	// DataPath is the warehouse's local directory when it's on local disk,
	// which the sandbox has to allow (see sandboxStmts). Empty for a
	// remote warehouse.
	DataPath string
}

// Options configures Open.
type Options struct {
	// BinaryPath is the path to the duckdb executable. Leave empty to have
	// Connect resolve (downloading if necessary, into StateDir) the pinned
	// version instead — see EnsureBinary.
	BinaryPath string
	// StateDir is the resolved on-disk directory (e.g. ".inngest") this
	// process's on-disk state lives under. It is used two ways: (1) when
	// BinaryPath is empty, Connect downloads/caches the pinned duckdb binary
	// under <StateDir>/bin (see EnsureBinary); (2) the spawned subprocess's
	// own extension cache and home directory are isolated under
	// <StateDir>/duckdb rather than using the ambient HOME (see
	// bootstrapStateDirLocked). Leave empty to disable both — the original
	// PATH-lookup, ambient-HOME behavior.
	StateDir string
	// DBFile is the path to the .duckdb catalog file, or ":memory:".
	DBFile string
	// Iceberg, when non-nil, installs and loads the iceberg extension and
	// attaches an Iceberg REST catalog as LakeAlias after every successful
	// health check of a freshly spawned subprocess — including the respawn
	// after a crash, since a new subprocess starts with an unattached session.
	// Leave nil (the zero value) for no lake.
	Iceberg *IcebergOptions

	// QuackAddr, when non-nil, starts a quack listener on the given address
	// after every successful health check of a freshly spawned subprocess.
	// Leave nil (the zero value) to disable quack entirely. Use
	// EphemeralQuackAddr unless a caller outside this process needs a fixed
	// port: quack_serve binds with address reuse, so a fixed port already
	// held by another listener binds "successfully" instead of failing.
	QuackAddr *string

	// QuackServeToken, when non-empty, is used as quack_serve's fixed auth
	// token instead of a fresh, randomly generated one (generateQuackToken).
	// The random default is right for this driver's own data-plane use —
	// nothing outside this same process ever needs to know the token, since
	// every quack session this driver opens against its own listener
	// (openQuackConn, currentQuackSession) already has it in memory. A fixed
	// token is only needed when some other process must independently know
	// it in advance to attach as a client (IE another process ATTACHing
	// 'quack:<addr>'). Leave empty for the ordinary random-token behavior.
	QuackServeToken string

	// QuackConns, when greater than 1, allows Open's *sql.DB to hand out up
	// to that many concurrent connections instead of the default single
	// serialized session — each one beyond the first a pooledQuackConn
	// opened via process.openQuackConn, genuinely concurrent against the
	// quack HTTP listener rather than serialized through process.exec's
	// mutex, while keeping the same restart-on-crash handling. Requires
	// QuackAddr to be set; Open returns an error otherwise. Leave at the
	// zero value (0 or 1) for the original single-connection behavior.
	QuackConns int

	// AllowUnsignedExtensions starts the subprocess with -unsigned, allowing
	// it to LOAD an extension that isn't signed by DuckDB — required for
	// LocalExtensionPaths, which points at a locally self-built
	// .duckdb_extension file. Leave false in every ordinary deployment;
	// this exists for testing/benchmarking against a custom-built DuckDB
	// commit (e.g. one with an upstream fix not yet published to the
	// extension repository under any resolvable name/version).
	AllowUnsignedExtensions bool
	// LocalExtensionPaths maps an extension name (e.g. "iceberg",
	// "httpfs", "quack") to a local .duckdb_extension file to LOAD
	// directly, instead of the ordinary INSTALL/LOAD-by-name pair that
	// resolves against DuckDB's extension repository — see
	// installOrLoadStmts. Useful only alongside a BinaryPath pointing at a
	// matching self-built duckdb binary; an extension built from a
	// different commit than the running binary will fail to load with an
	// ABI/version mismatch. Leave nil for the ordinary name-based
	// resolution used everywhere in production.
	//
	// For an extension statically linked into that binary (no separate
	// file), set its value to just the extension's bare name (e.g.
	// "iceberg": "iceberg") rather than a path: LOAD accepts a quoted bare name
	// identically to the unquoted form, so this still resolves via the
	// already-linked-in extension. It only matters here because it skips
	// the plain INSTALL/LOAD pair's INSTALL statement, which always hits
	// the network regardless of static linkage and fails outright for a
	// commit never published to the extension repository.
	//
	// An "httpfs" entry is special-cased by startQuackLocked: if present, it
	// loads before quack bootstraps, purely as a prerequisite quack_serve
	// needs on a from-scratch self-build to register a real (rather than
	// read-only) crypto engine — see the "DuckDB currently has a read-only
	// crypto module loaded" error otherwise. An ordinary DuckDB build
	// already has this without any extra loading, so there is nothing to
	// set here outside of this same self-built-binary scenario.
	LocalExtensionPaths map[string]string

	// RestrictExternalAccess, when true, turns off DuckDB's external access
	// (enable_external_access=false) once every spawn's bootstrap is done,
	// leaving only the Iceberg catalog URI and warehouse DataPath reachable
	// (see sandboxStmts). Set it
	// whenever the *sql.DB also runs user-supplied SQL (Insights), as
	// defense in depth: a query that slips past the caller's own validation
	// still can't read files (read_text, read_csv, glob), environment
	// variables (getenv), or attach/copy anything. The setting is global to
	// the subprocess, so it constrains every connection, not just the
	// caller's; statements that need external access after bootstrap (a
	// later INSTALL/LOAD/ATTACH) will fail.
	RestrictExternalAccess bool

	// ResourceLimits, when non-nil, caps DuckDB's memory_limit, threads,
	// and max_temp_directory_size on every spawn (see
	// bootstrapResourceLimitsLocked). Leave nil for DuckDB's own defaults
	// (80% of RAM, every core, 90% of free disk). Like
	// RestrictExternalAccess, these are GLOBAL to the subprocess, so they
	// bound every connection's statements together, not just the caller's.
	// DefaultResourceLimits is a conservative starting point for a handle
	// that also runs Insights.
	ResourceLimits *ResourceLimits

	// MaxAbandonedStatements caps how many statements abandoned mid-flight
	// (a cancelled ctx, e.g. an Insights timeout) the server may still be
	// running before an OpenShared connection refuses to open another quack
	// session, failing w/ ErrTooManyAbandoned instead. Each abandoned
	// statement keeps running until DuckDB finishes or honors its interrupt
	// (some never do), while its caller moves on to a fresh session, so
	// repeated cancels would otherwise pile up background work. Only
	// OpenShared connections are gated: the primary connection and
	// Connector.Connect's own pool (dual-write's writes) always reconnect.
	// Leave at the zero value for DefaultMaxAbandonedStatements.
	MaxAbandonedStatements int
}

// DefaultMaxAbandonedStatements is Options.MaxAbandonedStatements' default.
const DefaultMaxAbandonedStatements = 4

// ErrTooManyAbandoned is returned by a statement on an OpenShared connection
// that needs a new quack session while Options.MaxAbandonedStatements
// abandoned statements are still running server-side. It's transient: retry
// once they finish.
var ErrTooManyAbandoned = errors.New("duckdb: too many cancelled statements are still running")

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
			resolved, err := EnsureBinary(ctx, c.opts.StateDir)
			if err != nil {
				return nil, fmt.Errorf("duckdb: resolving pinned binary: %w", err)
			}
			binPath = resolved
		}
		p, err := startProcessWithLake(ctx, binPath, c.opts.DBFile, c.opts.Iceberg, c.opts.QuackAddr,
			withStateDir(c.opts.StateDir),
			withAllowUnsignedExtensions(c.opts.AllowUnsignedExtensions),
			withLocalExtensionPaths(c.opts.LocalExtensionPaths),
			withQuackServeToken(c.opts.QuackServeToken),
			withRestrictExternalAccess(c.opts.RestrictExternalAccess),
			withResourceLimits(c.opts.ResourceLimits),
			withMaxAbandoned(c.opts.MaxAbandonedStatements),
		)
		if err != nil {
			return nil, err
		}
		c.proc = p
		return &conn{sess: c.proc}, nil
	}

	// A second and later connection when QuackConns opts into concurrency:
	// a pooled quack connection instead of the shared c.proc, so this
	// connection can genuinely run alongside others rather than queueing
	// behind process.exec's mutex. See openQuackConn's doc comment.
	if c.opts.QuackConns > 1 {
		sess, err := c.proc.openQuackConn(ctx)
		if err != nil {
			return nil, err
		}
		return &conn{sess: sess}, nil
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
// serialized session. The returned *sql.DB's Close method terminates the
// subprocess (Connector implements io.Closer); the subprocess is
// health-checked before Open returns (startProcess).
//
// If opts.QuackConns > 1, that constraint is relaxed instead:
// SetMaxOpenConns(opts.QuackConns) allows database/sql's pool to open that
// many connections, each beyond the first served by its own quack.Session
// (see Connector.Connect and process.openQuackConn) — opts.QuackAddr must
// be set in that case, or Open returns an error before spawning anything.
func Open(ctx context.Context, opts Options) (*sql.DB, error) {
	_, db, err := OpenConnector(ctx, opts)
	return db, err
}

// OpenConnector is Open, but also returns the *Connector Open normally
// keeps to itself. A caller that needs more connections than *sql.DB's own
// pool semantics would hand out — one dedicated, held-for-its-lifetime
// connection per parallel worker, say, rather than one checked out and
// returned per statement — can call Connector.Connect directly for each
// one, bypassing the pool entirely (each call, beyond the first, opens an
// independent quack.Session — see Connector.Connect and
// process.openQuackConn — so this is only useful once opts.QuackConns > 1).
func OpenConnector(ctx context.Context, opts Options) (*Connector, *sql.DB, error) {
	if opts.QuackConns > 1 && opts.QuackAddr == nil {
		return nil, nil, fmt.Errorf("duckdb: QuackConns > 1 requires QuackAddr to be set")
	}

	c := &Connector{opts: opts}
	if _, err := c.Connect(ctx); err != nil {
		return nil, nil, err
	}
	db := sql.OpenDB(c)
	if opts.QuackConns > 1 {
		db.SetMaxOpenConns(opts.QuackConns)
	} else {
		db.SetMaxOpenConns(1)
	}
	return c, db, nil
}

// OpenShared returns another *sql.DB over c's subprocess with its own pool,
// capped at maxConns open connections, for a caller that mustn't be able to
// take every connection another handle relies on (e.g. user queries next to
// dual-write's writes). When Options.QuackAddr is set, each of its
// connections is its own quack session regardless of Options.QuackConns, so
// it never shares the primary connection (and its mutex) w/ the other
// handle; without a quack listener it can only share the primary connection.
// Closing it only closes its own connections: the subprocess stays up until
// the *sql.DB OpenConnector returned is closed. Its quack sessions are gated
// on Options.MaxAbandonedStatements.
func (c *Connector) OpenShared(maxConns int) *sql.DB {
	db := sql.OpenDB(sharedConnector{c})
	db.SetMaxOpenConns(maxConns)
	return db
}

// sharedConnector is a Connector w/o its io.Closer, so *sql.DB.Close on an
// OpenShared handle doesn't tear down the subprocess.
type sharedConnector struct{ c *Connector }

func (s sharedConnector) Connect(ctx context.Context) (driver.Conn, error) {
	if s.c.opts.QuackAddr == nil {
		return s.c.Connect(ctx)
	}
	sess, err := s.c.proc.openGatedQuackConn(ctx)
	if err != nil {
		return nil, err
	}
	return &conn{sess: sess}, nil
}

func (s sharedConnector) Driver() driver.Driver { return s.c.Driver() }
