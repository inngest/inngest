// Package dualwrite implements the DuckDB-writing
// execution.SyncLifecycleListener for the POC described in
// docs/plans/006-duckdb-poc-subprocess-dual-write.md: the executor and
// runner call this listener's hooks synchronously, so every hook body must
// do nothing but build a row and non-blocking-send it onto a per-table
// channel. Separate background goroutines (batch.go) drain those channels
// and flush batches into DuckDB staging tables. Compaction of staged rows out
// to Hive-partitioned Parquet is deliberately out of scope for this POC and
// is not implemented here.
package dualwrite

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev1 "github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
)

// listener implements execution.SyncLifecycleListener. Every hook body does
// nothing but build a row and non-blocking-send it onto the matching
// channel — no I/O, no locking, no flush logic — which is what makes it safe
// to call synchronously from the executor and runner. See
// docs/plans/006-duckdb-poc-subprocess-dual-write.md.
type listener struct {
	execution.NoopSyncLifecycleListener

	runs   chan map[string]any
	spans  chan map[string]any
	events chan map[string]any

	droppedRuns   atomic.Int64
	droppedSpans  atomic.Int64
	droppedEvents atomic.Int64

	// db and batchers/wg back Close (below) — the shutdown path that stops
	// the background batcher goroutines this listener starts and closes the
	// db handed to NewListener. Task 8 deliberately left this out (see
	// NewListener's doc comment); Task 9 (pkg/devserver) is the first real
	// caller with a shutdown path to drive it.
	db       *sql.DB
	batchers []*batcher
	wg       sync.WaitGroup
}

func newListenerWithChannels(runsCap, spansCap, eventsCap int) *listener {
	return &listener{
		runs:   make(chan map[string]any, runsCap),
		spans:  make(chan map[string]any, spansCap),
		events: make(chan map[string]any, eventsCap),
	}
}

func (l *listener) sendRun(row map[string]any) {
	select {
	case l.runs <- row:
	default:
		l.droppedRuns.Add(1)
	}
}

func (l *listener) sendSpan(row map[string]any) {
	select {
	case l.spans <- row:
	default:
		l.droppedSpans.Add(1)
	}
}

func (l *listener) sendEvent(row map[string]any) {
	select {
	case l.events <- row:
	default:
		l.droppedEvents.Add(1)
	}
}

// runPartitionFields returns the account_id/workspace_id/year/month columns
// shared by every runs_staging and run_spans_staging row. year/month are
// derived from the run's own ID ULID timestamp (matching the existing
// RunStartedAt derivation convention in pkg/devserver/lifecycle.go), computed
// here in Go rather than by DuckDB: plain DuckDB's `COPY ... PARTITION_BY`
// clause (used by the compactor) partitions by columns present in the query
// result, not by expressions computed inline from a timestamp.
func runPartitionFields(md sv2.Metadata) map[string]any {
	ts := ulid.Time(md.ID.RunID.Time())
	return map[string]any{
		"account_id":   md.ID.Tenant.AccountID.String(),
		"workspace_id": md.ID.Tenant.EnvID.String(),
		"year":         ts.Year(),
		"month":        int(ts.Month()),
	}
}

// eventPartitionFields is the events_staging equivalent of
// runPartitionFields, deriving year/month from the event's own internal ID
// ULID timestamp instead of a run ID.
func eventPartitionFields(evt event.TrackedEvent) map[string]any {
	ts := ulid.Time(evt.GetInternalID().Time())
	return map[string]any{
		"account_id":   evt.GetAccountID().String(),
		"workspace_id": evt.GetWorkspaceID().String(),
		"year":         ts.Year(),
		"month":        int(ts.Month()),
	}
}

func (l *listener) OnFunctionScheduled(_ context.Context, md sv2.Metadata, _ queue.Item, _ []event.TrackedEvent) {
	row := map[string]any{
		"run_id":      md.ID.RunID.String(),
		"function_id": md.ID.FunctionID.String(),
		"event_type":  "function_scheduled",
		"created_at":  time.Now().UTC(),
	}
	for k, v := range runPartitionFields(md) {
		row[k] = v
	}
	l.sendRun(row)
}

func (l *listener) OnFunctionStarted(_ context.Context, md sv2.Metadata, _ queue.Item, _ []json.RawMessage) {
	row := map[string]any{
		"run_id":      md.ID.RunID.String(),
		"function_id": md.ID.FunctionID.String(),
		"event_type":  "function_started",
		"created_at":  time.Now().UTC(),
	}
	for k, v := range runPartitionFields(md) {
		row[k] = v
	}
	l.sendRun(row)
}

func (l *listener) OnFunctionFinished(_ context.Context, md sv2.Metadata, _ queue.Item, _ []json.RawMessage, resp statev1.DriverResponse) {
	status := "completed"
	if resp.Err != nil {
		status = "failed"
	}
	row := map[string]any{
		"run_id":      md.ID.RunID.String(),
		"function_id": md.ID.FunctionID.String(),
		"event_type":  "function_finished",
		"status":      status,
		"created_at":  time.Now().UTC(),
	}
	for k, v := range runPartitionFields(md) {
		row[k] = v
	}
	l.sendRun(row)
}

func (l *listener) OnFunctionCancelled(_ context.Context, md sv2.Metadata, _ execution.CancelRequest, _ []json.RawMessage) {
	row := map[string]any{
		"run_id":      md.ID.RunID.String(),
		"function_id": md.ID.FunctionID.String(),
		"event_type":  "function_cancelled",
		"created_at":  time.Now().UTC(),
	}
	for k, v := range runPartitionFields(md) {
		row[k] = v
	}
	l.sendRun(row)
}

