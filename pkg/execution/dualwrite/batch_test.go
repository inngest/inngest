package dualwrite

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/stretchr/testify/require"
)

// eventRow builds a row shaped exactly like the listener's OnEventReceived
// output (see eventPartitionFields in listener.go): every real row includes
// account_id/workspace_id/year/month unconditionally, and events_staging's
// year/month columns are NOT NULL, so a test row omitting them would trip a
// real constraint violation against the actual duckdb subprocess.
func eventRow(id, name string) map[string]any {
	return map[string]any{
		"event_id":     id,
		"event_name":   name,
		"account_id":   "acct-1",
		"workspace_id": "ws-1",
		"year":         2026,
		"month":        8,
		"occurred_at":  time.Now().UTC(),
	}
}

func TestBatcherFlushesOnSize(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	ch := make(chan map[string]any, 10)
	b := newBatcher(db, "events_staging", ch, batcherOpts{maxSize: 2, flushInterval: time.Hour})
	go b.run(t.Context())
	defer b.stop()

	ch <- eventRow("1", "a")
	ch <- eventRow("2", "b")

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(context.Background(), "SELECT count(*) FROM events_staging;")
		if err := row.Scan(&count); err != nil {
			return false
		}
		return count == 2
	}, 2*time.Second, 50*time.Millisecond)
}

func TestBatcherFlushesOnTimeout(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	ch := make(chan map[string]any, 10)
	b := newBatcher(db, "events_staging", ch, batcherOpts{maxSize: 100, flushInterval: 50 * time.Millisecond})
	go b.run(t.Context())
	defer b.stop()

	ch <- eventRow("1", "a")

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(context.Background(), "SELECT count(*) FROM events_staging;")
		if err := row.Scan(&count); err != nil {
			return false
		}
		return count == 1
	}, 2*time.Second, 20*time.Millisecond)
}

// TestBatcherHandlesMixedColumnsAcrossRowsInABatch is a regression test for
// a real bug found while fixing this batcher: insert() used to build its
// column list from rows[0]'s keys alone. Several listener hooks only add an
// optional key conditionally (OnStepScheduled's step_name, OnStepFinished's
// error), so a run_spans_staging batch can freely mix rows with different
// key sets. With the old rows[0]-only logic, a step_started row (no
// step_name key) landing first in the batch would silently drop step_name
// from the INSERT for every row in the batch — including a later
// step_scheduled row that *did* have a step_name value. This proves the
// fixed union-of-keys column list preserves every row's data regardless of
// its position in the batch.
func TestBatcherHandlesMixedColumnsAcrossRowsInABatch(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	ch := make(chan map[string]any, 10)
	b := newBatcher(db, "run_spans_staging", ch, batcherOpts{maxSize: 2, flushInterval: time.Hour})
	go b.run(t.Context())
	defer b.stop()

	// step_started row first, deliberately without a step_name key, so a
	// rows[0]-only column list would omit step_name from the whole batch.
	ch <- map[string]any{
		"run_id":     "run-1",
		"event_type": "step_started",
		"account_id": "acct-1", "workspace_id": "ws-1", "year": 2026, "month": 8,
		"created_at": time.Now().UTC(),
	}
	ch <- map[string]any{
		"run_id":     "run-1",
		"event_type": "step_scheduled",
		"step_name":  "my-step",
		"account_id": "acct-1", "workspace_id": "ws-1", "year": 2026, "month": 8,
		"created_at": time.Now().UTC(),
	}

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(context.Background(), "SELECT count(*) FROM run_spans_staging;")
		if err := row.Scan(&count); err != nil {
			return false
		}
		return count == 2
	}, 2*time.Second, 50*time.Millisecond)

	var stepName sql.NullString
	require.NoError(t, db.QueryRowContext(context.Background(),
		"SELECT step_name FROM run_spans_staging WHERE event_type = 'step_scheduled';",
	).Scan(&stepName))
	require.True(t, stepName.Valid, "step_scheduled row's step_name should not have been dropped")
	require.Equal(t, "my-step", stepName.String)

	require.NoError(t, db.QueryRowContext(context.Background(),
		"SELECT step_name FROM run_spans_staging WHERE event_type = 'step_started';",
	).Scan(&stepName))
	require.False(t, stepName.Valid, "step_started row never had a step_name")
}

// TestBatcherDrainsChannelOnStopBeforeExiting is a regression test for a
// real shutdown data-loss bug found while wiring Close() into pkg/devserver
// (Task 9): stop() closes stopc, but run()'s select statement has no
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
	ch := make(chan map[string]any, n)
	b := newBatcher(db, "events_staging", ch, batcherOpts{maxSize: n * 10, flushInterval: time.Hour})

	for i := 0; i < n; i++ {
		ch <- eventRow(fmt.Sprintf("id-%d", i), "burst")
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
	require.NoError(t, db.QueryRowContext(context.Background(), "SELECT count(*) FROM events_staging;").Scan(&count))
	require.Equal(t, n, count, "every row buffered before stop() must be flushed, not dropped")
}

// disabledDriver is a database/sql driver whose every statement fails with
// duckdb.ErrDisabled — exactly what the real driver returns once the
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
	return nil, duckdb.ErrDisabled
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

	ch := make(chan map[string]any, 100)
	b := newBatcher(db, "events_staging", ch, batcherOpts{maxSize: 1, flushInterval: 10 * time.Millisecond})

	done := make(chan struct{})
	go func() {
		b.run(context.Background())
		close(done)
	}()
	t.Cleanup(b.stop)

	ch <- eventRow("1", "a")

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
	ch <- eventRow("2", "b")
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
	discoverer := make(chan map[string]any, 10)
	bystander := make(chan map[string]any, 10)

	b1 := newBatcher(db, "events_staging", discoverer, batcherOpts{maxSize: 1, flushInterval: 10 * time.Millisecond, disabled: shared})
	b2 := newBatcher(db, "runs_staging", bystander, batcherOpts{maxSize: 1, flushInterval: 10 * time.Millisecond, disabled: shared})

	done1, done2 := make(chan struct{}), make(chan struct{})
	go func() { b1.run(context.Background()); close(done1) }()
	go func() { b2.run(context.Background()); close(done2) }()
	t.Cleanup(func() { b1.stop(); b2.stop() })

	discoverer <- eventRow("1", "a")

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
