package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/google/uuid"
)

const maxResponseBytes = 5 << 20

type EnvironmentResolver func(ctx context.Context, workspaceID uuid.UUID) (string, error)

type RESTConfig struct {
	BaseURL             string
	APIKey              string
	Environment         string
	EnvironmentResolver EnvironmentResolver
	HTTPClient          *http.Client
}

type RESTProvider struct {
	baseURL             string
	apiKey              string
	environment         string
	environmentResolver EnvironmentResolver
	httpClient          *http.Client
}

func NewRESTProvider(config RESTConfig) (*RESTProvider, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("sandbox REST base URL must be absolute")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("sandbox REST base URL must not contain user info, query, or fragment")
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname())) {
		return nil, errors.New("sandbox REST base URL must use HTTPS unless it is loopback")
	}
	if strings.TrimSpace(config.APIKey) == "" {
		return nil, errors.New("sandbox REST API key is required")
	}
	if strings.TrimSpace(config.Environment) != "" && config.EnvironmentResolver != nil {
		return nil, errors.New("configure either a static sandbox environment or an environment resolver")
	}
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	clientCopy := *client
	clientCopy.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &RESTProvider{
		baseURL:             baseURL,
		apiKey:              config.APIKey,
		environment:         strings.TrimSpace(config.Environment),
		environmentResolver: config.EnvironmentResolver,
		httpClient:          &clientCopy,
	}, nil
}

func (p *RESTProvider) Execute(ctx context.Context, workspaceID uuid.UUID, operation Operation) (Result, *OperationError) {
	environment := p.environment
	if p.environmentResolver != nil {
		var err error
		environment, err = p.environmentResolver(ctx, workspaceID)
		if err != nil {
			result := operationError("compute_unavailable", "Could not resolve sandbox workspace")
			result.Retryable = true
			result.Details = []ErrorDetail{{"reason": err.Error()}}
			return Result{}, result
		}
	}
	switch operation.Action {
	case ActionCreate:
		return p.create(ctx, environment, *operation.Create)
	case ActionList:
		return p.list(ctx, environment, *operation.List)
	case ActionGet:
		return p.get(ctx, environment, operation.Get.SandboxID)
	case ActionExec:
		return p.exec(ctx, environment, operation.Target.SandboxID, *operation.Exec)
	case ActionDestroy:
		return p.destroy(ctx, environment, operation.Target.SandboxID)
	case ActionProcessStart:
		return p.startProcess(ctx, environment, operation.Target.SandboxID, *operation.ProcessSpec)
	case ActionProcessList:
		return p.listProcesses(ctx, environment, operation.Target.SandboxID)
	case ActionProcessGet:
		return p.getProcess(ctx, environment, *operation.ProcessTarget)
	case ActionProcessSignal:
		return p.signalProcess(ctx, environment, *operation.ProcessTarget, *operation.Signal)
	case ActionProcessWait:
		return p.waitProcess(ctx, environment, *operation.ProcessTarget, *operation.Wait)
	case ActionProcessOutput:
		return p.getProcessOutput(ctx, environment, *operation.ProcessTarget, *operation.Output)
	default:
		return Result{}, invalidOperation(fmt.Errorf("unsupported action %q", operation.Action))
	}
}

func (p *RESTProvider) create(ctx context.Context, environment string, input CreateInput) (Result, *OperationError) {
	envelope, operationErr := p.request(ctx, environment, http.MethodPost, "/v2/sandboxes", input, []int{http.StatusCreated, http.StatusAccepted}, true)
	if operationErr != nil {
		return Result{}, operationErr
	}
	resource, err := parseResource(envelope.Data)
	if err != nil {
		return Result{}, malformedResponse(ActionCreate, "", "", err)
	}
	if (envelope.Status == http.StatusCreated && resource.Status != "RUNNING") ||
		(envelope.Status == http.StatusAccepted && resource.Status != "STARTING") {
		return Result{}, malformedResponse(
			ActionCreate,
			resource.ID,
			"",
			fmt.Errorf("HTTP %d returned sandbox status %s", envelope.Status, resource.Status),
		)
	}
	return Result{Sandbox: resource}, nil
}

