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
	"time"

	"github.com/inngest/inngest/pkg/logger"
)

// ErrDisabled is returned once a process has been permanently disabled after
// a failed restart attempt. Per the "one restart attempt; if that also
// fails, permanently disable" policy, callers must treat this as
// "dual-write disabled for this process's lifetime" and stop issuing
// statements — a disabled process never attempts to respawn again, so every
// subsequent call fails identically.
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
	duckLake  *DuckLakeOptions
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

	procCtx    context.Context
	procCancel context.CancelFunc

	mu  sync.Mutex
	cmd *exec.Cmd
	// sess is the currently active sqlExecer: the jsonlines session
	// (rows.go) spawnLocked always creates first, or — once
	// startQuackLocked's bootstrap succeeds, when quackAddr is set — the
	// quackSession it swaps in. Every exec/healthCheck call after that swap
	// goes over quack instead; the jsonlines pipe stays open (never
	// written to again) purely so the CLI doesn't exit, since closing stdin
	// would kill the quack listener along with the rest of the process.
	sess sqlExecer
	// quackListenURL/quackToken are set by startQuackLocked once the quack
	// listener is up, and read by openQuackConn to hand out additional,
	// independent quackSession connections beyond the primary one in sess —
	// see Options.QuackConns.
	quackListenURL string
	quackToken     string
	stdin          io.WriteCloser
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
	return startProcessWithDuckLake(ctx, binaryPath, dbFile, nil, nil)
}

// processOption configures optional per-spawn behavior beyond
// startProcessWithDuckLake's required parameters — currently just state-dir
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

// startProcessWithDuckLake is startProcess with the opt-in DuckLake bootstrap.
// duckLake may be nil, which is exactly the pre-DuckLake behaviour.
func startProcessWithDuckLake(ctx context.Context, binaryPath, dbFile string, duckLake *DuckLakeOptions, quackAddr *string, opts ...processOption) (*process, error) {
	procCtx, procCancel := context.WithCancel(context.Background())
	p := &process{
		binaryPath: binaryPath,
		dbFile:     dbFile,
		duckLake:   duckLake,
		quackAddr:  quackAddr,
		procCtx:    procCtx,
		procCancel: procCancel,
	}
	for _, opt := range opts {
		opt(p)
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
	p.sess = newSession(stdin, outR)
	p.started = true
	return nil
}

// healthCheckLocked assumes mu is already held.
func (p *process) healthCheckLocked(ctx context.Context) error {
	_, rows, err := p.sess.exec(ctx, "SELECT 1 AS ok;")
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

	if err := p.bootstrapStateDirLocked(ctx); err != nil {
		return err
	}

	if err := p.bootstrapDuckLakeLocked(ctx); err != nil {
		return err
	}

	if err := p.startQuackLocked(ctx); err != nil {
		return err
	}

	return nil
}

// startQuackLocked bootstraps a quack listener inside the freshly spawned
// subprocess over the jsonlines control channel (p.sess, still the jsonlines
// session at this point — see spawnLocked), then swaps p.sess to a
// quackSession pointed at the listener so every subsequent exec/healthCheck
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
	for _, stmt := range bootstrapStmts {
		if _, _, err := p.sess.exec(ctx, stmt); err != nil {
			return fmt.Errorf("duckdb: quack bootstrap failed on %q: %w", stmt, err)
		}
	}

	serveStmt := fmt.Sprintf("CALL quack_serve(%s, token = %s);", quackAddrLiteral, tokenLiteral)
	_, rows, err := p.sess.exec(ctx, serveStmt)
	if err != nil {
		return fmt.Errorf("duckdb: quack bootstrap failed on %q: %w", serveStmt, err)
	}
	if len(rows) != 1 {
		return fmt.Errorf("duckdb: quack_serve returned %d rows, expected 1", len(rows))
	}
	listenURL, ok := rows[0]["listen_url"].(string)
	if !ok || listenURL == "" {
		return fmt.Errorf("duckdb: quack_serve response missing listen_url (row: %v)", rows[0])
	}

	quackSess, err := newQuackSession(ctx, listenURL, token)
	if err != nil {
		return fmt.Errorf("duckdb: connecting to quack listener at %s: %w", listenURL, err)
	}
	p.sess = quackSess
	p.quackListenURL = listenURL
	p.quackToken = token

	l.Info("duckdb: quack transport active", "listen_url", listenURL)
	return nil
}

