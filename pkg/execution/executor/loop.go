package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/domain"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/loops"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/util"
	"github.com/xhit/go-str2duration/v2"
)

// setLoopSemaphoreKey puts the resolved key on the keyed function semaphore
// that the loop key expression owns.  without it a tick or a wake would land
// in the bucket of whatever the event evaluated to.  a function synced
// without that semaphore has no exclusivity per key, so it is an error.
func setLoopSemaphoreKey(sems []constraintapi.Semaphore, fn inngest.Function, key string) error {
	id := constraintapi.SemaphoreIDFnKey(fn.ID, fn.Loop.Key)
	for i := range sems {
		if sems[i].ID == id {
			sems[i].EvaluatedKeyHash = util.XXHash(key)
			return nil
		}
	}
	return ErrNoLoopSemaphore
}

// openLoop makes sure the loop instance exists before the run is created.
// a trigger opens the instance.  any other input loads it, and a missing
// instance returns ErrLoopEnded so the input is dropped.
func (e *executor) openLoop(ctx context.Context, req execution.ScheduleRequest, in loops.Input, runID domain.RunID) error {
	id := loops.NewID(domain.NewTenant(req.AccountID, req.WorkspaceID), req.Function.ID, in.Key)
	if in.Kind == loops.InputEvent {
		if _, err := e.loops.Open(ctx, id, runID); err != nil {
			return fmt.Errorf("error opening loop instance: %w", err)
		}
		return nil
	}
	_, err := e.loops.Load(ctx, id)
	switch {
	case errors.Is(err, loops.ErrNotFound):
		return ErrLoopEnded
	case err != nil:
		return fmt.Errorf("error loading loop instance: %w", err)
	}
	return nil
}

// loopInput derives the input of a run from its first event.  the run
// stores nothing about the loop, so every request and every return reads
// it from the event again.
func loopInput(ctx context.Context, fn inngest.Function, events []json.RawMessage) (loops.Input, error) {
	if len(events) == 0 {
		return loops.Input{}, fmt.Errorf("loop run has no event")
	}
	evt := map[string]any{}
	if err := json.Unmarshal(events[0], &evt); err != nil {
		return loops.Input{}, fmt.Errorf("error decoding loop run event: %w", err)
	}
	return loops.InputFromEvent(ctx, *fn.Loop, evt)
}

