package quack

import (
	"encoding/binary"
	"fmt"

	"github.com/google/uuid"
)

// Vector is one column of a chunk in column-major form, so a producer that
// already holds columns (a ClickHouse block) can be encoded without boxing
// every value in an interface. Only the field matching Kind is read:
//
//   - ColumnUUID: UUIDs
//   - ColumnTimestampMS: Int64s, as Unix milliseconds
//   - ColumnBigint: Int64s
//   - ColumnBool: Bools
//   - ColumnVarchar, ColumnJSON: Bytes and Ends; row i's bytes are
//     Bytes[Ends[i-1]:Ends[i]] (row 0 starts at 0)
//
// Nulls, when non-nil, marks NULL rows (true is NULL); a NULL row's value is
// ignored.
type Vector struct {
	Kind   ColumnKind
	Nulls  []bool
	UUIDs  []uuid.UUID
	Int64s []int64
	Bools  []bool
	Bytes  []byte
	Ends   []int
}

// Len is the vector's row count.
func (v Vector) Len() int {
	switch v.Kind {
	case ColumnUUID:
		return len(v.UUIDs)
	case ColumnTimestampMS, ColumnBigint:
		return len(v.Int64s)
	case ColumnBool:
		return len(v.Bools)
	default:
		return len(v.Ends)
	}
}

// EncodeVectors encodes equal-length vectors as DataChunks of at most
// StandardVectorSize rows: the same bytes rowsToChunks writes for the same
// values in row form.
func EncodeVectors(vectors []Vector) (blob []byte, chunkCount uint64, err error) {
	if len(vectors) == 0 {
		return nil, 0, fmt.Errorf("duckdb: quack vectors: no columns")
	}
	n := vectors[0].Len()
	for i, v := range vectors {
		if v.Len() != n {
			return nil, 0, fmt.Errorf("duckdb: quack vectors: column %d has %d rows, want %d", i, v.Len(), n)
		}
		if v.Nulls != nil && len(v.Nulls) != n {
			return nil, 0, fmt.Errorf("duckdb: quack vectors: column %d has %d null flags, want %d", i, len(v.Nulls), n)
		}
	}
	for start := 0; start < n; start += StandardVectorSize {
		end := min(start+StandardVectorSize, n)
		chunk, err := encodeVectorChunk(vectors, start, end)
		if err != nil {
			return nil, 0, fmt.Errorf("duckdb: quack vectors: chunk %d: %w", chunkCount, err)
		}
		blob = append(blob, chunk...)
		chunkCount++
	}
	return blob, chunkCount, nil
}

func encodeVectorChunk(vectors []Vector, start, end int) ([]byte, error) {
	w := &writer{}
	w.beginObject()
	w.writeUint64(100, uint64(end-start))

	w.writeFieldID(101)
	w.beginList(uint64(len(vectors)))
	for _, v := range vectors {
		id, alias := v.Kind.wireID()
		encodeLogicalType(w, id, alias)
	}

	w.writeFieldID(102)
	w.beginList(uint64(len(vectors)))
	for i, v := range vectors {
		if err := encodeTypedVector(w, v, start, end); err != nil {
			return nil, fmt.Errorf("column %d: %w", i, err)
		}
	}
	w.endObject()
	return w.bytes(), nil
}

func (v Vector) null(i int) bool { return v.Nulls != nil && v.Nulls[i] }

// encodeTypedVector is encodeVectorColumn for rows [start, end) of v.
func encodeTypedVector(w *writer, v Vector, start, end int) error {
	n := end - start
	mask := make([]byte, ((n+63)/64)*8)
	hasNull := false
	for i := range n {
		if v.null(start + i) {
			hasNull = true
			continue
		}
		mask[i/8] |= 1 << uint(i%8)
	}

	w.beginObject()
	if hasNull {
		w.writeBool(100, true)
		w.writeFieldID(101)
		w.writeData(mask)
	} else {
		w.writeBool(100, false)
	}

	switch v.Kind {
	case ColumnUUID:
		w.writeFieldID(102)
		data := make([]byte, n*16)
		for i := range n {
			if v.null(start + i) {
				continue
			}
			wire := encodeUUID(v.UUIDs[start+i])
			copy(data[i*16:], wire[:])
		}
		w.writeData(data)
	case ColumnTimestampMS, ColumnBigint:
		w.writeFieldID(102)
		data := make([]byte, n*8)
		for i := range n {
			if v.null(start + i) {
				continue
			}
			putLE64(data[i*8:i*8+8], uint64(v.Int64s[start+i]))
		}
		w.writeData(data)
	case ColumnBool:
		w.writeFieldID(102)
		data := make([]byte, n)
		for i := range n {
			if !v.null(start+i) && v.Bools[start+i] {
				data[i] = 1
			}
		}
		w.writeData(data)
	case ColumnVarchar, ColumnJSON:
		encodeTypedVarchar(w, v, start, end)
	default:
		return fmt.Errorf("unsupported ColumnKind %d", v.Kind)
	}
	w.endObject()
	return nil
}

// encodeTypedVarchar writes a VARCHAR vector's data fields, as
// encodeVarcharVectorData does. When no NULL row carries bytes, the rows'
// bytes are already back to back in v.Bytes and go out as one slice.
func encodeTypedVarchar(w *writer, v Vector, start, end int) {
	n := end - start
	from := 0
	if start > 0 {
		from = v.Ends[start-1]
	}
	lengths := make([]byte, n*4)
	contiguous := true
	prev := from
	for i := range n {
		e := v.Ends[start+i]
		l := e - prev
		if v.null(start + i) {
			if l != 0 {
				contiguous = false
			}
			l = 0
		}
		binary.LittleEndian.PutUint32(lengths[i*4:i*4+4], uint32(l))
		prev = e
	}

	var data []byte
	if contiguous {
		data = v.Bytes[from:v.Ends[end-1]]
	} else {
		prev = from
		for i := range n {
			e := v.Ends[start+i]
			if !v.null(start + i) {
				data = append(data, v.Bytes[prev:e]...)
			}
			prev = e
		}
	}
	if n == 0 {
		data = nil
	}

	w.writeFieldID(107)
	w.writeUnsignedLeb128(uint64(len(data)))
	w.writeFieldID(108)
	w.writeData(lengths)
	w.writeFieldID(109)
	w.writeData(data)
}
