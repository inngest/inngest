package devserver

import (
	"testing"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/stretchr/testify/require"
)

// fakeFlatTraceReader embeds a nil cqrs.TraceReader to satisfy the large
// interface for free (its methods are never called in this test) and adds
// the flatSpanSource marker cqrsFunctionTraceReader must forward.
type fakeFlatTraceReader struct{ cqrs.TraceReader }

func (fakeFlatTraceReader) FlatSpans() bool { return true }

// TestFunctionTraceReaderForwardsFlatSpansMarker proves
// cqrsFunctionTraceReader.FlatSpans() forwards its wrapped reader's own
// marker — without this, loader.ConvertRunSpanFor would never detect a
// DuckDB-backed reader through this wrapper (see FlatSpans' doc comment on
// function_trace_reader.go) and apiv2 trace responses would silently use
// the wrong (rollup-aware) span converter.
func TestFunctionTraceReaderForwardsFlatSpansMarker(t *testing.T) {
	flat := NewFunctionTraceReader(fakeFlatTraceReader{})
	fs, ok := flat.(interface{ FlatSpans() bool })
	require.True(t, ok, "cqrsFunctionTraceReader must implement FlatSpans()")
	require.True(t, fs.FlatSpans())

	plain := NewFunctionTraceReader(struct{ cqrs.TraceReader }{})
	fs2, ok := plain.(interface{ FlatSpans() bool })
	require.True(t, ok, "cqrsFunctionTraceReader must implement FlatSpans()")
	require.False(t, fs2.FlatSpans(), "a reader with no FlatSpans marker must default to false")
}
