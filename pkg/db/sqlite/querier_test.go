package sqlite

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBytesToNullString(t *testing.T) {
	t.Run("nil returns invalid null string", func(t *testing.T) {
		assert.Equal(t, false, bytesToNullString(nil).Valid)
	})

	t.Run("empty slice returns invalid null string", func(t *testing.T) {
		assert.Equal(t, false, bytesToNullString([]byte{}).Valid)
	})

	t.Run("converts bytes to valid null string", func(t *testing.T) {
		input := []byte(`{"key":"value"}`)
		got := bytesToNullString(input)

		assert.True(t, got.Valid)
		assert.Equal(t, `{"key":"value"}`, got.String)
	})
}

func TestToBytes(t *testing.T) {
	t.Run("json raw message returns bytes", func(t *testing.T) {
		input := json.RawMessage(`{"key":"value"}`)
		assert.Equal(t, []byte(`{"key":"value"}`), toBytes(input))
	})

	t.Run("plain bytes still return bytes", func(t *testing.T) {
		input := []byte(`{"key":"value"}`)
		assert.Equal(t, input, toBytes(input))
	})
}
