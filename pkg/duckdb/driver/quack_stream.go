package driver

import (
	"context"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/quack"
)

// QuackStream feeds one relation of a query directly from Go, over quack's
// SEND_DATA mechanism, with no table in between. The query reads it through
// QuackStreamScan's table function (columns c0, c1, ... typed per Columns),
// and QueryContext feeds it while the query runs, so rows arrive as the
// source produces them.
//
// A stream can be read once: a query that reads it more than once must put
// the scan in a MATERIALIZED CTE.
type QuackStream struct {
	ID      string
	Columns []QuackColumnKind
	// Next returns the next rows (one value per column, in Columns' wire
	// forms, as for QuackAppender.AppendRow), or io.EOF once done. Any
	// other error aborts the query and is returned from QueryContext. It's
	// called from its own goroutine and must honor ctx.
	Next func(ctx context.Context) ([][]any, error)
	// NextEncoded, when set, replaces Next for a producer that encodes its
	// own batches with EncodeQuackVectors (column-major, no per-value
	// boxing): it returns a batch's blob and chunk count, or io.EOF.
	NextEncoded func(ctx context.Context) (blob []byte, chunks uint64, err error)
}

// QuackVector is one column of a batch in column-major form; see
// EncodeQuackVectors.
type QuackVector = quack.Vector

// EncodeQuackVectors encodes equal-length vectors as a stream batch for
// QuackStream.NextEncoded: the same bytes the row path writes for the same
// values.
func EncodeQuackVectors(vectors []QuackVector) (blob []byte, chunks uint64, err error) {
	return quack.EncodeVectors(vectors)
}

// NewQuackStreamID returns a fresh stream id.
func NewQuackStreamID() (string, error) { return quack.NewStreamID() }

// QuackStreamScan returns the table-function call that reads stream id in
// a query, e.g. scan_data_from_quack_client('id', NULL::STRUCT(c0 VARCHAR), ordered := true).
func QuackStreamScan(id string, columns []QuackColumnKind) (string, error) {
	return quack.StreamScanSQL(id, columns)
}

type quackStreamsKey struct{}

// WithQuackStreams attaches streams to ctx: a QueryContext on a quack
// connection with this ctx feeds them while its query runs. Only quack
// connections support it; any other transport fails the query.
func WithQuackStreams(ctx context.Context, streams ...QuackStream) context.Context {
	return context.WithValue(ctx, quackStreamsKey{}, streams)
}

func quackStreamsFrom(ctx context.Context) []QuackStream {
	streams, _ := ctx.Value(quackStreamsKey{}).([]QuackStream)
	return streams
}

func toSendStreams(streams []QuackStream) []quack.SendStream {
	out := make([]quack.SendStream, len(streams))
	for i, s := range streams {
		out[i] = quack.SendStream{ID: s.ID, Columns: s.Columns, Next: s.Next, NextEncoded: s.NextEncoded}
	}
	return out
}
