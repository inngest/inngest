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

package utils

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"unsafe"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/bitutil"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/internal/utils"
)

// masks for grabbing the trailing bits based on the number of trailing bits desired
var trailingMask [64]uint64

func init() {
	// generate the masks at init so we don't have to hard code them.
	for i := 0; i < 64; i++ {
		trailingMask[i] = (math.MaxUint64 >> (64 - i))
	}
}

// trailingBits returns a value constructed from the bits trailing bits of
// the value v that is passed in. If bits >= 64, then we just return v.
func trailingBits(v uint64, bits uint) uint64 {
	if bits >= 64 {
		return v
	}
	return v & trailingMask[bits]
}

// reader is a useful interface to define the functionality we need for implementation
type reader interface {
	io.Reader
	io.ReaderAt
	io.Seeker
}

// default buffer length
const buflen = 1024

// BitReader implements functionality for reading bits or bytes buffering up to a uint64
// at a time from the reader in order to improve efficiency. It also provides
// methods to read multiple bytes in one read such as encoded ints/values.
//
// This BitReader is the basis for the other utility classes like RLE decoding
// and such, providing the necessary functions for interpreting the values.
type BitReader struct {
	reader     reader
	buffer     uint64
	byteoffset int64
	bitoffset  uint
	validBits  uint
	raw        [8]byte

	unpackBuf [buflen]uint32
}

// NewBitReader takes in a reader that implements io.Reader, io.ReaderAt and io.Seeker
// interfaces and returns a BitReader for use with various bit level manipulations.
func NewBitReader(r reader) *BitReader {
	return &BitReader{reader: r}
}

// CurOffset returns the current Byte offset into the data that the reader is at.
func (b *BitReader) CurOffset() int64 {
	return b.byteoffset + bitutil.BytesForBits(int64(b.bitoffset))
}

// Reset allows reusing a BitReader by setting a new reader and resetting the internal
// state back to zeros.
func (b *BitReader) Reset(r reader) {
	b.reader = r
	b.buffer = 0
	b.byteoffset = 0
	b.bitoffset = 0
	b.validBits = 0
}

// GetVlqInt reads a Vlq encoded int from the stream. The encoded value must start
// at the beginning of a byte and this returns false if there weren't enough bytes
// in the buffer or reader. This will call `ReadByte` which in turn retrieves byte
// aligned values from the reader
func (b *BitReader) GetVlqInt() (uint64, bool) {
	tmp, err := binary.ReadUvarint(b)
	if err != nil {
		return 0, false
	}
	return tmp, true
}

// GetZigZagVlqInt reads a zigzag encoded integer, returning false if there weren't
// enough bytes remaining.
func (b *BitReader) GetZigZagVlqInt() (int64, bool) {
	u, ok := b.GetVlqInt()
	if !ok {
		return 0, false
	}

	return int64(u>>1) ^ -int64(u&1), true
}

// ReadByte reads a single aligned byte from the underlying stream, or populating
// error if there aren't enough bytes left.
func (b *BitReader) ReadByte() (byte, error) {
	var tmp byte
	if ok := b.getAlignedUint8(1, &tmp); !ok {
		return 0, errors.New("failed to read byte")
	}

	return tmp, nil
}

// getAlignedUint8 reads nbytes from the underlying stream into the passed uint8 value.
// Returning false if there aren't enough bytes remaining in the stream or if an invalid
// type is passed. The bytes are read aligned to byte boundaries.
func (b *BitReader) getAlignedUint8(nbytes int, v *uint8) bool {
	if nbytes > 1 {
		return false
	}

	bread := bitutil.BytesForBits(int64(b.bitoffset))
	b.byteoffset += bread
	n, err := b.reader.ReadAt(b.raw[:nbytes], b.byteoffset)
	if err != nil && err != io.EOF {
		return false
	}
	if n != nbytes {
		return false
	}

	*v = b.raw[0]

	b.byteoffset += int64(nbytes)
	b.bitoffset = 0
	return b.fillbuffer() == nil
}

