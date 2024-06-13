package loader

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/graph-gophers/dataloader"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
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

			// TODO: build tree from spans
			tb, err := NewTraceTreeBuilder(TraceTreeBuilderOpts{
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

			tree, err := tb.Build(ctx)
			if err != nil {
				res.Error = fmt.Errorf("error run details: %w", err)
				return
			}

			res.Data = tree
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

						if span.ChildrenSpans != nil && len(span.ChildrenSpans) > 0 {
							primeTree(ctx, span.ChildrenSpans)
						}
					}
				}
			}

			primeTree(ctx, []*models.RunTraceSpan{tree})
		}(ctx, results[i])
	}

	wg.Wait()

	return results
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

// TraceTreeBuilder builds the span tree used for the API
type TraceTreeBuilder struct {
	// root is the root span of the tree
	root *cqrs.Span
	// trigger is the trigger span of the tree
	trigger *cqrs.Span

	// spans is used for quick access seen spans,
	// and also assigning children span to them.
	// - key: spanID
	spans map[string]*cqrs.Span
	// groups is used for grouping spans together
	// - key: groupID
	groups map[string][]*cqrs.Span

	// processed is used for tracking already processed spans
	// this helps with skipping work for spans that already have been processed
	// because they were in a group
	processed map[string]bool

	// identifiers
	accID uuid.UUID
	wsID  uuid.UUID
	appID uuid.UUID
	fnID  uuid.UUID
	runID ulid.ULID
}

type TraceTreeBuilderOpts struct {
	AccountID   uuid.UUID
	WorkspaceID uuid.UUID
	AppID       uuid.UUID
	FunctionID  uuid.UUID
	RunID       ulid.ULID
	Spans       []*cqrs.Span
}

type SpanConverter func(*cqrs.Span)

func NewTraceTreeBuilder(opts TraceTreeBuilderOpts) (*TraceTreeBuilder, error) {
	ttb := &TraceTreeBuilder{
		accID:     opts.AccountID,
		wsID:      opts.WorkspaceID,
		appID:     opts.AppID,
		fnID:      opts.FunctionID,
		runID:     opts.RunID,
		spans:     map[string]*cqrs.Span{},
		groups:    map[string][]*cqrs.Span{},
		processed: map[string]bool{},
	}

	for _, s := range opts.Spans {
		if s.ScopeName == consts.OtelScopeFunction {
			ttb.root = s
		}

		if s.ScopeName == consts.OtelScopeTrigger {
			ttb.trigger = s
		}

		ttb.spans[s.SpanID] = s
		if s.ParentSpanID != nil {
			if groupID := s.GroupID(); groupID != nil {
				if _, exists := ttb.groups[*groupID]; !exists {
					ttb.groups[*groupID] = []*cqrs.Span{}
				}
				ttb.groups[*groupID] = append(ttb.groups[*groupID], s)
			}
		}
	}

	// loop through again to construct parent/child relationship
	for _, s := range opts.Spans {
		if s.ParentSpanID != nil {
			if parent, ok := ttb.spans[*s.ParentSpanID]; ok {
				if parent.Children == nil {
					parent.Children = []*cqrs.Span{}
				}
				parent.Children = append(parent.Children, s)
			}
		}
	}

	if ttb.root == nil {
		return nil, fmt.Errorf("no function run span found")
	}
	if ttb.trigger == nil {
		return nil, fmt.Errorf("no trigger span found")
	}

	return ttb, nil
}

// Build goes through the tree and construct the trace for API to be consumed
func (tb *TraceTreeBuilder) Build(ctx context.Context) (*models.RunTraceSpan, error) {
	root, _, err := tb.toRunTraceSpan(ctx, tb.root)
	if err != nil {
		return nil, fmt.Errorf("error converting function span: %w", err)
	}
	root.IsRoot = true

	// sort it in asc order before proceeding
	spans := tb.root.Children
	sort.Slice(spans, func(i, j int) bool {
		return spans[i].Timestamp.UnixMilli() < spans[j].Timestamp.UnixMilli()
	})

	// these are the execution or steps for the function run
	for _, span := range spans {
		tspan, skipped, err := tb.toRunTraceSpan(ctx, span)
		if err != nil {
			return nil, fmt.Errorf("error converting execution span: %w", err)
		}
		// means this span was already processed so no-op here
		if skipped {
			continue
		}
		root.ChildrenSpans = append(root.ChildrenSpans, tspan)
	}

	return root, nil
}

