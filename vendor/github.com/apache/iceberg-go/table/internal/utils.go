// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package internal

import (
	"container/heap"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"iter"
	"maps"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
	_ "unsafe"

	"github.com/apache/arrow-go/v18/arrow/decimal"
	"github.com/apache/arrow-go/v18/arrow/decimal128"
	"github.com/apache/iceberg-go"
	iceberginternal "github.com/apache/iceberg-go/internal"
	"github.com/twmb/avro/atype"
	"golang.org/x/sync/errgroup"
)

// Enumerated is a quick way to represent a sequenced value that can
// be processed in parallel and then needs to be reordered.
type Enumerated[T any] struct {
	Value T
	Index int
	Last  bool
}

// a simple priority queue
type pqueue[T any] struct {
	queue   []*T
	compare func(a, b *T) bool
}

func (pq *pqueue[T]) Len() int { return len(pq.queue) }
func (pq *pqueue[T]) Less(i, j int) bool {
	return pq.compare(pq.queue[i], pq.queue[j])
}

func (pq *pqueue[T]) Swap(i, j int) {
	pq.queue[i], pq.queue[j] = pq.queue[j], pq.queue[i]
}

func (pq *pqueue[T]) Push(x any) {
	pq.queue = append(pq.queue, x.(*T))
}

func (pq *pqueue[T]) Pop() any {
	old := pq.queue
	n := len(old)

	item := old[n-1]
	old[n-1] = nil
	pq.queue = old[0 : n-1]

	return item
}

// MakeSequencedChan creates a channel that outputs values in a given order
// based on the comesAfter and isNext functions. The values are read in from
// the provided source and then re-ordered before being sent to the output.
func MakeSequencedChan[T any](bufferSize uint, source <-chan T, comesAfter, isNext func(a, b *T) bool, initial T) <-chan T {
	return MakeSequencedChanWithDiscard(bufferSize, source, comesAfter, isNext, initial, nil)
}

// MakeSequencedChanWithDiscard is MakeSequencedChan with a callback for values
// that remain in the reorder queue when the source is closed. Values already
// sent to the output are not passed to discard.
func MakeSequencedChanWithDiscard[T any](bufferSize uint, source <-chan T, comesAfter, isNext func(a, b *T) bool, initial T, discard func(T)) <-chan T {
	pq := pqueue[T]{queue: make([]*T, 0), compare: comesAfter}
	heap.Init(&pq)
	previous, out := &initial, make(chan T, bufferSize)
	go func() {
		defer close(out)
		defer func() {
			if discard == nil {
				return
			}

			for _, val := range pq.queue {
				discard(*val)
			}
		}()
		for val := range source {
			heap.Push(&pq, &val)
			for pq.Len() > 0 && isNext(previous, pq.queue[0]) {
				previous = heap.Pop(&pq).(*T)
				out <- *previous
			}
		}
	}()

	return out
}

func u64FromBigEndianShifted(buf []byte) uint64 {
	var bytes [8]byte
	copy(bytes[8-len(buf):], buf)

	return binary.BigEndian.Uint64(bytes[:])
}

func BigEndianToDecimal(buf []byte) (decimal.Decimal128, error) {
	const (
		minDecBytes = 1
		maxDecBytes = 16
	)

	if len(buf) < minDecBytes || len(buf) > maxDecBytes {
		return decimal.Decimal128{},
			fmt.Errorf("invalid length for conversion to decimal: %d, must be between %d and %d",
				len(buf), minDecBytes, maxDecBytes)
	}

	// big endian, so first byte is the MSB and host the sign bit
	isNeg := int8(buf[0]) < 0

	var hi, lo int64
	// step 1. extract high bits
	highBitsOffset := max(0, len(buf)-8)
	highBits := u64FromBigEndianShifted(buf[:highBitsOffset])

	if highBitsOffset == 8 {
		hi = int64(highBits)
	} else {
		if isNeg && len(buf) < maxDecBytes {
			hi = -1
		}

		hi = int64(uint64(hi) << (uint64(highBitsOffset) * 8))
		hi |= int64(highBits)
	}

	// step 2. extract low bits
	lowBitsOffset := min(len(buf), 8)
	lowBits := u64FromBigEndianShifted(buf[highBitsOffset:])
	if lowBitsOffset == 8 {
		lo = int64(lowBits)
	} else {
		if isNeg && len(buf) < 8 {
			lo = -1
		}

		lo = int64(uint64(lo) << (uint64(lowBitsOffset) * 8))
		lo |= int64(lowBits)
	}

	return decimal128.New(hi, uint64(lo)), nil
}

