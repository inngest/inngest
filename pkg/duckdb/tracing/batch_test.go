package tracing

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	duckdbdriver "github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

var (
	batchTestAccountID  = uuid.New()
	batchTestEnvID      = uuid.New()
	batchTestAppID      = uuid.New()
	batchTestFunctionID = uuid.New()
)

// testEvent builds an Event shaped like the listener's OnEventReceived output
// (see inngest.events in pkg/db/duckdb/migrations/000001_baseline.sql):
// account_id/env_id/internal_id/received_at/source/event_id/event_name/
// event_data/event_v/event_ts are all NOT NULL, so a test row omitting any
// of them would trip a real constraint violation against the actual duckdb
// subprocess.
func testEvent(id, name string) Event {
	return Event{
		AccountID:  batchTestAccountID,
		EnvID:      batchTestEnvID,
		InternalID: ulid.Make(),
		ReceivedAt: time.Now().UTC(),
		Source:     "test",
		EventID:    id,
		EventName:  name,
		EventData:  json.RawMessage(`{}`),
		EventMeta:  json.RawMessage(`{}`),
		EventV:     "1",
		EventTS:    time.Now().UTC(),
	}
}

func TestBatcherFlushesOnSize(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	ch := make(chan Event, 10)
	b := newBatcher("events", ch, NewDuckDBInserter(db).InsertEvents, batcherOpts{maxSize: 2, flushInterval: time.Hour})
	go b.run(t.Context())
	defer b.stop()

	ch <- testEvent("1", "a")
	ch <- testEvent("2", "b")

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(context.Background(), "SELECT count(*) FROM inngest.events;")
		if err := row.Scan(&count); err != nil {
			return false
		}
		return count == 2
	}, 2*time.Second, 50*time.Millisecond)
}

func TestBatcherFlushesOnTimeout(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	ch := make(chan Event, 10)
	b := newBatcher("events", ch, NewDuckDBInserter(db).InsertEvents, batcherOpts{maxSize: 100, flushInterval: 50 * time.Millisecond})
	go b.run(t.Context())
	defer b.stop()

	ch <- testEvent("1", "a")

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(context.Background(), "SELECT count(*) FROM inngest.events;")
		if err := row.Scan(&count); err != nil {
			return false
		}
		return count == 1
	}, 2*time.Second, 20*time.Millisecond)
}

