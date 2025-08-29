package tracing

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest/version"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var defaultPropagator = propagation.NewCompositeTextMapPropagator(
	propagation.TraceContext{},
	propagation.Baggage{},
)

// TracerProvider defines the interface for tracing providers.
type TracerProvider interface {
	CreateSpan(name string, opts *CreateSpanOptions) (*meta.SpanReference, error)
	CreateDroppableSpan(name string, opts *CreateSpanOptions) (*DroppableSpan, error)
	UpdateSpan(opts *UpdateSpanOptions) error
}

type SpanDebugData struct {
	Location string
}

type DroppableSpan struct {
	span trace.Span
	Ref  *meta.SpanReference
}

type CreateSpanOptions struct {
	Attributes         *meta.SerializableAttrs
	Carriers           []map[string]any
	Debug              *SpanDebugData
	FollowsFrom        *meta.SpanReference
	Metadata           *statev2.Metadata
	Parent             *meta.SpanReference
	QueueItem          *queue.Item
	RawOtelSpanOptions []trace.SpanStartOption
	StartTime          time.Time
	EndTime            time.Time
}

type UpdateSpanOptions struct {
	Attributes         *meta.SerializableAttrs
	Debug              *SpanDebugData
	EndTime            time.Time
	Metadata           *statev2.Metadata
	QueueItem          *queue.Item
	RawOtelSpanOptions []trace.SpanStartOption
	Status             enums.StepStatus
	TargetSpan         *meta.SpanReference
}

// otelTracerProvider implements TracerProvider.
type otelTracerProvider struct {
	exp sdktrace.SpanExporter
}

func NewOtelTracerProvider(exp sdktrace.SpanExporter) TracerProvider {
	return &otelTracerProvider{
		exp: exp,
	}
}

func (tp *otelTracerProvider) getTracer(md *statev2.Metadata, qi *queue.Item) trace.Tracer {
	base := sdktrace.NewSimpleSpanProcessor(tp.exp)

	otelTP := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(newExecutionProcessor(md, qi, base)),
		// sdktrace.WithIDGenerator(), // Deterministic span IDs for idempotency pls
	)

	tracer := otelTP.Tracer("inngest", trace.WithInstrumentationVersion(version.Print()))

	return tracer
}

func (d *DroppableSpan) Drop() {
	d.span.SetAttributes(attribute.Bool(meta.Attrs.DropSpan.Key(), true))
	// Send span but we don't care if it makes it or not, as we're dropping
	// anyway
	d.span.End()
}

// TODO Sync send span; might wait for flush channel
func (d *DroppableSpan) Send() error {
	d.span.End()
	return nil
}

func (tp *otelTracerProvider) CreateSpan(
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
func (tp *otelTracerProvider) CreateDroppableSpan(
	name string,
	opts *CreateSpanOptions,
) (*DroppableSpan, error) {
	st := opts.StartTime
	if st.IsZero() {
		st = time.Now()
	}

	ctx := context.Background()
	if opts.Parent != nil {
		carrier := propagation.MapCarrier{
			"traceparent": opts.Parent.TraceParent,
			"tracestate":  opts.Parent.TraceState,
		}
		ctx = defaultPropagator.Extract(context.Background(), carrier)
	}

	attrs := opts.Attributes
	if attrs == nil {
		attrs = meta.NewAttrSet()
	}
	if opts.Debug != nil {
		if opts.Debug.Location != "" {
			meta.AddAttr(attrs, meta.Attrs.InternalLocation, &opts.Debug.Location)
		}
	}
	if !opts.EndTime.IsZero() {
		meta.AddAttr(attrs, meta.Attrs.EndedAt, &opts.EndTime)
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
	refTp := carrier["traceparent"]
	refTs := carrier["tracestate"]

	spanRef := &meta.SpanReference{
		TraceParent: refTp,
		TraceState:  refTs,
	}

	spanRef.DynamicSpanID = span.SpanContext().SpanID().String()

	if opts.Parent != nil {
		// If the span has a parent, set some attributes so we can extend it later
		// and pick the same trace and parent span IDs for the extension span.
		spanRef.DynamicSpanTraceParent = opts.Parent.TraceParent
		spanRef.DynamicSpanTraceState = opts.Parent.TraceState
	} else {
		// If we don't have a parent, this is a top-level span (e.g. the run
		// span), so we use this span as the dynamic reference instead.
		spanRef.DynamicSpanTraceParent = refTp
		spanRef.DynamicSpanTraceState = refTs
	}

	span.SetAttributes(
		attribute.String(meta.Attrs.DynamicSpanID.Key(), spanRef.DynamicSpanID),
	)

	if len(opts.Carriers) > 0 {
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

// Returns nothing, as the span is only extended and no further context is given
func (tp *otelTracerProvider) UpdateSpan(
	opts *UpdateSpanOptions,
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

	carrier := propagation.MapCarrier{
		"traceparent": opts.TargetSpan.DynamicSpanTraceParent,
		"tracestate":  opts.TargetSpan.DynamicSpanTraceState,
	}
	ctx := defaultPropagator.Extract(context.Background(), carrier)

	attrs := meta.NewAttrSet(
		meta.Attr(meta.Attrs.DynamicSpanID, &opts.TargetSpan.DynamicSpanID),
		meta.Attr(meta.Attrs.DynamicStatus, &opts.Status),
	)

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

	spanOpts := append(
		[]trace.SpanStartOption{
			trace.WithAttributes(attrs.Serialize()...),
			trace.WithTimestamp(ts),
		},
		opts.RawOtelSpanOptions...,
	)

	tracer := tp.getTracer(opts.Metadata, opts.QueueItem)
	_, span := tracer.Start(ctx, meta.SpanNameDynamicExtension, spanOpts...)

	span.End()
	return nil
}
