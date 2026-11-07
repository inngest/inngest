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
	"errors"
	"fmt"
	"math"
	"unsafe"

	"github.com/apache/arrow-go/v18/internal/utils"
	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/internal/encoding/streaming"
)

// PlainFixedLenByteArrayDecoder is a plain encoding decoder for Fixed Length Byte Arrays
type PlainFixedLenByteArrayDecoder struct {
	decoder

	// src drives the streaming path (PageStreamingEnabled) when non-nil.
	src streaming.ValueBuffer
}

// Type returns the physical type this decoder operates on, FixedLength Byte Arrays
func (PlainFixedLenByteArrayDecoder) Type() parquet.Type {
	return parquet.Types.FixedLenByteArray
}

func (pflba *PlainFixedLenByteArrayDecoder) SetSource(nvals int, src streaming.ValueBuffer) {
	pflba.src, pflba.nvals = src, nvals
}

func (pflba *PlainFixedLenByteArrayDecoder) SetData(nvals int, data []byte) error {
	pflba.src = nil
	return pflba.decoder.SetData(nvals, data)
}

// decodeStreaming aliases each value in the buffer; see the byte-array decoder for the
// Recycle/alias lifetime.
func (pflba *PlainFixedLenByteArrayDecoder) decodeStreaming(out []parquet.FixedLenByteArray) (int, error) {
	pflba.src.Recycle()
	max := utils.Min(len(out), pflba.nvals)
	w := pflba.typeLen
	for i := 0; i < max; i++ {
		val, err := pflba.src.Fill(w)
		if err != nil {
			return i, errors.New("parquet: eof exception")
		}
		out[i] = val[:w:w]
		pflba.src.Advance(w)
	}
	pflba.nvals -= max
	return max, nil
}

func (pflba *PlainFixedLenByteArrayDecoder) discardStreaming(n int) (int, error) {
	n = min(n, pflba.nvals)
	pflba.src.Recycle()
	if err := pflba.src.Skip(n * pflba.typeLen); err != nil {
		return 0, errors.New("parquet: eof exception")
	}
	pflba.nvals -= n
	return n, nil
}

func (pflba *PlainFixedLenByteArrayDecoder) Discard(n int) (int, error) {
	if pflba.src != nil {
		return pflba.discardStreaming(n)
	}
	n = min(n, pflba.nvals)
	numBytesNeeded := n * pflba.typeLen
	if numBytesNeeded > len(pflba.data) || numBytesNeeded > math.MaxInt32 {
		return 0, errors.New("parquet: eof exception")
	}

	pflba.data = pflba.data[numBytesNeeded:]
	pflba.nvals -= n
	return n, nil
}

// Decode populates out with fixed length byte array values until either there are no more
// values to decode or the length of out has been filled. Then returns the total number of values
// that were decoded.
func (pflba *PlainFixedLenByteArrayDecoder) Decode(out []parquet.FixedLenByteArray) (int, error) {
	if pflba.src != nil {
		return pflba.decodeStreaming(out)
	}

	max := utils.Min(len(out), pflba.nvals)
	numBytesNeeded := max * pflba.typeLen
	if numBytesNeeded > len(pflba.data) || numBytesNeeded > math.MaxInt32 {
		return 0, errors.New("parquet: eof exception")
	}

	for idx := range out[:max] {
		out[idx] = pflba.data[:pflba.typeLen]
		pflba.data = pflba.data[pflba.typeLen:]
	}

	pflba.nvals -= max
	return max, nil
}

// DecodeSpaced does the same as Decode but spaces out the resulting slice according to the bitmap leaving space for null values
func (pflba *PlainFixedLenByteArrayDecoder) DecodeSpaced(out []parquet.FixedLenByteArray, nullCount int, validBits []byte, validBitsOffset int64) (int, error) {
	toRead := len(out) - nullCount
	valuesRead, err := pflba.Decode(out[:toRead])
	if err != nil {
		return valuesRead, err
	}
	if valuesRead != toRead {
		return valuesRead, errors.New("parquet: number of values / definitions levels read did not match")
	}

	// Keep every output slot independent because a later decoder may write through
	// these slices when this buffer is reused across pages.
	return spacedExpandSwap(out, nullCount, validBits, validBitsOffset), nil
}

// ByteStreamSplitFixedLenByteArrayDecoder is a decoder for BYTE_STREAM_SPLIT-encoded
// bytes representing FixedLenByteArray values
type ByteStreamSplitFixedLenByteArrayDecoder struct {
	decoder
	stride int

	// storage is the most recent block this decoder allocated for output values.
	// Decode may only write through an output slice backed by this block; see owns.
	storage []byte
}

func (dec *ByteStreamSplitFixedLenByteArrayDecoder) Type() parquet.Type {
	return parquet.Types.FixedLenByteArray
}

func (dec *ByteStreamSplitFixedLenByteArrayDecoder) SetData(nvals int, data []byte) error {
	encodedNvals, err := validateByteStreamSplitPageData(dec.typeLen, nvals, data)
	if err != nil {
		return err
	}
	if dec.descr != nil && dec.descr.MaxDefinitionLevel() == 0 && encodedNvals != nvals {
		return fmt.Errorf("BYTE_STREAM_SPLIT data contains %d values, expected %d", encodedNvals, nvals)
	}

	dec.stride = encodedNvals
	return dec.decoder.SetData(encodedNvals, data)
}

