package dualwrite

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	tracingv3 "github.com/inngest/inngest/pkg/tracing/v3"
	"github.com/oklog/ulid/v2"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// newListenerTracerProvider returns a tracingv3.TracerProvider private to
// one listener: every hook in listener.go that creates a span goes through
// it (l.tp.CreateSpan(...)) with the exact same CreateSpanOptions/Seed/
// Parent plumbing executor.go's own e.tracerProvider uses, rather than
// hand-computing deterministic IDs and row shapes as this package used to
// (see git history). That gets identity computation and Seed/Parent
// semantics for free from the one real implementation, instead of a second,
// hand-maintained one that can drift out of sync with it.
//
// It uses pkg/tracing/v3 rather than pkg/tracing's own TracerProvider: that
// package's executionProcessor (see execution_processor.go's OnStart) leans
// on ambient, context-derived state (ExecutionContext, set only by
// pkg/execution/executor via tracing.WithExecutionContext) that this
// package has no supported way to read, plus per-span-name special cases
// this package's hooks don't uniformly need. Rather than depend on that
// ambient behavior implicitly, every hook below sets the equivalent
// attributes explicitly — see addTenantAndDebugAttrs, called from
// l.createSpan, and the per-span-name attrs added in OnFunctionFinished.
//
// It's a SEPARATE TracerProvider from the process's real one (pkg/devserver
// wires that one straight to sqlite) rather than fanned into it, because
// which spans this package emits, and with what attributes, is expected to
// diverge over time — see SpanExporter's doc comment.
func newListenerTracerProvider(se *SpanExporter, batchTimeout time.Duration) tracingv3.TracerProvider {
	return tracingv3.NewOtelTracerProvider(se, batchTimeout)
}

// addTenantAndDebugAttrs replicates the always-on part of pkg/tracing's
// executionProcessor.OnStart — the part that runs for every span with
// Metadata set, regardless of span name (see AddMetadataTenantAttrs) —
// since this package's TracerProvider is built WithoutExecutionProcessor
// and would otherwise silently drop these from every span it creates.
func addTenantAndDebugAttrs(attrs *meta.SerializableAttrs, md *sv2.Metadata) {
	if md == nil {
		return
	}
	tracing.AddMetadataTenantAttrs(attrs, md.ID)
	meta.AddAttr(attrs, meta.Attrs.DebugRunID, md.Config.DebugRunID())
	meta.AddAttr(attrs, meta.Attrs.DebugSessionID, md.Config.DebugSessionID())
}

// addRunSpanAttrs replicates executionProcessor.OnStart's meta.SpanNameRun
// case (see execution_processor.go) for OnFunctionFinished's span, the only
// hook in this package that uses that name: FunctionVersion, EventIDs,
// CronSchedule, BatchID/BatchTimestamp, and QueuedAt falling back to the
// run ID's own embedded timestamp if nothing set it already (OnFunctionFinished
// sets QueuedAt from item.EnqueuedAt via tracing.AddTimingAttrs before
// calling this, so the fallback here only matters when that was zero).
func addRunSpanAttrs(attrs *meta.SerializableAttrs, md *sv2.Metadata) {
	if md == nil {
		return
	}

	queuedAt := md.ID.RunID.Timestamp()
	meta.AddAttrIfUnset(attrs, meta.Attrs.QueuedAt, &queuedAt)

	meta.AddAttr(attrs, meta.Attrs.FunctionVersion, &md.Config.FunctionVersion)

	eventIDs := make([]string, len(md.Config.EventIDs))
	for i, id := range md.Config.EventIDs {
		eventIDs[i] = id.String()
	}
	meta.AddAttr(attrs, meta.Attrs.EventIDs, &eventIDs)

	if cron := md.Config.CronSchedule(); cron != nil {
		meta.AddAttr(attrs, meta.Attrs.CronSchedule, cron)
	}

	if md.Config.BatchID != nil {
		batchTS := md.Config.BatchID.Timestamp()
		meta.AddAttr(attrs, meta.Attrs.BatchID, md.Config.BatchID)
		meta.AddAttr(attrs, meta.Attrs.BatchTimestamp, &batchTS)
	}
}