// loopRequest builds the loop block for one SDK request.  the run holds no
// copy of the state, so every request reads the instance from the store.
// a function without Loop gets nil.
//
// a run that already committed or ended does not reach the SDK again.  the
// store names the last run that committed, and a match returns a replay
// response in place of the loop block.  a run that ended the loop saved
// its LoopReturn under loops.ReturnStepID first, and a run whose stack
// names that step is replayed the same way.  run hands the replay to the
// normal handler, which commits nothing new and finishes the run with the
// stored state.  a trigger whose instance is gone opens a fresh one, since
// the instance ended after the trigger was scheduled.  a tick or a wake
// whose instance is gone returns ErrLoopEnded, and run finishes the run as
// cancelled without a request.
//
// the record's OpenedBy and a replayed saved return are kept on i for the
// handler, and i.loopReplay is set for a replay.
func (e *executor) loopRequest(ctx context.Context, i *runInstance) (*driver.SDKLoopRequest, *state.DriverResponse, error) {
	md, fn := i.md, i.f
	if fn.Loop == nil {
		return nil, nil, nil
	}
	if e.loops == nil {
		return nil, nil, queue.NeverRetryError(ErrNoLoopStore)
	}
	in, err := loopInput(ctx, fn, i.events)
	if err != nil {
		return nil, nil, queue.NeverRetryError(fmt.Errorf("loop run %s: %w", md.ID.RunID, err))
	}
	id := loops.NewID(domain.NewTenant(md.ID.Tenant.AccountID, md.ID.Tenant.EnvID), md.ID.FunctionID, in.Key)
	snap, err := e.loops.Load(ctx, id)
	found := err == nil
	if err != nil && !errors.Is(err, loops.ErrNotFound) {
		return nil, nil, fmt.Errorf("error loading loop instance: %w", err)
	}
	if found && snap.LastRunID != nil && *snap.LastRunID == md.ID.RunID {
		st := snap.State
		if st == nil {
			st = json.RawMessage("null")
		}
		i.loopReplay = true
		i.loopOpenedBy = snap.OpenedBy
		return nil, loopReplay(fn, state.LoopReturn{
			State:   st,
			Version: snap.Counter - 1,
		}), nil
	}
	// a saved step id is on the run's stack, and Execute loads the
	// metadata on every attempt, so the stack names a return saved by an
	// earlier attempt of this run.  the step is read only then.
	if slices.Contains(md.Stack, loops.ReturnStepID) {
		saved, err := e.loadLoopReturn(ctx, md.ID)
		if err != nil {
			return nil, nil, err
		}
		i.loopReplay = true
		i.loopSaved = &saved
		return nil, loopReplay(fn, saved.Return), nil
	}
	if !found {
		if in.Kind != loops.InputEvent {
			return nil, nil, fmt.Errorf("%w: run %s", ErrLoopEnded, md.ID.RunID)
		}
		snap, err = e.loops.Open(ctx, id, md.ID.RunID)
		if err != nil {
			return nil, nil, fmt.Errorf("error opening loop instance: %w", err)
		}
	}
	i.loopOpenedBy = snap.OpenedBy
	return &driver.SDKLoopRequest{
		State:   snap.State,
		Version: snap.Counter,
		Key:     in.Key,
		Input: driver.SDKLoopInput{
			Kind: in.Kind,
			ID:   in.ID,
		},
	}, nil, nil
}

// savedLoopReturn is the value a run saves under loops.ReturnStepID before
// it ends the loop.  OpenedBy and Counter name the record the end is for,
// so a replayed end deletes that record and no other.
type savedLoopReturn struct {
	Return   state.LoopReturn `json:"return"`
	OpenedBy domain.RunID     `json:"opened_by"`
	Counter  int64            `json:"counter"`
}

// loadLoopReturn reads the return a run saved before it ended the loop.
// one hash field is read.  a missing step fails the run without retry,
// since the stack named it.
func (e *executor) loadLoopReturn(ctx context.Context, id sv2.ID) (savedLoopReturn, error) {
	steps, err := e.smv2.LoadStepsWithIDs(ctx, id, []string{loops.ReturnStepID})
	if err != nil {
		return savedLoopReturn{}, fmt.Errorf("error loading saved loop return: %w", err)
	}
	raw, ok := steps[loops.ReturnStepID]
	if !ok {
		return savedLoopReturn{}, queue.NeverRetryError(fmt.Errorf("saved loop return of run %s is missing", id.RunID))
	}
	var saved savedLoopReturn
	if err := json.Unmarshal(raw, &saved); err != nil {
		return savedLoopReturn{}, queue.NeverRetryError(fmt.Errorf("saved loop return of run %s is invalid: %w", id.RunID, err))
	}
	return saved, nil
}

// cancelLoopRun finishes a tick or wake run whose instance ended after the
// item was enqueued.  the SDK is not called, the run has no output, and
// the run finishes as cancelled.  the error returned is
// state.ErrFunctionCancelled, which the queue drops without a retry.
func (e *executor) cancelLoopRun(ctx context.Context, i *runInstance) error {
	if err := e.Finalize(ctx, execution.FinalizeOpts{
		Metadata: i.md,
		Response: execution.FinalizeResponse{
			Type:           execution.FinalizeResponseDriver,
			DriverResponse: state.DriverResponse{},
		},
		Optional: execution.FinalizeOptional{
			FnSlug:      i.f.GetSlug(),
			InputEvents: i.events,
			Cancel:      true,
			Reason:      "loop-ended",
		},
	}); err != nil {
		return fmt.Errorf("error finalizing run of an ended loop: %w", err)
	}
	req := execution.CancelRequest{}
	for _, l := range e.lifecycles {
		go l.OnFunctionCancelled(context.WithoutCancel(ctx), i.md, req, i.events)
	}
	syncNow := e.now()
	execution.SafelyInvokeSyncListeners(ctx, e.log, e.syncLifecycles, "OnFunctionCancelled", func(sl execution.SyncLifecycleListener) {
		sl.OnFunctionCancelled(ctx, i.md, req, i.events, syncNow)
	})
	e.runEventLifecycles(ctx, func(ctx context.Context, l execution.EventLifecycleListener) {
		l.OnRunCancelled(ctx, i.md.ID, req)
	})
	return state.ErrFunctionCancelled
}

