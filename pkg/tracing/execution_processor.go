package tracing

import (
	"context"
	"time"

	"github.com/inngest/inngest/pkg/enums"
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
		qi:   qi,
		next: next,
	}
}

func (p *executionProcessor) OnStart(parent context.Context, s sdktrace.ReadWriteSpan) {
	attrs := []attribute.KeyValue{}
	now := time.Now() // TODO This should be something on qi etc

	if p.md != nil {
		attrs = append(attrs,
			attribute.String(meta.AttributeRunID, p.md.ID.RunID.String()),
		)
	} else if p.qi != nil {
		attrs = append(attrs,
			attribute.String(meta.AttributeRunID, p.qi.Identifier.RunID.String()),
		)
	}

	// Do not set extra contextual data on extension spans
	switch s.Name() {
	case meta.SpanNameRun:
		{
			attrs = append(attrs,
				attribute.Int64(meta.AttributeQueuedAt, now.UnixMilli()),
			)

			if p.md != nil {

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
			}

			break
		}

	case meta.SpanNameStepDiscovery:
		{
			attrs = append(attrs,
				attribute.Int64(meta.AttributeQueuedAt, now.UnixMilli()),
			)

			break
		}

	case meta.SpanNameStep:
		{
			attrs = append(attrs,
				attribute.Int64(meta.AttributeQueuedAt, now.UnixMilli()),
			)

			if p.qi != nil {
				attrs = append(attrs,
					attribute.Int(meta.AttributeStepMaxAttempts, p.qi.GetMaxAttempts()),
					attribute.Int(meta.AttributeStepAttempt, p.qi.Attempt),
				)

				// Some steps "start" as soon as they are queued
				startWhenQueued := p.qi.Kind == queue.KindSleep
				if !startWhenQueued {
					for _, attr := range s.Attributes() {
						if string(attr.Key) == meta.AttributeStepOp {
							if attr.Value.Type() == attribute.STRING && attr.Value.AsString() == enums.OpcodeWaitForEvent.String() {
								startWhenQueued = true
								break
							}
						}

					}
				}

				if startWhenQueued {
					attrs = append(attrs,
						attribute.Int64(meta.AttributeStartedAt, now.UnixMilli()),
					)
				}
			}

			break
		}

	case meta.SpanNameExecution:
		{
			attrs = append(attrs,
				attribute.Int64(meta.AttributeStartedAt, now.UnixMilli()),
			)

			if p.qi != nil {
				attrs = append(attrs,
					attribute.Int(meta.AttributeStepAttempt, p.qi.Attempt),
				)
			}

			break
		}

	case meta.SpanNameDynamicExtension:
		{
			for _, attr := range s.Attributes() {
				if string(attr.Key) == meta.AttributeDynamicStatus {
					if attr.Value.Type() == attribute.INT64 && enums.RunStatusEnded(enums.RunStatus(attr.Value.AsInt64())) {
						attrs = append(attrs,
							attribute.Int64(meta.AttributeEndedAt, now.UnixMilli()),
						)
					}

					break
				}
			}

			break
		}
	}

	s.SetAttributes(attrs...)
	p.next.OnStart(parent, s)
}

func (p *executionProcessor) OnEnd(s sdktrace.ReadOnlySpan) {
	// If the span isn't an extension span, judge if it should be dropped.
	if s.Name() != meta.SpanNameDynamicExtension {
		for _, attr := range s.Attributes() {
			if string(attr.Key) == meta.AttributeDropSpan && attr.Value.AsBool() {
				// Toggle this on and off to see or remove dropped spans
				return // Don't export dropped spans
			}
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
