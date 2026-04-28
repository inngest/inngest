package sqlc

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/sqlc-dev/pqtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSpanOutputRowToSQLite(t *testing.T) {
	t.Run("preserves raw json payloads", func(t *testing.T) {
		row := &GetSpanOutputRow{
			Input: pqtype.NullRawMessage{
				RawMessage: json.RawMessage(`{"input":{"ok":true}}`),
				Valid:      true,
			},
			Output: pqtype.NullRawMessage{
				RawMessage: json.RawMessage(`{"data":{"num":42}}`),
				Valid:      true,
			},
		}

		got, err := row.ToSQLite()
		require.NoError(t, err)
		assert.Equal(t, json.RawMessage(`{"input":{"ok":true}}`), got.Input)
		assert.Equal(t, json.RawMessage(`{"data":{"num":42}}`), got.Output)
	})

	t.Run("keeps null payloads nil", func(t *testing.T) {
		row := &GetSpanOutputRow{}

		got, err := row.ToSQLite()
		require.NoError(t, err)
		assert.Nil(t, got.Input)
		assert.Nil(t, got.Output)
	})
}

func TestToNullRawMessage(t *testing.T) {
	t.Run("sql null string keeps raw json bytes", func(t *testing.T) {
		got := toNullRawMessage(sql.NullString{
			String: `{"data":{"num":42}}`,
			Valid:  true,
		})

		require.True(t, got.Valid)
		assert.Equal(t, json.RawMessage(`{"data":{"num":42}}`), got.RawMessage)
	})

	t.Run("invalid sql null string stays null", func(t *testing.T) {
		got := toNullRawMessage(sql.NullString{})
		assert.False(t, got.Valid)
	})
}
