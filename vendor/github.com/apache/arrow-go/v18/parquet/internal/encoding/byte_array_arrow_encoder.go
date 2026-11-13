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

package encoding

import (
	"encoding/binary"

	"github.com/apache/arrow-go/v18/internal/bitutils"
	"github.com/apache/arrow-go/v18/parquet"
)

type arrowByteArrayOffset interface {
	~int32 | ~int64
}

func putArrowPlain[T arrowByteArrayOffset](sink *PooledBufferWriter, values []byte, offsets []T) {
	if len(offsets) < 2 {
		return
	}

	encodedSize := 0
	for i := 0; i < len(offsets)-1; i++ {
		encodedSize += int(offsets[i+1]-offsets[i]) + 4
	}

	sink.Reserve(encodedSize)
	out := sink.buf.Buf()[sink.pos : sink.pos+encodedSize]
	for i := 0; i < len(offsets)-1; i++ {
		start := int(offsets[i])
		end := int(offsets[i+1])
		binary.LittleEndian.PutUint32(out, uint32(end-start))
		copy(out[4:], values[start:end])
		out = out[4+end-start:]
	}
	sink.pos += encodedSize
}

func putArrowPlainSpaced[T arrowByteArrayOffset](sink *PooledBufferWriter, values []byte, offsets []T, validBits []byte, validBitsOffset int64) {
	if len(offsets) < 2 {
		return
	}
	if validBits == nil {
		putArrowPlain(sink, values, offsets)
		return
	}

	bitutils.VisitSetBitRunsNoErr(validBits, validBitsOffset, int64(len(offsets)-1), func(pos, length int64) {
		putArrowPlain(sink, values, offsets[pos:pos+length+1])
	})
}

func (enc *PlainByteArrayEncoder) PutArrow(values []byte, offsets []int32) {
	putArrowPlain(enc.sink, values, offsets)
}

func (enc *PlainByteArrayEncoder) PutArrow64(values []byte, offsets []int64) {
	putArrowPlain(enc.sink, values, offsets)
}

func (enc *PlainByteArrayEncoder) PutArrowSpaced(values []byte, offsets []int32, validBits []byte, validBitsOffset int64) {
	putArrowPlainSpaced(enc.sink, values, offsets, validBits, validBitsOffset)
}

func (enc *PlainByteArrayEncoder) PutArrowSpaced64(values []byte, offsets []int64, validBits []byte, validBitsOffset int64) {
	putArrowPlainSpaced(enc.sink, values, offsets, validBits, validBitsOffset)
}

func putArrowDeltaLength[T arrowByteArrayOffset](enc *DeltaLengthByteArrayEncoder, values []byte, offsets []T, validBits []byte, validBitsOffset int64) {
	if len(offsets) < 2 {
		return
	}

	if validBits == nil {
		batchSize := 0
		totalLen := 0
		for i := 0; i < len(offsets)-1; i++ {
			start := int(offsets[i])
			end := int(offsets[i+1])
			enc.lengths[batchSize] = int32(end - start)
			totalLen += end - start
			batchSize++
			if batchSize == len(enc.lengths) {
				enc.lengthEncoder.Put(enc.lengths[:batchSize])
				batchSize = 0
			}
		}
		if batchSize != 0 {
			enc.lengthEncoder.Put(enc.lengths[:batchSize])
		}

		enc.sink.Reserve(totalLen)
		for i := 0; i < len(offsets)-1; i++ {
			enc.sink.UnsafeWrite(values[int(offsets[i]):int(offsets[i+1])])
		}
		return
	}

	batchSize := 0
	totalLen := 0
	visit := func(pos, length int64) {
		for i := pos; i < pos+length; i++ {
			start := int(offsets[i])
			end := int(offsets[i+1])
			enc.lengths[batchSize] = int32(end - start)
			totalLen += end - start
			batchSize++
			if batchSize == len(enc.lengths) {
				enc.lengthEncoder.Put(enc.lengths[:batchSize])
				batchSize = 0
			}
		}
	}
	bitutils.VisitSetBitRunsNoErr(validBits, validBitsOffset, int64(len(offsets)-1), visit)
	if batchSize != 0 {
		enc.lengthEncoder.Put(enc.lengths[:batchSize])
	}

	enc.sink.Reserve(totalLen)
	bitutils.VisitSetBitRunsNoErr(validBits, validBitsOffset, int64(len(offsets)-1), func(pos, length int64) {
		for i := pos; i < pos+length; i++ {
			enc.sink.UnsafeWrite(values[int(offsets[i]):int(offsets[i+1])])
		}
	})
}

