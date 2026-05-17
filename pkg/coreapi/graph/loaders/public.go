package loader

import (
	"context"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
)

func ConvertRunSpan(ctx context.Context, span *cqrs.OtelSpan) (*models.RunTraceSpan, error) {
	return (&traceReader{}).convertRunSpanToGQL(ctx, span)
}
