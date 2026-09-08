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

	// Trailing data after an object is rejected, matching json.Unmarshal, so
	// the raw payload falls through to the text handling.
	r.Equal(string(`{"a":1}garbage`), ParseResponse([]byte(`{"a":1}garbage`)))
}

// TestParseResponsePreservesLargeIntegers guards against precision loss for
// large integers (e.g. snowflake IDs) inside object responses. Decoding into a
// map[string]interface{} via json.Unmarshal would store the value as a float64,
// whose 53-bit mantissa cannot represent integers beyond 2^53 exactly, so
// re-marshalling the output would silently round the value.
func TestParseResponsePreservesLargeIntegers(t *testing.T) {
	r := require.New(t)

	const snowflake = "616581622363398142"

	assertPreserved := func(input []byte) {
		out := ParseResponse(input)
		m, ok := out.(map[string]any)
		r.True(ok, "expected object response to decode into a map, got %T", out)
		r.Equal(json.Number(snowflake), m["id"])

		// The value must survive a marshal round-trip without rounding, since
		// the output is later re-serialised before being stored and returned.
		byt, err := json.Marshal(m)
		r.NoError(err)
		r.JSONEq(`{"id":`+snowflake+`}`, string(byt))
	}

	// Plain object response.
	assertPreserved([]byte(`{"id":` + snowflake + `}`))

	// Double-encoded object response.
	assertPreserved([]byte(`"{\"id\":` + snowflake + `}"`))
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
