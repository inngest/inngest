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

	// "output" is JSON text, stored as inserted -- scan into `any` rather
	// than sql.NullString so a NULL reads back as nil.
	var output any
	require.NoError(t, db.QueryRowContext(context.Background(),
		"SELECT output FROM inngest.run_trace_spans WHERE span_id = 'span-with-output';",
	).Scan(&output))
	require.Equal(t, `{"data":"my-output"}`, output,
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

// cooldownDriver is a database/sql driver whose ExecContext returns
// duckdbdriver.ErrDisabled while disabled is true, and succeeds once cleared.
type cooldownDriver struct {
	disabled  atomic.Bool
	attempts  atomic.Int64
	succeeded atomic.Int64
}

func (d *cooldownDriver) Open(string) (driver.Conn, error) { return &cooldownConn{drv: d}, nil }

type cooldownConn struct{ drv *cooldownDriver }

func (c *cooldownConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not supported")
}
func (c *cooldownConn) Close() error              { return nil }
func (c *cooldownConn) Begin() (driver.Tx, error) { return nil, errors.New("not supported") }
func (c *cooldownConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	c.drv.attempts.Add(1)
	if c.drv.disabled.Load() {
		return nil, duckdbdriver.ErrDisabled
	}
	c.drv.succeeded.Add(1)
	return driver.RowsAffected(1), nil
}

// TestBatcherResumesAfterDriverCooldown verifies that driver.ErrDisabled
// (temporary restart cooldown) drops the failed batch, counts it, and keeps
// the batcher running so dual-write automatically resumes once DuckDB recovers.
func TestBatcherResumesAfterDriverCooldown(t *testing.T) {
	drv := &cooldownDriver{}
	drv.disabled.Store(true)
	name := fmt.Sprintf("duckdb-cooldown-test-%d", time.Now().UnixNano())
	sql.Register(name, drv)

	db, err := sql.Open(name, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	ctx, logs := logCtx(t)
	var dropped atomic.Int64
	ch := make(chan Event, 100)
	b := newBatcher("events", ch, NewDuckDBInserter(db).InsertEvents, batcherOpts{
		maxSize:       1,
		flushInterval: 10 * time.Millisecond,
		dropped:       &dropped,
	})

	done := make(chan struct{})
	go func() {
		b.run(ctx)
		close(done)
	}()
	t.Cleanup(func() {
		b.stop()
		<-done
	})

	ch <- testEvent("1", "a")

	require.Eventually(t, func() bool {
		return drv.attempts.Load() >= 1 && dropped.Load() == 1
	}, 2*time.Second, 10*time.Millisecond)
	require.Contains(t, logs.String(), "dualwrite: duckdb temporarily unavailable in restart cooldown")

	// Batcher must still be running (not permanently exited).
	select {
	case <-done:
		t.Fatal("batcher exited permanently on temporary ErrDisabled")
	default:
	}

	// Simulate driver recovery after cooldown: subsequent batches flush.
	drv.disabled.Store(false)
	ch <- testEvent("2", "b")

	require.Eventually(t, func() bool {
		return drv.succeeded.Load() == 1
	}, 2*time.Second, 10*time.Millisecond)
	require.Equal(t, int64(1), dropped.Load(), "recovered batch must not be counted as dropped")
}

// TestDisabledStateSharedAcrossBatchersRateLimitsWarnings proves the shared
// disabledState rate-limits cooldown warnings across all tables while keeping
// every batcher running.
func TestDisabledStateSharedAcrossBatchersRateLimitsWarnings(t *testing.T) {
	shared := &disabledState{warnInterval: time.Hour}

	drv := &cooldownDriver{}
	drv.disabled.Store(true)
	name := fmt.Sprintf("duckdb-disabled-shared-test-%d", time.Now().UnixNano())
	sql.Register(name, drv)
	db, err := sql.Open(name, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	ctx, logs := logCtx(t)
	ch1 := make(chan Event, 10)
	ch2 := make(chan RunMetadata, 10)

	ins := NewDuckDBInserter(db)
	b1 := newBatcher("events", ch1, ins.InsertEvents, batcherOpts{maxSize: 1, flushInterval: 10 * time.Millisecond, disabled: shared})
	b2 := newBatcher("run_metadata", ch2, ins.InsertRunMetadata, batcherOpts{maxSize: 1, flushInterval: 10 * time.Millisecond, disabled: shared})

	done1, done2 := make(chan struct{}), make(chan struct{})
	go func() { b1.run(ctx); close(done1) }()
	go func() { b2.run(ctx); close(done2) }()
	t.Cleanup(func() {
		b1.stop()
		b2.stop()
		<-done1
		<-done2
	})

	ch1 <- testEvent("1", "a")
	ch1 <- testEvent("2", "b")
	ch2 <- RunMetadata{
		AccountID: batchTestAccountID.String(),
		EnvID:     batchTestEnvID.String(),
		RunID:     ulid.Make().String(),
		Kind:      "test",
		Values:    json.RawMessage(`{}`),
		CreatedAt: time.Now().UTC(),
	}

	require.Eventually(t, func() bool {
		return drv.attempts.Load() >= 3
	}, 2*time.Second, 10*time.Millisecond)

	// Both batchers must still be alive, and only one warning should be logged.
	select {
	case <-done1:
		t.Fatal("b1 exited permanently on temporary ErrDisabled")
	case <-done2:
		t.Fatal("b2 exited permanently on temporary ErrDisabled")
	default:
	}

	require.Equal(t, 1, strings.Count(logs.String(), "dualwrite: duckdb temporarily unavailable in restart cooldown"),
		"cooldown warnings must be rate-limited across batchers")
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

// errConflict is what an Iceberg commit conflict looks like coming out of
// the driver: the catalog's 409 as a statement error, so it definitely
// didn't commit.
var errConflict = fmt.Errorf("%w: TransactionContext Error: Failed to commit: Failed to commit Iceberg transaction: Request to 'http://127.0.0.1:12345/v1' returned a non-200 status code (Conflict_409).", duckdbdriver.ErrStatementFailed)

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
		{"iceberg conflict", errConflict, true},
		// The commit may have gone through before the catalog failed.
		{"iceberg catalog 5xx", fmt.Errorf("%w: TransactionContext Error: Failed to commit: Failed to commit Iceberg transaction: Request to 'http://127.0.0.1:12345/v1' returned a non-200 status code (InternalServerError_500).", duckdbdriver.ErrStatementFailed), false},
		{"duckdb conflict", fmt.Errorf("%w: TransactionContext Error: Catalog write-write conflict on create with \"runs\"", duckdbdriver.ErrStatementFailed), true},
		{"aborted txn", fmt.Errorf("%w: TransactionContext Error: Current transaction is aborted (please ROLLBACK)", duckdbdriver.ErrStatementFailed), true},
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

	ins := &flakyInserter{next: NewDuckDBInserter(db), errs: []error{errConflict, errConflict}}
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
		return errConflict
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
			return errConflict
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
		{"always conflicts", func(context.Context, []Event) error { return errConflict }},
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

// After a partialFlushError only its retry runs again, never the whole
// batch (which would insert the part that already committed twice).
func TestBatcherRetriesOnlyUncommittedPart(t *testing.T) {
	for _, stopDuringBackoff := range []bool{false, true} {
		t.Run(fmt.Sprintf("stop during backoff %v", stopDuringBackoff), func(t *testing.T) {
			var flushes, retries atomic.Int64
			failed := make(chan struct{})
			retried := make(chan struct{})
			retry := func(context.Context) error {
				if retries.Add(1) == 1 && !stopDuringBackoff {
					return errConflict
				}
				close(retried)
				return nil
			}
			flush := func(context.Context, []Event) error {
				if flushes.Add(1) == 1 {
					close(failed)
					return &partialFlushError{err: errConflict, retry: retry}
				}
				return nil
			}
			opts := fastRetry(batcherOpts{maxSize: 1, flushInterval: time.Hour})
			if stopDuringBackoff {
				opts.retryBaseDelay, opts.retryMaxDelay = time.Hour, time.Hour
			}
			ch := make(chan Event, 10)
			b := newBatcher("run_trace_spans", ch, flush, opts)
			done := make(chan struct{})
			go func() { b.run(t.Context()); close(done) }()

			ch <- testEvent("1", "a")
			<-failed
			if stopDuringBackoff {
				b.stop()
			}
			select {
			case <-retried:
			case <-time.After(5 * time.Second):
				t.Fatal("retry never ran")
			}
			if !stopDuringBackoff {
				b.stop()
			}
			<-done
			require.Equal(t, int64(1), flushes.Load(), "the batch itself is flushed once")
			want := int64(2)
			if stopDuringBackoff {
				want = 1 // by the final flush
			}
			require.Equal(t, want, retries.Load())
		})
	}
}

// An interrupted flush's leftover (resume) can be retried by a non-final
// flush, if run takes another item off b.in before it sees stop. That
// retry's backoff is interrupted by stop right away, and it has to keep
// resume for the final flush rather than drop it. run's select picks
// between the two at random, so this runs a bunch of times.
func TestBatcherKeepsResumeInterruptedAgain(t *testing.T) {
	for i := range 20 {
		var flushes, retries atomic.Int64
		var succeeded atomic.Bool
		failed := make(chan struct{})
		retry := func(context.Context) error {
			if retries.Add(1) == 1 {
				return errConflict
			}
			succeeded.Store(true)
			return nil
		}
		var got []string
		flush := func(_ context.Context, items []Event) error {
			if flushes.Add(1) == 1 {
				close(failed)
				return &partialFlushError{err: errConflict, retry: retry}
			}
			for _, it := range items {
				got = append(got, it.EventName)
			}
			return nil
		}
		opts := batcherOpts{maxSize: 1, flushInterval: time.Hour, retryBaseDelay: 100 * time.Millisecond, retryMaxDelay: 100 * time.Millisecond}
		ch := make(chan Event, 10)
		b := newBatcher("run_trace_spans", ch, flush, opts)
		done := make(chan struct{})
		go func() { b.run(t.Context()); close(done) }()

		ch <- testEvent("1", "a")
		<-failed
		ch <- testEvent("2", "b")
		b.stop()
		<-done
		require.True(t, succeeded.Load(), "run %d: the uncommitted part was dropped", i)
		require.Equal(t, []string{"b"}, got, "run %d", i)
	}
}
