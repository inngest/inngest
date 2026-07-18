package sandbox

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const (
	testVPCID     = "11111111-1111-4111-8111-111111111111"
	testSandboxID = "22222222-2222-4222-8222-222222222222"
	testProcessID = "33333333-3333-4333-8333-333333333333"
	testTimestamp = "2026-07-28T00:00:00Z"
)

func canonicalOpcodeOpts(action Action) map[string]any {
	operation := map[string]any{"protocolVersion": ProtocolVersion, "action": string(action)}
	sandboxTarget := map[string]any{"sandboxId": testSandboxID}
	processTarget := map[string]any{"sandboxId": testSandboxID, "processId": testProcessID}
	switch action {
	case ActionCreate:
		operation["input"] = []any{map[string]any{"name": "eval_run-1", "vcpu": 2, "memoryMb": 2048}}
	case ActionList:
		operation["input"] = []any{map[string]any{"limit": 50}}
	case ActionGet:
		operation["input"] = []any{map[string]any{"sandboxId": testSandboxID}}
	case ActionExec:
		operation["target"] = sandboxTarget
		operation["input"] = []any{map[string]any{"command": []string{"/bin/true"}, "timeoutMs": 1000}}
	case ActionDestroy, ActionProcessList:
		operation["target"] = sandboxTarget
		operation["input"] = []any{}
	case ActionProcessStart:
		operation["target"] = sandboxTarget
		operation["input"] = []any{map[string]any{"command": []string{"/bin/sh", "-lc", "sleep 30"}, "environment": map[string]string{"WITH.DOT": "ok"}}}
	case ActionProcessGet:
		operation["target"] = processTarget
		operation["input"] = []any{}
	case ActionProcessSignal:
		operation["target"] = processTarget
		operation["input"] = []any{map[string]any{"signal": 15, "includeChildren": true}}
	case ActionProcessWait:
		operation["target"] = processTarget
		operation["input"] = []any{map[string]any{"timeoutMs": 5000}}
	case ActionProcessOutput:
		operation["target"] = processTarget
		operation["input"] = []any{map[string]any{"tailBytes": 123}}
	default:
		panic("unsupported test action")
	}
	return map[string]any{"type": "step.sandbox." + string(action), "sandbox": operation}
}

func parseCanonicalOpcode(t *testing.T, action Action) Operation {
	t.Helper()
	operation, err := ParseOpcodeOpts(canonicalOpcodeOpts(action))
	require.NoError(t, err)
	return operation
}

func TestParseCanonicalOpcodeOpts(t *testing.T) {
	actions := []Action{
		ActionCreate, ActionList, ActionGet, ActionExec, ActionDestroy,
		ActionProcessStart, ActionProcessList, ActionProcessGet,
		ActionProcessSignal, ActionProcessWait, ActionProcessOutput,
	}
	for _, action := range actions {
		t.Run(string(action), func(t *testing.T) {
			operation := parseCanonicalOpcode(t, action)
			require.Equal(t, action, operation.Action)
			input, err := operation.Input()
			require.NoError(t, err)
			require.NotEmpty(t, input)
		})
	}

	invalidEnv := canonicalOpcodeOpts(ActionProcessStart)
	invalidEnv["sandbox"].(map[string]any)["input"] = []any{map[string]any{
		"command": []string{"/bin/true"}, "environment": map[string]string{"A=B": "x"},
	}}
	_, err := ParseOpcodeOpts(invalidEnv)
	require.ErrorContains(t, err, "environment keys")

	relative := canonicalOpcodeOpts(ActionExec)
	relative["sandbox"].(map[string]any)["input"] = []any{map[string]any{
		"command": []string{"bin/true"}, "timeoutMs": 1000,
	}}
	_, err = ParseOpcodeOpts(relative)
	require.ErrorContains(t, err, "absolute executable")
}

