package duckdb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// This file implements MessageType.SEND_DATA_REQUEST/SEND_DATA_RESPONSE (ids
// 9/14 — see quack_protocol.go), quack protocol version 3's replacement for
// v1.5's standalone AppendRequest (see quack_append.go's own doc comment for
// how that removal was discovered). Confirmed against the real duckdb-quack
// source (github.com/duckdb/duckdb-quack, main branch:
// src/include/quack_message.hpp, src/quack_message.cpp,
// src/quack_send_data.cpp, src/quack_scan_from_client.cpp) and verified
// end-to-end against a real duckdb v2.1.0-alpha40409 subprocess before this
// file was written.
//
// Unlike AppendRequest, SEND_DATA_REQUEST is not a standalone bulk-insert
// RPC: a stream_id only means something because it appears inside a SQL
// statement referencing the internal table function
// scan_data_from_quack_client(stream_id, row_type_prototype, ordered :=
// true), which this client's PrepareRequest issues as an ordinary INSERT
// (buildQuackSendDataInsertSQL). The server's query executor opens the
// stream during that statement's BIND phase (before any row is scanned),
// then blocks the statement's execution on receiving batches tagged with
// that stream_id via SEND_DATA_REQUEST — so the flow is inherently
// two-sided: this client must have the INSERT's PrepareRequest in flight
// (blocked, server-side, waiting for data) at the same time it POSTs
// batches, which is why sendDataAppend runs the PrepareRequest on its own
// goroutine rather than sequentially before sending any data.
//
// scan_data_from_quack_client is marked "internal" in duckdb_functions()
// (it panics with "scan_data_from_quack_client is an internal function
// driven by the quack server" if bound outside an active per-connection
// quack session) but is not otherwise blocked from appearing in a client's
// own PrepareRequest SQL text — that check is exactly what makes it work
// here, since every PrepareRequest this client sends already executes
// inside that per-connection session state.
func generateQuackStreamID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "inngest-appender-" + hex.EncodeToString(b[:]), nil
}

// quackSendDataInsertColumnType maps a QuackColumnKind to the DuckDB type
// name buildQuackSendDataInsertSQL casts the scan_data_from_quack_client
// row prototype's matching field to. This only needs to be castable to
// whatever the real target column's type is (INSERT INTO ... SELECT *
// matches by position, not name, so the prototype's own field names are
// synthetic and never seen again) — it does not need to match the target
// table's declared type exactly, e.g. QuackColumnJSON's VARCHAR here
// implicit-casts fine into a JSON-typed target column.
func quackSendDataInsertColumnType(k QuackColumnKind) (string, error) {
	switch k {
	case QuackColumnUUID:
		return "UUID", nil
	case QuackColumnVarchar, QuackColumnJSON:
		return "VARCHAR", nil
	case QuackColumnTimestampMS:
		return "TIMESTAMP_MS", nil
	default:
		return "", fmt.Errorf("duckdb: quack appender: unsupported QuackColumnKind %d", k)
	}
}

// buildQuackSendDataInsertSQL builds the INSERT statement this client's
// PrepareRequest sends to open a stream: an ordinary INSERT selecting from
// scan_data_from_quack_client, whose second argument is a NULL cast to a
// STRUCT literally spelling out the incoming batch's column types. The
// prototype's field names are synthetic (c0, c1, ...) since only position,
// not name, matters for INSERT INTO ... SELECT *.
func buildQuackSendDataInsertSQL(schema, table string, columns []QuackColumnKind, streamID string) (string, error) {
	if len(columns) == 0 {
		return "", fmt.Errorf("duckdb: quack appender: no columns")
	}
	fields := make([]string, len(columns))
	for i, k := range columns {
		typeName, err := quackSendDataInsertColumnType(k)
		if err != nil {
			return "", err
		}
		fields[i] = fmt.Sprintf("c%d %s", i, typeName)
	}

	tableRef := quackIdentifier(table)
	if schema != "" {
		tableRef = quackIdentifier(schema) + "." + tableRef
	}
	streamIDLiteral, err := encodeLiteral(streamID)
	if err != nil {
		return "", fmt.Errorf("duckdb: quack appender: encoding stream id: %w", err)
	}

	return fmt.Sprintf(
		"INSERT INTO %s SELECT * FROM scan_data_from_quack_client(%s, NULL::STRUCT(%s), ordered := true);",
		tableRef, streamIDLiteral, strings.Join(fields, ", "),
	), nil
}

