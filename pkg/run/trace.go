package run

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
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
	ErrNoGroupFound      = fmt.Errorf("no group execution found for span")
	ErrStepNoGroup       = fmt.Errorf("span is not part of group")
	ErrRedundantExecSpan = fmt.Errorf("redundant execution span")
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
		// ignore parallelism planning spans
		if s.StepOpCode() == enums.OpcodeStepPlanned {
			if _, ok := s.SpanAttributes[consts.OtelSysStepPlan]; ok {
				continue
			}
		}

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
		// ignore parallelism planning spans
		if s.StepOpCode() == enums.OpcodeStepPlanned {
			if _, ok := s.SpanAttributes[consts.OtelSysStepPlan]; ok {
				continue
			}
		}

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
	case rpbv2.SpanStatus_RUNNING:
		root.EndedAt = nil
	case rpbv2.SpanStatus_COMPLETED, rpbv2.SpanStatus_FAILED, rpbv2.SpanStatus_CANCELLED, rpbv2.SpanStatus_SKIPPED:
		finished = true
	}

	var last *rpbv2.RunSpan
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
		last = tspan
	}

	// append a queued span if function is not done yet
	if !hasFinished(root) && hasFinished(last) {
		queued := &rpbv2.RunSpan{
			AccountId:    tb.acctID.String(),
			WorkspaceId:  tb.wsID.String(),
			AppId:        tb.appID.String(),
			FunctionId:   tb.fnID.String(),
			RunId:        tb.runID.String(),
			TraceId:      last.TraceId,
			ParentSpanId: &root.SpanId,
			SpanId:       "queued",
			Name:         "Queued step",
			Status:       rpbv2.SpanStatus_QUEUED,
			QueuedAt:     last.EndedAt,
		}
		root.Children = append(root.Children, queued)
	}

	return root, nil
}

func (tb *runTree) toRunSpan(ctx context.Context, s *cqrs.Span) (span *rpbv2.RunSpan, skipped bool, err error) {
	res, skipped := tb.constructSpan(ctx, s)
	if skipped {
		return nil, skipped, nil
	}

	// NOTE: step status will be updated in the individual opcode updates
	switch s.ScopeName {
	case consts.OtelScopeFunction:
		// default to running, since in-progress spans don't have status codes
		fnstatus := rpbv2.SpanStatus_RUNNING
		switch s.FunctionStatus() {
		case enums.RunStatusCompleted:
			fnstatus = rpbv2.SpanStatus_COMPLETED
		case enums.RunStatusCancelled:
			fnstatus = rpbv2.SpanStatus_CANCELLED
		case enums.RunStatusFailed, enums.RunStatusOverflowed:
			fnstatus = rpbv2.SpanStatus_FAILED
		case enums.RunStatusSkipped:
			fnstatus = rpbv2.SpanStatus_SKIPPED
		}
		res.Status = fnstatus

	// step scope are the spans that hold the actual data of a step execution
	// process these directly
	// e.g. sleep, invoke, wait
	case consts.OtelScopeStep:
		switch s.StepOpCode() {
		case enums.OpcodeSleep:
			if err := tb.processSleep(ctx, s, res); err != nil {
				return nil, false, fmt.Errorf("error parsing invoke: %w", err)
			}
		case enums.OpcodeWaitForEvent:
			if err := tb.processWaitForEvent(ctx, s, res); err != nil {
				return nil, false, fmt.Errorf("error parsing invoke: %w", err)
			}
		case enums.OpcodeInvokeFunction:
			if err := tb.processInvoke(ctx, s, res); err != nil {
				return nil, false, fmt.Errorf("error parsing invoke: %w", err)
			}
		}

	// the rest are grouped executions
	default:
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
			if err := tb.processStepRunGroup(ctx, s, res); err != nil {
				return nil, false, fmt.Errorf("error grouping step runs: %w", err)
			}
		case enums.OpcodeSleep:
			if err := tb.processSleepGroup(ctx, s, res); err != nil {
				if err == ErrRedundantExecSpan {
					return nil, true, nil // no-op
				}
				return nil, false, fmt.Errorf("error grouping sleeps: %w", err)
			}
		case enums.OpcodeWaitForEvent:
			if err := tb.processWaitForEventGroup(ctx, s, res); err != nil {
				if err == ErrRedundantExecSpan {
					return nil, true, nil // no-op
				}
				return nil, false, fmt.Errorf("error grouping waitForEvent: %w", err)
			}
		case enums.OpcodeInvokeFunction:
			if err := tb.processInvokeGroup(ctx, s, res); err != nil {
				if err == ErrRedundantExecSpan {
					return nil, true, nil // no-op
				}

				return nil, false, fmt.Errorf("error grouping invoke: %w", err)
			}
		default:
			// execution spans
			if s.ScopeName == consts.OtelScopeExecution {
				if err := tb.processExecGroup(ctx, s, res); err != nil {
					return nil, false, fmt.Errorf("error grouping executions: %w", err)
				}
			}
		}
	}

	tb.markProcessed(s)
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

	var stepID *string
	if attrStepID, ok := s.SpanAttributes[consts.OtelSysStepID]; ok && attrStepID != "" {
		stepID = &attrStepID
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
		StepId:       stepID,
	}, false
}

