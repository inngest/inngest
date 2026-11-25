package driver

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/result"
	"github.com/inngest/inngest/pkg/duckdb/worker"
	"github.com/inngest/inngest/pkg/inngest/version"
	"github.com/inngest/inngest/pkg/logger"
)

// ErrDisabled is returned while a process is in a restart cooldown after a
// failed restart attempt. Callers must treat this as "subprocess temporarily
// unavailable" and back off until the cooldown expires (10s -> 30s -> 60s),
// after which the next statement retries spawning a fresh subprocess.
var ErrDisabled = errors.New("duckdb: subprocess disabled in cooldown after a failed restart attempt")

// restartCooldowns is the exponential backoff schedule after consecutive
// failed restart attempts.
var restartCooldowns = []time.Duration{
	10 * time.Second,
	30 * time.Second,
	60 * time.Second,
}

// ErrStatementFailed marks an error duckdb itself reported for a statement
// (the worker answered, it just rejected it), as opposed to a transport failure.
var ErrStatementFailed = result.ErrStatementFailed

// restartHealthTimeout bounds the health check of an exec-triggered restart.
const restartHealthTimeout = 10 * time.Second

// process supervises one DuckDB UDS worker (either a managed
// inngest-duckdb-worker child subprocess or an external sidecar socket):
// spawn/dial, bootstrap (timezone, state dir, Iceberg, resource limits,
// sandbox), crash detection, restart with exponential backoff cooldown, and
// graceful shutdown.
type process struct {
	workerBinary            string
	workerSocket            string
	workerAuthToken         string
	libPath                 string
	dbFile                  string
	iceberg                 *IcebergOptions
	stateDir                string
	allowUnsigned           bool
	localExtensionPaths     map[string]string
	quackServeToken         string
	restrictExternalAccess  bool
	resourceLimits          *ResourceLimits
	onBootstrap             func(ctx context.Context, db *sql.DB) error
	abandoned               atomic.Int64
	maxAbandoned            int

	procCtx    context.Context
	procCancel context.CancelFunc

	mu          sync.Mutex
	client      *worker.Client
	socketDir   string
	primarySess *worker.Session
	sess        sqlExecer
	workerEP    atomic.Pointer[workerEndpoint]
	quackGen    uint64
	redactor    *secretRedactor
	started       bool
	closed        bool
	disabledUntil time.Time
	restartFails  int
	now           func() time.Time
}

func (p *process) clockNow() time.Time {
	if p.now != nil {
		return p.now()
	}
	return time.Now()
}

func (p *process) isClosedLocked() bool {
	return p.closed || (p.procCtx != nil && p.procCtx.Err() != nil)
}

func (p *process) isDisabledLocked() bool {
	return !p.disabledUntil.IsZero() && p.clockNow().Before(p.disabledUntil)
}

func (p *process) pid() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.client == nil {
		return 0
	}
	return p.client.PID()
}

func (p *process) alive() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.client == nil {
		return false
	}
	return p.client.Alive()
}

type processOption func(*process)

func withStateDir(stateDir string) processOption {
	return func(p *process) { p.stateDir = stateDir }
}

func withAllowUnsignedExtensions(allow bool) processOption {
	return func(p *process) { p.allowUnsigned = allow }
}

func withLocalExtensionPaths(paths map[string]string) processOption {
	return func(p *process) { p.localExtensionPaths = paths }
}

func withQuackServeToken(token string) processOption {
	return func(p *process) { p.quackServeToken = token }
}

func withWorkerSocket(sock string) processOption {
	return func(p *process) { p.workerSocket = sock }
}

func withWorkerAuthToken(token string) processOption {
	return func(p *process) { p.workerAuthToken = token }
}

func withOnBootstrap(fn func(ctx context.Context, db *sql.DB) error) processOption {
	return func(p *process) { p.onBootstrap = fn }
}

func withLibPath(libPath string) processOption {
	return func(p *process) { p.libPath = libPath }
}

func startProcess(ctx context.Context, binaryPath, dbFile string) (*process, error) {
	return startProcessWithLake(ctx, binaryPath, dbFile, nil, nil)
}

