package duckdb

import (
	"encoding/json"
	"fmt"
	"math"
	"time"
)

// Quack message types. Mirrors duckdb_quack::MessageType
// (src/include/quack_message.hpp in duckdb/duckdb-quack v1.5-variegata),
// via the quack-net client's documented port.
const (
	quackMsgConnectionRequest  byte = 1
	quackMsgConnectionResponse byte = 2
	quackMsgPrepareRequest     byte = 3
	quackMsgPrepareResponse    byte = 4
	quackMsgErrorResponse      byte = 100
)

// Every quack message is two top-level objects back to back: a MessageHeader
// (field1 type, field2 connection_id default-omit, field3 client_query_id —
// an optional_idx whose "not set" sentinel is uint64 max) followed by the
// message-specific body object.
const quackOptionalIdxInvalid = ^uint64(0)

type quackMessageHeader struct {
	Type         byte
	ConnectionID string
}

func encodeQuackMessage(msgType byte, connectionID string, writeBody func(w *quackWriter)) []byte {
	w := &quackWriter{}
	w.beginObject()
	w.writeByte(1, msgType)
	w.writeStringDefault(2, connectionID)
	w.writeUint64(3, quackOptionalIdxInvalid)
	w.endObject()
	w.beginObject()
	writeBody(w)
	w.endObject()
	return w.bytes()
}

// decodeQuackMessageHeader reads the header object and leaves r positioned at
// the start of the body object.
func decodeQuackMessageHeader(r *quackReader) (quackMessageHeader, error) {
	r.beginObject()
	if err := r.beginProperty(1); err != nil {
		return quackMessageHeader{}, err
	}
	msgType, err := r.readByte()
	if err != nil {
		return quackMessageHeader{}, err
	}

	var connID string
	present, err := r.tryBeginProperty(2)
	if err != nil {
		return quackMessageHeader{}, err
	}
	if present {
		connID, err = r.readString()
		if err != nil {
			return quackMessageHeader{}, err
		}
	}

	if err := r.beginProperty(3); err != nil {
		return quackMessageHeader{}, err
	}
	if _, err := r.readUInt64(); err != nil { // client_query_id, unused by this client
		return quackMessageHeader{}, err
	}
	if err := r.endObject(); err != nil {
		return quackMessageHeader{}, err
	}

	r.beginObject() // positions r at the start of the body object
	return quackMessageHeader{Type: msgType, ConnectionID: connID}, nil
}

// ---------- ConnectionRequest / ConnectionResponse ----------

type quackConnectionRequest struct {
	AuthString               string
	ClientDuckDBVersion      string
	ClientPlatform           string
	MinSupportedQuackVersion uint64
	MaxSupportedQuackVersion uint64
}

func (m quackConnectionRequest) encode() []byte {
	return encodeQuackMessage(quackMsgConnectionRequest, "", func(w *quackWriter) {
		w.writeStringDefault(1, m.AuthString)
		w.writeStringDefault(2, m.ClientDuckDBVersion)
		w.writeStringDefault(3, m.ClientPlatform)
		w.writeUint64Default(4, m.MinSupportedQuackVersion)
		w.writeUint64Default(5, m.MaxSupportedQuackVersion)
	})
}

type quackConnectionResponse struct {
	ServerDuckDBVersion string
	ServerPlatform      string
	QuackVersion        uint64
}

func decodeQuackConnectionResponseBody(r *quackReader) (quackConnectionResponse, error) {
	var resp quackConnectionResponse
	if ok, err := r.tryBeginProperty(1); err != nil {
		return resp, err
	} else if ok {
		v, err := r.readString()
		if err != nil {
			return resp, err
		}
		resp.ServerDuckDBVersion = v
	}
	if ok, err := r.tryBeginProperty(2); err != nil {
		return resp, err
	} else if ok {
		v, err := r.readString()
		if err != nil {
			return resp, err
		}
		resp.ServerPlatform = v
	}
	if ok, err := r.tryBeginProperty(3); err != nil {
		return resp, err
	} else if ok {
		v, err := r.readUInt64()
		if err != nil {
			return resp, err
		}
		resp.QuackVersion = v
	}
	if err := r.endObject(); err != nil {
		return resp, err
	}
	return resp, nil
}

