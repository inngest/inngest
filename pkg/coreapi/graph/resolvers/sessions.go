package resolvers

import (
	"context"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
)

func (qr *queryResolver) SessionKeys(ctx context.Context, search *string) ([]*models.SessionKey, error) {
	query := ""
	if search != nil {
		query = *search
	}

	keys, err := qr.Data.GetSessionKeys(ctx, consts.DevServerEnvID, query)
	if err != nil {
		return nil, err
	}

	out := make([]*models.SessionKey, 0, len(keys))
	for _, key := range keys {
		out = append(out, &models.SessionKey{
			SessionKey: key.SessionKey,
			CreatedAt:  key.CreatedAt,
		})
	}
	return out, nil
}

func (qr *queryResolver) Sessions(ctx context.Context, sessionKey string, sessionIDSearch *string, timeRange *models.TimeRangeInput) ([]*models.SessionGroup, error) {
	search := ""
	if sessionIDSearch != nil {
		search = *sessionIDSearch
	}

	groups, err := qr.Data.GetSessions(ctx, consts.DevServerEnvID, sessionKey, search, sessionTimeRange(timeRange))
	if err != nil {
		return nil, err
	}

	out := make([]*models.SessionGroup, 0, len(groups))
	for _, group := range groups {
		functions := make([]*models.SessionFunction, 0, len(group.Functions))
		for _, fn := range group.Functions {
			functions = append(functions, &models.SessionFunction{Slug: fn.Slug, Name: fn.Name})
		}

		out = append(out, &models.SessionGroup{
			SessionKey:     group.SessionKey,
			SessionID:      group.SessionID,
			RunCount:       group.RunCount,
			FailedRunCount: group.FailedRunCount,
			FailureRate:    group.FailureRate,
			LastActiveAt:   group.LastActiveAt,
			Functions:      functions,
		})
	}
	return out, nil
}

func (qr *queryResolver) SessionRuns(ctx context.Context, sessionKey string, sessionID string, timeRange *models.TimeRangeInput) ([]*models.SessionRun, error) {
	runs, err := qr.Data.GetSessionRuns(ctx, consts.DevServerEnvID, sessionKey, sessionID, sessionTimeRange(timeRange))
	if err != nil {
		return nil, err
	}

	out := make([]*models.SessionRun, 0, len(runs))
	for _, run := range runs {
		out = append(out, &models.SessionRun{
			ID:           run.ID.String(),
			FunctionSlug: run.FunctionSlug,
			EventName:    run.EventName,
			Status:       run.Status.String(),
			QueuedAt:     run.QueuedAt,
			StartedAt:    run.StartedAt,
			EndedAt:      run.EndedAt,
		})
	}
	return out, nil
}

func sessionTimeRange(input *models.TimeRangeInput) cqrs.SessionTimeRange {
	now := time.Now()
	if input == nil {
		return cqrs.SessionTimeRange{From: now.Add(-7 * 24 * time.Hour), Until: now}
	}

	until := now
	if input.Until != nil {
		until = *input.Until
	}
	return cqrs.SessionTimeRange{From: input.From, Until: until}
}
