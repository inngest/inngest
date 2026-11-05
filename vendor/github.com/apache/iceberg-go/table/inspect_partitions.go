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
	"slices"
	"sort"
	"strings"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/scalar"
	"github.com/apache/iceberg-go"
)

type inspectPartitionAggregate struct {
	specID              int32
	partition           map[int]any
	dataRecordCount     int64
	dataFileCount       int32
	dataFileSize        int64
	positionDeleteCount int64
	positionDeleteFiles int32
	equalityDeleteCount int64
	equalityDeleteFiles int32
	lastUpdatedAt       *int64
	lastUpdatedSnapshot *int64
	orderingKey         string
}

type inspectPartitionAggregateTree struct {
	children  map[any]*inspectPartitionAggregateTree
	aggregate *inspectPartitionAggregate
}

func newInspectPartitionAggregateTree() *inspectPartitionAggregateTree {
	return &inspectPartitionAggregateTree{
		children: make(map[any]*inspectPartitionAggregateTree),
	}
}

func (t *inspectPartitionAggregateTree) lookup(record partitionRecord) *inspectPartitionAggregate {
	node := t
	for _, value := range record {
		child, ok := node.children[comparablePartitionKey(value)]
		if !ok {
			return nil
		}
		node = child
	}

	return node.aggregate
}

func (t *inspectPartitionAggregateTree) lookupPartition(
	partition map[int]any,
	partitionType *iceberg.StructType,
) *inspectPartitionAggregate {
	node := t
	for _, field := range partitionType.FieldList {
		child, ok := node.children[comparablePartitionKey(partition[field.ID])]
		if !ok {
			return nil
		}
		node = child
	}

	return node.aggregate
}

// insert walks the positional record, while lookupPartition walks partition
// field IDs. Build records with the same partition type so their field order
// stays aligned.
func (t *inspectPartitionAggregateTree) insert(record partitionRecord, aggregate *inspectPartitionAggregate) {
	node := t
	for _, value := range record {
		key := comparablePartitionKey(value)
		child, ok := node.children[key]
		if !ok {
			child = newInspectPartitionAggregateTree()
			node.children[key] = child
		}
		node = child
	}
	node.aggregate = aggregate
}

// Partitions returns one row per live partition, aggregating data and delete
// files from the current snapshot. Files from evolved specs are coerced into
// the table-wide partition type before grouping.
func (i InspectTable) Partitions(ctx context.Context) (array.RecordReader, error) {
	partitionType, err := inspectPartitionType(i.tbl.metadata)
	if err != nil {
		return nil, fmt.Errorf("inspect partitions: %w", err)
	}
	schema := PartitionsSchema(partitionType)
	arrowSchema, err := SchemaToArrowSchema(schema, nil, true, false)
	if err != nil {
		return nil, fmt.Errorf("inspect partitions: build arrow schema: %w", err)
	}

	aggregates, err := i.partitionAggregates(ctx, partitionType)
	if err != nil {
		return nil, fmt.Errorf("inspect partitions: %w", err)
	}

	bldr := array.NewRecordBuilder(i.alloc, arrowSchema)
	defer bldr.Release()
	for _, aggregate := range aggregates {
		if err := appendPartitionAggregate(bldr, partitionType, aggregate); err != nil {
			return nil, fmt.Errorf("inspect partitions: append partition: %w", err)
		}
	}

	rr, err := singleBatchReader(arrowSchema, bldr)
	if err != nil {
		return nil, fmt.Errorf("inspect partitions: %w", err)
	}

	return rr, nil
}