func (tb *runTree) processStepRunGroup(ctx context.Context, span *cqrs.Span, mod *rpbv2.RunSpan) error {
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

	if v, ok := span.SpanAttributes[consts.OtelSysStepRunType]; ok {
		mod.StepInfo = &rpbv2.StepInfo{
			Info: &rpbv2.StepInfo_Run{
				Run: &rpbv2.StepInfoRun{
					Type: &v,
				},
			},
		}
	}

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

		mod.Status = rpbv2.SpanStatus_COMPLETED
		mod.OutputId = &outputID

		tb.processed[span.SpanID] = true
		return nil
	}

	mod.EndedAt = nil
	// modify the span as a group, and nest each peer under it instead
	mod.SpanId = fmt.Sprintf("steprun-%s", *groupID)
	if mod.Children == nil {
		mod.Children = []*rpbv2.RunSpan{}
	}

	var (
		attempt     int32
		notFinished bool
	)
	for i, p := range peers {
		if i == 0 {
			mod.StartedAt = timestamppb.New(p.Timestamp)
			dur := time.Since(mod.StartedAt.AsTime())
			mod.DurationMs = int64(dur / time.Millisecond)
		}

		nested, skipped := tb.constructSpan(ctx, p)
		// NOTE: might be able to handle this better
		if skipped {
			continue
		}

		status := toProtoStatus(p)
		// output
		outputID, err := tb.outputID(nested)
		if err != nil {
			return err
		}

		nested.StepOp = &stepOp
		nested.Attempts = attempt
		nested.Status = status
		switch status { // only set outputID if the span is already finished
		case rpbv2.SpanStatus_RUNNING:
			nested.EndedAt = nil
		case rpbv2.SpanStatus_COMPLETED, rpbv2.SpanStatus_FAILED:
			nested.OutputId = &outputID
		}

		// last one
		// update end time and status of the group as well
		if i == len(peers)-1 {
			mod.Name = nested.Name
			mod.Attempts = attempt

			switch status {
			case rpbv2.SpanStatus_COMPLETED:
				mod.Status = rpbv2.SpanStatus_COMPLETED
				mod.OutputId = &outputID
				mod.EndedAt = nested.EndedAt
			case rpbv2.SpanStatus_FAILED:
				// check if this failure is the final failure of all attempts
				if attempt == maxAttempts-1 {
					mod.EndedAt = nested.EndedAt
					mod.Status = rpbv2.SpanStatus_FAILED
					mod.Attempts = maxAttempts
					mod.OutputId = &outputID
				} else {
					notFinished = true
				}
			}

			if mod.EndedAt != nil {
				dur := mod.EndedAt.AsTime().Sub(mod.StartedAt.AsTime())
				mod.DurationMs = int64(dur / time.Millisecond)
			}
		}
		nested.Name = fmt.Sprintf("Attempt %d", attempt)
		mod.Children = append(mod.Children, nested)
		tb.processed[p.SpanID] = true
		attempt++

		if notFinished {
			queued := tb.queuedSpan(nested)
			mod.Children = append(mod.Children, queued)
		}
	}

	// check if the nested span is the same one, if so discard it
	if hasFinished(mod) && hasIdenticalChild(mod, span) {
		mod.SpanId = span.SpanID // reset the spanID
		mod.Children = nil
	}

	return nil
}

