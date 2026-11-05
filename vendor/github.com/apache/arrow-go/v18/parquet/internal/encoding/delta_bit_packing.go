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
	"fmt"
	"io"
	"math"
	"math/bits"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/memory"
	shared_utils "github.com/apache/arrow-go/v18/internal/utils"
	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/internal/utils"
)

const deltaBitPackScratchSize = 1024

// see the deltaBitPack encoder for a description of the encoding format that is
// used for delta-bitpacking.
type deltaBitPackDecoder[T int32 | int64] struct {
	decoder

	mem memory.Allocator

	usedFirst            bool
	bitdecoder           *utils.BitReader
	blockSize            uint64
	currentBlockVals     uint32
	miniBlocksPerBlock   uint64
	valsPerMini          uint32
	currentMiniBlockVals uint32
	minDelta             int64
	miniBlockIdx         uint64

	deltaBitWidths *memory.Buffer
	deltaBitWidth  byte

	totalValues uint64
	lastVal     int64

	miniBlockValues []T
	deltaBuf        []uint64
}

// returns the number of bytes read so far
func (d *deltaBitPackDecoder[T]) bytesRead() int64 {
	return d.bitdecoder.CurOffset()
}

func (d *deltaBitPackDecoder[T]) Allocator() memory.Allocator { return d.mem }

// SetData sets the bytes and the expected number of values to decode
// into the decoder, updating the decoder and allowing it to be reused.
func (d *deltaBitPackDecoder[T]) SetData(nvalues int, data []byte) error {
	// set our data into the underlying decoder for the type
	if err := d.decoder.SetData(nvalues, data); err != nil {
		return err
	}
	// create a bit reader for our decoder's values
	d.bitdecoder = utils.NewBitReader(bytes.NewReader(d.data))
	d.currentBlockVals = 0
	d.currentMiniBlockVals = 0
	if d.deltaBitWidths == nil {
		d.deltaBitWidths = memory.NewResizableBuffer(d.mem)
	}

	var ok bool
	d.blockSize, ok = d.bitdecoder.GetVlqInt()
	if !ok {
		return errors.New("parquet: eof exception")
	}
	if d.blockSize == 0 {
		return errors.New("parquet: cannot have zero values per block")
	}
	if d.blockSize > math.MaxUint32 || d.blockSize%128 != 0 {
		return fmt.Errorf("parquet: values per block must be a multiple of 128, got %d", d.blockSize)
	}

	if d.miniBlocksPerBlock, ok = d.bitdecoder.GetVlqInt(); !ok {
		return errors.New("parquet: eof exception")
	}
	if d.miniBlocksPerBlock == 0 {
		return errors.New("parquet: cannot have zero miniblock per block")
	}
	if d.miniBlocksPerBlock > math.MaxUint32 {
		return fmt.Errorf("parquet: too many miniblocks per block: %d", d.miniBlocksPerBlock)
	}
	if d.blockSize%d.miniBlocksPerBlock != 0 {
		return fmt.Errorf("parquet: miniblocks per block must divide the block size, got %d miniblocks for %d values", d.miniBlocksPerBlock, d.blockSize)
	}

	valuesPerMini := d.blockSize / d.miniBlocksPerBlock
	if valuesPerMini == 0 || valuesPerMini%32 != 0 {
		return fmt.Errorf("parquet: values per miniblock must be a nonzero multiple of 32, got %d", valuesPerMini)
	}

	if d.totalValues, ok = d.bitdecoder.GetVlqInt(); !ok {
		return errors.New("parquet: eof exception")
	}
	if d.totalValues > math.MaxUint32 || nvalues < 0 || d.totalValues > uint64(nvalues) {
		return fmt.Errorf("parquet: invalid delta value count %d for %d buffered values", d.totalValues, nvalues)
	}

	if d.lastVal, ok = d.bitdecoder.GetZigZagVlqInt(); !ok {
		return errors.New("parquet: eof exception")
	}
	if d.Type() == parquet.Types.Int32 && (d.lastVal < math.MinInt32 || d.lastVal > math.MaxInt32) {
		return fmt.Errorf("parquet: initial delta value %d exceeds INT32 range", d.lastVal)
	}

	d.valsPerMini = uint32(valuesPerMini)
	d.usedFirst = false
	d.nvals = int(d.totalValues)
	return nil
}

