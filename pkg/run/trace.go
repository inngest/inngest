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

var (
	ErrNoGroupFound = fmt.Errorf("no group execution found for span")
	ErrStepNoGroup  = fmt.Errorf("span is not part of group")
)

// runTree builds the span tree used for the API
type runTree struct {
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

func NewRunTree(opts RunTreeOpts) (*runTree, error) {
	b := &runTree{
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

	// sort it
	for _, g := range b.groups {
		if len(g) > 1 {
			sort.Slice(g, func(i, j int) bool {
				return g[i].Timestamp.UnixMilli() < g[j].Timestamp.UnixMilli()
			})
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

func (tb *runTree) ToRunSpan(ctx context.Context) (*rpbv2.RunSpan, error) {
	root, _, err := tb.toRunSpan(ctx, tb.root)
	if err != nil {
		return nil, fmt.Errorf("error converting function span: %w", err)
	}
	root.ParentSpanId = nil
	root.IsRoot = true

	// sort it in asc order before proceeding
	spans := tb.root.Children
	sort.Slice(spans, func(i, j int) bool {
		return spans[i].Timestamp.UnixMilli() < spans[j].Timestamp.UnixMilli()
	})

	var finished bool
	switch root.Status {
	case rpbv2.SpanStatus_COMPLETED, rpbv2.SpanStatus_FAILED:
		finished = true
	}

	// these are the execution or steps for the function run
	for _, span := range spans {
		tspan, skipped, err := tb.toRunSpan(ctx, span)
		if err != nil {
			return nil, fmt.Errorf("error converting execution span: %w", err)
		}
		// means this span was already processed so no-op here
		if skipped {
			continue
		}
		// the last output is the run's output
		if finished {
			root.OutputId = tspan.OutputId
		}
		root.Children = append(root.Children, tspan)
	}

	return root, nil
}

func (tb *runTree) toRunSpan(ctx context.Context, s *cqrs.Span) (*rpbv2.RunSpan, bool, error) {
	res, skipped := tb.constructSpan(ctx, s)
	if skipped {
		return nil, skipped, nil
	}

	// NOTE: step status will be updated in the individual opcode updates
	if s.ScopeName == consts.OtelScopeFunction {
		// default to running, since in-progress spans don't have status codes
		fnstatus := rpbv2.SpanStatus_RUNNING
		switch s.FunctionStatus() {
		case enums.RunStatusCompleted:
			fnstatus = rpbv2.SpanStatus_COMPLETED
		case enums.RunStatusCancelled:
			fnstatus = rpbv2.SpanStatus_CANCELLED
		case enums.RunStatusFailed, enums.RunStatusOverflowed:
			fnstatus = rpbv2.SpanStatus_FAILED
		}
		res.Status = fnstatus
	} else {
		// NOTE:
		// check last item in group for op code
		// due to how we wrap up function errors with the next step execution, first item might not hold the accurate op code
		group, err := tb.findGroup(s)
		if err != nil {
			return nil, false, err
		}
		last := group[len(group)-1]

		// handle each opcode separately
		// process stepinfo based on stepOp
		switch last.StepOpCode() {
		case enums.OpcodeStepRun:
			if err := tb.processStepRun(ctx, s, res); err != nil {
				return nil, false, fmt.Errorf("error parsing step run span: %w", err)
			}
		case enums.OpcodeSleep:
			if err := tb.processSleepGroup(ctx, s, res); err != nil {
				return nil, false, fmt.Errorf("error parsing sleep span: %w", err)
			}
		case enums.OpcodeWaitForEvent:
			if err := tb.processWaitForEventGroup(ctx, s, res); err != nil {
				return nil, false, fmt.Errorf("error parsing waitForEvent span: %w", err)
			}
		case enums.OpcodeInvokeFunction:
			if err := tb.processInvoke(ctx, s, res); err != nil {
				return nil, false, fmt.Errorf("error parsing invoke span: %w", err)
			}
		default:
			// execution spans
			if s.ScopeName == consts.OtelScopeExecution {
				if err := tb.processExec(ctx, s, res); err != nil {
					return nil, false, fmt.Errorf("error parsing execution span: %w", err)
				}
			}
		}
	}

	// mark the span as processed
	tb.processed[s.SpanID] = true

	return res, false, nil
}

func (tb *runTree) findGroup(s *cqrs.Span) ([]*cqrs.Span, error) {
	groupID := s.GroupID()
	if groupID == nil {
		return nil, ErrStepNoGroup
	}

	group, ok := tb.groups[*groupID]
	if !ok {
		return nil, ErrNoGroupFound
	}

	return group, nil
}

func (tb *runTree) constructSpan(ctx context.Context, s *cqrs.Span) (*rpbv2.RunSpan, bool) {
	// already processed skip it
	if _, ok := tb.processed[s.SpanID]; ok {
		return nil, true
	}

	var (
		acctID uuid.UUID
		wsID   uuid.UUID
		appID  uuid.UUID
		fnID   uuid.UUID
		runID  ulid.ULID
	)

	name := s.SpanName
	if s.StepDisplayName() != nil {
		name = *s.StepDisplayName()
	}

	if s.RunID != nil {
		runID = *s.RunID
	}
	if id := s.AccountID(); id != nil {
		acctID = *id
	}
	if id := s.WorkspaceID(); id != nil {
		wsID = *id
	}
	if id := s.AppID(); id != nil {
		appID = *id
	}
	if id := s.FunctionID(); id != nil {
		fnID = *id
	}

	dur := s.DurationMS()
	endedAt := s.Timestamp.Add(s.Duration)

	queuedAt := ulid.Time(runID.Time())
	// non function scope need to calculate from delay
	if s.ScopeName != consts.OtelScopeFunction {
		queuedAt = s.Timestamp
		if str, ok := s.SpanAttributes[consts.OtelSysDelaySystem]; ok {
			if ms, err := strconv.Atoi(str); err == nil {
				if ms > 0 {
					dur := time.Duration(ms) * time.Millisecond
					queuedAt = s.Timestamp.Add(-1 * dur)
				}
			}
		}
	}

	return &rpbv2.RunSpan{
		AccountId:    acctID.String(),
		WorkspaceId:  wsID.String(),
		AppId:        appID.String(),
		FunctionId:   fnID.String(),
		RunId:        runID.String(),
		TraceId:      s.TraceID,
		ParentSpanId: s.ParentSpanID,
		SpanId:       s.SpanID,
		Name:         name,
		Status:       rpbv2.SpanStatus_RUNNING,
		QueuedAt:     timestamppb.New(queuedAt),
		StartedAt:    timestamppb.New(s.Timestamp),
		EndedAt:      timestamppb.New(endedAt),
		DurationMs:   dur,
	}, false
}

func (tb *runTree) processStepRun(ctx context.Context, span *cqrs.Span, mod *rpbv2.RunSpan) error {
	// step runs should always have groupIDs
	groupID := span.GroupID()
	if groupID == nil {
		return fmt.Errorf("step run missing group ID")
	}

	var maxAttempts int32
	ma, err := strconv.ParseInt(span.SpanAttributes[consts.OtelSysStepMaxAttempt], 10, 32)
	if err != nil {
		return fmt.Errorf("error parsing max attempts: %w", err)
	}
	maxAttempts = int32(ma)

	// check how many peers are there in the group
	peers, ok := tb.groups[*groupID]
	if !ok {
		return fmt.Errorf("internal error: groupID not registered: %s", *groupID)
	}

	stepOp := rpbv2.SpanStepOp_RUN
	mod.StepOp = &stepOp

	// not need to provide nesting if it's just itself and it's successful
	if len(peers) == 1 && span.Status() == cqrs.SpanStatusOk {
		ident := &cqrs.SpanIdentifier{
			AccountID:   tb.acctID,
			WorkspaceID: tb.wsID,
			AppID:       tb.appID,
			FunctionID:  tb.fnID,
			TraceID:     span.TraceID,
			SpanID:      span.SpanID,
		}
		outputID, err := ident.Encode()
		if err != nil {
			return err
		}

		mod.Attempts = 1
		mod.Status = rpbv2.SpanStatus_COMPLETED
		mod.OutputId = &outputID

		tb.processed[span.SpanID] = true
		return nil
	}

	// modify the span as a group, and nest each peer under it instead
	mod.SpanId = fmt.Sprintf("steprun-%s", *groupID)
	if mod.Children == nil {
		mod.Children = []*rpbv2.RunSpan{}
	}

	for i, p := range peers {
		nested, skipped := tb.constructSpan(ctx, p)
		// NOTE: might be able to handle this better
		if skipped {
			continue
		}

		var attempt int32
		attempt = 1
		if str, ok := p.SpanAttributes[consts.OtelSysStepAttempt]; ok {
			if count, err := strconv.ParseInt(str, 10, 32); err == nil {
				attempt = int32(count) + 1
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

		// output
		ident := &cqrs.SpanIdentifier{
			AccountID:   tb.acctID,
			WorkspaceID: tb.wsID,
			AppID:       tb.appID,
			FunctionID:  tb.fnID,
			TraceID:     nested.TraceId,
			SpanID:      nested.SpanId,
		}

		var outputID *string
		switch status { // only set outputID if the span is already finished
		case rpbv2.SpanStatus_COMPLETED, rpbv2.SpanStatus_FAILED:
			id, err := ident.Encode()
			if err != nil {
				return err
			}
			outputID = &id
		}

		nested.Name = fmt.Sprintf("Attempt %d", attempt)
		nested.StepOp = &stepOp
		nested.Attempts = attempt
		nested.Status = status
		nested.OutputId = outputID

		// last one
		// update end time and status of the group as well
		if i == len(peers)-1 {
			pend := p.Timestamp.Add(p.Duration)

			// TODO: check if the span has already completed or not
			dur := int64(pend.Sub(span.Timestamp) / time.Millisecond)
			mod.Attempts = attempt
			mod.DurationMs = dur
			mod.EndedAt = timestamppb.New(pend)

			switch status {
			case rpbv2.SpanStatus_RUNNING:
				mod.EndedAt = nil
			case rpbv2.SpanStatus_COMPLETED:
				mod.Status = rpbv2.SpanStatus_COMPLETED
				mod.OutputId = outputID
			case rpbv2.SpanStatus_FAILED:
				// check if this failure is the final failure of all attempts
				if attempt == maxAttempts {
					mod.Status = rpbv2.SpanStatus_FAILED
					mod.Attempts = maxAttempts
					mod.OutputId = outputID
				}
			}
		}

		mod.Children = append(mod.Children, nested)
		tb.processed[p.SpanID] = true
	}

	return nil
}

func (tb *runTree) processSleepGroup(ctx context.Context, span *cqrs.Span, mod *rpbv2.RunSpan) error {
	group, err := tb.findGroup(span)
	if err != nil {
		return err
	}

	stepOp := rpbv2.SpanStepOp_SLEEP
	mod.StepOp = &stepOp

	if len(group) == 1 {
		return tb.processSleep(ctx, span, mod)
	}

	// if there are more than one, that means this is not the first attempt to execute
	var startedAt time.Time

	for _, peer := range group {
		if startedAt.IsZero() {
			startedAt = peer.Timestamp
		}

		nested, skipped := tb.constructSpan(ctx, peer)
		if skipped {
			continue
		}

		status := toProtoStatus(peer)
		if status == rpbv2.SpanStatus_RUNNING {
			nested.EndedAt = nil
		}

		outputID, err := tb.outputID(nested)
		if err != nil {
			return err
		}

		nested.Status = status
		nested.OutputId = &outputID

		// process sleep span
		if peer.StepOpCode() == enums.OpcodeSleep {
			if err := tb.processSleep(ctx, peer, mod); err != nil {
				return err
			}
			nested.OutputId = nil
			nested.StepInfo = mod.StepInfo
			nested.DurationMs = mod.DurationMs
			nested.StartedAt = mod.StartedAt
			nested.Status = mod.Status
		}

		mod.Children = append(mod.Children, nested)
		// mark as processed
		tb.processed[peer.SpanID] = true
	}
	mod.StartedAt = timestamppb.New(startedAt)
	if mod.EndedAt != nil {
		dur := mod.EndedAt.AsTime().Sub(startedAt)
		mod.DurationMs = int64(dur / time.Millisecond)
	}

	return nil
}

func (tb *runTree) processSleep(ctx context.Context, span *cqrs.Span, mod *rpbv2.RunSpan) error {
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
	mod.Name = *sleep.StepDisplayName()
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

func (tb *runTree) processWaitForEventGroup(ctx context.Context, span *cqrs.Span, mod *rpbv2.RunSpan) error {
	group, err := tb.findGroup(span)
	if err != nil {
		return err
	}

	if len(group) == 1 {
		return tb.processWaitForEvent(ctx, span, mod)
	}

	stepOp := rpbv2.SpanStepOp_WAIT_FOR_EVENT
	mod.StepOp = &stepOp
	// if there are more than one, that means this is not the first attempt to execute
	var startedAt time.Time

	for _, peer := range group {
		if startedAt.IsZero() {
			startedAt = peer.Timestamp
		}

		nested, skipped := tb.constructSpan(ctx, peer)
		if skipped {
			continue
		}

		status := toProtoStatus(peer)
		if status == rpbv2.SpanStatus_RUNNING {
			nested.EndedAt = nil
		}

		outputID, err := tb.outputID(nested)
		if err != nil {
			return err
		}

		nested.Status = status
		nested.OutputId = &outputID

		// process wait span
		if peer.StepOpCode() == enums.OpcodeWaitForEvent {
			if err := tb.processWaitForEvent(ctx, peer, mod); err != nil {
				return err
			}
			nested.OutputId = mod.OutputId
			nested.StepInfo = mod.StepInfo
			nested.StartedAt = mod.StartedAt
			nested.Status = mod.Status
		}

		mod.Children = append(mod.Children, nested)
		// mark as processed
		tb.processed[peer.SpanID] = true
	}
	mod.StartedAt = timestamppb.New(startedAt)
	if mod.EndedAt != nil {
		dur := mod.EndedAt.AsTime().Sub(startedAt)
		mod.DurationMs = int64(dur / time.Millisecond)
	}

	return nil
}

func (tb *runTree) processWaitForEvent(ctx context.Context, span *cqrs.Span, mod *rpbv2.RunSpan) error {
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
	mod.Name = *span.StepDisplayName()
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

	// output
	if foundEvtID != nil && mod.Status == rpbv2.SpanStatus_COMPLETED {
		ident := &cqrs.SpanIdentifier{
			AccountID:   tb.acctID,
			WorkspaceID: tb.wsID,
			AppID:       tb.appID,
			FunctionID:  tb.fnID,
			TraceID:     wait.TraceID,
			SpanID:      wait.SpanID,
		}
		id, err := ident.Encode()
		if err != nil {
			return err
		}
		mod.OutputId = &id
	}

	tb.processed[span.SpanID] = true
	tb.processed[wait.SpanID] = true

	return nil
}

func (tb *runTree) processInvoke(ctx context.Context, span *cqrs.Span, mod *rpbv2.RunSpan) error {
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
		return fmt.Errorf("error parsing triggering event ID: %w", err)
	}

	// target function ID
	fnID, ok := invoke.SpanAttributes[consts.OtelSysStepInvokeTargetFnID]
	if !ok {
		return fmt.Errorf("missing target function ID")
	}

	// run ID
	if str, ok := invoke.SpanAttributes[consts.OtelSysStepInvokeRunID]; ok && str != "" {
		runid, err := ulid.Parse(str)
		if err != nil {
			return fmt.Errorf("error parsing run ID: %w", err)
		}
		id := runid.String()
		runID = &id
	}

	// return event ID
	if str, ok := invoke.SpanAttributes[consts.OtelSysStepInvokeReturnedEventID]; ok && str != "" {
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
		if invoke.Status() == cqrs.SpanStatusError {
			status = rpbv2.SpanStatus_FAILED
		}
		mod.Status = status
	} else {
		if timedOut != nil && *timedOut {
			mod.Status = rpbv2.SpanStatus_FAILED
		}
	}

	// output
	if returnEventID != nil || mod.Status == rpbv2.SpanStatus_FAILED {
		ident := &cqrs.SpanIdentifier{
			AccountID:   tb.acctID,
			WorkspaceID: tb.wsID,
			AppID:       tb.appID,
			FunctionID:  tb.fnID,
			TraceID:     invoke.TraceID,
			SpanID:      invoke.SpanID,
		}
		id, err := ident.Encode()
		if err != nil {
			return err
		}
		mod.OutputId = &id
	}

	// mark as processed
	tb.processed[span.SpanID] = true
	tb.processed[invoke.SpanID] = true

	return nil
}

func (tb *runTree) processExec(ctx context.Context, span *cqrs.Span, mod *rpbv2.RunSpan) error {
	// check groupIDs
	groupID := span.GroupID()
	if groupID == nil {
		return fmt.Errorf("execution missing group ID")
	}

	var maxAttempts int32
	ma, err := strconv.ParseInt(span.SpanAttributes[consts.OtelSysStepMaxAttempt], 10, 32)
	if err != nil {
		return fmt.Errorf("error parsing max attempts: %w", err)
	}
	maxAttempts = int32(ma)

	// check how many peers are there in the group
	peers, ok := tb.groups[*groupID]
	if !ok {
		return fmt.Errorf("internal error: groupID not registered: %s", *groupID)
	}

	if len(peers) == 1 && span.Status() == cqrs.SpanStatusOk {
		ident := &cqrs.SpanIdentifier{
			AccountID:   tb.acctID,
			WorkspaceID: tb.wsID,
			AppID:       tb.appID,
			FunctionID:  tb.fnID,
			TraceID:     span.TraceID,
			SpanID:      span.SpanID,
		}
		outputID, err := ident.Encode()
		if err != nil {
			return err
		}

		mod.Attempts = 1
		mod.Status = rpbv2.SpanStatus_COMPLETED
		mod.OutputId = &outputID

		tb.processed[span.SpanID] = true
		return nil
	}

	mod.SpanId = fmt.Sprintf("exec-%s", *groupID)
	if mod.Children == nil {
		mod.Children = []*rpbv2.RunSpan{}
	}

	for i, p := range peers {
		nested, skipped := tb.constructSpan(ctx, p)
		// NOTE: might be able to handle this better
		if skipped {
			continue
		}

		var attempt int32
		attempt = 1
		if str, ok := p.SpanAttributes[consts.OtelSysStepAttempt]; ok {
			if count, err := strconv.ParseInt(str, 10, 32); err == nil {
				attempt = int32(count) + 1
			}
		}

		status := toProtoStatus(p)
		if status == rpbv2.SpanStatus_FAILED {
			nested.EndedAt = nil
		}

		// output
		outputID, err := tb.outputID(nested)
		if err != nil {
			return err
		}

		nested.Name = fmt.Sprintf("Attempt %d", attempt)
		nested.Attempts = attempt
		nested.Status = status
		nested.OutputId = &outputID

		// last one
		// update end time of the group as well
		if i == len(peers)-1 {
			pend := p.Timestamp.Add(p.Duration)

			// TODO: check if the span has already completed or not
			dur := int64(pend.Sub(span.Timestamp) / time.Millisecond)
			mod.Attempts = attempt
			mod.DurationMs = dur
			mod.EndedAt = timestamppb.New(pend)

			switch status {
			case rpbv2.SpanStatus_RUNNING:
				mod.EndedAt = nil
			case rpbv2.SpanStatus_COMPLETED:
				mod.Status = rpbv2.SpanStatus_COMPLETED
				mod.OutputId = &outputID

				if p.SpanName == consts.OtelExecFnOk {
					mod.Name = consts.OtelExecFnOk
				}
			case rpbv2.SpanStatus_FAILED:
				// check if this failure is the final failure of all attempts
				if attempt == maxAttempts {
					mod.Status = rpbv2.SpanStatus_FAILED
					mod.Attempts = maxAttempts
					mod.OutputId = &outputID
				}
			}

			// if the name is `function error`, it's already finished
			// and mark it as failed
			if mod.Name == consts.OtelExecFnErr {
				mod.Status = rpbv2.SpanStatus_FAILED
				mod.OutputId = &outputID
			}
		}

		mod.Children = append(mod.Children, nested)
		tb.processed[p.SpanID] = true
	}

	return nil
}

func (tb *runTree) outputID(span *rpbv2.RunSpan) (string, error) {
	ident := &cqrs.SpanIdentifier{
		AccountID:   tb.acctID,
		WorkspaceID: tb.wsID,
		AppID:       tb.appID,
		FunctionID:  tb.fnID,
		TraceID:     span.TraceId,
		SpanID:      span.SpanId,
	}
	return ident.Encode()
}

func toProtoStatus(span *cqrs.Span) rpbv2.SpanStatus {
	switch span.Status() {
	case cqrs.SpanStatusOk:
		return rpbv2.SpanStatus_COMPLETED
	case cqrs.SpanStatusError:
		return rpbv2.SpanStatus_FAILED
	}
	return rpbv2.SpanStatus_RUNNING
}
