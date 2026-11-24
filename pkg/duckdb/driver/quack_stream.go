package driver

import (
	"context"
	"errors"
	"fmt"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/result"
	"github.com/inngest/inngest/pkg/duckdb/parser"
	"github.com/inngest/inngest/pkg/duckdb/worker"
)

// QuackStream feeds one relation of a query directly from Go into the DuckDB
// worker's `scan_data_from_quack_client` table function.
type QuackStream struct {
	ID          string
	Columns     []QuackColumnKind
	Next        func(ctx context.Context) ([][]any, error)
	NextEncoded func(ctx context.Context) (blob []byte, chunks uint64, err error)
}

// QuackVector is one column of a batch in column-major form; see EncodeQuackVectors.
type QuackVector = worker.Vector

// EncodeQuackVectors encodes equal-length vectors as a stream batch for
// QuackStream.NextEncoded.
func EncodeQuackVectors(vectors []QuackVector) (blob []byte, chunks uint64, err error) {
	return worker.EncodeVectors(vectors)
}

// NewQuackStreamID returns a fresh stream id.
func NewQuackStreamID() (string, error) { return worker.NewStreamID() }

// QuackStreamScan returns the table function that reads stream id in a query:
// `scan_data_from_quack_client('id', NULL::STRUCT(c0 VARCHAR), ordered := true)`.
func QuackStreamScan(id string, columns []QuackColumnKind) (*parser.TableFunctionRef, error) {
	if len(columns) == 0 {
		return nil, fmt.Errorf("duckdb: quack stream: no columns")
	}
	fields := make([]*parser.TypeField, len(columns))
	for i, k := range columns {
		name, err := worker.StreamColumnType(k)
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

var errQuackStreamsTransport = errors.New("duckdb: streams require a worker-transport connection")

type streamQuerier interface {
	QueryWithStreams(ctx context.Context, sqlText string, streams []worker.SendStream) (cols []string, types []string, rows []result.Row, err error)
}

type quackStreamsKey struct{}

// WithQuackStreams attaches streams to ctx: a QueryContext with this ctx feeds
// them while its query runs.
func WithQuackStreams(ctx context.Context, streams ...QuackStream) context.Context {
	return context.WithValue(ctx, quackStreamsKey{}, streams)
}

func quackStreamsFrom(ctx context.Context) []QuackStream {
	streams, _ := ctx.Value(quackStreamsKey{}).([]QuackStream)
	return streams
}

func toSendStreams(streams []QuackStream) []worker.SendStream {
	out := make([]worker.SendStream, len(streams))
	for i, s := range streams {
		out[i] = worker.SendStream{ID: s.ID, Columns: s.Columns, Next: s.Next, NextEncoded: s.NextEncoded}
	}
	return out
}