func (tb *runTree) processSleepGroup(ctx context.Context, span *cqrs.Span, mod *rpbv2.RunSpan) error {
	group, err := tb.findGroup(span)
	if err != nil {
		return err
	}

	if len(group) == 1 {
		return tb.processSleep(ctx, span, mod)
	}

	stepOp := rpbv2.SpanStepOp_SLEEP
	mod.StepOp = &stepOp

	var i int
	// if there are more than one, that means this is not the first attempt to execute
	for _, peer := range group {
		if i == 0 {
			mod.StartedAt = timestamppb.New(peer.Timestamp)
		}

		opcode := peer.StepOpCode()
		if opcode == enums.OpcodeSleep && peer.ScopeName == consts.OtelScopeExecution {
			// ignore this span since it's not needed
			tb.markProcessed(peer)
			continue
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
			err := tb.processSleep(ctx, peer, nested)
			switch err {
			case nil: // no-op
			case ErrRedundantExecSpan:
				tb.markProcessed(peer)
				continue
			default:
				return err
			}

			mod.Name = nested.Name
			mod.OutputId = nil
			mod.StepInfo = nested.StepInfo
			mod.EndedAt = nested.EndedAt
			mod.Status = nested.Status

			if mod.EndedAt != nil {
				dur := mod.EndedAt.AsTime().Sub(mod.StartedAt.AsTime())
				mod.DurationMs = int64(dur / time.Millisecond)
			}
		}
		nested.Name = fmt.Sprintf("Attempt %d", i)
		mod.Children = append(mod.Children, nested)
		tb.markProcessed(peer)
		i++
	}

	// if the total number of children span end up with just one, it means
	// redundant spans has been excluded, so it's basically the same span
	// as the parent. We can discard it in this case
	if hasFinished(mod) && hasIdenticalChild(mod, span) {
		mod.Children = nil
	}

	return nil
}

func (tb *runTree) processSleep(ctx context.Context, span *cqrs.Span, mod *rpbv2.RunSpan) error {
	defer tb.markProcessed(span)

	if span.ScopeName == consts.OtelScopeExecution {
		// ignore this span type for sleep
		return ErrRedundantExecSpan
	}

	stepOp := rpbv2.SpanStepOp_SLEEP

	dur := span.DurationMS()
	until := span.Timestamp.Add(span.Duration)
	if v, ok := span.SpanAttributes[consts.OtelSysStepSleepEndAt]; ok {
		if unixms, err := strconv.ParseInt(v, 10, 64); err == nil {
			until = time.UnixMilli(unixms)
		}
	}

	// set sleep details
	mod.Name = *span.StepDisplayName()
	mod.OutputId = nil
	mod.StepOp = &stepOp
	mod.DurationMs = dur
	mod.StartedAt = timestamppb.New(span.Timestamp)
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

	var i int
	// if there are more than one, that means this is not the first attempt to execute
	for _, peer := range group {
		if i == 0 {
			mod.StartedAt = timestamppb.New(peer.Timestamp)
		}

		opcode := peer.StepOpCode()
		if opcode == enums.OpcodeWaitForEvent && peer.ScopeName == consts.OtelScopeExecution {
			// ignore this span since it's not needed
			tb.markProcessed(peer)
			continue
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
			err = tb.processWaitForEvent(ctx, peer, nested)
			switch err {
			case nil: // no-op
			case ErrRedundantExecSpan:
				tb.markProcessed(peer)
				continue
			default:
				return err
			}

			// update group
			mod.Name = nested.Name
			mod.OutputId = nested.OutputId
			mod.StepInfo = nested.StepInfo
			mod.EndedAt = nested.EndedAt
			mod.Status = nested.Status

			if mod.EndedAt != nil {
				dur := mod.EndedAt.AsTime().Sub(mod.StartedAt.AsTime())
				mod.DurationMs = int64(dur / time.Millisecond)
			}
		}
		nested.Name = fmt.Sprintf("Attempt %d", i)
		mod.Children = append(mod.Children, nested)
		tb.markProcessed(peer)
		i++
	}

	// if the total number of children span end up with just one, it means
	// redundant spans has been excluded, so it's basically the same span
	// as the parent. We can discard it in this case
	if len(mod.Children) == 1 && mod.StepOp.String() == mod.Children[0].StepOp.String() {
		mod.Children = nil
	}

	return nil
}

