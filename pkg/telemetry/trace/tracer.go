package trace

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/exporters"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TracerType int8

const (
	TracerTypeNoop = iota
	TracerTypeIO
	TracerTypeOTLP
	TracerTypeJaeger
	TracerTypeOTLPHTTP
	TracerTypeNATS
	TracerTypeKafka
)

var (
	userTracer Tracer
	onceUser   sync.Once

	systemTracer Tracer
	onceSystem   sync.Once
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

	// DropBlockingSpans specifies whether in-flight spans are dropped
	// on a full buffer, instead of blocking span sending.
	DropBlockingSpans bool

	NATS  []exporters.NatsExporterOpts
	Kafka []exporters.KafkaSpansExporterOpts
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

	return (consts.AbsoluteMaxEventSize + consts.MaxSDKResponseBodySize) * 2
}

func NewUserTracer(ctx context.Context, opts TracerOpts) error {
	var err error
	onceUser.Do(func() {
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

func NewSystemTracer(ctx context.Context, opts TracerOpts) error {
	var err error
	onceSystem.Do(func() {
		systemTracer, err = newTracer(ctx, opts)
	})
	return err
}

func SystemTracer() Tracer {
	if systemTracer == nil {
		if err := NewSystemTracer(context.Background(), TracerOpts{
			ServiceName: "default",
			Type:        TracerTypeNoop,
		}); err != nil {
			panic("fail to setup default system tracer")
		}
	}
	return systemTracer
}

func CloseSystemTracer(ctx context.Context) error {
	if systemTracer != nil {
		systemTracer.Shutdown(ctx)
	}
	return nil
}

// HeadersFromTraceState is used to set trace state headers from the current
// context so that the SDK can parrot them back to us for userland spans.
//
// Even if returning an error, headers will be returned, albeit empty.
//
// After a tracing refactor, this will no longer be required to send to SDKs
// because they will not need to parrot back any data. It is required now as
// trace ingestion is not critical and so could be delayed, meaning ingestion
// endpoints cannot reliably access previous spans as they are not guaranteed to
// be written.
func HeadersFromTraceState(
	ctx context.Context,
	// The span ID that should be used in the trace state, which is not
	// necessarily the current span ID if context is managed elsewhere, e.g. if
	// a span is created in a lifecycle.
	spanID string,
	appID string,
	functionID string,
) (map[string]string, error) {
	headers := make(map[string]string)
	legacySpan := oteltrace.SpanFromContext(ctx)
	lsc := legacySpan.SpanContext()

	ts, err := lsc.TraceState().Insert("inngest@app", appID)
	if err != nil {
		return headers, fmt.Errorf("failed to add app ID to trace state: %w", err)
	}

	ts, err = ts.Insert("inngest@fn", functionID)
	if err != nil {
		return headers, fmt.Errorf("failed to add function ID to trace state: %w", err)
	}

	if span, err := meta.GetSpanReferenceFromCtx(ctx); err == nil {
		if spanStr, err := span.MarshalForSDK(); err == nil {
			if ts, err = ts.Insert("inngest@traceref", spanStr); err != nil {
				return headers, fmt.Errorf("failed to add trace reference to trace state: %w", err)
			}
		}
	}

	lsc = oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    lsc.TraceID(),
		SpanID:     lsc.SpanID(),
		TraceFlags: lsc.TraceFlags(),
		TraceState: ts,
		Remote:     lsc.IsRemote(),
	})

	newCtx := oteltrace.ContextWithSpanContext(ctx, lsc)
	UserTracer().Propagator().Inject(newCtx, propagation.MapCarrier(headers))

	if headers["traceparent"] != "" {
		// The span ID will be incorrect here as lifecycles can not affect the
		// ctx. To patch, we manually set the span ID here to what we know it
		// should be based on the item
		parts := strings.Split(headers["traceparent"], "-")
		if len(parts) == 4 {
			parts[2] = spanID
			headers["traceparent"] = strings.Join(parts, "-")
		}
	}

	return headers, nil
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
		logger.StdlibLogger(context.Background()).Trace("no exporter available to export custom spans")
		return nil
	}

	t.processor.OnEnd(span)
	return nil
}

func TracerSetup(svc string, ttype TracerType) (func(), error) {
	ctx := context.Background()

	tracer, err := newTracer(ctx, TracerOpts{
		ServiceName: svc,
		Type:        ttype,
	})
	if err != nil {
		return nil, err
	}

	otel.SetTracerProvider(tracer.Provider())
	otel.SetTextMapPropagator(
		newTextMapPropagator(),
	)

	return func() {
		tracer.Shutdown(ctx)
	}, nil
}

