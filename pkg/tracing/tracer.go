package tracing

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
		// sdktrace.WithIDGenerator(), // Deterministic span IDs for idempotency pls
	)

	tracer := otelTP.Tracer("inngest", trace.WithInstrumentationVersion(version.Print()))

	return tracer
}

type DroppableSpan struct {
	span trace.Span
	Ref  *meta.SpanReference
}

func (d *DroppableSpan) Drop() {
	d.span.SetAttributes(attribute.Bool(meta.AttributeDropSpan, true))
	// Send span but we don't care if it makes it or not, as we're dropping
	// anyway
	d.span.End()
}

// TODO Sync send span; might wait for flush channel
func (d *DroppableSpan) Send() error {
	d.span.End()
	return nil
}

type CreateSpanOptions struct {
	Carriers    []map[string]any
	FollowsFrom *meta.SpanReference
	Location    string
	Metadata    *statev2.Metadata
	Parent      *meta.SpanReference
	QueueItem   *queue.Item
	SpanOptions []trace.SpanStartOption
}

func (tp *TracerProvider) CreateSpan(
	name string,
	opts *CreateSpanOptions,
) (*meta.SpanReference, error) {
	ds, err := tp.CreateDroppableSpan(name, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to CreateSpan: %w", err)
	}

	err = ds.Send()
	if err != nil {
		return nil, fmt.Errorf("failed to send span during creation: %w", err)
	}

	return ds.Ref, nil
}

// CreateDroppableSpan creates a span that can be dropped and relies on us
// calling `.End()`.
func (tp *TracerProvider) CreateDroppableSpan(
	name string,
	opts *CreateSpanOptions,
) (*DroppableSpan, error) {
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

	spanRef := &meta.SpanReference{
		TraceParent: carrier["traceparent"],
		TraceState:  carrier["tracestate"],
	}

	// Only spans with parents can be dynamic? Hm.
	if opts.Parent != nil {
		spanRef.DynamicSpanTraceParent = opts.Parent.TraceParent
		spanRef.DynamicSpanTraceState = opts.Parent.TraceState
		spanRef.DynamicSpanID = span.SpanContext().SpanID().String()
	}

	span.SetAttributes(
		attribute.String(meta.AttributeDynamicSpanID, spanRef.DynamicSpanID),
	)

	if len(opts.Carriers) > 0 {
		// TODO err
		byt, err := json.Marshal(spanRef)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal span metadata when injecting to carriers: %w", err)
		}

		for _, carrier := range opts.Carriers {
			carrier[meta.PropagationKey] = string(byt)
		}
	}

	return &DroppableSpan{
		span: span,
		Ref:  spanRef,
	}, nil
}

type UpdateSpanOptions struct {
	Carrier     map[string]string
	EndTime     time.Time
	Location    string
	Metadata    *statev2.Metadata
	QueueItem   *queue.Item
	SpanOptions []trace.SpanStartOption
	Status      codes.Code
	TargetSpan  *meta.SpanReference
}

// Returns nothing, as the span is only extended and no further context is given
func (tp *TracerProvider) UpdateSpan(
	opts *UpdateSpanOptions,
) error {
	if opts.TargetSpan == nil {
		return fmt.Errorf("no target span")
	}

	if opts.TargetSpan.DynamicSpanID == "" {
		// Oof. Not good.
		return fmt.Errorf("target span is not dynamic; has no DynamicSpanID")
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
	return nil
}