func (tb *runTree) processWaitForEvent(ctx context.Context, span *cqrs.Span, mod *rpbv2.RunSpan) error {
	defer tb.markProcessed(span)

	if span.ScopeName == consts.OtelScopeExecution {
		// ignore this span type for sleep
		return ErrRedundantExecSpan
	}

	now := time.Now()
	stepOp := rpbv2.SpanStepOp_WAIT_FOR_EVENT
	status := rpbv2.SpanStatus_WAITING
	dur := span.DurationMS()
	var (
		evtName    string
		expr       *string
		timeout    time.Time
		foundEvtID *string
		expired    *bool
	)
	if v, ok := span.SpanAttributes[consts.OtelSysStepWaitEventName]; ok {
		evtName = v
	}
	if v, ok := span.SpanAttributes[consts.OtelSysStepWaitExpression]; ok {
		expr = &v
	}
	if v, ok := span.SpanAttributes[consts.OtelSysStepWaitExpires]; ok {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			timeout = time.UnixMilli(ts)
		}
	}
	if v, ok := span.SpanAttributes[consts.OtelSysStepWaitMatchedEventID]; ok {
		if evtID, err := ulid.Parse(v); err == nil {
			id := evtID.String()
			foundEvtID = &id
			status = rpbv2.SpanStatus_COMPLETED
			exp := false
			expired = &exp
		}
	}
	if !timeout.IsZero() && timeout.Before(now) {
		status = rpbv2.SpanStatus_COMPLETED
		if v, ok := span.SpanAttributes[consts.OtelSysStepWaitExpired]; ok {
			if exp, err := strconv.ParseBool(v); err == nil {
				expired = &exp
			}
		}
	}

	endedAt := span.Timestamp.Add(time.Duration(dur * int64(time.Millisecond)))

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
	// if event was found or has ended
	if foundEvtID != nil || ((expired != nil && *expired) || (!timeout.IsZero() && timeout.Before(now))) {
		mod.EndedAt = timestamppb.New(endedAt)
	}

	// output
	var outputID *string
	if foundEvtID != nil && mod.Status == rpbv2.SpanStatus_COMPLETED {
		ident := &cqrs.SpanIdentifier{
			AccountID:   tb.acctID,
			WorkspaceID: tb.wsID,
			AppID:       tb.appID,
			FunctionID:  tb.fnID,
			TraceID:     span.TraceID,
			SpanID:      span.SpanID,
		}
		id, err := ident.Encode()
		if err != nil {
			return err
		}
		outputID = &id
	}
	mod.OutputId = outputID

	return nil
}