// openQuackConn hands out a new, independent quackSession against the same
// already-running quack listener p.sess (the primary connection) uses —
// unlike every call through p.exec, this bypasses p.mu entirely, so it can
// genuinely execute concurrently with the primary connection and with other
// openQuackConn-returned sessions. Used only when Options.QuackConns > 1
// (see Connector.Connect); the primary connection keeps its usual
// restart-on-crash handling via p.exec, but a session returned here does
// not — a crash invalidates it outright, which database/sql surfaces as a
// query error on that connection rather than a transparent retry, and
// database/sql may hand any caller (including dual-write's own writes,
// once QuackConns > 1) one of these instead of the restart-managed primary
// connection — there's no way to pin a specific caller to connection #1.
// Accepted deliberately: a genuine subprocess crash (as opposed to a
// caller's ctx merely ending, which runWithRestartLocked no longer treats
// as crash-worthy at all — see its own doc comment) is rare, and losing
// one write to it without a retry is a smaller cost than serializing every
// write behind however long an unrelated ad hoc Insights query takes.
func (p *process) openQuackConn(ctx context.Context) (sqlExecer, error) {
	p.mu.Lock()
	url, token := p.quackListenURL, p.quackToken
	p.mu.Unlock()

	if url == "" {
		return nil, fmt.Errorf("duckdb: quack listener not bootstrapped; QuackAddr must be set to use QuackConns")
	}
	return newQuackSession(ctx, url, token)
}

// currentQuackSession returns the primary connection's current transport if
// it's quack, for callers (quack_append.go's NewQuackAppender) that need the
// real *quackSession rather than the sqlExecer interface — AppendRequest has
// no jsonlines equivalent, so there's nothing to abstract over. Like
// openQuackConn, this bypasses p.exec's restart-on-crash handling: a crash
// mid-append surfaces as a plain error, not a transparent retry. Returns an
// error if the primary connection is still jsonlines (Options.QuackAddr
// unset, or the quack bootstrap hasn't swapped p.sess yet).
func (p *process) currentQuackSession() (*quackSession, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	sess, ok := p.sess.(*quackSession)
	if !ok {
		return nil, fmt.Errorf("duckdb: quack appender requires a quack-transport connection (Options.QuackAddr)")
	}
	return sess, nil
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

// bootstrapStateDirLocked assumes mu is already held. It is a no-op unless
// the caller set Options.StateDir, so every existing caller (tests,
// cmd/duckdbseed) is unaffected.
//
// A duckdb subprocess derives its extension cache and other home-directory
// state from the ambient HOME by default, which would otherwise be shared
// (and potentially polluted) across every instance running on a machine.
// Pointing both at <stateDir>/duckdb isolates that state the same way
// Options.DBFile/DuckLakeOptions already isolate the catalog and lake data.
//
// This must run before bootstrapDuckLakeLocked/startQuackLocked's INSTALL
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
	for _, stmt := range stmts {
		if _, _, err := p.sess.exec(ctx, stmt); err != nil {
			return fmt.Errorf("duckdb: state dir bootstrap failed on %q: %w", stmt, err)
		}
	}
	return nil
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
//	ATTACH IF NOT EXISTS 'ducklake:<catalog>' AS lake (DATA_PATH '<data>/', DATA_INLINING_ROW_LIMIT 1000);
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

	stmts, err := duckLakeBootstrapStmts(opts, p.localExtensionPaths)
	if err != nil {
		return err
	}

	if duckLakePostgresInliningOverridden(opts) {
		logger.StdlibLogger(ctx).Warn(
			"duckdb: DuckLake DataInliningRowLimit is ignored for a postgres catalog; inlining is forced off (see https://github.com/duckdb/ducklake/issues/1175)",
			"configured_row_limit", opts.DataInliningRowLimit,
		)
	}

	// DuckLake requires the data directory to exist before ATTACH runs.
	// MkdirAll is idempotent, so re-running it on every respawn is free.
	if err := os.MkdirAll(opts.DataPath, 0o755); err != nil {
		return fmt.Errorf("duckdb: creating DuckLake data path %q: %w", opts.DataPath, err)
	}

	for _, stmt := range stmts {
		if _, _, err := p.sess.exec(ctx, stmt); err != nil {
			return fmt.Errorf("duckdb: DuckLake bootstrap failed on %q: %w", stmt, err)
		}
	}
	return nil
}