// getAlignedUint16 reads nbytes from the underlying stream into the passed uint16 value.
func (b *BitReader) getAlignedUint16(nbytes int, v *uint16) bool {
	if nbytes > 2 {
		return false
	}

	bread := bitutil.BytesForBits(int64(b.bitoffset))
	b.byteoffset += bread
	n, err := b.reader.ReadAt(b.raw[:nbytes], b.byteoffset)
	if err != nil && err != io.EOF {
		return false
	}
	if n != nbytes {
		return false
	}

	// zero pad the bytes
	memory.Set(b.raw[n:2], 0)

	*v = binary.LittleEndian.Uint16(b.raw[:2])

	b.byteoffset += int64(nbytes)
	b.bitoffset = 0
	return b.fillbuffer() == nil
}

// getAlignedUint32 reads nbytes from the underlying stream into the passed uint32 value.
func (b *BitReader) getAlignedUint32(nbytes int, v *uint32) bool {
	if nbytes > 4 {
		return false
	}

	bread := bitutil.BytesForBits(int64(b.bitoffset))
	b.byteoffset += bread
	n, err := b.reader.ReadAt(b.raw[:nbytes], b.byteoffset)
	if err != nil && err != io.EOF {
		return false
	}
	if n != nbytes {
		return false
	}

	// zero pad the bytes
	memory.Set(b.raw[n:4], 0)

	*v = binary.LittleEndian.Uint32(b.raw[:4])

	b.byteoffset += int64(nbytes)
	b.bitoffset = 0
	return b.fillbuffer() == nil
}

// getAlignedUint64 reads nbytes from the underlying stream into the passed uint64 value.
func (b *BitReader) getAlignedUint64(nbytes int, v *uint64) bool {
	if nbytes > 8 {
		return false
	}

	bread := bitutil.BytesForBits(int64(b.bitoffset))
	b.byteoffset += bread
	n, err := b.reader.ReadAt(b.raw[:nbytes], b.byteoffset)
	if err != nil && err != io.EOF {
		return false
	}
	if n != nbytes {
		return false
	}

	// zero pad the bytes
	memory.Set(b.raw[n:8], 0)

	*v = binary.LittleEndian.Uint64(b.raw[:8])

	b.byteoffset += int64(nbytes)
	b.bitoffset = 0
	return b.fillbuffer() == nil
}

// fillbuffer fills the uint64 buffer with bytes from the underlying stream
func (b *BitReader) fillbuffer() error {
	n, err := b.reader.ReadAt(b.raw[:], b.byteoffset)
	if err != nil && err != io.EOF {
		return err
	}
	b.validBits = uint(n * 8)
	for i := n; i < 8; i++ {
		b.raw[i] = 0
	}
	b.buffer = binary.LittleEndian.Uint64(b.raw[:])
	return nil
}

// next reads an integral value from the next bits in the buffer
func (b *BitReader) next(bits uint) (v uint64, err error) {
	if bits == 0 {
		return 0, nil
	}

	if b.bitoffset == 64 {
		b.byteoffset += 8
		b.bitoffset = 0
		if err = b.fillbuffer(); err != nil {
			return 0, err
		}
	}

	end := b.bitoffset + bits
	if end <= 64 {
		if end > b.validBits {
			return 0, io.ErrUnexpectedEOF
		}
		v = trailingBits(b.buffer, end) >> b.bitoffset
		b.bitoffset = end
		return v, nil
	}

	if b.validBits < 64 {
		return 0, io.ErrUnexpectedEOF
	}

	firstBits := 64 - b.bitoffset
	v = b.buffer >> b.bitoffset
	b.byteoffset += 8
	b.bitoffset = 0
	if err = b.fillbuffer(); err != nil {
		return 0, err
	}

	remaining := bits - firstBits
	if remaining > b.validBits {
		return 0, io.ErrUnexpectedEOF
	}
	v |= trailingBits(b.buffer, remaining) << firstBits
	b.bitoffset = remaining
	return v, nil
}

// nextBool reads one bit after the caller has verified that its byte is
// available. GetBatchBools performs that check once per byte so its unaligned
// and trailing reads do not pay the scalar validBits check for every value.
func (b *BitReader) nextBool() (bool, error) {
	if b.bitoffset == 64 {
		b.byteoffset += 8
		b.bitoffset = 0
		if err := b.fillbuffer(); err != nil {
			return false, err
		}
	}

	v := b.buffer&(uint64(1)<<b.bitoffset) != 0
	b.bitoffset++
	return v, nil
}

