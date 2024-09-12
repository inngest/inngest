package httpdriver

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseResponse(t *testing.T) {
	r := require.New(t)

	r.Equal(json.RawMessage(`"a"`), parseResponse([]byte(`"a"`)))
	r.Equal(json.RawMessage("1"), parseResponse([]byte("1")))
	r.Equal(json.RawMessage("true"), parseResponse([]byte("true")))
	r.Equal(json.RawMessage(`{}`), parseResponse([]byte(`{}`)))
	r.Equal(json.RawMessage(`{"nested": {"deep": 1}}`), parseResponse([]byte(`{"nested": {"deep": 1}}`)))
	r.Equal(json.RawMessage(`[{"nested": {"deep": 1}}]`), parseResponse([]byte(`[{"nested": {"deep": 1}}]`)))
	r.Equal(string("<html>hi</html>"), parseResponse([]byte("<html>hi</html>")))
	r.Equal(string(`{"partial-json"`), parseResponse([]byte(`{"partial-json"`)))

}
