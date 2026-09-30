package metadata

import (
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSandboxMetadataSerialize(t *testing.T) {
	t.Parallel()

	exitCode := 0
	md := SandboxMetadata{
		Version:        1,
		Action:         "exec",
		Statement:      "commands.run",
		StatementID:    "abc123",
		Role:           SandboxRoleStatement,
		SandboxID:      "9ecbb10f-de90-47c3-94ef-9685bb5c5b61",
		SandboxName:    "box",
		Command:        []string{"/bin/sh", "-c", "npm test"},
		CommandDisplay: "npm test",
		ExitCode:       &exitCode,
	}

	assert.Equal(t, KindInngestSandbox, md.Kind())
	assert.Equal(t, enums.MetadataOpcodeMerge, md.Op())

	values, err := md.Serialize()
	require.NoError(t, err)

	assert.JSONEq(t, `"exec"`, string(values["action"]))
	assert.JSONEq(t, `"abc123"`, string(values["statement_id"]))
	assert.JSONEq(t, `"statement"`, string(values["role"]))
	assert.JSONEq(t, `["/bin/sh","-c","npm test"]`, string(values["command"]))
	// A zero exit code is meaningful and must survive omitempty.
	assert.JSONEq(t, `0`, string(values["exit_code"]))
	assert.NotContains(t, values, "snapshot_id")
	assert.NotContains(t, values, "error_code")

	var roundTrip SandboxMetadata
	raw, err := json.Marshal(values)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &roundTrip))
	assert.Equal(t, md, roundTrip)
}

func TestSandboxMetadataUpdateAllowed(t *testing.T) {
	t.Parallel()

	values, err := SandboxMetadata{Version: 1, Action: "create"}.Serialize()
	require.NoError(t, err)

	update := Update{RawUpdate: RawUpdate{
		Kind:   KindInngestSandbox,
		Op:     enums.MetadataOpcodeMerge,
		Values: values,
	}}
	require.NoError(t, update.ValidateAllowed())
}