// loopReplay builds the response a run gets in place of an SDK request
// when its LoopReturn already took effect.  the response holds one
// LoopReturn with no wake, so the handler enqueues nothing.
func loopReplay(fn inngest.Function, ret state.LoopReturn) *state.DriverResponse {
	data, err := json.Marshal(ret)
	if err != nil {
		data = json.RawMessage(`{"state":null,"version":0}`)
	}
	resp := &state.DriverResponse{
		StatusCode: 206,
		Generator: []*state.GeneratorOpcode{
			{
				ID:   "loop-return",
				Op:   enums.OpcodeLoopReturn,
				Data: data,
			},
		},
	}
	if len(fn.Steps) > 0 {
		resp.Step = fn.Steps[0]
	}
	return resp
}

// loopInvalid returns the error that fails a loop run without retry.
// HandleResponse serializes msg as the run output.
func loopInvalid(msg string) error {
	return state.WrapInStandardError(state.ErrLoopInvalid, state.InngestErrLoopInvalid, msg, "")
}

// validateLoopResponse checks a successful SDK response against the function
// before any opcode is handled.  a LoopReturn must come from a loop function.
// it may share the response with step.run outputs and nothing else.  a loop
// function must end every run with a LoopReturn, so a plain output or a
// RunComplete from one fails the run and commits nothing.
func validateLoopResponse(fn inngest.Function, resp *state.DriverResponse) error {
	returns := 0
	for _, op := range resp.Generator {
		if op != nil && op.Op == enums.OpcodeLoopReturn {
			returns++
		}
	}
	if returns > 0 {
		if returns > 1 {
			return loopInvalid(fmt.Sprintf("a response may hold one LoopReturn, got %d", returns))
		}
		for _, op := range resp.Generator {
			if op == nil {
				continue
			}
			switch op.Op {
			case enums.OpcodeLoopReturn, enums.OpcodeStepRun, enums.OpcodeStep:
			default:
				return loopInvalid(fmt.Sprintf("a LoopReturn may only be returned with step.run outputs, got %s", op.Op))
			}
		}
		if fn.Loop == nil {
			return loopInvalid(fmt.Sprintf("function %s is not a loop and returned a LoopReturn", fn.GetSlug()))
		}
		return nil
	}
	if fn.Loop != nil && resp.IsFunctionResult() {
		return loopInvalid("loop run ended without a LoopReturn")
	}
	return nil
}

// runFunction returns the function of the run.  the executor's own run
// instance already holds it.  any other RunContext loads it again.
func (e *executor) runFunction(ctx context.Context, runCtx execution.RunContext) (inngest.Function, error) {
	if i, ok := runCtx.(*runInstance); ok {
		return i.f, nil
	}
	md := runCtx.Metadata()
	ef, err := e.fl.LoadFunction(ctx, md.ID.Tenant.EnvID, md.ID.FunctionID)
	if err != nil {
		return inngest.Function{}, fmt.Errorf("error loading function for loop return: %w", err)
	}
	if ef == nil || ef.Function == nil {
		return inngest.Function{}, fmt.Errorf("function %s not found for loop return", md.ID.FunctionID)
	}
	return *ef.Function, nil
}

