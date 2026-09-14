// Package dualwrite implements a DuckDB-writing execution.SyncLifecycleListener.
// The executor and runner call its hooks synchronously, so every hook must do
// nothing but build a row and non-blocking-send it onto a per-table channel;
// background goroutines (batch.go) drain those channels and flush batches
// into DuckDB staging tables.
//
// run_spans (inngest.run_trace_spans) are written differently from
// runs/events: the per-step hooks below create real spans through this
// listener's own tracingv3.TracerProvider (see tracing.go) instead of
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
	"github.com/inngest/inngestgo"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/trace"
)

// listener implements execution.SyncLifecycleListener. Every hook body does
// nothing but build a row and non-blocking-send it onto the matching
// channel — no I/O, no locking, no flush logic — which is what makes it safe
// to call synchronously from the executor and runner.
type listener struct {
	execution.NoopSyncLifecycleListener

	events   chan map[string]any
	metadata chan map[string]any

	droppedEvents   atomic.Int64
	droppedMetadata atomic.Int64

	// spanExporter/tp back every per-step hook's span creation — see
	// tracing.go. tp is what hooks call (l.createSpan); spanExporter is the
	// sdktrace.SpanExporter tp is wired to, kept here so Close can shut it
	// down.
	spanExporter *SpanExporter
	tp           tracingv3.TracerProvider

	// db and batchers/wg back Close: they stop the background batcher
	// goroutines this listener starts and close the db handed to NewListener.
	db       *sql.DB
	batchers []*batcher
	wg       sync.WaitGroup
}

func newListenerWithChannels(eventsCap, metadataCap int) *listener {
	return &listener{
		events:   make(chan map[string]any, eventsCap),
		metadata: make(chan map[string]any, metadataCap),
	}
}

func (l *listener) sendEvent(row map[string]any) {
	select {
	case l.events <- row:
	default:
		l.droppedEvents.Add(1)
	}
}

func (l *listener) sendMetadata(row map[string]any) {
	select {
	case l.metadata <- row:
	default:
		l.droppedMetadata.Add(1)
	}
}

