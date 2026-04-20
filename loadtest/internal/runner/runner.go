// Package runner orchestrates a single load-test run: health check, spawn
// workers, wait for ready, fire events, tear down, aggregate.
package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/inngest/inngest/loadtest/internal/config"
	"github.com/inngest/inngest/loadtest/internal/firer"
	"github.com/inngest/inngest/loadtest/internal/metrics"
	"github.com/inngest/inngest/loadtest/internal/storage"
	"github.com/inngest/inngest/loadtest/internal/telemetry"
)

// Options configures one Runner. WorkerBin is the path to the compiled
// cmd/worker-go binary.
type Options struct {
	WorkerBin string
	SocketDir string // directory under which unix-socket files are placed
	BasePort  int    // base TCP port for worker SDK handlers; each app gets BasePort+offset
}

// LogEntry is a structured phase-log entry, persisted in run summary and
// exposed to the UI via LiveStats.
type LogEntry struct {
	TS    int64  `json:"ts"`    // unix nanos
	Level string `json:"level"` // info | warn | error
	Msg   string `json:"msg"`
}

// Runner is live state for one in-progress run. It is created by Start and
// discarded when the run ends.
type Runner struct {
	runID string
	cfg   config.RunConfig
	opts  Options
	store *storage.Store

	srv     *telemetry.Server
	sock    string
	workers []*workerProc
	readyN  int32
	ready   chan struct{}

	firerRef atomic.Pointer[firer.Firer]

	cntFnStart   uint64
	cntFnEnd     uint64
	cntStepStart uint64
	cntStepEnd   uint64
	cntSamples   uint64
	cntReadyWork uint64

	logMu sync.Mutex
	log   []LogEntry

	mu          sync.Mutex
	sampleBatch []telemetry.Frame
}

// workerProc ties one subprocess to its assigned config.
type workerProc struct {
	cmd      *exec.Cmd
	cfg      config.WorkerConfig
	stderr   *bytes.Buffer
	exitCode int
	exited   atomic.Bool
}

// LiveStats is a snapshot of mid-run counters for the UI.
type LiveStats struct {
	Status           string     `json:"status"`
	EventsFired      uint64     `json:"eventsFired"`
	EventsFailed     uint64     `json:"eventsFailed"`
	LastFireError    string     `json:"lastFireError,omitempty"`
	FunctionsStarted uint64     `json:"functionsStarted"`
	FunctionsEnded   uint64     `json:"functionsEnded"`
	StepsStarted     uint64     `json:"stepsStarted"`
	StepsEnded       uint64     `json:"stepsEnded"`
	SamplesObserved  uint64     `json:"samplesObserved"`
	WorkersReady     uint64     `json:"workersReady"`
	WorkersAlive     int        `json:"workersAlive"`
	WorkersFailed    int        `json:"workersFailed"`
	Log              []LogEntry `json:"log,omitempty"`
	WorkerStderr     map[string]string `json:"workerStderr,omitempty"`
}

// Start runs the load test to completion, registering with the Manager so the
// API layer can observe live counters.
func Start(ctx context.Context, runID string, cfg config.RunConfig, opts Options, store *storage.Store, reg func(*Runner), dereg func(*Runner)) error {
	if err := cfg.Validate(); err != nil {
		_ = store.MarkRun(runID, "failed", map[string]any{"error": fmt.Sprintf("config invalid: %v", err)})
		return err
	}
	r := &Runner{
		runID: runID,
		cfg:   cfg,
		opts:  opts,
		store: store,
		ready: make(chan struct{}),
	}
	if reg != nil {
		reg(r)
	}
	if dereg != nil {
		defer dereg(r)
	}
	return r.run(ctx)
}

// RunID returns the run's id.
func (r *Runner) RunID() string { return r.runID }

// logf appends to the phase log, writes to stderr so it shows up in the
// harness console, and is safe to call from any goroutine.
func (r *Runner) logf(level, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	r.logMu.Lock()
	r.log = append(r.log, LogEntry{TS: time.Now().UnixNano(), Level: level, Msg: msg})
	r.logMu.Unlock()
	log.Printf("run %s [%s] %s", r.runID, level, msg)
}

