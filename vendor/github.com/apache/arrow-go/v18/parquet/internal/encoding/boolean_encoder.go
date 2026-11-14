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

	"github.com/apache/arrow-go/v18/arrow/bitutil"
	"github.com/apache/arrow-go/v18/internal/bitutils"
	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/internal/debug"
	"github.com/apache/arrow-go/v18/parquet/internal/utils"
)

const (
	boolBufSize          = 1024
	boolsInBuf           = boolBufSize * 8
	scalarBitmapRunLimit = 32
)

// PlainBooleanEncoder encodes bools as a bitmap as per the Plain Encoding
type PlainBooleanEncoder struct {
	encoder
	bitsBuffer    []byte
	spacedScratch []bool
	wr            utils.BitmapWriter
}

// Type for the PlainBooleanEncoder is parquet.Types.Boolean
func (PlainBooleanEncoder) Type() parquet.Type {
	return parquet.Types.Boolean
}

// Put encodes the contents of in into the underlying data buffer.
func (enc *PlainBooleanEncoder) Put(in []bool) {
	enc.initBitmapWriter()
	if len(in) == 0 {
		return
	}

	n := enc.wr.AppendBools(in)
	for n < len(in) {
		enc.wr.Finish()
		enc.append(enc.bitsBuffer)
		enc.wr.Reset(0, boolsInBuf)
		in = in[n:]
		n = enc.wr.AppendBools(in)
	}
}

func (enc *PlainBooleanEncoder) initBitmapWriter() {
	if enc.bitsBuffer == nil {
		enc.bitsBuffer = make([]byte, boolBufSize)
	}
	if enc.wr == nil {
		enc.wr = utils.NewBitmapWriter(enc.bitsBuffer, 0, boolsInBuf)
	}
}

// PutBitmap encodes boolean values directly from a bitmap without converting to []bool.
// This avoids the 8x memory overhead of bool slices.
func (enc *PlainBooleanEncoder) PutBitmap(bitmap []byte, offset int64, length int64) {
	enc.initBitmapWriter()
	if length == 0 {
		return
	}

	for length > 0 {
		n := enc.wr.AppendBitmap(bitmap, offset, length)
		offset += n
		length -= n

		if length > 0 {
			// Buffer is full, flush it
			enc.wr.Finish()
			enc.append(enc.bitsBuffer)
			enc.wr.Reset(0, boolsInBuf)
		}
	}
}

func (enc *PlainBooleanEncoder) putBitmapScalar(bitmap []byte, offset, length int64) {
	enc.initBitmapWriter()
	for i := int64(0); i < length; i++ {
		if enc.wr.Pos() == boolsInBuf {
			enc.wr.Finish()
			enc.append(enc.bitsBuffer)
			enc.wr.Reset(0, boolsInBuf)
		}
		if bitutil.BitIsSet(bitmap, int(offset+i)) {
			enc.wr.Set()
		} else {
			enc.wr.Clear()
		}
		enc.wr.Next()
	}
	enc.wr.Finish()
}

// PutSpaced will use the validBits bitmap to determine which values are nulls
// and can be left out from the slice, and the encoded without those nulls.
func (enc *PlainBooleanEncoder) PutSpaced(in []bool, validBits []byte, validBitsOffset int64) {
	if cap(enc.spacedScratch) < len(in) {
		enc.spacedScratch = make([]bool, len(in))
	} else {
		enc.spacedScratch = enc.spacedScratch[:len(in)]
	}
	nvalid := spacedCompress(in, enc.spacedScratch, validBits, validBitsOffset)
	enc.Put(enc.spacedScratch[:nvalid])
}

// PutSpacedBitmap encodes boolean values directly from a bitmap with validity information,
// without converting to []bool. This avoids the 8x memory overhead of bool slices.
// It appends valid runs directly to the encoder bitmap.
func (enc *PlainBooleanEncoder) PutSpacedBitmap(bitmap []byte, bitmapOffset int64, numValues int64, validBits []byte, validBitsOffset int64) int64 {
	if numValues == 0 {
		return 0
	}

	numValid := int64(0)
	reader := bitutils.NewSetBitRunReader(validBits, validBitsOffset, numValues)
	for {
		run := reader.NextRun()
		if run.Length == 0 {
			break
		}
		// Bitmap copies need word readers for unaligned ranges. Keep the
		// scalar path for short runs unless the run is an exact byte multiple:
		// when both ranges are byte-aligned, PutBitmap uses a cheaper byte copy.
		switch run.Length {
		case 8, 16, 24:
			srcOffset := bitmapOffset + run.Pos
			dstOffset := int64(0)
			if enc.wr != nil {
				dstOffset = int64(enc.wr.Pos())
			}
			if srcOffset%8 != 0 || dstOffset%8 != 0 {
				enc.putBitmapScalar(bitmap, srcOffset, run.Length)
			} else {
				enc.PutBitmap(bitmap, srcOffset, run.Length)
			}
		default:
			if run.Length < scalarBitmapRunLimit {
				enc.putBitmapScalar(bitmap, bitmapOffset+run.Pos, run.Length)
			} else {
				enc.PutBitmap(bitmap, bitmapOffset+run.Pos, run.Length)
			}
		}
		numValid += run.Length
	}

	return numValid
}

