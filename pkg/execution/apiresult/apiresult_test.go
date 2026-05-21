package apiresult

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIResult_UnmarshalJSON(t *testing.T) {
	t.Run("decodes body string", func(t *testing.T) {
		raw := `{"status":201,"headers":{"X-Foo":"bar"},"body":"{\"a\":1}","version":2}`

		var r APIResult
		require.NoError(t, json.Unmarshal([]byte(raw), &r))

		assert.Equal(t, 201, r.StatusCode)
		assert.Equal(t, map[string]string{"X-Foo": "bar"}, r.Headers)
		assert.Equal(t, `{"a":1}`, r.Body)
	})

	t.Run("absent body decodes to empty string", func(t *testing.T) {
		// Streaming/SSE responses post no body — the wire form omits "body"
		// entirely and we should land on the zero value rather than a
		// silently-introduced placeholder.
		raw := `{"status":200,"headers":{}}`

		var r APIResult
		require.NoError(t, json.Unmarshal([]byte(raw), &r))

		assert.Equal(t, "", r.Body)
	})

	t.Run("explicit empty body string decodes to empty string", func(t *testing.T) {
		raw := `{"status":200,"headers":{},"body":""}`

		var r APIResult
		require.NoError(t, json.Unmarshal([]byte(raw), &r))

		assert.Equal(t, "", r.Body)
	})
}
