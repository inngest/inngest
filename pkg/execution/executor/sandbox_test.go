package executor

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/util/sandbox"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

const (
	executorSandboxID = "22222222-2222-4222-8222-222222222222"
	executorProcessID = "33333333-3333-4333-8333-333333333333"
)

type sandboxTestProvider struct {
	calls atomic.Int32
	err   *sandbox.OperationError
}

func (p *sandboxTestProvider) Execute(_ context.Context, _ uuid.UUID, operation sandbox.Operation) (sandbox.Result, *sandbox.OperationError) {
	p.calls.Add(1)
	if p.err != nil {
		return sandbox.Result{}, p.err
	}
	resource := &sandbox.Resource{
		Kind: "inngest/sandbox", Version: 1,
		ID: executorSandboxID, VPCID: "11111111-1111-4111-8111-111111111111",
		Name: "eval_run-1", Status: "RUNNING", ImageRef: "default",
		Resources: sandbox.Resources{VCPU: 2, MemoryMB: 2048},
		CreatedAt: "2026-07-28T00:00:00Z",
	}
	process := &sandbox.ProcessResource{
		Kind: "inngest/sandbox.process", Version: 1,
		SandboxID: executorSandboxID, ID: executorProcessID,
		Command: []string{"/bin/sh", "-lc", "sleep 30"}, State: "RUNNING",
	}
	switch operation.Action {
	case sandbox.ActionCreate, sandbox.ActionGet:
		return sandbox.Result{Sandbox: resource}, nil
	case sandbox.ActionList:
		return sandbox.Result{List: &sandbox.ListResult{
			Sandboxes: []*sandbox.Resource{resource},
			Page:      sandbox.Page{Limit: 50}, FetchedAt: "2026-07-28T00:00:00Z",
		}}, nil
	case sandbox.ActionExec:
		return sandbox.Result{Command: &sandbox.CommandResult{
			Stdout: []byte("ok\n"), Stderr: []byte{}, Encoding: "base64", ExitCode: 0,
		}}, nil
	case sandbox.ActionDestroy:
		resource.Status = "TERMINATING"
		return sandbox.Result{Destroy: &sandbox.DestroyResult{
			Status: "TERMINATING", Sandbox: resource,
		}}, nil
	case sandbox.ActionProcessStart, sandbox.ActionProcessGet:
		return sandbox.Result{Process: process}, nil
	case sandbox.ActionProcessList:
		return sandbox.Result{Processes: []*sandbox.ProcessResource{process}}, nil
	case sandbox.ActionProcessSignal:
		return sandbox.Result{SignalDone: true}, nil
	case sandbox.ActionProcessWait:
		signal := int32(15)
		return sandbox.Result{Wait: &sandbox.WaitProcessResult{
			Kind: "inngest/sandbox.process", Version: 1,
			SandboxID: executorSandboxID, ID: executorProcessID,
			State: "KILLED", TerminationSignal: &signal,
		}}, nil
	case sandbox.ActionProcessOutput:
		return sandbox.Result{Output: &sandbox.OutputResult{Chunks: []sandbox.OutputChunk{
			{Stream: "STDOUT", Data: []byte{0, 255}, Encoding: "base64"},
		}}}, nil
	default:
		panic("unexpected sandbox action")
	}
}

type sandboxRunService struct {
	sv2.RunService
	saves  atomic.Int32
	output []byte
	fail   bool
}

func (s *sandboxRunService) SaveStep(_ context.Context, _ sv2.ID, _ string, output []byte) (bool, error) {
	if s.fail && s.saves.Add(1) == 1 {
		return false, errors.New("temporary state failure")
	}
	s.output = append([]byte(nil), output...)
	return true, nil
}

