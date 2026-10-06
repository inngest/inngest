package extractors

import (
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractSandboxStatementOptsMetadata_Sleep(t *testing.T) {
	t.Parallel()

	// What inngest-js sends for a step.sleep inside withSandboxStatement().
	opts := map[string]any{
		"duration": "1s",
		"sandboxStatement": map[string]any{
			"statement":      "commands.run",
			"statement_id":   "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3",
			"statement_name": "build › install",
			"sandbox_id":     "22222222-2222-4222-8222-222222222222",
			"sandbox_name":   "ci-box",
		},
	}

	md, err := ExtractSandboxStatementOptsMetadata(opts, SandboxActionSleep)
	require.NoError(t, err)
	require.NotNil(t, md)
	assert.Equal(t, metadata.KindInngestSandbox, md.Kind())
	require.NoError(t, md.Kind().ValidateAllowed())

	raw, err := md.Serialize()
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		"version":        float64(1),
		"action":         "sleep",
		"statement":      "commands.run",
		"statement_id":   "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3",
		"statement_name": "build › install",
		"role":           "internal",
		"sandbox_id":     "22222222-2222-4222-8222-222222222222",
		"sandbox_name":   "ci-box",
	}, decodeValues(t, raw))
}

func TestExtractSandboxStatementOptsMetadata_RawJSON(t *testing.T) {
	t.Parallel()

	opts := json.RawMessage(`{"duration":"1s","sandboxStatement":{"statement":"processes.start","statement_id":"abc","statement_name":"server"}}`)

	md, err := ExtractSandboxStatementOptsMetadata(opts, SandboxActionSleep)
	require.NoError(t, err)
	require.NotNil(t, md)

	raw, err := md.Serialize()
	require.NoError(t, err)

	decoded := decodeValues(t, raw)
	assert.Equal(t, "abc", decoded["statement_id"])
	assert.Equal(t, "server", decoded["statement_name"])
	assert.NotContains(t, decoded, "sandbox_id")
}

func TestExtractSandboxStatementOptsMetadata_None(t *testing.T) {
	t.Parallel()

	for name, opts := range map[string]any{
		"nil":          nil,
		"plain sleep":  map[string]any{"duration": "1s"},
		"no statement": map[string]any{"sandboxStatement": map[string]any{"statement": "commands.run"}},
		"wrong shape":  map[string]any{"sandboxStatement": "nope"},
		"invalid json": json.RawMessage(`{`),
	} {
		t.Run(name, func(t *testing.T) {
			md, err := ExtractSandboxStatementOptsMetadata(opts, SandboxActionSleep)
			require.NoError(t, err)
			assert.Nil(t, md)
		})
	}
}
