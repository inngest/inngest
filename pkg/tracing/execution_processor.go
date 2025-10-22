package tracing

import (
	"context"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state/v2"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type _spanCtxKeyT struct{}

var _spanCtxKeyV _spanCtxKeyT

type ExecutionContext struct {
	Identifier  state.ID
	Attempt     int
	MaxAttempts *int
	// QueueKind is the queue kind string, eg. "sleep" - the type of job enqueued in our system.
	QueueKind string
}

// WithExecutionContext stores data for spans in context.
func WithExecutionContext(ctx context.Context, e ExecutionContext) context.Context {
	return context.WithValue(ctx, _spanCtxKeyV, &e)
}

func mixinExecutonContext(input context.Context, with context.Context) context.Context {
	ec := getExecutionContext(input)
	return context.WithValue(with, _spanCtxKeyV, ec)
}

type executionProcessor struct {
	md   *statev2.Metadata
	next sdktrace.SpanProcessor
}

func newExecutionProcessor(md *statev2.Metadata, next sdktrace.SpanProcessor) sdktrace.SpanProcessor {
	return &executionProcessor{
		md:   md,
		next: next,
	}
}

// AddMetadataTenantAttrs adds all attrs  from the metadata ID to the trace.
func AddMetadataTenantAttrs(rawAttrs *meta.SerializableAttrs, id statev2.ID) {
	meta.AddAttr(rawAttrs, meta.Attrs.RunID, &id.RunID)
	meta.AddAttr(rawAttrs, meta.Attrs.FunctionID, &id.FunctionID)
	meta.AddAttr(rawAttrs, meta.Attrs.AccountID, &id.Tenant.AccountID)
	meta.AddAttr(rawAttrs, meta.Attrs.EnvID, &id.Tenant.EnvID)
	meta.AddAttr(rawAttrs, meta.Attrs.AppID, &id.Tenant.AppID)
}

func getExecutionContext(ctx context.Context) *ExecutionContext {
	ec, ok := ctx.Value(_spanCtxKeyV).(*ExecutionContext)
	if ok {
		return ec
	}
	return nil
}

func (p *executionProcessor) OnStart(parent context.Context, s sdktrace.ReadWriteSpan) {
	rawAttrs := meta.NewAttrSet()
	now := s.StartTime()

	if p.md != nil {
		AddMetadataTenantAttrs(rawAttrs, p.md.ID)
		meta.AddAttr(rawAttrs, meta.Attrs.DebugRunID, p.md.Config.DebugRunID())
		meta.AddAttr(rawAttrs, meta.Attrs.DebugSessionID, p.md.Config.DebugSessionID())
	}

	ec := getExecutionContext(parent)

	if ec != nil {
		meta.AddAttr(rawAttrs, meta.Attrs.RunID, &ec.Identifier.RunID)
		meta.AddAttr(rawAttrs, meta.Attrs.FunctionID, &ec.Identifier.FunctionID)
		meta.AddAttr(rawAttrs, meta.Attrs.AccountID, &ec.Identifier.Tenant.AccountID)
		meta.AddAttr(rawAttrs, meta.Attrs.EnvID, &ec.Identifier.Tenant.EnvID)
		meta.AddAttr(rawAttrs, meta.Attrs.AppID, &ec.Identifier.Tenant.AppID)
	}

	// Do not set extra contextual data on extension spans
	switch s.Name() {
	case meta.SpanNameRun:
		{

			// The "queued at" time should always be the same time as the run ID.  This way,
			// we ensure there's no drift between run ID creation time and the queued at time.
			ts := now
			if p.md != nil {
				ts = p.md.ID.RunID.Timestamp()
			}

			meta.AddAttrIfUnset(rawAttrs, meta.Attrs.QueuedAt, &ts)

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
			meta.AddAttrIfUnset(rawAttrs, meta.Attrs.QueuedAt, &now)

			if ec != nil {
				meta.AddAttr(rawAttrs, meta.Attrs.StepMaxAttempts, ec.MaxAttempts)
				meta.AddAttr(rawAttrs, meta.Attrs.StepAttempt, &ec.Attempt)

				// Some steps "start" as soon as they are queued
				startWhenQueued := ec.QueueKind == queue.KindSleep
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
			meta.AddAttrIfUnset(rawAttrs, meta.Attrs.StartedAt, &now)

			if ec != nil {
				meta.AddAttr(rawAttrs, meta.Attrs.StepAttempt, &ec.Attempt)
			}

			break
		}

	case meta.SpanNameDynamicExtension:
		{
			var (
				ds *attribute.KeyValue
				ea *attribute.KeyValue
			)

			for _, attr := range s.Attributes() {
				a := attr

				switch string(attr.Key) {
				case meta.Attrs.DynamicStatus.Key():
					ds = &a
				case meta.Attrs.EndedAt.Key():
					ea = &a
				}
			}

			// Only overwrite the EndedAt time if we don't already have one
			if ea == nil && ds != nil {
				if ds.Value.Type() == attribute.INT64 && enums.RunStatusEnded(enums.RunStatus(ds.Value.AsInt64())) {
					meta.AddAttr(rawAttrs, meta.Attrs.EndedAt, &now)
				}
			}
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
