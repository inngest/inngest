// Package v3 provides a second, narrower TracerProvider implementation.
// Unlike pkg/tracing's own otelTracerProvider (behind tracing.TracerProvider,
// which this package deliberately does not implement), its span pipeline
// never wires in pkg/tracing's executionProcessor, and it has no droppable-
// span behavior: CreateSpan always creates and immediately ends a span,
// there's no CreateDroppableSpan/Drop()/Send() to defer that decision.
//
// executionProcessor leans on ambient, context-derived state
// (ExecutionContext, set only by pkg/execution/executor via
// tracing.WithExecutionContext) and per-span-name special cases (see
// pkg/tracing/execution_processor.go's OnStart) that only make sense for the
// executor's own real span stack. Callers using this package's
// TracerProvider are expected to set every attribute the processor would
// have (tenant/debug attrs, per-span-name extras) directly on each
// tracing.CreateSpanOptions.Attributes themselves — see
// pkg/execution/dualwrite/tracing.go's addTenantAndDebugAttrs/
// addRunSpanAttrs, its only caller.
package v3

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/inngest/version"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// propagator handles this package's own W3C traceparent/tracestate
// encoding and decoding. Deliberately a separate instance from
// pkg/tracing's own defaultPropagator, even though it's configured
// identically — the two TracerProvider implementations don't share state,
// so a change to one's propagation setup can't silently affect the other.
var propagator = propagation.NewCompositeTextMapPropagator(
	propagation.TraceContext{},
	propagation.Baggage{},
)

// TracerProvider is a narrower counterpart to tracing.TracerProvider: no
// CreateDroppableSpan, since nothing in this package's only caller
// (pkg/execution/dualwrite) ever creates a span it might not send — every
// span it creates is unconditionally recorded.
type TracerProvider interface {
	CreateSpan(ctx context.Context, name string, opts *tracing.CreateSpanOptions) (*meta.SpanReference, error)
	UpdateSpan(ctx context.Context, opts *tracing.UpdateSpanOptions) error
}

// otelTracerProvider implements TracerProvider — see this package's doc
// comment.
type otelTracerProvider struct {
	exp sdktrace.SpanExporter
	bt  time.Duration
}

// NewOtelTracerProvider returns a TracerProvider whose span pipeline skips
// pkg/tracing's executionProcessor entirely — see this package's doc
// comment.
func NewOtelTracerProvider(exp sdktrace.SpanExporter, batchTimeout time.Duration) TracerProvider {
	return &otelTracerProvider{
		exp: exp,
		bt:  batchTimeout,
	}
}

func (tp *otelTracerProvider) getTracer() trace.Tracer {
	otelTP := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(tp.exp)),
		sdktrace.WithIDGenerator(tracing.IDGenerator),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	return otelTP.Tracer(
		"inngest",
		trace.WithInstrumentationVersion(version.Print()),
	)
}

