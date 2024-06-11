package resolvers

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
)

func (r *functionRunV2Resolver) Function(ctx context.Context, fn *models.FunctionRunV2) (*models.Function, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *functionRunV2Resolver) Trace(ctx context.Context, fn *models.FunctionRunV2) (*models.RunTraceSpan, error) {
	return nil, fmt.Errorf("not implemented")
}
