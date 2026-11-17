package tracing

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/inngest/inngest/pkg/logger"
)

// disabledState tracks the driver's terminal state — driver.ErrDisabled, i.e.
// the subprocess died, its one restart attempt failed, and it will never be
// respawned. It is shared by every batcher so the warning is logged once for
// the whole dual-write path, not once per table per flush.
type disabledState struct {
	flag atomic.Bool
	once sync.Once
}

func (d *disabledState) disable(ctx context.Context, err error) {
	d.flag.Store(true)
	d.once.Do(func() {
		logger.StdlibLogger(ctx).Warn(
			"dualwrite: duckdb dual-write permanently disabled for this process's lifetime; no further rows will be staged",
			"error", err,
		)
	})
}

func (d *disabledState) disabled() bool { return d.flag.Load() }

type batcherOpts struct {
	maxSize       int
	flushInterval time.Duration
	// disabled is shared across every table's batcher (NewListener passes one
	// instance to all of them); left nil, each batcher gets its own, which is
	// fine for tests but not production.
	disabled *disabledState

	// Retry bounds (see flushWithRetry); zero means the package default.
	// Only tests override these.
	maxAttempts      int
	retryBaseDelay   time.Duration
	retryMaxDelay    time.Duration
	finalFlushBudget time.Duration
}

// batcher drains a single per-entity channel, buffering items until maxSize
// or flushInterval, then hands them to flush (an Inserter method). A flush
// that fails w/ an error that definitely didn't commit (see
// retryableFlushError) is retried w/ bounded backoff; any other failure
// (or running out of attempts) is logged and the batch is dropped -- it is
// never surfaced to the channel's senders. driver.ErrDisabled is terminal,
// so it stops the batcher for good (see run) instead of being re-logged on
// every flush.
type batcher[T any] struct {
	// name identifies the entity in logs (e.g. "events").
	name  string
	flush func(ctx context.Context, items []T) error
	in    chan T
	opts  batcherOpts
	stopc chan struct{}
}

func newBatcher[T any](name string, in chan T, flush func(ctx context.Context, items []T) error, opts batcherOpts) *batcher[T] {
	if opts.maxSize <= 0 {
		opts.maxSize = 10_000
	}
	if opts.flushInterval <= 0 {
		opts.flushInterval = 200 * time.Millisecond
	}
	if opts.maxAttempts <= 0 {
		opts.maxAttempts = flushMaxAttempts
	}
	if opts.retryBaseDelay <= 0 {
		opts.retryBaseDelay = flushRetryBaseDelay
	}
	if opts.retryMaxDelay <= 0 {
		opts.retryMaxDelay = flushRetryMaxDelay
	}
	if opts.finalFlushBudget <= 0 {
		opts.finalFlushBudget = finalFlushBudget
	}
	if opts.disabled == nil {
		opts.disabled = &disabledState{}
	}
	return &batcher[T]{name: name, flush: flush, in: in, opts: opts, stopc: make(chan struct{})}
}

func (b *batcher[T]) stop() { close(b.stopc) }

// flushExecTimeout bounds a single flush's Inserter call
// regardless of run's own ctx, which NewListener starts with
// context.Background() and so never expires on its own. Without this, a
// flush wedged on the duckdb subprocess (e.g. a genuine hang, not just a
// slow one) holds process.exec's mutex forever, and Close's own shutdown
// path can then never acquire that same mutex to kill the subprocess — the
// bounded bounds Close already has (see Closer, driver.Connector)
// only cover work *after* the lock is acquired, not the lock acquisition
// itself. quack.Session's HTTP client has no timeout of its own by design
// (see newQuackHTTPClient's doc comment) specifically so a caller can opt
// into a longer-running call like lake compaction; a batch flush is not
// that caller, so it gets its own short leash instead. A statement that
// merely times out here surfaces as an ordinary dropped-batch warning
// (see flush below) rather than a subprocess restart — see
// runWithRestartLocked's ctx.Err() branch.
const flushExecTimeout = 30 * time.Second

