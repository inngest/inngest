package run

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"google.golang.org/protobuf/types/known/timestamppb"

	rpbv2 "github.com/inngest/inngest/proto/gen/run/v2"
	"github.com/oklog/ulid/v2"
)

// runTreeBuilder builds the span tree used for the API
type runTreeBuilder struct {
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
	acctID uuid.UUID
	wsID   uuid.UUID
	appID  uuid.UUID
	fnID   uuid.UUID
	runID  ulid.ULID
}

type RunTreeOpts struct {
	AccountID   uuid.UUID
	WorkspaceID uuid.UUID
	AppID       uuid.UUID
	FunctionID  uuid.UUID
	RunID       ulid.ULID
	Spans       []*cqrs.Span
}

func NewRunTreeBuilder(opts RunTreeOpts) (*runTreeBuilder, error) {
	b := &runTreeBuilder{
		acctID:    opts.AccountID,
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
			b.root = s
		}

		if s.ScopeName == consts.OtelScopeTrigger {
			b.trigger = s
		}

		b.spans[s.SpanID] = s
		if s.ParentSpanID != nil {
			if groupID := s.GroupID(); groupID != nil {
				if _, exists := b.groups[*groupID]; !exists {
					b.groups[*groupID] = []*cqrs.Span{}
				}
				b.groups[*groupID] = append(b.groups[*groupID], s)
			}
		}
	}

	// loop through again to construct parent/child relationship
	for _, s := range opts.Spans {
		if s.ParentSpanID != nil {
			if parent, ok := b.spans[*s.ParentSpanID]; ok {
				if parent.Children == nil {
					parent.Children = []*cqrs.Span{}
				}
				parent.Children = append(parent.Children, s)
			}
		}
	}

	if b.root == nil {
		return nil, fmt.Errorf("no function run span found")
	}
	if b.trigger == nil {
		return nil, fmt.Errorf("no trigger span found")
	}

	return b, nil
}

func (tb *runTreeBuilder) Build(ctx context.Context) (*rpbv2.RunSpan, error) {
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
		root.Children = append(root.Children, tspan)
	}

	return root, nil
}

func (tb *runTreeBuilder) toRunTraceSpan(ctx context.Context, s *cqrs.Span) (*rpbv2.RunSpan, bool, error) {
	res, skipped := tb.constructSpan(ctx, s)
	if skipped {
		return nil, skipped, nil
	}

	// NOTE: step status will be updated in the individual opcode updates
	if s.ScopeName == consts.OtelScopeFunction {
		switch s.FunctionStatus() {
		case enums.RunStatusRunning:
			res.Status = rpbv2.SpanStatus_RUNNING
		case enums.RunStatusCompleted:
			res.Status = rpbv2.SpanStatus_COMPLETED
		case enums.RunStatusCancelled:
			res.Status = rpbv2.SpanStatus_CANCELLED
		case enums.RunStatusFailed, enums.RunStatusOverflowed:
			res.Status = rpbv2.SpanStatus_CANCELLED
		default:
			return nil, false, fmt.Errorf("unexpected run status: %v", s.FunctionStatus())
		}
	}

	// handle each opcode separately
	// process stepinfo based on stepOp
	switch s.StepOpCode() {
	case enums.OpcodeStepRun:
		if err := tb.processStepRun(ctx, s, res); err != nil {
			return nil, false, fmt.Errorf("error parsing step run span: %w", err)
		}
	case enums.OpcodeSleep:
		if err := tb.processSleep(ctx, s, res); err != nil {
			return nil, false, fmt.Errorf("error parsing sleep span: %w", err)
		}
	case enums.OpcodeWaitForEvent:
		if err := tb.processWaitForEvent(ctx, s, res); err != nil {
			return nil, false, fmt.Errorf("error parsing waitForEvent span: %w", err)
		}
	case enums.OpcodeInvokeFunction:
		if err := tb.processInvoke(ctx, s, res); err != nil {
			return nil, false, fmt.Errorf("error parsing invoke span: %w", err)
		}
	default: // these are likely execution spans
		if err := tb.processExec(ctx, s, res); err != nil {
			return nil, false, fmt.Errorf("error parsing execution span: %w", err)
		}
	}

	// mark the span as processed
	tb.processed[s.SpanID] = true

	return res, false, nil
}

