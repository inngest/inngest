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
	"bytes"
	"errors"

	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/bitutil"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/internal/bitutils"
	shared_utils "github.com/apache/arrow-go/v18/internal/utils"
	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/internal/debug"
	format "github.com/apache/arrow-go/v18/parquet/internal/gen-go/parquet"
	"github.com/apache/arrow-go/v18/parquet/internal/utils"
	"github.com/apache/arrow-go/v18/parquet/schema"
)

// DecoderTraits provides an interface for more easily interacting with types
// to generate decoders for specific types.
type DecoderTraits interface {
	Decoder(e parquet.Encoding, descr *schema.Column, useDict bool, mem memory.Allocator) TypedDecoder
	BytesRequired(int) int
}

// NewDecoder constructs a decoder for a given type and encoding
func NewDecoder(t parquet.Type, e parquet.Encoding, descr *schema.Column, mem memory.Allocator) TypedDecoder {
	traits := getDecodingTraits(t)
	if traits == nil {
		return nil
	}

	return traits.Decoder(e, descr, false /* use dictionary */, mem)
}

// NewDictDecoder is like NewDecoder but for dictionary encodings, panics if type is bool.
//
// if mem is nil, memory.DefaultAllocator will be used
func NewDictDecoder(t parquet.Type, descr *schema.Column, mem memory.Allocator) DictDecoder {
	traits := getDecodingTraits(t)
	if traits == nil {
		return nil
	}

	if mem == nil {
		mem = memory.DefaultAllocator
	}

	return traits.Decoder(parquet.Encodings.RLEDict, descr, true /* use dictionary */, mem).(DictDecoder)
}

type decoder struct {
	descr    *schema.Column
	encoding format.Encoding
	nvals    int
	data     []byte
	typeLen  int
}

// newDecoderBase constructs the base decoding object that is embedded in the
// type specific decoders.
func newDecoderBase(e format.Encoding, descr *schema.Column) decoder {
	typeLen := -1
	if descr != nil && descr.PhysicalType() == parquet.Types.FixedLenByteArray {
		typeLen = int(descr.TypeLength())
	}

	return decoder{
		descr:    descr,
		encoding: e,
		typeLen:  typeLen,
	}
}

// SetData sets the data for decoding into the decoder to update the available
// data bytes and number of values available.
func (d *decoder) SetData(nvals int, data []byte) error {
	d.data = data
	d.nvals = nvals
	return nil
}

// ValuesLeft returns the number of remaining values that can be decoded
func (d *decoder) ValuesLeft() int { return d.nvals }

// Encoding returns the encoding type used by this decoder to decode the bytes.
func (d *decoder) Encoding() parquet.Encoding { return parquet.Encoding(d.encoding) }

type dictDecoder[T parquet.ColumnTypes] struct {
	decoder
	mem              memory.Allocator
	dictValueDecoder utils.DictionaryConverter[T]
	idxDecoder       *utils.TypedRleDecoder[T]
	idxReader        bytes.Reader

	idxScratchSpace  []uint64
	idxAppendScratch []int
	validScratch     []bool
}

// SetDict sets a decoder that can be used to decode the dictionary that is
// used for this column in order to return the proper values.
func (d *dictDecoder[T]) SetDict(dict TypedDecoder) {
	if dict.Type() != d.descr.PhysicalType() {
		panic("parquet: mismatch dictionary and column data type")
	}

	d.dictValueDecoder = NewDictConverter[T](dict)
}

// SetData sets the index value data into the decoder.
func (d *dictDecoder[T]) SetData(nvals int, data []byte) error {
	d.nvals = nvals
	if len(data) == 0 {
		// no data, bitwidth can safely be 0
		d.resetIndexDecoder(data, 0 /* bitwidth */)
		return nil
	}

	// grab the bit width from the first byte
	width := uint8(data[0])
	if width >= 64 {
		return errors.New("parquet: invalid or corrupted bit width")
	}

	// pass the rest of the data, minus that first byte, to the decoder
	d.resetIndexDecoder(data[1:], int(width))
	return nil
}

func (d *dictDecoder[T]) resetIndexDecoder(data []byte, width int) {
	d.idxReader.Reset(data)
	if d.idxDecoder == nil {
		d.idxDecoder = utils.NewTypedRleDecoder[T](&d.idxReader, width)
		return
	}

	d.idxDecoder.Reset(&d.idxReader, width)
}

func (d *dictDecoder[T]) discard(n int) (int, error) {
	n = d.idxDecoder.Discard(n)
	d.nvals -= n
	return n, nil
}

func (d *dictDecoder[T]) decode(out []T) (int, error) {
	n, err := d.idxDecoder.GetBatchWithDict(d.dictValueDecoder, out)
	d.nvals -= n
	return n, err
}

func (d *dictDecoder[T]) decodeSpaced(out []T, nullCount int, validBits []byte, validBitsOffset int64) (int, error) {
	n, err := d.idxDecoder.GetBatchWithDictSpaced(d.dictValueDecoder, out, nullCount, validBits, validBitsOffset)
	d.nvals -= n
	return n, err
}