func (r *Runner) snapshotLog() []LogEntry {
	r.logMu.Lock()
	defer r.logMu.Unlock()
	out := make([]LogEntry, len(r.log))
	copy(out, r.log)
	return out
}

// LiveStats returns a snapshot of current run progress.
func (r *Runner) LiveStats() LiveStats {
	ls := LiveStats{
		FunctionsStarted: atomic.LoadUint64(&r.cntFnStart),
		FunctionsEnded:   atomic.LoadUint64(&r.cntFnEnd),
		StepsStarted:     atomic.LoadUint64(&r.cntStepStart),
		StepsEnded:       atomic.LoadUint64(&r.cntStepEnd),
		SamplesObserved:  atomic.LoadUint64(&r.cntSamples),
		WorkersReady:     atomic.LoadUint64(&r.cntReadyWork),
		Log:              r.snapshotLog(),
	}
	if f := r.firerRef.Load(); f != nil {
		ls.EventsFired = f.Fired()
		ls.EventsFailed = f.Failed()
		ls.LastFireError = f.LastError()
	}
	alive, failed := 0, 0
	for _, w := range r.workers {
		if !w.exited.Load() {
			alive++
		} else if w.exitCode != 0 {
			failed++
		}
	}
	ls.WorkersAlive = alive
	ls.WorkersFailed = failed
	if tails := r.workerStderrTails(); len(tails) > 0 {
		ls.WorkerStderr = tails
	}
	return ls
}

func (r *Runner) markFailed(err error, extra map[string]any) {
	if extra == nil {
		extra = map[string]any{}
	}
	extra["error"] = err.Error()
	extra["log"] = r.snapshotLog()
	if tails := r.workerStderrTails(); len(tails) > 0 {
		extra["workerStderr"] = tails
	}
	r.logf("error", "run failed: %v", err)
	_ = r.store.MarkRun(r.runID, "failed", extra)
}