// scanTriggerEvents collects every triggering event's Meta.Sessions into the
// run-level session list, deduped and sorted by (Key, ID) for determinism,
// and capped at consts.MaxRunSessions. A malformed entry is skipped rather
// than failing the whole span.
func scanTriggerEvents(evts []json.RawMessage) (sessions meta.EventSessions) {
	for _, raw := range evts {
		var evt event.Event
		if err := json.Unmarshal(raw, &evt); err != nil {
			continue
		}
		for name, id := range evt.Meta.Sessions {
			sessions = append(sessions, meta.EventSession{Key: name, ID: id})
		}
	}
	if len(sessions) == 0 {
		return nil
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
	return sessions
}

// addRunEventAttrs sets the run-level, event-derived span attributes
// tracing.go's materializeRuns reads: event.ids and event.sessions. Both are
// omitted entirely rather than set empty when not applicable (e.g. a
// cron-only run has no triggering event).
func addRunEventAttrs(attrs *meta.SerializableAttrs, md sv2.Metadata, evts []json.RawMessage) {
	if len(md.Config.EventIDs) > 0 {
		ids := make([]string, len(md.Config.EventIDs))
		for i, id := range md.Config.EventIDs {
			ids[i] = id.String()
		}
		meta.AddAttr(attrs, meta.Attrs.EventIDs, &ids)
	}
	if sessions := scanTriggerEvents(evts); len(sessions) > 0 {
		meta.AddAttr(attrs, meta.Attrs.Sessions, &sessions)
	}
}

// addEventsInputAttr sets meta.Attrs.EventsInput to evts marshaled as a
// single JSON array, so the run spans' `input` column (see spanExportRow)
// carries the triggering events.
func addEventsInputAttr(ctx context.Context, attrs *meta.SerializableAttrs, evts []json.RawMessage) {
	byt, err := json.Marshal(evts)
	if err != nil {
		logger.StdlibLogger(ctx).Error("dualwrite: failed to marshal events for EventsInput attribute", "error", err)
		return
	}
	str := string(byt)
	meta.AddAttr(attrs, meta.Attrs.EventsInput, &str)

	addDeferParentAttrs(attrs, deferredScheduleMetadata(ctx, evts))
}

// createSpan is nil-safe: l.tp is nil on a *listener built via
// newListenerWithChannels directly (see listener_test.go) rather than
// NewListener, and calling a method on a nil interface panics, so every hook
// must go through this rather than l.tp.CreateSpan directly.
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
	// A point span: start and end both pinned to queuedAt. StartedAt/EndedAt
	// are explicitly set nil (rather than left for tracingv3.CreateSpan to
	// auto-populate from StartTime/EndTime) since the run has only been
	// queued, not started or ended.
	queuedAt := ulid.Time(md.ID.RunID.Time())
	attrs := meta.NewAttrSet()
	meta.AddAttr(attrs, meta.Attrs.QueuedAt, &queuedAt)
	meta.AddAttr(attrs, meta.Attrs.StartedAt, (*time.Time)(nil))
	meta.AddAttr(attrs, meta.Attrs.EndedAt, (*time.Time)(nil))
	meta.AddAttr(attrs, meta.Attrs.DynamicStatus, inngestgo.Ptr(enums.StepStatusQueued))
	addEventsInputAttr(ctx, attrs, evts)
	addRunEventAttrs(attrs, md, evts)

	mdPtr := safeMetadata(md)
	_, _ = l.createSpan(ctx, tracingv3.SpanNameRunQueued, &tracing.CreateSpanOptions{
		Metadata:   mdPtr,
		QueueItem:  &item,
		Parent:     tracing.RunSpanRefFromMetadata(mdPtr),
		Attributes: attrs,
		StartTime:  queuedAt,
		EndTime:    queuedAt,
	})
}

// deferredScheduleMetadata extracts every valid inngest/deferred.schedule
// trigger event's DeferredScheduleMetadata from evts, skipping and logging
// any that fail to parse or validate.
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

// addDeferParentAttrs stamps the child run's DeferParentRunIDs/
// DeferParentFnSlug attrs from the run's deferred-schedule trigger events.
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

func (l *listener) OnFunctionStarted(ctx context.Context, md sv2.Metadata, item queue.Item, evts []json.RawMessage) {
	// Not a point span: physical start is queuedAt, physical end is
	// md.Config.StartedAt.
	queuedAt := ulid.Time(md.ID.RunID.Time())
	attrs := meta.NewAttrSet()
	if !md.Config.StartedAt.IsZero() {
		meta.AddAttr(attrs, meta.Attrs.StartedAt, &md.Config.StartedAt)
	}
	meta.AddAttr(attrs, meta.Attrs.DynamicStatus, inngestgo.Ptr(enums.StepStatusRunning))
	addEventsInputAttr(ctx, attrs, evts)
	addRunEventAttrs(attrs, md, evts)

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

// runFinishedStatus derives the run's finished status from resp: a
// retryable resp.Err has already looped back through OnStepScheduled
// instead of reaching here, so only these two outcomes remain.
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

	// Two spans, matching the real topology: a stable root "executor.run"
	// span (Seed=md.ID.RunID[:]) and a child nonstep span for the
	// function's own output event — see emitOnFunctionFinishedNonStepSpan
	// below.
	//
	// QueuedAt uses the run's own ULID-embedded timestamp, not
	// item.EnqueuedAt: item here is whichever queue.Item triggered this
	// specific finish call, which for a multi-step function isn't the run's
	// original enqueue.
	//
	// ScheduledAt is omitted: item.At would be equally wrong for the same
	// reason, and the run's real scheduled_at isn't otherwise available
	// here. TODO: thread the run's actual scheduled_at through sv2.Metadata.
	runAttrs := meta.NewAttrSet()
	meta.AddAttr(runAttrs, meta.Attrs.DynamicStatus, &stepStatus)
	addEventsInputAttr(ctx, runAttrs, evts)
	addRunEventAttrs(runAttrs, md, evts)
	tracing.AddTimingAttrs(runAttrs, queuedAt, time.Time{}, start, end)
	addRunSpanAttrs(runAttrs, mdPtr)
	// Mirrors executor.Finalize: production stamps the function output onto
	// both the nonstep span and the root run span, so this root span needs
	// the same attrs the nonstep span below gets via DriverResponseOutputAttrs.
	if fnOutput != "" {
		isFunctionOutput := true
		meta.AddAttr(runAttrs, meta.Attrs.IsFunctionOutput, &isFunctionOutput)
		meta.AddAttr(runAttrs, meta.Attrs.StepOutput, &fnOutput)
	}
	_, _ = l.createSpan(ctx, tracingv3.SpanNameRun, &tracing.CreateSpanOptions{
		Seed:       md.ID.RunID[:],
		Metadata:   mdPtr,
		QueueItem:  &item,
		StartTime:  queuedAt,
		EndTime:    end,
		Attributes: runAttrs,
	})

	// Emit the nonstep span before returning, so nothing reading trace data
	// ever observes this run as finished before its corresponding span
	// exists.
	l.emitOnFunctionFinishedNonStepSpan(ctx, mdPtr, item, resp, stepStatus, start, end)
}