// ---------- ErrorResponse ----------

func decodeQuackErrorResponseBody(r *quackReader) (string, error) {
	var message string
	if ok, err := r.tryBeginProperty(1); err != nil {
		return "", err
	} else if ok {
		v, err := r.readString()
		if err != nil {
			return "", err
		}
		message = v
	}
	if err := r.endObject(); err != nil {
		return "", err
	}
	return message, nil
}

// ---------- PrepareRequest / PrepareResponse ----------

func encodeQuackPrepareRequest(connectionID, sql string) []byte {
	return encodeQuackMessage(quackMsgPrepareRequest, connectionID, func(w *quackWriter) {
		w.writeStringDefault(1, sql)
	})
}

// decodeQuackPrepareResponseBody reads a PrepareResponse body and returns
// rows keyed by result column name, built from every inline DataChunk the
// server returned. needsMoreFetch signals the query has more rows than fit
// in this response; this client does not implement FetchRequest (see
// quack_session.go), so a caller seeing needsMoreFetch=true should treat it
// as an error rather than silently returning a truncated result.
func decodeQuackPrepareResponseBody(r *quackReader) (rows []map[string]any, needsMoreFetch bool, err error) {
	var names []string
	if ok, terr := r.tryBeginProperty(1); terr != nil {
		return nil, false, terr
	} else if ok {
		n, terr := r.beginList()
		if terr != nil {
			return nil, false, terr
		}
		for i := uint64(0); i < n; i++ {
			// LogicalType object: consumed here only to stay positioned
			// correctly on the wire; decodeQuackDataChunk re-reads each
			// chunk's own (equivalent) per-column types independently.
			if _, terr := decodeQuackLogicalType(r); terr != nil {
				return nil, false, terr
			}
		}
	}

	if ok, terr := r.tryBeginProperty(2); terr != nil {
		return nil, false, terr
	} else if ok {
		n, terr := r.beginList()
		if terr != nil {
			return nil, false, terr
		}
		names = make([]string, n)
		for i := range names {
			names[i], terr = r.readString()
			if terr != nil {
				return nil, false, terr
			}
		}
	}

	if ok, terr := r.tryBeginProperty(3); terr != nil {
		return nil, false, terr
	} else if ok {
		needsMoreFetch, terr = r.readBool()
		if terr != nil {
			return nil, false, terr
		}
	}

	if ok, terr := r.tryBeginProperty(4); terr != nil {
		return nil, false, terr
	} else if ok {
		n, terr := r.beginList()
		if terr != nil {
			return nil, false, terr
		}
		for i := uint64(0); i < n; i++ {
			present, terr := r.beginNullable()
			if terr != nil {
				return nil, false, terr
			}
			if !present {
				continue
			}
			// DataChunkWrapper is itself an object wrapping field 300, whose
			// value is the DataChunk object decodeQuackDataChunk reads
			// (including that inner object's own terminator) — so an outer
			// endObject is still needed here for the wrapper's terminator.
			r.beginObject()
			if terr := r.beginProperty(300); terr != nil {
				return nil, false, terr
			}
			chunk, terr := decodeQuackDataChunk(r)
			if terr != nil {
				return nil, false, terr
			}
			if terr := r.endObject(); terr != nil {
				return nil, false, terr
			}
			chunkRows, terr := chunk.namedRows(names)
			if terr != nil {
				return nil, false, terr
			}
			rows = append(rows, chunkRows...)
		}
	}

	// field5 result_uuid: hugeint (signed-leb128 upper, unsigned-leb128
	// lower). Unused by this client — Exec never needs to correlate a
	// follow-up Fetch against it since NeedsMoreFetch is treated as an
	// error, not a "call Fetch" signal.
	if err := r.beginProperty(5); err != nil {
		return nil, false, err
	}
	if _, err := r.readSignedLeb128(); err != nil {
		return nil, false, err
	}
	if _, err := r.readUnsignedLeb128(); err != nil {
		return nil, false, err
	}

	if err := r.endObject(); err != nil {
		return nil, false, err
	}
	return rows, needsMoreFetch, nil
}

