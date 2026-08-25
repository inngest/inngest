package duckdb

import (
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
// check, goose's version table, and DDL/INSERT result rows. Anything else
// (JSON, UUID, structs, ...) is a read-side concern deferred to the
// query-layer spec — see decodeQuackDataChunk's default case.
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
)

// decodeQuackLogicalType reads a LogicalType object (field100 id, optional
// field101 type_info). It errors if type_info is present since none of this
// client's supported types carry one.
func decodeQuackLogicalType(r *quackReader) (byte, error) {
	r.beginObject()
	if err := r.beginProperty(100); err != nil {
		return 0, err
	}
	id, err := r.readByte()
	if err != nil {
		return 0, err
	}
	if ok, err := r.tryBeginProperty(101); err != nil {
		return 0, err
	} else if ok {
		present, err := r.beginNullable()
		if err != nil {
			return 0, err
		}
		if present {
			return 0, fmt.Errorf("duckdb: quack LogicalType id %d has extended type_info, which this client does not support", id)
		}
	}
	if err := r.endObject(); err != nil {
		return 0, err
	}
	return id, nil
}

// ---------- DataChunk / Vector ----------

type quackColumn struct {
	typeID byte
	// Exactly one of these is populated, chosen by typeID's physical shape.
	fixedData []byte // constant-size types, column-major, no per-row length
	varchar   [][]byte
	rowCount  int
}

// values decodes the column's raw bytes into canonical driver.Value-shaped Go
// types: bool, int64, float64, string, or time.Time. NULLs are not handled —
// this client's supported statements (health check, goose's version table,
// DDL/INSERT result rows) never produce one; see decodeFlatVector.
func (c quackColumn) values() ([]any, error) {
	out := make([]any, c.rowCount)
	switch c.typeID {
	case quackLogicalTypeBoolean:
		for i := range out {
			out[i] = c.fixedData[i] != 0
		}
	case quackLogicalTypeSmallInt:
		for i := range out {
			out[i] = int64(int16(le16(c.fixedData[i*2:])))
		}
	case quackLogicalTypeInteger:
		for i := range out {
			out[i] = int64(int32(le32(c.fixedData[i*4:])))
		}
	case quackLogicalTypeBigInt:
		for i := range out {
			out[i] = int64(le64(c.fixedData[i*8:]))
		}
	case quackLogicalTypeDouble:
		for i := range out {
			out[i] = le64ToFloat64(le64(c.fixedData[i*8:]))
		}
	case quackLogicalTypeVarchar:
		for i := range out {
			out[i] = string(c.varchar[i])
		}
	case quackLogicalTypeTimestamp, quackLogicalTypeTimestampMs, quackLogicalTypeTimestampSec, quackLogicalTypeTimestampNs:
		for i := range out {
			raw := int64(le64(c.fixedData[i*8:]))
			out[i] = quackTimestampToTime(c.typeID, raw)
		}
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
	typeIDs := make([]byte, typesCount)
	for i := range typeIDs {
		typeIDs[i], err = decodeQuackLogicalType(r)
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
			col, terr := decodeQuackVector(r, typeIDs[i], int(rowCount))
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
func decodeQuackVector(r *quackReader, typeID byte, count int) (quackColumn, error) {
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
	return decodeFlatVector(r, typeID, count)
}

func decodeFlatVector(r *quackReader, typeID byte, count int) (quackColumn, error) {
	if err := r.beginProperty(100); err != nil {
		return quackColumn{}, err
	}
	hasValidity, err := r.readBool()
	if err != nil {
		return quackColumn{}, err
	}
	if hasValidity {
		// This client's supported statements never produce NULL columns
		// (see quackColumn.values), so a present validity mask is
		// unexpected data this client can't interpret correctly rather than
		// something safe to silently ignore.
		return quackColumn{}, fmt.Errorf("duckdb: quack column has a validity mask (nullable result), which this client does not support decoding")
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
		return quackColumn{typeID: typeID, varchar: values, rowCount: count}, nil
	}

	if err := r.beginProperty(102); err != nil {
		return quackColumn{}, err
	}
	data, err := r.readData()
	if err != nil {
		return quackColumn{}, err
	}
	return quackColumn{typeID: typeID, fixedData: data, rowCount: count}, nil
}
