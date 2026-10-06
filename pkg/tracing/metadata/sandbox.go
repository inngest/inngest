package metadata

import (
	"github.com/inngest/inngest/pkg/enums"
)

//tygo:generate
const (
	KindInngestSandbox Kind = "inngest.sandbox"
)

// SandboxMetadata describes a step that acted on an Inngest sandbox, so trace
// UIs and APIs can present it as a sandbox action instead of a plain step.
//
// It is step scoped. Each step attempt emits exactly one entry, carrying the
// attempt's full value set. Entries for the same span and kind are folded as
// merge patches (see Op), which never clear a key a later entry omits, so an
// entry must never be a partial update that relies on an earlier one.
//
// Values stay flat: scalars and short string arrays only, no nested objects,
// so they round-trip through ClickHouse JSON and DuckDB VARIANT storage.
//
//tygo:generate
type SandboxMetadata struct {
	// Version of this shape. Currently 1.
	Version int `json:"version"`

	// Action is the sandbox API operation this step performed, like "create",
	// "exec", "snapshot.create" or "snapshot.waitUntilReady".
	Action string `json:"action"`

	// Method is the SDK method the user called, like "create",
	// "commands.run" or "snapshot".
	Method string `json:"method"`

	// SandboxID and SandboxName identify the machine this step acted on, or
	// created. Every step on the same machine shares the ID, and it's the key
	// to group by: a machine may have been created, and may be destroyed,
	// outside this run. A failed create has only the name.
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
