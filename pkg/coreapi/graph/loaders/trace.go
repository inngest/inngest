package loader

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/graph-gophers/dataloader"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/run"
	rpbv2 "github.com/inngest/inngest/proto/gen/run/v2"
	"github.com/oklog/ulid/v2"
)

var (
	ErrSkipSuccess = fmt.Errorf("skip success span")
)

type TraceRequestKey struct {
	*cqrs.TraceRunIdentifier
}

func (k *TraceRequestKey) Raw() any {
	return k
}

func (k *TraceRequestKey) String() string {
	return fmt.Sprintf("%s:%s", k.TraceID, k.RunID)
}

type traceReader struct {
	loaders *loaders
	reader  cqrs.TraceReader
}

func (tr *traceReader) GetRunTrace(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	results := make([]*dataloader.Result, len(keys))

	var wg sync.WaitGroup
	for i, key := range keys {
		results[i] = &dataloader.Result{}

		req, ok := key.Raw().(*TraceRequestKey)
		if !ok {
			results[i].Error = fmt.Errorf("unexpected type %T", key.Raw())
			continue
		}

		wg.Add(1)
		go func(ctx context.Context, res *dataloader.Result) {
			defer wg.Done()

			spans, err := tr.reader.GetTraceSpansByRun(ctx, *req.TraceRunIdentifier)
			if err != nil {
				res.Error = fmt.Errorf("error retrieving span: %w", err)
				return
			}
			if len(spans) < 1 {
				return
			}

			// build tree from spans
			tree, err := run.NewRunTree(run.RunTreeOpts{
				AccountID:   req.AccountID,
				WorkspaceID: req.WorkspaceID,
				AppID:       req.AppID,
				FunctionID:  req.FunctionID,
				RunID:       req.RunID,
				Spans:       spans,
			})
			if err != nil {
				res.Error = err
				return
			}

			// convert the tree to a span tree structure for the API
			root, err := tree.ToRunSpan(ctx)
			if err != nil {
				res.Error = fmt.Errorf("error building run tree: %w", err)
				return
			}
			data, err := convertRunTreeToGQLModel(root)
			if err != nil {
				res.Error = fmt.Errorf("error parsing run tree: %w", err)
				return
			}

			res.Data = data
			var primeTree func(context.Context, []*models.RunTraceSpan)
			primeTree = func(ctx context.Context, tspans []*models.RunTraceSpan) {
				for _, span := range tspans {
					if span != nil {
						if span.SpanID != "" {
							tr.loaders.RunSpanLoader.Prime(
								ctx,
								&SpanRequestKey{
									TraceRunIdentifier: req.TraceRunIdentifier,
									SpanID:             span.SpanID,
								},
								span,
							)
						}

						if len(span.ChildrenSpans) > 0 {
							primeTree(ctx, span.ChildrenSpans)
						}
					}
				}
			}

			primeTree(ctx, []*models.RunTraceSpan{data})
		}(ctx, results[i])
	}

	wg.Wait()

	return results
}

