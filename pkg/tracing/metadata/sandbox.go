package metadata

//tygo:generate
const (
	KindInngestSandbox Kind = "inngest.sandbox"
)

// SandboxMetadata describes a step that acted on an Inngest sandbox. The SDK
// emits it, one entry per step attempt with the attempt's full value set, as
// merges never clear a key. Values stay flat (scalars and string arrays) so
// they round-trip through ClickHouse JSON and DuckDB VARIANT storage.
//
//tygo:generate
type SandboxMetadata struct {
	// Version of this shape. Currently 1.
	Version int `json:"version"`

	// Action is the sandbox API operation, like "exec" or "snapshot.create".
	Action string `json:"action"`

	// Method is the SDK method the user called, like "commands.run".
	Method string `json:"method"`

	// SandboxID identifies the machine and is shared by every step on it. A
	// failed create has only the name.
	SandboxID   string `json:"sandbox_id,omitempty"`
	SandboxName string `json:"sandbox_name,omitempty"`

	// SourceSnapshotID is the snapshot a created sandbox was cloned from.
	SourceSnapshotID string `json:"source_snapshot_id,omitempty"`

	// Command is the argv that ran; CommandDisplay is the shell string the user
	// wrote, if any. CommandTruncated is true when either was cut short.
	Command          []string `json:"command,omitempty"`
	CommandDisplay   string   `json:"command_display,omitempty"`
	CommandTruncated bool     `json:"command_truncated,omitempty"`
	Cwd              string   `json:"cwd,omitempty"`

	ProcessID         string `json:"process_id,omitempty"`
	ProcessState      string `json:"process_state,omitempty"`
	ExitCode          *int   `json:"exit_code,omitempty"`
	TerminationSignal *int   `json:"termination_signal,omitempty"`

	// OutputTruncated is true when captured stdout/stderr was cut short.
	OutputTruncated bool `json:"output_truncated,omitempty"`

	SnapshotID     string `json:"snapshot_id,omitempty"`
	SnapshotStatus string `json:"snapshot_status,omitempty"`

	// ErrorCode is the sandbox API's error code when the step failed.
	ErrorCode string `json:"error_code,omitempty"`
}