// emitOnFunctionFinishedNonStepSpan creates the span for the function's own
// output event, a child of the root "executor.run" span (see
// OnFunctionFinished above), using the same Seed
// (tracing.NonStepDynamicSeed(item)) the real system's nonstep span uses, so
// a reader can correlate the two by identity. tracingv3.SpanNameError/
// SpanNameFinal split by outcome, rather than the real system's single
// meta.SpanNameNonStep name for both.
func (l *listener) emitOnFunctionFinishedNonStepSpan(ctx context.Context, mdPtr *sv2.Metadata, item queue.Item, resp statev1.DriverResponse, status enums.StepStatus, start, end time.Time) {
	// tracing.DriverResponseOutputAttrs is the same builder
	// executor.emitNonStepSpan uses, so this span's attrs match the real
	// one.
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
	// TODO: emit a span here maybe?
}

// OnStepScheduled creates a point-in-time marker span (tracingv3.SpanNameStepPlanned)
// for when a step is scheduled — a distinct span kind from the real
// "executor.step" span, with its own random span_id rather than a seed
// derived from the step: reusing the eventual finished step span's identity
// here would collide with it once that span is inserted (this package only
// ever inserts, never updates in place — see SpanExporter's doc comment).
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

// OnExtendedTraceSpan creates the userland (extended-trace) span through
// this listener's own TracerProvider, same as every other span-producing
// hook, so the resulting row goes through the same spanExportRow conversion
// rather than a separate hand-rolled path. Tenant/run identity is set
// explicitly from span's own typed fields, since there's no sv2.Metadata
// here for addTenantAndDebugAttrs to read.
func (l *listener) OnExtendedTraceSpan(ctx context.Context, span execution.ExtendedTraceSpan) {
	attrs := meta.NewAttrSet()
	meta.AddAttr(attrs, meta.Attrs.AccountID, &span.AccountID)
	meta.AddAttr(attrs, meta.Attrs.EnvID, &span.EnvID)
	meta.AddAttr(attrs, meta.Attrs.AppID, &span.AppID)
	meta.AddAttr(attrs, meta.Attrs.FunctionID, &span.FunctionID)
	meta.AddAttr(attrs, meta.Attrs.RunID, &span.RunID)
	if span.FunctionSlug != "" {
		meta.AddAttr(attrs, meta.Attrs.FunctionSlug, &span.FunctionSlug)
	}
	if span.AppName != "" {
		meta.AddAttr(attrs, meta.Attrs.AppName, &span.AppName)
	}

	_, _ = l.createSpan(ctx, tracingv3.SpanNameExtendedTrace, &tracing.CreateSpanOptions{
		Debug:              &tracing.SpanDebugData{Location: "dualwrite.listener.OnExtendedTraceSpan"},
		Attributes:         attrs,
		StartTime:          span.StartTime,
		EndTime:            span.EndTime,
		Parent:             span.Parent,
		RawOtelSpanOptions: []trace.SpanStartOption{trace.WithAttributes(span.Attributes...)},
		SpanID:             span.SpanID,
	})
}