func convertRunTreeToGQLModel(pb *rpbv2.RunSpan) (*models.RunTraceSpan, error) {
	// no need to show the function success span, if it's the only one and has no children
	// meaning, there were no function level retries
	if pb.GetName() == consts.OtelExecFnOk && pb.GetStatus() == rpbv2.SpanStatus_COMPLETED && len(pb.GetChildren()) < 1 {
		return nil, ErrSkipSuccess
	}

	var (
		startedAt *time.Time
		endedAt   *time.Time
		stepOp    *models.StepOp
	)

	if pb.GetStartedAt() != nil {
		ts := pb.GetStartedAt().AsTime()
		startedAt = &ts
	}
	if pb.GetEndedAt() != nil {
		ts := pb.GetEndedAt().AsTime()
		endedAt = &ts
	}

	status := models.RunTraceSpanStatusRunning
	switch pb.GetStatus() {
	case rpbv2.SpanStatus_QUEUED, rpbv2.SpanStatus_SCHEDULED:
		status = models.RunTraceSpanStatusQueued
	case rpbv2.SpanStatus_RUNNING:
		status = models.RunTraceSpanStatusRunning
	case rpbv2.SpanStatus_WAITING:
		status = models.RunTraceSpanStatusWaiting
	case rpbv2.SpanStatus_COMPLETED:
		status = models.RunTraceSpanStatusCompleted
	case rpbv2.SpanStatus_CANCELLED:
		status = models.RunTraceSpanStatusCancelled
	case rpbv2.SpanStatus_FAILED:
		status = models.RunTraceSpanStatusFailed
	}

	if pb.StepOp != nil {
		switch *pb.StepOp {
		case rpbv2.SpanStepOp_RUN:
			op := models.StepOpRun
			stepOp = &op
		case rpbv2.SpanStepOp_INVOKE:
			op := models.StepOpInvoke
			stepOp = &op
		case rpbv2.SpanStepOp_SLEEP:
			op := models.StepOpSleep
			stepOp = &op
		case rpbv2.SpanStepOp_WAIT_FOR_EVENT:
			op := models.StepOpWaitForEvent
			stepOp = &op
		}
	}

	attempts := int(pb.GetAttempts())
	duration := int(pb.GetDurationMs())

	span := &models.RunTraceSpan{
		AppID:        uuid.MustParse(pb.GetAppId()),
		FunctionID:   uuid.MustParse(pb.GetFunctionId()),
		TraceID:      pb.GetTraceId(),
		ParentSpanID: pb.ParentSpanId,
		SpanID:       pb.GetSpanId(),
		RunID:        ulid.MustParse(pb.GetRunId()),
		IsRoot:       pb.GetIsRoot(),
		Name:         pb.GetName(),
		Status:       status,
		Attempts:     &attempts,
		Duration:     &duration,
		QueuedAt:     pb.GetQueuedAt().AsTime(),
		StartedAt:    startedAt,
		EndedAt:      endedAt,
		OutputID:     pb.OutputId,
		StepOp:       stepOp,
		StepID:       pb.StepId,
	}

	if pb.GetStepInfo() != nil {
		// step info
		switch v := pb.GetStepInfo().GetInfo().(type) {
		case *rpbv2.StepInfo_Run:
			span.StepInfo = models.RunStepInfo{
				Type: v.Run.Type,
			}
		case *rpbv2.StepInfo_Sleep:
			span.StepInfo = models.SleepStepInfo{
				SleepUntil: v.Sleep.SleepUntil.AsTime(),
			}
		case *rpbv2.StepInfo_Wait:
			wait := v.Wait

			var foundEvtID *ulid.ULID
			if wait.FoundEventId != nil {
				if id, err := ulid.Parse(*wait.FoundEventId); err == nil {
					foundEvtID = &id
				}
			}

			span.StepInfo = models.WaitForEventStepInfo{
				EventName:    wait.EventName,
				Expression:   wait.Expression,
				Timeout:      wait.Timeout.AsTime(),
				FoundEventID: foundEvtID,
				TimedOut:     wait.TimedOut,
			}
		case *rpbv2.StepInfo_Invoke:
			var (
				returnEvtID *ulid.ULID
				runID       *ulid.ULID
			)
			invoke := v.Invoke

			if invoke.ReturnEventId != nil {
				if id, err := ulid.Parse(*invoke.ReturnEventId); err == nil {
					returnEvtID = &id
				}
			}
			if invoke.RunId != nil {
				if id, err := ulid.Parse(*invoke.RunId); err == nil {
					runID = &id
				}
			}

			span.StepInfo = models.InvokeStepInfo{
				TriggeringEventID: ulid.MustParse(invoke.TriggeringEventId),
				FunctionID:        invoke.FunctionId,
				Timeout:           invoke.Timeout.AsTime(),
				ReturnEventID:     returnEvtID,
				RunID:             runID,
				TimedOut:          invoke.TimedOut,
			}
		}
	}

	// iterate over children recursively
	if len(pb.Children) > 0 {
		span.ChildrenSpans = []*models.RunTraceSpan{}

		for _, cp := range pb.Children {
			cspan, err := convertRunTreeToGQLModel(cp)
			if err == ErrSkipSuccess {
				continue
			}
			if err != nil {
				return nil, err
			}
			span.ChildrenSpans = append(span.ChildrenSpans, cspan)
		}
	}

	return span, nil
}

type SpanRequestKey struct {
	*cqrs.TraceRunIdentifier `json:"trident,omitempty"`
	SpanID                   string `json:"sid"`
}

func (k *SpanRequestKey) Raw() any {
	return k
}

func (k *SpanRequestKey) String() string {
	return fmt.Sprintf("%s:%s:%s", k.TraceID, k.RunID, k.SpanID)
}

func (tr *traceReader) GetSpanRun(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	results := make([]*dataloader.Result, len(keys))

	var wg sync.WaitGroup
	for i, key := range keys {
		results[i] = &dataloader.Result{}

		req, ok := key.Raw().(*SpanRequestKey)
		if !ok {
			results[i].Error = fmt.Errorf("unexpected type for span %T", key.Raw())
			continue
		}

		wg.Add(1)
		go func(ctx context.Context, res *dataloader.Result) {
			defer wg.Done()

			// If we're here, we're requested a span ID that wasn't primed by
			// GetRunTrace. Span IDs can sometimes be virtualized based on the
			// entire trace, so here we refetch the entire trace for each key and
			// pick out the spans we need.
			//
			// Because this is calling another loader, duplicate requests will
			// still be filtered out.
			rootSpan, err := LoadOne[models.RunTraceSpan](
				ctx,
				tr.loaders.RunTraceLoader,
				&TraceRequestKey{TraceRunIdentifier: req.TraceRunIdentifier},
			)
			if err != nil {
				res.Error = fmt.Errorf("failed to get run trace: %w", err)
			}

			var findNestedSpan func([]*models.RunTraceSpan) *models.RunTraceSpan
			findNestedSpan = func(spans []*models.RunTraceSpan) *models.RunTraceSpan {
				for _, span := range spans {
					if span == nil {
						continue
					}
					if span.SpanID == req.SpanID {
						return span
					}

					if len(span.ChildrenSpans) > 0 {
						nestedSpan := findNestedSpan(span.ChildrenSpans)
						if nestedSpan != nil {
							return nestedSpan
						}
					}
				}
				return nil
			}

			res.Data = findNestedSpan([]*models.RunTraceSpan{rootSpan})
		}(ctx, results[i])
	}

	wg.Wait()
	return results
}
