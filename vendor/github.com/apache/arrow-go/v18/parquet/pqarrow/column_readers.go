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

package pqarrow

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/bitutil"
	"github.com/apache/arrow-go/v18/arrow/compute"
	"github.com/apache/arrow-go/v18/arrow/decimal128"
	"github.com/apache/arrow-go/v18/arrow/decimal256"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/internal/utils"
	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/file"
	"github.com/apache/arrow-go/v18/parquet/schema"
	"golang.org/x/sync/errgroup"
)

// column reader for leaf columns (non-nested)
type leafReader struct {
	out       *arrow.Chunked
	rctx      *readerCtx
	field     *arrow.Field
	input     *columnIterator
	descr     *schema.Column
	recordRdr file.RecordReader
	props     ArrowReadProperties

	// current row group size metadata, used to proportion binary data pre-allocation
	curRGUncompressedBytes int64
	curRGNumRows           int64

	refCount atomic.Int64
}

func newLeafReader(rctx *readerCtx, field *arrow.Field, input *columnIterator, leafInfo file.LevelInfo, props ArrowReadProperties, bufferPool *sync.Pool) (*ColumnReader, error) {
	ret := &leafReader{
		rctx:      rctx,
		field:     field,
		input:     input,
		descr:     input.Descr(),
		recordRdr: file.NewRecordReader(input.Descr(), leafInfo, field.Type, rctx.mem, bufferPool),
		props:     props,
	}
	ret.refCount.Add(1)

	err := ret.nextRowGroup(0)
	return &ColumnReader{ret}, err
}

func (lr *leafReader) Retain() {
	lr.refCount.Add(1)
}

func (lr *leafReader) Release() {
	if lr.refCount.Add(-1) == 0 {
		lr.releaseOut()
		if lr.recordRdr != nil {
			lr.recordRdr.Release()
			lr.recordRdr = nil
		}
	}
}

func (lr *leafReader) GetDefLevels() ([]int16, error) {
	return lr.recordRdr.DefLevels()[:int(lr.recordRdr.LevelsPos())], nil
}

func (lr *leafReader) GetRepLevels() ([]int16, error) {
	return lr.recordRdr.RepLevels()[:int(lr.recordRdr.LevelsPos())], nil
}

func (lr *leafReader) IsOrHasRepeatedChild() bool { return false }

func (lr *leafReader) LoadBatch(nrecords int64) (err error) {
	lr.releaseOut()
	lr.recordRdr.Reset()
	// The binary builder was reset by GetBuilderChunks() at the end of the
	// previous LoadBatch. Pre-allocate its data buffer now, while it's fresh.
	lr.reserveBinaryData(nrecords)

	if err := lr.recordRdr.Reserve(nrecords); err != nil {
		return err
	}
	for nrecords > 0 {
		if !lr.recordRdr.HasMore() {
			break
		}
		numRead, err := lr.recordRdr.ReadRecords(nrecords)
		if err != nil {
			return err
		}
		nrecords -= numRead
		if numRead == 0 {
			if err = lr.nextRowGroup(nrecords); err != nil {
				return err
			}
		}
	}
	lr.out, err = transferColumnData(lr.recordRdr, lr.field.Type, lr.descr, lr.rctx.mem)
	return
}

func (lr *leafReader) BuildArray(int64) (*arrow.Chunked, error) {
	return lr.clearOut(), nil
}

// reserveBinaryData pre-allocates the underlying BinaryBuilder's data buffer
// proportionally: (rowsToRead / curRGNumRows) * curRGUncompressedBytes.
// It is a no-op for non-binary columns, when size metadata is unavailable,
// or when PreAllocBinaryData is not enabled in the read properties.
func (lr *leafReader) reserveBinaryData(rowsToRead int64) {
	if !lr.props.PreAllocBinaryData {
		return
	}
	brdr, ok := lr.recordRdr.(file.BinaryRecordReader)
	if !ok || lr.curRGNumRows <= 0 || lr.curRGUncompressedBytes <= 0 {
		return
	}
	effective := rowsToRead
	if effective <= 0 || effective > lr.curRGNumRows {
		effective = lr.curRGNumRows
	}
	brdr.ReserveData(lr.curRGUncompressedBytes * effective / lr.curRGNumRows)
}

// releaseOut will clear lr.out as well as release it if it wasn't nil
func (lr *leafReader) releaseOut() {
	if out := lr.clearOut(); out != nil {
		out.Release()
	}
}

// clearOut will clear lt.out and return the old value
func (lr *leafReader) clearOut() (out *arrow.Chunked) {
	out, lr.out = lr.out, nil
	return out
}

func (lr *leafReader) Field() *arrow.Field { return lr.field }

func (lr *leafReader) setPageReader(pr file.PageReader) error {
	rr, ok := lr.recordRdr.(file.RecordReaderWithError)
	if !ok {
		if pr != nil {
			_ = pr.Close()
		}
		return errors.New("record reader does not support error-returning page reader replacement")
	}
	if err := rr.SetPageReaderWithError(pr); err != nil {
		if pr != nil {
			_ = pr.Close()
		}
		return err
	}
	return nil
}

func (lr *leafReader) SeekToRow(rowIdx int64) error {
	pr, offset, err := lr.input.FindChunkForRow(rowIdx)
	if err != nil {
		return err
	}

	if err := lr.setPageReader(pr); err != nil {
		return err
	}
	return lr.recordRdr.SeekToRow(offset)
}