func (p *RESTProvider) list(ctx context.Context, environment string, input ListInput) (Result, *OperationError) {
	query := url.Values{"limit": []string{fmt.Sprintf("%d", input.Limit)}}
	if input.Cursor != "" {
		query.Set("cursor", input.Cursor)
	}
	envelope, operationErr := p.request(ctx, environment, http.MethodGet, "/v2/sandboxes?"+query.Encode(), nil, []int{http.StatusOK}, false)
	if operationErr != nil {
		return Result{}, operationErr
	}
	var resourcesRaw []json.RawMessage
	if err := decodeStrict(envelope.Data, &resourcesRaw); err != nil {
		return Result{}, malformedResponse(ActionList, "", "", err)
	}
	resources := make([]*Resource, 0, len(resourcesRaw))
	for _, raw := range resourcesRaw {
		resource, err := parseResource(raw)
		if err != nil {
			return Result{}, malformedResponse(ActionList, "", "", err)
		}
		resources = append(resources, resource)
	}
	var page Page
	if err := decodeStrict(envelope.Page, &page); err != nil || page.Limit < 1 || page.Limit > 250 {
		if err == nil {
			err = errors.New("invalid page")
		}
		return Result{}, malformedResponse(ActionList, "", "", err)
	}
	var metadata struct {
		FetchedAt string `json:"fetchedAt"`
	}
	if err := json.Unmarshal(envelope.Metadata, &metadata); err != nil || metadata.FetchedAt == "" {
		if err == nil {
			err = errors.New("missing fetchedAt")
		}
		return Result{}, malformedResponse(ActionList, "", "", err)
	}
	return Result{List: &ListResult{Sandboxes: resources, Page: page, FetchedAt: metadata.FetchedAt}}, nil
}

func (p *RESTProvider) get(ctx context.Context, environment, sandboxID string) (Result, *OperationError) {
	envelope, operationErr := p.request(ctx, environment, http.MethodGet, sandboxPath(sandboxID), nil, []int{http.StatusOK}, false)
	if operationErr != nil {
		if operationErr.Status == http.StatusNotFound {
			return Result{}, nil
		}
		operationErr.SandboxID = sandboxID
		return Result{}, operationErr
	}
	resource, err := parseResource(envelope.Data)
	if err != nil {
		return Result{}, malformedResponse(ActionGet, sandboxID, "", err)
	}
	return Result{Sandbox: resource}, nil
}

func (p *RESTProvider) exec(ctx context.Context, environment, sandboxID string, input ExecInput) (Result, *OperationError) {
	envelope, operationErr := p.request(ctx, environment, http.MethodPost, sandboxPath(sandboxID)+"/exec", struct {
		Command     []string          `json:"command"`
		Environment map[string]string `json:"environment,omitempty"`
		CWD         string            `json:"cwd,omitempty"`
		Timeout     string            `json:"timeout"`
	}{
		Command:     input.Command,
		Environment: input.Environment,
		CWD:         input.CWD,
		Timeout:     fmt.Sprintf("%dms", input.TimeoutMS),
	}, []int{http.StatusOK}, true)
	if operationErr != nil {
		operationErr.SandboxID = sandboxID
		return Result{}, operationErr
	}
	var result CommandResult
	if err := decodeStrict(envelope.Data, &result); err != nil || result.Encoding != "base64" {
		if err == nil {
			err = errors.New("exec encoding must be base64")
		}
		return Result{}, malformedResponse(ActionExec, sandboxID, "", err)
	}
	return Result{Command: &result}, nil
}

func (p *RESTProvider) destroy(ctx context.Context, environment, sandboxID string) (Result, *OperationError) {
	envelope, operationErr := p.request(ctx, environment, http.MethodDelete, sandboxPath(sandboxID), nil, []int{http.StatusAccepted, http.StatusNoContent}, true)
	if operationErr != nil {
		if operationErr.Status == http.StatusNotFound {
			return Result{Destroy: &DestroyResult{Status: "TERMINATED"}}, nil
		}
		operationErr.SandboxID = sandboxID
		return Result{}, operationErr
	}
	if envelope.Status == http.StatusNoContent {
		return Result{Destroy: &DestroyResult{Status: "TERMINATED"}}, nil
	}
	resource, err := parseResource(envelope.Data)
	if err != nil {
		return Result{}, malformedResponse(ActionDestroy, sandboxID, "", err)
	}
	if resource.Status != "TERMINATING" {
		return Result{}, malformedResponse(
			ActionDestroy,
			sandboxID,
			"",
			fmt.Errorf("destroy returned sandbox status %s", resource.Status),
		)
	}
	return Result{Destroy: &DestroyResult{Status: "TERMINATING", Sandbox: resource}}, nil
}