func sandboxOpcodeOpts(action sandbox.Action) map[string]any {
	operation := map[string]any{"protocolVersion": 1, "action": string(action)}
	sandboxTarget := map[string]any{"sandboxId": executorSandboxID}
	processTarget := map[string]any{"sandboxId": executorSandboxID, "processId": executorProcessID}
	switch action {
	case sandbox.ActionCreate:
		operation["input"] = []any{map[string]any{"name": "eval_run-1", "vcpu": 2, "memoryMb": 2048}}
	case sandbox.ActionList:
		operation["input"] = []any{map[string]any{"limit": 50}}
	case sandbox.ActionGet:
		operation["input"] = []any{map[string]any{"sandboxId": executorSandboxID}}
	case sandbox.ActionExec:
		operation["target"] = sandboxTarget
		operation["input"] = []any{map[string]any{"command": []string{"/bin/true"}, "timeoutMs": 1000}}
	case sandbox.ActionDestroy, sandbox.ActionProcessList:
		operation["target"] = sandboxTarget
		operation["input"] = []any{}
	case sandbox.ActionProcessStart:
		operation["target"] = sandboxTarget
		operation["input"] = []any{map[string]any{"command": []string{"/bin/sh", "-lc", "sleep 30"}}}
	case sandbox.ActionProcessGet:
		operation["target"] = processTarget
		operation["input"] = []any{}
	case sandbox.ActionProcessSignal:
		operation["target"] = processTarget
		operation["input"] = []any{map[string]any{"signal": 15, "includeChildren": false}}
	case sandbox.ActionProcessWait:
		operation["target"] = processTarget
		operation["input"] = []any{map[string]any{"timeoutMs": 5000}}
	case sandbox.ActionProcessOutput:
		operation["target"] = processTarget
		operation["input"] = []any{map[string]any{"tailBytes": 123}}
	default:
		panic("unexpected sandbox action")
	}
	return map[string]any{"type": "step.sandbox." + string(action), "sandbox": operation}
}

func newSandboxRunContext() *mockRunContext {
	return &mockRunContext{md: sv2.Metadata{ID: sv2.ID{
		RunID: ulid.Make(), FunctionID: uuid.New(),
		Tenant: sv2.Tenant{AccountID: uuid.New(), EnvID: uuid.New(), AppID: uuid.New()},
	}, Config: *sv2.InitConfig(&sv2.Config{})}}
}

func TestWithSandboxProviderConfiguresDispatcher(t *testing.T) {
	exec := &executor{}
	require.NoError(t, WithSandboxProvider(&sandboxTestProvider{}, sandbox.NewMemoryDispatchStore())(exec))
	require.NotNil(t, exec.sandboxDispatcher)
}

func TestSandboxDispatchPersistsAcrossExecutorRetry(t *testing.T) {
	for _, action := range []sandbox.Action{
		sandbox.ActionCreate,
		sandbox.ActionExec,
		sandbox.ActionDestroy,
		sandbox.ActionProcessStart,
		sandbox.ActionProcessSignal,
	} {
		t.Run(string(action), func(t *testing.T) {
			provider := &sandboxTestProvider{}
			store := sandbox.NewMemoryDispatchStore()
			runService := &sandboxRunService{fail: true}
			runCtx := newSandboxRunContext()
			gen := state.GeneratorOpcode{Op: enums.OpcodeSandbox, ID: "hashed-step-id", Opts: sandboxOpcodeOpts(action)}

			first := &executor{
				smv2: runService, tracerProvider: tracing.NewNoopTracerProvider(),
				sandboxDispatcher: sandbox.NewDispatcher(provider, store),
			}
			err := first.handleGeneratorSandbox(context.Background(), runCtx, gen, queue.PayloadEdge{}, OpcodeGroup{})
			require.ErrorContains(t, err, "temporary state failure")

			restarted := &executor{
				smv2: runService, tracerProvider: tracing.NewNoopTracerProvider(),
				sandboxDispatcher: sandbox.NewDispatcher(provider, store),
			}
			require.NoError(t, restarted.handleGeneratorSandbox(context.Background(), runCtx, gen, queue.PayloadEdge{}, OpcodeGroup{}))
			require.Equal(t, int32(1), provider.calls.Load())

			var output struct {
				Data map[string]any `json:"data"`
			}
			require.NoError(t, json.Unmarshal(runService.output, &output))
			require.Equal(t, string(action), output.Data["action"])
		})
	}
}

