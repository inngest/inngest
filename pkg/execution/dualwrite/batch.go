package dualwrite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
	// afterInsert, if set, runs once per flush right after the INSERT
	// succeeds, given exactly the rows that flush just wrote. Used by
	// tracing.go's newSpanExporter (materializeRuns); most tables leave it
	// nil.
	afterInsert func(ctx context.Context, db *sql.DB, rows []map[string]any) error
}

// batcher drains a single per-table channel, buffering rows until maxSize or
// flushInterval, then flushes them into <table> via INSERT. A flush failure
// (e.g. subprocess down, or a row DuckDB rejects) is logged and the batch is
// dropped — it is never surfaced to the channel's senders. duckdb.ErrDisabled
// is the one exception: it's terminal, so it stops the batcher for good
// (see run) instead of being retried and re-logged on every flush.
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
		logger.StdlibLogger(ctx).Debug("dualwrite: flushing batch", "table", b.table, "rows", len(buf))
		if err := b.insert(ctx, buf); err != nil {
			// duckdb.ErrDisabled is terminal, so record it (logging once
			// across all tables) instead of warning on every flush forever.
			if errors.Is(err, duckdb.ErrDisabled) {
				b.opts.disabled.disable(ctx, err)
			} else {
				logger.StdlibLogger(ctx).Warn("dualwrite: dropping batch after flush failure", "table", b.table, "error", err, "rows", len(buf))
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
		// fill up and each send takes the drop-and-count path instead.
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

	// Rows within one flush batch don't reliably share an identical key set
	// (some hooks only add a key when there's a value for it), so build the
	// column list from the union of every row's keys rather than just
	// rows[0] — otherwise a column the first row omits would silently drop
	// for the whole batch. A row missing a given key falls through as an
	// explicit NULL via row[col]'s zero value.
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
	// Sort for a deterministic column order across flushes/tests.
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

	if _, err := b.db.ExecContext(ctx, sb.String(), args...); err != nil {
		return err
	}

	if b.opts.afterInsert != nil {
		return b.opts.afterInsert(ctx, b.db, rows)
	}
	return nil
}
