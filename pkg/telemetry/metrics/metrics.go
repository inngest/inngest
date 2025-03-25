package metrics

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

type MeterType int8

const (
	MeterTypeIO = iota
	MeterTypeOTLP
	MeterTypePrometheus
)

func MeterSetup(svc string, mtype MeterType) (func(), error) {
	ctx := context.Background()

	meter, err := NewMeterProvider(ctx, svc, mtype)
	if err != nil {
		return nil, err
	}

	otel.SetMeterProvider(meter.Provider)

	return func() { meter.Shutdown() }, nil
}

func NewMeterProvider(ctx context.Context, svc string, mtype MeterType) (*meter, error) {
	switch mtype {
	case MeterTypeOTLP:
		return NewOTLPMeterProvider(ctx, svc)
	case MeterTypePrometheus:
		return NewPrometheusMeterProvider(ctx, svc)
	default:
		return NewIOMeterProvider(ctx, svc)
	}
}

func NewPrometheusMeterProvider(ctx context.Context, svc string) (*meter, error) {
	exp, err := prometheus.New() // is both a reader and exporter
	if err != nil {
		return nil, fmt.Errorf("error setting up prometheus exporter: %w", err)
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(exp),
		metric.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(svc),
		)),
	)

	return &meter{
		Provider: mp,
		Shutdown: func() {
			_ = exp.Shutdown(ctx)
		},
	}, nil
}

func NewIOMeterProvider(ctx context.Context, svc string) (*meter, error) {
	exp, err := stdoutmetric.New()
	if err != nil {
		return nil, fmt.Errorf("error setting up stdout metric exporter: %w", err)
	}

	reader := metric.NewPeriodicReader(exp)
	mp := metric.NewMeterProvider(
		metric.WithReader(reader),
		metric.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(svc),
		)),
	)

	return &meter{
		Provider: mp,
		Shutdown: func() {
			_ = exp.Shutdown(ctx)
		},
	}, nil
}

func NewOTLPMeterProvider(ctx context.Context, svc string) (*meter, error) {
	endpoint := os.Getenv("OTEL_METRICS_COLLECTOR_ENDPOINT")
	if endpoint == "" {
		endpoint = "otel-collector:4317"
	}

	// NOTE:
	// assuming the otel collector is within the same private network, we can
	// skip grpc auth, but probably still better to get it work for production
	conn, err := grpc.Dial(endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to metrics otel collector via grpc: %w", err)
	}

	exp, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithGRPCConn(conn),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create new otlp metrics exporter: %w", err)
	}

	reader := metric.NewPeriodicReader(exp)
	mp := metric.NewMeterProvider(
		metric.WithReader(reader),
		metric.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(svc),
		)),
	)

	return &meter{
		Provider: mp,
		Shutdown: func() {
			_ = exp.Shutdown(ctx)
		},
	}, nil
}

type meter struct {
	Provider *metric.MeterProvider
	Shutdown func()
}
