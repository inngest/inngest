// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package table

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/compute"
	"github.com/apache/arrow-go/v18/arrow/scalar"
	"github.com/apache/iceberg-go"
	iceinternal "github.com/apache/iceberg-go/internal"
	iceio "github.com/apache/iceberg-go/io"
	"github.com/apache/iceberg-go/table/dv"
	tblutils "github.com/apache/iceberg-go/table/internal"
)

const (
	positionDeleteFilePathID       = math.MaxInt32 - 101
	positionDeletePosID            = math.MaxInt32 - 102
	positionDeleteRowID            = math.MaxInt32 - 103
	positionDeletePartitionID      = math.MaxInt32 - 5
	positionDeleteSpecID           = math.MaxInt32 - 4
	positionDeletePhysicalPathID   = math.MaxInt32 - 1
	positionDeleteContentOffsetID  = math.MaxInt32 - 6
	positionDeleteContentSizeID    = math.MaxInt32 - 7
	positionDeletePhysicalPathName = "delete_file_path"
)

// PositionDeletes returns the individual position-delete records referenced
// by the current snapshot. Parquet position-delete files and V3 deletion
// vectors are exposed through the same schema.
func (i InspectTable) PositionDeletes(ctx context.Context) (array.RecordReader, error) {
	partitionType, partitionIDs, err := positionDeletesPartitionType(i.tbl.metadata)
	if err != nil {
		return nil, fmt.Errorf("inspect position deletes: %w", err)
	}
	schema := PositionDeletesSchema(i.tbl.metadata.CurrentSchema(), partitionType, i.tbl.metadata.Version())
	arrowSchema, err := SchemaToArrowSchema(schema, nil, true, false)
	if err != nil {
		return nil, fmt.Errorf("inspect position deletes: build arrow schema: %w", err)
	}

	fs, manifests, err := i.currentPositionDeleteManifests(ctx)
	if err != nil {
		return nil, fmt.Errorf("inspect position deletes: %w", err)
	}
	ctx = compute.WithAllocator(ctx, i.alloc)

	return i.positionDeleteRecordReader(
		ctx, arrowSchema, fs, manifests, partitionType, partitionIDs, i.tbl.metadata.Version()), nil
}

func (i InspectTable) currentPositionDeleteManifests(
	ctx context.Context,
) (iceio.IO, []iceberg.ManifestFile, error) {
	snapshot := i.tbl.metadata.CurrentSnapshot()
	if snapshot == nil {
		return nil, nil, nil
	}
	if i.tbl.fsF == nil {
		return nil, nil, errors.New("table file IO is not configured")
	}
	manifestSet, err := i.tbl.manifestSet(ctx, *snapshot)
	if err != nil {
		return nil, nil, err
	}
	fs, err := i.tbl.fsF(ctx)
	if err != nil {
		return nil, nil, err
	}

	return fs, manifestSet.allManifests(), nil
}

type positionDeleteRecordAppender struct {
	filePath         *array.StringBuilder
	pos              *array.Int64Builder
	row              *array.StructBuilder
	partition        *inspectPartitionBuilder
	specID           *array.Int32Builder
	deleteFilePath   *array.StringBuilder
	contentOffset    *array.Int64Builder
	contentSize      *array.Int64Builder
	partitionType    *iceberg.StructType
	partitionIDByOld map[int]int
	formatVersion    int
	projection       positionDeleteProjectionOptions
}

type positionDeleteFileMeta struct {
	partition      map[int]any
	specID         int32
	deleteFilePath string
	contentOffset  *int64
	contentSize    *int64
}

type positionDeleteProjectionOptions struct {
	ctx             context.Context
	formatVersion   int
	tableSchema     *iceberg.Schema
	nameMapping     iceberg.NameMapping
	tableProperties iceberg.Properties
}

