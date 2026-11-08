package tracing

import (
	"context"
	"errors"
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
}

// batcher drains a single per-entity channel, buffering items until maxSize
// or flushInterval, then hands them to flush (an Inserter method). A flush
// failure (e.g. subprocess down, or a row the store rejects) is logged and
// the batch is dropped — it is never surfaced to the channel's senders.
// driver.ErrDisabled is the one exception: it's terminal, so it stops the
// batcher for good (see run) instead of being retried and re-logged on every
// flush.
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
// into a longer-running call like DuckLake compaction; a batch flush is not
// that caller, so it gets its own short leash instead. A statement that
// merely times out here surfaces as an ordinary dropped-batch warning
// (see flush below) rather than a subprocess restart — see
// runWithRestartLocked's ctx.Err() branch.
const flushExecTimeout = 30 * time.Second

func (b *batcher[T]) run(ctx context.Context) {
	buf := make([]T, 0, b.opts.maxSize)
	timer := time.NewTimer(b.opts.flushInterval)
	defer timer.Stop()

	flushBuf := func() {
		if len(buf) == 0 {
			return
		}
		if b.opts.disabled.disabled() {
			buf = buf[:0]
			return
		}
		logger.StdlibLogger(ctx).Debug("dualwrite: flushing batch", "entity", b.name, "rows", len(buf))
		flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), flushExecTimeout)
		err := b.flush(flushCtx, buf)
		cancel()
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
				flushBuf()
				timer.Reset(b.opts.flushInterval)
			}
		case <-timer.C:
			flushBuf()
			timer.Reset(b.opts.flushInterval)
		case <-b.stopc:
			drainRemaining()
			flushBuf()
			return
		case <-ctx.Done():
			drainRemaining()
			flushBuf()
			return
		}
	}
}