func (r *Runner) run(ctx context.Context) error {
	r.logf("info", "run starting: target=%s mode=%s apps=%d eventRate=%d/s duration=%s",
		r.cfg.Target.URL, r.cfg.Target.Mode, r.cfg.Apps, r.cfg.EventRate, r.cfg.Duration)

	if err := r.healthCheck(ctx); err != nil {
		r.markFailed(fmt.Errorf("health check failed: %w", err), nil)
		return err
	}
	r.logf("info", "health check passed")

	r.sock = filepath.Join(r.opts.SocketDir, fmt.Sprintf("lt-%s.sock", r.runID))
	srv, err := telemetry.Listen(r.sock, r.onFrame)
	if err != nil {
		r.markFailed(fmt.Errorf("telemetry listen %s: %w", r.sock, err), nil)
		return err
	}
	r.srv = srv
	defer func() { _ = r.srv.Close(context.Background()) }()
	r.logf("info", "telemetry socket ready: %s", r.sock)

	flushCtx, cancelFlush := context.WithCancel(context.Background())
	defer cancelFlush()
	flushDone := make(chan struct{})
	go r.flushLoop(flushCtx, flushDone)

	// Spawn workers.
	apps := make([]string, r.cfg.Apps)
	shapeList := shapesFromMix(r.cfg.ShapeMix)
	for i := 0; i < r.cfg.Apps; i++ {
		appID := fmt.Sprintf("loadtest-%s-%d", r.runID, i)
		apps[i] = appID
		wc := config.WorkerConfig{
			RunID:           r.runID,
			WorkerID:        fmt.Sprintf("w-%d", i),
			AppID:           appID,
			Target:          r.cfg.Target,
			Shapes:          shapeList,
			TelemetrySocket: r.sock,
			HTTPPort:        0,
		}
		wp, err := r.spawnWorker(wc)
		if err != nil {
			r.teardown()
			cancelFlush()
			<-flushDone
			r.markFailed(fmt.Errorf("spawn worker %s: %w", wc.WorkerID, err), nil)
			return err
		}
		r.workers = append(r.workers, wp)
		go r.watchWorker(wp)
		r.logf("info", "spawned worker %s (app=%s pid=%d)", wc.WorkerID, appID, wp.cmd.Process.Pid)
	}

	// Wait for all workers to become ready.
	r.logf("info", "waiting for %d workers to register with target...", r.cfg.Apps)
	if err := r.waitReady(ctx, 30*time.Second); err != nil {
		r.teardown()
		cancelFlush()
		<-flushDone
		hint := "check worker stderr below for the exact registration error. Common causes: wrong target URL, event key mismatch, signing key mismatch, or /fn/register unreachable."
		r.markFailed(fmt.Errorf("%s — %s", err, hint), nil)
		return err
	}
	r.logf("info", "all %d workers registered and ready", r.cfg.Apps)

	// Fire events.
	fireCtx, cancelFire := context.WithTimeout(ctx, r.cfg.Duration)
	defer cancelFire()
	_ = r.store.MarkRun(r.runID, "running", nil)
	warmupEndNs := time.Now().Add(r.cfg.Warmup).UnixNano()
	f := firer.New(r.cfg.Target, r.cfg.EventRate, r.cfg.BatchSize, apps, r.cfg.ShapeMix)
	r.firerRef.Store(f)
	r.logf("info", "firing events: rate=%d/s batchSize=%d workers=%d duration=%s warmup=%s",
		r.cfg.EventRate, r.cfg.BatchSize, r.cfg.Concurrency, r.cfg.Duration, r.cfg.Warmup)

	fireErr := f.Run(fireCtx, r.store, r.runID, r.cfg.Concurrency)
	if le := f.LastError(); le != "" {
		r.logf("error", "firer last error: %s", le)
	}
	r.logf("info", "firing ended: fired=%d failed=%d", f.Fired(), f.Failed())

	// Tail window so in-flight runs finish.
	tail := 10 * time.Second
	r.logf("info", "tail window %s", tail)
	select {
	case <-time.After(tail):
	case <-ctx.Done():
	}

	r.teardown()
	cancelFlush()
	<-flushDone

	// Compute aggregates.
	aggs, aggErr := metrics.Compute(r.store.DB(), r.runID, warmupEndNs)
	if aggErr != nil {
		r.logf("warn", "aggregate: %v", aggErr)
	} else {
		_ = r.store.WriteAggregates(r.runID, aggs)
	}

	ls := r.LiveStats()
	summary := map[string]any{
		"eventsFired":      ls.EventsFired,
		"eventsFailed":     ls.EventsFailed,
		"functionsStarted": ls.FunctionsStarted,
		"functionsEnded":   ls.FunctionsEnded,
		"stepsStarted":     ls.StepsStarted,
		"stepsEnded":       ls.StepsEnded,
		"samplesObserved":  ls.SamplesObserved,
		"workersFailed":    ls.WorkersFailed,
		"log":              ls.Log,
	}
	if ls.LastFireError != "" {
		summary["lastFireError"] = ls.LastFireError
	}
	if ls.WorkerStderr != nil {
		summary["workerStderr"] = ls.WorkerStderr
	}

	switch {
	case ctx.Err() != nil:
		r.logf("info", "run stopped by user")
		_ = r.store.MarkRun(r.runID, "stopped", summary)
		return nil
	case fireErr != nil && !errors.Is(fireErr, context.DeadlineExceeded) && !errors.Is(fireErr, context.Canceled):
		summary["error"] = fireErr.Error()
		r.logf("error", "fire error: %v", fireErr)
		_ = r.store.MarkRun(r.runID, "failed", summary)
		return fireErr
	default:
		// If no events landed (zero samples observed AND zero fired OR
		// no runs completed), mark failed with a helpful message rather than
		// silently "completed".
		if ls.FunctionsStarted == 0 {
			summary["error"] = "no functions were invoked during the run — events may not be reaching the target, or the target is not routing them to registered workers"
			r.logf("error", "%v", summary["error"])
			_ = r.store.MarkRun(r.runID, "failed", summary)
			return nil
		}
		_ = r.store.MarkRun(r.runID, "completed", summary)
		return nil
	}
}