// ---------- LogicalType ----------

// quackLogicalTypeInteger and friends are DuckDB's LogicalTypeId values
// (src/include/duckdb/common/types.hpp in duckdb/duckdb) for the subset this
// client supports decoding: what's actually exercised by this phase's health
// check, goose's version table, DDL/INSERT result rows, and the query-layer's
// runs/events/spans reads (see pkg/cqrs/duckdbquery). Anything else (UUID,
// structs, ...) is a read-side concern deferred to whenever a caller actually
// needs it — see decodeQuackDataChunk's default case.
const (
	quackLogicalTypeBoolean      byte = 10
	quackLogicalTypeSmallInt     byte = 12
	quackLogicalTypeInteger      byte = 13
	quackLogicalTypeBigInt       byte = 14
	quackLogicalTypeTimestampSec byte = 17
	quackLogicalTypeTimestampMs  byte = 18
	quackLogicalTypeTimestamp    byte = 19
	quackLogicalTypeTimestampNs  byte = 20
	quackLogicalTypeDouble       byte = 23
	quackLogicalTypeVarchar      byte = 25
	quackLogicalTypeUUID         byte = 54
	quackLogicalTypeList         byte = 101
)

// quackAliasJSON is the LogicalType alias DuckDB's JSON type carries: it's
// physically a VARCHAR (LogicalType::JSON() is
// `LogicalType(LogicalTypeId::VARCHAR)` with `SetAlias("JSON")` — no
// structural difference from plain VARCHAR on the wire), distinguished only
// by this alias in its ExtraTypeInfo. Confirmed against a real duckdb
// subprocess: a JSON column's LogicalType serializes as id=25 (VARCHAR) with
// type_info = {100: extraTypeInfoKind (byte, value 1 observed), 101: "JSON"}.
const quackAliasJSON = "JSON"

// quackLogicalType is a decoded LogicalType: the base id, its alias if any
// ("" when untagged — e.g. "JSON" for DuckDB's JSON type), and, for LIST
// (id == quackLogicalTypeList), the element type nested at type_info's
// field200.
type quackLogicalType struct {
	id    byte
	alias string
	child *quackLogicalType
}

// decodeQuackLogicalType reads a LogicalType object (field100 id, optional
// field101 type_info). type_info, when present, is ExtraTypeInfo's own
// object — {100: extraTypeInfoKind byte, 101: alias string, 200: nested
// child LogicalType (LIST only), optionally more structured fields for
// types like DECIMAL/ENUM this client doesn't understand}. Only this shape
// (verified empirically for DuckDB's JSON and LIST types) is decoded;
// anything with additional fields beyond that — surfaced as an unexpected
// field id where the terminator was expected — still errors, exactly as
// before, since this client has no way to interpret those types' extra
// structure.
func decodeQuackLogicalType(r *quackReader) (quackLogicalType, error) {
	r.beginObject()
	if err := r.beginProperty(100); err != nil {
		return quackLogicalType{}, err
	}
	id, err := r.readByte()
	if err != nil {
		return quackLogicalType{}, err
	}
	lt := quackLogicalType{id: id}
	if ok, err := r.tryBeginProperty(101); err != nil {
		return quackLogicalType{}, err
	} else if ok {
		present, err := r.beginNullable()
		if err != nil {
			return quackLogicalType{}, err
		}
		if present {
			r.beginObject()
			if err := r.beginProperty(100); err != nil {
				return quackLogicalType{}, fmt.Errorf("duckdb: quack LogicalType id %d has unsupported extended type_info: %w", id, err)
			}
			if _, err := r.readByte(); err != nil { // extraTypeInfoKind; only used to stay positioned on the wire
				return quackLogicalType{}, err
			}
			if ok, err := r.tryBeginProperty(101); err != nil {
				return quackLogicalType{}, err
			} else if ok {
				lt.alias, err = r.readString()
				if err != nil {
					return quackLogicalType{}, err
				}
			}
			if ok, err := r.tryBeginProperty(200); err != nil {
				return quackLogicalType{}, err
			} else if ok {
				child, cerr := decodeQuackLogicalType(r)
				if cerr != nil {
					return quackLogicalType{}, cerr
				}
				lt.child = &child
			}
			if err := r.endObject(); err != nil {
				return quackLogicalType{}, fmt.Errorf("duckdb: quack LogicalType id %d has extended type_info this client does not support: %w", id, err)
			}
		}
	}
	if err := r.endObject(); err != nil {
		return quackLogicalType{}, err
	}
	return lt, nil
}

