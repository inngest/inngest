package federate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// EncodedDeltaStreamer is an optional DeltaStreamer extension: it streams a
// delta already encoded for quack (driver.EncodeQuackVectors), column by
// column, so the executor's streaming path sends it without boxing each
// value. The executor prefers it to Stream.
type EncodedDeltaStreamer interface {
	StreamEncoded(ctx context.Context, req DeltaRequest) (EncodedReader, error)
}

// EncodedReader is a delta stream of encoded batches. NextEncoded returns
// io.EOF once the stream is exhausted. Close is safe to call more than once.
type EncodedReader interface {
	Schema() []Column
	NextEncoded(ctx context.Context) (blob []byte, chunks uint64, rows int, err error)
	Close() error
}

// encodedDeltaStream is deltaStream for an EncodedReader.
func encodedDeltaStream(t Table, r EncodedReader, rowCap int, kind deltaKind) (driver.QuackStream, *parser.CTE, error) {
	cols := r.Schema()
	if len(cols) == 0 {
		return driver.QuackStream{}, nil, fmt.Errorf("federate: delta for %s has no columns", t)
	}
	kinds, scan, id, err := deltaScanSelect(cols)
	if err != nil {
		return driver.QuackStream{}, nil, err
	}
	def := materializedCTE(deltaCTEName(t), scan)
	count := &deltaCount{table: t, kind: kind, rowCap: rowCap}
	next := func(ctx context.Context) ([]byte, uint64, error) {
		blob, chunks, rows, err := r.NextEncoded(ctx)
		if errors.Is(err, io.EOF) {
			count.done(ctx)
		}
		if err != nil {
			return nil, 0, err // io.EOF ends the stream
		}
		if err := count.add(ctx, rows); err != nil {
			return nil, 0, fmt.Errorf("%w: %s delta has more than %d rows", err, t, rowCap)
		}
		return blob, chunks, nil
	}
	return driver.QuackStream{ID: id, Columns: kinds, NextEncoded: next}, def, nil
}

// deltaColumnsUsed works out which delta columns the query reads, from its
// AST (sourceColumnsRead): the columns it reads from each source CTE, plus
// the delta side's own bounds (deltaBoundColumns).
//
// A table maps to nil when the query never reaches its source (it's in
// Tables but unread), or when its shape isn't one the lineage models: stream
// every column.
func deltaColumnsUsed(q Query, needed []Table) (map[Table][]Column, error) {
	if len(needed) == 0 {
		return nil, nil
	}
	stmt, _, err := assemble(q, needed, map[Table]string{}, nil, nil)
	if err != nil {
		return nil, err
	}
	read, err := sourceColumnsRead(stmt, needed)
	if errors.Is(err, errLineage) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := map[Table][]Column{}
	for _, t := range needed {
		d, found := read[t]
		if !found || d.all {
			out[t] = nil
			continue
		}
		names := map[string]bool{}
		for n := range d.names {
			names[n] = true
		}
		for _, c := range deltaBoundColumns(t, q.AllTenants) {
			names[c] = true
		}
		cols := []Column{}
		for _, c := range DeltaColumns(t) {
			if names[strings.ToLower(c.Name)] {
				cols = append(cols, c)
			}
		}
		out[t] = cols
	}
	return out, nil
}

func columnSet(cols []Column) map[string]bool {
	set := make(map[string]bool, len(cols))
	for _, c := range cols {
		set[c.Name] = true
	}
	return set
}