// TestBatcherHandlesOptionalSpanColumnsInABatch is a regression test for a
// real bug in the old map-row inserter, which built its column list from
// rows[0]'s keys alone: inngest.run_trace_spans' "output" column is optional
// (spanExportRow only sets it when the span carries a StepOutput attribute),
// so a batch whose first span had no output silently dropped every later
// span's output too. Typed Spans always write every column; this pins that a
// span without output (NULL) and one with output mix correctly in one batch.
func TestBatcherHandlesOptionalSpanColumnsInABatch(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	ch := make(chan Span, 10)
	b := newBatcher("run_trace_spans", ch, NewDuckDBInserter(db).InsertSpans, batcherOpts{maxSize: 2, flushInterval: time.Hour})
	go b.run(t.Context())
	defer b.stop()

	base := Span{
		AccountID: batchTestAccountID.String(), EnvID: batchTestEnvID.String(),
		RunID: ulid.Make().String(), RunQueuedAt: time.Now().UTC(),
		AppID: batchTestAppID.String(), AppName: "test-app",
		FunctionID: batchTestFunctionID.String(), FunctionSlug: "test-fn",
		Name: "executor.step", StartTime: time.Now().UTC(), EndTime: time.Now().UTC(),
		TraceID: "trace-1", Attributes: json.RawMessage(`{}`),
	}

	// span-without-output first, so an inserter that derived its columns
	// from the first row would omit "output" for the whole batch.
	withoutOutput := base
	withoutOutput.SpanID = "span-without-output"
	withOutput := base
	withOutput.SpanID = "span-with-output"
	withOutput.Output = json.RawMessage(`{"data":"my-output"}`)
	withOutput.ParentSpanID = "span-without-output"
	ch <- withoutOutput
	ch <- withOutput

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(context.Background(), "SELECT count(*) FROM inngest.run_trace_spans;")
		if err := row.Scan(&count); err != nil {
			return false
		}
		return count == 2
	}, 2*time.Second, 50*time.Millisecond)

	// "output" is a JSON column, so the driver decodes it straight into a Go
	// map rather than handing back the raw string that was inserted (see
	// pkg/db/duckdb/rows.go) — scan into `any` rather than sql.NullString.
	var output any
	require.NoError(t, db.QueryRowContext(context.Background(),
		"SELECT output FROM inngest.run_trace_spans WHERE span_id = 'span-with-output';",
	).Scan(&output))
	require.Equal(t, map[string]any{"data": "my-output"}, output,
		"span-with-output row's output should not have been dropped")

	require.NoError(t, db.QueryRowContext(context.Background(),
		"SELECT output FROM inngest.run_trace_spans WHERE span_id = 'span-without-output';",
	).Scan(&output))
	require.Nil(t, output, "span-without-output row never had output")

	// A root span's parent_span_id is NULL, not "".
	parents := map[string]sql.NullString{}
	rows, err := db.QueryContext(context.Background(), "SELECT span_id, parent_span_id FROM inngest.run_trace_spans;")
	require.NoError(t, err)
	for rows.Next() {
		var id string
		var parent sql.NullString
		require.NoError(t, rows.Scan(&id, &parent))
		parents[id] = parent
	}
	require.NoError(t, rows.Err())
	require.Equal(t, map[string]sql.NullString{
		"span-without-output": {},
		"span-with-output":    {String: "span-without-output", Valid: true},
	}, parents)
}

// TestBatcherDrainsChannelOnStopBeforeExiting is a regression test for a
// real shutdown data-loss bug found while wiring Close() into pkg/devserver:
// stop() closes stopc, but run()'s select statement has no
// priority ordering between `case row := <-b.in` and `case <-b.stopc` — when
// both are simultaneously ready (rows already buffered in the channel at
// the moment stop() is called), Go's select picks between them uniformly at
// random. Before this fix, that meant a row already sent by a synchronous
// hook call (e.g. OnEventReceived) right before shutdown had roughly even
// odds of being silently dropped instead of flushed, once per affected row.
//
// This test forces the exact race: stop() is called before run() ever
// starts, so run()'s very first select iteration must choose between a
// ready b.in (n rows already buffered) and an already-closed stopc. Without
// drainRemaining, this is not merely flaky -- it fails close to
// deterministically, since the very first select is guaranteed to have both
// cases ready simultaneously.
func TestBatcherDrainsChannelOnStopBeforeExiting(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	const n = 50
	ch := make(chan Event, n)
	b := newBatcher("events", ch, NewDuckDBInserter(db).InsertEvents, batcherOpts{maxSize: n * 10, flushInterval: time.Hour})

	for i := 0; i < n; i++ {
		ch <- testEvent(fmt.Sprintf("id-%d", i), "burst")
	}
	b.stop()

	done := make(chan struct{})
	go func() {
		b.run(context.Background())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("batcher did not exit after stop")
	}

	var count int
	require.NoError(t, db.QueryRowContext(context.Background(), "SELECT count(*) FROM inngest.events;").Scan(&count))
	require.Equal(t, n, count, "every row buffered before stop() must be flushed, not dropped")
}

// disabledDriver is a database/sql driver whose every statement fails with
// duckdbdriver.ErrDisabled — exactly what the real driver returns once the
// subprocess has died and its one restart attempt has failed. It counts
// attempts so a test can prove the batcher stops trying.
type disabledDriver struct{ attempts atomic.Int64 }

func (d *disabledDriver) Open(string) (driver.Conn, error) { return &disabledConn{drv: d}, nil }

type disabledConn struct{ drv *disabledDriver }

