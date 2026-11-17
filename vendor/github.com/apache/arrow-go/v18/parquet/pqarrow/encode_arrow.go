// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !pqarrow_read_only

package pqarrow

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"unsafe"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/bitutil"
	"github.com/apache/arrow-go/v18/arrow/decimal128"
	"github.com/apache/arrow-go/v18/arrow/decimal256"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/internal/utils"
	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/file"
	"github.com/apache/arrow-go/v18/parquet/schema"
)

// get the count of the number of leaf arrays for the type
func calcLeafCount(dt arrow.DataType) int {
	switch dt := dt.(type) {
	case arrow.ExtensionType:
		return calcLeafCount(dt.StorageType())
	case arrow.NestedType:
		nleaves := 0
		for _, f := range dt.Fields() {
			nleaves += calcLeafCount(f.Type)
		}
		return nleaves
	case *arrow.DictionaryType:
		return calcLeafCount(dt.ValueType)
	default:
		return 1
	}
}

func nullableRoot(manifest *SchemaManifest, field *SchemaField) bool {
	curField := field
	nullable := field.Field.Nullable
	for curField != nil {
		nullable = curField.Field.Nullable
		curField = manifest.GetParent(curField)
	}
	return nullable
}

func decimal128FitsInt32(val decimal128.Num) bool {
	switch val.HighBits() {
	case 0:
		return val.LowBits() <= math.MaxInt32
	case -1:
		return val.LowBits() >= uint64(0xffffffff80000000)
	default:
		return false
	}
}

func decimal128FitsInt64(val decimal128.Num) bool {
	switch val.HighBits() {
	case 0:
		return val.LowBits() <= math.MaxInt64
	case -1:
		return val.LowBits() >= uint64(0x8000000000000000)
	default:
		return false
	}
}

func decimal256FitsInt32(val decimal256.Num) bool {
	words := val.Array()
	switch {
	case words[1] == 0 && words[2] == 0 && words[3] == 0:
		return words[0] <= math.MaxInt32
	case words[1] == math.MaxUint64 && words[2] == math.MaxUint64 && words[3] == math.MaxUint64:
		return words[0] >= uint64(0xffffffff80000000)
	default:
		return false
	}
}

func decimal256FitsInt64(val decimal256.Num) bool {
	words := val.Array()
	switch {
	case words[1] == 0 && words[2] == 0 && words[3] == 0:
		return words[0] <= math.MaxInt64
	case words[1] == math.MaxUint64 && words[2] == math.MaxUint64 && words[3] == math.MaxUint64:
		return words[0] >= uint64(0x8000000000000000)
	default:
		return false
	}
}

func targetDecimalPrecision(cw file.ColumnChunkWriter) (int32, bool) {
	logical, ok := cw.Descr().LogicalType().(schema.DecimalLogicalType)
	if !ok {
		return 0, false
	}
	return logical.Precision(), true
}

func decimal128FitsTargetPrecision(val decimal128.Num, precision int32) bool {
	if precision <= 0 || precision > decimal128.MaxPrecision {
		return false
	}
	// Abs overflows for the minimum two's-complement value.
	if val.Sign() < 0 && val.Negate() == val {
		return false
	}
	return val.FitsInPrecision(precision)
}

func decimal256FitsTargetPrecision(val decimal256.Num, precision int32) bool {
	if precision <= 0 || precision > decimal256.MaxPrecision {
		return false
	}
	// Abs overflows for the minimum two's-complement value.
	if val.Sign() < 0 && val.Negate() == val {
		return false
	}
	return val.FitsInPrecision(precision)
}

// validatePresentDecimalValues visits only values represented by a definition
// level. Values below the repeated ancestor definition level are list
// placeholders and do not have a corresponding entry in arr.
func validatePresentDecimalValues(arr arrow.Array, defLevels []int16, levelInfo file.LevelInfo, validate func(int) error) error {
	if len(defLevels) == 0 {
		for idx := 0; idx < arr.Len(); idx++ {
			if arr.IsValid(idx) {
				if err := validate(idx); err != nil {
					return err
				}
			}
		}
		return nil
	}

	valueIdx := 0
	for _, defLevel := range defLevels {
		if defLevel < levelInfo.RepeatedAncestorDefLevel {
			continue
		}
		if valueIdx >= arr.Len() {
			break
		}
		if defLevel == levelInfo.DefLevel && arr.IsValid(valueIdx) {
			if err := validate(valueIdx); err != nil {
				return err
			}
		}
		valueIdx++
	}
	return nil
}

// arrowColumnWriter is a convenience object for easily writing arrow data to a specific
// set of columns in a parquet file. Since a single arrow array can itself be a nested type
// consisting of multiple columns of data, this will write to all of the appropriate leaves in
// the parquet file, allowing easy writing of nested columns.
type arrowColumnWriter struct {
	builders  []*multipathLevelBuilder
	leafCount int
	colIdx    int
	rgw       file.RowGroupWriter
}

