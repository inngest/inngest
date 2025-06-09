package exechttp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerializableRequestEncoding(t *testing.T) {
	t.Run("it should pass through JSON", func(t *testing.T) {
		jsonContent := json.RawMessage(`{"key": "value"}`)

		r := SerializableRequest{
			Method: http.MethodPost,
			Body:   jsonContent,
			URL:    "https://example.com",
			Header: map[string][]string{
				"Content-Type": {"application/json"},
			},
		}

		encoded, err := json.Marshal(r)
		require.NoError(t, err)

		assert.Equal(t, `{"Method":"POST","Body":{"key":"value"},"URL":"https://example.com","Header":{"Content-Type":["application/json"]}}`, string(encoded))
	})

	t.Run("json.Marshal always escapes HTML (bad)", func(t *testing.T) {
		t.Run("with regular field", func(t *testing.T) {
			type test struct {
				Name string `json:"name"`
			}

			tt := test{"this&will&be&escaped"}
			encoded, err := json.Marshal(tt)
			require.NoError(t, err)
			assert.Equal(t, `{"name":"this\u0026will\u0026be\u0026escaped"}`, string(encoded))
		})

		t.Run("with RawMessage", func(t *testing.T) {
			type test struct {
				PassThrough json.RawMessage `json:"passThrough"`
			}

			tt := test{json.RawMessage(`{"name": "this&will&be&escaped"}`)}
			encoded, err := json.Marshal(tt)
			require.NoError(t, err)
			assert.Equal(t, `{"passThrough":{"name":"this\u0026will\u0026be\u0026escaped"}}`, string(encoded))
		})

		t.Run("with *RawMessage", func(t *testing.T) {
			type test struct {
				PassThrough *json.RawMessage `json:"passThrough"`
			}

			rm := json.RawMessage(`{"name": "this&will&be&escaped"}`)

			tt := test{&rm}
			encoded, err := json.Marshal(tt)
			require.NoError(t, err)
			assert.Equal(t, `{"passThrough":{"name":"this\u0026will\u0026be\u0026escaped"}}`, string(encoded))
		})
	})

	t.Run("it should not double encode JSON", func(t *testing.T) {
		jsonContent := json.RawMessage(`{"key": "&value"}`)

		r := SerializableRequest{
			Method: http.MethodPost,
			Body:   jsonContent,
			URL:    "https://example.com&test",
			Header: map[string][]string{
				"Content-Type": {"application/json"},
			},
		}

		reader, err := r.Bytes()
		require.NoError(t, err)

		byt, err := io.ReadAll(reader)
		require.NoError(t, err)

		assert.Equal(t, `{"Method":"POST","Body":{"key":"&value"},"URL":"https://example.com&test","Header":{"Content-Type":["application/json"]}}`, string(byt))
	})

	t.Run("it should rehydrate request", func(t *testing.T) {
		serialized := bytes.NewBuffer([]byte(`{"Method":"POST","Body":{"key":"&value"},"URL":"https://example.com&test","Header":{"Content-Type":["application/json"]}}`))

		parsed := SerializableRequest{}
		err := json.NewDecoder(serialized).Decode(&parsed)
		require.NoError(t, err)

		req, err := parsed.HTTPRequest()
		require.NoError(t, err)

		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "https://example.com&test", req.URL.String())
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

		bodyBytes, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.Equal(t, `{"key":"&value"}`, string(bodyBytes))
	})
}