func (tb *runTreeBuilder) constructSpan(ctx context.Context, s *cqrs.Span) (*rpbv2.RunSpan, bool) {
	// already processed skip it
	if _, ok := tb.processed[s.SpanID]; ok {
		return nil, true
	}
	// NOTE: is this check sufficient?
	if s.SpanName == "function success" {
		return nil, true
	}

	var (
		appID uuid.UUID
		fnID  uuid.UUID
		runID ulid.ULID
	)

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

	return &rpbv2.RunSpan{
		AppId:        appID.String(),
		FunctionId:   fnID.String(),
		RunId:        runID.String(),
		TraceId:      s.TraceID,
		ParentSpanId: s.ParentSpanID,
		SpanId:       s.SpanID,
		Name:         name,
		Status:       rpbv2.SpanStatus_RUNNING,
		QueuedAt:     timestamppb.New(ulid.Time(runID.Time())),
		StartedAt:    timestamppb.New(s.Timestamp),
		EndedAt:      timestamppb.New(endedAt),
		DurationMs:   dur,
	}, false
}

func (tb *runTreeBuilder) processStepRun(ctx context.Context, span *cqrs.Span, mod *rpbv2.RunSpan) error {
	// step runs should always have groupIDs
	groupID := span.GroupID()
	if groupID == nil {
		return fmt.Errorf("step run missing group ID")
	}

	maxAttempts, err := strconv.Atoi(span.SpanAttributes[consts.OtelSysStepMaxAttempt])
	if err != nil {
		return fmt.Errorf("error parsing max attempts: %w", err)
	}

	// check how many peers are there in the group
	peers, ok := tb.groups[*groupID]
	if !ok {
		return fmt.Errorf("internal error: groupID not registered: %s", *groupID)
	}

	stepOp := rpbv2.SpanStepOp_RUN
	mod.StepOp = &stepOp

	// not need to provide nesting if it's just itself and it's successful
	if len(peers) == 1 && span.Status() == cqrs.SpanStatusOk {
		mod.Attempts = 1
		mod.Status = rpbv2.SpanStatus_COMPLETED
		tb.processed[span.SpanID] = true
		return nil
	}

	// modify the span as a group, and nest each peer under it instead
	mod.SpanId = fmt.Sprintf("steprun-%s", *groupID)
	if mod.Children == nil {
		mod.Children = []*rpbv2.RunSpan{}
	}

	// sort the peers in asc order before proceeding
	sort.Slice(peers, func(i, j int) bool {
		return peers[i].Timestamp.UnixMilli() < peers[j].Timestamp.UnixMilli()
	})

	for i, p := range peers {
		nested, skipped := tb.constructSpan(ctx, p)
		// NOTE: might be able to handle this better
		if skipped {
			continue
		}

		attempt := 1
		if str, ok := p.SpanAttributes[consts.OtelSysStepAttempt]; ok {
			if count, err := strconv.Atoi(str); err == nil {
				attempt = count + 1
			}
		}

		status := rpbv2.SpanStatus_RUNNING
		switch p.Status() {
		case cqrs.SpanStatusOk:
			status = rpbv2.SpanStatus_COMPLETED
		case cqrs.SpanStatusError:
			status = rpbv2.SpanStatus_FAILED
		default:
			nested.EndedAt = nil
		}

		nested.Name = fmt.Sprintf("Attempt %d", attempt)
		nested.StepOp = &stepOp
		nested.Attempts = int32(attempt)
		nested.Status = status

		// TODO: output

		// last one
		// update end time of the group as well
		if i == len(peers)-1 {
			pend := p.Timestamp.Add(p.Duration)

			// TODO: check if the span has already completed or not
			dur := int64(pend.Sub(span.Timestamp) / time.Millisecond)
			mod.Attempts = int32(attempt)
			mod.DurationMs = dur
			mod.EndedAt = timestamppb.New(pend)

			switch status {
			case rpbv2.SpanStatus_RUNNING:
				mod.EndedAt = nil
			case rpbv2.SpanStatus_COMPLETED:
				mod.Status = rpbv2.SpanStatus_COMPLETED
			case rpbv2.SpanStatus_FAILED:
				// check if this failure is the final failure of all attempts
				if attempt == maxAttempts {
					mod.Status = rpbv2.SpanStatus_FAILED
					mod.Attempts = int32(maxAttempts)
				}
			}
		}

		mod.Children = append(mod.Children, nested)
		tb.processed[p.SpanID] = true
	}

	return nil
}

func (tb *runTreeBuilder) processSleep(ctx context.Context, span *cqrs.Span, mod *rpbv2.RunSpan) error {
	// sleep span always have a nested span that stores the details of the sleep itself
	if len(span.Children) != 1 {
		return fmt.Errorf("missing sleep details")
	}
	stepOp := rpbv2.SpanStepOp_SLEEP

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
	mod.DurationMs = dur
	mod.StartedAt = timestamppb.New(sleep.Timestamp)
	mod.StepInfo = &rpbv2.StepInfo{
		Info: &rpbv2.StepInfo_Sleep{
			Sleep: &rpbv2.StepInfoSleep{
				SleepUntil: timestamppb.New(until),
			},
		},
	}
	if until.Before(time.Now()) {
		mod.Status = rpbv2.SpanStatus_COMPLETED
		mod.EndedAt = timestamppb.New(until)
	}

	// mark as processed
	tb.processed[span.SpanID] = true
	tb.processed[sleep.SpanID] = true

	return nil
}

