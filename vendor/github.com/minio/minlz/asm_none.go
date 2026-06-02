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

//go:build !amd64 || appengine || !gc || noasm || purego

package minlz

import (
	"fmt"
	"math/bits"
)

const hasAsm = false

// encodeBlock encodes a non-empty src to a guaranteed-large-enough dst. It
// assumes that the varint-encoded length of the decompressed bytes has already
// been written.
//
// It also assumes that:
//
//	len(dst) >= MaxEncodedLen(len(src))
func encodeBlock(dst, src []byte) (d int) {
	if len(src) < minNonLiteralBlockSize {
		return 0
	}
	return encodeBlockGo(dst, src)
}

// encodeBlockBetter encodes a non-empty src to a guaranteed-large-enough dst. It
// assumes that the varint-encoded length of the decompressed bytes has already
// been written.
//
// It also assumes that:
//
//	len(dst) >= MaxEncodedLen(len(src))
func encodeBlockBetter(dst, src []byte) (d int) {
	return encodeBlockBetterGo(dst, src)
}

// emitLiteral writes a literal chunk and returns the number of bytes written.
//
// It assumes that:
//
//	dst is long enough to hold the encoded bytes
//	0 <= len(lit) && len(lit) <= math.MaxUint32
func emitLiteral(dst, lit []byte) int {
	// 0-28: Length 1 -> 29
	// 29: Length (Read 1) + 1
	// 30: Length (Read 2) + 1
	// 31: Length (Read 3) + 1
	if len(lit) == 0 {
		return 0
	}
	if debugEncode {
		fmt.Println("(literal)", len(lit))
	}
	i, n := 0, uint(len(lit)-1)

	switch {
	case n < 29:
		store8(dst, 0, uint8(n)<<3|tagLiteral)
		i = 1
	case n < 1<<8+29:
		store8(dst, 1, uint8(n-29))
		store8(dst, 0, 29<<3|tagLiteral)
		i = 2
	case n < 1<<16+29:
		n -= 29
		dst[2] = uint8(n >> 8)
		dst[1] = uint8(n)
		dst[0] = 30<<3 | tagLiteral
		i = 3
	case n < 1<<24+29:
		n -= 29
		dst[3] = uint8(n >> 16)
		dst[2] = uint8(n >> 8)
		dst[1] = uint8(n)
		dst[0] = 31<<3 | tagLiteral
		i = 4
	default:
		panic("literal block too long")
	}
	return i + copy(dst[i:], lit)
}

// emitRepeat writes a repeat chunk and returns the number of bytes written.
func emitRepeat(dst []byte, length int) int {
	// Repeat offset, make length cheaper
	if debugEncode {
		fmt.Println("(repeat)", length)
	}

	if debugEncode && length < 0 {
		panic(fmt.Sprintf("invalid length %d", length))
	}
	if length < 30 {
		store8(dst, 0, uint8(length-1)<<3|tagRepeat)
		return 1
	}
	length -= 30
	if length < 256 {
		store8(dst, 1, uint8(length>>0))
		store8(dst, 0, 29<<3|tagRepeat)
		return 2
	}

	if length < 65536 {
		dst[2] = uint8(length >> 8)
		dst[1] = uint8(length >> 0)
		dst[0] = 30<<3 | tagRepeat
		return 3
	}
	dst[3] = uint8(length >> 16)
	dst[2] = uint8(length >> 8)
	dst[1] = uint8(length >> 0)
	dst[0] = 31<<3 | tagRepeat
	return 4
}

// encodeCopy3 encodes a copy operation with 24 bit offset.
// length must be at least 1 and < 1<<24
func encodeCopy3(dst []byte, offset, length, lits int) int {
	// Repeat offset, make length cheaper
	length -= 4
	if debugEncode && length < 0 {
		panic(fmt.Sprintf("invalid length %d", length))
	}
	if debugEncode && offset < 65536 {
		panic(fmt.Sprintf("invalid offset %d", offset))
	}

	// Encode offset
	var encoded uint32
	encoded = uint32(offset-65536)<<11 | tagCopy3 | uint32(lits<<3)

	if length <= 60 {
		encoded |= uint32(length << 5)
		store32(dst, 0, encoded)
		return 4
	}
	length -= 60
	if length < 256 {
		store8(dst, 4, uint8(length>>0))
		encoded |= 61 << 5
		store32(dst, 0, encoded)
		return 5
	}

	if length < 65536 {
		encoded |= 62 << 5
		dst[5] = uint8(length >> 8)
		dst[4] = uint8(length >> 0)
		store32(dst, 0, encoded)
		return 6
	}
	encoded |= 63 << 5
	dst[6] = uint8(length >> 16)
	dst[5] = uint8(length >> 8)
	dst[4] = uint8(length >> 0)
	store32(dst, 0, encoded)
	return 7
}

