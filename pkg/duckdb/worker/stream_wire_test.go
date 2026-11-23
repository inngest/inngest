package worker

import (
	"encoding/binary"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDecodeBatchRejectsMalformedBatches(t *testing.T) {
	t.Run("row count exceeds StandardVectorSize", func(t *testing.T) {
		buf := make([]byte, 4+6)
		binary.LittleEndian.PutUint32(buf[0:4], 1)                      // chunkCount = 1
		binary.LittleEndian.PutUint32(buf[4:8], StandardVectorSize+1)   // rowCount = 2049
		binary.LittleEndian.PutUint16(buf[8:10], 0)                     // colCount = 0
		if _, err := DecodeBatch(buf); err == nil {
			t.Fatal("expected error for rowCount > StandardVectorSize")
		}
	})

	t.Run("non-monotonic varchar offsets", func(t *testing.T) {
		buf := make([]byte, 0, 32)
		buf = binary.LittleEndian.AppendUint32(buf, 1) // chunkCount = 1
		buf = binary.LittleEndian.AppendUint32(buf, 2) // rowCount = 2
		buf = binary.LittleEndian.AppendUint16(buf, 1) // colCount = 1
		buf = append(buf, byte(StreamColumnVarchar), 0)
		buf = binary.LittleEndian.AppendUint32(buf, 4) // totalBytes = 4
		buf = binary.LittleEndian.AppendUint32(buf, 3) // row 0 end = 3
		buf = binary.LittleEndian.AppendUint32(buf, 2) // row 1 end = 2 (< prev 3)
		buf = append(buf, "abcd"...)
		if _, err := DecodeBatch(buf); err == nil {
			t.Fatal("expected error for non-monotonic varchar offsets")
		}
	})

	t.Run("out-of-bounds varchar offset", func(t *testing.T) {
		buf := make([]byte, 0, 32)
		buf = binary.LittleEndian.AppendUint32(buf, 1) // chunkCount = 1
		buf = binary.LittleEndian.AppendUint32(buf, 1) // rowCount = 1
		buf = binary.LittleEndian.AppendUint16(buf, 1) // colCount = 1
		buf = append(buf, byte(StreamColumnVarchar), 0)
		buf = binary.LittleEndian.AppendUint32(buf, 2) // totalBytes = 2
		buf = binary.LittleEndian.AppendUint32(buf, 5) // row 0 end = 5 (> totalBytes 2)
		buf = append(buf, "ab"...)
		if _, err := DecodeBatch(buf); err == nil {
			t.Fatal("expected error for out-of-bounds varchar offset")
		}
	})

	t.Run("huge chunkCount rejected before allocation", func(t *testing.T) {
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf[0:4], 1_000_000)
		if _, err := DecodeBatch(buf); err == nil {
			t.Fatal("expected error for truncated batch chunks")
		}
	})
}

func TestSplitBatchBlobSplitsWideChunksAndRoundTrips(t *testing.T) {
	cols := []StreamColumnKind{
		StreamColumnUUID,
		StreamColumnVarchar,
		StreamColumnTimestampTZ,
		StreamColumnBigint,
		StreamColumnBool,
	}
	baseTime := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	const nRows = 16
	rows := make([][]any, nRows)
	uuids := make([]uuid.UUID, nRows)
	for i := 0; i < nRows; i++ {
		uuids[i] = uuid.New()
		var s any = strings.Repeat("a", 2048) + string(rune('A'+i))
		var b any = i%2 == 0
		if i == 5 {
			s = nil
			b = nil
		}
		rows[i] = []any{
			uuids[i],
			s,
			baseTime.Add(time.Duration(i) * time.Second),
			int64(i * 100),
			b,
		}
	}

	blob, _, err := EncodeRowsBatch(cols, rows)
	if err != nil {
		t.Fatalf("EncodeRowsBatch: %v", err)
	}
	const maxBytes = 10 << 10
	if len(blob) <= maxBytes {
		t.Fatalf("test blob too small: %d <= %d", len(blob), maxBytes)
	}

	parts, err := splitBatchBlob(blob, maxBytes)
	if err != nil {
		t.Fatalf("splitBatchBlob: %v", err)
	}
	if len(parts) < 3 {
		t.Fatalf("expected >= 3 split parts, got %d", len(parts))
	}

	rowIdx := 0
	for pIdx, part := range parts {
		if len(part) > maxBytes {
			t.Fatalf("part %d size %d exceeds maxBytes %d", pIdx, len(part), maxBytes)
		}
		chunks, err := DecodeBatch(part)
		if err != nil {
			t.Fatalf("DecodeBatch(part %d): %v", pIdx, err)
		}
		for _, ch := range chunks {
			for r := 0; r < ch.RowCount; r++ {
				if ch.Columns[0].UUIDs[r] != uuids[rowIdx] {
					t.Fatalf("row %d uuid mismatch", rowIdx)
				}
				if rowIdx == 5 {
					if ch.Columns[1].Nulls == nil || !ch.Columns[1].Nulls[r] {
						t.Fatalf("row 5 varchar expected null")
					}
					if ch.Columns[4].Nulls == nil || !ch.Columns[4].Nulls[r] {
						t.Fatalf("row 5 bool expected null")
					}
				} else {
					var start uint32
					if r > 0 {
						start = ch.Columns[1].Ends[r-1]
					}
					gotStr := string(ch.Columns[1].Bytes[start:ch.Columns[1].Ends[r]])
					wantStr := strings.Repeat("a", 2048) + string(rune('A'+rowIdx))
					if gotStr != wantStr {
						t.Fatalf("row %d varchar len=%d, want %d", rowIdx, len(gotStr), len(wantStr))
					}
					if ch.Columns[4].Bools[r] != (rowIdx%2 == 0) {
						t.Fatalf("row %d bool mismatch", rowIdx)
					}
				}
				wantMicros := baseTime.Add(time.Duration(rowIdx)*time.Second).UnixMilli() * 1000
				if ch.Columns[2].Int64s[r] != wantMicros {
					t.Fatalf("row %d ts = %d, want %d", rowIdx, ch.Columns[2].Int64s[r], wantMicros)
				}
				if ch.Columns[3].Int64s[r] != int64(rowIdx*100) {
					t.Fatalf("row %d bigint = %d, want %d", rowIdx, ch.Columns[3].Int64s[r], rowIdx*100)
				}
				rowIdx++
			}
		}
	}
	if rowIdx != nRows {
		t.Fatalf("decoded %d rows, want %d", rowIdx, nRows)
	}
}
