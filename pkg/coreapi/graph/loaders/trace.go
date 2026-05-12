package loader

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/graph-gophers/dataloader"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/run"
	"github.com/inngest/inngest/pkg/tracing/meta"
	rpbv2 "github.com/inngest/inngest/proto/gen/run/v2"
	"github.com/oklog/ulid/v2"
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

type DebugRunRequestKey struct {
	DebugRunID ulid.ULID
}

func (k *DebugRunRequestKey) Raw() any {
	return k
}

func (k *DebugRunRequestKey) String() string {
	return k.DebugRunID.String()
}

type DebugSessionRequestKey struct {
	DebugSessionID ulid.ULID
}

func (k *DebugSessionRequestKey) Raw() any {
	return k
}

func (k *DebugSessionRequestKey) String() string {
	return k.DebugSessionID.String()
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

			// Make the run's canonical status available during conversion so
			// descendant resolvers can suppress terminal "function
			// success"/"function error" naming for runs that are still
			// retrying. Prefer function_runs / function_finishes — they
			// record the run-level outcome — and fall back to the root
			// span's DynamicStatus if those rows aren't there yet.
			if rootSpan != nil {
				rootStatus := models.RunTraceSpanStatusRunning
				if fr, ferr := tr.reader.GetRun(ctx, req.RunID, uuid.Nil, uuid.Nil); ferr == nil && fr != nil {
					if mapped, mappedErr := models.ToFunctionRunStatus(fr.Status); mappedErr == nil {
						switch mapped {
						case models.FunctionRunStatusCompleted:
							rootStatus = models.RunTraceSpanStatusCompleted
						case models.FunctionRunStatusFailed:
							rootStatus = models.RunTraceSpanStatusFailed
						case models.FunctionRunStatusCancelled:
							rootStatus = models.RunTraceSpanStatusCancelled
						}
					}
				} else if rootSpan.Attributes != nil && rootSpan.Attributes.DynamicStatus != nil {
					if mapped := tr.stepStatusToGQL(rootSpan.Attributes.DynamicStatus); mapped != nil {
						rootStatus = *mapped
					}
				}
				ctx = withRootRunStatus(ctx, rootStatus)
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
		Status:         status,
		TraceID:        span.GetTraceID(),
		DebugRunID:     debugRunID,
		DebugSessionID: debugSessionID,
		SpanTypeName:   span.Name,
		IsUserland:     isUserland,
		UserlandSpan:   userlandSpan,
	}

	// The run span records its events input via OTel attributes, so the
	// derived `OutputID` is non-nil even before the function produces any
	// output. Hide it until the run has actually ended so consumers don't
	// mistake the input handle for a real output.
	if span.Name == meta.SpanNameRun {
		// Reconcile root status with the canonical run-level outcome.
		if rootStatus, ok := rootRunStatusFromCtx(ctx); ok {
			gqlSpan.Status = rootStatus
			status = rootStatus
		}
		if !models.RunTraceEnded(status) {
			gqlSpan.OutputID = nil
		}
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

				// On invoke timeout the invoked function may still have
				// started (and its run ID recorded); the resolved RunID
				// represents a completed handoff, so clear it when the
				// invoke timed out before the function returned a result.
				if si.TimedOut != nil && *si.TimedOut {
					si.RunID = nil
					si.ReturnEventID = nil
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

				// Promote any visible step children of an omitted discovery
				// to the parent. Some opcodes (e.g. sleep) parent their step
				// span under the previous discovery rather than the run, so
				// without this hoist the step would be lost when the
				// discovery is hidden.
				if child.SpanTypeName == meta.SpanNameStepDiscovery {
					for _, gc := range child.ChildrenSpans {
						if gc == nil || gc.Omit {
							continue
						}
						if gc.SpanTypeName != meta.SpanNameStep {
							continue
						}
						gqlSpan.ChildrenSpans = append(gqlSpan.ChildrenSpans, gc)
					}
				}

				continue
			}

			if cs.MarkedAsDropped {
				continue
			}

			showSpan = true

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
						// The function's terminal output span: name it based
						// on outcome to match the legacy executor names
						// consumers (and tests) check against. While the run
						// is still in progress (a retry pending), we keep
						// the legacy "execute" placeholder name instead, so
						// in-progress traces don't prematurely report a
						// failure. We also only rename when this attempt's
						// status matches the run-level outcome — intermediate
						// failed attempts of a successful run should not show
						// up as "function error".
						rootStatus, hasRoot := rootRunStatusFromCtx(ctx)
						runTerminal := hasRoot && models.RunTraceEnded(rootStatus)
						matchesRun := hasRoot && rootStatus == child.Status
						switch {
						case !runTerminal:
							gqlSpan.Name = consts.OtelExecPlaceholder
							// The wrapping span should reflect the run-level
							// state, not the most recent failed attempt.
							gqlSpan.Status = models.RunTraceSpanStatusRunning
							gqlSpan.EndedAt = nil
							// Suppress the latest attempt's output handle —
							// the placeholder itself hasn't produced output
							// yet from the consumer's perspective.
							gqlSpan.OutputID = nil
						case matchesRun && (child.Status == models.RunTraceSpanStatusFailed ||
							child.Status == models.RunTraceSpanStatusCancelled):
							gqlSpan.Name = consts.OtelExecFnErr
						case matchesRun && child.Status == models.RunTraceSpanStatusCompleted:
							gqlSpan.Name = consts.OtelExecFnOk
						case !hasRoot:
							// Best-effort naming when we can't see the
							// run-level status.
							switch child.Status {
							case models.RunTraceSpanStatusFailed, models.RunTraceSpanStatusCancelled:
								gqlSpan.Name = consts.OtelExecFnErr
							case models.RunTraceSpanStatusCompleted:
								gqlSpan.Name = consts.OtelExecFnOk
							default:
								gqlSpan.Name = FinalizationSpanName
							}
						default:
							// Intermediate function-output attempt from a
							// run that ultimately resolved differently —
							// hide it so the displayed tree only contains
							// the work that contributed to the final
							// outcome (matching v1 behavior).
							showSpan = false
							gqlSpan.Omit = true
							continue
						}
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
			} else if span.Name == meta.SpanNameStepDiscovery && isFunctionLevelExecutionGroup(span) {
				// Function-level retries appear in v2 as a single discovery
				// span with multiple `executor.execution` children (one per
				// attempt). Render the legacy v1 grouping span so consumers
				// see the attempts under a single node.
				switch gqlSpan.Status {
				case models.RunTraceSpanStatusCompleted:
					gqlSpan.Name = consts.OtelExecFnOk
				default:
					gqlSpan.Name = consts.OtelExecPlaceholder
				}
				gqlSpan.StepOp = nil
				gqlSpan.StepInfo = nil
				count := len(gqlSpan.ChildrenSpans)
				gqlSpan.Attempts = &count
				// Legacy consumers expect each attempt's parent to be the
				// run root, mirroring the synthetic v1 structure.
				for _, attempt := range gqlSpan.ChildrenSpans {
					attempt.ParentSpanID = gqlSpan.ParentSpanID
				}
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
		} else if gqlSpan.StartedAt != nil && gqlSpan.EndedAt != nil {
			// Discovery spans don't carry timing attributes of their own;
			// fall back to the merged child timings derived above.
			dur := int(gqlSpan.EndedAt.Sub(*gqlSpan.StartedAt).Milliseconds())
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

type rootRunStatusCtxKey struct{}

func withRootRunStatus(ctx context.Context, status models.RunTraceSpanStatus) context.Context {
	return context.WithValue(ctx, rootRunStatusCtxKey{}, status)
}

func rootRunStatusFromCtx(ctx context.Context) (models.RunTraceSpanStatus, bool) {
	v, ok := ctx.Value(rootRunStatusCtxKey{}).(models.RunTraceSpanStatus)
	return v, ok
}

// isFunctionLevelExecutionGroup reports whether a discovery span is wrapping a
// set of function-level execution attempts (no enclosing step.Run/step.Sleep
// etc). In v1 these attempts were grouped under an "execute" placeholder
// span; v2 emits them as `executor.execution` children of a single discovery,
// so we synthesize the placeholder during GQL conversion to preserve the
// legacy shape consumers expect.
func isFunctionLevelExecutionGroup(span *cqrs.OtelSpan) bool {
	if span.Name != meta.SpanNameStepDiscovery || len(span.Children) < 2 {
		return false
	}
	for _, c := range span.Children {
		if c == nil || c.Name != meta.SpanNameExecution {
			return false
		}
		if c.Attributes == nil {
			continue
		}
		if c.Attributes.StepID != nil && *c.Attributes.StepID != "" {
			return false
		}
		if c.Attributes.StepName != nil && *c.Attributes.StepName != "" {
			return false
		}
		if c.Attributes.StepOp != nil {
			return false
		}
	}
	return true
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
		ts := pb.GetStartedAt().AsTime().Truncate(time.Millisecond)
		startedAt = &ts
	}
	if pb.GetEndedAt() != nil {
		ts := pb.GetEndedAt().AsTime().Truncate(time.Millisecond)
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
		AppID:          uuid.MustParse(pb.GetAppId()),
		FunctionID:     uuid.MustParse(pb.GetFunctionId()),
		TraceID:        pb.GetTraceId(),
		ParentSpanID:   pb.ParentSpanId,
		SpanID:         pb.GetSpanId(),
		RunID:          ulid.MustParse(pb.GetRunId()),
		IsRoot:         pb.GetIsRoot(),
		IsUserland:     pb.GetIsUserland(),
		UserlandSpan:   userlandSpan,
		Name:           pb.GetName(),
		Status:         status,
		Attempts:       &attempts,
		Duration:       &duration,
		QueuedAt:       pb.GetQueuedAt().AsTime().Truncate(time.Millisecond),
		StartedAt:      startedAt,
		EndedAt:        endedAt,
		OutputID:       pb.OutputId,
		StepOp:         stepOp,
		StepID:         pb.StepId,
		DebugRunID:     nil, // Not available in legacy protobuf format
		DebugSessionID: nil, // Not available in legacy protobuf format
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