// getBatchIndexScalar decodes indexes from the buffered scalar path. It checks
// availability once per buffer and decodes all complete values in that buffer
// without repeating the validBits check for every index.
func (b *BitReader) getBatchIndexScalar(bits uint, out []IndexType, stopAtBufferBoundary bool) (int, error) {
	i := 0
	for i < len(out) {
		if stopAtBufferBoundary && (b.bitoffset == 0 || b.bitoffset == 64) {
			if b.bitoffset == 64 {
				b.byteoffset += 8
				b.bitoffset = 0
			}
			return i, nil
		}

		if b.bitoffset < b.validBits {
			available := int((b.validBits - b.bitoffset) / bits)
			available = min(available, len(out)-i)
			for range available {
				end := b.bitoffset + bits
				out[i] = IndexType(trailingBits(b.buffer, end) >> b.bitoffset)
				b.bitoffset = end
				i++
			}
			if available > 0 {
				continue
			}
		}

		val, err := b.next(bits)
		if err != nil {
			return i, err
		}
		out[i] = IndexType(val)
		i++
	}

	return i, nil
}

// GetBatchIndex is like GetBatch but for IndexType (used for dictionary decoding)
func (b *BitReader) GetBatchIndex(bits uint, out []IndexType) (i int, err error) {
	// IndexType is a 32-bit value so bits must be less than 32 when unpacking
	// values using the bitreader.
	if bits > 32 {
		return 0, errors.New("must be 32 bits or less per read")
	}
	if bits == 0 {
		clear(out)
		return len(out), nil
	}

	length := len(out)
	// If the buffer is partially consumed, read indexes until the next buffer boundary.
	if b.bitoffset != 0 {
		var n int
		n, err = b.getBatchIndexScalar(bits, out, true)
		i += n
		if err != nil {
			return i, err
		}
		if i == length {
			return i, nil
		}
	}

	if _, err = b.reader.Seek(b.byteoffset, io.SeekStart); err != nil {
		return i, err
	}
	// grab as many 32 byte chunks as possible in one shot
	if i < length { // IndexType should be a 32 bit value so we can do quick unpacking right into the output
		numUnpacked, unpackErr := unpack32(b.reader, (*(*[]uint32)(unsafe.Pointer(&out)))[i:], int(bits))
		i += numUnpacked
		b.byteoffset += int64(numUnpacked * int(bits) / 8)
		if unpackErr != nil {
			return i, unpackErr
		}
	}

	// re-fill our buffer just in case.
	if err = b.fillbuffer(); err != nil {
		return i, err
	}
	// grab the remaining values that aren't 32 byte aligned
	n, err := b.getBatchIndexScalar(bits, out[i:], false)
	i += n
	return i, err
}

// GetBatchBools is like GetBatch but optimized for reading bits as boolean values
func (b *BitReader) GetBatchBools(out []bool) (int, error) {
	length := len(out)

	i := 0
	// read until we are byte-aligned
	for ; i < length && b.bitoffset != 0; i++ {
		if i == 0 || b.bitoffset%8 == 0 {
			if err := b.ensureBoolBitAvailable(); err != nil {
				return i, err
			}
		}
		val, err := b.nextBool()
		out[i] = val
		if err != nil {
			return i, err
		}
	}

	if _, err := b.reader.Seek(b.byteoffset, io.SeekStart); err != nil {
		return i, err
	}
	buf := arrow.Uint32Traits.CastToBytes(b.unpackBuf[:])
	blen := buflen * 8
	for length-i >= 8 {
		// grab byte-aligned bits in a loop since it's more efficient than going
		// bit by bit when you can grab 8 bools at a time.
		unpackSize := utils.Min(blen, length-i) / 8 * 8
		bytesToRead := int(bitutil.BytesForBits(int64(unpackSize)))
		n := 0
		for n < bytesToRead {
			nread, err := b.reader.Read(buf[n:bytesToRead])
			n += nread
			if n == bytesToRead {
				break
			}
			if err != nil {
				if n > 0 {
					BytesToBools(buf[:n], out[i:])
				}
				i += n * 8
				b.byteoffset += int64(n)
				if err == io.EOF && n > 0 {
					err = io.ErrUnexpectedEOF
				}
				return i, err
			}
			if nread == 0 {
				if n > 0 {
					BytesToBools(buf[:n], out[i:])
				}
				i += n * 8
				b.byteoffset += int64(n)
				return i, io.ErrNoProgress
			}
		}
		BytesToBools(buf[:n], out[i:])
		i += n * 8
		b.byteoffset += int64(n)
	}

	if err := b.fillbuffer(); err != nil {
		return i, err
	}
	// grab the trailing bits
	for ; i < length; i++ {
		if b.bitoffset%8 == 0 {
			if err := b.ensureBoolBitAvailable(); err != nil {
				return i, err
			}
		}
		val, err := b.nextBool()
		out[i] = val
		if err != nil {
			return i, err
		}
	}

	return i, nil
}