func (i InspectTable) partitionAggregates(ctx context.Context, partitionType *iceberg.StructType) ([]inspectPartitionAggregate, error) {
	snapshot := i.tbl.metadata.CurrentSnapshot()
	if snapshot == nil {
		return nil, nil
	}
	if i.tbl.fsF == nil {
		return nil, errors.New("table file IO is not configured")
	}
	manifestSet, err := i.tbl.manifestSet(ctx, *snapshot)
	if err != nil {
		return nil, err
	}
	fs, err := i.tbl.fsF(ctx)
	if err != nil {
		return nil, err
	}
	manifests := manifestSet.allManifests()

	snapshotTimes := make(map[int64]int64, len(i.tbl.metadata.Snapshots()))
	for _, snapshot := range i.tbl.metadata.Snapshots() {
		snapshotTimes[snapshot.SnapshotID] = snapshot.TimestampMs
	}

	aggregates := make([]*inspectPartitionAggregate, 0)
	aggregateTree := newInspectPartitionAggregateTree()
	for _, manifest := range manifests {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for entry, err := range manifest.Entries(fs, true) {
			if err != nil {
				return nil, err
			}
			file := entry.DataFile()
			partitionValues := dataFilePartition(file)
			aggregate := aggregateTree.lookupPartition(partitionValues, partitionType)
			if aggregate == nil {
				partition := inspectCoercePartition(partitionValues, partitionType)
				record := newPartitionRecord(partition, partitionType)
				aggregate = &inspectPartitionAggregate{
					partition:   cloneInspectPartition(partition),
					orderingKey: inspectPartitionKey(partition),
				}
				aggregates = append(aggregates, aggregate)
				aggregateTree.insert(record, aggregate)
			}

			switch file.ContentType() {
			case iceberg.EntryContentData:
				aggregate.dataRecordCount += file.Count()
				aggregate.dataFileCount++
				aggregate.dataFileSize += file.FileSizeBytes()
			case iceberg.EntryContentPosDeletes:
				aggregate.positionDeleteCount += file.Count()
				aggregate.positionDeleteFiles++
			case iceberg.EntryContentEqDeletes:
				aggregate.equalityDeleteCount += file.Count()
				aggregate.equalityDeleteFiles++
			}

			if file.ContentType() == iceberg.EntryContentData ||
				file.ContentType() == iceberg.EntryContentPosDeletes ||
				file.ContentType() == iceberg.EntryContentEqDeletes {
				i.updatePartitionTimestamp(aggregate, snapshotTimes, entry.SnapshotID(), file.SpecID())
			}
		}
	}

	result := make([]inspectPartitionAggregate, 0, len(aggregates))
	for _, aggregate := range aggregates {
		result = append(result, *aggregate)
	}
	sort.Slice(result, func(left, right int) bool { return result[left].orderingKey < result[right].orderingKey })

	return result, nil
}

func (i InspectTable) updatePartitionTimestamp(
	aggregate *inspectPartitionAggregate,
	snapshotTimes map[int64]int64,
	snapshotID int64,
	specID int32,
) {
	if snapshotID < 0 {
		return
	}
	timestampMs, ok := snapshotTimes[snapshotID]
	if !ok {
		return
	}
	timestamp := timestampMs * 1000
	if aggregate.lastUpdatedAt != nil && timestamp <= *aggregate.lastUpdatedAt {
		return
	}
	aggregate.lastUpdatedAt = &timestamp
	id := snapshotID
	aggregate.lastUpdatedSnapshot = &id
	aggregate.specID = specID
}

func inspectCoercePartition(values map[int]any, partitionType *iceberg.StructType) map[int]any {
	if partitionType == nil || len(partitionType.FieldList) == 0 {
		return nil
	}

	coerced := make(map[int]any, len(partitionType.FieldList))
	for _, field := range partitionType.FieldList {
		if value, ok := values[field.ID]; ok {
			coerced[field.ID] = value
		}
	}

	return coerced
}

