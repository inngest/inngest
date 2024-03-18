package telemetry

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/inngest/log"
	"github.com/rs/zerolog"
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
	TracerTypeNoop = iota
	TracerTypeIO
	TracerTypeOTLP
	TracerTypeJaeger
)

type tracer struct {
	provider   *trace.TracerProvider
	propagator propagation.TextMapPropagator
	shutdown   func(context.Context)
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

func TracerSetup(svc string, ttype TracerType) (func(), error) {
	ctx := context.Background()

	tracer, err := NewTracer(ctx, svc, ttype)
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
func NewTracer(ctx context.Context, svc string, ttype TracerType) (Tracer, error) {
	switch ttype {
	case TracerTypeOTLP:
		return newOLTPTraceProvider(ctx, svc)
	case TracerTypeJaeger:
		return newJaegerTraceProvider(ctx, svc)
	case TracerTypeIO:
		return newIOTraceProvider(ctx, svc)
	default:
		return newNoopTraceProvider(ctx, svc)
	}
}

func newJaegerTraceProvider(ctx context.Context, svc string) (Tracer, error) {
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
		provider:   tp,
		propagator: newTextMapPropagator(),
		shutdown: func(ctx context.Context) {
			_ = tp.ForceFlush(ctx)
			_ = tp.Shutdown(ctx)
		},
	}, nil
}

// IOTraceProvider is expected to be used for debugging purposes and not for production usage
func newIOTraceProvider(ctx context.Context, svc string) (Tracer, error) {
	exp, err := stdouttrace.New(
		stdouttrace.WithWriter(log.New(zerolog.TraceLevel)),
	)
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
		provider:   tp,
		propagator: newTextMapPropagator(),
		shutdown: func(ctx context.Context) {
			_ = tp.ForceFlush(ctx)
			_ = tp.Shutdown(ctx)
		},
	}, nil
}

func newNoopTraceProvider(ctx context.Context, svc string) (Tracer, error) {
	tp := trace.NewTracerProvider(
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(svc),
			semconv.DeploymentEnvironmentKey.String(env()),
		)),
	)
	return &tracer{
		provider:   tp,
		propagator: newTextMapPropagator(),
		shutdown: func(ctx context.Context) {
		},
	}, nil
}

func newOLTPTraceProvider(ctx context.Context, svc string) (Tracer, error) {
	endpoint := os.Getenv("OTEL_TRACES_COLLECTOR_ENDPOINT")
	if endpoint == "" {
		endpoint = "otel-collector:4317"
	}

	var maxPayloadSize int
	maxPayloadSize, _ = strconv.Atoi(os.Getenv("OTEL_TRACES_MAX_PAYLOAD_SIZE_BYTES"))
	if maxPayloadSize == 0 {
		maxPayloadSize = (consts.AbsoluteMaxEventSize + consts.MaxBodySize) * 2
	}

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

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(svc),
		)),
	)

	return &tracer{
		provider:   tp,
		propagator: newTextMapPropagator(),
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
