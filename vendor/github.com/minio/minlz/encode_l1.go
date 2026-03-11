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
	"encoding/hex"
	"fmt"
	"math/bits"
)

// hash6 returns the hash of the lowest 6 bytes of u to fit in a hash table with h bits.
// Preferably h should be a constant and should always be <64.
func hash6(u uint64, h uint8) uint32 {
	const prime6bytes = 227718039650203
	return uint32(((u << (64 - 48)) * prime6bytes) >> ((64 - h) & 63))
}

// encodeBlockGo encodes a non-empty src to a guaranteed-large-enough dst. It
// assumes that the varint-encoded length of the decompressed bytes has already
// been written.
//
// It also assumes that:
//
//	len(dst) >= MaxEncodedLen(len(src)) &&
//	minNonLiteralBlockSize <= len(src) && len(src) <= maxBlockSize
func encodeBlockGo(dst, src []byte) (d int) {
	// Initialize the hash table.
	const (
		tableBits    = 15
		maxTableSize = 1 << tableBits
		skipLog      = 6

		debug = debugEncode
	)
	if len(src) <= 65536 {
		return encodeBlockGo64K(dst, src)
	}
	// Having values inside the table is ~the same speed as looking up
	// - maybe slightly faster on bigger blocks.
	// We go for the smaller stack allocation for now.
	var table [maxTableSize]uint32

	// sLimit is when to stop looking for offset/length copies. The inputMargin
	// lets us use a fast path for emitLiteral in the main loop, while we are
	// looking for copies.
	sLimit := len(src) - inputMargin

	// Bail if we can't compress to at least this.
	dstLimit := len(src) - len(src)>>5 - 6

	// nextEmit is where in src the next emitLiteral should start from.
	nextEmit := 0

	// The encoded form must start with a literal, as there are no previous
	// bytes to copy, so we start looking for hash matches at s == 1.
	s := 1
	cv := load64(src, s)

	// We search for a repeat at -1, but don't output repeats when nextEmit == 0
	repeat := 1
	if debugEncode {
		fmt.Println("encodeBlockGo: Starting encode")
	}
	for {
		candidate := 0
		for {
			// Next src position to check
			nextS := s + (s-nextEmit)>>skipLog + 4
			if nextS > sLimit {
				goto emitRemainder
			}
			minSrcPos := s - maxCopy3Offset
			hash0 := hash6(cv, tableBits)
			hash1 := hash6(cv>>8, tableBits)
			candidate = int(table[hash0])
			candidate2 := int(table[hash1])
			table[hash0] = uint32(s)
			table[hash1] = uint32(s + 1)
			hash2 := hash6(cv>>16, tableBits)

			// Check repeat at offset checkRep.
			// Speed impact is very small.
			const checkRep = 1
			if uint32(cv>>(checkRep*8)) == load32(src, s-repeat+checkRep) {
				base := s + checkRep
				// Extend back
				for i := base - repeat; base > nextEmit && i > 0 && src[i-1] == src[base-1]; {
					i--
					base--
				}
				// Bail if we exceed the maximum size.
				if d+(base-nextEmit) > dstLimit {
					return 0
				}

				d += emitLiteral(dst[d:], src[nextEmit:base])
				if debugEncode {
					fmt.Println(nextEmit, "(lits) length:", base-nextEmit, "d-after:", d)
				}

				// Extend forward
				candidate := s - repeat + 4 + checkRep
				s += 4 + checkRep
				for s <= sLimit {
					if diff := load64(src, s) ^ load64(src, candidate); diff != 0 {
						s += bits.TrailingZeros64(diff) >> 3
						break
					}
					s += 8
					candidate += 8
				}
				if debug {
					// Validate match.
					if s <= candidate {
						panic("s <= candidate")
					}
					a := src[base:s]
					b := src[base-repeat : base-repeat+(s-base)]
					if !bytes.Equal(a, b) {
						panic("mismatch")
					}
				}
				d += emitRepeat(dst[d:], s-base)
				if debugEncode {
					fmt.Println(base, "(repeat) length:", s-base, "offset:", repeat, "d-after:", d)
				}
				nextEmit = s
				if s >= sLimit {
					goto emitRemainder
				}

				cv = load64(src, s)
				continue
			}

			if candidate >= minSrcPos && uint32(cv) == load32(src, candidate) {
				break
			}
			candidate = int(table[hash2])
			if candidate2 >= minSrcPos && uint32(cv>>8) == load32(src, candidate2) {
				table[hash2] = uint32(s + 2)
				candidate = candidate2
				s++
				break
			}
			table[hash2] = uint32(s + 2)
			if candidate >= minSrcPos && uint32(cv>>16) == load32(src, candidate) {
				s += 2
				break
			}

			cv = load64(src, nextS)
			s = nextS
		}

		// Extend backwards.
		// The top bytes will be rechecked to get the full match.
		for candidate > 0 && s > nextEmit && src[candidate-1] == src[s-1] {
			candidate--
			s--
		}

		// A 4-byte match has been found. We'll later see if more than 4 bytes match.
		base := s
		repeat = base - candidate

		// Extend the 4-byte match as long as possible.
		s += 4
		candidate += 4
		for s <= len(src)-8 {
			if diff := load64(src, s) ^ load64(src, candidate); diff != 0 {
				s += bits.TrailingZeros64(diff) >> 3
				break
			}
			s += 8
			candidate += 8
		}
		length := s - base
		if nextEmit != base {
			if base-nextEmit > maxCopy3Lits || repeat < minCopy2Offset {
				// Bail if we exceed the maximum size.
				// We will not exceed dstLimit with the other encodings.
				if d+(s-nextEmit) > dstLimit {
					return 0
				}
				d += emitLiteral(dst[d:], src[nextEmit:base])
				d += emitCopy(dst[d:], repeat, length)
			} else if repeat <= maxCopy2Offset {
				d += emitCopyLits2(dst[d:], src[nextEmit:base], repeat, length)
			} else {
				d += emitCopyLits3(dst[d:], src[nextEmit:base], repeat, length)
			}
		} else {
			d += emitCopy(dst[d:], repeat, length)
		}
		if debugEncode {
			fmt.Println(base, "(copy) length:", s-base, "offset:", repeat, "d-after:", d)
		}
		if debug {
			// Validate match.
			if s <= candidate {
				panic("s <= candidate")
			}
			a := src[base:s]
			b := src[base-repeat : base-repeat+(s-base)]
			if !bytes.Equal(a, b) {
				panic(fmt.Sprintf("mismatch: source: %v != target: %v", hex.EncodeToString(a), hex.EncodeToString(b)))
			}
		}

		for {
			nextEmit = s
			if s >= sLimit {
				goto emitRemainder
			}
			x := load64(src, s-2)

			if d > dstLimit {
				// Do we have space for more, if not bail.
				return 0
			}
			// Check for an immediate match, otherwise start search at s+1
			m2Hash := hash6(x, tableBits)
			x = x >> 16
			currHash := hash6(x, tableBits)
			candidate = int(table[currHash])
			table[m2Hash] = uint32(s - 2)
			table[currHash] = uint32(s)
			if debug && s == candidate {
				panic("s == candidate")
			}
			if s-candidate > maxCopy3Offset || uint32(x) != load32(src, candidate) {
				cv = load64(src, s+1)
				s++
				break
			}

			repeat = s - candidate
			base = s
			s += 4
			candidate += 4
			for s <= len(src)-8 {
				if diff := load64(src, s) ^ load64(src, candidate); diff != 0 {
					s += bits.TrailingZeros64(diff) >> 3
					break
				}
				s += 8
				candidate += 8
			}
			d += emitCopy(dst[d:], repeat, s-base)
			if debugEncode {
				fmt.Println(base, "(copy) length:", s-base, "offset:", repeat, "d-after:", d)
			}
		}
	}

emitRemainder:
	if nextEmit < len(src) {
		if debugEncode {
			fmt.Println(nextEmit, "emit remainder", len(src)-nextEmit, "d:", d)
		}
		// Bail if we exceed the maximum size.
		if d+len(src)-nextEmit > dstLimit {
			if debugEncode {
				fmt.Println("emit remainder", d+len(src)-nextEmit, " exceeds limit", dstLimit)
			}
			return 0
		}
		d += emitLiteral(dst[d:], src[nextEmit:])
	}
	return d
}

