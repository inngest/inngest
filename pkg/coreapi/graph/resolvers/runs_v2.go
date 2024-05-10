package resolvers

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
)

func (r *queryResolver) Runs(ctx context.Context, num int, cursor *string, order []*models.RunsV2OrderBy, filter models.RunsFilterV2) (*models.RunsV2Connection, error) {
	return nil, fmt.Errorf("not implemented")
}
