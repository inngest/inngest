package run

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev1 "github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/v2"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/telemetry"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func NewTraceLifecycleListener(l *slog.Logger, run sv2.RunService) execution.LifecycleListener {
	if l == nil {
		l = slog.Default()
	}

	return traceLifecycle{
		log: l,
		run: run,
	}
}

type traceLifecycle struct {
	execution.NoopLifecyceListener

	log *slog.Logger
	run sv2.RunService
}

func (l traceLifecycle) OnFunctionScheduled(ctx context.Context, md state.Metadata, item queue.Item, evts []event.TrackedEvent) {
	runID := md.ID.RunID
	evtIDs := []string{}
	for _, e := range evts {
		id := e.GetInternalID()
		evtIDs = append(evtIDs, id.String())
	}

	// span that tells when the function was queued
	_, span := telemetry.NewSpan(ctx,
		telemetry.WithScope(consts.OtelScopeTrigger),
		telemetry.WithName(consts.OtelSpanTrigger),
		telemetry.WithTimestamp(ulid.Time(runID.Time())),
		telemetry.WithSpanAttributes(
			attribute.Bool(consts.OtelUserTraceFilterKey, true),
			attribute.String(consts.OtelSysAccountID, md.ID.Tenant.AccountID.String()),
			attribute.String(consts.OtelSysWorkspaceID, md.ID.Tenant.EnvID.String()),
			attribute.String(consts.OtelSysAppID, md.ID.Tenant.AppID.String()),
			attribute.String(consts.OtelSysFunctionID, md.ID.FunctionID.String()),
			attribute.String(consts.OtelSysFunctionSlug, md.Config.FunctionSlug()),
			attribute.Int(consts.OtelSysFunctionVersion, md.Config.FunctionVersion),
			attribute.String(consts.OtelAttrSDKRunID, runID.String()),
			attribute.Int64(consts.OtelSysFunctionStatusCode, enums.RunStatusScheduled.ToCode()),
			attribute.String(consts.OtelSysEventIDs, strings.Join(evtIDs, ",")),
		),
	)
	defer span.End()

	schedule := md.Config.CronSchedule()
	if schedule != nil {
		span.SetAttributes(attribute.String(consts.OtelSysCronExpr, *schedule))
	}

	batchID := md.Config.BatchID
	if batchID != nil {
		span.SetAttributes(
			attribute.String(consts.OtelSysBatchID, batchID.String()),
			attribute.Int64(consts.OtelSysBatchTS, int64(batchID.Time())),
		)
	}
	if md.Config.DebounceFlag() {
		span.SetAttributes(attribute.Bool(consts.OtelSysDebounceTimeout, true))
	}
	if md.Config.Context != nil {
		if val, ok := md.Config.Context[consts.OtelPropagationLinkKey]; ok {
			if link, ok := val.(string); ok {
				span.SetAttributes(attribute.String(consts.OtelPropagationLinkKey, link))
			}
		}
	}

	for _, e := range evts {
		evt := e.GetEvent()
		// serialize event data to the span
		if byt, err := json.Marshal(evt); err == nil {
			span.AddEvent(string(byt), trace.WithAttributes(
				attribute.Bool(consts.OtelSysEventData, true),
			))
		}
	}
}

func (l traceLifecycle) OnFunctionStarted(
	ctx context.Context,
	md sv2.Metadata,
	item queue.Item,
	evts []json.RawMessage,
) {
	// reassign here to make sure we have the right traceID and such
	ctx = l.extractTraceCtx(ctx, md, &item)

	start := time.Now()
	if !md.Config.StartedAt.IsZero() {
		start = md.Config.StartedAt
	}

	// spanID should always exists
	spanID, err := md.Config.GetSpanID()
	if err != nil {
		// generate a new one here to be used for subsequent runs.
		// this could happen for runs that started before this feature was introduced.
		sid := telemetry.NewSpanID(ctx)
		spanID = &sid
	}

	runID := md.ID.RunID
	slug := md.Config.FunctionSlug()

	evtIDs := make([]string, len(md.Config.EventIDs))
	for i, e := range md.Config.EventIDs {
		evtIDs[i] = e.String()
	}

	// (re)Construct function span to force update the end time
	_, span := telemetry.NewSpan(ctx,
		telemetry.WithScope(consts.OtelScopeFunction),
		telemetry.WithName(slug),
		telemetry.WithTimestamp(start),
		telemetry.WithSpanID(*spanID),
		telemetry.WithSpanAttributes(
			attribute.Bool(consts.OtelUserTraceFilterKey, true),
			attribute.String(consts.OtelSysAccountID, md.ID.Tenant.AccountID.String()),
			attribute.String(consts.OtelSysWorkspaceID, md.ID.Tenant.EnvID.String()),
			attribute.String(consts.OtelSysAppID, md.ID.Tenant.AppID.String()),
			attribute.String(consts.OtelSysFunctionID, md.ID.FunctionID.String()),
			attribute.String(consts.OtelSysFunctionSlug, slug),
			attribute.Int(consts.OtelSysFunctionVersion, md.Config.FunctionVersion),
			attribute.String(consts.OtelAttrSDKRunID, runID.String()),
			attribute.String(consts.OtelSysEventIDs, strings.Join(evtIDs, ",")),
			attribute.String(consts.OtelSysIdempotencyKey, md.Config.Idempotency),
			attribute.Int64(consts.OtelSysFunctionStatusCode, enums.RunStatusRunning.ToCode()),
			attribute.Bool(consts.OtelSysStepFirst, true),
		),
	)
	defer span.End()

	schedule := md.Config.CronSchedule()
	if schedule != nil {
		span.SetAttributes(attribute.String(consts.OtelSysCronExpr, *schedule))
	}
	batchID := md.Config.BatchID
	if batchID != nil {
		span.SetAttributes(
			attribute.String(consts.OtelSysBatchID, batchID.String()),
			attribute.Int64(consts.OtelSysBatchTS, int64(batchID.Time())),
		)
	}
	if md.Config.TraceLink() != nil {
		span.SetAttributes(attribute.String(consts.OtelSysFunctionLink, *md.Config.TraceLink()))
	}

	for _, e := range evts {
		span.AddEvent(string(e), trace.WithAttributes(
			attribute.Bool(consts.OtelSysEventData, true),
		))
	}
}

