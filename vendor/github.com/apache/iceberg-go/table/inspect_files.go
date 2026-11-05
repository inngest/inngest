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
	"slices"
	"sort"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/arrow/scalar"
	"github.com/apache/iceberg-go"
	iceinternal "github.com/apache/iceberg-go/internal"
	iceio "github.com/apache/iceberg-go/io"
	"github.com/google/uuid"
)

// DataFiles returns the live data files in the current snapshot. Deleted
// manifest entries are omitted, matching the data_files metadata table.
func (i InspectTable) DataFiles(ctx context.Context) (array.RecordReader, error) {
	return i.inspectFiles(ctx, "data files", DataFilesSchema, false,
		func(manifest iceberg.ManifestFile) bool {
			return manifest.ManifestContent() == iceberg.ManifestContentData
		})
}

// Files returns all live data and delete files in the current snapshot.
// Deleted manifest entries are omitted.
func (i InspectTable) Files(ctx context.Context) (array.RecordReader, error) {
	return i.inspectFiles(ctx, "files", FilesSchema, false, nil)
}

// AllFiles returns the live data and delete files reachable from every
// snapshot currently tracked by the table. Shared manifests are scanned once,
// while duplicate file rows from different manifests are preserved.
func (i InspectTable) AllFiles(ctx context.Context) (array.RecordReader, error) {
	return i.inspectFiles(ctx, "all files", AllFilesSchema, true, nil)
}

// AllDataFiles returns the live data files reachable from every snapshot
// currently tracked by the table.
func (i InspectTable) AllDataFiles(ctx context.Context) (array.RecordReader, error) {
	return i.inspectFiles(ctx, "all data files", AllDataFilesSchema, true,
		func(manifest iceberg.ManifestFile) bool {
			return manifest.ManifestContent() == iceberg.ManifestContentData
		})
}

// AllDeleteFiles returns the live delete files reachable from every snapshot
// currently tracked by the table.
func (i InspectTable) AllDeleteFiles(ctx context.Context) (array.RecordReader, error) {
	return i.inspectFiles(ctx, "all delete files", AllDeleteFilesSchema, true,
		func(manifest iceberg.ManifestFile) bool {
			return manifest.ManifestContent() == iceberg.ManifestContentDeletes
		})
}

type inspectFilesSchema func(*iceberg.StructType) *iceberg.Schema

func (i InspectTable) inspectFiles(
	ctx context.Context,
	name string,
	schemaFn inspectFilesSchema,
	allSnapshots bool,
	includeManifest func(iceberg.ManifestFile) bool,
) (array.RecordReader, error) {
	partitionType, err := inspectPartitionType(i.tbl.metadata)
	if err != nil {
		return nil, fmt.Errorf("inspect %s: %w", name, err)
	}
	arrowSchema, err := SchemaToArrowSchema(schemaFn(partitionType), nil, true, false)
	if err != nil {
		return nil, fmt.Errorf("inspect %s: build arrow schema: %w", name, err)
	}

	appendFile := newInspectContentFileAppender(partitionType)
	appendEntry := func(bldr *array.RecordBuilder, entry iceberg.ManifestEntry) error {
		return appendFile(bldr, entry.DataFile())
	}
	var rr array.RecordReader
	if allSnapshots {
		rr, err = i.allManifestEntryReader(ctx, arrowSchema, true, includeManifest, appendEntry)
	} else {
		rr, err = i.manifestEntryReader(ctx, arrowSchema, true, includeManifest, appendEntry)
	}
	if err != nil {
		return nil, fmt.Errorf("inspect %s: %w", name, err)
	}

	return rr, nil
}

const inspectRecordBatchSize = 4096

// manifestEntryReader streams manifest entries into bounded Arrow record
// batches. It keeps only the current batch and manifest decoder state in
// memory instead of materializing every entry before returning a reader.
func (i InspectTable) manifestEntryReader(
	ctx context.Context,
	arrowSchema *arrow.Schema,
	discardDeleted bool,
	includeManifest func(iceberg.ManifestFile) bool,
	appendEntry func(*array.RecordBuilder, iceberg.ManifestEntry) error,
) (array.RecordReader, error) {
	snapshot := i.tbl.metadata.CurrentSnapshot()
	if snapshot == nil {
		return array.ReaderFromIter(arrowSchema, emptyInspectRecordBatch(i.alloc, arrowSchema)), nil
	}
	if i.tbl.fsF == nil {
		return nil, errors.New("table file IO is not configured")
	}

	manifestFS := sharedSnapshotManifestFSF(i.tbl.fsF)
	manifestSet, err := i.tbl.manifestSetWithFSF(ctx, *snapshot, manifestFS)
	if err != nil {
		return nil, err
	}
	fs, err := manifestFS(ctx)
	if err != nil {
		return nil, err
	}
	manifests := manifestSet.allManifests()

	return i.manifestEntryReaderFromManifestSource(
		ctx, arrowSchema, fs, discardDeleted, includeManifest, appendEntry,
		func(yield func(iceberg.ManifestFile, error) bool) {
			for _, manifest := range manifests {
				if !yield(manifest, nil) {
					return
				}
			}
		}), nil
}

