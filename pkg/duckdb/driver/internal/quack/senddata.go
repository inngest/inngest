package quack

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/result"
)

// This file implements MessageType.SEND_DATA_REQUEST/SEND_DATA_RESPONSE (ids
// 9/14 — see protocol.go), quack protocol version 3's replacement for
// v1.5's standalone AppendRequest (see append.go).
//
// Unlike AppendRequest, SEND_DATA_REQUEST is not a standalone bulk-insert
// RPC: a stream_id only means something inside a SQL statement referencing
// the internal table function scan_data_from_quack_client(stream_id,
// row_type_prototype, ordered := true), which this client's PrepareRequest
// issues as an ordinary INSERT (buildSendDataInsertSQL). The server
// opens the stream during that statement's BIND phase, then blocks execution
// until it receives batches tagged with that stream_id — so the flow is
// two-sided: the INSERT's PrepareRequest must be in flight (blocked
// server-side) while this client POSTs batches, which is why Append
// runs the PrepareRequest on its own goroutine instead of sequentially.
func generateStreamID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "inngest-appender-" + hex.EncodeToString(b[:]), nil
}

// sendDataColumnType maps a ColumnKind to the DuckDB type a
// scan_data_from_quack_client STRUCT prototype casts the row's matching
// field to — shared by buildSendDataInsertSQL (INSERT) and
// buildSendDataMergeSQL (MERGE, merge.go). It only needs to be
// implicit-castable to the target column's real type (e.g. ColumnJSON's
// VARCHAR casts fine into a JSON column): INSERT matches by position, and
// MERGE's ON/UpdateSet reference the STRUCT's own field names directly, so
// neither depends on this being the target column's exact declared type.
func sendDataColumnType(k ColumnKind) (string, error) {
	switch k {
	case ColumnUUID:
		return "UUID", nil
	case ColumnVarchar, ColumnJSON:
		return "VARCHAR", nil
	case ColumnTimestampMS:
		return "TIMESTAMP_MS", nil
	default:
		return "", fmt.Errorf("duckdb: quack appender: unsupported ColumnKind %d", k)
	}
}

// buildSendDataInsertSQL builds the INSERT statement that opens a
// stream: selecting from scan_data_from_quack_client, whose second argument
// is a NULL cast to a STRUCT spelling out the batch's column types (fields
// named c0, c1, ... since only position matters).
func buildSendDataInsertSQL(schema, table string, columns []ColumnKind, streamID string) (string, error) {
	if len(columns) == 0 {
		return "", fmt.Errorf("duckdb: quack appender: no columns")
	}
	fields := make([]string, len(columns))
	for i, k := range columns {
		typeName, err := sendDataColumnType(k)
		if err != nil {
			return "", err
		}
		fields[i] = fmt.Sprintf("c%d %s", i, typeName)
	}

	tableRef := Identifier(table)
	if schema != "" {
		tableRef = Identifier(schema) + "." + tableRef
	}
	streamIDLiteral := stringLiteral(streamID)

	return fmt.Sprintf(
		"INSERT INTO %s SELECT * FROM scan_data_from_quack_client(%s, NULL::STRUCT(%s), ordered := true);",
		tableRef, streamIDLiteral, strings.Join(fields, ", "),
	), nil
}

