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

package table

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"math"
	"math/bits"
	"slices"
	"strings"
	"sync"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/compute"
	"github.com/apache/arrow-go/v18/arrow/decimal128"
	"github.com/apache/arrow-go/v18/arrow/extensions"
	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/internal"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

// partitionedFanoutWriter distributes Arrow records across multiple partitions based on
// a partition specification, writing data to separate files for each partition using
// a fanout pattern with configurable parallelism.
type partitionedFanoutWriter struct {
	partitionSpec iceberg.PartitionSpec
	schema        *iceberg.Schema
	itr           iter.Seq2[arrow.RecordBatch, error]
	writerFactory *writerFactory
	// Equality-delete inputs may contain partition columns omitted from the output schema,
	// so compile from the first actual batch and share the immutable plan across workers.
	planOnce sync.Once
	plan     *partitionExtractionPlan
	planErr  error
}

// PartitionInfo holds the row indices and partition values for a specific partition,
// used during the fanout process to group rows by their partition key.
type partitionInfo struct {
	rows            []int64
	partitionValues map[int]any
	partitionRec    partitionRecord // The actual partition values for generating the path
}

type partitionFieldInfo struct {
	sourceField iceberg.PartitionField
	sourceName  string
	fieldID     int
	sourceType  iceberg.Type
	columnPath  []int
	valueAt     func(arrow.Array, int) (any, error)
}

type partitionExtractionPlan struct {
	spec         iceberg.PartitionSpec
	schema       *iceberg.Schema
	recordSchema *arrow.Schema
	fields       []partitionFieldInfo
	pathPlan     partitionPathPlan
}

type (
	binaryPartitionKey  string
	float32PartitionKey uint32
	float64PartitionKey uint64
)

const (
	// Canonical NaN payloads make all NaNs of the same width compare equal.
	// The width-specific key types preserve float width, while using raw bits
	// intentionally keeps -0.0 and +0.0 distinct.
	canonicalFloat32NaNBits uint32 = 0x7fc00000
	canonicalFloat64NaNBits uint64 = 0x7ff8000000000000
)

func comparablePartitionKey(value any) any {
	switch value := value.(type) {
	case []byte:
		return binaryPartitionKey(value)
	case float32:
		rawBits := math.Float32bits(value)
		if math.IsNaN(float64(value)) {
			rawBits = canonicalFloat32NaNBits
		}

		return float32PartitionKey(rawBits)
	case float64:
		rawBits := math.Float64bits(value)
		if math.IsNaN(value) {
			rawBits = canonicalFloat64NaNBits
		}

		return float64PartitionKey(rawBits)
	}

	return value
}

func partitionValuesEqual(left, right any) bool {
	// Supported partition values are primitive literals; []byte is converted to
	// a string key and float values are converted to comparable width-aware keys
	// above. The default branch preserves comparable scalar values.
	return comparablePartitionKey(left) == comparablePartitionKey(right)
}

func partitionRecordsEqual(left, right partitionRecord) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if !partitionValuesEqual(left[i], right[i]) {
			return false
		}
	}

	return true
}

func clonePartitionValue(value any) any {
	if bytes, ok := value.([]byte); ok {
		return slices.Clone(bytes)
	}

	return value
}

// NewPartitionedFanoutWriter creates a new PartitionedFanoutWriter with the specified
// partition specification, schema, record iterator, and writerFactory.
func newPartitionedFanoutWriter(partitionSpec iceberg.PartitionSpec, schema *iceberg.Schema, itr iter.Seq2[arrow.RecordBatch, error], writerFactory *writerFactory) *partitionedFanoutWriter {
	return &partitionedFanoutWriter{
		partitionSpec: partitionSpec,
		schema:        schema,
		itr:           itr,
		writerFactory: writerFactory,
	}
}

func (p *partitionedFanoutWriter) partitionPath(data partitionRecord) string {
	return p.plan.pathPlan.format(data)
}

// Write writes the Arrow records to the specified location using a fanout pattern with
// the specified number of workers. The returned iterator yields the data files written
// by the fanout process.
func (p *partitionedFanoutWriter) Write(ctx context.Context, workers int) iter.Seq2[iceberg.DataFile, error] {
	inputRecordsCh := make(chan arrow.RecordBatch, workers)
	outputDataFilesCh := make(chan iceberg.DataFile, workers)

	fanoutBaseCtx, fanoutCancel := context.WithCancel(ctx)
	fanoutWorkers, fanoutCtx := errgroup.WithContext(fanoutBaseCtx)
	writerCtx, writerCancel := context.WithCancel(ctx)
	cancel := func() {
		fanoutCancel()
		writerCancel()
	}
	startRecordFeeder(fanoutCtx, p.itr, fanoutWorkers, inputRecordsCh)

	for range workers {
		fanoutWorkers.Go(func() error {
			return p.fanout(fanoutCtx, writerCtx, inputRecordsCh, outputDataFilesCh)
		})
	}

	return p.yieldDataFiles(fanoutWorkers, inputRecordsCh, outputDataFilesCh, cancel)
}

