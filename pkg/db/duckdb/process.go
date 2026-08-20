package duckdb

import (
	"bufio"
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/inngest/inngest/pkg/logger"
)

// process supervises one duckdb CLI subprocess: spawn, health check, one
// restart attempt on death, graceful shutdown. It never blocks or fails the
// caller — dualwrite (Task 8) treats an unhealthy process as "dual-write
// disabled" rather than propagating an error up the primary write path.
type process struct {
	binaryPath string
	dbFile     string

	mu      sync.Mutex
	cmd     *exec.Cmd
	sess    *session
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	started bool
}

func startProcess(ctx context.Context, binaryPath, dbFile string) (*process, error) {
	p := &process{binaryPath: binaryPath, dbFile: dbFile}
	if err := p.spawn(ctx); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *process) spawn(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	cmd := exec.CommandContext(ctx, p.binaryPath, p.dbFile, "-jsonlines")

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

	go logStderr(ctx, stderr)

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

func (p *process) healthCheck(ctx context.Context) error {
	rows, err := p.sess.exec(ctx, "SELECT 1 AS ok;")
	if err != nil {
		return fmt.Errorf("duckdb: health check failed: %w", err)
	}
	if len(rows) != 1 {
		return fmt.Errorf("duckdb: health check returned %d rows, expected 1", len(rows))
	}
	return nil
}

// restart attempts one respawn after a detected failure. Callers that give up
// after a failed restart must disable dual-write for the process lifetime,
// per docs/plans/006-duckdb-poc-subprocess-dual-write.md.
func (p *process) restart(ctx context.Context) error {
	_ = p.close(ctx)
	return p.spawn(ctx)
}

func (p *process) close(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

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

// Options configures Open.
type Options struct {
	// BinaryPath is the path to the duckdb executable. Leave empty to resolve
	// "duckdb" from PATH.
	BinaryPath string
	// DBFile is the path to the .duckdb catalog file, or ":memory:".
	DBFile string
}

// Connector implements database/sql/driver.Connector over one supervised
// duckdb subprocess.
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
	return &conn{sess: c.proc.sess}, nil
}

func (c *Connector) Driver() driver.Driver { return &Driver{} }

// Open returns a *sql.DB backed by a single supervised duckdb subprocess,
// with SetMaxOpenConns(1) already applied to reflect the POC's single
// serialized session — see docs/plans/006-duckdb-poc-subprocess-dual-write.md.
func Open(ctx context.Context, opts Options) (*sql.DB, error) {
	c := &Connector{opts: opts}
	if _, err := c.Connect(ctx); err != nil {
		return nil, err
	}
	db := sql.OpenDB(c)
	db.SetMaxOpenConns(1)
	return db, nil
}