// NewTracerProvider creates a new tracer with a provider and exporter based
// on the passed in `TraceType`.
func newTracer(ctx context.Context, opts TracerOpts) (Tracer, error) {
	switch opts.Type {
	case TracerTypeOTLP:
		return newOTLPGRPCTraceProvider(ctx, opts)
	case TracerTypeOTLPHTTP:
		return newOTLPHTTPTraceProvider(ctx, opts)
	case TracerTypeJaeger:
		return newJaegerTraceProvider(ctx, opts)
	case TracerTypeIO:
		return newIOTraceProvider(ctx, opts)
	case TracerTypeNATS:
		return newNatsTraceProvider(ctx, opts)
	case TracerTypeKafka:
		return newKafkaTraceExporter(ctx, opts)
	default:
		return newNoopTraceProvider(ctx, opts)
	}
}

func newJaegerTraceProvider(ctx context.Context, opts TracerOpts) (Tracer, error) {
	exp, err := jaegerExporter()
	if err != nil {
		return nil, fmt.Errorf("error setting up Jaeger exporter: %w", err)
	}

	sp := trace.NewBatchSpanProcessor(exp)
	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(sp),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(opts.ServiceName),
		)),
	)
	return &tracer{
		provider:   tp,
		propagator: newTextMapPropagator(),
		processor:  sp,
		shutdown: func(ctx context.Context) {
			_ = tp.ForceFlush(ctx)
			_ = tp.Shutdown(ctx)
		},
	}, nil
}

// IOTraceProvider is expected to be used for debugging purposes and not for production usage
func newIOTraceProvider(ctx context.Context, opts TracerOpts) (Tracer, error) {
	exp, err := stdouttrace.New(
		stdouttrace.WithWriter(os.Stderr),
	)
	if err != nil {
		return nil, fmt.Errorf("error settings up stdout trace exporter: %w", err)
	}

	sp := trace.NewBatchSpanProcessor(exp)
	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(sp),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(opts.ServiceName),
		)),
	)

	return &tracer{
		provider:   tp,
		propagator: newTextMapPropagator(),
		processor:  sp,
		shutdown: func(ctx context.Context) {
			_ = exp.Shutdown(ctx)
			_ = tp.ForceFlush(ctx)
			_ = tp.Shutdown(ctx)
		},
	}, nil
}

func newNoopTraceProvider(ctx context.Context, opts TracerOpts) (Tracer, error) {
	tp := trace.NewTracerProvider(
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(opts.ServiceName),
		)),
	)
	return &tracer{
		provider:   tp,
		propagator: newTextMapPropagator(),
		shutdown:   func(ctx context.Context) {},
	}, nil
}

func newOTLPHTTPTraceProvider(ctx context.Context, opts TracerOpts) (Tracer, error) {
	endpoint := opts.Endpoint()
	urlpath := opts.URLPath()

	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithURLPath(urlpath),
		otlptracehttp.WithInsecure(),
	)

	exp, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("error create otlp http trace client: %w", err)
	}

	sp := trace.NewBatchSpanProcessor(exp, trace.WithBatchTimeout(100*time.Millisecond))
	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(sp),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(opts.ServiceName),
		)),
	)

	return &tracer{
		provider:   tp,
		propagator: newTextMapPropagator(),
		processor:  sp,
		shutdown: func(ctx context.Context) {
			_ = tp.ForceFlush(ctx)
			_ = exp.Shutdown(ctx)
			_ = tp.Shutdown(ctx)
		},
	}, nil
}

func newOTLPGRPCTraceProvider(ctx context.Context, opts TracerOpts) (Tracer, error) {
	endpoint := opts.Endpoint()
	maxPayloadSize := opts.MaxPayloadSizeBytes()

	// NOTE:
	// assuming the otel collector is within the same private network, we can
	// skip grpc authn, but probably still better to get it work for production eventually
	conn, err := grpc.Dial(endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallSendMsgSize(maxPayloadSize),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to otel collector via grpc: %w", err)
	}

	client := otlptracegrpc.NewClient(
		otlptracegrpc.WithGRPCConn(conn),
	)

	exp, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("error creating otlp trace client: %w", err)
	}

	sp := trace.NewBatchSpanProcessor(exp)
	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(sp),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(opts.ServiceName),
		)),
	)

	return &tracer{
		provider:   tp,
		propagator: newTextMapPropagator(),
		processor:  sp,
		shutdown: func(ctx context.Context) {
			_ = tp.ForceFlush(ctx)
			_ = exp.Shutdown(ctx)
			_ = tp.Shutdown(ctx)
		},
	}, nil
}

func jaegerExporter() (trace.SpanExporter, error) {
	// NOTE: use the environment variables to set Jaeger exporter
	// https://pkg.go.dev/go.opentelemetry.io/otel/exporters/jaeger#readme-environment-variables
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint())
	if err != nil {
		return nil, fmt.Errorf("error creating jaeger trace exporter: %w", err)
	}
	return exp, nil
}

func newTextMapPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newNatsTraceProvider(ctx context.Context, opts TracerOpts) (Tracer, error) {
	if len(opts.NATS) == 0 {
		return nil, fmt.Errorf("nats options not provided")
	}

	exp, err := exporters.NewNATSSpanExporter(ctx, opts.NATS...)
	if err != nil {
		return nil, fmt.Errorf("error creating NATS trace client: %w", err)
	}

	// configure options
	bopts := []exporters.BatchSpanProcessorOpt{}
	{
		val := os.Getenv("SPAN_BATCH_PROCESSOR_BUFFER_SIZE")
		if val != "" {
			bufferSize, err := strconv.Atoi(val)
			if err == nil && bufferSize > 0 {
				bopts = append(bopts, exporters.WithBatchProcessorBufferSize(bufferSize))
			}
		}
	}

	{
		val := os.Getenv("SPAN_BATCH_PROCESSOR_INTERVAL")
		if val != "" {
			if dur, err := time.ParseDuration(val); err == nil {
				bopts = append(bopts, exporters.WithBatchProcessorInterval(dur))
			}
		}
	}

	{
		val := os.Getenv("SPAN_BATCH_PROCESSOR_CONCURRENCY")
		if val != "" {
			c, err := strconv.Atoi(val)
			if err == nil && c > 0 {
				bopts = append(bopts, exporters.WithBatchProcessorConcurrency(c))
			}
		}
	}

	sp := exporters.NewBatchSpanProcessor(ctx, exp, bopts...)
	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(sp),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(opts.ServiceName),
		)),
	)

	return &tracer{
		provider:   tp,
		propagator: newTextMapPropagator(),
		processor:  sp,
		shutdown: func(ctx context.Context) {
			_ = tp.ForceFlush(ctx)
			_ = tp.Shutdown(ctx)
			_ = exp.Shutdown(ctx)
		},
	}, nil
}

func newKafkaTraceExporter(ctx context.Context, opts TracerOpts) (Tracer, error) {
	exp, err := exporters.NewKafkaSpanExporter(ctx, opts.Kafka...)
	if err != nil {
		return nil, fmt.Errorf("error creating Kafka trace client: %w", err)
	}

	bopts := []exporters.BatchSpanProcessorOpt{}
	{
		val := os.Getenv("SPAN_BATCH_PROCESSOR_BUFFER_SIZE")
		if val != "" {
			bufferSize, err := strconv.Atoi(val)
			if err == nil && bufferSize > 0 {
				bopts = append(bopts, exporters.WithBatchProcessorBufferSize(bufferSize))
			}
		}
	}

	{
		val := os.Getenv("SPAN_BATCH_PROCESSOR_INTERVAL")
		if val != "" {
			if dur, err := time.ParseDuration(val); err == nil {
				bopts = append(bopts, exporters.WithBatchProcessorInterval(dur))
			}
		}
	}

	{
		val := os.Getenv("SPAN_BATCH_PROCESSOR_CONCURRENCY")
		if val != "" {
			c, err := strconv.Atoi(val)
			if err == nil && c > 0 {
				bopts = append(bopts, exporters.WithBatchProcessorConcurrency(c))
			}
		}
	}

	if opts.DropBlockingSpans {
		bopts = append(bopts, exporters.WithBatchProcessorDropBlockingSpans())
	}

	sp := exporters.NewBatchSpanProcessor(ctx, exp, bopts...)
	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(sp),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(opts.ServiceName),
		)),
	)

	return &tracer{
		provider:   tp,
		propagator: newTextMapPropagator(),
		processor:  sp,
		shutdown: func(ctx context.Context) {
			_ = tp.ForceFlush(ctx)
			_ = tp.Shutdown(ctx)
			_ = exp.Shutdown(ctx)
		},
	}, nil
}

func ConnectTracer() oteltrace.Tracer {
	l := logger.StdlibLogger(context.Background())

	systemTracer := SystemTracer()
	if systemTracer == nil {
		l.Error("system tracer is nil")
	}

	provider := systemTracer.Provider()
	if provider == nil {
		l.Error("trace provider is nil in system tracer")
	}

	tracer := provider.Tracer("connect")
	if tracer == nil {
		l.Error("connect tracer is nil")
	}

	return tracer
}

func QueueTracer() oteltrace.Tracer {
	l := logger.StdlibLogger(context.Background())

	systemTracer := SystemTracer()
	if systemTracer == nil {
		l.Error("system tracer is nil")
	}

	provider := systemTracer.Provider()
	if provider == nil {
		l.Error("trace provider is nil in system tracer")
	}

	tracer := provider.Tracer("queue")
	if tracer == nil {
		l.Error("queue tracer is nil")
	}

	return tracer
}