func PartitionRecordValue(field iceberg.PartitionField, val iceberg.Literal, schema *iceberg.Schema) (iceberg.Optional[iceberg.Literal], error) {
	var ret iceberg.Optional[iceberg.Literal]
	if val == nil {
		return ret, nil
	}

	f, ok := schema.FindFieldByIDRef(field.SourceID(), iceberginternal.SchemaRef{})
	if !ok {
		return ret, fmt.Errorf("%w: could not find source field in schema for %s",
			iceberg.ErrInvalidSchema, field.String())
	}

	return partitionRecordValue(val, f.Type)
}

func partitionRecordValue(val iceberg.Literal, sourceType iceberg.Type) (iceberg.Optional[iceberg.Literal], error) {
	var ret iceberg.Optional[iceberg.Literal]
	if val == nil {
		return ret, nil
	}

	out, err := val.To(sourceType)
	if err != nil {
		return ret, err
	}

	ret.Val = out
	ret.Valid = true

	return ret, nil
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}

	return v
}

type TypedStats[T iceberg.LiteralType] interface {
	Min() T
	Max() T
}

type StatsAgg interface {
	Min() iceberg.Literal
	Max() iceberg.Literal
	Update(stats interface{ HasMinMax() bool })
	MinAsBytes() ([]byte, error)
	MaxAsBytes() ([]byte, error)
}

type DataFileStatistics struct {
	RecordCount      int64
	ColSizes         map[int]int64
	ValueCounts      map[int]int64
	NullValueCounts  map[int]int64
	NanValueCounts   map[int]int64
	ColAggs          map[int]StatsAgg
	SplitOffsets     []int64
	EqualityFieldIDs []int
	// Variant bounds: serialized variant objects keyed by parent variant field id.
	VariantLowerBounds map[int][]byte
	VariantUpperBounds map[int][]byte
}

func (d *DataFileStatistics) PartitionValue(field iceberg.PartitionField, sc *iceberg.Schema) any {
	agg, ok := d.ColAggs[field.SourceID()]
	if !ok {
		return nil
	}

	if !field.Transform.PreservesOrder() {
		return nil
	}

	sourceField, ok := sc.FindFieldByIDRef(field.SourceID(), iceberginternal.SchemaRef{})
	if !ok {
		panic(fmt.Errorf("%w: could not find source field in schema for %s",
			iceberg.ErrInvalidSchema, field.String()))
	}

	return d.partitionValueWithSourceType(field, agg, sourceField.Type)
}

func (d *DataFileStatistics) partitionValueWithSourceType(field iceberg.PartitionField, agg StatsAgg, sourceType iceberg.Type) any {
	if agg == nil {
		return nil
	}

	lowerRec := must(partitionRecordValue(agg.Min(), sourceType))
	upperRec := must(partitionRecordValue(agg.Max(), sourceType))

	lowerT := field.Transform.Apply(lowerRec)
	upperT := field.Transform.Apply(upperRec)

	if !lowerT.Valid || !upperT.Valid {
		return nil
	}

	if !lowerT.Val.Equals(upperT.Val) {
		panic(fmt.Errorf("cannot infer partition value from parquet metadata as there is more than one value for partition field: %s. (low: %s, high: %s)",
			field.Name, lowerT.Val, upperT.Val))
	}

	return lowerT.Val.Any()
}

// DataFileOpts groups the fields needed to finalize a DataFile from
// DataFileStatistics. Collapsing the former positional arguments into a
// struct keeps call sites readable and makes future additions source-
// compatible.
type DataFileOpts struct {
	Schema          *iceberg.Schema
	Spec            iceberg.PartitionSpec
	Path            string
	Format          iceberg.FileFormat
	Content         iceberg.ManifestEntryContent
	FileSize        int64
	PartitionValues map[int]any
	// SortOrderID claims the file's rows are fully sorted by that order; zero
	// makes no claim and leaves the field absent.
	SortOrderID int
	// FirstRowID is honored only for EntryContentData; the spec requires it to
	// be null for delete files.
	FirstRowID         *int64
	ReferencedDataFile *string
}