// encodeQuackSendDataRequest builds a SEND_DATA_REQUEST message: field1
// stream_id, field2 chunk_count (default-omit, so the terminal message with
// chunkCount 0 sends no field2 at all — matching
// SendDataRequestMessage::Serialize's WritePropertyWithDefault<idx_t>(2, ...,
// 0)), field3 total_batches and field4 batch_index (both optional_idx,
// nil meaning "unset" — set on the terminal message and every batch
// respectively, per quack_send_data.cpp), followed directly by chunkBlob:
// chunkCount bare DataChunk objects concatenated back-to-back, NOT wrapped
// in DataChunkWrapper's field300 the way PrepareResponse's chunks are (see
// quack_append.go's doc comment). This trailing blob is why
// SendDataRequestMessage has its own DecodeBlob override server-side
// instead of an ordinary vector<...> field.
func encodeQuackSendDataRequest(connectionID, streamID string, chunkCount uint64, batchIndex, totalBatches *uint64, chunkBlob []byte) []byte {
	msg := encodeQuackMessage(quackMsgSendDataRequest, connectionID, func(w *quackWriter) {
		w.writeStringDefault(1, streamID)
		w.writeUint64Default(2, chunkCount)
		if totalBatches != nil {
			w.writeUint64(3, *totalBatches)
		}
		if batchIndex != nil {
			w.writeUint64(4, *batchIndex)
		}
	})
	return append(msg, chunkBlob...)
}

// decodeQuackSendDataResponseBody consumes a SEND_DATA_RESPONSE body: an
// optional field1 accept_budget (optional_idx) this client doesn't use —
// SendDataResponseMessage's own doc comment in the real source calls it "a
// placeholder for a future flow-control hint... the client currently
// ignores it".
func decodeQuackSendDataResponseBody(r *quackReader) error {
	if ok, err := r.tryBeginProperty(1); err != nil {
		return err
	} else if ok {
		if _, err := r.readUnsignedLeb128(); err != nil {
			return err
		}
	}
	return r.endObject()
}

// quackStreamNotActiveMessage is the exact ErrorResponse text the server
// sends for a SEND_DATA_REQUEST whose stream_id names no currently-open
// stream — confirmed empirically (an entirely empty SEND_DATA_REQUEST body
// gets back `No active data stream ''`; a body naming stream_id "foo" gets
// back `No active data stream 'foo'`). sendDataSendBatch retries on this
// specific text while the companion PrepareRequest is still reaching BIND
// (see its own doc comment for why there's no other "ready" signal).
const quackStreamNotActiveMessage = "No active data stream"

func isQuackStreamNotActiveErr(err error) bool {
	return errors.Is(err, errStatementFailed) && strings.Contains(err.Error(), quackStreamNotActiveMessage)
}

// quackSendDataStreamReadyTimeout bounds how long sendDataSendBatch retries
// a SEND_DATA_REQUEST that the server rejects as naming no active stream,
// before giving up. Comfortably above what a real BIND ever takes locally,
// but short enough that a genuinely stuck server fails an append quickly
// rather than hanging.
const quackSendDataStreamReadyTimeout = 5 * time.Second

// quackSendDataStreamReadyMinBackoff/MaxBackoff bound sendDataSendBatch's
// retry backoff: small enough that the common case (the PrepareRequest
// goroutine reaches BIND well within one backoff step) adds negligible
// latency, capped so a slow BIND doesn't busy-loop the retry.
const (
	quackSendDataStreamReadyMinBackoff = 2 * time.Millisecond
	quackSendDataStreamReadyMaxBackoff = 100 * time.Millisecond
)

// quackStandardVectorSize is DuckDB's own STANDARD_VECTOR_SIZE: the hard cap
// on how many rows a single DataChunk (here, one bare chunk object in a
// SEND_DATA_REQUEST's blob) may hold. Unlike v1.5's standalone AppendRequest
// — which accepted one arbitrarily large DataChunk with no such limit,
// apparently bypassing this check entirely — a SEND_DATA_REQUEST's chunks
// flow through DuckDB's ordinary execution engine (scan_data_from_quack_client
// feeds a real pipeline), which enforces it strictly: confirmed empirically,
// a single chunk over this size fails downstream expression evaluation with
// "Vector::SetSize out of range - trying to set size to N for vector with
// capacity 2048" (seen via CastVarcharToJSON, but the limit is general, not
// specific to that cast). rowsToQuackChunks splits a large row buffer into
// chunks at this boundary instead.
const quackStandardVectorSize = 2048

// rowsToQuackChunks splits rows into chunks of at most quackStandardVectorSize
// rows each (see that constant's doc comment), encoding each via
// encodeQuackDataChunk and concatenating the results — exactly the shape
// SEND_DATA_REQUEST's trailing blob wants for a chunkCount > 1 batch: bare
// chunk objects back-to-back, no separator or wrapper between them.
func rowsToQuackChunks(cols []QuackColumnKind, rows [][]any) (blob []byte, chunkCount uint64, err error) {
	for start := 0; start < len(rows); start += quackStandardVectorSize {
		end := min(start+quackStandardVectorSize, len(rows))
		chunk, err := encodeQuackDataChunk(cols, rows[start:end])
		if err != nil {
			return nil, 0, fmt.Errorf("duckdb: quack appender: encoding chunk %d: %w", chunkCount, err)
		}
		blob = append(blob, chunk...)
		chunkCount++
	}
	return blob, chunkCount, nil
}