// OnMetadataEntry writes one metadata emission directly into
// inngest.run_metadata — a standalone table, so (unlike OnExtendedTraceSpan)
// this builds a row by hand rather than creating a span. span_id is
// entry.Parent's own identity, the join key back to inngest.run_trace_spans.
// op (merge/set/delete/add) is intentionally not stored — see
// execution.MetadataEntry's doc comment.
func (l *listener) OnMetadataEntry(ctx context.Context, entry execution.MetadataEntry) {
	valuesByt, err := json.Marshal(entry.Values)
	if err != nil {
		logger.StdlibLogger(ctx).Error("dualwrite: failed to marshal metadata values", "kind", entry.Kind.String(), "error", err)
		return
	}

	spanID := tracing.SpanContextFromMetadata(entry.Parent).SpanID().String()

	row := map[string]any{
		"account_id":    entry.AccountID.String(),
		"env_id":        entry.EnvID.String(),
		"run_id":        entry.RunID.String(),
		"run_queued_at": ulid.Time(entry.RunID.Time()),
		"span_id":       spanID,
		"scope":         entry.Scope.String(),
		"kind":          entry.Kind.Suffix(),
		"is_user":       entry.Kind.IsUser(),
		"values":        string(valuesByt),
		"created_at":    entry.CreatedAt,
	}
	if entry.StepID != "" {
		row["step_id"] = entry.StepID
	}
	if entry.StepIndex != nil {
		row["step_index"] = *entry.StepIndex
	}
	if entry.StepAttempt != nil {
		row["step_attempt"] = *entry.StepAttempt
	}

	l.sendMetadata(row)
}

// OnDeferAdd writes the run's own executor.defer span. Multiple physical
// rows can exist for the same logical defer (this row, and later an Abort
// row from OnDeferAbort); duckdbquery's read side (GetRunDefers) groups them
// by the defer.hashed_id attribute rather than needing a dedicated
// dynamic_span_id column, so this span gets no Seed and its span_id is left
// to the normal ID generator.
func (l *listener) OnDeferAdd(ctx context.Context, md sv2.Metadata, d sv2.Defer, userlandID string, now time.Time) {
	mdPtr := safeMetadata(md)
	eventID, err := event.DeferEventID(md.ID.RunID, d.HashedID)
	if err != nil {
		logger.StdlibLogger(ctx).Error("dualwrite: failed to compute defer event ID", "error", err, "run_id", md.ID.RunID.String(), "hashed_id", d.HashedID)
	}

	_, _ = l.createSpan(ctx, tracingv3.SpanNameDefer, &tracing.CreateSpanOptions{
		Metadata:  mdPtr,
		Parent:    tracing.RunSpanRefFromMetadata(mdPtr),
		StartTime: now,
		EndTime:   now,
		Attributes: meta.NewAttrSet(
			meta.Attr(meta.Attrs.DeferHashedID, &d.HashedID),
			meta.Attr(meta.Attrs.DeferUserlandID, &userlandID),
			meta.Attr(meta.Attrs.DeferFnSlug, &d.FnSlug),
			meta.Attr(meta.Attrs.DeferStatus, &d.ScheduleStatus),
			meta.Attr(meta.Attrs.DeferEventID, &eventID),
		),
	})
}