func (tb *runTree) processInvokeGroup(ctx context.Context, span *cqrs.Span, mod *rpbv2.RunSpan) error {
	group, err := tb.findGroup(span)
	if err != nil {
		return err
	}

	if len(group) == 1 {
		return tb.processInvoke(ctx, span, mod)
	}

	stepOp := rpbv2.SpanStepOp_INVOKE
	mod.StepOp = &stepOp

	var i int
	// if there are more than one, that means this is not the first attempt to execute
	for _, peer := range group {
		if i == 0 {
			mod.StartedAt = timestamppb.New(peer.Timestamp)
		}

		opcode := peer.StepOpCode()
		if opcode == enums.OpcodeInvokeFunction && peer.ScopeName == consts.OtelScopeExecution {
			// ignore this span since it's not needed
			tb.markProcessed(peer)
			continue
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

		// process invoke span
		if opcode == enums.OpcodeInvokeFunction {
			err = tb.processInvoke(ctx, peer, nested)
			switch err {
			case nil: // no-op
			case ErrRedundantExecSpan:
				tb.markProcessed(span)
				continue
			default:
				return err
			}

			mod.Name = nested.Name
			mod.OutputId = nested.OutputId
			mod.StepInfo = nested.StepInfo
			mod.EndedAt = nested.EndedAt
			mod.Status = nested.Status

			if mod.EndedAt != nil {
				dur := mod.EndedAt.AsTime().Sub(mod.StartedAt.AsTime())
				mod.DurationMs = int64(dur / time.Millisecond)
			}
		}
		nested.Name = fmt.Sprintf("Attempt %d", i)
		mod.Children = append(mod.Children, nested)
		tb.markProcessed(peer)
		i++
	}

	// if the total number of children span end up with just one, it means
	// redundant spans has been excluded, so it's basically the same span
	// as the parent. We can discard it in this case
	if hasFinished(mod) && hasIdenticalChild(mod, span) {
		mod.Children = nil
	}

	return nil
}

func (tb *runTree) processInvoke(ctx context.Context, span *cqrs.Span, mod *rpbv2.RunSpan) error {
	defer tb.markProcessed(span)

	if span.ScopeName == consts.OtelScopeExecution {
		// ignore this span type for invoke
		return ErrRedundantExecSpan
	}

	stepOp := rpbv2.SpanStepOp_INVOKE
	var (
		runID         *string
		returnEventID *string
		timedOut      *bool
	)

	// timeout
	expstr, ok := span.SpanAttributes[consts.OtelSysStepInvokeExpires]
	if !ok {
		return fmt.Errorf("missing invoke expiration time")
	}
	exp, err := strconv.ParseInt(expstr, 10, 64)
	if err != nil {
		return fmt.Errorf("error parsing expiration timestamp")
	}
	timeout := time.UnixMilli(exp)

	// triggering event ID
	evtIDstr, ok := span.SpanAttributes[consts.OtelSysStepInvokeTriggeringEventID]
	if !ok {
		return fmt.Errorf("missing invoke triggering event ID")
	}
	triggeringEventID, err := ulid.Parse(evtIDstr)
	if err != nil {
		return fmt.Errorf("error parsing triggering event ID: %w", err)
	}

	// target function ID
	fnID, ok := span.SpanAttributes[consts.OtelSysStepInvokeTargetFnID]
	if !ok {
		return fmt.Errorf("missing target function ID")
	}

	// run ID
	if str, ok := span.SpanAttributes[consts.OtelSysStepInvokeRunID]; ok && str != "" {
		runid, err := ulid.Parse(str)
		if err != nil {
			return fmt.Errorf("error parsing run ID: %w", err)
		}
		id := runid.String()
		runID = &id
	}

	// return event ID
	if str, ok := span.SpanAttributes[consts.OtelSysStepInvokeReturnedEventID]; ok && str != "" {
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
		if str, ok := span.SpanAttributes[consts.OtelSysStepInvokeExpired]; ok {
			exp, _ = strconv.ParseBool(str)
		}
		timedOut = &exp
	}

	// set invoke details
	mod.StepOp = &stepOp
	mod.Name = *span.StepDisplayName()
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
		if span.Status() == cqrs.SpanStatusError {
			status = rpbv2.SpanStatus_FAILED
		}
		mod.Status = status
	} else {
		if timedOut != nil && *timedOut {
			mod.Status = rpbv2.SpanStatus_FAILED
		}
	}

	if hasFinished(mod) {
		end := mod.StartedAt.AsTime().Add(span.Duration)
		mod.DurationMs = span.DurationMS()
		mod.EndedAt = timestamppb.New(end)
	}

	// output
	if returnEventID != nil || mod.Status == rpbv2.SpanStatus_FAILED {
		ident := &cqrs.SpanIdentifier{
			AccountID:   tb.acctID,
			WorkspaceID: tb.wsID,
			AppID:       tb.appID,
			FunctionID:  tb.fnID,
			TraceID:     span.TraceID,
			SpanID:      span.SpanID,
		}
		id, err := ident.Encode()
		if err != nil {
			return err
		}
		mod.OutputId = &id
	}

	return nil
}

