package loader

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/graph-gophers/dataloader"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/meta"
)

const (
	RunSpanName              = "Run"
	UnknownStepSpanName      = "Unknown step"
	DiscoveryStepSpanName    = "Discovery step"
	GenericExecutionSpanName = "Execution"
	FinalizationSpanName     = "Finalization"

	// SDKExecutionSpanName is an alias for meta.SDKExecutionSpanName
	// used locally for readability.
	SDKExecutionSpanName = meta.SDKExecutionSpanName
)

var ErrSkipSuccess = fmt.Errorf("skip success span")

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
	loaders *Loaders
	reader  cqrs.TraceReader
	// convert renders a *cqrs.OtelSpan tree as GQL. Defaults to
	// convertRunSpanToGQL (the rollup-aware converter) when nil, so every
	// existing construction site (debug.go's, public.go's) that doesn't
	// set this field keeps its current behavior unchanged. NewLoaders sets
	// it to convertFlatSpanToGQL when params.DB is a flatSpanSource.
	convert func(ctx context.Context, span *cqrs.OtelSpan) (*models.RunTraceSpan, error)
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

			convert := tr.convert
			if convert == nil {
				convert = tr.convertRunSpanToGQL
			}
			gqlRoot, err := convert(ctx, rootSpan)
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

func (tr *traceReader) opcodeToGQL(op *enums.Opcode) *models.StepOp {
	if op == nil {
		return nil
	}

	switch *op {
	case enums.OpcodeStepRun, enums.OpcodeStepError, enums.OpcodeStepPlanned:
		op := models.StepOpRun
		return &op
	case enums.OpcodeAIGateway, enums.OpcodeGateway: // TODO gateway separate
		op := models.StepOpAiGateway
		return &op
	case enums.OpcodeInvokeFunction:
		op := models.StepOpInvoke
		return &op
	case enums.OpcodeSleep:
		op := models.StepOpSleep
		return &op
	case enums.OpcodeWaitForEvent:
		op := models.StepOpWaitForEvent
		return &op
	case enums.OpcodeWaitForSignal:
		op := models.StepOpWaitForSignal
		return &op
	}

	return nil
}

func (tr *traceReader) stepStatusToGQL(status *enums.StepStatus) *models.RunTraceSpanStatus {
	if status == nil {
		return nil
	}

	switch *status {
	case enums.StepStatusRunning, enums.StepStatusInvoking:
		s := models.RunTraceSpanStatusRunning
		return &s
	case enums.StepStatusCompleted, enums.StepStatusTimedOut:
		s := models.RunTraceSpanStatusCompleted
		return &s
	case enums.StepStatusFailed, enums.StepStatusErrored:
		s := models.RunTraceSpanStatusFailed
		return &s
	case enums.StepStatusCancelled:
		s := models.RunTraceSpanStatusCancelled
		return &s
	case enums.StepStatusScheduled, enums.StepStatusQueued:
		s := models.RunTraceSpanStatusQueued
		return &s
	case enums.StepStatusSleeping, enums.StepStatusWaiting:
		s := models.RunTraceSpanStatusWaiting
		return &s
	case enums.StepStatusSkipped:
		s := models.RunTraceSpanStatusSkipped
		return &s
	}

	return nil
}