// emitCopy writes a copy chunk and returns the number of bytes written.
//
// It assumes that:
//
//	dst is long enough to hold the encoded bytes
func emitCopy(dst []byte, offset, length int) int {
	if debugEncode {
		fmt.Println("(copy) length:", length, "offset:", offset)
		if offset == 0 || offset > maxCopy3Offset {
			panic(fmt.Sprintf("(emitCopy) invalid offset %d", offset))
		}
	}

	if offset > maxCopy2Offset {
		// return encodeCopy3(dst, offset, length, 0) expanded...
		// Repeat offset, make length cheaper
		length -= 4
		if debugEncode && length < 0 {
			panic(fmt.Sprintf("invalid length %d", length))
		}
		if debugEncode && offset < 65536 {
			panic(fmt.Sprintf("invalid offset %d", offset))
		}

		// Encode offset
		var encoded uint32
		encoded = uint32(offset-65536)<<11 | tagCopy3

		if length <= 60 {
			encoded |= uint32(length << 5)
			store32(dst, 0, encoded)
			return 4
		}
		length -= 60
		if length < 256 {
			dst[4] = uint8(length >> 0)
			encoded |= 61 << 5
			store32(dst, 0, encoded)
			return 5
		}

		if length < 65536 {
			encoded |= 62 << 5
			store16(dst[:], 4, uint16(length))
			store32(dst, 0, encoded)
			return 6
		}
		encoded |= 63 << 5
		dst[6] = uint8(length >> 16)
		dst[5] = uint8(length >> 8)
		dst[4] = uint8(length >> 0)
		store32(dst, 0, encoded)
		return 7
	}

	// Small offset. Use copy1
	if offset <= maxCopy1Offset {
		offset--
		if length < 15+4 {
			x := uint16(offset<<6) | uint16(length-4)<<2 | tagCopy1
			store16(dst, 0, x)
			return 2
		}
		if length < 256+18 {
			x := uint16(offset<<6) | (uint16(15)<<2 | tagCopy1)
			store16(dst, 0, x)
			dst[2] = uint8(length - 18)
			return 3
		}
		// Encode as Copy1 and repeat
		x := uint16(offset<<6) | uint16(14)<<2 | tagCopy1
		store16(dst, 0, x)
		return 2 + emitRepeat(dst[2:], length-18)
	}

	return encodeCopy2(dst, offset, length)
}

// emitCopyLits2 emit 2 byte offset copy with literals.
// len(lits) must be 1 - 4.
// The caller should only call when the offset can contain a literal encoding.
// Longer copies are emitted as copy+repeat.
func emitCopyLits2(dst, lits []byte, offset, length int) int {
	if debugEncode {
		if offset < minCopy2Offset || offset > maxCopy2Offset {
			panic(fmt.Sprintf("invalid offset %d", offset))
		}
		if len(lits) > maxCopy2Lits {
			panic(fmt.Sprintf("invalid literal count %d", len(lits)))
		}
		fmt.Println("(copy2) lits:", len(lits), "length:", length, "offset:", offset)
	}
	offset -= minCopy2Offset
	// Emit as literal + 2 byte offset code.
	// If longer than 11 use repeat for remaining.
	length -= 4
	const copy2LitMaxLenRaw = copy2LitMaxLen - 4
	if length > copy2LitMaxLenRaw {
		store16(dst, 1, uint16(offset))
		store8(dst, 0, tagCopy2Fused|uint8((copy2LitMaxLenRaw)<<5)|uint8(len(lits)-1)<<3)
		n := copy(dst[3:], lits) + 3
		return n + emitRepeat(dst[n:], length-copy2LitMaxLenRaw)
	}
	store16(dst, 1, uint16(offset))
	store8(dst, 0, tagCopy2Fused|uint8(length<<5)|uint8(len(lits)-1)<<3)
	return copy(dst[3:], lits) + 3
}

// emitCopyLits3 emit a 3 byte offset copy with literals.
// len(lits) must be 1 - 3.
// The caller should only call when the offset can contain a literal encoding.
func emitCopyLits3(dst, lits []byte, offset, length int) int {
	if debugEncode {
		fmt.Println("(copy3) lits:", len(lits), "length:", length, "offset:", offset)
		if offset > maxCopy3Offset {
			panic(fmt.Sprintf("(emitCopyLits3) invalid offset %d", offset))
		}
	}
	n := encodeCopy3(dst, offset, length, len(lits))
	copy(dst[n:], lits)
	return n + len(lits)
}

// matchLen returns how many bytes match in a and b
//
// It assumes that:
//
//	len(a) <= len(b)
func matchLen(a []byte, b []byte) int {
	b = b[:len(a)]
	var checked int
	for len(a)-checked >= 8 {
		if diff := load64(a, checked) ^ load64(b, checked); diff != 0 {
			return checked + (bits.TrailingZeros64(diff) >> 3)
		}
		checked += 8
	}
	a = a[checked:]
	b = b[checked:]
	b = b[:len(a)]
	for i := range a {
		if a[i] != b[i] {
			return int(i) + checked
		}
	}
	return len(a) + checked
}

// cvtLZ4Block converts an LZ4 block to MinLZ
func cvtLZ4BlockAsm(dst []byte, src []byte) (uncompressed int, dstUsed int) {
	panic("not implemented")
}