func (l traceLifecycle) OnFunctionFinished(
	ctx context.Context,
	md state.Metadata,
	item queue.Item,
	evts []json.RawMessage,
	resp statev1.DriverResponse,
) {
	// reassign here to make sure we have the right traceID and such
	ctx = l.extractTraceCtx(ctx, md, &item)

	start := time.Now()
	if !md.Config.StartedAt.IsZero() {
		start = md.Config.StartedAt
	}

	// spanID should always exists
	spanID, err := md.Config.GetSpanID()
	if err != nil {
		// generate a new one here to be used for subsequent runs.
		// this could happen for runs that started before this feature was introduced.
		sid := telemetry.NewSpanID(ctx)
		spanID = &sid
	}

	runID := md.ID.RunID
	slug := md.Config.FunctionSlug()

	evtIDs := make([]string, len(md.Config.EventIDs))
	for i, e := range md.Config.EventIDs {
		evtIDs[i] = e.String()
	}

	// (re)Construct function span to force update the end time
	_, span := telemetry.NewSpan(ctx,
		telemetry.WithScope(consts.OtelScopeFunction),
		telemetry.WithName(slug),
		telemetry.WithTimestamp(start),
		telemetry.WithSpanID(*spanID),
		telemetry.WithSpanAttributes(
			attribute.Bool(consts.OtelUserTraceFilterKey, true),
			attribute.String(consts.OtelSysAccountID, md.ID.Tenant.AccountID.String()),
			attribute.String(consts.OtelSysWorkspaceID, md.ID.Tenant.EnvID.String()),
			attribute.String(consts.OtelSysAppID, md.ID.Tenant.AppID.String()),
			attribute.String(consts.OtelSysFunctionID, md.ID.FunctionID.String()),
			attribute.String(consts.OtelSysFunctionSlug, slug),
			attribute.Int(consts.OtelSysFunctionVersion, md.Config.FunctionVersion),
			attribute.String(consts.OtelAttrSDKRunID, runID.String()),
			attribute.String(consts.OtelSysEventIDs, strings.Join(evtIDs, ",")),
			attribute.String(consts.OtelSysIdempotencyKey, md.Config.Idempotency),
			attribute.Bool(consts.OtelSysStepFirst, true),
		),
	)
	defer span.End()

	schedule := md.Config.CronSchedule()
	if schedule != nil {
		span.SetAttributes(attribute.String(consts.OtelSysCronExpr, *schedule))
	}
	batchID := md.Config.BatchID
	if batchID != nil {
		span.SetAttributes(
			attribute.String(consts.OtelSysBatchID, batchID.String()),
			attribute.Int64(consts.OtelSysBatchTS, int64(batchID.Time())),
		)
	}
	if md.Config.TraceLink() != nil {
		span.SetAttributes(attribute.String(consts.OtelSysFunctionLink, *md.Config.TraceLink()))
	}

	for _, e := range evts {
		span.AddEvent(string(e), trace.WithAttributes(
			attribute.Bool(consts.OtelSysEventData, true),
		))
	}

	switch resp.StatusCode {
	case 200:
		span.SetStatus(codes.Ok, "success")
		span.SetAttributes(attribute.Int64(consts.OtelSysFunctionStatusCode, enums.RunStatusCompleted.ToCode()))
	default: // everything else are errors
		span.SetStatus(codes.Error, resp.Error())
		span.SetAttributes(attribute.Int64(consts.OtelSysFunctionStatusCode, enums.RunStatusFailed.ToCode()))
	}

	span.SetFnOutput(resp.Output)
}