// allManifestEntryReader scans each manifest reachable from the table's
// tracked snapshots exactly once. Manifest paths are immutable and unique, so
// the path is the stable identity used by Java and PyIceberg's all_* tables.
func (i InspectTable) allManifestEntryReader(
	ctx context.Context,
	arrowSchema *arrow.Schema,
	discardDeleted bool,
	includeManifest func(iceberg.ManifestFile) bool,
	appendEntry func(*array.RecordBuilder, iceberg.ManifestEntry) error,
) (array.RecordReader, error) {
	snapshots := i.tbl.metadata.Snapshots()
	if len(snapshots) == 0 {
		return array.ReaderFromIter(arrowSchema, emptyInspectRecordBatch(i.alloc, arrowSchema)), nil
	}
	if i.tbl.fsF == nil {
		return nil, errors.New("table file IO is not configured")
	}

	manifestFS := sharedSnapshotManifestFSF(i.tbl.fsF)
	fs, err := manifestFS(ctx)
	if err != nil {
		return nil, err
	}

	return i.manifestEntryReaderFromManifestSource(
		ctx, arrowSchema, fs, discardDeleted, includeManifest, appendEntry,
		func(yield func(iceberg.ManifestFile, error) bool) {
			seen := make(map[string]struct{})
			for _, snapshot := range snapshots {
				if err := ctx.Err(); err != nil {
					yield(nil, err)

					return
				}
				manifestSet, err := i.tbl.manifestSetWithFSF(ctx, snapshot, manifestFS)
				if err != nil {
					yield(nil, fmt.Errorf("read snapshot %d manifests: %w", snapshot.SnapshotID, err))

					return
				}
				for _, manifest := range manifestSet.allManifests() {
					if _, ok := seen[manifest.FilePath()]; ok {
						continue
					}
					seen[manifest.FilePath()] = struct{}{}
					if !yield(manifest, nil) {
						return
					}
				}
			}
		}), nil
}

func (i InspectTable) manifestEntryReaderFromManifestSource(
	ctx context.Context,
	arrowSchema *arrow.Schema,
	fs iceio.IO,
	discardDeleted bool,
	includeManifest func(iceberg.ManifestFile) bool,
	appendEntry func(*array.RecordBuilder, iceberg.ManifestEntry) error,
	source iter.Seq2[iceberg.ManifestFile, error],
) array.RecordReader {
	return array.ReaderFromIter(arrowSchema, func(yield func(arrow.RecordBatch, error) bool) {
		bldr := array.NewRecordBuilder(i.alloc, arrowSchema)
		defer bldr.Release()

		rows := 0
		emitted := false
		emit := func() bool {
			if rows == 0 {
				return true
			}

			batch := bldr.NewRecordBatch()
			rows = 0
			emitted = true

			return yield(batch, nil)
		}
		emitEmpty := func() {
			batch := bldr.NewRecordBatch()
			emitted = true
			_ = yield(batch, nil)
		}
		yieldError := func(err error) {
			_ = yield(nil, err)
		}

		for manifest, sourceErr := range source {
			if sourceErr != nil {
				yieldError(sourceErr)

				return
			}
			if err := ctx.Err(); err != nil {
				yieldError(err)

				return
			}
			if includeManifest != nil && !includeManifest(manifest) {
				continue
			}

			for entry, err := range manifest.Entries(fs, discardDeleted) {
				if err != nil {
					yieldError(fmt.Errorf("read manifest %s: %w", manifest.FilePath(), err))

					return
				}
				if err := ctx.Err(); err != nil {
					yieldError(err)

					return
				}
				if err := appendEntry(bldr, entry); err != nil {
					yieldError(fmt.Errorf("append manifest entry from %s: %w", manifest.FilePath(), err))

					return
				}
				rows++
				if rows == inspectRecordBatchSize && !emit() {
					return
				}
			}
		}

		if rows > 0 {
			_ = emit()
		} else if !emitted {
			emitEmpty()
		}
	})
}

func emptyInspectRecordBatch(alloc memory.Allocator, schema *arrow.Schema) iter.Seq2[arrow.RecordBatch, error] {
	return func(yield func(arrow.RecordBatch, error) bool) {
		bldr := array.NewRecordBuilder(alloc, schema)
		defer bldr.Release()

		batch := bldr.NewRecordBatch()
		_ = yield(batch, nil)
	}
}

// inspectPartitionType returns the table-wide partition type. It contains the
// union of partition fields from every spec, which lets metadata tables
// represent live files written before partition evolution.
func inspectPartitionType(metadata Metadata) (*iceberg.StructType, error) {
	return inspectPartitionTypeWithSchema(metadata, metadata.CurrentSchema())
}

