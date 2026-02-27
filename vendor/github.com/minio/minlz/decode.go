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
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/klauspost/compress/s2"
)

const (
	decodeErrCodeCorrupt = 1
)

var (
	// ErrCorrupt reports that the input is invalid.
	ErrCorrupt = errors.New("minlz: corrupt input")
	// ErrCRC reports that the input failed CRC validation (streams only)
	ErrCRC = errors.New("minlz: corrupt input, crc mismatch")
	// ErrTooLarge reports that the uncompressed length is too large.
	ErrTooLarge = errors.New("minlz: decoded block is too large")
	// ErrUnsupported reports that the input isn't supported.
	ErrUnsupported = errors.New("minlz: unsupported input")
	// ErrInvalidLevel is returned when an invalid compression level is requested.
	ErrInvalidLevel = errors.New("minlz: invalid compression level")
)

// Decode returns the decoded form of src. The returned slice may be a sub-
// slice of dst if dst was large enough to hold the entire decoded block.
// Otherwise, a newly allocated slice will be returned.
//
// This decoder has automatic fallback to Snappy/S2.
// To reject fallback check with IsMinLZ.
//
// The dst and src must not overlap. It is valid to pass a nil dst.
func Decode(dst, src []byte) ([]byte, error) {
	isMLZ, lits, block, dLen, err := isMinLZ(src)
	if err != nil {
		return nil, err
	}
	if lits {
		return append(dst[:0], block...), nil
	}

	if !isMLZ {
		if l, _ := s2.DecodedLen(block); l > s2.MaxBlockSize {
			return nil, ErrTooLarge
		}
		if dst, err := s2.Decode(dst, block); err != nil {
			return nil, ErrCorrupt
		} else {
			return dst, nil
		}
	}
	if dLen <= cap(dst) {
		dst = dst[:dLen]
	} else {
		dst = make([]byte, dLen, dLen)
	}
	if minLZDecode(dst, block) != 0 {
		return dst, ErrCorrupt
	}
	return dst, nil
}

// AppendDecoded will append the decoded version of src to dst.
// If the decoded content cannot fit within dst, it will cause an allocation.
// This decoder has automatic fallback to Snappy/S2.
// To reject fallback check with IsMinLZ.
// The dst and src must not overlap. It is valid to pass a nil dst.
func AppendDecoded(dst, src []byte) ([]byte, error) {
	dLen, err := DecodedLen(src)
	if err != nil {
		return dst, err
	}
	if dLen > cap(dst)-len(dst) {
		oLen := len(dst)
		dst = append(dst, make([]byte, dLen)...)
		dst = dst[:oLen]
	}
	d, err := Decode(dst[len(dst):], src)
	if err != nil {
		return dst, err
	}
	if len(d) != dLen {
		return dst, ErrCorrupt
	}
	return dst[:len(dst)+dLen], nil
}

// DecodedLen returns the length of the decoded block.
// This length will never be exceeded when decoding a block.
func DecodedLen(src []byte) (int, error) {
	_, _, _, v, err := isMinLZ(src)
	return v, err
}

// IsMinLZ returns whether the block is a minlz block
// and returns the size of the decompressed block.
func IsMinLZ(src []byte) (ok bool, size int, err error) {
	ok, _, _, size, err = isMinLZ(src)
	return
}

// IsMinLZ returns true if the block is a minlz block.
func isMinLZ(src []byte) (ok, literals bool, block []byte, size int, err error) {
	if len(src) <= 1 {
		if len(src) == 0 {
			err = ErrCorrupt
			return
		}
		if src[0] == 0 {
			// Size 0 block. Could be MinLZ.
			return true, true, src[1:], 0, nil
		}
	}
	if src[0] != 0 {
		// Older - Snappy or S2...
		v, _, err := decodedLen(src)
		return false, false, src, v, err
	}
	src = src[1:]
	v, headerLen, err := decodedLen(src)
	if err != nil {
		return false, false, nil, 0, err
	}
	if v > MaxBlockSize {
		return false, false, nil, 0, ErrTooLarge
	}
	src = src[headerLen:]
	if len(src) == 0 {
		return false, false, nil, 0, ErrCorrupt
	}
	if v == 0 {
		// Literals, rest of block...
		return true, true, src, len(src), nil
	}
	if v < len(src) {
		return false, false, src, v, ErrCorrupt
	}
	return true, false, src, v, nil
}

