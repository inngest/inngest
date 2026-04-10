package httpdriver

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
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

func TestParseGeneratorRejectsImplicitNoneObject(t *testing.T) {
	t.Parallel()

	_, err := ParseGenerator(context.Background(), []byte(`{}`), false)
	require.Error(t, err)
	require.Contains(t, err.Error(), `must include "op"`)
}

func TestParseGeneratorRejectsImplicitNoneArrayItem(t *testing.T) {
	t.Parallel()

	_, err := ParseGenerator(context.Background(), []byte(`[{}]`), false)
	require.Error(t, err)
	require.Contains(t, err.Error(), `must include "op"`)
}

func TestParseGeneratorRejectsNullArrayItem(t *testing.T) {
	t.Parallel()

	_, err := ParseGenerator(context.Background(), []byte(`[null]`), false)
	require.Error(t, err)
	require.Contains(t, err.Error(), `must be an object with an "op" field`)
}

func TestParseGeneratorAllowsExplicitStepRun(t *testing.T) {
	t.Parallel()

	ops, err := ParseGenerator(context.Background(), []byte(`[{"op":"StepRun","id":"step-1","name":"step"}]`), false)
	require.NoError(t, err)
	require.Len(t, ops, 1)
	require.Equal(t, enums.OpcodeStepRun, ops[0].Op)
	require.Equal(t, "step-1", ops[0].ID)
}
