package trace

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// clearOTLPEnv makes the OTLP environment hermetic: mirror.go and the upstream
// exporters both read process env directly, and compression or insecure values
// left in a developer's shell would break the end-to-end tests.
func clearOTLPEnv(t *testing.T) {
	t.Helper()

	for _, k := range []string{
		envEndpoint,
		envSignalEndpoint,
		envProtocol,
		envSignalProtocol,
		"OTEL_EXPORTER_OTLP_COMPRESSION",
		"OTEL_EXPORTER_OTLP_TRACES_COMPRESSION",
		"OTEL_EXPORTER_OTLP_INSECURE",
		"OTEL_EXPORTER_OTLP_TRACES_INSECURE",
	} {
		t.Setenv(k, "")
	}
}

func TestMirrorEndpointResolution(t *testing.T) {
	const (
		generic = "http://generic-collector:4318"
		signal  = "http://signal-collector:4318"
	)

	tests := []struct {
		name        string
		env         map[string]string
		expected    string
		wantEnabled bool
	}{
		{
			name:        "no endpoint leaves the mirror disabled",
			env:         map[string]string{},
			expected:    "",
			wantEnabled: false,
		},
		{
			name:        "generic endpoint enables the mirror",
			env:         map[string]string{envEndpoint: generic},
			expected:    generic,
			wantEnabled: true,
		},
		{
			name:        "signal endpoint enables the mirror",
			env:         map[string]string{envSignalEndpoint: signal},
			expected:    signal,
			wantEnabled: true,
		},
		{
			name: "signal endpoint wins over generic",
			env: map[string]string{
				envEndpoint:       generic,
				envSignalEndpoint: signal,
			},
			expected:    signal,
			wantEnabled: true,
		},
		{
			name: "whitespace-only signal endpoint falls back to generic",
			env: map[string]string{
				envEndpoint:       generic,
				envSignalEndpoint: "   ",
			},
			expected:    generic,
			wantEnabled: true,
		},
		{
			name: "whitespace-only endpoints leave the mirror disabled",
			env: map[string]string{
				envEndpoint:       " \t\n ",
				envSignalEndpoint: "  ",
			},
			expected:    "",
			wantEnabled: false,
		},
		{
			name:        "surrounding whitespace is trimmed off the endpoint",
			env:         map[string]string{envSignalEndpoint: "  " + signal + "\n"},
			expected:    signal,
			wantEnabled: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearOTLPEnv(t)
			for k, v := range test.env {
				t.Setenv(k, v)
			}

			require.Equal(t, test.expected, mirrorEndpoint())
			require.Equal(t, test.wantEnabled, mirrorEnabled())
		})
	}
}

func TestMirrorProtocolResolution(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		expected string
	}{
		{
			// The spec mandates http/protobuf when nothing is set.
			name:     "unset protocol resolves to the spec-mandated protocol",
			env:      map[string]string{},
			expected: protocolHTTP,
		},
		{
			name:     "generic protocol is honoured",
			env:      map[string]string{envProtocol: protocolGRPC},
			expected: protocolGRPC,
		},
		{
			name:     "signal protocol is honoured",
			env:      map[string]string{envSignalProtocol: protocolGRPC},
			expected: protocolGRPC,
		},
		{
			name: "signal protocol wins over generic",
			env: map[string]string{
				envProtocol:       protocolHTTP,
				envSignalProtocol: protocolGRPC,
			},
			expected: protocolGRPC,
		},
		{
			name: "whitespace-only signal protocol falls back to generic",
			env: map[string]string{
				envProtocol:       protocolGRPC,
				envSignalProtocol: "   ",
			},
			expected: protocolGRPC,
		},
		{
			name: "whitespace-only protocols fall back to the spec protocol",
			env: map[string]string{
				envProtocol:       "  ",
				envSignalProtocol: " \t ",
			},
			expected: protocolHTTP,
		},
		{
			name:     "surrounding whitespace is trimmed off the protocol",
			env:      map[string]string{envSignalProtocol: " " + protocolGRPC + "\n"},
			expected: protocolGRPC,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearOTLPEnv(t)
			for k, v := range test.env {
				t.Setenv(k, v)
			}

			require.Equal(t, test.expected, mirrorProtocol())
		})
	}
}