// sendDataAppend bulk-inserts every row in rows into schema.table via
// quack's SEND_DATA_REQUEST mechanism (see this file's own doc comment for
// the full protocol). It opens a stream by issuing an INSERT...FROM
// scan_data_from_quack_client(...) PrepareRequest on its own goroutine
// (that request blocks server-side until the stream receives its data),
// sends the one batch this appender ever needs — as however many
// quackStandardVectorSize-capped chunks rows requires — plus the terminal
// total_batches message on the caller's goroutine, then joins the
// PrepareRequest goroutine and returns its result (or its error, e.g. a
// missing table or a type mismatch the INSERT itself rejected).
func (s *quackSession) sendDataAppend(ctx context.Context, schema, table string, columns []QuackColumnKind, rows [][]any) error {
	streamID, err := generateQuackStreamID()
	if err != nil {
		return fmt.Errorf("duckdb: quack appender: generating stream id: %w", err)
	}
	sql, err := buildQuackSendDataInsertSQL(schema, table, columns, streamID)
	if err != nil {
		return err
	}
	blob, chunkCount, err := rowsToQuackChunks(columns, rows)
	if err != nil {
		return err
	}

	prepareDone := make(chan error, 1)
	go func() {
		_, _, err := s.exec(ctx, sql)
		prepareDone <- err
	}()

	const batchIndex = uint64(1)
	if err := s.sendDataSendBatch(ctx, streamID, batchIndex, chunkCount, blob, prepareDone); err != nil {
		return err
	}
	if err := s.sendDataSendTerminal(ctx, streamID, batchIndex); err != nil {
		return err
	}

	return <-prepareDone
}

// sendDataSendBatch sends the one dense batch (index batchIndex) this
// appender's single-shot Flush ever needs, retrying while the server
// reports no active stream by that id yet — the PrepareRequest goroutine's
// BIND phase is what opens it (see sendDataAppend), and trying-then-
// observing the ErrorResponse is the only "is it open yet" signal this
// protocol gives a client. prepareDone reporting before the stream ever
// opens (the INSERT itself was rejected) is surfaced directly rather than
// retried against a stream that will never exist.
func (s *quackSession) sendDataSendBatch(ctx context.Context, streamID string, batchIndex, chunkCount uint64, blob []byte, prepareDone <-chan error) error {
	deadline := time.Now().Add(quackSendDataStreamReadyTimeout)
	backoff := quackSendDataStreamReadyMinBackoff
	for {
		payload := encodeQuackSendDataRequest(s.connectionID, streamID, chunkCount, &batchIndex, nil, blob)
		err := s.sendDataRequest(ctx, payload)
		if err == nil {
			return nil
		}
		if !isQuackStreamNotActiveErr(err) {
			return err
		}

		select {
		case prepErr := <-prepareDone:
			if prepErr != nil {
				return fmt.Errorf("duckdb: quack appender: prepare ended before the stream ever opened: %w", prepErr)
			}
			return fmt.Errorf("duckdb: quack appender: prepare completed without the stream ever accepting data")
		default:
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("duckdb: quack appender: stream %q never became active: %w", streamID, err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > quackSendDataStreamReadyMaxBackoff {
			backoff = quackSendDataStreamReadyMaxBackoff
		}
	}
}

// sendDataSendTerminal closes streamID against a single batch (batchIndex),
// per quack_send_data.cpp's QuackSendDataEmitter::Finish: "an ordinary
// SEND_DATA with no chunks... closes the stream against the batch count, so
// a lost batch fails the statement." No retry here: by the time
// sendDataSendBatch has already succeeded, the stream is known to be open.
func (s *quackSession) sendDataSendTerminal(ctx context.Context, streamID string, totalBatches uint64) error {
	payload := encodeQuackSendDataRequest(s.connectionID, streamID, 0, nil, &totalBatches, nil)
	return s.sendDataRequest(ctx, payload)
}

// sendDataRequest POSTs one already-encoded SEND_DATA_REQUEST and maps its
// response the same way exec/query map a PrepareRequest's: a statement-level
// rejection (including "no active stream", which sendDataSendBatch handles
// specially) wraps errStatementFailed; anything else unexpected is a
// transport-level error.
func (s *quackSession) sendDataRequest(ctx context.Context, payload []byte) error {
	hdr, r, err := s.send(ctx, payload)
	if err != nil {
		return err
	}
	switch hdr.Type {
	case quackMsgSendDataResponse:
		return decodeQuackSendDataResponseBody(r)
	case quackMsgErrorResponse:
		return decodeQuackStatementError(r)
	default:
		return fmt.Errorf("duckdb: quack send_data: unexpected response message type %d", hdr.Type)
	}
}