func (tr *traceReader) convertRunSpanToGQL(ctx context.Context, span *cqrs.OtelSpan) (*models.RunTraceSpan, error) {
	status := models.RunTraceSpanStatusRunning

	// Make sure we parse dynamic statuses from updates
	if span.Attributes.DynamicStatus != nil {
		if gqlStatus := tr.stepStatusToGQL(span.Attributes.DynamicStatus); gqlStatus != nil {
			status = *gqlStatus
		}
	}

	attempts := span.GetAttempts()

	debugRunID := span.GetDebugRunID()
	debugSessionID := span.GetDebugSessionID()

	isUserland := false
	var userlandSpan *models.UserlandSpan

	if span.Attributes.IsUserland != nil && *span.Attributes.IsUserland {
		isUserland = true

		filteredAttrs := make(map[string]any)
		for k, v := range span.RawOtelSpan.Attributes {
			if !strings.HasPrefix(k, meta.AttrKeyPrefix) {
				filteredAttrs[k] = v
			}
		}

		filteredAttrsByt, err := json.Marshal(filteredAttrs)
		if err != nil {
			return nil, fmt.Errorf("error marshalling filtered attributes: %w", err)
		}

		filteredAttrsStr := string(filteredAttrsByt)

		userlandSpan = &models.UserlandSpan{
			SpanName:     span.Attributes.UserlandName,
			SpanKind:     span.Attributes.UserlandKind,
			ScopeName:    span.Attributes.UserlandScopeName,
			ScopeVersion: span.Attributes.UserlandScopeVersion,
			ServiceName:  span.Attributes.UserlandServiceName,
			SpanAttrs:    &filteredAttrsStr,
		}

	}

	name := span.GetStepName()
	if isUserland {
		name = *userlandSpan.SpanName
	}

	gqlSpan := &models.RunTraceSpan{
		AppID:          span.GetAppID(),
		Attempts:       &attempts,
		GroupID:        span.Attributes.GroupID,
		EndedAt:        span.GetEndedAtTime(),
		FunctionID:     span.GetFunctionID(),
		IsRoot:         span.GetIsRoot(),
		Name:           name,
		OutputID:       span.GetOutputID(),
		ParentSpanID:   span.GetParentSpanID(),
		QueuedAt:       span.GetQueuedAtTime(),
		RunID:          span.GetRunID(),
		SpanID:         span.GetSpanID(),
		StartedAt:      span.GetStartedAtTime(),
		ScheduledAt:    span.GetScheduledAtTime(),
		Status:         status,
		TraceID:        span.GetTraceID(),
		DebugRunID:     debugRunID,
		DebugSessionID: debugSessionID,
		SpanTypeName:   span.Name,
		IsUserland:     isUserland,
		UserlandSpan:   userlandSpan,
	}

	if span.Attributes.SkipReason != nil {
		reason := span.Attributes.SkipReason.String()
		gqlSpan.SkipReason = &reason
	}
	if span.Attributes.SkipExistingRunID != nil {
		gqlSpan.SkipExistingRunID = span.Attributes.SkipExistingRunID
	}

	if span.Attributes.ResponseStatusCode != nil && span.Attributes.ResponseHeaders != nil {
		gqlSpan.Response = &models.RunTraceSpanResponseInfo{
			StatusCode: *span.Attributes.ResponseStatusCode,
			Headers:    *span.Attributes.ResponseHeaders,
		}
	}

	// If this was a discovery span, we may not want to show it.
	showSpan := span.Name != meta.SpanNameStepDiscovery

	if span.Attributes.StepOp != nil {
		gqlSpan.StepOp = tr.opcodeToGQL(span.Attributes.StepOp)
	}

	if span.Attributes.StepID != nil {
		gqlSpan.StepID = span.Attributes.StepID
	}

	if gqlSpan.StepOp != nil {
		switch *gqlSpan.StepOp {
		case models.StepOpRun:
			{
				gqlSpan.StepInfo = &models.RunStepInfo{
					Type: span.Attributes.StepRunType,
				}
			}
		case models.StepOpInvoke:
			{
				si := &models.InvokeStepInfo{
					TimedOut:      span.Attributes.StepWaitExpired,
					ReturnEventID: span.Attributes.StepInvokeFinishEventID,
					RunID:         span.Attributes.StepInvokeRunID,
				}

				if span.Attributes.StepInvokeTriggerEventID != nil {
					si.TriggeringEventID = *span.Attributes.StepInvokeTriggerEventID
				}

				if span.Attributes.StepInvokeFunctionID != nil {
					si.FunctionID = *span.Attributes.StepInvokeFunctionID
				}

				if span.Attributes.StepWaitExpiry != nil {
					si.Timeout = *span.Attributes.StepWaitExpiry
				}

				gqlSpan.StepInfo = si
			}
		case models.StepOpSleep:
			{
				if span.Attributes.StepSleepDuration != nil {
					gqlSpan.StepInfo = &models.SleepStepInfo{
						SleepUntil: span.GetQueuedAtTime().Add(*span.Attributes.StepSleepDuration),
					}
				}
			}
		case models.StepOpWaitForEvent:
			{
				si := &models.WaitForEventStepInfo{
					Expression:   span.Attributes.StepWaitForEventIf,
					TimedOut:     span.Attributes.StepWaitExpired,
					FoundEventID: span.Attributes.StepWaitForEventMatchedID,
				}

				if span.Attributes.StepWaitForEventName != nil {
					si.EventName = *span.Attributes.StepWaitForEventName
				}

				if span.Attributes.StepWaitExpiry != nil {
					si.Timeout = *span.Attributes.StepWaitExpiry
				}

				gqlSpan.StepInfo = si
			}
		case models.StepOpWaitForSignal:
			{
				si := &models.WaitForSignalStepInfo{
					TimedOut: span.Attributes.StepWaitExpired,
				}

				if span.Attributes.StepSignalName != nil {
					si.Signal = *span.Attributes.StepSignalName
				}

				if span.Attributes.StepWaitExpiry != nil {
					si.Timeout = *span.Attributes.StepWaitExpiry
				}

				gqlSpan.StepInfo = si
			}
		}
	}

	hasFinalizationChild := false

	if len(span.Children) > 0 {
		gqlSpan.ChildrenSpans = []*models.RunTraceSpan{}
		lastStepQueueTime := &gqlSpan.QueuedAt
		isFirstChild := true
		var omittedStepMetadata []*models.SpanMetadata
		haveSetRunStartTime := span.Name != meta.SpanNameRun

		// If there's a run start time on the overall parent, use that.  Sometimes this
		// is the case for eg. sync based runs.
		if span.GetStartedAtTime() != nil {
			haveSetRunStartTime = true
		}

		for i, cs := range span.Children {
			child, err := tr.convertRunSpanToGQL(ctx, cs)
			if err != nil {
				return nil, fmt.Errorf("error converting child span: %w", err)
			}

			// We could also not have a child, for example if we're
			// intentionally skipping it
			if child == nil {
				continue
			}

			if child.Omit {
				// We're skipping this child, but we may still want to use
				// its data for timings.
				if child.SpanTypeName == meta.SpanNameStepDiscovery && !haveSetRunStartTime {
					// Discovery spans can be used to set the start time of
					// the step if it's the first child.
					gqlSpan.StartedAt = child.StartedAt
					haveSetRunStartTime = true
				}

				// Preserve metadata from omitted step discovery spans so
				// it can be transferred to the next visible step sibling.
				// The execution span (which holds timing metadata) is
				// parented to the step discovery span, not the step span.
				// When the discovery span is omitted, its metadata must
				// be promoted to the corresponding visible step span.
				if len(child.Metadata) > 0 && child.SpanTypeName == meta.SpanNameStepDiscovery {
					omittedStepMetadata = append(omittedStepMetadata, child.Metadata...)
				}

				continue
			}

			if !cs.MarkedAsDropped {
				showSpan = true
			}

			// Transfer any accumulated metadata from preceding omitted
			// step discovery spans to this visible step sibling. Each
			// discovery span precedes its corresponding step span in the
			// child list, so we attach metadata to the next visible step
			// we encounter rather than collecting everything for a
			// post-loop pass.
			if len(omittedStepMetadata) > 0 && child.SpanTypeName == meta.SpanNameStep {
				child.Metadata = append(child.Metadata, omittedStepMetadata...)
				omittedStepMetadata = nil
			}

			// Decide on changes to this parent span based on the children.
			switch span.Name {
			case meta.SpanNameRun:
				{
					// Only one step-level finalization span is shown.
					if child.Name == FinalizationSpanName {
						if hasFinalizationChild {
							continue
						}

						hasFinalizationChild = true
					}
				}
			case meta.SpanNameStepDiscovery, meta.SpanNameStep:
				{
					// Userland spans don't carry step execution metadata;
					// so skip all parent-property propagation for them.
					if child.IsUserland {
						break
					}

					gqlSpan.EndedAt = child.EndedAt
					gqlSpan.Status = child.Status

					if isFirstChild {
						isFirstChild = false
						gqlSpan.StartedAt = child.StartedAt
					}

					if child.OutputID != nil && *child.OutputID != "" {
						gqlSpan.OutputID = child.OutputID
					}

					if cs.Attributes.IsFunctionOutput != nil && *cs.Attributes.IsFunctionOutput {
						gqlSpan.Name = FinalizationSpanName
					} else if strings.HasPrefix(gqlSpan.Name, "executor.") && child.Name != "" {
						gqlSpan.Name = child.Name
					}
					child.Name = fmt.Sprintf("Attempt %d", i)
					if child.StepOp != nil {
						gqlSpan.StepOp = child.StepOp
					}
					if child.StepID != nil && *child.StepID != "" {
						gqlSpan.StepID = child.StepID
					}
					if child.StepInfo != nil {
						gqlSpan.StepInfo = child.StepInfo
					}
					if child.StepType != "" {
						gqlSpan.StepType = child.StepType
					}
					if child.Attempts != nil && *child.Attempts > *gqlSpan.Attempts {
						gqlSpan.Attempts = child.Attempts
					}

					// Executions should have queue times related to their
					// siblings
					if lastStepQueueTime != nil {
						child.QueuedAt = *lastStepQueueTime
					}
					if child.EndedAt != nil {
						lastStepQueueTime = child.EndedAt
					}
				}
			}

			gqlSpan.ChildrenSpans = append(gqlSpan.ChildrenSpans, child)
		}

		// A discovery-derived finalization group aggregates the run's terminal
		// attempts, but the loose executor.nonstep siblings (emitted once per
		// attempt, parented to the run root) carry the same attempts and
		// output. Clear StepID and OutputID so this group matches the cloud
		// renderer's shape and clients render finalization from the nonstep
		// spans instead of showing the same work twice; the per-attempt
		// children keep their own output IDs.
		if gqlSpan.Name == FinalizationSpanName &&
			(span.Name == meta.SpanNameStepDiscovery || span.Name == meta.SpanNameStep) {
			gqlSpan.StepID = nil
			gqlSpan.OutputID = nil
		}

		// If we only have a single child, this span isn't a userland span,
		// but the single child is the SDK's `"inngest.execution"` wrapper,
		// collapse it by returning its children (if any).
		//
		// We do this because userland spans are always underneath an
		// `"inngest.execution"` span created by an SDK, which houses useful
		// information about the environment, versions, scope, etc.
		//
		// Critically, this means we also ignore the `"inngest.execution"`
		// span itself, as we never want to display it to the user.
		//
		// We only collapse when the child is specifically the SDK execution
		// wrapper span. Other userland spans with children (e.g., spans
		// within checkpointed steps) must be preserved in the tree.
		if !gqlSpan.IsUserland && len(gqlSpan.ChildrenSpans) == 1 && gqlSpan.ChildrenSpans[0].IsUserland && gqlSpan.ChildrenSpans[0].Name == SDKExecutionSpanName {
			gqlSpan.ChildrenSpans = gqlSpan.ChildrenSpans[0].ChildrenSpans
		}

		// For the run span, the start is the first child span's start
		if span.Name == meta.SpanNameRun && len(gqlSpan.ChildrenSpans) > 0 {
			if (gqlSpan.StartedAt == nil || !haveSetRunStartTime) && gqlSpan.ChildrenSpans[0].StartedAt != nil {
				gqlSpan.StartedAt = gqlSpan.ChildrenSpans[0].StartedAt
			}

			if gqlSpan.EndedAt != nil && gqlSpan.StartedAt != nil {
				dur := int(gqlSpan.EndedAt.Sub(*gqlSpan.StartedAt).Milliseconds())
				gqlSpan.Duration = &dur
			}
		}

		isStep := span.Name == meta.SpanNameStep || span.Name == meta.SpanNameStepDiscovery
		if isStep {
			// Step spans should not show attempts if they only have one and
			// have resolved
			if len(gqlSpan.ChildrenSpans) == 1 && !gqlSpan.ChildrenSpans[0].IsUserland && gqlSpan.ChildrenSpans[0].Status == models.RunTraceSpanStatusCompleted {
				gqlSpan.Response = gqlSpan.ChildrenSpans[0].Response
				gqlSpan.Metadata = append(gqlSpan.Metadata, gqlSpan.ChildrenSpans[0].Metadata...)
				// However, we preserve any userland spans from the
				// successful execution if we have any.
				gqlSpan.ChildrenSpans = gqlSpan.ChildrenSpans[0].ChildrenSpans
			}
		}

		// Give spans some more meaningful names if somehow we don't have the
		// correct information. This shouldn't be possible, but is a final
		// pass to ensure we filter out internal-looking span names.
		switch gqlSpan.Name {
		case meta.SpanNameRun:
			{
				gqlSpan.Name = RunSpanName
			}
		case meta.SpanNameStep:
			{
				gqlSpan.Name = UnknownStepSpanName
			}
		case meta.SpanNameStepDiscovery:
			{
				gqlSpan.Name = DiscoveryStepSpanName
			}
		case meta.SpanNameExecution:
			{
				gqlSpan.Name = GenericExecutionSpanName
			}
		}

		// Any remaining omittedStepMetadata at this point means
		// there were trailing omitted discovery spans with no
		// subsequent visible step child — intentionally discarded.
	}

	if !showSpan {
		gqlSpan.Omit = true
	}

	if gqlSpan.Name == FinalizationSpanName {
		gqlSpan.StepType = strings.ToUpper(FinalizationSpanName)
	} else if span.Attributes.StepRunType != nil {
		gqlSpan.StepType = *span.Attributes.StepRunType
	} else if gqlSpan.StepOp != nil {
		gqlSpan.StepType = gqlSpan.StepOp.String()
	}

	if models.RunTraceEnded(gqlSpan.Status) || gqlSpan.IsUserland {
		startedAt := span.GetStartedAtTime()
		endedAt := span.GetEndedAtTime()
		if startedAt != nil && endedAt != nil {
			dur := int(endedAt.Sub(*startedAt).Milliseconds())
			gqlSpan.Duration = &dur
		}
	} else {
		// Remove ended at.  There's an issue in the data that CQRS is passed in which
		// sometimes all spans have an EndedAt field, which actually denotes when the
		// span was committed.
		//
		// EndedAt, to GQL, denotes the step ending, and we merge start and stop spans
		// together.
		gqlSpan.EndedAt = nil
	}

	for _, md := range span.Metadata {
		gqlSpan.Metadata = append(gqlSpan.Metadata, &models.SpanMetadata{
			Kind:      md.Kind,
			Scope:     md.Scope,
			Values:    md.Values,
			UpdatedAt: md.UpdatedAt,
		})
	}

	return gqlSpan, nil
}

