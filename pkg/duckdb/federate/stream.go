package federate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/driver"
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
		return driver.QuackColumnTimestampMS
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

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// deltaScanSQL returns the wire kinds of cols, the SELECT reading a quack
// stream of them (each column cast to its DuckDB type and named), and the
// stream's fresh id.
func deltaScanSQL(cols []Column) ([]driver.QuackColumnKind, string, string, error) {
	kinds := make([]driver.QuackColumnKind, len(cols))
	sel := make([]string, len(cols))
	for i, c := range cols {
		kinds[i] = quackKind(c.Type)
		sel[i] = fmt.Sprintf("CAST(c%d AS %s) AS %s", i, c.Type.DuckDBType(), quoteIdent(c.Name))
	}
	id, err := driver.NewQuackStreamID()
	if err != nil {
		return nil, "", "", err
	}
	scan, err := driver.QuackStreamScan(id, kinds)
	if err != nil {
		return nil, "", "", err
	}
	return kinds, fmt.Sprintf("SELECT %s FROM %s", strings.Join(sel, ", "), scan), id, nil
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
func deltaCTEName(t Table) string { return "__delta_" + string(t) }

// deltaStream turns a delta reader into a quack stream and the MATERIALIZED
// CTE definition that reads it, each column typed and named as its lake
// column. Exceeding rowCap fails the query with ErrRowCapExceeded.
func deltaStream(t Table, r BatchReader, rowCap int) (driver.QuackStream, string, error) {
	st, scan, err := deltaScan(t, r, rowCap)
	if err != nil {
		return driver.QuackStream{}, "", err
	}
	return st, fmt.Sprintf("%s AS MATERIALIZED (%s)", quoteIdent(deltaCTEName(t)), scan), nil
}

// deltaScan turns a delta reader into a quack stream and the SELECT that
// reads it, each column typed and named as its lake column.
func deltaScan(t Table, r BatchReader, rowCap int) (driver.QuackStream, string, error) {
	schema := r.Schema()
	if len(schema) == 0 {
		return driver.QuackStream{}, "", fmt.Errorf("federate: delta for %s has no columns", t)
	}
	kinds, def, id, err := deltaScanSQL(schema)
	if err != nil {
		return driver.QuackStream{}, "", err
	}

	count := &deltaCount{table: t, kind: deltaRows, rowCap: rowCap}
	next := func(ctx context.Context) ([][]any, error) {
		b, err := r.Next(ctx)
		if errors.Is(err, io.EOF) {
			count.done(ctx)
		}
		if err != nil {
			return nil, err // io.EOF ends the stream
		}
		if err := count.add(ctx, b.Len()); err != nil {
			return nil, fmt.Errorf("%w: %s delta has more than %d rows", err, t, rowCap)
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