func (b *BitReader) ensureBoolBitAvailable() error {
	var probe [1]byte
	n, err := b.reader.ReadAt(probe[:], b.byteoffset+int64(b.bitoffset/8))
	if n == 1 {
		return nil
	}
	if err == nil {
		return io.ErrNoProgress
	}
	if errors.Is(err, io.EOF) {
		return io.ErrUnexpectedEOF
	}
	return err
}

func (b *BitReader) Discard(bits uint, n int) (int, error) {
	if bits > 64 {
		return 0, errors.New("must be 64 bits or less per read")
	}

	i := 0
	for ; i < n && b.bitoffset != 0; i++ {
		if _, err := b.next(bits); err != nil {
			return i, err
		}
	}

	if n-i > 32 {
		toSkip := (n - i) / 32 * 32

		bytesToSkip := bitutil.BytesForBits(int64(toSkip * int(bits)))
		if bytesToSkip > 0 {
			var last [1]byte
			if nread, err := b.reader.ReadAt(last[:], b.byteoffset+int64(bytesToSkip)-1); nread != 1 {
				if err == nil {
					err = io.ErrUnexpectedEOF
				}
				return i, err
			}
		}
		b.byteoffset += int64(bytesToSkip)
		i += toSkip
	}

	if err := b.fillbuffer(); err != nil {
		return i, err
	}
	for ; i < n; i++ {
		if _, err := b.next(bits); err != nil {
			return i, err
		}
	}
	return n, nil
}

// GetBatch fills out by decoding values repeated from the stream that are encoded
// using bits as the number of bits per value. The values are expected to be bit packed
// so we will unpack the values to populate.
func (b *BitReader) GetBatch(bits uint, out []uint64) (int, error) {
	// since we're unpacking into uint64 values, we can't support bits being
	// larger than 64 here as that's the largest size value we're reading
	if bits > 64 {
		return 0, errors.New("must be 64 bits or less per read")
	}

	length := len(out)

	i := 0
	// read until we are byte aligned
	for ; i < length && b.bitoffset != 0; i++ {
		val, err := b.next(bits)
		out[i] = val
		if err != nil {
			return i, err
		}
	}

	if bits <= 32 {
		if _, err := b.reader.Seek(b.byteoffset, io.SeekStart); err != nil {
			return i, err
		}
		for i < length {
			// unpack groups of 32 bytes at a time into a buffer since it's more efficient
			unpackSize := utils.Min(buflen, length-i)
			numUnpacked, err := unpack32(b.reader, b.unpackBuf[:unpackSize], int(bits))

			for k := 0; k < numUnpacked; k++ {
				out[i+k] = uint64(b.unpackBuf[k])
			}
			i += numUnpacked
			b.byteoffset += int64(numUnpacked * int(bits) / 8)
			if err != nil {
				return i, err
			}
			if numUnpacked == 0 {
				break
			}
		}
	}

	if err := b.fillbuffer(); err != nil {
		return i, err
	}
	// and then the remaining trailing values
	for ; i < length; i++ {
		val, err := b.next(bits)
		out[i] = val
		if err != nil {
			return i, err
		}
	}

	return i, nil
}

