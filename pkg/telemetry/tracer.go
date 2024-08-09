package telemetry

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/inngest/log"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/nats-io/nats.go"
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
	Type                     TracerType
	ServiceName              string
	TraceEndpoint            string
	TraceURLPath             string
	TraceMaxPayloadSizeBytes int

	NATS *NatsExporterOpts
}

func (o TracerOpts) Endpoint() string {
	if o.TraceEndpoint != "" {
		return o.TraceEndpoint
	}
	if os.Getenv("OTEL_TRACES_COLLECTOR_ENDPOINT") != "" {
		return os.Getenv("OTEL_TRACES_COLLECTOR_ENDPOINT")
	}

	// default
	return "otel-collector:4317"
}

func (o TracerOpts) URLPath() string {
	if o.TraceURLPath != "" {
		return o.TraceURLPath
	}

	urlpath := os.Getenv("OTEL_TRACE_COLLECTOR_URL_PATH")
	if urlpath == "" {
		return urlpath
	}

	return "/v1/traces"
}

func (o TracerOpts) MaxPayloadSizeBytes() int {
	if o.TraceMaxPayloadSizeBytes != 0 {
		return o.TraceMaxPayloadSizeBytes
	}

	size, _ := strconv.Atoi(os.Getenv("OTEL_TRACES_MAX_PAYLOAD_SIZE_BYTES"))
	if size != 0 {
		return size
	}

	return (consts.AbsoluteMaxEventSize + consts.MaxBodySize) * 2
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

// NATS span exporter
// TODO: Hold NATS connection
// - subject to write to
// - is jetstream or not?
type natsSpanExporter struct {
	conn *nats.Conn
}

type NatsExporterOpts struct {
	URLs []string
}

// NewNATSSpanExporter creates an otel compatible exporter that ships the spans to NATS
func NewNATSSpanExporter(ctx context.Context, opts *NatsExporterOpts) (trace.SpanExporter, error) {
	if opts == nil {
		return nil, fmt.Errorf("nats exporter setup options unavailable")
	}

	host, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("error retrieving hostname: %w", err)
	}

	urls := strings.Join(opts.URLs, ",")
	name := fmt.Sprintf("trace-exporter-%s", host)

	// urls should be comma delimited strings
	conn, err := nats.Connect(urls,
		nats.Name(name),
	)
	if err != nil {
		return nil, fmt.Errorf("error connecting to NATS: %w", err)
	}

	logger.StdlibLogger(ctx).Info("established connection with NATS server",
		"urls", opts.URLs,
		"name", name,
	)

	return &natsSpanExporter{
		conn: conn,
	}, nil
}

func (e *natsSpanExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	return fmt.Errorf("not implemented")
}

func (e *natsSpanExporter) Shutdown(ctx context.Context) error {
	e.conn.Close()
	return nil
}
