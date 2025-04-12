package tracing

import (
	"context"
	"encoding/json"
	"time"

	"github.com/davecgh/go-spew/spew"
	sqlc "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/sqlite"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest/version"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type TracerProvider struct {
	q sqlc.Querier
}

var defaultPropagator = propagation.NewCompositeTextMapPropagator(
	propagation.TraceContext{},
	propagation.Baggage{},
)

func NewTracerProvider(q sqlc.Querier) *TracerProvider {
	return &TracerProvider{
		q: q,
	}
}

func (tp *TracerProvider) getTracer(md *statev2.Metadata, qi *queue.Item) trace.Tracer {
	exp := &DBExporter{q: tp.q}
	base := sdktrace.NewSimpleSpanProcessor(exp)

	otelTP := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(newExecutionProcessor(md, qi, base)),
	)

	tracer := otelTP.Tracer("inngest", trace.WithInstrumentationVersion(version.Print()))

	return tracer
}

type CreateSpanOptions struct {
	Carrier     map[string]string
	FollowsFrom *meta.SpanMetadata
	Location    string
	Metadata    *statev2.Metadata
	Parent      *meta.SpanMetadata
	QueueItem   *queue.Item
	SpanOptions []trace.SpanStartOption
}

func (tp *TracerProvider) CreateSpan(
	name string,
	opts *CreateSpanOptions,
) *meta.SpanMetadata {
	span, spanMetadata := tp.CreateDroppableSpan(name, opts)
	span.End()

	return spanMetadata
}

// CreateDroppableSpan creates a span that can be dropped and relies on us
// calling `.End()`.
func (tp *TracerProvider) CreateDroppableSpan(
	name string,
	opts *CreateSpanOptions,
) (trace.Span, *meta.SpanMetadata) {
	ctx := context.Background()
	if opts.Parent != nil {
		carrier := propagation.MapCarrier{
			"traceparent": opts.Parent.TraceParent,
			"tracestate":  opts.Parent.TraceState,
		}
		ctx = defaultPropagator.Extract(context.Background(), carrier)
	}

	spanOptions := append(opts.SpanOptions, trace.WithSpanKind(trace.SpanKindServer))
	if opts.FollowsFrom != nil {
		spanOptions = append(
			spanOptions,
			trace.WithLinks(trace.Link{
				SpanContext: spanContextFromMetadata(opts.FollowsFrom),
				Attributes: []attribute.KeyValue{
					attribute.String(meta.LinkAttributeType, meta.LinkAttributeTypeFollowsFrom),
				},
			}),
		)
	}

	tracer := tp.getTracer(opts.Metadata, opts.QueueItem)
	ctx, span := tracer.Start(ctx, name, spanOptions...)

	carrier := propagation.MapCarrier{}
	defaultPropagator.Inject(ctx, carrier)

	spanMetadata := &meta.SpanMetadata{
		TraceParent: carrier["traceparent"],
		TraceState:  carrier["tracestate"],
	}

	// Only spans with parents can be dynamic? Hm.
	if opts.Parent != nil {
		spanMetadata.DynamicSpanTraceParent = opts.Parent.TraceParent
		spanMetadata.DynamicSpanTraceState = opts.Parent.TraceState
		spanMetadata.DynamicSpanID = span.SpanContext().SpanID().String()
	}

	span.SetAttributes(
		attribute.String(meta.AttributeDynamicSpanID, spanMetadata.DynamicSpanID),
	)

	if opts.Carrier != nil {
		// TODO err
		byt, _ := json.Marshal(spanMetadata)
		opts.Carrier[meta.PropagationKey] = string(byt)
	}

	spew.Dump("tracing.CreateSpan", name, opts.Location, spanMetadata)

	return span, spanMetadata
}

type ExtendSpanOptions struct {
	Carrier     map[string]string
	EndTime     time.Time
	Location    string
	Metadata    *statev2.Metadata
	QueueItem   *queue.Item
	SpanOptions []trace.SpanStartOption
	Status      codes.Code
	TargetSpan  *meta.SpanMetadata
}

// Returns nothing, as the span is only extended and no further context is given
func (tp *TracerProvider) ExtendSpan(
	opts *ExtendSpanOptions,
) {
	spew.Dump("tracing.ExtendSpan", opts.TargetSpan)

	if opts.TargetSpan == nil || opts.TargetSpan.DynamicSpanID == "" {
		// Oof. Not good.
		panic("no target span")
	}

	carrier := propagation.MapCarrier{
		"traceparent": opts.TargetSpan.DynamicSpanTraceParent,
		"tracestate":  opts.TargetSpan.DynamicSpanTraceState,
	}
	ctx := defaultPropagator.Extract(context.Background(), carrier)

	tracer := tp.getTracer(opts.Metadata, opts.QueueItem)
	_, span := tracer.Start(ctx, meta.SpanNameDynamicExtension, opts.SpanOptions...)

	span.SetAttributes(
		attribute.String(meta.AttributeDynamicSpanID, opts.TargetSpan.DynamicSpanID),
		attribute.String(meta.AttributeDynamicStatus, opts.Status.String()),
	)

	span.End()
}