func (c *disabledConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not supported")
}
func (c *disabledConn) Close() error              { return nil }
func (c *disabledConn) Begin() (driver.Tx, error) { return nil, errors.New("not supported") }
func (c *disabledConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	c.drv.attempts.Add(1)
	return nil, duckdbdriver.ErrDisabled
}

// TestBatcherStopsFlushingOnceDriverDisabled covers Fix 5: before it, the
// driver's terminal "disabled" state was unobservable to the batcher, so the
// batchers kept draining channels, building INSERTs, calling ExecContext and
// logging a Warn on every single flush, forever. The spec wants dual-write
// disabled for the process lifetime with the warning logged once.
func TestBatcherStopsFlushingOnceDriverDisabled(t *testing.T) {
	drv := &disabledDriver{}
	name := fmt.Sprintf("duckdb-disabled-test-%d", time.Now().UnixNano())
	sql.Register(name, drv)

	db, err := sql.Open(name, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	ch := make(chan Event, 100)
	b := newBatcher("events", ch, NewDuckDBInserter(db).InsertEvents, batcherOpts{maxSize: 1, flushInterval: 10 * time.Millisecond})

	done := make(chan struct{})
	go func() {
		b.run(context.Background())
		close(done)
	}()
	t.Cleanup(b.stop)

	ch <- testEvent("1", "a")

	// The batcher must exit of its own accord once it observes ErrDisabled,
	// rather than spinning on a subprocess that will never come back.
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("batcher kept running after the driver reported ErrDisabled")
	}

	require.Equal(t, int64(1), drv.attempts.Load(),
		"exactly one flush attempt should have been made before giving up")
	require.True(t, b.opts.disabled.disabled())

	// Further rows are simply never flushed: no more ExecContext calls.
	ch <- testEvent("2", "b")
	time.Sleep(100 * time.Millisecond)
	require.Equal(t, int64(1), drv.attempts.Load())
}

// TestDisabledStateSharedAcrossBatchersLogsOnce proves the shared
// disabledState NewListener hands to every table's batcher makes the terminal
// state stop the whole dual-write path — one batcher observing ErrDisabled
// stops the others too — and that the warning is emitted once rather than
// once per table.
func TestDisabledStateSharedAcrossBatchersLogsOnce(t *testing.T) {
	shared := &disabledState{}

	drv := &disabledDriver{}
	name := fmt.Sprintf("duckdb-disabled-shared-test-%d", time.Now().UnixNano())
	sql.Register(name, drv)
	db, err := sql.Open(name, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	// Only this batcher ever gets a row, so only it can discover the
	// terminal state.
	discoverer := make(chan Event, 10)
	bystander := make(chan RunMetadata, 10)

	ins := NewDuckDBInserter(db)
	b1 := newBatcher("events", discoverer, ins.InsertEvents, batcherOpts{maxSize: 1, flushInterval: 10 * time.Millisecond, disabled: shared})
	b2 := newBatcher("run_metadata", bystander, ins.InsertRunMetadata, batcherOpts{maxSize: 1, flushInterval: 10 * time.Millisecond, disabled: shared})

	done1, done2 := make(chan struct{}), make(chan struct{})
	go func() { b1.run(context.Background()); close(done1) }()
	go func() { b2.run(context.Background()); close(done2) }()
	t.Cleanup(func() { b1.stop(); b2.stop() })

	discoverer <- testEvent("1", "a")

	for _, done := range []chan struct{}{done1, done2} {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("a batcher kept running after the shared disabled state was set")
		}
	}

	// The bystander never even attempted a flush, so the driver saw exactly
	// the discoverer's single attempt.
	require.Equal(t, int64(1), drv.attempts.Load())

	// sync.Once is what bounds the warning to one emission; calling disable
	// again must not re-log.
	logged := 0
	shared.once.Do(func() { logged++ })
	require.Equal(t, 0, logged, "the warning must already have been logged exactly once")
}