func newPositionDeleteRecordAppender(
	bldr *array.RecordBuilder,
	partitionType *iceberg.StructType,
	partitionIDByOld map[int]int,
	formatVersion int,
	options ...positionDeleteProjectionOptions,
) (positionDeleteRecordAppender, error) {
	nextField := 3
	projection := positionDeleteProjectionOptions{
		ctx:           context.Background(),
		formatVersion: formatVersion,
	}
	if len(options) > 1 {
		return positionDeleteRecordAppender{}, fmt.Errorf("%w: expected at most one position delete projection option", iceberg.ErrInvalidArgument)
	}
	if len(options) == 1 {
		projection = options[0]
		if projection.ctx == nil {
			projection.ctx = context.Background()
		}
		if projection.formatVersion == 0 {
			projection.formatVersion = formatVersion
		}
	}
	if projection.tableSchema == nil {
		rowType, ok := bldr.Field(2).Type().(*arrow.StructType)
		if !ok {
			return positionDeleteRecordAppender{}, fmt.Errorf("%w: destination row has type %s, want struct", iceberg.ErrInvalidSchema, bldr.Field(2).Type())
		}
		rowSchema, err := ArrowSchemaToIcebergWithOptions(
			arrow.NewSchema(rowType.Fields(), nil),
			ArrowToIcebergOptions{TableProperties: projection.tableProperties},
		)
		if err != nil {
			return positionDeleteRecordAppender{}, fmt.Errorf("%w: resolve destination row schema: %v", iceberg.ErrInvalidSchema, err)
		}
		projection.tableSchema = rowSchema
	}
	if projection.nameMapping == nil && projection.tableSchema != nil {
		projection.nameMapping = projection.tableSchema.NameMapping()
	}
	out := positionDeleteRecordAppender{
		filePath:         bldr.Field(0).(*array.StringBuilder),
		pos:              bldr.Field(1).(*array.Int64Builder),
		row:              bldr.Field(2).(*array.StructBuilder),
		partitionType:    partitionType,
		partitionIDByOld: partitionIDByOld,
		formatVersion:    formatVersion,
		projection:       projection,
	}
	if len(partitionType.FieldList) > 0 {
		partitionBuilder, err := newInspectPartitionBuilder(
			bldr.Field(nextField).(*array.StructBuilder), partitionType)
		if err != nil {
			return out, err
		}
		out.partition = partitionBuilder
		nextField++
	}
	out.specID = bldr.Field(nextField).(*array.Int32Builder)
	out.deleteFilePath = bldr.Field(nextField + 1).(*array.StringBuilder)
	if formatVersion >= 3 {
		out.contentOffset = bldr.Field(nextField + 2).(*array.Int64Builder)
		out.contentSize = bldr.Field(nextField + 3).(*array.Int64Builder)
	}

	return out, nil
}

func (a positionDeleteRecordAppender) append(
	file iceberg.DataFile,
	dataFilePath string,
	pos int64,
	deletedRow scalar.Scalar,
	projected ...bool,
) error {
	return a.appendPrepared(a.prepareFile(file), dataFilePath, pos, deletedRow, projected...)
}

func (a positionDeleteRecordAppender) prepareFile(file iceberg.DataFile) positionDeleteFileMeta {
	meta := positionDeleteFileMeta{
		specID:         file.SpecID(),
		deleteFilePath: file.FilePath(),
	}
	if a.partition != nil {
		borrowedPartition := dataFilePartition(file)
		meta.partition = make(map[int]any, len(borrowedPartition))
		for oldID, value := range borrowedPartition {
			newID, ok := a.partitionIDByOld[oldID]
			if !ok {
				continue
			}
			if bytes, ok := value.([]byte); ok {
				value = slices.Clone(bytes)
			}
			meta.partition[newID] = value
		}
	}
	if a.formatVersion >= 3 || file.FileFormat() == iceberg.PuffinFile {
		meta.contentOffset = file.ContentOffset()
		meta.contentSize = positionDeleteContentSize(file)
	}

	return meta
}

func (a positionDeleteRecordAppender) appendPrepared(
	meta positionDeleteFileMeta,
	dataFilePath string,
	pos int64,
	deletedRow scalar.Scalar,
	projected ...bool,
) error {
	a.filePath.Append(dataFilePath)
	a.pos.Append(pos)
	var err error
	if len(projected) > 0 && projected[0] {
		if deletedRow == nil || !deletedRow.IsValid() {
			a.row.AppendNull()
		} else {
			err = appendPositionDeleteProjectedScalar(a.row, deletedRow)
		}
	} else {
		err = appendProjectedPositionDeleteRow(a.row, deletedRow, a.projection)
	}
	if err != nil {
		return fmt.Errorf("append deleted row: %w", err)
	}

	if a.partition != nil {
		if err := a.partition.append(meta.partition); err != nil {
			return err
		}
	}
	a.specID.Append(meta.specID)
	a.deleteFilePath.Append(meta.deleteFilePath)
	if a.formatVersion >= 3 {
		appendInspectOptionalInt64(a.contentOffset, meta.contentOffset)
		appendInspectOptionalInt64(a.contentSize, meta.contentSize)
	}

	return nil
}