// EstimatedDataEncodedSize returns the current number of bytes that have
// been buffered so far
func (enc *PlainBooleanEncoder) EstimatedDataEncodedSize() int64 {
	return int64(enc.sink.Len() + int(bitutil.BytesForBits(int64(enc.wr.Pos()))))
}

// FlushValues returns the buffered data, the responsibility is on the caller
// to release the buffer memory
func (enc *PlainBooleanEncoder) FlushValues() (Buffer, error) {
	if enc.wr.Pos() > 0 {
		toFlush := int(enc.wr.Pos())
		enc.append(enc.bitsBuffer[:bitutil.BytesForBits(int64(toFlush))])
	}

	enc.wr.Reset(0, boolsInBuf)

	return enc.sink.Finish(), nil
}

const rleLengthInBytes = 4

type RleBooleanEncoder struct {
	encoder

	bufferedValues []bool
	spacedScratch  []bool
}

func (RleBooleanEncoder) Type() parquet.Type {
	return parquet.Types.Boolean
}

func (enc *RleBooleanEncoder) Put(in []bool) {
	enc.bufferedValues = append(enc.bufferedValues, in...)
}

func (enc *RleBooleanEncoder) PutSpaced(in []bool, validBits []byte, validBitsOffset int64) {
	if cap(enc.spacedScratch) < len(in) {
		enc.spacedScratch = make([]bool, len(in))
	} else {
		enc.spacedScratch = enc.spacedScratch[:len(in)]
	}
	nvalid := spacedCompress(in, enc.spacedScratch, validBits, validBitsOffset)
	enc.Put(enc.spacedScratch[:nvalid])
}

// PutSpacedBitmap encodes boolean values from a bitmap with validity information.
// Note: RleBooleanEncoder buffers values as []bool for RLE encoding at flush time,
// so we extract valid bits to []bool. This is still better than the caller doing
// the full bitmap-to-[]bool conversion for all values.
func (enc *RleBooleanEncoder) PutSpacedBitmap(bitmap []byte, bitmapOffset int64, numValues int64, validBits []byte, validBitsOffset int64) int64 {
	if numValues == 0 {
		return 0
	}

	// Count valid values
	numValid := int64(bitutil.CountSetBits(validBits, int(validBitsOffset), int(numValues)))
	if numValid == 0 {
		return 0
	}

	// Extract valid bits to []bool for buffering
	// Use SetBitRunReader to efficiently iterate over valid values
	nvalid := int(numValid)
	if cap(enc.spacedScratch) < nvalid {
		enc.spacedScratch = make([]bool, nvalid)
	} else {
		enc.spacedScratch = enc.spacedScratch[:nvalid]
	}
	idx := 0

	reader := bitutils.NewSetBitRunReader(validBits, validBitsOffset, numValues)
	for {
		run := reader.NextRun()
		if run.Length == 0 {
			break
		}

		// Convert this run of bits to bools
		for i := int64(0); i < run.Length; i++ {
			enc.spacedScratch[idx] = bitutil.BitIsSet(bitmap, int(bitmapOffset+run.Pos+i))
			idx++
		}
	}

	enc.Put(enc.spacedScratch[:idx])
	return numValid
}

func (enc *RleBooleanEncoder) EstimatedDataEncodedSize() int64 {
	return rleLengthInBytes + int64(enc.maxRleBufferSize())
}

func (enc *RleBooleanEncoder) maxRleBufferSize() int {
	return utils.MaxRLEBufferSize(1, len(enc.bufferedValues)) +
		utils.MinRLEBufferSize(1)
}

func (enc *RleBooleanEncoder) FlushValues() (Buffer, error) {
	rleBufferSizeMax := enc.maxRleBufferSize()
	enc.sink.SetOffset(rleLengthInBytes)
	enc.sink.Reserve(rleBufferSizeMax)

	rleEncoder := utils.NewRleEncoder(enc.sink, 1)
	for _, v := range enc.bufferedValues {
		if v {
			rleEncoder.Put(1)
		} else {
			rleEncoder.Put(0)
		}
	}
	n := rleEncoder.Flush()
	debug.Assert(n <= rleBufferSizeMax, "num encoded bytes larger than expected max")
	buf := enc.sink.Finish()
	binary.LittleEndian.PutUint32(buf.Bytes(), uint32(n))

	return buf, nil
}