// encodeSendDataRequest builds a SEND_DATA_REQUEST message: field1
// stream_id, field2 chunk_count (default-omit — the terminal message with
// chunkCount 0 omits it), field3 total_batches and field4 batch_index (both
// optional, set on the terminal message and every batch respectively),
// followed by chunkBlob: chunkCount bare DataChunk objects concatenated
// back-to-back, unlike PrepareResponse's DataChunkWrapper-wrapped chunks
// (see append.go).
func encodeSendDataRequest(connectionID, streamID string, chunkCount uint64, batchIndex, totalBatches *uint64, chunkBlob []byte) []byte {
	msg := encodeMessage(msgSendDataRequest, connectionID, func(w *writer) {
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

// decodeSendDataResponseBody consumes a SEND_DATA_RESPONSE body: an
// optional field1 accept_budget this client ignores (a flow-control hint
// reserved for future use).
func decodeSendDataResponseBody(r *reader) error {
	if ok, err := r.tryBeginProperty(1); err != nil {
		return err
	} else if ok {
		if _, err := r.readUnsignedLeb128(); err != nil {
			return err
		}
	}
	return r.endObject()
}

// streamNotActiveMessage is the ErrorResponse text the server sends for
// a SEND_DATA_REQUEST whose stream_id names no currently-open stream.
// sendDataSendBatch retries on this while the companion PrepareRequest is
// still reaching BIND.
const streamNotActiveMessage = "No active data stream"

func isStreamNotActiveErr(err error) bool {
	return errors.Is(err, result.ErrStatementFailed) && strings.Contains(err.Error(), streamNotActiveMessage)
}

// sendDataStreamReadyTimeout bounds how long sendDataSendBatch retries
// a "no active stream" rejection before giving up.
const sendDataStreamReadyTimeout = 5 * time.Second

// sendDataStreamReadyMinBackoff/MaxBackoff bound sendDataSendBatch's
// retry backoff.
const (
	sendDataStreamReadyMinBackoff = 2 * time.Millisecond
	sendDataStreamReadyMaxBackoff = 100 * time.Millisecond
)

// StandardVectorSize is DuckDB's own STANDARD_VECTOR_SIZE: the hard cap
// on rows per DataChunk. Unlike v1.5's AppendRequest, a SEND_DATA_REQUEST's
// chunks flow through DuckDB's real execution engine, which enforces this
// strictly (a chunk over this size fails with "Vector::SetSize out of
// range"). rowsToChunks splits large row buffers at this boundary.
const StandardVectorSize = 2048

// rowsToChunks splits rows into chunks of at most StandardVectorSize
// rows, encoding each via encodeDataChunk and concatenating the results
// — bare chunk objects back-to-back, as SEND_DATA_REQUEST's blob expects.
func rowsToChunks(cols []ColumnKind, rows [][]any) (blob []byte, chunkCount uint64, err error) {
	for start := 0; start < len(rows); start += StandardVectorSize {
		end := min(start+StandardVectorSize, len(rows))
		chunk, err := encodeDataChunk(cols, rows[start:end])
		if err != nil {
			return nil, 0, fmt.Errorf("duckdb: quack appender: encoding chunk %d: %w", chunkCount, err)
		}
		blob = append(blob, chunk...)
		chunkCount++
	}
	return blob, chunkCount, nil
}

// Append bulk-inserts every row in rows into schema.table via
// quack's SEND_DATA_REQUEST mechanism (see this file's doc comment).
func (s *Session) Append(ctx context.Context, schema, table string, columns []ColumnKind, rows [][]any) error {
	streamID, err := generateStreamID()
	if err != nil {
		return fmt.Errorf("duckdb: quack appender: generating stream id: %w", err)
	}
	sql, err := buildSendDataInsertSQL(schema, table, columns, streamID)
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
// result. Shared by Append (a plain INSERT) and Merge (a
// MERGE INTO, merge.go) — the two differ only in what sql says to do
// with the streamed rows once scan_data_from_quack_client yields them.
func (s *Session) sendDataDrive(ctx context.Context, sql, streamID string, columns []ColumnKind, rows [][]any) error {
	blob, chunkCount, err := rowsToChunks(columns, rows)
	if err != nil {
		return err
	}

	prepareDone := make(chan error, 1)
	go func() {
		_, _, err := s.Exec(ctx, sql)
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
func (s *Session) sendDataSendBatch(ctx context.Context, streamID string, batchIndex, chunkCount uint64, blob []byte, prepareDone <-chan error) error {
	deadline := time.Now().Add(sendDataStreamReadyTimeout)
	backoff := sendDataStreamReadyMinBackoff
	for {
		payload := encodeSendDataRequest(s.connectionID, streamID, chunkCount, &batchIndex, nil, blob)
		err := s.sendDataRequest(ctx, payload)
		if err == nil {
			return nil
		}
		if !isStreamNotActiveErr(err) {
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
		if backoff > sendDataStreamReadyMaxBackoff {
			backoff = sendDataStreamReadyMaxBackoff
		}
	}
}

// sendDataSendTerminal closes streamID against totalBatches: a SEND_DATA
// with no chunks closes the stream against the batch count, so a lost batch
// fails the statement. No retry here, since sendDataSendBatch having
// succeeded means the stream is known to be open.
func (s *Session) sendDataSendTerminal(ctx context.Context, streamID string, totalBatches uint64) error {
	payload := encodeSendDataRequest(s.connectionID, streamID, 0, nil, &totalBatches, nil)
	return s.sendDataRequest(ctx, payload)
}

// sendDataRequest POSTs one already-encoded SEND_DATA_REQUEST and maps its
// response the same way exec/query map a PrepareRequest's: a statement-level
// rejection (including "no active stream", which sendDataSendBatch handles
// specially) wraps result.ErrStatementFailed; anything else unexpected is a
// transport-level error.
func (s *Session) sendDataRequest(ctx context.Context, payload []byte) error {
	hdr, r, err := s.send(ctx, payload)
	if err != nil {
		return err
	}
	switch hdr.Type {
	case msgSendDataResponse:
		return decodeSendDataResponseBody(r)
	case msgErrorResponse:
		return decodeStatementError(r)
	default:
		return fmt.Errorf("duckdb: quack send_data: unexpected response message type %d", hdr.Type)
	}
}
