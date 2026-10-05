// Package federate answers queries over the union of an account's lake
// tables and the not-yet-exported delta held in a buffer database (ClickHouse
// in Cloud). The delta is streamed in as batches (DeltaStreamer), ingested
// into per-connection TEMP tables (Ingester), and unioned with the lake by
// logical views (views.go) that apply the latest-wins / is_final rules.
//
// The batch type here is deliberately tiny and dependency-free: OSS carries
// no Arrow dependency. A streamer backed by an Arrow source (e.g. ClickHouse
// FORMAT ArrowStream) converts record batches into Batches itself.
package federate

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/inngest/inngest/pkg/duckdb/schema"

	"github.com/google/uuid"
)

// Column is one named, typed column of a Batch.
type Column struct {
	Name string
	Type schema.Type
}

// Batch is a column-major chunk of rows. Columns[i] holds every row's value
// for Schema[i], and all columns have the same length. A nil value is NULL.
type Batch struct {
	Schema  []Column
	Columns [][]any
}

// Len returns the number of rows in b.
func (b *Batch) Len() int {
	if len(b.Columns) == 0 {
		return 0
	}
	return len(b.Columns[0])
}

// Row returns row i's values in Schema order.
func (b *Batch) Row(i int) []any {
	row := make([]any, len(b.Columns))
	for c := range b.Columns {
		row[c] = b.Columns[c][i]
	}
	return row
}

// Validate checks that every column has one value per row.
func (b *Batch) Validate() error {
	if len(b.Columns) != len(b.Schema) {
		return fmt.Errorf("federate: batch has %d columns for a %d-column schema", len(b.Columns), len(b.Schema))
	}
	n := b.Len()
	for i, col := range b.Columns {
		if len(col) != n {
			return fmt.Errorf("federate: column %q has %d values, want %d", b.Schema[i].Name, len(col), n)
		}
	}
	return nil
}

// BatchReader streams Batches that share one schema. Next returns io.EOF
// once the stream is exhausted. Close releases the underlying stream and is
// safe to call more than once.
type BatchReader interface {
	Schema() []Column
	Next(ctx context.Context) (*Batch, error)
	Close() error
}

// Table is a logical table the views expose and the delta is streamed for.
type Table string

const (
	TableRuns   Table = "runs"
	TableSpans  Table = "spans"
	TableEvents Table = "events"
)

// Op is a predicate's operator: a comparison, LIKE, or a boolean
// combination.
type Op string

const (
	OpEq  Op = "="
	OpNeq Op = "!="
	OpLt  Op = "<"
	OpLte Op = "<="
	OpGt  Op = ">"
	OpGte Op = ">="
	OpIn  Op = "IN"
	// OpLike matches Value, a DuckDB LIKE pattern (no ESCAPE clause).
	OpLike Op = "LIKE"
	// OpAnd and OpOr combine Args; OpNot negates its one Arg.
	OpAnd Op = "AND"
	OpOr  Op = "OR"
	OpNot Op = "NOT"
)

// Predicate is one pushable condition: a comparison on a column or a JSON
// key (Column, Path, Op, Value), or a boolean combination of predicates (Op
// AND or OR over Args, NOT over one Arg). It is a pre-filter: the query
// re-applies the full condition, so a streamer may drop any part it can't
// translate as long as what it applies never keeps fewer rows than the
// condition would (a NOT, or an OR's every branch, has to be translated
// exactly or not at all). Value is a scalar, or a []any for OpIn.
type Predicate struct {
	Column string
	// Path, if set, is a top-level key of the JSON column Column: the
	// predicate compares the key's value as text (col->>'key'). Value is
	// then a string (or []any of strings for OpIn).
	Path  string
	Op    Op
	Value any
	// Args are an AND's or OR's operands, or a NOT's one operand.
	Args []Predicate
}

// IsBool reports whether p combines other predicates (AND, OR, NOT).
func (p Predicate) IsBool() bool { return p.Op == OpAnd || p.Op == OpOr || p.Op == OpNot }

// DeltaRequest asks a DeltaStreamer for one logical table's delta: the rows
// not yet in the lake (bucket_at > Watermark), for one tenant and run-time
// range.
type DeltaRequest struct {
	AccountID uuid.UUID
	EnvID     uuid.UUID
	// AllTenants (internal raw mode only) ignores AccountID/EnvID and
	// returns every tenant's delta.
	AllTenants bool
	Table      Table
	// Columns is the exact projection the views need, in order. The
	// returned BatchReader's schema must match it.
	Columns []Column
	// From and To bound the run's queued time, [From, To) (for events,
	// the event's received time).
	From, To time.Time
	// Watermark is W: the lake holds bucket_at <= W.
	Watermark time.Time
	// Predicates are conjuncts the streamer may pre-filter on: it applies
	// each one it can represent (never more strictly than written) and
	// drops the rest, which the query re-applies.
	Predicates []Predicate
	// RowCap bounds the delta; streaming more is an error, not truncation.
	RowCap int
}

// DeltaStreamer returns the delta for one table.
type DeltaStreamer interface {
	Stream(ctx context.Context, req DeltaRequest) (BatchReader, error)
}

// SliceReader is a BatchReader over in-memory Batches.
type SliceReader struct {
	schema  []Column
	batches []*Batch
	i       int
}

// NewSliceReader returns a BatchReader yielding batches in order.
func NewSliceReader(schema []Column, batches ...*Batch) *SliceReader {
	return &SliceReader{schema: schema, batches: batches}
}

func (r *SliceReader) Schema() []Column { return r.schema }

func (r *SliceReader) Next(context.Context) (*Batch, error) {
	if r.i >= len(r.batches) {
		return nil, io.EOF
	}
	b := r.batches[r.i]
	r.i++
	return b, nil
}

func (r *SliceReader) Close() error { return nil }
