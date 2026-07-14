package extractors_test

import (
	"embed"
	"testing"

	"github.com/stretchr/testify/require"
	collecttrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

//go:embed testdata
var fixtures embed.FS

// loadOTLPSpans reads an OTLP/JSON ExportTraceServiceRequest fixture and returns
// its spans, flattened across resource/scope. Fixtures are real spans captured
// from instrumented AI SDKs. This loader decodes them through the identical
// protojson.Unmarshal path the ingest endpoint uses (pkg/api/apiv1/traces.go).
func loadOTLPSpans(t *testing.T, name string) []*tracev1.Span {
	t.Helper()

	body, err := fixtures.ReadFile("testdata/" + name)
	require.NoError(t, err, "read fixture %s", name)

	req := &collecttrace.ExportTraceServiceRequest{}
	require.NoError(t, protojson.Unmarshal(body, req), "unmarshal OTLP fixture %s", name)

	var spans []*tracev1.Span
	for _, rs := range req.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			spans = append(spans, ss.Spans...)
		}
	}
	require.NotEmpty(t, spans, "fixture %s contained no spans", name)
	return spans
}