func startRecordFeeder(ctx context.Context, itr iter.Seq2[arrow.RecordBatch, error], fanoutWorkers *errgroup.Group, inputRecordsCh chan<- arrow.RecordBatch) {
	fanoutWorkers.Go(func() error {
		defer close(inputRecordsCh)

		for record, err := range itr {
			if err != nil {
				return err
			}

			record.Retain()
			select {
			case <-ctx.Done():
				record.Release()

				return context.Cause(ctx)
			case inputRecordsCh <- record:
			}
		}

		return nil
	})
}

func (p *partitionedFanoutWriter) fanout(ctx context.Context, writerCtx context.Context, inputRecordsCh <-chan arrow.RecordBatch, dataFilesChannel chan<- iceberg.DataFile) error {
	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)

		case record, ok := <-inputRecordsCh:
			if !ok {
				return nil
			}

			if err := p.processRecord(ctx, writerCtx, record, dataFilesChannel); err != nil {
				return err
			}
		}
	}
}

// processRecord partitions a single record batch and writes sub-batches to
// the appropriate rolling data writers. The record is released when this
// function returns, bounding Arrow memory to one batch per fanout worker.
func (p *partitionedFanoutWriter) processRecord(ctx context.Context, writerCtx context.Context, record arrow.RecordBatch, dataFilesChannel chan<- iceberg.DataFile) error {
	defer record.Release()

	partitions, err := p.getPartitions(record)
	if err != nil {
		return err
	}

	for _, val := range partitions {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		default:
		}

		partitionRecord, err := partitionBatchByKey(ctx)(record, val.rows)
		if err != nil {
			return err
		}

		partitionPath := p.partitionPath(val.partitionRec)
		rollingDataWriter, err := p.writerFactory.getOrCreateRollingDataWriter(writerCtx, partitionPath, val.partitionValues, dataFilesChannel)
		if err != nil {
			partitionRecord.Release()

			return err
		}

		addErr := rollingDataWriter.Add(partitionRecord)
		partitionRecord.Release()
		if addErr != nil {
			return addErr
		}
	}

	return nil
}

func (p *partitionedFanoutWriter) yieldDataFiles(fanoutWorkers *errgroup.Group, inputRecordsCh <-chan arrow.RecordBatch, outputDataFilesCh chan iceberg.DataFile, cancel context.CancelFunc) iter.Seq2[iceberg.DataFile, error] {
	return yieldDataFiles(
		p.writerFactory,
		fanoutWorkers,
		inputRecordsCh,
		outputDataFilesCh,
		p.writerFactory.closeAll,
		p.writerFactory.abortAll,
		cancel,
	)
}

func yieldDataFiles(
	writerFactory *writerFactory,
	fanoutWorkers *errgroup.Group,
	inputRecordsCh <-chan arrow.RecordBatch,
	outputDataFilesCh chan iceberg.DataFile,
	closeAll func() error,
	abortAll func(),
	cancel context.CancelFunc,
) iter.Seq2[iceberg.DataFile, error] {
	// Use a channel to safely communicate the error from the goroutine
	// to avoid a data race between writing err in the goroutine and reading it in the iterator.
	errCh := make(chan error, 1)
	go func() {
		defer close(outputDataFilesCh)
		defer cancel()
		err := fanoutWorkers.Wait()
		// Wait includes the feeder, which closes inputRecordsCh, so draining cannot
		// block. Any remaining batches were retained by the feeder but never dequeued;
		// workers release dequeued batches themselves.
		for record := range inputRecordsCh {
			record.Release()
		}
		if err != nil {
			abortAll()
		} else {
			err = errors.Join(err, closeAll())
		}
		errCh <- err
		close(errCh)
	}()

	return func(yield func(iceberg.DataFile, error) bool) {
		// LIFO defer order matters: cancel signals the producer first
		// (synchronous, instant), then the drain pulls outputDataFilesCh so
		// any in-flight stream send can complete and the producer's
		// closeAll / fanoutWorkers.Wait paths unblock.
		defer func() {
			for range outputDataFilesCh {
			}
		}()
		defer cancel()

		// Yield data files as they arrive - no error yet since goroutine is still running
		for f := range outputDataFilesCh {
			if !yield(f, nil) {
				return
			}
		}

		// Channel is closed, now safe to read the error
		if err := <-errCh; err != nil {
			yield(nil, err)
		}
	}
}

func (p *partitionedFanoutWriter) getPartitions(record arrow.RecordBatch) ([]*partitionInfo, error) {
	p.planOnce.Do(func() {
		p.plan, p.planErr = newPartitionExtractionPlan(p.partitionSpec, p.schema, record.Schema())
	})
	if p.planErr != nil {
		return nil, p.planErr
	}

	return p.plan.getRecordPartitions(record)
}

