package duckdb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestQuackEncodeConnectionRequestThenDecodeHeader(t *testing.T) {
	req := quackConnectionRequest{
		AuthString:               "tok",
		ClientDuckDBVersion:      "inngest-quack 0.0.1",
		ClientPlatform:           "go",
		MinSupportedQuackVersion: 1,
		MaxSupportedQuackVersion: 1,
	}
	msg := req.encode()

	r := newQuackReader(msg)
	hdr, err := decodeQuackMessageHeader(r)
	require.NoError(t, err)
	require.Equal(t, byte(quackMsgConnectionRequest), hdr.Type)
	require.Equal(t, "", hdr.ConnectionID) // no connection yet at handshake time
}

func TestQuackDecodeConnectionResponse(t *testing.T) {
	// Build bytes shaped exactly like a real quack_serve ConnectionResponse:
	// header{type=2, connection_id="abc"} body{server_version, platform, quack_version}.
	w := &quackWriter{}
	w.beginObject()
	w.writeByte(1, quackMsgConnectionResponse)
	w.writeString(2, "abc")
	w.writeUint64(3, ^uint64(0))
	w.endObject()
	w.beginObject()
	w.writeStringDefault(1, "v1.5.5")
	w.writeStringDefault(2, "osx_arm64")
	w.writeUint64Default(3, 1)
	w.endObject()

	r := newQuackReader(w.bytes())
	hdr, err := decodeQuackMessageHeader(r)
	require.NoError(t, err)
	require.Equal(t, byte(quackMsgConnectionResponse), hdr.Type)
	require.Equal(t, "abc", hdr.ConnectionID)

	resp, err := decodeQuackConnectionResponseBody(r)
	require.NoError(t, err)
	require.Equal(t, "v1.5.5", resp.ServerDuckDBVersion)
	require.Equal(t, "osx_arm64", resp.ServerPlatform)
	require.EqualValues(t, 1, resp.QuackVersion)
}

func TestQuackDecodeErrorResponse(t *testing.T) {
	w := &quackWriter{}
	w.beginObject()
	w.writeByte(1, quackMsgErrorResponse)
	w.writeUint64(3, ^uint64(0)) // client_query_id is mandatory on the wire, not default-omit
	w.endObject()
	w.beginObject()
	w.writeStringDefault(1, "Table with name t does not exist!")
	w.endObject()

	r := newQuackReader(w.bytes())
	hdr, err := decodeQuackMessageHeader(r)
	require.NoError(t, err)
	require.Equal(t, byte(quackMsgErrorResponse), hdr.Type)

	msg, err := decodeQuackErrorResponseBody(r)
	require.NoError(t, err)
	require.Equal(t, "Table with name t does not exist!", msg)
}

func TestQuackEncodePrepareRequest(t *testing.T) {
	msg := encodeQuackPrepareRequest("conn-1", "SELECT 1")
	r := newQuackReader(msg)
	hdr, err := decodeQuackMessageHeader(r)
	require.NoError(t, err)
	require.Equal(t, byte(quackMsgPrepareRequest), hdr.Type)
	require.Equal(t, "conn-1", hdr.ConnectionID)
}

// buildFlatIntegerChunk constructs the wire bytes for a single-column,
// single-row DataChunk of LogicalTypeId Integer with a Flat vector — the
// same shape verified against a real `duckdb` quack server in exploration.
func buildFlatIntegerChunk(t *testing.T, value int32) []byte {
	t.Helper()
	w := &quackWriter{}
	// DataChunk object: field100 rows, field101 types, field102 columns.
	w.beginObject()
	w.writeUint64(100, 1) // rows
	w.writeFieldID(101)
	w.beginList(1)
	// LogicalType object: field100 id (no field101 type_info).
	w.beginObject()
	w.writeByte(100, 13) // LogicalTypeId.Integer
	w.endObject()
	w.writeFieldID(102)
	w.beginList(1)
	// Vector object (Flat, so field90 omitted): field100 has_validity=false,
	// field102 data.
	w.beginObject()
	w.writeBool(100, false)
	w.writeFieldID(102)
	data := make([]byte, 4)
	data[0] = byte(value)
	data[1] = byte(value >> 8)
	data[2] = byte(value >> 16)
	data[3] = byte(value >> 24)
	w.writeData(data)
	w.endObject()
	w.endObject()
	return w.bytes()
}

