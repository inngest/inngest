package duckdb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// This file implements MessageType.APPEND_REQUEST (id 9 — see
// quack_protocol.go), quack's native bulk-insert message: a schema/table
// name plus one binary DataChunk, applied directly to the table with no SQL
// parsing or binding at all. It exists because every write this client
// otherwise makes goes through PrepareRequest (a real SQL PREPARE over
// fully-interpolated literal text — see quack_session.go's exec doc
// comment), and binding a single INSERT's giant literal VALUES list scales
// far worse than linearly with statement size (empirically: jsonlines holds
// ~42µs/row from a 1,000- to a 10,000-row batch; quack via PrepareRequest
// goes from ~9µs/row-marginal to ~860µs/row). AppendRequest sidesteps that
// entirely — it was already present in the exact protocol version this
// client targets (duckdb-quack branch v1.5-variegata, matching duckdb
// v1.5.5), so this isn't new server-side functionality, only a client gap.
//
// Wire shapes below (AppendRequestMessage's fields, DataChunkWrapper's field
// 300, DataChunk's fields 100/101/102) are taken directly from
// duckdb-quack's serialize_quack_message.cpp/quack_message.cpp at that
// branch, and from this file's own decode-side counterparts
// (quack_protocol.go's decodeQuackDataChunk/decodeQuackVector/
// decodeFlatVector), which are already verified against a real duckdb
// subprocess — the wire format is symmetric, so encoding mirrors decoding
// field-for-field. AppendRequestMessage's single (non-list) "append_chunk"
// unique_ptr<DataChunkWrapper> property is written with the same
// field-id-then-presence-byte-then-object shape observed on the decode side
// for LogicalType's type_info and for PrepareResponse's list<DataChunkWrapper>
// results, since DuckDB's BinarySerializer applies that shape uniformly to
// any WritePropertyWithDefault<unique_ptr<T>>/shared_ptr<T>> property,
// scalar or not — confirmed empirically in quack_append_test.go's
// integration test against a real subprocess.
//
// Type coverage is intentionally scoped to exactly what inngest.run_trace_spans
// uses today (UUID, VARCHAR, VARCHAR-aliased-JSON, TIMESTAMP_MS) — extend as
// new callers need new types, matching the read-side decode's own
// established "implement only what's been exercised" pattern (see
// docs/plans/008-duckdb-gql-resolvers-plan.md's account of how the JSON,
// NULL-validity, and UUID decode gaps were each found and fixed in turn).

// QuackColumnKind identifies one column's physical wire type for
// QuackAppender.AppendRow.
type QuackColumnKind int

const (
	// QuackColumnUUID accepts a string (parsed via uuid.Parse) or nil.
	QuackColumnUUID QuackColumnKind = iota
	// QuackColumnVarchar accepts a string or nil.
	QuackColumnVarchar
	// QuackColumnJSON accepts a string of raw JSON text (stored verbatim,
	// not re-marshaled) or nil — the VARCHAR-aliased-"JSON" wire shape.
	QuackColumnJSON
	// QuackColumnTimestampMS accepts a time.Time or nil.
	QuackColumnTimestampMS
)

// wireID returns this kind's LogicalTypeId and, for QuackColumnJSON, its
// wire alias.
//
// A native LIST wire type (for e.g. inngest.runs.event_ids, VARCHAR[]) was
// prototyped here and reverted: the client-side encoding round-tripped
// correctly through this file's own decoder, but the real duckdb-quack
// server (v1.5-variegata branch) returns an empty-bodied HTTP 500 for any
// AppendRequest containing a LIST-typed column — verified down to the
// minimal single-row, single-element case, and confirmed via
// duckdb-quack's own server source (quack_server.cpp) that the crash
// happens before the append handler's try/catch (which does gracefully
// convert a std::exception to an ErrorResponse), most likely during
// DuckDB core's DataChunk::Deserialize for the LIST vector. duckdb-quack's
// own test suite has no LIST/array coverage for the append/DML path
// either. Not fixable from this client alone.
//
// The actual need (event_ids) doesn't require the native LIST type at all:
// DuckDB's Appender does implicit VARCHAR->LIST casting from array-literal
// text (e.g. `["a","b"]`, or `[]` for empty, or nil for NULL) — verified
// empirically against a real duckdb-quack server for the empty/NULL/
// multi-element cases. Callers needing a VARCHAR[]-typed column should
// encode it as QuackColumnVarchar (or QuackColumnJSON) array-literal text
// instead.
func (k QuackColumnKind) wireID() (id byte, alias string) {
	switch k {
	case QuackColumnUUID:
		return quackLogicalTypeUUID, ""
	case QuackColumnJSON:
		return quackLogicalTypeVarchar, quackAliasJSON
	default: // QuackColumnVarchar, QuackColumnTimestampMS
		if k == QuackColumnTimestampMS {
			return quackLogicalTypeTimestampMs, ""
		}
		return quackLogicalTypeVarchar, ""
	}
}

