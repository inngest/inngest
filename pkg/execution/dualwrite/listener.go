// Package dualwrite implements the DuckDB-writing
// execution.SyncLifecycleListener for the POC: the executor and runner call
// this listener's hooks synchronously, so every hook body must
// do nothing but build a row and non-blocking-send it onto a per-table
// channel. Separate background goroutines (batch.go) drain those channels
// and flush batches into DuckDB staging tables. Compaction of staged rows out
// to Hive-partitioned Parquet is deliberately out of scope for this POC and
// is not implemented here.
//
// run_spans (inngest.run_trace_spans) are written differently from
// runs/events: the per-step hooks below create real spans through this
// listener's own private tracingv3.TracerProvider (see tracing.go) instead of
// building rows by hand.
package dualwrite

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev1 "github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	tracingv3 "github.com/inngest/inngest/pkg/tracing/v3"
	"github.com/inngest/inngest/pkg/util/interval"
	"github.com/oklog/ulid/v2"
)

// listener implements execution.SyncLifecycleListener. Every hook body does
// nothing but build a row and non-blocking-send it onto the matching
// channel — no I/O, no locking, no flush logic — which is what makes it safe
// to call synchronously from the executor and runner.
type listener struct {
	execution.NoopSyncLifecycleListener

	runs   chan map[string]any
	events chan map[string]any

	droppedRuns   atomic.Int64
	droppedEvents atomic.Int64

	// spanExporter/tp back every per-step hook's span creation — see
	// tracing.go's doc comments. tp is what hooks actually call
	// (l.createSpan(...)); spanExporter is the sdktrace.SpanExporter tp
	// is wired to, kept here only so Close can shut it down.
	spanExporter *SpanExporter
	tp           tracingv3.TracerProvider

	// db and batchers/wg back Close (below) — the shutdown path that stops
	// the background batcher goroutines this listener starts and closes the
	// db handed to NewListener. Task 8 deliberately left this out (see
	// NewListener's doc comment); Task 9 (pkg/devserver) is the first real
	// caller with a shutdown path to drive it.
	db       *sql.DB
	batchers []*batcher
	wg       sync.WaitGroup
}