func (r *Runner) onFrame(f telemetry.Frame) {
	if f.Phase == telemetry.PhaseReady {
		n := atomic.AddUint64(&r.cntReadyWork, 1)
		r.logf("info", "worker ready (%d/%d)", n, r.cfg.Apps)
		if atomic.AddInt32(&r.readyN, 1) == int32(r.cfg.Apps) {
			close(r.ready)
		}
		return
	}
	atomic.AddUint64(&r.cntSamples, 1)
	switch f.Phase {
	case telemetry.PhaseFnStart:
		atomic.AddUint64(&r.cntFnStart, 1)
	case telemetry.PhaseFnEnd:
		atomic.AddUint64(&r.cntFnEnd, 1)
	case telemetry.PhaseStepStart:
		atomic.AddUint64(&r.cntStepStart, 1)
	case telemetry.PhaseStepEnd:
		atomic.AddUint64(&r.cntStepEnd, 1)
	}
	r.mu.Lock()
	r.sampleBatch = append(r.sampleBatch, f)
	r.mu.Unlock()
}

func (r *Runner) flushLoop(ctx context.Context, done chan struct{}) {
	defer close(done)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	flush := func() {
		r.mu.Lock()
		batch := r.sampleBatch
		r.sampleBatch = nil
		r.mu.Unlock()
		if len(batch) == 0 {
			return
		}
		for start := 0; start < len(batch); start += 1000 {
			end := start + 1000
			if end > len(batch) {
				end = len(batch)
			}
			if err := r.store.InsertSamples(r.runID, batch[start:end]); err != nil {
				r.logf("warn", "flush samples: %v", err)
				return
			}
		}
	}
	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case <-ticker.C:
			flush()
		}
	}
}

func (r *Runner) waitReady(ctx context.Context, timeout time.Duration) error {
	t := time.NewTimer(timeout)
	defer t.Stop()
	// Poll worker liveness so we can abort fast if every worker dies before
	// any of them signals ready. Without this the run waits the full timeout
	// before reporting "worker exited code=1" to the user.
	poll := time.NewTicker(250 * time.Millisecond)
	defer poll.Stop()
	for {
		select {
		case <-r.ready:
			return nil
		case <-t.C:
			return fmt.Errorf("timed out after %s waiting for %d worker(s) to register", timeout, r.cfg.Apps)
		case <-ctx.Done():
			return ctx.Err()
		case <-poll.C:
			allDead := len(r.workers) > 0
			for _, w := range r.workers {
				if !w.exited.Load() {
					allDead = false
					break
				}
			}
			if allDead {
				return fmt.Errorf("all %d worker(s) exited before becoming ready — see worker stderr below", len(r.workers))
			}
		}
	}
}

func (r *Runner) spawnWorker(wc config.WorkerConfig) (*workerProc, error) {
	payload, err := json.Marshal(wc)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(r.opts.WorkerBin)
	cmd.Stdin = bytes.NewReader(payload)
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	cmd.Stdout = io.Discard
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &workerProc{cmd: cmd, cfg: wc, stderr: stderr}, nil
}

// watchWorker waits for a worker to exit and logs its stderr. Runs until the
// process ends.
func (r *Runner) watchWorker(w *workerProc) {
	err := w.cmd.Wait()
	w.exited.Store(true)
	if w.cmd.ProcessState != nil {
		w.exitCode = w.cmd.ProcessState.ExitCode()
	}
	tail := w.stderr.String()
	if len(tail) > 4096 {
		tail = "…" + tail[len(tail)-4096:]
	}
	if err != nil || w.exitCode != 0 {
		r.logf("error", "worker %s exited code=%d err=%v stderr=%s", w.cfg.WorkerID, w.exitCode, err, singleLine(tail))
	} else if tail != "" {
		r.logf("info", "worker %s exited cleanly; stderr=%s", w.cfg.WorkerID, singleLine(tail))
	} else {
		r.logf("info", "worker %s exited cleanly", w.cfg.WorkerID)
	}
}