func positionDeleteContentSize(file iceberg.DataFile) *int64 {
	if file.FileFormat() != iceberg.PuffinFile {
		return nil
	}

	return file.ContentSizeInBytes()
}

// appendPositionDeleteRow projects a row from a position-delete file onto the
// current table schema. The shared Arrow projection path resolves field IDs,
// name mappings, nested types, promotions, and initial defaults consistently
// with normal table scans.
func appendProjectedPositionDeleteRow(
	builder *array.StructBuilder,
	deletedRow scalar.Scalar,
	projection positionDeleteProjectionOptions,
) error {
	if deletedRow == nil || !deletedRow.IsValid() {
		builder.AppendNull()

		return nil
	}

	row, ok := deletedRow.(*scalar.Struct)
	if !ok {
		return fmt.Errorf("%w: row has type %s, want struct", iceberg.ErrInvalidSchema, deletedRow.DataType())
	}
	rowType, ok := row.DataType().(*arrow.StructType)
	if !ok {
		return fmt.Errorf("%w: row has type %s, want struct", iceberg.ErrInvalidSchema, row.DataType())
	}
	rowBuilder := array.NewStructBuilder(compute.GetAllocator(projection.ctx), rowType)
	defer rowBuilder.Release()
	if err := appendPositionDeleteProjectedScalar(rowBuilder, row); err != nil {
		return fmt.Errorf("make row array: %w", err)
	}
	rowArray := rowBuilder.NewArray()
	defer rowArray.Release()

	rowStruct, ok := rowArray.(*array.Struct)
	if !ok {
		return fmt.Errorf("%w: row array has type %s, want struct", iceberg.ErrInvalidSchema, rowArray.DataType())
	}
	projected, err := projectPositionDeleteRows(projection.ctx, rowStruct, projection)
	if err != nil {
		return err
	}
	defer projected.Release()

	projectedStruct := array.RecordToStructArray(projected)
	defer projectedStruct.Release()
	projectedRow, err := scalar.GetScalar(projectedStruct, 0)
	if err != nil {
		return err
	}
	defer releasePositionDeleteScalar(projectedRow)

	return appendPositionDeleteProjectedScalar(builder, projectedRow)
}

func releasePositionDeleteScalar(value scalar.Scalar) {
	if releasable, ok := value.(interface{ Release() }); ok {
		releasable.Release()
	}
}

