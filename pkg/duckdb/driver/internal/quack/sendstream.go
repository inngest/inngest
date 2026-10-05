package quack

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/result"
)

// SendStream is one SEND_DATA stream a query reads, through the table
// function StreamScanSQL returns, while QueryWithStreams feeds it.
type SendStream struct {
	ID      string
	Columns []ColumnKind
	// Next returns the next rows to send, or io.EOF once the stream is
	// complete. Any other error aborts the query. It's called from its own
	// goroutine, and must honor ctx.
	Next func(ctx context.Context) ([][]any, error)
	// NextEncoded, when set, replaces Next for a producer that encodes its
	// own batches (EncodeVectors): it returns a batch's DataChunks blob and
	// chunk count, or io.EOF once the stream is complete.
	NextEncoded func(ctx context.Context) (blob []byte, chunks uint64, err error)
}

// NewStreamID returns a fresh, unguessable stream id.
func NewStreamID() (string, error) { return generateStreamID() }

// StreamScanSQL returns the table-function call that reads stream id inside
// a statement, its rows typed by columns (fields c0, c1, ...).
func StreamScanSQL(id string, columns []ColumnKind) (string, error) {
	if len(columns) == 0 {
		return "", fmt.Errorf("duckdb: quack stream: no columns")
	}
	fields := make([]string, len(columns))
	for i, k := range columns {
		typeName, err := sendDataColumnType(k)
		if err != nil {
			return "", err
		}
		fields[i] = fmt.Sprintf("c%d %s", i, typeName)
	}
	return fmt.Sprintf("scan_data_from_quack_client(%s, NULL::STRUCT(%s), ordered := true)",
		stringLiteral(id), strings.Join(fields, ", ")), nil
}

// errStatementEnded reports that the statement finished before a stream it
// was meant to read ever opened.
var errStatementEnded = errors.New("duckdb: quack stream: statement ended before the stream opened")

// QueryWithStreams runs sqlText (a query reading every stream through
// StreamScanSQL) while feeding each stream concurrently from its Next.
//
// The server opens a statement's streams at BIND and blocks execution until
// they're complete, so the query runs on its own goroutine while each
// stream's batches are POSTed as Next produces them, then closed against
// their batch count. A failing stream cancels the query and its error is
// returned; a failing query stops every stream. A stream the statement
// never opens (e.g. the planner pruned its scan) is simply not sent.
func (s *Session) QueryWithStreams(ctx context.Context, sqlText string, streams []SendStream) (cols []string, types []string, rows []result.Row, err error) {
	// Whichever side fails first cancels the other with its error as the
	// cause, and that error is the one returned: the other side's failure
	// is then only the cancellation (a query that fails to bind must not
	// surface as its streams' "context canceled", nor a stream's source
	// error as the query's).
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	done := make(chan struct{})
	var queryErr error
	go func() {
		defer close(done)
		cols, types, rows, queryErr = s.Query(ctx, sqlText)
		if queryErr != nil {
			cancel(queryErr) // unblock any stream still waiting on its source
		}
	}()

	feedErrs := make(chan error, len(streams))
	for _, st := range streams {
		go func() { feedErrs <- s.feedStream(ctx, st, done) }()
	}
	var feedErr error
	for range streams {
		if err := <-feedErrs; err != nil && feedErr == nil {
			feedErr = err
			cancel(err)
		}
	}
	<-done
	if queryErr != nil && context.Cause(ctx) == queryErr {
		return nil, nil, nil, queryErr
	}
	if feedErr != nil {
		return nil, nil, nil, feedErr
	}
	return cols, types, rows, queryErr
}

// feedStream sends every batch st.Next produces, then the terminal message.
func (s *Session) feedStream(ctx context.Context, st SendStream, done <-chan struct{}) error {
	var batches uint64
	for {
		blob, chunks, err := st.nextBatch(ctx)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("duckdb: quack stream %s: %w", st.ID, err)
		}
		if chunks == 0 {
			continue
		}
		batches++
		idx := batches
		err = s.sendWhenActive(ctx, done, func() []byte {
			return encodeSendDataRequest(s.connectionID, st.ID, chunks, &idx, nil, blob)
		})
		if errors.Is(err, errStatementEnded) {
			return nil
		}
		if err != nil {
			return err
		}
	}
	total := batches
	err := s.sendWhenActive(ctx, done, func() []byte {
		return encodeSendDataRequest(s.connectionID, st.ID, 0, nil, &total, nil)
	})
	if errors.Is(err, errStatementEnded) {
		return nil
	}
	return err
}

// nextBatch returns the stream's next batch, encoded.
func (st SendStream) nextBatch(ctx context.Context) ([]byte, uint64, error) {
	if st.NextEncoded != nil {
		return st.NextEncoded(ctx)
	}
	rows, err := st.Next(ctx)
	if err != nil || len(rows) == 0 {
		return nil, 0, err
	}
	return rowsToChunks(st.Columns, rows)
}

// sendWhenActive POSTs payload(), retrying while the server reports the
// stream isn't open yet (the statement hasn't reached BIND), until the
// statement itself ends.
func (s *Session) sendWhenActive(ctx context.Context, done <-chan struct{}, payload func() []byte) error {
	deadline := time.Now().Add(sendDataStreamReadyTimeout)
	backoff := sendDataStreamReadyMinBackoff
	for {
		err := s.sendDataRequest(ctx, payload())
		if err == nil || !isStreamNotActiveErr(err) {
			return err
		}
		select {
		case <-done:
			return errStatementEnded
		default:
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("duckdb: quack stream never became active: %w", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-done:
			return errStatementEnded
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, sendDataStreamReadyMaxBackoff)
	}
}