func startProcessWithLake(ctx context.Context, binaryPath, dbFile string, iceberg *IcebergOptions, _ *string, opts ...processOption) (*process, error) {
	procCtx, procCancel := context.WithCancel(context.Background())
	p := &process{
		workerBinary: binaryPath,
		dbFile:       dbFile,
		iceberg:      iceberg,
		procCtx:      procCtx,
		procCancel:   procCancel,
		redactor:     &secretRedactor{},
	}
	for _, opt := range opts {
		opt(p)
	}
	p.redactor.add(p.quackServeToken)
	p.redactor.add(p.workerAuthToken)
	if iceberg != nil {
		p.redactor.add(iceberg.Token)
	}

	p.mu.Lock()
	err := p.spawnLocked(ctx)
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
		procCancel()
		return nil, fmt.Errorf("duckdb: subprocess failed initial startup: %w", err)
	}
	return p, nil
}

func (p *process) spawnLocked(ctx context.Context) error {
	var client *worker.Client
	var err error

	if p.workerSocket != "" {
		client, err = worker.DialWithToken(ctx, p.workerSocket, p.workerAuthToken)
		if err != nil {
			return fmt.Errorf("duckdb: dialing external worker socket %q: %w", p.workerSocket, err)
		}
	} else {
		base := ""
		if runtime.GOOS != "windows" {
			base = "/tmp"
		}
		sockDir, mkErr := os.MkdirTemp(base, "idw-*")
		if mkErr != nil {
			return fmt.Errorf("duckdb: creating worker socket directory: %w", mkErr)
		}
		sockPath := filepath.Join(sockDir, "w.sock")

		libPath := p.libPath
		if libPath == "" {
			libPath = os.Getenv("DUCKDB_LIB_PATH")
		}

		client, err = worker.Spawn(ctx, worker.SpawnConfig{
			WorkerBinary:            p.workerBinary,
			SocketPath:              sockPath,
			LibPath:                 libPath,
			DBPath:                  p.dbFile,
			StateDir:                p.stateDir,
			AllowUnsignedExtensions: p.allowUnsigned,
		})
		if err != nil {
			_ = os.RemoveAll(sockDir)
			return fmt.Errorf("duckdb: starting worker subprocess: %w", err)
		}
		p.socketDir = sockDir
	}

	sess, err := client.OpenSession(ctx, worker.WithAbandonedCounter(&p.abandoned))
	if err != nil {
		_ = client.Close()
		if p.socketDir != "" {
			_ = os.RemoveAll(p.socketDir)
			p.socketDir = ""
		}
		return fmt.Errorf("duckdb: opening primary worker session: %w", err)
	}

	p.client = client
	p.primarySess = sess
	p.sess = &workerSessionExecer{sess: sess}
	p.started = true
	return nil
}

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

func (p *process) initSessionLocked(ctx context.Context) error {
	if err := p.healthCheckLocked(ctx); err != nil {
		return err
	}

	if err := p.bootstrapTimeZoneLocked(ctx); err != nil {
		return err
	}

	sharedWorker := p.workerSocket != ""
	alreadySandboxed := false
	if sharedWorker && (p.stateDir != "" || p.iceberg != nil || p.restrictExternalAccess) {
		var err error
		alreadySandboxed, err = p.externalAccessDisabledLocked(ctx)
		if err != nil {
			return fmt.Errorf("duckdb: checking worker external access state: %w", p.redactor.redactErr(err))
		}
	}

	if err := p.bootstrapStateDirLocked(ctx, &alreadySandboxed); err != nil {
		return err
	}

	if err := p.bootstrapIcebergLocked(ctx, &alreadySandboxed); err != nil {
		return err
	}

	if err := p.bootstrapResourceLimitsLocked(ctx); err != nil {
		return err
	}

	// Last: it forbids the INSTALL/LOAD/ATTACH the phases above rely on.
	if err := p.bootstrapSandboxLocked(ctx, &alreadySandboxed); err != nil {
		return err
	}

	if p.onBootstrap != nil {
		bootstrapDB := sql.OpenDB(bootstrapConnector{sess: p.sess})
		bootstrapDB.SetMaxOpenConns(1)
		err := p.onBootstrap(ctx, bootstrapDB)
		_ = bootstrapDB.Close()
		if err != nil {
			return fmt.Errorf("duckdb: on-bootstrap hook failed: %w", p.redactor.redactErr(err))
		}
	}

	p.quackGen++
	p.workerEP.Store(&workerEndpoint{client: p.client, gen: p.quackGen})
	return nil
}