func appendPositionDeleteProjectedScalar(builder array.Builder, value scalar.Scalar) error {
	if value == nil || !value.IsValid() {
		builder.AppendNull()

		return nil
	}

	switch builder := builder.(type) {
	case *array.StructBuilder:
		source, ok := value.(*scalar.Struct)
		if !ok {
			return fmt.Errorf("%w: projected struct has type %s, want struct",
				iceberg.ErrInvalidSchema, value.DataType())
		}
		if len(source.Value) != builder.NumField() {
			return fmt.Errorf("%w: projected struct has %d fields, want %d",
				iceberg.ErrInvalidSchema, len(source.Value), builder.NumField())
		}
		builder.Append(true)
		for index, child := range source.Value {
			if err := appendPositionDeleteProjectedScalar(builder.FieldBuilder(index), child); err != nil {
				return fmt.Errorf("projected struct field %d: %w", index, err)
			}
		}

		return nil
	case *array.MapBuilder:
		source, ok := value.(*scalar.Map)
		if !ok {
			return fmt.Errorf("%w: projected map has type %s", iceberg.ErrInvalidSchema, value.DataType())
		}
		entries := source.GetList()
		if entries == nil {
			return fmt.Errorf("%w: projected map has no entries", iceberg.ErrInvalidSchema)
		}
		entryType, ok := entries.DataType().(*arrow.StructType)
		if !ok || entryType.NumFields() != 2 {
			return fmt.Errorf("%w: projected map entries have type %s", iceberg.ErrInvalidSchema, entries.DataType())
		}
		builder.Append(true)
		for index := range entries.Len() {
			entry, err := scalar.GetScalar(entries, index)
			if err != nil {
				return err
			}
			appendErr := func() error {
				defer releasePositionDeleteScalar(entry)
				entryStruct, ok := entry.(*scalar.Struct)
				if !ok || len(entryStruct.Value) != 2 {
					return fmt.Errorf("%w: projected map entry %d is not a two-field struct",
						iceberg.ErrInvalidSchema, index)
				}
				if entryStruct.Value[0] == nil || !entryStruct.Value[0].IsValid() {
					return fmt.Errorf("%w: projected map entry %d has a null key",
						iceberg.ErrInvalidSchema, index)
				}
				if err := appendPositionDeleteProjectedScalar(builder.KeyBuilder(), entryStruct.Value[0]); err != nil {
					return fmt.Errorf("map entry %d key: %w", index, err)
				}
				if err := appendPositionDeleteProjectedScalar(builder.ItemBuilder(), entryStruct.Value[1]); err != nil {
					return fmt.Errorf("map entry %d value: %w", index, err)
				}

				return nil
			}()
			if appendErr != nil {
				return appendErr
			}
		}

		return nil
	case array.ListLikeBuilder:
		source, ok := value.(scalar.ListScalar)
		if !ok {
			return fmt.Errorf("%w: projected list has type %s", iceberg.ErrInvalidSchema, value.DataType())
		}
		values := source.GetList()
		if values == nil {
			return fmt.Errorf("%w: projected list has no values", iceberg.ErrInvalidSchema)
		}
		if destination, ok := builder.Type().(*arrow.FixedSizeListType); ok &&
			values.Len() != int(destination.Len()) {
			return fmt.Errorf("%w: projected list has %d elements, want %d",
				iceberg.ErrInvalidSchema, values.Len(), destination.Len())
		}
		builder.Append(true)
		for index := range values.Len() {
			child, err := scalar.GetScalar(values, index)
			if err != nil {
				return err
			}
			appendErr := appendPositionDeleteProjectedScalar(builder.ValueBuilder(), child)
			releasePositionDeleteScalar(child)
			if appendErr != nil {
				return fmt.Errorf("list element %d: %w", index, appendErr)
			}
		}

		return nil
	default:
		return scalar.Append(builder, value)
	}
}

func projectPositionDeleteRows(
	ctx context.Context,
	rows *array.Struct,
	projection positionDeleteProjectionOptions,
) (arrow.RecordBatch, error) {
	if projection.tableSchema == nil {
		return nil, fmt.Errorf("%w: position delete projection has no table schema", iceberg.ErrInvalidSchema)
	}
	if rows == nil {
		return nil, fmt.Errorf("%w: position delete row array is nil", iceberg.ErrInvalidSchema)
	}
	rowType, ok := rows.DataType().(*arrow.StructType)
	if !ok {
		return nil, fmt.Errorf("%w: row has type %s, want struct", iceberg.ErrInvalidSchema, rows.DataType())
	}

	sourceArrowSchema := arrow.NewSchema(rowType.Fields(), nil)
	fileSchema, err := ArrowSchemaToIcebergWithOptions(sourceArrowSchema, ArrowToIcebergOptions{
		NameMapping:     projection.nameMapping,
		TableSchema:     projection.tableSchema,
		TableProperties: projection.tableProperties,
	})
	if err != nil {
		return nil, fmt.Errorf("resolve position delete row schema: %w", err)
	}

	sourceBatch := array.RecordFromStructArray(rows, sourceArrowSchema)
	defer sourceBatch.Release()

	return ToRequestedSchema(ctx, projection.tableSchema, fileSchema, sourceBatch, SchemaOptions{
		FormatVersion:        projection.formatVersion,
		IncludeFieldIDs:      true,
		AllowMissingRequired: true,
		TableProperties:      projection.tableProperties,
	})
}

