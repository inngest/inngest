package federate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/inngest/inngest/pkg/duckdb/parser"
	"github.com/inngest/inngest/pkg/duckdb/schema"
)

// ErrRowCapExceeded is returned when a delta exceeds its row cap.
var ErrRowCapExceeded = errors.New("federate: delta row cap exceeded")

func quackKind(t schema.Type) driver.QuackColumnKind {
	switch t {
	case schema.TypeUUID:
		return driver.QuackColumnUUID
	case schema.TypeJSON:
		return driver.QuackColumnJSON
	case schema.TypeTimestamp:
		return driver.QuackColumnTimestampTZ
	case schema.TypeBool:
		return driver.QuackColumnBool
	case schema.TypeBigint:
		return driver.QuackColumnBigint
	default: // schema.TypeVarchar, schema.TypeVarcharList (as list-literal text)
		return driver.QuackColumnVarchar
	}
}

// toWire converts one row's values into what the quack appender accepts for
// each column's kind.
func toWire(cols []Column, row []any) ([]any, error) {
	out := make([]any, len(row))
	for i, v := range row {
		if v == nil {
			continue
		}
		switch cols[i].Type {
		case schema.TypeVarcharList:
			list, ok := v.([]string)
			if !ok {
				return nil, fmt.Errorf("column %q: expected []string, got %T", cols[i].Name, v)
			}
			out[i] = string(AppendListLiteral(nil, list))
		case schema.TypeJSON:
			switch j := v.(type) {
			case string:
				out[i] = j
			case []byte:
				out[i] = string(j)
			case json.RawMessage:
				out[i] = string(j)
			default:
				return nil, fmt.Errorf("column %q: expected JSON text, got %T", cols[i].Name, v)
			}
		default:
			out[i] = v
		}
	}
	return out, nil
}

// deltaScanSelect returns the wire kinds of cols, the SELECT reading a quack
// stream of them (each column cast to its DuckDB type and named), and the
// stream's fresh id.
func deltaScanSelect(cols []Column) ([]driver.QuackColumnKind, *parser.SelectStatement, string, error) {
	kinds := make([]driver.QuackColumnKind, len(cols))
	sel := make([]*parser.SelectItem, len(cols))
	for i, c := range cols {
		kinds[i] = quackKind(c.Type)
		sel[i] = item(cast(colRef(fmt.Sprintf("c%d", i)), c.Type.DuckDBType()), c.Name)
	}
	id, err := driver.NewQuackStreamID()
	if err != nil {
		return nil, nil, "", err
	}
	scan, err := driver.QuackStreamScan(id, kinds)
	if err != nil {
		return nil, nil, "", err
	}
	return kinds, &parser.SelectStatement{Columns: sel, From: from(scan)}, id, nil
}

// AppendListLiteral appends vals as the DuckDB list literal a VARCHAR →
// VARCHAR[] cast parses (["a","b"]): each element double-quoted, with only
// '\\' and '"' backslash-escaped. DuckDB's list parser reads a backslash as
// "the next character, literally" and knows no JSON escapes, so JSON text
// isn't safe here: json.Marshal's \u003c for '<' would arrive as "u003c".
func AppendListLiteral(buf []byte, vals []string) []byte { return appendList(buf, vals) }

// AppendListLiteralBytes is AppendListLiteral for byte-slice elements, so a
// caller holding them in a buffer needn't allocate strings.
func AppendListLiteralBytes(buf []byte, vals [][]byte) []byte { return appendList(buf, vals) }

func appendList[S ~string | ~[]byte](buf []byte, vals []S) []byte {
	buf = append(buf, '[')
	for i, v := range vals {
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = append(buf, '"')
		for j := 0; j < len(v); j++ {
			if c := v[j]; c == '\\' || c == '"' {
				buf = append(buf, '\\')
			}
			buf = append(buf, v[j])
		}
		buf = append(buf, '"')
	}
	return append(buf, ']')
}

// WireKind is the quack wire kind a column of type t travels as: the kind
// a streamed delta's scan declares for it.
func WireKind(t schema.Type) driver.QuackColumnKind { return quackKind(t) }

// deltaCTEName is the MATERIALIZED CTE a streamed delta is read through.
// Like SourceName, it's under GeneratedPrefix.
func deltaCTEName(t Table) string { return GeneratedPrefix + "delta_" + string(t) }

// deltaStream turns a delta reader into a quack stream and the MATERIALIZED
// CTE definition that reads it, each column typed and named as its lake
// column. Exceeding rowCap fails the query with ErrRowCapExceeded.
func deltaStream(t Table, r BatchReader, rowCap int) (driver.QuackStream, *parser.CTE, error) {
	st, scan, err := deltaScan(t, r, rowCap)
	if err != nil {
		return driver.QuackStream{}, nil, err
	}
	return st, materializedCTE(deltaCTEName(t), scan), nil
}

// deltaScan turns a delta reader into a quack stream and the SELECT that
// reads it, each column typed and named as its lake column.
func deltaScan(t Table, r BatchReader, rowCap int) (driver.QuackStream, *parser.SelectStatement, error) {
	schema := r.Schema()
	if len(schema) == 0 {
		return driver.QuackStream{}, nil, fmt.Errorf("federate: delta for %s has no columns", t)
	}
	kinds, def, id, err := deltaScanSelect(schema)
	if err != nil {
		return driver.QuackStream{}, nil, err
	}

	var total int
	next := func(ctx context.Context) ([][]any, error) {
		b, err := r.Next(ctx)
		if err != nil {
			return nil, err // io.EOF ends the stream
		}
		// A ragged batch would panic encoding its rows, on the stream's own
		// goroutine.
		if err := b.validateFor(schema); err != nil {
			return nil, fmt.Errorf("federate: %s delta: %w", t, err)
		}
		total += b.Len()
		if rowCap > 0 && total > rowCap {
			return nil, fmt.Errorf("%w: %s delta has more than %d rows", ErrRowCapExceeded, t, rowCap)
		}
		rows := make([][]any, b.Len())
		for i := range rows {
			if rows[i], err = toWire(schema, b.Row(i)); err != nil {
				return nil, fmt.Errorf("federate: %s delta: %w", t, err)
			}
		}
		return rows, nil
	}
	return driver.QuackStream{ID: id, Columns: kinds, Next: next}, def, nil
}