// OnDeferAbort re-emits the defer's executor.defer span (same
// defer.hashed_id as OnDeferAdd, but its own random span_id) with
// status=Aborted, now also carrying fn_slug/userland_id so every abort row
// is self-describing.
func (l *listener) OnDeferAbort(ctx context.Context, md sv2.Metadata, hashedID, fnSlug, userlandID string, now time.Time) {
	mdPtr := safeMetadata(md)
	eventID, err := event.DeferEventID(md.ID.RunID, hashedID)
	if err != nil {
		logger.StdlibLogger(ctx).Error("dualwrite: failed to compute defer event ID", "error", err, "run_id", md.ID.RunID.String(), "hashed_id", hashedID)
	}

	status := enums.DeferStatusAborted
	_, _ = l.createSpan(ctx, tracingv3.SpanNameDefer, &tracing.CreateSpanOptions{
		Metadata:  mdPtr,
		Parent:    tracing.RunSpanRefFromMetadata(mdPtr),
		StartTime: now,
		EndTime:   now,
		Attributes: meta.NewAttrSet(
			meta.Attr(meta.Attrs.DeferHashedID, &hashedID),
			meta.Attr(meta.Attrs.DeferFnSlug, &fnSlug),
			meta.Attr(meta.Attrs.DeferUserlandID, &userlandID),
			meta.Attr(meta.Attrs.DeferStatus, &status),
			meta.Attr(meta.Attrs.DeferEventID, &eventID),
		),
	})
}

// The per-step hooks below create real spans through l.tp, using the same
// Seed values the real system's CreateSpan calls use for that step, so a
// span here lines up with the real one by span_id/trace_id.
//
// The three pause-backed opcodes (wait-for-event, wait-for-signal, invoke)
// each create two spans: a point-in-time marker when the pause begins, and
// a second span spanning the pause's full duration with a deterministic
// span_id.
//
// OnStepFinished's span (tracingv3.SpanNameExecution) gets a random span_id
// rather than a deterministic one: it fires once per SDK request, which may
// cover several steps or none, so there's no single step identity to key
// off of.
//
// OnStepStarted is not implemented — it carries too little to put on a
// span — and is covered as a no-op by execution.NoopSyncLifecycleListener.