func (i InspectTable) positionDeleteRecordReader(
	ctx context.Context,
	arrowSchema *arrow.Schema,
	fs iceio.IO,
	manifests []iceberg.ManifestFile,
	partitionType *iceberg.StructType,
	partitionIDByOld map[int]int,
	formatVersion int,
) array.RecordReader {
	return array.ReaderFromIter(arrowSchema, func(yield func(arrow.RecordBatch, error) bool) {
		if err := ctx.Err(); err != nil {
			_ = yield(nil, err)

			return
		}

		bldr := array.NewRecordBuilder(i.alloc, arrowSchema)
		defer bldr.Release()
		appender, err := newPositionDeleteRecordAppender(
			bldr, partitionType, partitionIDByOld, formatVersion,
			positionDeleteProjectionOptions{
				ctx:             ctx,
				formatVersion:   formatVersion,
				tableSchema:     i.tbl.metadata.CurrentSchema(),
				nameMapping:     i.tbl.metadata.NameMapping(),
				tableProperties: i.tbl.metadata.Properties(),
			},
		)
		if err != nil {
			_ = yield(nil, err)

			return
		}

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
		yieldError := func(err error) {
			_ = yield(nil, err)
		}
		appendRow := func(meta positionDeleteFileMeta, path string, pos int64, deletedRow scalar.Scalar, projected bool) (bool, error) {
			if err := appender.appendPrepared(meta, path, pos, deletedRow, projected); err != nil {
				return false, err
			}
			rows++
			if rows == inspectRecordBatchSize {
				return emit(), nil
			}

			return true, nil
		}

		for _, manifest := range manifests {
			if err := ctx.Err(); err != nil {
				yieldError(err)

				return
			}
			if manifest.ManifestContent() != iceberg.ManifestContentDeletes {
				continue
			}

			for entry, err := range manifest.Entries(fs, true) {
				if err != nil {
					yieldError(fmt.Errorf("read manifest %s: %w", manifest.FilePath(), err))

					return
				}
				if err := ctx.Err(); err != nil {
					yieldError(err)

					return
				}
				file := entry.DataFile()
				if file.ContentType() != iceberg.EntryContentPosDeletes {
					continue
				}

				var keepGoing bool
				switch file.FileFormat() {
				case iceberg.PuffinFile:
					meta := appender.prepareFile(file)
					keepGoing, err = appendDeletionVectorRows(ctx, fs, file, meta, appendRow)
				case iceberg.ParquetFile:
					meta := appender.prepareFile(file)
					keepGoing, err = appendParquetPositionDeleteRows(
						ctx, fs, file, meta, appendRow, appender.projection)
				default:
					keepGoing = false
					err = fmt.Errorf("%w: unsupported position delete file format %s",
						iceberg.ErrNotImplemented, file.FileFormat())
				}
				if err != nil {
					yieldError(fmt.Errorf("read position delete file %s: %w", file.FilePath(), err))

					return
				}
				if !keepGoing {
					return
				}
			}
		}

		if rows > 0 {
			_ = emit()
		} else if !emitted {
			batch := bldr.NewRecordBatch()
			_ = yield(batch, nil)
		}
	})
}

type appendPositionDeleteRow func(positionDeleteFileMeta, string, int64, scalar.Scalar, bool) (bool, error)

func appendDeletionVectorRows(
	ctx context.Context,
	fs iceio.IO,
	file iceberg.DataFile,
	meta positionDeleteFileMeta,
	appendRow appendPositionDeleteRow,
) (bool, error) {
	referencedDataFile := file.ReferencedDataFile()
	if referencedDataFile == nil {
		return false, fmt.Errorf("%w: deletion vector is missing referenced_data_file",
			iceberg.ErrInvalidSchema)
	}
	if meta.contentOffset == nil || meta.contentSize == nil {
		return false, fmt.Errorf("%w: deletion vector is missing content_offset/content_size_in_bytes",
			iceberg.ErrInvalidSchema)
	}
	bitmap, err := dv.ReadDV(fs, file)
	if err != nil {
		return false, err
	}
	for position := range bitmap.Positions() {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if position > math.MaxInt64 {
			return false, fmt.Errorf("%w: deletion position %d exceeds int64", iceberg.ErrInvalidSchema, position)
		}
		keepGoing, err := appendRow(meta, *referencedDataFile, int64(position), nil, false)
		if err != nil || !keepGoing {
			return keepGoing, err
		}
	}

	return true, nil
}

