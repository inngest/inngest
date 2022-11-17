package resolvers

import (
	"context"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
)

func (r *eventResolver) FunctionRuns(ctx context.Context, obj *models.Event) ([]*models.FunctionRun, error) {
	return nil, nil
}
