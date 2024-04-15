package telemetry

import (
	"context"
	"sync"

	"github.com/inngest/inngest/pkg/inngest/log"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
)

var (
	userTracer Tracer
	o          sync.Once
)

// Tracer is a wrapper around the otel's tracing library to allow combining
// usage with the official library or workarounds that are necessary that the
// official library and spec doesn't support for our use cases.
type Tracer interface {
	// Provider returns the configured provider for tracing
	Provider() *trace.TracerProvider
	// Propagator returns the configured context propagator
	Propagator() propagation.TextMapPropagator
	// Shutdown runs the shutdown process for the tracer
	// includes flusing and closing connections, etc
	Shutdown(ctx context.Context) func()
	// Export allows exporting spans directly outside the limit
	// of the otel's trace library.
	// This can be used for sending out spans prior to ending, or
	// send out duplicate spans, which we can dedup later ourselves.
	Export(span trace.ReadOnlySpan) error
}

type TracerOpts struct {
	ServiceName string
	Type        TracerType
}

func NewUserTracer(ctx context.Context, opts TracerOpts) error {
	var err error
	o.Do(func() {
		userTracer, err = newTracer(ctx, opts)
	})
	return err
}

func UserTracer() Tracer {
	if userTracer == nil {
		if err := NewUserTracer(context.Background(), TracerOpts{
			ServiceName: "default",
			Type:        TracerTypeNoop,
		}); err != nil {
			panic("fail to setup default user tracer")
		}
	}
	return userTracer
}

func CloseUserTracer(ctx context.Context) error {
	if userTracer != nil {
		userTracer.Shutdown(ctx)
	}
	return nil
}

type tracer struct {
	provider   *trace.TracerProvider
	propagator propagation.TextMapPropagator
	shutdown   func(context.Context)
	processor  trace.SpanProcessor
}

func (t *tracer) Provider() *trace.TracerProvider {
	return t.provider
}

func (t *tracer) Propagator() propagation.TextMapPropagator {
	return t.propagator
}

func (t *tracer) Shutdown(ctx context.Context) func() {
	return func() {
		t.shutdown(ctx)
	}
}

func (t *tracer) Export(span trace.ReadOnlySpan) error {
	if t.processor == nil {
		ctx := context.Background()
		log.From(ctx).Trace().Msg("no exporter available to export custom spans")
		return nil
	}

	t.processor.OnEnd(span)
	return nil
}
