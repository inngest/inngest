package quack

import (
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestEncodeVectorsMatchesRows checks the column-major encoder writes the
// same bytes as the row encoder for the same values, across every kind,
// NULLs, chunk boundaries, and NULL rows that still carry string bytes.
func TestEncodeVectorsMatchesRows(t *testing.T) {
	kinds := []ColumnKind{ColumnUUID, ColumnVarchar, ColumnJSON, ColumnTimestampMS, ColumnBool, ColumnBigint}
	for _, n := range []int{1, 7, StandardVectorSize, StandardVectorSize + 1, 5000} {
		for _, nullBytes := range []bool{false, true} {
			r := rand.New(rand.NewSource(int64(n)))
			rows := make([][]any, n)
			for i := range rows {
				rows[i] = make([]any, len(kinds))
			}
			vectors := make([]Vector, len(kinds))
			for c, k := range kinds {
				v := Vector{Kind: k, Nulls: make([]bool, n)}
				for i := range n {
					null := r.Intn(5) == 0
					v.Nulls[i] = null
					switch k {
					case ColumnUUID:
						id := uuid.New()
						v.UUIDs = append(v.UUIDs, id)
						if !null {
							rows[i][c] = id
						}
					case ColumnTimestampMS:
						ms := r.Int63n(4_000_000_000_000)
						v.Int64s = append(v.Int64s, ms)
						if !null {
							rows[i][c] = time.UnixMilli(ms)
						}
					case ColumnBigint:
						x := r.Int63() - r.Int63()
						v.Int64s = append(v.Int64s, x)
						if !null {
							rows[i][c] = x
						}
					case ColumnBool:
						b := r.Intn(2) == 0
						v.Bools = append(v.Bools, b)
						if !null {
							rows[i][c] = b
						}
					default:
						s := randString(r)
						if null && !nullBytes {
							s = ""
						}
						v.Bytes = append(v.Bytes, s...)
						v.Ends = append(v.Ends, len(v.Bytes))
						if !null {
							rows[i][c] = s
						}
					}
				}
				vectors[c] = v
			}

			wantBlob, wantChunks, err := rowsToChunks(kinds, rows)
			require.NoError(t, err)
			gotBlob, gotChunks, err := EncodeVectors(vectors)
			require.NoError(t, err)
			require.Equal(t, wantChunks, gotChunks, "n=%d", n)
			require.Equal(t, wantBlob, gotBlob, "n=%d nullBytes=%v", n, nullBytes)
		}
	}
}

func TestEncodeVectorsRejectsRaggedColumns(t *testing.T) {
	_, _, err := EncodeVectors([]Vector{
		{Kind: ColumnBigint, Int64s: []int64{1, 2}},
		{Kind: ColumnBool, Bools: []bool{true}},
	})
	require.ErrorContains(t, err, "column 1 has 1 rows, want 2")
}

func randString(r *rand.Rand) string {
	const chars = `abcdefghijklmnopqrstuvwxyz0123456789 "\{}é漢`
	rs := []rune(chars)
	b := make([]rune, r.Intn(40))
	for i := range b {
		b[i] = rs[r.Intn(len(rs))]
	}
	return string(b)
}