// handleGeneratorLoopReturn applies a LoopReturn and finishes the run with
// the loop state as the output.  the whole opcode is checked before any
// store or queue write, so an invalid one changes nothing.
//
// a return with end saves the return under loops.ReturnStepID in run state,
// then deletes the instance in one compare and swap on the echoed version
// and skips the commit.  an instance that is already gone counts as ended.
// any other return enqueues the wake and the tick first and commits second.
// a timer that lands without its commit is harmless, since the timer
// handler drops an input for an instance that is gone, and the SDK handler
// receives a wake id its state does not know and ignores it.  an error
// after the save or the commit retries the attempt, and loopRequest replays
// the saved or committed return on the retry, so the SDK never sees the
// same input twice.  a replay enqueues no timer.  a replayed end deletes
// the record only when it is the one the run saved, by OpenedBy, since a
// new trigger may have opened a fresh instance since.  a replayed commit
// is answered by the store with the state the run already committed.
func (e *executor) handleGeneratorLoopReturn(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
	md := runCtx.Metadata()
	fn, err := e.runFunction(ctx, runCtx)
	if err != nil {
		return err
	}
	if fn.Loop == nil {
		return loopInvalid(fmt.Sprintf("function %s is not a loop and returned a LoopReturn", fn.GetSlug()))
	}
	if e.loops == nil {
		return queue.NeverRetryError(ErrNoLoopStore)
	}
	in, err := loopInput(ctx, fn, runCtx.Events())
	if err != nil {
		return queue.NeverRetryError(fmt.Errorf("loop run %s: %w", md.ID.RunID, err))
	}
	var (
		replay   bool
		openedBy domain.RunID
		saved    *savedLoopReturn
	)
	if i, ok := runCtx.(*runInstance); ok {
		replay = i.loopReplay
		openedBy = i.loopOpenedBy
		saved = i.loopSaved
	}

	ret, err := gen.LoopReturn()
	if err != nil {
		return loopInvalid(fmt.Sprintf("invalid LoopReturn: %s", err))
	}
	if len(ret.Subscribe) > 0 || len(ret.Unsubscribe) > 0 {
		return loopInvalid("subscriptions are not supported yet")
	}
	if len(ret.State) > consts.MaxLoopStateSize {
		return loopInvalid(fmt.Sprintf("loop state is %d bytes, the limit is %d bytes", len(ret.State), consts.MaxLoopStateSize))
	}
	// a config that did not go through sync, such as a test fixture, has no
	// TTL.  without the default every wake is rejected and every commit
	// expires the instance at once.
	ttl := fn.Loop.IdleTTL
	if ttl <= 0 {
		ttl = consts.DefaultLoopIdleTTL
	}
	var after time.Duration
	if ret.Wake != nil && !ret.End {
		if ret.Wake.ID == "" {
			return loopInvalid("loop wake has no id")
		}
		after, err = str2duration.ParseDuration(ret.Wake.After)
		if err != nil || after <= 0 {
			return loopInvalid(fmt.Sprintf("loop wake %q has an invalid duration %q", ret.Wake.ID, ret.Wake.After))
		}
		// a longer wake keeps an idle instance alive past its TTL.  this cap
		// means every instance commits or expires within its TTL plus one
		// hour.
		if after > ttl {
			return loopInvalid(fmt.Sprintf("loop wake %q is %s away, the limit is the idle_ttl of %s.  to wait longer, wake again from the wake run", ret.Wake.ID, after, ttl))
		}
		if e.shards == nil {
			return queue.NeverRetryError(fmt.Errorf("executor has no shard registry"))
		}
	}

	id := loops.NewID(domain.NewTenant(md.ID.Tenant.AccountID, md.ID.Tenant.EnvID), md.ID.FunctionID, in.Key)
	output := ret.State
	if ret.End {
		if replay {
			if saved != nil {
				if err := e.endLoopAgain(ctx, id, *saved); err != nil {
					return err
				}
			}
		} else {
			openedBy, err = e.loopOpenedBy(ctx, id, openedBy)
			if err != nil {
				return err
			}
			if err := e.saveLoopReturn(ctx, md.ID, savedLoopReturn{
				Return:   *ret,
				OpenedBy: openedBy,
				Counter:  ret.Version,
			}); err != nil {
				return err
			}
			err := e.loops.End(ctx, loops.Snapshot{
				ID:      id,
				Counter: ret.Version,
			})
			switch {
			case errors.Is(err, loops.ErrNotFound):
			case errors.Is(err, loops.ErrConflict):
				return loopInvalid(fmt.Sprintf("loop state moved under the run, version %d is stale", ret.Version))
			case err != nil:
				return fmt.Errorf("error ending loop instance: %w", err)
			}
		}
	} else {
		if ret.Wake != nil && !replay {
			if err := e.enqueueLoopWake(ctx, *md, id, ret.Wake.ID, after); err != nil {
				return fmt.Errorf("error enqueueing loop wake: %w", err)
			}
		}
		if !replay {
			if err := e.enqueueLoopTick(ctx, *md, fn, id); err != nil {
				return fmt.Errorf("error enqueueing loop tick: %w", err)
			}
		}
		commit := loops.Commit{
			ID:       id,
			RunID:    md.ID.RunID,
			Counter:  ret.Version,
			State:    ret.State,
			OpenedBy: openedBy,
		}
		// a wake gets one hour past its time, so a wake at the full TTL that
		// the queue runs late still finds the instance.
		expiry := ttl
		if ret.Wake != nil {
			expiry = max(ttl, after+time.Hour)
		}
		commit.ExpiresAt = e.now().Add(expiry)
		snap, err := e.loops.Commit(ctx, commit)
		switch {
		case errors.Is(err, loops.ErrConflict):
			return loopInvalid(fmt.Sprintf("loop state moved under the run, version %d is stale", ret.Version))
		case err != nil:
			return fmt.Errorf("error committing loop state: %w", err)
		}
		output = snap.State
	}

	done := gen
	done.Op = enums.OpcodeRunComplete
	done.Data = output
	if len(done.Data) == 0 {
		done.Data = json.RawMessage("null")
	}
	// the run span and the finished listeners read the output from the
	// response, so the RunComplete replaces the LoopReturn there.
	if resp := runCtx.DriverResponse(); resp != nil {
		for i, op := range resp.Generator {
			if op != nil && op.Op == enums.OpcodeLoopReturn {
				resp.Generator[i] = &done
				break
			}
		}
	}
	return e.handleGeneratorFunctionFinished(ctx, runCtx, done, edge)
}

