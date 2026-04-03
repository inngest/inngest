package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBytesToAny(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		assert.Nil(t, bytesToAny(nil))
	})

	t.Run("empty slice returns nil", func(t *testing.T) {
		assert.Nil(t, bytesToAny([]byte{}))
	})

	t.Run("converts bytes to string", func(t *testing.T) {
		// Critical: SQLite JSON columns require TEXT, not BLOB.
		// []byte passed as interface{} to database/sql is stored as BLOB,
		// but string is stored as TEXT. json_group_array/json_object fail
		// on BLOB values with "JSON cannot hold BLOB values".
		input := []byte(`{"key":"value"}`)
		got := bytesToAny(input)

		s, ok := got.(string)
		assert.True(t, ok, "result must be a string, not []byte")
		assert.Equal(t, `{"key":"value"}`, s)
	})
}
