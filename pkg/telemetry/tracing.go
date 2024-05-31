package telemetry

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/inngest/log"
	"github.com/rs/zerolog"
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
)

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
		return newOLTPGRPCTraceProvider(ctx, opts)
	case TracerTypeOTLPHTTP:
		return newOTLPHTTPTraceProvider(ctx, opts)
	case TracerTypeJaeger:
		return newJaegerTraceProvider(ctx, opts)
	case TracerTypeIO:
		return newIOTraceProvider(ctx, opts)
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
			semconv.DeploymentEnvironmentKey.String(env()),
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
		stdouttrace.WithWriter(log.New(zerolog.TraceLevel)),
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
			semconv.DeploymentEnvironmentKey.String(env()),
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
			semconv.DeploymentEnvironmentKey.String(env()),
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

func newOLTPGRPCTraceProvider(ctx context.Context, opts TracerOpts) (Tracer, error) {
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