// syncBuffer is a goroutine-safe log sink for asserting on batcher logs.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func logCtx(t *testing.T) (context.Context, *syncBuffer) {
	buf := &syncBuffer{}
	l := logger.FromSlog(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})), slog.LevelDebug)
	return logger.WithStdlib(t.Context(), l), buf
}

// conflictErr is what a DuckLake commit conflict looks like coming out of
// the driver: a statement error, so it definitely didn't commit.
var conflictErr = fmt.Errorf("%w: TransactionContext Error: Failed to commit DuckLake transaction: Transaction conflict - attempting to insert into table with index \"1\"", duckdbdriver.ErrStatementFailed)

// fastRetry keeps retry tests quick.
func fastRetry(o batcherOpts) batcherOpts {
	o.retryBaseDelay = 5 * time.Millisecond
	o.retryMaxDelay = 20 * time.Millisecond
	return o
}

func TestRetryableFlushError(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"ducklake conflict", conflictErr, true},
		{"aborted txn", fmt.Errorf("%w: TransactionContext Error: Current transaction is aborted (please ROLLBACK)", duckdbdriver.ErrStatementFailed), true},
		{"sqlite catalog busy", fmt.Errorf("%w: IO Error: database is locked", duckdbdriver.ErrStatementFailed), true},
		{"postgres catalog serialization", fmt.Errorf("%w: could not serialize access due to concurrent update", duckdbdriver.ErrStatementFailed), true},
		{"constraint", fmt.Errorf("%w: Constraint Error: NOT NULL constraint failed: runs.status", duckdbdriver.ErrStatementFailed), false},
		// Ambiguous: the statement may have committed before the response
		// was lost, so a retry could insert twice.
		{"transport", errors.New("duckdb: quack prepare: connection reset by peer"), false},
		{"transport mentioning conflict", errors.New("Transaction conflict, then EOF"), false},
		{"timeout", context.DeadlineExceeded, false},
		{"disabled", fmt.Errorf("%w: boom", duckdbdriver.ErrDisabled), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, retryableFlushError(tc.err))
		})
	}
}

// flakyInserter fails InsertEvents w/ errs (in order) before passing
// through to next, counting every call.
type flakyInserter struct {
	next  *DuckDBInserter
	mu    sync.Mutex
	errs  []error
	calls int
}

func (f *flakyInserter) InsertEvents(ctx context.Context, events []Event) error {
	f.mu.Lock()
	f.calls++
	var err error
	if len(f.errs) > 0 {
		err, f.errs = f.errs[0], f.errs[1:]
	}
	f.mu.Unlock()
	if err != nil {
		return err
	}
	return f.next.InsertEvents(ctx, events)
}

func (f *flakyInserter) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func countEvents(t *testing.T, db *sql.DB) int {
	var n int
	require.NoError(t, db.QueryRowContext(context.Background(), "SELECT count(*) FROM inngest.events;").Scan(&n))
	return n
}

func TestBatcherRetriesTransientFlushFailureWithoutDuplicates(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	ins := &flakyInserter{next: NewDuckDBInserter(db), errs: []error{conflictErr, conflictErr}}
	ch := make(chan Event, 10)
	b := newBatcher("events", ch, ins.InsertEvents, fastRetry(batcherOpts{maxSize: 2, flushInterval: time.Hour}))
	go b.run(t.Context())
	defer b.stop()

	ch <- testEvent("1", "a")
	ch <- testEvent("2", "b")

	require.Eventually(t, func() bool { return countEvents(t, db) == 2 }, 5*time.Second, 20*time.Millisecond)
	require.Equal(t, 3, ins.callCount(), "2 conflicts then a success")

	// Nothing else lands later either.
	time.Sleep(100 * time.Millisecond)
	require.Equal(t, 2, countEvents(t, db))
}

