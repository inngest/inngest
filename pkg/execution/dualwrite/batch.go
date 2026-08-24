package dualwrite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/inngest/inngest/pkg/logger"
)

// disabledState tracks the driver's terminal state — duckdb.ErrDisabled, i.e.
// the subprocess died, its one restart attempt failed, and it will never be
// respawned. It is shared by every batcher so the spec's "dual-write disabled
// for the process lifetime; warning logged once" really is once for the whole
// dual-write path, not once per table per flush.
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
	// maxSize mirrors pkg/telemetry/exporters' defaultBatchMaxSize (10_000)
	// for consistency; tests use small values for fast, deterministic runs.
	maxSize int
	// flushInterval mirrors pkg/telemetry/exporters' defaultBatchTimeout (200ms).
	flushInterval time.Duration
	// disabled is shared across every table's batcher (NewListener passes one
	// instance to all of them). Left nil, each batcher gets its own — fine
	// for tests, wrong for production, which is why NewListener sets it.
	disabled *disabledState
}

// batcher drains a single per-table channel, buffering rows until maxSize or
// flushInterval, then flushes them into <table> via INSERT. A flush failure
// (e.g. subprocess down, or a row DuckDB rejects) is logged and the batch is
// dropped — it is never surfaced to the channel's senders. The one exception
// is duckdb.ErrDisabled, which is terminal rather than transient: it stops
// the batcher for good (see run) instead of being retried and re-logged on
// every subsequent flush.
type batcher struct {
	db    *sql.DB
	table string
	in    chan map[string]any
	opts  batcherOpts
	stopc chan struct{}
}

func newBatcher(db *sql.DB, table string, in chan map[string]any, opts batcherOpts) *batcher {
	if opts.maxSize <= 0 {
		opts.maxSize = 10_000
	}
	if opts.flushInterval <= 0 {
		opts.flushInterval = 200 * time.Millisecond
	}
	if opts.disabled == nil {
		opts.disabled = &disabledState{}
	}
	return &batcher{db: db, table: table, in: in, opts: opts, stopc: make(chan struct{})}
}

func (b *batcher) stop() { close(b.stopc) }

func (b *batcher) run(ctx context.Context) {
	buf := make([]map[string]any, 0, b.opts.maxSize)
	timer := time.NewTimer(b.opts.flushInterval)
	defer timer.Stop()

	flush := func() {
		if len(buf) == 0 {
			return
		}
		if b.opts.disabled.disabled() {
			buf = buf[:0]
			return
		}
		log.Println("dualwrite: flushing batch", "table", b.table, "rows", len(buf))
		if err := b.insert(ctx, buf); err != nil {
			// duckdb.ErrDisabled is terminal: the subprocess is gone for
			// good, so every future flush would fail identically. Record it
			// (logging exactly once across all tables) instead of warning
			// on every flush forever.
			if errors.Is(err, duckdb.ErrDisabled) {
				b.opts.disabled.disable(ctx, err)
			} else {
				logger.StdlibLogger(ctx).Warn("dualwrite: dropping batch after flush failure", "table", b.table, "error", err, "rows", len(buf))
			}
		}
		buf = buf[:0]
	}

	// drainRemaining does one final non-blocking sweep of b.in before the
	// last flush on exit. Without this, a row already sitting in the
	// channel when stop()/ctx cancellation fires can be lost: select's
	// pseudo-random tie-break between a ready `row := <-b.in` case and a
	// ready `<-b.stopc`/`<-ctx.Done()` case can pick the exit case even
	// though a row is waiting, and once this goroutine returns nothing
	// ever reads that row again. Only a non-blocking drain is safe here —
	// senders never close b.in (see sendEvent/sendRun/sendSpan), so a
	// blocking drain-until-empty could run forever if a hook is still
	// actively sending.
	drainRemaining := func() {
		for {
			select {
			case row := <-b.in:
				buf = append(buf, row)
			default:
				return
			}
		}
	}

	for {
		// Terminal state: stop draining and flushing entirely rather than
		// building INSERTs for a subprocess that no longer exists. The
		// listener's hooks keep working untouched — their channels simply
		// fill up, after which each send takes the drop-and-count path,
		// which is the designed no-backpressure behaviour.
		//
		// A batcher that did not discover the state itself notices within
		// one flushInterval (200ms in production), since the timer case
		// wakes it even with an idle channel.
		if b.opts.disabled.disabled() {
			return
		}

		select {
		case row := <-b.in:
			buf = append(buf, row)
			if len(buf) >= b.opts.maxSize {
				flush()
				timer.Reset(b.opts.flushInterval)
			}
		case <-timer.C:
			flush()
			timer.Reset(b.opts.flushInterval)
		case <-b.stopc:
			drainRemaining()
			flush()
			return
		case <-ctx.Done():
			drainRemaining()
			flush()
			return
		}
	}
}

func (b *batcher) insert(ctx context.Context, rows []map[string]any) error {
	if len(rows) == 0 {
		return nil
	}

	// Rows within one flush batch do NOT reliably share an identical key
	// set: several listener hooks only add an optional key when there's a
	// value for it (e.g. OnStepScheduled's step_name, OnStepFinished's
	// error), so a single run_spans_staging batch can freely mix e.g. a
	// step_scheduled row (has step_name) with a step_started row (doesn't).
	// Building the column list from rows[0] alone would silently drop any
	// column that the first row happens to omit, for every row in the
	// batch — not just the ones missing it — which is real, silent data
	// loss rather than a crash, since these optional columns are nullable.
	// Union the keys across every row instead, and let a row missing a
	// given key fall through as an explicit NULL via row[col]'s zero value.
	seen := make(map[string]struct{})
	var cols []string
	for _, row := range rows {
		for k := range row {
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			cols = append(cols, k)
		}
	}
	// Sort for a deterministic column order across flushes/tests; the
	// column list and the "?" placeholders are always generated together
	// from this same slice, so any consistent order is correct.
	sort.Strings(cols)

	var sb strings.Builder
	fmt.Fprintf(&sb, "INSERT INTO %s (%s) VALUES ", b.table, strings.Join(cols, ", "))

	args := make([]any, 0, len(rows)*len(cols))
	for i, row := range rows {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("(")
		for j, col := range cols {
			if j > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("?")
			args = append(args, row[col])
		}
		sb.WriteString(")")
	}
	sb.WriteString(";")

	_, err := b.db.ExecContext(ctx, sb.String(), args...)
	return err
}
