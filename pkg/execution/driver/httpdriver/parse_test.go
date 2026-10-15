package httpdriver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/stretchr/testify/require"
)

func TestParseResponse(t *testing.T) {
	r := require.New(t)

	// JSON string
	r.Equal(json.RawMessage(`"a"`), ParseResponse([]byte(`"a"`)))

	// JSON number
	r.Equal(json.RawMessage("1"), ParseResponse([]byte("1")))

	// JSON boolean
	r.Equal(json.RawMessage("true"), ParseResponse([]byte("true")))

	// Empty JSON object
	r.Equal(map[string]any{}, ParseResponse([]byte(`{}`)))

	// JSON object
	r.Equal(
		map[string]any{"nested": map[string]any{"deep": "hi"}},
		ParseResponse([]byte(`{"nested": {"deep": "hi"}}`)),
	)

	// JSON array
	r.Equal(
		json.RawMessage(`[{"nested": {"deep": "hi"}}]`),
		ParseResponse([]byte(`[{"nested": {"deep": "hi"}}]`)),
	)

	// HTML (e.g. gateway timeout)
	r.Equal(string("<html>hi</html>"), ParseResponse([]byte("<html>hi</html>")))

	// Partial JSON (e.g. JSON body too large)
	r.Equal(string(`{"data"`), ParseResponse([]byte(`{"data"`)))

	// Double-encoded JSON
	r.Equal(
		map[string]any{"nested": map[string]any{"deep": "hi"}},
		ParseResponse([]byte(`"{\"nested\": {\"deep\": \"hi\"}}"`)),
	)
}

func TestParseGeneratorAllowsExplicitEmptyArray(t *testing.T) {
	t.Parallel()

	ops, err := ParseGenerator(context.Background(), []byte("[]"), false)
	require.NoError(t, err)
	require.Len(t, ops, 1)
	require.Equal(t, enums.OpcodeNone, ops[0].Op)
}

func TestParseGeneratorRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := ParseGenerator(context.Background(), []byte("not-json"), false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "error reading generator opcode response")
}

func TestParseGeneratorUnknownOpcodePromptsUpgrade(t *testing.T) {
	t.Parallel()

	// An opcode a future SDK might return, in both the single-object and array
	// response shapes.
	for _, body := range []string{
		`{"op":"SomeFutureOpcode","id":"x"}`,
		`[{"op":"SomeFutureOpcode","id":"x"}]`,
	} {
		_, err := ParseGenerator(context.Background(), []byte(body), false)
		require.Error(t, err)

		require.Contains(t, err.Error(), "update your Inngest server")
		require.NotContains(t, err.Error(), "error reading generator opcode response")

		var ue *enums.UnknownOpcodeError
		require.True(t, errors.As(err, &ue))
		require.Equal(t, "SomeFutureOpcode", ue.Opcode)

		// Non-retriable: version skew is deterministic.
		require.False(t, queue.ShouldRetry(err, 0, 10))
	}
}

func TestParseGeneratorRejectsNullArrayItem(t *testing.T) {
	t.Parallel()

	_, err := ParseGenerator(context.Background(), []byte(`[null]`), false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "opcode cannot be null")
}

func TestParseGeneratorAllowsExplicitStepRun(t *testing.T) {
	t.Parallel()

	ops, err := ParseGenerator(context.Background(), []byte(`[{"op":"StepRun","id":"step-1","name":"step"}]`), false)
	require.NoError(t, err)
	require.Len(t, ops, 1)
	require.Equal(t, enums.OpcodeStepRun, ops[0].Op)
	require.Equal(t, "step-1", ops[0].ID)
}