// initialize a block to decode
func (d *deltaBitPackDecoder[T]) initBlock() error {
	// first we grab the min delta value that we'll start from
	var ok bool
	if d.minDelta, ok = d.bitdecoder.GetZigZagVlqInt(); !ok {
		return errors.New("parquet: eof exception")
	}
	if d.Type() == parquet.Types.Int32 && (d.minDelta < math.MinInt32 || d.minDelta > math.MaxInt32) {
		return fmt.Errorf("parquet: minimum delta %d exceeds INT32 range", d.minDelta)
	}

	// ensure we have enough space for our miniblocks to decode the widths
	d.deltaBitWidths.Resize(int(d.miniBlocksPerBlock))

	var err error
	for i := uint64(0); i < d.miniBlocksPerBlock; i++ {
		if d.deltaBitWidths.Bytes()[i], err = d.bitdecoder.ReadByte(); err != nil {
			return err
		}
	}

	d.miniBlockIdx = 0
	d.deltaBitWidth = d.deltaBitWidths.Bytes()[0]
	d.currentBlockVals = uint32(d.blockSize)
	return nil
}

func (d *deltaBitPackDecoder[T]) unpackNextMini() error {
	if d.miniBlockValues == nil {
		d.miniBlockValues = make([]T, 0, int(d.valsPerMini))
	} else {
		d.miniBlockValues = d.miniBlockValues[:0]
	}
	d.deltaBitWidth = d.deltaBitWidths.Bytes()[int(d.miniBlockIdx)]
	maxBitWidth := byte(64)
	if d.Type() == parquet.Types.Int32 {
		maxBitWidth = 32
	}
	if d.deltaBitWidth > maxBitWidth {
		return fmt.Errorf("parquet: delta bit width %d exceeds decoder width %d", d.deltaBitWidth, maxBitWidth)
	}
	d.currentMiniBlockVals = d.valsPerMini

	n := int(d.valsPerMini)
	width := uint(d.deltaBitWidth)

	if width == 0 {
		for j := 0; j < n; j++ {
			d.lastVal += int64(d.minDelta)
			d.miniBlockValues = append(d.miniBlockValues, T(d.lastVal))
		}
	} else {
		scratchSize := min(n, deltaBitPackScratchSize)
		if cap(d.deltaBuf) < scratchSize {
			d.deltaBuf = make([]uint64, scratchSize)
		}
		d.deltaBuf = d.deltaBuf[:scratchSize]
		minDelta := d.minDelta

		for remaining := n; remaining > 0; {
			batchSize := min(remaining, len(d.deltaBuf))
			nread, err := d.bitdecoder.GetBatch(width, d.deltaBuf[:batchSize])
			if nread != batchSize {
				if err == nil || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
					return io.ErrUnexpectedEOF
				}
				return err
			}
			if err != nil {
				return err
			}

			for _, delta := range d.deltaBuf[:nread] {
				d.lastVal += int64(delta) + minDelta
				d.miniBlockValues = append(d.miniBlockValues, T(d.lastVal))
			}
			remaining -= batchSize
		}
	}
	d.miniBlockIdx++
	return nil
}

func (d *deltaBitPackDecoder[T]) Discard(n int) (int, error) {
	n = min(n, int(d.nvals))
	if n == 0 {
		return 0, nil
	}

	var (
		err       error
		remaining = n
	)

	if !d.usedFirst {
		d.usedFirst = true
		remaining--
	}

	for remaining > 0 {
		if d.currentBlockVals == 0 {
			if err = d.initBlock(); err != nil {
				return n - remaining, err
			}
		}

		if d.currentMiniBlockVals == 0 {
			if err = d.unpackNextMini(); err != nil {
				return n - remaining, err
			}
		}

		start := d.valsPerMini - d.currentMiniBlockVals
		numToDiscard := len(d.miniBlockValues[start:])
		if numToDiscard > remaining {
			numToDiscard = remaining
		}

		d.currentBlockVals -= uint32(numToDiscard)
		d.currentMiniBlockVals -= uint32(numToDiscard)
		remaining -= numToDiscard
	}

	d.nvals -= n
	return n, nil
}

// Decode retrieves min(remaining values, len(out)) values from the data and returns the number
// of values actually decoded and any errors encountered.
func (d *deltaBitPackDecoder[T]) Decode(out []T) (int, error) {
	max := shared_utils.Min(len(out), int(d.nvals))
	if max == 0 {
		return 0, nil
	}

	out = out[:max]
	if !d.usedFirst { // starting value to calculate deltas against
		out[0] = T(d.lastVal)
		out = out[1:]
		d.usedFirst = true
	}

	var err error
	for len(out) > 0 { // unpack mini blocks until we get all the values we need
		if d.currentBlockVals == 0 {
			err = d.initBlock()
			if err != nil {
				return 0, err
			}
		}
		if d.currentMiniBlockVals == 0 {
			err = d.unpackNextMini()
		}
		if err != nil {
			return 0, err
		}

		// copy as many values from our mini block as we can into out
		start := int(d.valsPerMini - d.currentMiniBlockVals)
		numCopied := copy(out, d.miniBlockValues[start:])

		out = out[numCopied:]
		d.currentBlockVals -= uint32(numCopied)
		d.currentMiniBlockVals -= uint32(numCopied)
	}
	d.nvals -= max
	return max, nil
}

