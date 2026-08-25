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

	rows, _, err := decodeQuackPrepareResponseBody(r)
	require.NoError(t, err)
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

func TestQuackDecodeChunkUnsupportedTypeErrors(t *testing.T) {
	w := &quackWriter{}
	w.beginObject()
	w.writeUint64(100, 1)
	w.writeFieldID(101)
	w.beginList(1)
	w.beginObject()
	w.writeByte(100, 101) // LogicalTypeId.List — not in our supported subset
	w.endObject()
	w.writeFieldID(102)
	w.beginList(1)
	w.beginObject()
	w.writeBool(100, false)
	// LIST vectors have a different wire shape (field104/105/106); we don't
	// need valid payload bytes here since decode should reject the type
	// before trying to read them.
	w.endObject()
	w.endObject()

	r := newQuackReader(w.bytes())
	_, err := decodeQuackDataChunk(r)
	require.Error(t, err)
}