func (l *listener) OnStepScheduled(_ context.Context, md sv2.Metadata, _ queue.Item, stepName *string) {
	row := map[string]any{
		"run_id":     md.ID.RunID.String(),
		"event_type": "step_scheduled",
		"created_at": time.Now().UTC(),
	}
	if stepName != nil {
		row["step_name"] = *stepName
	}
	for k, v := range runPartitionFields(md) {
		row[k] = v
	}
	l.sendSpan(row)
}

func (l *listener) OnStepStarted(_ context.Context, md sv2.Metadata, _ queue.Item, _ inngest.Edge, _ string) {
	row := map[string]any{
		"run_id":     md.ID.RunID.String(),
		"event_type": "step_started",
		"created_at": time.Now().UTC(),
	}
	for k, v := range runPartitionFields(md) {
		row[k] = v
	}
	l.sendSpan(row)
}

func (l *listener) OnStepFinished(_ context.Context, md sv2.Metadata, _ queue.Item, _ inngest.Edge, _ *statev1.DriverResponse, stepErr error) {
	row := map[string]any{
		"run_id":     md.ID.RunID.String(),
		"event_type": "step_finished",
		"created_at": time.Now().UTC(),
	}
	if stepErr != nil {
		row["error"] = stepErr.Error()
	}
	for k, v := range runPartitionFields(md) {
		row[k] = v
	}
	l.sendSpan(row)
}

func (l *listener) OnEventReceived(_ context.Context, evt event.TrackedEvent) {
	row := map[string]any{
		"event_id":    evt.GetInternalID().String(),
		"event_name":  evt.GetEvent().Name,
		"occurred_at": time.Now().UTC(),
	}
	for k, v := range eventPartitionFields(evt) {
		row[k] = v
	}
	l.sendEvent(row)
}

// Option configures NewListener.
type Option func(*setupOpts)

type setupOpts struct {
	runsCap, spansCap, eventsCap int
	batchMaxSize                 int
	batchInterval                time.Duration
}

func defaultSetupOpts() setupOpts {
	return setupOpts{
		runsCap:       10_000,
		spansCap:      10_000,
		eventsCap:     10_000,
		batchMaxSize:  10_000,
		batchInterval: 200 * time.Millisecond,
	}
}

// Closer is implemented by the listener NewListener returns. Callers hold
// an execution.SyncLifecycleListener, so reaching this requires a type
// assertion (`l.(dualwrite.Closer)`); it is exported specifically to make
// that assertion possible from outside this package (e.g. pkg/devserver's
// shutdown path) without depending on the unexported listener type.
type Closer interface {
	// Close stops every batcher goroutine started by NewListener — each
	// flushes any rows it's buffered one final time before exiting (see
	// batcher.run's stopc case) — waits for them to exit (bounded by ctx),
	// then closes the db passed to NewListener. Call at most once.
	Close(ctx context.Context) error
}

// NewListener returns an execution.SyncLifecycleListener that dual-writes
// runs/run_spans/events into db, and starts its own background batching
// goroutines (batch.go) that drain the listener's channels and flush into
// the runs_staging/run_spans_staging/events_staging tables. The batching
// goroutines run for the lifetime of the process (context.Background())
// unless the caller stops them via Close (the returned value always
// implements Closer). Compaction of staged rows out to Parquet is out of
// scope for this POC's minimal wiring (descoped by the coordinator; see
// docs/plans/006-duckdb-poc-subprocess-dual-write.md).
func NewListener(db *sql.DB, opts ...Option) execution.SyncLifecycleListener {
	o := defaultSetupOpts()
	for _, apply := range opts {
		apply(&o)
	}

	l := newListenerWithChannels(o.runsCap, o.spansCap, o.eventsCap)
	l.db = db

	tables := map[string]chan map[string]any{
		"runs_staging":      l.runs,
		"run_spans_staging": l.spans,
		"events_staging":    l.events,
	}
	// One shared disabledState across all three batchers, so the driver's
	// terminal duckdb.ErrDisabled state stops the whole dual-write path and
	// is logged exactly once rather than once per table.
	disabled := &disabledState{}
	for table, ch := range tables {
		b := newBatcher(db, table, ch, batcherOpts{maxSize: o.batchMaxSize, flushInterval: o.batchInterval, disabled: disabled})
		l.batchers = append(l.batchers, b)
		l.wg.Add(1)
		go func() {
			defer l.wg.Done()
			b.run(context.Background())
		}()
	}

	return l
}

// Close implements Closer. See the Closer doc comment.
//
// If ctx is bounded (callers should always pass a bounded ctx — see
// pkg/devserver's stopDualWrite caller) and expires before every batcher has
// drained, Close does not wait forever: it falls through and closes db
// anyway. session.exec (pkg/db/duckdb/rows.go) blocks on a synchronous
// stdout read that ignores context cancellation entirely, so a batcher
// genuinely wedged on a dead/hung subprocess cannot be interrupted directly
// — but db.Close() kills the subprocess (process.closeLocked), which closes
// its stdout pipe and unblocks that read with an error. That, in turn, lets
// the wedged batcher's flush return, the batcher goroutine exit, and the
// internal wg-wait goroutine above finish and exit too — so the "leak" in
// the timeout case is self-resolving within roughly db.Close()'s own
// teardown bound (closeLocked's 2s Wait-then-Kill, plus Connector.Close's
// 5s ceiling), not permanent, and it exists only for a process that is
// itself already exiting.
func (l *listener) Close(ctx context.Context) error {
	for _, b := range l.batchers {
		b.stop()
	}

	done := make(chan struct{})
	go func() {
		l.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
	}

	if l.db != nil {
		return l.db.Close()
	}
	return nil
}