// stepTiming replicates executor.opcodeTiming's fallback logic: gen.Timing
// is only populated by newer SDKs, so it can come back zero — this package
// has no RunContext, so it falls back to the run's own
// md.Config.StartedAt/now instead.
func stepTiming(item queue.Item, md sv2.Metadata, gen *statev1.GeneratorOpcode, now time.Time) (queuedAt, scheduledAt, startedAt, endedAt time.Time) {
	queuedAt = item.EnqueuedAt
	scheduledAt = item.At
	if scheduledAt.Before(queuedAt) {
		scheduledAt = queuedAt
	}

	// interval.Interval{}.Start() decodes an unset Timing as the Unix epoch,
	// not time.Time's zero value, so check explicitly rather than relying on
	// startedAt.IsZero() below.
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
// (OnSleep/OnStepRunFinished/OnStepGatewayRequestFinished): the same
// tracing.GeneratorAttrs builder and timing attrs the real spans use.
// Returns startedAt/endedAt too, so callers can reuse them as the span's own
// start/end.
func stepAttrs(gen *statev1.GeneratorOpcode, item queue.Item, md sv2.Metadata, now time.Time) (attrs *meta.SerializableAttrs, startedAt, endedAt time.Time) {
	attrs = tracing.GeneratorAttrs(gen)
	queuedAt, scheduledAt, startedAt, endedAt := stepTiming(item, md, gen, now)
	tracing.AddTimingAttrs(attrs, queuedAt, scheduledAt, startedAt, endedAt)
	return attrs, startedAt, endedAt
}

func (l *listener) OnSleep(ctx context.Context, md sv2.Metadata, item queue.Item, gen statev1.GeneratorOpcode, _ time.Time, now time.Time) {
	// tracing.GeneratorAttrs already sets StepSleepDuration from the step's
	// configured duration, so there's nothing to add on top of it here.
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
	// tracing.GeneratorAttrs already handles OpcodeWaitForEvent
	// (StepWaitForEventName/If, StepWaitExpiry), same as OnSleep via
	// stepAttrs.
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

// createPauseStartedSpan creates the point-in-time marker span for when a
// pause begins, dated to the pause's own CreatedAt. OnInvokeFunction passes
// a zero Pause; CreateSpanOptions defaults StartTime to "now" in that case.
func (l *listener) createPauseStartedSpan(ctx context.Context, md sv2.Metadata, item queue.Item, attrs *meta.SerializableAttrs, pause statev1.Pause) {
	// ScheduledAt is never allowed to precede the pause's own CreatedAt.
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
// called at resume time, from pause.CreatedAt through to now. Its Seed is
// the same tracing.FinalizedStepDynamicSeed the step's eventual finished
// span uses, so a reader can correlate the two by identity.
func (l *listener) createPauseSpan(ctx context.Context, md sv2.Metadata, pause statev1.Pause, r execution.ResumeRequest, now time.Time) {
	attrs := resumeAttrs(pause, r)
	// A timeout is an expected outcome, not a failure.
	status := enums.StepStatusCompleted
	if r.IsTimeout {
		status = enums.StepStatusTimedOut
	}
	meta.AddAttr(attrs, meta.Attrs.DynamicStatus, &status)

	// QueuedAt/ScheduledAt fall back to pause.CreatedAt, the only timestamp
	// available here. now is the caller's own resume-time timestamp, and
	// must be passed as EndTime so the tracer adds the EndedAt attribute.
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

// resumeAttrs is shared by the three *Resumed hooks above: StepID/StepOp
// are recoverable from pause alone (StepID from pause.Outgoing, StepOp from
// whichever of pause.IsWaitForEvent()/IsInvoke()/IsSignal() matches). Unlike
// the real system's tracing.ResumeAttrs, this sets every opcode-specific
// field explicitly rather than relying on a read-time merge with the
// pause-started span's fragment — this package's flat model never merges
// rows.
func resumeAttrs(pause statev1.Pause, r execution.ResumeRequest) *meta.SerializableAttrs {
	attrs := meta.NewAttrSet()
	name := r.StepName
	if name == "" {
		name = pause.StepName
	}
	if name != "" {
		meta.AddAttr(attrs, meta.Attrs.StepName, &name)
	}
	// pause.Outgoing is the step's own ID, set unconditionally since every
	// pause type carries it (unlike the opcode-specific fields below).
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
		// TriggeringEventID holds the invocation event's own ID for an
		// invoke pause specifically, not "the event that triggered the
		// original run" as its generic doc comment suggests.
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
	// Failed) as Failed only: retryability isn't derivable here (no
	// runCtx/attempt count).
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

// OnStepRunFinished creates a span for a plain step.run step completing —
// see executor.handleGeneratorStep, which uses the same seed
// (tracing.FinalizedStepDynamicSeed(gen.ID)).
func (l *listener) OnStepRunFinished(ctx context.Context, md sv2.Metadata, item queue.Item, _ inngest.Edge, gen statev1.GeneratorOpcode, now time.Time) {
	// tracing.GeneratorAttrs already sets StepOutput, so only DynamicStatus
	// needs setting here.
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

// OnStepRunFailed creates a span for a plain step.run step erroring or
// failing. status is Errored when the step will be retried, else Failed; a
// retry gets a per-attempt seed distinct from the step's eventual finalized
// span, while Failed reuses the same finalized seed as OnStepRunFinished.
func (l *listener) OnStepRunFailed(ctx context.Context, md sv2.Metadata, item queue.Item, _ inngest.Edge, gen statev1.GeneratorOpcode, status enums.StepStatus, attempt int, now time.Time) {
	attrs, startedAt, endedAt := stepAttrs(&gen, item, md, now)
	meta.AddAttr(attrs, meta.Attrs.DynamicStatus, &status)
	meta.AddAttr(attrs, meta.Attrs.StepAttempt, &attempt)

	seed := tracing.FinalizedStepDynamicSeed(gen.ID)
	if status == enums.StepStatusErrored {
		seed = tracing.RetryStepDynamicSeed(gen.ID, attempt)
	}

	mdPtr := safeMetadata(md)
	_, _ = l.createSpan(ctx, tracingv3.SpanNameStep, &tracing.CreateSpanOptions{
		Seed:       seed,
		Metadata:   mdPtr,
		QueueItem:  &item,
		Parent:     tracing.RunSpanRefFromMetadata(mdPtr),
		Attributes: attrs,
		StartTime:  startedAt,
		EndTime:    endedAt,
	})
}

// OnStepFinished creates a span for tracingv3.SpanNameExecution, with a
// random span_id (see the doc comment above OnSleep). resp may be nil — a
// request-level failure before any response was parsed.
func (l *listener) OnStepFinished(ctx context.Context, md sv2.Metadata, item queue.Item, _ inngest.Edge, resp *statev1.DriverResponse, stepErr error, reqStart time.Time, now time.Time) {
	// This span covers the whole SDK request, which may span several steps
	// or none, so no step-specific attributes belong here.
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

	// end is "now"; start is reqStart, captured immediately before the
	// request was sent.
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
		// Same redaction tracing.DriverResponseAttrs applies.
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

	// A retryable failure never reaches OnFunctionFinished — the executor
	// loops back through OnStepScheduled instead — so this is the only place
	// its nonstep span would otherwise be missed.
	if resp != nil && resp.Err != nil && resp.Retryable() {
		l.emitOnFunctionFinishedNonStepSpan(ctx, mdPtr, item, *resp, enums.StepStatusErrored, start, end)
	}
}

// Option configures NewListener.
type Option func(*setupOpts)

type setupOpts struct {
	runsCap, spansCap, eventsCap, metadataCap int
	batchMaxSize                              int
	batchInterval                             time.Duration
}

func defaultSetupOpts() setupOpts {
	return setupOpts{
		runsCap:       10_000,
		spansCap:      10_000,
		eventsCap:     10_000,
		metadataCap:   10_000,
		batchMaxSize:  10_000,
		batchInterval: 200 * time.Millisecond,
	}
}

// Closer is implemented by the listener NewListener returns. Callers hold
// an execution.SyncLifecycleListener, so reaching this requires a type
// assertion (`l.(dualwrite.Closer)`).
type Closer interface {
	// Close stops every batcher goroutine, waits for them to exit (bounded
	// by ctx), then closes the db passed to NewListener. Call at most once.
	Close(ctx context.Context) error
}

// NewListener returns an execution.SyncLifecycleListener that dual-writes
// runs/events into db, and starts its own background batching goroutines
// (batch.go) that drain the listener's channels and flush into staging
// tables. It also starts a standalone SpanExporter (tracing.go) backing
// this listener's own TracerProvider, sharing db. The batching goroutines
// run for the lifetime of the process unless the caller stops them via
// Close.
func NewListener(db *sql.DB, opts ...Option) execution.SyncLifecycleListener {
	o := defaultSetupOpts()
	for _, apply := range opts {
		apply(&o)
	}

	l := newListenerWithChannels(o.eventsCap, o.metadataCap)
	l.db = db

	tables := map[string]chan map[string]any{
		"inngest.events":       l.events,
		"inngest.run_metadata": l.metadata,
	}
	// One shared disabledState across every batcher, so the driver's
	// terminal duckdb.ErrDisabled state stops the whole dual-write path and
	// is logged once rather than once per table.
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

// Close implements Closer.
//
// If ctx expires before every batcher has drained, Close does not wait
// forever: it closes db anyway, which kills any subprocess a batcher is
// still blocked on and unblocks it — so a wedged batcher self-resolves
// within roughly db.Close()'s own teardown bound rather than leaking
// permanently.
func (l *listener) Close(ctx context.Context) error {
	for _, b := range l.batchers {
		b.stop()
	}
	// Stop the span exporter's own batcher too, before db is closed below.
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
