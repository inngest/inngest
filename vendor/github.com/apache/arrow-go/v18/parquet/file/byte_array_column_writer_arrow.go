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

package file

import (
	"fmt"

	"github.com/apache/arrow-go/v18/internal/utils"
	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/metadata"
)

type byteArrayArrowOffset interface {
	~int32 | ~int64
}

type byteArrayArrowEncoder32 interface {
	PutArrow([]byte, []int32)
	PutArrowSpaced([]byte, []int32, []byte, int64)
}

type byteArrayArrowEncoder64 interface {
	PutArrow64([]byte, []int64)
	PutArrowSpaced64([]byte, []int64, []byte, int64)
}

func arrowOffsetsForBatch[T byteArrayArrowOffset](offsets []T, offset, count int64) []T {
	if offset < 0 || offset > int64(len(offsets)) {
		panic("parquet: Arrow offset index is out of bounds")
	}
	if count == 0 {
		if offset == int64(len(offsets)) {
			return nil
		}
		return offsets[offset : offset+1]
	}

	end := offset + count + 1
	if end > int64(len(offsets)) {
		panic("parquet: Arrow offset range is out of bounds")
	}
	return offsets[offset:end]
}

// SupportsArrowOffsets reports whether the active byte-array encoder can consume
// Arrow's value buffer and offsets without materializing parquet.ByteArray values.
func (w *ByteArrayColumnChunkWriter) SupportsArrowOffsets() bool {
	_, supports32 := w.currentEncoder.(byteArrayArrowEncoder32)
	_, supports64 := w.currentEncoder.(byteArrayArrowEncoder64)
	return supports32 || supports64
}

func writeArrowValues[T byteArrayArrowOffset](w *ByteArrayColumnChunkWriter, values []byte, offsets []T, numNulls int64) {
	switch offsets := any(offsets).(type) {
	case []int32:
		enc, ok := w.currentEncoder.(byteArrayArrowEncoder32)
		if !ok {
			panic("parquet: current byte-array encoder does not support Arrow offsets")
		}
		enc.PutArrow(values, offsets)
		if w.pageStatistics != nil {
			w.pageStatistics.(*metadata.ByteArrayStatistics).UpdateFromArrowOffsets(values, offsets, numNulls)
		}
		if w.bloomFilter != nil && w.currentEncoder.Encoding() != parquet.Encodings.PlainDict {
			metadata.InsertArrowOffsetHashes(w.bloomFilter, values, offsets)
		}
	case []int64:
		enc, ok := w.currentEncoder.(byteArrayArrowEncoder64)
		if !ok {
			panic("parquet: current byte-array encoder does not support Arrow offsets")
		}
		enc.PutArrow64(values, offsets)
		if w.pageStatistics != nil {
			w.pageStatistics.(*metadata.ByteArrayStatistics).UpdateFromArrowOffsets64(values, offsets, numNulls)
		}
		if w.bloomFilter != nil && w.currentEncoder.Encoding() != parquet.Encodings.PlainDict {
			metadata.InsertArrowOffsetHashes64(w.bloomFilter, values, offsets)
		}
	}
}

func writeArrowValuesSpaced[T byteArrayArrowOffset](w *ByteArrayColumnChunkWriter, values []byte, offsets []T, numRead, numValues int64, validBits []byte, validBitsOffset int64) {
	numSpaced := int64(0)
	if len(offsets) > 0 {
		numSpaced = int64(len(offsets) - 1)
	}

	switch offsets := any(offsets).(type) {
	case []int32:
		enc, ok := w.currentEncoder.(byteArrayArrowEncoder32)
		if !ok {
			panic("parquet: current byte-array encoder does not support Arrow offsets")
		}
		if numSpaced != numRead {
			enc.PutArrowSpaced(values, offsets, validBits, validBitsOffset)
		} else {
			enc.PutArrow(values, offsets)
		}
		if w.pageStatistics != nil {
			stats := w.pageStatistics.(*metadata.ByteArrayStatistics)
			stats.UpdateFromArrowOffsetsSpaced(values, offsets, validBits, validBitsOffset, numSpaced-numRead)
			stats.IncNulls(numValues - numSpaced)
		}
		if w.bloomFilter != nil && w.currentEncoder.Encoding() != parquet.Encodings.PlainDict {
			metadata.InsertSpacedArrowOffsetHashes(w.bloomFilter, numRead, values, offsets, validBits, validBitsOffset)
		}
	case []int64:
		enc, ok := w.currentEncoder.(byteArrayArrowEncoder64)
		if !ok {
			panic("parquet: current byte-array encoder does not support Arrow offsets")
		}
		if numSpaced != numRead {
			enc.PutArrowSpaced64(values, offsets, validBits, validBitsOffset)
		} else {
			enc.PutArrow64(values, offsets)
		}
		if w.pageStatistics != nil {
			stats := w.pageStatistics.(*metadata.ByteArrayStatistics)
			stats.UpdateFromArrowOffsetsSpaced64(values, offsets, validBits, validBitsOffset, numSpaced-numRead)
			stats.IncNulls(numValues - numSpaced)
		}
		if w.bloomFilter != nil && w.currentEncoder.Encoding() != parquet.Encodings.PlainDict {
			metadata.InsertSpacedArrowOffsetHashes64(w.bloomFilter, numRead, values, offsets, validBits, validBitsOffset)
		}
	}
}

