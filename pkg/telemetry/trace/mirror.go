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

// Inngest's own tracers are wired to internal sinks: `inngest dev` and `inngest
// start` point them at the server's `/dev/traces` ingestion endpoint, which
// persists spans for the run details UI. Repointing that exporter is not an
// option for self-hosted users who also want their spans in an external
// collector, because it would silently empty the UI.
//
// The mirror is therefore additive: when an OTLP endpoint is configured via the
// standard OpenTelemetry environment variables, spans are exported to it *in
// addition* to whatever sink the tracer was constructed with. Nothing changes
// when the variables are unset.
//
// Configuration is delegated to the upstream OTLP exporters, which read the
// full spec-defined variable set themselves (endpoint, headers, TLS,
// compression, timeout). We only decide whether a mirror is wanted at all, and
// over which protocol.
const (
	// envEndpoint and envSignalEndpoint gate the mirror. Either one being
	// non-empty enables it; the signal-specific variable wins, per spec.
	//
	// Their path semantics differ, also per spec, and the upstream exporters
	// implement it: the generic variable is a base URL that has "/v1/traces"
	// appended, while the signal-specific variable is used verbatim and must
	// therefore already carry the full path.
	envEndpoint       = "OTEL_EXPORTER_OTLP_ENDPOINT"
	envSignalEndpoint = "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"

	// envProtocol and envSignalProtocol select the wire protocol. Spec values
	// are "grpc", "http/protobuf", and "http/json"; the Go SDK has no
	// http/json trace exporter, so it is rejected rather than silently
	// downgraded.
	envProtocol       = "OTEL_EXPORTER_OTLP_PROTOCOL"
	envSignalProtocol = "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL"

	protocolGRPC = "grpc"
	protocolHTTP = "http/protobuf"
)

// mirrorEnabled reports whether an external OTLP trace endpoint is configured.
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

// newMirrorSpanProcessor builds a batch span processor that ships spans to the
// externally configured OTLP endpoint. It returns a nil processor when no
// endpoint is configured, which callers MUST treat as "no mirror wanted"
// rather than an error.
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
		// No WithEndpoint/WithInsecure here on purpose: the exporter reads
		// OTEL_EXPORTER_OTLP_[TRACES_]ENDPOINT itself and derives TLS from the
		// scheme, so `http://` collectors work without a separate insecure
		// flag.
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

	// Registered on the provider, so provider ForceFlush/Shutdown already
	// drains and closes this processor and its exporter.
	return trace.NewBatchSpanProcessor(exp), nil
}
