package sandbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/google/uuid"
)

const ProtocolVersion = 1

type Action string

const (
	ActionCreate        Action = "create"
	ActionList          Action = "list"
	ActionGet           Action = "get"
	ActionExec          Action = "exec"
	ActionDestroy       Action = "destroy"
	ActionProcessStart  Action = "process.start"
	ActionProcessList   Action = "process.list"
	ActionProcessGet    Action = "process.get"
	ActionProcessSignal Action = "process.signal"
	ActionProcessWait   Action = "process.wait"
	ActionProcessOutput Action = "process.output"
)

func (a Action) Valid() bool {
	switch a {
	case ActionCreate,
		ActionList,
		ActionGet,
		ActionExec,
		ActionDestroy,
		ActionProcessStart,
		ActionProcessList,
		ActionProcessGet,
		ActionProcessSignal,
		ActionProcessWait,
		ActionProcessOutput:
		return true
	default:
		return false
	}
}

// RequiresDispatchFence identifies operations that can have external effects.
// Reads remain safe to redispatch after availability failures.
func (a Action) RequiresDispatchFence() bool {
	switch a {
	case ActionCreate, ActionExec, ActionDestroy, ActionProcessStart, ActionProcessSignal:
		return true
	default:
		return false
	}
}

type Target struct {
	SandboxID string `json:"sandboxId"`
}

type ProcessTarget struct {
	SandboxID string `json:"sandboxId"`
	ProcessID string `json:"processId"`
}

type CreateInput struct {
	Name        string            `json:"name"`
	VCPU        uint32            `json:"vcpu"`
	MemoryMB    uint32            `json:"memoryMb"`
	Environment map[string]string `json:"environment,omitempty"`
}

type ListInput struct {
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit"`
}

type GetInput struct {
	SandboxID string `json:"sandboxId"`
}

type ProcessSpecInput struct {
	Command     []string          `json:"command"`
	Environment map[string]string `json:"environment,omitempty"`
	CWD         string            `json:"cwd,omitempty"`
}

type ExecInput struct {
	ProcessSpecInput
	TimeoutMS int `json:"timeoutMs"`
}

type SignalInput struct {
	Signal          int32 `json:"signal"`
	IncludeChildren bool  `json:"includeChildren"`
}

type WaitInput struct {
	TimeoutMS int `json:"timeoutMs"`
}

type OutputInput struct {
	TailBytes uint32 `json:"tailBytes"`
}

type Operation struct {
	ProtocolVersion int
	Action          Action
	Create          *CreateInput
	List            *ListInput
	Get             *GetInput
	Exec            *ExecInput
	ProcessSpec     *ProcessSpecInput
	Signal          *SignalInput
	Wait            *WaitInput
	Output          *OutputInput
	Target          *Target
	ProcessTarget   *ProcessTarget
}

type Resources struct {
	VCPU     uint32 `json:"vcpu"`
	MemoryMB uint32 `json:"memoryMb"`
}

type Resource struct {
	Kind      string    `json:"kind"`
	Version   int       `json:"version"`
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	VPCID     string    `json:"vpcId"`
	ImageRef  string    `json:"imageRef"`
	Resources Resources `json:"resources"`
	CreatedAt string    `json:"createdAt"`
	StartedAt *string   `json:"startedAt,omitempty"`
	EndedAt   *string   `json:"endedAt,omitempty"`
	Error     *string   `json:"error,omitempty"`
}

type Page struct {
	Cursor  string `json:"cursor,omitempty"`
	HasMore bool   `json:"hasMore"`
	Limit   int    `json:"limit"`
}

type ListResult struct {
	Sandboxes []*Resource `json:"sandboxes"`
	Page      Page        `json:"page"`
	FetchedAt string      `json:"fetchedAt"`
}

type CommandResult struct {
	Stdout   []byte `json:"stdout"`
	Stderr   []byte `json:"stderr"`
	Encoding string `json:"encoding"`
	ExitCode int32  `json:"exitCode"`
}

