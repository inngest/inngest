package resolvers

import (
	"context"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
)

func (r *queryResolver) Runs(ctx context.Context, num int, cur *string, order []*models.RunsV2OrderBy, filter models.RunsFilterV2) (*models.RunsV2Connection, error) {
	tsfield := enums.TraceRunTimeQueuedAt
	switch *filter.TimeField {
	case models.FunctionRunTimeFieldV2StartedAt:
		tsfield = enums.TraceRunTimeStartedAt
	case models.FunctionRunTimeFieldV2EndedAt:
		tsfield = enums.TraceRunTimeEndedAt
	}

	statuses := []enums.RunStatus{}
	if len(filter.Status) > 0 {
		for _, s := range filter.Status {
			var status enums.RunStatus
			switch s {
			case models.FunctionRunStatusQueued:
				status = enums.RunStatusScheduled
			case models.FunctionRunStatusRunning:
				status = enums.RunStatusRunning
			case models.FunctionRunStatusCompleted:
				status = enums.RunStatusCompleted
			case models.FunctionRunStatusCancelled:
				status = enums.RunStatusCancelled
			case models.FunctionRunStatusFailed:
				status = enums.RunStatusFailed
			default:
				// unknown status
				continue
			}
			statuses = append(statuses, status)
		}
	}

	orderBy := []cqrs.GetTraceRunOrder{}
	for _, o := range order {
		field := enums.TraceRunTimeQueuedAt
		dir := enums.TraceRunOrderDesc

		switch o.Field {
		case models.RunsV2OrderByFieldQueuedAt:
			field = enums.TraceRunTimeQueuedAt
		case models.RunsV2OrderByFieldStartedAt:
			field = enums.TraceRunTimeStartedAt
		case models.RunsV2OrderByFieldEndedAt:
			field = enums.TraceRunTimeEndedAt
		default: // unknown, skip
			continue
		}

		switch o.Direction {
		case models.RunsOrderByDirectionAsc:
			dir = enums.TraceRunOrderAsc
		case models.RunsOrderByDirectionDesc:
			dir = enums.TraceRunOrderDesc
		default: // unknown, skip
			continue
		}

		orderBy = append(orderBy, cqrs.GetTraceRunOrder{Field: field, Direction: dir})
	}

	var cursor string
	if cur != nil {
		cursor = *cur
	}

	var cel string
	if filter.Query != nil {
		cel = *filter.Query
	}

	opts := cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AppID:      filter.AppIDs,
			FunctionID: filter.FunctionIDs,
			TimeField:  tsfield,
			Status:     statuses,
			CEL:        cel,
		},
		Order:  orderBy,
		Cursor: cursor,
		Items:  uint(num),
	}

	runs, err := r.Data.GetTraceRuns(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("error retrieving runs: %w", err)
	}

	edges := []*models.FunctionRunV2Edge{}
	for _, r := range runs {
		var (
			started  *time.Time
			ended    *time.Time
			sourceID *string
			output   *string
		)

		if r.StartedAt.UnixMilli() > 0 {
			started = &r.StartedAt
		}
		if r.EndedAt.UnixMilli() > 0 {
			ended = &r.EndedAt
		}
		if len(r.SourceID) > 0 {
			sourceID = &r.SourceID
		}
		if len(r.Output) > 0 {
			s := string(r.Output)
			output = &s
		}

		node := &models.FunctionRunV2{
			ID:         r.RunID,
			AppID:      r.AppID,
			FunctionID: r.FunctionID,
			TraceID:    r.TraceID,
			QueuedAt:   r.QueuedAt,
			StartedAt:  started,
			EndedAt:    ended,
			SourceID:   sourceID,
			TriggerIDs: r.TriggerIDs,
			Triggers:   []string{},
			Output:     output,
			IsBatch:    r.IsBatch,
		}

		edges = append(edges, &models.FunctionRunV2Edge{
			Node:   node,
			Cursor: "",
		})
	}

	return &models.RunsV2Connection{
		Edges:    edges,
		PageInfo: &models.PageInfo{},
	}, nil
}