func inspectPartitionTypeWithSchema(
	metadata Metadata,
	currentSchema *iceberg.Schema,
) (*iceberg.StructType, error) {
	specs := metadata.PartitionSpecs()
	sort.Slice(specs, func(left, right int) bool {
		return specs[left].ID() > specs[right].ID()
	})

	selected := make(map[int]iceberg.PartitionField)
	fieldsByID := make(map[int]iceberg.NestedField)
	for _, spec := range specs {
		for _, field := range spec.Fields() {
			if isInspectUnknownTransform(field.Transform) {
				return nil, fmt.Errorf("%w: cannot build metadata partition type for field %d with unknown transform %s",
					iceberg.ErrInvalidPartitionSpec, field.FieldID, field.Transform)
			}
		}

		partitionType := spec.PartitionType(currentSchema)
		for idx, field := range spec.Fields() {
			active := true
			for _, sourceID := range field.SourceIDs {
				// This is an existence check only. FindTypeByID clones the
				// complete nested type on every call, which is wasteful while
				// building the union of position-delete partition fields.
				if _, ok := currentSchema.FindFieldByIDRef(sourceID, iceinternal.SchemaRef{}); !ok {
					active = false

					break
				}
			}
			if !active || idx >= len(partitionType.FieldList) {
				continue
			}

			if previous, exists := selected[field.FieldID]; exists {
				if !slices.Equal(previous.SourceIDs, field.SourceIDs) {
					return nil, fmt.Errorf("%w: partition field ID %d has incompatible source IDs %v and %v",
						iceberg.ErrInvalidPartitionSpec, field.FieldID, previous.SourceIDs, field.SourceIDs)
				}

				previousVoid := isInspectVoidTransform(previous.Transform)
				fieldVoid := isInspectVoidTransform(field.Transform)
				if previousVoid || fieldVoid {
					if previousVoid && !fieldVoid {
						selected[field.FieldID] = field
						old := fieldsByID[field.FieldID]
						old.Type = partitionType.FieldList[idx].Type
						fieldsByID[field.FieldID] = old
					}

					continue
				}

				if !previous.Transform.Equals(field.Transform) {
					return nil, fmt.Errorf("%w: partition field ID %d has incompatible transforms %q and %q",
						iceberg.ErrInvalidPartitionSpec, field.FieldID, previous.Transform, field.Transform)
				}

				continue
			}

			selected[field.FieldID] = field
			fieldsByID[field.FieldID] = iceberg.NestedField{
				ID:       field.FieldID,
				Name:     field.Name,
				Type:     partitionType.FieldList[idx].Type,
				Required: false,
			}
		}
	}

	fields := make([]iceberg.NestedField, 0, len(fieldsByID))
	for _, field := range fieldsByID {
		fields = append(fields, field)
	}
	sort.Slice(fields, func(left, right int) bool { return fields[left].ID < fields[right].ID })

	return &iceberg.StructType{FieldList: fields}, nil
}

func isInspectVoidTransform(transform iceberg.Transform) bool {
	switch transform.(type) {
	case iceberg.VoidTransform, *iceberg.VoidTransform:
		return true
	default:
		return false
	}
}

func isInspectUnknownTransform(transform iceberg.Transform) bool {
	switch transform.(type) {
	case iceberg.UnknownTransform, *iceberg.UnknownTransform:
		return true
	default:
		return false
	}
}

// DataFilesSchema returns the common content-file schema used by the files
// metadata tables. The partition field is omitted for an unpartitioned table,
// as required by the Iceberg metadata-table spec.
func DataFilesSchema(partitionType *iceberg.StructType) *iceberg.Schema {
	return iceberg.NewSchema(0, inspectContentFileFields(partitionType)...)
}

// FilesSchema returns the schema shared by the files metadata tables.
func FilesSchema(partitionType *iceberg.StructType) *iceberg.Schema {
	return DataFilesSchema(partitionType)
}

// AllFilesSchema returns the schema of the all_files metadata table.
func AllFilesSchema(partitionType *iceberg.StructType) *iceberg.Schema {
	return DataFilesSchema(partitionType)
}

// AllDataFilesSchema returns the schema of the all_data_files metadata table.
func AllDataFilesSchema(partitionType *iceberg.StructType) *iceberg.Schema {
	return DataFilesSchema(partitionType)
}

// AllDeleteFilesSchema returns the schema of the all_delete_files metadata table.
func AllDeleteFilesSchema(partitionType *iceberg.StructType) *iceberg.Schema {
	return DataFilesSchema(partitionType)
}

const (
	inspectContentFieldIDContent            = 134
	inspectContentFieldIDFilePath           = 100
	inspectContentFieldIDFileFormat         = 101
	inspectContentFieldIDSpecID             = 141
	inspectContentFieldIDPartition          = 102
	inspectContentFieldIDRecordCount        = 103
	inspectContentFieldIDFileSize           = 104
	inspectContentFieldIDColumnSizes        = 108
	inspectContentFieldIDValueCounts        = 109
	inspectContentFieldIDNullValueCounts    = 110
	inspectContentFieldIDNaNValueCounts     = 137
	inspectContentFieldIDLowerBounds        = 125
	inspectContentFieldIDUpperBounds        = 128
	inspectContentFieldIDKeyMetadata        = 131
	inspectContentFieldIDSplitOffsets       = 132
	inspectContentFieldIDEqualityIDs        = 135
	inspectContentFieldIDSortOrderID        = 140
	inspectContentFieldIDFirstRowID         = 142
	inspectContentFieldIDReferencedDataFile = 143
	inspectContentFieldIDContentOffset      = 144
	inspectContentFieldIDContentSize        = 145
)