// newArrowColumnWriter returns a new writer using the chunked array to determine the number of leaf columns,
// and the provided schema manifest to determine the paths for writing the columns.
//
// Using an arrow column writer is a convenience to avoid having to process the arrow array yourself
// and determine the correct definition and repetition levels manually.
func newArrowColumnWriter(data *arrow.Chunked, offset, size int64, manifest *SchemaManifest, rgw file.RowGroupWriter, leafColIdx int) (arrowColumnWriter, error) {
	if data.Len() == 0 {
		return arrowColumnWriter{leafCount: calcLeafCount(data.DataType()), rgw: rgw}, nil
	}

	var (
		absPos      int64
		chunkOffset int64
		chunkIdx    int
		values      int64
	)

	for idx, chnk := range data.Chunks() {
		chunkIdx = idx
		if absPos >= offset {
			break
		}

		chunkLen := int64(chnk.Len())
		if absPos+chunkLen > offset {
			chunkOffset = offset - absPos
			break
		}

		absPos += chunkLen
	}

	if absPos >= int64(data.Len()) {
		return arrowColumnWriter{}, errors.New("cannot write data at offset past end of chunked array")
	}

	leafCount := calcLeafCount(data.DataType())
	isNullable := false
	// row group writer hasn't been advanced yet so add 1 to the current
	// which is the one this instance will start writing for
	// colIdx := rgw.CurrentColumn() + 1

	schemaField, err := manifest.GetColumnField(leafColIdx)
	if err != nil {
		return arrowColumnWriter{}, err
	}
	isNullable = nullableRoot(manifest, schemaField)

	builders := make([]*multipathLevelBuilder, 0)
	releaseBuilders := func() {
		for _, bldr := range builders {
			bldr.Release()
		}
	}
	for values < size {
		chunk := data.Chunk(chunkIdx)
		available := int64(chunk.Len() - int(chunkOffset))
		chunkWriteSize := utils.Min(size-values, available)

		// the chunk offset will be 0 here except for possibly the first chunk
		// because of the above advancing logic
		arrToWrite := array.NewSlice(chunk, chunkOffset, chunkOffset+chunkWriteSize)

		if arrToWrite.Len() > 0 {
			bldr, err := newMultipathLevelBuilder(arrToWrite, isNullable)
			arrToWrite.Release()
			if err != nil {
				releaseBuilders()
				return arrowColumnWriter{}, err
			}
			if leafCount != bldr.leafCount() {
				bldr.Release()
				releaseBuilders()
				return arrowColumnWriter{}, fmt.Errorf("data type leaf_count != builder leaf_count: %d - %d", leafCount, bldr.leafCount())
			}
			builders = append(builders, bldr)
		} else {
			arrToWrite.Release()
		}

		if chunkWriteSize == available {
			chunkOffset = 0
			chunkIdx++
		}
		values += chunkWriteSize
	}

	return arrowColumnWriter{builders: builders, leafCount: leafCount, rgw: rgw, colIdx: leafColIdx}, nil
}

func (acw *arrowColumnWriter) Write(ctx context.Context) error {
	arrCtx := arrowCtxFromContext(ctx)
	for leafIdx := 0; leafIdx < acw.leafCount; leafIdx++ {
		var (
			cw  file.ColumnChunkWriter
			err error
		)

		if acw.rgw.Buffered() {
			cw, err = acw.rgw.(file.BufferedRowGroupWriter).Column(acw.colIdx + leafIdx)
		} else {
			cw, err = acw.rgw.(file.SerialRowGroupWriter).NextColumn()
		}

		if err != nil {
			return err
		}

		for _, bldr := range acw.builders {
			if leafIdx == 0 {
				defer bldr.Release()
			}
			res, err := bldr.write(leafIdx, arrCtx)
			if err != nil {
				return err
			}
			defer res.Release()

			if len(res.postListVisitedElems) != 1 {
				return errors.New("lists with non-zero length null components are not supported")
			}
			rng := res.postListVisitedElems[0]
			values := array.NewSlice(res.leafArr, rng.start, rng.end)
			defer values.Release()
			if err = WriteArrowToColumn(ctx, cw, values, res.defLevels, res.repLevels, res.leafIsNullable); err != nil {
				return err
			}
		}
	}
	return nil
}