// QuackAppender bulk-loads rows into one table over quack's AppendRequest
// message. AppendRow buffers rows in memory; Flush sends everything buffered
// as one AppendRequest (one DataChunk covering every buffered row — unlike
// PrepareRequest, whose cost grows worse than linearly with statement size,
// AppendRequest carries columnar binary data with no parse/bind step, so
// there is no equivalent reason to chunk a large buffer before flushing).
// Not safe for concurrent use.
type QuackAppender struct {
	session *quackSession
	schema  string
	table   string
	columns []QuackColumnKind
	rows    [][]any
}

// NewQuackAppender returns a QuackAppender for catalog.schema.table, reading
// db's underlying quack session directly (via *sql.Conn.Raw, so this works
// against any *sql.DB opened by duckdb.Open — the type assertion below is
// what rejects a jsonlines-only connection). db must have been opened with
// Options.QuackAddr set; otherwise this returns an error rather than
// silently falling back to a slower transport.
//
// AppendRequestMessage's wire shape has no catalog field (see
// quack_append.go's package doc comment) — it resolves schema.table against
// whatever the target quack connection's *default* catalog happens to be.
// Confirmed empirically that this is genuinely per-connection, not shared
// with whatever the bootstrapping CLI session (process.go's
// bootstrapDuckLakeLocked) did: appending against a DuckLake-attached table
// fails with "Table main.<table> does not exist" — a graceful ErrorResponse,
// not a crash — until this connection's own default catalog is switched.
// So, when catalog is non-empty, NewQuackAppender issues "USE <catalog>;"
// once on the resolved session before returning. This is a real, global
// mutation of that session's default catalog for every later statement, not
// scoped to this appender — safe here only because every other caller in
// this codebase already fully qualifies table names with DuckLakeAlias (see
// e.g. cmd/duckdbseed/insert.go), so nothing relies on unqualified names
// resolving to any other catalog.
func NewQuackAppender(ctx context.Context, db *sql.DB, catalog, schema, table string, columns []QuackColumnKind) (*QuackAppender, error) {
	sqlConn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("duckdb: quack appender: acquiring connection: %w", err)
	}
	defer sqlConn.Close()

	var appender *QuackAppender
	err = sqlConn.Raw(func(driverConn any) error {
		var err error
		appender, err = newQuackAppenderFromDriverConn(ctx, driverConn, catalog, schema, table, columns)
		return err
	})
	if err != nil {
		return nil, err
	}
	return appender, nil
}

// NewQuackAppenderFromConn is NewQuackAppender for a driver.Conn the caller
// already owns outright — typically one obtained via Connector.Connect
// directly (see OpenConnector), rather than one *sql.DB.Conn hands out from
// its pool. Unlike a pooled *sql.Conn, this driverConn is never returned to
// anything: the caller must Close it itself once done with the appender.
// This is the mechanism for genuinely parallel Appenders — one dedicated
// connection per worker, held for the worker's whole lifetime — since
// *sql.DB's pool checks a connection back in (available for a different
// caller's db.Conn to receive next) the moment NewQuackAppender's own
// sqlConn.Close() runs, which would risk two callers racing over what they
// each believe is "their" connection.
func NewQuackAppenderFromConn(ctx context.Context, driverConn driver.Conn, catalog, schema, table string, columns []QuackColumnKind) (*QuackAppender, error) {
	return newQuackAppenderFromDriverConn(ctx, driverConn, catalog, schema, table, columns)
}

// newQuackAppenderFromDriverConn holds the logic NewQuackAppender and
// NewQuackAppenderFromConn share: resolve driverConn to its underlying
// quackSession, optionally switch catalog, construct the QuackAppender.
func newQuackAppenderFromDriverConn(ctx context.Context, driverConn any, catalog, schema, table string, columns []QuackColumnKind) (*QuackAppender, error) {
	c, ok := driverConn.(*conn)
	if !ok {
		return nil, fmt.Errorf("duckdb: quack appender: unexpected driver connection type %T", driverConn)
	}

	// The primary connection's sess is *process (crash-restart handling
	// wraps the real transport — see process.go), not a bare *quackSession
	// directly; an additional connection opened via Options.QuackConns > 1
	// (process.openQuackConn) is already a bare *quackSession. Handle both.
	var sess *quackSession
	switch s := c.sess.(type) {
	case *quackSession:
		sess = s
	case *process:
		var err error
		sess, err = s.currentQuackSession()
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("duckdb: quack appender: unexpected transport %T", c.sess)
	}

	if catalog != "" {
		if _, _, err := sess.exec(ctx, fmt.Sprintf("USE %s;", quackIdentifier(catalog))); err != nil {
			return nil, fmt.Errorf("duckdb: quack appender: switching to catalog %q: %w", catalog, err)
		}
	}

	return &QuackAppender{session: sess, schema: schema, table: table, columns: columns}, nil
}