func encodeBlockGo64K(dst, src []byte) (d int) {
	// Initialize the hash table.
	const (
		tableBits    = 13
		maxTableSize = 1 << tableBits
		skipLog      = 5
		debug        = debugEncode
	)
	// Having values inside the table is ~the same speed as looking up
	// - maybe slightly faster on bigger blocks.
	// We go for the smaller stack allocation for now.
	var table [maxTableSize]uint16

	// sLimit is when to stop looking for offset/length copies. The inputMargin
	// lets us use a fast path for emitLiteral in the main loop, while we are
	// looking for copies.
	sLimit := len(src) - inputMargin

	// Bail if we can't compress to at least this.
	dstLimit := len(src) - len(src)>>5 - 6

	// nextEmit is where in src the next emitLiteral should start from.
	nextEmit := 0

	// The encoded form must start with a literal, as there are no previous
	// bytes to copy, so we start looking for hash matches at s == 1.
	s := 1
	cv := load64(src, s)

	// We search for a repeat at -1, but don't output repeats when nextEmit == 0
	repeat := 1
	if debugEncode {
		fmt.Println("encodeBlockGo: Starting encode")
	}
	for {
		candidate := 0
		for {
			// Next src position to check
			nextS := s + (s-nextEmit)>>skipLog + 4
			if nextS > sLimit {
				goto emitRemainder
			}
			hash0 := hash5(cv, tableBits)
			hash1 := hash5(cv>>8, tableBits)
			candidate = int(table[hash0])
			candidate2 := int(table[hash1])
			table[hash0] = uint16(s)
			table[hash1] = uint16(s + 1)
			hash2 := hash5(cv>>16, tableBits)

			// Check repeat at offset checkRep.
			// Speed impact is very small.
			const checkRep = 1
			if uint32(cv>>(checkRep*8)) == load32(src, s-repeat+checkRep) {
				base := s + checkRep
				// Extend back
				for i := base - repeat; base > nextEmit && i > 0 && src[i-1] == src[base-1]; {
					i--
					base--
				}
				// Bail if we exceed the maximum size.
				if d+(base-nextEmit) > dstLimit {
					return 0
				}

				d += emitLiteral(dst[d:], src[nextEmit:base])
				if debugEncode {
					fmt.Println(nextEmit, "(lits) length:", base-nextEmit, "d-after:", d)
				}

				// Extend forward
				candidate := s - repeat + 4 + checkRep
				s += 4 + checkRep
				for s <= sLimit {
					if diff := load64(src, s) ^ load64(src, candidate); diff != 0 {
						s += bits.TrailingZeros64(diff) >> 3
						break
					}
					s += 8
					candidate += 8
				}
				if debug {
					// Validate match.
					if s <= candidate {
						panic("s <= candidate")
					}
					a := src[base:s]
					b := src[base-repeat : base-repeat+(s-base)]
					if !bytes.Equal(a, b) {
						panic("mismatch")
					}
				}
				d += emitRepeat(dst[d:], s-base)
				if debugEncode {
					fmt.Println(base, "(repeat) length:", s-base, "offset:", repeat, "d-after:", d)
				}
				nextEmit = s
				if s >= sLimit {
					goto emitRemainder
				}

				cv = load64(src, s)
				continue
			}

			if uint32(cv) == load32(src, candidate) {
				break
			}
			candidate = int(table[hash2])
			if uint32(cv>>8) == load32(src, candidate2) {
				table[hash2] = uint16(s + 2)
				candidate = candidate2
				s++
				break
			}
			table[hash2] = uint16(s + 2)
			if uint32(cv>>16) == load32(src, candidate) {
				s += 2
				break
			}

			cv = load64(src, nextS)
			s = nextS
		}

		// Extend backwards.
		// The top bytes will be rechecked to get the full match.
		for candidate > 0 && s > nextEmit && src[candidate-1] == src[s-1] {
			candidate--
			s--
		}

		// A 4-byte match has been found. We'll later see if more than 4 bytes match.
		base := s
		repeat = base - candidate

		// Extend the 4-byte match as long as possible.
		s += 4
		candidate += 4
		for s <= len(src)-8 {
			if diff := load64(src, s) ^ load64(src, candidate); diff != 0 {
				s += bits.TrailingZeros64(diff) >> 3
				break
			}
			s += 8
			candidate += 8
		}
		length := s - base
		if nextEmit != base {
			if base-nextEmit > maxCopy2Lits || repeat < minCopy2Offset {
				// Bail if we exceed the maximum size.
				// We will not exceed dstLimit with the other encodings.
				if d+(s-nextEmit) > dstLimit {
					return 0
				}
				d += emitLiteral(dst[d:], src[nextEmit:base])
				d += emitCopy(dst[d:], repeat, length)
			} else {
				d += emitCopyLits2(dst[d:], src[nextEmit:base], repeat, length)
			}
		} else {
			d += emitCopy(dst[d:], repeat, length)
		}
		if debugEncode {
			fmt.Println(base, "(copy) length:", s-base, "offset:", repeat, "d-after:", d)
		}
		if debug {
			// Validate match.
			if s <= candidate {
				panic("s <= candidate")
			}
			a := src[base:s]
			b := src[base-repeat : base-repeat+(s-base)]
			if !bytes.Equal(a, b) {
				panic(fmt.Sprintf("mismatch: source: %v != target: %v", hex.EncodeToString(a), hex.EncodeToString(b)))
			}
		}

		for {
			nextEmit = s
			if s >= sLimit {
				goto emitRemainder
			}
			x := load64(src, s-2)

			if d > dstLimit {
				// Do we have space for more, if not bail.
				return 0
			}
			// Check for an immediate match, otherwise start search at s+1
			m2Hash := hash5(x, tableBits)
			x = x >> 16
			currHash := hash5(x, tableBits)
			candidate = int(table[currHash])
			table[m2Hash] = uint16(s - 2)
			table[currHash] = uint16(s)
			if debug && s == candidate {
				panic("s == candidate")
			}
			if uint32(x) != load32(src, candidate) {
				cv = load64(src, s+1)
				s++
				break
			}

			repeat = s - candidate
			base = s
			s += 4
			candidate += 4
			for s <= len(src)-8 {
				if diff := load64(src, s) ^ load64(src, candidate); diff != 0 {
					s += bits.TrailingZeros64(diff) >> 3
					break
				}
				s += 8
				candidate += 8
			}
			d += emitCopy(dst[d:], repeat, s-base)
			if debugEncode {
				fmt.Println(base, "(copy) length:", s-base, "offset:", repeat, "d-after:", d)
			}
		}
	}

emitRemainder:
	if nextEmit < len(src) {
		if debugEncode {
			fmt.Println(nextEmit, "emit remainder", len(src)-nextEmit, "d:", d)
		}
		// Bail if we exceed the maximum size.
		if d+len(src)-nextEmit > dstLimit {
			if debugEncode {
				fmt.Println("emit remainder", d+len(src)-nextEmit, " exceeds limit", dstLimit)
			}
			return 0
		}
		d += emitLiteral(dst[d:], src[nextEmit:])
	}
	return d
}