func writeBatchArrow[T byteArrayArrowOffset](w *ByteArrayColumnChunkWriter, values []byte, offsets []T, defLevels, repLevels []int16) (valueOffset int64) {
	var n int64
	if defLevels != nil {
		n = int64(len(defLevels))
	} else if len(offsets) > 0 {
		n = int64(len(offsets) - 1)
	}
	if n == 0 {
		return 0
	}

	const maxSafeBatchDataSize int64 = 1 << 30
	batchSize := w.props.WriteBatchSize()
	maxDefLevel := w.descr.MaxDefinitionLevel()
	requiresRowAlignment := (w.props.DataPageVersion() != parquet.DataPageV1 ||
		w.props.PageIndexEnabledFor(w.descr.Path())) &&
		repLevels != nil && w.descr.MaxRepetitionLevel() > 0
	levelOffset := int64(0)
	valueCount := int64(0)
	if len(offsets) > 0 {
		valueCount = int64(len(offsets) - 1)
	}

	if requiresRowAlignment {
		if int64(len(repLevels)) < n {
			panic("columnwriter: not enough repetition levels for batch to write")
		}
		if repLevels[0] != 0 {
			panic("columnwriter: row-aligned batch writing must start at a row boundary")
		}
		repLevels = repLevels[:n]
	}

	for levelOffset < n {
		remaining := n - levelOffset
		batch := min(remaining, batchSize)

		var cumDataSize int64
		valueScan := valueOffset
		for li := int64(0); li < batch; li++ {
			isValue := defLevels == nil || maxDefLevel == 0 || defLevels[levelOffset+li] == maxDefLevel
			if isValue && valueScan < valueCount {
				valueSize := int64(offsets[valueScan+1]) - int64(offsets[valueScan]) + 4
				if cumDataSize+valueSize > maxSafeBatchDataSize && li > 0 {
					batch = li
					break
				}
				cumDataSize += valueSize
				valueScan++
			}
		}

		if requiresRowAlignment {
			batch = alignBatchToRowBoundary(repLevels, levelOffset, batch)
		}
		if batch < 1 {
			batch = 1
		}

		toWrite := w.writeLevels(batch, levelSliceOrNil(defLevels, levelOffset, batch),
			levelSliceOrNil(repLevels, levelOffset, batch))
		batchOffsets := arrowOffsetsForBatch(offsets, valueOffset, toWrite)
		writeArrowValues(w, values, batchOffsets, batch-toWrite)
		if err := w.commitWriteAndCheckPageLimit(batch, toWrite); err != nil {
			panic(err)
		}
		valueOffset += toWrite
		w.checkDictionarySizeLimit()
		levelOffset += batch
	}
	return valueOffset
}