// AppendRow buffers one row. len(vals) must equal len(columns) from
// NewQuackAppender, in the same order. No I/O happens here — call Flush or
// Close to actually send buffered rows.
func (a *QuackAppender) AppendRow(vals ...any) error {
	if len(vals) != len(a.columns) {
		return fmt.Errorf("duckdb: quack appender: got %d values, expected %d columns", len(vals), len(a.columns))
	}
	a.rows = append(a.rows, vals)
	return nil
}

// Flush sends every buffered row as one AppendRequest and clears the
// buffer. A no-op if nothing is buffered.
func (a *QuackAppender) Flush(ctx context.Context) error {
	if len(a.rows) == 0 {
		return nil
	}
	chunk, err := encodeQuackDataChunk(a.columns, a.rows)
	if err != nil {
		return fmt.Errorf("duckdb: quack appender: encoding chunk: %w", err)
	}
	if err := a.session.appendChunk(ctx, a.schema, a.table, chunk); err != nil {
		return err
	}
	a.rows = a.rows[:0]
	return nil
}

// Close flushes any remaining buffered rows. The appender must not be used
// afterward.
func (a *QuackAppender) Close(ctx context.Context) error {
	return a.Flush(ctx)
}

// encodeQuackDataChunk builds the wire bytes for a DataChunk object (field100
// row count, field101 list<LogicalType>, field102 list<Vector>) covering
// every column in cols, populated from rows (one []any per row, in column
// order).
func encodeQuackDataChunk(cols []QuackColumnKind, rows [][]any) ([]byte, error) {
	w := &quackWriter{}
	w.beginObject()
	w.writeUint64(100, uint64(len(rows)))

	w.writeFieldID(101)
	w.beginList(uint64(len(cols)))
	for _, k := range cols {
		id, alias := k.wireID()
		encodeQuackLogicalType(w, id, alias)
	}

	w.writeFieldID(102)
	w.beginList(uint64(len(cols)))
	for i, k := range cols {
		if err := encodeQuackVectorColumn(w, k, rows, i); err != nil {
			return nil, fmt.Errorf("column %d: %w", i, err)
		}
	}
	w.endObject()
	return w.bytes(), nil
}

