package apiresult

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIResult_MarshalJSON(t *testing.T) {
	t.Run("body is encoded as a JSON string, not base64", func(t *testing.T) {
		r := APIResult{
			StatusCode: 200,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       []byte(`{"hello":"world"}`),
		}

		got, err := json.Marshal(r)
		require.NoError(t, err)

		// Decode into a generic map to inspect wire form.
		var wire map[string]any
		require.NoError(t, json.Unmarshal(got, &wire))

		assert.Equal(t, float64(200), wire["status"])
		assert.Equal(t, `{"hello":"world"}`, wire["body"])
	})

	t.Run("Duration is not serialized", func(t *testing.T) {
		r := APIResult{StatusCode: 200, Duration: 12345}
		got, err := json.Marshal(r)
		require.NoError(t, err)
		assert.NotContains(t, string(got), "duration")
		assert.NotContains(t, string(got), "Duration")
	})
}

func TestAPIResult_UnmarshalJSON(t *testing.T) {
	t.Run("decodes body string into []byte", func(t *testing.T) {
		raw := `{"status":201,"headers":{"X-Foo":"bar"},"body":"{\"a\":1}","version":2}`

		var r APIResult
		require.NoError(t, json.Unmarshal([]byte(raw), &r))

		assert.Equal(t, 201, r.StatusCode)
		assert.Equal(t, map[string]string{"X-Foo": "bar"}, r.Headers)
		assert.Equal(t, []byte(`{"a":1}`), r.Body)
	})
}