func (d *dictDecoder[T]) DecodeIndices(numValues int, bldr array.Builder) (int, error) {
	n := shared_utils.Min(numValues, d.nvals)
	if cap(d.idxScratchSpace) < n {
		d.idxScratchSpace = make([]uint64, n, bitutil.NextPowerOf2(n))
	} else {
		d.idxScratchSpace = d.idxScratchSpace[:n]
	}

	n, err := d.idxDecoder.GetBatch(d.idxScratchSpace)

	if cap(d.idxAppendScratch) < n {
		d.idxAppendScratch = make([]int, n, bitutil.NextPowerOf2(n))
	} else {
		d.idxAppendScratch = d.idxAppendScratch[:n]
	}
	for i, v := range d.idxScratchSpace[:n] {
		d.idxAppendScratch[i] = int(v)
	}
	bldr.(*array.BinaryDictionaryBuilder).AppendIndices(d.idxAppendScratch, nil)
	d.nvals -= n
	return n, err
}

func (d *dictDecoder[T]) DecodeIndicesSpaced(numValues, nullCount int, validBits []byte, offset int64, bldr array.Builder) (int, error) {
	if cap(d.idxScratchSpace) < numValues {
		d.idxScratchSpace = make([]uint64, numValues, bitutil.NextPowerOf2(numValues))
	} else {
		d.idxScratchSpace = d.idxScratchSpace[:numValues]
	}

	n, err := d.idxDecoder.GetBatchSpaced(d.idxScratchSpace, nullCount, validBits, offset)
	if err != nil {
		return n, err
	}

	if cap(d.validScratch) < n {
		d.validScratch = make([]bool, n, bitutil.NextPowerOf2(n))
	} else {
		d.validScratch = d.validScratch[:n]
	}
	clear(d.validScratch)
	bitutils.VisitBitBlocks(validBits, offset, int64(n),
		func(pos int64) { d.validScratch[pos] = true }, func() {})

	if cap(d.idxAppendScratch) < n {
		d.idxAppendScratch = make([]int, n, bitutil.NextPowerOf2(n))
	} else {
		d.idxAppendScratch = d.idxAppendScratch[:n]
	}
	for i, v := range d.idxScratchSpace[:n] {
		d.idxAppendScratch[i] = int(v)
	}
	bldr.(*array.BinaryDictionaryBuilder).AppendIndices(d.idxAppendScratch, d.validScratch)
	d.nvals -= n - nullCount
	return n, nil
}

// spacedExpandSwap is spacedExpand for reusable slices whose entries may later be
// written through. spacedExpand moves values with copy and leaves the null slots alone,
// which for the slice-header column types (ByteArray / FixedLenByteArray) leaves
// duplicate headers behind: a null slot and a valid slot end up referencing the same
// backing array. A later decoder that writes through those entries would then clobber
// one value when the buffer is reused for another page.
//
// Swapping instead of copying keeps the buffer a permutation of its original elements,
// so no slot aliases another and every slot keeps its reusable capacity.
func spacedExpandSwap[T parquet.ColumnTypes](buffer []T, nullCount int, validBits []byte, validBitsOffset int64) int {
	numValues := len(buffer)

	idxDecode := int64(numValues - nullCount)
	if idxDecode == 0 {
		return numValues
	}

	rdr := bitutils.NewReverseSetBitRunReader(validBits, validBitsOffset, int64(numValues))
	for {
		run := rdr.NextRun()
		if run.Length == 0 {
			break
		}

		idxDecode -= run.Length
		// Once the decoded prefix is already aligned every remaining swap is a
		// self-swap, so there is nothing left to do. Mirrors spacedExpand.
		if idxDecode == run.Pos {
			return numValues
		}
		for k := run.Length - 1; k >= 0; k-- {
			dst, src := run.Pos+k, idxDecode+k
			buffer[dst], buffer[src] = buffer[src], buffer[dst]
		}
	}

	return numValues
}

// spacedExpand is used to take a slice of data and utilize the bitmap provided to fill in nulls into the
// correct slots according to the bitmap in order to produce a fully expanded result slice with nulls
// in the correct slots.
func spacedExpand[T parquet.ColumnTypes](buffer []T, nullCount int, validBits []byte, validBitsOffset int64) int {
	numValues := len(buffer)

	idxDecode := int64(numValues - nullCount)
	if idxDecode == 0 { // if there's nothing to decode there's nothing to do.
		return numValues
	}

	// read the bitmap in reverse grabbing runs of valid bits where possible.
	rdr := bitutils.NewReverseSetBitRunReader(validBits, validBitsOffset, int64(numValues))
	for {
		run := rdr.NextRun()
		if run.Length == 0 {
			break
		}

		// copy data from the end of the slice to it's proper location in the slice after accounting for the nulls
		// because we technically don't care what is in the null slots we don't actually have to clean
		// up after ourselves because we're doing this in reverse to guarantee that we'll always simply
		// overwrite any existing data with the correctly spaced data. Any data that happens to be left in the null
		// slots is fine since it shouldn't matter and saves us work.
		idxDecode -= run.Length
		if idxDecode == run.Pos {
			return numValues
		}
		n := copy(buffer[run.Pos:], buffer[idxDecode:int64(idxDecode)+run.Length])
		debug.Assert(n == int(run.Length), "copy copied incorrect number of elements in spacedExpand")
	}

	return numValues
}
