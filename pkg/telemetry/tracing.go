package telemetry

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
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
	TracerTypeIO = iota
	TracerTypeOTLP
	TracerTypeJaeger
)

type tracer struct {
	Provider *trace.TracerProvider
	Shutdown func()
}

func TracerSetup(svc string, ttype TracerType) (func(), error) {
	ctx := context.Background()

	tracer, err := NewTracerProvider(ctx, svc, ttype)
	if err != nil {
		return nil, err
	}

	otel.SetTracerProvider(tracer.Provider)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return func() { tracer.Shutdown() }, nil
}

// NewTracerProvider creates a new tracer with a provider and exporter based
// on the passed in `TraceType`.
func NewTracerProvider(ctx context.Context, svc string, ttype TracerType) (*tracer, error) {
	switch ttype {
	case TracerTypeOTLP:
		return NewOLTPTraceProvider(ctx, svc)
	case TracerTypeJaeger:
		return NewJaegerTraceProvider(ctx, svc)
	default:
		return NewIOTraceProvider(ctx, svc)
	}
}

func NewJaegerTraceProvider(ctx context.Context, svc string) (*tracer, error) {
	exp, err := jaegerExporter()
	if err != nil {
		return nil, fmt.Errorf("error setting up Jaeger exporter: %w", err)
	}
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(svc),
			semconv.DeploymentEnvironmentKey.String(env()),
		)),
	)
	return &tracer{
		Provider: tp,
		Shutdown: func() {
			_ = tp.ForceFlush(ctx)
		},
	}, nil
}

func NewIOTraceProvider(ctx context.Context, svc string) (*tracer, error) {
	exp, err := stdouttrace.New()
	if err != nil {
		return nil, fmt.Errorf("error settings up stdout trace exporter: %w", err)
	}
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(svc),
			semconv.DeploymentEnvironmentKey.String(env()),
		)),
	)
	return &tracer{
		Provider: tp,
		Shutdown: func() {
			_ = tp.Shutdown(ctx)
		},
	}, nil
}

func NewOLTPTraceProvider(ctx context.Context, svc string) (*tracer, error) {
	endpoint := os.Getenv("OTEL_TRACES_COLLECTOR_ENDPOINT")
	if endpoint == "" {
		endpoint = "otel-collector:4317"
	}

	// NOTE:
	// assuming the otel collector is within the same private network, we can
	// skip grpc authn, but probably still better to get it work for production
	conn, err := grpc.Dial(endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
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

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(svc),
			semconv.DeploymentEnvironmentKey.String(env()),
		)),
	)

	return &tracer{
		Provider: tp,
		Shutdown: func() {
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