// nextRowGroup advances to the next row group. remainingRows is the number of
// records still to be read in the current batch; pass 0 during initialization
// (no batch is in progress yet, so no pre-allocation is needed).
func (lr *leafReader) nextRowGroup(remainingRows int64) error {
	pr, uncompressedBytes, numRows, err := lr.input.NextChunk()
	if err != nil {
		return err
	}
	if err := lr.setPageReader(pr); err != nil {
		return err
	}
	lr.curRGUncompressedBytes = uncompressedBytes
	lr.curRGNumRows = numRows
	// When called mid-batch, extend the builder's data buffer for the new row group.
	if remainingRows > 0 {
		lr.reserveBinaryData(remainingRows)
	}
	return nil
}

// column reader for struct arrays, has readers for each child which could
// themselves be nested or leaf columns.
type structReader struct {
	rctx             *readerCtx
	filtered         *arrow.Field
	levelInfo        file.LevelInfo
	children         []*ColumnReader
	defRepLevelChild *ColumnReader
	hasRepeatedChild bool
	props            ArrowReadProperties

	refCount atomic.Int64
}

func (sr *structReader) Retain() {
	sr.refCount.Add(1)
}

func (sr *structReader) Release() {
	if sr.refCount.Add(-1) == 0 {
		if sr.defRepLevelChild != nil {
			sr.defRepLevelChild.Release()
			sr.defRepLevelChild = nil
		}
		for _, c := range sr.children {
			c.Release()
		}
		sr.children = nil
	}
}

func newStructReader(rctx *readerCtx, filtered *arrow.Field, levelInfo file.LevelInfo, children []*ColumnReader, props ArrowReadProperties) *ColumnReader {
	ret := &structReader{
		rctx:      rctx,
		filtered:  filtered,
		levelInfo: levelInfo,
		children:  children,
		props:     props,
	}
	ret.refCount.Add(1)

	// there could be a mix of children some might be repeated and some might not be
	// if possible use one that isn't since that will be guaranteed to have the least
	// number of levels to reconstruct a nullable bitmap
	for _, child := range children {
		if !child.IsOrHasRepeatedChild() {
			ret.defRepLevelChild = child
			break
		}
	}

	if ret.defRepLevelChild == nil {
		ret.defRepLevelChild = children[0]
		ret.hasRepeatedChild = true
	}
	ret.defRepLevelChild.Retain()
	return &ColumnReader{ret}
}

func (sr *structReader) IsOrHasRepeatedChild() bool { return sr.hasRepeatedChild }

func (sr *structReader) GetDefLevels() ([]int16, error) {
	if len(sr.children) == 0 {
		return nil, errors.New("struct reader has no children")
	}

	// this method should only be called when this struct or one of its parents
	// are optional/repeated or has a repeated child
	// meaning all children must have rep/def levels associated with them
	return sr.defRepLevelChild.GetDefLevels()
}

func (sr *structReader) GetRepLevels() ([]int16, error) {
	if len(sr.children) == 0 {
		return nil, errors.New("struct reader has no children")
	}

	// this method should only be called when this struct or one of its parents
	// are optional/repeated or has a repeated child
	// meaning all children must have rep/def levels associated with them
	return sr.defRepLevelChild.GetRepLevels()
}

