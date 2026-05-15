// Copyright 2025 MinIO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package minlz

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
)

const (
	// LevelFastest is the fastest compression level.
	LevelFastest = 1

	// LevelBalanced is the balanced compression level.
	// This is targeted to be approximately half the speed of LevelFastest.
	LevelBalanced = 2

	// LevelSmallest will attempt the best possible compression.
	// There is no speed target for this level.
	LevelSmallest = 3

	// Internal use only
	copyLitBits = 2

	maxCopy1Offset = 1024

	minCopy2Offset = 64
	maxCopy2Offset = minCopy2Offset + 65535 // 2MiB
	copy2LitMaxLen = 7 + 4                  // max length
	maxCopy2Lits   = 1 << copyLitBits
	minCopy2Length = 64

	maxCopy3Lits   = 1<<copyLitBits - 1
	minCopy3Offset = 65536
	maxCopy3Offset = 2<<20 + 65535 // 2MiB
	minCopy3Length = 64
)

// Encode returns the encoded form of src. The returned slice may be a sub-
// slice of dst if dst was large enough to hold the entire encoded block.
// Otherwise, a newly allocated slice will be returned.
//
// The dst and src must not overlap. It is valid to pass a nil dst.
//
// The blocks will require the same amount of memory to decode as encoding,
// and does not make for concurrent decoding.
// Also note that blocks do not contain CRC information, so corruption may be undetected.
//
// If you need to encode larger amounts of data, consider using
// the streaming interface which gives all of these features.
func Encode(dst, src []byte, level int) ([]byte, error) {
	if n := MaxEncodedLen(len(src)); n < 0 {
		return nil, ErrTooLarge
	} else if cap(dst) < n {
		dst = make([]byte, n)
	} else {
		dst = dst[:n]
	}

	if len(src) < minNonLiteralBlockSize {
		return encodeUncompressed(dst[:0], src), nil
	}

	dst[0] = 0
	d := 1
	// The block starts with the varint-encoded length of the decompressed bytes.
	d += binary.PutUvarint(dst[d:], uint64(len(src)))

	var n int
	switch level {
	case LevelFastest:
		n = encodeBlock(dst[d:], src)
	case LevelBalanced:
		n = encodeBlockBetter(dst[d:], src)
	case LevelSmallest:
		n = encodeBlockBest(dst[d:], src, nil)
	default:
		return nil, ErrInvalidLevel
	}

	if n > 0 {
		if debugValidateBlocks {
			block := dst[d : d+n]
			dst := make([]byte, len(src), len(src))
			ret := minLZDecode(dst, block)
			if !bytes.Equal(dst, src) {
				n := matchLen(dst, src)
				x := crc32.ChecksumIEEE(src)
				name := fmt.Sprintf("errs/block-%08x", x)
				fmt.Println(name, "mismatch at pos", n)
				os.WriteFile(name+"input.bin", src, 0644)
				os.WriteFile(name+"decoded.bin", dst, 0644)
				os.WriteFile(name+"compressed.bin", block, 0644)
			}
			if ret != 0 {
				panic("decode error")
			}
		}
		d += n
		return dst[:d], nil
	}
	// Not compressible, emit as uncompressed.
	return encodeUncompressed(dst[:0], src), nil
}

// AppendEncoded will append the encoded version of src to dst.
// If dst has MaxEncodedLen(len(src)) capacity left it will be done without allocation.
// See Encode for more information.
func AppendEncoded(dst, src []byte, level int) ([]byte, error) {
	wantDst := MaxEncodedLen(len(src))
	if wantDst < 0 {
		return nil, ErrTooLarge
	}
	d := len(dst)
	if cap(dst) < wantDst+d {
		dst = append(dst, make([]byte, wantDst)...)
	}
	dst = dst[:d+wantDst]
	res, err := Encode(dst[d:], src, level)
	if err != nil {
		return nil, err
	}
	if len(res) > wantDst {
		panic("overflow")
	}
	return dst[:d+len(res)], nil
}