// externalAccessDisabledLocked reports whether the connected worker already
// has enable_external_access turned off (e.g. a shared external sidecar that
// was previously sandboxed by an earlier connection).
func (p *process) externalAccessDisabledLocked(ctx context.Context) (bool, error) {
	_, rows, err := p.sess.Exec(ctx, "SELECT current_setting('enable_external_access') AS v;")
	if err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}
	switch v := rows[0].Get("v").(type) {
	case bool:
		return !v, nil
	case string:
		return strings.EqualFold(v, "false"), nil
	default:
		return false, nil
	}
}

// lakeAttachedLocked reports whether LakeAlias is already attached on the
// connected worker.
func (p *process) lakeAttachedLocked(ctx context.Context) (bool, error) {
	aliasLit, err := encodeLiteral(LakeAlias)
	if err != nil {
		return false, err
	}
	_, rows, err := p.sess.Exec(ctx, fmt.Sprintf("SELECT 1 AS attached FROM duckdb_databases() WHERE database_name = %s;", aliasLit))
	if err != nil {
		return false, err
	}
	return len(rows) > 0, nil
}

type bootstrapConnector struct {
	sess sqlExecer
}

func (b bootstrapConnector) Connect(context.Context) (driver.Conn, error) {
	return &conn{sess: b.sess}, nil
}

func (b bootstrapConnector) Driver() driver.Driver { return &Driver{} }

func (p *process) bootstrapExecLocked(ctx context.Context, phase string, stmts []string) error {
	for _, stmt := range stmts {
		if _, _, err := p.sess.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("duckdb: %s bootstrap failed on %s: %w", phase, stmtSummary(stmt), p.redactor.redactErr(err))
		}
	}
	return nil
}

func (p *process) openQuackConn(ctx context.Context) (sqlExecer, error) {
	return p.openPooledQuackConn(ctx, false)
}

func (p *process) openGatedQuackConn(ctx context.Context) (sqlExecer, error) {
	return p.openPooledQuackConn(ctx, true)
}