// unsortedSortOrderID mirrors table.UnsortedSortOrderID.
const unsortedSortOrderID = 0

func (d *DataFileStatistics) ToDataFile(opts DataFileOpts) iceberg.DataFile {
	var fieldIDToPartitionData map[int]any
	fieldIDToLogicalType := make(map[int]string)

	if !opts.Spec.Equals(*iceberg.UnpartitionedSpec) {
		fieldIDToPartitionData = make(map[int]any)
		for _, field := range opts.Spec.Fields() {
			sourceField, sourceFieldFound := opts.Schema.FindFieldByIDRef(
				field.SourceID(), iceberginternal.SchemaRef{})
			partitionVal := opts.PartitionValues[field.FieldID]
			switch {
			case partitionVal != nil:
				// prioritizing caller-supplied value.
				fieldIDToPartitionData[field.FieldID] = partitionVal
			case field.Transform.PreservesOrder() && sourceFieldFound:
				fieldIDToPartitionData[field.FieldID] = d.partitionValueWithSourceType(
					field, d.ColAggs[field.SourceID()], sourceField.Type)
			case field.Transform.PreservesOrder():
				// Preserve the existing error path for an invalid schema. This branch
				// is not part of the valid hot path because sourceFieldFound is false.
				fieldIDToPartitionData[field.FieldID] = d.PartitionValue(field, opts.Schema)
			default:
				fieldIDToPartitionData[field.FieldID] = nil
			}

			if sourceFieldFound {
				resultType := field.Transform.ResultType(sourceField.Type)

				switch resultType.(type) {
				case iceberg.DateType:
					fieldIDToLogicalType[field.FieldID] = atype.Date
				case iceberg.TimeType:
					fieldIDToLogicalType[field.FieldID] = atype.TimeMicros
				case iceberg.TimestampType:
					fieldIDToLogicalType[field.FieldID] = atype.TimestampMicros
				case iceberg.TimestampTzType:
					fieldIDToLogicalType[field.FieldID] = atype.TimestampMicros
				case iceberg.DecimalType:
					fieldIDToLogicalType[field.FieldID] = atype.Decimal
				case iceberg.UUIDType:
					fieldIDToLogicalType[field.FieldID] = atype.UUID
				}
			}
		}
	}

	bldr, err := iceberg.NewDataFileBuilder(opts.Spec, opts.Content,
		opts.Path, opts.Format, fieldIDToPartitionData, fieldIDToLogicalType, nil, d.RecordCount, opts.FileSize)
	if err != nil {
		panic(err)
	}
	lowerBounds := make(map[int][]byte)
	upperBounds := make(map[int][]byte)

	for fieldID, agg := range d.ColAggs {
		min := must(agg.MinAsBytes())
		max := must(agg.MaxAsBytes())
		if len(min) > 0 {
			lowerBounds[fieldID] = min
		}
		if len(max) > 0 {
			upperBounds[fieldID] = max
		}
	}

	// Variant bounds are serialized objects keyed by the parent variant field id.
	maps.Copy(lowerBounds, d.VariantLowerBounds)
	maps.Copy(upperBounds, d.VariantUpperBounds)

	if len(lowerBounds) > 0 {
		bldr.LowerBoundValues(lowerBounds)
	}
	if len(upperBounds) > 0 {
		bldr.UpperBoundValues(upperBounds)
	}

	bldr.ColumnSizes(d.ColSizes)
	bldr.ValueCounts(d.ValueCounts)
	bldr.NullValueCounts(d.NullValueCounts)
	bldr.NaNValueCounts(d.NanValueCounts)
	bldr.SplitOffsets(d.SplitOffsets)

	if len(d.EqualityFieldIDs) > 0 {
		bldr.EqualityFieldIDs(d.EqualityFieldIDs)
	}

	// Claim invariants are enforced upstream in defaultDataFileWriter.writeFile.
	if opts.SortOrderID != unsortedSortOrderID {
		bldr.SortOrderID(opts.SortOrderID)
	}

	if opts.FirstRowID != nil && opts.Content == iceberg.EntryContentData {
		bldr.FirstRowID(*opts.FirstRowID)
	}

	if opts.ReferencedDataFile != nil && opts.Content == iceberg.EntryContentPosDeletes {
		bldr.ReferencedDataFile(*opts.ReferencedDataFile)
	}

	return bldr.Build()
}

