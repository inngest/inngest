package apiv2

import (
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
)

const (
	defaultRunItems = 40
	maxRunItems     = 400
)

// mapListRunsFilter converts v2 API filter parameters to GraphQL filter model
func (s *Service) mapListRunsFilter(req *apiv2.ListRunsRequest, startTime, endTime time.Time) (models.RunsFilterV2, error) {
	// Determine time field for filtering
	timeField := models.RunsV2OrderByFieldQueuedAt
	if req.TimeField != nil {
		switch *req.TimeField {
		case apiv2.RunTimeField_QUEUED_AT:
			timeField = models.RunsV2OrderByFieldQueuedAt
		case apiv2.RunTimeField_STARTED_AT:
			timeField = models.RunsV2OrderByFieldStartedAt
		case apiv2.RunTimeField_ENDED_AT:
			timeField = models.RunsV2OrderByFieldEndedAt
		}
	}

	// Convert status filters
	var statuses []models.FunctionRunStatus
	for _, protoStatus := range req.Status {
		var status models.FunctionRunStatus
		switch protoStatus {
		case apiv2.RunStatus_RUNNING:
			status = models.FunctionRunStatusRunning
		case apiv2.RunStatus_COMPLETED:
			status = models.FunctionRunStatusCompleted
		case apiv2.RunStatus_FAILED:
			status = models.FunctionRunStatusFailed
		case apiv2.RunStatus_CANCELLED:
			status = models.FunctionRunStatusCancelled
		case apiv2.RunStatus_SCHEDULED:
			status = models.FunctionRunStatusQueued
		default:
			// Skip unknown statuses
			continue
		}
		statuses = append(statuses, status)
	}

	filter := models.RunsFilterV2{
		From:      startTime,
		Until:     &endTime,
		TimeField: &timeField,
		Status:    statuses,
	}

	return filter, nil
}

// mapListRunsOrderBy converts v2 API ordering to GraphQL order model
func (s *Service) mapListRunsOrderBy(req *apiv2.ListRunsRequest) []*models.RunsV2OrderBy {
	// Default ordering: descending by queued time
	orderBy := []*models.RunsV2OrderBy{
		{
			Field:     models.RunsV2OrderByFieldQueuedAt,
			Direction: models.RunsOrderByDirectionDesc,
		},
	}

	return orderBy
}

// toRunsQueryOpt converts filter and order parameters to cqrs query options
// This is adapted from pkg/coreapi/graph/resolvers/runs_v2.go
func (s *Service) toRunsQueryOpt(
	num int,
	cur *string,
	order []*models.RunsV2OrderBy,
	filter models.RunsFilterV2,
) cqrs.GetTraceRunOpt {
	tsfield := enums.TraceRunTimeQueuedAt
	if filter.TimeField != nil {
		switch *filter.TimeField {
		case models.RunsV2OrderByFieldStartedAt:
			tsfield = enums.TraceRunTimeStartedAt
		case models.RunsV2OrderByFieldEndedAt:
			tsfield = enums.TraceRunTimeEndedAt
		}
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
		var (
			field enums.TraceRunTime
			dir   enums.TraceRunOrder
		)

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

	until := time.Now()
	if filter.Until != nil {
		until = *filter.Until
	}

	items := defaultRunItems
	if num > 0 && num < maxRunItems {
		items = num
	}

	var appIDs []uuid.UUID
	if filter.AppIDs != nil {
		appIDs = filter.AppIDs
	}

	var functionIDs []uuid.UUID
	if filter.FunctionIDs != nil {
		functionIDs = filter.FunctionIDs
	}

	return cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AppID:      appIDs,
			FunctionID: functionIDs,
			TimeField:  tsfield,
			From:       filter.From,
			Until:      until,
			Status:     statuses,
			CEL:        cel,
		},
		Order:   orderBy,
		Cursor:  cursor,
		Items:   uint(items),
		Preview: false,
	}
}
