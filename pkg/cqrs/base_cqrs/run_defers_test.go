package base_cqrs

import (
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/require"
)

// TestDecodeOpcodes exercises the loose-typed coercion of history.Result.RawOutput
// into a typed []state.GeneratorOpcode. The parent run's history table stores
// raw_output as either a JSON-encoded string of an opcode array (the real
// shape observed in dev-server SQLite) or the array itself. Both must work.
func TestDecodeOpcodes(t *testing.T) {
	deferAddArr := `[{"id":"hash1","op":"DeferAdd","name":"foo","opts":{"fn_slug":"slug-a","input":{}},"userland":{"id":"foo"}}]`

	t.Run("string-wrapped JSON array (real-world shape)", func(t *testing.T) {
		// raw_output unmarshals into a Go string holding the array text.
		ops, err := decodeOpcodes(deferAddArr)
		require.NoError(t, err)
		require.Len(t, ops, 1)
		require.Equal(t, enums.OpcodeDeferAdd, ops[0].Op)
		require.Equal(t, "hash1", ops[0].ID)
		require.NotNil(t, ops[0].Userland)
		require.Equal(t, "foo", ops[0].Userland.ID)
	})

	t.Run("raw bytes", func(t *testing.T) {
		ops, err := decodeOpcodes([]byte(deferAddArr))
		require.NoError(t, err)
		require.Len(t, ops, 1)
	})

	t.Run("json.RawMessage", func(t *testing.T) {
		ops, err := decodeOpcodes(json.RawMessage(deferAddArr))
		require.NoError(t, err)
		require.Len(t, ops, 1)
	})

	t.Run("any-typed slice from generic JSON unmarshal", func(t *testing.T) {
		var v any
		require.NoError(t, json.Unmarshal([]byte(deferAddArr), &v))
		ops, err := decodeOpcodes(v)
		require.NoError(t, err)
		require.Len(t, ops, 1)
		require.Equal(t, enums.OpcodeDeferAdd, ops[0].Op)
	})

	t.Run("nil returns nothing", func(t *testing.T) {
		ops, err := decodeOpcodes(nil)
		require.NoError(t, err)
		require.Nil(t, ops)
	})

	t.Run("garbage returns error", func(t *testing.T) {
		_, err := decodeOpcodes("not json")
		require.Error(t, err)
	})
}