func TestBatcherDropsNonRetryableFlushFailure(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"statement error", fmt.Errorf("%w: Constraint Error: NOT NULL constraint failed", duckdbdriver.ErrStatementFailed)},
		{"ambiguous transport error", errors.New("duckdb: quack prepare: unexpected EOF")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, logs := logCtx(t)
			var calls atomic.Int64
			flush := func(context.Context, []Event) error {
				calls.Add(1)
				return tc.err
			}
			ch := make(chan Event, 10)
			b := newBatcher("events", ch, flush, fastRetry(batcherOpts{maxSize: 1, flushInterval: time.Hour}))
			done := make(chan struct{})
			go func() { b.run(ctx); close(done) }()

			ch <- testEvent("1", "a")
			require.Eventually(t, func() bool {
				return strings.Contains(logs.String(), "dualwrite: dropping batch after flush failure")
			}, 2*time.Second, 10*time.Millisecond)
			require.Equal(t, int64(1), calls.Load(), "no retry")

			b.stop()
			<-done
			require.Equal(t, int64(1), calls.Load(), "the dropped batch isn't flushed again on stop")
		})
	}
}

func TestBatcherGivesUpAfterMaxAttempts(t *testing.T) {
	ctx, logs := logCtx(t)
	var calls atomic.Int64
	flush := func(context.Context, []Event) error {
		calls.Add(1)
		return conflictErr
	}
	ch := make(chan Event, 10)
	b := newBatcher("events", ch, flush, fastRetry(batcherOpts{maxSize: 1, flushInterval: time.Hour, maxAttempts: 3}))
	go b.run(ctx)
	defer b.stop()

	ch <- testEvent("1", "a")
	require.Eventually(t, func() bool {
		return strings.Contains(logs.String(), "dualwrite: dropping batch after flush failure")
	}, 2*time.Second, 10*time.Millisecond)
	require.Equal(t, int64(3), calls.Load())
}

// A batch whose retry backoff is interrupted by stop isn't dropped: the
// final flush picks it up.
func TestBatcherStopDuringBackoffKeepsBatchForFinalFlush(t *testing.T) {
	var calls atomic.Int64
	var flushed atomic.Int64
	firstFailed := make(chan struct{})
	flush := func(_ context.Context, items []Event) error {
		if calls.Add(1) == 1 {
			close(firstFailed)
			return conflictErr
		}
		flushed.Add(int64(len(items)))
		return nil
	}
	ch := make(chan Event, 10)
	// A long backoff, so stop always lands during it.
	b := newBatcher("events", ch, flush, batcherOpts{maxSize: 1, flushInterval: time.Hour, retryBaseDelay: time.Hour, retryMaxDelay: time.Hour})
	done := make(chan struct{})
	go func() { b.run(t.Context()); close(done) }()

	ch <- testEvent("1", "a")
	<-firstFailed
	b.stop()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("batcher did not exit after stop during backoff")
	}
	require.Equal(t, int64(2), calls.Load())
	require.Equal(t, int64(1), flushed.Load(), "the batch is flushed exactly once, by the final flush")
}

// Shutdown never waits on retries for longer than finalFlushBudget, even
// against a store that keeps conflicting or hangs.
func TestBatcherFinalFlushIsBounded(t *testing.T) {
	for _, tc := range []struct {
		name  string
		flush func(ctx context.Context, _ []Event) error
	}{
		{"always conflicts", func(context.Context, []Event) error { return conflictErr }},
		{"hangs until ctx ends", func(ctx context.Context, _ []Event) error {
			<-ctx.Done()
			return ctx.Err()
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ch := make(chan Event, 10)
			b := newBatcher("events", ch, tc.flush, batcherOpts{
				maxSize: 100, flushInterval: time.Hour,
				maxAttempts: 1000, retryBaseDelay: 10 * time.Millisecond, retryMaxDelay: 10 * time.Millisecond,
				finalFlushBudget: 200 * time.Millisecond,
			})
			ch <- testEvent("1", "a")
			b.stop()

			start := time.Now()
			done := make(chan struct{})
			go func() { b.run(t.Context()); close(done) }()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("final flush hung shutdown")
			}
			require.Less(t, time.Since(start), 2*time.Second)
		})
	}
}
