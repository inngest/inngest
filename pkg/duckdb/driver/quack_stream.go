package driver

import (
	"context"
	"errors"
	"fmt"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/quack"
	"github.com/inngest/inngest/pkg/duckdb/driver/internal/result"
	"github.com/inngest/inngest/pkg/duckdb/parser"
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

// QuackStreamScan returns the table function that reads stream id in a
// query, e.g. scan_data_from_quack_client('id', NULL::STRUCT(c0 VARCHAR), ordered := true):
// its rows are typed by columns, in fields c0, c1, ...
func QuackStreamScan(id string, columns []QuackColumnKind) (*parser.TableFunctionRef, error) {
	if len(columns) == 0 {
		return nil, fmt.Errorf("duckdb: quack stream: no columns")
	}
	fields := make([]*parser.TypeField, len(columns))
	for i, k := range columns {
		name, err := quack.StreamColumnType(k)
		if err != nil {
			return nil, err
		}
		typ, err := parser.ParseType(name)
		if err != nil {
			return nil, err
		}
		fields[i] = &parser.TypeField{Name: fmt.Sprintf("c%d", i), Type: typ}
	}
	return &parser.TableFunctionRef{
		Name: []string{"scan_data_from_quack_client"},
		Args: []parser.Expr{
			&parser.Literal{Kind: parser.LitString, Text: id},
			&parser.CastExpr{
				X:    &parser.Literal{Kind: parser.LitNull},
				Type: &parser.TypeName{Name: []string{"STRUCT"}, Keyword: true, Fields: fields},
			},
			&parser.NamedArg{Name: "ordered", Value: &parser.Literal{Kind: parser.LitTrue}},
		},
	}, nil
}

// errQuackStreamsTransport is returned for a stream query on a connection
// that isn't quack.
var errQuackStreamsTransport = errors.New("duckdb: quack streams require a quack-transport connection (Options.QuackAddr)")

// streamQuerier is a transport that can run a statement reading quack
// streams.
type streamQuerier interface {
	QueryWithStreams(ctx context.Context, sqlText string, streams []quack.SendStream) (cols []string, types []string, rows []result.Row, err error)
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