// Retry bounds for flushWithRetry. 5 attempts w/ 100ms -> 800ms backoff is
// ~1.5s of sleeping at most, so a conflict storm stalls one batcher for a few
// seconds (its channel keeps buffering meanwhile) rather than forever.
const (
	flushMaxAttempts    = 5
	flushRetryBaseDelay = 100 * time.Millisecond
	flushRetryMaxDelay  = 2 * time.Second
)

// finalFlushBudget bounds the shutdown flush (attempts + backoff together),
// so Close never waits on retries for longer than this. It's well under
// flushExecTimeout on purpose: shutdown should give up on a wedged
// subprocess quickly, not wait out a full statement timeout.
const finalFlushBudget = 10 * time.Second

// retryableFlushError reports whether a failed flush is safe to retry
// without risking duplicate rows.
//
// None of the dual-write tables dedupe on write: run_trace_spans, events and
// run_metadata are plain appends (run_metadata_rollup takes the latest row
// per kind, so a dup there is mostly harmless, but spans/events dups show up
// as-is), and runs only gets away w/ it because raw_runs collapses to the
// latest row per run (except in flat-lake mode, which skips the collapse).
// So a retry is only OK when we know the statement didn't commit, IE duckdb
// itself answered w/ an error (driver.ErrStatementFailed) and that error is a
// transient concurrency failure:
//
//   - an Iceberg commit conflict: the catalog answered the commit w/ a 409
//     (CommitFailedException, "Conflict_409" in duckdb's error), so the
//     table's metadata never moved and nothing landed. Only the 409: a 5xx
//     from the catalog is ambiguous (the commit may have gone through before
//     it failed)
//   - duckdb's own transaction conflicts ("Transaction conflict",
//     "write-write conflict") on an in-memory/local db: the commit was
//     rejected, nothing landed
//   - an aborted transaction ("Current transaction is aborted"), same thing
//
// Everything else is dropped:
//
//   - other statement errors (constraint/type/schema) are deterministic, so
//     a retry would fail identically
//   - transport errors and ctx timeouts are ambiguous: the statement may
//     have committed server-side before the response was lost (or keeps
//     running after a cancel), so retrying could double insert. The driver
//     already restarts + retries once on its own for a dead subprocess (see
//     driver's runWithRestartLocked)
func retryableFlushError(err error) bool {
	if err == nil || errors.Is(err, driver.ErrDisabled) || !errors.Is(err, driver.ErrStatementFailed) {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, s := range []string{
		"(conflict_409)",
		"transaction conflict",
		"write-write conflict",
		"current transaction is aborted",
	} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

// partialFlushError is a flush error after part of the batch already
// committed (IE InsertSpans' spans on a lake w/o atomic multi-table
// commits). Retrying the whole batch would insert that part twice, so
// flushWithRetry only ever retries retry, the part that didn't commit.
type partialFlushError struct {
	err   error
	retry func(ctx context.Context) error
}

func (e *partialFlushError) Error() string { return e.err.Error() }
func (e *partialFlushError) Unwrap() error { return e.err }

// flushWithRetry runs flush (rows is just for logs), retrying
// retryableFlushError failures up to maxAttempts times w/ exponential
// backoff (after a partialFlushError, only its retry). base bounds every
// attempt and the backoff between them (the final flush's budget).
// interrupted is true when a non-final flush gave up mid-backoff because
// stopc closed (or ctx ended), in which case rest is what's still left to
// run and the caller keeps it for the final flush.
func (b *batcher[T]) flushWithRetry(ctx, base context.Context, flush func(context.Context) error, rows int, final bool, stopc <-chan struct{}) (interrupted bool, rest func(context.Context) error, err error) {
	var done <-chan struct{}
	if !final {
		done = ctx.Done()
	}

	rest = flush
	delay := b.opts.retryBaseDelay
	for attempt := 1; ; attempt++ {
		flushCtx, cancel := context.WithTimeout(base, flushExecTimeout)
		err = rest(flushCtx)
		cancel()
		if p := (*partialFlushError)(nil); errors.As(err, &p) {
			rest = p.retry
		}
		if err == nil || !retryableFlushError(err) || attempt >= b.opts.maxAttempts || base.Err() != nil {
			return false, nil, err
		}

		logger.StdlibLogger(ctx).Debug("dualwrite: retrying batch after transient flush failure",
			"entity", b.name, "attempt", attempt, "error", err, "rows", rows)
		t := time.NewTimer(delay)
		select {
		case <-t.C:
		case <-stopc:
			t.Stop()
			return true, rest, err
		case <-done:
			t.Stop()
			return true, rest, err
		case <-base.Done():
			t.Stop()
			return false, nil, err
		}
		delay = min(delay*2, b.opts.retryMaxDelay)
	}
}

func (b *batcher[T]) run(ctx context.Context) {
	buf := make([]T, 0, b.opts.maxSize)
	timer := time.NewTimer(b.opts.flushInterval)
	defer timer.Stop()

	// flushBuf flushes buf w/ retries (see flushWithRetry). final is the
	// shutdown flush: it gets a bounded budget and can't be interrupted by
	// stop. A non-final flush whose retry backoff is interrupted by stop
	// keeps what's left of it (resume, covering buf[:resumeN]) for the final
	// flush instead of dropping it.
	var (
		resume  func(context.Context) error
		resumeN int
	)
	flushBuf := func(final bool) {
		if len(buf) == 0 {
			return
		}
		if b.opts.disabled.disabled() {
			buf, resume = buf[:0], nil
			return
		}
		logger.StdlibLogger(ctx).Debug("dualwrite: flushing batch", "entity", b.name, "rows", len(buf))
		base := context.WithoutCancel(ctx)
		stopc := b.stopc
		if final {
			var cancel context.CancelFunc
			base, cancel = context.WithTimeout(base, b.opts.finalFlushBudget)
			defer cancel()
			stopc = nil
		}

		items := buf
		var err error
		if resume != nil {
			// Only reachable from the final flush (an interrupted flush is
			// always followed by it), so this can't be interrupted again.
			_, _, err = b.flushWithRetry(ctx, base, resume, resumeN, final, stopc)
			items, resume = buf[resumeN:], nil
		}
		if len(items) > 0 {
			interrupted, rest, itemsErr := b.flushWithRetry(ctx, base, func(ctx context.Context) error { return b.flush(ctx, items) }, len(items), final, stopc)
			if interrupted {
				resume, resumeN = rest, len(buf)
				return
			}
			err = errors.Join(err, itemsErr)
		}
		if err != nil {
			// driver.ErrDisabled is terminal, so record it (logging once
			// across all tables) instead of warning on every flush forever.
			if errors.Is(err, driver.ErrDisabled) {
				b.opts.disabled.disable(ctx, err)
			} else {
				logger.StdlibLogger(ctx).Warn("dualwrite: dropping batch after flush failure", "entity", b.name, "error", err, "rows", len(buf))
			}
		}
		buf = buf[:0]
	}

	// drainRemaining does one final non-blocking sweep of b.in before the
	// last flush on exit, so a row already sitting in the channel when
	// stop()/ctx cancellation fires isn't lost to select's pseudo-random
	// tie-break. It must stay non-blocking: senders never close b.in, so a
	// drain-until-empty could run forever if a hook is still sending.
	drainRemaining := func() {
		for {
			select {
			case item := <-b.in:
				buf = append(buf, item)
			default:
				return
			}
		}
	}

	for {
		// Terminal state: stop draining and flushing entirely rather than
		// building INSERTs for a subprocess that no longer exists. The
		// listener's hooks keep working untouched — their channels simply
		// fill up and each send takes the drop-and-count path instead.
		if b.opts.disabled.disabled() {
			return
		}

		select {
		case item := <-b.in:
			buf = append(buf, item)
			if len(buf) >= b.opts.maxSize {
				flushBuf(false)
				timer.Reset(b.opts.flushInterval)
			}
		case <-timer.C:
			flushBuf(false)
			timer.Reset(b.opts.flushInterval)
		case <-b.stopc:
			drainRemaining()
			flushBuf(true)
			return
		case <-ctx.Done():
			drainRemaining()
			flushBuf(true)
			return
		}
	}
}
