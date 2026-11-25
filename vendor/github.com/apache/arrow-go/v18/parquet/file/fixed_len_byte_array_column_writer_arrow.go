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

type fixedLenByteArrayArrowEncoder interface {
	PutArrow([]byte)
	PutArrowSpaced([]byte, []byte, int64)
}

// SupportsArrowValues reports whether the active fixed-length byte-array
// encoder and configured batch size support writing an Arrow value buffer directly.
func (w *FixedLenByteArrayColumnChunkWriter) SupportsArrowValues() bool {
	if _, ok := w.currentEncoder.(fixedLenByteArrayArrowEncoder); !ok {
		return false
	}
	typeLen := int64(w.descr.TypeLength())
	batchSize := w.props.WriteBatchSize()
	const maxSafeBatchDataSize int64 = 1 << 30
	if typeLen <= 0 || batchSize <= 0 || batchSize > max(1, maxSafeBatchDataSize/(typeLen+4)) {
		return false
	}
	if w.pageStatistics == nil {
		return true
	}
	_, ok := w.pageStatistics.(*metadata.FixedLenByteArrayStatistics)
	return ok
}

func (w *FixedLenByteArrayColumnChunkWriter) writeArrowValues(values []byte, byteWidth int, numNulls int64) {
	w.currentEncoder.(fixedLenByteArrayArrowEncoder).PutArrow(values)
	if stats, ok := w.pageStatistics.(*metadata.FixedLenByteArrayStatistics); ok {
		stats.UpdateFromArrowFixedWidth(values, byteWidth, numNulls)
	}
	if w.bloomFilter != nil && w.currentEncoder.Encoding() != parquet.Encodings.PlainDict {
		metadata.InsertArrowFixedLenHashes(w.bloomFilter, values, byteWidth)
	}
}

func (w *FixedLenByteArrayColumnChunkWriter) writeArrowValuesSpaced(values []byte, byteWidth int, numRead, numValues int64, validBits []byte, validBitsOffset int64) {
	enc := w.currentEncoder.(fixedLenByteArrayArrowEncoder)
	numSpaced := int64(len(values) / byteWidth)
	if numSpaced == numRead {
		enc.PutArrow(values)
	} else {
		enc.PutArrowSpaced(values, validBits, validBitsOffset)
	}

	if stats, ok := w.pageStatistics.(*metadata.FixedLenByteArrayStatistics); ok {
		stats.UpdateFromArrowFixedWidthSpaced(values, byteWidth, validBits, validBitsOffset, numSpaced-numRead)
		stats.IncNulls(numValues - numSpaced)
	}
	if w.bloomFilter != nil && w.currentEncoder.Encoding() != parquet.Encodings.PlainDict {
		metadata.InsertSpacedArrowFixedLenHashes(w.bloomFilter, numRead, values, byteWidth, validBits, validBitsOffset)
	}
}

// WriteBatchArrow writes fixed-length byte-array values directly from an Arrow
// value buffer. The buffer contains typeLength bytes per value.
func (w *FixedLenByteArrayColumnChunkWriter) WriteBatchArrow(values []byte, defLevels, repLevels []int16) (valueOffset int64, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = utils.FormatRecoveredError("unknown error type", r)
		}
	}()
	if !w.SupportsArrowValues() {
		return 0, fmt.Errorf("parquet: current fixed-length byte-array encoder does not support Arrow values")
	}

	typeLen := int(w.descr.TypeLength())
	if typeLen <= 0 || len(values)%typeLen != 0 {
		return 0, fmt.Errorf("parquet: Arrow fixed-length values are not aligned to the type length")
	}
	length := len(values) / typeLen
	if defLevels != nil {
		length = len(defLevels)
	}
	if length == 0 {
		return 0, nil
	}

	w.doBatches(int64(length), repLevels, func(offset, batch int64) {
		toWrite := w.writeLevels(batch, levelSliceOrNil(defLevels, offset, batch), levelSliceOrNil(repLevels, offset, batch))
		start := int(valueOffset) * typeLen
		end := int(valueOffset+toWrite) * typeLen
		w.writeArrowValues(values[start:end], typeLen, batch-toWrite)
		if err := w.commitWriteAndCheckPageLimit(batch, toWrite); err != nil {
			panic(err)
		}
		valueOffset += toWrite
		w.checkDictionarySizeLimit()
	})
	return valueOffset, nil
}

// WriteBatchSpacedArrow writes fixed-length byte-array values directly from an
// Arrow value buffer while using validBits to skip null values.
func (w *FixedLenByteArrayColumnChunkWriter) WriteBatchSpacedArrow(values []byte, defLevels, repLevels []int16, validBits []byte, validBitsOffset int64) (valueOffset int64, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = utils.FormatRecoveredError("unknown error type", r)
		}
	}()
	if !w.SupportsArrowValues() {
		return 0, fmt.Errorf("parquet: current fixed-length byte-array encoder does not support Arrow values")
	}

	typeLen := int(w.descr.TypeLength())
	if typeLen <= 0 || len(values)%typeLen != 0 {
		return 0, fmt.Errorf("parquet: Arrow fixed-length values are not aligned to the type length")
	}
	length := len(defLevels)
	if defLevels == nil {
		length = len(values) / typeLen
	}
	if length == 0 {
		return 0, nil
	}

	w.doBatches(int64(length), repLevels, func(offset, batch int64) {
		info := w.maybeCalculateValidityBits(levelSliceOrNil(defLevels, offset, batch), batch)
		w.writeLevelsSpaced(batch, levelSliceOrNil(defLevels, offset, batch), levelSliceOrNil(repLevels, offset, batch))

		start := int(valueOffset) * typeLen
		end := int(valueOffset+info.numSpaced()) * typeLen
		writeBits := validBits
		writeBitsOffset := validBitsOffset + valueOffset
		if w.bitsBuffer != nil {
			writeBits = w.bitsBuffer.Bytes()
			writeBitsOffset = 0
		}
		w.writeArrowValuesSpaced(values[start:end], typeLen, info.batchNum, batch, writeBits, writeBitsOffset)
		if err := w.commitWriteAndCheckPageLimit(batch, info.numSpaced()); err != nil {
			panic(err)
		}
		valueOffset += info.numSpaced()
		w.checkDictionarySizeLimit()
	})
	return valueOffset, nil
}
