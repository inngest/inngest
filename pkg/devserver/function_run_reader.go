package devserver

import (
	"context"
	"errors"
	"fmt"

	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	loader "github.com/inngest/inngest/pkg/coreapi/graph/loaders"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
)

type spanFunctionRunReader struct {
	reader spanRunReader
}

type spanRunReader interface {
	GetSpansByRunID(ctx context.Context, runID ulid.ULID) (*cqrs.OtelSpan, error)
}

func NewFunctionRunReader(reader spanRunReader) apiv2.FunctionRunReader {
	return &spanFunctionRunReader{reader: reader}
}

func (r *spanFunctionRunReader) GetFunctionRun(ctx context.Context, runID ulid.ULID, _ apiv2.GetFunctionRunOpts) (*cqrs.FunctionRun, error) {
	root, err := r.reader.GetSpansByRunID(ctx, runID)
	if err != nil {
		return nil, err
	}
	if root == nil {
		return nil, errors.New("run not found")
	}
	if root.Attributes == nil {
		root.Attributes = &meta.ExtractedValues{}
	}

	span, err := loader.ConvertRunSpan(ctx, root)
	if err != nil {
		return nil, fmt.Errorf("error converting run span: %w", err)
	}

	eventID, err := runEventID(root)
	if err != nil {
		return nil, err
	}

	startedAt := root.StartTime
	if span.StartedAt != nil {
		startedAt = *span.StartedAt
	}

	status := enums.RunStatusRunning
	if root.Status != enums.StepStatusUnknown {
		status = enums.StepStatusToRunStatus(root.Status)
	}

	run := &cqrs.FunctionRun{
		RunID:        runID,
		RunStartedAt: startedAt,
		FunctionID:   root.GetFunctionID(),
		EventID:      eventID,
		Status:       status,
		EndedAt:      span.EndedAt,
	}

	if root.Attributes.BatchID != nil {
		run.BatchID = root.Attributes.BatchID
	}
	if root.Attributes.CronSchedule != nil {
		run.Cron = root.Attributes.CronSchedule
	}

	return run, nil
}

func runEventID(root *cqrs.OtelSpan) (ulid.ULID, error) {
	if root.Attributes == nil || root.Attributes.EventIDs == nil || len(*root.Attributes.EventIDs) == 0 {
		return ulid.Zero, errors.New("run span missing event ID")
	}

	eventID, err := ulid.Parse((*root.Attributes.EventIDs)[0])
	if err != nil {
		return ulid.Zero, fmt.Errorf("invalid run event ID: %w", err)
	}

	return eventID, nil
}
