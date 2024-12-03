package httpdriver

import (
	"encoding/json"
	"testing"

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