// DecodeSpaced is like Decode, but the result is spaced out appropriately based on the passed in bitmap
func (d *deltaBitPackDecoder[T]) DecodeSpaced(out []T, nullCount int, validBits []byte, validBitsOffset int64) (int, error) {
	toread := len(out) - nullCount
	values, err := d.Decode(out[:toread])
	if err != nil {
		return values, err
	}
	if values != toread {
		return values, errors.New("parquet: number of values / definition levels read did not match")
	}

	return spacedExpand(out, nullCount, validBits, validBitsOffset), nil
}

// Type returns the underlying physical type this decoder works with
func (dec *deltaBitPackDecoder[T]) Type() parquet.Type {
	switch v := any(dec).(type) {
	case *deltaBitPackDecoder[int32]:
		return parquet.Types.Int32
	case *deltaBitPackDecoder[int64]:
		return parquet.Types.Int64
	default:
		panic(fmt.Sprintf("deltaBitPackDecoder is not supported for type: %T", v))
	}
}

// DeltaBitPackInt32Decoder decodes Int32 values which are packed using the Delta BitPacking algorithm.
type DeltaBitPackInt32Decoder = deltaBitPackDecoder[int32]

// DeltaBitPackInt64Decoder decodes Int64 values which are packed using the Delta BitPacking algorithm.
type DeltaBitPackInt64Decoder = deltaBitPackDecoder[int64]

const (
	// block size must be a multiple of 128
	defaultBlockSize     = 128
	defaultNumMiniBlocks = 4
	// block size / number of mini blocks must result in a multiple of 32
	defaultNumValuesPerMini = 32
	// max size of the header for the delta blocks
	maxHeaderWriterSize = 32
)

// deltaBitPackEncoder is an encoder for the DeltaBinary Packing format
// as per the parquet spec.
//
// Consists of a header followed by blocks of delta encoded values binary packed.
//
//	Format
//		[header] [block 1] [block 2] ... [block N]
//
//	Header
//		[block size] [number of mini blocks per block] [total value count] [first value]
//
//	Block
//		[min delta] [list of bitwidths of the miniblocks] [miniblocks...]
//
// Sets aside bytes at the start of the internal buffer where the header will be written,
// and only writes the header when FlushValues is called before returning it.
type deltaBitPackEncoder[T int32 | int64] struct {
	encoder

	bitWriter  *utils.BitWriter
	totalVals  uint64
	firstVal   int64
	currentVal int64

	blockSize     uint64
	miniBlockSize uint64
	numMiniBlocks uint64
	deltas        []int64
	spacedScratch *memory.Buffer
}

// flushBlock flushes out a finished block for writing to the underlying encoder
func (enc *deltaBitPackEncoder[T]) flushBlock() {
	if len(enc.deltas) == 0 {
		return
	}

	// determine the minimum delta value
	minDelta := int64(math.MaxInt64)
	for _, delta := range enc.deltas {
		if delta < minDelta {
			minDelta = delta
		}
	}

	enc.bitWriter.WriteZigZagVlqInt(minDelta)
	// reserve enough bytes to write out our miniblock deltas
	offset, _ := enc.bitWriter.SkipBytes(int(enc.numMiniBlocks))

	valuesToWrite := int64(len(enc.deltas))
	for i := 0; i < int(enc.numMiniBlocks); i++ {
		n := shared_utils.Min(int64(enc.miniBlockSize), valuesToWrite)
		if n == 0 {
			break
		}

		maxDelta := int64(math.MinInt64)
		start := i * int(enc.miniBlockSize)
		for _, val := range enc.deltas[start : start+int(n)] {
			maxDelta = shared_utils.Max(maxDelta, val)
		}

		// compute bit width to store (max_delta - min_delta)
		width := uint(bits.Len64(uint64(maxDelta - minDelta)))
		// write out the bit width we used into the bytes we reserved earlier
		enc.bitWriter.WriteAt([]byte{byte(width)}, int64(offset+i))

		// write out our deltas
		for _, val := range enc.deltas[start : start+int(n)] {
			enc.bitWriter.WriteValue(uint64(val-minDelta), width)
		}

		valuesToWrite -= n

		// pad the last block if n < miniBlockSize
		for ; n < int64(enc.miniBlockSize); n++ {
			enc.bitWriter.WriteValue(0, width)
		}
	}
	enc.deltas = enc.deltas[:0]
}

