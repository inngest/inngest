package duckdb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// This file builds the DataChunk bytes QuackAppender bulk-loads into a
// table; quack_senddata.go drives the actual wire request. It exists because
// every other write goes through PrepareRequest (a real SQL PREPARE over
// interpolated literal text), and binding a giant literal VALUES list scales
// far worse than linearly with statement size. DataChunk's binary columnar
// wire shape sidesteps that: no SQL parsing/binding cost at all.
//
// DuckDB v1.5.5's quack extension sent this as a standalone
// MessageType.APPEND_REQUEST RPC. DuckDB v2.1.0-alpha (protocol version 3)
// removed that message type; in its place is SEND_DATA_REQUEST/RESPONSE, a
// mechanism driven by the server's query executor rather than a
// self-contained RPC (see quack_senddata.go).
//
// Wire shapes below (DataChunk's fields 100/101/102) mirror this file's own
// decode-side counterparts in quack_protocol.go
// (decodeQuackDataChunk/decodeQuackVector/decodeFlatVector) field-for-field;
// this is also the exact shape one bare chunk in SEND_DATA_REQUEST's
// trailing blob needs — no DataChunkWrapper field-300 wrapping the way
// PrepareResponse's chunks get.
//
// Type coverage is scoped to exactly what inngest.run_trace_spans uses today
// (UUID, VARCHAR, VARCHAR-aliased-JSON, TIMESTAMP_MS) — extend as new
// callers need new types.

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
// A native LIST wire type was prototyped and reverted: the real duckdb-quack
// server crashes (empty-bodied HTTP 500) on any AppendRequest containing a
// LIST-typed column, not fixable from this client alone. Callers needing a
// VARCHAR[] column should instead encode QuackColumnVarchar (or
// QuackColumnJSON) array-literal text (e.g. `["a","b"]`) — DuckDB's Appender
// implicit-casts that to LIST.
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

// QuackAppender bulk-loads rows into one table over quack's
// SEND_DATA_REQUEST mechanism (see quack_senddata.go). AppendRow buffers
// rows in memory; Flush sends everything buffered as one batch. Not safe for
// concurrent use.
type QuackAppender struct {
	session *quackSession
	schema  string
	table   string
	columns []QuackColumnKind
	rows    [][]any
}

// NewQuackAppender returns a QuackAppender for catalog.schema.table, reading
// db's underlying quack session directly (via *sql.Conn.Raw — the type
// assertion below rejects a jsonlines-only connection). db must have been
// opened with Options.QuackAddr set; otherwise this returns an error rather
// than silently falling back to a slower transport.
//
// The wire protocol has no catalog field: it resolves schema.table against
// the target connection's own default catalog, which is per-connection, not
// shared with the bootstrapping CLI session. So when catalog is non-empty,
// NewQuackAppender issues "USE <catalog>;" once on the resolved session
// before returning. This is a real, global mutation of that session's
// default catalog for every later statement — safe here only because every
// other caller already fully qualifies table names with DuckLakeAlias (see
// cmd/duckdbseed/insert.go).
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
// already owns outright — typically via Connector.Connect directly (see
// OpenConnector), rather than one *sql.DB.Conn hands out from its pool. The
// caller must Close driverConn itself once done with the appender. This is
// the mechanism for genuinely parallel Appenders (one dedicated connection
// per worker), since a pooled *sql.Conn would be returned to the pool — and
// available to a different caller — the moment NewQuackAppender closed it.
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
	// wraps the real transport); an extra connection from Options.QuackConns
	// > 1 is already a bare *quackSession. Handle both.
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

// Flush sends every buffered row as one SEND_DATA_REQUEST batch and clears
// the buffer. A no-op if nothing is buffered.
func (a *QuackAppender) Flush(ctx context.Context) error {
	if len(a.rows) == 0 {
		return nil
	}
	if err := a.session.sendDataAppend(ctx, a.schema, a.table, a.columns, a.rows); err != nil {
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
// and — only when alias is non-empty — field101 type_info, in the shape
// decodeQuackLogicalType expects.
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
	case QuackColumnUUID:
		w.writeFieldID(102)
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
		w.writeFieldID(102)
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
		if err := encodeQuackVarcharVectorData(w, rows, colIdx); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported QuackColumnKind %d", kind)
	}
	w.endObject()
	return nil
}

// encodeQuackVarcharVectorData writes a VARCHAR Vector's data fields (100
// has_validity / 101 validity mask already written by the caller): the
// inverse of decodeQuackVarcharFlatVector's field107/108/109 shape.
//
//   - field107: total byte length of every row's string data added together.
//   - field108: one little-endian uint32 byte-length per row, in row order —
//     a NULL row gets a real zero-length entry, never omitted.
//   - field109: every row's string bytes concatenated back-to-back, no
//     separator.
func encodeQuackVarcharVectorData(w *quackWriter, rows [][]any, colIdx int) error {
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
// resulting last byte.
func quackEncodeUUID(id uuid.UUID) [16]byte {
	var b [16]byte
	for i := range b {
		b[i] = id[15-i]
	}
	b[15] ^= 0x80
	return b
}

// quackIdentifier double-quotes a DuckDB identifier, doubling any embedded
// double quote. Used only for the "USE <catalog>;" statement
// NewQuackAppender issues; treated as injection-sensitive per this package's
// convention even though every caller passes DuckLakeAlias, not external
// input.
func quackIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// putLE64 writes v little-endian into dst[0:8].
func putLE64(dst []byte, v uint64) {
	for i := range 8 {
		dst[i] = byte(v >> (8 * i))
	}
}