// WriteArrowToColumn writes apache arrow columnar data directly to a ColumnWriter.
// Returns non-nil error if the array data type is not compatible with the concrete
// writer type.
//
// leafArr is always a primitive (possibly dictionary encoded type).
// Leaf_field_nullable indicates whether the leaf array is considered nullable
// according to its schema in a Table or its parent array.
func WriteArrowToColumn(ctx context.Context, cw file.ColumnChunkWriter, leafArr arrow.Array, defLevels, repLevels []int16, leafFieldNullable bool) error {
	// Leaf nulls are canonical when there is only a single null element after a list
	// and it is at the leaf.
	colLevelInfo := cw.LevelInfo()
	singleNullable := (colLevelInfo.DefLevel == colLevelInfo.RepeatedAncestorDefLevel+1) && leafFieldNullable
	maybeParentNulls := colLevelInfo.HasNullableValues() && !singleNullable

	if maybeParentNulls && !cw.HasBitsBuffer() {
		buf := memory.NewResizableBuffer(cw.Properties().Allocator())
		buf.Resize(int(bitutil.BytesForBits(cw.Properties().WriteBatchSize())))
		cw.SetBitsBuffer(buf)
	}

	arrCtx := arrowCtxFromContext(ctx)
	defer func() {
		if arrCtx.dataBuffer != nil {
			arrCtx.dataBuffer.Release()
			arrCtx.dataBuffer = nil
		}
	}()

	if leafArr.DataType().ID() == arrow.DICTIONARY {
		return writeDictionaryArrow(arrCtx, cw, leafArr, defLevels, repLevels, maybeParentNulls)
	}
	return writeDenseArrow(arrCtx, cw, leafArr, defLevels, repLevels, maybeParentNulls)
}

type binaryarr interface {
	ValueOffsets() []int32
}

type binary64arr interface {
	ValueOffsets() []int64
}