func newListenerWithChannels(runsCap, eventsCap int) *listener {
	return &listener{
		runs:   make(chan map[string]any, runsCap),
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

func (l *listener) sendEvent(row map[string]any) {
	select {
	case l.events <- row:
	default:
		l.droppedEvents.Add(1)
	}
}

// TODO: use this instead of the map[string]anys in the listener hooks
type Run struct {
	AccountID   uuid.UUID         `json:"account_id"`
	EnvID       uuid.UUID         `json:"env_id"`
	AppID       uuid.UUID         `json:"app_id"`
	FunctionID  uuid.UUID         `json:"function_id"`
	RunID       ulid.ULID         `json:"run_id"`
	QueuedAt    time.Time         `json:"queued_at"`
	ScheduledAt time.Time         `json:"scheduled_at"`
	StartedAt   time.Time         `json:"started_at"`
	EndedAt     time.Time         `json:"ended_at"`
	Status      enums.StepStatus  `json:"status"`
	Inputs      []json.RawMessage `json:"inputs"`
	Output      json.RawMessage   `json:"output"`
}

// runCommonFields' scheduledAt follows the same max(item.At, queuedAt) rule
// opcodeTiming uses (see stepTiming's doc comment) — a job scheduled to run
// immediately has item.At == the queue enqueue time, which can land
// fractionally before queuedAt (the run ID's own embedded timestamp),
// so scheduledAt is never allowed to precede it.
func runCommonFields(md sv2.Metadata, item queue.Item, evts []json.RawMessage) map[string]any {
	ts := ulid.Time(md.ID.RunID.Time())
	scheduledAt := item.At
	if scheduledAt.Before(ts) {
		scheduledAt = ts
	}
	row := map[string]any{
		"account_id": md.ID.Tenant.AccountID,
		"env_id":     md.ID.Tenant.EnvID,
		"run_id":     md.ID.RunID,
		"queued_at":  ts,
		// TODO: make this consistent between the various run status update rows. Maybe store in the state store
		// like StartedAt?
		"scheduled_at": scheduledAt,
		"function_id":  md.ID.FunctionID,
		"app_id":       md.ID.Tenant.AppID,
		"inputs":       evts,
	}
	// md.Config.EventIDs is the run's own persisted trigger event ID list
	// (set once at Schedule time, round-tripped through state on every
	// subsequent load — see pkg/execution/executor/executor.go's Schedule
	// and pkg/execution/state/v2's Config.EventIDs). Omitted entirely for
	// cron-only runs, which have no triggering event at all.
	// TODO: validate that omitting this for crons is valid
	if len(md.Config.EventIDs) > 0 {
		ids := make([]string, len(md.Config.EventIDs))
		for i, id := range md.Config.EventIDs {
			ids[i] = id.String()
		}
		row["event_ids"] = ids
	}
	sessions, isDeferred := scanTriggerEvents(evts)
	if len(sessions) > 0 {
		row["sessions"] = sessions
	}
	if isDeferred {
		row["is_deferred"] = true
	}
	return row
}

// scanTriggerEvents makes one pass over evts (each already this package's
// own json.Marshal(evt) output — see executor.go's Schedule path) to derive
// both of run_common_fields' event-derived columns at once, rather than
// unmarshaling every trigger event twice:
//
//   - sessions mirrors pkg/execution/executor's normalizeRunSessions:
//     collects every triggering event's Meta.Sessions into the run-level
//     pair list (the run-level form of event.EventMeta.Sessions, since a
//     run triggered by multiple events, e.g. a batch, can carry several
//     session pairs — see meta.EventSessions' own doc comment), then
//     dedupes, sorts by (Key, then ID) for determinism across retries
//     (event/batch iteration order isn't stable), and caps at
//     consts.MaxRunSessions.
//   - isDeferred reports whether evts contains an inngest/deferred.schedule
//     trigger event -- i.e. this run was scheduled via defer(), not a
//     normal event/cron trigger. Only the event name is checked, not full
//     DeferredScheduleMetadata validity: is_deferred is a coarse flag, not
//     a source of parent-linkage data (that's deferredScheduleMetadata's
//     job, used by OnFunctionScheduled alone).
//
// An unmarshal failure here would indicate a bug elsewhere, not bad
// external input — a malformed entry is skipped rather than failing the
// whole row.
func scanTriggerEvents(evts []json.RawMessage) (sessions meta.EventSessions, isDeferred bool) {
	for _, raw := range evts {
		var evt event.Event
		if err := json.Unmarshal(raw, &evt); err != nil {
			continue
		}
		for name, id := range evt.Meta.Sessions {
			sessions = append(sessions, meta.EventSession{Key: name, ID: id})
		}
		if evt.Name == consts.FnDeferScheduleName {
			isDeferred = true
		}
	}
	if len(sessions) == 0 {
		return nil, isDeferred
	}

	slices.SortFunc(sessions, func(a, b meta.EventSession) int {
		if c := cmp.Compare(a.Key, b.Key); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
	sessions = slices.Compact(sessions)

	if len(sessions) > consts.MaxRunSessions {
		sessions = sessions[:consts.MaxRunSessions]
	}
	return sessions, isDeferred
}

// addEventsInputAttr sets meta.Attrs.EventsInput to evts marshaled as a
// single JSON array, the same way executor.Schedule's own runSpanOpts does
// (strEvts) for the real run span — so the run.queued/started/ended spans'
// `input` column (see spanExportRow) carries the triggering events, the
// same shape a real reader of EventsInput already expects.
func addEventsInputAttr(ctx context.Context, attrs *meta.SerializableAttrs, evts []json.RawMessage) {
	byt, err := json.Marshal(evts)
	if err != nil {
		logger.StdlibLogger(ctx).Error("dualwrite: failed to marshal events for EventsInput attribute", "error", err)
		return
	}
	str := string(byt)
	meta.AddAttr(attrs, meta.Attrs.EventsInput, &str)
}

// createSpan is nil-safe: l.tp is a tracingv3.TracerProvider interface
// value, nil on a *listener built via newListenerWithChannels directly (see
// listener_test.go) rather than NewListener. Unlike a nil pointer receiver,
// calling a method on a nil interface value panics unconditionally — there
// is no concrete type to dispatch to — so every hook below must go through
// this rather than l.tp.CreateSpan directly; this package's hooks must
// never crash the executor's critical path over that.
func (l *listener) createSpan(ctx context.Context, name string, opts *tracing.CreateSpanOptions) (*meta.SpanReference, error) {
	if l.tp == nil {
		return nil, nil
	}
	if opts.Attributes == nil {
		opts.Attributes = meta.NewAttrSet()
	}
	addTenantAndDebugAttrs(opts.Attributes, opts.Metadata)
	addQueueItemAttrs(opts.Attributes, opts.QueueItem)
	ref, err := l.tp.CreateSpan(ctx, name, opts)
	if err != nil {
		logger.StdlibLogger(ctx).Error("dualwrite: failed to create span", "error", err, "name", name)
	}
	return ref, err
}

func (l *listener) OnFunctionScheduled(ctx context.Context, md sv2.Metadata, item queue.Item, evts []json.RawMessage) {
	run := runCommonFields(md, item, evts)
	run["status"] = enums.StepStatusQueued
	l.sendRun(run)

	// A point span: physical start and end both pinned to queuedAt (needed
	// for GetSpansByRunID's ORDER BY start_time ASC and this span's own
	// deterministic point-in-time identity) — but StartedAt/EndedAt must
	// NOT be derived from that: the run has only been queued at this point,
	// not started or ended. tracingv3.CreateSpan would otherwise
	// auto-populate both from CreateSpanOptions.StartTime/EndTime (via its
	// own meta.AddAttrIfUnset calls), so they're explicitly suppressed here
	// first. meta.AddAttr with a nil *time.Time serializes as
	// meta.BlankAttr (an empty-key, empty-value attribute — a pre-existing
	// "explicitly absent" pattern already built into pkg/tracing/meta's
	// TimeAttr, not a new hack), which is enough to make AddAttrIfUnset see
	// the key as already set and skip its own assignment; confirmed against
	// a real subprocess that the resulting blank key never actually reaches
	// the exported span attributes (the OTel SDK drops it), so there's no
	// stray "" key in the stored attributes JSON.
	deferLinks := deferredScheduleMetadata(ctx, evts)

	queuedAt := ulid.Time(md.ID.RunID.Time())
	attrs := meta.NewAttrSet()
	meta.AddAttr(attrs, meta.Attrs.QueuedAt, &queuedAt)
	meta.AddAttr(attrs, meta.Attrs.StartedAt, (*time.Time)(nil))
	meta.AddAttr(attrs, meta.Attrs.EndedAt, (*time.Time)(nil))
	addEventsInputAttr(ctx, attrs, evts)
	addDeferParentAttrs(attrs, deferLinks)

	mdPtr := safeMetadata(md)
	_, _ = l.createSpan(ctx, tracingv3.SpanNameRunQueued, &tracing.CreateSpanOptions{
		Metadata:   mdPtr,
		QueueItem:  &item,
		Parent:     tracing.RunSpanRefFromMetadata(mdPtr),
		Attributes: attrs,
		StartTime:  queuedAt,
		EndTime:    queuedAt,
	})

	l.updateParentDeferSpans(ctx, md, queuedAt, deferLinks)
}

// deferredScheduleMetadata extracts every valid inngest/deferred.schedule
// trigger event's DeferredScheduleMetadata from evts, skipping (and
// logging) any that fail to parse or validate -- mirrors
// executor.updateDeferSpans' own tolerance for malformed/invalid entries, so
// one bad entry never blocks scheduling the run itself.
func deferredScheduleMetadata(ctx context.Context, evts []json.RawMessage) []*event.DeferredScheduleMetadata {
	var out []*event.DeferredScheduleMetadata
	for _, raw := range evts {
		var evt event.Event
		if err := json.Unmarshal(raw, &evt); err != nil {
			continue
		}
		if evt.Name != consts.FnDeferScheduleName {
			continue
		}
		m, err := evt.DeferredScheduleMetadata()
		if err != nil {
			logger.StdlibLogger(ctx).Error("dualwrite: malformed deferred schedule metadata", "error", err)
			continue
		}
		if err := m.Validate(); err != nil {
			logger.StdlibLogger(ctx).Error("dualwrite: invalid deferred schedule metadata", "error", err)
			continue
		}
		out = append(out, m)
	}
	return out
}

// addDeferParentAttrs stamps the child run's own DeferParentRunIDs/
// DeferParentFnSlug attrs, mirroring executor.updateDeferSpans' identical
// (and identically odd -- see its own comment) "one fn slug for possibly
// many parent run IDs" shape.
func addDeferParentAttrs(attrs *meta.SerializableAttrs, links []*event.DeferredScheduleMetadata) {
	if len(links) == 0 {
		return
	}
	var parentRunIDs []string
	var parentFnSlug string
	for _, m := range links {
		parentRunIDs = append(parentRunIDs, m.ParentRunID.String())
		parentFnSlug = m.ParentFnSlug
	}
	meta.AddAttr(attrs, meta.Attrs.DeferParentRunIDs, &parentRunIDs)
	meta.AddAttr(attrs, meta.Attrs.DeferParentFnSlug, &parentFnSlug)
}

// updateParentDeferSpans re-emits each linked defer's executor.defer span
// row for the PARENT run, now with DeferChildRunID stamped onto it --
// mirrors executor.updateDeferSpans' UpdateSpan call, which this package
// can't reuse (run_trace_spans is append-only, see this file's own package
// doc). Reuses the original's deterministic span_id
// (tracing.DeferSpanSeed(parentRunID, hashedID)) rather than a fresh one, so
// duckdbquery's read side can collapse both rows to one logical defer by
// (run_id, span_id); it picks the most-advanced row, so this one need only
// carry the fields known here, not everything the original DeferAdd row
// had (e.g. UserlandID is never available past the original SDK request,
// and is simply absent from this row -- a documented, bounded gap).
// queuedAt is the child run's own queued time (this hook has no `now` of
// its own -- see OnFunctionScheduled's caller-supplied-timestamp
// convention on every hook that does).
func (l *listener) updateParentDeferSpans(ctx context.Context, childMD sv2.Metadata, queuedAt time.Time, links []*event.DeferredScheduleMetadata) {
	for _, m := range links {
		if m.HashedID == "" {
			// Pre-existing event from before HashedID was added to
			// DeferredScheduleMetadata (see pkg/event/defer.go) -- without
			// it, DeferSpanSeed here would compute a different span_id than
			// the original DeferAdd row, landing as an orphan rather than
			// collapsing with it. Narrow window (only events already
			// in-flight across this exact deploy), not worth writing a
			// wrong row over.
			logger.StdlibLogger(ctx).Warn(
				"dualwrite: deferred schedule metadata missing hashed ID, skipping parent defer span update",
				"parent_run_id", m.ParentRunID.String(),
			)
			continue
		}

		parentMD := safeMetadata(sv2.Metadata{
			ID: sv2.ID{
				RunID:      m.ParentRunID,
				FunctionID: m.ParentFnID,
				Tenant: sv2.Tenant{
					AccountID: childMD.ID.Tenant.AccountID,
					EnvID:     childMD.ID.Tenant.EnvID,
					AppID:     m.ParentAppID,
				},
			},
		})
		childRunID := childMD.ID.RunID
		status := enums.DeferStatusAfterRun
		_, _ = l.createSpan(ctx, meta.SpanNameDefer, &tracing.CreateSpanOptions{
			Metadata:  parentMD,
			Parent:    tracing.RunSpanRefFromMetadata(parentMD),
			Seed:      tracing.DeferSpanSeed(m.ParentRunID, m.HashedID),
			StartTime: queuedAt,
			EndTime:   queuedAt,
			Attributes: meta.NewAttrSet(
				meta.Attr(meta.Attrs.DeferHashedID, &m.HashedID),
				meta.Attr(meta.Attrs.DeferFnSlug, &m.FnSlug),
				meta.Attr(meta.Attrs.DeferStatus, &status),
				meta.Attr(meta.Attrs.DeferChildRunID, &childRunID),
			),
		})
	}
}

func (l *listener) OnFunctionStarted(ctx context.Context, md sv2.Metadata, item queue.Item, evts []json.RawMessage) {
	row := runCommonFields(md, item, evts)
	row["status"] = enums.StepStatusRunning
	row["started_at"] = md.Config.StartedAt
	l.sendRun(row)

	// Not a point span: physical start is queuedAt, physical end is
	// md.Config.StartedAt, so the span's own duration reflects the time
	// this run actually spent queued before it started.
	queuedAt := ulid.Time(md.ID.RunID.Time())
	attrs := meta.NewAttrSet()
	if !md.Config.StartedAt.IsZero() {
		meta.AddAttr(attrs, meta.Attrs.StartedAt, &md.Config.StartedAt)
	}
	addEventsInputAttr(ctx, attrs, evts)

	mdPtr := safeMetadata(md)
	_, _ = l.createSpan(ctx, tracingv3.SpanNameRunStarted, &tracing.CreateSpanOptions{
		Metadata:   mdPtr,
		QueueItem:  &item,
		Parent:     tracing.RunSpanRefFromMetadata(mdPtr),
		Attributes: attrs,
		StartTime:  queuedAt,
		EndTime:    md.Config.StartedAt,
	})
}

// runFinishedStatus derives the run's finished status from resp, the same
// way the real system's non-step completion span does
// (executor.emitNonStepSpan's callers in executor.HandleResponse): by the
// time OnFunctionFinished fires, a retryable resp.Err has already looped
// back through OnStepScheduled instead of reaching here, so the only two
// outcomes left to distinguish are these.
func runFinishedStatus(resp statev1.DriverResponse) enums.StepStatus {
	if resp.Err != nil {
		return enums.StepStatusFailed
	}
	return enums.StepStatusCompleted
}

func (l *listener) OnFunctionFinished(ctx context.Context, md sv2.Metadata, item queue.Item, evts []json.RawMessage, resp statev1.DriverResponse, now time.Time) {
	stepStatus := runFinishedStatus(resp)

	fnOutput, err := resp.GetTraceFunctionOutput()
	if err != nil {
		logger.StdlibLogger(ctx).Error("dualwrite: OnFunctionFinished failed to get function output", "error", err)
	}

	queuedAt := ulid.Time(md.ID.RunID.Time())
	start := md.Config.StartedAt
	if start.IsZero() {
		start = queuedAt
	}
	end := now
	mdPtr := safeMetadata(md)

	// Two spans, matching the real production topology: a stable root
	// "executor.run" (created once per run, Seed=md.ID.RunID[:] — the same
	// identity tracing.RunSpanRefFromMetadata already computes and every
	// other hook in this file already parents its own spans under, but
	// which nothing had actually inserted a row for until now), and a
	// child nonstep span for the function's own output event — see
	// emitOnFunctionFinishedNonStepSpan below.
	// QueuedAt is the run's own canonical queued_at (the ULID-embedded
	// timestamp every other hook in this file uses — runCommonFields,
	// OnFunctionScheduled, OnFunctionStarted), not item.EnqueuedAt: item
	// here is whichever queue.Item happened to trigger this specific finish
	// call, which for a multi-step function is a later step's item, not the
	// run's original enqueue.
	//
	// ScheduledAt is deliberately omitted (not derived from item.At): item
	// is that same later step's queue item here, not the run's original
	// scheduling item, so item.At is simply the wrong value, not just one
	// needing a clamp — the run's real scheduled_at is never plumbed
	// through the state store to this hook. TODO: thread the run's actual
	// scheduled_at through sv2.Metadata (or similar) so it's available here
	// and at OnFunctionScheduled without relying on whichever item happens
	// to be in scope.
	runAttrs := meta.NewAttrSet()
	meta.AddAttr(runAttrs, meta.Attrs.DynamicStatus, &stepStatus)
	addEventsInputAttr(ctx, runAttrs, evts)
	tracing.AddTimingAttrs(runAttrs, queuedAt, time.Time{}, start, end)
	addRunSpanAttrs(runAttrs, mdPtr)
	_, _ = l.createSpan(ctx, meta.SpanNameRun, &tracing.CreateSpanOptions{
		Seed:       md.ID.RunID[:],
		Metadata:   mdPtr,
		QueueItem:  &item,
		StartTime:  queuedAt,
		EndTime:    end,
		Attributes: runAttrs,
	})

	// Create the nonstep span before the run list entry below, so nothing
	// reading inngest.runs ever observes a finished run before its
	// corresponding trace data exists.
	l.emitOnFunctionFinishedNonStepSpan(ctx, mdPtr, item, resp, stepStatus, start, end)

	// status/output derived the same way as the spans above, from the same
	// stepStatus/fnOutput — see runFinishedStatus.
	row := runCommonFields(md, item, evts)
	row["status"] = stepStatus
	if fnOutput != "" {
		row["output"] = fnOutput
	}
	row["started_at"] = md.Config.StartedAt
	row["ended_at"] = end
	l.sendRun(row)
}

// emitOnFunctionFinishedNonStepSpan creates the span for the function's own
// output event, a child of the root "executor.run" span (see
// OnFunctionFinished above) — the same role executor.emitNonStepSpan's span
// plays in the real production pipeline, and the same Seed
// (tracing.NonStepDynamicSeed(item)) it uses, so a reader can correlate the
// two by identity. tracingv3.SpanNameError/SpanNameFinal split by outcome,
// rather than the real system's single meta.SpanNameNonStep name for both
// (kept as-is for executor.emitNonStepSpan itself).
func (l *listener) emitOnFunctionFinishedNonStepSpan(ctx context.Context, mdPtr *sv2.Metadata, item queue.Item, resp statev1.DriverResponse, status enums.StepStatus, start, end time.Time) {
	// tracing.DriverResponseOutputAttrs is the exact same builder
	// executor.emitNonStepSpan uses (IsFunctionOutput/StepOutput/Retryable/
	// ResponseOutputSize) — this span was never dynamic, same as the real
	// one, so its attribute set should match exactly.
	attrs := tracing.DriverResponseOutputAttrs(&resp)
	meta.AddAttr(attrs, meta.Attrs.DynamicStatus, &status)
	if resp.Err != nil {
		attrs.AddErr(errors.New(*resp.Err))
	}
	tracing.AddTimingAttrs(attrs, item.EnqueuedAt, item.At, start, end)

	spanName := tracingv3.SpanNameFinal
	if status == enums.StepStatusFailed || status == enums.StepStatusErrored {
		spanName = tracingv3.SpanNameError
	}

	_, _ = l.createSpan(ctx, spanName, &tracing.CreateSpanOptions{
		Seed:       tracing.NonStepDynamicSeed(item),
		Metadata:   mdPtr,
		QueueItem:  &item,
		Parent:     tracing.RunSpanRefFromMetadata(mdPtr),
		StartTime:  start,
		EndTime:    end,
		Attributes: attrs,
	})
}

func (l *listener) OnFunctionCancelled(_ context.Context, md sv2.Metadata, _ execution.CancelRequest, evts []json.RawMessage, now time.Time) {
	row := runCommonFields(md, queue.Item{}, evts)
	row["status"] = enums.StepStatusCancelled
	if !md.Config.StartedAt.IsZero() {
		row["started_at"] = md.Config.StartedAt
	}
	row["ended_at"] = now
	l.sendRun(row)
}

// OnStepScheduled creates a point-in-time marker span (tracingv3.SpanNameStepPlanned)
// for the moment a step is scheduled/planned — a distinct span kind from the
// real "executor.step" span, with its own random span_id rather than
// FinalizedStepDynamicSeed(gen.ID): this hook receives no GeneratorOpcode to
// derive that seed from (only a step name), and even if it did, reusing the
// eventual finished step span's identity here would collide with it once
// that real span is inserted — this package only ever inserts, never
// updates in place (see SpanExporter's doc comment).
func (l *listener) OnStepScheduled(ctx context.Context, md sv2.Metadata, item queue.Item, stepName *string, now time.Time) {
	attrs := meta.NewAttrSet()
	if stepName != nil {
		meta.AddAttr(attrs, meta.Attrs.StepName, stepName)
	}

	mdPtr := safeMetadata(md)
	_, _ = l.createSpan(ctx, tracingv3.SpanNameStepPlanned, &tracing.CreateSpanOptions{
		Metadata:   mdPtr,
		QueueItem:  &item,
		Parent:     tracing.RunSpanRefFromMetadata(mdPtr),
		Attributes: attrs,
		StartTime:  now,
		EndTime:    now,
	})
}

func (l *listener) OnEventReceived(ctx context.Context, evt event.TrackedEvent) {
	event := evt.GetEvent()
	eventDataBytes, err := json.Marshal(event.Data)
	if err != nil {
		logger.StdlibLogger(ctx).Error("dualwrite: failed to marshal event data", "event", event.Name, "error", err)
		return
	}

	eventMetaBytes, err := json.Marshal(event.Meta)
	if err != nil {
		logger.StdlibLogger(ctx).Error("dualwrite: failed to marshal event meta", "event", event.Name, "error", err)
		return
	}

	internalID := evt.GetInternalID()
	// NOTE: Use internalID.Timestamp() instead of time.Now() for ordering simplicity in queries
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

// OnDeferAdd writes the run's own executor.defer span (a new row, seeded
// deterministically from (run_id, hashedID) via tracing.DeferSpanSeed --
// see pkg/execution/defers.createDeferSpan, which does the same for the
// real system). now is d's own accept/reject moment, not necessarily
// "now" by the time this fires -- see the interface doc comment.
func (l *listener) OnDeferAdd(ctx context.Context, md sv2.Metadata, d sv2.Defer, userlandID string, now time.Time) {
	mdPtr := safeMetadata(md)
	_, _ = l.createSpan(ctx, meta.SpanNameDefer, &tracing.CreateSpanOptions{
		Metadata:  mdPtr,
		Parent:    tracing.RunSpanRefFromMetadata(mdPtr),
		Seed:      tracing.DeferSpanSeed(md.ID.RunID, d.HashedID),
		StartTime: now,
		EndTime:   now,
		Attributes: meta.NewAttrSet(
			meta.Attr(meta.Attrs.DeferHashedID, &d.HashedID),
			meta.Attr(meta.Attrs.DeferUserlandID, &userlandID),
			meta.Attr(meta.Attrs.DeferFnSlug, &d.FnSlug),
			meta.Attr(meta.Attrs.DeferStatus, &d.ScheduleStatus),
		),
	})
}

// OnDeferAbort re-emits the defer's executor.defer span (same deterministic
// span_id as OnDeferAdd's own write -- see its doc comment) with
// status=Aborted. fn_slug/userland_id aren't known at abort time (see
// pkg/execution/defers.AbortFromOp, which doesn't have them either) and are
// simply absent from this row; duckdbquery's read side collapses to the
// most-advanced row per (run_id, span_id), so this is the terminal state
// for this defer regardless.
func (l *listener) OnDeferAbort(ctx context.Context, md sv2.Metadata, hashedID string, now time.Time) {
	mdPtr := safeMetadata(md)
	status := enums.DeferStatusAborted
	_, _ = l.createSpan(ctx, meta.SpanNameDefer, &tracing.CreateSpanOptions{
		Metadata:  mdPtr,
		Parent:    tracing.RunSpanRefFromMetadata(mdPtr),
		Seed:      tracing.DeferSpanSeed(md.ID.RunID, hashedID),
		StartTime: now,
		EndTime:   now,
		Attributes: meta.NewAttrSet(
			meta.Attr(meta.Attrs.DeferHashedID, &hashedID),
			meta.Attr(meta.Attrs.DeferStatus, &status),
		),
	})
}

// The per-step hooks below create real spans through l.tp (this listener's
// private tracingv3.TracerProvider — see tracing.go), using the same Seed
// values the real system's own CreateSpan calls use for that step
// (executor.emitStepSpan/OnSleep) — not a fabricated or random identity —
// so a span here can line up with the real one by span_id/trace_id.
//
// The three pause-backed opcodes (wait-for-event, wait-for-signal, invoke)
// each create two separate spans rather than one: a point-in-time marker
// with a random span_id when the pause begins, and a second span
// encompassing the pause's full duration (both dated from the pause's own
// CreatedAt — the first through to itself, the second through to resume),
// with the deterministic span_id.
//
// OnStepFinished's real counterpart (tracingv3.SpanNameExecution, created in
// executor.Execute's CreateSpan call with no Seed at all) is a genuinely
// random, per-attempt span with no deterministic identity to reconstruct
// from a hook's arguments — and OnStepFinished itself fires once per SDK
// request, which may cover several opcodes (or none) rather than one step,
// so there's no single step identity to key off of either. Its span (see
// OnStepFinished below) therefore gets the same real name
// (tracingv3.SpanNameExecution) but a random span_id, the same as the
// OnFunctionScheduled/OnFunctionStarted markers above.
//
// OnStepStarted is still NOT implemented: it carries even less than
// OnStepFinished (no GeneratorOpcode, no DriverResponse — just a URL
// string), so there's nothing meaningful to put on a span at all.
// execution.NoopSyncLifecycleListener covers it as a no-op.

// stepTiming replicates executor.opcodeTiming's fallback logic (see
// executor.go): gen.Timing is only populated by newer SDKs that report
// per-opcode timing, so gen.Timing.Start()/End() can come back zero (or,
// for a replayed/backfilled opcode, before the step was even queued) —
// opcodeTiming falls back to runCtx.StartTime()/e.now() in that case; this
// package has no RunContext, so it falls back to the run's own
// md.Config.StartedAt/now instead, the closest equivalents it has access
// to. now is always the caller's own timestamp (see the SyncLifecycleListener
// hooks' own doc comments) — this package never calls time.Now() itself.
func stepTiming(item queue.Item, md sv2.Metadata, gen *statev1.GeneratorOpcode, now time.Time) (queuedAt, scheduledAt, startedAt, endedAt time.Time) {
	queuedAt = item.EnqueuedAt
	scheduledAt = item.At
	if scheduledAt.Before(queuedAt) {
		scheduledAt = queuedAt
	}

	// interval.Interval{}.Start() decodes an unset Timing as the Unix
	// epoch, not time.Time's own zero value, so an explicit zero-value
	// check here (rather than relying on startedAt.IsZero() below) is
	// required to actually detect "this SDK never sent Timing" — otherwise
	// the fallback below never triggers and every untimed opcode gets a
	// span pinned to 1970.
	if gen != nil && gen.Timing != (interval.Interval{}) {
		startedAt = gen.Timing.Start()
		endedAt = gen.Timing.End()
	}

	if startedAt.IsZero() || startedAt.Before(queuedAt) {
		startedAt = md.Config.StartedAt
	}
	if endedAt.IsZero() || endedAt.Before(startedAt) {
		endedAt = now
	}

	return queuedAt, scheduledAt, startedAt, endedAt
}

// stepAttrs builds the attribute set for a real "executor.step" span
// (OnSleep/OnStepRunFinished/OnStepGatewayRequestFinished — never dynamic,
// same as the real spans they mirror, so their attribute set should match
// exactly): tracing.GeneratorAttrs(gen) is the exact same builder
// executor.emitStepSpan/handleGeneratorSleep use (covering StepID/StepOp/
// StepName/StepInput/StepOutput/opcode-specific fields — see
// pkg/tracing/util.go's generatorAttrs), plus the same timing attrs
// tracing.AddTimingAttrs adds there. Tenant attrs need no separate call:
// pkg/tracing's executionProcessor.OnStart adds those automatically for
// any span with Metadata set, which every span this package creates has.
// Returns startedAt/endedAt too, so callers can use the exact same
// timestamps for the span's own physical start/end.
func stepAttrs(gen *statev1.GeneratorOpcode, item queue.Item, md sv2.Metadata, now time.Time) (attrs *meta.SerializableAttrs, startedAt, endedAt time.Time) {
	attrs = tracing.GeneratorAttrs(gen)
	queuedAt, scheduledAt, startedAt, endedAt := stepTiming(item, md, gen, now)
	tracing.AddTimingAttrs(attrs, queuedAt, scheduledAt, startedAt, endedAt)
	return attrs, startedAt, endedAt
}

func (l *listener) OnSleep(ctx context.Context, md sv2.Metadata, item queue.Item, gen statev1.GeneratorOpcode, _ time.Time, now time.Time) {
	// tracing.GeneratorAttrs is the exact same attribute builder
	// executor.emitStepSpan/handleGeneratorSleep use — it already sets
	// StepSleepDuration from the step's actual configured duration
	// (op.SleepDuration()), which is more accurate than anything derivable
	// from this hook's own "until" argument, so there's nothing to add on
	// top of it here.
	attrs, startedAt, endedAt := stepAttrs(&gen, item, md, now)

	mdPtr := safeMetadata(md)
	_, _ = l.createSpan(ctx, tracingv3.SpanNameStep, &tracing.CreateSpanOptions{
		Seed:       tracing.SleepStepDynamicSeed(gen.ID),
		Metadata:   mdPtr,
		QueueItem:  &item,
		Parent:     tracing.RunSpanRefFromMetadata(mdPtr),
		Attributes: attrs,
		StartTime:  startedAt,
		EndTime:    endedAt,
	})
}

func (l *listener) OnWaitForEvent(ctx context.Context, md sv2.Metadata, item queue.Item, gen statev1.GeneratorOpcode, pause statev1.Pause) {
	// tracing.GeneratorAttrs (not a bespoke StepName/StepOp-only builder) —
	// it's the exact same opcode-specific attribute builder OnSleep uses via
	// stepAttrs, and OpcodeWaitForEvent is one of the cases it already
	// handles (StepWaitForEventName/If, StepWaitExpiry), so this pause's own
	// GeneratorOpcode already carries everything needed for
	// convertFlatSpanToGQL's WaitForEventStepInfo to render correctly.
	l.createPauseStartedSpan(ctx, md, item, tracing.GeneratorAttrs(&gen), pause)
}

func (l *listener) OnWaitForEventResumed(ctx context.Context, md sv2.Metadata, pause statev1.Pause, r execution.ResumeRequest, now time.Time) {
	l.createPauseSpan(ctx, md, pause, r, now)
}

func (l *listener) OnWaitForSignal(ctx context.Context, md sv2.Metadata, item queue.Item, gen statev1.GeneratorOpcode, pause statev1.Pause) {
	l.createPauseStartedSpan(ctx, md, item, tracing.GeneratorAttrs(&gen), pause)
}

func (l *listener) OnWaitForSignalResumed(ctx context.Context, md sv2.Metadata, pause statev1.Pause, r execution.ResumeRequest, now time.Time) {
	l.createPauseSpan(ctx, md, pause, r, now)
}

func (l *listener) OnInvokeFunction(ctx context.Context, md sv2.Metadata, item queue.Item, gen statev1.GeneratorOpcode, _ event.Event) {
	l.createPauseStartedSpan(ctx, md, item, tracing.GeneratorAttrs(&gen), statev1.Pause{})
}

func (l *listener) OnInvokeFunctionResumed(ctx context.Context, md sv2.Metadata, pause statev1.Pause, r execution.ResumeRequest, now time.Time) {
	l.createPauseSpan(ctx, md, pause, r, now)
}

// createPauseStartedSpan creates the point-in-time marker span for the
// moment a pause begins — see the doc comment above OnSleep. Dated to the
// pause's own CreatedAt rather than "now" at hook-call time, the same
// timestamp createPauseSpan uses as its start. OnInvokeFunction doesn't
// receive a Pause at all, so it passes a zero value; a zero CreatedAt
// (also possible on a real Pause — see its doc comment) leaves StartTime
// zero too, which CreateSpanOptions already defaults to "now".
func (l *listener) createPauseStartedSpan(ctx context.Context, md sv2.Metadata, item queue.Item, attrs *meta.SerializableAttrs, pause statev1.Pause) {
	// ScheduledAt follows the same max(item.At, pinned-timestamp) rule
	// runCommonFields/stepTiming use elsewhere in this file — a job
	// scheduled to run immediately can have item.At land fractionally
	// before the pause's own CreatedAt, so scheduledAt is never allowed to
	// precede it.
	scheduledAt := item.At
	if scheduledAt.Before(pause.CreatedAt) {
		scheduledAt = pause.CreatedAt
	}
	if !scheduledAt.IsZero() {
		meta.AddAttr(attrs, meta.Attrs.ScheduledAt, &scheduledAt)
	}

	mdPtr := safeMetadata(md)
	_, _ = l.createSpan(ctx, tracingv3.SpanNameStepPauseStarted, &tracing.CreateSpanOptions{
		Metadata:   mdPtr,
		QueueItem:  &item,
		Parent:     tracing.RunSpanRefFromMetadata(mdPtr),
		Attributes: attrs,
		StartTime:  pause.CreatedAt,
	})
}

// createPauseSpan creates the span encompassing a pause's full duration —
// called at resume time, once both boundaries are known: pause.CreatedAt
// (since these hooks never see the original call that created the pause)
// through to now, the resume time (CreateSpan always ends a span "now",
// when called). Its Seed is the same tracing.FinalizedStepDynamicSeed the
// step's eventual real finished span would also use, so a reader can
// correlate this span to that one by identity even though this one was
// never updated in place the way the real system's is — see
// SpanExporter's doc comment on this package only ever inserting.
func (l *listener) createPauseSpan(ctx context.Context, md sv2.Metadata, pause statev1.Pause, r execution.ResumeRequest, now time.Time) {
	attrs := resumeAttrs(pause, r)
	// A timeout is an expected outcome (e.g. a wait-for-event's timeout
	// branch), not a failure — StepStatusTimedOut says so directly, the
	// same way emitStepSpan's own DynamicStatus does for every other step
	// span, rather than layering an "error" attribute onto a Completed
	// status that would otherwise default to implying success.
	status := enums.StepStatusCompleted
	if r.IsTimeout {
		status = enums.StepStatusTimedOut
	}
	meta.AddAttr(attrs, meta.Attrs.DynamicStatus, &status)

	// QueuedAt/ScheduledAt: these hooks receive no queue.Item (unlike
	// createPauseStartedSpan), so pause.CreatedAt — the only timestamp
	// otherwise available — is the closest equivalent to "when this step
	// was queued/scheduled", the same "closest available timestamp"
	// fallback stepTiming/OnFunctionScheduled use elsewhere in this file.
	// StartedAt is deliberately left zero here (AddTimingAttrs skips zero
	// values): CreateSpanOptions.StartTime below already gets it set
	// automatically (see tracingv3's CreateSpan), so setting it again would
	// be redundant. now is the caller's own resume-time timestamp — the
	// same instant executor.Resume/ResumePauseTimeout use for their own
	// UpdateSpan(EndTime: ...) call on the real (non-dualwrite) span, kept
	// consistent by threading it through the SyncLifecycleListener
	// interface rather than this package reading a fresh time.Now() of its
	// own — and must also be passed as CreateSpanOptions.EndTime: that is
	// what makes the tracer add the EndedAt attribute, not merely end the
	// physical span at that instant.
	tracing.AddTimingAttrs(attrs, pause.CreatedAt, pause.CreatedAt, time.Time{}, now)

	mdPtr := safeMetadata(md)
	_, _ = l.createSpan(ctx, tracingv3.SpanNameStep, &tracing.CreateSpanOptions{
		Seed:       tracing.FinalizedStepDynamicSeed(pause.Outgoing),
		Metadata:   mdPtr,
		Parent:     tracing.RunSpanRefFromMetadata(mdPtr),
		StartTime:  pause.CreatedAt,
		EndTime:    now,
		Attributes: attrs,
	})
}

// resumeAttrs is shared by the three *Resumed hooks above, whose signatures
// and available data are identical: a resume never receives a
// GeneratorOpcode the way createPauseStartedSpan's tracing.GeneratorAttrs
// call does, but StepID/StepOp are still fully recoverable from pause alone
// — StepID from pause.Outgoing (the step's own ID), StepOp from whichever
// of pause.IsWaitForEvent()/IsInvoke()/IsSignal() matches (the exact opcode
// the pause was created for, not merely inferred).
//
// The real system's own resume-time attribute builder, tracing.ResumeAttrs,
// deliberately does NOT re-set StepWaitForEventName/If/StepSignalName/
// StepInvokeFunctionID/StepInvokeTriggerEventID here — its rollup/
// fragment-merge model inherits those from the pause-started span's own
// fragment when the two get merged into one logical span at read time (see
// pkg/cqrs/manager's dynamic-span-id merge). This package's flat model has
// no merge step at all — createPauseStartedSpan and createPauseSpan are two
// permanently separate physical rows — so every opcode-specific field a
// reader might need has to be set explicitly here too, derived straight
// from pause's own fields.
//
// pause.Event/Expression/TriggeringEventID are reused across pause types
// with different meanings depending on which opcode created the pause (an
// invoke pause's Event/Expression hold its own internal
// function-finished-matching plumbing, not a user-facing wait condition;
// see handleGeneratorInvokeFunction in pkg/execution/executor/executor.go),
// so each field below is gated by the specific opcode that gives it its
// documented meaning — mirroring tracing.ResumeAttrs' own
// pause.IsInvoke()/IsWaitForEvent() gating for the fields it does set
// (StepInvokeFinishEventID/StepInvokeRunID/StepWaitForEventMatchedID from
// ResumeRequest.EventID/RunID, reused verbatim below).
func resumeAttrs(pause statev1.Pause, r execution.ResumeRequest) *meta.SerializableAttrs {
	attrs := meta.NewAttrSet()
	name := r.StepName
	if name == "" {
		name = pause.StepName
	}
	if name != "" {
		meta.AddAttr(attrs, meta.Attrs.StepName, &name)
	}
	// pause.Outgoing is the step's own ID — the same value
	// tracing.FinalizedStepDynamicSeed(pause.Outgoing) below derives this
	// span's deterministic identity from, and the same field
	// tracing.GeneratorAttrs sets from gen.ID at pause-started time (see
	// createPauseStartedSpan) — set unconditionally, unlike the
	// opcode-specific fields below, since every pause type carries it.
	// pause.Outgoing is the step's own ID — the same value
	// tracing.FinalizedStepDynamicSeed(pause.Outgoing) below derives this
	// span's deterministic identity from, and the same field
	// tracing.GeneratorAttrs sets from gen.ID at pause-started time (see
	// createPauseStartedSpan) — set unconditionally, unlike the
	// opcode-specific fields below, since every pause type carries it.
	if pause.Outgoing != "" {
		meta.AddAttr(attrs, meta.Attrs.StepID, &pause.Outgoing)
	}

	switch {
	case pause.IsWaitForEvent():
		op := enums.OpcodeWaitForEvent
		meta.AddAttr(attrs, meta.Attrs.StepOp, &op)
		if pause.Event != nil {
			meta.AddAttr(attrs, meta.Attrs.StepWaitForEventName, pause.Event)
		}
		if pause.Expression != nil {
			meta.AddAttr(attrs, meta.Attrs.StepWaitForEventIf, pause.Expression)
		}
		if r.EventID != nil {
			meta.AddAttr(attrs, meta.Attrs.StepWaitForEventMatchedID, r.EventID)
		}
	case pause.IsInvoke():
		op := enums.OpcodeInvokeFunction
		meta.AddAttr(attrs, meta.Attrs.StepOp, &op)
		if pause.InvokeTargetFnID != nil {
			meta.AddAttr(attrs, meta.Attrs.StepInvokeFunctionID, pause.InvokeTargetFnID)
		}
		// TriggeringEventID, despite its generic doc comment ("the event
		// that triggered the original run"), is set to the invocation
		// event's own ID for an invoke pause specifically — see
		// handleGeneratorInvokeFunction's `TriggeringEventID: &evt.Event.ID`
		// (evt being the synthetic invocation event created to trigger the
		// target function), the same event tracing.GeneratorAttrs' Invoke
		// case reads via opts.Payload.ID at pause-started time.
		if pause.TriggeringEventID != nil {
			if id, err := ulid.Parse(*pause.TriggeringEventID); err == nil {
				meta.AddAttr(attrs, meta.Attrs.StepInvokeTriggerEventID, &id)
			}
		}
		if r.EventID != nil {
			meta.AddAttr(attrs, meta.Attrs.StepInvokeFinishEventID, r.EventID)
		}
		if r.RunID != nil {
			meta.AddAttr(attrs, meta.Attrs.StepInvokeRunID, r.RunID)
		}
	case pause.IsSignal():
		op := enums.OpcodeWaitForSignal
		meta.AddAttr(attrs, meta.Attrs.StepOp, &op)
		if pause.SignalID != nil {
			meta.AddAttr(attrs, meta.Attrs.StepSignalName, pause.SignalID)
		}
	}

	if expiry := time.Time(pause.Expires); !expiry.IsZero() {
		meta.AddAttr(attrs, meta.Attrs.StepWaitExpiry, &expiry)
	}
	meta.AddAttr(attrs, meta.Attrs.StepWaitExpired, &r.IsTimeout)

	return attrs
}

func (l *listener) OnStepGatewayRequestFinished(ctx context.Context, md sv2.Metadata, item queue.Item, _ inngest.Edge, gen statev1.GeneratorOpcode, _ *http.Response, userErr *statev1.UserError, now time.Time) {
	attrs, startedAt, endedAt := stepAttrs(&gen, item, md, now)
	// Approximates emitStepSpan's real switch (Errored if retryable, else
	// Failed) as a simple two-way split: retryability isn't derivable from
	// this hook's arguments alone (no runCtx/attempt count), only whether
	// the gateway request itself errored.
	status := enums.StepStatusCompleted
	if userErr != nil {
		status = enums.StepStatusFailed
		attrs.AddErr(errors.New(userErr.Message))
	}
	meta.AddAttr(attrs, meta.Attrs.DynamicStatus, &status)

	mdPtr := safeMetadata(md)
	_, _ = l.createSpan(ctx, tracingv3.SpanNameStep, &tracing.CreateSpanOptions{
		Seed:       tracing.FinalizedStepDynamicSeed(gen.ID),
		Metadata:   mdPtr,
		QueueItem:  &item,
		Parent:     tracing.RunSpanRefFromMetadata(mdPtr),
		Attributes: attrs,
		StartTime:  startedAt,
		EndTime:    endedAt,
	})
}

// OnStepRunFinished creates a span for a plain step.run step
// (enums.OpcodeStep/OpcodeStepRun) completing — see
// executor.handleGeneratorStep, which calls this alongside its own
// e.emitStepSpan using the exact same seed
// (tracing.FinalizedStepDynamicSeed(gen.ID)), so this span's identity
// matches that real one.
func (l *listener) OnStepRunFinished(ctx context.Context, md sv2.Metadata, item queue.Item, _ inngest.Edge, gen statev1.GeneratorOpcode, now time.Time) {
	// tracing.GeneratorAttrs already sets StepOutput (via op.Output(), same
	// as gen.Output() below) for OpcodeStep/OpcodeStepRun, so stepAttrs
	// covers it — only DynamicStatus needs setting on top, matching
	// emitStepSpan's default-case status for these two opcodes.
	attrs, startedAt, endedAt := stepAttrs(&gen, item, md, now)
	status := enums.StepStatusCompleted
	meta.AddAttr(attrs, meta.Attrs.DynamicStatus, &status)

	mdPtr := safeMetadata(md)
	_, _ = l.createSpan(ctx, tracingv3.SpanNameStep, &tracing.CreateSpanOptions{
		Seed:       tracing.FinalizedStepDynamicSeed(gen.ID),
		Metadata:   mdPtr,
		QueueItem:  &item,
		Parent:     tracing.RunSpanRefFromMetadata(mdPtr),
		Attributes: attrs,
		StartTime:  startedAt,
		EndTime:    endedAt,
	})
}

// OnStepFinished creates a span for tracingv3.SpanNameExecution — see the doc
// comment above OnSleep for why this gets a random span_id rather than a
// deterministic one, the same as OnFunctionScheduled/OnFunctionStarted's
// markers. resp may be nil (a request-level failure before any response
// was parsed — see executor.go's ExecutePost, which calls OnStepFinished
// with resp == nil and err != nil in that case).
func (l *listener) OnStepFinished(ctx context.Context, md sv2.Metadata, item queue.Item, _ inngest.Edge, resp *statev1.DriverResponse, stepErr error, reqStart time.Time, now time.Time) {
	// This span covers the whole SDK request, which may span several steps
	// (or none) — see the doc comment above OnSleep — so no step-specific
	// attributes (name, attempt, etc.) belong here, matching the real
	// execution span's own attrs (tracing.FunctionAttrs + DriverResponseAttrs,
	// neither of which set anything step-scoped).
	attrs := meta.NewAttrSet()

	status := enums.StepStatusCompleted
	switch {
	case stepErr != nil:
		status = enums.StepStatusFailed
		attrs.AddErr(stepErr)
	case resp != nil && resp.Err != nil:
		status = enums.StepStatusFailed
		attrs.AddErr(errors.New(*resp.Err))
	case resp != nil && resp.UserError != nil:
		status = enums.StepStatusFailed
		attrs.AddErr(errors.New(resp.UserError.Message))
	}
	meta.AddAttr(attrs, meta.Attrs.DynamicStatus, &status)

	// end is "now", when the SDK request has finished and this hook fires;
	// start is reqStart, captured by the caller immediately before the
	// request was sent (see runInstance.reqStart's doc comment) — the real
	// boundary, rather than approximating it by subtracting resp.Duration
	// from end.
	end := now
	start := reqStart
	if start.IsZero() {
		start = end
	}
	if resp != nil {
		fnOutput, err := resp.GetTraceFunctionOutput()
		if err == nil && fnOutput != "" {
			meta.AddAttr(attrs, meta.Attrs.StepOutput, &fnOutput)
		}
		// Same redaction/compaction tracing.DriverResponseAttrs applies for
		// the real execution span's own ResponseHeaders attribute.
		redactedHeaders := headers.Compact(headers.Redact(resp.Header))
		meta.AddAttr(attrs, meta.Attrs.ResponseHeaders, &redactedHeaders)

		// Same fallback tracing.DriverResponseAttrs uses: resp.OutputSize is
		// the driver-reported payload size, but falls back to the extracted
		// function output's own length when the driver didn't report one.
		size := resp.OutputSize
		if size == 0 && fnOutput != "" {
			size = len(fnOutput)
		}
		meta.AddAttr(attrs, meta.Attrs.ResponseOutputSize, &size)
		meta.AddAttr(attrs, meta.Attrs.ResponseStatusCode, &resp.StatusCode)

		// Every opcode the SDK reported in this response, the same debugging
		// attribute tracing.DriverResponseAttrs always adds regardless of
		// how many ops the response carries.
		steps := make(meta.ResponseOps, len(resp.Generator))
		for i, s := range resp.Generator {
			steps[i] = meta.ResponseOp{Op: s.Op, ID: s.ID, Name: s.Name}
		}
		meta.AddAttr(attrs, meta.Attrs.ResponseSteps, &steps)
	}

	mdPtr := safeMetadata(md)
	_, _ = l.createSpan(ctx, tracingv3.SpanNameExecution, &tracing.CreateSpanOptions{
		Metadata:   mdPtr,
		QueueItem:  &item,
		Parent:     tracing.RunSpanRefFromMetadata(mdPtr),
		Attributes: attrs,
		StartTime:  start,
		EndTime:    end,
	})

	// A transient (retryable) failure never reaches OnFunctionFinished — the
	// executor loops back through OnStepScheduled to retry instead (see
	// runFinishedStatus's doc comment) — so this is the only place that
	// hook's own nonstep span (see emitOnFunctionFinishedNonStepSpan) would
	// otherwise be missed for this one outcome, matching
	// executor.go's own emitNonStepSpan(..., StepStatusErrored) call for a
	// retryable resp.Err.
	if resp != nil && resp.Err != nil && resp.Retryable() {
		l.emitOnFunctionFinishedNonStepSpan(ctx, mdPtr, item, *resp, enums.StepStatusErrored, start, end)
	}
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
// runs/events into db, and starts its own background batching goroutines
// (batch.go) that drain the listener's channels and flush into the
// runs_staging/events_staging tables. It also starts a standalone
// SpanExporter (tracing.go) backing this listener's own private
// tracingv3.TracerProvider, sharing db. The batching goroutines run for the
// lifetime of the process (context.Background()) unless the caller stops
// them via Close (the returned value always implements Closer). Compaction
// of staged rows out to Parquet is out of scope for this POC's minimal
// wiring (descoped by the coordinator).
func NewListener(db *sql.DB, opts ...Option) execution.SyncLifecycleListener {
	o := defaultSetupOpts()
	for _, apply := range opts {
		apply(&o)
	}

	l := newListenerWithChannels(o.runsCap, o.eventsCap)
	l.db = db

	tables := map[string]chan map[string]any{
		"inngest.runs":   l.runs,
		"inngest.events": l.events,
	}
	// One shared disabledState across every batcher (runs/events here, plus
	// the span exporter's own below), so the driver's terminal
	// duckdb.ErrDisabled state stops the whole dual-write path and is logged
	// exactly once rather than once per table.
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

	l.spanExporter = newSpanExporter(db, o.spansCap, batcherOpts{maxSize: o.batchMaxSize, flushInterval: o.batchInterval, disabled: disabled})
	l.tp = newListenerTracerProvider(l.spanExporter, o.batchInterval)

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
	// Stop the span exporter's own batcher/goroutine too, before db is
	// closed below — SpanExporter.Shutdown never touches db itself, since
	// this listener owns that lifecycle.
	if l.spanExporter != nil {
		_ = l.spanExporter.Shutdown(ctx)
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
