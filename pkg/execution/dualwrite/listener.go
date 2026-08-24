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
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
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

// TODO: use this instead of the map[string]anys in the listener hooks, and have the batcher
type Run struct {
	AccountID  uuid.UUID         `json:"account_id"`
	EnvID      uuid.UUID         `json:"env_id"`
	AppID      uuid.UUID         `json:"app_id"`
	FunctionID uuid.UUID         `json:"function_id"`
	RunID      ulid.ULID         `json:"run_id"`
	QueuedAt   time.Time         `json:"queued_at"`
	StartedAt  time.Time         `json:"started_at"`
	EndedAt    time.Time         `json:"ended_at"`
	Status     enums.RunStatus   `json:"status"`
	Inputs     []json.RawMessage `json:"inputs"`
	Output     json.RawMessage   `json:"output"`
}

func runCommonFields(md sv2.Metadata, evts []json.RawMessage) map[string]any {
	ts := ulid.Time(md.ID.RunID.Time())
	return map[string]any{
		"account_id":  md.ID.Tenant.AccountID,
		"env_id":      md.ID.Tenant.EnvID,
		"run_id":      md.ID.RunID,
		"queued_at":   ts,
		"function_id": md.ID.FunctionID,
		"app_id":      md.ID.Tenant.AppID,
		"inputs":      evts,
	}
}

func runTraceSpanCommonFields(md sv2.Metadata) map[string]any {
	ts := ulid.Time(md.ID.RunID.Time())
	return map[string]any{
		"account_id":    md.ID.Tenant.AccountID,
		"env_id":        md.ID.Tenant.EnvID,
		"run_id":        md.ID.RunID,
		"run_queued_at": ts,
		"function_id":   md.ID.FunctionID,
		"app_id":        md.ID.Tenant.AppID,
	}
}

func (l *listener) OnFunctionScheduled(_ context.Context, md sv2.Metadata, item queue.Item, evts []json.RawMessage) {
	run := runCommonFields(md, evts)
	run["status"] = enums.RunStatusScheduled // TODO: queued here maybe?

	l.sendRun(run)

	span := spanRow(md, "function_scheduled")
	l.sendSpan(span)

	// TODO: span here too maybe?
}

func (l *listener) OnFunctionStarted(_ context.Context, md sv2.Metadata, _ queue.Item, evts []json.RawMessage) {
	row := runCommonFields(md, evts)
	row["status"] = enums.RunStatusRunning
	row["started_at"] = md.Config.StartedAt
	l.sendRun(row)
}

func (l *listener) OnFunctionFinished(_ context.Context, md sv2.Metadata, _ queue.Item, evts []json.RawMessage, resp statev1.DriverResponse) {
	row := runCommonFields(md, evts)
	row["status"] = enums.RunStatusCompleted // TODO: error handling
	row["output"] = resp.Output
	row["started_at"] = md.Config.StartedAt
	row["ended_at"] = time.Now()
	// TODO: real started_at/ended_at
	l.sendRun(row)

	log.Println("dualwrite: OnFunctionFinished resp:", resp)

	// TODO: span here too
}

func (l *listener) OnFunctionCancelled(_ context.Context, md sv2.Metadata, _ execution.CancelRequest, evts []json.RawMessage) {
	row := runCommonFields(md, evts)
	row["status"] = enums.RunStatusCancelled
	// TODO: started_at/ended_at
	l.sendRun(row)

	// TODO: span here too
}

func (l *listener) OnStepStarted(_ context.Context, md sv2.Metadata, _ queue.Item, _ inngest.Edge, _ string) {
	l.sendSpan(spanRow(md, "step_started"))
}

func (l *listener) OnStepFinished(_ context.Context, md sv2.Metadata, _ queue.Item, _ inngest.Edge, _ *statev1.DriverResponse, stepErr error) {
	row := spanRow(md, "step_finished")
	if stepErr != nil {
		row["error"] = stepErr.Error()
	}
	l.sendSpan(row)
}

// spanRow is the shared shape every run_spans_staging row uses: the run it
// belongs to, what happened, when, and the partition columns. Optional
// columns (step_name, error) are only added when there is a value for them,
// exactly as OnStepScheduled/OnStepFinished already do — batcher.insert
// unions the keys across a batch, so a row omitting one still writes NULL for
// it (see batch.go).
func spanRow(md sv2.Metadata, eventType string) map[string]any {
	row := runTraceSpanCommonFields(md)
	return row
}

// The generator-opcode hooks below cover the sleep/wait/invoke half of the
// spec's run_spans hook coverage. Like the step hooks above, each produces one
// append-only row from a single hook call's own data — no correlation between
// a wait and its resume is done in memory; a reader reconstructs the pairing
// from run_id + step_name + created_at at query time.

