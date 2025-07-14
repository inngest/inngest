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
	rawAttrs := meta.NewAttrSet()
	now := time.Now() // TODO This should be something on qi etc

	if p.md != nil {
		meta.AddAttr(rawAttrs, meta.Attrs.RunID, &p.md.ID.RunID)
		meta.AddAttr(rawAttrs, meta.Attrs.FunctionID, &p.md.ID.FunctionID)
		meta.AddAttr(rawAttrs, meta.Attrs.AccountID, &p.md.ID.Tenant.AccountID)
		meta.AddAttr(rawAttrs, meta.Attrs.EnvID, &p.md.ID.Tenant.EnvID)
		meta.AddAttr(rawAttrs, meta.Attrs.AppID, &p.md.ID.Tenant.AppID)
	} else if p.qi != nil {
		meta.AddAttr(rawAttrs, meta.Attrs.RunID, &p.qi.Identifier.RunID)
		meta.AddAttr(rawAttrs, meta.Attrs.FunctionID, &p.qi.Identifier.WorkflowID)
		meta.AddAttr(rawAttrs, meta.Attrs.AccountID, &p.qi.Identifier.AccountID)
		meta.AddAttr(rawAttrs, meta.Attrs.EnvID, &p.qi.Identifier.WorkspaceID)
		meta.AddAttr(rawAttrs, meta.Attrs.AppID, &p.qi.Identifier.AppID)
	}

	// Do not set extra contextual data on extension spans
	switch s.Name() {
	case meta.SpanNameRun:
		{
			meta.AddAttr(rawAttrs, meta.Attrs.QueuedAt, &now)

			if p.md != nil {

				eventIDs := make([]string, len(p.md.Config.EventIDs))
				for i, id := range p.md.Config.EventIDs {
					eventIDs[i] = id.String()
				}

				meta.AddAttr(rawAttrs, meta.Attrs.FunctionVersion, &p.md.Config.FunctionVersion)
				meta.AddAttr(rawAttrs, meta.Attrs.EventIDs, &eventIDs)

				if p.md.Config.CronSchedule() != nil {
					meta.AddAttr(rawAttrs, meta.Attrs.CronSchedule, p.md.Config.CronSchedule())
				}

				if p.md.Config.BatchID != nil {
					batchTS := time.UnixMilli(int64(p.md.Config.BatchID.Time()))
					meta.AddAttr(rawAttrs, meta.Attrs.BatchID, p.md.Config.BatchID)
					meta.AddAttr(rawAttrs, meta.Attrs.BatchTimestamp, &batchTS)
				}
			}

			break
		}

	case meta.SpanNameStepDiscovery:
		{
			meta.AddAttr(rawAttrs, meta.Attrs.QueuedAt, &now)

			break
		}

	case meta.SpanNameStep:
		{
			meta.AddAttr(rawAttrs, meta.Attrs.QueuedAt, &now)

			if p.qi != nil {
				meta.AddAttr(rawAttrs, meta.Attrs.StepMaxAttempts, p.qi.MaxAttempts)
				meta.AddAttr(rawAttrs, meta.Attrs.StepAttempt, &p.qi.Attempt)

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
					meta.AddAttr(rawAttrs, meta.Attrs.StartedAt, &now)
				}
			}

			break
		}

	case meta.SpanNameExecution:
		{
			meta.AddAttr(rawAttrs, meta.Attrs.StartedAt, &now)

			if p.qi != nil {
				meta.AddAttr(rawAttrs, meta.Attrs.StepAttempt, &p.qi.Attempt)
			}

			break
		}

	case meta.SpanNameDynamicExtension:
		{
			for _, attr := range s.Attributes() {
				if string(attr.Key) == meta.Attrs.DynamicStatus.Key() {
					if attr.Value.Type() == attribute.INT64 && enums.RunStatusEnded(enums.RunStatus(attr.Value.AsInt64())) {
						meta.AddAttr(rawAttrs, meta.Attrs.EndedAt, &now)
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
