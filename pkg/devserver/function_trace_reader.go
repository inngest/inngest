package devserver

import (
	"context"

	"github.com/google/uuid"
	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
)

type cqrsFunctionTraceReader struct {
	reader cqrs.TraceReader
}

func NewFunctionTraceReader(reader cqrs.TraceReader) apiv2.FunctionTraceReader {
	return &cqrsFunctionTraceReader{reader: reader}
}

func (r *cqrsFunctionTraceReader) GetSpansByRunID(ctx context.Context, runID ulid.ULID) (*cqrs.OtelSpan, error) {
	return r.reader.GetSpansByRunID(ctx, runID)
}

func (r *cqrsFunctionTraceReader) GetSpanOutput(ctx context.Context, id cqrs.SpanIdentifier) (*cqrs.SpanOutput, error) {
	return r.reader.GetSpanOutput(ctx, id)
}

func (r *cqrsFunctionTraceReader) GetStepSpanByStepID(ctx context.Context, runID ulid.ULID, stepID string, accountID, workspaceID uuid.UUID) (*cqrs.OtelSpan, error) {
	return r.reader.GetStepSpanByStepID(ctx, runID, stepID, accountID, workspaceID)
}

// FlatSpans forwards the wrapped reader's own flatSpanSource marker (see
// pkg/cqrs/duckdbquery.Manager.FlatSpans) so loader.ConvertRunSpanFor can
// pick the right *cqrs.OtelSpan-to-REST converter through this wrapper —
// without it, the marker would be hidden behind cqrsFunctionTraceReader's
// unexported `reader` field and every apiv2 trace response would silently
// fall back to the rollup-aware converter even when DuckDB-backed.
func (r *cqrsFunctionTraceReader) FlatSpans() bool {
	fs, ok := r.reader.(interface{ FlatSpans() bool })
	return ok && fs.FlatSpans()
}