func writeBatchSpacedArrow[T byteArrayArrowOffset](w *ByteArrayColumnChunkWriter, values []byte, offsets []T, defLevels, repLevels []int16, validBits []byte, validBitsOffset int64) (valueOffset int64) {
	length := len(defLevels)
	if defLevels == nil {
		if len(offsets) > 0 {
			length = len(offsets) - 1
		} else {
			length = 0
		}
	}
	if length == 0 {
		return 0
	}

	const maxSafeBatchDataSize int64 = 1 << 30
	batchSize := w.props.WriteBatchSize()
	requiresRowAlignment := (w.props.DataPageVersion() != parquet.DataPageV1 ||
		w.props.PageIndexEnabledFor(w.descr.Path())) &&
		repLevels != nil && w.descr.MaxRepetitionLevel() > 0
	levelOffset := int64(0)
	n := int64(length)
	valueCount := int64(0)
	if len(offsets) > 0 {
		valueCount = int64(len(offsets) - 1)
	}

	if requiresRowAlignment {
		if int64(len(repLevels)) < n {
			panic("columnwriter: not enough repetition levels for batch to write")
		}
		if repLevels[0] != 0 {
			panic("columnwriter: row-aligned batch writing must start at a row boundary")
		}
		repLevels = repLevels[:n]
	}

	for levelOffset < n {
		remaining := n - levelOffset
		batch := min(remaining, batchSize)

		var cumDataSize int64
		for vi := int64(0); vi < batch && valueOffset+vi < valueCount; vi++ {
			valueSize := int64(offsets[valueOffset+vi+1]) - int64(offsets[valueOffset+vi]) + 4
			if cumDataSize+valueSize > maxSafeBatchDataSize && vi > 0 {
				batch = vi
				break
			}
			cumDataSize += valueSize
		}

		if requiresRowAlignment {
			batch = alignBatchToRowBoundary(repLevels, levelOffset, batch)
		}
		if batch < 1 {
			batch = 1
		}

		info := w.maybeCalculateValidityBits(levelSliceOrNil(defLevels, levelOffset, batch), batch)
		w.writeLevelsSpaced(batch, levelSliceOrNil(defLevels, levelOffset, batch),
			levelSliceOrNil(repLevels, levelOffset, batch))
		batchOffsets := arrowOffsetsForBatch(offsets, valueOffset, info.numSpaced())

		writeBits := validBits
		writeBitsOffset := validBitsOffset + valueOffset
		if w.bitsBuffer != nil {
			writeBits = w.bitsBuffer.Bytes()
			writeBitsOffset = 0
		}
		writeArrowValuesSpaced(w, values, batchOffsets, info.batchNum, batch, writeBits, writeBitsOffset)
		if err := w.commitWriteAndCheckPageLimit(batch, info.numSpaced()); err != nil {
			panic(err)
		}
		valueOffset += info.numSpaced()
		w.checkDictionarySizeLimit()
		levelOffset += batch
	}
	return valueOffset
}

// WriteBatchArrow writes Arrow binary data using its value buffer and 32-bit offsets.
// The active encoder must support Arrow offsets; dictionary encoders use the existing
// parquet.ByteArray path instead.
func (w *ByteArrayColumnChunkWriter) WriteBatchArrow(values []byte, offsets []int32, defLevels, repLevels []int16) (valueOffset int64, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = utils.FormatRecoveredError("unknown error type", r)
		}
	}()
	if _, ok := w.currentEncoder.(byteArrayArrowEncoder32); !ok {
		return 0, fmt.Errorf("parquet: current byte-array encoder does not support 32-bit Arrow offsets")
	}
	return writeBatchArrow(w, values, offsets, defLevels, repLevels), nil
}

// WriteBatchArrow64 writes Arrow binary data using its value buffer and 64-bit offsets.
func (w *ByteArrayColumnChunkWriter) WriteBatchArrow64(values []byte, offsets []int64, defLevels, repLevels []int16) (valueOffset int64, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = utils.FormatRecoveredError("unknown error type", r)
		}
	}()
	if _, ok := w.currentEncoder.(byteArrayArrowEncoder64); !ok {
		return 0, fmt.Errorf("parquet: current byte-array encoder does not support 64-bit Arrow offsets")
	}
	return writeBatchArrow(w, values, offsets, defLevels, repLevels), nil
}

// WriteBatchSpacedArrow writes spaced Arrow binary data using 32-bit offsets.
func (w *ByteArrayColumnChunkWriter) WriteBatchSpacedArrow(values []byte, offsets []int32, defLevels, repLevels []int16, validBits []byte, validBitsOffset int64) (valueOffset int64, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = utils.FormatRecoveredError("unknown error type", r)
		}
	}()
	if _, ok := w.currentEncoder.(byteArrayArrowEncoder32); !ok {
		return 0, fmt.Errorf("parquet: current byte-array encoder does not support 32-bit Arrow offsets")
	}
	return writeBatchSpacedArrow(w, values, offsets, defLevels, repLevels, validBits, validBitsOffset), nil
}

// WriteBatchSpacedArrow64 writes spaced Arrow binary data using 64-bit offsets.
func (w *ByteArrayColumnChunkWriter) WriteBatchSpacedArrow64(values []byte, offsets []int64, defLevels, repLevels []int16, validBits []byte, validBitsOffset int64) (valueOffset int64, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = utils.FormatRecoveredError("unknown error type", r)
		}
	}()
	if _, ok := w.currentEncoder.(byteArrayArrowEncoder64); !ok {
		return 0, fmt.Errorf("parquet: current byte-array encoder does not support 64-bit Arrow offsets")
	}
	return writeBatchSpacedArrow(w, values, offsets, defLevels, repLevels, validBits, validBitsOffset), nil
}
