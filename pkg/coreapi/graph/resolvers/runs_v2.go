package resolvers

import (
	"context"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

const (
	defaultRunItems = 40
	maxRunItems     = 400
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

	opts := cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AppID:      filter.AppIDs,
			FunctionID: filter.FunctionIDs,
			TimeField:  tsfield,
			From:       filter.From,
			Until:      until,
			Status:     statuses,
			CEL:        cel,
		},
		Order:  orderBy,
		Cursor: cursor,
		Items:  uint(items),
	}

	runs, err := r.Data.GetTraceRuns(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("error retrieving runs: %w", err)
	}

	var (
		scursor *string
		ecursor *string
	)
	edges := []*models.FunctionRunV2Edge{}
	total := len(runs)
	for i, r := range runs {
		var (
			started  *time.Time
			ended    *time.Time
			sourceID *string
			output   *string
		)

		c := r.Cursor
		if i == 0 {
			scursor = &c // start cursor
		}
		if i == total-1 {
			ecursor = &c // end cursor
		}

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

		status, err := models.ToFunctionRunStatus(r.Status)
		if err != nil {
			continue
		}

		triggerIDS := []ulid.ULID{}
		for _, tid := range r.TriggerIDs {
			if id, err := ulid.Parse(tid); err == nil {
				triggerIDS = append(triggerIDS, id)
			}
		}

		node := &models.FunctionRunV2{
			ID:         ulid.MustParse(r.RunID),
			AppID:      r.AppID,
			FunctionID: r.FunctionID,
			TraceID:    r.TraceID,
			QueuedAt:   r.QueuedAt,
			StartedAt:  started,
			EndedAt:    ended,
			SourceID:   sourceID,
			Status:     status,
			TriggerIDs: triggerIDS,
			Triggers:   []string{},
			Output:     output,
			IsBatch:    r.IsBatch,
		}

		edges = append(edges, &models.FunctionRunV2Edge{
			Node:   node,
			Cursor: r.Cursor,
		})
	}

	pageInfo := &models.PageInfo{
		HasNextPage: total == int(opts.Items),
		StartCursor: scursor,
		EndCursor:   ecursor,
	}

	return &models.RunsV2Connection{
		Edges:    edges,
		PageInfo: pageInfo,
	}, nil
}

func (r *queryResolver) Run(ctx context.Context, runID string) (*models.FunctionRunV2, error) {
	runid, err := ulid.Parse(runID)
	if err != nil {
		return nil, fmt.Errorf("error parsing runID: %w", err)
	}

	run, err := r.Data.GetTraceRun(ctx, cqrs.TraceRunIdentifier{RunID: runid})
	if err != nil {
		return nil, fmt.Errorf("error retrieving run: %w", err)
	}

	var (
		startedAt *time.Time
		endedAt   *time.Time
		sourceID  *string
		output    *string
		batchTS   *time.Time
	)

	if run.StartedAt.UnixMilli() > 0 {
		startedAt = &run.StartedAt
	}
	if run.EndedAt.UnixMilli() > 0 {
		endedAt = &run.EndedAt
	}
	if run.SourceID != "" {
		sourceID = &run.SourceID
	}

	triggerIDs := []ulid.ULID{}
	for _, evtID := range run.TriggerIDs {
		if id, err := ulid.Parse(evtID); err == nil {
			triggerIDs = append(triggerIDs, id)
		}
	}

	triggers := []string{}
	for _, byt := range run.Triggers {
		triggers = append(triggers, string(byt))
	}

	if len(run.Output) > 0 {
		o := string(run.Output)
		output = &o
	}

	if run.BatchID != nil {
		ts := ulid.Time(run.BatchID.Time())
		batchTS = &ts
	}

	status, err := models.ToFunctionRunStatus(run.Status)
	if err != nil {
		return nil, fmt.Errorf("error parsing status: %w", err)
	}

	res := models.FunctionRunV2{
		ID:             runid,
		AppID:          run.AppID,
		FunctionID:     run.FunctionID,
		TraceID:        run.TraceID,
		QueuedAt:       run.QueuedAt,
		StartedAt:      startedAt,
		EndedAt:        endedAt,
		Status:         status,
		SourceID:       sourceID,
		TriggerIDs:     triggerIDs,
		Triggers:       triggers,
		IsBatch:        run.IsBatch,
		BatchCreatedAt: batchTS,
		CronSchedule:   run.CronSchedule,
		Output:         output,
	}

	return &res, nil
}

func (r *queryResolver) RunTraceSpanOutputByID(ctx context.Context, outputID string) (*models.RunTraceSpanOutput, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *queryResolver) RunTrigger(ctx context.Context, runID string) (*models.RunTraceTrigger, error) {
	return nil, fmt.Errorf("not implemented")
}