func TestQuackDecodeChunkFlatInteger(t *testing.T) {
	body := buildFlatIntegerChunk(t, 42)
	r := newQuackReader(body)
	c, err := decodeQuackDataChunk(r)
	require.NoError(t, err)
	require.Equal(t, 1, c.rowCount)
	require.Len(t, c.columns, 1)
	require.Equal(t, quackLogicalTypeInteger, c.columns[0].typeID)

	values, err := c.columns[0].values()
	require.NoError(t, err)
	require.Equal(t, []any{int64(42)}, values)
}

// buildVarcharJSONChunk builds a one-row DataChunk whose sole column is a
// VARCHAR LogicalType with type_info aliasing it "JSON" — DuckDB's actual
// wire shape for its JSON type (LogicalType::JSON() is a plain VARCHAR with
// SetAlias("JSON"); verified against a real duckdb subprocess: type_info is
// {100: extraTypeInfoKind byte, 101: alias string}, then the object
// terminator, no further fields).
func buildVarcharJSONChunk(t *testing.T, jsonText string) []byte {
	t.Helper()
	w := &quackWriter{}
	w.beginObject()
	w.writeUint64(100, 1) // rows
	w.writeFieldID(101)
	w.beginList(1)
	// LogicalType object: field100 id=VARCHAR, field101 type_info={100: kind, 101: alias}.
	w.beginObject()
	w.writeByte(100, quackLogicalTypeVarchar)
	w.writeFieldID(101)
	w.buf.WriteByte(1) // nullable "present" marker for type_info
	w.beginObject()
	w.writeByte(100, 1) // extraTypeInfoKind — value observed empirically, not otherwise interpreted
	w.writeString(101, "JSON")
	w.endObject()
	w.endObject()
	w.writeFieldID(102)
	w.beginList(1)
	// Vector object (Flat): field100 has_validity=false, field102 data (a
	// list<data> of length 1 for a VARCHAR vector).
	w.beginObject()
	w.writeBool(100, false)
	w.writeFieldID(102)
	w.beginList(1)
	w.writeData([]byte(jsonText))
	w.endObject()
	w.endObject()
	return w.bytes()
}

func TestQuackDecodeChunkVarcharAliasedAsJSONDecodesNestedValue(t *testing.T) {
	body := buildVarcharJSONChunk(t, `{"k":1,"arr":[1,2,3]}`)
	r := newQuackReader(body)
	c, err := decodeQuackDataChunk(r)
	require.NoError(t, err)
	require.Equal(t, "JSON", c.columns[0].alias)

	values, err := c.columns[0].values()
	require.NoError(t, err)
	require.Equal(t, []any{map[string]any{"k": float64(1), "arr": []any{float64(1), float64(2), float64(3)}}}, values)
}

// buildFlatIntegerChunkWithValidity builds a two-row INTEGER DataChunk with a
// validity mask marking rowNull as SQL NULL. Mirrors buildFlatIntegerChunk's
// shape but with has_validity=true and a field101 mask byte inserted before
// field102's data — DuckDB's wire format for a nullable column (verified
// against a real duckdb subprocess: a packed bit mask, LSB-first, 1=valid).
func buildFlatIntegerChunkWithValidity(t *testing.T, values [2]int32, rowNull int) []byte {
	t.Helper()
	w := &quackWriter{}
	w.beginObject()
	w.writeUint64(100, 2) // rows
	w.writeFieldID(101)
	w.beginList(1) // one column
	w.beginObject()
	w.writeByte(100, quackLogicalTypeInteger)
	w.endObject()
	w.writeFieldID(102)
	w.beginList(1) // that one column's Vector
	w.beginObject()
	w.writeBool(100, true)
	w.writeFieldID(101)
	var mask byte
	for i := range 2 {
		if i != rowNull {
			mask |= 1 << uint(i)
		}
	}
	w.writeData([]byte{mask})
	w.writeFieldID(102)
	data := make([]byte, 8)
	for i, v := range values {
		data[i*4] = byte(v)
		data[i*4+1] = byte(v >> 8)
		data[i*4+2] = byte(v >> 16)
		data[i*4+3] = byte(v >> 24)
	}
	w.writeData(data)
	w.endObject()
	w.endObject()
	return w.bytes()
}

func TestQuackDecodeChunkValidityMaskProducesNilForNullRows(t *testing.T) {
	body := buildFlatIntegerChunkWithValidity(t, [2]int32{7, 9}, 1)
	r := newQuackReader(body)
	c, err := decodeQuackDataChunk(r)
	require.NoError(t, err)

	values, err := c.columns[0].values()
	require.NoError(t, err)
	require.Equal(t, []any{int64(7), nil}, values)
}