// loopOpenedBy returns the opener of the record an end is for.  a known
// opener is returned as is.  a zero one, from a RunContext that is not the
// executor's own run instance, is read from the record, so the saved
// return names the record and a replayed end can delete it.  a record
// that is gone gives zero.
func (e *executor) loopOpenedBy(ctx context.Context, id loops.ID, openedBy domain.RunID) (domain.RunID, error) {
	if openedBy != (domain.RunID{}) {
		return openedBy, nil
	}
	snap, err := e.loops.Load(ctx, id)
	switch {
	case errors.Is(err, loops.ErrNotFound):
		return domain.RunID{}, nil
	case err != nil:
		return domain.RunID{}, fmt.Errorf("error loading loop instance to end it: %w", err)
	}
	return snap.OpenedBy, nil
}

// saveLoopReturn writes the return under loops.ReturnStepID in run state.
// a step that already holds the same bytes counts as saved.  a step that
// holds other bytes is an error.
func (e *executor) saveLoopReturn(ctx context.Context, id sv2.ID, saved savedLoopReturn) error {
	data, err := json.Marshal(saved)
	if err != nil {
		return queue.NeverRetryError(fmt.Errorf("error encoding loop return: %w", err))
	}
	_, err = e.smv2.SaveStep(ctx, id, loops.ReturnStepID, data)
	if errors.Is(err, state.ErrIdempotentResponse) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("error saving loop return: %w", err)
	}
	return nil
}

