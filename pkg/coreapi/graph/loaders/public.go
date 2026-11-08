package loader

import (
	"context"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
)

func ConvertRunSpanFor(ctx context.Context, reader any, span *cqrs.OtelSpan) (*models.RunTraceSpan, error) {
	// TODO: this is ugly, but the loader package is already a mess of circular dependencies and this is the least-bad way to avoid having to pass a
	// cqrs.TraceReader through every call site just to check if it's flat or not. If we ever refactor the loader package, this should be cleaned up.
	if flat, ok := reader.(interface{ FlatSpans() bool }); ok && flat.FlatSpans() {
		return convertFlatRunSpanToGQL(ctx, span)
	}

	return convertDynamicRunSpanToGQL(ctx, span)
}