func writeDenseArrow(ctx *arrowWriteContext, cw file.ColumnChunkWriter, leafArr arrow.Array, defLevels, repLevels []int16, maybeParentNulls bool) (err error) {
	if leafArr.DataType().ID() == arrow.EXTENSION {
		extensionArray := leafArr.(array.ExtensionArray)
		// Replace leafArr with its underlying storage array
		leafArr = extensionArray.Storage()
	}

	noNulls := cw.Descr().SchemaNode().RepetitionType() == parquet.Repetitions.Required || leafArr.NullN() == 0

	if ctx.dataBuffer == nil {
		ctx.dataBuffer = memory.NewResizableBuffer(cw.Properties().Allocator())
	}

	switch wr := cw.(type) {
	case *file.BooleanColumnChunkWriter:
		if leafArr.DataType().ID() != arrow.BOOL {
			return fmt.Errorf("type mismatch, column is %s, array is %s", cw.Type(), leafArr.DataType().ID())
		}
		if leafArr.Len() == 0 {
			_, err = wr.WriteBatch(nil, defLevels, repLevels)
			break
		}

		// Optimized path for non-nullable: write directly from bitmap without conversion to []bool
		// For nullable columns, fall back to []bool conversion to properly handle null positions
		if !maybeParentNulls && noNulls {
			boolArr := leafArr.(*array.Boolean)
			bitmapBytes := boolArr.Data().Buffers()[1].Bytes() // value bitmap
			bitmapOffset := int64(boolArr.Data().Offset())
			numValues := leafArr.Len()

			// Non-nullable: use direct bitmap write
			_, err = wr.WriteBitmapBatch(bitmapBytes, bitmapOffset, numValues, defLevels, repLevels)
		} else {
			// Nullable: use []bool path to properly handle nulls
			// (bitmap values at null positions are undefined)
			ctx.dataBuffer.ResizeNoShrink(leafArr.Len())
			buf := ctx.dataBuffer.Bytes()
			data := *(*[]bool)(unsafe.Pointer(&buf))
			for idx := range data {
				data[idx] = leafArr.(*array.Boolean).Value(idx)
			}
			_, err = wr.WriteBatchSpacedWithError(data, defLevels, repLevels, leafArr.NullBitmapBytes(), int64(leafArr.Data().Offset()))
		}
	case *file.Int32ColumnChunkWriter:
		var data []int32
		switch leafArr.DataType().ID() {
		case arrow.INT32:
			data = leafArr.(*array.Int32).Int32Values()
		case arrow.DATE32, arrow.UINT32:
			if leafArr.Data().Buffers()[1] != nil {
				data = arrow.Int32Traits.CastFromBytes(leafArr.Data().Buffers()[1].Bytes())
				data = data[leafArr.Data().Offset() : leafArr.Data().Offset()+leafArr.Len()]
			}
		case arrow.TIME32:
			if leafArr.DataType().(*arrow.Time32Type).Unit != arrow.Second {
				if leafArr.Data().Buffers()[1] != nil {
					data = arrow.Int32Traits.CastFromBytes(leafArr.Data().Buffers()[1].Bytes())
					data = data[leafArr.Data().Offset() : leafArr.Data().Offset()+leafArr.Len()]
				}
			} else { // coerce time32 if necessary by multiplying by 1000
				ctx.dataBuffer.ResizeNoShrink(arrow.Int32Traits.BytesRequired(leafArr.Len()))
				data = arrow.Int32Traits.CastFromBytes(ctx.dataBuffer.Bytes())
				for idx, val := range leafArr.(*array.Time32).Time32Values() {
					data[idx] = int32(val) * 1000
				}
			}
		case arrow.NULL:
			_, err = wr.WriteBatchSpacedWithError(nil, defLevels, repLevels, leafArr.NullBitmapBytes(), 0)
			return

		default:
			// simple integral cases, parquet physical storage is int32 or int64
			// so we have to create a new array of int32's for anything smaller than
			// 32-bits
			ctx.dataBuffer.ResizeNoShrink(arrow.Int32Traits.BytesRequired(leafArr.Len()))
			data = arrow.Int32Traits.CastFromBytes(ctx.dataBuffer.Bytes())
			switch leafArr.DataType().ID() {
			case arrow.UINT8:
				for idx, val := range leafArr.(*array.Uint8).Uint8Values() {
					data[idx] = int32(val)
				}
			case arrow.INT8:
				for idx, val := range leafArr.(*array.Int8).Int8Values() {
					data[idx] = int32(val)
				}
			case arrow.UINT16:
				for idx, val := range leafArr.(*array.Uint16).Uint16Values() {
					data[idx] = int32(val)
				}
			case arrow.INT16:
				for idx, val := range leafArr.(*array.Int16).Int16Values() {
					data[idx] = int32(val)
				}
			case arrow.DATE64:
				for idx, val := range leafArr.(*array.Date64).Date64Values() {
					data[idx] = int32(val / 86400000) // coerce date64 values
				}
			case arrow.DECIMAL128:
				arr := leafArr.(*array.Decimal128)
				precision, hasPrecision := targetDecimalPrecision(cw)
				if err := validatePresentDecimalValues(arr, defLevels, cw.LevelInfo(), func(idx int) error {
					val := arr.Value(idx)
					if !decimal128FitsInt32(val) {
						return fmt.Errorf("%w: Decimal128 value at index %d does not fit in Parquet INT32", arrow.ErrInvalid, idx)
					}
					if hasPrecision && !decimal128FitsTargetPrecision(val, precision) {
						return fmt.Errorf("%w: Decimal128 value at index %d does not fit Parquet DECIMAL precision %d", arrow.ErrInvalid, idx, precision)
					}
					return nil
				}); err != nil {
					return err
				}
				for idx, val := range arr.Values() {
					data[idx] = int32(val.LowBits())
				}
			case arrow.DECIMAL256:
				arr := leafArr.(*array.Decimal256)
				precision, hasPrecision := targetDecimalPrecision(cw)
				if err := validatePresentDecimalValues(arr, defLevels, cw.LevelInfo(), func(idx int) error {
					val := arr.Value(idx)
					if !decimal256FitsInt32(val) {
						return fmt.Errorf("%w: Decimal256 value at index %d does not fit in Parquet INT32", arrow.ErrInvalid, idx)
					}
					if hasPrecision && !decimal256FitsTargetPrecision(val, precision) {
						return fmt.Errorf("%w: Decimal256 value at index %d does not fit Parquet DECIMAL precision %d", arrow.ErrInvalid, idx, precision)
					}
					return nil
				}); err != nil {
					return err
				}
				for idx, val := range arr.Values() {
					data[idx] = int32(val.LowBits())
				}
			default:
				return fmt.Errorf("type mismatch, column is int32 writer, arrow array is %s, and not a compatible type", leafArr.DataType().Name())
			}
		}

		if !maybeParentNulls && noNulls {
			_, err = wr.WriteBatch(data, defLevels, repLevels)
		} else {
			nulls := leafArr.NullBitmapBytes()
			_, err = wr.WriteBatchSpacedWithError(data, defLevels, repLevels, nulls, int64(leafArr.Data().Offset()))
		}
	case *file.Int64ColumnChunkWriter:
		var data []int64
		switch leafArr.DataType().ID() {
		case arrow.TIMESTAMP:
			tstype := leafArr.DataType().(*arrow.TimestampType)
			if ctx.props.coerceTimestamps {
				// user explicitly requested coercion to specific unit
				if tstype.Unit == ctx.props.coerceTimestampUnit {
					// no conversion necessary
					if leafArr.Data().Buffers()[1] != nil {
						data = arrow.Int64Traits.CastFromBytes(leafArr.Data().Buffers()[1].Bytes())
						data = data[leafArr.Data().Offset() : leafArr.Data().Offset()+leafArr.Len()]
					}
				} else {
					ctx.dataBuffer.ResizeNoShrink(arrow.Int64Traits.BytesRequired(leafArr.Len()))
					data = arrow.Int64Traits.CastFromBytes(ctx.dataBuffer.Bytes())
					if err := writeCoerceTimestamps(leafArr.(*array.Timestamp), &ctx.props, data); err != nil {
						return err
					}
				}
			} else if (cw.Properties().Version() == parquet.V1_0 || cw.Properties().Version() == parquet.V2_4) && tstype.Unit == arrow.Nanosecond {
				// absent superceding user instructions, when writing a Parquet Version <=2.4 File,
				// timestamps in nanoseconds are coerced to microseconds
				ctx.dataBuffer.ResizeNoShrink(arrow.Int64Traits.BytesRequired(leafArr.Len()))
				data = arrow.Int64Traits.CastFromBytes(ctx.dataBuffer.Bytes())
				p := NewArrowWriterProperties(WithCoerceTimestamps(arrow.Microsecond), WithTruncatedTimestamps(true))
				if err := writeCoerceTimestamps(leafArr.(*array.Timestamp), &p, data); err != nil {
					return err
				}
			} else if tstype.Unit == arrow.Second {
				// absent superceding user instructions, timestamps in seconds are coerced
				// to milliseconds
				p := NewArrowWriterProperties(WithCoerceTimestamps(arrow.Millisecond))
				ctx.dataBuffer.ResizeNoShrink(arrow.Int64Traits.BytesRequired(leafArr.Len()))
				data = arrow.Int64Traits.CastFromBytes(ctx.dataBuffer.Bytes())
				if err := writeCoerceTimestamps(leafArr.(*array.Timestamp), &p, data); err != nil {
					return err
				}
			} else {
				// no data conversion necessary
				if leafArr.Data().Buffers()[1] != nil {
					data = arrow.Int64Traits.CastFromBytes(leafArr.Data().Buffers()[1].Bytes())
					data = data[leafArr.Data().Offset() : leafArr.Data().Offset()+leafArr.Len()]
				}
			}
		case arrow.UINT32:
			ctx.dataBuffer.ResizeNoShrink(arrow.Int64Traits.BytesRequired(leafArr.Len()))
			data = arrow.Int64Traits.CastFromBytes(ctx.dataBuffer.Bytes())
			for idx, val := range leafArr.(*array.Uint32).Uint32Values() {
				data[idx] = int64(val)
			}
		case arrow.INT64:
			data = leafArr.(*array.Int64).Int64Values()
		case arrow.UINT64, arrow.TIME64, arrow.DATE64:
			if leafArr.Data().Buffers()[1] != nil {
				data = arrow.Int64Traits.CastFromBytes(leafArr.Data().Buffers()[1].Bytes())
				data = data[leafArr.Data().Offset() : leafArr.Data().Offset()+leafArr.Len()]
			}
		case arrow.DECIMAL128:
			ctx.dataBuffer.ResizeNoShrink(arrow.Int64Traits.BytesRequired(leafArr.Len()))
			data = arrow.Int64Traits.CastFromBytes(ctx.dataBuffer.Bytes())
			arr := leafArr.(*array.Decimal128)
			precision, hasPrecision := targetDecimalPrecision(cw)
			if err := validatePresentDecimalValues(arr, defLevels, cw.LevelInfo(), func(idx int) error {
				val := arr.Value(idx)
				if !decimal128FitsInt64(val) {
					return fmt.Errorf("%w: Decimal128 value at index %d does not fit in Parquet INT64", arrow.ErrInvalid, idx)
				}
				if hasPrecision && !decimal128FitsTargetPrecision(val, precision) {
					return fmt.Errorf("%w: Decimal128 value at index %d does not fit Parquet DECIMAL precision %d", arrow.ErrInvalid, idx, precision)
				}
				return nil
			}); err != nil {
				return err
			}
			for idx, val := range arr.Values() {
				data[idx] = int64(val.LowBits())
			}
		case arrow.DECIMAL256:
			ctx.dataBuffer.ResizeNoShrink(arrow.Int64Traits.BytesRequired(leafArr.Len()))
			data = arrow.Int64Traits.CastFromBytes(ctx.dataBuffer.Bytes())
			arr := leafArr.(*array.Decimal256)
			precision, hasPrecision := targetDecimalPrecision(cw)
			if err := validatePresentDecimalValues(arr, defLevels, cw.LevelInfo(), func(idx int) error {
				val := arr.Value(idx)
				if !decimal256FitsInt64(val) {
					return fmt.Errorf("%w: Decimal256 value at index %d does not fit in Parquet INT64", arrow.ErrInvalid, idx)
				}
				if hasPrecision && !decimal256FitsTargetPrecision(val, precision) {
					return fmt.Errorf("%w: Decimal256 value at index %d does not fit Parquet DECIMAL precision %d", arrow.ErrInvalid, idx, precision)
				}
				return nil
			}); err != nil {
				return err
			}
			for idx, val := range arr.Values() {
				data[idx] = int64(val.LowBits())
			}
		default:
			return fmt.Errorf("unimplemented arrow type to write to int64 column: %s", leafArr.DataType().Name())
		}

		if !maybeParentNulls && noNulls {
			_, err = wr.WriteBatch(data, defLevels, repLevels)
		} else {
			nulls := leafArr.NullBitmapBytes()
			_, err = wr.WriteBatchSpacedWithError(data, defLevels, repLevels, nulls, int64(leafArr.Data().Offset()))
		}
	case *file.Int96ColumnChunkWriter:
		if leafArr.DataType().ID() != arrow.TIMESTAMP {
			return errors.New("unsupported arrow type to write to Int96 column")
		}
		ctx.dataBuffer.ResizeNoShrink(parquet.Int96Traits.BytesRequired(leafArr.Len()))
		data := parquet.Int96Traits.CastFromBytes(ctx.dataBuffer.Bytes())
		input := leafArr.(*array.Timestamp).TimestampValues()
		unit := leafArr.DataType().(*arrow.TimestampType).Unit
		for idx, val := range input {
			arrowTimestampToImpalaTimestamp(unit, int64(val), &data[idx])
		}

		if !maybeParentNulls && noNulls {
			_, err = wr.WriteBatch(data, defLevels, repLevels)
		} else {
			nulls := leafArr.NullBitmapBytes()
			_, err = wr.WriteBatchSpacedWithError(data, defLevels, repLevels, nulls, int64(leafArr.Data().Offset()))
		}
	case *file.Float32ColumnChunkWriter:
		if leafArr.DataType().ID() != arrow.FLOAT32 {
			return errors.New("invalid column type to write to Float")
		}
		if !maybeParentNulls && noNulls {
			_, err = wr.WriteBatch(leafArr.(*array.Float32).Float32Values(), defLevels, repLevels)
		} else {
			_, err = wr.WriteBatchSpacedWithError(leafArr.(*array.Float32).Float32Values(), defLevels, repLevels, leafArr.NullBitmapBytes(), int64(leafArr.Data().Offset()))
		}
	case *file.Float64ColumnChunkWriter:
		if leafArr.DataType().ID() != arrow.FLOAT64 {
			return errors.New("invalid column type to write to Float")
		}
		if !maybeParentNulls && noNulls {
			_, err = wr.WriteBatch(leafArr.(*array.Float64).Float64Values(), defLevels, repLevels)
		} else {
			_, err = wr.WriteBatchSpacedWithError(leafArr.(*array.Float64).Float64Values(), defLevels, repLevels, leafArr.NullBitmapBytes(), int64(leafArr.Data().Offset()))
		}
	case *file.ByteArrayColumnChunkWriter:
		var (
			buffer   = leafArr.Data().Buffers()[2]
			valueBuf []byte
		)

		if buffer == nil {
			valueBuf = []byte{}
		} else {
			valueBuf = buffer.Bytes()
		}

		if wr.SupportsArrowOffsets() {
			switch leafArr.DataType().ID() {
			case arrow.BINARY, arrow.STRING:
				offsets := leafArr.(binaryarr).ValueOffsets()
				if !maybeParentNulls && noNulls {
					_, err = wr.WriteBatchArrow(valueBuf, offsets, defLevels, repLevels)
				} else {
					_, err = wr.WriteBatchSpacedArrow(valueBuf, offsets, defLevels, repLevels,
						leafArr.NullBitmapBytes(), int64(leafArr.Data().Offset()))
				}
			case arrow.LARGE_BINARY, arrow.LARGE_STRING:
				offsets := leafArr.(binary64arr).ValueOffsets()
				if !maybeParentNulls && noNulls {
					_, err = wr.WriteBatchArrow64(valueBuf, offsets, defLevels, repLevels)
				} else {
					_, err = wr.WriteBatchSpacedArrow64(valueBuf, offsets, defLevels, repLevels,
						leafArr.NullBitmapBytes(), int64(leafArr.Data().Offset()))
				}
			default:
				return fmt.Errorf("%w: invalid column type to write to ByteArray: %s", arrow.ErrInvalid, leafArr.DataType().Name())
			}
			return err
		}

		data := make([]parquet.ByteArray, leafArr.Len())
		switch leafArr.DataType().ID() {
		case arrow.BINARY, arrow.STRING:
			offsets := leafArr.(binaryarr).ValueOffsets()
			for i := range data {
				data[i] = parquet.ByteArray(valueBuf[offsets[i]:offsets[i+1]])
			}
		case arrow.LARGE_BINARY, arrow.LARGE_STRING:
			offsets := leafArr.(binary64arr).ValueOffsets()
			for i := range data {
				data[i] = parquet.ByteArray(valueBuf[offsets[i]:offsets[i+1]])
			}
		default:
			return fmt.Errorf("%w: invalid column type to write to ByteArray: %s", arrow.ErrInvalid, leafArr.DataType().Name())
		}

		if !maybeParentNulls && noNulls {
			_, err = wr.WriteBatch(data, defLevels, repLevels)
		} else {
			_, err = wr.WriteBatchSpacedWithError(data, defLevels, repLevels, leafArr.NullBitmapBytes(), int64(leafArr.Data().Offset()))
		}

	case *file.FixedLenByteArrayColumnChunkWriter:
		switch dt := leafArr.DataType().(type) {
		case *arrow.FixedSizeBinaryType:
			if wr.SupportsArrowValues() {
				buffer := leafArr.Data().Buffers()[1]
				var valueBuf []byte
				if buffer != nil {
					start := leafArr.Data().Offset() * dt.ByteWidth
					end := start + leafArr.Len()*dt.ByteWidth
					valueBuf = buffer.Bytes()[start:end]
				}
				if !maybeParentNulls && noNulls {
					_, err = wr.WriteBatchArrow(valueBuf, defLevels, repLevels)
				} else {
					_, err = wr.WriteBatchSpacedArrow(valueBuf, defLevels, repLevels, leafArr.NullBitmapBytes(), int64(leafArr.Data().Offset()))
				}
				return err
			}
			data := make([]parquet.FixedLenByteArray, leafArr.Len())
			for idx := range data {
				data[idx] = leafArr.(*array.FixedSizeBinary).Value(idx)
			}
			if !maybeParentNulls && noNulls {
				_, err = wr.WriteBatch(data, defLevels, repLevels)
			} else {
				_, err = wr.WriteBatchSpacedWithError(data, defLevels, repLevels, leafArr.NullBitmapBytes(), int64(leafArr.Data().Offset()))
			}
		case *arrow.Decimal128Type:
			// parquet decimal are stored with FixedLength values where the length is
			// proportional to the precision. Arrow's Decimal are always stored with 16/32
			// bytes. thus the internal FLBA must be adjusted by the offset calculation
			offset := int(bitutil.BytesForBits(int64(dt.BitWidth()))) - int(DecimalSize(dt.Precision))
			ctx.dataBuffer.ResizeNoShrink((leafArr.Len() - leafArr.NullN()) * dt.BitWidth())
			scratch := ctx.dataBuffer.Bytes()
			typeLen := wr.Descr().TypeLength()
			fixDecimalEndianness := func(in decimal128.Num) parquet.FixedLenByteArray {
				out := scratch[offset : offset+typeLen]
				binary.BigEndian.PutUint64(scratch, uint64(in.HighBits()))
				binary.BigEndian.PutUint64(scratch[arrow.Uint64SizeBytes:], in.LowBits())
				scratch = scratch[2*arrow.Uint64SizeBytes:]
				return out
			}

			data := make([]parquet.FixedLenByteArray, leafArr.Len())
			arr := leafArr.(*array.Decimal128)
			if leafArr.NullN() == 0 {
				for idx := range data {
					data[idx] = fixDecimalEndianness(arr.Value(idx))
				}
				_, err = wr.WriteBatch(data, defLevels, repLevels)
			} else {
				for idx := range data {
					if arr.IsValid(idx) {
						data[idx] = fixDecimalEndianness(arr.Value(idx))
					}
				}
				_, err = wr.WriteBatchSpacedWithError(data, defLevels, repLevels, arr.NullBitmapBytes(), int64(arr.Data().Offset()))
			}
		case *arrow.Decimal256Type:
			// parquet decimal are stored with FixedLength values where the length is
			// proportional to the precision. Arrow's Decimal are always stored with 16/32
			// bytes. thus the internal FLBA must be adjusted by the offset calculation
			offset := int(bitutil.BytesForBits(int64(dt.BitWidth()))) - int(DecimalSize(dt.Precision))
			ctx.dataBuffer.ResizeNoShrink((leafArr.Len() - leafArr.NullN()) * dt.BitWidth())
			scratch := ctx.dataBuffer.Bytes()
			typeLen := wr.Descr().TypeLength()
			fixDecimalEndianness := func(in decimal256.Num) parquet.FixedLenByteArray {
				out := scratch[offset : offset+typeLen]
				vals := in.Array()
				binary.BigEndian.PutUint64(scratch, vals[3])
				binary.BigEndian.PutUint64(scratch[arrow.Uint64SizeBytes:], vals[2])
				binary.BigEndian.PutUint64(scratch[2*arrow.Uint64SizeBytes:], vals[1])
				binary.BigEndian.PutUint64(scratch[3*arrow.Uint64SizeBytes:], vals[0])
				scratch = scratch[4*arrow.Uint64SizeBytes:]
				return out
			}

			data := make([]parquet.FixedLenByteArray, leafArr.Len())
			arr := leafArr.(*array.Decimal256)
			if leafArr.NullN() == 0 {
				for idx := range data {
					data[idx] = fixDecimalEndianness(arr.Value(idx))
				}
				_, err = wr.WriteBatch(data, defLevels, repLevels)
			} else {
				for idx := range data {
					if arr.IsValid(idx) {
						data[idx] = fixDecimalEndianness(arr.Value(idx))
					}
				}
				_, err = wr.WriteBatchSpacedWithError(data, defLevels, repLevels, arr.NullBitmapBytes(), int64(arr.Data().Offset()))
			}
		case *arrow.Float16Type:
			typeLen := wr.Descr().TypeLength()
			if typeLen != arrow.Float16SizeBytes {
				return fmt.Errorf("%w: invalid FixedLenByteArray length to write from float16 column: %d", arrow.ErrInvalid, typeLen)
			}

			arr := leafArr.(*array.Float16)
			rawValues := arrow.Float16Traits.CastToBytes(arr.Values())
			data := make([]parquet.FixedLenByteArray, arr.Len())

			if arr.NullN() == 0 {
				for idx := range data {
					offset := idx * typeLen
					data[idx] = rawValues[offset : offset+typeLen]
				}
				_, err = wr.WriteBatch(data, defLevels, repLevels)
			} else {
				for idx := range data {
					if arr.IsValid(idx) {
						offset := idx * typeLen
						data[idx] = rawValues[offset : offset+typeLen]
					}
				}
				_, err = wr.WriteBatchSpacedWithError(data, defLevels, repLevels, arr.NullBitmapBytes(), int64(arr.Data().Offset()))
			}
		default:
			return fmt.Errorf("%w: invalid column type to write to FixedLenByteArray: %s", arrow.ErrInvalid, leafArr.DataType().Name())
		}
	default:
		return errors.New("unknown column writer physical type")
	}
	return
}