func (p *RESTProvider) startProcess(ctx context.Context, environment, sandboxID string, input ProcessSpecInput) (Result, *OperationError) {
	envelope, operationErr := p.request(ctx, environment, http.MethodPost, sandboxPath(sandboxID)+"/processes", input, []int{http.StatusCreated}, true)
	if operationErr != nil {
		operationErr.SandboxID = sandboxID
		return Result{}, operationErr
	}
	process, err := parseProcess(envelope.Data, sandboxID, true)
	if err != nil {
		return Result{}, malformedResponse(ActionProcessStart, sandboxID, "", err)
	}
	if process.State != "RUNNING" || process.PID == nil {
		return Result{}, malformedResponse(
			ActionProcessStart,
			sandboxID,
			process.ID,
			errors.New("process Start did not return a RUNNING process with a PID"),
		)
	}
	return Result{Process: process}, nil
}

func (p *RESTProvider) listProcesses(ctx context.Context, environment, sandboxID string) (Result, *OperationError) {
	envelope, operationErr := p.request(ctx, environment, http.MethodGet, sandboxPath(sandboxID)+"/processes", nil, []int{http.StatusOK}, false)
	if operationErr != nil {
		operationErr.SandboxID = sandboxID
		return Result{}, operationErr
	}
	var rawProcesses []json.RawMessage
	if err := decodeStrict(envelope.Data, &rawProcesses); err != nil {
		return Result{}, malformedResponse(ActionProcessList, sandboxID, "", err)
	}
	processes := make([]*ProcessResource, 0, len(rawProcesses))
	for _, raw := range rawProcesses {
		process, err := parseProcess(raw, sandboxID, true)
		if err != nil {
			return Result{}, malformedResponse(ActionProcessList, sandboxID, "", err)
		}
		processes = append(processes, process)
	}
	sort.Slice(processes, func(i, j int) bool { return processes[i].ID < processes[j].ID })
	return Result{Processes: processes}, nil
}

func (p *RESTProvider) getProcess(ctx context.Context, environment string, target ProcessTarget) (Result, *OperationError) {
	envelope, operationErr := p.request(ctx, environment, http.MethodGet, processPath(target), nil, []int{http.StatusOK}, false)
	if operationErr != nil {
		if operationErr.Status == http.StatusNotFound {
			return Result{}, nil
		}
		setErrorTarget(operationErr, target)
		return Result{}, operationErr
	}
	process, err := parseProcess(envelope.Data, target.SandboxID, true)
	if err != nil {
		return Result{}, malformedResponse(ActionProcessGet, target.SandboxID, target.ProcessID, err)
	}
	return Result{Process: process}, nil
}

func (p *RESTProvider) signalProcess(ctx context.Context, environment string, target ProcessTarget, input SignalInput) (Result, *OperationError) {
	_, operationErr := p.request(ctx, environment, http.MethodPost, processPath(target)+"/signals", input, []int{http.StatusNoContent}, true)
	if operationErr != nil {
		setErrorTarget(operationErr, target)
		return Result{}, operationErr
	}
	return Result{SignalDone: true}, nil
}

func (p *RESTProvider) waitProcess(ctx context.Context, environment string, target ProcessTarget, input WaitInput) (Result, *OperationError) {
	query := url.Values{"timeout": []string{fmt.Sprintf("%dms", input.TimeoutMS)}}
	envelope, operationErr := p.request(ctx, environment, http.MethodPost, processPath(target)+"/wait?"+query.Encode(), nil, []int{http.StatusOK}, false)
	if operationErr != nil {
		setErrorTarget(operationErr, target)
		return Result{}, operationErr
	}
	var source struct {
		ID                string `json:"id"`
		State             string `json:"state"`
		ExitCode          *int32 `json:"exitCode,omitempty"`
		TerminationSignal *int32 `json:"terminationSignal,omitempty"`
	}
	if err := decodeStrict(envelope.Data, &source); err != nil {
		return Result{}, malformedResponse(ActionProcessWait, target.SandboxID, target.ProcessID, err)
	}
	if source.ID != target.ProcessID {
		return Result{}, malformedResponse(ActionProcessWait, target.SandboxID, target.ProcessID, errors.New("wait returned a different process ID"))
	}
	if err := validateTerminalProcess(source.State, source.ExitCode, source.TerminationSignal); err != nil {
		return Result{}, malformedResponse(ActionProcessWait, target.SandboxID, target.ProcessID, err)
	}
	return Result{Wait: &WaitProcessResult{
		Kind:              "inngest/sandbox.process",
		Version:           ProtocolVersion,
		SandboxID:         target.SandboxID,
		ID:                source.ID,
		State:             source.State,
		ExitCode:          source.ExitCode,
		TerminationSignal: source.TerminationSignal,
	}}, nil
}