type DestroyResult struct {
	Status  string    `json:"status"`
	Sandbox *Resource `json:"sandbox"`
}

type ProcessResource struct {
	Kind              string   `json:"kind"`
	Version           int      `json:"version"`
	SandboxID         string   `json:"sandboxId"`
	ID                string   `json:"id"`
	Command           []string `json:"command"`
	PID               *int32   `json:"pid,omitempty"`
	State             string   `json:"state"`
	ExitCode          *int32   `json:"exitCode,omitempty"`
	TerminationSignal *int32   `json:"terminationSignal,omitempty"`
	StartedAt         *string  `json:"startedAt,omitempty"`
	EndedAt           *string  `json:"endedAt,omitempty"`
}

type WaitProcessResult struct {
	Kind              string `json:"kind"`
	Version           int    `json:"version"`
	SandboxID         string `json:"sandboxId"`
	ID                string `json:"id"`
	State             string `json:"state"`
	ExitCode          *int32 `json:"exitCode,omitempty"`
	TerminationSignal *int32 `json:"terminationSignal,omitempty"`
}

type OutputChunk struct {
	Stream   string `json:"stream"`
	Data     []byte `json:"data"`
	Encoding string `json:"encoding"`
	At       string `json:"at,omitempty"`
}

type OutputResult struct {
	Chunks []OutputChunk `json:"chunks"`
}

type Result struct {
	Sandbox    *Resource          `json:"sandbox,omitempty"`
	List       *ListResult        `json:"list,omitempty"`
	Command    *CommandResult     `json:"command,omitempty"`
	Destroy    *DestroyResult     `json:"destroy,omitempty"`
	Process    *ProcessResource   `json:"process,omitempty"`
	Processes  []*ProcessResource `json:"processes,omitempty"`
	Wait       *WaitProcessResult `json:"wait,omitempty"`
	Output     *OutputResult      `json:"output,omitempty"`
	SignalDone bool               `json:"signalDone,omitempty"`
}

type ErrorDetail map[string]any

type OperationError struct {
	Code      string        `json:"code"`
	Message   string        `json:"message"`
	Status    int           `json:"status,omitempty"`
	SandboxID string        `json:"sandboxId,omitempty"`
	ProcessID string        `json:"processId,omitempty"`
	Ambiguous bool          `json:"ambiguous"`
	Retryable bool          `json:"retryable"`
	RequestID string        `json:"requestId,omitempty"`
	Details   []ErrorDetail `json:"details"`
}

func (e *OperationError) Error() string { return e.Message }

type Provider interface {
	Execute(ctx context.Context, workspaceID uuid.UUID, operation Operation) (Result, *OperationError)
}

type DispatchKey struct {
	AccountID   uuid.UUID
	WorkspaceID uuid.UUID
	RunID       string
	StepID      string
}

type DispatchState string

const (
	DispatchStateDispatching DispatchState = "dispatching"
	DispatchStateComplete    DispatchState = "complete"
)