type MetricModeType string

const (
	MetricModeTruncate MetricModeType = "truncate"
	MetricModeNone     MetricModeType = "none"
	MetricModeCounts   MetricModeType = "counts"
	MetricModeFull     MetricModeType = "full"
)

type MetricsMode struct {
	Typ MetricModeType
	Len int
}

var truncationExpr = regexp.MustCompile(`^truncate\((\d+)\)$`)

func MatchMetricsMode(mode string) (MetricsMode, error) {
	sanitized := strings.ToLower(strings.TrimSpace(mode))
	if strings.HasPrefix(sanitized, string(MetricModeTruncate)) {
		m := truncationExpr.FindStringSubmatch(sanitized)
		if len(m) < 2 {
			return MetricsMode{}, fmt.Errorf("malformed truncate metrics mode: %s", mode)
		}

		truncLen, err := strconv.Atoi(m[1])
		if err != nil {
			return MetricsMode{}, fmt.Errorf("malformed truncate metrics mode: %s", mode)
		}

		if truncLen <= 0 {
			return MetricsMode{}, fmt.Errorf("invalid truncate length: %d", truncLen)
		}

		return MetricsMode{Typ: MetricModeTruncate, Len: truncLen}, nil
	}

	switch sanitized {
	case string(MetricModeNone):
		return MetricsMode{Typ: MetricModeNone}, nil
	case string(MetricModeCounts):
		return MetricsMode{Typ: MetricModeCounts}, nil
	case string(MetricModeFull):
		return MetricsMode{Typ: MetricModeFull}, nil
	default:
		return MetricsMode{}, fmt.Errorf("unsupported metrics mode: %s", mode)
	}
}

type StatisticsCollector struct {
	FieldID    int
	IcebergTyp iceberg.PrimitiveType
	Mode       MetricsMode
	ColName    string
}

type typedStat[T iceberg.LiteralType] interface {
	Min() T
	Max() T
}

type statsAggregator[T iceberg.LiteralType] struct {
	curMin iceberg.TypedLiteral[T]
	curMax iceberg.TypedLiteral[T]

	cmp           iceberg.Comparator[T]
	primitiveType iceberg.PrimitiveType
	truncLen      int
}

func newStatAgg[T iceberg.LiteralType](typ iceberg.PrimitiveType, trunc int) StatsAgg {
	var z T

	return &statsAggregator[T]{
		primitiveType: typ,
		truncLen:      trunc,
		cmp:           iceberg.NewLiteral(z).(iceberg.TypedLiteral[T]).Comparator(),
	}
}

func (s *statsAggregator[T]) Min() iceberg.Literal { return s.curMin }
func (s *statsAggregator[T]) Max() iceberg.Literal { return s.curMax }

func (s *statsAggregator[T]) Update(stats interface{ HasMinMax() bool }) {
	st := stats.(typedStat[T])
	s.updateMin(st.Min())
	s.updateMax(st.Max())
}

func (s *statsAggregator[T]) toBytes(val iceberg.Literal) ([]byte, error) {
	v, err := val.To(s.primitiveType)
	if err != nil {
		return nil, err
	}

	return v.MarshalBinary()
}

func (s *statsAggregator[T]) updateMin(val T) {
	if s.curMin == nil {
		s.curMin = iceberg.NewLiteral(val).(iceberg.TypedLiteral[T])
	} else {
		if s.cmp(val, s.curMin.Value()) < 0 {
			s.curMin = iceberg.NewLiteral(val).(iceberg.TypedLiteral[T])
		}
	}
}

func (s *statsAggregator[T]) updateMax(val T) {
	if s.curMax == nil {
		s.curMax = iceberg.NewLiteral(val).(iceberg.TypedLiteral[T])
	} else {
		if s.cmp(val, s.curMax.Value()) > 0 {
			s.curMax = iceberg.NewLiteral(val).(iceberg.TypedLiteral[T])
		}
	}
}