func (tb *runTree) processExecGroup(ctx context.Context, span *cqrs.Span, mod *rpbv2.RunSpan) error {
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

		mod.Status = rpbv2.SpanStatus_COMPLETED
		mod.OutputId = &outputID

		tb.processed[span.SpanID] = true
		return nil
	}

	mod.EndedAt = nil
	mod.SpanId = fmt.Sprintf("exec-%s", *groupID)
	if mod.Children == nil {
		mod.Children = []*rpbv2.RunSpan{}
	}

	var (
		attempt     int32
		notFinished bool
	)
	for i, p := range peers {
		if i == 0 {
			mod.StartedAt = timestamppb.New(p.Timestamp)
			dur := time.Since(mod.StartedAt.AsTime())
			mod.DurationMs = int64(dur / time.Millisecond)
		}

		nested, skipped := tb.constructSpan(ctx, p)
		// NOTE: might be able to handle this better
		if skipped {
			continue
		}

		status := toProtoStatus(p)
		// output
		outputID, err := tb.outputID(nested)
		if err != nil {
			return err
		}

		nested.Name = fmt.Sprintf("Attempt %d", attempt)
		nested.Attempts = attempt
		nested.Status = status
		switch status {
		case rpbv2.SpanStatus_RUNNING:
			nested.EndedAt = nil
		case rpbv2.SpanStatus_CANCELLED, rpbv2.SpanStatus_FAILED:
			nested.OutputId = &outputID
		}

		// last one
		// update end time of the group as well
		if i == len(peers)-1 {
			mod.Attempts = attempt

			switch status {
			case rpbv2.SpanStatus_COMPLETED:
				mod.Status = rpbv2.SpanStatus_COMPLETED
				mod.OutputId = &outputID
				mod.EndedAt = nested.EndedAt

				if p.SpanName == consts.OtelExecFnOk {
					mod.Name = consts.OtelExecFnOk
				}
			case rpbv2.SpanStatus_FAILED:
				// check if this failure is the final failure of all attempts
				if attempt == maxAttempts-1 {
					mod.EndedAt = nested.EndedAt
					mod.Status = rpbv2.SpanStatus_FAILED
					mod.Attempts = maxAttempts
					mod.OutputId = &outputID
				} else {
					notFinished = true
				}
			}

			// if the name is `function error`, it's already finished
			// and mark it as failed
			if mod.Name == consts.OtelExecFnErr {
				mod.EndedAt = nested.EndedAt
				mod.Status = rpbv2.SpanStatus_FAILED
				mod.OutputId = &outputID
			}

			if mod.EndedAt != nil {
				dur := mod.EndedAt.AsTime().Sub(mod.StartedAt.AsTime())
				mod.DurationMs = int64(dur / time.Millisecond)
			}
		}

		mod.Children = append(mod.Children, nested)
		tb.markProcessed(p)
		attempt++

		if notFinished {
			queued := tb.queuedSpan(nested)
			mod.Children = append(mod.Children, queued)
		}
	}

	// check if the nested span is the same one, if so discard it
	if hasFinished(mod) && hasIdenticalChild(mod, span) {
		mod.SpanId = span.SpanID // reset the spanID
		mod.Children = nil
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

func (tb *runTree) markProcessed(span *cqrs.Span) {
	tb.processed[span.SpanID] = true
}

func (tb *runTree) queuedSpan(peer *rpbv2.RunSpan) *rpbv2.RunSpan {
	ts := time.Now()
	if peer.EndedAt != nil {
		ts = peer.EndedAt.AsTime()
	}

	name := "Queued step"
	if strings.Contains(peer.GetName(), "Attempt") {
		name = fmt.Sprintf("Attempt %d", peer.Attempts+1)
	}

	return &rpbv2.RunSpan{
		AccountId:    tb.acctID.String(),
		WorkspaceId:  tb.wsID.String(),
		AppId:        tb.appID.String(),
		FunctionId:   tb.fnID.String(),
		RunId:        tb.runID.String(),
		TraceId:      peer.TraceId,
		ParentSpanId: &peer.SpanId,
		SpanId:       "queued",
		Name:         name,
		Status:       rpbv2.SpanStatus_SCHEDULED,
		Attempts:     peer.Attempts + 1,
		QueuedAt:     timestamppb.New(ts),
		StepOp:       peer.StepOp,
	}
}

func toProtoStatus(span *cqrs.Span) rpbv2.SpanStatus {
	switch span.Status() {
	case cqrs.SpanStatusQueued:
		return rpbv2.SpanStatus_QUEUED
	case cqrs.SpanStatusOk:
		return rpbv2.SpanStatus_COMPLETED
	case cqrs.SpanStatusError:
		return rpbv2.SpanStatus_FAILED
	}
	return rpbv2.SpanStatus_RUNNING
}

func hasFinished(rs *rpbv2.RunSpan) bool {
	if rs == nil {
		return false
	}
	switch rs.Status {
	case rpbv2.SpanStatus_CANCELLED, rpbv2.SpanStatus_COMPLETED, rpbv2.SpanStatus_FAILED, rpbv2.SpanStatus_SKIPPED:
		return true
	default:
		return false
	}
}

func hasIdenticalChild(rs *rpbv2.RunSpan, s *cqrs.Span) bool {
	return len(rs.Children) == 1 && rs.SpanId == s.SpanID && rs.Name == s.SpanName
}