func (dec *ByteStreamSplitFixedLenByteArrayDecoder) Discard(n int) (int, error) {
	n = min(n, dec.nvals)
	numBytesNeeded := n * dec.typeLen
	if numBytesNeeded > len(dec.data) || numBytesNeeded > math.MaxInt32 {
		return 0, errors.New("parquet: eof exception")
	}

	dec.nvals -= n
	dec.data = dec.data[n:]
	return n, nil
}

func (dec *ByteStreamSplitFixedLenByteArrayDecoder) Decode(out []parquet.FixedLenByteArray) (int, error) {
	toRead := min(len(out), dec.nvals)
	numBytesNeeded := toRead * dec.typeLen
	if numBytesNeeded > len(dec.data) || numBytesNeeded > math.MaxInt32 {
		return 0, errors.New("parquet: eof exception")
	}

	out = out[:toRead]
	dec.prepareOutput(out)

	switch dec.typeLen {
	case 2:
		decodeByteStreamSplitBatchFLBAWidth2(dec.data, toRead, dec.stride, out)
	case 4:
		decodeByteStreamSplitBatchFLBAWidth4(dec.data, toRead, dec.stride, out)
	case 8:
		decodeByteStreamSplitBatchFLBAWidth8(dec.data, toRead, dec.stride, out)
	default:
		decodeByteStreamSplitBatchFLBA(dec.data, toRead, dec.stride, dec.typeLen, out)
	}

	dec.nvals -= toRead
	dec.data = dec.data[toRead:]
	return toRead, nil
}

// owns reports whether every entry of out is backed by the block this decoder most
// recently allocated, and so may be written through.
//
// The check matters because this decoder decodes in place: it writes bytes through the
// slice headers the caller hands it. Reusing a header whose memory belongs to something
// else corrupts that memory. Two earlier decoders leave such headers in a shared value
// buffer, neither of which requires nulls to do so:
//
//   - RLE_DICTIONARY assigns dict[idx] into every slot holding that index, so repeated
//     indices leave several slots aliasing one dictionary-backed slice. Writing through
//     them clobbers a decoded value and corrupts the dictionary itself.
//   - PLAIN slices the page buffer directly, so every slot points into that page's data.
//
// Capacity alone cannot distinguish these from our own storage, so we compare against
// the bounds of the block we allocated.
func (dec *ByteStreamSplitFixedLenByteArrayDecoder) owns(out []parquet.FixedLenByteArray) bool {
	if len(dec.storage) == 0 {
		return false
	}

	base := uintptr(unsafe.Pointer(unsafe.SliceData(dec.storage)))
	end := base + uintptr(len(dec.storage))
	for idx := range out {
		if cap(out[idx]) < dec.typeLen {
			return false
		}
		p := uintptr(unsafe.Pointer(unsafe.SliceData(out[idx])))
		if p < base || p+uintptr(dec.typeLen) > end {
			return false
		}
	}
	return true
}

// prepareOutput points every entry of out at storage this decoder owns, so that decoding
// in place cannot write through into memory belonging to a previous page's decoder.
//
// When out already sits entirely within our own block the headers are reused as-is and
// nothing is allocated, which keeps repeated decodes into the same buffer allocation
// free. Otherwise a single block is allocated for the whole window. Earlier windows keep
// pointing at the blocks they were given, so callers that decode into successive windows
// of one buffer (as the record reader does) keep their previously decoded values.
func (dec *ByteStreamSplitFixedLenByteArrayDecoder) prepareOutput(out []parquet.FixedLenByteArray) {
	if dec.owns(out) {
		for idx := range out {
			out[idx] = out[idx][:dec.typeLen]
		}
		return
	}

	storage := make([]byte, len(out)*dec.typeLen)
	dec.storage = storage
	for idx := range out {
		out[idx] = storage[:dec.typeLen:dec.typeLen]
		storage = storage[dec.typeLen:]
	}
}

func (dec *ByteStreamSplitFixedLenByteArrayDecoder) DecodeSpaced(out []parquet.FixedLenByteArray, nullCount int, validBits []byte, validBitsOffset int64) (int, error) {
	toRead := len(out) - nullCount

	// Back every slot, the null slots included, before decoding. The expansion below
	// permutes headers across the whole window, so preparing only the decoded prefix
	// would let headers we do not own migrate into it and force the next page to
	// reallocate. Preparing the window once keeps repeated decodes allocation free.
	dec.prepareOutput(out)

	valuesRead, err := dec.Decode(out[:toRead])
	if err != nil {
		return valuesRead, err
	}
	if valuesRead != toRead {
		return valuesRead, errors.New("parquet: number of values / definitions levels read did not match")
	}

	// This decoder writes through the caller's slices, so it must not leave aliased
	// headers behind for the next page; see spacedExpandSwap.
	return spacedExpandSwap(out, nullCount, validBits, validBitsOffset), nil
}
