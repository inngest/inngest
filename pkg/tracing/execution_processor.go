package tracing

import (
	"context"

	"github.com/inngest/inngest/pkg/execution/queue"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type executionProcessor struct {
	md   *statev2.Metadata
	qi   *queue.Item
	next sdktrace.SpanProcessor
}

func newExecutionProcessor(md *statev2.Metadata, qi *queue.Item, next sdktrace.SpanProcessor) sdktrace.SpanProcessor {
	return &executionProcessor{
		md:   md,
		next: next,
	}
}

func (p *executionProcessor) OnStart(parent context.Context, s sdktrace.ReadWriteSpan) {
	attrs := []attribute.KeyValue{}

	if p.md != nil {
		attrs = append(attrs,
			attribute.String(meta.AttributeRunID, p.md.ID.RunID.String()),
		)
	}

	switch s.Name() {
	case meta.SpanNameRun:
		{
			if p.md == nil {
				break
			}

			eventIDs := make([]string, len(p.md.Config.EventIDs))
			for i, id := range p.md.Config.EventIDs {
				eventIDs[i] = id.String()
			}

			attrs = append(attrs,
				attribute.String(meta.AttributeFunctionID, p.md.ID.FunctionID.String()),
				attribute.Int(meta.AttributeFunctionVersion, p.md.Config.FunctionVersion),
				attribute.StringSlice(meta.AttributeEventIDs, eventIDs),
				attribute.String(meta.AttributeAccountID, p.md.ID.Tenant.AccountID.String()),
				attribute.String(meta.AttributeEnvID, p.md.ID.Tenant.EnvID.String()),
				attribute.String(meta.AttributeAppID, p.md.ID.Tenant.AppID.String()),
				attribute.Bool(meta.AttributeDynamicSpanID, true),
			)

			if p.md.Config.CronSchedule() != nil {
				attrs = append(attrs,
					attribute.String(meta.AttributeCronSchedule, *p.md.Config.CronSchedule()),
				)
			}

			if p.md.Config.BatchID != nil {
				attrs = append(attrs,
					attribute.String(meta.AttributeBatchID, p.md.Config.BatchID.String()),
					attribute.Int64(meta.AttributeBatchTimestamp, int64(p.md.Config.BatchID.Time())),
				)
			}

			break
		}

	case meta.SpanNameStep:
		{
			attrs = append(attrs,
				attribute.Bool(meta.AttributeDynamicSpanID, true),
			)

			if p.qi != nil {
				attrs = append(attrs,
					attribute.Int(meta.AttributeStepMaxAttempts, p.qi.GetMaxAttempts()),
					attribute.Int(meta.AttributeStepAttempt, p.qi.Attempt),
				)
			}

			break
		}

	case meta.SpanNameExecution:
		{
			break
		}
	}

	s.SetAttributes(attrs...)
	p.next.OnStart(parent, s)
}

func (p *executionProcessor) OnEnd(s sdktrace.ReadOnlySpan) {
	for _, attr := range s.Attributes() {
		if string(attr.Key) == meta.AttributeDropSpan && attr.Value.AsBool() {
			// Toggle this on and off to see or remove dropped spans
			return // Don't export dropped spans
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