// TestNewMirrorSpanProcessorWithoutEndpoint pins the invariant the whole
// feature rests on: no endpoint, no mirror, and that is not an error.
func TestNewMirrorSpanProcessorWithoutEndpoint(t *testing.T) {
	clearOTLPEnv(t)

	// A protocol on its own must not be enough to turn the mirror on.
	t.Setenv(envSignalProtocol, protocolGRPC)

	sp, err := newMirrorSpanProcessor(context.Background())
	require.NoError(t, err)
	require.Nil(t, sp)
}

func TestNewMirrorSpanProcessorSupportedProtocols(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
	}{
		{name: "grpc", protocol: protocolGRPC},
		{name: "http/protobuf", protocol: protocolHTTP},
		{name: "unset protocol still yields a usable mirror", protocol: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()

			clearOTLPEnv(t)
			t.Setenv(envSignalEndpoint, "http://127.0.0.1:4318")
			if test.protocol != "" {
				t.Setenv(envSignalProtocol, test.protocol)
			}

			sp, err := newMirrorSpanProcessor(ctx)
			require.NoError(t, err)
			require.NotNil(t, sp)
			require.NoError(t, sp.Shutdown(ctx))
		})
	}
}

// Asking for a protocol we cannot speak must be reported, not silently served
// over a different wire format.
func TestNewMirrorSpanProcessorRejectsUnsupportedProtocol(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
	}{
		{
			name:     "http/json has no Go trace exporter",
			protocol: "http/json",
		},
		{
			name:     "unknown protocol",
			protocol: "thrift",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearOTLPEnv(t)
			t.Setenv(envSignalEndpoint, "http://127.0.0.1:4318")
			t.Setenv(envSignalProtocol, test.protocol)

			sp, err := newMirrorSpanProcessor(context.Background())
			require.Nil(t, sp)
			require.Error(t, err)

			require.ErrorContains(t, err, test.protocol)
			require.ErrorContains(t, err, protocolHTTP)
			require.ErrorContains(t, err, protocolGRPC)
		})
	}
}

// TestNewTracerExportsToSinkAndMirror is the contract that justifies the
// feature: one span reaches both the tracer's own sink (`/dev/traces`, which
// feeds the run details UI) and the external collector.
func TestNewTracerExportsToSinkAndMirror(t *testing.T) {
	ctx := context.Background()

	clearOTLPEnv(t)

	sink := newTraceSink(t)
	collector := newTraceSink(t)

	// The exporter reads this itself, and http:// is what makes it plaintext.
	// A per-signal endpoint is used as-is, so a URL with no path is served at
	// "/" - hence the sink records paths instead of asserting one.
	t.Setenv(envSignalEndpoint, collector.URL)

	tracer, err := newTracer(ctx, TracerOpts{
		Type:          TracerTypeOTLPHTTP,
		TraceEndpoint: sink.Listener.Addr().String(),
		TraceURLPath:  "/dev/traces",
		ServiceName:   "test",
	})
	require.NoError(t, err)
	t.Cleanup(tracer.Shutdown(ctx))

	_, span := tracer.Provider().Tracer("test").Start(ctx, "mirrored-span")
	span.End()

	// ForceFlush drains every processor on the provider, and each OTLP export
	// blocks until its collector responds, so no polling is needed.
	require.NoError(t, tracer.Provider().ForceFlush(ctx))

	// Asserting service.name proves the mirror rides the tracer's own provider
	// rather than standing up an unattributed one.
	expected := map[string]string{"mirrored-span": "test"}
	require.Equal(t, expected, sink.spans(t), "internal sink did not receive the span")
	require.Equal(t, expected, collector.spans(t), "external collector did not receive the mirrored span")

	// The mirror must not repoint the internal sink.
	for _, req := range sink.received() {
		require.Equal(t, "/dev/traces", req.path)
	}
}