// TestQuackDecodeChunkUUIDDecodesToStandardString locks in the wire-byte
// pair discovered while fixing this: 'a1b2c3d4-e5f6-4789-a0b1-c2d3e4f56789'
// serializes as these exact 16 bytes on the wire (confirmed against a real
// duckdb subprocess) — DuckDB's UUID-as-signed-hugeint representation,
// reversed byte order with the top bit of the last wire byte flipped.
func TestQuackDecodeChunkUUIDDecodesToStandardString(t *testing.T) {
	wireBytes := []byte{137, 103, 245, 228, 211, 194, 177, 160, 137, 71, 246, 229, 212, 195, 178, 33}

	w := &quackWriter{}
	w.beginObject()
	w.writeUint64(100, 1) // rows
	w.writeFieldID(101)
	w.beginList(1)
	w.beginObject()
	w.writeByte(100, quackLogicalTypeUUID)
	w.endObject()
	w.writeFieldID(102)
	w.beginList(1)
	w.beginObject()
	w.writeBool(100, false)
	w.writeFieldID(102)
	w.writeData(wireBytes)
	w.endObject()
	w.endObject()

	r := newQuackReader(w.bytes())
	c, err := decodeQuackDataChunk(r)
	require.NoError(t, err)

	values, err := c.columns[0].values()
	require.NoError(t, err)
	require.Equal(t, []any{"a1b2c3d4-e5f6-4789-a0b1-c2d3e4f56789"}, values)
}

func TestQuackDecodePrepareResponseBuildsNamedRows(t *testing.T) {
	w := &quackWriter{}
	w.beginObject()
	w.writeByte(1, quackMsgPrepareResponse)
	w.writeString(2, "conn-1")
	w.writeUint64(3, ^uint64(0))
	w.endObject()

	w.beginObject()
	// field1 result_types: list<LogicalType>
	w.writeFieldID(1)
	w.beginList(1)
	w.beginObject()
	w.writeByte(100, 13) // Integer
	w.endObject()
	// field2 result_names: list<string>. List elements are NOT field-tagged
	// (unlike object properties), so this is a raw LEB128-length-prefixed
	// string, not writeString's writeFieldID+string.
	w.writeFieldID(2)
	w.beginList(1)
	w.writeUnsignedLeb128(uint64(len("ok")))
	w.buf.WriteString("ok")
	// field3 needs_more_fetch
	w.writeBool(3, false)
	// field4 results: list<DataChunkWrapper>
	w.writeFieldID(4)
	w.beginList(1)
	w.buf.WriteByte(1) // nullable "present" byte — raw, not field-tagged
	w.beginObject()
	w.writeFieldID(300)
	chunkBytes := buildFlatIntegerChunk(t, 7)
	// The chunk itself is already a fully framed object (with its own
	// terminator), so splice its raw bytes in directly rather than
	// re-serializing.
	w.buf.Write(chunkBytes)
	w.endObject()
	// field5 result_uuid (hugeint: signed-leb128 upper, unsigned-leb128 lower)
	w.writeFieldID(5)
	w.writeSignedLeb128(0)
	w.writeUnsignedLeb128(0)
	w.endObject()

	r := newQuackReader(w.bytes())
	hdr, err := decodeQuackMessageHeader(r)
	require.NoError(t, err)
	require.Equal(t, byte(quackMsgPrepareResponse), hdr.Type)

	names, rows, _, _, err := decodeQuackPrepareResponseBody(r)
	require.NoError(t, err)
	require.Equal(t, []string{"ok"}, names)
	require.Len(t, rows, 1)
	require.Equal(t, int64(7), rows[0]["ok"])
}

func TestQuackTimestampMicrosDecodesToUTCTime(t *testing.T) {
	// Timestamp physical representation is int64 microseconds since epoch.
	want := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	micros := want.UnixMicro()

	w := &quackWriter{}
	w.beginObject()
	w.writeUint64(100, 1)
	w.writeFieldID(101)
	w.beginList(1)
	w.beginObject()
	w.writeByte(100, 19) // LogicalTypeId.Timestamp
	w.endObject()
	w.writeFieldID(102)
	w.beginList(1)
	w.beginObject()
	w.writeBool(100, false)
	w.writeFieldID(102)
	data := make([]byte, 8)
	for i := 0; i < 8; i++ {
		data[i] = byte(micros >> (8 * i))
	}
	w.writeData(data)
	w.endObject()
	w.endObject()

	r := newQuackReader(w.bytes())
	c, err := decodeQuackDataChunk(r)
	require.NoError(t, err)
	values, err := c.columns[0].values()
	require.NoError(t, err)
	require.Len(t, values, 1)
	require.True(t, want.Equal(values[0].(time.Time)), "got %v want %v", values[0], want)
}

