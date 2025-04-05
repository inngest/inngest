package tracing

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/davecgh/go-spew/spew"
	sqlc "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/sqlite"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest/version"
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

func (tp *TracerProvider) NewTracer(ctx context.Context, md statev2.Metadata, qi *queue.Item) (context.Context, *Tracer) {
	exp := &DBExporter{q: tp.q}
	base := sdktrace.NewSimpleSpanProcessor(exp)

	otelTP := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(newExecutionProcessor(md, base)),
	)

	var metadata map[string]string
	if qi != nil && qi.Metadata != nil {
		if carrier, ok := qi.Metadata["wobbly"]; ok {
			metadata = make(map[string]string)
			if err := json.Unmarshal([]byte(carrier), &metadata); err != nil {
				spew.Dump("error unmarshalling carrier", err)
			}
		}
	}

	if metadata == nil {
		metadata = md.Config.NewFunctionTrace()
		spew.Dump("md metadata", metadata)
	}

	return tp.extract(ctx, metadata), &Tracer{tp: otelTP}
}

func (tp *TracerProvider) GetLineage(ctx context.Context) map[string]string {
	metadata := make(map[string]string)
	tp.Inject(ctx, metadata)

	return metadata
}

func (tp *TracerProvider) Inject(ctx context.Context, metadata map[string]string) {
	defaultPropagator.Inject(ctx, propagation.MapCarrier(metadata))
}

func (tp *TracerProvider) extract(ctx context.Context, metadata map[string]string) context.Context {
	if metadata == nil {
		metadata = make(map[string]string)
	}

	return defaultPropagator.Extract(ctx, propagation.MapCarrier(metadata))
}

func (tp *TracerProvider) UpdateSpanEnd(ctx context.Context, endTime time.Time, endAttrs []attribute.KeyValue) error {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return nil
	}

	attrs := make(map[string]interface{})
	for _, attr := range endAttrs {
		attrs[string(attr.Key)] = attr.Value.AsInterface()
	}
	data, err := json.Marshal(attrs)
	if err != nil {
		// TODO Log error
		spew.Dump("Failed to marshal span attributes", err)
		return err
	}

	tp.q.UpdateSpanEnd(ctx, sqlc.UpdateSpanEndParams{
		SpanID:        span.SpanContext().SpanID().String(),
		EndTime:       sql.NullTime{Time: endTime, Valid: true},
		EndAttributes: string(data),
	})

	return nil
}

type Tracer struct {
	tp *sdktrace.TracerProvider
}

func (t *Tracer) Tracer() trace.Tracer {
	return t.tp.Tracer("inngest", trace.WithInstrumentationVersion(version.Print()))
}