// SpanExporter is a standalone, DuckDB-specific sdktrace.SpanExporter
// backing dualwrite's private TracerProvider (see newListenerTracerProvider)
// with the same channel+batcher machinery as listener's runs/events. Every
// span dualwrite's hooks create — via l.tp.CreateSpan, never by hand —
// arrives here as a real OTel ReadOnlySpan, converted into an
// inngest.run_trace_spans row the same way pkg/tracing/tracer_sqlc.go's
// dbExporter converts one for pkg/db/sqlite's `spans` table (see
// spanExportRow), so the same span, by the same span_id/trace_id, lands
// equivalently in both stores. It's a deliberately separate exporter
// (rather than folding into dbExporter, or wiring into the process's real
// TracerProvider) because which spans and which attributes land in DuckDB
// is expected to diverge from sqlite over time.
type SpanExporter struct {
	spans   chan map[string]any
	dropped atomic.Int64
	b       *batcher
	wg      sync.WaitGroup
}

func newSpanExporter(db *sql.DB, spansCap int, opts batcherOpts) *SpanExporter {
	ch := make(chan map[string]any, spansCap)
	se := &SpanExporter{
		spans: ch,
		b:     newBatcher(db, "inngest.run_trace_spans", ch, opts),
	}
	se.wg.Add(1)
	go func() {
		defer se.wg.Done()
		se.b.run(context.Background())
	}()
	return se
}

// ExportSpans implements sdktrace.SpanExporter. dualwrite's private
// TracerProvider uses a SimpleSpanProcessor (see NewOtelTracerProvider), so
// this runs synchronously on the calling hook's own goroutine — but it does
// no I/O of its own: converting a span to a row and non-blocking
// channel-sending it (send) is exactly as cheap as every other hook body in
// this package. batcher.go's own background goroutine is what actually
// talks to the duckdb subprocess.
func (se *SpanExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, span := range spans {
		row, ok := spanExportRow(ctx, span)
		if !ok {
			continue
		}
		se.send(row)
	}
	return nil
}

// send is nil-receiver-safe: a *listener built via newListenerWithChannels
// directly (see listener_test.go) rather than NewListener has no
// spanExporter/tracer provider at all, and this package's hooks must never
// crash the executor's critical path over that.
func (se *SpanExporter) send(row map[string]any) {
	if se == nil {
		return
	}
	select {
	case se.spans <- row:
	default:
		se.dropped.Add(1)
	}
}

// Shutdown implements sdktrace.SpanExporter. It stops the batcher and waits
// (bounded by ctx) for it to drain and flush, but never closes db —
// listener.Close owns that, since this exporter shares its db connection
// with the listener's runs/events batchers.
func (se *SpanExporter) Shutdown(ctx context.Context) error {
	se.b.stop()

	done := make(chan struct{})
	go func() {
		se.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
	}
	return nil
}

