package tracing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnyToBytes(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		assert.Nil(t, anyToBytes(nil))
	})

	t.Run("empty string returns nil", func(t *testing.T) {
		assert.Nil(t, anyToBytes(""))
	})

	t.Run("empty byte slice returns nil", func(t *testing.T) {
		assert.Nil(t, anyToBytes([]byte{}))
	})

	t.Run("string passed through without double encoding", func(t *testing.T) {
		// This is the critical regression test: a JSON string like
		// `{"data":{"num":12}}` must NOT be wrapped in extra quotes.
		input := `{"data":{"num":12}}`
		got := anyToBytes(input)
		assert.Equal(t, []byte(input), got)
		// Verify it doesn't start with a quote (double-encoding symptom)
		assert.NotEqual(t, byte('"'), got[0], "output should not be double-encoded")
	})

	t.Run("byte slice passed through as-is", func(t *testing.T) {
		input := []byte(`{"key":"value"}`)
		got := anyToBytes(input)
		assert.Equal(t, input, got)
	})

	t.Run("other types are JSON marshaled", func(t *testing.T) {
		got := anyToBytes(map[string]int{"x": 1})
		assert.Equal(t, []byte(`{"x":1}`), got)
	})
}