func appendParquetPositionDeleteRows(
	ctx context.Context,
	fs iceio.IO,
	file iceberg.DataFile,
	meta positionDeleteFileMeta,
	appendRow appendPositionDeleteRow,
	options ...positionDeleteProjectionOptions,
) (keepGoing bool, err error) {
	projection := positionDeleteProjectionOptions{ctx: ctx}
	if len(options) > 1 {
		return false, fmt.Errorf("%w: expected at most one position delete projection option", iceberg.ErrInvalidArgument)
	}
	if len(options) == 1 {
		projection = options[0]
		if projection.ctx == nil {
			projection.ctx = ctx
		}
	}
	source, err := tblutils.GetFile(ctx, fs, file, true)
	if err != nil {
		return false, err
	}
	reader, err := source.GetReader(ctx)
	if err != nil {
		return false, err
	}
	defer func() {
		if closeErr := reader.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	records, err := reader.GetRecords(ctx, nil, nil)
	if err != nil {
		return false, err
	}
	defer records.Release()

	for records.Next() {
		continueReading, appendErr := appendPositionDeleteRecord(
			ctx, meta, records.RecordBatch(), appendRow, projection)
		if appendErr != nil || !continueReading {
			return continueReading, appendErr
		}
	}
	if err := records.Err(); err != nil {
		return false, err
	}

	return true, nil
}

func appendPositionDeleteRecord(
	ctx context.Context,
	meta positionDeleteFileMeta,
	record arrow.RecordBatch,
	appendRow appendPositionDeleteRow,
	projection positionDeleteProjectionOptions,
) (bool, error) {
	filePathIndex, posIndex, err := positionDeleteColumnIndices(record.Schema())
	if err != nil {
		return false, err
	}
	filePaths, err := filePathValues(record.Column(filePathIndex))
	if err != nil {
		return false, err
	}
	if err := validatePositionDeleteFilePathValues(filePaths, record.Column(filePathIndex)); err != nil {
		return false, err
	}
	positions, ok := record.Column(posIndex).(*array.Int64)
	if !ok {
		return false, fmt.Errorf("%w: pos column has type %s, want int64",
			iceberg.ErrInvalidSchema, record.Column(posIndex).DataType())
	}
	if positions.NullN() > 0 {
		return false, fmt.Errorf("%w: null pos in position delete file", iceberg.ErrInvalidSchema)
	}
	if err := validatePositionDeletePositions(positions); err != nil {
		return false, err
	}

	rowIndex := -1
	if indices := record.Schema().FieldIndices("row"); len(indices) > 1 {
		return false, fmt.Errorf("%w: position delete file contains multiple row columns",
			iceberg.ErrInvalidSchema)
	} else if len(indices) == 1 {
		rowIndex = indices[0]
	}
	if rowIndex >= 0 && record.Column(rowIndex).NullN() > 0 {
		return false, fmt.Errorf("%w: null row in position delete file", iceberg.ErrInvalidSchema)
	}

	var projectedRecord arrow.RecordBatch
	var projectedRows *array.Struct
	if rowIndex >= 0 && projection.tableSchema != nil {
		rowArray, ok := record.Column(rowIndex).(*array.Struct)
		if !ok {
			return false, fmt.Errorf("%w: row column has type %s, want struct",
				iceberg.ErrInvalidSchema, record.Column(rowIndex).DataType())
		}
		projectedRecord, err = projectPositionDeleteRows(ctx, rowArray, projection)
		if err != nil {
			return false, err
		}
		projectedRows = array.RecordToStructArray(projectedRecord)
		defer projectedRecord.Release()
		defer projectedRows.Release()
	}

	for row := range int(record.NumRows()) {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		var deletedRow scalar.Scalar
		projected := false
		if rowIndex >= 0 {
			if projectedRows != nil {
				deletedRow, err = scalar.GetScalar(projectedRows, row)
				projected = true
			} else if !record.Column(rowIndex).IsNull(row) {
				deletedRow, err = scalar.GetScalar(record.Column(rowIndex), row)
			}
			if err != nil {
				return false, err
			}
		}
		continueReading, appendErr := appendRow(
			meta, filePaths.Value(row), positions.Value(row), deletedRow, projected)
		releasePositionDeleteScalar(deletedRow)
		if appendErr != nil || !continueReading {
			return continueReading, appendErr
		}
	}

	return true, nil
}

func validatePositionDeleteFilePathValues(values arrow.TypedArray[string], arr arrow.Array) error {
	var dictionary arrow.Array
	var indices *array.Dictionary
	if dict, ok := arr.(*array.Dictionary); ok {
		dictionary = dict.Dictionary()
		indices = dict
	}

	for index := range values.Len() {
		if values.IsNull(index) {
			return fmt.Errorf("%w: null file_path in position delete file", iceberg.ErrInvalidSchema)
		}
		if dictionary != nil && dictionary.IsNull(indices.GetValueIndex(index)) {
			return fmt.Errorf("%w: null file_path dictionary value in position delete file", iceberg.ErrInvalidSchema)
		}
	}

	return nil
}

func positionDeletesPartitionType(
	metadata Metadata,
) (*iceberg.StructType, map[int]int, error) {
	currentSchema := metadata.CurrentSchema()
	base, err := inspectPartitionTypeWithSchema(metadata, currentSchema)
	if err != nil {
		return nil, nil, err
	}

	used := map[int]struct{}{
		positionDeleteFilePathID:      {},
		positionDeletePosID:           {},
		positionDeleteRowID:           {},
		positionDeletePartitionID:     {},
		positionDeleteSpecID:          {},
		positionDeletePhysicalPathID:  {},
		positionDeleteContentOffsetID: {},
		positionDeleteContentSizeID:   {},
	}
	for _, field := range base.FieldList {
		used[field.ID] = struct{}{}
	}
	fields := make([]iceberg.NestedField, len(base.FieldList))
	idByOld := make(map[int]int, len(base.FieldList))
	maxFieldID := int64(math.MaxInt32)
	nextID := int64(1)
	wrapped := false
	lastColumnID := metadata.LastColumnID()
	if lastColumnID >= 0 && int64(lastColumnID) < maxFieldID {
		nextID = int64(lastColumnID) + 1
	} else if int64(lastColumnID) >= maxFieldID {
		wrapped = true
	}
	for index, field := range base.FieldList {
		for {
			if nextID > maxFieldID {
				if wrapped {
					return nil, nil, fmt.Errorf("%w: no field ID available for position delete partition field %q",
						iceberg.ErrInvalidSchema, field.Name)
				}
				nextID = 1
				wrapped = true
			}
			fieldID := int(nextID)
			if _, exists := used[fieldID]; !exists {
				if _, exists := currentSchema.FindFieldByIDRef(fieldID, iceinternal.SchemaRef{}); !exists {
					break
				}
			}
			nextID++
		}
		fieldID := int(nextID)
		idByOld[field.ID] = fieldID
		field.ID = fieldID
		fields[index] = field
		used[fieldID] = struct{}{}
		nextID++
	}

	return &iceberg.StructType{FieldList: fields}, idByOld, nil
}

// PositionDeletesSchema returns the schema of the position_deletes metadata table.
func PositionDeletesSchema(
	tableSchema *iceberg.Schema,
	partitionType *iceberg.StructType,
	formatVersion int,
) *iceberg.Schema {
	rowType := iceberg.StructType{}
	if tableSchema != nil {
		rowType = tableSchema.AsStruct()
	}
	fields := []iceberg.NestedField{
		{ID: positionDeleteFilePathID, Name: "file_path", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: positionDeletePosID, Name: "pos", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: positionDeleteRowID, Name: "row", Type: &rowType, Required: false},
	}
	if len(partitionType.FieldList) > 0 {
		fields = append(fields, iceberg.NestedField{
			ID:       positionDeletePartitionID,
			Name:     "partition",
			Type:     partitionType,
			Required: true,
		})
	}
	fields = append(fields,
		iceberg.NestedField{
			ID:       positionDeleteSpecID,
			Name:     "spec_id",
			Type:     iceberg.PrimitiveTypes.Int32,
			Required: true,
		},
		iceberg.NestedField{
			ID:       positionDeletePhysicalPathID,
			Name:     positionDeletePhysicalPathName,
			Type:     iceberg.PrimitiveTypes.String,
			Required: true,
		},
	)
	if formatVersion >= 3 {
		fields = append(fields,
			iceberg.NestedField{
				ID:       positionDeleteContentOffsetID,
				Name:     "content_offset",
				Type:     iceberg.PrimitiveTypes.Int64,
				Required: false,
			},
			iceberg.NestedField{
				ID:       positionDeleteContentSizeID,
				Name:     "content_size_in_bytes",
				Type:     iceberg.PrimitiveTypes.Int64,
				Required: false,
			},
		)
	}

	return iceberg.NewSchema(0, fields...)
}