// spanExportRow converts one real OTel span into an inngest.run_trace_spans
// row, mirroring pkg/tracing/tracer_sqlc.go's dbExporter.ExportSpans
// attribute extraction almost exactly — except using meta.AttrsByKey's
// WrapValue rather than a plain kv.Value.AsInterface(), so a JSON-typed
// attribute round-trips as json.RawMessage instead of a double-encoded
// string. ok is false when the span can't be stored (missing/invalid run
// ID, or a marshaling failure) — attributes is NOT NULL, so a row silently
// missing whatever it was trying to say is worse than not emitting it at
// all.
func spanExportRow(ctx context.Context, span sdktrace.ReadOnlySpan) (row map[string]any, ok bool) {
	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()
	parentID := span.Parent().SpanID().String()

	var accountID, envID, appID, functionID, runID string
	var output, input any

	attrs := make(map[string]any)
	for _, attr := range span.Attributes() {
		key := string(attr.Key)

		switch key {
		case meta.Attrs.StepOutput.Key():
			output = attr.Value.AsInterface()
			continue
		case meta.Attrs.EventsInput.Key(), meta.Attrs.StepInput.Key():
			input = attr.Value.AsInterface()
			continue
		case meta.Attrs.AccountID.Key():
			accountID = attr.Value.AsString()
		case meta.Attrs.EnvID.Key():
			envID = attr.Value.AsString()
		case meta.Attrs.RunID.Key():
			runID = attr.Value.AsString()
		case meta.Attrs.AppID.Key():
			appID = attr.Value.AsString()
		case meta.Attrs.FunctionID.Key():
			functionID = attr.Value.AsString()
		case meta.Attrs.DynamicTraceID.Key():
			traceID = attr.Value.AsString()
		}

		if s, ok := meta.AttrsByKey[key]; ok {
			attrs[key] = s.WrapValue(attr.Value)
		} else {
			attrs[key] = attr.Value.AsInterface()
		}
	}

	l := logger.StdlibLogger(ctx)

	if runID == "" {
		l.Error("dualwrite: span export missing run ID", "span_id", spanID, "trace_id", traceID, "name", span.Name())
		return nil, false
	}
	rid, err := ulid.Parse(runID)
	if err != nil {
		l.Error("dualwrite: span export run ID is not a valid ULID", "run_id", runID, "error", err)
		return nil, false
	}

	attrsByt, err := json.Marshal(attrs)
	if err != nil {
		l.Error("dualwrite: failed to marshal span attributes", "span_id", spanID, "trace_id", traceID, "error", err)
		return nil, false
	}
	linksByt, err := json.Marshal(span.Links())
	if err != nil {
		l.Error("dualwrite: failed to marshal span links", "span_id", spanID, "trace_id", traceID, "error", err)
		return nil, false
	}

	row = map[string]any{
		"span_id":        spanID,
		"trace_id":       traceID,
		"parent_span_id": parentID,
		"name":           span.Name(),
		"start_time":     span.StartTime().Round(0), // strip monotonic clock reading, as dbExporter does
		"end_time":       span.EndTime().Round(0),
		"run_id":         runID,
		"run_queued_at":  rid.Timestamp(),
		"account_id":     accountID,
		"env_id":         envID,
		"app_id":         appID,
		"function_id":    functionID,
		"attributes":     string(attrsByt),
		"links":          string(linksByt),
	}
	if outByt := anyToJSONBytes(output); len(outByt) > 0 {
		row["output"] = string(outByt)
	}
	if inByt := anyToJSONBytes(input); len(inByt) > 0 {
		row["input"] = string(inByt)
	}

	return row, true
}

// anyToJSONBytes converts a value to []byte for storage in a JSON column,
// the same rule pkg/tracing/tracer_sqlc.go's dbExporter uses for its
// output/input columns: strings and byte slices are used directly to avoid
// double-encoding; other types are JSON-marshaled.
func anyToJSONBytes(v any) []byte {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case string:
		if val == "" {
			return nil
		}
		return []byte(val)
	case []byte:
		if len(val) == 0 {
			return nil
		}
		return val
	default:
		byt, _ := json.Marshal(val)
		return byt
	}
}

// safeMetadata returns a pointer to a copy of md with its Config's internal
// lock initialized. pkg/tracing's executionProcessor.OnStart unconditionally
// calls md.Config.DebugRunID()/DebugSessionID() for any span with Metadata
// set, both of which lock a *sync.RWMutex that's nil until sv2.InitConfig
// has run (true for state loaded through the normal store path, but this
// package must never crash the executor's critical path over a caller that
// built Metadata some other way — see listener_test.go, which does). Always
// use this instead of &md when building CreateSpanOptions.Metadata.
func safeMetadata(md sv2.Metadata) *sv2.Metadata {
	sv2.InitConfig(&md.Config)
	return &md
}
