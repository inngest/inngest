package metadata

import (
	"github.com/inngest/inngest/pkg/enums"
)

//tygo:generate
const (
	KindInngestSandbox Kind = "inngest.sandbox"
)

// SandboxRole says how a step relates to the user's code.
//
// A "statement" step is the one the user wrote, like `box.snapshot("snap")`,
// and is the row a trace shows. An "internal" step is extra work the SDK did to
// serve that statement, like waiting for the snapshot to be ready, and belongs
// to the statement's row rather than being a row of its own.
//
//tygo:generate
type SandboxRole string

const (
	SandboxRoleStatement SandboxRole = "statement"
	SandboxRoleInternal  SandboxRole = "internal"
)

// SandboxMetadata describes a step that acted on an Inngest sandbox, so trace
// UIs and APIs can present it as a sandbox action instead of a plain step.
//
// It is step scoped. Today the SDK attaches it to the steps behind
// `step.sandbox`; an executor that runs sandbox operations natively can emit
// the same shape.
//
//tygo:generate
type SandboxMetadata struct {
	// Version of this shape. Currently 1.
	Version int `json:"version"`

	// Action is the sandbox API operation this step performed, like "create",
	// "exec", "snapshot.create" or "snapshot.waitUntilReady".
	Action string `json:"action"`

	// Statement is the SDK method the user called, like "create",
	// "commands.run" or "snapshot". Internal steps carry their statement's
	// method, not their own action.
	Statement string `json:"statement"`

	// StatementID is the step ID of the statement step this step belongs to.
	// For a statement step it's the step's own ID.
	StatementID string `json:"statement_id"`

	Role SandboxRole `json:"role"`

	// SandboxID and SandboxName identify the machine this step acted on, or
	// created. Every step on the same machine shares the ID.
	SandboxID   string `json:"sandbox_id,omitempty"`
	SandboxName string `json:"sandbox_name,omitempty"`

	// SourceSnapshotID is the snapshot a created sandbox was cloned from.
	SourceSnapshotID string `json:"source_snapshot_id,omitempty"`

	// Command is the argv a command or process ran. CommandDisplay is the
	// command as the user wrote it, when they passed a shell string.
	// CommandTruncated is true when either was cut short to keep metadata
	// small.
	Command          []string `json:"command,omitempty"`
	CommandDisplay   string   `json:"command_display,omitempty"`
	CommandTruncated bool     `json:"command_truncated,omitempty"`
	Cwd              string   `json:"cwd,omitempty"`

	ProcessID         string `json:"process_id,omitempty"`
	ProcessState      string `json:"process_state,omitempty"`
	ExitCode          *int   `json:"exit_code,omitempty"`
	TerminationSignal *int   `json:"termination_signal,omitempty"`

	// OutputTruncated is true when a captured command's stdout/stderr was cut
	// to fit in the step's output.
	OutputTruncated bool `json:"output_truncated,omitempty"`

	SnapshotID     string `json:"snapshot_id,omitempty"`
	SnapshotStatus string `json:"snapshot_status,omitempty"`

	// ErrorCode is the sandbox API's error code when the step failed.
	ErrorCode string `json:"error_code,omitempty"`
}

func (sm SandboxMetadata) Kind() Kind {
	return KindInngestSandbox
}

func (sm SandboxMetadata) Op() enums.MetadataOpcode {
	return enums.MetadataOpcodeMerge
}

func (sm SandboxMetadata) Serialize() (Values, error) {
	var values Values
	if err := values.FromStruct(sm); err != nil {
		return nil, err
	}

	return values, nil
}