// decodedLen returns the length of the decoded block and the number of bytes
// that the length header occupied.
func decodedLen(src []byte) (blockLen, headerLen int, err error) {
	v, n := binary.Uvarint(src)
	if n <= 0 || v > 0xffffffff {
		return 0, 0, ErrCorrupt
	}

	const wordSize = 32 << (^uint(0) >> 32 & 1)
	if wordSize == 32 && v > 0x7fffffff {
		return 0, 0, ErrTooLarge
	}
	return int(v), n, nil
}

// minLZDecode writes the decoding of src to dst. It assumes that the varint-encoded
// length of the decompressed bytes has already been read, and that len(dst)
// equals that length.
//
// It returns 0 on success or a decodeErrCodeXxx error code on failure.
func minLZDecodeGo(dst, src []byte) int {
	const debug = debugDecode
	const debugErrors = false || debug
	if debug {
		fmt.Println("Starting decode, src:", len(src), "dst len:", len(dst))
	}
	var d, s, length int
	offset := 1

	// As long as we can read at least 11 bytes... (longest code possible +4 lits)
	for s < len(src)-11 {
		// Maximum input needed.
		if debug {
			//fmt.Printf("in:%x, tag: %02b va:%x - src: %d, dst: %d\n", src[s], src[s]&3, src[s]>>2, s, d)
		}

		switch load8(src, s) & 0x03 {
		case tagLiteral:
			v := load8(src, s)
			x := v >> 3
			switch {
			case x < 29:
				s++
				length = int(x) + 1
			case x == 29: // 1 byte length
				length = 30 + int(load8(src, s+1))
				s += 2
			case x == 30: // 2 byte length
				length = 30 + int(load16(src, s+1))
				s += 3
			default:
				// case x == 31: // 3 byte length
				// Load as 32 bit and shift down.
				length = 30 + int(load32(src, s)>>8)
				s += 4
			}
			if v&4 != 0 {
				// repeat
				if debug {
					fmt.Print(d, ": (repeat)")
				}
				goto docopy
			}
			if length > len(dst)-d || length > len(src)-s {
				if debugErrors {
					fmt.Println("corrupt: lit size", length, "dst avail:", len(dst)-d, "src avail:", len(src)-s, "dst pos:", d)
				}
				return decodeErrCodeCorrupt
			}
			if debug {
				fmt.Print(d, ": (literals) length: ", length, " d-after: ", d+length, "\n")
			}

			copy(dst[d:], src[s:s+length])
			d += length
			s += length
			continue

		case tagCopy1:
			if debug {
				fmt.Print(d, ": (copy1) ")
			}
			length = int(load8(src, s)) >> 2 & 15
			offset = int(load16(src, s)>>6) + 1
			if length == 15 {
				// Extended length with 1 byte
				length = int(load8(src, s+2)) + 18
				s += 3
			} else {
				length += 4
				s += 2
			}
		case tagCopy2:
			if debug {
				fmt.Print(d, ": (copy2)")
			}
			length = int(load8(src, s)) >> 2
			offset = int(load16(src, s+1))
			if length <= 60 {
				length += 4
				s += 3
			} else {
				switch length {
				case 61: // 1 byte + 64
					length = int(load8(src, s+3)) + 64
					s += 4
				case 62: // 2 bytes + 64
					length = int(load16(src, s+3)) + 64
					s += 5
				case 63: // 3 bytes + 64
					// Load as 32 bit and shift down.
					length = int(load32(src, s+2)>>8) + 64
					s += 6
				}
			}
			offset += minCopy2Offset
		case 0x3:
			val := load32(src, s)
			isCopy3 := val&4 != 0
			litLen := int(val>>3) & 3
			if !isCopy3 {
				if debug {
					fmt.Print(d, ": (copy2 fused) ")
				}
				length = 4 + int(val>>5)&7
				offset = int(val>>8)&65535 + minCopy2Offset
				s += 3
				litLen++
			} else {
				lengthTmp := (val >> 5) & 63
				offset = int(val>>11) + minCopy3Offset
				if debug {
					fmt.Print(d, ": (copy3)")
				}
				switch {
				case lengthTmp < 61:
					length = int(lengthTmp) + 4
					s += 4
				case lengthTmp == 61:
					length = int(load8(src, s+4)) + 64
					s += 5
				case lengthTmp == 62:
					length = int(load16(src, s+4)) + 64
					s += 6
				case lengthTmp == 63:
					length = int(load32(src, s+3)>>8) + 64
					s += 7
				default:
					panic("unreachable")
				}
			}
			if litLen > 0 {
				if debug {
					fmt.Print(" - lits: ", litLen, " ")
				}
				if len(dst)-d < 4 {
					if debugErrors {
						fmt.Println("corrupt: lit size", length+litLen, "dst avail:", len(dst)-d, "src avail:", len(src)-s, "dst pos:", d)
					}
					return decodeErrCodeCorrupt
				}
				// We will always have room to read
				store32(dst, d, load32(src, s))
				s += litLen
				d += litLen
			}
		}
	docopy:
		if d < offset || length > len(dst)-d {
			if debugErrors {
				fmt.Println("corrupt: match, length", length, "offset:", offset, "dst avail:", len(dst)-d, "dst pos:", d)
			}
			return decodeErrCodeCorrupt
		}

		if debug {
			fmt.Println("- copy, length:", length, "offset:", offset, "d-after:", d+length, "s-after:", s)
		}

		// Copy from an earlier sub-slice of dst to a later sub-slice.
		// If no overlap, use the built-in copy:
		if offset > length {
			copy(dst[d:d+length], dst[d-offset:])
			d += length
			continue
		}

		// Unlike the built-in copy function, this byte-by-byte copy always runs
		// forwards, even if the slices overlap. Conceptually, this is:
		//
		// d += forwardCopy(dst[d:d+length], dst[d-offset:])
		//
		// We align the slices into a and b and show the compiler they are the same size.
		// This allows the loop to run without bounds checks.
		a := dst[d : d+length]
		b := dst[d-offset:]
		b = b[:len(a)]
		for i := range a {
			a[i] = b[i]
		}
		d += length
	}

	// Remaining with extra checks...
	for s < len(src) {
		if debug {
			//fmt.Printf("in:%x, tag: %02b va:%x - src: %d, dst: %d\n", src[s], src[s]&3, src[s]>>2, s, d)
		}
		switch load8(src, s) & 0x03 {
		case tagLiteral:
			v := load8(src, s)
			x := v >> 3
			switch {
			case x < 29:
				s++
				length = int(x + 1)
			case x == 29:
				s += 2
				if s > len(src) {
					if debugErrors {
						fmt.Println("(1)read out of bounds, src pos:", s, "dst pos:", d)
					}
					return decodeErrCodeCorrupt
				}
				length = int(uint32(src[s-1]) + 30)
			case x == 30:
				s += 3
				if s > len(src) {
					if debugErrors {
						fmt.Println("(2)read out of bounds, src pos:", s, "dst pos:", d)
					}
					return decodeErrCodeCorrupt
				}
				length = int(uint32(src[s-2]) | uint32(src[s-1])<<8 + 30)
			default:
				//			case x == 31:
				s += 4
				if s > len(src) {
					if debugErrors {
						fmt.Println("(3)read out of bounds, src pos:", s, "dst pos:", d)
					}
					return decodeErrCodeCorrupt
				}
				length = int(uint32(src[s-3]) | uint32(src[s-2])<<8 | uint32(src[s-1])<<16 + 30)
			}

			if v&4 != 0 {
				// repeat
				if debug {
					fmt.Print(d, ": (repeat)")
				}
				goto doCopy2
			}

			if length > len(dst)-d || length > len(src)-s {
				if debugErrors {
					fmt.Println("corrupt: lit size", length, "dst avail:", len(dst)-d, "src avail:", len(src)-s, "dst pos:", d)
				}
				return decodeErrCodeCorrupt
			}
			if debug {
				fmt.Print(d, ": (literals), length: ", length, " d-after: ", d+length)
			}

			copy(dst[d:], src[s:s+length])
			d += length
			s += length

			if debug {
				fmt.Println("")
			}
			continue

		case tagCopy1:
			if debug {
				fmt.Print(d, ": (copy1 -wut?) ")
			}
			s += 2
			if s > len(src) {
				if debugErrors {
					fmt.Println("(5-1)read out of bounds, src pos:", s, "dst pos:", d)
				}
				return decodeErrCodeCorrupt
			}

			length = int(src[s-2]) >> 2 & 15
			offset = int(load16(src, s-2)>>6) + 1
			if length == 15 {
				s++
				if s > len(src) {
					if debugErrors {
						fmt.Println("(5)read out of bounds, src pos:", s, "dst pos:", d)
					}
					return decodeErrCodeCorrupt
				}
				length = int(src[s-1]) + 18
			} else {
				length += 4
			}
		case tagCopy2:
			if debug {
				fmt.Print(d, ": (copy2) ")
			}
			s += 3
			if uint(s) > uint(len(src)) {
				if debugErrors {
					fmt.Println("(7)read out of bounds, src pos:", s, "dst pos:", d)
				}
				return decodeErrCodeCorrupt
			}
			length = int(src[s-3]) >> 2
			offset = int(uint32(src[s-2]) | uint32(src[s-1])<<8)
			if length <= 60 {
				length += 4
			} else {
				switch length {
				case 61:
					s++
					if uint(s) > uint(len(src)) {
						if debugErrors {
							fmt.Println("(8)read out of bounds, src pos:", s, "dst pos:", d)
						}
						return decodeErrCodeCorrupt
					}
					length = int(src[s-1]) + 64
				case 62:
					s += 2
					if uint(s) > uint(len(src)) {
						if debugErrors {
							fmt.Println("(9)read out of bounds, src pos:", s, "dst pos:", d)
						}
						return decodeErrCodeCorrupt
					}
					length = int(src[s-2]) | int(src[s-1])<<8 + 64
				case 63:
					s += 3
					if s > len(src) {
						if debugErrors {
							fmt.Println("(10)read out of bounds, src pos:", s, "dst pos:", d)
						}
						return decodeErrCodeCorrupt
					}
					length = int(src[s-3]) | int(src[s-2])<<8 | int(src[s-1])<<16 + 64
				}
			}
			offset += minCopy2Offset
		case 0x3:
			s += 4
			if s > len(src) {
				if debugErrors {
					fmt.Println("(11)read out of bounds, src pos:", s, "dst pos:", d)
				}
				return decodeErrCodeCorrupt
			}
			val := load32(src, s-4)
			isCopy3 := val&4 != 0
			litLen := int(val>>3) & 3
			if !isCopy3 {
				if debug {
					fmt.Print(d, ": (copy2 fused) ")
				}
				length = 4 + int(val>>5)&7
				offset = int(val>>8)&65535 + minCopy2Offset
				s--
				litLen++
			} else {
				if debug {
					fmt.Print(d, ": (copy3) ")
				}
				lengthTmp := (val >> 5) & 63
				offset = int(val>>11) + minCopy3Offset
				if lengthTmp >= 61 {
					switch lengthTmp {
					case 61:
						s++
						if s > len(src) {
							if debugErrors {
								fmt.Println("(13)read out of bounds, src pos:", s, "dst pos:", d)
							}
							return decodeErrCodeCorrupt
						}
						length = int(src[s-1]) + 64
					case 62:
						s += 2
						if s > len(src) {
							if debugErrors {
								fmt.Println("(14)read out of bounds, src pos:", s, "dst pos:", d)
							}
							return decodeErrCodeCorrupt
						}
						length = (int(src[s-2]) | int(src[s-1])<<8) + 64
					default:
						s += 3
						if s > len(src) {
							if debugErrors {
								fmt.Println("(15)read out of bounds, src pos:", s, "dst pos:", d)
							}
							return decodeErrCodeCorrupt
						}
						length = int(src[s-3]) | int(src[s-2])<<8 | int(src[s-1])<<16 + 64
					}
				} else {
					length = int(lengthTmp + 4)
				}
			}

			if litLen > 0 {
				if litLen > len(dst)-d || s+litLen > len(src) {
					if debugErrors {
						fmt.Println("corrupt: lits size", litLen, "dst avail:", len(dst)-d, "src avail:", len(src)-s)
					}
					return decodeErrCodeCorrupt
				}
				copy(dst[d:], src[s:s+litLen])
				d += litLen
				s += litLen
			}
		}

	doCopy2:
		if offset <= 0 || d < offset || length > len(dst)-d {
			if debugErrors {
				fmt.Println("corrupt: match, length", length, "offset:", offset, "dst avail:", len(dst)-d, "dst pos:", d)
			}
			return decodeErrCodeCorrupt
		}

		if debug {
			fmt.Println(" - copy, length:", length, "offset:", offset, "d-after:", d+length, "s-after:", s)
		}

		// Copy from an earlier sub-slice of dst to a later sub-slice.
		// If no overlap, use the built-in copy:
		if offset > length {
			copy(dst[d:d+length], dst[d-offset:])
			d += length
			continue
		}

		// Unlike the built-in copy function, this byte-by-byte copy always runs
		// forwards, even if the slices overlap. Conceptually, this is:
		//
		// d += forwardCopy(dst[d:d+length], dst[d-offset:])
		//
		// We align the slices into a and b and show the compiler they are the same size.
		// This allows the loop to run without bounds checks.
		a := dst[d : d+length]
		b := dst[d-offset:]
		b = b[:len(a)]
		for i := range a {
			a[i] = b[i]
		}
		d += length
	}
	if debug {
		fmt.Println("Done, d:", d, "s:", s, len(dst))
	}
	if d != len(dst) {
		if debugErrors {
			fmt.Println("corrupt: dst len:", len(dst), "d:", d)
		}
		return decodeErrCodeCorrupt
	}
	return 0
}