// duckLakeBootstrapStmts builds the SQL statement sequence
// bootstrapDuckLakeLocked runs to attach a DuckLake catalog, and validates
// opts up front. It is a pure function (no I/O, no subprocess) precisely so
// this validation and statement construction — including the postgres-catalog
// forced inlining override below — can be unit tested without a live duckdb
// subprocess or a real Postgres server.
//
// Exactly one of CatalogPath (a local DuckLake catalog file),
// PostgresCatalogURI (an external Postgres database holding the catalog
// metadata), or SQLiteCatalogPath (a local SQLite catalog file) is required;
// DataPath is always required, since Parquet data files stay on local disk
// in every mode.
//
// localExtensionPaths overrides how the ducklake/postgres extensions are
// loaded — see installOrLoadStmts and Options.LocalExtensionPaths. Nil
// behaves exactly like the ordinary name-based INSTALL/LOAD.
func duckLakeBootstrapStmts(opts DuckLakeOptions, localExtensionPaths map[string]string) ([]string, error) {
	usingPostgres := opts.PostgresCatalogURI != ""
	usingSQLite := opts.SQLiteCatalogPath != ""
	usingQuack := opts.QuackCatalogAddr != ""
	catalogSourcesSet := 0
	for _, set := range []bool{opts.CatalogPath != "", usingPostgres, usingSQLite, usingQuack} {
		if set {
			catalogSourcesSet++
		}
	}
	if catalogSourcesSet > 1 {
		return nil, fmt.Errorf("duckdb: DuckLake enabled with more than one of CatalogPath/PostgresCatalogURI/SQLiteCatalogPath/QuackCatalogAddr set; exactly one catalog source is required")
	}
	if catalogSourcesSet == 0 {
		return nil, fmt.Errorf("duckdb: DuckLake enabled but CatalogPath is empty")
	}
	if opts.DataPath == "" {
		return nil, fmt.Errorf("duckdb: DuckLake enabled but DataPath is empty")
	}
	if usingQuack && opts.QuackCatalogToken == "" {
		return nil, fmt.Errorf("duckdb: DuckLake enabled with QuackCatalogAddr but QuackCatalogToken is empty")
	}

	// DATA_PATH is interpreted as a directory only when it ends in a
	// separator; the paths themselves are quoted through the same escaping the
	// literal encoder uses, since this transport has no parameter binding.
	catalogTarget := "ducklake:" + opts.CatalogPath
	switch {
	case usingPostgres:
		catalogTarget = "ducklake:postgres:" + opts.PostgresCatalogURI
	case usingSQLite:
		catalogTarget = "ducklake:sqlite:" + opts.SQLiteCatalogPath
	case usingQuack:
		catalogTarget = "ducklake:quack:" + opts.QuackCatalogAddr
	}
	catalogLiteral, err := encodeLiteral(catalogTarget)
	if err != nil {
		return nil, fmt.Errorf("duckdb: encoding DuckLake catalog target: %w", err)
	}
	dataLiteral, err := encodeLiteral(strings.TrimSuffix(opts.DataPath, "/") + "/")
	if err != nil {
		return nil, fmt.Errorf("duckdb: encoding DuckLake data path: %w", err)
	}

	rowLimit := opts.DataInliningRowLimit
	switch {
	case usingPostgres:
		// https://github.com/duckdb/ducklake/issues/1175: DuckLake's postgres
		// catalog backend corrupts inlined rows that contain JSON columns (it
		// writes a byte-escaped literal where postgres expects valid JSON),
		// so inlining is forced off for a postgres catalog regardless of
		// whatever DataInliningRowLimit the caller configured, until that bug
		// is fixed upstream.
		rowLimit = 0
	case rowLimit == 0:
		rowLimit = DefaultDataInliningRowLimit
	case rowLimit < 0:
		// A negative value means "explicitly disable inlining" — distinct
		// from the zero value, which means "unset, use the default". See
		// DataInliningRowLimit's doc comment.
		rowLimit = 0
	}

	var stmts []string
	switch {
	case usingPostgres:
		// DuckLake's postgres catalog backend depends on the postgres
		// extension to talk to the metadata database; install/load it before
		// ducklake so the ATTACH below can actually reach it.
		postgresStmts, err := installOrLoadStmts("postgres", localExtensionPaths)
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, postgresStmts...)
	case usingSQLite:
		// Same reasoning, against the sqlite extension instead.
		sqliteStmts, err := installOrLoadStmts("sqlite", localExtensionPaths)
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, sqliteStmts...)
	case usingQuack:
		// httpfs is a prerequisite for the same reason startQuackLocked
		// loads it before quack_serve on a self-built binary: a real crypto
		// engine for the ATTACH itself, confirmed by a real "Extension
		// Autoloading Error: ... required extension 'httpfs" failure
		// attaching against a self-built duckdb without it. Only relevant
		// via LocalExtensionPaths (a self-built binary); an ordinary
		// published DuckDB build already has a real crypto engine
		// registered without it, exactly as startQuackLocked's own comment
		// on this describes.
		if httpfsPath, ok := localExtensionPaths["httpfs"]; ok && httpfsPath != "" {
			httpfsStmts, err := installOrLoadStmts("httpfs", localExtensionPaths)
			if err != nil {
				return nil, err
			}
			stmts = append(stmts, httpfsStmts...)
		}
		// Same reasoning as postgres/sqlite above, against the quack
		// extension instead — this process needs it loaded to act as a
		// quack *client* attaching to QuackCatalogAddr's listener, the same
		// requirement any other quack client has (this driver's own
		// data-plane QuackAppender included).
		quackStmts, err := installOrLoadStmts("quack", localExtensionPaths)
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, quackStmts...)
	}
	ducklakeStmts, err := installOrLoadStmts("ducklake", localExtensionPaths)
	if err != nil {
		return nil, err
	}
	stmts = append(stmts, ducklakeStmts...)

	if usingQuack {
		// Authenticates this process as a quack client of QuackCatalogAddr's
		// listener before the ATTACH below needs to reach it — SCOPE alone
		// (no explicit SECRET name on the ATTACH itself) is enough for quack
		// to resolve which secret to use, confirmed against a real
		// quack_serve listener. CREATE OR REPLACE so a crash-triggered
		// subprocess restart re-bootstrapping this same sequence doesn't
		// fail on an "already exists" error.
		scopeLiteral, err := encodeLiteral("quack:" + opts.QuackCatalogAddr)
		if err != nil {
			return nil, fmt.Errorf("duckdb: encoding DuckLake quack catalog scope: %w", err)
		}
		tokenLiteral, err := encodeLiteral(opts.QuackCatalogToken)
		if err != nil {
			return nil, fmt.Errorf("duckdb: encoding DuckLake quack catalog token: %w", err)
		}
		stmts = append(stmts, fmt.Sprintf(
			"CREATE OR REPLACE SECRET ducklake_quack_catalog (TYPE quack, TOKEN %s, SCOPE %s);",
			tokenLiteral, scopeLiteral,
		))
	}

	attachOpts := fmt.Sprintf("DATA_PATH %s, DATA_INLINING_ROW_LIMIT %d", dataLiteral, rowLimit)
	if opts.MetadataSchema != "" {
		schemaLiteral, err := encodeLiteral(opts.MetadataSchema)
		if err != nil {
			return nil, fmt.Errorf("duckdb: encoding DuckLake metadata schema: %w", err)
		}
		attachOpts += ", METADATA_SCHEMA " + schemaLiteral
	}
	stmts = append(stmts,
		// DATA_INLINING_ROW_LIMIT keeps small writes (dual-write's batched
		// flushes, cmd/duckdbseed's batched inserts) stored directly in the
		// catalog instead of each becoming its own tiny Parquet file — see
		// DuckLakeOptions.DataInliningRowLimit's doc comment.
		fmt.Sprintf("ATTACH IF NOT EXISTS %s AS %s (%s);", catalogLiteral, DuckLakeAlias, attachOpts),
	)
	return stmts, nil
}

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

