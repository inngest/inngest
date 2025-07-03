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
	rawAttrs := meta.RawAttrs{}
	now := time.Now() // TODO This should be something on qi etc

	if p.md != nil {
		meta.AddRawAttr(&rawAttrs, meta.Attrs.RunID, &p.md.ID.RunID)
	} else if p.qi != nil {
		meta.AddRawAttr(&rawAttrs, meta.Attrs.RunID, &p.qi.Identifier.RunID)
	}

	// Do not set extra contextual data on extension spans
	switch s.Name() {
	case meta.SpanNameRun:
		{
			meta.AddRawAttr(&rawAttrs, meta.Attrs.QueuedAt, &now)

			if p.md != nil {

				eventIDs := make([]string, len(p.md.Config.EventIDs))
				for i, id := range p.md.Config.EventIDs {
					eventIDs[i] = id.String()
				}

				meta.AddRawAttr(&rawAttrs, meta.Attrs.FunctionID, &p.md.ID.FunctionID)
				meta.AddRawAttr(&rawAttrs, meta.Attrs.FunctionVersion, &p.md.Config.FunctionVersion)
				meta.AddRawAttr(&rawAttrs, meta.Attrs.EventIDs, &eventIDs)
				meta.AddRawAttr(&rawAttrs, meta.Attrs.AccountID, &p.md.ID.Tenant.AccountID)
				meta.AddRawAttr(&rawAttrs, meta.Attrs.EnvID, &p.md.ID.Tenant.EnvID)
				meta.AddRawAttr(&rawAttrs, meta.Attrs.AppID, &p.md.ID.Tenant.AppID)

				if p.md.Config.CronSchedule() != nil {
					meta.AddRawAttr(&rawAttrs, meta.Attrs.CronSchedule, p.md.Config.CronSchedule())
				}

				if p.md.Config.BatchID != nil {
					batchTS := time.UnixMilli(int64(p.md.Config.BatchID.Time()))
					meta.AddRawAttr(&rawAttrs, meta.Attrs.BatchID, p.md.Config.BatchID)
					meta.AddRawAttr(&rawAttrs, meta.Attrs.BatchTimestamp, &batchTS)
				}
			}

			break
		}

	case meta.SpanNameStepDiscovery:
		{
			meta.AddRawAttr(&rawAttrs, meta.Attrs.QueuedAt, &now)

			break
		}

	case meta.SpanNameStep:
		{
			meta.AddRawAttr(&rawAttrs, meta.Attrs.QueuedAt, &now)

			if p.qi != nil {
				meta.AddRawAttr(&rawAttrs, meta.Attrs.StepMaxAttempts, p.qi.MaxAttempts)
				meta.AddRawAttr(&rawAttrs, meta.Attrs.StepAttempt, &p.qi.Attempt)

				// Some steps "start" as soon as they are queued
				startWhenQueued := p.qi.Kind == queue.KindSleep
				if !startWhenQueued {
					for _, attr := range s.Attributes() {
						if string(attr.Key) == meta.Attrs.StepOp.Key() {
							if attr.Value.Type() == attribute.STRING && attr.Value.AsString() == enums.OpcodeWaitForEvent.String() {
								startWhenQueued = true
								break
							}
						}

					}
				}

				if startWhenQueued {
					meta.AddRawAttr(&rawAttrs, meta.Attrs.StartedAt, &now)
				}
			}

			break
		}

	case meta.SpanNameExecution:
		{
			meta.AddRawAttr(&rawAttrs, meta.Attrs.StartedAt, &now)

			if p.qi != nil {
				meta.AddRawAttr(&rawAttrs, meta.Attrs.StepAttempt, &p.qi.Attempt)
			}

			break
		}

	case meta.SpanNameDynamicExtension:
		{
			for _, attr := range s.Attributes() {
				if string(attr.Key) == meta.Attrs.DynamicStatus.Key() {
					if attr.Value.Type() == attribute.INT64 && enums.RunStatusEnded(enums.RunStatus(attr.Value.AsInt64())) {
						meta.AddRawAttr(&rawAttrs, meta.Attrs.EndedAt, &now)
					}

					break
				}
			}

			break
		}
	}

	s.SetAttributes(rawAttrs.Serialize()...)
	p.next.OnStart(parent, s)
}

func (p *executionProcessor) OnEnd(s sdktrace.ReadOnlySpan) {
	// If the span isn't an extension span, judge if it should be dropped.
	if s.Name() != meta.SpanNameDynamicExtension {
		for _, attr := range s.Attributes() {
			if string(attr.Key) == meta.Attrs.DropSpan.Key() && attr.Value.AsBool() {
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