// endLoopAgain deletes the record a replayed end is for, when the first
// end never landed.  the record is read first.  a record that is gone, or
// that a later trigger created, is left alone.  the saved counter is the
// compare and swap, so a record that moved since fails the run without
// retry like a first end.
func (e *executor) endLoopAgain(ctx context.Context, id loops.ID, saved savedLoopReturn) error {
	if saved.OpenedBy == (domain.RunID{}) {
		return nil
	}
	snap, err := e.loops.Load(ctx, id)
	switch {
	case errors.Is(err, loops.ErrNotFound):
		return nil
	case err != nil:
		return fmt.Errorf("error loading loop instance to end it again: %w", err)
	}
	if snap.OpenedBy != saved.OpenedBy {
		return nil
	}
	err = e.loops.End(ctx, loops.Snapshot{
		ID:      id,
		Counter: saved.Counter,
	})
	switch {
	case errors.Is(err, loops.ErrNotFound):
		return nil
	case errors.Is(err, loops.ErrConflict):
		return loopInvalid(fmt.Sprintf("loop state moved under the run, version %d is stale", saved.Counter))
	case err != nil:
		return fmt.Errorf("error ending loop instance again: %w", err)
	}
	return nil
}

// enqueueLoopWake puts one KindLoopWake item on the function's partition at
// now plus after.  the job id is wake:<instance>:<wake id>, so a later return
// with the same wake id moves that item instead of adding one.  the item has
// no run id, no semaphore, no concurrency key, and no idempotency period, so
// a wake id can be used again once its item ran.  an item that is leased
// right now is left alone, since that wake is already being turned into an
// input.  the requeue happens before the commit, so a commit conflict after
// it leaves the moved item at its new time.  the wake then runs at the time
// the losing run asked for, and the SDK sees a wake id its state may not
// know.
func (e *executor) enqueueLoopWake(ctx context.Context, md sv2.Metadata, id loops.ID, wakeID string, after time.Duration) error {
	if e.shards == nil {
		return fmt.Errorf("executor has no shard registry")
	}
	jobID := loopTimerJobID(id, loops.InputWake, wakeID)
	at := e.now().Add(after)
	item := loopTimerItem(md, id, jobID, queue.KindLoopWake, loops.InputWake, wakeID)
	noIdempotency := time.Duration(0)
	opts := queue.EnqueueOpts{
		IdempotencyPeriod: &noIdempotency,
	}
	err := e.queue.Enqueue(ctx, item, at, opts)
	if !errors.Is(err, queue.ErrQueueItemExists) {
		return err
	}

	scope := queue.Scope{
		AccountID:  md.ID.Tenant.AccountID,
		EnvID:      md.ID.Tenant.EnvID,
		FunctionID: md.ID.FunctionID,
	}
	shard, err := e.shards.Resolve(ctx, scope, nil)
	if err != nil {
		return fmt.Errorf("error resolving shard for loop wake: %w", err)
	}
	err = e.queue.RequeueByJobID(ctx, scope, shard.Name(), jobID, at)
	switch {
	case errors.Is(err, queue.ErrQueueItemNotFound):
		return e.queue.Enqueue(ctx, item, at, opts)
	case errors.Is(err, queue.ErrQueueItemAlreadyLeased):
		e.log.Warn("loop wake item is leased, leaving it in place",
			"job_id", jobID,
			"run_id", md.ID.RunID,
		)
		return nil
	}
	return err
}

