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
	"github.com/inngest/inngest/pkg/execution/state/v2"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/telemetry"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/attribute"
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