func (l *listener) OnSleep(_ context.Context, md sv2.Metadata, _ queue.Item, gen statev1.GeneratorOpcode, until time.Time) {
	row := spanRow(md, "sleep")
	if gen.Name != "" {
		row["step_name"] = gen.Name
	}
	l.sendSpan(row)
}

func (l *listener) OnWaitForEvent(_ context.Context, md sv2.Metadata, _ queue.Item, gen statev1.GeneratorOpcode, _ statev1.Pause) {
	row := spanRow(md, "wait_for_event")
	if gen.Name != "" {
		row["step_name"] = gen.Name
	}
	l.sendSpan(row)
}

func (l *listener) OnWaitForEventResumed(_ context.Context, md sv2.Metadata, pause statev1.Pause, r execution.ResumeRequest) {
	l.sendSpan(resumeRow(md, "wait_for_event_resumed", pause, r))
}

func (l *listener) OnWaitForSignal(_ context.Context, md sv2.Metadata, _ queue.Item, gen statev1.GeneratorOpcode, _ statev1.Pause) {
	row := spanRow(md, "wait_for_signal")
	if gen.Name != "" {
		row["step_name"] = gen.Name
	}
	l.sendSpan(row)
}

func (l *listener) OnWaitForSignalResumed(_ context.Context, md sv2.Metadata, pause statev1.Pause, r execution.ResumeRequest) {
	l.sendSpan(resumeRow(md, "wait_for_signal_resumed", pause, r))
}

func (l *listener) OnInvokeFunction(_ context.Context, md sv2.Metadata, _ queue.Item, gen statev1.GeneratorOpcode, _ event.Event) {
	row := spanRow(md, "invoke_function")
	if gen.Name != "" {
		row["step_name"] = gen.Name
	}
	l.sendSpan(row)
}

func (l *listener) OnInvokeFunctionResumed(_ context.Context, md sv2.Metadata, pause statev1.Pause, r execution.ResumeRequest) {
	l.sendSpan(resumeRow(md, "invoke_function_resumed", pause, r))
}

// resumeRow is shared by the three *Resumed hooks, whose signatures and
// available data are identical. A resume carries its step name on either the
// ResumeRequest or the pause, and a timed-out resume is recorded through the
// error column rather than a dedicated event_type, keeping the resume rows
// uniform.
func resumeRow(md sv2.Metadata, eventType string, pause statev1.Pause, r execution.ResumeRequest) map[string]any {
	row := spanRow(md, eventType)
	switch {
	case r.StepName != "":
		row["step_name"] = r.StepName
	case pause.StepName != "":
		row["step_name"] = pause.StepName
	}
	if r.IsTimeout {
		row["error"] = "timeout"
	}
	return row
}

func (l *listener) OnStepGatewayRequestFinished(_ context.Context, md sv2.Metadata, _ queue.Item, _ inngest.Edge, gen statev1.GeneratorOpcode, _ *http.Response, userErr *statev1.UserError) {
	row := spanRow(md, "step_gateway_request_finished")
	if gen.Name != "" {
		row["step_name"] = gen.Name
	}
	if userErr != nil {
		row["error"] = userErr.Message
	}
	l.sendSpan(row)
}

func (l *listener) OnEventReceived(_ context.Context, evt event.TrackedEvent) {
	event := evt.GetEvent()
	eventDataBytes, err := json.Marshal(event.Data)
	if err != nil {
		log.Printf("dualwrite: failed to marshal event data for event %s: %v", event.Name, err)
		return
	}

	eventMetaBytes, err := json.Marshal(event.Meta)
	if err != nil {
		log.Printf("dualwrite: failed to marshal event meta for event %s: %v", event.Name, err)
		return
	}

	internalID := evt.GetInternalID()
	// TODO: verify that a minor semantics change like this is safe.
	// Afaict the internal id & received_at timestamp are created at the same time but not using the same
	// literal timestamp value.
	receivedAt := internalID.Timestamp()

	row := map[string]any{
		"account_id":  evt.GetAccountID(),
		"env_id":      evt.GetWorkspaceID(),
		"internal_id": internalID,
		"received_at": receivedAt,
		// TODO: source/source_id?
		"source":     "",
		"event_id":   event.ID,
		"event_name": event.Name,
		"event_ts":   time.UnixMilli(event.Timestamp),
		"event_data": string(eventDataBytes),
		"event_v":    event.Version,
		"event_meta": string(eventMetaBytes),
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
		"inngest.runs":            l.runs,
		"inngest.run_trace_spans": l.spans,
		"inngest.events":          l.events,
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
