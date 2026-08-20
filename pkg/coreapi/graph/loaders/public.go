package loader

import (
	"context"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
)

func ConvertRunSpan(ctx context.Context, span *cqrs.OtelSpan) (*models.RunTraceSpan, error) {
	return (&traceReader{}).convertRunSpanToGQL(ctx, span)
}

// ConvertRunSpanFor is ConvertRunSpan, but selects the flat-tree converter
// (convertFlatSpanToGQL) when reader implements the flatSpanSource marker —
// mirrors NewLoaders' own dispatch (loader.go), for REST call sites (apiv2's
// FunctionTraceReader, devserver's runProvider) that build a *cqrs.OtelSpan
// tree from a reader that may or may not be DuckDB-backed, and so can't
// assume the rollup-aware converter unconditionally the way ConvertRunSpan
// does.
func ConvertRunSpanFor(ctx context.Context, reader any, span *cqrs.OtelSpan) (*models.RunTraceSpan, error) {
	if fs, ok := reader.(flatSpanSource); ok && fs.FlatSpans() {
		return convertFlatSpanToGQL(ctx, span)
	}
	return (&traceReader{}).convertRunSpanToGQL(ctx, span)
}