func inspectContentFileFields(partitionType *iceberg.StructType) []iceberg.NestedField {
	fields := []iceberg.NestedField{
		{ID: inspectContentFieldIDContent, Name: "content", Type: iceberg.PrimitiveTypes.Int32, Required: false},
		{ID: inspectContentFieldIDFilePath, Name: "file_path", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: inspectContentFieldIDFileFormat, Name: "file_format", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: inspectContentFieldIDSpecID, Name: "spec_id", Type: iceberg.PrimitiveTypes.Int32, Required: false},
	}
	if partitionType != nil && len(partitionType.FieldList) > 0 {
		fields = append(fields, iceberg.NestedField{
			ID: inspectContentFieldIDPartition, Name: "partition", Type: partitionType, Required: true,
		})
	}
	fields = append(fields,
		iceberg.NestedField{ID: inspectContentFieldIDRecordCount, Name: "record_count", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		iceberg.NestedField{ID: inspectContentFieldIDFileSize, Name: "file_size_in_bytes", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		iceberg.NestedField{ID: inspectContentFieldIDColumnSizes, Name: "column_sizes", Type: inspectInt64MapType(117, 118), Required: false},
		iceberg.NestedField{ID: inspectContentFieldIDValueCounts, Name: "value_counts", Type: inspectInt64MapType(119, 120), Required: false},
		iceberg.NestedField{ID: inspectContentFieldIDNullValueCounts, Name: "null_value_counts", Type: inspectInt64MapType(121, 122), Required: false},
		iceberg.NestedField{ID: inspectContentFieldIDNaNValueCounts, Name: "nan_value_counts", Type: inspectInt64MapType(138, 139), Required: false},
		iceberg.NestedField{ID: inspectContentFieldIDLowerBounds, Name: "lower_bounds", Type: inspectBinaryMapType(126, 127), Required: false},
		iceberg.NestedField{ID: inspectContentFieldIDUpperBounds, Name: "upper_bounds", Type: inspectBinaryMapType(129, 130), Required: false},
		iceberg.NestedField{ID: inspectContentFieldIDKeyMetadata, Name: "key_metadata", Type: iceberg.PrimitiveTypes.Binary, Required: false},
		iceberg.NestedField{ID: inspectContentFieldIDSplitOffsets, Name: "split_offsets", Type: &iceberg.ListType{ElementID: 133, Element: iceberg.PrimitiveTypes.Int64, ElementRequired: true}, Required: false},
		iceberg.NestedField{ID: inspectContentFieldIDEqualityIDs, Name: "equality_ids", Type: &iceberg.ListType{ElementID: 136, Element: iceberg.PrimitiveTypes.Int32, ElementRequired: true}, Required: false},
		iceberg.NestedField{ID: inspectContentFieldIDSortOrderID, Name: "sort_order_id", Type: iceberg.PrimitiveTypes.Int32, Required: false},
		iceberg.NestedField{ID: inspectContentFieldIDFirstRowID, Name: "first_row_id", Type: iceberg.PrimitiveTypes.Int64, Required: false},
		iceberg.NestedField{ID: inspectContentFieldIDReferencedDataFile, Name: "referenced_data_file", Type: iceberg.PrimitiveTypes.String, Required: false},
		iceberg.NestedField{ID: inspectContentFieldIDContentOffset, Name: "content_offset", Type: iceberg.PrimitiveTypes.Int64, Required: false},
		iceberg.NestedField{ID: inspectContentFieldIDContentSize, Name: "content_size_in_bytes", Type: iceberg.PrimitiveTypes.Int64, Required: false},
	)

	return fields
}

func inspectInt64MapType(keyID, valueID int) *iceberg.MapType {
	return &iceberg.MapType{KeyID: keyID, KeyType: iceberg.PrimitiveTypes.Int32, ValueID: valueID, ValueType: iceberg.PrimitiveTypes.Int64, ValueRequired: true}
}

func inspectBinaryMapType(keyID, valueID int) *iceberg.MapType {
	return &iceberg.MapType{KeyID: keyID, KeyType: iceberg.PrimitiveTypes.Int32, ValueID: valueID, ValueType: iceberg.PrimitiveTypes.Binary, ValueRequired: true}
}

func appendContentFileRecord(bldr *array.RecordBuilder, partitionType *iceberg.StructType, file iceberg.DataFile) error {
	contentFileBuilder, err := newInspectContentFileBuilder(bldr, partitionType)
	if err != nil {
		return err
	}

	return contentFileBuilder.append(file)
}

func newInspectContentFileAppender(partitionType *iceberg.StructType) func(*array.RecordBuilder, iceberg.DataFile) error {
	var current *array.RecordBuilder
	var contentFileBuilder inspectContentFileBuilder
	var bindErr error

	return func(bldr *array.RecordBuilder, file iceberg.DataFile) error {
		if bldr != current {
			contentFileBuilder, bindErr = newInspectContentFileBuilder(bldr, partitionType)
			current = bldr
		}
		if bindErr != nil {
			return bindErr
		}

		return contentFileBuilder.append(file)
	}
}

type inspectContentFileBuilder struct {
	content            *array.Int32Builder
	filePath           *array.StringBuilder
	fileFormat         *array.StringBuilder
	specID             *array.Int32Builder
	partition          *inspectPartitionBuilder
	recordCount        *array.Int64Builder
	fileSize           *array.Int64Builder
	columnSizes        *array.MapBuilder
	valueCounts        *array.MapBuilder
	nullValueCounts    *array.MapBuilder
	nanValueCounts     *array.MapBuilder
	lowerBounds        *array.MapBuilder
	upperBounds        *array.MapBuilder
	keyMetadata        *array.BinaryBuilder
	splitOffsets       *array.ListBuilder
	equalityIDs        *array.ListBuilder
	sortOrderID        *array.Int32Builder
	firstRowID         *array.Int64Builder
	referencedDataFile *array.StringBuilder
	contentOffset      *array.Int64Builder
	contentSize        *array.Int64Builder
	mapFieldIDs        []int
}

type inspectPartitionBuilder struct {
	builder *array.StructBuilder
	fields  []inspectPartitionFieldBuilder
}

type inspectPartitionFieldBuilder struct {
	id        int
	name      string
	typ       iceberg.Type
	arrowType arrow.DataType
	builder   array.Builder
}

func newInspectContentFileBuilder(bldr *array.RecordBuilder, partitionType *iceberg.StructType) (inspectContentFileBuilder, error) {
	return newInspectContentFileBuilderFromFields(bldr.Schema().Fields(), bldr.Field, partitionType)
}

func newInspectContentFileStructBuilder(bldr *array.StructBuilder, partitionType *iceberg.StructType) (inspectContentFileBuilder, error) {
	structType, ok := bldr.Type().(*arrow.StructType)
	if !ok {
		return inspectContentFileBuilder{}, fmt.Errorf("content-file builder has type %T, want struct", bldr.Type())
	}

	return newInspectContentFileBuilderFromFields(structType.Fields(), bldr.FieldBuilder, partitionType)
}

func newInspectContentFileBuilderFromFields(
	fields []arrow.Field,
	builder func(int) array.Builder,
	partitionType *iceberg.StructType,
) (inspectContentFileBuilder, error) {
	lookup, err := newInspectBuilderLookup("content-file", fields, builder)
	if err != nil {
		return inspectContentFileBuilder{}, err
	}
	if err := validateInspectBuilderLookup("content-file", lookup, inspectContentFileFieldIDs(partitionType)); err != nil {
		return inspectContentFileBuilder{}, err
	}

	var out inspectContentFileBuilder
	if out.content, err = inspectBuilderAs[*array.Int32Builder](lookup, inspectContentFieldIDContent, "content"); err != nil {
		return out, err
	}
	if out.filePath, err = inspectBuilderAs[*array.StringBuilder](lookup, inspectContentFieldIDFilePath, "file_path"); err != nil {
		return out, err
	}
	if out.fileFormat, err = inspectBuilderAs[*array.StringBuilder](lookup, inspectContentFieldIDFileFormat, "file_format"); err != nil {
		return out, err
	}
	if out.specID, err = inspectBuilderAs[*array.Int32Builder](lookup, inspectContentFieldIDSpecID, "spec_id"); err != nil {
		return out, err
	}
	if partitionType != nil && len(partitionType.FieldList) > 0 {
		partitionBuilder, bindErr := inspectBuilderAs[*array.StructBuilder](lookup, inspectContentFieldIDPartition, "partition")
		if bindErr != nil {
			return out, bindErr
		}
		if out.partition, err = newInspectPartitionBuilder(partitionBuilder, partitionType); err != nil {
			return out, err
		}
	}
	if out.recordCount, err = inspectBuilderAs[*array.Int64Builder](lookup, inspectContentFieldIDRecordCount, "record_count"); err != nil {
		return out, err
	}
	if out.fileSize, err = inspectBuilderAs[*array.Int64Builder](lookup, inspectContentFieldIDFileSize, "file_size_in_bytes"); err != nil {
		return out, err
	}
	if out.columnSizes, err = inspectBuilderAs[*array.MapBuilder](lookup, inspectContentFieldIDColumnSizes, "column_sizes"); err != nil {
		return out, err
	}
	if out.valueCounts, err = inspectBuilderAs[*array.MapBuilder](lookup, inspectContentFieldIDValueCounts, "value_counts"); err != nil {
		return out, err
	}
	if out.nullValueCounts, err = inspectBuilderAs[*array.MapBuilder](lookup, inspectContentFieldIDNullValueCounts, "null_value_counts"); err != nil {
		return out, err
	}
	if out.nanValueCounts, err = inspectBuilderAs[*array.MapBuilder](lookup, inspectContentFieldIDNaNValueCounts, "nan_value_counts"); err != nil {
		return out, err
	}
	if out.lowerBounds, err = inspectBuilderAs[*array.MapBuilder](lookup, inspectContentFieldIDLowerBounds, "lower_bounds"); err != nil {
		return out, err
	}
	if out.upperBounds, err = inspectBuilderAs[*array.MapBuilder](lookup, inspectContentFieldIDUpperBounds, "upper_bounds"); err != nil {
		return out, err
	}
	if out.keyMetadata, err = inspectBuilderAs[*array.BinaryBuilder](lookup, inspectContentFieldIDKeyMetadata, "key_metadata"); err != nil {
		return out, err
	}
	if out.splitOffsets, err = inspectBuilderAs[*array.ListBuilder](lookup, inspectContentFieldIDSplitOffsets, "split_offsets"); err != nil {
		return out, err
	}
	if out.equalityIDs, err = inspectBuilderAs[*array.ListBuilder](lookup, inspectContentFieldIDEqualityIDs, "equality_ids"); err != nil {
		return out, err
	}
	if out.sortOrderID, err = inspectBuilderAs[*array.Int32Builder](lookup, inspectContentFieldIDSortOrderID, "sort_order_id"); err != nil {
		return out, err
	}
	if out.firstRowID, err = inspectBuilderAs[*array.Int64Builder](lookup, inspectContentFieldIDFirstRowID, "first_row_id"); err != nil {
		return out, err
	}
	if out.referencedDataFile, err = inspectBuilderAs[*array.StringBuilder](lookup, inspectContentFieldIDReferencedDataFile, "referenced_data_file"); err != nil {
		return out, err
	}
	if out.contentOffset, err = inspectBuilderAs[*array.Int64Builder](lookup, inspectContentFieldIDContentOffset, "content_offset"); err != nil {
		return out, err
	}
	if out.contentSize, err = inspectBuilderAs[*array.Int64Builder](lookup, inspectContentFieldIDContentSize, "content_size_in_bytes"); err != nil {
		return out, err
	}

	return out, nil
}

func (b *inspectContentFileBuilder) append(file iceberg.DataFile) error {
	b.content.Append(int32(file.ContentType()))
	b.filePath.Append(file.FilePath())
	b.fileFormat.Append(string(file.FileFormat()))
	b.specID.Append(file.SpecID())

	if b.partition != nil {
		if err := b.partition.append(dataFilePartition(file)); err != nil {
			return err
		}
	}

	b.recordCount.Append(file.Count())
	b.fileSize.Append(file.FileSizeBytes())
	columnSizes, keyMetadata, splitOffsets, equalityFieldIDs := dataFileCollections(file)
	if err := appendInspectInt64Map(b.columnSizes, columnSizes, &b.mapFieldIDs); err != nil {
		return err
	}
	valueCounts, nullCounts, nanCounts, lowerBounds, upperBounds := dataFileStats(file)
	if err := appendInspectInt64Map(b.valueCounts, valueCounts, &b.mapFieldIDs); err != nil {
		return err
	}
	if err := appendInspectInt64Map(b.nullValueCounts, nullCounts, &b.mapFieldIDs); err != nil {
		return err
	}
	if err := appendInspectInt64Map(b.nanValueCounts, nanCounts, &b.mapFieldIDs); err != nil {
		return err
	}
	if err := appendInspectBinaryMap(b.lowerBounds, lowerBounds, &b.mapFieldIDs); err != nil {
		return err
	}
	if err := appendInspectBinaryMap(b.upperBounds, upperBounds, &b.mapFieldIDs); err != nil {
		return err
	}
	appendInspectBytes(b.keyMetadata, keyMetadata)
	if err := appendInspectInt64List(b.splitOffsets, splitOffsets); err != nil {
		return err
	}
	if err := appendInspectInt32List(b.equalityIDs, equalityFieldIDs); err != nil {
		return err
	}
	if err := appendInspectOptionalInt32(b.sortOrderID, file.SortOrderID()); err != nil {
		return err
	}
	appendInspectOptionalInt64(b.firstRowID, file.FirstRowID())
	appendInspectOptionalString(b.referencedDataFile, file.ReferencedDataFile())
	appendInspectOptionalInt64(b.contentOffset, file.ContentOffset())
	appendInspectOptionalInt64(b.contentSize, file.ContentSizeInBytes())

	return nil
}

func newInspectPartitionBuilder(
	builder *array.StructBuilder,
	partitionType *iceberg.StructType,
) (*inspectPartitionBuilder, error) {
	arrowType, ok := builder.Type().(*arrow.StructType)
	if !ok {
		return nil, fmt.Errorf("partition builder has type %T, want struct", builder.Type())
	}
	lookup, err := newInspectBuilderLookup("partition", arrowType.Fields(), builder.FieldBuilder)
	if err != nil {
		return nil, err
	}
	expected := make(map[int]struct{}, len(partitionType.FieldList))
	for _, field := range partitionType.FieldList {
		expected[field.ID] = struct{}{}
	}
	if err := validateInspectBuilderLookup("partition", lookup, expected); err != nil {
		return nil, err
	}

	fields := make([]inspectPartitionFieldBuilder, 0, len(partitionType.FieldList))
	for _, field := range partitionType.FieldList {
		fieldBuilder := lookup[field.ID]
		fields = append(fields, inspectPartitionFieldBuilder{
			id:        field.ID,
			name:      field.Name,
			typ:       field.Type,
			arrowType: fieldBuilder.Type(),
			builder:   fieldBuilder,
		})
	}

	return &inspectPartitionBuilder{builder: builder, fields: fields}, nil
}

func (b *inspectPartitionBuilder) append(values map[int]any) error {
	b.builder.Append(true)
	for _, field := range b.fields {
		value := values[field.id]
		if value == nil {
			field.builder.AppendNull()

			continue
		}
		sc, err := inspectValueScalar(value, field.typ, field.arrowType)
		if err != nil {
			return fmt.Errorf("partition field %q: %w", field.name, err)
		}
		if err := scalar.Append(field.builder, sc); err != nil {
			return err
		}
	}

	return nil
}

func inspectValueScalar(value any, typ iceberg.Type, arrowType arrow.DataType) (scalar.Scalar, error) {
	switch typ.(type) {
	case iceberg.DateType:
		switch value := value.(type) {
		case iceberg.Date:
			return scalar.NewDate32Scalar(arrow.Date32(value)), nil
		case int32:
			return scalar.NewDate32Scalar(arrow.Date32(value)), nil
		default:
			return nil, fmt.Errorf("unsupported date partition value %T", value)
		}
	case iceberg.TimeType:
		if value, ok := value.(iceberg.Time); ok {
			return scalar.NewTime64Scalar(arrow.Time64(value), arrowType), nil
		}

		return nil, fmt.Errorf("unsupported time partition value %T", value)
	case iceberg.TimestampType, iceberg.TimestampTzType:
		if value, ok := value.(iceberg.Timestamp); ok {
			return scalar.NewTimestampScalar(arrow.Timestamp(value), arrowType), nil
		}

		return nil, fmt.Errorf("unsupported timestamp partition value %T", value)
	case iceberg.TimestampNsType, iceberg.TimestampTzNsType:
		if value, ok := value.(iceberg.TimestampNano); ok {
			return scalar.NewTimestampScalar(arrow.Timestamp(value), arrowType), nil
		}

		return nil, fmt.Errorf("unsupported nanosecond timestamp partition value %T", value)
	case iceberg.UUIDType:
		if value, ok := value.(uuid.UUID); ok {
			return scalar.MakeScalarParam(value[:], arrowType)
		}

		return nil, fmt.Errorf("unsupported UUID partition value %T", value)
	case iceberg.DecimalType:
		switch value := value.(type) {
		case iceberg.DecimalLiteral:
			return scalar.NewDecimal128Scalar(value.Val, arrowType), nil
		case iceberg.Decimal:
			return scalar.NewDecimal128Scalar(value.Val, arrowType), nil
		default:
			return nil, fmt.Errorf("unsupported decimal partition value %T", value)
		}
	}

	return scalar.MakeScalarParam(value, arrowType)
}

func inspectContentFileFieldIDs(partitionType *iceberg.StructType) map[int]struct{} {
	ids := map[int]struct{}{
		inspectContentFieldIDContent:            {},
		inspectContentFieldIDFilePath:           {},
		inspectContentFieldIDFileFormat:         {},
		inspectContentFieldIDSpecID:             {},
		inspectContentFieldIDRecordCount:        {},
		inspectContentFieldIDFileSize:           {},
		inspectContentFieldIDColumnSizes:        {},
		inspectContentFieldIDValueCounts:        {},
		inspectContentFieldIDNullValueCounts:    {},
		inspectContentFieldIDNaNValueCounts:     {},
		inspectContentFieldIDLowerBounds:        {},
		inspectContentFieldIDUpperBounds:        {},
		inspectContentFieldIDKeyMetadata:        {},
		inspectContentFieldIDSplitOffsets:       {},
		inspectContentFieldIDEqualityIDs:        {},
		inspectContentFieldIDSortOrderID:        {},
		inspectContentFieldIDFirstRowID:         {},
		inspectContentFieldIDReferencedDataFile: {},
		inspectContentFieldIDContentOffset:      {},
		inspectContentFieldIDContentSize:        {},
	}
	if partitionType != nil && len(partitionType.FieldList) > 0 {
		ids[inspectContentFieldIDPartition] = struct{}{}
	}

	return ids
}

func newInspectBuilderLookup(scope string, fields []arrow.Field, builder func(int) array.Builder) (map[int]array.Builder, error) {
	lookup := make(map[int]array.Builder, len(fields))
	for idx, field := range fields {
		id := getFieldID(field)
		if id == nil {
			return nil, fmt.Errorf("%s field %q is missing a valid field ID", scope, field.Name)
		}
		if _, exists := lookup[*id]; exists {
			return nil, fmt.Errorf("%s schema contains duplicate field ID %d", scope, *id)
		}
		lookup[*id] = builder(idx)
	}

	return lookup, nil
}

func validateInspectBuilderLookup(scope string, lookup map[int]array.Builder, expected map[int]struct{}) error {
	if len(lookup) != len(expected) {
		return fmt.Errorf("%s schema has %d fields, want %d", scope, len(lookup), len(expected))
	}
	for id := range lookup {
		if _, ok := expected[id]; !ok {
			return fmt.Errorf("%s schema contains unexpected field ID %d", scope, id)
		}
	}
	for id := range expected {
		if _, ok := lookup[id]; !ok {
			return fmt.Errorf("%s schema is missing field ID %d", scope, id)
		}
	}

	return nil
}

func inspectBuilderAs[T any](lookup map[int]array.Builder, id int, name string) (T, error) {
	var zero T
	builder, ok := lookup[id]
	if !ok {
		return zero, fmt.Errorf("content-file schema is missing field %q (ID %d)", name, id)
	}
	typed, ok := builder.(T)
	if !ok {
		return zero, fmt.Errorf("content-file field %q (ID %d) has builder type %T", name, id, builder)
	}

	return typed, nil
}

func inspectInt32Value(value int, name string) (int32, error) {
	if value < math.MinInt32 || value > math.MaxInt32 {
		return 0, fmt.Errorf("%s %d does not fit in int32", name, value)
	}

	return int32(value), nil
}

func appendInspectInt64Map(builder *array.MapBuilder, values map[int]int64, scratch *[]int) error {
	if values == nil {
		builder.AppendNull()

		return nil
	}
	builder.Append(true)
	keys, ok := builder.KeyBuilder().(*array.Int32Builder)
	if !ok {
		return fmt.Errorf("map key builder has type %T, want int32", builder.KeyBuilder())
	}
	items, ok := builder.ItemBuilder().(*array.Int64Builder)
	if !ok {
		return fmt.Errorf("map item builder has type %T, want int64", builder.ItemBuilder())
	}
	ids := (*scratch)[:0]
	for id := range values {
		ids = append(ids, id)
	}
	*scratch = ids
	sort.Ints(ids)
	for _, id := range ids {
		key, err := inspectInt32Value(id, "field ID")
		if err != nil {
			return err
		}
		keys.Append(key)
		items.Append(values[id])
	}

	return nil
}

func appendInspectBinaryMap(builder *array.MapBuilder, values map[int][]byte, scratch *[]int) error {
	if values == nil {
		builder.AppendNull()

		return nil
	}
	builder.Append(true)
	keys, ok := builder.KeyBuilder().(*array.Int32Builder)
	if !ok {
		return fmt.Errorf("map key builder has type %T, want int32", builder.KeyBuilder())
	}
	items, ok := builder.ItemBuilder().(*array.BinaryBuilder)
	if !ok {
		return fmt.Errorf("map item builder has type %T, want binary", builder.ItemBuilder())
	}
	ids := (*scratch)[:0]
	for id := range values {
		ids = append(ids, id)
	}
	*scratch = ids
	sort.Ints(ids)
	for _, id := range ids {
		key, err := inspectInt32Value(id, "field ID")
		if err != nil {
			return err
		}
		keys.Append(key)
		items.Append(values[id])
	}

	return nil
}

func appendInspectBytes(builder *array.BinaryBuilder, value []byte) {
	if value == nil {
		builder.AppendNull()

		return
	}
	builder.Append(value)
}

func appendInspectInt64List(builder *array.ListBuilder, values []int64) error {
	if values == nil {
		builder.AppendNull()

		return nil
	}
	builder.Append(true)
	items, ok := builder.ValueBuilder().(*array.Int64Builder)
	if !ok {
		return fmt.Errorf("list item builder has type %T, want int64", builder.ValueBuilder())
	}
	items.AppendValues(values, nil)

	return nil
}

func appendInspectInt32List(builder *array.ListBuilder, values []int) error {
	if values == nil {
		builder.AppendNull()

		return nil
	}
	builder.Append(true)
	items, ok := builder.ValueBuilder().(*array.Int32Builder)
	if !ok {
		return fmt.Errorf("list item builder has type %T, want int32", builder.ValueBuilder())
	}
	for _, value := range values {
		converted, err := inspectInt32Value(value, "equality field ID")
		if err != nil {
			return err
		}
		items.Append(converted)
	}

	return nil
}

func appendInspectOptionalInt32(builder *array.Int32Builder, value *int) error {
	if value == nil {
		builder.AppendNull()

		return nil
	}
	converted, err := inspectInt32Value(*value, "sort order ID")
	if err != nil {
		return err
	}
	builder.Append(converted)

	return nil
}

func appendInspectOptionalInt64(builder *array.Int64Builder, value *int64) {
	if value == nil {
		builder.AppendNull()

		return
	}
	builder.Append(*value)
}

func appendInspectOptionalString(builder *array.StringBuilder, value *string) {
	if value == nil {
		builder.AppendNull()

		return
	}
	builder.Append(*value)
}
