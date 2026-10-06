package extractors

import (
	"encoding/json"

	"github.com/inngest/inngest/pkg/tracing/metadata"
)

// SandboxActionSleep is the inngest.sandbox action for a step.sleep that
// waits on behalf of a sandbox statement, like the pause between a long
// command's polls.
const SandboxActionSleep = "sleep"

// sandboxStatementOpts mirrors the object the SDK puts on a step's opcode
// opts as "sandboxStatement" when the step runs inside a sandbox statement
// scope (inngest-js `withSandboxStatement()`). Its keys already match the
// inngest.sandbox metadata fields.
type sandboxStatementOpts struct {
	SandboxStatement *struct {
		Statement     string `json:"statement"`
		StatementID   string `json:"statement_id"`
		StatementName string `json:"statement_name"`
		SandboxID     string `json:"sandbox_id"`
		SandboxName   string `json:"sandbox_name"`
	} `json:"sandboxStatement"`
}

// ExtractSandboxStatementOptsMetadata inspects GeneratorOpcode.Opts for the
// sandbox statement a step serves and returns an internal inngest.sandbox
// entry for it with the given action.
//
// It exists for steps that never run a handler, like step.sleep, so can't
// attach their own metadata. Every step of the statement then shares its
// statement_id and the trace can fold them into one row.
//
// Returns (nil, nil) when opts carry no statement, which is the expected case
// for every step outside a sandbox statement.
func ExtractSandboxStatementOptsMetadata(opts any, action string) (metadata.Structured, error) {
	if opts == nil {
		return nil, nil
	}

	var raw []byte
	switch v := opts.(type) {
	case []byte:
		raw = v
	case json.RawMessage:
		raw = v
	default:
		b, err := json.Marshal(opts)
		if err != nil {
			return nil, err
		}
		raw = b
	}

	var parsed sandboxStatementOpts
	if err := json.Unmarshal(raw, &parsed); err != nil {
		// Opts of another shape just don't carry a statement.
		return nil, nil
	}

	s := parsed.SandboxStatement
	if s == nil || s.StatementID == "" {
		return nil, nil
	}

	return metadata.SandboxMetadata{
		Version:       1,
		Action:        action,
		Statement:     s.Statement,
		StatementID:   s.StatementID,
		Role:          metadata.SandboxRoleInternal,
		StatementName: s.StatementName,
		SandboxID:     s.SandboxID,
		SandboxName:   s.SandboxName,
	}, nil
}