func (tb *TraceTreeBuilder) toRunTraceSpan(ctx context.Context, s *cqrs.Span) (*models.RunTraceSpan, bool, error) {
	// already processed skip it
	if _, ok := tb.processed[s.SpanID]; ok {
		return nil, true, nil
	}
	// NOTE: is this check sufficient?
	if s.SpanName == "function success" {
		return nil, true, nil
	}

	var (
		appID         uuid.UUID
		fnID          uuid.UUID
		runID         ulid.ULID
		defaulAttempt int
	)

	// TODO:
	// - check for group
	// - if there are multiple entries, construct a grouping with all the spans in the group
	// - mark the spans as converted so they don't get processed again

	name := s.SpanName
	if s.StepDisplayName() != nil {
		name = *s.StepDisplayName()
	}

	if s.RunID != nil {
		runID = *s.RunID
	}
	if id := s.AppID(); id != nil {
		appID = *id
	}
	if id := s.FunctionID(); id != nil {
		fnID = *id
	}

	dur := s.DurationMS()
	endedAt := s.Timestamp.Add(s.Duration)

	res := models.RunTraceSpan{
		AppID:        appID,
		FunctionID:   fnID,
		RunID:        runID,
		TraceID:      s.TraceID,
		ParentSpanID: s.ParentSpanID,
		SpanID:       s.SpanID,
		Name:         name,
		Status:       models.RunTraceSpanStatusRunning,
		QueuedAt:     ulid.Time(runID.Time()),
		StartedAt:    &s.Timestamp,
		EndedAt:      &endedAt,
		Duration:     &dur,
		Attempts:     &defaulAttempt,
	}

	// TODO: assign step status
	if s.ScopeName == consts.OtelScopeFunction {
		switch s.FunctionStatus() {
		case enums.RunStatusRunning:
			res.Status = models.RunTraceSpanStatusRunning
		case enums.RunStatusCompleted:
			res.Status = models.RunTraceSpanStatusCompleted
		case enums.RunStatusCancelled:
			res.Status = models.RunTraceSpanStatusCancelled
		case enums.RunStatusFailed, enums.RunStatusOverflowed:
			res.Status = models.RunTraceSpanStatusFailed
		default:
			return nil, false, fmt.Errorf("unexpected run status: %v", s.FunctionStatus())
		}
	} else { // step or execution status
	}

	// handle each opcode separately
	// process stepinfo based on stepOp
	switch s.StepOpCode() {
	case enums.OpcodeStepRun:
		if err := tb.processStepRun(ctx, s, &res); err != nil {
			return nil, false, fmt.Errorf("error parsing step run span: %w", err)
		}
	case enums.OpcodeSleep:
		if err := tb.processSleep(ctx, s, &res); err != nil {
			return nil, false, fmt.Errorf("error parsing sleep span: %w", err)
		}
	case enums.OpcodeWaitForEvent:
		if err := tb.processWaitForEvent(ctx, s, &res); err != nil {
			return nil, false, fmt.Errorf("error parsing waitForEvent span: %w", err)
		}
	case enums.OpcodeInvokeFunction:
		if err := tb.processInvoke(ctx, s, &res); err != nil {
			return nil, false, fmt.Errorf("error parsing invoke span: %w", err)
		}
	default: // these are likely execution spans
		if err := tb.processExec(ctx, s, &res); err != nil {
			return nil, false, fmt.Errorf("error parsing execution span: %w", err)
		}
	}

	// mark the span as processed
	tb.processed[s.SpanID] = true

	return &res, false, nil
}

func (tb *TraceTreeBuilder) processStepRun(ctx context.Context, span *cqrs.Span, mod *models.RunTraceSpan) error {
	return nil
}

func (tb *TraceTreeBuilder) processSleep(ctx context.Context, span *cqrs.Span, mod *models.RunTraceSpan) error {
	// sleep span always have a nested span that stores the details of the sleep itself
	if len(span.Children) != 1 {
		return fmt.Errorf("missing sleep details")
	}
	stepOp := models.StepOpSleep

	sleep := span.Children[0]
	dur := sleep.DurationMS()
	until := sleep.Timestamp.Add(sleep.Duration)
	if v, ok := sleep.SpanAttributes[consts.OtelSysStepSleepEndAt]; ok {
		if unixms, err := strconv.ParseInt(v, 10, 64); err == nil {
			until = time.UnixMilli(unixms)
		}
	}

	// set sleep details
	mod.StepOp = &stepOp
	mod.Duration = &dur
	mod.StartedAt = &sleep.Timestamp
	mod.StepInfo = models.SleepStepInfo{
		SleepUntil: until,
	}
	if until.Before(time.Now()) {
		mod.Status = models.RunTraceSpanStatusCompleted
		mod.EndedAt = &until
	}

	// mark as processed
	tb.processed[span.SpanID] = true
	tb.processed[sleep.SpanID] = true

	return nil
}