// CreateSpan creates a span and immediately ends it — see this package's
// doc comment on why there's no droppable/deferred-end variant.
func (tp *otelTracerProvider) CreateSpan(
	ctx context.Context,
	name string,
	opts *tracing.CreateSpanOptions,
) (*meta.SpanReference, error) {
	attrs := opts.Attributes
	if attrs == nil {
		attrs = meta.NewAttrSet()
	}

	st := opts.StartTime
	if st.IsZero() {
		st = time.Now()
	} else {
		meta.AddAttrIfUnset(attrs, meta.Attrs.StartedAt, &st)
	}

	if opts.Parent != nil {
		carrier := propagation.MapCarrier{
			"traceparent": opts.Parent.TraceParent,
			"tracestate":  opts.Parent.TraceState,
		}
		// A fresh context, carrying only what the parent's traceparent
		// encodes — unlike pkg/tracing's own otelTracerProvider, there's no
		// ExecutionContext to mix in here, since nothing downstream of this
		// package's span pipeline (no executionProcessor) ever reads it.
		ctx = propagator.Extract(context.Background(), carrier)
	} else {
		// Use a fresh context for parent traces so that there's no pollution from any
		// other tracing.
		ctx = context.Background()
	}

	if opts.Debug != nil {
		if opts.Debug.Location != "" {
			meta.AddAttr(attrs, meta.Attrs.InternalLocation, &opts.Debug.Location)
		}
	}
	if !opts.StartTime.IsZero() {
		meta.AddAttrIfUnset(attrs, meta.Attrs.StartedAt, &opts.StartTime)
	}
	if !opts.EndTime.IsZero() {
		meta.AddAttrIfUnset(attrs, meta.Attrs.EndedAt, &opts.EndTime)
	}

	spanOptions := append(
		[]trace.SpanStartOption{
			trace.WithAttributes(attrs.Serialize()...),
			trace.WithTimestamp(st),
		},
		opts.RawOtelSpanOptions...,
	)

	spanOptions = append(spanOptions, trace.WithSpanKind(trace.SpanKindServer))

	if opts.FollowsFrom != nil {
		spanOptions = append(
			spanOptions,
			trace.WithLinks(trace.Link{
				SpanContext: tracing.SpanContextFromMetadata(opts.FollowsFrom),
				Attributes: []attribute.KeyValue{
					attribute.String(meta.LinkAttributeType, meta.LinkAttributeTypeFollowsFrom),
				},
			}),
		)
	}

	// IF THERE IS SEED, we're creating something with deterministic span and trace IDs.
	// YAY.  We love determinism.  This is important for eg. root spans.
	if len(opts.Seed) > 0 {
		ctx = tracing.SetDeterministicIDs(ctx, tracing.DeterministicSpanConfig(opts.Seed))
	} else if opts.SpanID.IsValid() && opts.Parent != nil {
		ctx = tracing.SetDeterministicIDs(ctx, tracing.DeterministicIDs{
			TraceID: trace.SpanContextFromContext(ctx).TraceID(),
			SpanID:  opts.SpanID,
		})
	}

	tracer := tp.getTracer()
	ctx, span := tracer.Start(ctx, name, spanOptions...)

	carrier := propagation.MapCarrier{}
	propagator.Inject(ctx, carrier)
	refTp := carrier["traceparent"]
	refTs := carrier["tracestate"]

	spanRef := &meta.SpanReference{
		TraceParent: refTp,
		TraceState:  refTs,
	}

	spanRef.DynamicSpanID = span.SpanContext().SpanID().String()
	if opts.DynamicSpanIDOverride != "" {
		spanRef.DynamicSpanID = opts.DynamicSpanIDOverride
	}

	if opts.Parent != nil {
		// If the span has a parent, set some attributes so we can extend it later
		// and pick the same trace and parent span IDs for the extension span.
		spanRef.DynamicSpanTraceParent = opts.Parent.TraceParent
		spanRef.DynamicSpanTraceState = opts.Parent.TraceState
	} else {
		// If we don't have a parent, this is a top-level span (e.g. the run
		// span), so we use this span as the dynamic reference instead.
		//
		// In this case, we forcibly set the span ID part of the traceparent
		// to the expected zero value, to be the same as the top-level span.
		// e.g. for "00-c0b6b7b1d103cd383d594e9ffa128965-930c339a6dbccb41-01",
		// produce "00-c0b6b7b1d103cd383d594e9ffa128965-0000000000000000-01"
		splitRefTp := strings.Split(refTp, "-")
		if len(splitRefTp) != 4 {
			return nil, fmt.Errorf("invalid traceparent format when setting dynamic span data: %q", refTp)
		}
		splitRefTp[2] = "0000000000000000"

		spanRef.DynamicSpanTraceParent = strings.Join(splitRefTp, "-")
		spanRef.DynamicSpanTraceState = refTs
	}

	span.SetAttributes(
		attribute.String(meta.Attrs.DynamicSpanID.Key(), spanRef.DynamicSpanID),
	)

	// End at opts.EndTime when the caller gave one — e.g. a step span whose
	// real start/end (gen.Timing.Start()/End()) is already known at creation
	// time — falling back to "now" (trace.Span.End()'s own default) for
	// point-in-time markers that never set EndTime.
	var endOpts []trace.SpanEndOption
	if !opts.EndTime.IsZero() {
		endOpts = append(endOpts, trace.WithTimestamp(opts.EndTime))
	}

	if len(opts.Carriers) > 0 {
		byt, err := json.Marshal(spanRef)
		if err != nil {
			span.End(endOpts...)
			return nil, fmt.Errorf("failed to marshal span metadata when injecting to carriers: %w", err)
		}

		for _, carrier := range opts.Carriers {
			carrier[meta.PropagationKey] = string(byt)
		}
	}

	span.End(endOpts...)

	return spanRef, nil
}

// UpdateSpan extends an existing span with a new dynamic-extension span.
// Returns nothing, as the span is only extended and no further context is
// given.
func (tp *otelTracerProvider) UpdateSpan(
	ctx context.Context,
	opts *tracing.UpdateSpanOptions,
) error {
	ts := opts.EndTime
	if ts.IsZero() {
		ts = time.Now()
	}

	if opts.TargetSpan == nil {
		return fmt.Errorf("no target span")
	}

	if opts.TargetSpan.DynamicSpanID == "" {
		return fmt.Errorf("target span is not dynamic; has no DynamicSpanID")
	}

	attrs := meta.NewAttrSet(
		meta.Attr(meta.Attrs.DynamicSpanID, &opts.TargetSpan.DynamicSpanID),
	)
	if opts.Status != enums.StepStatusUnknown {
		meta.AddAttr(attrs, meta.Attrs.DynamicStatus, &opts.Status)
	}

	if opts.TargetSpan.DynamicSpanTraceParent != "" {
		splitTp := strings.Split(opts.TargetSpan.DynamicSpanTraceParent, "-")
		if len(splitTp) != 4 {
			attrs.AddErr(fmt.Errorf("invalid traceparent format when setting dynamic span data: %q", opts.TargetSpan.DynamicSpanTraceParent))
		} else {
			meta.AddAttr(attrs, meta.Attrs.DynamicTraceID, &splitTp[1])
		}
	}

	carrier := propagation.MapCarrier{
		"traceparent": opts.TargetSpan.DynamicSpanTraceParent,
		"tracestate":  opts.TargetSpan.DynamicSpanTraceState,
	}
	// See CreateSpan's comment on why there's no ExecutionContext mixed into
	// this extracted context.
	ctx = propagator.Extract(context.Background(), carrier)

	if opts.Status.IsEnded() {
		meta.AddAttr(attrs, meta.Attrs.EndedAt, &ts)
	}

	if opts.Debug != nil {
		if opts.Debug.Location != "" {
			meta.AddAttr(attrs, meta.Attrs.InternalLocation, &opts.Debug.Location)
		}
	}

	// Be careful to make sure that whatever attrs we specify here are
	// overwritten by whatever is given in options; the caller knows best.
	if opts.Attributes != nil {
		attrs = attrs.Merge(opts.Attributes)
	}

	tsWithOffset := ts.Add(opts.EndTimeOffset)

	spanOpts := append(
		[]trace.SpanStartOption{
			trace.WithAttributes(attrs.Serialize()...),
			trace.WithTimestamp(tsWithOffset),
		},
		opts.RawOtelSpanOptions...,
	)

	tracer := tp.getTracer()
	_, span := tracer.Start(ctx, meta.SpanNameDynamicExtension, spanOpts...)

	span.End(trace.WithTimestamp(tsWithOffset))
	return nil
}
