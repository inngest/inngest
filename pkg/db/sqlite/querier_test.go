package sqlite

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBytesToRawMessage(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		assert.Nil(t, bytesToRawMessage(nil))
	})

	t.Run("empty slice returns nil", func(t *testing.T) {
		assert.Nil(t, bytesToRawMessage([]byte{}))
	})

	t.Run("converts bytes to raw message", func(t *testing.T) {
		input := []byte(`{"key":"value"}`)
		got := bytesToRawMessage(input)

		assert.Equal(t, json.RawMessage(`{"key":"value"}`), got)
	})
}