func getRecordPartitions(spec iceberg.PartitionSpec, schema *iceberg.Schema, record arrow.RecordBatch) ([]*partitionInfo, error) {
	plan, err := newPartitionExtractionPlan(spec, schema, record.Schema())
	if err != nil {
		return nil, err
	}

	return plan.getRecordPartitions(record)
}

func newPartitionExtractionPlan(spec iceberg.PartitionSpec, schema *iceberg.Schema, recordSchema *arrow.Schema) (*partitionExtractionPlan, error) {
	partitionFields := spec.PartitionType(schema).FieldList
	partitionFieldsInfo := make([]partitionFieldInfo, len(partitionFields))
	pathPlan := partitionPathPlan{fields: make([]partitionPathField, len(partitionFields))}
	specFieldsByID := make(map[int]iceberg.PartitionField, spec.NumFields())
	for _, field := range spec.Fields() {
		specFieldsByID[field.FieldID] = field
	}

	for i, partitionField := range partitionFields {
		sourceField, ok := specFieldsByID[partitionField.ID]
		if !ok {
			return nil, fmt.Errorf("failed to find partition field ID %d in spec", partitionField.ID)
		}
		pathField := partitionPathField{
			escapedName: sourceField.EscapedName(),
			transform:   sourceField.Transform,
			resultType:  partitionField.Type,
		}
		pathPlan.fields[i] = pathField
		pathPlan.estimatedSize += len(pathField.escapedName) + 20
		partitionFieldsInfo[i].fieldID = partitionField.ID
		colName, ok := schema.FindColumnName(sourceField.SourceID())
		if !ok {
			continue
		}
		columnPath, ok := resolveArrowColumnPath(recordSchema, colName)
		if !ok {
			return nil, fmt.Errorf("failed to find source column %q in record schema", colName)
		}
		sourceType, ok := schema.FindTypeByID(sourceField.SourceID())
		if !ok {
			return nil, fmt.Errorf("failed to find type for source field ID %d in schema", sourceField.SourceID())
		}
		partitionFieldsInfo[i] = partitionFieldInfo{
			sourceField: sourceField,
			sourceName:  colName,
			fieldID:     sourceField.FieldID,
			sourceType:  sourceType,
			columnPath:  columnPath,
			valueAt:     bindPartitionValue(sourceField.Transform, sourceType),
		}
	}

	return &partitionExtractionPlan{
		spec:         spec,
		schema:       schema,
		recordSchema: recordSchema,
		fields:       partitionFieldsInfo,
		pathPlan:     pathPlan,
	}, nil
}

// arrowFieldContainer is a proxy for both [arrow.Schema] and [arrow.StructType],
type arrowFieldContainer interface {
	FieldIndices(name string) []int
	Field(i int) arrow.Field
}

// resolveArrowColumnPath resolves a dot-joined nested field name into a path of Arrow field indices.
func resolveArrowColumnPath(recordSchema *arrow.Schema, colName string) ([]int, bool) {
	// note that a pathological schema could lead to a stack overflow as this is recursive
	return resolveArrowColumnPathIn(recordSchema, colName)
}

func resolveArrowColumnPathIn(container arrowFieldContainer, remaining string) ([]int, bool) {
	// any field name COULD contain a literal `.` so we try the longest possible name and backtrack
	for prefixEnd := len(remaining); prefixEnd > 0; prefixEnd = strings.LastIndex(remaining[:prefixEnd], ".") {
		candidate := remaining[:prefixEnd]
		indices := container.FieldIndices(candidate)
		if indices == nil {
			continue
		}

		if prefixEnd == len(remaining) {
			return []int{indices[0]}, true
		}

		structType, ok := container.Field(indices[0]).Type.(*arrow.StructType)
		if !ok {
			continue
		}

		if subPath, ok := resolveArrowColumnPathIn(structType, remaining[prefixEnd+1:]); ok {
			return append([]int{indices[0]}, subPath...), true
		}
	}

	return nil, false
}

// resolveColumnChain walks a resolved column path from the record's top-level column
// down to the leaf, returning the array at each step of the path.
func resolveColumnChain(record arrow.RecordBatch, path []int) ([]arrow.Array, error) {
	chain := make([]arrow.Array, len(path))
	col := record.Column(path[0])
	chain[0] = col
	for i, fieldIdx := range path[1:] {
		structCol, ok := col.(*array.Struct)
		if !ok {
			return nil, fmt.Errorf("expected struct array in column path, got %T", col)
		}
		col = structCol.Field(fieldIdx)
		chain[i+1] = col
	}

	return chain, nil
}

// leafColumnAt returns the leaf array for a resolved column chain at the given row,
// or false if the leaf or any ancestor struct is null at that row.
func leafColumnAt(chain []arrow.Array, row int) (arrow.Array, bool) {
	for _, col := range chain[:len(chain)-1] {
		if col.IsNull(row) {
			return nil, false
		}
	}

	leaf := chain[len(chain)-1]
	if leaf.IsNull(row) {
		return nil, false
	}

	return leaf, true
}

