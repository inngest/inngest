package executor

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func NewTraceRunLifecycleListener(l *slog.Logger, run sv2.RunService) execution.LifecycleListener {
	if l == nil {
		l = slog.Default()
	}

	return lifecycle{
		log: l,
		run: run,
	}
}

type lifecycle struct {
	execution.NoopLifecyceListener

	log *slog.Logger
	run sv2.RunService
}

func (l lifecycle) OnFunctionCancelled(ctx context.Context, md sv2.Metadata, req execution.CancelRequest) {
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

	events, err := l.run.LoadEvents(ctx, md.ID)
	if err != nil {
		l.log.Error("error loading events from state on function cancelled for trace runs",
			"err", err,
			"identifier", md.ID,
		)
	}

	for _, evt := range events {
		span.AddEvent(string(evt), trace.WithAttributes(
			attribute.Bool(consts.OtelSysEventData, true),
		))
	}
}