// ---------- DataChunk / Vector ----------

type quackColumn struct {
	typeID byte
	// alias is the LogicalType's ExtraTypeInfo alias, if any ("" for a plain
	// unaliased type) — e.g. "JSON" for DuckDB's JSON type, which is
	// otherwise a physically ordinary VARCHAR (see quackAliasJSON).
	alias string
	// validity is nil when the column has no NULLs (DuckDB omits the mask
	// entirely in that case — see decodeFlatVector); otherwise one entry per
	// row, true = valid, false = SQL NULL.
	validity []bool
	// Exactly one of these is populated, chosen by typeID's physical shape.
	// DuckDB still writes a real (unspecified/garbage) entry at a NULL row's
	// position in both, so indexing is always safe — values() just discards
	// whatever's there and substitutes nil when validity says so.
	fixedData []byte // constant-size types, column-major, no per-row length
	varchar   [][]byte
	// list holds one entry per row for a LIST column: nil for a SQL NULL
	// row, otherwise a []any of the row's decoded child values (empty, not
	// nil, for a zero-length list). Populated directly by
	// decodeQuackListVector rather than derived in values(), since building
	// it requires the list_entry_t offsets/lengths that aren't otherwise
	// kept on quackColumn.
	list     []any
	rowCount int
}

// isNull reports whether row i is a SQL NULL.
func (c quackColumn) isNull(i int) bool {
	return c.validity != nil && !c.validity[i]
}

// values decodes the column's raw bytes into canonical driver.Value-shaped Go
// types: nil (for a NULL row), bool, int64, float64, string, time.Time, or —
// for a VARCHAR aliased as JSON — the value's own json.Unmarshal result
// (map[string]any, []any, a scalar, or nil), matching the shape the
// stdio/-jsonlines transport already produces for the same column type (that
// transport's own JSON-lines encoding round-trips a JSON-typed column's
// value as nested JSON automatically; quack has no equivalent for free, so
// this replicates it explicitly).
func (c quackColumn) values() ([]any, error) {
	out := make([]any, c.rowCount)
	switch c.typeID {
	case quackLogicalTypeBoolean:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = c.fixedData[i] != 0
		}
	case quackLogicalTypeSmallInt:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = int64(int16(le16(c.fixedData[i*2:])))
		}
	case quackLogicalTypeInteger:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = int64(int32(le32(c.fixedData[i*4:])))
		}
	case quackLogicalTypeBigInt:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = int64(le64(c.fixedData[i*8:]))
		}
	case quackLogicalTypeDouble:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = le64ToFloat64(le64(c.fixedData[i*8:]))
		}
	case quackLogicalTypeVarchar:
		if c.alias == quackAliasJSON {
			for i := range out {
				if c.isNull(i) {
					continue
				}
				var v any
				if err := json.Unmarshal(c.varchar[i], &v); err != nil {
					return nil, fmt.Errorf("duckdb: quack column decode: unmarshaling JSON column value: %w", err)
				}
				out[i] = v
			}
			return out, nil
		}
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = string(c.varchar[i])
		}
	case quackLogicalTypeTimestamp, quackLogicalTypeTimestampMs, quackLogicalTypeTimestampSec, quackLogicalTypeTimestampNs:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			raw := int64(le64(c.fixedData[i*8:]))
			out[i] = quackTimestampToTime(c.typeID, raw)
		}
	case quackLogicalTypeUUID:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = quackUUIDToString(c.fixedData[i*16 : i*16+16])
		}
	case quackLogicalTypeList:
		copy(out, c.list)
	default:
		return nil, fmt.Errorf("duckdb: quack column decode: unsupported LogicalTypeId %d", c.typeID)
	}
	return out, nil
}