func TestRESTProviderCanonicalLifecycle(t *testing.T) {
	resource := map[string]any{
		"id": testSandboxID, "name": "eval_run-1", "status": "RUNNING",
		"vpcId": testVPCID, "imageRef": "default",
		"resources": map[string]any{"vcpu": 2, "memoryMb": 2048},
		"createdAt": testTimestamp, "startedAt": testTimestamp, "endedAt": nil,
	}
	process := map[string]any{
		"id": testProcessID, "command": []string{"/bin/sh", "-lc", "sleep 30"},
		"pid": 42, "state": "RUNNING", "startedAt": testTimestamp,
	}
	writeData := func(w http.ResponseWriter, status int, data any, page ...any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		envelope := map[string]any{"data": data, "metadata": map[string]any{"fetchedAt": testTimestamp}}
		if len(page) > 0 {
			envelope["page"] = page[0]
		}
		require.NoError(t, json.NewEncoder(w).Encode(envelope))
	}
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		require.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
		require.Equal(t, "workspace-"+r.Header.Get("X-Test-Workspace"), r.Header.Get("X-Inngest-Env"))
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/sandboxes":
			writeData(w, http.StatusCreated, resource)
		case r.Method == http.MethodGet && r.URL.Path == "/v2/sandboxes":
			writeData(w, http.StatusOK, []any{resource}, map[string]any{"hasMore": false, "limit": 50})
		case r.Method == http.MethodGet && r.URL.Path == "/v2/sandboxes/"+testSandboxID:
			writeData(w, http.StatusOK, resource)
		case r.Method == http.MethodPost && r.URL.Path == "/v2/sandboxes/"+testSandboxID+"/exec":
			writeData(w, http.StatusOK, map[string]any{"stdout": "AP8=", "stderr": "", "encoding": "base64", "exitCode": 0})
		case r.Method == http.MethodDelete && r.URL.Path == "/v2/sandboxes/"+testSandboxID:
			writeData(w, http.StatusAccepted, map[string]any{
				"id": testSandboxID, "name": "eval_run-1", "status": "TERMINATING",
				"vpcId": testVPCID, "imageRef": "default",
				"resources": map[string]any{"vcpu": 2, "memoryMb": 2048},
				"createdAt": testTimestamp, "startedAt": testTimestamp, "endedAt": nil,
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v2/sandboxes/"+testSandboxID+"/processes":
			writeData(w, http.StatusCreated, process)
		case r.Method == http.MethodGet && r.URL.Path == "/v2/sandboxes/"+testSandboxID+"/processes":
			writeData(w, http.StatusOK, []any{process})
		case r.Method == http.MethodGet && r.URL.Path == "/v2/sandboxes/"+testSandboxID+"/processes/"+testProcessID:
			writeData(w, http.StatusOK, process)
		case r.Method == http.MethodPost && r.URL.Path == "/v2/sandboxes/"+testSandboxID+"/processes/"+testProcessID+"/signals":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/v2/sandboxes/"+testSandboxID+"/processes/"+testProcessID+"/wait":
			writeData(w, http.StatusOK, map[string]any{"id": testProcessID, "state": "KILLED", "terminationSignal": 15})
		case r.Method == http.MethodGet && r.URL.Path == "/v2/sandboxes/"+testSandboxID+"/processes/"+testProcessID+"/output":
			writeData(w, http.StatusOK, map[string]any{"chunks": []any{
				map[string]any{"stream": "STDOUT", "data": "AP8=", "encoding": "base64", "at": testTimestamp},
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	workspaceID := uuid.New()
	provider, err := NewRESTProvider(RESTConfig{
		BaseURL: server.URL,
		APIKey:  "test-api-key",
		EnvironmentResolver: func(_ context.Context, requested uuid.UUID) (string, error) {
			require.Equal(t, workspaceID, requested)
			return "workspace-" + requested.String(), nil
		},
		HTTPClient: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
			request.Header.Set("X-Test-Workspace", workspaceID.String())
			return http.DefaultTransport.RoundTrip(request)
		}).Client(),
	})
	require.NoError(t, err)

	for _, action := range []Action{
		ActionCreate, ActionList, ActionGet, ActionExec, ActionDestroy,
		ActionProcessStart, ActionProcessList, ActionProcessGet,
		ActionProcessSignal, ActionProcessWait, ActionProcessOutput,
	} {
		result, operationErr := provider.Execute(context.Background(), workspaceID, parseCanonicalOpcode(t, action))
		require.Nil(t, operationErr, action)
		switch action {
		case ActionExec:
			require.Equal(t, []byte{0, 255}, result.Command.Stdout)
		case ActionProcessWait:
			require.Equal(t, "KILLED", result.Wait.State)
		case ActionProcessOutput:
			require.Equal(t, []byte{0, 255}, result.Output.Chunks[0].Data)
		}
	}
	require.Equal(t, int32(11), calls.Load())
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func (f roundTripperFunc) Client() *http.Client {
	return &http.Client{Transport: f}
}

type countingProvider struct {
	calls       atomic.Int32
	firstResult *OperationError
}

func (p *countingProvider) Execute(_ context.Context, _ uuid.UUID, operation Operation) (Result, *OperationError) {
	p.calls.Add(1)
	if p.firstResult != nil {
		result := p.firstResult
		p.firstResult = nil
		return Result{}, result
	}
	return Result{Sandbox: &Resource{
		Kind: "inngest/sandbox", Version: ProtocolVersion,
		ID: testSandboxID, Name: "eval_run-1", Status: "RUNNING",
		VPCID: testVPCID, ImageRef: "default",
		Resources: Resources{VCPU: 2, MemoryMB: 2048}, CreatedAt: testTimestamp,
	}}, nil
}

func TestDispatcherPersistsMutationResultAcrossInstances(t *testing.T) {
	store := NewMemoryDispatchStore()
	provider := &countingProvider{}
	key := DispatchKey{AccountID: uuid.New(), WorkspaceID: uuid.New(), RunID: "run", StepID: "step"}
	operation := parseCanonicalOpcode(t, ActionCreate)

	first, firstErr := NewDispatcher(provider, store).Execute(context.Background(), key, operation)
	second, secondErr := NewDispatcher(provider, store).Execute(context.Background(), key, operation)
	require.Nil(t, firstErr)
	require.Nil(t, secondErr)
	require.Equal(t, first.Sandbox.ID, second.Sandbox.ID)
	require.Equal(t, int32(1), provider.calls.Load())

	changed := operation
	changed.Create = &CreateInput{Name: "different", VCPU: 2, MemoryMB: 2048}
	_, conflict := NewDispatcher(provider, store).Execute(context.Background(), key, changed)
	require.Equal(t, "idempotency_conflict", conflict.Code)
	require.Equal(t, int32(1), provider.calls.Load())
}

func TestDispatcherDoesNotRedispatchUncertainIntent(t *testing.T) {
	store := NewMemoryDispatchStore()
	provider := &countingProvider{}
	key := DispatchKey{AccountID: uuid.New(), WorkspaceID: uuid.New(), RunID: "run", StepID: "step"}
	operation := parseCanonicalOpcode(t, ActionProcessSignal)
	digest, err := operation.Digest()
	require.NoError(t, err)
	_, claimed, err := store.Claim(context.Background(), key, DispatchRecord{
		Digest: fmtDigest(digest), Action: operation.Action, State: DispatchStateDispatching,
	})
	require.NoError(t, err)
	require.True(t, claimed)

	_, operationErr := NewDispatcher(provider, store).Execute(context.Background(), key, operation)
	require.Equal(t, "operation_ambiguous", operationErr.Code)
	require.True(t, operationErr.Ambiguous)
	require.Zero(t, provider.calls.Load())
}

func TestDispatcherReleasesConfirmedPredispatchFailure(t *testing.T) {
	provider := &countingProvider{firstResult: &OperationError{
		Code: "compute_unavailable", Message: "not dispatched", Retryable: true, Details: []ErrorDetail{},
	}}
	dispatcher := NewDispatcher(provider, NewMemoryDispatchStore())
	key := DispatchKey{AccountID: uuid.New(), WorkspaceID: uuid.New(), RunID: "run", StepID: "step"}
	operation := parseCanonicalOpcode(t, ActionProcessStart)

	_, firstErr := dispatcher.Execute(context.Background(), key, operation)
	require.True(t, firstErr.Retryable)
	_, secondErr := dispatcher.Execute(context.Background(), key, operation)
	require.Nil(t, secondErr)
	require.Equal(t, int32(2), provider.calls.Load())
}

func fmtDigest(digest [32]byte) string {
	return fmt.Sprintf("%x", digest[:])
}
