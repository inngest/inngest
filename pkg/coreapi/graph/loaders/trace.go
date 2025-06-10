package loader

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/graph-gophers/dataloader"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/run"
	"github.com/inngest/inngest/pkg/tracing/meta"
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

// just run id
func (tr *traceReader) GetRunTrace(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	results := make([]*dataloader.Result, len(keys))
	var wg sync.WaitGroup

	for i, key := range keys {
		results[i] = &dataloader.Result{}

		wg.Add(1)
		go func(ctx context.Context, res *dataloader.Result, key dataloader.Key) {
			defer wg.Done()

			req, ok := key.Raw().(*TraceRequestKey)
			if !ok {
				res.Error = fmt.Errorf("unexpected type %T", key.Raw())
				return
			}

			rootSpan, err := tr.reader.GetSpansByRunID(ctx, req.RunID)
			if err != nil {
				res.Error = fmt.Errorf("error retrieving trace: %w", err)
				return
			}

			spew.Dump(rootSpan)

			gqlRoot, err := tr.convertRunSpanToGQL(ctx, rootSpan)
			if err != nil {
				res.Error = fmt.Errorf("error converting run root to GQL: %w", err)
				return
			}

			res.Data = gqlRoot
			// TODO prime
		}(ctx, results[i], key)
	}

	wg.Wait()

	return results
}

func (tr *traceReader) convertRunSpanToGQL(ctx context.Context, span *cqrs.JackSpan) (*models.RunTraceSpan, error) {
	var duration *int
	status := models.RunTraceSpanStatusRunning
	startedAt := span.GetStartedAtTime()
	endedAt := span.GetEndedAtTime()
	if startedAt != nil && endedAt != nil {
		dur := int(endedAt.Sub(*startedAt).Milliseconds())
		duration = &dur
		status = models.RunTraceSpanStatusCompleted // TODO the actual statuses
	}

	gqlSpan := &models.RunTraceSpan{
		AppID:        span.GetAppID(),
		FunctionID:   span.GetFunctionID(),
		RunID:        span.GetRunID(),
		SpanID:       span.GetSpanID(),
		TraceID:      span.GetTraceID(),
		Name:         span.GetName(),
		Status:       status,
		Attempts:     span.GetAttempts(),
		ParentSpanID: span.GetParentSpanID(),
		IsRoot:       span.GetIsRoot(),

		Duration: duration,
		// OutputID: , TODO

		QueuedAt:  span.GetQueuedAtTime(),
		StartedAt: span.GetStartedAtTime(),
		EndedAt:   span.GetEndedAtTime(),

		// StepOp: ,
		// StepID: ,
		// StepInfo: ,

		// IsUserland: ,
		// UserlandSpan: ,
	}

	if len(span.Children) > 0 {
		gqlSpan.ChildrenSpans = []*models.RunTraceSpan{}

		for i, cs := range span.Children {
			child, err := tr.convertRunSpanToGQL(ctx, cs)
			if err != nil {
				return nil, fmt.Errorf("error converting child span: %w", err)
			}

			// Decide on changes to this parent span based on the children.
			switch span.Name {
			case meta.SpanNameStepDiscovery:
				{
					gqlSpan.Status = child.Status
					gqlSpan.StartedAt = child.StartedAt // TODO only first
					gqlSpan.EndedAt = child.EndedAt
					if child.Name != "" {
						gqlSpan.Name = child.Name
						child.Name = fmt.Sprintf("Attempt %d", i+1)
					}
					break
				}

			case meta.SpanNameStep:
				{
					// TODO multiple executions?
					gqlSpan.Status = child.Status
					gqlSpan.StartedAt = child.StartedAt // TODO only first
					gqlSpan.EndedAt = child.EndedAt
					if child.Name != "" {
						gqlSpan.Name = child.Name // TODO only first
					}
					break
				}
			}

			gqlSpan.ChildrenSpans = append(gqlSpan.ChildrenSpans, child)
		}

		// For the run span, the start is the first child span's start
		if span.Name == meta.SpanNameRun {
			gqlSpan.StartedAt = &gqlSpan.ChildrenSpans[0].QueuedAt
			if gqlSpan.EndedAt != nil {
				dur := int(gqlSpan.EndedAt.Sub(*gqlSpan.StartedAt).Milliseconds())
				gqlSpan.Duration = &dur
				gqlSpan.Status = models.RunTraceSpanStatusCompleted // TODO the actual statuses
			}
		}
	}

	return gqlSpan, nil
}

func (tr *traceReader) GetLegacyRunTrace(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
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

func convertUserlandSpan(pb *rpbv2.UserlandSpan) *models.UserlandSpan {
	if pb == nil {
		return nil
	}
	spanAttrs := string(pb.SpanAttrs)
	resourceAttrs := string(pb.ResourceAttrs)

	return &models.UserlandSpan{
		SpanName:      &pb.SpanName,
		SpanKind:      &pb.SpanKind,
		ServiceName:   pb.ServiceName,
		ScopeName:     pb.ScopeName,
		ScopeVersion:  pb.ScopeVersion,
		SpanAttrs:     &spanAttrs,
		ResourceAttrs: &resourceAttrs,
	}
}

func convertRunTreeToGQLModel(pb *rpbv2.RunSpan) (*models.RunTraceSpan, error) {
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
		case rpbv2.SpanStepOp_AI_GATEWAY:
			op := models.StepOpAiGateway
			stepOp = &op
		case rpbv2.SpanStepOp_WAIT_FOR_SIGNAL:
			op := models.StepOpWaitForSignal
			stepOp = &op
		}
	}

	attempts := int(pb.GetAttempts())
	duration := int(pb.GetDurationMs())

	var userlandSpan *models.UserlandSpan
	if pb.GetIsUserland() {
		userlandSpan = convertUserlandSpan(pb.GetUserlandSpan())
	}

	span := &models.RunTraceSpan{
		AppID:        uuid.MustParse(pb.GetAppId()),
		FunctionID:   uuid.MustParse(pb.GetFunctionId()),
		TraceID:      pb.GetTraceId(),
		ParentSpanID: pb.ParentSpanId,
		SpanID:       pb.GetSpanId(),
		RunID:        ulid.MustParse(pb.GetRunId()),
		IsRoot:       pb.GetIsRoot(),
		IsUserland:   pb.GetIsUserland(),
		UserlandSpan: userlandSpan,
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
		case *rpbv2.StepInfo_WaitForSignal:
			wait := v.WaitForSignal

			span.StepInfo = models.WaitForSignalStepInfo{
				Signal:   wait.Signal,
				Timeout:  wait.Timeout.AsTime(),
				TimedOut: wait.TimedOut,
			}
		}
	}

	// iterate over children recursively
	if len(pb.Children) > 0 {
		span.ChildrenSpans = []*models.RunTraceSpan{}

		for _, cp := range pb.Children {
			cspan, err := convertRunTreeToGQLModel(cp)

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

func (tr *traceReader) GetLegacySpanRun(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
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
				tr.loaders.LegacyRunTraceLoader,
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