// TestQuackDecodeChunkListLogicalTypeMissingChildErrors proves a LIST
// LogicalType with no field200 child (which decodeQuackLogicalType leaves as
// a nil child) is rejected explicitly by decodeQuackListVector rather than
// panicking on a nil dereference — this client has no way to decode a LIST
// vector's child data without knowing its type.
func TestQuackDecodeChunkListLogicalTypeMissingChildErrors(t *testing.T) {
	w := &quackWriter{}
	w.beginObject()
	w.writeUint64(100, 1)
	w.writeFieldID(101)
	w.beginList(1)
	w.beginObject()
	w.writeByte(100, quackLogicalTypeList) // no type_info at all, so no child
	w.endObject()
	w.writeFieldID(102)
	w.beginList(1)
	w.beginObject()
	w.writeBool(100, false)
	// LIST vectors have a different wire shape (field104/105/106); we don't
	// need valid payload bytes here since decode should reject the missing
	// child type before trying to read them.
	w.endObject()
	w.endObject()

	r := newQuackReader(w.bytes())
	_, err := decodeQuackDataChunk(r)
	require.Error(t, err)
}

// buildListVarcharChunk builds a three-row LIST<VARCHAR> DataChunk covering
// a multi-element list, an empty list, and a SQL NULL list — the same shapes
// verified against a real duckdb subprocess (see decodeQuackListVector's doc
// comment): field100 hasValidity, field101 validity mask, field104 the
// flattened child vector's total element count, field105 a list of
// list_entry_t {100: offset, 101: length} objects (one per row, a NULL row's
// entry carrying an unspecified-but-harmless offset/length), and field106
// the child vector itself wrapped in its own object.
func buildListVarcharChunk(t *testing.T) []byte {
	t.Helper()
	w := &quackWriter{}
	w.beginObject()
	w.writeUint64(100, 3) // rows
	w.writeFieldID(101)
	w.beginList(1) // 1 column type
	// LogicalType: id=LIST, type_info={100: kind, 200: child LogicalType VARCHAR}.
	w.beginObject()
	w.writeByte(100, quackLogicalTypeList)
	w.writeFieldID(101)
	w.buf.WriteByte(1) // type_info nullable "present" marker
	w.beginObject()
	w.writeByte(100, 0) // extraTypeInfoKind — unused by this client
	w.writeFieldID(200)
	w.beginObject()
	w.writeByte(100, quackLogicalTypeVarchar)
	w.endObject() // child LogicalType terminator
	w.endObject() // type_info terminator
	w.endObject() // outer LogicalType terminator
	w.writeFieldID(102)
	w.beginList(1) // 1 column's Vector
	w.beginObject()
	w.writeBool(100, true) // hasValidity: row2 is NULL
	w.writeFieldID(101)
	w.writeData([]byte{0b011}) // row0 valid, row1 valid, row2 NULL
	w.writeUint64(104, 2)      // flattened child element count: "a","b"
	w.writeFieldID(105)
	w.beginList(3) // 3 list_entry_t rows
	w.beginObject()
	w.writeUint64(100, 0) // row0 offset
	w.writeUint64(101, 2) // row0 length: ["a","b"]
	w.endObject()
	w.beginObject()
	w.writeUint64(100, 2) // row1 offset
	w.writeUint64(101, 0) // row1 length: [] (empty list)
	w.endObject()
	w.beginObject()
	w.writeUint64(100, 2) // row2 offset (placeholder; row is NULL)
	w.writeUint64(101, 0) // row2 length (placeholder; row is NULL)
	w.endObject()
	w.writeFieldID(106)
	w.beginObject()
	w.writeBool(100, false) // child vector has no nulls
	w.writeFieldID(102)
	w.beginList(2)
	w.writeData([]byte("a"))
	w.writeData([]byte("b"))
	w.endObject() // child vector terminator (consumed inside decodeQuackListVector)
	w.endObject() // outer LIST vector terminator
	w.endObject() // DataChunk terminator
	return w.bytes()
}

func TestQuackDecodeChunkListVarcharDecodesRowsWithEmptyAndNull(t *testing.T) {
	body := buildListVarcharChunk(t)
	r := newQuackReader(body)
	c, err := decodeQuackDataChunk(r)
	require.NoError(t, err)
	require.Equal(t, quackLogicalTypeList, c.columns[0].typeID)

	values, err := c.columns[0].values()
	require.NoError(t, err)
	require.Equal(t, []any{
		[]any{"a", "b"},
		[]any{},
		nil,
	}, values)
}