func (p *partitionExtractionPlan) getRecordPartitions(record arrow.RecordBatch) ([]*partitionInfo, error) {
	// Preserve support for iterators whose batch schema changes. The usual path compares
	// schema pointers; equivalent independently-built schemas also reuse the plan.
	if !p.matchesSchema(record.Schema()) {
		plan, err := newPartitionExtractionPlan(p.spec, p.schema, record.Schema())
		if err != nil {
			return nil, err
		}

		return plan.getRecordPartitions(record)
	}

	partitionMap := newPartitionMapNode()
	partitionRec := make(partitionRecord, len(p.fields))

	// we track these separately to avoid traversing the chain for top level fields.
	topLevelColumns := make([]arrow.Array, len(p.fields))
	columnChains := make([][]arrow.Array, len(p.fields))
	for i, fieldInfo := range p.fields {
		switch len(fieldInfo.columnPath) {
		case 0: // an empty (nil) path means the source field isn't in the schema
		case 1:
			topLevelColumns[i] = record.Column(fieldInfo.columnPath[0])
		default:
			chain, err := resolveColumnChain(record, fieldInfo.columnPath)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve source column %q (field ID %d): %w",
					fieldInfo.sourceName, fieldInfo.sourceField.SourceID(), err)
			}
			columnChains[i] = chain
		}
	}

	for row := range record.NumRows() {
		for i, fieldInfo := range p.fields {
			var col arrow.Array
			var ok bool

			switch {
			case topLevelColumns[i] != nil:
				col = topLevelColumns[i]
				ok = !col.IsNull(int(row))
			case columnChains[i] != nil:
				col, ok = leafColumnAt(columnChains[i], int(row))
			default:
				partitionRec[i] = nil

				continue
			}

			if ok {
				value, err := fieldInfo.valueAt(col, int(row))
				if err != nil {
					return nil, fmt.Errorf(
						"failed to convert source column %q (field ID %d) from Arrow type %s to Iceberg type %s: %w",
						fieldInfo.sourceName,
						fieldInfo.sourceField.SourceID(),
						col.DataType(),
						fieldInfo.sourceType,
						err,
					)
				}

				partitionRec[i] = value
			} else {
				partitionRec[i] = nil
			}
		}

		// Get or create partition info for this partition key
		partVal := partitionMap.getOrCreate(partitionRec, p.fields, record.NumRows())
		partVal.rows = append(partVal.rows, row)
	}

	return partitionMap.collectPartitions(), nil
}

func (p *partitionExtractionPlan) matchesSchema(recordSchema *arrow.Schema) bool {
	return recordSchema == p.recordSchema || recordSchema.Equal(p.recordSchema)
}

func bindPartitionValue(transform iceberg.Transform, sourceType iceberg.Type) func(arrow.Array, int) (any, error) {
	bound, ok := bindPartitionTransform(transform, sourceType)
	if !ok {
		return func(column arrow.Array, row int) (any, error) {
			value, err := getArrowValueAsIcebergLiteral(column, row, sourceType)
			if err != nil {
				return nil, err
			}
			if value == nil {
				return nil, nil
			}

			transformed := transform.Apply(iceberg.Optional[iceberg.Literal]{Valid: true, Val: value})
			if !transformed.Valid {
				return nil, nil
			}

			return transformed.Val.Any(), nil
		}
	}

	return func(column arrow.Array, row int) (any, error) {
		value, err := getArrowValueAsIcebergValue(column, row, sourceType)
		if err != nil {
			return nil, err
		}
		if value == nil {
			return nil, nil
		}

		return bound(value), nil
	}
}

func bindPartitionTransform(transform iceberg.Transform, sourceType iceberg.Type) (func(any) any, bool) {
	optionalInt := func(transformer func(any) iceberg.Optional[int32]) func(any) any {
		return func(value any) any {
			transformed := transformer(value)
			if !transformed.Valid {
				return nil
			}

			return transformed.Val
		}
	}
	optionalDate := func(transformer func(any) iceberg.Optional[int32]) func(any) any {
		return func(value any) any {
			transformed := transformer(value)
			if !transformed.Valid {
				return nil
			}

			return iceberg.Date(transformed.Val)
		}
	}

	switch typed := transform.(type) {
	case iceberg.IdentityTransform:
		return func(value any) any { return value }, true
	case iceberg.VoidTransform:
		return func(any) any { return nil }, true
	case iceberg.BucketTransform:
		if !typed.CanTransform(sourceType) {
			break
		}

		return optionalInt(typed.Transformer(sourceType)), true
	case iceberg.TruncateTransform:
		transformer, err := typed.Transformer(sourceType)
		if err == nil {
			return transformer, true
		}
	case iceberg.UnknownTransform:
		return func(any) any { return nil }, true
	}

	// Year, month, and hour transforms bind through TimeTransform.
	if typed, ok := transform.(iceberg.TimeTransform); ok {
		transformer, err := typed.Transformer(sourceType)
		if err == nil {
			if _, ok := transform.(iceberg.DayTransform); ok {
				return optionalDate(transformer), true
			}

			return optionalInt(transformer), true
		}
	}

	return nil, false
}

