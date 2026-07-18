package executor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/util/sandbox"
)

func (e *executor) handleGeneratorSandbox(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge, group OpcodeGroup) error {
	start := e.now()
	gen.Timing.A = start.UnixNano()

	operation, err := gen.SandboxOpts()
	if err != nil {
		return e.saveSandboxError(ctx, runCtx, gen, edge, group, "", &sandbox.OperationError{
			Code:    "invalid_request",
			Message: fmt.Sprintf("Invalid sandbox operation: %v", err),
			Details: []sandbox.ErrorDetail{},
		})
	}
	if e.sandboxDispatcher == nil {
		return e.saveSandboxError(ctx, runCtx, gen, edge, group, operation.Action, &sandbox.OperationError{
			Code:    "not_implemented",
			Message: "Sandbox execution is not configured on this server",
			Details: []sandbox.ErrorDetail{},
		})
	}

	md := runCtx.Metadata()
	result, operationErr := e.sandboxDispatcher.Execute(ctx, sandbox.DispatchKey{
		AccountID:   md.ID.Tenant.AccountID,
		WorkspaceID: md.ID.Tenant.EnvID,
		RunID:       md.ID.RunID.String(),
		StepID:      gen.ID,
	}, operation)
	gen.Timing.B = e.now().Sub(start).Nanoseconds()
	if operationErr != nil {
		return e.saveSandboxError(ctx, runCtx, gen, edge, group, operation.Action, operationErr)
	}

	protocolResult, err := sandbox.MarshalResult(operation.Action, result)
	if err != nil {
		return e.saveSandboxError(ctx, runCtx, gen, edge, group, operation.Action, &sandbox.OperationError{
			Code:      "internal_error",
			Message:   "Sandbox provider returned an invalid result",
			Ambiguous: operation.Action.RequiresDispatchFence(),
			Details:   []sandbox.ErrorDetail{{"reason": err.Error()}},
		})
	}
	output, err := json.Marshal(map[string]json.RawMessage{execution.StateDataKey: protocolResult})
	if err != nil {
		return err
	}
	if len(output) > consts.MaxStepOutputSize {
		return e.saveSandboxError(ctx, runCtx, gen, edge, group, operation.Action, &sandbox.OperationError{
			Code:      "internal_error",
			Message:   "Sandbox result exceeds the Inngest step output limit",
			Ambiguous: operation.Action.RequiresDispatchFence(),
			Details:   []sandbox.ErrorDetail{},
		})
	}
	runCtx.UpdateOpcodeOutput(&gen, output)
	return e.saveSandboxStep(ctx, runCtx, gen, edge, group, output)
}

func (e *executor) saveSandboxError(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge, group OpcodeGroup, action sandbox.Action, operationErr *sandbox.OperationError) error {
	var cause json.RawMessage
	if action.Valid() {
		var err error
		cause, err = sandbox.MarshalError(action, operationErr)
		if err != nil {
			return err
		}
	}
	userErr := state.UserError{Name: "SandboxError", Message: operationErr.Message, NoRetry: !operationErr.Retryable, Cause: cause}
	userErrJSON, err := json.Marshal(userErr)
	if err != nil {
		return err
	}
	output, err := json.Marshal(map[string]json.RawMessage{execution.StateErrorKey: userErrJSON})
	if err != nil {
		return err
	}
	gen.Error = &userErr
	runCtx.UpdateOpcodeError(&gen, userErr)
	return e.saveSandboxStep(ctx, runCtx, gen, edge, group, output)
}

func (e *executor) saveSandboxStep(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge, group OpcodeGroup, output []byte) error {
	if len(output) > consts.MaxStepOutputSize {
		return state.ErrStepOutputTooLarge
	}

	status := enums.StepStatusCompleted
	if gen.Error != nil {
		status = enums.StepStatusFailed
	}
	stepOutput := string(output)
	e.emitStepSpan(ctx, runCtx, &gen, nil, meta.NewAttrSet(
		meta.Attr(meta.Attrs.DynamicStatus, &status),
		meta.Attr(meta.Attrs.StepOutput, &stepOutput),
	))

	return e.saveStepAndEnqueueDiscovery(ctx, runCtx, gen, edge, output, &group.ParallelCoalesceKey)
}