// duckLakePostgresInliningOverridden reports whether opts silently discards a
// caller-configured DataInliningRowLimit because a postgres catalog forces
// inlining off (see duckLakeBootstrapStmts). Only an explicit positive limit
// counts: zero means "unset, use the default" and a negative value already
// means "explicitly disable inlining" — both already agree with, or are
// subsumed by, the forced override, so neither is worth warning about.
func duckLakePostgresInliningOverridden(opts DuckLakeOptions) bool {
	return opts.PostgresCatalogURI != "" && opts.DataInliningRowLimit > 0
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

// exec is the entry point conn.go's ExecContext calls to run a statement
// against the supervised subprocess. See runWithRestartLocked for the
// crash/restart classification shared with query below.
func (p *process) exec(ctx context.Context, sqlText string) (cols []string, rows []map[string]any, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.disabled {
		return nil, nil, ErrDisabled
	}

	runErr := p.runWithRestartLocked(ctx, func() error {
		var e error
		cols, rows, e = p.sess.exec(ctx, sqlText)
		return e
	})
	if runErr != nil {
		return nil, nil, runErr
	}
	return cols, rows, nil
}

// query is exec's counterpart for conn.go's QueryContext, additionally
// carrying the result's column types through — see sqlExecer's doc comment
// (conn.go) and rows.go's session.query / quack_session.go's
// quackSession.query for what each transport does to produce them.
func (p *process) query(ctx context.Context, sqlText string) (cols []string, types []string, rows []map[string]any, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.disabled {
		return nil, nil, nil, ErrDisabled
	}

	runErr := p.runWithRestartLocked(ctx, func() error {
		var e error
		cols, types, rows, e = p.sess.query(ctx, sqlText)
		return e
	})
	if runErr != nil {
		return nil, nil, nil, runErr
	}
	return cols, types, rows, nil
}

// runWithRestartLocked assumes mu is already held (see exec/query above). It
// runs fn once and classifies whatever it returns into the three cases that
// need different handling:
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
	err := fn()
	if err == nil {
		return nil
	}

	// A statement DuckDB itself rejected is not a transport failure: the
	// subprocess is healthy and an identical retry would fail identically,
	// so surface it to the caller untouched.
	if errors.Is(err, errStatementFailed) {
		return err
	}

	l := logger.StdlibLogger(ctx)

	// A session left mid-statement-desynced (jsonlines only — see
	// errSessionDesynced's doc comment) can never be trusted again no
	// matter what ctx says: the subprocess's own stdout stream still has
	// the abandoned statement's output queued on it, which would
	// misattribute to whatever statement runs next unless the whole
	// subprocess respawns. Everything below this assumes err does NOT wrap
	// errSessionDesynced.
	if !errors.Is(err, errSessionDesynced) && ctx.Err() != nil {
		// The caller's own context ended (cancelled or timed out), and the
		// transport in use doesn't leave anything behind that needs fixing
		// for that: quack's HTTP transport is a self-contained request per
		// statement, so an aborted request doesn't desync anything for a
		// later request on this same connection, or for any other caller
		// sharing it — verified empirically (a fresh query on the same
		// connection succeeds in ~1ms after an abort). quackSession.query's
		// own watchForCancel already sent the server a real CancelRequest
		// for this statement the moment ctx ended (verified empirically to
		// actually stop the abandoned query's CPU usage server-side, not
		// just abandon the wait for it — see encodeQuackCancelRequest's doc
		// comment), so there's nothing left for a restart to clean up here.
		// Restarting the whole subprocess would actively hurt a caller
		// sharing this connection with another workload (dual-write, say):
		// every other in-flight or queued statement gets torn down for a
		// failure that was never the subprocess's fault. It would also be
		// pure waste — the ctx.Err() check further down already skips the
		// retry once the context is done, so today's restart-then-give-up
		// pays the full cost of a restart for zero benefit in exactly this
		// case.
		l.Warn("duckdb: statement failed because its context ended; surfacing the error without restarting the subprocess", "error", err)
		return err
	}

	// Everything else means the pipe broke, the subprocess died, or (for
	// jsonlines specifically) the caller's ctx was cancelled mid-statement,
	// leaving that session protocol-desynced. Both are only recoverable by
	// respawning.
	if errors.Is(err, errSessionDesynced) {
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
	l.Info("duckdb: subprocess restarted successfully after a crash")

	if ctx.Err() != nil {
		// The restart resynced the session, but retrying under a context
		// that is already done cannot succeed.
		return err
	}

	return fn()
}

// closeLocked assumes mu is already held.
func (p *process) closeLocked(ctx context.Context) error {
	if !p.started {
		return nil
	}
	p.started = false

	// Retire the jsonlines session's reader goroutine first so it can never
	// block handing a line to an exec that will never run again. A
	// quackSession (see startQuackLocked) has no such goroutine — it's a
	// plain HTTP client — so this is a no-op once the transport has been
	// swapped.
	if closer, ok := p.sess.(interface{ close() }); ok {
		closer.close()
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
// callers address DuckLake-backed tables as inngest.<table>. It is fixed for this
// POC; making it configurable is future work.
const DuckLakeAlias = "inngest"

// DuckLakeOptions opts a process into DuckLake. It is exploratory groundwork:
// nothing in the existing Migrate / dual-write path sets it yet.
//
// CatalogPath/DataPath (or PostgresCatalogURI/DataPath, or
// SQLiteCatalogPath/DataPath) are required when this struct is used.
// Presence of the struct itself is the enable switch (Options.DuckLake ==
// nil means "disabled, behave exactly as before"), so there is no separate
// boolean to keep in sync.
type DuckLakeOptions struct {
	// CatalogPath is the path to the DuckLake metadata catalog file, attached
	// as 'ducklake:<CatalogPath>'. It is created by DuckDB on first attach and
	// is distinct from Options.DBFile: the main database can be ":memory:"
	// while the lake still persists to disk.
	//
	// Mutually exclusive with PostgresCatalogURI and SQLiteCatalogPath —
	// exactly one catalog source is required.
	CatalogPath string
	// PostgresCatalogURI, when set, attaches the DuckLake catalog against a
	// Postgres database instead of a local catalog file: 'ducklake:postgres:
	// <PostgresCatalogURI>'. Parquet data still lands under DataPath on local
	// disk in this mode — only the catalog metadata moves to Postgres.
	//
	// Mutually exclusive with CatalogPath and SQLiteCatalogPath — exactly one
	// catalog source is required. Setting this also forces
	// DataInliningRowLimit to 0 regardless of its configured value: see
	// https://github.com/duckdb/ducklake/issues/1175 (DuckLake's postgres
	// catalog backend corrupts inlined rows that contain JSON columns).
	PostgresCatalogURI string
	// SQLiteCatalogPath, when set, attaches the DuckLake catalog against a
	// SQLite database file instead of a local DuckDB catalog file:
	// 'ducklake:sqlite:<SQLiteCatalogPath>'. Like CatalogPath, it is created
	// on first attach; unlike PostgresCatalogURI, it needs no external
	// server, since the catalog file itself is the whole "server". Parquet
	// data still lands under DataPath on local disk in this mode — only the
	// catalog metadata moves to a SQLite file instead of a DuckDB one.
	//
	// Mutually exclusive with CatalogPath, PostgresCatalogURI, and
	// QuackCatalogAddr — exactly one catalog source is required. Unlike
	// PostgresCatalogURI, this does not force DataInliningRowLimit off:
	// DuckLake's SQLite metadata manager has no equivalent of ducklake#1175
	// (JSON isn't in its TypeIsNativelySupported/SupportsInlining exclusion
	// list — only FLOAT/DOUBLE/TIMESTAMP_TZ/STRUCT/MAP/LIST/VARIANT are).
	SQLiteCatalogPath string
	// QuackCatalogAddr, when set, attaches the DuckLake catalog against a
	// remote DuckDB instance over the quack wire protocol instead of a local
	// catalog file, Postgres, or SQLite: 'ducklake:quack:<QuackCatalogAddr>'
	// (host:port, no "quack:" prefix — that's added here, matching
	// Options.QuackAddr's own convention). DuckLake ships this as a first-
	// class metadata backend (QuackMetadataManager, registered as "quack"/
	// "quack_scanner") — including an optional server-side commit fast path
	// (FlushChangesServerSide) when the remote instance also has the
	// ducklake extension loaded, though that's not required for correctness:
	// confirmed working (falling back to an ordinary client-driven commit)
	// against a remote instance with only the quack extension loaded.
	//
	// The remote instance is expected to already be listening (e.g. another
	// Options.QuackAddr-configured Open call, possibly this same process's
	// own — nothing prevents a self-referential attach) with
	// QuackCatalogToken as its quack_serve auth token; this driver does not
	// spawn or manage that process. A DuckLake attach needs to authenticate
	// as a client to it (CREATE SECRET ... SCOPE 'quack:<QuackCatalogAddr>'
	// before ATTACH), which is exactly the same relationship any other quack
	// client (including this driver's own data-plane QuackAppender) has to
	// a quack listener — this just happens to be DuckLake attaching to one
	// as its metadata store instead of this driver's Go client attaching to
	// one for data.
	//
	// Mutually exclusive with CatalogPath, PostgresCatalogURI, and
	// SQLiteCatalogPath — exactly one catalog source is required. Like
	// SQLiteCatalogPath, does not force DataInliningRowLimit off: the remote
	// instance is an ordinary DuckDB database with no restricted type
	// support to work around.
	QuackCatalogAddr string
	// QuackCatalogToken is the auth token QuackCatalogAddr's listener was
	// started with (its quack_serve token) — required whenever
	// QuackCatalogAddr is set, since the CREATE SECRET this driver issues
	// before ATTACH has nothing to authenticate with otherwise.
	QuackCatalogToken string
	// MetadataSchema, when non-empty, adds a METADATA_SCHEMA clause to the
	// ATTACH statement, scoping the catalog's tables to that schema instead
	// of DuckLake's own default. Only meaningful alongside
	// PostgresCatalogURI: unlike a file catalog (already isolated per
	// CatalogPath) or a SQLite one (already isolated per SQLiteCatalogPath),
	// every attach against the same Postgres database otherwise shares one
	// namespace, so repeated/parallel runs against a long-lived Postgres
	// instance (tests, benchmarks) need a distinct schema each to avoid
	// colliding on each other's tables and DATA_PATH. Leave empty to use
	// DuckLake's own default schema.
	MetadataSchema string
	// DataPath is the directory DuckLake writes its Parquet data files into.
	// It is created (os.MkdirAll) before ATTACH runs, because DuckLake
	// requires it to exist. A trailing separator is added if absent, since
	// DuckLake only treats DATA_PATH as a directory when it ends in one.
	DataPath string
	// DataInliningRowLimit sets DuckLake's DATA_INLINING_ROW_LIMIT: writes
	// at or below this many rows stay stored directly in the catalog
	// instead of each becoming its own Parquet file — verified empirically
	// (ducklake_test.go's TestDuckLakeInlinesSmallInsertsUpToRowLimit):
	// without any limit, five separate 200-row inserts produce five Parquet
	// files; with it set high enough, the same rows stay fully inlined.
	// Leave at the zero value to use DefaultDataInliningRowLimit. Set to a
	// negative value to disable inlining entirely (DATA_INLINING_ROW_LIMIT
	// 0) — the zero value itself can't mean that, since it already means
	// "unset" for every caller that doesn't set this field at all.
	//
	// Ignored (forced to 0) when PostgresCatalogURI is set — see that field's
	// doc comment.
	DataInliningRowLimit int
}

// DefaultDataInliningRowLimit is the DATA_INLINING_ROW_LIMIT
// bootstrapDuckLakeLocked uses when DuckLakeOptions.DataInliningRowLimit is
// left at its zero value.
const DefaultDataInliningRowLimit = 1000

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
	// DuckLake, when non-nil, installs and loads the ducklake extension and
	// attaches a DuckLake catalog as DuckLakeAlias after every successful
	// health check of a freshly spawned subprocess — including the respawn
	// after a crash, since a new subprocess starts with an unattached session.
	// Leave nil (the zero value) to disable DuckLake entirely; that path is
	// byte-for-byte the previous behaviour.
	DuckLake *DuckLakeOptions

	// QuackAddr, when non-nil, starts a quack listener on the given address
	// after every successful health check of a freshly spawned subprocess.
	// Leave nil (the zero value) to disable quack entirely.
	QuackAddr *string

	// QuackServeToken, when non-empty, is used as quack_serve's fixed auth
	// token instead of a fresh, randomly generated one (generateQuackToken).
	// The random default is right for this driver's own data-plane use —
	// nothing outside this same process ever needs to know the token, since
	// every quack session this driver opens against its own listener
	// (openQuackConn, currentQuackSession) already has it in memory. A fixed
	// token is only needed when some other process must independently know
	// it in advance to attach as a client — e.g. a second Open call's
	// DuckLakeOptions.QuackCatalogAddr/QuackCatalogToken, attaching a
	// DuckLake catalog against this listener as its remote metadata store.
	// Leave empty for the ordinary random-token behavior.
	QuackServeToken string

	// QuackConns, when greater than 1, allows Open's *sql.DB to hand out up
	// to that many concurrent connections instead of the default single
	// serialized session — each one beyond the first an independent
	// quackSession opened via process.openQuackConn, genuinely concurrent
	// against the quack HTTP listener rather than serialized through
	// process.exec's mutex. Requires QuackAddr to be set; Open returns an
	// error otherwise. Leave at the zero value (0 or 1) for the original
	// single-connection behavior. See openQuackConn's doc comment for what
	// a caller (dual-write included, as of this field's use in
	// pkg/devserver/dualwrite.go) trades away by raising this: every
	// connection beyond the first loses process.exec's restart-on-crash
	// handling, and database/sql may hand any given caller either kind.
	QuackConns int

	// AllowUnsignedExtensions starts the subprocess with -unsigned, allowing
	// it to LOAD an extension that isn't signed by DuckDB — required for
	// LocalExtensionPaths, which points at a locally self-built
	// .duckdb_extension file. Leave false in every ordinary deployment;
	// this exists for testing/benchmarking against a custom-built DuckDB
	// commit (e.g. one with an upstream fix not yet published to the
	// extension repository under any resolvable name/version).
	AllowUnsignedExtensions bool
	// LocalExtensionPaths maps an extension name (e.g. "ducklake",
	// "postgres", "quack") to a local .duckdb_extension file to LOAD
	// directly, instead of the ordinary INSTALL/LOAD-by-name pair that
	// resolves against DuckDB's extension repository — see
	// installOrLoadStmts. Useful only alongside a BinaryPath pointing at a
	// matching self-built duckdb binary; an extension built from a
	// different commit than the running binary will fail to load with an
	// ABI/version mismatch. Leave nil for the ordinary name-based
	// resolution used everywhere in production.
	//
	// For an extension statically linked into that binary (no separate
	// file — DuckLake's own build does this for "ducklake" itself), set
	// its value to just the extension's bare name (e.g. "ducklake":
	// "ducklake") rather than a path: LOAD accepts a quoted bare name
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
			resolved, err := EnsureBinary(ctx, c.opts.StateDir)
			if err != nil {
				return nil, fmt.Errorf("duckdb: resolving pinned binary: %w", err)
			}
			binPath = resolved
		}
		p, err := startProcessWithDuckLake(ctx, binPath, c.opts.DBFile, c.opts.DuckLake, c.opts.QuackAddr,
			withStateDir(c.opts.StateDir),
			withAllowUnsignedExtensions(c.opts.AllowUnsignedExtensions),
			withLocalExtensionPaths(c.opts.LocalExtensionPaths),
			withQuackServeToken(c.opts.QuackServeToken),
		)
		if err != nil {
			return nil, err
		}
		c.proc = p
		return &conn{sess: c.proc}, nil
	}

	// A second and later connection when QuackConns opts into concurrency:
	// a fresh, independent quackSession instead of the shared c.proc, so
	// this connection can genuinely run alongside others rather than
	// queueing behind process.exec's mutex. See openQuackConn's doc comment
	// for what this trades away (no crash-restart handling).
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
// many connections, each beyond the first served by its own quackSession
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
// independent quackSession — see Connector.Connect and
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