// partitionMapNode represents a simple tree structure for storing partitionInfo.
//
// Each key is the partition value at that level of the tree, and the key hierarchy
// is in the order of the partition spec.
// The value is either a *partitionMapNode or a *partitionInfo.
type partitionMapNode struct {
	children map[any]any
	// partitionCount is maintained by the root node for the current batch and
	// sizes the single result slice returned by collectPartitions.
	partitionCount int
}

func newPartitionMapNode() *partitionMapNode {
	return &partitionMapNode{
		children: make(map[any]any),
	}
}

// getOrCreate navigates the tree and returns the partitionInfo for the given partition key,
// creating nodes along the way if they don't exist
func (n *partitionMapNode) getOrCreate(partitionRec partitionRecord, fieldInfo []partitionFieldInfo, numRows int64) *partitionInfo {
	// Navigate through all but the last partition field
	node := n
	for _, part := range partitionRec[:len(partitionRec)-1] {
		key := comparablePartitionKey(part)
		val, ok := node.children[key]
		if !ok {
			newNode := newPartitionMapNode()
			node.children[key] = newNode
			node = newNode
		} else {
			node = val.(*partitionMapNode)
		}
	}

	// Last level stores the actual partitionInfo
	lastKey := comparablePartitionKey(partitionRec[len(partitionRec)-1])
	partVal, ok := node.children[lastKey].(*partitionInfo)
	if ok {
		return partVal
	}

	// First time seeing this partition - create partitionValues map
	partitionValues := make(map[int]any, len(partitionRec))

	// Copy partitionRec values so they don't get overwritten
	partRecCopy := make(partitionRecord, len(partitionRec))
	for i := range partitionRec {
		value := clonePartitionValue(partitionRec[i])
		partitionValues[fieldInfo[i].fieldID] = value
		partRecCopy[i] = value
	}

	partVal = &partitionInfo{
		rows:            make([]int64, 0, initialPartitionRowCapacity(numRows, n.partitionCount)),
		partitionValues: partitionValues,
		partitionRec:    partRecCopy,
	}
	node.children[lastKey] = partVal
	n.partitionCount++

	return partVal
}

const maxInitialPartitionRowCapacity = 128

func initialPartitionRowCapacity(numRows int64, partitionCount int) int {
	if numRows <= 0 || partitionCount < 0 || int64(partitionCount) >= numRows {
		return 1
	}

	estimatedRows := numRows / int64(partitionCount+1)
	if estimatedRows < 1 {
		return 1
	}
	if estimatedRows > maxInitialPartitionRowCapacity {
		return maxInitialPartitionRowCapacity
	}

	// Use power-of-two capacities so a late-discovered partition grows through
	// the old 128-row allocation instead of jumping past it from a capacity
	// such as 127.
	return 1 << bits.Len64(uint64(estimatedRows-1))
}

// collectPartitions returns every partitionInfo in the tree in
// arbitrary order. Callers that need a deterministic order (such as
// the clustered writer, whose revisit check would otherwise depend on
// Go's randomized map iteration) must sort the result themselves.
func (n *partitionMapNode) collectPartitions() []*partitionInfo {
	result := make([]*partitionInfo, 0, n.partitionCount)

	return n.appendPartitions(result)
}

func (n *partitionMapNode) appendPartitions(result []*partitionInfo) []*partitionInfo {
	for _, v := range n.children {
		switch node := v.(type) {
		case *partitionInfo:
			result = append(result, node)
		case *partitionMapNode:
			result = node.appendPartitions(result)
		}
	}

	return result
}

type partitionBatchFn func(arrow.RecordBatch, []int64) (arrow.RecordBatch, error)

func partitionBatchByKey(ctx context.Context) partitionBatchFn {
	mem := compute.GetAllocator(ctx)

	return func(record arrow.RecordBatch, rowIndices []int64) (arrow.RecordBatch, error) {
		if len(rowIndices) == 0 && record.NumRows() == 0 {
			record.Retain()

			return record, nil
		}

		if start, end, ok := contiguousRowRange(rowIndices, record.NumRows()); ok {
			if start == 0 && end == record.NumRows() {
				record.Retain()

				return record, nil
			}

			if contiguousSliceHasBoundedRetention(start, end, record.NumRows()) && recordHasRowBoundedStorage(record) {
				return record.NewSlice(start, end), nil
			}
		}

		bldr := array.NewInt64Builder(mem)
		defer bldr.Release()

		bldr.AppendValues(rowIndices, nil)
		rowIndicesArr := bldr.NewInt64Array()
		defer rowIndicesArr.Release()

		recordMetadata := arrow.Metadata{}
		if recordWithMetadata, ok := record.(arrow.RecordBatchWithMetadata); ok {
			recordMetadata = recordWithMetadata.Metadata()
		}

		partitionedRecord, err := compute.Take(
			ctx,
			*compute.DefaultTakeOptions(),
			compute.NewDatumWithoutOwning(record),
			compute.NewDatumWithoutOwning(rowIndicesArr),
		)
		if err != nil {
			return nil, err
		}

		return materializeDictionaryColumns(ctx, partitionedRecord.(*compute.RecordDatum).Value, recordMetadata)
	}
}

