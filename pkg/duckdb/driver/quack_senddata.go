package driver

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
// v1.5's standalone AppendRequest (see quack_append.go).
//
// Unlike AppendRequest, SEND_DATA_REQUEST is not a standalone bulk-insert
// RPC: a stream_id only means something inside a SQL statement referencing
// the internal table function scan_data_from_quack_client(stream_id,
// row_type_prototype, ordered := true), which this client's PrepareRequest
// issues as an ordinary INSERT (buildQuackSendDataInsertSQL). The server
// opens the stream during that statement's BIND phase, then blocks execution
// until it receives batches tagged with that stream_id — so the flow is
// two-sided: the INSERT's PrepareRequest must be in flight (blocked
// server-side) while this client POSTs batches, which is why sendDataAppend
// runs the PrepareRequest on its own goroutine instead of sequentially.
func generateQuackStreamID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "inngest-appender-" + hex.EncodeToString(b[:]), nil
}

// quackSendDataColumnType maps a QuackColumnKind to the DuckDB type a
// scan_data_from_quack_client STRUCT prototype casts the row's matching
// field to — shared by buildQuackSendDataInsertSQL (INSERT) and
// buildQuackSendDataMergeSQL (MERGE, quack_merge.go). It only needs to be
// implicit-castable to the target column's real type (e.g. QuackColumnJSON's
// VARCHAR casts fine into a JSON column): INSERT matches by position, and
// MERGE's ON/UpdateSet reference the STRUCT's own field names directly, so
// neither depends on this being the target column's exact declared type.
func quackSendDataColumnType(k QuackColumnKind) (string, error) {
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

// buildQuackSendDataInsertSQL builds the INSERT statement that opens a
// stream: selecting from scan_data_from_quack_client, whose second argument
// is a NULL cast to a STRUCT spelling out the batch's column types (fields
// named c0, c1, ... since only position matters).
func buildQuackSendDataInsertSQL(schema, table string, columns []QuackColumnKind, streamID string) (string, error) {
	if len(columns) == 0 {
		return "", fmt.Errorf("duckdb: quack appender: no columns")
	}
	fields := make([]string, len(columns))
	for i, k := range columns {
		typeName, err := quackSendDataColumnType(k)
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
// stream_id, field2 chunk_count (default-omit — the terminal message with
// chunkCount 0 omits it), field3 total_batches and field4 batch_index (both
// optional, set on the terminal message and every batch respectively),
// followed by chunkBlob: chunkCount bare DataChunk objects concatenated
// back-to-back, unlike PrepareResponse's DataChunkWrapper-wrapped chunks
// (see quack_append.go).
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
// optional field1 accept_budget this client ignores (a flow-control hint
// reserved for future use).
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

// quackStreamNotActiveMessage is the ErrorResponse text the server sends for
// a SEND_DATA_REQUEST whose stream_id names no currently-open stream.
// sendDataSendBatch retries on this while the companion PrepareRequest is
// still reaching BIND.
const quackStreamNotActiveMessage = "No active data stream"

func isQuackStreamNotActiveErr(err error) bool {
	return errors.Is(err, errStatementFailed) && strings.Contains(err.Error(), quackStreamNotActiveMessage)
}

// quackSendDataStreamReadyTimeout bounds how long sendDataSendBatch retries
// a "no active stream" rejection before giving up.
const quackSendDataStreamReadyTimeout = 5 * time.Second

// quackSendDataStreamReadyMinBackoff/MaxBackoff bound sendDataSendBatch's
// retry backoff.
const (
	quackSendDataStreamReadyMinBackoff = 2 * time.Millisecond
	quackSendDataStreamReadyMaxBackoff = 100 * time.Millisecond
)

// quackStandardVectorSize is DuckDB's own STANDARD_VECTOR_SIZE: the hard cap
// on rows per DataChunk. Unlike v1.5's AppendRequest, a SEND_DATA_REQUEST's
// chunks flow through DuckDB's real execution engine, which enforces this
// strictly (a chunk over this size fails with "Vector::SetSize out of
// range"). rowsToQuackChunks splits large row buffers at this boundary.
const quackStandardVectorSize = 2048

// rowsToQuackChunks splits rows into chunks of at most quackStandardVectorSize
// rows, encoding each via encodeQuackDataChunk and concatenating the results
// — bare chunk objects back-to-back, as SEND_DATA_REQUEST's blob expects.
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
// quack's SEND_DATA_REQUEST mechanism (see this file's doc comment).
func (s *quackSession) sendDataAppend(ctx context.Context, schema, table string, columns []QuackColumnKind, rows [][]any) error {
	streamID, err := generateQuackStreamID()
	if err != nil {
		return fmt.Errorf("duckdb: quack appender: generating stream id: %w", err)
	}
	sql, err := buildQuackSendDataInsertSQL(schema, table, columns, streamID)
	if err != nil {
		return err
	}
	return s.sendDataDrive(ctx, sql, streamID, columns, rows)
}

// sendDataDrive runs sql (which must reference
// scan_data_from_quack_client(streamID, ...) somewhere in its FROM/USING
// clause) as a PrepareRequest on its own goroutine — blocked server-side
// until data arrives — while pushing every row in rows as the one batch that
// PrepareRequest is waiting on, tagged with streamID, followed by a terminal
// message. It then joins the PrepareRequest goroutine and returns its
// result. Shared by sendDataAppend (a plain INSERT) and sendDataMerge (a
// MERGE INTO, quack_merge.go) — the two differ only in what sql says to do
// with the streamed rows once scan_data_from_quack_client yields them.
func (s *quackSession) sendDataDrive(ctx context.Context, sql, streamID string, columns []QuackColumnKind, rows [][]any) error {
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

// sendDataSendBatch sends the one dense batch this appender needs, retrying
// while the server reports no active stream yet (opened once the
// PrepareRequest goroutine's BIND phase runs). If prepareDone reports before
// the stream ever opens (the INSERT was rejected), that error is surfaced
// directly instead of retrying against a stream that will never exist.
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

// sendDataSendTerminal closes streamID against totalBatches: a SEND_DATA
// with no chunks closes the stream against the batch count, so a lost batch
// fails the statement. No retry here, since sendDataSendBatch having
// succeeded means the stream is known to be open.
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