// TestNewTracerWithoutMirrorEnv is the inverse regression: unset variables mean
// no mirror and pre-feature behaviour.
func TestNewTracerWithoutMirrorEnv(t *testing.T) {
	ctx := context.Background()

	clearOTLPEnv(t)

	mirror, err := newMirrorSpanProcessor(ctx)
	require.NoError(t, err)
	require.Nil(t, mirror, "an unconfigured process must not build a mirror")

	sink := newTraceSink(t)

	tracer, err := newTracer(ctx, TracerOpts{
		Type:          TracerTypeOTLPHTTP,
		TraceEndpoint: sink.Listener.Addr().String(),
		TraceURLPath:  "/dev/traces",
		ServiceName:   "test",
	})
	require.NoError(t, err)
	t.Cleanup(tracer.Shutdown(ctx))

	_, span := tracer.Provider().Tracer("test").Start(ctx, "unmirrored-span")
	span.End()

	require.NoError(t, tracer.Provider().ForceFlush(ctx))

	require.Equal(t, map[string]string{"unmirrored-span": "test"}, sink.spans(t))
}

// TestTracerExportFansOutToSinkAndMirror covers the path carrying the entire
// legacy run and step span tree: pkg/run ends its hand-built ReadOnlySpans
// through Export, which writes to the processors and never touches the
// provider. No provider-level test can catch a gap here.
func TestTracerExportFansOutToSinkAndMirror(t *testing.T) {
	ctx := context.Background()

	clearOTLPEnv(t)

	sink := newTraceSink(t)
	collector := newTraceSink(t)

	// Per-signal endpoints are used verbatim, so spell out the full path.
	t.Setenv(envSignalEndpoint, collector.URL+"/v1/traces")

	tracer, err := newTracer(ctx, TracerOpts{
		Type:          TracerTypeOTLPHTTP,
		TraceEndpoint: sink.Listener.Addr().String(),
		TraceURLPath:  "/dev/traces",
		ServiceName:   "test",
	})
	require.NoError(t, err)
	t.Cleanup(tracer.Shutdown(ctx))

	require.NoError(t, tracer.Export(exportedSpan("exported-span")))

	// Both processors are registered on the provider, so its ForceFlush drains
	// them however the span was queued. Shutdown would too, but also closes
	// the tracer.
	require.NoError(t, tracer.Provider().ForceFlush(ctx))

	// Export bypasses the provider, so the resource reaching a collector is the
	// span's own. Pinning it proves the mirror ships the span it was handed.
	expected := map[string]string{"exported-span": "test"}
	require.Equal(t, expected, sink.spans(t), "internal sink did not receive the exported span")
	require.Equal(t, expected, collector.spans(t), "external collector did not receive the exported span")

	for _, req := range sink.received() {
		require.Equal(t, "/dev/traces", req.path)
	}
}

// TestTracerExportWithoutMirror is the other half: with no mirror to fan out
// to, Export must still reach the sink rather than nil-deref or bail early.
func TestTracerExportWithoutMirror(t *testing.T) {
	ctx := context.Background()

	clearOTLPEnv(t)

	sink := newTraceSink(t)

	tr, err := newTracer(ctx, TracerOpts{
		Type:          TracerTypeOTLPHTTP,
		TraceEndpoint: sink.Listener.Addr().String(),
		TraceURLPath:  "/dev/traces",
		ServiceName:   "test",
	})
	require.NoError(t, err)
	t.Cleanup(tr.Shutdown(ctx))

	concrete, ok := tr.(*tracer)
	require.True(t, ok)
	require.Nil(t, concrete.mirror, "an unconfigured process must not attach a mirror to the tracer")

	require.NoError(t, tr.Export(exportedSpan("unmirrored-export")))
	require.NoError(t, tr.Provider().ForceFlush(ctx))

	require.Equal(t, map[string]string{"unmirrored-export": "test"}, sink.spans(t))
}