func (sr *structReader) SeekToRow(rowIdx int64) error {
	if !sr.props.Parallel {
		var firstErr error
		for _, rdr := range sr.children {
			if err := rdr.SeekToRow(rowIdx); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		return firstErr
	}

	var g errgroup.Group
	for _, rdr := range sr.children {
		rdr := rdr
		g.Go(func() error {
			return rdr.SeekToRow(rowIdx)
		})
	}

	return g.Wait()
}

func (sr *structReader) LoadBatch(nrecords int64) error {
	if !sr.props.Parallel {
		var firstErr error
		for _, rdr := range sr.children {
			if err := rdr.LoadBatch(nrecords); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		return firstErr
	}

	// Load batches in parallel
	// When reading structs with large numbers of columns, the serial load is very slow.
	// This is especially true when reading Cloud Storage. Loading concurrently
	// greatly improves performance.
	g := new(errgroup.Group)
	for _, rdr := range sr.children {
		rdr := rdr
		g.Go(func() error {
			return rdr.LoadBatch(nrecords)
		})
	}

	return g.Wait()
}

func (sr *structReader) Field() *arrow.Field { return sr.filtered }

func (sr *structReader) BuildArray(lenBound int64) (*arrow.Chunked, error) {
	validityIO := file.ValidityBitmapInputOutput{
		ReadUpperBound: lenBound,
		Read:           lenBound,
	}

	var nullBitmap *memory.Buffer

	if lenBound > 0 && (sr.hasRepeatedChild || sr.filtered.Nullable) {
		nullBitmap = memory.NewResizableBuffer(sr.rctx.mem)
		nullBitmap.Resize(int(bitutil.BytesForBits(lenBound)))
		defer nullBitmap.Release()
		validityIO.ValidBits = nullBitmap.Bytes()
		defLevels, err := sr.GetDefLevels()
		if err != nil {
			return nil, err
		}

		if sr.hasRepeatedChild {
			repLevels, err := sr.GetRepLevels()
			if err != nil {
				return nil, err
			}

			if err := file.DefRepLevelsToBitmap(defLevels, repLevels, sr.levelInfo, &validityIO); err != nil {
				return nil, err
			}
		} else {
			file.DefLevelsToBitmap(defLevels, sr.levelInfo, &validityIO)
		}
	}

	if nullBitmap != nil {
		nullBitmap.Resize(int(bitutil.BytesForBits(validityIO.Read)))
	}

	childArrData := make([]arrow.ArrayData, len(sr.children))
	defer releaseArrayData(childArrData)
	// gather children arrays and def levels
	for i, child := range sr.children {
		field, err := child.BuildArray(lenBound)
		if err != nil {
			return nil, err
		}

		childArrData[i], err = chunksToSingle(field, sr.rctx.mem)
		field.Release() // release field before checking
		if err != nil {
			return nil, err
		}
	}

	if !sr.filtered.Nullable && !sr.hasRepeatedChild {
		validityIO.Read = int64(childArrData[0].Len())
	}

	buffers := make([]*memory.Buffer, 1)
	if validityIO.NullCount > 0 {
		buffers[0] = nullBitmap
	}

	data := array.NewData(sr.filtered.Type, int(validityIO.Read), buffers, childArrData, int(validityIO.NullCount), 0)
	defer data.Release()
	arr := array.NewStructData(data)
	defer arr.Release()
	return arrow.NewChunked(sr.filtered.Type, []arrow.Array{arr}), nil
}

// column reader for repeated columns specifically for list arrays
type listReader struct {
	rctx     *readerCtx
	field    *arrow.Field
	info     file.LevelInfo
	itemRdr  *ColumnReader
	props    ArrowReadProperties
	refCount atomic.Int64
}

func newListReader(rctx *readerCtx, field *arrow.Field, info file.LevelInfo, childRdr *ColumnReader, props ArrowReadProperties) *ColumnReader {
	childRdr.Retain()
	lr := &listReader{rctx: rctx, field: field, info: info, itemRdr: childRdr, props: props}
	lr.refCount.Add(1)
	return &ColumnReader{
		lr,
	}
}

func (lr *listReader) Retain() {
	lr.refCount.Add(1)
}

func (lr *listReader) Release() {
	if lr.refCount.Add(-1) == 0 {
		if lr.itemRdr != nil {
			lr.itemRdr.Release()
			lr.itemRdr = nil
		}
	}
}

func (lr *listReader) GetDefLevels() ([]int16, error) {
	return lr.itemRdr.GetDefLevels()
}

func (lr *listReader) GetRepLevels() ([]int16, error) {
	return lr.itemRdr.GetRepLevels()
}

func (lr *listReader) Field() *arrow.Field { return lr.field }

func (lr *listReader) IsOrHasRepeatedChild() bool { return true }

func (lr *listReader) SeekToRow(rowIdx int64) error {
	return lr.itemRdr.SeekToRow(rowIdx)
}

func (lr *listReader) LoadBatch(nrecords int64) error {
	return lr.itemRdr.LoadBatch(nrecords)
}

func (lr *listReader) BuildArray(lenBound int64) (*arrow.Chunked, error) {
	return lr.buildArray(lenBound)
}

func (lr *listReader) buildArray(lenBound int64) (*arrow.Chunked, error) {
	var (
		defLevels      []int16
		repLevels      []int16
		err            error
		validityBuffer *memory.Buffer
	)

	if defLevels, err = lr.itemRdr.GetDefLevels(); err != nil {
		return nil, err
	}
	if repLevels, err = lr.itemRdr.GetRepLevels(); err != nil {
		return nil, err
	}

	validityIO := file.ValidityBitmapInputOutput{ReadUpperBound: lenBound}
	if lr.field.Nullable {
		validityBuffer = memory.NewResizableBuffer(lr.rctx.mem)
		validityBuffer.Resize(int(bitutil.BytesForBits(lenBound)))
		defer validityBuffer.Release()
		validityIO.ValidBits = validityBuffer.Bytes()
	}
	offsetsBuffer := memory.NewResizableBuffer(lr.rctx.mem)
	if lr.field.Type.ID() == arrow.LARGE_LIST {
		offsetsBuffer.Resize(arrow.Int64Traits.BytesRequired(int(lenBound) + 1))
	} else {
		offsetsBuffer.Resize(arrow.Int32Traits.BytesRequired(int(lenBound) + 1))
	}
	defer offsetsBuffer.Release()

	var boundedLen int64
	if lr.field.Type.ID() == arrow.LARGE_LIST {
		offsetData := arrow.Int64Traits.CastFromBytes(offsetsBuffer.Bytes())
		if err = file.DefRepLevelsToListInfo(defLevels, repLevels, lr.info, &validityIO, offsetData); err != nil {
			return nil, err
		}
		boundedLen = offsetData[int(validityIO.Read)]
	} else {
		offsetData := arrow.Int32Traits.CastFromBytes(offsetsBuffer.Bytes())
		if err = file.DefRepLevelsToListInfo(defLevels, repLevels, lr.info, &validityIO, offsetData); err != nil {
			return nil, err
		}
		boundedLen = int64(offsetData[int(validityIO.Read)])
	}

	// if the parent (itemRdr) has nulls and is a nested type like list
	// then we need BuildArray to account for that with the number of
	// definition levels when building out the bitmap. So the upper bound
	// to make sure we have the space for is the worst case scenario,
	// the upper bound is the value of the last offset + the nullcount
	arr, err := lr.itemRdr.BuildArray(boundedLen)
	if err != nil {
		return nil, err
	}
	defer arr.Release()

	// resize to actual number of elems returned
	if lr.field.Type.ID() == arrow.LARGE_LIST {
		offsetsBuffer.Resize(arrow.Int64Traits.BytesRequired(int(validityIO.Read) + 1))
	} else {
		offsetsBuffer.Resize(arrow.Int32Traits.BytesRequired(int(validityIO.Read) + 1))
	}
	if validityBuffer != nil {
		validityBuffer.Resize(int(bitutil.BytesForBits(validityIO.Read)))
	}

	item, err := chunksToSingle(arr, lr.rctx.mem)
	if err != nil {
		return nil, err
	}
	defer item.Release()
	itemArr := array.MakeFromData(item)
	defer itemArr.Release()

	if lr.field.Type.ID() == arrow.FIXED_SIZE_LIST {
		offsetData := arrow.Int32Traits.CastFromBytes(offsetsBuffer.Bytes())
		return lr.buildFixedSizeListArray(int(validityIO.Read), offsetData, validityBuffer,
			validityIO.NullCount, itemArr)
	}

	buffers := []*memory.Buffer{nil, offsetsBuffer}
	if validityIO.NullCount > 0 {
		buffers[0] = validityBuffer
	}

	data := array.NewData(lr.field.Type, int(validityIO.Read), buffers, []arrow.ArrayData{item}, int(validityIO.NullCount), 0)
	defer data.Release()
	out := array.MakeFromData(data)
	defer out.Release()
	return arrow.NewChunked(lr.field.Type, []arrow.Array{out}), nil
}

func (lr *listReader) buildFixedSizeListArray(length int, offsets []int32, validityBuffer *memory.Buffer,
	nullCount int64, item arrow.Array) (*arrow.Chunked, error) {
	listType := lr.field.Type.(*arrow.FixedSizeListType)
	listSize := int(listType.Len())

	if offsets[0] != 0 {
		return nil, fmt.Errorf("fixed-size list first offset must be zero, got %d", offsets[0])
	}
	if int64(offsets[length]) != int64(item.Len()) {
		return nil, fmt.Errorf("fixed-size list final offset %d does not match decoded child length %d",
			offsets[length], item.Len())
	}

	if nullCount == 0 {
		for idx := 0; idx < length; idx++ {
			if size := offsets[idx+1] - offsets[idx]; size != int32(listSize) {
				return nil, fmt.Errorf("expected all lists to be of size=%d, but index %d had size=%d", listSize, idx, size)
			}
		}
		data := array.NewData(lr.field.Type, length, []*memory.Buffer{nil},
			[]arrow.ArrayData{item.Data()}, 0, 0)
		defer data.Release()
		out := array.MakeFromData(data)
		defer out.Release()
		return arrow.NewChunked(lr.field.Type, []arrow.Array{out}), nil
	}

	validity := validityBuffer.Bytes()
	runCount := 0
	previousValid := false
	for idx := 0; idx < length; idx++ {
		valid := !lr.field.Nullable || bitutil.BitIsSet(validity, idx)
		if valid {
			if size := offsets[idx+1] - offsets[idx]; size != int32(listSize) {
				return nil, fmt.Errorf("expected all lists to be of size=%d, but index %d had size=%d", listSize, idx, size)
			}
		} else {
			if size := offsets[idx+1] - offsets[idx]; size != 0 {
				return nil, fmt.Errorf("null fixed-size list at index %d consumed %d child values", idx, size)
			}
		}
		if idx == 0 || valid != previousValid {
			runCount++
		}
		previousValid = valid
	}

	var child arrow.Array
	var err error
	// For a small number of runs, concatenating slices avoids materializing
	// indices. Once the run count is high, the temporary arrays dominate.
	const minRunsForTake = 1024
	if runCount < minRunsForTake {
		pieces := make([]arrow.Array, 0, runCount)
		defer func() { releaseArrays(pieces) }()
		for i := 0; i < length; {
			valid := !lr.field.Nullable || bitutil.BitIsSet(validity, i)
			end := i + 1
			for end < length {
				nextValid := !lr.field.Nullable || bitutil.BitIsSet(validity, end)
				if nextValid != valid {
					break
				}
				end++
			}
			if valid {
				pieces = append(pieces, array.NewSlice(item, int64(offsets[i]), int64(offsets[end])))
			} else {
				pieces = append(pieces, array.MakeArrayOfNull(lr.rctx.mem, listType.Elem(), (end-i)*listSize))
			}
			i = end
		}
		if len(pieces) == 0 {
			pieces = append(pieces, array.MakeArrayOfNull(lr.rctx.mem, listType.Elem(), 0))
		}
		child, err = array.Concatenate(pieces, lr.rctx.mem)
	} else {
		childLength := length * listSize
		indicesBuffer := memory.NewResizableBuffer(lr.rctx.mem)
		defer indicesBuffer.Release()
		indicesBuffer.Resize(arrow.Int32Traits.BytesRequired(childLength))
		indices := arrow.Int32Traits.CastFromBytes(indicesBuffer.Bytes())

		indicesValidity := memory.NewResizableBuffer(lr.rctx.mem)
		defer indicesValidity.Release()
		indicesValidity.Resize(int(bitutil.BytesForBits(int64(childLength))))
		clear(indicesValidity.Bytes())

		for i := 0; i < length; {
			valid := !lr.field.Nullable || bitutil.BitIsSet(validity, i)
			end := i + 1
			for end < length {
				nextValid := !lr.field.Nullable || bitutil.BitIsSet(validity, end)
				if nextValid != valid {
					break
				}
				end++
			}
			if valid {
				childStart := i * listSize
				runChildLength := (end - i) * listSize
				bitutil.SetBitsTo(indicesValidity.Bytes(), int64(childStart), int64(runChildLength), true)
				for idx := 0; idx < runChildLength; idx++ {
					indices[childStart+idx] = offsets[i] + int32(idx)
				}
			}
			i = end
		}

		indicesData := array.NewData(arrow.PrimitiveTypes.Int32, childLength,
			[]*memory.Buffer{indicesValidity, indicesBuffer}, nil, int(nullCount)*listSize, 0)
		defer indicesData.Release()
		indicesArr := array.NewInt32Data(indicesData)
		defer indicesArr.Release()

		ctx := compute.WithAllocator(context.Background(), lr.rctx.mem)
		child, err = compute.TakeArrayOpts(ctx, item, indicesArr, compute.TakeOptions{BoundsCheck: false})
	}
	if err != nil {
		return nil, err
	}
	defer child.Release()

	buffers := []*memory.Buffer{nil}
	if nullCount > 0 {
		buffers[0] = validityBuffer
	}
	data := array.NewData(lr.field.Type, length, buffers, []arrow.ArrayData{child.Data()}, int(nullCount), 0)
	defer data.Release()
	out := array.MakeFromData(data)
	defer out.Release()
	return arrow.NewChunked(lr.field.Type, []arrow.Array{out}), nil
}

func newFixedSizeListReader(rctx *readerCtx, field *arrow.Field, info file.LevelInfo, childRdr *ColumnReader, props ArrowReadProperties) *ColumnReader {
	return newListReader(rctx, field, info, childRdr, props)
}

// helper function to combine chunks into a single array.
func chunksToSingle(chunked *arrow.Chunked, mem memory.Allocator) (arrow.ArrayData, error) {
	switch len(chunked.Chunks()) {
	case 0:
		return array.NewData(chunked.DataType(), 0, []*memory.Buffer{nil, nil}, nil, 0, 0), nil
	case 1:
		data := chunked.Chunk(0).Data()
		data.Retain() // we pass control to the caller
		return data, nil
	default:
		// concatenate multiple chunks into a single array
		concatenated, err := array.Concatenate(chunked.Chunks(), mem)
		if err != nil {
			return nil, err
		}
		defer concatenated.Release()

		data := concatenated.Data()
		data.Retain()
		return data, nil
	}
}

// create a chunked arrow array from the raw record data
func transferColumnData(rdr file.RecordReader, valueType arrow.DataType, descr *schema.Column, mem memory.Allocator) (*arrow.Chunked, error) {
	dt := valueType
	if valueType.ID() == arrow.EXTENSION {
		dt = valueType.(arrow.ExtensionType).StorageType()
	}

	var (
		data arrow.ArrayData
		err  error
	)
	switch dt.ID() {
	case arrow.DICTIONARY:
		return transferDictionary(rdr, valueType, mem)
	case arrow.NULL:
		return arrow.NewChunked(arrow.Null, []arrow.Array{array.NewNull(rdr.ValuesWritten())}), nil
	case arrow.INT32, arrow.INT64, arrow.FLOAT32, arrow.FLOAT64:
		data = transferZeroCopy(rdr, valueType) // can just reference the raw data without copying
	case arrow.BOOL:
		data = transferBool(rdr)
	case arrow.UINT8,
		arrow.UINT16,
		arrow.UINT32,
		arrow.UINT64,
		arrow.INT8,
		arrow.INT16,
		arrow.DATE32,
		arrow.TIME32,
		arrow.TIME64:
		data = transferInt(rdr, valueType)
	case arrow.DATE64:
		data = transferDate64(rdr, valueType)
	case arrow.FIXED_SIZE_BINARY, arrow.BINARY, arrow.STRING, arrow.LARGE_BINARY, arrow.LARGE_STRING:
		return transferBinary(rdr, valueType, mem)
	case arrow.DECIMAL, arrow.DECIMAL256:
		switch descr.PhysicalType() {
		case parquet.Types.Int32, parquet.Types.Int64:
			data = transferDecimalInteger(rdr, valueType)
		case parquet.Types.ByteArray, parquet.Types.FixedLenByteArray:
			return transferDecimalBytes(rdr.(file.BinaryRecordReader), valueType)
		default:
			return nil, errors.New("physical type for decimal128/decimal256 must be int32, int64, bytearray or fixed len byte array")
		}
	case arrow.TIMESTAMP:
		tstype := valueType.(*arrow.TimestampType)
		switch tstype.Unit {
		case arrow.Millisecond, arrow.Microsecond:
			data = transferZeroCopy(rdr, valueType)
		case arrow.Nanosecond:
			if descr.PhysicalType() == parquet.Types.Int96 {
				data, err = transferInt96(rdr, valueType)
				if err != nil {
					return nil, err
				}
			} else {
				data = transferZeroCopy(rdr, valueType)
			}
		default:
			return nil, errors.New("time unit not supported")
		}
	case arrow.FLOAT16:
		if descr.PhysicalType() != parquet.Types.FixedLenByteArray {
			return nil, errors.New("physical type for float16 must be fixed len byte array")
		}
		if len := arrow.Float16SizeBytes; descr.TypeLength() != len {
			return nil, fmt.Errorf("fixed len byte array length for float16 must be %d", len)
		}
		return transferBinary(rdr, valueType, mem)
	default:
		return nil, fmt.Errorf("no support for reading columns of type: %s", valueType.Name())
	}

	defer data.Release()
	arr := array.MakeFromData(data)
	defer arr.Release()
	return arrow.NewChunked(valueType, []arrow.Array{arr}), nil
}

func transferZeroCopy(rdr file.RecordReader, dt arrow.DataType) arrow.ArrayData {
	bitmap := rdr.ReleaseValidBits()
	values := rdr.ReleaseValues()
	defer func() {
		if bitmap != nil {
			bitmap.Release()
		}
		if values != nil {
			values.Release()
		}
	}()

	return array.NewData(dt, rdr.ValuesWritten(),
		[]*memory.Buffer{bitmap, values},
		nil, int(rdr.NullCount()), 0)
}

func transferBinary(rdr file.RecordReader, dt arrow.DataType, mem memory.Allocator) (*arrow.Chunked, error) {
	brdr := rdr.(file.BinaryRecordReader)
	if brdr.ReadDictionary() {
		return transferDictionary(brdr, &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Int32, ValueType: dt}, mem)
	}
	chunks := brdr.GetBuilderChunks()
	defer releaseArrays(chunks)

	switch dt := dt.(type) {
	case arrow.ExtensionType:
		for idx, chunk := range chunks {
			chunks[idx] = array.NewExtensionArrayWithStorage(dt, chunk)
			chunk.Release()
		}
	case *arrow.StringType, *arrow.LargeStringType:
		for idx, chunk := range chunks {
			chunks[idx] = array.MakeFromData(chunk.Data())
			chunk.Release()
		}
	case *arrow.Float16Type:
		for idx, chunk := range chunks {
			data := chunk.Data()
			f16_data := array.NewData(dt, data.Len(), data.Buffers(), nil, data.NullN(), data.Offset())
			defer f16_data.Release()
			chunks[idx] = array.NewFloat16Data(f16_data)
			chunk.Release()
		}
	}
	return arrow.NewChunked(dt, chunks), nil
}

type parquetInteger interface {
	~int32 | ~int64
}

type arrowInteger interface {
	~int8 | ~uint8 | ~int16 | ~uint16 | ~int32 | ~uint32 | ~int64 | ~uint64
}

func convertIntegerValues[Out arrowInteger, In parquetInteger](out []Out, values []In) {
	for i, value := range values {
		out[i] = Out(value)
	}
}

func transferIntegerValues[In parquetInteger](values []In, data []byte, dt arrow.Type) {
	switch dt {
	case arrow.INT8:
		convertIntegerValues(arrow.Int8Traits.CastFromBytes(data), values)
	case arrow.UINT8:
		convertIntegerValues(arrow.Uint8Traits.CastFromBytes(data), values)
	case arrow.INT16:
		convertIntegerValues(arrow.Int16Traits.CastFromBytes(data), values)
	case arrow.UINT16:
		convertIntegerValues(arrow.Uint16Traits.CastFromBytes(data), values)
	case arrow.UINT32:
		convertIntegerValues(arrow.Uint32Traits.CastFromBytes(data), values)
	case arrow.UINT64:
		convertIntegerValues(arrow.Uint64Traits.CastFromBytes(data), values)
	case arrow.DATE32:
		convertIntegerValues(arrow.Date32Traits.CastFromBytes(data), values)
	case arrow.TIME32:
		convertIntegerValues(arrow.Time32Traits.CastFromBytes(data), values)
	case arrow.TIME64:
		convertIntegerValues(arrow.Time64Traits.CastFromBytes(data), values)
	}
}

func transferInt(rdr file.RecordReader, dt arrow.DataType) arrow.ArrayData {
	// create buffer for proper type since parquet only has int32 and int64
	// physical representations, but we want the correct type representation
	// for Arrow's in memory buffer.
	data := make([]byte, rdr.ValuesWritten()*int(bitutil.BytesForBits(int64(dt.(arrow.FixedWidthDataType).BitWidth()))))

	length := rdr.ValuesWritten()
	// copy the values semantically with the correct types
	switch rdr.Type() {
	case parquet.Types.Int32:
		values := arrow.Int32Traits.CastFromBytes(rdr.Values())[:length]
		transferIntegerValues(values, data, dt.ID())
	case parquet.Types.Int64:
		values := arrow.Int64Traits.CastFromBytes(rdr.Values())[:length]
		transferIntegerValues(values, data, dt.ID())
	}

	bitmap := rdr.ReleaseValidBits()
	if bitmap != nil {
		defer bitmap.Release()
	}

	return array.NewData(dt, rdr.ValuesWritten(), []*memory.Buffer{
		bitmap, memory.NewBufferBytes(data),
	}, nil, int(rdr.NullCount()), 0)
}

func transferBool(rdr file.RecordReader) arrow.ArrayData {
	length := rdr.ValuesWritten()
	bitmap := rdr.ReleaseValidBits()
	if bitmap != nil {
		defer bitmap.Release()
	}
	var values *memory.Buffer
	if boolReader, ok := rdr.(file.BooleanRecordReader); ok {
		values = boolReader.ReleaseValueBitmap()
	} else {
		// Keep the bridge compatible with RecordReader implementations that do
		// not expose the packed Boolean fast path.
		data := make([]byte, int(bitutil.BytesForBits(int64(length))))
		for idx, value := range rdr.Values()[:length] {
			if value != 0 {
				bitutil.SetBit(data, idx)
			}
		}
		values = memory.NewBufferBytes(data)
	}
	defer values.Release()
	return array.NewData(&arrow.BooleanType{}, length, []*memory.Buffer{
		bitmap, values,
	}, nil, int(rdr.NullCount()), 0)
}

var milliPerDay = time.Duration(24 * time.Hour).Milliseconds()

// parquet equivalent for date64 is a 32-bit integer of the number of days
// since the epoch. Convert each value to milliseconds for date64
func transferDate64(rdr file.RecordReader, dt arrow.DataType) arrow.ArrayData {
	length := rdr.ValuesWritten()
	values := arrow.Int32Traits.CastFromBytes(rdr.Values())

	data := make([]byte, arrow.Int64Traits.BytesRequired(length))
	out := arrow.Int64Traits.CastFromBytes(data)
	for idx, val := range values[:length] {
		out[idx] = int64(val) * milliPerDay
	}

	bitmap := rdr.ReleaseValidBits()
	if bitmap != nil {
		defer bitmap.Release()
	}
	return array.NewData(dt, length, []*memory.Buffer{
		bitmap, memory.NewBufferBytes(data),
	}, nil, int(rdr.NullCount()), 0)
}

// coerce int96 to nanosecond timestamp
func transferInt96(rdr file.RecordReader, dt arrow.DataType) (arrow.ArrayData, error) {
	length := rdr.ValuesWritten()
	values := parquet.Int96Traits.CastFromBytes(rdr.Values())

	bitmap := rdr.ReleaseValidBits()
	if bitmap != nil {
		defer bitmap.Release()
	}

	data := make([]byte, arrow.Int64SizeBytes*length)
	out := arrow.Int64Traits.CastFromBytes(data)

	for idx, val := range values[:length] {
		if bitmap == nil || bitutil.BitIsSet(bitmap.Bytes(), idx) {
			timestamp, err := val.ToTimestamp()
			if err != nil {
				return nil, fmt.Errorf("parquet INT96 timestamp at index %d: %w", idx, err)
			}
			out[idx] = int64(timestamp)
		}
	}

	return array.NewData(dt, length, []*memory.Buffer{
		bitmap, memory.NewBufferBytes(data),
	}, nil, int(rdr.NullCount()), 0), nil
}

// convert physical integer storage of a decimal logical type to a decimal128 typed array
func transferDecimalIntegerValues[In parquetInteger](values []In, dt arrow.Type) []byte {
	switch dt {
	case arrow.DECIMAL128:
		data := make([]byte, arrow.Decimal128Traits.BytesRequired(len(values)))
		out := arrow.Decimal128Traits.CastFromBytes(data)
		for i, value := range values {
			out[i] = decimal128.FromI64(int64(value))
		}
		return data
	case arrow.DECIMAL256:
		data := make([]byte, arrow.Decimal256Traits.BytesRequired(len(values)))
		out := arrow.Decimal256Traits.CastFromBytes(data)
		for i, value := range values {
			out[i] = decimal256.FromI64(int64(value))
		}
		return data
	}
	return nil
}

func transferDecimalInteger(rdr file.RecordReader, dt arrow.DataType) arrow.ArrayData {
	length := rdr.ValuesWritten()

	var data []byte
	switch rdr.Type() {
	case parquet.Types.Int32:
		values := arrow.Int32Traits.CastFromBytes(rdr.Values())[:length]
		data = transferDecimalIntegerValues(values, dt.ID())
	case parquet.Types.Int64:
		values := arrow.Int64Traits.CastFromBytes(rdr.Values())[:length]
		data = transferDecimalIntegerValues(values, dt.ID())
	}

	var nullmap *memory.Buffer
	if rdr.NullCount() > 0 {
		nullmap = rdr.ReleaseValidBits()
		defer nullmap.Release()
	}
	return array.NewData(dt, length, []*memory.Buffer{
		nullmap, memory.NewBufferBytes(data),
	}, nil, int(rdr.NullCount()), 0)
}

func uint64FromBigEndianShifted(buf []byte) uint64 {
	var bytes [8]byte
	copy(bytes[8-len(buf):], buf)
	return binary.BigEndian.Uint64(bytes[:])
}

// parquet's defined encoding for decimal data is for it to be written as big
// endian bytes, so convert a bit endian byte order to a decimal128
func bigEndianToDecimal128(buf []byte) (decimal128.Num, error) {
	const (
		minDecimalBytes = 1
		maxDecimalBytes = 16
	)

	if len(buf) < minDecimalBytes || len(buf) > maxDecimalBytes {
		return decimal128.Num{}, fmt.Errorf("length of byte array passed to bigEndianToDecimal128 was %d but must be between %d and %d",
			len(buf), minDecimalBytes, maxDecimalBytes)
	}

	// bytes are big endian so first byte is MSB and holds the sign bit
	isNeg := int8(buf[0]) < 0

	// 1. extract high bits
	highBitsOffset := utils.Max(0, len(buf)-8)
	var (
		highBits uint64
		lowBits  uint64
		hi       int64
		lo       int64
	)
	highBits = uint64FromBigEndianShifted(buf[:highBitsOffset])

	if highBitsOffset == 8 {
		hi = int64(highBits)
	} else {
		if isNeg && len(buf) < maxDecimalBytes {
			hi = -1
		}

		hi = int64(uint64(hi) << (uint64(highBitsOffset) * 8))
		hi |= int64(highBits)
	}

	// 2. extract lower bits
	lowBitsOffset := utils.Min(len(buf), 8)
	lowBits = uint64FromBigEndianShifted(buf[highBitsOffset:])

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

func bigEndianToDecimal256(buf []byte) (decimal256.Num, error) {
	const (
		minDecimalBytes = 1
		maxDecimalBytes = 32
	)

	if len(buf) < minDecimalBytes || len(buf) > maxDecimalBytes {
		return decimal256.Num{},
			fmt.Errorf("%w: length of byte array for bigEndianToDecimal256 was %d but must be between %d and %d",
				arrow.ErrInvalid, len(buf), minDecimalBytes, maxDecimalBytes)
	}

	var littleEndian [4]uint64
	// bytes are coming in big-endian, so the first byte is the MSB and
	// therefore holds the sign bit
	initWord, isNeg := uint64(0), int8(buf[0]) < 0
	if isNeg {
		// sign extend if necessary
		initWord = uint64(0xFFFFFFFFFFFFFFFF)
	}

	for wordIdx := 0; wordIdx < 4; wordIdx++ {
		wordLen := utils.Min(len(buf), arrow.Uint64SizeBytes)
		word := buf[len(buf)-wordLen:]

		if wordLen == 8 {
			// full words can be assigned as-is
			littleEndian[wordIdx] = binary.BigEndian.Uint64(word)
		} else {
			result := initWord
			if len(buf) > 0 {
				// incorporate the actual values if present
				// shift left enough bits to make room for the incoming bytes
				result = result << uint64(wordLen*8)
				// preserve the upper bits by inplace OR-ing the int64
				result |= uint64FromBigEndianShifted(word)
			}
			littleEndian[wordIdx] = result
		}

		buf = buf[:len(buf)-wordLen]
	}

	return decimal256.New(littleEndian[3], littleEndian[2], littleEndian[1], littleEndian[0]), nil
}

type varOrFixedBin interface {
	arrow.Array
	Value(i int) []byte
}

// convert physical byte storage, instead of integers, to decimal128
func transferDecimalBytes(rdr file.BinaryRecordReader, dt arrow.DataType) (*arrow.Chunked, error) {
	convert128 := func(in varOrFixedBin) (arrow.Array, error) {
		length := in.Len()
		data := make([]byte, arrow.Decimal128Traits.BytesRequired(length))
		out := arrow.Decimal128Traits.CastFromBytes(data)

		nullCount := in.NullN()
		var err error
		for i := 0; i < length; i++ {
			if nullCount > 0 && in.IsNull(i) {
				continue
			}

			rec := in.Value(i)
			if len(rec) <= 0 {
				return nil, fmt.Errorf("invalid BYTEARRAY length for type: %s", dt)
			}
			out[i], err = bigEndianToDecimal128(rec)
			if err != nil {
				return nil, err
			}
		}

		ret := array.NewData(dt, length, []*memory.Buffer{
			in.Data().Buffers()[0], memory.NewBufferBytes(data),
		}, nil, nullCount, 0)
		defer ret.Release()
		return array.MakeFromData(ret), nil
	}

	convert256 := func(in varOrFixedBin) (arrow.Array, error) {
		length := in.Len()
		data := make([]byte, arrow.Decimal256Traits.BytesRequired(length))
		out := arrow.Decimal256Traits.CastFromBytes(data)

		nullCount := in.NullN()
		var err error
		for i := 0; i < length; i++ {
			if nullCount > 0 && in.IsNull(i) {
				continue
			}

			rec := in.Value(i)
			if len(rec) <= 0 {
				return nil, fmt.Errorf("invalid BYTEARRAY length for type: %s", dt)
			}
			out[i], err = bigEndianToDecimal256(rec)
			if err != nil {
				return nil, err
			}
		}

		ret := array.NewData(dt, length, []*memory.Buffer{
			in.Data().Buffers()[0], memory.NewBufferBytes(data),
		}, nil, nullCount, 0)
		defer ret.Release()
		return array.MakeFromData(ret), nil
	}

	convert := func(arr arrow.Array) (arrow.Array, error) {
		switch dt.ID() {
		case arrow.DECIMAL128:
			return convert128(arr.(varOrFixedBin))
		case arrow.DECIMAL256:
			return convert256(arr.(varOrFixedBin))
		}
		return nil, arrow.ErrNotImplemented
	}

	chunks := rdr.GetBuilderChunks()
	var err error
	for idx, chunk := range chunks {
		defer chunk.Release()
		if chunks[idx], err = convert(chunk); err != nil {
			return nil, err
		}
		defer chunks[idx].Release()
	}
	return arrow.NewChunked(dt, chunks), nil
}

func transferDictionary(rdr file.RecordReader, logicalValueType arrow.DataType, mem memory.Allocator) (*arrow.Chunked, error) {
	brdr := rdr.(file.BinaryRecordReader)
	chunks := brdr.GetBuilderChunks()
	defer releaseArrays(chunks)

	dictType, ok := logicalValueType.(*arrow.DictionaryType)
	if !ok || dictType.IndexType.ID() == arrow.INT32 {
		return arrow.NewChunked(logicalValueType, chunks), nil
	}

	ctx := compute.WithAllocator(context.Background(), mem)
	for idx, chunk := range chunks {
		dictArr, ok := chunk.(*array.Dictionary)
		if !ok {
			return nil, fmt.Errorf("expected dictionary array, got %T", chunk)
		}

		indices, err := compute.CastArray(ctx, dictArr.Indices(), compute.SafeCastOptions(dictType.IndexType))
		if err != nil {
			return nil, err
		}
		converted, err := array.NewValidatedDictionaryArray(dictType, indices, dictArr.Dictionary())
		indices.Release()
		if err != nil {
			return nil, err
		}
		chunk.Release()
		chunks[idx] = converted
	}
	return arrow.NewChunked(logicalValueType, chunks), nil
}