func (p *RESTProvider) getProcessOutput(ctx context.Context, environment string, target ProcessTarget, input OutputInput) (Result, *OperationError) {
	query := url.Values{"tailBytes": []string{fmt.Sprintf("%d", input.TailBytes)}}
	envelope, operationErr := p.request(ctx, environment, http.MethodGet, processPath(target)+"/output?"+query.Encode(), nil, []int{http.StatusOK}, false)
	if operationErr != nil {
		setErrorTarget(operationErr, target)
		return Result{}, operationErr
	}
	var output OutputResult
	if err := decodeStrict(envelope.Data, &output); err != nil {
		return Result{}, malformedResponse(ActionProcessOutput, target.SandboxID, target.ProcessID, err)
	}
	for _, chunk := range output.Chunks {
		if (chunk.Stream != "STDOUT" && chunk.Stream != "STDERR") || chunk.Encoding != "base64" {
			return Result{}, malformedResponse(ActionProcessOutput, target.SandboxID, target.ProcessID, errors.New("invalid output chunk"))
		}
	}
	return Result{Output: &output}, nil
}

type apiEnvelope struct {
	Status   int
	Data     json.RawMessage `json:"data"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
	Page     json.RawMessage `json:"page,omitempty"`
}

type apiErrorEnvelope struct {
	Errors []ErrorDetail `json:"errors"`
}

func (p *RESTProvider) request(ctx context.Context, environment, method, path string, body any, expectedStatuses []int, mutation bool) (apiEnvelope, *OperationError) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return apiEnvelope{}, invalidOperation(err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, reader)
	if err != nil {
		return apiEnvelope{}, invalidOperation(err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	if environment != "" {
		req.Header.Set("X-Inngest-Env", environment)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := p.httpClient.Do(req)
	if err != nil {
		return apiEnvelope{}, transportError(method, mutation, err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil || len(raw) > maxResponseBytes {
		if err == nil {
			err = errors.New("response exceeds 5 MiB")
		}
		return apiEnvelope{}, transportError(method, mutation, err)
	}
	expected := false
	for _, status := range expectedStatuses {
		expected = expected || response.StatusCode == status
	}
	requestID := response.Header.Get("X-Request-ID")
	if !expected {
		var envelope apiErrorEnvelope
		_ = json.Unmarshal(raw, &envelope)
		code := "internal_error"
		message := fmt.Sprintf("Sandbox API returned HTTP %d", response.StatusCode)
		if len(envelope.Errors) > 0 {
			if value, ok := envelope.Errors[0]["code"].(string); ok && value != "" {
				code = value
			}
			if value, ok := envelope.Errors[0]["message"].(string); ok && value != "" {
				message = value
			}
		}
		ambiguous := code == "operation_ambiguous"
		retryable := !ambiguous && (response.StatusCode == http.StatusTooManyRequests || response.StatusCode == http.StatusServiceUnavailable)
		if mutation && response.StatusCode >= 500 && !retryable {
			ambiguous = true
		}
		return apiEnvelope{}, &OperationError{
			Code: code, Message: message, Status: response.StatusCode,
			Ambiguous: ambiguous, Retryable: retryable,
			RequestID: requestID, Details: envelope.Errors,
		}
	}
	envelope := apiEnvelope{Status: response.StatusCode}
	if response.StatusCode == http.StatusNoContent {
		return envelope, nil
	}
	if err := decodeStrict(raw, &envelope); err != nil || len(envelope.Data) == 0 {
		if err == nil {
			err = errors.New("response has no data")
		}
		return apiEnvelope{}, &OperationError{
			Code: "internal_error", Message: "Sandbox API returned an invalid response",
			Status: response.StatusCode, RequestID: requestID,
			Ambiguous: mutation, Details: []ErrorDetail{{"reason": err.Error()}},
		}
	}
	envelope.Status = response.StatusCode
	return envelope, nil
}

type apiResource struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	VPCID     string    `json:"vpcId"`
	ImageRef  string    `json:"imageRef"`
	Resources Resources `json:"resources"`
	CreatedAt string    `json:"createdAt"`
	StartedAt *string   `json:"startedAt"`
	EndedAt   *string   `json:"endedAt"`
	Error     *string   `json:"error,omitempty"`
}

func parseResource(raw json.RawMessage) (*Resource, error) {
	var source apiResource
	if err := decodeStrict(raw, &source); err != nil {
		return nil, err
	}
	if err := validateUUID(source.ID, "sandbox id"); err != nil {
		return nil, err
	}
	if err := validateUUID(source.VPCID, "vpc id"); err != nil {
		return nil, err
	}
	if !namePattern.MatchString(source.Name) || source.ImageRef == "" || source.Resources.VCPU == 0 || source.Resources.MemoryMB == 0 || source.CreatedAt == "" {
		return nil, errors.New("sandbox resource is missing required fields")
	}
	switch source.Status {
	case "PENDING", "STARTING", "RUNNING", "PAUSED", "TERMINATING", "TERMINATED", "FAILED":
	default:
		return nil, fmt.Errorf("invalid sandbox status %q", source.Status)
	}
	return &Resource{
		Kind: "inngest/sandbox", Version: ProtocolVersion,
		ID: source.ID, Name: source.Name, Status: source.Status,
		VPCID: source.VPCID, ImageRef: source.ImageRef, Resources: source.Resources,
		CreatedAt: source.CreatedAt, StartedAt: source.StartedAt,
		EndedAt: source.EndedAt, Error: source.Error,
	}, nil
}

func parseProcess(raw json.RawMessage, sandboxID string, requireCommand bool) (*ProcessResource, error) {
	var source struct {
		ID                string   `json:"id"`
		Command           []string `json:"command,omitempty"`
		PID               *int32   `json:"pid,omitempty"`
		State             string   `json:"state"`
		ExitCode          *int32   `json:"exitCode,omitempty"`
		TerminationSignal *int32   `json:"terminationSignal,omitempty"`
		StartedAt         *string  `json:"startedAt,omitempty"`
		EndedAt           *string  `json:"endedAt,omitempty"`
	}
	if err := decodeStrict(raw, &source); err != nil {
		return nil, err
	}
	if err := validateUUID(source.ID, "process id"); err != nil {
		return nil, err
	}
	if requireCommand && len(source.Command) == 0 {
		return nil, errors.New("process command is required")
	}
	if source.PID != nil && *source.PID <= 0 {
		return nil, errors.New("process PID must be positive")
	}
	if err := validateProcessState(source.State, source.ExitCode, source.TerminationSignal); err != nil {
		return nil, err
	}
	return &ProcessResource{
		Kind: "inngest/sandbox.process", Version: ProtocolVersion,
		SandboxID: sandboxID, ID: source.ID, Command: source.Command,
		PID: source.PID, State: source.State, ExitCode: source.ExitCode,
		TerminationSignal: source.TerminationSignal,
		StartedAt:         source.StartedAt, EndedAt: source.EndedAt,
	}, nil
}

func validateProcessState(state string, exitCode, terminationSignal *int32) error {
	switch state {
	case "STARTING", "RUNNING", "FAILED", "LOST":
		if exitCode != nil || terminationSignal != nil {
			return errors.New("non-terminal process fields are inconsistent")
		}
	case "EXITED":
		if exitCode == nil || terminationSignal != nil {
			return errors.New("EXITED process must contain only exitCode")
		}
	case "KILLED":
		if terminationSignal == nil || exitCode != nil {
			return errors.New("KILLED process must contain only terminationSignal")
		}
	default:
		return fmt.Errorf("invalid process state %q", state)
	}
	return nil
}

func validateTerminalProcess(state string, exitCode, terminationSignal *int32) error {
	if state != "EXITED" && state != "KILLED" && state != "FAILED" && state != "LOST" {
		return fmt.Errorf("wait returned non-terminal process state %q", state)
	}
	return validateProcessState(state, exitCode, terminationSignal)
}

func sandboxPath(sandboxID string) string {
	return "/v2/sandboxes/" + url.PathEscape(sandboxID)
}

func processPath(target ProcessTarget) string {
	return sandboxPath(target.SandboxID) + "/processes/" + url.PathEscape(target.ProcessID)
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func transportError(method string, mutation bool, err error) *OperationError {
	code, message := "compute_unavailable", "Sandbox API request failed"
	if mutation {
		code, message = "operation_ambiguous", "Sandbox mutation result is ambiguous"
	}
	return &OperationError{
		Code: code, Message: message, Ambiguous: mutation, Retryable: !mutation,
		Details: []ErrorDetail{{"reason": err.Error(), "method": method}},
	}
}

func malformedResponse(action Action, sandboxID, processID string, err error) *OperationError {
	return &OperationError{
		Code: "internal_error", Message: "Sandbox API returned an invalid response",
		SandboxID: sandboxID, ProcessID: processID,
		Ambiguous: action.RequiresDispatchFence(), Details: []ErrorDetail{{"reason": err.Error()}},
	}
}

func setErrorTarget(err *OperationError, target ProcessTarget) {
	err.SandboxID = target.SandboxID
	err.ProcessID = target.ProcessID
}