func quackTimestampToTime(typeID byte, raw int64) time.Time {
	switch typeID {
	case quackLogicalTypeTimestampSec:
		return time.Unix(raw, 0).UTC()
	case quackLogicalTypeTimestampMs:
		return time.UnixMilli(raw).UTC()
	case quackLogicalTypeTimestampNs:
		return time.Unix(0, raw).UTC()
	default: // microseconds (Timestamp)
		return time.UnixMicro(raw).UTC()
	}
}

// quackUUIDToString decodes a UUID column's 16-byte wire representation into
// standard "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" text. DuckDB stores a UUID
// internally as a signed hugeint (so it can reuse hugeint comparison for
// ordering) built from the UUID's 16 bytes with the sign bit of the first
// byte flipped, then serializes that hugeint little-endian — which, from the
// wire's perspective, is the UUID's bytes in *reverse* order with the *last*
// wire byte's top bit flipped instead. Confirmed against a real duckdb
// subprocess by round-tripping a known UUID: reversing b and then XORing
// byte 0 with 0x80 reconstructs the original UUID bytes exactly.
func quackUUIDToString(b []byte) string {
	var u [16]byte
	for i := range u {
		u[i] = b[15-i]
	}
	u[0] ^= 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}

func le16(b []byte) uint16 { return uint16(b[0]) | uint16(b[1])<<8 }
func le32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}
func le64(b []byte) uint64 {
	var v uint64
	for i := 0; i < 8; i++ {
		v |= uint64(b[i]) << (8 * i)
	}
	return v
}
func le64ToFloat64(bits uint64) float64 {
	return math.Float64frombits(bits)
}

type quackDataChunk struct {
	rowCount int
	columns  []quackColumn
}

// namedRows zips this chunk's columns against names (from the enclosing
// PrepareResponse's result_names) into one map[string]any per row, matching
// the shape process.exec has always returned (see sqlExecer).
func (c quackDataChunk) namedRows(names []string) ([]map[string]any, error) {
	if len(names) != len(c.columns) {
		return nil, fmt.Errorf("duckdb: quack result has %d names but %d columns", len(names), len(c.columns))
	}
	colValues := make([][]any, len(c.columns))
	for i, col := range c.columns {
		vs, err := col.values()
		if err != nil {
			return nil, err
		}
		colValues[i] = vs
	}
	rows := make([]map[string]any, c.rowCount)
	for r := 0; r < c.rowCount; r++ {
		row := make(map[string]any, len(names))
		for i, name := range names {
			row[name] = colValues[i][r]
		}
		rows[r] = row
	}
	return rows, nil
}

// decodeQuackDataChunk reads a DataChunk object: field100 rows, field101
// types (list<LogicalType>), field102 columns (list of Vector objects).
func decodeQuackDataChunk(r *quackReader) (quackDataChunk, error) {
	r.beginObject()
	if err := r.beginProperty(100); err != nil {
		return quackDataChunk{}, err
	}
	rowCount, err := r.readUInt32()
	if err != nil {
		return quackDataChunk{}, err
	}

	if err := r.beginProperty(101); err != nil {
		return quackDataChunk{}, err
	}
	typesCount, err := r.beginList()
	if err != nil {
		return quackDataChunk{}, err
	}
	types := make([]quackLogicalType, typesCount)
	for i := range types {
		types[i], err = decodeQuackLogicalType(r)
		if err != nil {
			return quackDataChunk{}, err
		}
	}

	var columns []quackColumn
	if ok, terr := r.tryBeginProperty(102); terr != nil {
		return quackDataChunk{}, terr
	} else if ok {
		colCount, terr := r.beginList()
		if terr != nil {
			return quackDataChunk{}, terr
		}
		if colCount != typesCount {
			return quackDataChunk{}, fmt.Errorf("duckdb: quack DataChunk has %d types but %d columns", typesCount, colCount)
		}
		columns = make([]quackColumn, colCount)
		for i := range columns {
			// Matches DataChunk::Deserialize: the object begin/end wrapping
			// a column's Vector is done by the DataChunk's own columns
			// loop, not by Vector::Deserialize itself.
			r.beginObject()
			col, terr := decodeQuackVector(r, types[i], int(rowCount))
			if terr != nil {
				return quackDataChunk{}, terr
			}
			if terr := r.endObject(); terr != nil {
				return quackDataChunk{}, terr
			}
			columns[i] = col
		}
	}

	if err := r.endObject(); err != nil {
		return quackDataChunk{}, err
	}
	return quackDataChunk{rowCount: int(rowCount), columns: columns}, nil
}