// materializeDictionaryColumns removes dictionary values that are not referenced
// by a partial result. Arrow's Take kernel copies dictionary indices but reuses
// the complete dictionary, which can otherwise retain a large variable-width
// buffer in a rolling writer queue.
func materializeDictionaryColumns(ctx context.Context, record arrow.RecordBatch, recordMetadata arrow.Metadata) (arrow.RecordBatch, error) {
	var columns []arrow.Array
	var fields []arrow.Field
	var materialized []arrow.Array

	for i, column := range record.Columns() {
		values, changed, err := materializeDictionaryArray(ctx, column)
		if err != nil {
			for _, materializedColumn := range materialized {
				materializedColumn.Release()
			}
			record.Release()

			return nil, err
		}
		if !changed {
			continue
		}

		if columns == nil {
			columns = slices.Clone(record.Columns())
			fields = record.Schema().Fields()
			materialized = make([]arrow.Array, 0, 1)
		}

		columns[i] = values
		fields[i].Type = values.DataType()
		materialized = append(materialized, values)
	}

	if columns == nil {
		return record, nil
	}

	metadata := record.Schema().Metadata()
	result := array.NewRecordBatchWithMetadata(arrow.NewSchema(fields, &metadata), columns, record.NumRows(), recordMetadata)
	for _, materializedColumn := range materialized {
		materializedColumn.Release()
	}
	record.Release()

	return result, nil
}

// materializeDictionaryArray removes dictionaries recursively from the nested
// array types used by Iceberg schemas. A dictionary array is decoded by taking
// its selected logical values, while nested arrays are rebuilt only when one of
// their children changes.
func materializeDictionaryArray(ctx context.Context, input arrow.Array) (arrow.Array, bool, error) {
	if input.DataType().ID() == arrow.DICTIONARY {
		dictionary := input.(*array.Dictionary)
		values, err := compute.TakeArray(ctx, dictionary.Dictionary(), dictionary.Indices())
		if err != nil {
			return nil, false, err
		}

		materializedValues, _, err := materializeDictionaryArray(ctx, values)
		if err != nil {
			values.Release()

			return nil, false, err
		}
		if materializedValues != values {
			values.Release()
			values = materializedValues
		}

		return values, true, nil
	}

	dataType := input.DataType()
	switch dataType.ID() {
	case arrow.STRUCT, arrow.LIST, arrow.LARGE_LIST, arrow.FIXED_SIZE_LIST, arrow.MAP:
	default:
		return input, false, nil
	}

	data := input.Data()
	children := make([]arrow.Array, len(data.Children()))
	changed := false
	for i, childData := range data.Children() {
		child := array.MakeFromData(childData)
		materializedChild, childChanged, err := materializeDictionaryArray(ctx, child)
		if err != nil {
			child.Release()
			for _, materializedChild := range children {
				if materializedChild != nil {
					materializedChild.Release()
				}
			}

			return nil, false, err
		}

		if childChanged {
			child.Release()
			child = materializedChild
			changed = true
		}
		children[i] = child
	}

	if !changed {
		for _, child := range children {
			child.Release()
		}

		return input, false, nil
	}

	newType, err := nestedArrayTypeWithChildren(dataType, children)
	if err != nil {
		for _, child := range children {
			child.Release()
		}

		return nil, false, err
	}

	childData := make([]arrow.ArrayData, len(children))
	for i, child := range children {
		childData[i] = child.Data()
	}
	newData := array.NewData(newType, data.Len(), data.Buffers(), childData, data.NullN(), data.Offset())
	result := array.MakeFromData(newData)
	newData.Release()
	for _, child := range children {
		child.Release()
	}

	return result, true, nil
}

