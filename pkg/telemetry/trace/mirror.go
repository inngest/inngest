package trace

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/trace"
)

// `inngest dev` and `inngest start` point their tracers at the server's own
// `/dev/traces` endpoint, which persists spans for the run details UI, so
// repointing that exporter would silently empty the UI. The mirror is
// additive instead: spans go to an external OTLP collector as well as the
// tracer's own sink, and only when one of these is set.
//
// Endpoint, headers, TLS, compression and timeout are all read by the
// upstream exporters themselves; we only pick whether and over what.
const (
	// Path semantics differ, per spec: the generic endpoint is a base URL
	// with "/v1/traces" appended, the signal-specific one is used verbatim
	// and must already carry the path. The signal-specific one wins.
	envEndpoint       = "OTEL_EXPORTER_OTLP_ENDPOINT"
	envSignalEndpoint = "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"

	envProtocol       = "OTEL_EXPORTER_OTLP_PROTOCOL"
	envSignalProtocol = "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL"

	protocolGRPC = "grpc"
	protocolHTTP = "http/protobuf"
)

func mirrorEnabled() bool {
	return mirrorEndpoint() != ""
}

func mirrorEndpoint() string {
	if v := strings.TrimSpace(os.Getenv(envSignalEndpoint)); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv(envEndpoint))
}

func mirrorProtocol() string {
	if v := strings.TrimSpace(os.Getenv(envSignalProtocol)); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv(envProtocol)); v != "" {
		return v
	}
	// Spec default.
	return protocolHTTP
}

// newMirrorSpanProcessor returns a nil processor, and no error, when no
// external endpoint is configured.
func newMirrorSpanProcessor(ctx context.Context) (trace.SpanProcessor, error) {
	if !mirrorEnabled() {
		return nil, nil
	}

	var (
		exp *otlptrace.Exporter
		err error
	)

	switch proto := mirrorProtocol(); proto {
	case protocolGRPC:
		// Endpoint deliberately left to the exporter: it reads the env vars
		// itself and derives TLS from the scheme, so http:// needs no
		// separate insecure flag.
		exp, err = otlptracegrpc.New(ctx)
	case protocolHTTP:
		exp, err = otlptracehttp.New(ctx)
	default:
		return nil, fmt.Errorf(
			"unsupported OTLP trace protocol %q: use %q or %q",
			proto, protocolHTTP, protocolGRPC,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("error creating mirrored OTLP trace exporter: %w", err)
	}

	return trace.NewBatchSpanProcessor(exp), nil
}