// enqueueLoopTick puts one KindLoopTick item on the function's partition at
// now plus the function's tick.  the job id is tick:<instance>, so a
// pending tick is kept and a second one is not added.  a function without
// a tick enqueues nothing.
func (e *executor) enqueueLoopTick(ctx context.Context, md sv2.Metadata, fn inngest.Function, id loops.ID) error {
	if fn.Loop == nil || fn.Loop.Tick <= 0 {
		return nil
	}
	jobID := loopTimerJobID(id, loops.InputTick, "")
	at := e.now().Add(fn.Loop.Tick)
	item := loopTimerItem(md, id, jobID, queue.KindLoopTick, loops.InputTick, "")
	noIdempotency := time.Duration(0)
	err := e.queue.Enqueue(ctx, item, at, queue.EnqueueOpts{
		IdempotencyPeriod: &noIdempotency,
	})
	if errors.Is(err, queue.ErrQueueItemExists) {
		return nil
	}
	return err
}

// loopTickOnFinalize enqueues the tick for a loop run that finishes as
// failed or cancelled, so the instance keeps going.  a completed run
// enqueued its tick in the LoopReturn handler.  the function is loaded
// here, since the run holds nothing about the loop beyond its semaphore.
// the run is already final, so an error is logged and not returned.
func (e *executor) loopTickOnFinalize(ctx context.Context, md sv2.Metadata, status enums.StepStatus, events []json.RawMessage) {
	if status != enums.StepStatusFailed && status != enums.StepStatusCancelled {
		return
	}
	if e.loops == nil || e.queue == nil || e.fl == nil || len(events) == 0 {
		return
	}
	// a loop run always carries its keyed semaphore, so a run without one
	// is not a loop run and the function is not loaded for it.
	if len(md.Config.Semaphores) == 0 {
		return
	}
	ef, err := e.fl.LoadFunction(ctx, md.ID.Tenant.EnvID, md.ID.FunctionID)
	if err != nil {
		e.log.Warn("error loading function to tick a finished loop run",
			"error", err,
			"run_id", md.ID.RunID,
		)
		return
	}
	if ef == nil || ef.Function == nil || ef.Function.Loop == nil || ef.Function.Loop.Tick <= 0 {
		return
	}
	fn := *ef.Function
	in, err := loopInput(ctx, fn, events)
	if err != nil {
		e.log.Warn("loop run finished and its tick could not be enqueued",
			"error", err,
			"run_id", md.ID.RunID,
		)
		return
	}
	id := loops.NewID(domain.NewTenant(md.ID.Tenant.AccountID, md.ID.Tenant.EnvID), md.ID.FunctionID, in.Key)
	if err := e.enqueueLoopTick(ctx, md, fn, id); err != nil {
		e.log.Error("error enqueueing loop tick after a finished run",
			"error", err,
			"run_id", md.ID.RunID,
		)
	}
}

// loopTimerJobID returns the queue job id of the tick item or the wake
// item of an instance.  the service builds the same id to name the event
// that a run of the item starts.
func loopTimerJobID(id loops.ID, kind string, wakeID string) string {
	if kind == loops.InputWake {
		return fmt.Sprintf("wake:%s:%s", id.InstanceID(), wakeID)
	}
	return fmt.Sprintf("tick:%s", id.InstanceID())
}

// loopTimerItem builds the queue item for a tick or a wake.  the item has
// the function, account, env, and app ids and no run id, no semaphore, and
// no concurrency key.  the run is created when the item is handled.  the
// payload nonce is the enqueuing run id.  a pending tick that is kept and
// a wake that is moved keep their nonce.
func loopTimerItem(md sv2.Metadata, id loops.ID, jobID string, kind string, input string, inputID string) queue.Item {
	maxAttempts := consts.MaxRetries + 1
	return queue.Item{
		JobID:       &jobID,
		WorkspaceID: md.ID.Tenant.EnvID,
		Kind:        kind,
		Identifier: state.Identifier{
			AccountID:   md.ID.Tenant.AccountID,
			WorkspaceID: md.ID.Tenant.EnvID,
			AppID:       md.ID.Tenant.AppID,
			WorkflowID:  md.ID.FunctionID,
		},
		Attempt:     0,
		MaxAttempts: &maxAttempts,
		Payload: queue.PayloadLoop{
			Key:   id.Key,
			Kind:  input,
			ID:    inputID,
			Nonce: md.ID.RunID.String(),
		},
	}
}
