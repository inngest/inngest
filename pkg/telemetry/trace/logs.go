package trace

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/telemetry/exporters"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

// tldr:  export traces as logs to any OTLP capable endpoint.
//
// newSpansAsLogsProcessor builds an OTLP/HTTP logs LoggerProvider and returns a
// SpanProcessor that converts every ended in-allowlist span into a LogRecord
//
// When opts.OTLPLogs.Enabled() is false this returns (nil, nil, nil) and the
// caller should skip wiring the side pipeline.
//
// The returned shutdown closes the LoggerProvider (which flushes the batch
// processor and shuts the exporter down) and is safe to call once.
func newSpansAsLogsProcessor(ctx context.Context, opts TracerOpts) (trace.SpanProcessor, func(context.Context) error, error) {
	if !opts.OTLPLogs.Enabled() {
		return nil, nil, nil
	}

	exp, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(opts.OTLPLogs.Endpoint),
		otlploghttp.WithURLPath(opts.OTLPLogs.urlPath()),
		otlploghttp.WithInsecure(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating otlp logs exporter: %w", err)
	}

	provider := log.NewLoggerProvider(
		log.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(opts.ServiceName),
		)),
		log.WithProcessor(log.NewBatchProcessor(exp)),
	)

	logger := provider.Logger("inngest.spans_as_logs")

	procOpts := []exporters.SpansAsLogsOpt{}
	if opts.OTLPLogs.PayloadCapBytes > 0 {
		procOpts = append(procOpts, exporters.WithLogsPayloadCapBytes(opts.OTLPLogs.PayloadCapBytes))
	}

	return exporters.NewSpansAsLogsProcessor(logger, procOpts...),
		provider.Shutdown,
		nil
}
