package worker

import (
	"encoding/binary"
	"testing"
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
