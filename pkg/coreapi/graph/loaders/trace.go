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

	// The executor records every opcode an SDK response returned on the span
	// that handled that response. More than one means the SDK planned them
	// together, which is the only authoritative statement of a parallel batch
	// available anywhere: step spans are all parented flat to the run span, so
	// clients otherwise have to infer fan-out from execution overlap.
	if steps := span.Attributes.ResponseSteps; steps != nil && len(*steps) > 0 {
		planned := make([]*models.RunStep, 0, len(*steps))
		for _, op := range *steps {
			step := &models.RunStep{
				StepID: op.ID,
				Name:   op.Name,
			}
			if stepOp := tr.opcodeToGQL(&op.Op); stepOp != nil {
				step.StepOp = stepOp
			}
			planned = append(planned, step)
		}
		gqlSpan.PlannedSteps = planned
	}

	// The SDK disambiguates repeated step IDs with an auto-incremented index
	// (step.run("work") twice becomes work:1 and work:2). Both the unhashed ID
	// and that index are recorded on the span; neither was exposed, which is why
	// two such steps are indistinguishable in the UI.
	gqlSpan.UserlandStepID = span.Attributes.StepUserlandID
	gqlSpan.UserlandStepIndex = span.Attributes.StepUserlandIndex
	if span.Attributes.StepParentIDs != nil {
		gqlSpan.ParentStepIDs = *span.Attributes.StepParentIDs
	}
	if span.Attributes.StepParentAlternateIDs != nil {
		gqlSpan.ParentAlternateStepIDs = *span.Attributes.StepParentAlternateIDs
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
		// Plans read off omitted discovery spans, keyed by each step the plan
		// named, so they can be promoted onto the visible step spans below.
		plansByStepID := map[string][]*models.RunStep{}
		// The step a step continues is a property of the STEP, not of any one
		// span: the executor stamps it on the span created when the step was
		// planned, and a later span records the same step completing. Rollup
		// keeps the latter, so collect the value across every span of the run
		// and stamp it on all of them.
		parentByStepID := map[string][]string{}
		altsByStepID := map[string][]string{}
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

			// Collect before any skipping: the executor stamps the parent on
			// the span created when the step was *planned*, and a later span
			// records the same step *completing*. Clients roll spans up by step
			// and keep the latter, so the value has to be gathered from every
			// span and re-stamped below.
			if child.StepID != nil {
				if len(child.ParentStepIDs) > 0 {
					parentByStepID[*child.StepID] = child.ParentStepIDs
				}
				if len(child.ParentAlternateStepIDs) > 0 {
					altsByStepID[*child.StepID] = child.ParentAlternateStepIDs
				}
			}

			// A discovery span is where the run changed shape. The tree drops
			// them, since it is a tree of steps, so gather them onto the run
			// itself before that happens — and carry up any a descendant found.
			gqlSpan.Discoveries = append(gqlSpan.Discoveries, child.Discoveries...)
			child.Discoveries = nil

			if child.SpanTypeName == meta.SpanNameStepDiscovery {
				gqlSpan.Discoveries = append(gqlSpan.Discoveries, discoveryOf(child))
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

				// A discovery span's subtree is the only place that records
				// which steps one SDK response planned together, and the whole
				// subtree is omitted from the tree. The attribute sits on the
				// execution span *under* the discovery span, so search the
				// subtree rather than just the child. Promote the plan onto the
				// step spans it named, so clients can tell a real parallel
				// batch from steps that merely happened to overlap.
				if child.SpanTypeName == meta.SpanNameStepDiscovery {
					for _, plan := range collectPlannedSteps(child) {
						if len(plan) < 2 {
							continue
						}
						for _, planned := range plan {
							plansByStepID[planned.StepID] = plan
						}
					}
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
				// The planned-step list lives on the execution span we are
				// about to discard, so lift it the same way Response and
				// Metadata are lifted.
				if len(gqlSpan.PlannedSteps) == 0 {
					gqlSpan.PlannedSteps = gqlSpan.ChildrenSpans[0].PlannedSteps
				}
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

		// Promote the plans collected from omitted discovery spans onto the
		// visible step spans they named. Done after the loop because a plan
		// covers several siblings, not just the next one.
		if len(plansByStepID) > 0 || len(parentByStepID) > 0 {
			for _, child := range gqlSpan.ChildrenSpans {
				if child.StepID == nil {
					continue
				}
				if plan, ok := plansByStepID[*child.StepID]; ok {
					child.PlannedSteps = plan
				}
				if parents, ok := parentByStepID[*child.StepID]; ok && len(child.ParentStepIDs) == 0 {
					child.ParentStepIDs = parents
				}
				if alts, ok := altsByStepID[*child.StepID]; ok && len(child.ParentAlternateStepIDs) == 0 {
					child.ParentAlternateStepIDs = alts
				}
			}
		}
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


// collectPlannedSteps gathers every planned-step list recorded anywhere in a
// span subtree. The executor stamps `response.step.ops` on the execution span
// that handled an SDK response, which for a discovery request sits one level
// below the discovery span — and the whole discovery subtree is omitted from
// the tree we return, so the lists have to be lifted out before it is dropped.
// discoveryOf summarises a discovery span for clients. The steps it planned sit
// on the execution span *under* it, not on the discovery span itself, so they
// are gathered from the whole subtree.
func discoveryOf(span *models.RunTraceSpan) *models.RunDiscovery {
	planned := []string{}
	for _, plan := range collectPlannedSteps(span) {
		for _, step := range plan {
			planned = append(planned, step.StepID)
		}
	}

	return &models.RunDiscovery{
		SpanID:         span.SpanID,
		Status:         span.Status,
		QueuedAt:       span.QueuedAt,
		StartedAt:      span.StartedAt,
		EndedAt:        span.EndedAt,
		PlannedStepIDs: planned,
	}
}

func collectPlannedSteps(span *models.RunTraceSpan) [][]*models.RunStep {
	if span == nil {
		return nil
	}

	var plans [][]*models.RunStep
	if len(span.PlannedSteps) > 0 {
		plans = append(plans, span.PlannedSteps)
	}
	for _, child := range span.ChildrenSpans {
		plans = append(plans, collectPlannedSteps(child)...)
	}
	return plans
}