// putInternal is the implementation for actually writing data which must be
// integral data as int, int8, int32, or int64.
func (enc *deltaBitPackEncoder[T]) Put(in []T) {
	if len(in) == 0 {
		return
	}

	idx := 0
	if enc.totalVals == 0 {
		enc.blockSize = defaultBlockSize
		enc.numMiniBlocks = defaultNumMiniBlocks
		enc.miniBlockSize = defaultNumValuesPerMini

		enc.firstVal = int64(in[0])
		enc.currentVal = enc.firstVal
		idx = 1

		enc.bitWriter = utils.NewBitWriter(enc.sink)
	}

	enc.totalVals += uint64(len(in))
	for ; idx < len(in); idx++ {
		// compute the delta at the physical type's width so that overflow wraps in
		// two's complement, as the spec requires. Widening to int64 first would produce
		// deltas (and miniblock bit-widths) beyond the type's width for INT32 columns,
		// which other readers (parquet-cpp, arrow-rs) reject as corrupt.
		delta := int64(in[idx] - T(enc.currentVal))
		enc.deltas = append(enc.deltas, delta)
		enc.currentVal = int64(in[idx])
		if len(enc.deltas) == int(enc.blockSize) {
			enc.flushBlock()
		}
	}
}

// FlushValues flushes any remaining data and returns the finished encoded buffer
// or returns nil and any error encountered during flushing.
func (enc *deltaBitPackEncoder[T]) FlushValues() (Buffer, error) {
	if enc.bitWriter != nil {
		// write any remaining values
		enc.flushBlock()
		enc.bitWriter.Flush(true)
	} else {
		enc.blockSize = defaultBlockSize
		enc.numMiniBlocks = defaultNumMiniBlocks
		enc.miniBlockSize = defaultNumValuesPerMini
	}

	buffer := make([]byte, maxHeaderWriterSize)
	headerWriter := utils.NewBitWriter(utils.NewWriterAtBuffer(buffer))

	headerWriter.WriteVlqInt(uint64(enc.blockSize))
	headerWriter.WriteVlqInt(uint64(enc.numMiniBlocks))
	headerWriter.WriteVlqInt(uint64(enc.totalVals))
	headerWriter.WriteZigZagVlqInt(int64(enc.firstVal))
	headerWriter.Flush(false)

	buffer = buffer[:headerWriter.Written()]
	enc.totalVals = 0

	if enc.bitWriter != nil {
		flushed := enc.sink.Finish()
		defer flushed.Release()

		buffer = append(buffer, flushed.Buf()[:enc.bitWriter.Written()]...)
	}
	return poolBuffer{memory.NewBufferBytes(buffer)}, nil
}

// EstimatedDataEncodedSize returns the current amount of data actually flushed out and written
func (enc *deltaBitPackEncoder[T]) EstimatedDataEncodedSize() int64 {
	if enc.bitWriter == nil {
		return 0
	}

	return int64(enc.bitWriter.Written())
}

func (enc *deltaBitPackEncoder[T]) Release() {
	enc.encoder.Release()
	if enc.spacedScratch != nil {
		enc.spacedScratch.Release()
		enc.spacedScratch = nil
	}
}

// PutSpaced takes a slice of values along with a bitmap that describes the nulls and an offset into the bitmap
// in order to write spaced data to the encoder.
func (enc *deltaBitPackEncoder[T]) PutSpaced(in []T, validBits []byte, validBitsOffset int64) {
	if enc.spacedScratch == nil {
		enc.spacedScratch = memory.NewResizableBuffer(enc.mem)
	}
	dt := arrow.GetDataType[T]().(arrow.FixedWidthDataType)
	enc.spacedScratch.ResizeNoShrink(dt.Bytes() * len(in))

	data := arrow.GetData[T](enc.spacedScratch.Buf())
	nvalid := spacedCompress(in, data, validBits, validBitsOffset)
	enc.Put(data[:nvalid])
}

// Type returns the underlying physical type this encoder works with
func (dec *deltaBitPackEncoder[T]) Type() parquet.Type {
	switch v := any(dec).(type) {
	case *deltaBitPackEncoder[int32]:
		return parquet.Types.Int32
	case *deltaBitPackEncoder[int64]:
		return parquet.Types.Int64
	default:
		panic(fmt.Sprintf("deltaBitPackEncoder is not supported for type: %T", v))
	}
}

// DeltaBitPackInt32Encoder is an encoder for the delta bitpacking encoding for Int32 data.
type DeltaBitPackInt32Encoder = deltaBitPackEncoder[int32]

// DeltaBitPackInt64Encoder is an encoder for the delta bitpacking encoding for Int64 data.
type DeltaBitPackInt64Encoder = deltaBitPackEncoder[int64]