func (enc *DeltaLengthByteArrayEncoder) PutArrow(values []byte, offsets []int32) {
	putArrowDeltaLength(enc, values, offsets, nil, 0)
}

func (enc *DeltaLengthByteArrayEncoder) PutArrow64(values []byte, offsets []int64) {
	putArrowDeltaLength(enc, values, offsets, nil, 0)
}

func (enc *DeltaLengthByteArrayEncoder) PutArrowSpaced(values []byte, offsets []int32, validBits []byte, validBitsOffset int64) {
	putArrowDeltaLength(enc, values, offsets, validBits, validBitsOffset)
}

func (enc *DeltaLengthByteArrayEncoder) PutArrowSpaced64(values []byte, offsets []int64, validBits []byte, validBitsOffset int64) {
	putArrowDeltaLength(enc, values, offsets, validBits, validBitsOffset)
}

func putArrowDeltaByte[T arrowByteArrayOffset](enc *DeltaByteArrayEncoder, values []byte, offsets []T, validBits []byte, validBitsOffset int64) {
	if len(offsets) < 2 {
		return
	}

	if enc.prefixEncoder == nil {
		enc.initEncoders()
	}

	lastVal := enc.lastVal
	if validBits == nil {
		batchSize := 0
		for i := 0; i < len(offsets)-1; i++ {
			val := parquet.ByteArray(values[int(offsets[i]):int(offsets[i+1])])
			prefixLength := commonPrefixLength(lastVal, val)
			lastVal = val
			enc.prefixLengths[batchSize] = int32(prefixLength)
			enc.suffixes[batchSize] = val[prefixLength:]
			batchSize++
			if batchSize == len(enc.suffixes) {
				enc.suffixEncoder.Put(enc.suffixes[:batchSize])
				enc.prefixEncoder.Put(enc.prefixLengths[:batchSize])
				clear(enc.suffixes[:batchSize])
				batchSize = 0
			}
		}
		if batchSize != 0 {
			enc.suffixEncoder.Put(enc.suffixes[:batchSize])
			enc.prefixEncoder.Put(enc.prefixLengths[:batchSize])
			clear(enc.suffixes[:batchSize])
		}
		enc.lastVal = append(enc.lastVal[:0], lastVal...)
		return
	}

	batchSize := 0
	visit := func(pos, length int64) {
		for i := pos; i < pos+length; i++ {
			val := parquet.ByteArray(values[int(offsets[i]):int(offsets[i+1])])
			prefixLength := commonPrefixLength(lastVal, val)
			lastVal = val
			enc.prefixLengths[batchSize] = int32(prefixLength)
			enc.suffixes[batchSize] = val[prefixLength:]
			batchSize++
			if batchSize == len(enc.suffixes) {
				enc.suffixEncoder.Put(enc.suffixes[:batchSize])
				enc.prefixEncoder.Put(enc.prefixLengths[:batchSize])
				clear(enc.suffixes[:batchSize])
				batchSize = 0
			}
		}
	}
	bitutils.VisitSetBitRunsNoErr(validBits, validBitsOffset, int64(len(offsets)-1), visit)
	if batchSize != 0 {
		enc.suffixEncoder.Put(enc.suffixes[:batchSize])
		enc.prefixEncoder.Put(enc.prefixLengths[:batchSize])
		clear(enc.suffixes[:batchSize])
	}

	enc.lastVal = append(enc.lastVal[:0], lastVal...)
}

func (enc *DeltaByteArrayEncoder) PutArrow(values []byte, offsets []int32) {
	putArrowDeltaByte(enc, values, offsets, nil, 0)
}

func (enc *DeltaByteArrayEncoder) PutArrow64(values []byte, offsets []int64) {
	putArrowDeltaByte(enc, values, offsets, nil, 0)
}

func (enc *DeltaByteArrayEncoder) PutArrowSpaced(values []byte, offsets []int32, validBits []byte, validBitsOffset int64) {
	putArrowDeltaByte(enc, values, offsets, validBits, validBitsOffset)
}

func (enc *DeltaByteArrayEncoder) PutArrowSpaced64(values []byte, offsets []int64, validBits []byte, validBitsOffset int64) {
	putArrowDeltaByte(enc, values, offsets, validBits, validBitsOffset)
}