func (l traceLifecycle) OnFunctionCancelled(ctx context.Context, md sv2.Metadata, req execution.CancelRequest, evts []json.RawMessage) {
	start := time.Now()
	if !md.Config.StartedAt.IsZero() {
		start = md.Config.StartedAt
	}

	fnSpanID, err := md.Config.GetSpanID()
	if err != nil {
		l.log.Error("error retrieving spanID for cancelled function run",
			"err", err,
			"identifier", md.ID,
		)
		return
	}

	evtIDs := make([]string, len(md.Config.EventIDs))
	for i, eid := range md.Config.EventIDs {
		evtIDs[i] = eid.String()
	}

	_, span := telemetry.NewSpan(ctx,
		telemetry.WithScope(consts.OtelScopeFunction),
		telemetry.WithName(md.Config.FunctionSlug()),
		telemetry.WithTimestamp(start),
		telemetry.WithSpanID(*fnSpanID),
		telemetry.WithSpanAttributes(
			attribute.Bool(consts.OtelUserTraceFilterKey, true),
			attribute.String(consts.OtelSysAccountID, md.ID.Tenant.AccountID.String()),
			attribute.String(consts.OtelSysWorkspaceID, md.ID.Tenant.EnvID.String()),
			attribute.String(consts.OtelSysAppID, md.ID.Tenant.AppID.String()),
			attribute.String(consts.OtelSysFunctionID, md.ID.FunctionID.String()),
			attribute.String(consts.OtelSysFunctionSlug, md.Config.FunctionSlug()),
			attribute.Int(consts.OtelSysFunctionVersion, md.Config.FunctionVersion),
			attribute.String(consts.OtelAttrSDKRunID, md.ID.RunID.String()),
			attribute.String(consts.OtelSysEventIDs, strings.Join(evtIDs, ",")),
			attribute.String(consts.OtelSysIdempotencyKey, md.IdempotencyKey()),
			attribute.Int64(consts.OtelSysFunctionStatusCode, enums.RunStatusCancelled.ToCode()),
		),
	)
	defer span.End()

	if md.Config.CronSchedule() != nil {
		span.SetAttributes(attribute.String(consts.OtelSysCronExpr, *md.Config.CronSchedule()))
	}
	if md.Config.BatchID != nil {
		span.SetAttributes(
			attribute.String(consts.OtelSysBatchID, md.Config.BatchID.String()),
			attribute.Int64(consts.OtelSysBatchTS, int64(md.Config.BatchID.Time())),
		)
	}

	for _, evt := range evts {
		span.AddEvent(string(evt), trace.WithAttributes(
			attribute.Bool(consts.OtelSysEventData, true),
		))
	}
}

// NOTE: this is copied from the same function inside executor.
// should probably delete it some time when it's no longer needed.
//
// extractTraceCtx extracts the trace context from the given item, if it exists.
// If it doesn't it falls back to extracting the trace for the run overall.
// If neither exist or they are invalid, it returns the original context.
func (l *traceLifecycle) extractTraceCtx(ctx context.Context, md sv2.Metadata, item *queue.Item) context.Context {
	fntrace := md.Config.FunctionTrace()
	if fntrace != nil {
		// NOTE:
		// this gymastics happens because the carrier stores the spanID separately.
		// it probably can be simplified
		tmp := telemetry.UserTracer().Propagator().Extract(ctx, propagation.MapCarrier(fntrace.Context))
		sctx := trace.SpanContextFromContext(tmp).WithSpanID(fntrace.SpanID())
		return trace.ContextWithSpanContext(ctx, sctx)
	}

	if item != nil {
		metadata := make(map[string]any)
		for k, v := range item.Metadata {
			metadata[k] = v
		}
		if newCtx, ok := extractTraceCtxFromMap(ctx, metadata); ok {
			return newCtx
		}
	}

	if md.Config.Context != nil {
		if newCtx, ok := extractTraceCtxFromMap(ctx, md.Config.Context); ok {
			return newCtx
		}
	}

	return ctx
}

// extractTraceCtxFromMap extracts the trace context from a map, if it exists.
// If it doesn't or it is invalid, it nil.
func extractTraceCtxFromMap(ctx context.Context, target map[string]any) (context.Context, bool) {
	if trace, ok := target[consts.OtelPropagationKey]; ok {
		carrier := telemetry.NewTraceCarrier()
		if err := carrier.Unmarshal(trace); err == nil {
			targetCtx := telemetry.UserTracer().Propagator().Extract(ctx, propagation.MapCarrier(carrier.Context))
			return targetCtx, true
		}
	}

	return ctx, false
}
