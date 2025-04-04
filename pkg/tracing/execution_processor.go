package tracing

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/inngest/inngest/pkg/execution/queue"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
)

type executionProcessor struct {
	md   statev2.Metadata
	qi   queue.Item
	next sdktrace.SpanProcessor
}

func newExecutionProcessor(md statev2.Metadata, qi queue.Item, next sdktrace.SpanProcessor) sdktrace.SpanProcessor {
	return &executionProcessor{
		md:   md,
		qi:   qi,
		next: next,
	}
}

func (p *executionProcessor) OnStart(parent context.Context, s sdktrace.ReadWriteSpan) {
	attrs := []attribute.KeyValue{
		attribute.String(AttributeRunID, p.md.ID.RunID.String()),
	}

	switch s.Name() {
	case SpanNameRun:
		{
			eventIDs := make([]string, len(p.md.Config.EventIDs))
			for i, id := range p.md.Config.EventIDs {
				eventIDs[i] = id.String()
			}

			attrs = append(attrs,
				attribute.String(AttributeFunctionID, p.md.ID.FunctionID.String()),
				attribute.Int(AttributeFunctionVersion, p.md.Config.FunctionVersion),
				attribute.StringSlice(AttributeEventIDs, eventIDs),
				attribute.String(AttributeAccountID, p.md.ID.Tenant.AccountID.String()),
				attribute.String(AttributeEnvID, p.md.ID.Tenant.EnvID.String()),
				attribute.String(AttributeAppID, p.md.ID.Tenant.AppID.String()),
			)

			if p.md.Config.CronSchedule() != nil {
				attrs = append(attrs,
					attribute.String(AttributeCronSchedule, *p.md.Config.CronSchedule()),
				)
			}

			if p.md.Config.BatchID != nil {
				attrs = append(attrs,
					attribute.String(AttributeBatchID, p.md.Config.BatchID.String()),
					attribute.Int64(AttributeBatchTimestamp, int64(p.md.Config.BatchID.Time())),
				)
			}
			break
		}

	case SpanNameStep:
		{
			break
		}

	case SpanNameExecution:
		{
			break
		}
	}

	s.SetAttributes(attrs...)
	p.next.OnStart(parent, s)
}

func (p *executionProcessor) OnEnd(s sdktrace.ReadOnlySpan) {
	for _, attr := range s.Attributes() {
		if string(attr.Key) == AttributeExecutionIsDiscovery && attr.Value.AsBool() {
			return // Don't export discovery spans
		}
	}

	p.next.OnEnd(s)
}

func (p *executionProcessor) Shutdown(ctx context.Context) error {
	return p.next.Shutdown(ctx)
}

func (p *executionProcessor) ForceFlush(ctx context.Context) error {
	return p.next.ForceFlush(ctx)
}