func (p *process) openPooledQuackConn(ctx context.Context, gated bool) (sqlExecer, error) {
	c := &pooledQuackConn{p: p, gated: gated}
	if _, err := c.currentSession(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

func (p *process) currentWorkerEndpoint(ctx context.Context) (*workerEndpoint, error) {
	if ep := p.workerEP.Load(); ep != nil {
		return ep, nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.isClosedLocked() {
		return nil, errQuackUnavailable
	}
	if p.isDisabledLocked() {
		return nil, ErrDisabled
	}
	if ep := p.workerEP.Load(); ep != nil {
		return ep, nil
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err := p.restartAfterFailureLocked(ctx, errQuackUnavailable); err != nil {
		return nil, err
	}
	if ep := p.workerEP.Load(); ep != nil {
		return ep, nil
	}
	return nil, errQuackUnavailable
}

func (p *process) recoverQuack(ctx context.Context, gen uint64, cause error) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.isClosedLocked() {
		return errQuackUnavailable
	}
	if p.isDisabledLocked() {
		return ErrDisabled
	}
	if p.quackGen != gen {
		logger.StdlibLogger(ctx).Debug("duckdb: pooled worker session is stale; subprocess already restarted",
			"transport", "worker", "stale_generation", gen, "generation", p.quackGen)
		return nil
	}
	return p.restartAfterFailureLocked(ctx, cause)
}

func (p *process) currentWorkerSession(ctx context.Context) (*worker.Session, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.primarySess == nil {
		return nil, errQuackUnavailable
	}
	if err := p.replaceAbandonedSessionLocked(ctx); err != nil {
		return nil, err
	}
	return p.primarySess, nil
}

func (p *process) bootstrapTimeZoneLocked(ctx context.Context) error {
	return p.bootstrapExecLocked(ctx, "time zone", []string{"SET GLOBAL TimeZone = 'UTC';"})
}

func (p *process) bootstrapStateDirLocked(ctx context.Context, alreadySandboxed *bool) error {
	if p.stateDir == "" {
		return nil
	}
	if alreadySandboxed != nil && *alreadySandboxed {
		return nil
	}
	homeDir := filepath.Join(p.stateDir, "duckdb")
	extDir := filepath.Join(homeDir, "extensions")
	tmpDir := filepath.Join(p.stateDir, "tmp")
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		return fmt.Errorf("duckdb: creating extension directory %q: %w", extDir, err)
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return fmt.Errorf("duckdb: creating temp directory %q: %w", tmpDir, err)
	}

	homeLiteral, err := encodeLiteral(homeDir)
	if err != nil {
		return fmt.Errorf("duckdb: encoding home directory: %w", err)
	}
	extLiteral, err := encodeLiteral(extDir)
	if err != nil {
		return fmt.Errorf("duckdb: encoding extension directory: %w", err)
	}
	tmpLiteral, err := encodeLiteral(tmpDir)
	if err != nil {
		return fmt.Errorf("duckdb: encoding temp directory: %w", err)
	}

	stmts := []string{
		fmt.Sprintf("SET home_directory=%s;", homeLiteral),
		fmt.Sprintf("SET extension_directories=[%s];", extLiteral),
		fmt.Sprintf("SET temp_directory=%s;", tmpLiteral),
	}
	err = p.bootstrapExecLocked(ctx, "state dir", stmts)
	if err != nil && p.workerSocket != "" {
		// Another client on the shared external worker may have finished
		// sandboxing it concurrently, which blocks changing these directories.
		if disabled, checkErr := p.externalAccessDisabledLocked(ctx); checkErr == nil && disabled {
			if alreadySandboxed != nil {
				*alreadySandboxed = true
			}
			return nil
		}
	}
	return err
}

func (p *process) bootstrapIcebergLocked(ctx context.Context, alreadySandboxed *bool) error {
	if p.iceberg == nil {
		return nil
	}
	sharedWorker := p.workerSocket != ""
	stmts, err := icebergBootstrapStmtsForWorker(*p.iceberg, p.localExtensionPaths, sharedWorker)
	if err != nil {
		return err
	}
	if sharedWorker && alreadySandboxed != nil && *alreadySandboxed {
		return p.verifySandboxedIcebergLocked(ctx)
	}
	err = p.bootstrapExecLocked(ctx, "Iceberg", stmts)
	if err != nil && sharedWorker {
		// If another client sandboxed the shared worker while we were
		// bootstrapping, INSTALL/LOAD/ATTACH will fail. Reuse the worker's
		// completed Iceberg setup instead of retrying blocked statements.
		if disabled, checkErr := p.externalAccessDisabledLocked(ctx); checkErr == nil && disabled {
			if alreadySandboxed != nil {
				*alreadySandboxed = true
			}
			return p.verifySandboxedIcebergLocked(ctx)
		}
		// On a shared external worker we never DETACH LakeAlias (another
		// client may be reading concurrently). If ATTACH with DEFAULT_SCHEMA
		// failed because LakeNamespace doesn't exist yet on a fresh catalog,
		// create the namespace via the Iceberg REST API and retry once.
		if nsErr := ensureIcebergNamespace(ctx, *p.iceberg); nsErr == nil {
			err = p.bootstrapExecLocked(ctx, "Iceberg", stmts)
			if err != nil {
				if disabled, checkErr := p.externalAccessDisabledLocked(ctx); checkErr == nil && disabled {
					if alreadySandboxed != nil {
						*alreadySandboxed = true
					}
					return p.verifySandboxedIcebergLocked(ctx)
				}
			}
		}
	}
	return err
}

func (p *process) verifySandboxedIcebergLocked(ctx context.Context) error {
	attached, err := p.lakeAttachedLocked(ctx)
	if err != nil {
		return fmt.Errorf("duckdb: checking Iceberg catalog %q on sandboxed worker: %w", LakeAlias, p.redactor.redactErr(err))
	}
	if !attached {
		return fmt.Errorf("duckdb: Iceberg bootstrap failed: external access is already disabled on shared worker and catalog %q is not attached", LakeAlias)
	}
	return nil
}

// ensureIcebergNamespace creates LakeNamespace on the Iceberg REST catalog via
// POST <URI>/v1[/<prefix>]/namespaces (respecting any prefix returned by
// GET <URI>/v1/config?warehouse=..., and treating 200 OK and 409 Conflict as
// success) so a shared external worker can ATTACH IF NOT EXISTS with
// DEFAULT_SCHEMA without ever detaching the shared catalog.
func ensureIcebergNamespace(ctx context.Context, opts IcebergOptions) error {
	if opts.URI == "" {
		return errors.New("empty iceberg URI")
	}
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	base := strings.TrimRight(opts.URI, "/")
	cl := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	var prefix string
	cfgURL := base + "/v1/config"
	if opts.Warehouse != "" {
		cfgURL += "?warehouse=" + url.QueryEscape(opts.Warehouse)
	}
	if cfgReq, err := http.NewRequestWithContext(reqCtx, http.MethodGet, cfgURL, nil); err == nil {
		if opts.Token != "" {
			cfgReq.Header.Set("Authorization", "Bearer "+opts.Token)
		}
		if cfgResp, err := cl.Do(cfgReq); err == nil {
			if cfgResp.StatusCode == http.StatusOK {
				var parsed struct {
					Defaults  map[string]string `json:"defaults"`
					Overrides map[string]string `json:"overrides"`
				}
				if json.NewDecoder(io.LimitReader(cfgResp.Body, 1<<20)).Decode(&parsed) == nil {
					if p := parsed.Overrides["prefix"]; p != "" {
						prefix = p
					} else if p := parsed.Defaults["prefix"]; p != "" {
						prefix = p
					}
				}
			}
			_, _ = io.Copy(io.Discard, io.LimitReader(cfgResp.Body, 4096))
			_ = cfgResp.Body.Close()
		}
	}

	endpoint := base + "/v1/namespaces"
	if p := strings.Trim(prefix, "/"); p != "" {
		endpoint = base + "/v1/" + p + "/namespaces"
	}
	body := []byte(fmt.Sprintf(`{"namespace":[%q]}`, LakeNamespace))
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if opts.Token != "" {
		req.Header.Set("Authorization", "Bearer "+opts.Token)
	}
	resp, err := cl.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
	}()
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusConflict {
		return nil
	}
	return fmt.Errorf("create namespace status %d", resp.StatusCode)
}

func icebergBootstrapStmts(opts IcebergOptions, localExtensionPaths map[string]string) ([]string, error) {
	return icebergBootstrapStmtsForWorker(opts, localExtensionPaths, false)
}

func icebergBootstrapStmtsForWorker(opts IcebergOptions, localExtensionPaths map[string]string, sharedWorker bool) ([]string, error) {
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
		stmts = append(stmts, fmt.Sprintf("CREATE OR REPLACE SECRET %s (TYPE iceberg, TOKEN %s);", icebergSecretName, tokenLit))
		attachOpts += ", SECRET " + icebergSecretName
	} else {
		attachOpts += ", AUTHORIZATION_TYPE 'none'"
	}

	if sharedWorker {
		// Never DETACH on a shared external worker: another client may be
		// actively reading from LakeAlias.
		return append(stmts,
			fmt.Sprintf("ATTACH IF NOT EXISTS %s AS %s (%s, DEFAULT_SCHEMA %s);", whLit, LakeAlias, attachOpts, nsLit),
			fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s.%s;", LakeAlias, LakeNamespace),
		), nil
	}

	return append(stmts,
		fmt.Sprintf("ATTACH IF NOT EXISTS %s AS %s (%s);", whLit, LakeAlias, attachOpts),
		fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s.%s;", LakeAlias, LakeNamespace),
		fmt.Sprintf("DETACH %s;", LakeAlias),
		fmt.Sprintf("ATTACH %s AS %s (%s, DEFAULT_SCHEMA %s);", whLit, LakeAlias, attachOpts, nsLit),
	), nil
}