type coerceType int8

const (
	coerceInvalid coerceType = iota
	coerceDivide
	coerceMultiply
)

type coercePair struct {
	typ    coerceType
	factor int64
}

var factors = map[arrow.TimeUnit]map[arrow.TimeUnit]coercePair{
	arrow.Second: {
		arrow.Second:      {coerceInvalid, 0},
		arrow.Millisecond: {coerceMultiply, 1000},
		arrow.Microsecond: {coerceMultiply, 1000000},
		arrow.Nanosecond:  {coerceMultiply, 1000000000},
	},
	arrow.Millisecond: {
		arrow.Second:      {coerceInvalid, 0},
		arrow.Millisecond: {coerceMultiply, 1},
		arrow.Microsecond: {coerceMultiply, 1000},
		arrow.Nanosecond:  {coerceMultiply, 1000000},
	},
	arrow.Microsecond: {
		arrow.Second:      {coerceInvalid, 0},
		arrow.Millisecond: {coerceDivide, 1000},
		arrow.Microsecond: {coerceMultiply, 1},
		arrow.Nanosecond:  {coerceMultiply, 1000},
	},
	arrow.Nanosecond: {
		arrow.Second:      {coerceInvalid, 0},
		arrow.Millisecond: {coerceDivide, 1000000},
		arrow.Microsecond: {coerceDivide, 1000},
		arrow.Nanosecond:  {coerceMultiply, 1},
	},
}