func TestSandboxAllActionsEncodeCanonicalResults(t *testing.T) {
	actions := []sandbox.Action{
		sandbox.ActionCreate, sandbox.ActionList, sandbox.ActionGet,
		sandbox.ActionExec, sandbox.ActionDestroy, sandbox.ActionProcessStart,
		sandbox.ActionProcessList, sandbox.ActionProcessGet,
		sandbox.ActionProcessSignal, sandbox.ActionProcessWait,
		sandbox.ActionProcessOutput,
	}
	for _, action := range actions {
		t.Run(string(action), func(t *testing.T) {
			runService := &sandboxRunService{}
			runCtx := newSandboxRunContext()
			exec := &executor{
				smv2: runService, tracerProvider: tracing.NewNoopTracerProvider(),
				sandboxDispatcher: sandbox.NewDispatcher(&sandboxTestProvider{}),
			}
			gen := state.GeneratorOpcode{Op: enums.OpcodeSandbox, ID: "step-" + string(action), Opts: sandboxOpcodeOpts(action)}
			require.NoError(t, exec.handleGeneratorSandbox(context.Background(), runCtx, gen, queue.PayloadEdge{}, OpcodeGroup{}))

			var output struct {
				Data map[string]any `json:"data"`
			}
			require.NoError(t, json.Unmarshal(runService.output, &output))
			require.EqualValues(t, 1, output.Data["protocolVersion"])
			require.Equal(t, string(action), output.Data["action"])
		})
	}
}

func TestSandboxEmitsVisibleStepSpan(t *testing.T) {
	tests := []struct {
		name           string
		providerError  *sandbox.OperationError
		expectedStatus enums.StepStatus
	}{
		{name: "success", expectedStatus: enums.StepStatusCompleted},
		{
			name: "error",
			providerError: &sandbox.OperationError{
				Code: "sandbox_unavailable", Message: "sandbox unavailable",
			},
			expectedStatus: enums.StepStatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := newRecordingTracerProvider()
			runService := &sandboxRunService{}
			exec := &executor{
				smv2:           runService,
				tracerProvider: recorder,
				sandboxDispatcher: sandbox.NewDispatcher(&sandboxTestProvider{
					err: tt.providerError,
				}),
			}
			gen := state.GeneratorOpcode{
				Op: enums.OpcodeSandbox, ID: "sandbox-step",
				Name: "create sandbox", Opts: sandboxOpcodeOpts(sandbox.ActionCreate),
			}

			require.NoError(t, exec.handleGeneratorSandbox(
				context.Background(), newSandboxRunContext(), gen,
				queue.PayloadEdge{}, OpcodeGroup{},
			))
			require.Len(t, recorder.createCalls, 1)

			call := recorder.createCalls[0]
			require.Equal(t, meta.SpanNameStep, call.name)

			stepOp, ok := call.opts.Attributes.Get(meta.Attrs.StepOp.Key()).(*enums.Opcode)
			require.True(t, ok)
			require.Equal(t, enums.OpcodeSandbox, *stepOp)

			stepType, ok := call.opts.Attributes.Get(meta.Attrs.StepType.Key()).(*enums.StepType)
			require.True(t, ok)
			require.Equal(t, enums.StepTypeStepSandboxCreate, *stepType)

			status, ok := call.opts.Attributes.Get(meta.Attrs.DynamicStatus.Key()).(*enums.StepStatus)
			require.True(t, ok)
			require.Equal(t, tt.expectedStatus, *status)

			output, ok := call.opts.Attributes.Get(meta.Attrs.StepOutput.Key()).(*string)
			require.True(t, ok)
			require.JSONEq(t, string(runService.output), *output)
		})
	}
}

func TestSandboxEnqueuesDiscoveryOnDuplicateSave(t *testing.T) {
	provider := &sandboxTestProvider{}
	q := &stubQueue{}
	runCtx := newSandboxRunContext()
	exec := &executor{
		smv2:              &stubRunService{saveStepErr: state.ErrDuplicateResponse},
		queue:             q,
		log:               logger.From(context.Background()),
		tracerProvider:    tracing.NewNoopTracerProvider(),
		sandboxDispatcher: sandbox.NewDispatcher(provider),
	}
	gen := state.GeneratorOpcode{
		Op: enums.OpcodeSandbox, ID: "hashed-step-id", Opts: sandboxOpcodeOpts(sandbox.ActionProcessStart),
	}

	err := exec.handleGeneratorSandbox(
		context.Background(),
		runCtx,
		gen,
		queue.PayloadEdge{Edge: inngest.Edge{Incoming: "step"}},
		OpcodeGroup{},
	)
	require.NoError(t, err)
	require.Equal(t, int32(1), provider.calls.Load())
	require.Len(t, q.enqueued, 1)
}