func (tb *runTreeBuilder) processWaitForEvent(ctx context.Context, span *cqrs.Span, mod *rpbv2.RunSpan) error {
	// wait span always have a nested span that stores the details of the wait
	if len(span.Children) != 1 {
		return fmt.Errorf("missing waitForEvent details")
	}
	wait := span.Children[0]

	stepOp := rpbv2.SpanStepOp_WAIT_FOR_EVENT
	status := rpbv2.SpanStatus_WAITING
	dur := wait.DurationMS()
	var (
		evtName    string
		expr       *string
		timeout    time.Time
		foundEvtID *string
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
			id := evtID.String()
			foundEvtID = &id
			status = rpbv2.SpanStatus_COMPLETED
			exp := false
			expired = &exp
		}
	}
	if !timeout.IsZero() && timeout.Before(time.Now()) {
		status = rpbv2.SpanStatus_COMPLETED
		if v, ok := wait.SpanAttributes[consts.OtelSysStepWaitExpired]; ok {
			if exp, err := strconv.ParseBool(v); err == nil {
				expired = &exp
			}
		}
	}

	// set wait details
	mod.StepOp = &stepOp
	mod.DurationMs = dur
	mod.Status = status
	mod.StepInfo = &rpbv2.StepInfo{
		Info: &rpbv2.StepInfo_Wait{
			Wait: &rpbv2.StepInfoWaitForEvent{
				EventName:    evtName,
				Expression:   expr,
				Timeout:      timestamppb.New(timeout),
				FoundEventId: foundEvtID,
				TimedOut:     expired,
			},
		},
	}

	// TODO: output

	tb.processed[span.SpanID] = true
	tb.processed[wait.SpanID] = true

	return nil
}

func (tb *runTreeBuilder) processInvoke(ctx context.Context, span *cqrs.Span, mod *rpbv2.RunSpan) error {
	// invoke span always have a nested span that stores the details of the invoke
	if len(span.Children) != 1 {
		return fmt.Errorf("missing invoke details")
	}
	invoke := span.Children[0]

	stepOp := rpbv2.SpanStepOp_INVOKE
	var (
		runID         *string
		returnEventID *string
		timedOut      *bool
	)

	// timeout
	expstr, ok := invoke.SpanAttributes[consts.OtelSysStepInvokeExpires]
	if !ok {
		return fmt.Errorf("missing invoke expiration time")
	}
	exp, err := strconv.ParseInt(expstr, 10, 64)
	if err != nil {
		return fmt.Errorf("error parsing expiration timestamp")
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
		runid, err := ulid.Parse(str)
		if err != nil {
			return fmt.Errorf("error parsing invoke run ID: %w", err)
		}
		id := runid.String()
		runID = &id
	}

	// return event ID
	if str, ok := invoke.SpanAttributes[consts.OtelSysStepInvokeReturnedEventID]; ok {
		evtID, err := ulid.Parse(str)
		if err != nil {
			return fmt.Errorf("error parsing invoke return event ID: %w", err)
		}
		id := evtID.String()
		returnEventID = &id

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
	mod.StepInfo = &rpbv2.StepInfo{
		Info: &rpbv2.StepInfo_Invoke{
			Invoke: &rpbv2.StepInfoInvoke{
				TriggeringEventId: triggeringEventID.String(),
				FunctionId:        fnID,
				RunId:             runID,
				ReturnEventId:     returnEventID,
				Timeout:           timestamppb.New(timeout),
				TimedOut:          timedOut,
			},
		},
	}

	if returnEventID != nil {
		status := rpbv2.SpanStatus_COMPLETED
		if invoke.StatusCode == "STATUS_CODE_ERROR" {
			status = rpbv2.SpanStatus_FAILED
		}
		mod.Status = status
	} else {
		if timedOut != nil && *timedOut {
			mod.Status = rpbv2.SpanStatus_FAILED
		}
	}

	// TODO: output

	// mark as processed
	tb.processed[span.SpanID] = true
	tb.processed[invoke.SpanID] = true

	return nil
}

func (tb *runTreeBuilder) processExec(ctx context.Context, span *cqrs.Span, mod *rpbv2.RunSpan) error {
	return nil
}