func nestedArrayTypeWithChildren(dataType arrow.DataType, children []arrow.Array) (arrow.DataType, error) {
	switch dataType := dataType.(type) {
	case *arrow.StructType:
		fields := dataType.Fields()
		if len(fields) != len(children) {
			return nil, fmt.Errorf("struct has %d fields but %d children", len(fields), len(children))
		}
		for i, child := range children {
			fields[i].Type = child.DataType()
		}

		return arrow.StructOf(fields...), nil
	case *arrow.ListType:
		if len(children) != 1 {
			return nil, fmt.Errorf("list has %d children", len(children))
		}
		field := dataType.ElemField()
		field.Type = children[0].DataType()

		return arrow.ListOfField(field), nil
	case *arrow.LargeListType:
		if len(children) != 1 {
			return nil, fmt.Errorf("large list has %d children", len(children))
		}
		field := dataType.ElemField()
		field.Type = children[0].DataType()

		return arrow.LargeListOfField(field), nil
	case *arrow.FixedSizeListType:
		if len(children) != 1 {
			return nil, fmt.Errorf("fixed-size list has %d children", len(children))
		}
		field := dataType.ElemField()
		field.Type = children[0].DataType()

		return arrow.FixedSizeListOfField(dataType.Len(), field), nil
	case *arrow.MapType:
		if len(children) != 1 {
			return nil, fmt.Errorf("map has %d children", len(children))
		}
		entryType, ok := children[0].DataType().(*arrow.StructType)
		if !ok || entryType.NumFields() != 2 {
			return nil, fmt.Errorf("map child has type %s, want a two-field struct", children[0].DataType())
		}

		result := arrow.MapOfFields(entryType.Field(0), entryType.Field(1))
		result.KeysSorted = dataType.KeysSorted

		return result, nil
	default:
		return nil, fmt.Errorf("unsupported nested array type %s", dataType)
	}
}

// recordHasRowBoundedStorage reports whether a partial zero-copy slice retains
// buffers whose size is bounded by the number of rows. Fixed-width arrays have
// row-bounded value and validity buffers. Dictionary arrays are excluded even
// though their indices are fixed-width because their dictionary values are not.
func recordHasRowBoundedStorage(record arrow.RecordBatch) bool {
	for _, column := range record.Columns() {
		dataType := column.Data().DataType()
		if dataType.ID() == arrow.DICTIONARY {
			return false
		}
		if _, ok := dataType.(arrow.FixedWidthDataType); !ok {
			return false
		}
	}

	return true
}

// contiguousSliceHasBoundedRetention reports whether a partial zero-copy slice
// retains no more than twice as many input rows as it returns. This bound is
// meaningful only for records whose storage is row-bounded.
func contiguousSliceHasBoundedRetention(start, end, numRows int64) bool {
	selectedRows := end - start

	return selectedRows >= numRows-selectedRows
}

func contiguousRowRange(rowIndices []int64, numRows int64) (start, end int64, ok bool) {
	if len(rowIndices) == 0 {
		return 0, 0, false
	}

	start = rowIndices[0]
	length := int64(len(rowIndices))
	if start < 0 || length > numRows || start > numRows-length {
		return 0, 0, false
	}

	for offset, row := range rowIndices[1:] {
		if row != start+int64(offset)+1 {
			return 0, 0, false
		}
	}

	return start, start + length, true
}

func getArrowValueAsIcebergLiteral(column arrow.Array, row int, sourceType iceberg.Type) (iceberg.Literal, error) {
	value, err := getArrowValueAsIcebergValue(column, row, sourceType)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, nil
	}

	literal, err := partitionLiteralFromValue(value)
	if err != nil {
		return nil, err
	}
	switch sourceType.(type) {
	case iceberg.BinaryType, iceberg.FixedType, iceberg.UUIDType:
		return literal.To(sourceType)
	default:
		return literal, nil
	}
}

func partitionLiteralFromValue(value any) (iceberg.Literal, error) {
	switch value := value.(type) {
	case bool:
		return iceberg.NewLiteral(value), nil
	case int32:
		return iceberg.NewLiteral(value), nil
	case int64:
		return iceberg.NewLiteral(value), nil
	case float32:
		return iceberg.NewLiteral(value), nil
	case float64:
		return iceberg.NewLiteral(value), nil
	case iceberg.Date:
		return iceberg.NewLiteral(value), nil
	case iceberg.Time:
		return iceberg.NewLiteral(value), nil
	case iceberg.Timestamp:
		return iceberg.NewLiteral(value), nil
	case iceberg.TimestampNano:
		return iceberg.NewLiteral(value), nil
	case string:
		return iceberg.NewLiteral(value), nil
	case []byte:
		return iceberg.NewLiteral(value), nil
	case uuid.UUID:
		return iceberg.NewLiteral(value), nil
	case iceberg.Decimal:
		return iceberg.NewLiteral(value), nil
	default:
		return nil, fmt.Errorf("unsupported Iceberg literal value type: %T", value)
	}
}

