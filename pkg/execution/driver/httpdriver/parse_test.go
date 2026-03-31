package httpdriver

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
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

// TestEmptyArrayNormalizedToOpcodeNone verifies the normalization pipeline
// and its interaction with IsFunctionResult (EXE-1545).
func TestEmptyArrayNormalizedToOpcodeNone(t *testing.T) {
	t.Parallel()

	t.Run("empty JSON array becomes OpcodeNone", func(t *testing.T) {
		ops, err := ParseGenerator(context.Background(), []byte("[]"), false)
		require.NoError(t, err)
		require.Len(t, ops, 1)
		require.Equal(t, enums.OpcodeNone, ops[0].Op)
	})

	t.Run("OpcodeNone from empty array is recognized as function result", func(t *testing.T) {
		ops, err := ParseGenerator(context.Background(), []byte("[]"), false)
		require.NoError(t, err)

		resp := &state.DriverResponse{
			Generator:  ops,
			Output:     nil,
			Err:        nil,
			StatusCode: 206,
		}

		require.True(t, resp.IsFunctionResult(),
			"response from empty SDK array should be a function result")
	})
}