func (s *statsAggregator[T]) MinAsBytes() ([]byte, error) {
	if s.curMin == nil {
		return nil, nil
	}

	if s.truncLen > 0 {
		return s.toBytes((&iceberg.TruncateTransform{Width: s.truncLen}).
			Apply(iceberg.Optional[iceberg.Literal]{Valid: true, Val: s.curMin}).Val)
	}

	return s.toBytes(s.curMin)
}

func (s *statsAggregator[T]) MaxAsBytes() ([]byte, error) {
	if s.curMax == nil {
		return nil, nil
	}

	if s.truncLen <= 0 {
		return s.toBytes(s.curMax)
	}

	switch s.primitiveType.(type) {
	case iceberg.StringType:
		if !s.curMax.Type().Equals(s.primitiveType) {
			return nil, errors.New("expected current max to be a string")
		}

		curMax := any(s.curMax.Value()).(string)
		result := TruncateUpperBoundText(curMax, s.truncLen)
		if result != "" {
			return s.toBytes(iceberg.StringLiteral(result))
		}

		return nil, nil
	case iceberg.BinaryType:
		if !s.curMax.Type().Equals(s.primitiveType) {
			return nil, errors.New("expected current max to be a binary")
		}

		curMax := any(s.curMax.Value()).([]byte)
		result := TruncateUpperBoundBinary(curMax, s.truncLen)
		if len(result) > 0 {
			return s.toBytes(iceberg.BinaryLiteral(result))
		}

		return nil, nil
	default:
		return nil, fmt.Errorf("%s cannot be truncated for upper bound", s.primitiveType)
	}
}

// truncateString returns the prefix containing at most width UTF-8 code
// points. The boolean is true when the input was actually truncated.
func truncateString(s string, width int) (string, bool) {
	for idx := range s {
		if width == 0 {
			return s[:idx], true
		}

		width--
	}

	return s, false
}

func TruncateUpperBoundText(s string, trunc int) string {
	result, _ := truncateUpperBoundText(s, trunc)

	return result
}

func truncateUpperBoundText(s string, trunc int) (string, bool) {
	result, truncated := truncateString(s, trunc)
	if !truncated {
		return s, false
	}

	for len(result) > 0 {
		r, size := utf8.DecodeLastRuneInString(result)
		if next, ok := nextValidRune(r); ok {
			return result[:len(result)-size] + string(next), true
		}

		result = result[:len(result)-size]
	}

	return "", true
}

func nextValidRune(r rune) (rune, bool) {
	switch {
	case r == '\uD7FF':
		return '\uE000', true
	case r >= utf8.MaxRune:
		return 0, false
	default:
		return r + 1, true
	}
}

func TruncateUpperBoundBinary(val []byte, trunc int) []byte {
	if trunc < 0 {
		// A negative width cannot describe a prefix. Keep the original bound
		// instead of panicking on the slice expression below.
		return slices.Clone(val)
	}
	if trunc >= len(val) {
		return slices.Clone(val)
	}

	result := slices.Clone(val[:trunc])

	for i := len(result) - 1; i >= 0; i-- {
		if result[i] < 255 {
			result[i]++

			return result[:i+1]
		}
	}

	return nil
}

func MapExec[T, S any](ctx context.Context, nWorkers int, slice iter.Seq[T], fn func(T) (S, error)) iter.Seq2[S, error] {
	if nWorkers <= 0 {
		nWorkers = runtime.GOMAXPROCS(0)
	}

	g, ctx := errgroup.WithContext(ctx)
	ch := make(chan T, nWorkers)
	out := make(chan S, nWorkers)

	for range nWorkers {
		g.Go(func() error {
			for v := range ch {
				result, err := fn(v)
				if err != nil {
					return err
				}
				select {
				case out <- result:
				case <-ctx.Done():
					return context.Cause(ctx)
				}
			}

			return nil
		})
	}

	var err error
	go func() {
		defer func() {
			close(ch)
			err = g.Wait()
			close(out)
		}()

		for v := range slice {
			select {
			case ch <- v:
			case <-ctx.Done():
				return
			}
		}
	}()

	return func(yield func(S, error) bool) {
		defer func() {
			// drain out if we exit early
			for range out {
			}
		}()

		for v := range out {
			if !yield(v, nil) {
				return
			}
		}

		if err != nil {
			var z S
			yield(z, err)
		}
	}
}