func singleLine(s string) string {
	// collapse newlines so log entries stay scannable in the UI
	s = stringsMap(s, func(r rune) rune {
		if r == '\n' || r == '\r' {
			return ' '
		}
		return r
	})
	if len(s) > 400 {
		return s[:400] + "…"
	}
	return s
}

// stringsMap is a trivial inlined map to avoid an extra import; strings.Map
// would do but we want control over the replacement cost.
func stringsMap(s string, f func(rune) rune) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if nr := f(r); nr >= 0 {
			out = append(out, nr)
		}
	}
	return string(out)
}

// workerStderrTails returns the last 2 KiB of stderr from each worker that
// has content, keyed by worker ID.
func (r *Runner) workerStderrTails() map[string]string {
	tails := map[string]string{}
	for _, w := range r.workers {
		s := w.stderr.String()
		if s == "" {
			continue
		}
		if len(s) > 2048 {
			s = "…" + s[len(s)-2048:]
		}
		tails[w.cfg.WorkerID] = s
	}
	return tails
}

func (r *Runner) teardown() {
	for _, w := range r.workers {
		if w.cmd.Process != nil && !w.exited.Load() {
			_ = w.cmd.Process.Signal(os.Interrupt)
		}
	}
	done := make(chan struct{})
	go func() {
		for _, w := range r.workers {
			_ = w.cmd.Wait()
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(8 * time.Second):
		for _, w := range r.workers {
			if w.cmd.Process != nil && !w.exited.Load() {
				_ = w.cmd.Process.Kill()
			}
		}
	}
}

// healthCheck verifies the target URL is reachable. For dev mode it expects
// the dev-only /dev endpoint to respond 200. For self-hosted it falls back
// to a TCP connection + a root GET (which may return any status code — we
// only care that something is listening).
func (r *Runner) healthCheck(ctx context.Context) error {
	u := r.cfg.Target.URL
	if r.cfg.Target.Mode == config.ModeDev {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u+"/dev", nil)
		if err != nil {
			return fmt.Errorf("build /dev request for %s: %w", u, err)
		}
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("GET %s/dev: %w (is `inngest dev` running?)", u, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return fmt.Errorf("GET %s/dev returned %d — this URL does not look like an `inngest dev` server. Switch Mode to 'selfhosted' in the UI if you're running `inngest start`.", u, resp.StatusCode)
		}
		return nil
	}

	// Self-hosted: just confirm something is accepting connections.
	parsed, err := url.Parse(u)
	if err != nil {
		return fmt.Errorf("parse target url %q: %w", u, err)
	}
	host := parsed.Host
	if host == "" {
		return fmt.Errorf("target url has no host: %q", u)
	}
	if !containsPort(host) {
		if parsed.Scheme == "https" {
			host += ":443"
		} else {
			host += ":80"
		}
	}
	d := net.Dialer{Timeout: 3 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", host)
	if err != nil {
		return fmt.Errorf("dial %s: %w", host, err)
	}
	_ = conn.Close()
	return nil
}

func containsPort(host string) bool {
	// net.SplitHostPort is strict about IPv6 formatting, so just check manually.
	// Matches any ":<digits>" suffix (after the last ']' if IPv6 literal).
	if i := bytesLastIndex(host, ']'); i >= 0 {
		host = host[i+1:]
	}
	c := bytesLastIndex(host, ':')
	return c >= 0 && c < len(host)-1
}

func bytesLastIndex(s string, b byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func shapesFromMix(m config.ShapeMix) []config.Shape {
	out := make([]config.Shape, 0, len(m))
	for s, w := range m {
		if w > 0 {
			out = append(out, s)
		}
	}
	return out
}