// decodeQuackVector reads one column's Vector object. Only VectorType.Flat
// (the wire default, field90 omitted) is supported — see the package doc on
// scope; Constant/Sequence/Dictionary vectors are a future-work decode
// concern once this client needs to support arbitrary SELECT results rather
// than just DDL/INSERT result rows and simple health-check queries.
func decodeQuackVector(r *quackReader, lt quackLogicalType, count int) (quackColumn, error) {
	if ok, err := r.tryBeginProperty(90); err != nil {
		return quackColumn{}, err
	} else if ok {
		vectorType, err := r.readByte()
		if err != nil {
			return quackColumn{}, err
		}
		if vectorType != 0 { // 0 == VectorType.Flat
			return quackColumn{}, fmt.Errorf("duckdb: quack vector type %d is not supported by this client (flat only)", vectorType)
		}
	}
	if lt.id == quackLogicalTypeList {
		return decodeQuackListVector(r, lt, count)
	}
	return decodeFlatVector(r, lt.id, lt.alias, count)
}

// decodeQuackListVector reads a LIST Vector object. Confirmed against a real
// duckdb subprocess (INTEGER[] and VARCHAR[] columns, with rows covering a
// NULL list, an empty list, and multi-element lists) by cross-referencing
// decoded offsets/lengths and child values against the known input:
//
//   - field100: hasValidity (for the list_entry_t data — whether a given
//     row's whole list is itself SQL NULL), same convention as
//     decodeFlatVector.
//   - field101: validity mask, present only when hasValidity is true — same
//     shape as decodeFlatVector's.
//   - field104: the flattened child vector's total element count
//     (DuckDB's ListVector::GetListSize()), a plain ULEB128.
//   - field105: a list (ULEB128 count, expected to equal count) of
//     list_entry_t objects, each {100: offset ULEB128, 101: length
//     ULEB128}. A NULL row's entry is still present with an
//     unspecified/garbage-but-typically-zero offset/length, exactly like a
//     NULL fixed-size row's data — validity governs it, not the entry.
//   - field106: the flattened child vector itself, wrapped in its own
//     object exactly like DataChunk's own per-column wrapping, decoded
//     recursively via decodeQuackVector so a LIST of LIST would also work.
func decodeQuackListVector(r *quackReader, lt quackLogicalType, count int) (quackColumn, error) {
	if lt.child == nil {
		return quackColumn{}, fmt.Errorf("duckdb: quack LIST LogicalType is missing its child type")
	}

	if err := r.beginProperty(100); err != nil {
		return quackColumn{}, err
	}
	hasValidity, err := r.readBool()
	if err != nil {
		return quackColumn{}, err
	}
	var validity []bool
	if hasValidity {
		if err := r.beginProperty(101); err != nil {
			return quackColumn{}, err
		}
		maskBytes, err := r.readData()
		if err != nil {
			return quackColumn{}, err
		}
		validity = decodeQuackValidityMask(maskBytes, count)
	}

	if err := r.beginProperty(104); err != nil {
		return quackColumn{}, err
	}
	childCount, err := r.readUnsignedLeb128()
	if err != nil {
		return quackColumn{}, err
	}

	if err := r.beginProperty(105); err != nil {
		return quackColumn{}, err
	}
	entryCount, err := r.beginList()
	if err != nil {
		return quackColumn{}, err
	}
	if int(entryCount) != count {
		return quackColumn{}, fmt.Errorf("duckdb: quack LIST vector has %d list_entry_t entries, expected %d", entryCount, count)
	}
	type listEntry struct{ offset, length uint64 }
	entries := make([]listEntry, count)
	for i := range entries {
		r.beginObject()
		if err := r.beginProperty(100); err != nil {
			return quackColumn{}, err
		}
		entries[i].offset, err = r.readUnsignedLeb128()
		if err != nil {
			return quackColumn{}, err
		}
		if err := r.beginProperty(101); err != nil {
			return quackColumn{}, err
		}
		entries[i].length, err = r.readUnsignedLeb128()
		if err != nil {
			return quackColumn{}, err
		}
		if err := r.endObject(); err != nil {
			return quackColumn{}, err
		}
	}

	if err := r.beginProperty(106); err != nil {
		return quackColumn{}, err
	}
	r.beginObject()
	child, err := decodeQuackVector(r, *lt.child, int(childCount))
	if err != nil {
		return quackColumn{}, err
	}
	if err := r.endObject(); err != nil {
		return quackColumn{}, err
	}

	childValues, err := child.values()
	if err != nil {
		return quackColumn{}, err
	}
	lists := make([]any, count)
	for i, e := range entries {
		if validity != nil && !validity[i] {
			continue
		}
		lists[i] = append([]any{}, childValues[e.offset:e.offset+e.length]...)
	}
	return quackColumn{typeID: quackLogicalTypeList, list: lists, rowCount: count}, nil
}

