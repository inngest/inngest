package quack

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestQuackEncodeUUIDMatchesPinnedWireBytes reuses the exact wire-byte pair
// quack_protocol_test.go's TestQuackDecodeChunkUUIDDecodesToStandardString
// already pins (confirmed against a real duckdb subprocess), proving
// encodeUUID is the true inverse of the decode side's transform rather
// than just self-consistent.
func TestQuackEncodeUUIDMatchesPinnedWireBytes(t *testing.T) {
	want := []byte{137, 103, 245, 228, 211, 194, 177, 160, 137, 71, 246, 229, 212, 195, 178, 33}
	id := uuid.MustParse("a1b2c3d4-e5f6-4789-a0b1-c2d3e4f56789")

	got := encodeUUID(id)
	require.Equal(t, want, got[:])
}

// TestQuackEncodeDataChunkDecodesBackToOriginalValues round-trips a
// multi-column, multi-row chunk (every ColumnKind, plus a NULL in each
// column) purely through this client's own encode/decode pair — no
// subprocess needed, since the wire format is symmetric and the decode side
// is already verified against a real duckdb subprocess elsewhere
// (quack_protocol_test.go).
func TestQuackEncodeDataChunkDecodesBackToOriginalValues(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	ts1 := time.UnixMilli(1_700_000_000_123).UTC()
	ts2 := time.UnixMilli(1_800_000_000_456).UTC()

	cols := []ColumnKind{ColumnUUID, ColumnVarchar, ColumnJSON, ColumnTimestampMS}
	rows := [][]any{
		{id1.String(), "span-a", `{"k":"v","n":1}`, ts1},
		{nil, nil, nil, nil},
		{id2.String(), "span-b", `{"arr":[1,2,3]}`, ts2},
	}

	chunk, err := encodeDataChunk(cols, rows)
	require.NoError(t, err)

	r := newReader(chunk)
	c, err := decodeDataChunk(r)
	require.NoError(t, err)
	require.Equal(t, 3, c.rowCount)
	require.Len(t, c.columns, 4)

	uuidVals, err := c.columns[0].values()
	require.NoError(t, err)
	require.Equal(t, []any{id1.String(), nil, id2.String()}, uuidVals)

	varcharVals, err := c.columns[1].values()
	require.NoError(t, err)
	require.Equal(t, []any{"span-a", nil, "span-b"}, varcharVals)

	require.Equal(t, "JSON", c.columns[2].alias)
	jsonVals, err := c.columns[2].values()
	require.NoError(t, err)
	require.Equal(t, []any{
		map[string]any{"k": "v", "n": float64(1)},
		nil,
		map[string]any{"arr": []any{float64(1), float64(2), float64(3)}},
	}, jsonVals)

	tsVals, err := c.columns[3].values()
	require.NoError(t, err)
	require.Len(t, tsVals, 3)
	require.True(t, ts1.Equal(tsVals[0].(time.Time)))
	require.Nil(t, tsVals[1])
	require.True(t, ts2.Equal(tsVals[2].(time.Time)))
}