// encodeQuackLogicalType writes one LogicalType object body: field100 id,
// and — only when alias is non-empty — field101 type_info, wrapped in the
// same field-id/presence-byte/object shape decodeQuackLogicalType expects
// (mirrors quack_wire_test.go's buildVarcharJSONChunk fixture exactly).
func encodeQuackLogicalType(w *quackWriter, id byte, alias string) {
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

// encodeQuackVectorColumn writes one Vector object body (field100
// has_validity, optional field101 validity mask, field102 data) for column
// index colIdx across every row, dispatching on kind's physical shape.
func encodeQuackVectorColumn(w *quackWriter, kind QuackColumnKind, rows [][]any, colIdx int) error {
	n := len(rows)
	// DuckDB's ValidityMask is physically an array of uint64_t words
	// (src/include/duckdb/common/types/validity_mask.hpp), not a tightly
	// packed byte array — the mask buffer must be padded to a multiple of 8
	// bytes (one 64-bit word) or the server's deserializer reads past the
	// end of a too-short buffer. Confirmed empirically: a 1-byte mask for a
	// single-row column crashes the server (HTTP 500, connection dropped)
	// where an 8-byte mask for the same row succeeds.
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
	w.writeFieldID(102)

	switch kind {
	case QuackColumnUUID:
		data := make([]byte, n*16)
		for i, row := range rows {
			if row[colIdx] == nil {
				continue // zero-filled placeholder; validity marks it invalid
			}
			id, err := quackValueToUUID(row[colIdx])
			if err != nil {
				return err
			}
			wire := quackEncodeUUID(id)
			copy(data[i*16:], wire[:])
		}
		w.writeData(data)
	case QuackColumnTimestampMS:
		data := make([]byte, n*8)
		for i, row := range rows {
			if row[colIdx] == nil {
				continue
			}
			micros, err := quackValueToTimestampMS(row[colIdx])
			if err != nil {
				return err
			}
			putLE64(data[i*8:i*8+8], uint64(micros))
		}
		w.writeData(data)
	case QuackColumnVarchar, QuackColumnJSON:
		w.beginList(uint64(n))
		for i, row := range rows {
			if row[colIdx] == nil {
				w.writeData(nil)
				continue
			}
			s, ok := row[colIdx].(string)
			if !ok {
				return fmt.Errorf("row %d: expected string, got %T", i, row[colIdx])
			}
			w.writeData([]byte(s))
		}
	default:
		return fmt.Errorf("unsupported QuackColumnKind %d", kind)
	}
	w.endObject()
	return nil
}

func quackValueToUUID(v any) (uuid.UUID, error) {
	switch val := v.(type) {
	case string:
		return uuid.Parse(val)
	case uuid.UUID:
		return val, nil
	default:
		return uuid.UUID{}, fmt.Errorf("expected string or uuid.UUID, got %T", v)
	}
}

func quackValueToTimestampMS(v any) (int64, error) {
	t, ok := v.(timeLike)
	if !ok {
		return 0, fmt.Errorf("expected time.Time, got %T", v)
	}
	return t.UnixMilli(), nil
}

// timeLike avoids importing "time" solely for a type assertion signature;
// time.Time satisfies it structurally.
type timeLike interface{ UnixMilli() int64 }

// quackEncodeUUID is the exact inverse of quack_protocol.go's
// quackUUIDToString: reverse the 16 bytes, then flip the top bit of the
// resulting last byte (equivalently: byte 15 of the wire form is id[0]^0x80,
// and wire byte j = id[15-j] for j = 0..14). Round-trip-verified in
// quack_append_test.go against the exact wire-byte pair
// quack_protocol_test.go's TestQuackDecodeChunkUUIDDecodesToStandardString
// already pins.
func quackEncodeUUID(id uuid.UUID) [16]byte {
	var b [16]byte
	for i := range b {
		b[i] = id[15-i]
	}
	b[15] ^= 0x80
	return b
}

// quackIdentifier double-quotes a DuckDB identifier, doubling any embedded
// double quote — the standard SQL identifier-escaping rule. Used only for
// the "USE <catalog>;" statement NewQuackAppender issues; every value that
// reaches it in this codebase is DuckLakeAlias, not external input, but this
// is still treated as injection-sensitive per this package's convention (see
// literal.go's encodeLiteral doc comment).
func quackIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// putLE64 writes v little-endian into dst[0:8].
func putLE64(dst []byte, v uint64) {
	for i := range 8 {
		dst[i] = byte(v >> (8 * i))
	}
}

// encodeQuackAppendRequest builds the wire bytes for an AppendRequest
// message: field1 schema_name, field2 table_name, field3 append_chunk (a
// DataChunkWrapper — an object wrapping the already-framed chunk bytes at
// field300, per DataChunkWrapper::Serialize in duckdb-quack's
// quack_message.cpp).
func encodeQuackAppendRequest(connectionID, schema, table string, chunk []byte) []byte {
	return encodeQuackMessage(quackMsgAppendRequest, connectionID, func(w *quackWriter) {
		w.writeStringDefault(1, schema)
		w.writeStringDefault(2, table)
		w.writeFieldID(3)
		w.buf.WriteByte(1) // presence marker — see encodeQuackLogicalType's doc comment
		w.beginObject()
		w.writeFieldID(300)
		w.buf.Write(chunk) // already a fully framed, terminated object
		w.endObject()
	})
}

// appendChunk sends one AppendRequest and waits for SuccessResponse. A
// statement DuckDB itself rejected (e.g. a missing table or a type
// mismatch) comes back wrapped in errStatementFailed, matching exec's
// convention (see quack_session.go) so process.exec's restart-vs-surface
// classification behaves identically for both message types.
func (s *quackSession) appendChunk(ctx context.Context, schema, table string, chunk []byte) error {
	hdr, r, err := s.send(ctx, encodeQuackAppendRequest(s.connectionID, schema, table, chunk))
	if err != nil {
		return err
	}
	switch hdr.Type {
	case quackMsgSuccessResponse:
		return decodeQuackSuccessResponseBody(r)
	case quackMsgErrorResponse:
		msg, derr := decodeQuackErrorResponseBody(r)
		if derr != nil {
			return fmt.Errorf("duckdb: quack append: server returned an error this client could not parse: %w", derr)
		}
		return fmt.Errorf("%w: %s", errStatementFailed, msg)
	default:
		return fmt.Errorf("duckdb: quack append: unexpected response message type %d", hdr.Type)
	}
}