func decodeFlatVector(r *quackReader, typeID byte, alias string, count int) (quackColumn, error) {
	if err := r.beginProperty(100); err != nil {
		return quackColumn{}, err
	}
	hasValidity, err := r.readBool()
	if err != nil {
		return quackColumn{}, err
	}
	// field101 (validity mask), present only when hasValidity is true, is a
	// packed-bit mask covering count rows, one bit per row, LSB-first within
	// each byte, 1=valid/0=null (confirmed against a real duckdb subprocess:
	// a 2-row column with only its second row NULL produced mask byte 0xFD =
	// 0b11111101 — bit 0 set, bit 1 clear). field102 (the column's data) is
	// always present regardless, at the same physical width/shape as an
	// all-valid column — DuckDB still writes a (unspecified/garbage) entry
	// for a null row rather than omitting it, so decoding proceeds exactly
	// as before and values() alone is responsible for substituting nil at
	// invalid positions.
	var validity []bool
	if hasValidity {
		if err := r.beginProperty(101); err != nil {
			return quackColumn{}, err
		}
		maskBytes, err := r.readData()
		if err != nil {
			return quackColumn{}, err
		}
		validity = decodeQuackValidityMask(maskBytes, count)
	}

	if typeID == quackLogicalTypeVarchar {
		if err := r.beginProperty(102); err != nil {
			return quackColumn{}, err
		}
		listCount, err := r.beginList()
		if err != nil {
			return quackColumn{}, err
		}
		if int(listCount) != count {
			return quackColumn{}, fmt.Errorf("duckdb: quack VARCHAR vector has %d entries, expected %d", listCount, count)
		}
		values := make([][]byte, count)
		for i := range values {
			values[i], err = r.readData()
			if err != nil {
				return quackColumn{}, err
			}
		}
		return quackColumn{typeID: typeID, alias: alias, validity: validity, varchar: values, rowCount: count}, nil
	}

	if err := r.beginProperty(102); err != nil {
		return quackColumn{}, err
	}
	data, err := r.readData()
	if err != nil {
		return quackColumn{}, err
	}
	return quackColumn{typeID: typeID, alias: alias, validity: validity, fixedData: data, rowCount: count}, nil
}

// decodeQuackValidityMask unpacks a DuckDB ValidityMask's packed-bit wire
// representation into one bool per row (true = valid, false = SQL NULL).
// mask may be shorter than ceil(count/8) bytes — DuckDB only grows a
// validity mask's backing storage to cover bits it has actually cleared, so
// any row past the mask's end is implicitly valid.
func decodeQuackValidityMask(mask []byte, count int) []bool {
	valid := make([]bool, count)
	for i := range valid {
		byteIdx, bitIdx := i/8, uint(i%8)
		if byteIdx >= len(mask) {
			valid[i] = true
			continue
		}
		valid[i] = mask[byteIdx]&(1<<bitIdx) != 0
	}
	return valid
}