// DispatchRecord is the durable intent/result stored before and after a
// mutation is sent. A dispatching record is deliberately treated as
// ambiguous after a crash: it is never safe to send the mutation again.
type DispatchRecord struct {
	Digest string          `json:"digest"`
	Action Action          `json:"action"`
	State  DispatchState   `json:"state"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *OperationError `json:"error,omitempty"`
}

type DispatchStore interface {
	Claim(ctx context.Context, key DispatchKey, record DispatchRecord) (stored DispatchRecord, claimed bool, err error)
	Complete(ctx context.Context, key DispatchKey, record DispatchRecord) error
	Release(ctx context.Context, key DispatchKey, digest string) error
}

type memoryDispatchStore struct {
	mu      sync.Mutex
	records map[DispatchKey]DispatchRecord
}

func NewMemoryDispatchStore() DispatchStore {
	return &memoryDispatchStore{records: map[DispatchKey]DispatchRecord{}}
}

func (s *memoryDispatchStore) Claim(_ context.Context, key DispatchKey, record DispatchRecord) (DispatchRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.records[key]; ok {
		return existing, false, nil
	}
	s.records[key] = cloneDispatchRecord(record)
	return record, true, nil
}

func (s *memoryDispatchStore) Complete(_ context.Context, key DispatchKey, record DispatchRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.records[key]
	if !ok || existing.Digest != record.Digest || existing.Action != record.Action {
		return errors.New("sandbox dispatch intent changed before completion")
	}
	s.records[key] = cloneDispatchRecord(record)
	return nil
}

func (s *memoryDispatchStore) Release(_ context.Context, key DispatchKey, digest string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.records[key]; ok && existing.Digest == digest && existing.State == DispatchStateDispatching {
		delete(s.records, key)
	}
	return nil
}

func cloneDispatchRecord(record DispatchRecord) DispatchRecord {
	cloned := record
	cloned.Result = append(json.RawMessage(nil), record.Result...)
	if record.Error != nil {
		value := *record.Error
		value.Details = append([]ErrorDetail(nil), record.Error.Details...)
		cloned.Error = &value
	}
	return cloned
}

type Dispatcher struct {
	provider Provider
	store    DispatchStore
}

func NewDispatcher(provider Provider, stores ...DispatchStore) *Dispatcher {
	store := DispatchStore(NewMemoryDispatchStore())
	if len(stores) > 0 && stores[0] != nil {
		store = stores[0]
	}
	return &Dispatcher{provider: provider, store: store}
}

func (d *Dispatcher) Execute(ctx context.Context, key DispatchKey, operation Operation) (Result, *OperationError) {
	if !operation.Action.RequiresDispatchFence() {
		return d.provider.Execute(ctx, key.WorkspaceID, operation)
	}
	digest, err := operation.Digest()
	if err != nil {
		return Result{}, invalidOperation(err)
	}
	digestString := hex.EncodeToString(digest[:])
	intent := DispatchRecord{
		Digest: digestString,
		Action: operation.Action,
		State:  DispatchStateDispatching,
	}
	stored, claimed, err := d.store.Claim(ctx, key, intent)
	if err != nil {
		return Result{}, retryableStoreError(err)
	}
	if !claimed {
		if stored.Digest != digestString || stored.Action != operation.Action {
			return Result{}, operationError("idempotency_conflict", "Sandbox step ID was reused with different input")
		}
		if stored.State != DispatchStateComplete {
			ambiguous := operationError("operation_ambiguous", "Sandbox mutation dispatch state is uncertain")
			ambiguous.Ambiguous = true
			ambiguous.SandboxID = operation.SandboxID()
			ambiguous.ProcessID = operation.ProcessID()
			return Result{}, ambiguous
		}
		if stored.Error != nil {
			return Result{}, stored.Error
		}
		var result Result
		if len(stored.Result) == 0 || json.Unmarshal(stored.Result, &result) != nil {
			ambiguous := operationError("operation_ambiguous", "Persisted sandbox mutation result is invalid")
			ambiguous.Ambiguous = true
			return Result{}, ambiguous
		}
		return result, nil
	}

	result, operationErr := d.provider.Execute(ctx, key.WorkspaceID, operation)
	if operationErr != nil {
		if operationErr.Details == nil {
			operationErr.Details = []ErrorDetail{}
		}
		if operationErr.Retryable && !operationErr.Ambiguous {
			if err := d.store.Release(ctx, key, digestString); err != nil {
				ambiguous := operationError("operation_ambiguous", "Could not safely release sandbox mutation intent")
				ambiguous.Ambiguous = true
				ambiguous.Details = []ErrorDetail{{"reason": err.Error()}}
				return Result{}, ambiguous
			}
			return Result{}, operationErr
		}
	}

	record := intent
	record.State = DispatchStateComplete
	record.Error = operationErr
	if operationErr == nil {
		record.Result, err = json.Marshal(result)
		if err != nil {
			ambiguous := operationError("operation_ambiguous", "Could not encode sandbox mutation result")
			ambiguous.Ambiguous = true
			ambiguous.Details = []ErrorDetail{{"reason": err.Error()}}
			return Result{}, ambiguous
		}
	}
	if err := d.store.Complete(ctx, key, record); err != nil {
		ambiguous := operationError("operation_ambiguous", "Could not persist sandbox mutation result")
		ambiguous.Ambiguous = true
		ambiguous.Details = []ErrorDetail{{"reason": err.Error()}}
		return Result{}, ambiguous
	}
	return result, operationErr
}

type opcodeOpts struct {
	Type              string          `json:"type"`
	Sandbox           json.RawMessage `json:"sandbox"`
	StackLine         string          `json:"stackLine,omitempty"`
	ParallelMode      string          `json:"parallelMode,omitempty"`
	ExperimentStepID  string          `json:"experimentStepID,omitempty"`
	ExperimentName    string          `json:"experimentName,omitempty"`
	Variant           string          `json:"variant,omitempty"`
	SelectionStrategy string          `json:"selectionStrategy,omitempty"`
}

func ParseOpcodeOpts(value any) (Operation, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return Operation{}, err
	}
	var opts opcodeOpts
	if err := decodeStrict(raw, &opts); err != nil {
		return Operation{}, fmt.Errorf("decode sandbox options: %w", err)
	}
	if opts.ParallelMode == "race" {
		return Operation{}, errors.New("sandbox operations cannot use race parallelism")
	}
	var discriminator struct {
		ProtocolVersion int    `json:"protocolVersion"`
		Action          Action `json:"action"`
	}
	if err := json.Unmarshal(opts.Sandbox, &discriminator); err != nil {
		return Operation{}, fmt.Errorf("decode sandbox operation: %w", err)
	}
	if discriminator.ProtocolVersion != ProtocolVersion {
		return Operation{}, fmt.Errorf("unsupported sandbox protocol version %d", discriminator.ProtocolVersion)
	}
	if opts.Type != "step.sandbox."+string(discriminator.Action) {
		return Operation{}, errors.New("sandbox type does not match its action")
	}
	return parseAction(discriminator.Action, opts.Sandbox)
}

func parseAction(action Action, raw json.RawMessage) (Operation, error) {
	base := Operation{ProtocolVersion: ProtocolVersion, Action: action}
	switch action {
	case ActionCreate:
		input, err := parseSingleInput[CreateInput](raw, action, nil)
		if err == nil {
			err = input.Validate()
		}
		base.Create = input
		return base, err
	case ActionList:
		input, err := parseSingleInput[ListInput](raw, action, nil)
		if err == nil {
			err = input.Validate()
		}
		base.List = input
		return base, err
	case ActionGet:
		input, err := parseSingleInput[GetInput](raw, action, nil)
		if err == nil {
			err = input.Validate()
		}
		base.Get = input
		return base, err
	case ActionExec:
		var target Target
		input, err := parseSingleInput[ExecInput](raw, action, &target)
		if err == nil {
			err = target.Validate()
		}
		if err == nil {
			err = input.Validate()
		}
		base.Target, base.Exec = &target, input
		return base, err
	case ActionDestroy:
		var target Target
		err := parseEmptyInput(raw, action, &target)
		if err == nil {
			err = target.Validate()
		}
		base.Target = &target
		return base, err
	case ActionProcessStart:
		var target Target
		input, err := parseSingleInput[ProcessSpecInput](raw, action, &target)
		if err == nil {
			err = target.Validate()
		}
		if err == nil {
			err = input.Validate()
		}
		base.Target, base.ProcessSpec = &target, input
		return base, err
	case ActionProcessList:
		var target Target
		err := parseEmptyInput(raw, action, &target)
		if err == nil {
			err = target.Validate()
		}
		base.Target = &target
		return base, err
	case ActionProcessGet:
		var target ProcessTarget
		err := parseEmptyInput(raw, action, &target)
		if err == nil {
			err = target.Validate()
		}
		base.ProcessTarget = &target
		return base, err
	case ActionProcessSignal:
		var target ProcessTarget
		input, err := parseSingleInput[SignalInput](raw, action, &target)
		if err == nil {
			err = target.Validate()
		}
		if err == nil {
			err = input.Validate()
		}
		base.ProcessTarget, base.Signal = &target, input
		return base, err
	case ActionProcessWait:
		var target ProcessTarget
		input, err := parseSingleInput[WaitInput](raw, action, &target)
		if err == nil {
			err = target.Validate()
		}
		if err == nil {
			err = input.Validate()
		}
		base.ProcessTarget, base.Wait = &target, input
		return base, err
	case ActionProcessOutput:
		var target ProcessTarget
		input, err := parseSingleInput[OutputInput](raw, action, &target)
		if err == nil {
			err = target.Validate()
		}
		if err == nil {
			err = input.Validate()
		}
		base.ProcessTarget, base.Output = &target, input
		return base, err
	default:
		return Operation{}, fmt.Errorf("unsupported sandbox action %q", action)
	}
}

func parseSingleInput[T any](raw json.RawMessage, action Action, target any) (*T, error) {
	payload := struct {
		ProtocolVersion int    `json:"protocolVersion"`
		Action          Action `json:"action"`
		Target          any    `json:"target,omitempty"`
		Input           []T    `json:"input"`
	}{Target: target}
	if err := decodeStrict(raw, &payload); err != nil {
		return nil, err
	}
	if payload.ProtocolVersion != ProtocolVersion || payload.Action != action {
		return nil, errors.New("sandbox operation discriminator changed")
	}
	if len(payload.Input) != 1 {
		return nil, fmt.Errorf("%s input must contain exactly one item", action)
	}
	return &payload.Input[0], nil
}

func parseEmptyInput(raw json.RawMessage, action Action, target any) error {
	payload := struct {
		ProtocolVersion int    `json:"protocolVersion"`
		Action          Action `json:"action"`
		Target          any    `json:"target"`
		Input           []any  `json:"input"`
	}{Target: target}
	if err := decodeStrict(raw, &payload); err != nil {
		return err
	}
	if payload.ProtocolVersion != ProtocolVersion || payload.Action != action {
		return errors.New("sandbox operation discriminator changed")
	}
	if len(payload.Input) != 0 {
		return fmt.Errorf("%s input must be empty", action)
	}
	return nil
}

func (o Operation) Input() ([]byte, error) {
	switch o.Action {
	case ActionCreate:
		return json.Marshal([]*CreateInput{o.Create})
	case ActionList:
		return json.Marshal([]*ListInput{o.List})
	case ActionGet:
		return json.Marshal([]*GetInput{o.Get})
	case ActionExec:
		return json.Marshal([]*ExecInput{o.Exec})
	case ActionDestroy, ActionProcessList, ActionProcessGet:
		return []byte("[]"), nil
	case ActionProcessStart:
		return json.Marshal([]*ProcessSpecInput{o.ProcessSpec})
	case ActionProcessSignal:
		return json.Marshal([]*SignalInput{o.Signal})
	case ActionProcessWait:
		return json.Marshal([]*WaitInput{o.Wait})
	case ActionProcessOutput:
		return json.Marshal([]*OutputInput{o.Output})
	default:
		return nil, fmt.Errorf("unsupported sandbox action %q", o.Action)
	}
}

func (o Operation) SandboxID() string {
	if o.Get != nil {
		return o.Get.SandboxID
	}
	if o.Target != nil {
		return o.Target.SandboxID
	}
	if o.ProcessTarget != nil {
		return o.ProcessTarget.SandboxID
	}
	return ""
}

func (o Operation) ProcessID() string {
	if o.ProcessTarget != nil {
		return o.ProcessTarget.ProcessID
	}
	return ""
}

func (o Operation) Digest() ([sha256.Size]byte, error) {
	raw, err := json.Marshal(o)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	return sha256.Sum256(raw), nil
}

func MarshalResult(action Action, result Result) ([]byte, error) {
	base := map[string]any{"protocolVersion": ProtocolVersion, "action": action}
	switch action {
	case ActionCreate:
		if result.Sandbox == nil {
			return nil, errors.New("create returned no sandbox")
		}
		base["sandbox"] = result.Sandbox
	case ActionList:
		if result.List == nil {
			return nil, errors.New("list returned no result")
		}
		base["sandboxes"], base["page"], base["fetchedAt"] = result.List.Sandboxes, result.List.Page, result.List.FetchedAt
	case ActionGet:
		base["sandbox"] = result.Sandbox
	case ActionExec:
		if result.Command == nil {
			return nil, errors.New("exec returned no result")
		}
		base["result"] = result.Command
	case ActionDestroy:
		if result.Destroy == nil {
			return nil, errors.New("destroy returned no result")
		}
		base["result"] = result.Destroy
	case ActionProcessStart:
		if result.Process == nil {
			return nil, errors.New("process start returned no process")
		}
		base["process"] = result.Process
	case ActionProcessList:
		if result.Processes == nil {
			result.Processes = []*ProcessResource{}
		}
		base["processes"] = result.Processes
	case ActionProcessGet:
		base["process"] = result.Process
	case ActionProcessSignal:
		if !result.SignalDone {
			return nil, errors.New("process signal was not confirmed")
		}
		base["result"] = nil
	case ActionProcessWait:
		if result.Wait == nil {
			return nil, errors.New("process wait returned no process")
		}
		base["process"] = result.Wait
	case ActionProcessOutput:
		if result.Output == nil {
			return nil, errors.New("process output returned no result")
		}
		base["result"] = result.Output
	default:
		return nil, fmt.Errorf("unsupported sandbox action %q", action)
	}
	return json.Marshal(base)
}

func MarshalError(action Action, operationErr *OperationError) ([]byte, error) {
	if !action.Valid() {
		return nil, fmt.Errorf("unsupported sandbox action %q", action)
	}
	if operationErr == nil {
		return nil, errors.New("sandbox operation error is required")
	}
	return json.Marshal(struct {
		ProtocolVersion int           `json:"protocolVersion"`
		Action          Action        `json:"action"`
		Code            string        `json:"code"`
		Message         string        `json:"message"`
		Status          int           `json:"status,omitempty"`
		SandboxID       string        `json:"sandboxId,omitempty"`
		ProcessID       string        `json:"processId,omitempty"`
		Ambiguous       bool          `json:"ambiguous"`
		Retryable       bool          `json:"retryable"`
		RequestID       string        `json:"requestId,omitempty"`
		Details         []ErrorDetail `json:"details"`
	}{
		ProtocolVersion: ProtocolVersion,
		Action:          action,
		Code:            operationErr.Code,
		Message:         operationErr.Message,
		Status:          operationErr.Status,
		SandboxID:       operationErr.SandboxID,
		ProcessID:       operationErr.ProcessID,
		Ambiguous:       operationErr.Ambiguous,
		Retryable:       operationErr.Retryable,
		RequestID:       operationErr.RequestID,
		Details:         operationErr.Details,
	})
}

var namePattern = regexp.MustCompile(`^[a-z0-9_-]{1,63}$`)

func (t Target) Validate() error {
	return validateUUID(t.SandboxID, "sandboxId")
}

func (t ProcessTarget) Validate() error {
	if err := validateUUID(t.SandboxID, "sandboxId"); err != nil {
		return err
	}
	return validateUUID(t.ProcessID, "processId")
}

func (i CreateInput) Validate() error {
	if !namePattern.MatchString(i.Name) {
		return errors.New("name must contain 1 to 63 lowercase letters, digits, underscores, or hyphens")
	}
	if i.VCPU == 0 || i.MemoryMB == 0 {
		return errors.New("vcpu and memoryMb must be positive uint32 values")
	}
	return validateEnvironment(i.Environment)
}

func (i ListInput) Validate() error {
	if i.Limit < 1 || i.Limit > 250 {
		return errors.New("limit must be between 1 and 250")
	}
	if strings.TrimSpace(i.Cursor) != i.Cursor {
		return errors.New("cursor must not contain surrounding whitespace")
	}
	return nil
}

func (i GetInput) Validate() error {
	return validateUUID(i.SandboxID, "sandboxId")
}

func (i ProcessSpecInput) Validate() error {
	if len(i.Command) < 1 || len(i.Command) > 128 || !strings.HasPrefix(i.Command[0], "/") {
		return errors.New("command must contain 1 to 128 entries and begin with an absolute executable path")
	}
	argvBytes := 0
	for _, value := range i.Command {
		if !portableString(value) {
			return errors.New("command must contain valid UTF-8 without NUL")
		}
		argvBytes += len(value)
	}
	if argvBytes > 32<<10 {
		return errors.New("command exceeds 32768 bytes")
	}
	if err := validateEnvironment(i.Environment); err != nil {
		return err
	}
	if !portableString(i.CWD) || len(i.CWD) > 4096 {
		return errors.New("cwd must be valid UTF-8 without NUL and at most 4096 bytes")
	}
	wire, err := json.Marshal(struct {
		Argv []string          `json:"argv"`
		Env  map[string]string `json:"env,omitempty"`
		CWD  string            `json:"cwd,omitempty"`
	}{Argv: i.Command, Env: i.Environment, CWD: i.CWD})
	if err != nil || len(wire) > 96<<10 {
		return errors.New("process spec exceeds 98304 encoded bytes")
	}
	return nil
}

func validateEnvironment(environment map[string]string) error {
	if len(environment) > 256 {
		return errors.New("environment exceeds 256 entries")
	}
	envBytes := 0
	for key, value := range environment {
		if key == "" || strings.ContainsAny(key, "=\x00") || !portableString(key) || !portableString(value) {
			return errors.New("environment keys must be nonempty without '=' or NUL and values must not contain NUL")
		}
		envBytes += len(key) + 1 + len(value)
	}
	if envBytes > 64<<10 {
		return errors.New("environment exceeds 65536 bytes")
	}
	return nil
}

func (i ExecInput) Validate() error {
	if err := i.ProcessSpecInput.Validate(); err != nil {
		return err
	}
	return validateTimeout(i.TimeoutMS)
}

func (i SignalInput) Validate() error {
	if i.Signal < 1 || i.Signal > 64 {
		return errors.New("signal must be between 1 and 64")
	}
	return nil
}

func (i WaitInput) Validate() error {
	return validateTimeout(i.TimeoutMS)
}

func (i OutputInput) Validate() error {
	if i.TailBytes > 512<<10 {
		return errors.New("tailBytes must not exceed 524288")
	}
	return nil
}

func validateTimeout(value int) error {
	if value < 1 || value > 300_000 {
		return errors.New("timeoutMs must be between 1 and 300000")
	}
	return nil
}

func validateUUID(value, field string) error {
	parsed, err := uuid.Parse(value)
	if err != nil || parsed == uuid.Nil || parsed.String() != value {
		return fmt.Errorf("%s must be a canonical lowercase non-nil UUID", field)
	}
	return nil
}

func portableString(value string) bool {
	return utf8.ValidString(value) && !bytes.ContainsRune([]byte(value), 0)
}

func decodeStrict(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("unexpected trailing JSON")
	}
	return nil
}

func operationError(code, message string) *OperationError {
	return &OperationError{Code: code, Message: message, Details: []ErrorDetail{}}
}

func invalidOperation(err error) *OperationError {
	return operationError("invalid_request", err.Error())
}

func retryableStoreError(err error) *OperationError {
	result := operationError("compute_unavailable", "Sandbox dispatch persistence is temporarily unavailable")
	result.Retryable = true
	result.Details = []ErrorDetail{{"reason": err.Error()}}
	return result
}
