package quack

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// This file builds the DataChunk bytes driver.QuackAppender bulk-loads into a
// table; senddata.go drives the actual wire request. It exists because
// every other write goes through PrepareRequest (a real SQL PREPARE over
// interpolated literal text), and binding a giant literal VALUES list scales
// far worse than linearly with statement size. DataChunk's binary columnar
// wire shape sidesteps that: no SQL parsing/binding cost at all.
//
// DuckDB v1.5.5's quack extension sent this as a standalone
// MessageType.APPEND_REQUEST RPC. DuckDB v2.1.0-alpha (protocol version 3)
// removed that message type; in its place is SEND_DATA_REQUEST/RESPONSE, a
// mechanism driven by the server's query executor rather than a
// self-contained RPC (see senddata.go).
//
// Wire shapes below (DataChunk's fields 100/101/102) mirror this file's own
// decode-side counterparts in protocol.go
// (decodeDataChunk/decodeVector/decodeFlatVector) field-for-field;
// this is also the exact shape one bare chunk in SEND_DATA_REQUEST's
// trailing blob needs — no DataChunkWrapper field-300 wrapping the way
// PrepareResponse's chunks get.
//
// Type coverage is scoped to exactly what inngest.run_trace_spans uses today
// (UUID, VARCHAR, VARCHAR-aliased-JSON, TIMESTAMP_MS) — extend as new
// callers need new types.

// ColumnKind identifies one column's physical wire type for
// driver.QuackAppender.AppendRow.
type ColumnKind int

const (
	// ColumnUUID accepts a string (parsed via uuid.Parse) or nil.
	ColumnUUID ColumnKind = iota
	// ColumnVarchar accepts a string or nil.
	ColumnVarchar
	// ColumnJSON accepts a string of raw JSON text (stored verbatim,
	// not re-marshaled) or nil — the VARCHAR-aliased-"JSON" wire shape.
	ColumnJSON
	// ColumnTimestampMS accepts a time.Time or nil.
	ColumnTimestampMS
)

// wireID returns this kind's LogicalTypeId and, for ColumnJSON, its
// wire alias.
//
// A native LIST wire type was prototyped and reverted: the real duckdb-quack
// server crashes (empty-bodied HTTP 500) on any AppendRequest containing a
// LIST-typed column, not fixable from this client alone. Callers needing a
// VARCHAR[] column should instead encode ColumnVarchar (or
// ColumnJSON) array-literal text (e.g. `["a","b"]`) — DuckDB's Appender
// implicit-casts that to LIST.
func (k ColumnKind) wireID() (id byte, alias string) {
	switch k {
	case ColumnUUID:
		return logicalTypeUUID, ""
	case ColumnJSON:
		return logicalTypeVarchar, aliasJSON
	default: // ColumnVarchar, ColumnTimestampMS
		if k == ColumnTimestampMS {
			return logicalTypeTimestampMs, ""
		}
		return logicalTypeVarchar, ""
	}
}

// encodeDataChunk builds the wire bytes for a DataChunk object (field100
// row count, field101 list<LogicalType>, field102 list<Vector>) covering
// every column in cols, populated from rows (one []any per row, in column
// order).
func encodeDataChunk(cols []ColumnKind, rows [][]any) ([]byte, error) {
	w := &writer{}
	w.beginObject()
	w.writeUint64(100, uint64(len(rows)))

	w.writeFieldID(101)
	w.beginList(uint64(len(cols)))
	for _, k := range cols {
		id, alias := k.wireID()
		encodeLogicalType(w, id, alias)
	}

	w.writeFieldID(102)
	w.beginList(uint64(len(cols)))
	for i, k := range cols {
		if err := encodeVectorColumn(w, k, rows, i); err != nil {
			return nil, fmt.Errorf("column %d: %w", i, err)
		}
	}
	w.endObject()
	return w.bytes(), nil
}

// encodeLogicalType writes one LogicalType object body: field100 id,
// and — only when alias is non-empty — field101 type_info, in the shape
// decodeLogicalType expects.
func encodeLogicalType(w *writer, id byte, alias string) {
	w.beginObject()
	w.writeByte(100, id)
	if alias != "" {
		w.writeFieldID(101)
		w.buf.WriteByte(1) // presence marker
		w.beginObject()
		w.writeByte(100, 1) // extraTypeInfoKind — only its presence is interpreted, matching decode
		w.writeString(101, alias)
		w.endObject()
	}
	w.endObject()
}