func inspectPartitionKey(partition map[int]any) string {
	ids := make([]int, 0, len(partition))
	for id := range partition {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	var key strings.Builder
	for _, id := range ids {
		fmt.Fprintf(&key, "%d:%T:%v;", id, partition[id], partition[id])
	}

	return key.String()
}

func cloneInspectPartition(partition map[int]any) map[int]any {
	if partition == nil {
		return nil
	}
	returnMap := make(map[int]any, len(partition))
	for id, value := range partition {
		if bytes, ok := value.([]byte); ok {
			value = slices.Clone(bytes)
		}
		returnMap[id] = value
	}

	return returnMap
}

// PartitionsSchema returns the partition metadata-table schema. For an
// unpartitioned table the partition and spec_id columns are omitted, matching
// Java's PartitionsTable behavior.
func PartitionsSchema(partitionType *iceberg.StructType) *iceberg.Schema {
	if partitionType == nil || len(partitionType.FieldList) == 0 {
		return iceberg.NewSchema(0,
			iceberg.NestedField{ID: 2, Name: "record_count", Type: iceberg.PrimitiveTypes.Int64, Required: true},
			iceberg.NestedField{ID: 3, Name: "file_count", Type: iceberg.PrimitiveTypes.Int32, Required: true},
			iceberg.NestedField{ID: 11, Name: "total_data_file_size_in_bytes", Type: iceberg.PrimitiveTypes.Int64, Required: true},
			iceberg.NestedField{ID: 5, Name: "position_delete_record_count", Type: iceberg.PrimitiveTypes.Int64, Required: true},
			iceberg.NestedField{ID: 6, Name: "position_delete_file_count", Type: iceberg.PrimitiveTypes.Int32, Required: true},
			iceberg.NestedField{ID: 7, Name: "equality_delete_record_count", Type: iceberg.PrimitiveTypes.Int64, Required: true},
			iceberg.NestedField{ID: 8, Name: "equality_delete_file_count", Type: iceberg.PrimitiveTypes.Int32, Required: true},
			iceberg.NestedField{ID: 9, Name: "last_updated_at", Type: iceberg.PrimitiveTypes.TimestampTz, Required: false},
			iceberg.NestedField{ID: 10, Name: "last_updated_snapshot_id", Type: iceberg.PrimitiveTypes.Int64, Required: false},
		)
	}

	return iceberg.NewSchema(0,
		iceberg.NestedField{ID: 1, Name: "partition", Type: partitionType, Required: true},
		iceberg.NestedField{ID: 4, Name: "spec_id", Type: iceberg.PrimitiveTypes.Int32, Required: true},
		iceberg.NestedField{ID: 2, Name: "record_count", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		iceberg.NestedField{ID: 3, Name: "file_count", Type: iceberg.PrimitiveTypes.Int32, Required: true},
		iceberg.NestedField{ID: 11, Name: "total_data_file_size_in_bytes", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		iceberg.NestedField{ID: 5, Name: "position_delete_record_count", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		iceberg.NestedField{ID: 6, Name: "position_delete_file_count", Type: iceberg.PrimitiveTypes.Int32, Required: true},
		iceberg.NestedField{ID: 7, Name: "equality_delete_record_count", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		iceberg.NestedField{ID: 8, Name: "equality_delete_file_count", Type: iceberg.PrimitiveTypes.Int32, Required: true},
		iceberg.NestedField{ID: 9, Name: "last_updated_at", Type: iceberg.PrimitiveTypes.TimestampTz, Required: false},
		iceberg.NestedField{ID: 10, Name: "last_updated_snapshot_id", Type: iceberg.PrimitiveTypes.Int64, Required: false},
	)
}

func appendPartitionAggregate(bldr *array.RecordBuilder, partitionType *iceberg.StructType, aggregate inspectPartitionAggregate) error {
	idx := 0
	if partitionType != nil && len(partitionType.FieldList) > 0 {
		partition := bldr.Field(idx).(*array.StructBuilder)
		arrowType := partition.Type().(*arrow.StructType)
		partition.Append(true)
		for fieldIndex, field := range partitionType.FieldList {
			value := aggregate.partition[field.ID]
			if value == nil {
				partition.FieldBuilder(fieldIndex).AppendNull()

				continue
			}
			valueScalar, err := inspectValueScalar(value, field.Type, arrowType.Field(fieldIndex).Type)
			if err != nil {
				return fmt.Errorf("partition field %q: %w", field.Name, err)
			}
			if err := scalar.Append(partition.FieldBuilder(fieldIndex), valueScalar); err != nil {
				return err
			}
		}
		idx++
		bldr.Field(idx).(*array.Int32Builder).Append(aggregate.specID)
		idx++
	}
	bldr.Field(idx).(*array.Int64Builder).Append(aggregate.dataRecordCount)
	idx++
	bldr.Field(idx).(*array.Int32Builder).Append(aggregate.dataFileCount)
	idx++
	bldr.Field(idx).(*array.Int64Builder).Append(aggregate.dataFileSize)
	idx++
	bldr.Field(idx).(*array.Int64Builder).Append(aggregate.positionDeleteCount)
	idx++
	bldr.Field(idx).(*array.Int32Builder).Append(aggregate.positionDeleteFiles)
	idx++
	bldr.Field(idx).(*array.Int64Builder).Append(aggregate.equalityDeleteCount)
	idx++
	bldr.Field(idx).(*array.Int32Builder).Append(aggregate.equalityDeleteFiles)
	idx++
	if aggregate.lastUpdatedAt == nil {
		bldr.Field(idx).(*array.TimestampBuilder).AppendNull()
	} else {
		bldr.Field(idx).(*array.TimestampBuilder).Append(arrow.Timestamp(*aggregate.lastUpdatedAt))
	}
	idx++
	if aggregate.lastUpdatedSnapshot == nil {
		bldr.Field(idx).(*array.Int64Builder).AppendNull()
	} else {
		bldr.Field(idx).(*array.Int64Builder).Append(*aggregate.lastUpdatedSnapshot)
	}

	return nil
}