func writeCoerceTimestamps(arr *array.Timestamp, props *ArrowWriterProperties, out []int64) error {
	source := arr.DataType().(*arrow.TimestampType).Unit
	target := props.coerceTimestampUnit
	truncation := props.allowTruncatedTimestamps

	vals := arr.TimestampValues()
	multiply := func(factor int64) error {
		for idx, val := range vals {
			if !arr.IsValid(idx) {
				continue
			}
			if int64(val) > math.MaxInt64/factor || int64(val) < math.MinInt64/factor {
				return fmt.Errorf("%w: casting timestamp from %s to %s would overflow", arrow.ErrInvalid, source, target)
			}
			out[idx] = int64(val) * factor
		}
		return nil
	}

	divide := func(factor int64) error {
		for idx, val := range vals {
			if !truncation && arr.IsValid(idx) && (int64(val)%factor != 0) {
				return fmt.Errorf("casting from %s to %s would lose data", source, target)
			}
			out[idx] = int64(val) / factor
		}
		return nil
	}

	coerce := factors[source][target]
	switch coerce.typ {
	case coerceMultiply:
		return multiply(coerce.factor)
	case coerceDivide:
		return divide(coerce.factor)
	default:
		panic("invalid coercion")
	}
}

const (
	julianEpochOffsetDays int64 = 2440588
	nanoSecondsPerDay           = 24 * 60 * 60 * 1000 * 1000 * 1000
)

func arrowTimestampToImpalaTimestamp(unit arrow.TimeUnit, t int64, out *parquet.Int96) {
	var unitsPerDay, nanosPerUnit int64
	switch unit {
	case arrow.Second:
		unitsPerDay, nanosPerUnit = 24*60*60, 1000*1000*1000
	case arrow.Millisecond:
		unitsPerDay, nanosPerUnit = 24*60*60*1000, 1000*1000
	case arrow.Microsecond:
		unitsPerDay, nanosPerUnit = 24*60*60*1000*1000, 1000
	case arrow.Nanosecond:
		unitsPerDay, nanosPerUnit = nanoSecondsPerDay, 1
	}

	days := t / unitsPerDay
	remainder := t % unitsPerDay
	if remainder < 0 {
		days--
		remainder += unitsPerDay
	}

	julianDays := days + julianEpochOffsetDays
	lastDayNanos := remainder * nanosPerUnit
	binary.LittleEndian.PutUint64((*out)[:8], uint64(lastDayNanos))
	binary.LittleEndian.PutUint32((*out)[8:], uint32(julianDays))
}
