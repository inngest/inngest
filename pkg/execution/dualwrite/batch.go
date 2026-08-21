package dualwrite

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/logger"
)

type batcherOpts struct {
	// maxSize mirrors pkg/telemetry/exporters' defaultBatchMaxSize (10_000)
	// for consistency; tests use small values for fast, deterministic runs.
	maxSize int
	// flushInterval mirrors pkg/telemetry/exporters' defaultBatchTimeout (200ms).
	flushInterval time.Duration
}

// batcher drains a single per-table channel, buffering rows until maxSize or
// flushInterval, then flushes them into <table> via INSERT. A flush failure
// (e.g. subprocess down) is logged and the batch is dropped — it is never
// surfaced to the channel's senders.
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
		if err := b.insert(ctx, buf); err != nil {
			logger.StdlibLogger(ctx).Warn("dualwrite: dropping batch after flush failure", "table", b.table, "error", err, "rows", len(buf))
		}
		buf = buf[:0]
	}

	for {
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
			flush()
			return
		case <-ctx.Done():
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