func (tb *TraceTreeBuilder) processWaitForEvent(ctx context.Context, span *cqrs.Span, mod *models.RunTraceSpan) error {
	// wait span always have a nested span that stores the details of the wait
	if len(span.Children) != 1 {
		return fmt.Errorf("missing waitForEvent details")
	}
	wait := span.Children[0]

	stepOp := models.StepOpWaitForEvent
	dur := wait.DurationMS()
	var (
		evtName    string
		expr       *string
		timeout    time.Time
		foundEvtID *ulid.ULID
		expired    *bool
	)
	if v, ok := wait.SpanAttributes[consts.OtelSysStepWaitEventName]; ok {
		evtName = v
	}
	if v, ok := wait.SpanAttributes[consts.OtelSysStepWaitExpression]; ok {
		expr = &v
	}
	if v, ok := wait.SpanAttributes[consts.OtelSysStepWaitExpires]; ok {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			timeout = time.UnixMilli(ts)
		}
	}
	if v, ok := wait.SpanAttributes[consts.OtelSysStepWaitMatchedEventID]; ok {
		if evtID, err := ulid.Parse(v); err == nil {
			foundEvtID = &evtID
			mod.Status = models.RunTraceSpanStatusCompleted
			exp := false
			expired = &exp
		}
	}
	if !timeout.IsZero() && timeout.Before(time.Now()) {
		mod.Status = models.RunTraceSpanStatusCompleted
		if v, ok := wait.SpanAttributes[consts.OtelSysStepWaitExpired]; ok {
			if exp, err := strconv.ParseBool(v); err == nil {
				expired = &exp
			}
		}
	}

	// set wait details
	mod.StepOp = &stepOp
	mod.Duration = &dur
	mod.StepInfo = models.WaitForEventStepInfo{
		EventName:    evtName,
		Expression:   expr,
		Timeout:      timeout,
		FoundEventID: foundEvtID,
		TimedOut:     expired,
	}

	// TODO: output

	tb.processed[span.SpanID] = true
	tb.processed[wait.SpanID] = true

	return nil
}

func (tb *TraceTreeBuilder) processInvoke(ctx context.Context, span *cqrs.Span, mod *models.RunTraceSpan) error {
	// invoke span always have a nested span that stores the details of the invoke
	if len(span.Children) != 1 {
		return fmt.Errorf("missing invoke details")
	}
	invoke := span.Children[0]

	stepOp := models.StepOpInvoke
	var (
		runID         *ulid.ULID
		returnEventID *ulid.ULID
		timedOut      *bool
	)

	// timeout
	expstr, ok := invoke.SpanAttributes[consts.OtelSysStepInvokeExpires]
	if !ok {
		fmt.Errorf("missing invoke expiration time")
	}
	exp, err := strconv.ParseInt(expstr, 10, 64)
	if err != nil {
	}
	timeout := time.UnixMilli(exp)

	// triggering event ID
	evtIDstr, ok := invoke.SpanAttributes[consts.OtelSysStepInvokeTriggeringEventID]
	if !ok {
		return fmt.Errorf("missing invoke triggering event ID")
	}
	triggeringEventID, err := ulid.Parse(evtIDstr)
	if err != nil {
		return fmt.Errorf("error parsing invoke triggering event ID: %w", err)
	}

	// target function ID
	fnID, ok := invoke.SpanAttributes[consts.OtelSysStepInvokeTargetFnID]
	if !ok {
		return fmt.Errorf("missing invoke target function ID for invoke")
	}

	// run ID
	if str, ok := invoke.SpanAttributes[consts.OtelSysStepInvokeRunID]; ok {
		id, err := ulid.Parse(str)
		if err != nil {
			return fmt.Errorf("error parsing invoke run ID: %w", err)
		}
		runID = &id
	}

	// return event ID
	if str, ok := invoke.SpanAttributes[consts.OtelSysStepInvokeReturnedEventID]; ok {
		evtID, err := ulid.Parse(str)
		if err != nil {
			return fmt.Errorf("error parsing invoke return event ID: %w", err)
		}
		returnEventID = &evtID

		exp := false
		timedOut = &exp
	}

	// final timed out check
	if timeout.Before(time.Now()) {
		var exp bool
		if str, ok := invoke.SpanAttributes[consts.OtelSysStepInvokeExpired]; ok {
			exp, _ = strconv.ParseBool(str)
		}
		timedOut = &exp
	}

	// set invoke details
	mod.StepOp = &stepOp
	mod.StepInfo = models.InvokeStepInfo{
		TriggeringEventID: triggeringEventID,
		FunctionID:        fnID,
		RunID:             runID,
		ReturnEventID:     returnEventID,
		Timeout:           timeout,
		TimedOut:          timedOut,
	}

	if returnEventID != nil {
		status := models.RunTraceSpanStatusCompleted
		if invoke.StatusCode == "STATUS_CODE_ERROR" {
			status = models.RunTraceSpanStatusFailed
		}
		mod.Status = status
	} else {
		if timedOut != nil && *timedOut {
			mod.Status = models.RunTraceSpanStatusFailed
		}
	}

	// TODO: output

	// mark as processed
	tb.processed[span.SpanID] = true
	tb.processed[invoke.SpanID] = true

	return nil
}

func (tb *TraceTreeBuilder) processExec(ctx context.Context, span *cqrs.Span, mod *models.RunTraceSpan) error {
	return nil
}