// TryEncode returns the encoded form of src, if compressible.
// The same limitations apply as the Encode function.
// If the block is incompressible or another error occurs,
// nil will be returned.
func TryEncode(dst, src []byte, level int) []byte {
	if n := MaxEncodedLen(len(src)); n < 0 {
		return nil
	} else if cap(dst) < n {
		dst = make([]byte, n)
	} else {
		dst = dst[:n]
	}

	if len(src) < minNonLiteralBlockSize {
		return nil
	}

	dst[0] = 0
	d := 1
	// The block starts with the varint-encoded length of the decompressed bytes.
	d += binary.PutUvarint(dst[d:], uint64(len(src)))

	var n int
	switch level {
	case LevelFastest:
		n = encodeBlock(dst[d:], src)
	case LevelBalanced:
		n = encodeBlockBetter(dst[d:], src)
	case LevelSmallest:
		n = encodeBlockBest(dst[d:], src, nil)
	default:
		return nil
	}

	if n > 0 && d+n < len(src) {
		d += n
		return dst[:d]
	}
	// Not compressible
	return nil
}

// inputMargin is the minimum number of extra input bytes to keep, inside
// encodeBlock's inner loop. On some architectures, this margin lets us
// implement a fast path for emitLiteral, where the copy of short (<= 16 byte)
// literals can be implemented as a single load to and store from a 16-byte
// register. That literal's actual length can be as short as 1 byte, so this
// can copy up to 15 bytes too much, but that's OK as subsequent iterations of
// the encoding loop will fix up the copy overrun, and this inputMargin ensures
// that we don't overrun the dst and src buffers.
const inputMargin = 8

// minNonLiteralBlockSize is the minimum size of the input to encodeBlock that
// will be accepted by the encoder.
const minNonLiteralBlockSize = 16

// encodeUncompressed will append src to dst as uncompressed data and return it.
func encodeUncompressed(dst, src []byte) []byte {
	if len(src) == 0 {
		return append(dst, 0)
	}
	return append(append(dst, 0, 0), src...)
}

// MaxEncodedLen returns the maximum length of a snappy block, given its
// uncompressed length.
//
// It will return a negative value if srcLen is too large to encode.
func MaxEncodedLen(srcLen int) int {
	n := uint64(srcLen)
	if n > MaxBlockSize {
		return -1
	}
	if srcLen == 0 {
		return 1
	}
	// Maximum overhead is 2 bytes.
	return int(n + 2)
}

// encodeCopy2 encodes a length and returns the number of bytes written.
func encodeCopy2(dst []byte, offset, length int) int {
	// Repeat offset, make length cheaper
	length -= 4
	offset -= minCopy2Offset
	if debugEncode {
		if length < 0 {
			panic(fmt.Sprintf("invalid length %d", length))
		}
		if offset < 0 {
			panic(fmt.Sprintf("invalid offset %d", offset))
		}
	}
	store16(dst, 1, uint16(offset))
	if length <= 60 {
		store8(dst, 0, uint8(length)<<2|tagCopy2)
		return 3
	}
	length -= 60
	if length < 256 {
		store8(dst, 3, uint8(length>>0))
		store8(dst, 0, 61<<2|tagCopy2)
		return 4
	}

	if length < 65536 {
		dst[4] = uint8(length >> 8)
		dst[3] = uint8(length >> 0)
		dst[0] = 62<<2 | tagCopy2
		return 5
	}
	dst[5] = uint8(length >> 16)
	dst[4] = uint8(length >> 8)
	dst[3] = uint8(length >> 0)
	dst[0] = 63<<2 | tagCopy2
	return 6
}

// emitLiteralSizeN returns the overhead of emitting n literal.
func emitLiteralSizeN(n int) int {
	if n == 0 {
		return 0
	}
	switch {
	case n <= 29:
		return 1
	case n < 29+(1<<8):
		return 2
	case n < 29+(1<<16):
		return 3
	default:
		return 4
	}
}