// TestNewTracerDegradesOnUnsupportedMirrorProtocol guards against a telemetry
// typo taking the server down: an error here propagates out of every component
// that traces and prevents startup, for a signal nothing depends on.
func TestNewTracerDegradesOnUnsupportedMirrorProtocol(t *testing.T) {
	ctx := context.Background()

	clearOTLPEnv(t)

	sink := newTraceSink(t)

	// Nothing needs to listen: the protocol is rejected before an exporter is
	// built. The endpoint only has to be non-empty to ask for a mirror.
	t.Setenv(envSignalEndpoint, "http://127.0.0.1:4318/v1/traces")
	t.Setenv(envSignalProtocol, "http/json")

	tr, err := newTracer(ctx, TracerOpts{
		Type:          TracerTypeOTLPHTTP,
		TraceEndpoint: sink.Listener.Addr().String(),
		TraceURLPath:  "/dev/traces",
		ServiceName:   "test",
	})
	require.NoError(t, err, "a mirror misconfiguration must not stop the server from booting")
	require.NotNil(t, tr)
	t.Cleanup(tr.Shutdown(ctx))

	concrete, ok := tr.(*tracer)
	require.True(t, ok)
	require.Nil(t, concrete.mirror, "a rejected mirror must not be attached to the tracer")

	// Degraded, not dead: the sink the platform itself depends on still works.
	require.NoError(t, tr.Export(exportedSpan("degraded-export")))
	require.NoError(t, tr.Provider().ForceFlush(ctx))

	require.Equal(t, map[string]string{"degraded-export": "test"}, sink.spans(t))
}

// traceSink is an in-process stand-in for an OTLP/HTTP collector, recording
// every request so tests can assert which sinks a span reached.
type traceSink struct {
	*httptest.Server

	mu       sync.Mutex
	requests []sinkRequest
}

type sinkRequest struct {
	method      string
	path        string
	contentType string
	body        []byte
}

func newTraceSink(t *testing.T) *traceSink {
	t.Helper()

	s := &traceSink{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		s.mu.Lock()
		s.requests = append(s.requests, sinkRequest{
			method:      r.Method,
			path:        r.URL.Path,
			contentType: r.Header.Get("Content-Type"),
			body:        body,
		})
		s.mu.Unlock()

		// An empty 200 is a valid OTLP/HTTP success response.
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(s.Close)

	return s
}

func (s *traceSink) received() []sinkRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.requests)
}

// spans decodes everything the sink was sent and returns span name ->
// service.name of the resource that carried it.
func (s *traceSink) spans(t *testing.T) map[string]string {
	t.Helper()

	out := map[string]string{}
	for _, req := range s.received() {
		require.Equal(t, http.MethodPost, req.method)
		require.Equal(t, "application/x-protobuf", req.contentType)
		require.NotEmpty(t, req.body)

		traces, err := (&ptrace.ProtoUnmarshaler{}).UnmarshalTraces(req.body)
		require.NoError(t, err, "sink was sent a body that is not OTLP protobuf")

		for i := range traces.ResourceSpans().Len() {
			rs := traces.ResourceSpans().At(i)

			var svc string
			if v, ok := rs.Resource().Attributes().Get(string(semconv.ServiceNameKey)); ok {
				svc = v.Str()
			}

			for j := range rs.ScopeSpans().Len() {
				spans := rs.ScopeSpans().At(j).Spans()
				for k := range spans.Len() {
					out[spans.At(k).Name()] = svc
				}
			}
		}
	}

	return out
}

// exportedSpan stands in for the spans pkg/run hands to Export. A real
// run.Span cannot be used because pkg/run imports this package. Fixed
// timestamps and IDs keep the payload identical across runs, and the resource
// carries service.name because Export bypasses the provider that would supply
// one.
func exportedSpan(name string) trace.ReadOnlySpan {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	return tracetest.SpanStub{
		Name: name,
		SpanContext: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
			TraceID:    oteltrace.TraceID{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0x10},
			SpanID:     oteltrace.SpanID{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8},
			TraceFlags: oteltrace.FlagsSampled,
		}),
		StartTime: start,
		EndTime:   start.Add(time.Millisecond),
		Resource: resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("test"),
		),
	}.Snapshot()
}