// encodeVectorColumn writes one Vector object body (field100
// has_validity, optional field101 validity mask, field102 data) for column
// index colIdx across every row, dispatching on kind's physical shape.
func encodeVectorColumn(w *writer, kind ColumnKind, rows [][]any, colIdx int) error {
	n := len(rows)
	// DuckDB's ValidityMask is an array of uint64_t words, not a tightly
	// packed byte array — the mask must be padded to a multiple of 8 bytes or
	// the server's deserializer reads past the end of a too-short buffer.
	mask := make([]byte, ((n+63)/64)*8)
	hasNull := false
	for i, row := range rows {
		if row[colIdx] == nil {
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

	switch kind {
	case ColumnUUID:
		w.writeFieldID(102)
		data := make([]byte, n*16)
		for i, row := range rows {
			if row[colIdx] == nil {
				continue // zero-filled placeholder; validity marks it invalid
			}
			id, err := valueToUUID(row[colIdx])
			if err != nil {
				return err
			}
			wire := encodeUUID(id)
			copy(data[i*16:], wire[:])
		}
		w.writeData(data)
	case ColumnTimestampMS:
		w.writeFieldID(102)
		data := make([]byte, n*8)
		for i, row := range rows {
			if row[colIdx] == nil {
				continue
			}
			micros, err := valueToTimestampMS(row[colIdx])
			if err != nil {
				return err
			}
			putLE64(data[i*8:i*8+8], uint64(micros))
		}
		w.writeData(data)
	case ColumnVarchar, ColumnJSON:
		if err := encodeVarcharVectorData(w, rows, colIdx); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported ColumnKind %d", kind)
	}
	w.endObject()
	return nil
}

// encodeVarcharVectorData writes a VARCHAR Vector's data fields (100
// has_validity / 101 validity mask already written by the caller): the
// inverse of decodeVarcharFlatVector's field107/108/109 shape.
//
//   - field107: total byte length of every row's string data added together.
//   - field108: one little-endian uint32 byte-length per row, in row order —
//     a NULL row gets a real zero-length entry, never omitted.
//   - field109: every row's string bytes concatenated back-to-back, no
//     separator.
func encodeVarcharVectorData(w *writer, rows [][]any, colIdx int) error {
	lengths := make([]byte, len(rows)*4)
	var allData []byte
	for i, row := range rows {
		if row[colIdx] == nil {
			continue // zero length already in place; validity marks it invalid
		}
		s, ok := row[colIdx].(string)
		if !ok {
			return fmt.Errorf("row %d: expected string, got %T", i, row[colIdx])
		}
		binary.LittleEndian.PutUint32(lengths[i*4:i*4+4], uint32(len(s)))
		allData = append(allData, s...)
	}

	w.writeFieldID(107)
	w.writeUnsignedLeb128(uint64(len(allData)))
	w.writeFieldID(108)
	w.writeData(lengths)
	w.writeFieldID(109)
	w.writeData(allData)
	return nil
}

func valueToUUID(v any) (uuid.UUID, error) {
	switch val := v.(type) {
	case string:
		return uuid.Parse(val)
	case uuid.UUID:
		return val, nil
	default:
		return uuid.UUID{}, fmt.Errorf("expected string or uuid.UUID, got %T", v)
	}
}

func valueToTimestampMS(v any) (int64, error) {
	t, ok := v.(timeLike)
	if !ok {
		return 0, fmt.Errorf("expected time.Time, got %T", v)
	}
	return t.UnixMilli(), nil
}

// timeLike avoids importing "time" solely for a type assertion signature;
// time.Time satisfies it structurally.
type timeLike interface{ UnixMilli() int64 }

// encodeUUID is the exact inverse of protocol.go's
// uuidToString: reverse the 16 bytes, then flip the top bit of the
// resulting last byte.
func encodeUUID(id uuid.UUID) [16]byte {
	var b [16]byte
	for i := range b {
		b[i] = id[15-i]
	}
	b[15] ^= 0x80
	return b
}

// Identifier double-quotes a DuckDB identifier, doubling any embedded
// double quote. Used only for the "USE <catalog>;" statement
// driver.NewQuackAppender issues; treated as injection-sensitive per this package's
// convention even though every caller passes DuckLakeAlias, not external
// stringLiteral quotes s as a SQL string literal. Only used for values this
// package generates itself (stream IDs), never caller input.
func stringLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// input.
func Identifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// putLE64 writes v little-endian into dst[0:8].
func putLE64(dst []byte, v uint64) {
	for i := range 8 {
		dst[i] = byte(v >> (8 * i))
	}
}