// GetBatchLevels fills out with bit-packed levels and returns the number equal
// to maxLevel.
func (b *BitReader) GetBatchLevels(bits uint, out []int16, maxLevel int16) (int, int64, error) {
	if bits > 16 {
		return 0, 0, errors.New("must be 16 bits or less per read")
	}

	var maxCount int64
	length := len(out)
	i := 0

	for ; i < length && b.bitoffset != 0; i++ {
		val, err := b.next(bits)
		if err != nil {
			return i, maxCount, err
		}
		level := int16(val)
		out[i] = level
		if level == maxLevel {
			maxCount++
		}
	}

	if _, err := b.reader.Seek(b.byteoffset, io.SeekStart); err != nil {
		return i, maxCount, err
	}
	for i < length {
		unpackSize := utils.Min(buflen, length-i)
		numUnpacked, err := unpack32(b.reader, b.unpackBuf[:unpackSize], int(bits))

		for k, val := range b.unpackBuf[:numUnpacked] {
			level := int16(val)
			out[i+k] = level
			if level == maxLevel {
				maxCount++
			}
		}
		i += numUnpacked
		b.byteoffset += int64(numUnpacked * int(bits) / 8)
		if err != nil {
			return i, maxCount, err
		}
		if numUnpacked == 0 {
			break
		}
	}

	if err := b.fillbuffer(); err != nil {
		return i, maxCount, err
	}
	for ; i < length; i++ {
		val, err := b.next(bits)
		if err != nil {
			return i, maxCount, err
		}
		level := int16(val)
		out[i] = level
		if level == maxLevel {
			maxCount++
		}
	}

	return i, maxCount, nil
}

// GetBatchBitmap fills out with bit-packed boolean values without unpacking
// them into a slice of integers. The input must be packed one bit per value.
func (b *BitReader) GetBatchBitmap(out []byte, outOffset, length int) (int, error) {
	if length == 0 {
		return 0, nil
	}

	i := 0
	// Match GetBatch's scalar handling when the current value is in the
	// middle of the reader's buffered word.
	for ; i < length && b.bitoffset != 0; i++ {
		val, err := b.next(1)
		if err != nil {
			return i, err
		}
		bitutil.SetBitTo(out, outOffset+i, val != 0)
	}

	if _, err := b.reader.Seek(b.byteoffset, io.SeekStart); err != nil {
		return i, err
	}

	// Read complete 32-value groups directly into the bitmap. Keep the
	// temporary packed data so a short final read cannot modify bits that were
	// not returned to the caller.
	for i < length {
		batch := min(buflen, length-i)
		batch = batch / 32 * 32
		if batch == 0 {
			break
		}

		packedBytes := batch / 8
		packed := arrow.Uint32Traits.CastToBytes(b.unpackBuf[:packedBytes/4])
		nread, err := io.ReadFull(b.reader, packed[:packedBytes])
		if err == io.ErrUnexpectedEOF && nread%4 == 0 {
			// Match unpack32's group-boundary behavior: an EOF between
			// complete 32-value groups is reported as io.EOF.
			err = io.EOF
		}
		completeBytes := nread / 4 * 4
		if completeBytes > 0 {
			if (outOffset+i)%8 == 0 {
				copy(out[(outOffset+i)/8:], packed[:completeBytes])
			} else {
				bitutil.CopyBitmap(packed, 0, completeBytes*8, out, outOffset+i)
			}
			i += completeBytes * 8
			b.byteoffset += int64(completeBytes)
		}
		if err != nil {
			return i, err
		}
	}

	if err := b.fillbuffer(); err != nil {
		return i, err
	}
	for ; i < length; i++ {
		val, err := b.next(1)
		if err != nil {
			return i, err
		}
		bitutil.SetBitTo(out, outOffset+i, val != 0)
	}

	return length, nil
}

// GetValue returns a single value that is bit packed using width as the number of bits
// and returns false if there weren't enough bits remaining.
func (b *BitReader) GetValue(width int) (uint64, bool) {
	v := make([]uint64, 1)
	n, _ := b.GetBatch(uint(width), v)
	return v[0], n == 1
}