const icebergSecretName = "inngest_iceberg"

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

func (p *process) restartLocked(ctx context.Context) error {
	_ = p.closeLocked(ctx)
	if err := p.spawnLocked(ctx); err != nil {
		return err
	}
	if err := p.initSessionLocked(ctx); err != nil {
		_ = p.closeLocked(context.Background())
		return err
	}
	return nil
}

func (p *process) restart(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.restartLocked(ctx)
}

func (p *process) Exec(ctx context.Context, sqlText string) (cols []string, rows []result.Row, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isDisabledLocked() {
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

func (p *process) Query(ctx context.Context, sqlText string) (cols []string, types []string, rows []result.Row, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isDisabledLocked() {
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

func (p *process) QueryWithStreams(ctx context.Context, sqlText string, streams []worker.SendStream) (cols []string, types []string, rows []result.Row, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isDisabledLocked() {
		return nil, nil, nil, ErrDisabled
	}

	query := onceOnly(func(sess *workerSessionExecer) error {
		var e error
		cols, types, rows, e = sess.QueryWithStreams(ctx, sqlText, streams)
		return e
	})
	runErr := p.runWithRestartLocked(ctx, func() error {
		sess, ok := p.sess.(*workerSessionExecer)
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

func onceOnly(fn func(*workerSessionExecer) error) func(*workerSessionExecer) error {
	var ran bool
	var first error
	return func(sess *workerSessionExecer) error {
		if ran {
			return first
		}
		ran = true
		first = fn(sess)
		return first
	}
}

func (p *process) runWithRestartLocked(ctx context.Context, fn func() error) error {
	if p.isClosedLocked() {
		return errQuackUnavailable
	}
	var err error
	if !p.started {
		err = errors.New("duckdb: subprocess not running")
	} else {
		err = p.replaceAbandonedSessionLocked(ctx)
		if err == nil {
			err = fn()
		}
	}
	if err == nil {
		return nil
	}

	if errors.Is(err, result.ErrStatementFailed) {
		return err
	}

	if errors.Is(err, worker.ErrStreamSource) {
		return err
	}

	if !errors.Is(err, result.ErrSessionDesynced) && ctx.Err() != nil {
		logger.StdlibLogger(ctx).Warn("duckdb: statement failed because its context ended; surfacing the error without restarting the subprocess", "error", err)
		return err
	}

	if restartErr := p.restartAfterFailureLocked(ctx, err); restartErr != nil {
		return restartErr
	}

	if ctx.Err() != nil {
		return err
	}

	return fn()
}

func (p *process) replaceAbandonedSessionLocked(ctx context.Context) error {
	if p.primarySess == nil || !p.primarySess.Abandoned() {
		return nil
	}
	ep := p.workerEP.Load()
	if ep == nil {
		return errQuackUnavailable
	}
	sess, err := ep.client.OpenSession(ctx, worker.WithAbandonedCounter(&p.abandoned))
	if err != nil {
		return fmt.Errorf("duckdb: replacing abandoned worker session: %w", err)
	}
	logger.StdlibLogger(ctx).Debug("duckdb: replaced worker session after an abandoned statement",
		"transport", "worker", "abandoned_session_id", p.primarySess.ID(), "session_id", sess.ID())
	p.primarySess = sess
	p.sess = &workerSessionExecer{sess: sess}
	return nil
}

func (p *process) restartAfterFailureLocked(ctx context.Context, err error) error {
	l := logger.StdlibLogger(ctx).With("transport", "worker", "generation", p.quackGen)
	l.Warn("duckdb: subprocess crashed; attempting one restart", "error", err)

	rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), restartHealthTimeout)
	defer cancel()
	if restartErr := p.restartLocked(rctx); restartErr != nil {
		idx := p.restartFails
		if idx >= len(restartCooldowns) {
			idx = len(restartCooldowns) - 1
		}
		cooldown := restartCooldowns[idx]
		p.restartFails++
		p.disabledUntil = p.clockNow().Add(cooldown)
		l.Error("duckdb: subprocess restart failed; entering cooldown before retry",
			"cooldown", cooldown,
			"consecutive_failures", p.restartFails,
			"exec_error", err,
			"restart_error", restartErr,
		)
		return fmt.Errorf("%w (cooldown %s; exec error: %v; restart error: %v)", ErrDisabled, cooldown, err, restartErr)
	}
	p.restartFails = 0
	p.disabledUntil = time.Time{}
	l.Info("duckdb: subprocess restarted successfully after a crash", "new_generation", p.quackGen)
	return nil
}

func (p *process) closeLocked(ctx context.Context) error {
	if !p.started {
		return nil
	}
	p.started = false
	p.workerEP.Store(nil)

	if p.primarySess != nil {
		closeCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		_ = p.primarySess.Close(closeCtx)
		cancel()
		p.primarySess = nil
	}
	if p.client != nil {
		_ = p.client.Close()
	}
	if p.socketDir != "" {
		_ = os.RemoveAll(p.socketDir)
		p.socketDir = ""
	}
	return nil
}

func (p *process) close(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	return p.closeLocked(ctx)
}

// EphemeralQuackAddr is retained for API compatibility with existing callers.
const EphemeralQuackAddr = "127.0.0.1:0"

// LakeAlias is the catalog name the lake attaches under, so callers address
// lake tables as inngest.<table>.
const LakeAlias = "inngest"

// LakeNamespace is the Iceberg namespace the lake tables live in, attached
// as LakeAlias's default schema.
const LakeNamespace = "main"

// IcebergOptions opts a process into an Iceberg lake: an Iceberg REST
// catalog attached as LakeAlias.
type IcebergOptions struct {
	URI       string
	Warehouse string
	Token     string
	DataPath  string
}

// Options configures Open and OpenConnector.
type Options struct {
	// WorkerSocket, when non-empty, connects to an already-running external
	// inngest-duckdb-worker sidecar listening on this Unix domain socket
	// instead of spawning a managed child worker subprocess.
	WorkerSocket string
	// WorkerAuthToken, when non-empty, is sent as the RPC auth token when
	// dialing an external WorkerSocket.
	WorkerAuthToken string
	// WorkerBinary is the path to the inngest-duckdb-worker executable in
	// managed child mode. Leave empty to fall back to BinaryPath (if set) or
	// EnsureWorkerBinary.
	WorkerBinary string
	// LibPath is an optional path to libduckdb (libduckdb.dylib / libduckdb.so
	// / duckdb.dll). When empty, the worker uses DUCKDB_LIB_PATH or lazily
	// downloads libduckdb into StateDir.
	LibPath string
	// BinaryPath is retained for compatibility: in managed child mode it is
	// used as the worker binary path when WorkerBinary is empty.
	BinaryPath string
	// StateDir is the resolved on-disk directory (e.g. ".inngest") this
	// process's on-disk state lives under.
	StateDir string
	// DBFile is the path to the .duckdb catalog file, or ":memory:".
	DBFile string
	// Iceberg, when non-nil, installs and loads the iceberg extension and
	// attaches an Iceberg REST catalog as LakeAlias.
	Iceberg *IcebergOptions

	// QuackAddr is retained for API compatibility; the worker backend natively
	// supports concurrent sessions, transactions, appenders, and streams.
	QuackAddr *string
	// QuackServeToken is retained for API compatibility.
	QuackServeToken string
	// QuackConns, when greater than 1, allows Open's *sql.DB to hand out up
	// to that many concurrent worker sessions.
	QuackConns int

	// AllowUnsignedExtensions allows loading unsigned local DuckDB extensions.
	AllowUnsignedExtensions bool
	// LocalExtensionPaths maps an extension name to a local .duckdb_extension file.
	LocalExtensionPaths map[string]string
	// RestrictExternalAccess turns off DuckDB's external access after bootstrap.
	RestrictExternalAccess bool
	// ResourceLimits caps DuckDB's memory_limit, threads, and max_temp_directory_size.
	ResourceLimits *ResourceLimits
	// MaxAbandonedStatements caps how many cancelled statements may still be
	// running on the worker before OpenShared connections fail with ErrTooManyAbandoned.
	MaxAbandonedStatements int
	// OnBootstrap, when non-nil, runs at the end of every worker session
	// bootstrap (both initial startup and crash/external-worker reconnects)
	// against a dedicated non-reentrant *sql.DB handle.
	OnBootstrap func(ctx context.Context, db *sql.DB) error
}

// DefaultMaxAbandonedStatements is Options.MaxAbandonedStatements' default.
const DefaultMaxAbandonedStatements = 4

// ErrTooManyAbandoned is returned by a statement on an OpenShared connection
// when too many abandoned statements are still running on the worker.
var ErrTooManyAbandoned = errors.New("duckdb: too many cancelled statements are still running")

// Connector implements database/sql/driver.Connector over one supervised
// DuckDB worker. It also implements io.Closer.
type Connector struct {
	opts Options
	proc *process
}

func (c *Connector) Connect(ctx context.Context) (driver.Conn, error) {
	if c.proc == nil {
		var workerBin string
		if c.opts.WorkerSocket == "" {
			workerBin = c.opts.WorkerBinary
			if workerBin == "" {
				workerBin = c.opts.BinaryPath
			}
			if workerBin == "" {
				resolved, err := EnsureWorkerBinary(ctx, c.opts.StateDir, version.Version)
				if err != nil {
					return nil, fmt.Errorf("duckdb: resolving worker binary: %w", err)
				}
				workerBin = resolved
			}
		}
		p, err := startProcessWithLake(ctx, workerBin, c.opts.DBFile, c.opts.Iceberg, c.opts.QuackAddr,
			withWorkerSocket(c.opts.WorkerSocket),
			withWorkerAuthToken(c.opts.WorkerAuthToken),
			withLibPath(c.opts.LibPath),
			withStateDir(c.opts.StateDir),
			withAllowUnsignedExtensions(c.opts.AllowUnsignedExtensions),
			withLocalExtensionPaths(c.opts.LocalExtensionPaths),
			withQuackServeToken(c.opts.QuackServeToken),
			withRestrictExternalAccess(c.opts.RestrictExternalAccess),
			withResourceLimits(c.opts.ResourceLimits),
			withMaxAbandoned(c.opts.MaxAbandonedStatements),
			withOnBootstrap(c.opts.OnBootstrap),
		)
		if err != nil {
			return nil, err
		}
		c.proc = p
		return &conn{sess: c.proc}, nil
	}

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

// Open returns a *sql.DB backed by a supervised DuckDB worker.
func Open(ctx context.Context, opts Options) (*sql.DB, error) {
	_, db, err := OpenConnector(ctx, opts)
	return db, err
}

// OpenConnector is Open, also returning the *Connector.
func OpenConnector(ctx context.Context, opts Options) (*Connector, *sql.DB, error) {
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

// OpenShared returns another *sql.DB over c's worker with its own pool, capped
// at maxConns open connections. Each connection opens its own independent
// worker session, gated on Options.MaxAbandonedStatements.
func (c *Connector) OpenShared(maxConns int) *sql.DB {
	db := sql.OpenDB(sharedConnector{c})
	db.SetMaxOpenConns(maxConns)
	return db
}

type sharedConnector struct{ c *Connector }

func (s sharedConnector) Connect(ctx context.Context) (driver.Conn, error) {
	sess, err := s.c.proc.openGatedQuackConn(ctx)
	if err != nil {
		return nil, err
	}
	return &conn{sess: sess}, nil
}

func (s sharedConnector) Driver() driver.Driver { return s.c.Driver() }
