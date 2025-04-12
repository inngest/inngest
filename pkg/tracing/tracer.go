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
	Location    string
	Metadata    *statev2.Metadata
	Parent      *meta.SpanMetadata
	QueueItem   *queue.Item
	SpanOptions []trace.SpanStartOption
}

// TODO
// I need this to be able to do two things:
// 1. Create a new trace (no parent, e.g. run span)
// 2. Create a new child span (e.g. step under run span) and set the
// dynamicspanID to the new span ID
// 3. Contribute to an existing dynamic span, taking dynamic span ID from the
// parent and adding it to the new span
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

	tracer := tp.getTracer(opts.Metadata, opts.QueueItem)
	ctx, span := tracer.Start(ctx, name, opts.SpanOptions...)

	carrier := propagation.MapCarrier{}
	defaultPropagator.Inject(ctx, carrier)
	// defaultPropagator.Inject(trace.ContextWithSpan(ctx, span), carrier)

	spanMetadata := &meta.SpanMetadata{
		TraceParent:   carrier["traceparent"],
		TraceState:    carrier["tracestate"],
		DynamicSpanID: span.SpanContext().SpanID().String(),
	}

	span.SetAttributes(
		attribute.String(meta.AttributeDynamicSpanID, spanMetadata.DynamicSpanID),
	)

	if opts.Carrier != nil {
		// TODO err
		byt, _ := json.Marshal(spanMetadata)
		opts.Carrier["wobbly"] = string(byt)
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
	TargetSpan  *meta.SpanMetadata
}

func (tp *TracerProvider) ExtendSpan(
	opts *ExtendSpanOptions,
) *meta.SpanMetadata {
	spew.Dump("tracing.ExtendSpan", opts.TargetSpan)

	if opts.TargetSpan == nil {
		// Oof. Not good.
		panic("no target span")
	}

	carrier := propagation.MapCarrier{
		"traceparent": opts.TargetSpan.TraceParent,
		"tracestate":  opts.TargetSpan.TraceState,
	}
	ctx := defaultPropagator.Extract(context.Background(), carrier)

	// TODO It shouldn't have all these extras added though...
	// Just `nil, nil` this?
	tracer := tp.getTracer(nil, nil)
	_, span := tracer.Start(ctx, "EXTEND", opts.SpanOptions...)

	spanMetadata := &meta.SpanMetadata{
		TraceParent:   carrier["traceparent"],
		TraceState:    carrier["tracestate"],
		DynamicSpanID: opts.TargetSpan.DynamicSpanID,
	}

	span.SetAttributes(
		attribute.String("dynamic_span_id", spanMetadata.DynamicSpanID),
	)

	span.End()

	return spanMetadata
}

// func (tp *TracerProvider) NewTracer(ctx context.Context, md statev2.Metadata, qi *queue.Item) (context.Context, *Tracer) {
// 	exp := &DBExporter{q: tp.q}
// 	base := sdktrace.NewSimpleSpanProcessor(exp)

// 	otelTP := sdktrace.NewTracerProvider(
// 		sdktrace.WithSpanProcessor(newExecutionProcessor(md, base)),
// 	)

// 	var metadata map[string]string
// 	if qi != nil && qi.Metadata != nil {
// 		if carrier, ok := qi.Metadata["wobbly"]; ok {
// 			metadata = make(map[string]string)
// 			if err := json.Unmarshal([]byte(carrier), &metadata); err != nil {
// 				spew.Dump("error unmarshalling carrier", err)
// 			}
// 		}
// 	}

// 	if metadata == nil {
// 		metadata = md.Config.NewFunctionTrace()
// 		spew.Dump("md metadata", metadata)
// 	}

// 	return tp.extract(ctx, metadata), &Tracer{tp: otelTP}
// }

// func (tp *TracerProvider) GetLineage(ctx context.Context) map[string]string {
// 	metadata := make(map[string]string)
// 	tp.Inject(ctx, metadata)

// 	return metadata
// }

// func (tp *TracerProvider) Inject(ctx context.Context, metadata map[string]string) {
// 	defaultPropagator.Inject(ctx, propagation.MapCarrier(metadata))
// }

// func (tp *TracerProvider) extract(ctx context.Context, metadata map[string]string) context.Context {
// 	if metadata == nil {
// 		metadata = make(map[string]string)
// 	}

// 	return defaultPropagator.Extract(ctx, propagation.MapCarrier(metadata))
// }

// func (tp *TracerProvider) UpdateSpanEnd(ctx context.Context, endTime time.Time, endAttrs []attribute.KeyValue) error {
// 	span := trace.SpanFromContext(ctx)
// 	if span == nil {
// 		return nil
// 	}

// 	attrs := make(map[string]interface{})
// 	for _, attr := range endAttrs {
// 		attrs[string(attr.Key)] = attr.Value.AsInterface()
// 	}
// 	data, err := json.Marshal(attrs)
// 	if err != nil {
// 		// TODO Log error
// 		spew.Dump("Failed to marshal span attributes", err)
// 		return err
// 	}

// 	tp.q.UpdateSpanEnd(ctx, sqlc.UpdateSpanEndParams{
// 		SpanID:        span.SpanContext().SpanID().String(),
// 		EndTime:       sql.NullTime{Time: endTime, Valid: true},
// 		EndAttributes: string(data),
// 	})

// 	return nil
// }

// type Tracer struct {
// 	tp *sdktrace.TracerProvider
// }

// func (t *Tracer) Tracer() trace.Tracer {
// 	return t.tp.Tracer("inngest", trace.WithInstrumentationVersion(version.Print()))
// }