func getArrowValueAsIcebergValue(column arrow.Array, row int, sourceType iceberg.Type) (any, error) {
	if column.IsNull(row) {
		return nil, nil
	}

	switch arr := column.(type) {
	case *array.Date32:

		return iceberg.Date(arr.Value(row)), nil
	case *array.Time64:
		dt, ok := arr.DataType().(*arrow.Time64Type)
		if !ok || dt.Unit != arrow.Microsecond {
			return nil, fmt.Errorf("%w: unsupported arrow type for conversion - %s", iceberg.ErrInvalidSchema, arr.DataType())
		}

		return iceberg.Time(arr.Value(row)), nil
	case *array.Timestamp:

		return timestampValueFromArrow(arr, row, sourceType)
	case *array.Decimal32:
		val := arr.Value(row)
		dec := iceberg.Decimal{
			Val:   decimal128.FromI64(int64(val)),
			Scale: int(arr.DataType().(*arrow.Decimal32Type).Scale),
		}

		return dec, nil
	case *array.Decimal64:
		val := arr.Value(row)
		dec := iceberg.Decimal{
			Val:   decimal128.FromI64(int64(val)),
			Scale: int(arr.DataType().(*arrow.Decimal64Type).Scale),
		}

		return dec, nil
	case *array.Decimal128:
		val := arr.Value(row)
		dec := iceberg.Decimal{
			Val:   val,
			Scale: int(arr.DataType().(*arrow.Decimal128Type).Scale),
		}

		return dec, nil
	case *extensions.UUIDArray:

		return arr.Value(row), nil

	case *array.String:

		return arr.Value(row), nil
	case *array.LargeString:

		return arr.Value(row), nil
	case *array.Int64:

		return arr.Value(row), nil
	case *array.Int32:

		return arr.Value(row), nil
	case *array.Int16:

		return int32(arr.Value(row)), nil
	case *array.Int8:

		return int32(arr.Value(row)), nil
	case *array.Uint64:

		return int64(arr.Value(row)), nil
	case *array.Uint32:

		return int32(arr.Value(row)), nil
	case *array.Uint16:

		return int32(arr.Value(row)), nil
	case *array.Uint8:

		return int32(arr.Value(row)), nil
	case *array.Float32:

		return arr.Value(row), nil
	case *array.Float64:

		return arr.Value(row), nil
	case *array.Boolean:

		return arr.Value(row), nil
	case *array.Binary:

		return arr.Value(row), nil
	case *array.LargeBinary:

		return arr.Value(row), nil
	case *array.FixedSizeBinary:
		switch sourceType.(type) {
		case iceberg.BinaryType, iceberg.FixedType, iceberg.UUIDType:
		default:
			return nil, fmt.Errorf("%w: cannot convert Arrow %s to Iceberg type %v", iceberg.ErrInvalidSchema, arr.DataType(), sourceType)
		}

		literal, err := iceberg.NewLiteral(arr.Value(row)).To(sourceType)
		if err != nil {
			return nil, err
		}

		return literal.Any(), nil

	default:
		val := column.GetOneForMarshal(row)

		return nil, fmt.Errorf("unsupported value type: %T", val)
	}
}

func timestampValueFromArrow(arr *array.Timestamp, row int, sourceType iceberg.Type) (any, error) {
	timestampType := arr.DataType().(*arrow.TimestampType)
	value := int64(arr.Value(row))

	switch sourceType.(type) {
	case iceberg.TimestampType, iceberg.TimestampTzType:
		micros, err := arrowTimestampToMicros(value, timestampType.Unit)
		if err != nil {
			return nil, err
		}

		return iceberg.Timestamp(micros), nil
	case iceberg.TimestampNsType, iceberg.TimestampTzNsType:
		nanos, err := arrowTimestampToNanos(value, timestampType.Unit)
		if err != nil {
			return nil, err
		}

		return iceberg.TimestampNano(nanos), nil
	default:
		return nil, fmt.Errorf("cannot convert arrow timestamp to iceberg literal for source type %v", sourceType)
	}
}

func arrowTimestampToMicros(value int64, unit arrow.TimeUnit) (int64, error) {
	switch unit {
	case arrow.Second:
		return scaleTimestamp(value, 1_000_000)
	case arrow.Millisecond:
		return scaleTimestamp(value, 1_000)
	case arrow.Microsecond:
		return value, nil
	case arrow.Nanosecond:
		return internal.FloorDiv(value, int64(1_000)), nil
	default:
		return 0, fmt.Errorf("unsupported arrow timestamp unit: %s", unit)
	}
}

func arrowTimestampToNanos(value int64, unit arrow.TimeUnit) (int64, error) {
	switch unit {
	case arrow.Second:
		return scaleTimestamp(value, 1_000_000_000)
	case arrow.Millisecond:
		return scaleTimestamp(value, 1_000_000)
	case arrow.Microsecond:
		return scaleTimestamp(value, 1_000)
	case arrow.Nanosecond:
		return value, nil
	default:
		return 0, fmt.Errorf("unsupported arrow timestamp unit: %s", unit)
	}
}

func scaleTimestamp(value, factor int64) (int64, error) {
	if (value > 0 && value > math.MaxInt64/factor) ||
		(value < 0 && value < math.MinInt64/factor) {
		return 0, fmt.Errorf("arrow timestamp value %d overflows int64 when scaled by %d", value, factor)
	}

	return value * factor, nil
}
